package main

import (
	"runtime"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LibrariesScreen struct {
	*tview.Box
	visible bool
}

func NewLibrariesScreen() *LibrariesScreen {
	return &LibrariesScreen{Box: tview.NewBox().SetBackgroundColor(tcell.ColorDefault)}
}

type library struct {
	name string
	url  string

	version           string
	customRevisionURL string

	license    string
	licenseURL string
}

var librariesList = []library{
	{name: "tview", url: "https://github.com/rivo/tview", customRevisionURL: "https://github.com/kivattt/tview", license: "MIT", licenseURL: "https://github.com/rivo/tview/blob/master/LICENSE.txt"},
	{name: "tcell", url: "https://github.com/gdamore/tcell", customRevisionURL: "https://github.com/kivattt/tcell-naively-faster", license: "Apache 2.0", licenseURL: "https://github.com/gdamore/tcell/blob/main/LICENSE"},
	{name: "fsnotify", url: "https://github.com/fsnotify/fsnotify", version: "v1.7.0", license: "BSD 3-Clause", licenseURL: "https://github.com/fsnotify/fsnotify/blob/main/LICENSE"},
	{name: "otiai10/copy", url: "https://github.com/otiai10/copy", version: "v1.14.0", license: "MIT", licenseURL: "https://github.com/otiai10/copy/blob/main/LICENSE"},
	{name: "gopher-lua", url: "https://github.com/yuin/gopher-lua", version: "v1.1.1", license: "MIT", licenseURL: "https://github.com/yuin/gopher-lua/blob/master/LICENSE"},
	{name: "gluamapper", url: "https://github.com/yuin/gluamapper", version: "commit d836955", license: "MIT", licenseURL: "https://github.com/yuin/gluamapper/blob/master/LICENSE"},
	{name: "gopher-luar", url: "https://layeh.com/gopher-luar", version: "v1.0.11", license: "MPL 2.0", licenseURL: "https://github.com/layeh/gopher-luar/blob/master/LICENSE"},
	{name: "rsc/getopt", url: "https://github.com/rsc/getopt", customRevisionURL: "https://github.com/kivattt/getopt", license: "BSD 3-Clause", licenseURL: "https://github.com/rsc/getopt/blob/master/LICENSE"},
	{name: "pkg/sftp", url: "https://github.com/pkg/sftp", version: "v1.13.7", license: "BSD 2-Clause", licenseURL: "https://github.com/pkg/sftp/blob/master/LICENSE"},
	{name: "kivattt/gogitstatus", url: "https://github.com/kivattt/gogitstatus", version: "commit 7362d58", license: "MIT", licenseURL: "https://github.com/kivattt/gogitstatus/blob/main/LICENSE"},
}

func (librariesScreen *LibrariesScreen) Draw(screen tcell.Screen) {
	if !librariesScreen.visible {
		return
	}

	x, y, w, h := librariesScreen.GetInnerRect()
	librariesScreen.Box.SetRect(x, y+1, w, h-2)
	librariesScreen.Box.DrawForSubclass(screen, librariesScreen)

	libraryListYOffset := max(6, h/2-len(librariesList)/2)
	yOffset := max(0, libraryListYOffset-6)

	tview.Print(screen, "[::r] Libraries used in fen "+version+" [::-]", x, yOffset, w, tview.AlignCenter, tcell.ColorDefault)
	tview.Print(screen, "https://github.com/kivattt/fen", 0, yOffset+2, w, tview.AlignCenter, tcell.NewRGBColor(0, 255, 0)) // Green

	libraryListXOffset := max(0, w/2-32)
	tview.Print(screen, "┌─[::b]Library", libraryListXOffset, libraryListYOffset-1, w, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, "┌─[::b]License", libraryListXOffset+54, libraryListYOffset-1, w, tview.AlignLeft, tcell.ColorWhite)

	darkGrayColor := "[gray:::"
	if runtime.GOOS == "freebsd" {
		darkGrayColor = "[black::b:"
	}

	for i, e := range librariesList {
		//name := "[:::" + e.url + "]" + e.name + "[:::-]"
		name := e.url
		if e.customRevisionURL != "" {
			name += " " + darkGrayColor + e.customRevisionURL + "]custom revision"
		} else {
			//name += " [black::b]" + e.version
			name += " " + darkGrayColor + "]" + e.version
		}

		tview.Print(screen, name, libraryListXOffset, i+libraryListYOffset, w, tview.AlignLeft, tcell.ColorDefault)
		tview.Print(screen, "[-:-:-:"+e.licenseURL+"]"+e.license, libraryListXOffset+54, i+libraryListYOffset, w, tview.AlignLeft, tcell.ColorDefault)
	}
}
