package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Ranger struct {
	wd      string
	sel     string
	history History

	selected []string

	historyMoment string

	topPane    *Bar
	leftPane   *FilesPane
	middlePane *FilesPane
	rightPane  *FilesPane
	bottomPane *Bar
}

func (r *Ranger) Init() error {
	var err error
	r.wd, err = os.Getwd()

	r.topPane = NewBar(&r.wd)

	r.leftPane = NewFilesPane(&r.selected)
	r.middlePane = NewFilesPane(&r.selected)
	r.rightPane = NewFilesPane(&r.selected)

	r.bottomPane = NewBar(&r.historyMoment)

	wdFiles, _ := os.ReadDir(r.wd)

	if len(wdFiles) > 0 {
		r.sel = filepath.Join(r.wd, wdFiles[0].Name())
	}

	r.history.AddToHistory(r.sel)
	r.UpdatePanes()

	return err
}

func (r *Ranger) UpdatePanes() {
	r.leftPane.SetEntries(filepath.Dir(r.wd))
	r.middlePane.SetEntries(r.wd)
	r.rightPane.SetEntries(r.sel)

/*	r.leftPane.entries, _ = os.ReadDir(filepath.Dir(r.wd))
	r.middlePane.entries, _ = os.ReadDir(r.wd)
	r.rightPane.entries, _ = os.ReadDir(r.sel)*/

	if r.wd != "/" {
		r.leftPane.SetSelectedEntryFromString(filepath.Base(r.wd))
	} else {
		r.leftPane.entries = []os.DirEntry{}
	}

	r.middlePane.SetSelectedEntryFromString(filepath.Base(r.sel))

	// FIXME: Generic bounds checking across all panes in this function
	if r.middlePane.selectedEntry >= len(r.middlePane.entries) {
		r.sel = r.middlePane.GetSelectedEntryFromIndex(len(r.middlePane.entries) - 1)
		r.middlePane.SetSelectedEntryFromString(filepath.Base(r.sel)) // Duplicated from above...
	}

	h, err := r.history.GetHistoryEntryForPath(r.sel)
	if err != nil {
		r.rightPane.SetSelectedEntryFromIndex(0)
		return
	}
	r.rightPane.SetSelectedEntryFromString(filepath.Base(h))
}

func (r *Ranger) ToggleSelection(filePath string) {
	if index := slices.Index(r.selected, filePath); index != -1 {
		r.selected = append(r.selected[:index], r.selected[index+1:]...)
		return
	}

	r.selected = append(r.selected, filePath)
}

func (r *Ranger) GetSelectedFilePath() string {
	if r.middlePane.selectedEntry >= len(r.middlePane.entries) {
		return ""
	}
	return filepath.Join(r.wd, r.middlePane.entries[r.middlePane.selectedEntry].Name())
}

func (r *Ranger) GoLeft() {
	if filepath.Dir(r.wd) == r.wd {
		return
	}

	r.sel = r.wd
	r.wd = filepath.Dir(r.wd)
}

func (r *Ranger) GoRight() {
	rightFiles, _ := os.ReadDir(r.sel)
	if len(rightFiles) <= 0 {
		return
	}

	r.wd = r.sel
	var err error
	r.sel, err = r.history.GetHistoryEntryForPath(r.wd)
	if err != nil {
		// FIXME
		r.sel = filepath.Join(r.wd, r.rightPane.GetSelectedEntryFromIndex(0))
	}
}

func (r *Ranger) GoUp() {
	if r.middlePane.selectedEntry-1 < 0 {
		r.sel = filepath.Join(r.wd, r.middlePane.GetSelectedEntryFromIndex(0))
		return
	}

	r.sel = filepath.Join(r.wd, r.middlePane.GetSelectedEntryFromIndex(r.middlePane.selectedEntry-1))
}

func (r *Ranger) GoDown() {
	if r.middlePane.selectedEntry+1 >= len(r.middlePane.entries) {
		r.sel = filepath.Join(r.wd, r.middlePane.GetSelectedEntryFromIndex(len(r.middlePane.entries)-1))
		return
	}

	r.sel = filepath.Join(r.wd, r.middlePane.GetSelectedEntryFromIndex(r.middlePane.selectedEntry+1))
}

func main() {
	var ranger Ranger
	ranger.Init()

	app := tview.NewApplication()

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' {
			app.Stop()
			return nil
		}

		if event.Key() == tcell.KeyF1 {
			cmd := exec.Command("nano", ranger.GetSelectedFilePath())
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				log.Fatal(err)
			}
			ranger.UpdatePanes()
			return nil
		}

		wasMovementKey := true
		if event.Key() == tcell.KeyLeft || event.Rune() == 'h' {
			ranger.GoLeft()
		} else if event.Key() == tcell.KeyRight || event.Rune() == 'l' {
			ranger.GoRight()
		} else if event.Key() == tcell.KeyUp || event.Rune() == 'k' {
			ranger.GoUp()
		} else if event.Key() == tcell.KeyDown || event.Rune() == 'j' {
			ranger.GoDown()
		} else if event.Rune() == ' ' {
			ranger.ToggleSelection(ranger.GetSelectedFilePath())
			ranger.historyMoment = strings.Join(ranger.selected, ", ")
			ranger.GoDown()
		} else if event.Key() == tcell.KeyHome || event.Rune() == 'g' {
			ranger.sel = ranger.middlePane.GetSelectedEntryFromIndex(0)
		} else if event.Key() == tcell.KeyEnd || event.Rune() == 'G' {
			ranger.sel = ranger.middlePane.GetSelectedEntryFromIndex(len(ranger.middlePane.entries) - 1)
		} else {
			wasMovementKey = false
		}

		if wasMovementKey {
			if !(event.Key() == tcell.KeyLeft || event.Rune() == 'h') {
				ranger.history.AddToHistory(ranger.sel)
			}

			ranger.UpdatePanes()
			return nil
		}

		if event.Key() == tcell.KeyDelete {
			fileToDelete := ranger.GetSelectedFilePath()
			os.Remove(fileToDelete)
			ranger.historyMoment = "Deleted " + fileToDelete

			ranger.UpdatePanes()
			return nil
		}

		return event
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ranger.topPane, 1, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(ranger.leftPane, 0, 1, false).
			AddItem(ranger.middlePane, 0, 2, false).
			AddItem(ranger.rightPane, 0, 2, false), 0, 1, false).
		AddItem(ranger.bottomPane, 1, 0, false)

	if err := app.SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}
