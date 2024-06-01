package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type OpenWithList struct {
	*tview.Box
	programs     *[]string
	descriptions *[]string
}

func NewOpenWithList(programs *[]string, descriptions *[]string) *OpenWithList {
	if len(*programs) != len(*descriptions) {
		panic("In NewOpenWithList: Length of programs and descriptions weren't the same")
	}

	return &OpenWithList{Box: tview.NewBox().SetBackgroundColor(tcell.ColorBlack), programs: programs, descriptions: descriptions}
}

func (openWithList *OpenWithList) Draw(screen tcell.Screen) {
	if len(*openWithList.programs) != len(*openWithList.descriptions) {
		panic("In openwithlist.go Draw(): Length of programs and descriptions weren't the same")
	}

	openWithList.Box.DrawForSubclass(screen, openWithList)

	x, y, w, _ := openWithList.GetInnerRect()
	for i, program := range *openWithList.programs {
		color := tcell.ColorDefault
		if i == 0 {
			color = tcell.ColorAqua
		}

		tview.Print(screen, tview.Escape(program), x+1, y+i, w, tview.AlignLeft, color)
		description := (*openWithList.descriptions)[i]
		tview.Print(screen, "[::d]"+tview.Escape(description), x-1, y+i, w, tview.AlignRight, color)
	}
}
