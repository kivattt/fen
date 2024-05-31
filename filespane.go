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
		Box:                 tview.NewBox().SetBackgroundColor(tcell.ColorDefault),
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

// Returns -1 if nothing was found
func (fp *FilesPane) GetSelectedIndexFromEntry(entryName string) int {
	for i, entry := range fp.entries {
		if entry.Name() == entryName {
			return i
		}
	}

	return -1
}

// Used as scroll offset aswell
func (fp *FilesPane) GetTopScreenEntryIndex() int {
	_, _, _, h := fp.GetInnerRect()
	topScreenEntryIndex := 0
	if fp.selectedEntry > h/2 {
		topScreenEntryIndex = fp.selectedEntry - h/2
	}

	if topScreenEntryIndex >= len(fp.entries) {
		topScreenEntryIndex = max(0, len(fp.entries)-1)
	}

	return topScreenEntryIndex
}

func (fp *FilesPane) GetBottomScreenEntryIndex() int {
	_, _, _, h := fp.GetInnerRect()
	bottomScreenEntryIndex := fp.GetTopScreenEntryIndex() + h - 1
	if bottomScreenEntryIndex >= len(fp.entries) {
		bottomScreenEntryIndex = max(0, len(fp.entries)-1)
	}

	return bottomScreenEntryIndex
}

func (fp *FilesPane) Draw(screen tcell.Screen) {
	fp.Box.DrawForSubclass(screen, fp)

	x, y, w, h := fp.GetInnerRect()

	if len(fp.entries) <= 0 && fp.folder != filepath.Dir(fp.folder) {
		tview.Print(screen, "[:red]empty", x, y, w, tview.AlignLeft, tcell.ColorDefault)
		return
	}

	scrollOffset := fp.GetTopScreenEntryIndex()

	for i, entry := range fp.entries[scrollOffset:] {
		if i >= h {
			break
		}

		entryFullPath := filepath.Join(fp.folder, entry.Name())
		stat, statErr := os.Stat(entryFullPath)

		color := tcell.ColorWhite

		bold := false
		if entry.IsDir() {
			color = tcell.ColorBlue
			bold = true
		} else if entry.Type().IsRegular() {
			if statErr == nil {
				// Executable?
				if stat.Mode()&0111 != 0 {
					color = tcell.NewRGBColor(0, 255, 0) // Green
					bold = true
				} else {
					color = FileColor(entry.Name())
				}
			}
		} else {
			color = tcell.ColorDarkGray
		}

		spaceForSelected := ""
		extraStyle := ""
		if i+scrollOffset == fp.selectedEntry {
			extraStyle = "[::r]" // Flip foreground and background
		}

		if slices.Contains(*fp.selected, entryFullPath) {
			spaceForSelected = " "
			color = tcell.ColorYellow
		}

		// Dim the entry if its in yankSelected
		dimColor := slices.Contains(*fp.yankSelected, entryFullPath)

		if dimColor {
			// Kinda cursed to have to add on to this extraStyle variable
			if extraStyle == "" {
				extraStyle = "[::d]"
			} else {
				extraStyle = "[::rd]"
			}
		}

		if bold {
			extraStyle += "[::b]"
		}

		tview.Print(screen, spaceForSelected+extraStyle+" "+tview.Escape(entry.Name())+strings.Repeat(" ", w), x, y+i, w-1, tview.AlignLeft, color)

		if !fp.showEntrySizes {
			continue
		}

		entrySizeText, err := EntrySize(entryFullPath, *fp.dontShowHiddenFiles)
		if err != nil {
			entrySizeText = "?"
		}

		tview.Print(screen, extraStyle+" "+tview.Escape(entrySizeText)+" ", x, y+i, w-1, tview.AlignRight, color)
	}
}
