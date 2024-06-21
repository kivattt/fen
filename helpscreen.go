package main

import (
	"os/user"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HelpScreen struct {
	*tview.Box
	fen *Fen
	visible bool
}

func NewHelpScreen(fen *Fen) *HelpScreen {
	return &HelpScreen{Box: tview.NewBox().SetBackgroundColor(tcell.ColorDefault), fen: fen}
}

type control struct {
	KeyBindings []string
	Description string
}

var helpScreenControlsList = []control{
	{KeyBindings: []string{"?", "F1"}, Description: "Toggle help menu (you are here!)"},
	{KeyBindings: []string{"q"}, Description: "Quit fen"},

	{KeyBindings: []string{"^Space", "^N"}, Description: "Open file(s) with specific program"},

	{KeyBindings: []string{"n"}, Description: "Create a new file"},
	{KeyBindings: []string{"N"}, Description: "Create a new folder"},
	{KeyBindings: []string{"y"}, Description: "Copy file"},
	{KeyBindings: []string{"d"}, Description: "Cut file"},
	{KeyBindings: []string{"p"}, Description: "Paste file"},
	{KeyBindings: []string{"a"}, Description: "Rename a file"},
	{KeyBindings: []string{"Delete"}, Description: "Delete file"},
	{KeyBindings: []string{"/", "^F"}, Description: "Search"},

	{KeyBindings: []string{"Home", "g"}, Description: "Go to the top"},
	{KeyBindings: []string{"End", "G"}, Description: "Go to the bottom"},
	{KeyBindings: []string{"M"}, Description: "Go to the middle"},
	{KeyBindings: []string{"H"}, Description: "Go to the top of the screen"},
	{KeyBindings: []string{"L"}, Description: "Go to the bottom of the screen"},
	{KeyBindings: []string{"Space"}, Description: "Select files"},
	{KeyBindings: []string{"A"}, Description: "Flip selection in folder (select all files)"},
	{KeyBindings: []string{"V"}, Description: "Start selecting by moving"},
	{KeyBindings: []string{"D"}, Description: "Deselect all, and un-yank"},
	{KeyBindings: []string{"z", "Backspace"}, Description: "Toggle hidden files"},
}

func (helpScreen *HelpScreen) Draw(screen tcell.Screen) {
	if !helpScreen.visible {
		return
	}

	x, y, w, h := helpScreen.GetInnerRect()
	helpScreen.Box.SetRect(x, y+1, w, h - 2)
	helpScreen.Box.DrawForSubclass(screen, helpScreen)

	// If fen.sel is a file with characters we escape, the red background of them will bleed into this "fen vX.X.X help menu" title
	// A possible solution is to just use a black background with "[:black]", but it is distinct from the default background...
	tview.Print(screen, " [::r]fen " + version + " help menu[::-] ", x, y, w, tview.AlignCenter, tcell.ColorDefault)

	longestDescriptionLength := 0
	for _, e := range helpScreenControlsList {
		if len(e.Description) > longestDescriptionLength {
			longestDescriptionLength = len(e.Description)
		}
	}

	controlsYOffset := h/2 - len(helpScreenControlsList) / 2
	if controlsYOffset < 1 {
		controlsYOffset = 1
	}
	username, groupname, err := FileUserAndGroupName(helpScreen.fen.sel)

	topUser, _ := user.Current()
	topUsernameColor := "[lime:]"
	if topUser.Uid == "0" {
		topUsernameColor = "[red:]"
	}

	tview.Print(screen, topUsernameColor + "|[-:-:-:-]", x, y+1, w, tview.AlignLeft, tcell.ColorDefault)
	tview.Print(screen, "[blue:]|", x+len(topUser.Username)+1, y+1, w, tview.AlignLeft, tcell.ColorDefault)

	tview.Print(screen, topUsernameColor + "|[-:-:-:-]", x, y+2, w, tview.AlignLeft, tcell.ColorDefault)
	tview.Print(screen, "[blue:]Path", x+len(topUser.Username)+1, y+2, w, tview.AlignLeft, tcell.ColorDefault)

	tview.Print(screen, topUsernameColor + "User[-:-:-:-]", x, y+3, w, tview.AlignLeft, tcell.ColorDefault)

	// There is no User:Group shown on Windows, so only describe the file permissions
	if err != nil {
		tview.Print(screen, "[teal:]File permissions", x, h-3, w, tview.AlignLeft, tcell.ColorDefault)
		tview.Print(screen, "[teal:]|", x, h-2, w, tview.AlignLeft, tcell.ColorDefault)
	} else {
		tview.Print(screen, "[teal:]File permissions", x, h-4, w, tview.AlignLeft, tcell.ColorDefault)
		tview.Print(screen, "[teal:]|[default:]", x, h-3, w, tview.AlignLeft, tcell.ColorDefault)
		tview.Print(screen, UsernameColor(username) + "User:[-:-:-:-]" + GroupnameColor(groupname) + "Group", x+10, h-3, w, tview.AlignLeft, tcell.ColorDefault)
		tview.Print(screen, UsernameColor(username) + "|[-:-:-:-]", x+10, h-2, w, tview.AlignLeft, tcell.ColorDefault)
	}

	tview.Print(screen, "[teal:]|[default:]", x, h-2, w, tview.AlignLeft, tcell.ColorDefault)

	tview.Print(screen, " Available disk space", x, h-3, w, tview.AlignRight, tcell.ColorDefault)
	tview.Print(screen, "|", x, h-2, w, tview.AlignRight, tcell.ColorDefault)

	for dY, e := range helpScreenControlsList {
		if dY >= h - 1 { // This just returns if we're outside of the terminal
			break
		}

		var keyBindingsStr strings.Builder
		for i, keyBinding := range e.KeyBindings {
			keyBindingsStr.WriteString("[blue:]" + keyBinding + "[default:]")
			if i < len(e.KeyBindings) - 1 {
				keyBindingsStr.WriteString(" or ")
			}
		}
		xPos := x + w/2 - (longestDescriptionLength + 15) / 2
		if xPos < len("|         User:Group") + 1 {
			xPos = len("|         User:Group") + 1
		}
		tview.Print(screen, keyBindingsStr.String(), xPos, y+dY+controlsYOffset, w, tview.AlignLeft, tcell.ColorDefault)
		tview.Print(screen, e.Description, xPos+15, y+dY+controlsYOffset, w, tview.AlignLeft, tcell.ColorDefault)
	}
}
