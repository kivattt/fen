package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"time"

	//	"runtime/pprof"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kivattt/getopt"
	"github.com/rivo/tview"
)

const version = "v1.7.16"

func main() {
	//	f, _ := os.Create("profile.prof")
	//	pprof.StartCPUProfile(f)
	//	defer pprof.StopCPUProfile()

	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	// For the dropdown in the options menu
	tview.Styles.MoreContrastBackgroundColor = tcell.ColorBlack

	tview.Styles.BorderColor = tcell.ColorDefault
	tview.Borders.Horizontal = '─'
	tview.Borders.Vertical = '│'

	if runtime.GOOS == "freebsd" {
		tview.Borders.TopLeft = '┌'
		tview.Borders.TopRight = '┐'
		tview.Borders.BottomLeft = '└'
		tview.Borders.BottomRight = '┘'
	} else {
		tview.Borders.TopLeft = '╭'
		tview.Borders.TopRight = '╮'
		tview.Borders.BottomLeft = '╰'
		tview.Borders.BottomRight = '╯'
	}

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
			if err != nil {
				continue
			}

			// Don't allow selecting the root folder, since it is normally impossible to do and relatively invisible to the user
			if pathAbsolute == filepath.Dir(pathAbsolute) {
				continue
			}

			_, err = os.Lstat(pathAbsolute)
			if err != nil {
				continue
			}
			fen.EnableSelection(pathAbsolute)
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

	if flagPassed("sort-by") {
		fen.config.SortBy = *sortBy
	}
	if flagPassed("sort-reverse") {
		fen.config.SortReverse = *sortReverse
	}

	app := tview.NewApplication()

	helpScreen := NewHelpScreen(&fen)
	librariesScreen := NewLibrariesScreen()

	err = fen.Init(path, app, &helpScreen.visible, &librariesScreen.visible)
	defer fen.Fini()
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
		if event.Key() == tcell.KeyDown || event.Rune() == 'j' {
			helpScreen.ScrollDown()
		} else if event.Key() == tcell.KeyUp || event.Rune() == 'k' {
			helpScreen.ScrollUp()
		} else if event.Key() == tcell.KeyF1 || event.Rune() == '?' || event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
			helpScreen.visible = false
			helpScreen.scrollIndex = 0
			pages.RemovePage("popup")
			fen.ShowFilepanes()
			return nil
		} else if event.Key() == tcell.KeyF2 {
			helpScreen.visible = false
			helpScreen.scrollIndex = 0
			pages.RemovePage("popup")
			librariesScreen.visible = true
			pages.AddPage("popup", librariesScreen, true, true)
			return nil
		}
		return event
	})

	librariesScreen.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyF2 || event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
			librariesScreen.visible = false
			pages.RemovePage("popup")
			fen.ShowFilepanes()
			return nil
		}

		if event.Key() == tcell.KeyF1 || event.Rune() == '?' {
			librariesScreen.visible = false
			pages.RemovePage("popup")
			helpScreen.visible = true
			pages.AddPage("popup", helpScreen, true, true)
			return nil
		}
		return event
	})

	lastWheelUpTime := time.Now()
	lastWheelDownTime := time.Now()
	app.SetMouseCapture(func(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
		if pages.HasPage("popup") {
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

		if action != tview.MouseMove {
			fen.bottomBar.alternateText = ""
		}

		// Setting the clipboard is disallowed in no-write mode because it runs a shell command
		if !fen.config.NoWrite && (runtime.GOOS == "linux" || runtime.GOOS == "freebsd") && (event.Buttons() == tcell.Button1 || event.Buttons() == tcell.Button2) {
			_, mouseY := event.Position()
			if mouseY == 0 {
				err := SetClipboardLinuxXClip(fen.sel)
				if err != nil {
					fen.topBar.additionalText = "[red::]Copy failed (install xclip)"
					fen.bottomBar.TemporarilyShowTextInstead(err.Error())
					return nil, action
				}
				fen.topBar.additionalText = "[#00ff00:]Copied to clipboard!"
				fen.topBar.showAdditionalText = true
				return event, action
			}
		} else if action == tview.MouseMove {
			if runtime.GOOS == "windows" {
				return event, action
			}

			_, mouseY := event.Position()
			if mouseY == 0 {
				if fen.showHomePathAsTilde {
					fen.showHomePathAsTilde = false
					if runtime.GOOS == "linux" || runtime.GOOS == "freebsd" {
						if fen.config.NoWrite {
							fen.topBar.additionalText = "[red]Copying unavailable (no-write)"
						} else {
							fen.topBar.additionalText = "Click to copy"
						}
						fen.topBar.showAdditionalText = true
					}
					return nil, action
				}
			} else {
				if !fen.showHomePathAsTilde {
					fen.showHomePathAsTilde = true
					fen.topBar.showAdditionalText = false
					return nil, action
				}
			}
			return event, action
		}

		if action == tview.MouseMove {
			return event, action
		}

		// Movement/navigation keys
		switch event.Buttons() {
		case tcell.Button1, tcell.Button2:
			// Small inconsistency with --ui-borders when clicking the left border of the middlepane, not important
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
				fen.GoIndex(mouseY - y + fen.middlePane.GetTopScreenEntryIndex())
			}
		case tcell.WheelLeft:
			fen.GoLeft()
		case tcell.WheelRight:
			fen.GoRight(app, "")
		case tcell.WheelUp:
			moved := false
			if time.Since(lastWheelUpTime) > time.Duration(30*time.Millisecond) {
				moved = fen.GoUp()
			} else {
				moved = fen.GoUp(fen.config.ScrollSpeed)
			}

			lastWheelUpTime = time.Now()
			if !moved {
				app.DontDrawOnThisEventMouse()
				return nil, action
			}
		case tcell.WheelDown:
			moved := false
			if time.Since(lastWheelDownTime) > time.Duration(30*time.Millisecond) {
				moved = fen.GoDown()
			} else {
				moved = fen.GoDown(fen.config.ScrollSpeed)
			}

			lastWheelDownTime = time.Now()
			if !moved {
				app.DontDrawOnThisEventMouse()
				return nil, action
			}
		default:
			return nil, action
		}

		if event.Buttons() != tcell.WheelLeft {
			fen.history.AddToHistory(fen.sel)
		}

		fen.UpdatePanes(false)
		return nil, action
	})

	enterWillSelectAutoCompleteInGotoPath := false

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if pages.HasPage("popup") {
			return event
		}

		fen.bottomBar.alternateText = ""

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
					pages.RemovePage("popup")

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

			pages.AddPage("popup", modal, true, true)
			app.SetFocus(modal)
			return nil
		}

		// Movement/navigation keys
		wasMovementKey := true
		if (event.Modifiers()&tcell.ModCtrl == 0 && event.Key() == tcell.KeyLeft) || event.Rune() == 'h' {
			fen.GoLeft()
		} else if (event.Modifiers()&tcell.ModCtrl == 0 && event.Key() == tcell.KeyRight) || event.Rune() == 'l' || event.Key() == tcell.KeyEnter {
			fen.GoRight(app, "")
		} else if event.Key() == tcell.KeyUp || event.Rune() == 'k' {
			if !fen.GoUp() {
				app.DontDrawOnThisEventKey()
				return nil
			}
		} else if event.Key() == tcell.KeyDown || event.Rune() == 'j' {
			if !fen.GoDown() {
				app.DontDrawOnThisEventKey()
				return nil
			}
		} else if event.Rune() == ' ' {
			fen.ToggleSelection(fen.sel)
			fen.GoDown()
		} else if event.Key() == tcell.KeyHome || event.Rune() == 'g' {
			if fen.config.FoldersFirst && fen.config.SplitHomeEnd {
				fen.GoTopFileOrTop()
			} else {
				fen.GoTop()
			}
		} else if event.Key() == tcell.KeyEnd || event.Rune() == 'G' {
			if fen.config.FoldersFirst && fen.config.SplitHomeEnd {
				fen.GoBottomFolderOrBottom()
			} else {
				fen.GoBottom()
			}
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
			if !((event.Modifiers()&tcell.ModCtrl == 0 && event.Key() == tcell.KeyLeft) || event.Rune() == 'h') {
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

			inputField.SetDoneFunc(func(key tcell.Key) {
				pages.RemovePage("popup")

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

			pages.AddPage("popup", centered(inputField, 3), true, true)
			return nil
		} else if event.Rune() == 'A' {
			for _, e := range fen.middlePane.entries.Load().([]os.DirEntry) {
				fen.ToggleSelection(filepath.Join(fen.wd, e.Name()))
			}
			fen.DisableSelectingWithV()
			return nil
		} else if event.Rune() == 'D' {
			if len(fen.selected) > 0 {
				fen.selected = make(map[string]bool)
				fen.bottomBar.TemporarilyShowTextInstead("Deselected!")
			} else if len(fen.yankSelected) > 0 {
				fen.yankSelected = make(map[string]bool)
				fen.bottomBar.TemporarilyShowTextInstead("Un-yanked!")
			}

			fen.DisableSelectingWithV()
			return nil
		} else if event.Rune() == 'a' {
			fen.DisableSelectingWithV()
			fileToRename := fen.sel

			inputField := tview.NewInputField().
				SetLabel(" Rename: ").
				SetText(filepath.Base(fileToRename)).
				SetFieldWidth(-1) // Special feature of my tview fork, github.com/kivattt/tview

			inputField.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					pages.RemovePage("popup")
					return
				} else if key == tcell.KeyEnter {
					if !fen.config.NoWrite {
						newPath := filepath.Join(filepath.Dir(fileToRename), inputField.GetText())
						_, err := os.Lstat(newPath)
						if err == nil {
							pages.RemovePage("popup")
							fen.bottomBar.TemporarilyShowTextInstead("Can't rename to an existing file")
							return
						}

						err = os.Rename(fileToRename, newPath)
						if err != nil {
							pages.RemovePage("popup")
							fen.bottomBar.TemporarilyShowTextInstead("Can't rename, no access")
							return
						}

						// These are also done by file system events, but let's be safe
						fen.RemoveFromSelectedAndYankSelected(fileToRename)
						fen.history.RemoveFromHistory(fileToRename)

						// We can't use fen.GoPath() here because it would enter directories
						fen.UpdatePanes(true)
						fen.sel = newPath
						fen.middlePane.SetSelectedEntryFromString(filepath.Base(fen.sel)) // fen.UpdatePanes() overwrites fen.sel, so we have to set the index
						fen.history.AddToHistory(newPath)
					} else {
						fen.bottomBar.TemporarilyShowTextInstead("Can't rename in no-write mode")
					}

					pages.RemovePage("popup")
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

			pages.AddPage("popup", centered(inputField, 3), true, true)
			app.SetFocus(inputField)
			return nil
		} else if event.Rune() == 'n' || event.Rune() == 'N' {
			fen.DisableSelectingWithV()
			inputField := tview.NewInputField().
				SetFieldWidth(-1) // Special feature of my tview fork, github.com/kivattt/tview

			if event.Rune() == 'n' {
				inputField.SetLabel(" New file: ")
			} else if event.Rune() == 'N' {
				inputField.SetLabel(" New folder: ")
			}

			inputField.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					pages.RemovePage("popup")
					return
				} else if key == tcell.KeyEnter {
					pathToUse := filepath.Join(fen.wd, inputField.GetText())
					if filepath.Dir(pathToUse) != fen.wd || (runtime.GOOS != "windows" && pathToUse == string(os.PathSeparator)) || strings.ContainsRune(inputField.GetText(), os.PathSeparator) {
						fen.bottomBar.TemporarilyShowTextInstead("Paths outside of the current folder are not yet supported")
						pages.RemovePage("popup")
						return
					}

					_, err := os.Stat(pathToUse) // Here to make sure we don't overwrite a file when making a new one
					if !fen.config.NoWrite && err != nil {
						var createFileOrFolderErr error
						if event.Rune() == 'n' {
							var file *os.File
							file, createFileOrFolderErr = os.Create(pathToUse)
							if createFileOrFolderErr == nil {
								defer file.Close()
							}
						} else if event.Rune() == 'N' {
							createFileOrFolderErr = os.Mkdir(pathToUse, 0775)
						}

						if createFileOrFolderErr != nil {
							fen.bottomBar.TemporarilyShowTextInstead(createFileOrFolderErr.Error())
						} else {
							fen.sel = pathToUse
							fen.history.AddToHistory(fen.sel)
						}
						fen.UpdatePanes(true)
					} else if fen.config.NoWrite {
						fen.bottomBar.TemporarilyShowTextInstead("Can't create new files in no-write mode")
					} else if err != nil {
						fen.bottomBar.TemporarilyShowTextInstead(err.Error())
					} else {
						fen.bottomBar.TemporarilyShowTextInstead("Can't create an existing file")
					}

					pages.RemovePage("popup")
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

			pages.AddPage("popup", centered(inputField, 3), true, true)
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
			return nil
		} else if event.Key() == tcell.KeyF1 || event.Rune() == '?' {
			helpScreen.visible = !helpScreen.visible
			if helpScreen.visible {
				pages.AddPage("popup", helpScreen, true, true)
				fen.HideFilepanes()
			} else {
				pages.RemovePage("popup")
				fen.ShowFilepanes()
			}
			return nil
		} else if event.Key() == tcell.KeyF2 {
			librariesScreen.visible = !librariesScreen.visible
			if librariesScreen.visible {
				pages.AddPage("popup", librariesScreen, true, true)
				fen.HideFilepanes()
			} else {
				pages.RemovePage("popup")
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
				fileToDeleteInfo, _ := os.Lstat(fileToDelete)
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
					pages.RemovePage("popup")

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

			pages.AddPage("popup", modal, true, true)
			app.SetFocus(modal)
			return nil
		} else if event.Rune() == 'c' {
			inputField := tview.NewInputField().
				SetLabel(" Goto path: ").
				SetPlaceholder("Relative or absolute path, case-sensitive").
				SetFieldWidth(-1) // Special feature of my tview fork, github.com/kivattt/tview

			inputField.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					pages.RemovePage("popup")
					return
				} else if key == tcell.KeyEnter {
					if inputField.GetText() == "" {
						pages.RemovePage("popup")
						return
					}

					path, err := fen.GoPath(inputField.GetText())
					if err != nil {
						pages.RemovePage("popup")
						fen.bottomBar.TemporarilyShowTextInstead(err.Error())
						return
					}

					pages.RemovePage("popup")
					fen.bottomBar.TemporarilyShowTextInstead("Moved to path: \"" + path + "\"")
					return
				}
			})

			inputField.SetAutocompleteFunc(func(currentText string) (entries []string) {
				if !enterWillSelectAutoCompleteInGotoPath {
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
					enterWillSelectAutoCompleteInGotoPath = true
					return tcell.NewEventKey(tcell.KeyDown, 'j', tcell.ModNone)
				} else if event.Key() == tcell.KeyBacktab {
					enterWillSelectAutoCompleteInGotoPath = true
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

			enterWillSelectAutoCompleteInGotoPath = false

			pages.AddPage("popup", centered(inputField, 3), true, true)
			app.SetFocus(inputField)
			return nil
		} else if event.Key() == tcell.KeyF5 {
			fen.UpdatePanes(true)
			app.Sync()
			fen.TriggerGitStatus()
			return nil
		} else if event.Rune() >= '0' && event.Rune() <= '9' {
			err := fen.GoBookmark(int(event.Rune()) - '0')
			if err != nil {
				fen.bottomBar.TemporarilyShowTextInstead(err.Error())
			}
			return nil
		} else if event.Modifiers()&tcell.ModCtrl != 0 && event.Key() == tcell.KeyRight { // Ctrl+Right
			stat, err := os.Lstat(fen.sel)
			if err == nil && stat.Mode()&os.ModeSymlink != 0 {
				err := fen.GoSymlink(fen.sel)
				if err != nil {
					fen.bottomBar.TemporarilyShowTextInstead(err.Error())
				}
				return nil
			}

			if !fen.config.GitStatus {
				fen.GoRightUpToHistory()
				return nil
			}

			path, err := fen.gitStatusHandler.TryFindParentGitRepository(filepath.Dir(fen.sel))
			if err != nil {
				fen.GoRightUpToHistory()
				return nil
			}

			err = fen.GoRightUpToFirstUnstagedOrUntracked(path, fen.sel)
			if err != nil {
				fen.GoRightUpToHistory()
				return nil
			}

			return nil
		} else if event.Modifiers()&tcell.ModCtrl != 0 && event.Key() == tcell.KeyLeft { // Ctrl+Left
			if !fen.config.GitStatus {
				fen.GoRootPath()
				return nil
			}

			stat, statErr := os.Lstat(filepath.Join(filepath.Dir(fen.sel), ".git"))
			repositoryPath, err := fen.gitStatusHandler.TryFindParentGitRepository(filepath.Dir(fen.sel))
			if err == nil && !(statErr == nil && stat.IsDir()) {
				fen.GoPath(repositoryPath)
			} else {
				fen.GoRootPath()
			}
			return nil
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
				if key == tcell.KeyEscape {
					pages.RemovePage("popup")
					return
				}

				programNameToUse := inputField.GetText()
				if programNameToUse == "" {
					if len(programs) > 0 {
						programNameToUse = programs[0]
					}
				}
				pages.RemovePage("popup")
				fen.GoRight(app, programNameToUse)
			})

			flex := tview.NewFlex().
				AddItem(inputField, inputFieldHeight, 1, true).SetDirection(tview.FlexRow).
				AddItem(programsList, len(programs), 1, false)

			flex.SetBorder(true)
			flex.SetBorderStyle(tcell.StyleDefault.Background(tcell.ColorBlack))

			pages.AddPage("popup", centered(flex, inputFieldHeight+2+len(programs)), true, true)
			return nil
		} else if event.Rune() == '!' {
			shellName := GetShellArgs()[0]
			inputField := tview.NewInputField().
				SetLabel(" Run " + filepath.Base(shellName) + " command: ").
				SetFieldWidth(-1) // Special feature of my tview fork, github.com/kivattt/tview

			inputField.SetBorder(true)
			inputField.SetBorderStyle(tcell.StyleDefault.Background(tcell.ColorBlack))
			inputField.SetTitleColor(tcell.ColorDefault)
			inputField.SetFieldBackgroundColor(tcell.ColorGray)
			inputField.SetFieldTextColor(tcell.ColorBlack)
			inputField.SetBackgroundColor(tcell.ColorBlack)

			inputField.SetLabelStyle(tcell.StyleDefault.Background(tcell.ColorBlack)) // This has to be before the .SetLabelColor
			inputField.SetLabelColor(tcell.NewRGBColor(0, 255, 0))                    // Green

			inputField.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					pages.RemovePage("popup")
					return
				}

				if fen.config.NoWrite {
					pages.RemovePage("popup")
					fen.bottomBar.TemporarilyShowTextInstead("Can't run shell commands in no-write mode")
					return
				}

				if inputField.GetText() == "" {
					pages.RemovePage("popup")
					fen.bottomBar.TemporarilyShowTextInstead("Empty command, nothing done")
					return
				}

				command := inputField.GetText()
				var err error
				var exitCode int
				app.Suspend(func() {
					err = InvokeShell(command, fen.wd)

					if err != nil {
						exitError, ok := err.(*exec.ExitError)
						if ok {
							exitCode = exitError.ExitCode()
						}
					}

					if err == nil || exitCode != 0 {
						fmt.Print("\n\x1b[1;30mFinished with exit code\x1b[0m ")
						if exitCode == 0 {
							fmt.Print("\x1b[0;92m")
						} else {
							fmt.Print("\x1b[0;91m")
						}
						fmt.Print(strconv.Itoa(exitCode) + "\x1b[0m\x1b[1;30m, ")
						anyKey := "press \x1b[4many key\x1b[0m\x1b[1;30m to continue..."
						enter := "press \x1b[4mEnter\x1b[0m\x1b[1;30m to continue..."
						PressAnyKeyToContinue(anyKey, enter)
						fmt.Print("\x1b[0m\n\n")
					}
				})

				if err != nil && exitCode == 0 {
					fen.bottomBar.TemporarilyShowTextInstead(err.Error())
				}

				pages.RemovePage("popup")
			})

			pages.AddPage("popup", centered(inputField, 3), true, true)
			return nil
		} else if event.Rune() == 'b' {
			err := fen.BulkRename(app)
			if err != nil {
				fen.bottomBar.TemporarilyShowTextInstead(err.Error())
				return nil
			}

			return nil
		} else if event.Rune() == 'o' {
			optionsForm := tview.NewForm()

			configTypes := reflect.TypeOf(fen.config)
			configValues := reflect.ValueOf(&fen.config).Elem()

			// Loop through the fields in alphabetically sorted order
			type indexAndText struct {
				index int
				text  string
			}
			sortedIndices := []indexAndText{}
			for i := 0; i < configTypes.NumField(); i++ {
				fieldName := configTypes.Field(i).Tag.Get(luaTagName)
				sortedIndices = append(sortedIndices, indexAndText{index: i, text: fieldName})
			}

			optionsAtTheTop := []string{
				"sort_by",
				"sort_reverse",
				"ui_borders",
			}
			slices.SortFunc(sortedIndices, func(a, b indexAndText) int {
				for _, shouldBeOnTop := range optionsAtTheTop {
					if a.text == shouldBeOnTop {
						return -1
					} else if b.text == shouldBeOnTop {
						return 1
					}
				}

				return strings.Compare(a.text, b.text)
			})

			if len(sortedIndices) != configTypes.NumField() {
				panic("Length of sorted field indices did not match the actual number of fields")
			}

			numOptions := 0
			for _, v := range sortedIndices {
				i := v.index
				value := configValues.Field(i)
				fieldPtr := value.Addr().Interface()
				fieldName := configTypes.Field(i).Tag.Get(luaTagName)

				if slices.Contains(ConfigKeysByTagNameNotToIncludeInOptionsMenu, fieldName) {
					continue
				}

				switch value.Kind() {
				case reflect.Bool:
					fieldValue := value.Bool()

					f := func(checked bool) {
						*fieldPtr.(*bool) = checked
						fen.UpdatePanes(true)
					}

					if fieldName == "mouse" {
						f = func(checked bool) {
							*fieldPtr.(*bool) = checked
							app.EnableMouse(checked)
							fen.UpdatePanes(true)
						}
					} else if fieldName == "git_status" {
						// Don't show the git_status option if it was disabled on startup, to prevent crashes
						if !fen.initializedGitStatus {
							continue
						}
					} else if fieldName == "show_hostname" && runtime.GOOS == "windows" {
						// Don't show the show_hostname option on Windows, it does nothing on Windows
						continue
					}

					optionsForm.AddCheckbox(fieldName, fieldValue, f)
				case reflect.String:
					if fieldName != "sort_by" {
						panic("Options menu expected the only config string to be sort_by")
					}

					fieldValue := value.String()
					optionsForm.AddDropDown(fieldName, ValidSortByValues[:], slices.Index(ValidSortByValues[:], fieldValue), func(option string, optionIndex int) {
						*fieldPtr.(*string) = option
						fen.UpdatePanes(true)
					})
				default:
					continue
				}

				numOptions++
			}

			optionsForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
					pages.RemovePage("popup")
					return nil
				}

				if event.Key() == tcell.KeyDown || event.Rune() == 'j' {
					return tcell.NewEventKey(tcell.KeyTab, event.Rune(), event.Modifiers())
				} else if event.Key() == tcell.KeyUp || event.Rune() == 'k' {
					return tcell.NewEventKey(tcell.KeyBacktab, event.Rune(), event.Modifiers())
				} else if event.Key() == tcell.KeyLeft || event.Key() == tcell.KeyRight || event.Rune() == 'h' || event.Rune() == 'l' {
					return tcell.NewEventKey(tcell.KeyEnter, event.Rune(), event.Modifiers())
				}

				return event
			})

			optionsForm.SetItemPadding(0)
			optionsForm.SetTitle("Options this session")
			optionsForm.SetBorder(true)
			optionsForm.SetBackgroundColor(tcell.ColorBlack)
			optionsForm.SetLabelColor(tcell.NewRGBColor(0, 255, 0)) // Green
			optionsForm.SetBorderPadding(0, 0, 1, 1)
			optionsForm.SetFieldBackgroundColor(tcell.ColorBlack)
			optionsForm.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
				if width < 75 {
					return x + 1, y + 1, width - 2, height - 1
				}
				xOffset := width/2 - 20
				theX := max(x+1, x+xOffset)
				return theX, y + 1, width - (theX - x) - 1, height - 1
			})

			pages.AddPage("popup", centered(optionsForm, numOptions+2), true, true)
			return nil
		}

		app.DontDrawOnThisEventKey()
		return event
	})

	if fen.config.TerminalTitle {
		fen.PushAndSetTerminalTitle()
	}
	if err := app.SetRoot(pages, true).EnableMouse(fen.config.Mouse).EnablePaste(true).Run(); err != nil {
		log.Fatal(err)
	}

	if fen.config.TerminalTitle {
		fen.PopTerminalTitle()
	}

	if *printFolderOnExit {
		fmt.Println(filepath.Dir(fen.sel))
	}
}
