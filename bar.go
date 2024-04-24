package main

import (
	"os/user"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Bar struct {
	*tview.Box
	str      *string
	isTopBar bool // For making the filepath base a different color
	noWrite  *bool
}

func NewBar(str *string, noWrite *bool) *Bar {
	return &Bar{
		Box:     tview.NewBox().SetBackgroundColor(tcell.ColorDefault),
		str:     str,
		noWrite: noWrite,
	}
}

func (bar *Bar) Draw(screen tcell.Screen) {
	if !bar.isTopBar {
		bar.Box.SetBackgroundColor(tcell.ColorBlack)
		//				bar.Box.SetBackgroundColor(tcell.ColorGray)
		//		bar.Box.SetBackgroundColor(tcell.ColorWhite)
	}
	bar.Box.DrawForSubclass(screen, bar)

	x, y, w, _ := bar.GetInnerRect()
	text := *bar.str
	if bar.isTopBar {
		user, _ := user.Current()
		usernameColor := "[lime:]"
		if user.Uid == "0" {
			usernameColor = "[red:]"
		}
		text = "[::b]" + usernameColor + user.Username + " [blue::B]" + PathWithEndSeparator(filepath.Dir(text)) + "[white::b]" + PathWithoutEndSeparator(filepath.Base(text))
	}

	noWriteEnabledText := ""
	if !bar.isTopBar && *bar.noWrite {
		noWriteEnabledText = " [red:]NO-WRITE ENABLED!"
	}
	tview.Print(screen, text+noWriteEnabledText, x, y, w, tview.AlignLeft, tcell.ColorBlue)
}
