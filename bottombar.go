package main

import (
	"os"
	"strconv"

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

	alternateTextToUse := bottomBar.alternateText
	if bottomBar.alternateText != "" {
		tview.Print(screen, "[teal:]"+tview.Escape(bottomBar.alternateText), x, y, w, tview.AlignLeft, tcell.ColorDefault)
		bottomBar.alternateText = ""
	}

	username, groupname, err := FileUserAndGroupName(bottomBar.fen.sel)
	fileOwners := ""
	if err == nil {
		fileOwners = " " + UsernameWithColor(username) + ":" + GroupnameWithColor(groupname)
	}
	filePermissions, _ := FilePermissionsString(bottomBar.fen.sel)
	text := "[teal:]" + filePermissions + fileOwners

	if !*bottomBar.fen.helpScreenVisible && !*bottomBar.fen.thirdPartySoftwareScreenVisible {
		fileLastModified, _ := FileLastModifiedString(bottomBar.fen.sel)
		text += " [default:]" + fileLastModified
	}

	noWriteEnabledText := ""
	if bottomBar.fen.config.NoWrite {
		noWriteEnabledText = " [red::r]no-write"
	}

	leftLength := 0
	if alternateTextToUse == "" {
		_, leftLength = tview.Print(screen, text+noWriteEnabledText, x, y, w, tview.AlignLeft, tcell.ColorBlue)
	}
	_, rightFreeBytesStrLength := tview.Print(screen, " "+tview.Escape(freeBytesStr), x, y, w, tview.AlignRight, tcell.ColorDefault)
	rightFreeBytesStrLength-- // To ignore the leading ^ space

	entriesLength := len(bottomBar.fen.middlePane.entries.Load().([]os.DirEntry))
	entriesLengthStr := strconv.Itoa(entriesLength)
	positionStr := strconv.Itoa(min(entriesLength, bottomBar.fen.middlePane.selectedEntryIndex+1)) + "/" + entriesLengthStr
	positionStrMaxLength := 2*len(entriesLengthStr) + len("/")

	// We multiply entriesLengthStr by 2 so that the help text won't suddenly change/move if the selected index string length changes
	rightLength := rightFreeBytesStrLength + positionStrMaxLength + 1 // For positioning the help text

	positionStrXPos := w - rightFreeBytesStrLength - len(positionStr) - 1
	positionStrLowerXPos := w - rightFreeBytesStrLength - positionStrMaxLength - 1
	positionStrHasNoSpace := positionStrLowerXPos < leftLength+1

	jobCountStr := ""
	jobCountStrAttributes := "d"
	bottomBar.fen.fileOperationsHandler.workCountMutex.Lock()
	if bottomBar.fen.config.AlwaysShowInfoNumbers || bottomBar.fen.fileOperationsHandler.workCount > 0 {
		jobCountStr = strconv.Itoa(bottomBar.fen.fileOperationsHandler.workCount)
		if bottomBar.fen.fileOperationsHandler.workCount > 0 {
			jobCountStrAttributes = ""
		}

		if *bottomBar.fen.helpScreenVisible || *bottomBar.fen.thirdPartySoftwareScreenVisible {
			jobCountStr += " jobs"
		}
	}
	bottomBar.fen.fileOperationsHandler.workCountMutex.Unlock()

	yankCountStr := ""
	yankCountStrAttributes := "d"
	if bottomBar.fen.config.AlwaysShowInfoNumbers || len(bottomBar.fen.yankSelected) > 0 {
		jobCountStr += " "

		yankCountStr = strconv.Itoa(len(bottomBar.fen.yankSelected))
		if len(bottomBar.fen.yankSelected) > 0 {
			yankCountStrAttributes = ""

			if bottomBar.fen.yankType == "copy" {
				yankCountStr += "y"
			} else if bottomBar.fen.yankType == "cut" {
				yankCountStr += "d"
			} else if bottomBar.fen.yankType != "" {
				panic("yankType was not \"copy\" or \"cut\"")
			}
		}

		if *bottomBar.fen.helpScreenVisible || *bottomBar.fen.thirdPartySoftwareScreenVisible {
			yankCountStr += " yanked"
		}
	}

	selectedCountStr := ""
	selectedCountStrAttributes := "d"
	if bottomBar.fen.config.AlwaysShowInfoNumbers || len(bottomBar.fen.selected) > 0 {
		yankCountStr += " "

		selectedCountStr = strconv.Itoa(len(bottomBar.fen.selected))
		if len(bottomBar.fen.selected) > 0 {
			selectedCountStrAttributes = ""
		}

		if *bottomBar.fen.helpScreenVisible || *bottomBar.fen.thirdPartySoftwareScreenVisible {
			selectedCountStr += " selected"
		}
	}

	var countStringsHasNoSpace bool
	if bottomBar.fen.config.ShowHelpText && !(*bottomBar.fen.helpScreenVisible || *bottomBar.fen.thirdPartySoftwareScreenVisible) {
		// We add 1 extra to hackily prevent the countStrings from showing up to the left of the helpText (might not work with different FileLastModifiedString() lengths)
		countStringsHasNoSpace = positionStrLowerXPos-1-len(jobCountStr)-len(selectedCountStr)-len(yankCountStr) < leftLength+1+1
	} else {
		countStringsHasNoSpace = positionStrLowerXPos-1-len(jobCountStr)-len(selectedCountStr)-len(yankCountStr) < leftLength+1
	}

	countStringsPrintedLength := 0
	if !countStringsHasNoSpace && !positionStrHasNoSpace {
		_, countStringsPrintedLength = tview.Print(screen, "[blue::"+jobCountStrAttributes+"]"+jobCountStr+"[-:-:-:-][#00ff00::"+yankCountStrAttributes+"]"+yankCountStr+"[-:-:-:-][yellow::"+selectedCountStrAttributes+"]"+selectedCountStr, positionStrLowerXPos-1-len(jobCountStr)-len(selectedCountStr)-len(yankCountStr), y, w, tview.AlignLeft, tcell.ColorDefault)

		if countStringsPrintedLength > 0 {
			countStringsPrintedLength += 1
		}
	}

	spaceForHelpText := w - leftLength - rightLength - countStringsPrintedLength

	if !bottomBar.fen.config.ShowHelpText || *bottomBar.fen.helpScreenVisible || *bottomBar.fen.thirdPartySoftwareScreenVisible {
		if !positionStrHasNoSpace {
			tview.Print(screen, positionStr, positionStrXPos, y, w, tview.AlignLeft, tcell.ColorDefault) // SAME AS
		}
		return
	}

	helpTextAlternatives := []string{
		"For help: Type ? or F1",
		"Help: Type ? or F1",
		"Help: Type ?",
		"Help: ?",
	}

	if spaceForHelpText-1 > len(helpTextAlternatives[len(helpTextAlternatives)-1]) {
		if !positionStrHasNoSpace {
			tview.Print(screen, positionStr, positionStrXPos, y, w, tview.AlignLeft, tcell.ColorDefault) // SAME AS
		}
	} else {
		// We hide the countStrings and positionStr to give enough space for the help text
		spaceForHelpText += rightLength + countStringsPrintedLength - rightFreeBytesStrLength

		// Show positionStr anyway if the help text has no chance of being shown even with the positionStr hidden
		if spaceForHelpText-1 <= len(helpTextAlternatives[len(helpTextAlternatives)-1]) {
			if !positionStrHasNoSpace {
				tview.Print(screen, positionStr, positionStrXPos, y, w, tview.AlignLeft, tcell.ColorDefault) // SAME AS
			}
			return
		}
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

	helpTextXPosBetween := x + leftLength + spaceForHelpText/2 - len(helpText)/2
	helpTextXPosMiddle := w/2 - len(helpText)/2

	helpTextXPos := helpTextXPosBetween
	if helpTextXPosMiddle > leftLength+3 {
		helpTextXPos = helpTextXPosMiddle
	}

	if helpTextXPos < x+leftLength {
		helpTextXPos = x + leftLength
	}

	if alternateTextToUse == "" {
		tview.Print(screen, "[::d]"+helpText, helpTextXPos, y, spaceForHelpText, tview.AlignLeft, tcell.ColorDefault)
	}
}
