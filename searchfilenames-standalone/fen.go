package main

import "github.com/rivo/tview"

type Config struct {
	FileEventIntervalMillis int
}

type Fen struct {
	config Config

	wd string
	app *tview.Application
	FileEventIntervalMillis int
}

func NewFen(app *tview.Application) *Fen {
	return &Fen{
		//wd: "../..",
		app: app,
		config: Config{333},
	}
}
