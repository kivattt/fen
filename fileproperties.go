package main

import (
	"sort"

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
		Box:     tview.NewBox().SetBorder(true).SetTitle("File properties"),
		visible: false,
	}
}

func (fileProperties *FileProperties) SetTable(table map[string]string) {
	fileProperties.table = table

	fileProperties.tableKeysOrdered = []string{}
	for k, _ := range fileProperties.table {
		fileProperties.tableKeysOrdered = append(fileProperties.tableKeysOrdered, k)
	}

	sort.Strings(fileProperties.tableKeysOrdered)
}

func (fileProperties *FileProperties) Draw(screen tcell.Screen) {
	if !fileProperties.visible {
		return
	}

	fileProperties.Box.DrawForSubclass(screen, fileProperties)

	x, y, w, _ := fileProperties.GetInnerRect()

	i := 0
	for _, key := range fileProperties.tableKeysOrdered {
		tview.Print(screen, key+" "+fileProperties.table[key], x, y+i, w, tview.AlignLeft, tcell.NewRGBColor(0, 255, 255))
		i++
	}
}
