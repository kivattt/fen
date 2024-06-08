package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rivo/tview"
)

type Fen struct {
	wd      string
	sel     string
	history History

	selected     []string
	yankSelected []string
	yankType     string // "", "copy", "cut"

	selectingWithV               bool
	selectingWithVStartIndex     int
	selectingWithVEndIndex       int
	selectedBeforeSelectingWithV []string

	bottomBarText string

	config Config

	topPane    *Bar
	leftPane   *FilesPane
	middlePane *FilesPane
	rightPane  *FilesPane
	bottomPane *Bar

	fileProperties *FileProperties
}

type Config struct {
	UiBorders           bool               `json:"ui-borders"`
	NoMouse             bool               `json:"no-mouse"`
	NoWrite             bool               `json:"no-write"`
	DontShowHiddenFiles bool               `json:"dont-show-hidden-files"`
	FoldersNotFirst     bool               `json:"folders-not-first"`
	PrintPathOnOpen     bool               `json:"print-path-on-open"`
	OpenWith            []OpenWithEntry    `json:"open-with"`
	PreviewWith         []PreviewWithEntry `json:"preview-with"`
	DontChangeTerminalTitle bool `json:"dont-change-terminal-title"`
}

type OpenWithEntry struct {
	Programs   []string `json:"programs"`
	Match      []string `json:"match"`
	DoNotMatch []string `json:"do-not-match"`
}

type PreviewWithEntry struct {
	Script     string   `json:"script"`
	Programs   []string `json:"programs"`
	Match      []string `json:"match"`
	DoNotMatch []string `json:"do-not-match"`
}

func (fen *Fen) Init(workingDirectory string) error {
	fen.selectingWithV = false
	fen.fileProperties = NewFileProperties()

	fen.wd = workingDirectory

	fen.topPane = NewBar(&fen.sel, &fen.sel, &fen.config.NoWrite)
	fen.topPane.isTopBar = true

	fen.leftPane = NewFilesPane(fen, false, false)
	fen.middlePane = NewFilesPane(fen, true, false)
	fen.rightPane = NewFilesPane(fen, false, true)

	if fen.config.UiBorders {
		fen.leftPane.SetBorder(true)
		fen.middlePane.SetBorder(true)
		fen.rightPane.SetBorder(true)
	}

	fen.bottomPane = NewBar(&fen.bottomBarText, &fen.sel, &fen.config.NoWrite)

	wdFiles, err := os.ReadDir(fen.wd)
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
		fen.middlePane.SetEntries(fen.wd, fen.config.FoldersNotFirst)
		fen.GoTop()
	}

	fen.history.AddToHistory(fen.sel)
	fen.UpdatePanes()

	return err
}

func (fen *Fen) ReadConfig(path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		// We don't want to exit if there is no config file
		// This should really be checked by the caller...
		return nil
	}

	err = json.Unmarshal(bytes, &fen.config)
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
	fen.selectingWithVStartIndex = fen.middlePane.selectedEntry
	fen.selectingWithVEndIndex = fen.selectingWithVStartIndex
	fen.selectedBeforeSelectingWithV = fen.selected
}

func (fen *Fen) DisableSelectingWithV() {
	if !fen.selectingWithV {
		return
	}

	fen.selectingWithV = false
	fen.selectedBeforeSelectingWithV = []string{}
}

func (fen *Fen) UpdatePanes() {
	fen.leftPane.SetEntries(filepath.Dir(fen.wd), fen.config.FoldersNotFirst)
	fen.middlePane.SetEntries(fen.wd, fen.config.FoldersNotFirst)

	if fen.wd != filepath.Dir(fen.wd) {
		fen.leftPane.SetSelectedEntryFromString(filepath.Base(fen.wd))
	} else {
		fen.leftPane.entries = []os.DirEntry{}
	}

	username, groupname, err := FileUserAndGroupName(fen.sel)
	fileOwners := ""
	if err == nil {
		fileOwners = " " + UsernameWithColor(username) + ":" + GroupnameWithColor(groupname)
	}
	filePermissions, _ := FilePermissionsString(fen.sel)
	fileLastModified, _ := FileLastModifiedString(fen.sel)
	fen.bottomBarText = "[teal:]" + filePermissions + fileOwners + " [default:]" + fileLastModified

	fen.middlePane.SetSelectedEntryFromString(filepath.Base(fen.sel))

	// FIXME: Generic bounds checking across all panes in this function
	if fen.middlePane.selectedEntry >= len(fen.middlePane.entries) {
		if len(fen.middlePane.entries) > 0 {
			fen.sel = fen.middlePane.GetSelectedEntryFromIndex(len(fen.middlePane.entries) - 1)
			fen.middlePane.SetSelectedEntryFromString(filepath.Base(fen.sel)) // Duplicated from above...
		}
	}

	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.selectedEntry))
	fen.rightPane.SetEntries(fen.sel, fen.config.FoldersNotFirst)

	// Prevents showing 'empty' a second time in rightPane, if middlePane is already showing 'empty'
	if len(fen.middlePane.entries) <= 0 {
		fen.rightPane.parentIsEmptyFolder = false
	}

	h, err := fen.history.GetHistoryEntryForPath(fen.sel, fen.config.DontShowHiddenFiles)
	if err != nil {
		fen.rightPane.SetSelectedEntryFromIndex(0)
	} else {
		fen.rightPane.SetSelectedEntryFromString(filepath.Base(h))
	}

	fen.UpdateSelectingWithV()

	if fen.fileProperties.visible {
		fen.fileProperties.UpdateTable(fen.sel)
	}
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

func (fen *Fen) EnableSelection(filePath string) {
	if index := slices.Index(fen.selected, filePath); index != -1 {
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

	fen.selectingWithV = false
	fen.selectedBeforeSelectingWithV = []string{}
}

func (fen *Fen) GoRight(app *tview.Application, openWith string) {
	if len(fen.middlePane.entries) <= 0 {
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
	fen.sel, err = fen.history.GetHistoryEntryForPath(fen.wd, fen.config.DontShowHiddenFiles)

	if err != nil {
		// FIXME
		fen.sel = filepath.Join(fen.wd, fen.rightPane.GetSelectedEntryFromIndex(0))
	}

	fen.selectingWithV = false
	fen.selectedBeforeSelectingWithV = []string{}
}

func (fen *Fen) GoUp() {
	if fen.middlePane.selectedEntry-1 < 0 {
		fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(0))
		return
	}

	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.selectedEntry-1))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = fen.middlePane.selectedEntry - 1 // Strange, but it works
	}
}

func (fen *Fen) GoDown() {
	if fen.middlePane.selectedEntry+1 >= len(fen.middlePane.entries) {
		fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(len(fen.middlePane.entries)-1))
		return
	}

	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.selectedEntry+1))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = fen.middlePane.selectedEntry + 1 // Strange, but it works
	}
}

func (fen *Fen) GoTop() {
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(0))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = 0 // Strange, but it works
	}
}

func (fen *Fen) GoMiddle() {
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex((len(fen.middlePane.entries)-1)/2))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = (len(fen.middlePane.entries) - 1) / 2 // Strange, but it works
	}
}

func (fen *Fen) GoBottom() {
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(len(fen.middlePane.entries)-1))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = len(fen.middlePane.entries) - 1 // Strange, but it works
	}
}

func (fen *Fen) GoTopScreen() {
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.GetTopScreenEntryIndex()))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = fen.middlePane.GetTopScreenEntryIndex() // Strange, but it works
	}
}

func (fen *Fen) GoBottomScreen() {
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(fen.middlePane.GetBottomScreenEntryIndex()))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = fen.middlePane.GetBottomScreenEntryIndex() // Strange, but it works
	}
}

func (fen *Fen) PageUp() {
	_, _, _, height := fen.middlePane.Box.GetInnerRect()
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(max(0, fen.middlePane.selectedEntry-height)))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = max(0, fen.middlePane.selectedEntry-height) // Strange, but it works
	}
}

func (fen *Fen) PageDown() {
	_, _, _, height := fen.middlePane.Box.GetInnerRect()
	fen.sel = filepath.Join(fen.wd, fen.middlePane.GetSelectedEntryFromIndex(min(len(fen.middlePane.entries)-1, fen.middlePane.selectedEntry+height)))

	if fen.selectingWithV {
		fen.selectingWithVEndIndex = min(len(fen.middlePane.entries)-1, fen.middlePane.selectedEntry+height) // Strange, but it works
	}
}

func (fen *Fen) GoSearchFirstMatch(searchTerm string) error {
	if searchTerm == "" {
		return errors.New("Empty search term")
	}

	for _, e := range fen.middlePane.entries {
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

	minIndex := min(fen.selectingWithVStartIndex, fen.selectingWithVEndIndex)
	maxIndex := max(fen.selectingWithVStartIndex, fen.selectingWithVEndIndex)

	fen.selected = fen.selectedBeforeSelectingWithV
	for i := minIndex; i <= maxIndex; i++ {
		fen.EnableSelection(fen.middlePane.GetSelectedPathFromIndex(i))
	}
}
