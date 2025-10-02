package main

import (
	"github.com/rivo/tview"
)

const (
	CASE_INSENSITIVE = "insensitive"
	CASE_SENSITIVE   = "sensitive"
)

type Config struct {
	FileEventIntervalMillis int
	HiddenFiles             bool
	FilenameSearchCase      string
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
		config: Config{FileEventIntervalMillis: 333, HiddenFiles: true, FilenameSearchCase: CASE_INSENSITIVE},
	}
}

func (fen *Fen) GoPath(path string) (string, error) {
	return "", nil
}
