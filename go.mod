module github.com/kivattt/fen

go 1.21.5

replace github.com/gdamore/tcell/v2 => github.com/kivattt/tcell-naively-faster/v2 v2.0.1

replace github.com/rivo/tview => github.com/kivattt/tview v1.0.4

require (
	github.com/fsnotify/fsnotify v1.7.0
	github.com/gdamore/tcell/v2 v2.7.4
	github.com/kivattt/getopt v0.0.0-20240907012637-674e0e42e04f
	github.com/otiai10/copy v1.14.0
	github.com/rivo/tview v0.0.0-20240818110301-fd649dbf1223
	github.com/yuin/gluamapper v0.0.0-20150323120927-d836955830e7
	github.com/yuin/gopher-lua v1.1.1
	golang.org/x/sys v0.25.0
	golang.org/x/term v0.24.0
	layeh.com/gopher-luar v1.0.11
)

require (
	github.com/gdamore/encoding v1.0.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/text v0.18.0 // indirect
)
