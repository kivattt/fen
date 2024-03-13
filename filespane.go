package main

import (
	"errors"
	"slices"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FilesPane struct {
	*tview.Box
	selected *[]string
	folder string
	entries       []os.DirEntry
	selectedEntry int
}

func NewFilesPane(selected *[]string) *FilesPane {
	return &FilesPane{
		Box:           tview.NewBox(),
		selected: selected,
		selectedEntry: 0,
	}
}

func (fp *FilesPane) SetEntries(path string) {
	fp.folder = path
	fp.entries, _ = os.ReadDir(fp.folder)
}

func (fp *FilesPane) SetSelectedEntryFromString(entryName string) error {
	for i, entry := range fp.entries {
		if entry.Name() == entryName {
			fp.selectedEntry = i
			return nil
		}
	}

	return errors.New("No entry with that name")
}

func (fp *FilesPane) SetSelectedEntryFromIndex(index int) {
	fp.selectedEntry = index
}

func (fp *FilesPane) GetSelectedEntryFromIndex(index int) string {
	if index >= len(fp.entries) {
		return ""
	}

	if index < 0 {
		return ""
	}

	return fp.entries[index].Name()
}

func (fp *FilesPane) Draw(screen tcell.Screen) {
	fp.Box.DrawForSubclass(screen, fp)

	x, y, w, h := fp.GetInnerRect()

	if len(fp.entries) <= 0 {
		tview.Print(screen, "empty", x, y, w, tview.AlignLeft, tcell.ColorRed)
		return
	}

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

		extraStyle := ""
		if i == fp.selectedEntry {
			extraStyle = "[:gray]"
			color = tcell.ColorBlack
		}

		if slices.Contains(*fp.selected, filepath.Join(fp.folder, entry.Name())) {
			extraStyle = "  " + extraStyle
			color = tcell.ColorYellow
		}

		tview.Print(screen, extraStyle + entry.Name(), x, y+i, w-3, tview.AlignLeft, color)
	}
}
