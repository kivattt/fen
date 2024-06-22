package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type BottomBar struct {
	*tview.Box
	fen           *Fen
	alternateText string
}

func NewBottomBar(fen *Fen) *BottomBar {
	return &BottomBar{
		Box: tview.NewBox().SetBackgroundColor(tcell.ColorDefault),
		fen: fen,
	}
}

func (bottomBar *BottomBar) TemporarilyShowTextInstead(text string) {
	bottomBar.alternateText = text
}

func (bottomBar *BottomBar) Draw(screen tcell.Screen) {
	bottomBar.Box.SetBackgroundColor(tcell.ColorBlack)
	bottomBar.Box.DrawForSubclass(screen, bottomBar)

	x, y, w, _ := bottomBar.GetInnerRect()

	freeBytes, err := FreeDiskSpaceBytes(bottomBar.fen.sel)
	freeBytesStr := BytesToHumanReadableUnitString(freeBytes, 3)
	if err != nil {
		freeBytesStr = "?"
	}

	freeBytesStr += " free"

	if bottomBar.alternateText != "" {
		tview.Print(screen, "[teal:]"+tview.Escape(bottomBar.alternateText), x, y, w, tview.AlignLeft, tcell.ColorDefault)
		// We still want to see the disk space left
		tview.Print(screen, tview.Escape(freeBytesStr), x, y, w, tview.AlignRight, tcell.ColorDefault)
		bottomBar.alternateText = ""
		return
	}

	username, groupname, err := FileUserAndGroupName(bottomBar.fen.sel)
	fileOwners := ""
	if err == nil {
		fileOwners = " " + UsernameWithColor(username) + ":" + GroupnameWithColor(groupname)
	}
	filePermissions, _ := FilePermissionsString(bottomBar.fen.sel)
	fileLastModified, _ := FileLastModifiedString(bottomBar.fen.sel)
	text := "[teal:]" + filePermissions + fileOwners + " [default:]" + fileLastModified

	noWriteEnabledText := ""
	if bottomBar.fen.config.NoWrite {
		noWriteEnabledText = " [red::r]no-write"
	}

	_, leftLength := tview.Print(screen, text+noWriteEnabledText, x, y, w, tview.AlignLeft, tcell.ColorBlue)
	_, rightLength := tview.Print(screen, tview.Escape(freeBytesStr), x, y, w, tview.AlignRight, tcell.ColorDefault)

	if bottomBar.fen.config.DontShowHelpText || *bottomBar.fen.helpScreenVisible {
		return
	}

	spaceForHelpText := w - leftLength - rightLength

	helpTextAlternatives := []string{
		"For help: Type ? or F1",
		"Help: Type ? or F1",
		"Help: Type ?",
		"Help: ?",
	}

	var helpText string
	for _, alternative := range helpTextAlternatives {
		if spaceForHelpText-1 > len(alternative) {
			helpText = alternative
			break
		}
	}

	if helpText == "" {
		return
	}

	helpTextXPosBetween := x + leftLength + (w-leftLength-rightLength)/2 - len(helpText)/2
	helpTextXPosMiddle := w/2 - len(helpText)/2

	linearInterpolationSteps := max(1, x+(leftLength-rightLength)/2)
	N := linearInterpolationSteps - min(linearInterpolationSteps, max(0, spaceForHelpText-leftLength-rightLength))
	helpTextXPos := leftLength + (helpTextXPosBetween-leftLength)*N/linearInterpolationSteps + (helpTextXPosMiddle-leftLength)*(1-N/linearInterpolationSteps)

	if helpTextXPos < x+leftLength {
		helpTextXPos = x + leftLength
	}

	tview.Print(screen, "[::d]"+helpText, helpTextXPos, y, spaceForHelpText, tview.AlignLeft, tcell.ColorDefault)
}
