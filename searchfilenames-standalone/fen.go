package main

import (
	"github.com/rivo/tview"
)

type Config struct {
	FileEventIntervalMillis int
	HiddenFiles             bool
}

type BottomBar struct {
}

func (bottomBar *BottomBar) TemporarilyShowTextInstead(text string) {
	//os.Stderr.WriteString("Temp text: " + text + "\n")
}

type Fen struct {
	config Config

	wd                      string
	app                     *tview.Application
	FileEventIntervalMillis int
	bottomBar               BottomBar
}

func NewFen(app *tview.Application) *Fen {
	return &Fen{
		//wd: "../..",
		app:    app,
		config: Config{FileEventIntervalMillis: 333, HiddenFiles: true},
	}
}

func (fen *Fen) GoPath(path string) (string, error) {
	return "", nil
}
