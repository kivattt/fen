package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yuin/gopher-lua"
	"layeh.com/gopher-luar"
)

type FilesPane struct {
	*tview.Box
	fen                 *Fen
	folder              string       // Can be a path to a file
	entries             atomic.Value // []os.DirEntry
	selectedEntryIndex  int
	showEntrySizes      bool
	isRightFilesPane    bool
	parentIsEmptyFolder bool
	Invisible           bool
	fileWatcher         *fsnotify.Watcher
}

func NewFilesPane(fen *Fen, showEntrySizes bool, isRightFilesPane bool) *FilesPane {
	newWatcher, _ := fsnotify.NewWatcher()
	return &FilesPane{
		Box:                tview.NewBox().SetBackgroundColor(tcell.ColorDefault),
		fen:                fen,
		selectedEntryIndex: 0,
		showEntrySizes:     showEntrySizes,
		isRightFilesPane:   isRightFilesPane,
		fileWatcher:        newWatcher,
	}
}

// Initializes empty entries and starts the file watcher
func (fp *FilesPane) Init() {
	fp.entries.Store([]os.DirEntry{})
	go func() {
		for {
			select {
			case event, ok := <-fp.fileWatcher.Events:
				if !ok {
					return
				}

				if event.Has(fsnotify.Create) {
					fp.AddEntry(event.Name)
				} else if event.Has(fsnotify.Chmod) || event.Has(fsnotify.Write) {
					fp.UpdateEntry(event.Name)
				} else if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
					// TODO ?
					// Since a Rename is immediately followed by a Create, we could maybe handle rename separately
					// And follow the new path if selected by updating fen.sel
					fp.RemoveEntry(event.Name)
				} else {
					panic("Got invalid file watcher event: " + strconv.Itoa(int(event.Op)))
				}
				fp.fen.app.QueueUpdateDraw(func() {
					fp.FilterAndSortEntries()
					fp.fen.UpdatePanes(false)
				})
			case _, ok := <-fp.fileWatcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()
}

func (fp *FilesPane) AddEntry(path string) error {
	alreadyHasEntryByThatName := slices.ContainsFunc(fp.entries.Load().([]os.DirEntry), func(e os.DirEntry) bool {
		return e.Name() == filepath.Base(path)
	})
	if alreadyHasEntryByThatName {
		return errors.New("Entry already exists") // Maybe we still want to re-stat the file
	}

	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	newEntry := fs.FileInfoToDirEntry(stat)
	fp.entries.Store(append(fp.entries.Load().([]os.DirEntry), newEntry))

	return nil
}

func (fp *FilesPane) RemoveEntry(path string) error {
	index := slices.IndexFunc(fp.entries.Load().([]os.DirEntry), func(e os.DirEntry) bool {
		return e.Name() == filepath.Base(path)
	})
	if index == -1 {
		return errors.New("Entry not found")
	}

	fp.entries.Store(append(fp.entries.Load().([]os.DirEntry)[:index], fp.entries.Load().([]os.DirEntry)[index+1:]...))
	return nil
}

func (fp *FilesPane) UpdateEntry(path string) error {
	index := slices.IndexFunc(fp.entries.Load().([]os.DirEntry), func(e os.DirEntry) bool {
		return e.Name() == filepath.Base(path)
	})
	if index == -1 {
		return errors.New("Entry not found")
	}

	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	updatedEntry := fs.FileInfoToDirEntry(stat)
	fp.entries.Store(append(append(fp.entries.Load().([]os.DirEntry)[:index], updatedEntry), fp.entries.Load().([]os.DirEntry)[index+1:]...))
	return nil
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

func (fp *FilesPane) ChangeDir(path string, forceReadDir bool) {
	fi, err := os.Stat(path)
	if !forceReadDir {
		if err != nil {
			fp.entries.Store([]os.DirEntry{})
			fp.parentIsEmptyFolder = true
			fp.folder = path // We need to set the folder variable so that the "if fp.folder == path" check below won't mess up next time
			return
		}

		if !fi.IsDir() {
			fp.entries.Store([]os.DirEntry{})
			fp.parentIsEmptyFolder = false
			fp.folder = path // We need to set the folder variable so that the "if fp.folder == path" check below won't mess up next time
			return
		}

		// path != "/" is a hacky fix so the left pane doesn't disappear when you go right from root
		if fp.folder == path && path != "/" {
			fp.parentIsEmptyFolder = len(fp.entries.Load().([]os.DirEntry)) <= 0
			return
		}
	}

	if fi.IsDir() {
		fp.fileWatcher.Remove(fp.folder)
		fp.folder = path
		newEntries, _ := os.ReadDir(fp.folder)
		fp.entries.Store(newEntries)
		fp.fileWatcher.Add(fp.folder) // This has to be after the os.ReadDir() so we have something to update
	} else {
		fp.entries.Store([]os.DirEntry{})
	}

	fp.FilterAndSortEntries()

	fp.parentIsEmptyFolder = fi.IsDir() && len(fp.entries.Load().([]os.DirEntry)) <= 0
}

func (fp *FilesPane) FilterAndSortEntries() {
	if !fp.fen.config.HiddenFiles {
		withoutHiddenFiles := []os.DirEntry{}
		for _, e := range fp.entries.Load().([]os.DirEntry) {
			if !strings.HasPrefix(e.Name(), ".") {
				withoutHiddenFiles = append(withoutHiddenFiles, e)
			}
		}

		fp.entries.Store(withoutHiddenFiles)

		// TODO: Generic bounds checking function?
		if len(fp.entries.Load().([]os.DirEntry)) > 0 && fp.selectedEntryIndex >= len(fp.entries.Load().([]os.DirEntry)) {
			fp.selectedEntryIndex = len(fp.entries.Load().([]os.DirEntry)) - 1
			//			fp.SetSelectedEntryFromIndex(len(fp.entries) - 1)
		}
	}

	switch fp.fen.config.SortBy {
	case "modified":
		slices.SortStableFunc(fp.entries.Load().([]os.DirEntry), func(a, b fs.DirEntry) int {
			aInfo, aErr := a.Info()
			bInfo, bErr := b.Info()
			if aErr != nil || bErr != nil {
				return 0
			}

			if aInfo.ModTime().Before(bInfo.ModTime()) {
				return -1
			}
			if aInfo.ModTime().Equal(bInfo.ModTime()) {
				return 0
			}

			return 1
		})
	case "size":
		slices.SortStableFunc(fp.entries.Load().([]os.DirEntry), func(a, b fs.DirEntry) int {
			aInfo, aErr := a.Info()
			bInfo, bErr := b.Info()
			if aErr != nil || bErr != nil {
				return 0
			}

			// If folder, we consider the folder file count as bytes (though it's kind of messed up with symlinks...)
			aSize := int(aInfo.Size())
			if a.IsDir() {
				aSize, _ = FolderFileCount(filepath.Join(fp.folder, a.Name()), fp.fen.config.HiddenFiles)
			}

			bSize := int(bInfo.Size())
			if b.IsDir() {
				bSize, _ = FolderFileCount(filepath.Join(fp.folder, b.Name()), fp.fen.config.HiddenFiles)
			}

			if aSize < bSize {
				return -1
			}

			if aSize == bSize {
				return 0
			}
			return 1
		})
	case "none":
	default:
		fmt.Fprintln(os.Stderr, "Invalid sort_by value \""+fp.fen.config.SortBy+"\"")
		fmt.Fprintln(os.Stderr, "Valid values: "+strings.Join(ValidSortByValues[:], ", "))
		os.Exit(1)
	}

	if fp.fen.config.FoldersFirst {
		fp.entries.Store(FoldersAtBeginning(fp.entries.Load().([]os.DirEntry)))
	}
}

func (fp *FilesPane) SetSelectedEntryFromString(entryName string) error {
	for i, entry := range fp.entries.Load().([]os.DirEntry) {
		if entry.Name() == entryName {
			fp.selectedEntryIndex = i
			return nil
		}
	}

	return errors.New("No entry with name: " + entryName)
}

func (fp *FilesPane) SetSelectedEntryFromIndex(index int) {
	fp.selectedEntryIndex = index
}

func (fp *FilesPane) GetSelectedEntryFromIndex(index int) string {
	if index >= len(fp.entries.Load().([]os.DirEntry)) {
		return ""
	}

	if index < 0 {
		return ""
	}

	return fp.entries.Load().([]os.DirEntry)[index].Name()
}

func (fp *FilesPane) GetSelectedPathFromIndex(index int) string {
	return filepath.Join(fp.folder, fp.GetSelectedEntryFromIndex(index))
}

// Returns -1 if nothing was found
func (fp *FilesPane) GetSelectedIndexFromEntry(entryName string) int {
	for i, entry := range fp.entries.Load().([]os.DirEntry) {
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
	if fp.selectedEntryIndex > h/2 {
		topScreenEntryIndex = fp.selectedEntryIndex - h/2
	}

	if topScreenEntryIndex >= len(fp.entries.Load().([]os.DirEntry)) {
		topScreenEntryIndex = max(0, len(fp.entries.Load().([]os.DirEntry))-1)
	}

	return topScreenEntryIndex
}

func (fp *FilesPane) GetBottomScreenEntryIndex() int {
	_, _, _, h := fp.GetInnerRect()
	bottomScreenEntryIndex := fp.GetTopScreenEntryIndex() + h - 1
	if bottomScreenEntryIndex >= len(fp.entries.Load().([]os.DirEntry)) {
		bottomScreenEntryIndex = max(0, len(fp.entries.Load().([]os.DirEntry))-1)
	}

	return bottomScreenEntryIndex
}

func (fp *FilesPane) Draw(screen tcell.Screen) {
	if fp.Invisible {
		return
	}

	fp.Box.DrawForSubclass(screen, fp)

	x, y, w, h := fp.GetInnerRect()
	if fp.isRightFilesPane && fp.parentIsEmptyFolder || (!fp.isRightFilesPane && len(fp.entries.Load().([]os.DirEntry)) <= 0) && fp.folder != filepath.Dir(fp.folder) {
		tview.Print(screen, "[:red]empty", x, y, w, tview.AlignLeft, tcell.ColorDefault)
		return
	}

	// File previews
	stat, statErr := os.Stat(fp.fen.sel)
	f, readErr := os.OpenFile(fp.fen.sel, os.O_RDONLY, 0)
	f.Close()
	if statErr == nil && stat.Mode().IsRegular() && readErr == nil && len(fp.entries.Load().([]os.DirEntry)) <= 0 && fp.isRightFilesPane {
		for _, previewWith := range fp.fen.config.Preview {
			matched := PathMatchesList(fp.fen.sel, previewWith.Match) && !PathMatchesList(fp.fen.sel, previewWith.DoNotMatch)
			if !matched {
				continue
			}

			if previewWith.Script != "" {
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
				err := L.DoFile(previewWith.Script)
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

			for _, program := range previewWith.Program {
				programSplitSpace := strings.Split(program, " ")

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
	for i, entry := range fp.entries.Load().([]os.DirEntry)[scrollOffset:] {
		if i >= h {
			break
		}

		entryFullPath := filepath.Join(fp.folder, entry.Name())
		entryInfo, _ := entry.Info() // This seems to immediately stat on Linux
		style := FileColor(entryInfo, entryFullPath)

		spaceForSelected := ""
		if i+scrollOffset == fp.selectedEntryIndex {
			style = style.Reverse(true)
		}

		if slices.Contains(fp.fen.selected, entryFullPath) {
			spaceForSelected = " "
			style = style.Foreground(tcell.ColorYellow)
		}

		entryInYankSelected := slices.Contains(fp.fen.yankSelected, entryFullPath)
		if entryInYankSelected {
			style = style.Dim(true)
		}

		entrySizeText := ""
		entrySizePrintedSize := 0
		if fp.showEntrySizes {
			var err error
			entryInfo, err := entry.Info()
			if err == nil {
				entrySizeText, err = EntrySizeText(entryInfo, entryFullPath, fp.fen.config.HiddenFiles)
				if err != nil {
					entrySizeText = "?"
				}
			}

			_, entrySizePrintedSize = tview.Print(screen, StyleToStyleTagString(style)+" "+tview.Escape(entrySizeText)+" ", x, y+i, w-1, tview.AlignRight, tcell.ColorDefault)
		}

		styleStr := StyleToStyleTagString(style)
		tview.Print(screen, spaceForSelected+styleStr+" "+FilenameInvisibleCharactersAsCodeHighlighted(tview.Escape(entry.Name()), styleStr)+strings.Repeat(" ", w), x, y+i, w-1-entrySizePrintedSize, tview.AlignLeft, tcell.ColorDefault)

		// You can't see which files are yanked in the FreeBSD tty terminal since dim doesn't change the color, so this makes it visible
		// Seems like the same issue is present on Windows, in cmd
		if (runtime.GOOS == "freebsd" || runtime.GOOS == "windows") && entryInYankSelected {
			tview.PrintSimple(screen, "*", x, y+i)
		}
	}
}
