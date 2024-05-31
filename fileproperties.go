package main

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FileProperties struct {
	*tview.Box
	table            map[string]string
	tableKeysOrdered []string
	visible          bool
}

func NewFileProperties() *FileProperties {
	return &FileProperties{
		Box:     tview.NewBox().SetBorder(true).SetTitle("File properties").SetBackgroundColor(tcell.ColorDefault).SetBorderColor(tcell.ColorDefault).SetTitleColor(tcell.ColorDefault),
		visible: false,
	}
}

func (fileProperties *FileProperties) UpdateTable(path string) {
	fi, err := os.Stat(path)
	if err != nil {
		return
	}

	fileProperties.table = map[string]string{
		"Name": filepath.Base(path),
	}

	if fi.Mode().IsRegular() {
		fileProperties.table["Size"] = BytesToHumanReadableUnitString(uint64(fi.Size()), -1)
		fileProperties.table["Size (bytes)"] = strconv.FormatInt(fi.Size(), 10) + " B"
	}

	fileProperties.tableKeysOrdered = []string{}
	for k := range fileProperties.table {
		fileProperties.tableKeysOrdered = append(fileProperties.tableKeysOrdered, k)
	}

	sort.Strings(fileProperties.tableKeysOrdered)
}

func (fileProperties *FileProperties) Draw(screen tcell.Screen) {
	if !fileProperties.visible {
		return
	}

	// Always draw 1 line higher to not overlap with the bottom bar
	rX, rY, rW, rH := fileProperties.GetRect()
	fileProperties.Box.SetRect(rX, rY, rW, rH-1)

	fileProperties.Box.DrawForSubclass(screen, fileProperties)

	x, y, w, _ := fileProperties.GetInnerRect()

	i := 0
	for _, key := range fileProperties.tableKeysOrdered {
		tview.Print(screen, tview.Escape(key+": "+fileProperties.table[key]), x, y+i, w, tview.AlignLeft, tcell.NewRGBColor(0, 255, 255))
		i++
	}
}
