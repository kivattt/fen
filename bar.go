package main

import (
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Bar struct {
	*tview.Box
	str      *string
	isTopBar bool // For making the filepath base a different color
}

func NewBar(str *string) *Bar {
	return &Bar{
		Box: tview.NewBox(),
		str: str,
	}
}

func (bar *Bar) Draw(screen tcell.Screen) {
	bar.Box.DrawForSubclass(screen, bar)

	x, y, w, _ := bar.GetInnerRect()
	text := *bar.str
	if bar.isTopBar {
		text = PathWithEndSeparator(filepath.Dir(text)) + "[white:]" + PathWithoutEndSeparator(filepath.Base(text))
	}
	tview.Print(screen, text, x, y, w, tview.AlignLeft, tcell.ColorBlue)
}
