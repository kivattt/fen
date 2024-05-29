package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kivattt/getopt"
	"github.com/rivo/tview"

	dirCopy "github.com/otiai10/copy"
)

const version = "v0.0.0-indev"

func main() {
	userConfigDir, err := os.UserConfigDir()
	configFilenamePath := ""
	if err == nil {
		configFilenamePath = filepath.Join(userConfigDir, "fen/fenrc.json")
	}

	v := flag.Bool("version", false, "output version information and exit")
	h := flag.Bool("help", false, "display this help and exit")
	noWrite := flag.Bool("no-write", false, "safe mode, no file write operations will be performed")
	configFilename := flag.String("config", configFilenamePath, "use configuration file")

	getopt.CommandLine.SetOutput(os.Stdout)
	getopt.CommandLine.Init("fen", flag.ExitOnError)
	getopt.Aliases(
		"v", "version",
		"h", "help",
	)

	err = getopt.CommandLine.Parse(os.Args[1:])
	if err != nil {
		os.Exit(0)
	}

	if *v {
		fmt.Println("fen " + version)
		os.Exit(0)
	}

	if *h {
		fmt.Println("Usage: " + filepath.Base(os.Args[0]) + " [OPTION] [PATH]")
		fmt.Println("Terminal file manager")
		fmt.Println()
		getopt.PrintDefaults()
		os.Exit(0)
	}

	workingDirectory, err := filepath.Abs(getopt.CommandLine.Arg(0))
	if err != nil {
		log.Fatalf(err.Error())
	}

	if workingDirectory == "" {
		workingDirectory, err := os.Getwd()

		// os.Getwd() will error if the working directory doesn't exist
		if err != nil {
			// https://cs.opensource.google/go/go/+/refs/tags/go1.22.1:src/os/getwd.go;l=23
			if runtime.GOOS == "windows" || runtime.GOOS == "plan9" {
				log.Fatalf("Unable to determine current working directory")
			}

			workingDirectory = os.Getenv("PWD")
			if workingDirectory == "" {
				log.Fatalf("PWD environment variable empty")
			}
		}
	}

	var fen Fen
	err = fen.ReadConfig(*configFilename)
	fen.config.NoWrite = fen.config.NoWrite || *noWrite // Command-line flag is higher priority than config
	if err != nil {
		fmt.Println("Invalid config " + *configFilename)
		log.Fatal(err)
	}

	err = fen.Init(workingDirectory)
	if err != nil {
		log.Fatal(err)
	}

	app := tview.NewApplication()

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(fen.topPane, 1, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(fen.leftPane, 0, 1, false).
			AddItem(fen.middlePane, 0, 3, false).
			AddItem(fen.rightPane, 0, 3, false), 0, 1, false).
		AddItem(fen.bottomPane, 1, 0, false)

	pages := tview.NewPages().
		AddPage("flex", flex, true, true)

	bottomRight := func(p tview.Primitive, width, height int) tview.Primitive {
		/*return tview.NewGrid().
		SetRows(30, 30).
		SetColumns(30, 30).
		AddItem(p, 1, 1, 1, 1, 5, 10, false)*/

		// Works, although no auto-resizing
		return tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(p, height, 1, true), width, 1, true)
	}

	pages.AddPage("fileproperties", bottomRight(fen.fileProperties, 64, 20), true, true)

	app.SetMouseCapture(func(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
		wasMovementKey := true

		// Required to prevent a nil dereference crash
		if event == nil {
			return nil, action
		}

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

			fen.bottomBarText = fen.sel
			fen.UpdatePanes()
			return nil, action
		}

		return event, action
	})

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if pages.HasPage("deletemodal") || pages.HasPage("inputfield") || pages.HasPage("newfilemodal") || pages.HasPage("searchbox") {
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
			fen.bottomBarText = strings.Join(fen.selected, ", ")
			fen.GoDown()
		} else if event.Key() == tcell.KeyHome || event.Rune() == 'g' {
			fen.GoTop()
		} else if event.Key() == tcell.KeyEnd || event.Rune() == 'G' {
			fen.GoBottom()
		} else if event.Rune() == 'M' {
			fen.GoMiddle()
		} else if event.Rune() == 'H' {
			fen.GoTopScreen()
		} else if event.Rune() == 'L' {
			fen.GoBottomScreen()
		} else if event.Key() == tcell.KeyPgUp {
			fen.PageUp()
		} else if event.Key() == tcell.KeyPgDn {
			fen.PageDown()
		} else {
			wasMovementKey = false
		}

		if wasMovementKey {
			if !(event.Key() == tcell.KeyLeft || event.Rune() == 'h') {
				fen.history.AddToHistory(fen.sel)
			}

			fen.bottomBarText = fen.sel
			fen.UpdatePanes()
			return nil
		}

		if event.Rune() == '/' {
			flex.RemoveItem(fen.bottomPane)
			inputField := tview.NewInputField().
				SetLabel("/").
				SetPlaceholder("<search>")

			inputField.SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
				return lastChar != '/' // FIXME: Hack to prevent the slash appearing in the search inputfield by just disallowing them
			})

			inputField.
				SetDoneFunc(func(key tcell.Key) {
					pages.RemovePage("searchbox")
					flex.AddItem(fen.bottomPane, 1, 0, false)

					if key == tcell.KeyEscape {
						return
					}

					err := fen.GoSearchFirstMatch(inputField.GetText())
					if err != nil {
						// FIXME: We need a log window or something
						fen.bottomBarText = "Nothing found"
					} else {
						// Same code as the wasMovementKey check
						fen.history.AddToHistory(fen.sel)
						fen.bottomBarText = fen.sel
						fen.UpdatePanes()
					}
				})

			pages.AddPage("searchbox", inputField, true, true)
			/*flex.AddItem(inputField, 1, 0, false)
			app.SetFocus(inputField)
			app.SetInputCapture(nil)*/
		} else if event.Rune() == 'A' {
			for _, e := range fen.middlePane.entries {
				fen.ToggleSelection(filepath.Join(fen.wd, e.Name()))
			}
			fen.DisableSelectingWithV()
			return nil
		} else if event.Rune() == 'D' {
			fen.selected = []string{}
			fen.yankSelected = []string{}
			fen.bottomBarText = "Deselected and un-yanked!"
			fen.DisableSelectingWithV()
			return nil
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
				SetLabel("Rename: ").
				SetText(filepath.Base(fileToRename)).
				SetFieldWidth(45)

			inputField.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					pages.RemovePage("inputfield")
					return
				} else if key == tcell.KeyEnter {
					if !fen.config.NoWrite {
						newPath := filepath.Join(filepath.Dir(fileToRename), inputField.GetText())
						os.Rename(fileToRename, newPath)

						fen.RemoveFromSelectedAndYankSelected(fileToRename)
						fen.history.RemoveFromHistory(fileToRename)
						fen.history.AddToHistory(newPath)
						fen.sel = newPath

						fen.UpdatePanes()
						//fen.bottomBarText = fen.sel
					}

					pages.RemovePage("inputfield")
					return
				}
			})

			inputField.SetBorder(true)
			inputField.SetBorderColor(tcell.ColorDefault)
			inputField.SetTitleColor(tcell.ColorDefault)

			pages.AddPage("inputfield", modal(inputField, 58, 3), true, true)
			app.SetFocus(inputField)
			return nil
		} else if event.Rune() == 'n' || event.Rune() == 'N' {
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
				SetLabel("Name: ").
				SetFieldWidth(45)

			inputField.SetBorderColor(tcell.ColorDefault)
			inputField.SetTitleColor(tcell.ColorDefault)

			if event.Rune() == 'n' {
				inputField.SetTitle("New file: ")
			} else if event.Rune() == 'N' {
				inputField.SetTitle("New folder: ")
			}

			inputField.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					pages.RemovePage("newfilemodal")
					return
				} else if key == tcell.KeyEnter {
					if !fen.config.NoWrite {
						if event.Rune() == 'n' {
							os.Create(filepath.Join(fen.wd, inputField.GetText()))
						} else if event.Rune() == 'N' {
							os.Mkdir(filepath.Join(fen.wd, inputField.GetText()), 0755)
						}
						fen.UpdatePanes()
					}

					pages.RemovePage("newfilemodal")
					return
				}
			})

			inputField.SetBorder(true)

			pages.AddPage("newfilemodal", modal(inputField, 58, 3), true, true)
			app.SetFocus(inputField)
			return nil
		} else if event.Rune() == 'y' {
			fen.yankType = "copy"
			if len(fen.selected) <= 0 {
				fen.yankSelected = []string{fen.sel}
			} else {
				fen.yankSelected = fen.selected
			}
			fen.bottomBarText = "Yank!"
			return nil
		} else if event.Rune() == 'd' {
			fen.yankType = "cut"
			if len(fen.selected) <= 0 {
				fen.yankSelected = []string{fen.sel}
			} else {
				fen.yankSelected = fen.selected
			}
			fen.bottomBarText = "Cut!"
			return nil
		} else if event.Rune() == 'z' || event.Key() == tcell.KeyBackspace {
			fen.dontShowHiddenFiles = !fen.dontShowHiddenFiles
			fen.DisableSelectingWithV() // FIXME: We shouldn't disable it, but fixing it to not be buggy would be annoying
			fen.UpdatePanes()
			fen.history.AddToHistory(fen.sel)
			return nil
		} else if event.Rune() == 'p' {
			if len(fen.yankSelected) <= 0 {
				fen.bottomBarText = "Nothing to paste..." // TODO: We need a log we can scroll through
				return nil
			}

			if fen.config.NoWrite {
				return nil // TODO: Need a msg showing nothing was done in a log (we can scroll through)
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
							fen.bottomBarText = newPath
						}
						//						fen.bottomBarText = fen.sel
						fen.bottomBarText = fen.wd

						err = dirCopy.Copy(e, newPath)
						if err != nil {
							// TODO: We need an error log we can scroll through
							continue
						}
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
			} else if fen.yankType == "cut" {
				for _, e := range fen.yankSelected {
					// Just to make sure the file exists?
					_, err := os.Stat(e)
					if err != nil {
						continue
					}

					newPath := filepath.Join(fen.wd, filepath.Base(e))
					err = os.Rename(e, newPath)
					if err != nil {
						// TODO: We need an error log we can scroll through
						continue
					}
				}
			} else {
				panic("yankType was not \"copy\" or \"cut\"")
			}

			// Reset selection after paste
			fen.yankSelected = []string{}
			fen.selected = []string{}

			fen.UpdatePanes()
			//			fen.bottomBarText = "Paste! (fen.sel = " + fen.sel + ")"

			return nil
		} else if event.Rune() == 'V' {
			fen.ToggleSelectingWithV()
			fen.UpdatePanes()
			return nil
		} else if event.Rune() == '?' {
			fen.fileProperties.visible = !fen.fileProperties.visible
			fen.UpdatePanes()

			return nil
		}

		if event.Key() == tcell.KeyDelete {
			modal := tview.NewModal()
			modal.SetBorderColor(tcell.ColorDefault)
			modal.SetTitleColor(tcell.ColorDefault)
			modal.SetTextColor(tcell.ColorDefault)
			modal.SetButtonTextColor(tcell.ColorDefault)
			modal.SetButtonBackgroundColor(tcell.ColorBlack)

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
					pages.RemovePage("deletemodal")
					if buttonLabel != "Yes" {
						return
					}

					if fen.config.NoWrite {
						return
					}

					if len(fen.selected) <= 0 {
						err := os.RemoveAll(fileToDelete)
						if err != nil {
							// TODO: We need an error log we can scroll through
							fen.bottomBarText = "Failed to delete!"
							return
						}
						fen.history.RemoveFromHistory(fileToDelete)
						fen.bottomBarText = "Deleted " + fileToDelete
					} else {
						for _, filePath := range fen.selected {
							err := os.RemoveAll(filePath)
							if err != nil {
								// TODO: We need an error log we can scroll through
								continue
							}
							fen.history.RemoveFromHistory(filePath)
						}

						fen.bottomBarText = "Deleted " + strings.Join(fen.selected, ", ")
					}

					fen.selected = []string{}

					// FIXME: CURSED
					// We need to update the middlePane entries for GoDown() and GoUp() to work properly, atleast when deleting the bottom entry
					fen.middlePane.SetEntries(fen.wd)
					fen.GoDown()
					fen.GoUp()
					fen.UpdatePanes()
				})

			pages.AddPage("deletemodal", modal, true, true)
			app.SetFocus(modal)
			return nil
		}

		return event
	})

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		log.Fatal(err)
	}
}
