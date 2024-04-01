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
	selected            *[]string
	yankSelected        *[]string
	dontShowHiddenFiles *bool
	folder              string
	entries             []os.DirEntry
	selectedEntry       int
	showEntrySizes      bool
}

func NewFilesPane(selected *[]string, yankSelected *[]string, dontShowHiddenFiles *bool, showEntrySizes bool) *FilesPane {
	return &FilesPane{
		Box:                 tview.NewBox(),
		selected:            selected,
		yankSelected:        yankSelected,
		dontShowHiddenFiles: dontShowHiddenFiles,
		selectedEntry:       0,
		showEntrySizes:      showEntrySizes,
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

	if *fp.dontShowHiddenFiles {
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

func (fp *FilesPane) GetSelectedPathFromIndex(index int) string {
	return filepath.Join(fp.folder, fp.GetSelectedEntryFromIndex(index))
}

func (fp *FilesPane) Draw(screen tcell.Screen) {
	fp.Box.DrawForSubclass(screen, fp)

	x, y, w, h := fp.GetInnerRect()

	if len(fp.entries) <= 0 && fp.folder != "/" {
		tview.Print(screen, "[:red]empty", x, y, w, tview.AlignLeft, tcell.ColorDefault)
		return
	}

	for i, entry := range fp.entries {
		if i >= h {
			break
		}

		entryFullPath := filepath.Join(fp.folder, entry.Name())
		stat, statErr := os.Stat(entryFullPath)

		color := tcell.ColorWhite

		if entry.IsDir() {
			color = tcell.ColorBlue
		} else if entry.Type().IsRegular() {
			if statErr == nil {
				// Executable?
				if stat.Mode()&0111 != 0 {
					color = tcell.NewRGBColor(0, 255, 0)
				} else {
					color = FileColor(entry.Name())
				}
			}
		} else {
			color = tcell.ColorGray
		}

		spaceForSelected := ""
		extraStyle := ""
		if i == fp.selectedEntry {
			extraStyle = "[::r]" // Flip foreground and background
		}

		if slices.Contains(*fp.selected, entryFullPath) {
			spaceForSelected = " "
			color = tcell.ColorYellow
		}

		// Dim the entry if its in yankSelected
		dimColor := slices.Contains(*fp.yankSelected, entryFullPath)

		if dimColor {
			r, g, b := color.RGB()
			dimAmount := 2
			color = tcell.NewRGBColor(r/int32(dimAmount), g/int32(dimAmount), b/int32(dimAmount))
		}

		tview.Print(screen, "[:bold]"+spaceForSelected+extraStyle+" "+entry.Name()+strings.Repeat(" ", w), x, y+i, w-1, tview.AlignLeft, color)

		if !fp.showEntrySizes {
			continue
		}

		entrySizeText, err := EntrySize(entryFullPath, *fp.dontShowHiddenFiles)
		if err != nil {
			entrySizeText = "?"
		}
		tview.Print(screen, "[:bold]"+extraStyle+entrySizeText, x, y+i, w-1, tview.AlignRight, color)
	}
}
