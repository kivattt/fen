package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FilesPane struct {
	*tview.Box
	selected        *[]string
	yankSelected    *[]string
	showHiddenFiles *bool
	folder          string
	entries         []os.DirEntry
	selectedEntry   int
}

func NewFilesPane(selected *[]string, yankSelected *[]string, showHiddenFiles *bool) *FilesPane {
	return &FilesPane{
		Box:             tview.NewBox(),
		selected:        selected,
		yankSelected:    yankSelected,
		showHiddenFiles: showHiddenFiles,
		selectedEntry:   0,
	}
}

func (fp *FilesPane) SetEntries(path string) {
	fi, err := os.Stat(path)
	if err != nil {
		fp.entries = []os.DirEntry{}
		return
	}

	if !fi.IsDir() {
		fp.entries = []os.DirEntry{}
		return
	}

	fp.folder = path
	fp.entries, _ = os.ReadDir(fp.folder)

	if !*fp.showHiddenFiles {
		withoutHiddenFiles := []os.DirEntry{}
		for _, e := range fp.entries {
			if !strings.HasPrefix(e.Name(), ".") {
				withoutHiddenFiles = append(withoutHiddenFiles, e)
			}
		}

		fp.entries = withoutHiddenFiles

		// TODO: Generic bounds checking function?
		if len(fp.entries) > 0 && fp.selectedEntry >= len(fp.entries) {
			fp.selectedEntry = len(fp.entries) - 1
			//			fp.SetSelectedEntryFromIndex(len(fp.entries) - 1)
		}
	}
}

func (fp *FilesPane) SetSelectedEntryFromString(entryName string) error {
	for i, entry := range fp.entries {
		if entry.Name() == entryName {
			fp.selectedEntry = i
			return nil
		}
	}

	fp.selectedEntry = 0
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

	if fp.folder == "/" {
		return
	}

	if len(fp.entries) <= 0 {
		tview.Print(screen, "[:red]empty", x, y, w, tview.AlignLeft, tcell.ColorDefault)
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
			fi, err := os.Stat(entry.Name())
			if err == nil {
				// Executable?
				if fi.Mode() & 0111 != 0 {
					color = tcell.NewRGBColor(0, 255, 0)
				} else {
					color = tcell.ColorWhite
				}
			}
		} else {
			color = tcell.ColorGray
		}

		extraStyle := ""
		if i == fp.selectedEntry {
			extraStyle = "[:gray]"
			color = tcell.ColorBlack
		}

		if slices.Contains(*fp.selected, filepath.Join(fp.folder, entry.Name())) {
			extraStyle = " " + extraStyle
			color = tcell.ColorYellow
		}

		// Dim the entry if its in yankSelected
		dimColor := slices.Contains(*fp.yankSelected, filepath.Join(fp.folder, entry.Name()))

		if dimColor {
			// FIXME: Dim any color, yellow should turn dim aswell as the "no color"
			tview.Print(screen, "[:bold]"+extraStyle+entry.Name(), x, y+i, w-3, tview.AlignLeft, tcell.ColorDimGray)
		} else {
			tview.Print(screen, "[:bold]"+extraStyle+entry.Name(), x, y+i, w-3, tview.AlignLeft, color)
		}
	}
}
