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

type TopBar struct {
	*tview.Box
	fen *Fen
}

func NewTopBar(fen *Fen) *TopBar {
	return &TopBar{
		Box: tview.NewBox().SetBackgroundColor(tcell.ColorDefault),
		fen: fen,
	}
}

func (topBar *TopBar) Draw(screen tcell.Screen) {
	x, y, w, _ := topBar.GetInnerRect()

	path := topBar.fen.sel

	var username string
	usernameColor := "[lime:]"

	user, err := user.Current()
	if err != nil {
		username = "unknown"
		usernameColor = "[yellow:]"
	} else {
		username = user.Username
		if user.Uid == "0" {
			usernameColor = "[red:]"
		}
	}

	pathToShow := filepath.Dir(path)
	if topBar.fen.effectiveShowHomePathAsTilde && runtime.GOOS != "windows" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			if strings.HasPrefix(pathToShow, homeDir) {
				pathToShow = filepath.Join("~", pathToShow[len(homeDir):])
			}
		}
	}

	usernameAndHostname := "[::b]" + usernameColor + tview.Escape(username)
	if topBar.fen.config.ShowHostname && runtime.GOOS != "windows" {
		hostname, err := os.Hostname()
		if err == nil {
			usernameAndHostname += tview.Escape("@" + hostname)
		}
	}

	_, usernameAndHostnameLength := tview.Print(screen, usernameAndHostname, x, y, w, tview.AlignLeft, tcell.ColorBlue)

	pathText := "[blue::b]" + FilenameInvisibleCharactersAsCodeHighlighted(tview.Escape(PathWithEndSeparator(pathToShow)), "[blue::b]") +
		"[white::b]" + FilenameInvisibleCharactersAsCodeHighlighted(tview.Escape(PathWithoutEndSeparator(filepath.Base(path))), "[white::b]")

	tview.Print(screen, pathText, x+1+usernameAndHostnameLength, y, w, tview.AlignLeft, tcell.ColorBlue)
}
