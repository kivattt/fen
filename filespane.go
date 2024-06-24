package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yuin/gopher-lua"
	"layeh.com/gopher-luar"
)

type FilesPane struct {
	*tview.Box
	fen                 *Fen
	folder              string
	entries             []os.DirEntry
	selectedEntry       int
	showEntrySizes      bool
	isRightFilesPane    bool
	parentIsEmptyFolder bool
	Invisible           bool
}

func NewFilesPane(fen *Fen, showEntrySizes bool, isRightFilesPane bool) *FilesPane {
	return &FilesPane{
		Box:              tview.NewBox().SetBackgroundColor(tcell.ColorDefault),
		fen:              fen,
		selectedEntry:    0,
		showEntrySizes:   showEntrySizes,
		isRightFilesPane: isRightFilesPane,
	}
}

type FenLuaGlobal struct {
	SelectedFile string
	Width        int
	Height       int
	x            int
	y            int
	screen       tcell.Screen
}

func (f *FenLuaGlobal) Print(text string, x, y, maxWidth, align int, color tcell.Color) int {
	if x < 0 || x > f.Width {
		return 0
	}
	if y < 0 || y >= f.Height {
		return 0
	}

	text = strings.ReplaceAll(text, "\t", "    ")
	_, widthPrinted := tview.Print(f.screen, text, x+f.x, y+f.y, maxWidth, align, color)
	return widthPrinted
}

func (f *FenLuaGlobal) PrintSimple(text string, x, y int) int {
	return f.Print(text, x, y, f.Width, 0, 0)
}

func (f *FenLuaGlobal) Escape(text string) string {
	return tview.Escape(text)
}

func (f *FenLuaGlobal) TranslateANSI(text string) string {
	return tview.TranslateANSI(text)
}

func (f *FenLuaGlobal) NewRGBColor(r, g, b int32) tcell.Color {
	return tcell.NewRGBColor(r, g, b)
}

func (f *FenLuaGlobal) ColorToString(color tcell.Color) string {
	return color.String()
}

func (f *FenLuaGlobal) RuntimeOS() string {
	return runtime.GOOS
}

func (f *FenLuaGlobal) Version() string {
	return version
}

func (fp *FilesPane) SetEntries(path string, foldersNotFirst bool) {
	fi, err := os.Stat(path)
	if err != nil {
		fp.entries = []os.DirEntry{}
		fp.parentIsEmptyFolder = true
		return
	}

	if !fi.IsDir() {
		fp.entries = []os.DirEntry{}
		fp.parentIsEmptyFolder = false
		return
	}

	fp.folder = path
	fp.entries, _ = os.ReadDir(fp.folder)

	if fp.fen.config.DontShowHiddenFiles {
		withoutHiddenFiles := []os.DirEntry{}
		for _, e := range fp.entries {
			if !strings.HasPrefix(e.Name(), ".") {
				withoutHiddenFiles = append(withoutHiddenFiles, e)
			}
		}

		fp.entries = withoutHiddenFiles

		// TODO: Generic bounds checking function?
		if len(fp.entries) > 0 && fp.selectedEntry >= len(fp.entries) {
			fp.selectedEntry = len(fp.entries) - 1
			//			fp.SetSelectedEntryFromIndex(len(fp.entries) - 1)
		}
	}

	if !foldersNotFirst {
		fp.entries = FoldersAtBeginning(fp.entries)
	}

	fp.parentIsEmptyFolder = len(fp.entries) <= 0
}

func (fp *FilesPane) SetSelectedEntryFromString(entryName string) error {
	for i, entry := range fp.entries {
		if entry.Name() == entryName {
			fp.selectedEntry = i
			return nil
		}
	}

	fp.selectedEntry = 0
	return errors.New("No entry with that name")
}

func (fp *FilesPane) SetSelectedEntryFromIndex(index int) {
	fp.selectedEntry = index
}

func (fp *FilesPane) GetSelectedEntryFromIndex(index int) string {
	if index >= len(fp.entries) {
		return ""
	}

	if index < 0 {
		return ""
	}

	return fp.entries[index].Name()
}

func (fp *FilesPane) GetSelectedPathFromIndex(index int) string {
	return filepath.Join(fp.folder, fp.GetSelectedEntryFromIndex(index))
}

// Returns -1 if nothing was found
func (fp *FilesPane) GetSelectedIndexFromEntry(entryName string) int {
	for i, entry := range fp.entries {
		if entry.Name() == entryName {
			return i
		}
	}

	return -1
}

// Used as scroll offset aswell
func (fp *FilesPane) GetTopScreenEntryIndex() int {
	_, _, _, h := fp.GetInnerRect()
	topScreenEntryIndex := 0
	if fp.selectedEntry > h/2 {
		topScreenEntryIndex = fp.selectedEntry - h/2
	}

	if topScreenEntryIndex >= len(fp.entries) {
		topScreenEntryIndex = max(0, len(fp.entries)-1)
	}

	return topScreenEntryIndex
}

func (fp *FilesPane) GetBottomScreenEntryIndex() int {
	_, _, _, h := fp.GetInnerRect()
	bottomScreenEntryIndex := fp.GetTopScreenEntryIndex() + h - 1
	if bottomScreenEntryIndex >= len(fp.entries) {
		bottomScreenEntryIndex = max(0, len(fp.entries)-1)
	}

	return bottomScreenEntryIndex
}

func (fp *FilesPane) Draw(screen tcell.Screen) {
	if fp.Invisible {
		return
	}

	fp.Box.DrawForSubclass(screen, fp)

	x, y, w, h := fp.GetInnerRect()

	if fp.parentIsEmptyFolder || !fp.isRightFilesPane && len(fp.entries) <= 0 && fp.folder != filepath.Dir(fp.folder) {
		tview.Print(screen, "[:red]empty", x, y, w, tview.AlignLeft, tcell.ColorDefault)
		return
	}

	// File previews
	stat, statErr := os.Stat(fp.fen.sel)
	f, readErr := os.OpenFile(fp.fen.sel, os.O_RDONLY, 0)
	f.Close()
	if statErr == nil && stat.Mode().IsRegular() && readErr == nil && len(fp.entries) <= 0 && fp.isRightFilesPane {
		for _, previewWith := range fp.fen.config.PreviewWith {
			matched := PathMatchesList(fp.fen.sel, previewWith.Match) && !PathMatchesList(fp.fen.sel, previewWith.DoNotMatch)
			if !matched {
				continue
			}

			userConfigDir, userConfigDirErr := os.UserConfigDir()
			if previewWith.Script != "" {
				scriptPath := previewWith.Script
				if userConfigDirErr == nil {
					scriptPath = strings.ReplaceAll(scriptPath, "FEN_CONFIG_PATH", filepath.Join(userConfigDir, "fen"))
				}

				L := lua.NewState()
				defer L.Close()

				fenLuaGlobal := &FenLuaGlobal{
					SelectedFile: fp.fen.sel,
					x:            x,
					y:            y,
					Width:        w,
					Height:       h,
					screen:       screen,
				}

				L.SetGlobal("fen", luar.New(L, fenLuaGlobal))
				luaFileAbsolutePath := ""
				if !filepath.IsAbs(scriptPath) {
					userConfigDir, err := os.UserConfigDir()
					if err == nil {
						luaFileAbsolutePath = filepath.Join(userConfigDir, "fen", scriptPath)
					}
				} else {
					luaFileAbsolutePath = scriptPath
				}
				err := L.DoFile(luaFileAbsolutePath)
				if err != nil {
					tview.Print(screen, "File preview Lua error:", x, y, w, tview.AlignLeft, tcell.ColorRed)
					lines := tview.WordWrap(err.Error(), w)
					i := 0
					for _, line := range lines {
						tview.Print(fenLuaGlobal.screen, line, x, y+1+i, w, tview.AlignLeft, tcell.ColorDefault)
						i++
					}
				}
				return
			}

			for _, program := range previewWith.Programs {
				programSplitSpace := strings.Split(program, " ")
				if userConfigDirErr == nil {
					for i, e := range programSplitSpace {
						programSplitSpace[i] = strings.ReplaceAll(e, "FEN_CONFIG_PATH", filepath.Join(userConfigDir, "fen"))
					}
				}

				programName := programSplitSpace[0]
				programArguments := []string{}
				if len(programSplitSpace) > 1 {
					programArguments = programSplitSpace[1:]
				}

				cmd := exec.Command(programName, append(programArguments, fp.fen.sel)...)

				textView := tview.NewTextView()
				textView.Box.SetRect(x, y, w, h)
				textView.SetBackgroundColor(tcell.ColorDefault)
				textView.SetTextColor(tcell.ColorDefault)
				textView.SetDynamicColors(true)

				cmd.Stdout = tview.ANSIWriter(textView)

				err := cmd.Run()
				if err == nil {
					textView.Draw(screen)
					return
				}
			}
		}
		return
	}

	scrollOffset := fp.GetTopScreenEntryIndex()
	for i, entry := range fp.entries[scrollOffset:] {
		if i >= h {
			break
		}

		entryFullPath := filepath.Join(fp.folder, entry.Name())
		style := FileColor(entryFullPath)

		spaceForSelected := ""
		if i+scrollOffset == fp.selectedEntry {
			style = style.Reverse(true)
		}

		if slices.Contains(fp.fen.selected, entryFullPath) {
			spaceForSelected = " "
			style = style.Foreground(tcell.ColorYellow)
		}

		// Dim the entry if its in yankSelected
		if slices.Contains(fp.fen.yankSelected, entryFullPath) {
			style = style.Dim(true)
		}

		entrySizeText := ""
		entrySizePrintedSize := 0
		if fp.showEntrySizes {
			var err error
			entrySizeText, err = EntrySize(entryFullPath, fp.fen.config.DontShowHiddenFiles)
			if err != nil {
				entrySizeText = "?"
			}

			_, entrySizePrintedSize = tview.Print(screen, StyleToStyleTagString(style)+" "+tview.Escape(entrySizeText)+" ", x, y+i, w-1, tview.AlignRight, tcell.ColorDefault)
		}

		styleStr := StyleToStyleTagString(style)
		tview.Print(screen, spaceForSelected+styleStr+" "+FilenameInvisibleCharactersAsCodeHighlighted(tview.Escape(entry.Name()), styleStr)+strings.Repeat(" ", w), x, y+i, w-1-entrySizePrintedSize, tview.AlignLeft, tcell.ColorDefault)
	}
}
