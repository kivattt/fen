package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Bar struct {
	*tview.Box
	str *string
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
	tview.Print(screen, *bar.str, x, y, w, tview.AlignLeft, tcell.NewRGBColor(0, 255, 0))
}
