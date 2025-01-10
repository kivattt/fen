package main

//lint:file-ignore ST1005 some user-visible messages are stored in error values and thus occasionally require capitalization

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/kivattt/gogitstatus"
	"github.com/rivo/tview"
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

type Fen struct {
	app              *tview.Application
	wd               string // Current working directory
	lastWD           string
	sel              string
	lastSel          string
	lastInRepository string
	history          History

	selected     map[string]bool
	yankSelected map[string]bool
	yankType     string // "", "copy", "cut"

	selectingWithV               bool
	selectingWithVStartIndex     int
	selectingWithVEndIndex       int
	selectedBeforeSelectingWithV map[string]bool

	config                Config
	configFilePath        string // Config path as read by ReadConfig()
	fileOperationsHandler FileOperationsHandler
	gitStatusHandler      GitStatusHandler

	helpScreenVisible      *bool
	librariesScreenVisible *bool

	runningGitStatus bool

	folderFileCountCache map[string]int

	topBar     *TopBar
	bottomBar  *BottomBar
	leftPane   *FilesPane
	middlePane *FilesPane
	rightPane  *FilesPane

	showHomePathAsTilde bool
}

// gluamapper lets you use Go variables like "UiBorders", using the name "ui_borders".
// I happen to like this, but since I can't set the "fen" global to the actual Config value,
// we have to define them manually with a new table where I use these struct tags to look up the names
const luaTagName = "lua"

var ConfigKeysByTagNameNotToIncludeInOptionsMenu = []string{
	"no_write",       // Would be unsafe to allow disabling no-write (always assume fen --no-write is being ran by a bad actor)
	"terminal_title", // The push/pop terminal title escape codes don't work properly while fen is running
}

const (
	// SORT_NONE should only be used if fen is too slow loading big folders, because it messes with some things
	SORT_NONE           = "none" // TODO: Make SORT_NONE also disable the implicit sorting of os.ReadDir()
	SORT_ALPHABETICAL   = "alphabetical"
	SORT_MODIFIED       = "modified"
	SORT_SIZE           = "size"
	SORT_FILE_EXTENSION = "file-extension"
)

var ValidSortByValues = [...]string{SORT_NONE, SORT_ALPHABETICAL, SORT_MODIFIED, SORT_SIZE, SORT_FILE_EXTENSION}

const (
	HUMAN_READABLE = "human-readable"
	BYTES          = "bytes"
)

var ValidFileSizeFormatValues = [...]string{HUMAN_READABLE, BYTES}

func isInvalidFileSizeFormatValue(format string) bool {
	for _, e := range ValidFileSizeFormatValues {
		if format == e {
			return false
		}
	}

	return true
}

func isInvalidSortByValue(sortBy string) bool {
	for _, e := range ValidSortByValues {
		if sortBy == e {
			return false
		}
	}

	return true
}

type Config struct {
	UiBorders               bool                 `lua:"ui_borders"`
	Mouse                   bool                 `lua:"mouse"`
	NoWrite                 bool                 `lua:"no_write"`
	HiddenFiles             bool                 `lua:"hidden_files"`
	FoldersFirst            bool                 `lua:"folders_first"`
	SplitHomeEnd            bool                 `lua:"split_home_end"`
	PrintPathOnOpen         bool                 `lua:"print_path_on_open"`
	TerminalTitle           bool                 `lua:"terminal_title"`
	ShowHelpText            bool                 `lua:"show_help_text"`
	ShowHostname            bool                 `lua:"show_hostname"`
	Open                    []PreviewOrOpenEntry `lua:"open"`
	Preview                 []PreviewOrOpenEntry `lua:"preview"`
	SortBy                  string               `lua:"sort_by"` /* Valid values defined in ValidSortByValues */
	SortReverse             bool                 `lua:"sort_reverse"`
	FileEventIntervalMillis int                  `lua:"file_event_interval_ms"`
	AlwaysShowInfoNumbers   bool                 `lua:"always_show_info_numbers"`
	ScrollSpeed             int                  `lua:"scroll_speed"`
	Bookmarks               [10]string           `lua:"bookmarks"`
	GitStatus               bool                 `lua:"git_status"`
	PreviewSafetyBlocklist  bool                 `lua:"preview_safety_blocklist"`
	CloseOnEscape           bool                 `lua:"close_on_escape"`
	FileSizeInAllPanes      bool                 `lua:"file_size_in_all_panes"`
	FileSizeFormat          string               `lua:"file_size_format"` /* Valid values defined in ValidFileSizeFormatValues */
}

func NewConfigDefaultValues() Config {
	// Anything not specified here will have the default value for its type, e.g. false for booleans
	return Config{
		Mouse:                   true,
		FoldersFirst:            true,
		TerminalTitle:           true,
		ShowHelpText:            true,
		ShowHostname:            true,
		SortBy:                  SORT_ALPHABETICAL,
		FileEventIntervalMillis: 300,
		ScrollSpeed:             2,
		PreviewSafetyBlocklist:  true,
		FileSizeFormat:          HUMAN_READABLE,
	}
}

// To prevent previewing sensitive files
var DefaultPreviewBlocklistCaseInsensitive = []string{
	// Filezilla passwords
	"sitemanager.xml",
	"filezilla.xml",

	// Other
	".gitconfig",
	".bash_history",
	".python_history",

	// Tokens
	".env",

	// Possible private keys
	"*.key",

	".Xauthority",

	"*.p12",
	"*.pfx",
	"*.pkcs12",
	"*.pri",
	"*.cer",
	"*.der",
	"*.pem",
	"*.p7a",
	"*.p7b",
	"*.p7c",
	"*.p7r",
	"*.spc",
	"*.p8",

	// Reaper license key
	"*.rk",

	// Databases
	"*.db",
	"*.accdb",
	"*.mdb",
	"*.mdf",
	"*.sqlite*",

	"*.bak",

	// Dataset
	"*.parquet",
}

type PreviewOrOpenEntry struct {
	Script     string
	Program    []string // The name used to be "Programs", but this makes more sense for the lua configuration
	Match      []string
	DoNotMatch []string
}

type PanePos int

const (
	LeftPane PanePos = iota
	MiddlePane
	RightPane
)

func (fen *Fen) Init(path string, app *tview.Application, helpScreenVisible *bool, librariesScreenVisible *bool) error {
	fen.app = app
	fen.fileOperationsHandler = FileOperationsHandler{fen: fen}
	fen.folderFileCountCache = make(map[string]int)

	fen.gitStatusHandler = GitStatusHandler{app: app, fen: fen}
	fen.gitStatusHandler.Init()

	fen.helpScreenVisible = helpScreenVisible
	fen.librariesScreenVisible = librariesScreenVisible
	fen.showHomePathAsTilde = true

	if fen.selected == nil {
		fen.selected = map[string]bool{}
	}

	fen.yankSelected = map[string]bool{}

	fen.selectedBeforeSelectingWithV = map[string]bool{}

	fen.wd = path
	fen.sel = path // fen.sel has to be set so fen.UpdatePanes() doesn't panic, it's set accordingly when fen.UpdatePanes() completes.

	fen.topBar = NewTopBar(fen)

	fen.leftPane = NewFilesPane(fen, LeftPane)
	fen.middlePane = NewFilesPane(fen, MiddlePane)
	fen.rightPane = NewFilesPane(fen, RightPane)

	fen.leftPane.Init()
	fen.middlePane.Init()
	fen.rightPane.Init()

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
		fen.GoTop(true)

		if shouldSelectSpecifiedFile {
			fen.sel = path
		}
	}

	fen.history.AddToHistory(fen.sel)
	fen.UpdatePanes(false)

	return err
}

func (fen *Fen) Fini() {
	fen.leftPane.fileWatcher.Close()
	fen.middlePane.fileWatcher.Close()
	fen.rightPane.fileWatcher.Close()

	fen.gitStatusHandler.gitIndexFileWatcher.Close()

	close(fen.gitStatusHandler.channel)
	fen.gitStatusHandler.wg.Wait()
}

func (fen *Fen) InvalidateFolderFileCountCache() {
	fen.folderFileCountCache = make(map[string]int)
}

func (fen *Fen) PushAndSetTerminalTitle() {
	if runtime.GOOS == "linux" {
		os.Stderr.WriteString("\x1b[22t")                       // Push current terminal title
		os.Stderr.WriteString("\x1b]0;fen " + version + "\x07") // Set terminal title to "fen <version>"
	}
}

func (fen *Fen) PopTerminalTitle() {
	if runtime.GOOS == "linux" {
		os.Stderr.WriteString("\x1b[23t") // Pop terminal title, sets it back to normal
	}
}

func (fen *Fen) ReadConfig(path string) error {
	fen.config = NewConfigDefaultValues()
	fen.configFilePath = path

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

	if err == nil {
		luaInitialConfigTable.RawSetString("config_path", lua.LString(PathWithEndSeparator(filepath.Dir(fen.configFilePath))))
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

	if !filepath.IsAbs(fen.sel) {
		panic("fen.sel was not an absolute path")
	}

	// TODO: Preserve last available selection index (so it doesn't reset to the top)
	_, err := os.Stat(fen.wd)
	for err != nil {
		if filepath.Dir(fen.wd) == fen.wd {
			panic("Could not find usable parent path")
		}

		fen.wd = filepath.Dir(fen.wd)
		_, err = os.Stat(fen.wd)
	}

	fen.leftPane.SetBorder(fen.config.UiBorders)
	fen.middlePane.SetBorder(fen.config.UiBorders)
	fen.rightPane.SetBorder(fen.config.UiBorders)

	if fen.wd != fen.lastWD {
		// Has to happen before the filespane ChangeDir() calls which will repopulate the cache
		fen.InvalidateFolderFileCountCache()
	}
	defer func() {
		fen.lastWD = fen.wd
	}()

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

	selStat, selStatErr := os.Lstat(fen.sel)
	if selStatErr != nil {
		return
	}

	// Overwrite the cached folder file count for the currently selected folder
	// If fen.wd changed, we already invalidated the cache so this isn't needed
	if fen.wd == fen.lastWD && selStat.IsDir() {
		count, err := FolderFileCount(fen.sel, fen.config.HiddenFiles)
		if err == nil {
			fen.folderFileCountCache[fen.sel] = count
		}
	}

	if !fen.config.GitStatus {
		return
	}

	defer func() {
		fen.lastSel = fen.sel
	}()

	// If the current Git repository changed or fen.sel is a directory, ask for a git status
	if fen.sel != fen.lastSel {
		var inRepository string
		if selStat.IsDir() {
			inRepository, err = fen.gitStatusHandler.TryFindParentGitRepository(fen.sel)
		} else {
			inRepository, err = fen.gitStatusHandler.TryFindParentGitRepository(fen.wd)
		}

		if err != nil {
			// When we're no longer in a Git repository, set empty so it can ask for a git status next time we enter one
			fen.lastInRepository = ""
		}

		if !selStat.IsDir() && err != nil {
			return
		}

		// Seems like the fsnotify events don't catch up on FreeBSD, need to always trigger a Git status
		if selStat.IsDir() || inRepository != fen.lastInRepository || runtime.GOOS == "freebsd" {
			fen.TriggerGitStatus() // TODO: Fix redundant os.Lstat() and TryFindParentGitRepository calls...
		}

		fen.lastInRepository = inRepository
	}
}

// Ask the git status handler to run a "git status" at the currently selected path.
// It may choose to ignore the request if for example, it would restart a git status on the same path or fen.git_status is false.
func (fen *Fen) TriggerGitStatus() {
	if !fen.config.GitStatus {
		return
	}

	stat, err := os.Lstat(fen.sel)
	if err != nil {
		return
	}

	var currentRepository string
	if stat.IsDir() {
		currentRepository, err = fen.gitStatusHandler.TryFindParentGitRepository(fen.sel)
	} else {
		currentRepository, err = fen.gitStatusHandler.TryFindParentGitRepository(fen.wd)
	}

	if err != nil {
		return
	}

	if currentRepository != fen.lastInRepository {
		// Remove previous watched path
		watchList := fen.gitStatusHandler.gitIndexFileWatcher.WatchList()
		if watchList == nil {
			return
		}

		for _, e := range watchList {
			fen.gitStatusHandler.gitIndexFileWatcher.Remove(e)
		}

		// Watch the new path
		fen.gitStatusHandler.gitIndexFileWatcher.Add(filepath.Join(currentRepository, ".git"))
	}

	if err == nil {
		if stat.IsDir() {
			fen.gitStatusHandler.channel <- fen.sel
		} else {
			fen.gitStatusHandler.channel <- fen.wd
		}
	}
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
		err := OpenFile(fen, app, openWith)
		if err != nil {
			fen.bottomBar.TemporarilyShowTextInstead(err.Error())
		}
		return
	}

	/*	rightFiles, _ := os.ReadDir(fen.sel)
		if len(rightFiles) <= 0 {
			return
		}*/

	fen.wd = fen.sel
	fen.sel, err = fen.history.GetHistoryEntryForPath(fen.wd, fen.config.HiddenFiles)

	if err != nil {
		fen.sel = filepath.Join(fen.wd, fen.rightPane.GetSelectedEntryFromIndex(0))
	}

	fen.DisableSelectingWithV()
}

// Returns false if nothing happened (already at the top, would've moved to the same position)
func (fen *Fen) GoUp(numEntries ...int) bool {
	if fen.middlePane.selectedEntryIndex <= 0 {
		return false
	}

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
		return true
	}

	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.selectedEntryIndex-numEntriesToMove))
	return true
}

// Returns false if nothing happened (already at the bottom, would've moved to the same position)
func (fen *Fen) GoDown(numEntries ...int) bool {
	if fen.middlePane.selectedEntryIndex >= len(fen.middlePane.entries.Load().([]os.DirEntry))-1 {
		return false
	}

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
		return true
	}

	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.selectedEntryIndex+numEntriesToMove))
	return true
}

// Returns false if nothing happened (already at the top, would've moved to the same position)
// If called as GoTop(true), it will always update fen.sel and return true (used in fen.Init() and fen.GoPath())
func (fen *Fen) GoTop(force ...bool) bool {
	if !(len(force) > 0 && force[0]) && fen.middlePane.selectedEntryIndex <= 0 {
		return false
	}

	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(0))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = 0
	}

	return true
}

func (fen *Fen) GoMiddle() {
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex((len(fen.middlePane.entries.Load().([]os.DirEntry))-1)/2))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = (len(fen.middlePane.entries.Load().([]os.DirEntry)) - 1) / 2
	}
}

// Returns false if nothing happened (already at the bottom, would've moved to the same position)
func (fen *Fen) GoBottom() bool {
	if fen.middlePane.selectedEntryIndex >= len(fen.middlePane.entries.Load().([]os.DirEntry))-1 {
		return false
	}

	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(len(fen.middlePane.entries.Load().([]os.DirEntry))-1))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = len(fen.middlePane.entries.Load().([]os.DirEntry)) - 1
	}

	return true
}

// Only meant to be used when fen.config.FoldersFirst is true, if not it will panic
// If a folder not at the bottom is selected, go to the bottom folder, otherwise go to the bottom
// Returns false if nothing happened (already at the bottom, where calling fen.GoBottom() would've moved to the same position)
func (fen *Fen) GoBottomFolderOrBottom() bool {
	if !fen.config.FoldersFirst {
		panic("GoBottomFolderOrBottom() was called with FoldersFirst disabled")
	}

	stat, err := os.Lstat(fen.sel)
	if err != nil {
		return true
	}

	findBottomFolder := func() (int, error) {
		for i := fen.middlePane.selectedEntryIndex; i < len(fen.middlePane.entries.Load().([]os.DirEntry)); i++ {
			if fen.middlePane.entries.Load().([]os.DirEntry)[i].IsDir() {
				continue
			}

			bottomFolderIndex := fen.middlePane.ClampEntryIndex(max(0, i-1))

			if bottomFolderIndex == fen.middlePane.selectedEntryIndex {
				return 0, errors.New("Bottom folder already selected")
			}
			return bottomFolderIndex, nil
		}

		return 0, errors.New("No folder found")
	}

	if stat.IsDir() {
		bottomFolder, err := findBottomFolder()
		if err != nil {
			return fen.GoBottom()
		}

		fen.GoIndex(bottomFolder)
	} else {
		return fen.GoBottom()
	}

	return true
}

// Only meant to be used when fen.config.FoldersFirst is true, if not it will panic
// If a file not at the top is selected, go to the top file, otherwise go to the top
// Returns false if nothing happened (already at the top, where calling fen.GoTop() would've moved to the same position)
func (fen *Fen) GoTopFileOrTop() bool {
	if !fen.config.FoldersFirst {
		panic("GoTopFileOrTop() was called with FoldersFirst disabled")
	}

	if fen.middlePane.selectedEntryIndex <= 0 {
		return false
	}

	stat, err := os.Lstat(fen.sel)
	if err != nil {
		return true
	}

	findTopFile := func() (int, error) {
		for i := fen.middlePane.selectedEntryIndex; i >= 0; i-- {
			if !fen.middlePane.entries.Load().([]os.DirEntry)[i].IsDir() {
				continue
			}

			topFileIndex := fen.middlePane.ClampEntryIndex(i + 1)

			if topFileIndex == fen.middlePane.selectedEntryIndex {
				return 0, errors.New("Top file already selected")
			}
			return topFileIndex, nil
		}

		return 0, errors.New("No file found")
	}

	if !stat.IsDir() {
		topFile, err := findTopFile()
		if err != nil {
			return fen.GoTop()
		}

		fen.GoIndex(topFile)
	} else {
		return fen.GoTop()
	}

	return true
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

	path := fen.config.Bookmarks[bookmarkNumber]
	if path == "" {
		return errors.New("No path configured for bookmark " + strconv.Itoa(bookmarkNumber+1))
	}

	pathMovedTo, err := fen.GoPath(path)
	if err != nil {
		return err
	}

	fen.DisableSelectingWithV()

	fen.bottomBar.TemporarilyShowTextInstead("Moved to bookmark: \"" + pathMovedTo + "\"")
	return nil
}

func (fen *Fen) GoIndex(index int) {
	clampedIndex := fen.middlePane.ClampEntryIndex(index)
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(clampedIndex))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = clampedIndex
	}
}

// Returns the absolute path that was moved to, unless there is an error.
// On completion, it always adds fen.sel to the history.
// Implicitly calls fen.UpdatePanes(false) when no error.
func (fen *Fen) GoPath(path string) (string, error) {
	// TODO: Add an option to not enter directories

	/* PLUGINS:
	 * We should fen.UpdatePanes(true) when we can't find the newPath in the current middlePane (for going to path on renaming)
	 * This would slow down "Goto path" when going to a non-existent path, but would guarantee this function always works predictably
	 * so that Lua plugins won't have to manually call fen.UpdatePanes(true) before a fen.GoPath() for recently renamed/created/etc. files
	 */
	if path == "" {
		return "", errors.New("Empty path provided")
	}

	pathToUse := filepath.Clean(path)
	if !filepath.IsAbs(pathToUse) {
		var err error
		pathToUse, err = filepath.Abs(filepath.Join(fen.wd, pathToUse))
		if err != nil {
			return "", err
		}
	}

	stat, err := os.Lstat(pathToUse)
	if err != nil {
		return "", errors.New("No such file or directory \"" + pathToUse + "\"")
	}

	if stat.IsDir() {
		if pathToUse != fen.wd {
			fen.DisableSelectingWithV()
		}
	} else {
		if filepath.Dir(pathToUse) != fen.wd {
			fen.DisableSelectingWithV()
		}
	}

	if stat.IsDir() {
		fen.wd = pathToUse
		h, err := fen.history.GetHistoryEntryForPath(pathToUse, fen.config.HiddenFiles)
		if err != nil {
			fen.UpdatePanes(false) // Need to do this first so the new selected path is added to history
			fen.GoTop(true)
		} else {
			fen.sel = h
		}
	} else {
		fen.wd = filepath.Dir(pathToUse)
		fen.sel = pathToUse
	}

	// XXX: Always adds to history when not at the root path
	if filepath.Dir(fen.sel) != filepath.Clean(fen.sel) {
		fen.history.AddToHistory(fen.sel)
	}

	if fen.selectingWithV {
		fen.UpdatePanes(false) // Have to update panes to get the selectedEntryIndex...
		fen.selectingWithVEndIndex = fen.middlePane.selectedEntryIndex
	}

	fen.UpdatePanes(false)

	return pathToUse, nil
}

func (fen *Fen) GoRootPath() {
	var path string
	if runtime.GOOS == "windows" {
		path = filepath.VolumeName(fen.sel) + string(os.PathSeparator)
	} else {
		path = "/"
	}
	fen.GoPath(path)
}

// Goes to the path furthest down in the history
func (fen *Fen) GoRightUpToHistory() {
	path, err := fen.history.GetHistoryFullPath(fen.sel, fen.config.HiddenFiles)
	if err != nil {
		return
	}

	path = filepath.Dir(path)

	rel, err := filepath.Rel(fen.sel, path)
	if err != nil {
		return
	}

	// If it would end up going to the left, return
	if strings.HasPrefix(rel, "..") {
		return
	}

	fen.GoPath(path)
}

// This goes to the changed file (any non-folder) closest to the root path of repoPath.
// If there are multiple candidates, it will select the one with the shortest filepath.
// If there are some filepaths of equal length, it will choose one randomly.
// TODO: Implement sorting function in gogitstatus so this is deterministic
func (fen *Fen) GoRightUpToFirstUnstagedOrUntracked(repoPath, currentPath string) error {
	fen.gitStatusHandler.trackedLocalGitReposMutex.Lock()
	defer fen.gitStatusHandler.trackedLocalGitReposMutex.Unlock()

	repo, ok := fen.gitStatusHandler.trackedLocalGitRepos[repoPath]
	if !ok {
		return errors.New("Not in a tracked local Git repository")
	}

	changedFileClosestToRoot := ""
	shortestPathSeparatorCount := 0
	for changedFilePath := range gogitstatus.ExcludingDirectories(repo.changedFiles) {
		bruhRel, bruhErr := filepath.Rel(repoPath, currentPath)
		if bruhErr != nil {
			continue
		}

		rel, err := filepath.Rel(bruhRel, changedFilePath)
		if err != nil {
			continue
		}

		if strings.HasPrefix(rel, "..") {
			continue
		}

		pathSeparatorCount := strings.Count(rel, string(os.PathSeparator))
		if changedFileClosestToRoot == "" || pathSeparatorCount < shortestPathSeparatorCount || (pathSeparatorCount == shortestPathSeparatorCount && len(rel) < len(changedFileClosestToRoot)) {
			changedFileClosestToRoot = rel
			shortestPathSeparatorCount = pathSeparatorCount
		}
	}

	if changedFileClosestToRoot == "" {
		return errors.New("No unstaged/untracked path found")
	}

	_, err := fen.GoPath(filepath.Join(currentPath, changedFileClosestToRoot))
	return err
}

// Resolves the symlink and uses fen.GoPath() under the hood
func (fen *Fen) GoSymlink(symlinkPath string) error {
	// Should not happen
	if !filepath.IsAbs(symlinkPath) {
		return errors.New("Selected file was not an absolute path")
	}

	target, err := os.Readlink(fen.sel)
	if err != nil {
		return errors.New("Unable to readlink selected file")
	}

	// FIXME: GoPath() enters directories, when we don't want to (need to Lstat the target)
	_, err = fen.GoPath(target)
	return err
}

func (fen *Fen) BulkRename(app *tview.Application) error {
	if fen.config.NoWrite {
		return errors.New("Can't bulkrename in no-write mode")
	}

	tempFile, err := os.CreateTemp("", "fenrename*.txt") // .txt for auto-detect what editor to use on Windows
	if err != nil {
		return err
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	preRenameList := []string{}

	if fen.middlePane.folder != fen.wd {
		panic("In BulkRename(): fen.middlePane.folder was not equal to fen.wd")
	}

	// Write paths to the temporary file
	if len(fen.selected) > 0 {
		// We loop over the middlepane entries because they are presumably already sorted
		fen.middlePane.FilterAndSortEntries()
		for _, entry := range fen.middlePane.entries.Load().([]os.DirEntry) {
			entryFullPath := filepath.Join(fen.middlePane.folder, entry.Name())

			_, selected := fen.selected[entryFullPath]
			if !selected {
				continue
			}

			// Only bulkrename selected files in the current working directory
			if filepath.Dir(entryFullPath) != fen.wd {
				panic("In BulkRename(): a selected path was not within fen.wd")
			}

			basePath := filepath.Base(entryFullPath)

			if strings.ContainsRune(basePath, '\n') {
				return errors.New("A selected path contains a newline, unable to bulkrename")
			}

			preRenameList = append(preRenameList, basePath)
		}
	} else {
		// Only bulkrename selected files in the current working directory
		if filepath.Dir(fen.sel) != fen.wd {
			return nil
		}

		basePath := filepath.Base(fen.sel)

		if strings.ContainsRune(basePath, '\n') {
			return errors.New("Path contains a newline, unable to bulkrename")
		}

		preRenameList = append(preRenameList, basePath)
	}

	if len(preRenameList) == 0 {
		panic("In BulkRename(): preRenameList was empty")
	}

	for _, basePath := range preRenameList {
		_, err := tempFile.WriteString(basePath + "\n")
		if err != nil {
			return err
		}
	}
	tempFile.Close()

	preRenameHashsum, err := SHA256HashSum(tempFile.Name())
	if err != nil {
		return err
	}

	app.Suspend(func() {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			// FIXME: The Windows program picker (below) runs in the background, making it impossible to bulkrename
			// Find a way to wait for it to exit, so we don't have to force the user to use notepad...
			//cmd = exec.Command(filepath.Join(os.Getenv("SYSTEMROOT"), "System32", "rundll32.exe"), "url.dll,FileProtocolHandler", tempFile.Name())

			cmd = exec.Command("notepad", tempFile.Name())
		} else {
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi" // vi is symlinked to vim on macOS, so it should work there aswell
			}
			cmd = exec.Command(editor, tempFile.Name())
		}
		cmd.Dir = fen.wd
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		_ = cmd.Run()
	})

	postRenameHashSum, err := SHA256HashSum(tempFile.Name())
	if err != nil {
		return errors.New("Nothing renamed! Was the temporary file deleted?")
	}

	if reflect.DeepEqual(preRenameHashsum, postRenameHashSum) {
		return errors.New("Nothing renamed!")
	}

	postRenameList := []string{}
	file, err := os.Open(tempFile.Name())
	if err != nil {
		return errors.New("Nothing renamed! Was the temporary file deleted?")
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, string(os.PathSeparator)) {
			return errors.New("Nothing renamed! Because a path contained a path separator \"" + string(os.PathSeparator) + "\"")
		}
		postRenameList = append(postRenameList, line)
	}

	if len(preRenameList) != len(postRenameList) {
		return errors.New("Nothing renamed! Wanted " + strconv.Itoa(len(preRenameList)) + " but got " + strconv.Itoa(len(postRenameList)) + " lines")
	}

	if reflect.DeepEqual(preRenameList, postRenameList) {
		// preRenameList equals postRenameList despite sha256 hashsum differing
		// This can happen due to the file being saved with carriage returns before newlines
		//  or more likely, the last line is missing a newline (it is visible and can be removed in notepad)
		return errors.New("Nothing renamed!")
	}

	firstDuplicate, err := StringSliceHasDuplicate(postRenameList)
	if err == nil {
		return errors.New("Nothing renamed! Duplicate filename \"" + firstDuplicate + "\"")
	}

	for i, e := range postRenameList {
		if e == "" {
			return errors.New("Nothing renamed! Empty filename for \"" + preRenameList[i] + "\"")
		}
	}

	shouldBulkRenamePrompt := false
	app.Suspend(func() {
		fmt.Print("Bulk-rename on " + strconv.Itoa(len(preRenameList)) + " files? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		confirmation, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		if strings.ToLower(strings.TrimSpace(confirmation)) == "y" {
			shouldBulkRenamePrompt = true
		}
	})

	if !shouldBulkRenamePrompt {
		return errors.New("Nothing renamed! Cancelled in prompt")
	}

	/* Remove unchanged entries from both preRenameList and postRenameList */
	// Code adapted from https://cs.opensource.google/go/go/+/refs/tags/go1.23.2:src/slices/slices.go;l=236
	i := 0
	for j := range preRenameList {
		// Move the changed entries we want to the front
		if preRenameList[j] != postRenameList[j] {
			preRenameList[i] = preRenameList[j]
			postRenameList[i] = postRenameList[j]
			i++
		}
	}

	// Zero out the removed values for GC
	clear(preRenameList[i:])
	clear(postRenameList[i:])

	// Remove the unchanged entries
	preRenameList = preRenameList[:i]
	postRenameList = postRenameList[:i]

	/* Generate new random names, if one collides with an already existing file then return an error */
	preRenameRandomNames := make([]string, len(preRenameList))
	for i := range preRenameList {
		randomName := "fen_" + RandomStringPathSafe(14) // 14 characters (pow(36, 14) combinations), only lowercase letters a-z and 0-9 numbers
		_, err := os.Lstat(filepath.Join(fen.wd, randomName))
		if err == nil {
			return errors.New("Nothing renamed! Random path \"" + randomName + "\" would've overwritten a file")
		}

		preRenameRandomNames[i] = randomName
	}

	if len(preRenameList) != len(preRenameRandomNames) {
		panic("In BulkRename(): preRenameList and preRenameRandomNames have unequal lengths")
	}

	/* Rename preRenameList files to their new random names */
	for i := range preRenameRandomNames {
		oldName := filepath.Join(fen.wd, preRenameList[i])
		newRandomName := filepath.Join(fen.wd, preRenameRandomNames[i])
		_, err := os.Lstat(newRandomName)
		if err == nil {
			panic("In BulkRename(): Would've overwritten a file: \"" + newRandomName + "\"")
		}

		err = os.Rename(oldName, newRandomName)
		if err != nil {
			return errors.New("Failed to rename \"" + preRenameList[i] + "\" to the random name \"" + preRenameRandomNames[i] + "\"")
		}
	}

	/* Rename preRenameRandomNames to their new correct names */
	fen.DisableSelectingWithV()
	fen.selected = make(map[string]bool)
	fen.yankSelected = make(map[string]bool)

	nFilesRenamed := 0
	nFilesRenamedFail := 0
	j := 0
	for i := 0; i < len(postRenameList); i++ {
		oldName := preRenameRandomNames[i]
		newName := postRenameList[i]

		if newName == oldName {
			continue
		}

		oldNameAbs := filepath.Join(fen.wd, oldName)
		newNameAbs := filepath.Join(fen.wd, newName)

		if !filepath.IsAbs(oldNameAbs) || !filepath.IsAbs(newNameAbs) {
			panic("In BulkRename(): Old random name or new name was a non-absolute path")
		}

		// Don't overwrite an existing file, rename back to the original name
		_, err := os.Lstat(newNameAbs)
		if err == nil {
			preRenameAbs := filepath.Join(fen.wd, preRenameList[i])
			_ = os.Rename(oldNameAbs, preRenameAbs)

			// We can't use fen.GoPath() here because it would enter directories
			fen.sel = preRenameAbs
			fen.middlePane.SetSelectedEntryFromString(filepath.Base(preRenameAbs)) // fen.UpdatePanes() overwrites fen.sel, so we have to set the index
			fen.history.AddToHistory(preRenameAbs)
			fen.UpdatePanes(true) // Need to force a read dir so the new entry is in the filespane for fen.GoPath

			nFilesRenamedFail++
			continue
		}

		err = os.Rename(oldNameAbs, newNameAbs)
		if err != nil {
			nFilesRenamedFail++
			continue
		}

		// This is also done by file system events, but let's be safe
		fen.history.RemoveFromHistory(oldNameAbs)

		// Select the new name of the first renamed path
		if j == 0 {
			// We can't use fen.GoPath() here because it would enter directories
			fen.UpdatePanes(true) // Need to force a read dir so the new entry is in the filespane
			fen.sel = newNameAbs
			fen.middlePane.SetSelectedEntryFromString(filepath.Base(newNameAbs)) // fen.UpdatePanes() overwrites fen.sel, so we have to set the index
			fen.history.AddToHistory(newNameAbs)
		}
		j++

		nFilesRenamed++
	}

	str := ""
	if nFilesRenamed == 0 {
		str = "Nothing renamed!"
	} else {
		str = "Renamed " + strconv.Itoa(nFilesRenamed)
		if nFilesRenamed == 1 {
			str += " file"
		} else {
			str += " files"
		}
	}

	if nFilesRenamedFail > 0 {
		str += " (" + strconv.Itoa(nFilesRenamedFail) + " failed)"
	}

	fen.bottomBar.TemporarilyShowTextInstead(str)
	return nil
}
