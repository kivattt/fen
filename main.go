package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"slices"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kivattt/getopt"
	"github.com/rivo/tview"
)

const version = "v1.7.24 pre-release"

func SetTviewStyles() {
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
}

var missingSpaceRune rune = '…'

func main() {
	SetTviewStyles()
	if runtime.GOOS == "freebsd" {
		missingSpaceRune = '~'
	}

	userConfigDir, err := os.UserConfigDir()
	defaultConfigFilenamePath := ""
	// If there was an error, defaultConfigFilenamePath will be an empty string ""
	//  and fen.ReadConfig() will return nil, just using the default config
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
	hiddenFiles := flag.Bool("hidden-files", defaultConfigValues.HiddenFiles, "make hidden files visible")
	foldersFirst := flag.Bool("folders-first", defaultConfigValues.FoldersFirst, "always show folders at the top")
	printPathOnOpen := flag.Bool("print-path-on-open", defaultConfigValues.PrintPathOnOpen, "output file path(s) and exit when opening file(s)")
	profileCpu := flag.Bool("profile-cpu", false, "generate a CPU profile .pprof file")
	printFolderOnExit := flag.Bool("print-folder-on-exit", false, "output the current working folder in fen on exit")
	allowTerminalTitle := flag.Bool("terminal-title", defaultConfigValues.TerminalTitle, "change terminal title to 'fen "+version+"' while open")
	showHelpText := flag.Bool("show-help-text", defaultConfigValues.ShowHelpText, "show the 'For help: ...' text")
	showHostname := flag.Bool("show-hostname", defaultConfigValues.ShowHostname, "show username@hostname in the top-left")
	closeOnEscape := flag.Bool("close-on-escape", defaultConfigValues.CloseOnEscape, "make the escape key exit fen")

	selectPaths := flag.Bool("select", false, "select PATHS")

	configFilename := flag.String("config", defaultConfigFilenamePath, "use configuration file")
	sortBy := flag.String("sort-by", defaultConfigValues.SortBy, "sort files ("+strings.Join(ValidSortByValues[:], ", ")+")")
	sortReverse := flag.Bool("sort-reverse", defaultConfigValues.SortReverse, "reverse sort")
	fileSizeFormat := flag.String("file-size-format", defaultConfigValues.FileSizeFormat, "file size format ("+strings.Join(ValidFileSizeFormatValues[:], ", ")+")")

	getopt.CommandLine.SetOutput(os.Stdout)
	getopt.CommandLine.Init("fen", flag.ExitOnError)
	getopt.Aliases(
		"v", "version",
		"h", "help",
	)

	err = getopt.CommandLine.Parse(os.Args[1:])
	if err != nil {
		os.Exit(2)
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

	if *profileCpu {
		outputFile, err := os.CreateTemp("", "fenprofile*.pprof")
		if err != nil {
			log.Fatal(err)
		}

		if err := pprof.StartCPUProfile(outputFile); err != nil {
			log.Fatal(err)
		}

		log.Println("cpu profiling enabled, " + outputFile.Name())
		defer func() {
			pprof.StopCPUProfile()
			outputFile.Close()
			log.Println("cpu profiling disabled, " + outputFile.Name())
		}()
	}

	path, err := filepath.Abs(getopt.CommandLine.Arg(0))
	if path == "" || err != nil {
		path, err = CurrentWorkingDirectory()
		if err != nil {
			log.Fatal(err)
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

	if !fen.config.NoWrite {
		os.Mkdir(filepath.Join(userConfigDir, "fen"), 0o775)
	}

	err = fen.ReadConfig(*configFilename)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		// Hacky, but gets the job done
		if !strings.HasSuffix(err.Error(), "config files can only be Lua.\n") {
			fmt.Println("Invalid config '" + *configFilename + "', exiting")
		} else if !fen.config.NoWrite {
			PromptForGenerateLuaConfig(*configFilename, &fen)
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
	if flagPassed("close-on-escape") {
		fen.config.CloseOnEscape = *closeOnEscape
	}
	if flagPassed("sort-by") {
		fen.config.SortBy = *sortBy
	}
	if flagPassed("sort-reverse") {
		fen.config.SortReverse = *sortReverse
	}
	if flagPassed("file-size-format") {
		fen.config.FileSizeFormat = *fileSizeFormat
	}

	if !slices.Contains(ValidFileSizeFormatValues[:], fen.config.FileSizeFormat) {
		fmt.Fprintln(os.Stderr, "Invalid file_size_format value \""+fen.config.FileSizeFormat+"\"")
		fmt.Fprintln(os.Stderr, "Valid values: "+strings.Join(ValidFileSizeFormatValues[:], ", "))
		os.Exit(1)
	}

	if !slices.Contains(ValidSortByValues[:], fen.config.SortBy) {
		fmt.Fprintln(os.Stderr, "Invalid sort_by value \""+fen.config.SortBy+"\"")
		fmt.Fprintln(os.Stderr, "Valid values: "+strings.Join(ValidSortByValues[:], ", "))
		os.Exit(1)
	}

	if !slices.Contains(ValidFilenameSearchCaseValues[:], fen.config.FilenameSearchCase) {
		fmt.Fprintln(os.Stderr, "Invalid filename_search_case value \""+fen.config.FilenameSearchCase+"\"")
		fmt.Fprintln(os.Stderr, "Valid values: "+strings.Join(ValidFilenameSearchCaseValues[:], ", "))
		os.Exit(1)
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

	setHelpInputHandler(pages, &fen, librariesScreen, helpScreen)
	setLibrariesInputHandler(pages, &fen, librariesScreen, helpScreen)
	setAppMouseHandler(app, pages, &fen)
	setAppInputHandler(app, pages, &fen, librariesScreen, helpScreen)

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
