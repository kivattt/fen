package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FilesPane struct {
	*tview.Box
	entries       *[]string
	selectedEntry int
}

func NewFilesPane(entries *[]string) *FilesPane {
	return &FilesPane{
		Box:           tview.NewBox(),
		entries:       entries,
		selectedEntry: 0,
	}
}

func (fp *FilesPane) SetSelectedEntryFromString(entryName string) {
	for i, entry := range *fp.entries {
		if entry == entryName {
			fp.selectedEntry = i
			return
		}
	}

	fp.selectedEntry = 0
}

func (fp *FilesPane) Draw(screen tcell.Screen) {
	fp.Box.DrawForSubclass(screen, fp)

	x, y, w, h := fp.GetInnerRect()

	for i, entry := range *fp.entries {
		if i >= h {
			break
		}

		color := tcell.ColorWhite
		if i == fp.selectedEntry {
			color = tcell.ColorYellow
		}

		tview.Print(screen, entry, x, y+i, w, tview.AlignLeft, color)
	}
}
