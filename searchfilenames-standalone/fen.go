package main

import "github.com/rivo/tview"

type Config struct {
	FileEventIntervalMillis int
}

type BottomBar struct {
}
func (bottomBar *BottomBar) TemporarilyShowTextInstead(text string) {
	return
}

type Fen struct {
	config Config

	wd string
	app *tview.Application
	FileEventIntervalMillis int
	bottomBar BottomBar
}

func NewFen(app *tview.Application) *Fen {
	return &Fen{
		//wd: "../..",
		app: app,
		config: Config{333},
	}
}

func (fen *Fen) GoPath(path string) (string, error) {
	return "", nil
}
