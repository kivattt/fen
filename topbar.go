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
	fen                *Fen
	additionalText     string
	showAdditionalText bool
}

func NewTopBar(fen *Fen) *TopBar {
	return &TopBar{
		Box:                tview.NewBox().SetBackgroundColor(tcell.ColorDefault),
		fen:                fen,
		showAdditionalText: false,
	}
}

func (topBar *TopBar) Draw(screen tcell.Screen) {
	if *topBar.fen.librariesScreenVisible {
		return
	}

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
	if topBar.fen.showHomePathAsTilde && runtime.GOOS != "windows" {
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

	_, pathPrintedLength := tview.Print(screen, pathText, x+1+usernameAndHostnameLength, y, w, tview.AlignLeft, tcell.ColorBlue)

	if topBar.showAdditionalText {
		tview.Print(screen, "« "+topBar.additionalText, x+usernameAndHostnameLength+1+pathPrintedLength+1, y, w, tview.AlignLeft, tcell.ColorDefault)
	}

	if topBar.fen.runningGitStatus {
		tview.Print(screen, "Refreshing Git status...", x, y, w, tview.AlignRight, tcell.ColorDefault)
	}
}
