// Copyright 2017 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build example

package main

import (
    "bytes"
    "fmt"
    "github.com/hajimehoshi/oto"
    "github.com/jroimartin/gocui"
    "github.com/ssor/go-mp3/consts"
    "io"
    "io/ioutil"
    "log"
    "os"
    "strings"
    "time"
)

const (
    viewHelp       = "viewHelp"
    viewFileInfo   = "viewFileInfo"
    viewPlayStatus = "viewPlayStatus"
    viewMessage    = "viewMessage"
)

type frame []byte

var (
    mp3FileName = "20180927sa_science.mp3"
    mp3Time     time.Duration
    loopStart   = 0
    loopEnd     = 0
    audioData   []frame
    frameCount  int
    player      io.Writer
    audioSrc    *AudioDataSource
    shiftStep   = 50
    speed       = 1
)

func prepareAudioData(audioFile string) ([]frame, int, error) {
    logMessage("trying to decode mp3 ...")
    f, err := os.Open(audioFile)
    if err != nil {
        return nil, 0, err
    }
    defer f.Close()

    d, err := mp3.NewDecoder(f)
    if err != nil {
        return nil, 0, err
    }
    defer d.Close()

    dataRaw, err := ioutil.ReadAll(d)
    if err != nil {
        logMessage("read decoded data failed")
        return nil, 0, err
    }
    logMessage("decode mp3 OK")

    totalLen := len(dataRaw)
    frameCount = totalLen / consts.BytesPerFrame
    //fmt.Println("total len: ", totalLen, "  frame count: ", frameCount)
    var data []frame

    for i := 0; i < totalLen; i += consts.BytesPerFrame {
        //fmt.Println("i = ", i)
        data = append(data, dataRaw[i:i+consts.BytesPerFrame])
    }

    if frameCount != len(data) {
        panic("frame parse error")
    }
    loopEnd = frameCount

    mp3Time = time.Duration(frameCount) * 26 * time.Millisecond
    updateFileInfo()

    logMessage("prepare audio data OK")
    return data, d.SampleRate(), nil
}

func preparePlayer(sampleRate int) (io.Writer, error) {
    p, err := oto.NewPlayer(sampleRate, 2, 2, 8192)
    if err != nil {
        return nil, err
    }
    logMessage("prepare player OK")
    return p, nil
}

func playData(data []byte) {
    if data == nil || len(data) <= 0 {
        return
    }
    _, err := player.Write(data)
    if err != nil {
        logMessage("write to player: " + err.Error())
        panic(err)
    }
}

func newAudioDataSource() *AudioDataSource {
    src := &AudioDataSource{
        endFrame: loopEnd,
    }
    return src
}

type AudioDataSource struct {
    startFrame   int
    endFrame     int
    currentFrame int
}

func (ads *AudioDataSource) NextFrame() []byte {
    if ads.endFrame <= ads.startFrame {
        ads.endFrame = frameCount - 1
    }
    ads.currentFrame++
    if ads.currentFrame+1 > ads.endFrame {
        ads.currentFrame = ads.startFrame
    }
    updatePlayStatus()
    return audioData[ads.currentFrame]
}

func (ads *AudioDataSource) updateEndFrame(s int) {
    if s < 0 || s >= frameCount {
        return
    }
    ads.endFrame = s
    updatePlayStatus()
}

func (ads *AudioDataSource) updateStartFrame(s int) {
    if s < 0 || s >= frameCount {
        return
    }
    ads.startFrame = s
    updatePlayStatus()
}

func playAudio(player io.Writer, src *AudioDataSource) {
    logMessage("playing ...")
    index := 0
    var temp frame
    for {
        index++
        if index <= speed {
            playData(temp)
            continue
        } else {
            index = 0
        }
        temp = src.NextFrame()
    }
}

func run() error {
    f, err := os.Open(mp3FileName)
    if err != nil {
        return err
    }
    defer f.Close()

    d, err := mp3.NewDecoder(f)
    if err != nil {
        return err
    }
    defer d.Close()

    p, err := oto.NewPlayer(d.SampleRate(), 2, 2, 8192)
    if err != nil {
        return err
    }
    defer p.Close()

    //fmt.Printf("Length: %d[bytes]\n", d.Length())

    dataAll, err := ioutil.ReadAll(d)
    if err != nil {
        log.Fatalln("read decoded data failed: ")
        return err
    }

    frameCount := len(dataAll) / consts.BytesPerFrame
    //log.Println(frameCount, " frames in audio file, and will cost about ", (time.Duration(frameCount) * 26 * time.Millisecond), " to play")
    mp3Time = time.Duration(frameCount) * 26 * time.Millisecond
    updateFileInfo()

    buf := bytes.NewBuffer(dataAll)
    logMessage("start playing ...")
    speed := 2
    index := 0
    var temp []byte
    for {
        index++
        if index <= speed {
            if temp != nil {
                _, err = p.Write(temp)
                if err != nil {
                    log.Println("write to player: ", err)
                    break
                }
            }
            continue
        } else {
            index = 0
        }
        frameData := buf.Next(consts.BytesPerFrame)
        temp = frameData
    }
    //if _, err := io.Copy(p, d); err != nil {
    //	return err
    //}
    return nil
}

func updateFileInfo() {
    //maxX, _ := gui.Size()
    gui.Update(func(g *gocui.Gui) error {
        v, err := g.View(viewFileInfo)
        if err != nil && err != gocui.ErrUnknownView {
            panic(err)
        }
        v.Clear()
        fmt.Fprintf(v, joinFileInfo(mp3FileName, mp3Time, frameCount))
        return nil
    })
}

func updatePlayStatus() {
    gui.Update(func(g *gocui.Gui) error {
        v, err := g.View(viewPlayStatus)
        if err != nil && err != gocui.ErrUnknownView {
            panic(err)
        }
        v.Clear()
        fmt.Fprintf(v, joinPlayStatus())
        return nil
    })
}

var (
    gui *gocui.Gui
)

func main() {
    g, err := gocui.NewGui(gocui.OutputNormal)
    if err != nil {
        log.Panicln(err)
    }
    defer g.Close()

    gui = g

    g.SetManagerFunc(layout)

    if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
        log.Panicln(err)
    }
    if err := g.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, loopStartSmaller); err != nil {
        log.Panicln(err)
    }
    if err := g.SetKeybinding("", gocui.KeyArrowRight, gocui.ModNone, loopStartLarger); err != nil {
        log.Panicln(err)
    }
    if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, loopEndLarger); err != nil {
        log.Panicln(err)
    }
    if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, loopEndSmaller); err != nil {
        log.Panicln(err)
    }
    if err := g.SetKeybinding("", gocui.KeySpace, gocui.ModNone, speedUpAndDown); err != nil {
        log.Panicln(err)
    }
    if err := g.SetKeybinding("", gocui.KeyCtrlW, gocui.ModNone, showNextWord); err != nil {
        log.Panicln(err)
    }

    go func() {
        frames, rate, err := prepareAudioData(mp3FileName)
        if err != nil {
            panic(err)
        }
        audioData = frames

        p, err := preparePlayer(rate)
        if err != nil {
            panic(err)
        }
        player = p

        audioSrc = newAudioDataSource()
        audioSrc.endFrame = frameCount
        playAudio(p, audioSrc)
    }()
    if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
        log.Panicln(err)
    }
}

func layout(g *gocui.Gui) error {
    maxX, maxY := g.Size()

    if v, err := g.SetView(viewHelp, 1, 1, maxX-1, 6); err != nil {
        if err != gocui.ErrUnknownView {
            return err
        }
        v.Title = "Help:"
        keyBindings1 := "Loop start point:            <- and ->"
        keyBindings2 := "Loop end point:              up arrow and down arrow"
        keyBindings3 := "Speed-Up and Speed-Down      space"
        keyBindings4 := "Show next word               ctrl+w"
        fmt.Fprintf(v, strings.Join([]string{keyBindings1, keyBindings2, keyBindings3, keyBindings4}, "\n"))
    }

    if v, err := g.SetView(viewFileInfo, 1, 7, maxX-1, 10); err != nil {
        if err != gocui.ErrUnknownView {
            return err
        }
        v.Title = "Playing File:"
        fmt.Fprintf(v, joinFileInfo(mp3FileName, mp3Time, frameCount))
    }

    if v, err := g.SetView(viewPlayStatus, 1, 11, maxX-1, 15); err != nil {
        if err != gocui.ErrUnknownView {
            return err
        }
        v.Title = "Status:"
        fmt.Fprintf(v, joinPlayStatus())
    }

    if v, err := g.SetView(viewMessage, -1, maxY-3, maxX, maxY-1); err != nil {
        if err != gocui.ErrUnknownView {
            return err
        }
        fmt.Fprintf(v, ":")
    }
    return nil
}

func joinFileInfo(name string, timeCost time.Duration, frames int) string {
    return fmt.Sprintf("File: %s\nTime: %s\nFrames: %d", name, timeCost, frames)
}

func joinPlayStatus() string {
    currentFrame := 0
    if audioSrc != nil {
        currentFrame = audioSrc.currentFrame
    }
    if frameCount <= 0 {
        frameCount = 1
    }
    percent := currentFrame * 100 / frameCount

    var percentTag string
    switch percent {
    case 0:
        percentTag = strings.Repeat("-", 100)
    case 100:
        percentTag = fmt.Sprintf("%s100", strings.Repeat("-", 99))
    default:
        percentTag = fmt.Sprintf("%s%d%s", strings.Repeat("-", percent-1), percent, strings.Repeat("-", 100-percent))
    }

    res := fmt.Sprintf("%-10s: %d (%d) ->  %d (%d)\n%-10s: %d/%d \n%s",
        "Repeat", loopStart, loopStart*100/frameCount, loopEnd, loopEnd*100/frameCount, "Frames", currentFrame, frameCount, percentTag)
    return res
}

func loopStartSmaller(g *gocui.Gui, v *gocui.View) error {
    loopStart -= shiftStep
    if loopStart < 0 {
        loopStart = 0
    }
    audioSrc.updateStartFrame(loopStart)
    return nil
}

func loopStartLarger(g *gocui.Gui, v *gocui.View) error {
    loopStart += shiftStep
    if loopStart >= frameCount {
        loopStart = frameCount - shiftStep
    }
    audioSrc.updateStartFrame(loopStart)
    return nil
}

func showNextWord(g *gocui.Gui, v *gocui.View) error {
    return nil
}
func speedDown(g *gocui.Gui, v *gocui.View) error {
    speed++
    if speed > 2 {
        speed = 2
    }
    return nil
}

func speedUpAndDown(g *gocui.Gui, v *gocui.View) error {
    if speed == 1 {
        speed = 2
    } else {
        speed = 1
    }
    return nil
}

func loopEndLarger(g *gocui.Gui, v *gocui.View) error {
    loopEnd += shiftStep
    if loopEnd >= frameCount {
        loopEnd = frameCount - 1
    }
    audioSrc.updateEndFrame(loopEnd)
    return nil
}

func loopEndSmaller(g *gocui.Gui, v *gocui.View) error {
    loopEnd -= shiftStep
    if loopEnd < 0 {
        loopEnd = 0
    }
    audioSrc.updateEndFrame(loopEnd)
    return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
    return gocui.ErrQuit
}

func logMessage(msg string) {
    gui.Update(func(g *gocui.Gui) error {
        v, err := g.View(viewMessage)
        if err != nil && err != gocui.ErrUnknownView {
            panic(err)
        }
        v.Clear()
        fmt.Fprintf(v, "-> "+msg)
        return nil
    })
}
