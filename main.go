package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	//	"runtime/pprof"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kivattt/getopt"
	"github.com/rivo/tview"
)

const version = "v1.6.7"

func main() {
	//	f, _ := os.Create("profile.prof")
	//	pprof.StartCPUProfile(f)
	//	defer pprof.StopCPUProfile()

	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault

	userConfigDir, err := os.UserConfigDir()
	defaultConfigFilenamePath := ""
	if err == nil {
		defaultConfigFilenamePath = filepath.Join(userConfigDir, "fen", "config.lua")
	}

	defaultConfigValues := NewConfigDefaultValues()

	// When adding new flags, make sure to duplicate the name when we check flagPassed lower in this file
	v := flag.Bool("version", false, "output version information and exit")
	h := flag.Bool("help", false, "display this help and exit")
	uiBorders := flag.Bool("ui-borders", defaultConfigValues.UiBorders, "enable UI borders")
	mouse := flag.Bool("mouse", defaultConfigValues.Mouse, "enable mouse events")
	noWrite := flag.Bool("no-write", defaultConfigValues.NoWrite, "safe mode, no file write operations will be performed")
	hiddenFiles := flag.Bool("hidden-files", defaultConfigValues.HiddenFiles, "")
	foldersFirst := flag.Bool("folders-first", defaultConfigValues.FoldersFirst, "always show folders at the top")
	printPathOnOpen := flag.Bool("print-path-on-open", defaultConfigValues.PrintPathOnOpen, "output file path(s) and exit when opening file(s)")
	printFolderOnExit := flag.Bool("print-folder-on-exit", false, "output the current working folder in fen on exit")
	allowTerminalTitle := flag.Bool("terminal-title", defaultConfigValues.TerminalTitle, "change terminal title to 'fen "+version+"' while open")
	showHelpText := flag.Bool("show-help-text", defaultConfigValues.ShowHelpText, "show the 'For help: ...' text")
	showHostname := flag.Bool("show-hostname", defaultConfigValues.ShowHostname, "show username@hostname in the top-left")

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		userHomeDir = "the home directory"
	}
	showHomePathAsTilde := flag.Bool("show-home-path-as-tilde", defaultConfigValues.ShowHomePathAsTilde, "replaces "+userHomeDir+" with the ~ symbol (not on Windows)")
	selectPaths := flag.Bool("select", false, "select PATHS")

	configFilename := flag.String("config", defaultConfigFilenamePath, "use configuration file")
	sortBy := flag.String("sort-by", defaultConfigValues.SortBy, "sort files ("+strings.Join(ValidSortByValues[:], ", ")+")")
	sortReverse := flag.Bool("sort-reverse", defaultConfigValues.SortReverse, "reverse sort")

	getopt.CommandLine.SetOutput(os.Stdout)
	getopt.CommandLine.Init("fen", flag.ExitOnError)
	getopt.Aliases(
		"v", "version",
		"h", "help",
		//		"s", "select", // This doesn't work for some reason
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
		fmt.Println("Usage: " + filepath.Base(os.Args[0]) + " [OPTIONS] [FILES]")
		fmt.Println("Terminal file manager")
		fmt.Println()
		getopt.PrintDefaults()
		os.Exit(0)
	}

	path, err := filepath.Abs(getopt.CommandLine.Arg(0))

	if path == "" || err != nil {
		path, err = os.Getwd()

		// os.Getwd() will error if the working directory doesn't exist
		if err != nil {
			// https://cs.opensource.google/go/go/+/refs/tags/go1.22.1:src/os/getwd.go;l=23
			if runtime.GOOS == "windows" || runtime.GOOS == "plan9" {
				log.Fatalf("Unable to determine current working directory")
			}

			path = os.Getenv("PWD")
			if path == "" {
				log.Fatalf("PWD environment variable empty")
			}
		}
	}

	var fen Fen
	// Presumably a different value passed by command-line argument
	if *configFilename != defaultConfigFilenamePath {
		_, err := os.Stat(*configFilename)
		if err != nil {
			log.Fatal("Could not find file: " + *configFilename)
		}
	}
	err = fen.ReadConfig(*configFilename)

	if !fen.config.NoWrite {
		os.Mkdir(filepath.Join(userConfigDir, "fen"), 0o775)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		// Hacky, but gets the job done
		if !strings.HasSuffix(err.Error(), "config files can only be Lua.\n") {
			fmt.Println("Invalid config '" + *configFilename + "', exiting")
		} else if !fen.config.NoWrite {
			fmt.Print("Generate config.lua from fenrc.json file? (This will not erase anything) [y/N] ")
			reader := bufio.NewReader(os.Stdin)
			confirmation, err := reader.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}

			if strings.ToLower(strings.TrimSpace(confirmation)) == "y" {
				oldConfigPath := filepath.Join(filepath.Dir(*configFilename), "fenrc.json")
				newConfigPath := filepath.Join(filepath.Dir(*configFilename), "config.lua")
				fmt.Print("Generate new config file: " + newConfigPath + " ? [y/N] ")
				reader := bufio.NewReader(os.Stdin)
				confirmation, err := reader.ReadString('\n')
				if err != nil {
					log.Fatal(err)
				}

				if strings.ToLower(strings.TrimSpace(confirmation)) == "y" {
					err = GenerateLuaConfigFromOldJSONConfig(oldConfigPath, newConfigPath, &fen)
					if err != nil {
						log.Fatal(err)
					}
					fmt.Println("Done! Your new config file: " + newConfigPath)
				} else {
					fmt.Println("Nothing done")
				}
			} else {
				fmt.Println("Nothing done")
			}
		}
		os.Exit(1)
	}

	// We have to check *selectPaths before flag.Parse()
	if *selectPaths {
		for _, arg := range getopt.CommandLine.Args() {
			pathAbsolute, err := filepath.Abs(arg)
			if err == nil { // TODO: Add an error msg to the log when an invalid path was specified
				fen.EnableSelection(pathAbsolute)
			}
		}
	}

	flag.Parse()
	flagPassed := func(name string) bool {
		found := false
		flag.Visit(func(f *flag.Flag) {
			if f.Name == name {
				found = true
			}
		})
		return found
	}

	// Maybe clean this up at some point
	if flagPassed("ui-borders") {
		fen.config.UiBorders = *uiBorders
	}
	if flagPassed("mouse") {
		fen.config.Mouse = *mouse
	}
	if flagPassed("no-write") {
		fen.config.NoWrite = *noWrite
	}
	if flagPassed("hidden-files") {
		fen.config.HiddenFiles = *hiddenFiles
	}
	if flagPassed("folders-first") {
		fen.config.FoldersFirst = *foldersFirst
	}
	if flagPassed("print-path-on-open") {
		fen.config.PrintPathOnOpen = *printPathOnOpen
	}
	if flagPassed("terminal-title") {
		fen.config.TerminalTitle = *allowTerminalTitle
	}
	if flagPassed("show-help-text") {
		fen.config.ShowHelpText = *showHelpText
	}
	if flagPassed("show-hostname") {
		fen.config.ShowHostname = *showHostname
	}

	fen.effectiveShowHomePathAsTilde = fen.config.ShowHomePathAsTilde
	if flagPassed("show-home-path-as-tilde") {
		fen.effectiveShowHomePathAsTilde = *showHomePathAsTilde
		fen.config.ShowHomePathAsTilde = *showHomePathAsTilde
	}
	if flagPassed("sort-by") {
		fen.config.SortBy = *sortBy
	}
	if flagPassed("sort-reverse") {
		fen.config.SortReverse = *sortReverse
	}

	app := tview.NewApplication()

	helpScreen := NewHelpScreen(&fen)

	err = fen.Init(path, app, &helpScreen.visible)
	if err != nil {
		log.Fatal(err)
	}

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(fen.topBar, 1, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(fen.leftPane, 0, 1, false).
			AddItem(fen.middlePane, 0, 3, false).
			AddItem(fen.rightPane, 0, 3, false), 0, 1, false).
		AddItem(fen.bottomBar, 1, 0, false)

	pages := tview.NewPages().
		AddPage("flex", flex, true, true)

	centered := func(p tview.Primitive, height int) tview.Primitive {
		return tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(p, height, 1, true).
				AddItem(nil, 0, 1, false), 0, 2, true).
			AddItem(nil, 0, 1, false)
	}

	helpScreen.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyF1 || event.Rune() == '?' || event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
			helpScreen.visible = false
			pages.RemovePage("helpscreen")
			fen.ShowFilepanes()
			return nil
		}
		return event
	})

	lastWheelUpTime := time.Now()
	lastWheelDownTime := time.Now()
	app.SetMouseCapture(func(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
		if pages.HasPage("deletemodal") || pages.HasPage("inputfield") || pages.HasPage("newfilemodal") || pages.HasPage("searchbox") || pages.HasPage("openwith") || pages.HasPage("forcequitmodal") || pages.HasPage("helpscreen") || pages.HasPage("gotofolder") {
			// Since `return nil, action` redraws the screen for some reason,
			// we have to manually pass through mouse movement events so the screen won't flicker when you move your mouse
			if action == tview.MouseMove {
				return event, action
			}

			return nil, action
		}

		// Required to prevent a nil dereference crash
		if event == nil {
			return nil, action
		}

		if action == tview.MouseMove {
			if !fen.config.ShowHomePathAsTilde || runtime.GOOS == "windows" {
				return event, action
			}

			_, mouseY := event.Position()
			if mouseY == 0 {
				if fen.effectiveShowHomePathAsTilde {
					fen.effectiveShowHomePathAsTilde = false
					return nil, action
				}
			} else {
				if !fen.effectiveShowHomePathAsTilde {
					fen.effectiveShowHomePathAsTilde = true
					return nil, action
				}
			}
			return event, action
		}

		switch event.Buttons() {
		case tcell.Button1:
			x, y, w, h := fen.middlePane.GetInnerRect()
			mouseX, mouseY := event.Position()

			if mouseY < y || mouseY > h { // We don't check > y+h so clicking the bottom row of the screen is ignored
				break
			}

			if mouseX < x {
				fen.GoLeft()
			} else if mouseX > x+w {
				fen.GoRight(app, "")
			} else {
				fen.GoIndex(fen.middlePane.ClampEntryIndex(mouseY - y + fen.middlePane.GetTopScreenEntryIndex()))
				// We need to call this here even though its being called at the end?
				// Probably because of history shenanigans that are solved by SetSelectedEntryFromString() which is at some point called in UpdatePanes()
				fen.UpdatePanes(false)
			}
		case tcell.WheelLeft:
			fen.GoLeft()
		case tcell.WheelRight:
			fen.GoRight(app, "")
		case tcell.WheelUp:
			if time.Since(lastWheelUpTime) > time.Duration(30*time.Millisecond) {
				fen.GoUp()
			} else {
				fen.GoUp(fen.config.ScrollSpeed)
			}
			lastWheelUpTime = time.Now()
		case tcell.WheelDown:
			if time.Since(lastWheelDownTime) > time.Duration(30*time.Millisecond) {
				fen.GoDown()
			} else {
				fen.GoDown(fen.config.ScrollSpeed)
			}
			lastWheelDownTime = time.Now()
		default:
			return nil, action
		}

		if event.Buttons() != tcell.WheelLeft {
			fen.history.AddToHistory(fen.sel)
		}

		fen.UpdatePanes(false)
		return nil, action
	})

	enterWillSelectAutoCompleteInGotoFolder := false

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if pages.HasPage("deletemodal") || pages.HasPage("inputfield") || pages.HasPage("newfilemodal") || pages.HasPage("searchbox") || pages.HasPage("openwith") || pages.HasPage("forcequitmodal") || pages.HasPage("helpscreen") || pages.HasPage("gotofolder") {
			return event
		}

		if event.Rune() == 'q' {
			fen.fileOperationsHandler.workCountMutex.Lock()
			if fen.fileOperationsHandler.workCount <= 0 {
				fen.fileOperationsHandler.workCountMutex.Unlock()
				app.Stop()
				return nil
			}

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

			modal.SetText(strconv.Itoa(fen.fileOperationsHandler.workCount) + " file operations in progress.\nQuitting can corrupt your files!")
			fen.fileOperationsHandler.workCountMutex.Unlock()
			modal.
				AddButtons([]string{"Force quit", "Cancel"}).
				SetFocus(1). // Default is "No"
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					pages.RemovePage("forcequitmodal")

					if buttonIndex != 0 {
						return
					}

					app.Stop()
				})
			modal.SetBorder(true)

			modal.Box.SetBackgroundColor(tcell.ColorBlack) // This sets the border background color
			modal.SetBackgroundColor(tcell.ColorBlack)

			modal.SetButtonBackgroundColor(tcell.ColorDefault)
			modal.SetButtonTextColor(tcell.ColorRed)

			pages.AddPage("forcequitmodal", modal, true, true)
			app.SetFocus(modal)
			return nil
		}

		wasMovementKey := true
		if event.Key() == tcell.KeyLeft || event.Rune() == 'h' {
			fen.GoLeft()
		} else if event.Key() == tcell.KeyRight || event.Rune() == 'l' || event.Key() == tcell.KeyEnter {
			fen.GoRight(app, "")
		} else if event.Key() == tcell.KeyCtrlSpace || event.Key() == tcell.KeyCtrlN {
			inputField := tview.NewInputField().
				SetLabel(" Open with: ").
				SetFieldWidth(-1) // Special feature of my tview fork, github.com/kivattt/tview

			inputField.SetTitleColor(tcell.ColorDefault)
			inputField.SetFieldBackgroundColor(tcell.ColorGray)
			inputField.SetFieldTextColor(tcell.ColorBlack)
			inputField.SetBackgroundColor(tcell.ColorBlack)

			inputField.SetLabelStyle(tcell.StyleDefault.Background(tcell.ColorBlack)) // This has to be before the .SetLabelColor
			inputField.SetLabelColor(tcell.NewRGBColor(0, 255, 0))                    // Green

			programs, descriptions := ProgramsAndDescriptionsForFile(&fen)
			programsList := NewOpenWithList(&programs, &descriptions)

			inputField.SetPlaceholderStyle(tcell.StyleDefault.Background(tcell.ColorGray).Dim(true))
			inputFieldHeight := 2
			if len(programs) > 0 {
				inputField.SetPlaceholder(programs[0])
			} else {
				inputFieldHeight = 1
			}

			inputField.SetDoneFunc(func(key tcell.Key) {
				pages.RemovePage("openwith")

				if key == tcell.KeyEscape {
					return
				}

				programNameToUse := inputField.GetText()
				if programNameToUse == "" {
					if len(programs) > 0 {
						programNameToUse = programs[0]
					}
				}
				fen.GoRight(app, programNameToUse)
			})

			flex := tview.NewFlex().
				AddItem(inputField, inputFieldHeight, 1, true).SetDirection(tview.FlexRow).
				AddItem(programsList, len(programs), 1, false)

			flex.SetBorder(true)
			flex.SetBorderStyle(tcell.StyleDefault.Background(tcell.ColorBlack))

			pages.AddPage("openwith", centered(flex, inputFieldHeight+2+len(programs)), true, true)
		} else if event.Key() == tcell.KeyUp || event.Rune() == 'k' {
			fen.GoUp()
		} else if event.Key() == tcell.KeyDown || event.Rune() == 'j' {
			fen.GoDown()
		} else if event.Rune() == ' ' {
			fen.ToggleSelection(fen.sel)
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

			fen.UpdatePanes(false)
			return nil
		}

		if event.Rune() == '/' || event.Key() == tcell.KeyCtrlF {
			inputField := tview.NewInputField().
				SetLabel(" Search: ").
				SetPlaceholder("case-insensitive").
				SetFieldWidth(-1) // Special feature of my tview fork, github.com/kivattt/tview

			inputField.SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
				return lastChar != '/' // FIXME: Hack to prevent the slash appearing in the search inputfield by just disallowing them
			})

			inputField.
				SetDoneFunc(func(key tcell.Key) {
					pages.RemovePage("searchbox")

					if key == tcell.KeyEscape {
						return
					}

					err := fen.GoSearchFirstMatch(inputField.GetText())
					if err != nil {
						// FIXME: We need a log window or something
						fen.bottomBar.TemporarilyShowTextInstead("Nothing found")
					} else {
						// Same code as the wasMovementKey check
						fen.history.AddToHistory(fen.sel)
						fen.UpdatePanes(false)
					}
				})

			inputField.SetBorder(true)
			inputField.SetBorderStyle(tcell.StyleDefault.Background(tcell.ColorBlack))
			inputField.SetTitleColor(tcell.ColorDefault)
			inputField.SetFieldBackgroundColor(tcell.ColorGray)
			inputField.SetFieldTextColor(tcell.ColorBlack)
			inputField.SetLabelStyle(tcell.StyleDefault.Background(tcell.ColorBlack))
			inputField.SetLabelColor(tcell.NewRGBColor(0, 255, 0)) // Green
			inputField.SetPlaceholderStyle(tcell.StyleDefault.Background(tcell.ColorGray).Dim(true))

			pages.AddPage("searchbox", centered(inputField, 3), true, true)
		} else if event.Rune() == 'A' {
			for _, e := range fen.middlePane.entries.Load().([]os.DirEntry) {
				fen.ToggleSelection(filepath.Join(fen.wd, e.Name()))
			}
			fen.DisableSelectingWithV()
			return nil
		} else if event.Rune() == 'D' {
			if len(fen.selected) > 0 || len(fen.yankSelected) > 0 {
				fen.selected = make(map[string]bool)
				fen.yankSelected = make(map[string]bool)
				fen.bottomBar.TemporarilyShowTextInstead("Deselected and un-yanked!")
			}

			fen.DisableSelectingWithV()
			return nil
		} else if event.Rune() == 'a' {
			fileToRename := fen.sel

			inputField := tview.NewInputField().
				SetLabel(" Rename: ").
				SetText(filepath.Base(fileToRename)).
				SetFieldWidth(-1) // Special feature of my tview fork, github.com/kivattt/tview

			inputField.SetDoneFunc(func(key tcell.Key) {

				if key == tcell.KeyEscape {
					pages.RemovePage("inputfield")
					return
				} else if key == tcell.KeyEnter {
					if !fen.config.NoWrite {
						newPath := filepath.Join(filepath.Dir(fileToRename), inputField.GetText())
						_, err := os.Stat(newPath)
						if err == nil {
							pages.RemovePage("inputfield")
							fen.bottomBar.TemporarilyShowTextInstead("Can't rename to an existing file")
							return
						}
						os.Rename(fileToRename, newPath)

						fen.RemoveFromSelectedAndYankSelected(fileToRename)
						fen.history.RemoveFromHistory(fileToRename)
						fen.sel = newPath
						fen.history.AddToHistory(fen.sel)

						fen.UpdatePanes(true)
					} else {
						fen.bottomBar.TemporarilyShowTextInstead("Can't rename in no-write mode")
					}

					pages.RemovePage("inputfield")
					return
				}
			})

			inputField.SetBorder(true)
			inputField.SetBorderStyle(tcell.StyleDefault.Background(tcell.ColorBlack))
			inputField.SetTitleColor(tcell.ColorDefault)
			inputField.SetFieldBackgroundColor(tcell.ColorGray)
			inputField.SetFieldTextColor(tcell.ColorBlack)
			inputField.SetLabelStyle(tcell.StyleDefault.Background(tcell.ColorBlack)) // This has to be before the .SetLabelColor
			inputField.SetLabelColor(tcell.NewRGBColor(0, 255, 0))                    // Green

			pages.AddPage("inputfield", centered(inputField, 3), true, true)
			app.SetFocus(inputField)
			return nil
		} else if event.Rune() == 'n' || event.Rune() == 'N' {
			inputField := tview.NewInputField().
				SetFieldWidth(-1) // Special feature of my tview fork, github.com/kivattt/tview

			if event.Rune() == 'n' {
				inputField.SetLabel(" New file: ")
			} else if event.Rune() == 'N' {
				inputField.SetLabel(" New folder: ")
			}

			inputField.SetDoneFunc(func(key tcell.Key) {

				if key == tcell.KeyEscape {
					pages.RemovePage("newfilemodal")
					return
				} else if key == tcell.KeyEnter {
					_, err := os.Stat(filepath.Join(fen.wd, inputField.GetText())) // Here to make sure we don't overwrite a file when making a new one
					if !fen.config.NoWrite && err != nil {
						var createFileOrFolderErr error
						if event.Rune() == 'n' {
							_, createFileOrFolderErr = os.Create(filepath.Join(fen.wd, inputField.GetText()))
						} else if event.Rune() == 'N' {
							createFileOrFolderErr = os.Mkdir(filepath.Join(fen.wd, inputField.GetText()), 0775)
						}

						if createFileOrFolderErr == nil {
							fen.sel = filepath.Join(fen.wd, inputField.GetText())
						}
						fen.UpdatePanes(true)
					} else if fen.config.NoWrite {
						fen.bottomBar.TemporarilyShowTextInstead("Can't create new files in no-write mode")
					} else if err != nil {
						fen.bottomBar.TemporarilyShowTextInstead(err.Error())
					} else {
						fen.bottomBar.TemporarilyShowTextInstead("Can't create an existing file")
					}

					pages.RemovePage("newfilemodal")
					return
				}
			})

			inputField.SetBorder(true)
			inputField.SetBorderStyle(tcell.StyleDefault.Background(tcell.ColorBlack))
			inputField.SetTitleColor(tcell.ColorDefault)
			inputField.SetFieldBackgroundColor(tcell.ColorGray)
			inputField.SetFieldTextColor(tcell.ColorBlack)

			inputField.SetLabelStyle(tcell.StyleDefault.Background(tcell.ColorBlack)) // This has to be before the .SetLabelColor
			inputField.SetLabelColor(tcell.NewRGBColor(0, 255, 0))                    // Green

			pages.AddPage("newfilemodal", centered(inputField, 3), true, true)
			app.SetFocus(inputField)
			return nil
		} else if event.Rune() == 'y' {
			fen.yankType = "copy"
			if len(fen.selected) <= 0 {
				fen.yankSelected = map[string]bool{fen.sel: true}
			} else {
				// We have to do this to copy fen.selected, and not a reference to it
				fen.yankSelected = make(map[string]bool)
				for k, v := range fen.selected {
					fen.yankSelected[k] = v
				}
			}

			fen.bottomBar.TemporarilyShowTextInstead("Yank!")
			return nil
		} else if event.Rune() == 'd' {
			fen.yankType = "cut"
			if len(fen.selected) <= 0 {
				fen.yankSelected = map[string]bool{fen.sel: true}
			} else {
				// We have to do this to copy fen.selected, and not a reference to it
				fen.yankSelected = make(map[string]bool)
				for k, v := range fen.selected {
					fen.yankSelected[k] = v
				}
			}

			fen.bottomBar.TemporarilyShowTextInstead("Cut!")
			return nil
		} else if event.Rune() == 'z' || event.Key() == tcell.KeyBackspace {
			fen.config.HiddenFiles = !fen.config.HiddenFiles
			fen.DisableSelectingWithV() // FIXME: We shouldn't disable it, but fixing it to not be buggy would be annoying
			fen.UpdatePanes(true)
			fen.history.AddToHistory(fen.sel)
			return nil
		} else if event.Rune() == 'p' {
			if len(fen.yankSelected) <= 0 {
				fen.bottomBar.TemporarilyShowTextInstead("Nothing to paste...") // TODO: We need a log we can scroll through
				return nil
			}

			if fen.config.NoWrite {
				fen.bottomBar.TemporarilyShowTextInstead("Can't paste in no-write mode")
				return nil // TODO: Need a msg showing nothing was done in a log (we can scroll through)
			}

			if fen.yankType == "copy" {
				for e := range fen.yankSelected {
					newPath := FilePathUniqueNameIfAlreadyExists(filepath.Join(fen.wd, filepath.Base(e)))
					go fen.fileOperationsHandler.QueueOperation(FileOperation{operation: Copy, path: e, newPath: newPath})
				}
			} else if fen.yankType == "cut" {
				for e := range fen.yankSelected {
					newPath := FilePathUniqueNameIfAlreadyExists(filepath.Join(fen.wd, filepath.Base(e)))

					// If we're cutting, then pasting the file to the same location, don't actually do anything
					if e == filepath.Join(fen.wd, filepath.Base(e)) {
						continue
					}

					go fen.fileOperationsHandler.QueueOperation(FileOperation{operation: Rename, path: e, newPath: newPath})
				}
			} else {
				panic("yankType was not \"copy\" or \"cut\"")
			}

			// Reset selection after paste
			fen.yankSelected = make(map[string]bool)

			fen.selected = make(map[string]bool)

			fen.DisableSelectingWithV()

			fen.UpdatePanes(false)
			fen.bottomBar.TemporarilyShowTextInstead("Paste!")

			return nil
		} else if event.Rune() == 'V' {
			fen.ToggleSelectingWithV()
			fen.UpdatePanes(false)
			return nil
		} else if event.Key() == tcell.KeyEscape {
			fen.DisableSelectingWithV()
			fen.UpdatePanes(false)
		} else if event.Key() == tcell.KeyF1 || event.Rune() == '?' {
			helpScreen.visible = !helpScreen.visible
			if helpScreen.visible {
				pages.AddPage("helpscreen", helpScreen, true, true)
				fen.HideFilepanes()
			} else {
				pages.RemovePage("helpscreen")
				fen.ShowFilepanes()
			}
			return nil
		} else if event.Key() == tcell.KeyDelete || event.Rune() == 'x' {
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
				fileToDeleteInfo, _ := os.Stat(fileToDelete)
				// When the text wraps, color styling gets reset on line breaks. I have not found a good solution yet
				styleStr := StyleToStyleTagString(FileColor(fileToDeleteInfo, fileToDelete))
				modal.SetText("[red::d]Delete[-:-:-:-] " + styleStr + FilenameInvisibleCharactersAsCodeHighlighted(tview.Escape(filepath.Base(fileToDelete)), styleStr) + "[-:-:-:-] ?")
			} else {
				selectedFromMultipleFolders := false

				var firstFolderFound string
				for path := range fen.selected {
					if firstFolderFound == "" {
						firstFolderFound = filepath.Dir(path)
						continue
					}

					if filepath.Dir(path) != firstFolderFound {
						selectedFromMultipleFolders = true
						break
					}
				}

				if selectedFromMultipleFolders {
					modal.SetText("[red::d]Delete[-:-:-:-] " + tview.Escape(strconv.Itoa(len(fen.selected))) + " selected files [:red]from multiple folders[-:-:-:-] ?")
				} else {
					modal.SetText("[red::d]Delete[-:-:-:-] " + tview.Escape(strconv.Itoa(len(fen.selected))) + " selected files ?")
				}
			}

			modal.
				AddButtons([]string{"Yes", "No"}).
				SetFocus(1). // Default is "No"
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					pages.RemovePage("deletemodal")

					if buttonIndex != 0 {
						return
					}

					if fen.config.NoWrite {
						fen.bottomBar.TemporarilyShowTextInstead("Can't delete in no-write mode")
						return
					}

					if len(fen.selected) <= 0 {
						go fen.fileOperationsHandler.QueueOperation(FileOperation{operation: Delete, path: fileToDelete})
					} else {
						for filePath := range fen.selected {
							go fen.fileOperationsHandler.QueueOperation(FileOperation{operation: Delete, path: filePath})
						}
					}

					fen.selected = make(map[string]bool)

					fen.DisableSelectingWithV()
					fen.UpdatePanes(false)
				})

			modal.SetBorder(true)

			modal.Box.SetBackgroundColor(tcell.ColorBlack) // This sets the border background color
			modal.SetBackgroundColor(tcell.ColorBlack)

			modal.SetButtonBackgroundColor(tcell.ColorDefault)
			modal.SetButtonTextColor(tcell.ColorRed)

			pages.AddPage("deletemodal", modal, true, true)
			app.SetFocus(modal)
			return nil
		} else if event.Rune() == 'c' {
			inputField := tview.NewInputField().
				SetLabel(" Goto folder: ").
				SetPlaceholder("Relative or absolute path, case-sensitive").
				SetFieldWidth(-1) // Special feature of my tview fork, github.com/kivattt/tview

			getPathToUse := func(inputFieldText string) (string, error) {
				pathToUse := filepath.Clean(inputFieldText)
				if !filepath.IsAbs(pathToUse) {
					pathToUse, err = filepath.Abs(filepath.Join(fen.wd, pathToUse))
					if err != nil {
						return "", err
					}
				}

				stat, err := os.Stat(pathToUse)
				if err != nil {
					return "", errors.New("No such folder \"" + pathToUse + "\"")
				}

				// FIXME: Go up parent paths until a folder is found to clean up this code
				if !stat.IsDir() {
					stat, err = os.Stat(filepath.Dir(pathToUse))
					if err != nil || !stat.IsDir() {
						return "", errors.New("No such folder \"" + filepath.Dir(pathToUse) + "\"")
					} else {
						pathToUse = filepath.Dir(pathToUse)
					}
				}

				return pathToUse, nil
			}

			inputField.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					pages.RemovePage("gotofolder")
					return
				} else if key == tcell.KeyEnter {
					if inputField.GetText() == "" {
						pages.RemovePage("gotofolder")
						return
					}

					pathToUse, err := getPathToUse(inputField.GetText())
					if err != nil {
						pages.RemovePage("gotofolder")
						fen.bottomBar.TemporarilyShowTextInstead(err.Error())
						return
					}

					// FIXME: When going to ".." it doesn't do the same thing as fen.GoLeft, doesn't set fen.sel correctly
					fen.wd = pathToUse
					if filepath.Dir(fen.sel) != filepath.Clean(fen.sel) {
						fen.history.AddToHistory(fen.sel)
					}
					fen.UpdatePanes(false)
					fen.bottomBar.TemporarilyShowTextInstead("Moved to folder: \"" + pathToUse + "\"")

					pages.RemovePage("gotofolder")
					return
				}
			})

			inputField.SetAutocompleteFunc(func(currentText string) (entries []string) {
				if !enterWillSelectAutoCompleteInGotoFolder {
					return []string{}
				}

				var pathToUse string
				if !filepath.IsAbs(currentText) {
					pathToUse, err = filepath.Abs(filepath.Join(fen.wd, pathToUse))
					if err != nil {
						return []string{}
					}
				} else {
					pathToUse = filepath.Dir(currentText)
				}

				dir, err := os.ReadDir(pathToUse)
				if err != nil {
					return []string{}
				}

				var ret []string
				for _, e := range dir {
					if e.IsDir() {
						if !fen.config.HiddenFiles && strings.HasPrefix(e.Name(), ".") {
							continue
						}
						ret = append(ret, filepath.Join(pathToUse, e.Name())+string(os.PathSeparator))
					}
				}
				return ret
			})
			inputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyTab {
					enterWillSelectAutoCompleteInGotoFolder = true
					return tcell.NewEventKey(tcell.KeyDown, 'j', tcell.ModNone)
				} else if event.Key() == tcell.KeyBacktab {
					enterWillSelectAutoCompleteInGotoFolder = true
					return tcell.NewEventKey(tcell.KeyUp, 'k', tcell.ModNone)
				}

				return event
			})

			inputField.SetAutocompleteStyles(tcell.ColorBlack, tcell.StyleDefault.Foreground(tcell.ColorBlue).Bold(true).Background(tcell.ColorBlack), tcell.StyleDefault.Foreground(tcell.ColorBlue).Bold(true).Background(tcell.ColorWhite))

			inputField.SetTitleColor(tcell.ColorDefault)
			inputField.SetFieldBackgroundColor(tcell.ColorGray)
			inputField.SetFieldTextColor(tcell.ColorBlack)
			inputField.SetBackgroundColor(tcell.ColorBlack)
			inputField.SetLabelStyle(tcell.StyleDefault.Background(tcell.ColorBlack)) // This has to be before the .SetLabelColor
			inputField.SetLabelColor(tcell.NewRGBColor(0, 255, 0))                    // Green
			inputField.SetPlaceholderStyle(tcell.StyleDefault.Background(tcell.ColorGray).Dim(true))
			inputField.SetBorder(true)
			inputField.SetBorderStyle(tcell.StyleDefault.Background(tcell.ColorBlack))

			enterWillSelectAutoCompleteInGotoFolder = false

			pages.AddPage("gotofolder", centered(inputField, 3), true, true)
			app.SetFocus(inputField)
			return nil
		} else if event.Key() == tcell.KeyF5 {
			app.Sync()
		} else if event.Rune() >= '0' && event.Rune() <= '9' {
			err := fen.GoBookmark(int(event.Rune()) - '0')
			if err != nil {
				fen.bottomBar.TemporarilyShowTextInstead(err.Error())
			}
		}

		return event
	})

	if fen.config.TerminalTitle && runtime.GOOS == "linux" {
		os.Stderr.WriteString("\x1b[22t")                       // Push current terminal title
		os.Stderr.WriteString("\x1b]0;fen " + version + "\x07") // Set terminal title to "fen"
	}
	if err := app.SetRoot(pages, true).EnableMouse(fen.config.Mouse).Run(); err != nil {
		log.Fatal(err)
	}
	if fen.config.TerminalTitle && runtime.GOOS == "linux" {
		os.Stderr.WriteString("\x1b[23t") // Pop terminal title, sets it back to normal
	}

	if *printFolderOnExit {
		fmt.Println(filepath.Dir(fen.sel))
	}
}
