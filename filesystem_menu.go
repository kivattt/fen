package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FileSystemMenu struct {
	*tview.Box
	pages               *tview.Pages
	fen                 *Fen
	options             []string
	selectedOptionIndex int
}

func NewFileSystemMenu(pages *tview.Pages, fen *Fen) *FileSystemMenu {
	return &FileSystemMenu{
		Box:   tview.NewBox().SetBackgroundColor(tcell.ColorDefault),
		pages: pages,
		fen:   fen,
	}
}

func (fsm *FileSystemMenu) GetOptions() []string {
	return []string{
		"Local",
		"SFTP server",
	}
}

func (fsm *FileSystemMenu) Draw(screen tcell.Screen) {
	x, y, w, _ := fsm.GetInnerRect()
	y++

	fsm.options = fsm.GetOptions()

	for i, option := range fsm.options {
		styleStr := ""
		if i == fsm.selectedOptionIndex {
			styleStr = "[::d]"
		}
		tview.Print(screen, styleStr+" "+option, x, y+i, w, tview.AlignLeft, tcell.ColorBlue)
	}
}

func (fsm *FileSystemMenu) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return fsm.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if event.Key() == tcell.KeyDown || event.Rune() == 'j' {
			fsm.selectedOptionIndex = min(len(fsm.options)-1, fsm.selectedOptionIndex+1)
		} else if event.Key() == tcell.KeyUp || event.Rune() == 'k' {
			fsm.selectedOptionIndex = max(0, fsm.selectedOptionIndex-1)
		} else if event.Key() == tcell.KeyRight || event.Key() == tcell.KeyEnter || event.Key() == tcell.KeyEscape || event.Rune() == 'q' || event.Rune() == 'l' {
			switch fsm.options[fsm.selectedOptionIndex] {
			case "Local":
				theFS = fileSystems[0]
			case "SFTP server":
				theFS = fileSystems[1]
			}

			fsm.fen.DisableSelectingWithV()
			fsm.fen.GoRootPath()
			fsm.fen.InvalidateFolderFileCountCache()
			fsm.fen.UpdatePanes(true)
			fsm.pages.RemovePage("popup")
		}
	})
}
