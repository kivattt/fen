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
	topBar.Box.DrawForSubclass(screen, topBar)

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
	if runtime.GOOS == "linux" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			if strings.HasPrefix(pathToShow, homeDir) {
				pathToShow = filepath.Join("~", pathToShow[len(homeDir):])
			}
		}
	}

	textToShow := "[::b]" + usernameColor + tview.Escape(username)
	if topBar.fen.config.ShowHostname && runtime.GOOS == "linux" {
		hostname, err := os.Hostname()
		if err == nil {
			textToShow += tview.Escape("@" + hostname)
		}
	}

	textToShow += " " +
		"[blue::B]" + FilenameInvisibleCharactersAsCodeHighlighted(tview.Escape(PathWithEndSeparator(pathToShow)), "[blue::B]") +
		"[white::b]" + FilenameInvisibleCharactersAsCodeHighlighted(tview.Escape(PathWithoutEndSeparator(filepath.Base(path))), "[white::b]")

	tview.Print(screen, textToShow, x, y, w, tview.AlignLeft, tcell.ColorBlue)
}
