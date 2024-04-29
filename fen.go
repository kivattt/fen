package main

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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

	dontShowHiddenFiles bool
	guiBorders          bool
	noWrite             bool

	topPane    *Bar
	leftPane   *FilesPane
	middlePane *FilesPane
	rightPane  *FilesPane
	bottomPane *Bar

	fileProperties *FileProperties
}

func (fen *Fen) Init(workingDirectory string) error {
	fen.dontShowHiddenFiles = false
	fen.selectingWithV = false
	fen.fileProperties = NewFileProperties()

	fen.wd = workingDirectory

	fen.topPane = NewBar(&fen.sel, &fen.sel, &fen.noWrite)
	fen.topPane.isTopBar = true

	fen.leftPane = NewFilesPane(&fen.selected, &fen.yankSelected, &fen.dontShowHiddenFiles, false)
	fen.middlePane = NewFilesPane(&fen.selected, &fen.yankSelected, &fen.dontShowHiddenFiles, true)
	fen.rightPane = NewFilesPane(&fen.selected, &fen.yankSelected, &fen.dontShowHiddenFiles, false)

	if fen.guiBorders {
		fen.leftPane.SetBorder(true)
		fen.middlePane.SetBorder(true)
		fen.rightPane.SetBorder(true)
	}

	fen.bottomPane = NewBar(&fen.bottomBarText, &fen.sel, &fen.noWrite)

	wdFiles, err := os.ReadDir(fen.wd)
	// If our working directory doesn't exist, go up a parent until it does
	for err != nil {
		//		if fen.wd == "/" || filepath.Dir(fen.wd) == fen.wd {
		if filepath.Dir(fen.wd) == fen.wd {
			return err
		}

		fen.wd = filepath.Dir(fen.wd)
		wdFiles, err = os.ReadDir(fen.wd)
	}

	if len(wdFiles) > 0 {
		fen.sel = filepath.Join(fen.wd, wdFiles[0].Name())
	}

	fen.history.AddToHistory(fen.sel)
	fen.UpdatePanes()

	return err
}

func (fen *Fen) ReadConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		// We don't want to close if there is no config file
		// This should really be checked by the caller...
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++

		line := strings.Trim(scanner.Text(), " \t")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		separatorIndex := strings.Index(line, ":")
		if separatorIndex == -1 {
			continue
		}

		key := line[:separatorIndex]
		value := strings.ToLower(strings.Trim(line[separatorIndex+1:], " \t"))

		if !(value == "yes" || value == "no") {
			return errors.New("Boolean flag '" + key + "' was not \"yes\" or \"no\", but \"" + value + "\"")
		}

		if key == "gui-borders" {
			fen.guiBorders = value == "yes"
		} else if key == "no-write" {
			fen.noWrite = value == "yes"
		}
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
	fen.leftPane.SetEntries(filepath.Dir(fen.wd))
	fen.middlePane.SetEntries(fen.wd)

	//	if fen.wd != "/" {
	if fen.wd != filepath.Dir(fen.wd) {
		fen.leftPane.SetSelectedEntryFromString(filepath.Base(fen.wd))
	} else {
		fen.leftPane.entries = []os.DirEntry{}
	}

	//	fen.bottomBarText = "Set selected entry from string: " + filepath.Base(fen.sel)
	username, groupname, err := FileUserAndGroupName(fen.sel)
	fileOwners := ""
	if err == nil {
		fileOwners = " " + UsernameWithColor(username) + ":" + GroupnameWithColor(groupname)
	}
	filePermissions, _ := FilePermissionsString(fen.sel)
	fileLastModified, _ := FileLastModifiedString(fen.sel)
	fen.bottomBarText = "[aqua:]" + filePermissions + fileOwners + " [white:]" + fileLastModified

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
	//	fen.bottomBarText = strings.Join(fen.history.history, ", ")

	h, err := fen.history.GetHistoryEntryForPath(fen.sel, fen.dontShowHiddenFiles)
	if err != nil {
		//		if !fen.dontShowHiddenFiles {
		fen.rightPane.SetSelectedEntryFromIndex(0)
		//		}
		//		fen.bottomBarText = "BRUH"
	} else {
		//	fen.bottomBarText = "BRUH 2.0: " + filepath.Base(h)
		//	if !fen.dontShowHiddenFiles {
		fen.rightPane.SetSelectedEntryFromString(filepath.Base(h))
		// }
	}

	fen.UpdateSelectingWithV()

	fi, err := os.Stat(fen.sel)
	if err != nil {
		return
	}

	if fen.fileProperties.visible {
		fen.fileProperties.SetTable(map[string]string{
			"Name": filepath.Base(fen.sel),
			"Size": strconv.FormatInt(fi.Size(), 10) + " bytes",
		})
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

func (fen *Fen) GoRight(app *tview.Application) {
	if len(fen.middlePane.entries) <= 0 {
		return
	}

	fi, err := os.Stat(fen.sel)
	if err != nil {
		return
	}

	if !fi.IsDir() {
		OpenFile(fen.sel, app)
		return
	}

	/*	rightFiles, _ := os.ReadDir(fen.sel)
		if len(rightFiles) <= 0 {
			return
		}*/

	fen.wd = fen.sel
	fen.sel, err = fen.history.GetHistoryEntryForPath(fen.wd, fen.dontShowHiddenFiles)

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
	for _, e := range fen.middlePane.entries {
		if strings.Contains(strings.ToLower(e.Name()), strings.ToLower(searchTerm)) {
			//			fen.middlePane.SetSelectedEntryFromString(e.Name())
			fen.sel = filepath.Join(fen.wd, e.Name())
			fen.selectingWithVEndIndex = fen.middlePane.GetSelectedIndexFromEntry(e.Name())
			/*			if fen.selectingWithV {
						fen.selectingWithVEndIndex = fen.middlePane.
					}*/
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
