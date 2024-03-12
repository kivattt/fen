package main

import (
	"errors"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func GetFiles(path string) []string {
	entries, _ := ioutil.ReadDir(path)
	var ret []string

	for _, e := range entries {
		ret = append(ret, e.Name())
	}

	return ret
}

type Ranger struct {
	wd           string
	furthestPath string

	left   []string
	middle []string
	right  []string

	topPane    *Bar
	leftPane   *FilesPane
	middlePane *FilesPane
	rightPane  *FilesPane
	bottomPane *Bar
}

func (r *Ranger) Init() error {
	var err error
	r.wd, err = os.Getwd()

	// TODO fix
	r.furthestPath = r.wd

	r.topPane = NewBar(&r.wd)
	r.leftPane = NewFilesPane(&r.left)
	r.middlePane = NewFilesPane(&r.middle)
	r.rightPane = NewFilesPane(&r.right)
	r.bottomPane = NewBar(&r.furthestPath)

	r.UpdatePanes()

	return err
}

func (r *Ranger) UpdateSelectedEntries() {
	return

	/*	if len(wdSplit) > len(furthestSplit) {
		r.furthestPath = r.wd
	}*/

	//	*leftPane.SetSelectedEntryFromString(filepath.Base(furthestPath)))
	//	r.middlePane.SetSelectedEntryFromString(filepath.Base(middleSelectedEntry))

	// FIXME: Don't use wd, use the furthest path up to wd thingy
	r.leftPane.SetSelectedEntryFromString(filepath.Base(filepath.Dir(r.wd)))
	//	r.middlePane.SetSelectedEntryFromString(filepath.Base(r.wd))
	r.middlePane.SetSelectedEntryFromString(filepath.Base(filepath.Dir(r.furthestPath)))

	rPath, _ := r.GetRightPath()
	r.rightPane.SetSelectedEntryFromString(filepath.Base(rPath))
}

func (r *Ranger) GetRightPath() (string, error) {
	if r.middlePane.selectedEntry >= len(r.middle) {
		return "", errors.New("Out of bounds")
	}

	return filepath.Join(r.wd, r.middle[r.middlePane.selectedEntry]), nil

	/*
	   wdSplit := filepath.SplitList(r.wd)
	   furthestSplit := filepath.SplitList(r.furthestPath)

	   furthestPathUpToWD := ""

	   // Gets furthestPath up to wd

	   	for i := 0; i < len(furthestSplit); i++ {
	   		furthestSection := furthestSplit[i]
	   		furthestPathUpToWD += "/" + furthestSection

	   		if i >= len(wdSplit) {
	   			break
	   		}

	   		if furthestSection != wdSplit[i] {
	   			break
	   		}
	   	}

	   	if furthestPathUpToWD == r.wd {
	   		return "", errors.New("Bruh")
	   	}

	   return furthestPathUpToWD, nil
	*/
}

func (r *Ranger) UpdatePanes() {
	parentDir := filepath.Dir(r.wd)

	r.left = GetFiles(parentDir)
	r.middle = GetFiles(r.wd)

	rPath, err := r.GetRightPath()
	if err == nil {
		r.right = GetFiles(rPath)
	} else {
		r.right = []string{}
	}

	r.UpdateSelectedEntries()
}

func (r *Ranger) GetSelectedFilePath() string {
	if r.middlePane.selectedEntry >= len(r.middle) {
		return ""
	}
	return filepath.Join(r.wd, r.middle[r.middlePane.selectedEntry])
}

func (r *Ranger) GoToParent() {
	r.wd = filepath.Dir(r.wd)
	r.UpdatePanes()
}

func (r *Ranger) GoToChild() error {
	if r.middlePane.selectedEntry >= len(r.middle) {
		return errors.New("Out of bounds")
	}

	r.wd = filepath.Join(r.wd, r.middle[r.middlePane.selectedEntry])

	// FIXME: Hacky, fix with the furthest up to wd thingy
	/*	if len(r.wd) > len(r.furthestPath) {
		r.furthestPath = r.wd
	}*/
	r.furthestPath = r.wd

	r.UpdatePanes()

	return nil
}

func (r *Ranger) GoUp() {
	r.middlePane.selectedEntry--
	if r.middlePane.selectedEntry < 0 {
		r.middlePane.selectedEntry = 0
	}

	if r.middlePane.selectedEntry < len(r.middle) {
		r.furthestPath = filepath.Join(r.wd, r.middle[r.middlePane.selectedEntry])
	}

	r.UpdatePanes()
}

func (r *Ranger) GoDown() {
	r.middlePane.selectedEntry++
	if r.middlePane.selectedEntry >= len(r.middle) {
		r.middlePane.selectedEntry = len(r.middle) - 1
	}

	if r.middlePane.selectedEntry < len(r.middle) {
		r.furthestPath = filepath.Join(r.wd, r.middle[r.middlePane.selectedEntry])
	}

	r.UpdatePanes()
}

func main() {
	var ranger Ranger
	ranger.Init()

	app := tview.NewApplication()

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' {
			app.Stop()
			return nil
		}

		if event.Key() == tcell.KeyEnter {
			cmd := exec.Command("vim", ranger.GetSelectedFilePath())
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				log.Fatal(err)
			}
			return nil
		}

		if event.Key() == tcell.KeyLeft {
			ranger.GoToParent()
			return nil
		}

		if event.Key() == tcell.KeyRight {
			ranger.GoToChild()
			return nil
		}

		if event.Key() == tcell.KeyUp {
			ranger.GoUp()
			return nil
		}

		if event.Key() == tcell.KeyDown {
			ranger.GoDown()
			return nil
		}
		return event
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ranger.topPane, 1, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(ranger.leftPane, 0, 1, false).
			AddItem(ranger.middlePane, 0, 2, false).
			AddItem(ranger.rightPane, 0, 2, false), 0, 1, false).
		AddItem(ranger.bottomPane, 1, 0, false)

	if err := app.SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}
