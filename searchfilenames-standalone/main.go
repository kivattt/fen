package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var missingSpaceRune rune = '…'

func FindSubstringAllStartIndices(s, searchText string) []int {
	if s == "" || searchText == "" {
		return []int{}
	}

	var result []int

	i := 0
	for {
		if i >= len(s) {
			break
		}

		found := strings.Index(s[i:], searchText)
		if found == -1 {
			break
		}

		i += found
		result = append(result, i)
		i += len(searchText)
	}

	return result
}

func SetTviewStyles() {
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	// For the dropdown in the options menu
	tview.Styles.MoreContrastBackgroundColor = tcell.ColorBlack

	tview.Styles.BorderColor = tcell.ColorDefault
	tview.Borders.Horizontal = '─'
	tview.Borders.Vertical = '│'

	if runtime.GOOS == "freebsd" {
		tview.Borders.TopLeft = '┌'
		tview.Borders.TopRight = '┐'
		tview.Borders.BottomLeft = '└'
		tview.Borders.BottomRight = '┘'
	} else {
		tview.Borders.TopLeft = '╭'
		tview.Borders.TopRight = '╮'
		tview.Borders.BottomLeft = '╰'
		tview.Borders.BottomRight = '╯'
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:", os.Args[0], " [folder to search in]")
		os.Exit(0)
	}
	folderToSearch := os.Args[1]

	SetTviewStyles()

	app := tview.NewApplication()
	appFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("press 'f'"), 1, 0, false)


	fen := NewFen(app)
	fen.wd = folderToSearch

	appFlex.SetBorder(true)

	// size = 5 is a reasonable large size
	centered_large := func(p tview.Primitive, size int) tview.Primitive {
		return tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(p, 0, size, true).
				AddItem(nil, 0, 1, false), 0, size, true).
			AddItem(nil, 0, 1, false)
	}

	pages := tview.NewPages().
		AddPage("flex", appFlex, true, true)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if pages.HasPage("popup") {
			return event
		}

		if event.Rune() == 'q' {
			app.Stop()
			return nil
		} else if event.Rune() == 'f' {
			inputField := tview.NewInputField().
				SetLabel(" Search: ").
				SetPlaceholder("case-sensitive"). // TODO: Smart-case
				SetFieldWidth(-1)                 // Special feature of my tview fork, github.com/kivattt/tview
			inputField.SetTitleColor(tcell.ColorDefault)
			inputField.SetFieldBackgroundColor(tcell.ColorGray)
			inputField.SetFieldBackgroundColor(tcell.ColorGray)
			inputField.SetFieldTextColor(tcell.ColorBlack)
			inputField.SetBackgroundColor(tcell.ColorDefault)

			inputField.SetLabelColor(tcell.NewRGBColor(0, 255, 0))                    // Green
			inputField.SetPlaceholderStyle(tcell.StyleDefault.Background(tcell.ColorGray).Dim(true))

			searchFilenames := NewSearchFilenames(fen)
			inputField.SetChangedFunc(func(text string) {
				searchFilenames.mutex.Lock()
				searchFilenames.searchTerm = text
				searchFilenames.Filter(text)
				searchFilenames.mutex.Unlock()
			})

			inputField.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					searchFilenames.mutex.Lock()
					searchFilenames.cancel = true
					searchFilenames.mutex.Unlock()
					pages.RemovePage("popup")
					return
				}
			})

			inputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyEnter {
					searchFilenames.mutex.Lock()
					if len(searchFilenames.filenamesFilteredIndices) > 0 {
						selectedFilename := searchFilenames.filenames[searchFilenames.filenamesFilteredIndices[searchFilenames.selectedFilenameIndex]]
						if selectedFilename == "" {
							panic("Empty string selected in search filenames popup")
						}

						_, err := fen.GoPath(selectedFilename)
						if err != nil {
							fen.bottomBar.TemporarilyShowTextInstead(err.Error())
						}
					}
					searchFilenames.cancel = true
					searchFilenames.mutex.Unlock()
					pages.RemovePage("popup")
					return nil
				}

				// These GoUp/Go.../PageUp/PageDown functions lock/unlock the mutex internally.
				// Keep this in mind if you want to make these blocks do more than just the Go... function call
				// since that might require moving the mutex lock/unlock into the if-block instead.
				if event.Key() == tcell.KeyUp {
					searchFilenames.GoUp()
					return nil
				} else if event.Key() == tcell.KeyDown {
					searchFilenames.GoDown()
					return nil
				} else if event.Key() == tcell.KeyPgUp {
					searchFilenames.PageUp()
					return nil
				} else if event.Key() == tcell.KeyPgDn {
					searchFilenames.PageDown()
					return nil
				} else if event.Modifiers() & tcell.ModCtrl != 0 && event.Key() == tcell.KeyHome {
					searchFilenames.GoTop()
					return nil
				} else if event.Modifiers() & tcell.ModCtrl != 0 && event.Key() == tcell.KeyEnd {
					searchFilenames.GoBottom()
					return nil
				}

				return event
			})

			flex := tview.NewFlex().
				AddItem(searchFilenames, 0, 1, false).SetDirection(tview.FlexRow).
				AddItem(inputField, 1, 1, true)

			flex.SetBorder(true)
			//flex.SetMouseCapture(nil)

			pages.AddPage("popup", centered_large(flex, 10), true, true)
			return nil
		}

		//app.DontDrawOnThisEventKey()
		return event
	})

	if err := app.SetRoot(pages, true).EnableMouse(true).EnablePaste(true).Run(); err != nil {
		log.Fatal(err)
	}
}
