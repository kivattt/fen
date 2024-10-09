package main

import (
	"os"
	"os/user"
	"runtime"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HelpScreen struct {
	*tview.Box
	fen         *Fen
	visible     bool
	scrollIndex int
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
	{KeyBindings: []string{"F2"}, Description: "Show third-party software used in fen"},
	{KeyBindings: []string{"q"}, Description: "Quit fen"},

	{KeyBindings: []string{"z", "Backspace"}, Description: "Toggle hidden files"},
	{KeyBindings: []string{"^Space", "^N"}, Description: "Open file(s) with specific program"},
	{KeyBindings: []string{"!"}, Description: "Run system shell command"},

	{KeyBindings: []string{"n"}, Description: "Create a new file"},
	{KeyBindings: []string{"N"}, Description: "Create a new folder"},
	{KeyBindings: []string{"y"}, Description: "Copy file"},
	{KeyBindings: []string{"d"}, Description: "Cut file"},
	{KeyBindings: []string{"p"}, Description: "Paste file"},
	{KeyBindings: []string{"a"}, Description: "Rename a file"},
	{KeyBindings: []string{"Del", "x"}, Description: "Delete file"},
	{KeyBindings: []string{"/", "^F"}, Description: "Search"},
	{KeyBindings: []string{"c"}, Description: "Goto path"},

	{KeyBindings: []string{"Home", "g"}, Description: "Go to the top"},
	{KeyBindings: []string{"End", "G"}, Description: "Go to the bottom"},
	{KeyBindings: []string{"Ctrl+Left"}, Description: "Go to the root folder"},
	{KeyBindings: []string{"Ctrl+Right"}, Description: "Go to the path furthest down in history"},
	{KeyBindings: []string{"M"}, Description: "Go to the middle"},
	{KeyBindings: []string{"PgUp", "PgDn"}, Description: "Scroll up/down an entire page"},
	{KeyBindings: []string{"H"}, Description: "Go to the top of the screen"},
	{KeyBindings: []string{"L"}, Description: "Go to the bottom of the screen"},
	{KeyBindings: []string{"Space"}, Description: "Select files"},
	{KeyBindings: []string{"A"}, Description: "Flip selection in folder (select all files)"},
	{KeyBindings: []string{"V"}, Description: "Start selecting by moving"},
	{KeyBindings: []string{"D"}, Description: "Deselect all, and un-yank"},
	{KeyBindings: []string{"F5"}, Description: "Sync the screen"},
	{KeyBindings: []string{"0-9"}, Description: "Go to a configured bookmark"},
}

func (helpScreen *HelpScreen) Draw(screen tcell.Screen) {
	if !helpScreen.visible {
		return
	}

	x, y, w, h := helpScreen.GetInnerRect()
	helpScreen.Box.SetRect(x, y+1, w, h-2)
	helpScreen.Box.DrawForSubclass(screen, helpScreen)

	tview.Print(screen, "[::r] fen "+version+" help menu [::-]", x, y+1, w, tview.AlignCenter, tcell.ColorDefault)

	longestDescriptionLength := 0
	for _, e := range helpScreenControlsList {
		if len(e.Description) > longestDescriptionLength {
			longestDescriptionLength = len(e.Description)
		}
	}

	controlsYOffset := h/2 - len(helpScreenControlsList)/2
	if controlsYOffset < 1 {
		controlsYOffset = 1
	}
	controlsYOffset++
	username, groupname, err := FileUserAndGroupName(helpScreen.fen.sel)

	topUser, _ := user.Current()
	topUsernameColor := "[lime::b]"
	if topUser.Uid == "0" {
		topUsernameColor = "[red::b]"
	}

	hostname := ""
	if helpScreen.fen.config.ShowHostname && runtime.GOOS != "windows" {
		hostname, _ = os.Hostname()
		hostname += " " // So the length includes the preceding '@' symbol from the topbar
	}

	tview.Print(screen, topUsernameColor+"|[-:-:-:-]", x, y+1, w, tview.AlignLeft, tcell.ColorDefault)
	tview.Print(screen, "[blue::b]|", x+len(topUser.Username)+1+len(hostname), y+1, w, tview.AlignLeft, tcell.ColorDefault)

	tview.Print(screen, topUsernameColor+"|[-:-:-:-]", x, y+2, w, tview.AlignLeft, tcell.ColorDefault)
	tview.Print(screen, "[blue::b]Path", x+len(topUser.Username)+1+len(hostname), y+2, w, tview.AlignLeft, tcell.ColorDefault)

	tview.Print(screen, topUsernameColor+"User[-:-:-:-]", x, y+3, w, tview.AlignLeft, tcell.ColorDefault)

	// There is no User:Group shown on Windows, so only describe the file permissions
	if err != nil {
		tview.Print(screen, "[teal:]File permissions", x, h-3, w, tview.AlignLeft, tcell.ColorDefault)
		tview.Print(screen, "[teal:]|", x, h-2, w, tview.AlignLeft, tcell.ColorDefault)
	} else {
		tview.Print(screen, "[teal:]File permissions", x, h-4, w, tview.AlignLeft, tcell.ColorDefault)
		tview.Print(screen, "[teal:]|[default:]", x, h-3, w, tview.AlignLeft, tcell.ColorDefault)
		tview.Print(screen, UsernameColor(username)+"User:[-:-:-:-]"+GroupnameColor(groupname)+"Group", x+10, h-3, w, tview.AlignLeft, tcell.ColorDefault)
		tview.Print(screen, UsernameColor(username)+"|[-:-:-:-]", x+10, h-2, w, tview.AlignLeft, tcell.ColorDefault)
	}

	tview.Print(screen, "[teal:]|[default:]", x, h-2, w, tview.AlignLeft, tcell.ColorDefault)

	for dY, e := range helpScreenControlsList {
		xPos := x + w/2 - (longestDescriptionLength+15)/2
		if xPos < len("|         User:Group")+1 {
			xPos = len("|         User:Group") + 1
		}

		yPos := y + dY + controlsYOffset + helpScreen.scrollIndex

		if yPos < 2 {
			continue
		}

		if yPos >= h-2 {
			tview.Print(screen, "▼ Press arrow keys/hjkl to scroll", 0, yPos, w, tview.AlignCenter, tcell.ColorDefault)
			break
		}

		keybindingsStrLengthWithoutStyleTags := 0
		var keyBindingsStrBuilder strings.Builder
		for i, keyBinding := range e.KeyBindings {
			keyBindingsStrBuilder.WriteString("[blue:]" + keyBinding + "[default:]")
			keybindingsStrLengthWithoutStyleTags += len(keyBinding)

			if i < len(e.KeyBindings)-1 {
				keyBindingsStrBuilder.WriteString(" or ")
				keybindingsStrLengthWithoutStyleTags += len(" or ")
			}
		}

		tview.Print(screen, " "+keyBindingsStrBuilder.String()+strings.Repeat(" ", 15-keybindingsStrLengthWithoutStyleTags), xPos-1, yPos, w, tview.AlignLeft, tcell.ColorDefault)
		tview.Print(screen, e.Description, xPos+15, yPos, w, tview.AlignLeft, tcell.ColorDefault)
	}

	// After the controls list so the leading space of " Available disk space" appears above
	tview.Print(screen, " Available disk space", x, h-3, w, tview.AlignRight, tcell.ColorDefault)
	tview.Print(screen, "|", x, h-2, w, tview.AlignRight, tcell.ColorDefault)
}

func (helpScreen *HelpScreen) ScrollDown() {
	helpScreen.scrollIndex--
	if helpScreen.scrollIndex <= -len(helpScreenControlsList)+5 {
		helpScreen.scrollIndex = -len(helpScreenControlsList) + 5 + 1
	}
}

func (helpScreen *HelpScreen) ScrollUp() {
	helpScreen.scrollIndex = min(0, helpScreen.scrollIndex+1)
}
