package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	dirCopy "github.com/otiai10/copy"
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
//	}
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

func main() {
	var fen Fen
	fen.Init()

	app := tview.NewApplication()

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(fen.topPane, 1, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(fen.leftPane, 0, 1, false).
			AddItem(fen.middlePane, 0, 2, false).
			AddItem(fen.rightPane, 0, 2, false), 0, 1, false).
		AddItem(fen.bottomPane, 1, 0, false)

	pages := tview.NewPages().
		AddPage("flex", flex, true, true)

	app.SetMouseCapture(func(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
		wasMovementKey := true

		switch event.Buttons() {
		case tcell.WheelLeft:
			fen.GoLeft()
		case tcell.WheelRight:
			fen.GoRight(app)
		case tcell.WheelUp:
			fen.GoUp()
		case tcell.WheelDown:
			fen.GoDown()
		default:
			wasMovementKey = false
		}

		if wasMovementKey {
			if !(event.Buttons() == tcell.WheelLeft) {
				fen.history.AddToHistory(fen.sel)
			}

			fen.historyMoment = fen.sel
			fen.UpdatePanes()
			return nil, action // ?
		}

		return event, action
	})

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if pages.HasPage("modal") || pages.HasPage("inputfield") {
			return event
		}

		if event.Rune() == 'q' {
			app.Stop()
			return nil
		}

		wasMovementKey := true
		if event.Key() == tcell.KeyLeft || event.Rune() == 'h' {
			fen.GoLeft()
		} else if event.Key() == tcell.KeyRight || event.Rune() == 'l' || event.Key() == tcell.KeyEnter {
			fen.GoRight(app)
		} else if event.Key() == tcell.KeyUp || event.Rune() == 'k' {
			fen.GoUp()
		} else if event.Key() == tcell.KeyDown || event.Rune() == 'j' {
			fen.GoDown()
		} else if event.Rune() == ' ' {
			fen.ToggleSelection(fen.sel)
			fen.historyMoment = strings.Join(fen.selected, ", ")
			fen.GoDown()
		} else if event.Key() == tcell.KeyHome || event.Rune() == 'g' {
			fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(0))
		} else if event.Key() == tcell.KeyEnd || event.Rune() == 'G' {
			fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(len(fen.middlePane.entries)-1))
		} else if event.Rune() == 'M' {
			fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex((len(fen.middlePane.entries)-1)/2))
		} else {
			wasMovementKey = false
		}

		if wasMovementKey {
			if !(event.Key() == tcell.KeyLeft || event.Rune() == 'h') {
				fen.history.AddToHistory(fen.sel)
			}

			fen.historyMoment = fen.sel
			fen.UpdatePanes()
			return nil
		}

		if event.Rune() == 'A' {
			for _, e := range fen.middlePane.entries {
				fen.ToggleSelection(filepath.Join(fen.wd, e.Name()))
			}
			return nil
		} else if event.Rune() == 'D' {
			fen.selected = []string{}
			fen.yankSelected = []string{}
			fen.historyMoment = "Deselected and un-yanked!"
		} else if event.Rune() == 'a' {
			fileToRename := fen.sel

			// https://github.com/rivo/tview/wiki/Modal
			modal := func(p tview.Primitive, width, height int) tview.Primitive {
				return tview.NewFlex().
					AddItem(nil, 0, 1, false).
					AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
						AddItem(nil, 0, 1, false).
						AddItem(p, height, 1, true).
						AddItem(nil, 0, 1, false), width, 1, true).
					AddItem(nil, 0, 1, false)
			}

			inputField := tview.NewInputField().
				SetLabel("New name: ").
				SetText(filepath.Base(fileToRename)).
				SetFieldWidth(45)

			inputField.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					pages.RemovePage("inputfield")
					return
				} else if key == tcell.KeyEnter {
					newPath := filepath.Join(filepath.Dir(fileToRename), inputField.GetText())
					os.Rename(fileToRename, newPath)

					fen.RemoveFromSelectedAndYankSelected(fileToRename)
					fen.history.RemoveFromHistory(fileToRename)
					fen.history.AddToHistory(newPath)
					fen.sel = newPath

					fen.UpdatePanes()
					//fen.historyMoment = fen.sel

					pages.RemovePage("inputfield")
					return
				}
			})

			inputField.SetBorder(true)

			pages.AddPage("inputfield", modal(inputField, 58, 3), true, true)
			app.SetFocus(inputField)
			return nil
		} else if event.Rune() == 'y' {
			fen.yankType = "copy"
			if len(fen.selected) <= 0 {
				fen.yankSelected = []string{fen.sel}
			} else {
				fen.yankSelected = fen.selected
			}
			fen.historyMoment = "Yank!"
			return nil
		} else if event.Rune() == 'd' {
			fen.yankType = "cut"
			fen.yankSelected = fen.selected
			fen.historyMoment = "Cut!"
			return nil
		} else if event.Rune() == 'z' {
			fen.showHiddenFiles = !fen.showHiddenFiles
			fen.UpdatePanes()
			fen.history.AddToHistory(fen.sel)
//			fen.historyMoment = strings.Join(fen.history.history, ", ") // TODO: remove later
//			fen.historyMoment = fen.sel
		} else if event.Rune() == 'p' {
			if len(fen.yankSelected) <= 0 {
				fen.historyMoment = "Nothing to paste..."
				return nil
			}

			if fen.yankType == "copy" {
				for _, e := range fen.yankSelected {
					fi, err := os.Stat(e)
					if err != nil {
						continue
					}

					newPath := filepath.Join(fen.wd, filepath.Base(e))
					if fi.IsDir() {
						err := os.Mkdir(newPath, 0755)
						if err != nil {
							// TODO: We need an error log we can scroll through
							fen.historyMoment = newPath
						}
						//						fen.historyMoment = fen.sel
						fen.historyMoment = fen.wd

						err = dirCopy.Copy(e, newPath)
					} else if fi.Mode().IsRegular() {
						source, err := os.Open(e)
						if err != nil {
							// TODO: We need an error log we can scroll through
							continue
						}
						defer source.Close()

						destination, err := os.Create(newPath)
						if err != nil {
							// TODO: We need an error log we can scroll through
							continue
						}
						defer destination.Close()

						_, err = io.Copy(destination, source)
						if err != nil {
							// TODO: We need an error log we can scroll through
							continue
						}
					}
				}
			}

			// Reset selection after paste
			fen.yankSelected = []string{}
			fen.selected = []string{}

			fen.UpdatePanes()
//			fen.historyMoment = "Paste! (fen.sel = " + fen.sel + ")"

			return nil
		}

		if event.Key() == tcell.KeyDelete {
			modal := tview.NewModal()
			modal.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				switch e.Rune() {
				case 'h':
					return tcell.NewEventKey(tcell.KeyLeft, e.Rune(), e.Modifiers())
				case 'l':
					return tcell.NewEventKey(tcell.KeyRight, e.Rune(), e.Modifiers())
				case 'j':
					return tcell.NewEventKey(tcell.KeyDown, e.Rune(), e.Modifiers())
				case 'k':
					return tcell.NewEventKey(tcell.KeyUp, e.Rune(), e.Modifiers())
				}

				return e
			})

			fileToDelete := ""

			if len(fen.selected) <= 0 {
				fileToDelete = fen.sel
				modal.SetText("Delete " + filepath.Base(fileToDelete) + " ?")
			} else {
				modal.SetText("Delete " + strconv.Itoa(len(fen.selected)) + " selected files?")
			}

			modal.
				AddButtons([]string{"Yes", "No"}).
				SetFocus(1). // Default is "No"
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					pages.RemovePage("modal")
					if buttonLabel != "Yes" {
						return
					}

					if len(fen.selected) <= 0 {
						err := os.RemoveAll(fileToDelete)
						if err != nil {
							// TODO: We need an error log we can scroll through
							fen.historyMoment = "Failed to delete!"
							return
						}
						fen.history.RemoveFromHistory(fileToDelete)
						fen.historyMoment = "Deleted " + fileToDelete
					} else {
						for _, filePath := range fen.selected {
							err := os.RemoveAll(filePath)
							if err != nil {
								// TODO: We need an error log we can scroll through
								continue
							}
							fen.history.RemoveFromHistory(filePath)
						}

						fen.historyMoment = "Deleted " + strings.Join(fen.selected, ", ")
					}

					fen.selected = []string{}

					fen.GoDown()
					fen.UpdatePanes()
				})

			pages.AddPage("modal", modal, true, true)
			app.SetFocus(modal)
			return nil
		}

		return event
	})

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
