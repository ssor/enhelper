module github.com/ssor/enhelper

replace github.com/ssor/go-mp3 v0.1.1 => ../go-mp3

replace golang.org/x/exp v0.0.0-20180710024300-14dda7b62fcd => ../../golang/exp

replace golang.org/x/mobile v0.0.0-20180806140643-507816974b79 => ../../golang/mobile

replace golang.org/x/image v0.0.0-20180708004352-c73c2afc3b81 => ../../golang/image

replace golang.org/x/sys v0.0.0-20180806082429-34b17bdb4300 => ../../golang/sys

require (
	github.com/hajimehoshi/oto v0.1.4
	github.com/jroimartin/gocui v0.4.0
	github.com/mattn/go-runewidth v0.0.3 // indirect
	github.com/nsf/termbox-go v0.0.0-20180819125858-b66b20ab708e // indirect
	github.com/ssor/go-mp3 v0.1.1
)
