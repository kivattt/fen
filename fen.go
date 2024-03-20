package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/rivo/tview"
)

type Fen struct {
	wd      string
	sel     string
	history History

	selected     []string
	yankSelected []string
	yankType     string // "", "copy", "cut"

	historyMoment string

	showHiddenFiles bool

	topPane    *Bar
	leftPane   *FilesPane
	middlePane *FilesPane
	rightPane  *FilesPane
	bottomPane *Bar
}

func (fen *Fen) Init() error {
	fen.showHiddenFiles = true

	var err error
	fen.wd, err = os.Getwd()

	fen.topPane = NewBar(&fen.wd)

	fen.leftPane = NewFilesPane(&fen.selected, &fen.yankSelected, &fen.showHiddenFiles)
	fen.middlePane = NewFilesPane(&fen.selected, &fen.yankSelected, &fen.showHiddenFiles)
	fen.rightPane = NewFilesPane(&fen.selected, &fen.yankSelected, &fen.showHiddenFiles)

	fen.bottomPane = NewBar(&fen.historyMoment)

	wdFiles, _ := os.ReadDir(fen.wd)

	if len(wdFiles) > 0 {
		fen.sel = filepath.Join(fen.wd, wdFiles[0].Name())
	}

	fen.history.AddToHistory(fen.sel)
	fen.UpdatePanes()

	return err
}

func (fen *Fen) UpdatePanes() {
	fen.leftPane.SetEntries(filepath.Dir(fen.wd))
	fen.middlePane.SetEntries(fen.wd)

	if fen.wd != "/" {
		fen.leftPane.SetSelectedEntryFromString(filepath.Base(fen.wd))
	} else {
		fen.leftPane.entries = []os.DirEntry{}
	}

	fen.historyMoment = "Set selected entry from string: " + filepath.Base(fen.sel)
	fen.middlePane.SetSelectedEntryFromString(filepath.Base(fen.sel))

	// FIXME: Generic bounds checking across all panes in this function
	if fen.middlePane.selectedEntry >= len(fen.middlePane.entries) {
		if len(fen.middlePane.entries) > 0 {
			fen.sel = fen.middlePane.GetSelectedEntryFromIndex(len(fen.middlePane.entries) - 1)
			fen.middlePane.SetSelectedEntryFromString(filepath.Base(fen.sel)) // Duplicated from above...
		}
	}

	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.selectedEntry))
	fen.rightPane.SetEntries(fen.sel)

	// DEBUG
	//	fen.historyMoment = strings.Join(fen.history.history, ", ")

	h, err := fen.history.GetHistoryEntryForPath(fen.sel, !fen.showHiddenFiles)
	if err != nil {
		//		if fen.showHiddenFiles {
		fen.rightPane.SetSelectedEntryFromIndex(0)
		//		}
		//		fen.historyMoment = "BRUH"
		return
	}

	//	fen.historyMoment = "BRUH 2.0: " + filepath.Base(h)
	//	if fen.showHiddenFiles {
	fen.rightPane.SetSelectedEntryFromString(filepath.Base(h))
	// }
}

func (fen *Fen) RemoveFromSelectedAndYankSelected(path string) {
	if index := slices.Index(fen.selected, path); index != -1 {
		fen.selected = append(fen.selected[:index], fen.selected[index+1:]...)
	}

	if index := slices.Index(fen.yankSelected, path); index != -1 {
		fen.yankSelected = append(fen.yankSelected[:index], fen.yankSelected[index+1:]...)
	}
}

func (fen *Fen) ToggleSelection(filePath string) {
	if index := slices.Index(fen.selected, filePath); index != -1 {
		fen.selected = append(fen.selected[:index], fen.selected[index+1:]...)
		return
	}

	fen.selected = append(fen.selected, filePath)
}

func (fen *Fen) GoLeft() {
	// Not sure if this is necessary
	if filepath.Dir(fen.wd) == fen.wd {
		return
	}

	fen.sel = fen.wd
	fen.wd = filepath.Dir(fen.wd)
}

func (fen *Fen) GoRight(app *tview.Application) {
	if len(fen.middlePane.entries) <= 0 {
		return
	}

	fi, err := os.Stat(fen.sel)
	if err != nil {
		return
	}

	if !fi.IsDir() {
		cmd := exec.Command("nvim", fen.sel)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		app.Suspend(func() {
			if err := cmd.Run(); err != nil {
				log.Fatal(err)
			}
		})

		return
	}

	/*	rightFiles, _ := os.ReadDir(fen.sel)
		if len(rightFiles) <= 0 {
			return
		}*/

	fen.wd = fen.sel
	fen.sel, err = fen.history.GetHistoryEntryForPath(fen.wd, !fen.showHiddenFiles)

	if err != nil {
		// FIXME
		fen.sel = filepath.Join(fen.wd, fen.rightPane.GetSelectedEntryFromIndex(0))
	}
}

func (fen *Fen) GoUp() {
	if fen.middlePane.selectedEntry-1 < 0 {
		fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(0))
		return
	}

	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.selectedEntry-1))
}

func (fen *Fen) GoDown() {
	if fen.middlePane.selectedEntry+1 >= len(fen.middlePane.entries) {
		fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(len(fen.middlePane.entries)-1))
		return
	}

	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.selectedEntry+1))
}
