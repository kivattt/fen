package main

import (
	"errors"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FilesPane struct {
	*tview.Box
//	entries       *[]string
	entries []os.DirEntry
	selectedEntry int
}

//func NewFilesPane(entries *[]string) *FilesPane {
func NewFilesPane() *FilesPane {
	return &FilesPane{
		Box:           tview.NewBox(),
//		entries:       entries,
		selectedEntry: 0,
	}
}

func (fp *FilesPane) SetSelectedEntryFromString(entryName string) error {
//	for i, entry := range *fp.entries {
	for i, entry := range fp.entries {
		if entry.Name() == entryName {
			fp.selectedEntry = i
			return nil
		}
	}

//	fp.selectedEntry = 0
	return errors.New("No entry with that name")
}

func (fp *FilesPane) SetSelectedEntryFromIndex(index int) {
	fp.selectedEntry = index
}

func (fp *FilesPane) GetSelectedEntryFromIndex(index int) string {
//	return (*fp.entries)[index]
	return fp.entries[index].Name()
}

func (fp *FilesPane) Draw(screen tcell.Screen) {
	fp.Box.DrawForSubclass(screen, fp)

	x, y, w, h := fp.GetInnerRect()

//	if len(*fp.entries) <= 0 {
	if len(fp.entries) <= 0 {
		tview.Print(screen, "empty", x, y, w, tview.AlignLeft, tcell.ColorRed)
		return
	}

//	for i, entry := range *fp.entries {
	for i, entry := range fp.entries {
		if i >= h {
			break
		}

		color := tcell.ColorWhite

		if entry.IsDir() {
			color = tcell.ColorBlue
		} else if entry.Type().IsRegular() {
			color = tcell.ColorWhite
		} else {
			color = tcell.ColorGray
		}

		if i == fp.selectedEntry {
			color = tcell.ColorYellow
		}

		tview.Print(screen, entry.Name(), x, y+i, w, tview.AlignLeft, color)
	}
}
