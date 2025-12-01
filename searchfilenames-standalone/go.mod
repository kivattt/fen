module main

go 1.21.5

replace github.com/gdamore/tcell/v2 => github.com/kivattt/tcell-naively-faster/v2 v2.0.1

replace github.com/rivo/tview => github.com/kivattt/tview v1.0.5

require (
	github.com/charlievieth/strcase v0.0.5
	github.com/gdamore/tcell/v2 v2.8.1
	github.com/rivo/tview v0.0.0-20250625164341-a4a78f1e05cb
)

require (
	github.com/gdamore/encoding v1.0.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/term v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)
