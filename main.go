package main

import (
	//	"io"
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

type Ranger struct {
	wd      string
	sel     string
	history History

	selected     []string
	yankSelected []string
	yankType     string // "", "copy", "cut"

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
	/*	_, err := os.Stat(r.sel)
		if err != nil {
			return
		}*/

	r.leftPane.SetEntries(filepath.Dir(r.wd))
	r.middlePane.SetEntries(r.wd)
	r.rightPane.SetEntries(r.sel)

	if r.wd != "/" {
		r.leftPane.SetSelectedEntryFromString(filepath.Base(r.wd))
	} else {
		r.leftPane.entries = []os.DirEntry{}
	}

	r.middlePane.SetSelectedEntryFromString(filepath.Base(r.sel))

	// FIXME: Generic bounds checking across all panes in this function
	if r.middlePane.selectedEntry >= len(r.middlePane.entries) {
		if len(r.middlePane.entries) > 0 {
			r.sel = r.middlePane.GetSelectedEntryFromIndex(len(r.middlePane.entries) - 1)
			r.middlePane.SetSelectedEntryFromString(filepath.Base(r.sel)) // Duplicated from above...
		}
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

func (r *Ranger) GoLeft() {
	if filepath.Dir(r.wd) == r.wd {
		return
	}

	r.sel = r.wd
	r.wd = filepath.Dir(r.wd)
}

func (r *Ranger) GoRight() {
	if len(r.middlePane.entries) <= 0 {
		return
	}

	fi, err := os.Stat(r.sel)
	if err != nil {
		return
	}

	if !fi.IsDir() {
		return
	}

	/*	rightFiles, _ := os.ReadDir(r.sel)
		if len(rightFiles) <= 0 {
			return
		}*/

	r.wd = r.sel
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

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ranger.topPane, 1, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(ranger.leftPane, 0, 1, false).
			AddItem(ranger.middlePane, 0, 2, false).
			AddItem(ranger.rightPane, 0, 2, false), 0, 1, false).
		AddItem(ranger.bottomPane, 1, 0, false)

	pages := tview.NewPages().
		AddPage("flex", flex, true, true)

	app.SetMouseCapture(func(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
		wasMovementKey := true

		switch event.Buttons() {
			case tcell.WheelLeft:
				ranger.GoLeft()
			case tcell.WheelRight:
				ranger.GoRight()
			case tcell.WheelUp:
				ranger.GoUp()
			case tcell.WheelDown:
				ranger.GoDown()
			default:
				wasMovementKey = false
		}

		if wasMovementKey {
			if !(event.Buttons() == tcell.WheelLeft) {
				ranger.history.AddToHistory(ranger.sel)
			}

			ranger.historyMoment = ranger.sel
			ranger.UpdatePanes()
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

		if event.Key() == tcell.KeyF1 {
//			cmd := exec.Command("nano", ranger.sel)
			cmd := exec.Command("nvim", ranger.sel)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			app.Suspend(func() {
				if err := cmd.Run(); err != nil {
					log.Fatal(err)
				}
			})

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
			ranger.ToggleSelection(ranger.sel)
			ranger.historyMoment = strings.Join(ranger.selected, ", ")
			ranger.GoDown()
		} else if event.Key() == tcell.KeyHome || event.Rune() == 'g' {
//			ranger.sel = ranger.middlePane.GetSelectedEntryFromIndex(0)
			ranger.sel = filepath.Join(ranger.wd, ranger.middlePane.GetSelectedEntryFromIndex(0))
		} else if event.Key() == tcell.KeyEnd || event.Rune() == 'G' {
//			ranger.sel = ranger.middlePane.GetSelectedEntryFromIndex(len(ranger.middlePane.entries) - 1)
			ranger.sel = filepath.Join(ranger.wd, ranger.middlePane.GetSelectedEntryFromIndex(len(ranger.middlePane.entries) - 1))
		} else if event.Rune() == 'M' {
//			ranger.sel = ranger.middlePane.GetSelectedEntryFromIndex((len(ranger.middlePane.entries) - 1) / 2)
			ranger.sel = filepath.Join(ranger.wd, ranger.middlePane.GetSelectedEntryFromIndex((len(ranger.middlePane.entries) - 1) / 2))
		} else {
			wasMovementKey = false
		}

		if wasMovementKey {
			if !(event.Key() == tcell.KeyLeft || event.Rune() == 'h') {
				ranger.history.AddToHistory(ranger.sel)
			}

			ranger.historyMoment = ranger.sel
			ranger.UpdatePanes()
			return nil
		}

		if event.Rune() == 'A' {
			for _, e := range ranger.middlePane.entries {
				ranger.ToggleSelection(filepath.Join(ranger.wd, e.Name()))
			}
			return nil
		} else if event.Rune() == 'D' {
			ranger.selected = []string{}
			ranger.yankSelected = []string{}
			ranger.historyMoment = "Deselected!"
		} else if event.Rune() == 'a' {
			fileToRename := ranger.sel

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
					os.Rename(fileToRename, filepath.Join(filepath.Dir(fileToRename), inputField.GetText()))
					ranger.history.RemoveFromHistory(fileToRename)

					ranger.UpdatePanes()
					ranger.sel = filepath.Join(ranger.wd, ranger.middlePane.GetSelectedEntryFromIndex(ranger.middlePane.selectedEntry))
					ranger.historyMoment = ranger.sel

					pages.RemovePage("inputfield")
					return
				}
			})

			inputField.SetBorder(true)

			pages.AddPage("inputfield", modal(inputField, 58, 3), true, true)
			app.SetFocus(inputField)
			return nil
		} else if event.Rune() == 'y' {
			ranger.yankType = "copy"
			ranger.yankSelected = ranger.selected
			ranger.historyMoment = "Yank!"
			return nil
		} else if event.Rune() == 'd' {
			ranger.yankType = "cut"
			ranger.yankSelected = ranger.selected
			ranger.historyMoment = "Cut!"
			return nil
		} else if event.Rune() == 'p' {
			if ranger.yankType == "copy" {
				for _, e := range ranger.yankSelected {
					fi, err := os.Stat(e)
					if err != nil {
						continue
					}

//					newPath := filepath.Join(ranger.sel, filepath.Base(e))
					newPath := filepath.Join(ranger.wd, filepath.Base(e))
					if fi.IsDir() {
//						newPath := filepath.Join(ranger.sel, filepath.Base(e))
						err := os.Mkdir(newPath, 0755)
						if err != nil {
							// TODO: We need an error log we can scroll through
							ranger.historyMoment = newPath
						}
//						ranger.historyMoment = ranger.sel
						ranger.historyMoment = ranger.wd

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
			ranger.yankSelected = []string{}
			ranger.selected = []string{}

			ranger.UpdatePanes()
			ranger.sel = filepath.Join(ranger.wd, ranger.middlePane.GetSelectedEntryFromIndex(ranger.middlePane.selectedEntry))

			ranger.historyMoment = "Paste! (ranger.sel = " + ranger.sel + ")"

			// TODO: Fix properly?
			// We do this twice to get rid of a bug where you'd paste a file and it would show up on both the middle and right pane
			// Probably because r.sel wasn't being updated properly prior to ranger.UpdatePanes(), but is after it so we run it twice
			ranger.UpdatePanes()
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

			if len(ranger.selected) <= 0 {
				fileToDelete = ranger.sel
				modal.SetText("Delete " + filepath.Base(fileToDelete) + " ?")
			} else {
				modal.SetText("Delete " + strconv.Itoa(len(ranger.selected)) + " selected files?")
			}

			modal.
				AddButtons([]string{"Yes", "No"}).
				SetFocus(1). // Default is "No"
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					pages.RemovePage("modal")
					if buttonLabel != "Yes" {
						return
					}

					if len(ranger.selected) <= 0 {
						err := os.RemoveAll(fileToDelete)
						if err != nil {
							// TODO: We need an error log we can scroll through
							ranger.historyMoment = "Failed to delete!"
							return
						}
						ranger.history.RemoveFromHistory(fileToDelete)
						ranger.historyMoment = "Deleted " + fileToDelete
					} else {
						for _, filePath := range ranger.selected {
							err := os.RemoveAll(filePath)
							if err != nil {
								// TODO: We need an error log we can scroll through
								continue
							}
							ranger.history.RemoveFromHistory(filePath)
						}

						ranger.historyMoment = "Deleted " + strings.Join(ranger.selected, ", ")
					}

					ranger.selected = []string{}

					ranger.UpdatePanes()
					ranger.sel = filepath.Join(ranger.wd, ranger.middlePane.GetSelectedEntryFromIndex(ranger.middlePane.selectedEntry))
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
