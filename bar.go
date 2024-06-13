package main

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Bar struct {
	*tview.Box
	str          *string
	selectedPath *string
	isTopBar     bool // TODO: Refactor bar.go into 2 files
	noWrite      *bool
}

func NewBar(str *string, selectedPath *string, noWrite *bool) *Bar {
	return &Bar{
		Box:          tview.NewBox().SetBackgroundColor(tcell.ColorDefault),
		str:          str,
		selectedPath: selectedPath,
		noWrite:      noWrite,
	}
}

func (bar *Bar) Draw(screen tcell.Screen) {
	if !bar.isTopBar {
		bar.Box.SetBackgroundColor(tcell.ColorBlack)
	}
	bar.Box.DrawForSubclass(screen, bar)

	x, y, w, _ := bar.GetInnerRect()
	text := *bar.str
	if bar.isTopBar {
		user, _ := user.Current()
		usernameColor := "[lime:]"
		if user.Uid == "0" {
			usernameColor = "[red:]"
		}

		pathToShow := filepath.Dir(text)
		if runtime.GOOS == "linux" {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				if strings.HasPrefix(pathToShow, homeDir) {
					pathToShow = filepath.Join("~", pathToShow[len(homeDir):])
				}
			}
		}
		text = "[::b]" + usernameColor + tview.Escape(user.Username) + " " +
			"[blue::B]" + FilenameInvisibleCharactersAsCodeHighlighted(tview.Escape(PathWithEndSeparator(pathToShow)), "[blue::B]") +
			"[white::b]" + FilenameInvisibleCharactersAsCodeHighlighted(tview.Escape(PathWithoutEndSeparator(filepath.Base(text))), "[white::b]")
	}

	noWriteEnabledText := ""
	if !bar.isTopBar && *bar.noWrite {
		noWriteEnabledText = " [red::r]no-write"
	}
	tview.Print(screen, text+noWriteEnabledText, x, y, w, tview.AlignLeft, tcell.ColorBlue)

	if !bar.isTopBar {
		freeBytes, err := FreeDiskSpaceBytes(*bar.selectedPath)
		freeBytesStr := BytesToHumanReadableUnitString(freeBytes, 3)
		if err != nil {
			freeBytesStr = "?"
		}

		freeBytesStr += " free"
		tview.Print(screen, tview.Escape(freeBytesStr), x, y, w, tview.AlignRight, tcell.ColorDefault)
	}
}
