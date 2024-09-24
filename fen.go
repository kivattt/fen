package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/rivo/tview"
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

type Fen struct {
	app     *tview.Application
	wd      string
	sel     string
	history History

	selected     map[string]bool
	yankSelected map[string]bool
	yankType     string // "", "copy", "cut"

	selectingWithV               bool
	selectingWithVStartIndex     int
	selectingWithVEndIndex       int
	selectedBeforeSelectingWithV map[string]bool

	config                Config
	fileOperationsHandler FileOperationsHandler

	helpScreenVisible *bool

	topBar     *TopBar
	bottomBar  *BottomBar
	leftPane   *FilesPane
	middlePane *FilesPane
	rightPane  *FilesPane

	effectiveShowHomePathAsTilde bool
}

// gluamapper lets you use Go variables like "UiBorders", using the name "ui_borders".
// I happen to like this, but since I can't set the "fen" global to the actual Config value,
// we have to define them manually with a new table where I use these struct tags to look up the names
const luaTagName = "lua"

type Config struct {
	UiBorders               bool                 `lua:"ui_borders"`
	Mouse                   bool                 `lua:"mouse"`
	NoWrite                 bool                 `lua:"no_write"`
	HiddenFiles             bool                 `lua:"hidden_files"`
	FoldersFirst            bool                 `lua:"folders_first"`
	PrintPathOnOpen         bool                 `lua:"print_path_on_open"`
	TerminalTitle           bool                 `lua:"terminal_title"`
	ShowHelpText            bool                 `lua:"show_help_text"`
	ShowHostname            bool                 `lua:"show_hostname"`
	ShowHomePathAsTilde     bool                 `lua:"show_home_path_as_tilde"` // Allows fen.effectiveShowHomePathAsTilde to change
	Open                    []PreviewOrOpenEntry `lua:"open"`
	Preview                 []PreviewOrOpenEntry `lua:"preview"`
	SortBy                  string               `lua:"sort_by"`
	SortReverse             bool                 `lua:"sort_reverse"`
	FileEventIntervalMillis int                  `lua:"file_event_interval_ms"`
	AlwaysShowInfoNumbers   bool                 `lua:"always_show_info_numbers"`
	ScrollSpeed             int                  `lua:"scroll_speed"`
	Bookmarks               [10]string           `lua:"bookmarks"`
}

var ValidSortByValues = [...]string{"none", "modified", "size", "file-extension"}

func NewConfigDefaultValues() Config {
	// Anything not specified here will have the default value for its type, e.g. false for booleans
	return Config{
		Mouse:                   true,
		FoldersFirst:            true,
		TerminalTitle:           true,
		ShowHelpText:            true,
		ShowHostname:            true,
		ShowHomePathAsTilde:     true,
		SortBy:                  "none",
		FileEventIntervalMillis: 300,
		ScrollSpeed:             2,
	}
}

type PreviewOrOpenEntry struct {
	Script     string
	Program    []string // The name used to be "Programs", but this makes more sense for the lua configuration
	Match      []string
	DoNotMatch []string
}

func (fen *Fen) Init(path string, app *tview.Application, helpScreenVisible *bool) error {
	fen.app = app
	fen.fileOperationsHandler = FileOperationsHandler{fen: fen}
	fen.helpScreenVisible = helpScreenVisible

	if fen.selected == nil {
		fen.selected = map[string]bool{}
	}

	fen.yankSelected = map[string]bool{}

	fen.selectedBeforeSelectingWithV = map[string]bool{}

	fen.wd = path

	fen.topBar = NewTopBar(fen)

	fen.leftPane = NewFilesPane(fen, false, false)
	fen.middlePane = NewFilesPane(fen, true, false)
	fen.rightPane = NewFilesPane(fen, false, true)

	fen.leftPane.Init()
	fen.middlePane.Init()
	fen.rightPane.Init()

	if fen.config.UiBorders {
		fen.leftPane.SetBorder(true)
		fen.middlePane.SetBorder(true)
		fen.rightPane.SetBorder(true)
	}

	fen.bottomBar = NewBottomBar(fen)

	wdFiles, err := os.ReadDir(fen.wd)
	shouldSelectSpecifiedFile := false

	stat, statErr := os.Stat(fen.wd)
	if statErr == nil && !stat.IsDir() {
		shouldSelectSpecifiedFile = true
	}

	// If our working directory doesn't exist, go up a parent until it does
	for err != nil {
		if filepath.Dir(fen.wd) == fen.wd {
			return err
		}

		fen.wd = filepath.Dir(fen.wd)
		wdFiles, err = os.ReadDir(fen.wd)
	}

	if len(wdFiles) > 0 {
		// HACKY: middlePane has to have entries so that GoTop() will work
		fen.middlePane.ChangeDir(fen.wd, true)
		fen.GoTop()

		if shouldSelectSpecifiedFile {
			fen.sel = path
		}
	}

	fen.history.AddToHistory(fen.sel)
	fen.UpdatePanes(false)

	return err
}

func (fen *Fen) ReadConfig(path string) error {
	fen.config = NewConfigDefaultValues()

	if !strings.HasSuffix(filepath.Base(path), ".lua") {
		fmt.Fprintln(os.Stderr, "Warning: Config file "+path+" has no .lua file extension.\nSince v1.3.0, config files can only be Lua.\n")
	}

	_, err := os.Stat(path)
	if err != nil {
		oldJSONConfigPath := filepath.Join(filepath.Dir(path), "fenrc.json")
		_, err := os.Stat(oldJSONConfigPath)
		if err == nil {
			return errors.New("Could not find " + path + ", but found " + oldJSONConfigPath + "\nSince v1.3.0, config files can only be Lua.\n")
		}

		// We don't want to exit if there is no config file
		// This should really be checked by the caller...
		return nil
	}

	L := lua.NewState()
	defer L.Close()

	// This is what we initially pass to config.lua
	luaInitialConfigTable := L.NewTable()

	defaultConfigReflectTypes := reflect.TypeOf(fen.config)
	defaultConfigReflectValues := reflect.ValueOf(fen.config)
	for i := 0; i < defaultConfigReflectValues.NumField(); i++ {
		fieldName := defaultConfigReflectTypes.Field(i).Tag.Get(luaTagName)

		switch defaultConfigReflectValues.Field(i).Kind() {
		case reflect.Bool:
			fieldValue := defaultConfigReflectValues.Field(i).Bool()
			luaInitialConfigTable.RawSetString(fieldName, lua.LBool(fieldValue))
		case reflect.Slice: // fen.open and fen.preview are set to empty lists (called a "table" in lua)
			luaInitialConfigTable.RawSetString(fieldName, L.NewTable())
		}
	}

	userConfigDir, err := os.UserConfigDir()
	if err == nil {
		luaInitialConfigTable.RawSetString("config_path", lua.LString(PathWithEndSeparator(filepath.Join(userConfigDir, "fen"))))
	}
	luaInitialConfigTable.RawSetString("version", lua.LString(version))
	luaInitialConfigTable.RawSetString("runtime_os", lua.LString(runtime.GOOS))
	userHomeDir, err := os.UserHomeDir()
	if err == nil {
		luaInitialConfigTable.RawSetString("home_path", lua.LString(PathWithEndSeparator(userHomeDir)))
	}
	L.SetGlobal("fen", luaInitialConfigTable)

	err = L.DoFile(path)
	if err != nil {
		return err
	}

	// TODO: Could probably refactor this and make it check via reflection of fen.config
	// In the Lua config, referring to variables by their Go name, e.g. "fen.UiBorders" instead of "fen.ui_borders" makes fen
	// pick between one or the other (previously set by luaInitialConfigTable) randomly.
	// Why? Because Golang maps.
	// You can actually still observe this kind of issue by using atleast 2 invalid fen global variable names in your config:
	//
	// config.lua:
	//   fen.UiBorders = false
	//   fen.AnotherOne = false
	//
	// then, by running fen multiple times you will see the "Invalid fen global variable name: ..." error msg show one of the two randomly
	//
	// So let's save the user most of this pain and not allow using the original name.
	mapper := gluamapper.NewMapper(gluamapper.Option{NameFunc: func(originalName string) string {
		newName := gluamapper.ToUpperCamelCase(originalName)

		// If ToUpperCamelCase did nothing, it indicates the use of a fen.config Go name instead of the intended Lua name
		if originalName == newName {
			// Since we unfortunately can't just return err like in ReadConfig(), let's just replicate the behaviour of handling the error from main.go
			fmt.Println("Invalid config '" + path + "', exiting")
			err := errors.New("Invalid fen global variable name: " + originalName)
			log.Fatal(err)
		}

		return newName
	}})

	fenGlobal := L.GetGlobal("fen")
	fenGlobalAsTablePointer, ok := L.GetGlobal("fen").(*lua.LTable)
	if !ok {
		return errors.New("Failed to convert \"fen\" (of type " + fenGlobal.Type().String() + ") to a *lua.LTable")
	}

	err = mapper.Map(fenGlobalAsTablePointer, &fen.config)
	if err != nil {
		return err
	}

	return nil
}

func (fen *Fen) ToggleSelectingWithV() {
	if !fen.selectingWithV {
		fen.EnableSelectingWithV()
	} else {
		fen.DisableSelectingWithV()
	}
}

func (fen *Fen) EnableSelectingWithV() {
	if fen.selectingWithV {
		return
	}

	fen.selectingWithV = true
	fen.selectingWithVStartIndex = fen.middlePane.selectedEntryIndex
	fen.selectingWithVEndIndex = fen.selectingWithVStartIndex

	// We have to do this to copy fen.selected, and not a reference to it
	fen.selectedBeforeSelectingWithV = make(map[string]bool)

	for k, v := range fen.selected {
		fen.selectedBeforeSelectingWithV[k] = v
	}
}

func (fen *Fen) DisableSelectingWithV() {
	if !fen.selectingWithV {
		return
	}

	fen.selectingWithV = false
	fen.selectedBeforeSelectingWithV = make(map[string]bool)
}

func (fen *Fen) KeepMiddlePaneSelectionInBounds() {
	// Don't know if necessary
	/*if fen.middlePane.selectedEntryIndex < 0 {
		fen.middlePane.selectedEntryIndex = 0
	}*/

	// I think Load()ing entries multiple times like this could be unsafe, but might realistically be very rare
	if fen.middlePane.selectedEntryIndex >= len(fen.middlePane.entries.Load().([]os.DirEntry)) {
		if len(fen.middlePane.entries.Load().([]os.DirEntry)) > 0 {
			fen.sel = fen.middlePane.GetSelectedEntryFromIndex(len(fen.middlePane.entries.Load().([]os.DirEntry)) - 1)
			err := fen.middlePane.SetSelectedEntryFromString(filepath.Base(fen.sel)) // Duplicated from above...
			if err != nil {
				panic("In KeepSelectionInBounds(): " + err.Error())
			}
		} else {
			fen.middlePane.SetSelectedEntryFromIndex(0)
		}
	}
}

// forceReadDir is used for making navigation better, like making a new file or folder selects the new path, renaming a file selecting the new path and toggling hidden files
// Since FilterAndSortEntries overwrites filespane entries
func (fen *Fen) UpdatePanes(forceReadDir bool) {
	// If working directory is not accessible, go up to the first accessible parent
	// FIXME: We need a log we can scroll through
	// This bottomBar message would not show up due to the file watcher updating after it has appeared
	/*if err != nil {
		fen.bottomBar.TemporarilyShowTextInstead(fen.wd + " became non-accessible, moved to a parent")
	}*/

	// TODO: Preserve last available selection index (so it doesn't reset to the top)
	_, err := os.Stat(fen.wd)
	for err != nil {
		if filepath.Dir(fen.wd) == fen.wd {
			panic("Could not find usable parent path")
		}

		fen.wd = filepath.Dir(fen.wd)
		_, err = os.Stat(fen.wd)
	}

	fen.leftPane.ChangeDir(filepath.Dir(fen.wd), forceReadDir)
	fen.middlePane.ChangeDir(fen.wd, forceReadDir)

	if filepath.Clean(fen.wd) == filepath.Dir(fen.wd) {
		fen.leftPane.entries.Store([]os.DirEntry{})
	} else {
		fen.leftPane.SetSelectedEntryFromString(filepath.Base(fen.wd))
	}

	fen.middlePane.SetSelectedEntryFromString(filepath.Base(fen.sel))
	fen.KeepMiddlePaneSelectionInBounds()

	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.selectedEntryIndex))
	fen.rightPane.ChangeDir(fen.sel, forceReadDir)

	// Prevents showing 'empty' a second time in rightPane, if middlePane is already showing 'empty'
	if len(fen.middlePane.entries.Load().([]os.DirEntry)) <= 0 {
		fen.rightPane.parentIsEmptyFolder = false
	}

	h, err := fen.history.GetHistoryEntryForPath(fen.sel, fen.config.HiddenFiles)
	if err != nil {
		fen.rightPane.SetSelectedEntryFromIndex(0)
	} else {
		fen.rightPane.SetSelectedEntryFromString(filepath.Base(h))
		fen.KeepMiddlePaneSelectionInBounds()
	}

	fen.UpdateSelectingWithV()
}

func (fen *Fen) HideFilepanes() {
	fen.leftPane.Invisible = true
	fen.middlePane.Invisible = true
	fen.rightPane.Invisible = true
}

func (fen *Fen) ShowFilepanes() {
	fen.leftPane.Invisible = false
	fen.middlePane.Invisible = false
	fen.rightPane.Invisible = false
}

func (fen *Fen) RemoveFromSelectedAndYankSelected(path string) {
	delete(fen.selected, path)
	delete(fen.yankSelected, path)
}

func (fen *Fen) ToggleSelection(filePath string) {
	_, exists := fen.selected[filePath]

	if exists {
		delete(fen.selected, filePath)
		return
	}

	fen.selected[filePath] = true
}

func (fen *Fen) EnableSelection(filePath string) {
	if fen.selected == nil {
		fen.selected = map[string]bool{}
	}

	fen.selected[filePath] = true
}

func (fen *Fen) GoLeft() {
	// Not sure if this is necessary
	if filepath.Dir(fen.wd) == filepath.Clean(fen.wd) {
		return
	}

	files, err := os.ReadDir(filepath.Dir(fen.wd))
	if err != nil || len(files) == 0 {
		return
	}

	fen.sel = fen.wd
	fen.wd = filepath.Dir(fen.wd)

	fen.DisableSelectingWithV()
}

func (fen *Fen) GoRight(app *tview.Application, openWith string) {
	if len(fen.middlePane.entries.Load().([]os.DirEntry)) <= 0 {
		return
	}

	fi, err := os.Stat(fen.sel)
	if err != nil {
		return
	}

	if !fi.IsDir() || openWith != "" {
		OpenFile(fen, app, openWith)
		return
	}

	/*	rightFiles, _ := os.ReadDir(fen.sel)
		if len(rightFiles) <= 0 {
			return
		}*/

	fen.wd = fen.sel
	fen.sel, err = fen.history.GetHistoryEntryForPath(fen.wd, fen.config.HiddenFiles)

	if err != nil {
		// FIXME
		// Uhh.. fix what?
		fen.sel = filepath.Join(fen.wd, fen.rightPane.GetSelectedEntryFromIndex(0))
	}

	fen.DisableSelectingWithV()
}

func (fen *Fen) GoUp(numEntries ...int) {
	numEntriesToMove := 1
	if len(numEntries) > 0 {
		numEntriesToMove = max(1, numEntries[0])
	}

	defer func() {
		if fen.selectingWithV {
			fen.selectingWithVEndIndex = max(0, fen.middlePane.selectedEntryIndex-numEntriesToMove)
		}
	}()

	if fen.middlePane.selectedEntryIndex-numEntriesToMove < 0 {
		fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(0))
		return
	}

	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.selectedEntryIndex-numEntriesToMove))
}

func (fen *Fen) GoDown(numEntries ...int) {
	numEntriesToMove := 1
	if len(numEntries) > 0 {
		numEntriesToMove = max(1, numEntries[0])
	}

	defer func() {
		if fen.selectingWithV {
			fen.selectingWithVEndIndex = min(len(fen.middlePane.entries.Load().([]os.DirEntry))-1, fen.middlePane.selectedEntryIndex+numEntriesToMove)
		}
	}()

	if fen.middlePane.selectedEntryIndex+numEntriesToMove >= len(fen.middlePane.entries.Load().([]os.DirEntry)) {
		fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(len(fen.middlePane.entries.Load().([]os.DirEntry))-1))
		return
	}

	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.selectedEntryIndex+numEntriesToMove))
}

// Does not do bounds checking, be careful!
func (fen *Fen) GoIndex(index int) {
	fen.sel = fen.middlePane.GetSelectedEntryFromIndex(index)

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = index
	}
}

func (fen *Fen) GoTop() {
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(0))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = 0
	}
}

func (fen *Fen) GoMiddle() {
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex((len(fen.middlePane.entries.Load().([]os.DirEntry))-1)/2))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = (len(fen.middlePane.entries.Load().([]os.DirEntry)) - 1) / 2
	}
}

func (fen *Fen) GoBottom() {
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(len(fen.middlePane.entries.Load().([]os.DirEntry))-1))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = len(fen.middlePane.entries.Load().([]os.DirEntry)) - 1
	}
}

func (fen *Fen) GoTopScreen() {
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.GetTopScreenEntryIndex()))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = fen.middlePane.GetTopScreenEntryIndex()
	}
}

func (fen *Fen) GoBottomScreen() {
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.GetBottomScreenEntryIndex()))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = fen.middlePane.GetBottomScreenEntryIndex()
	}
}

func (fen *Fen) PageUp() {
	_, _, _, height := fen.middlePane.Box.GetInnerRect()
	height = max(5, height-10) // Padding
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(max(0, fen.middlePane.selectedEntryIndex-height)))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = max(0, fen.middlePane.selectedEntryIndex-height)
	}
}

func (fen *Fen) PageDown() {
	_, _, _, height := fen.middlePane.Box.GetInnerRect()
	height = max(5, height-10) // Padding
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(min(len(fen.middlePane.entries.Load().([]os.DirEntry))-1, fen.middlePane.selectedEntryIndex+height)))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = min(len(fen.middlePane.entries.Load().([]os.DirEntry))-1, fen.middlePane.selectedEntryIndex+height)
	}
}

func (fen *Fen) GoSearchFirstMatch(searchTerm string) error {
	if searchTerm == "" {
		return errors.New("Empty search term")
	}

	for _, e := range fen.middlePane.entries.Load().([]os.DirEntry) {
		if strings.Contains(strings.ToLower(e.Name()), strings.ToLower(searchTerm)) {
			fen.sel = filepath.Join(fen.wd, e.Name())
			fen.selectingWithVEndIndex = fen.middlePane.GetSelectedIndexFromEntry(e.Name())
			return nil
		}
	}

	return errors.New("Nothing found")
}

func (fen *Fen) UpdateSelectingWithV() {
	if !fen.selectingWithV {
		return
	}

	fen.selectingWithVStartIndex = min(len(fen.middlePane.entries.Load().([]os.DirEntry))-1, max(0, fen.selectingWithVStartIndex))
	fen.selectingWithVEndIndex = min(len(fen.middlePane.entries.Load().([]os.DirEntry))-1, max(0, fen.selectingWithVEndIndex))

	minIndex := min(fen.selectingWithVStartIndex, fen.selectingWithVEndIndex)
	maxIndex := max(fen.selectingWithVStartIndex, fen.selectingWithVEndIndex)

	// We have to do this to copy fen.selectedBeforeSelectingWithV, and not a reference to it
	fen.selected = make(map[string]bool)
	for k, v := range fen.selectedBeforeSelectingWithV {
		fen.selected[k] = v
	}

	for i := minIndex; i <= maxIndex; i++ {
		fen.EnableSelection(fen.middlePane.GetSelectedPathFromIndex(i))
	}
}

func (fen *Fen) GoBookmark(bookmarkNumber int) error {
	if bookmarkNumber < 0 || bookmarkNumber > 9 {
		panic("Invalid bookmark number")
	}

	// This is so that pressing '0' uses the 10th bookmark index from config.lua
	if bookmarkNumber == 0 {
		bookmarkNumber = 9
	} else {
		bookmarkNumber--
	}

	pathToUse := fen.config.Bookmarks[bookmarkNumber]
	if pathToUse == "" {
		return errors.New("No path configured for bookmark " + strconv.Itoa(bookmarkNumber+1))
	}

	pathToUse = filepath.Clean(pathToUse)
	if !filepath.IsAbs(pathToUse) {
		var err error
		pathToUse, err = filepath.Abs(filepath.Join(fen.wd, pathToUse))
		if err != nil {
			return err
		}
	}

	stat, err := os.Stat(pathToUse)
	if err != nil {
		return errors.New("No such folder or file \"" + pathToUse + "\"")
	}

	if !stat.IsDir() {
		fen.wd = filepath.Dir(pathToUse)
		fen.sel = pathToUse
	} else {
		fen.wd = pathToUse
	}

	if filepath.Dir(fen.sel) != filepath.Clean(fen.sel) {
		fen.history.AddToHistory(fen.sel)
	}
	fen.UpdatePanes(false)

	fen.bottomBar.TemporarilyShowTextInstead("Moved to bookmark: \"" + pathToUse + "\"")
	return nil
}
