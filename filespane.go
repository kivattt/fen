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

func (fp *FilesPane) SetEntries(path string, foldersNotFirst bool) {
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

	if !foldersNotFirst {
		fp.entries = FoldersAtBeginning(fp.entries)
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
		style := FileColor(entryFullPath)

		spaceForSelected := ""
		if i+scrollOffset == fp.selectedEntry {
			style = style.Reverse(true)
		}

		if slices.Contains(*fp.selected, entryFullPath) {
			spaceForSelected = " "
			style = style.Foreground(tcell.ColorYellow)
		}

		// Dim the entry if its in yankSelected
		if slices.Contains(*fp.yankSelected, entryFullPath) {
			style = style.Dim(true)
		}

		tview.Print(screen, spaceForSelected+StyleToStyleTagString(style)+" "+tview.Escape(entry.Name())+strings.Repeat(" ", w), x, y+i, w-1, tview.AlignLeft, tcell.ColorDefault)

		if !fp.showEntrySizes {
			continue
		}

		entrySizeText, err := EntrySize(entryFullPath, *fp.dontShowHiddenFiles)
		if err != nil {
			entrySizeText = "?"
		}

		tview.Print(screen, StyleToStyleTagString(style)+" "+tview.Escape(entrySizeText)+" ", x, y+i, w-1, tview.AlignRight, tcell.ColorDefault)
	}
}
