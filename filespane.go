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
	"sync"
	"sync/atomic"
	"time"

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
	lastFileEventTime   time.Time
	fileEventBatch      []fsnotify.Event
	fileEventBatchMutex sync.Mutex

	lastRenamedPath     string
	lastRenamedPathTime time.Time
}

func NewFilesPane(fen *Fen, showEntrySizes, isRightFilesPane bool) *FilesPane {
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

				// We need to check this since we can be stuck handling an event from a previously removed watcher
				// All this fileWatcher stuff causes data races
				if !strings.HasPrefix(event.Name, fp.folder) {
					break
				}

				lastFileEventTime := fp.lastFileEventTime
				fp.lastFileEventTime = time.Now() // I want to set this to the time before the file event is handled

				// If it has been longer than FileEventInterval since the last event, immediately handle and update the screen.
				fp.fileEventBatchMutex.Lock()
				if fp.fen.config.FileEventIntervalMillis <= 0 || (time.Since(lastFileEventTime) > time.Duration(fp.fen.config.FileEventIntervalMillis)*time.Millisecond && len(fp.fileEventBatch) == 0) {
					fp.fileEventBatchMutex.Unlock()
					fp.HandleFileEvent(event)
					fp.fen.app.QueueUpdateDraw(func() {
						fp.FilterAndSortEntries()
						fp.fen.UpdatePanes(false)
						fp.fen.TriggerGitStatus() // Ask for a new git status on a file event
					})
				} else {
					fp.fileEventBatch = AddEventToBatch(fp.fileEventBatch, event)
					fp.fileEventBatchMutex.Unlock()
				}
			case _, ok := <-fp.fileWatcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	go func() {
		for {
			time.Sleep(time.Duration(fp.fen.config.FileEventIntervalMillis) * time.Millisecond)

			fp.fileEventBatchMutex.Lock()
			if len(fp.fileEventBatch) == 0 {
				fp.fileEventBatchMutex.Unlock()
				continue
			}

			for _, e := range fp.fileEventBatch {
				fp.HandleFileEvent(e)
			}
			fp.fileEventBatch = []fsnotify.Event{}
			fp.fileEventBatchMutex.Unlock()

			fp.fen.app.QueueUpdateDraw(func() {
				fp.FilterAndSortEntries()
				fp.fen.UpdatePanes(false)
				fp.fen.TriggerGitStatus() // Ask for a new git status on a file event
			})
		}
	}()
}

// Adds newEvent to oldEvents, removing duplicate and unnecessary prior events
func AddEventToBatch(oldEvents []fsnotify.Event, newEvent fsnotify.Event) []fsnotify.Event {
	newEventPathIsUnique := !slices.ContainsFunc(oldEvents, func(oldEvent fsnotify.Event) bool {
		return oldEvent.Name == newEvent.Name
	})

	if newEventPathIsUnique {
		oldEvents = append(oldEvents, newEvent)
		return oldEvents
	}

	if newEvent.Has(fsnotify.Remove) {
		// File was removed, remove all prior events
		oldEvents = slices.DeleteFunc(oldEvents, func(oldEvent fsnotify.Event) bool {
			return oldEvent.Name == newEvent.Name
		})
	}

	// Remove any duplicate events
	oldEvents = slices.DeleteFunc(oldEvents, func(oldEvent fsnotify.Event) bool {
		return oldEvent.Has(newEvent.Op) && oldEvent.Name == newEvent.Name
	})

	oldEvents = append(oldEvents, newEvent)

	return oldEvents
}

func (fp *FilesPane) HandleFileEvent(event fsnotify.Event) error {
	if event.Has(fsnotify.Create) {
		// A file temporarily renamed, then renamed back to its old path within 200 milliseconds is added back to the history.
		// This is a hack to fix navigation because when vim saves a file it temporarily renames the file by appending a tilde (~),
		//  then renaming it back to the original path within a very short period of time.
		if time.Since(fp.lastRenamedPathTime) < 200*time.Millisecond {
			if event.Name == fp.lastRenamedPath {
				fp.fen.history.RemoveFromHistory(fp.GetSelectedPathFromIndex(fp.selectedEntryIndex))
				fp.fen.history.AddToHistory(event.Name)
			}
		}
		return fp.AddEntry(event.Name)
	}

	if event.Has(fsnotify.Chmod) || event.Has(fsnotify.Write) {
		return fp.UpdateEntry(event.Name)
	}

	if event.Has(fsnotify.Rename) {
		fp.lastRenamedPath = event.Name
		fp.lastRenamedPathTime = time.Now()
	}

	if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		return fp.RemoveEntry(event.Name)
	}

	panic("Got invalid file watcher event: " + strconv.Itoa(int(event.Op)))
}

func (fp *FilesPane) AddEntry(path string) error {
	alreadyHasEntryByThatName := slices.ContainsFunc(fp.entries.Load().([]os.DirEntry), func(e os.DirEntry) bool {
		return e.Name() == filepath.Base(path)
	})
	if alreadyHasEntryByThatName {
		return errors.New("Entry already exists") // Maybe we still want to re-stat the file
	}

	stat, err := os.Lstat(path)
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
	fp.fen.RemoveFromSelectedAndYankSelected(path) // FIXME: Panic when deleting 4000 files

	fp.fen.history.RemoveFromHistory(path)
	fp.fen.history.AddToHistory(fp.GetSelectedPathFromIndex(fp.selectedEntryIndex))

	return nil
}

func (fp *FilesPane) UpdateEntry(path string) error {
	index := slices.IndexFunc(fp.entries.Load().([]os.DirEntry), func(e os.DirEntry) bool {
		return e.Name() == filepath.Base(path)
	})
	if index == -1 {
		return errors.New("Entry not found")
	}

	stat, err := os.Lstat(path)
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

// It might os.ReadDir() even if forceReadDir is false. If forceReadDir is true, it will always os.ReadDir() if path is a folder.
func (fp *FilesPane) ChangeDir(path string, forceReadDir bool) {
	stat, err := os.Stat(path)
	statIsDir := false
	if err == nil {
		statIsDir = stat.IsDir()
	}

	if !forceReadDir {
		if !statIsDir {
			fp.fileWatcher.Remove(fp.folder)
			fp.entries.Store([]os.DirEntry{})
			fp.parentIsEmptyFolder = false
			fp.folder = path // We need to set the folder variable so that the "if fp.folder == path" check below won't mess up next time
			return
		}

		if err != nil {
			fp.fileWatcher.Remove(fp.folder)
			fp.entries.Store([]os.DirEntry{})
			fp.parentIsEmptyFolder = true
			fp.folder = path // We need to set the folder variable so that the "if fp.folder == path" check below won't mess up next time
			return
		}

		// filepath.Clean(path) != filepath.Dir(path) is a hacky fix so the left pane doesn't disappear when you go right from root
		if fp.folder == path && filepath.Clean(path) != filepath.Dir(path) {
			fp.parentIsEmptyFolder = len(fp.entries.Load().([]os.DirEntry)) <= 0
			return
		}
	}

	if err == nil && statIsDir {
		fp.fileWatcher.Remove(fp.folder)
		fp.folder = path
		newEntries, _ := os.ReadDir(fp.folder)
		fp.entries.Store(newEntries)
		fp.fileWatcher.Add(fp.folder) // This has to be after the os.ReadDir() so we have something to update

		fp.FilterAndSortEntries()
	} else {
		fp.fileWatcher.Remove(fp.folder)
		fp.entries.Store([]os.DirEntry{})
	}

	fp.parentIsEmptyFolder = statIsDir && len(fp.entries.Load().([]os.DirEntry)) <= 0
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
		fp.keepSelectionInBounds()
	}

	// Sort the files as os.ReadDir() would, to guarantee the order
	if fp.fen.config.SortBy != SORT_NONE {
		// Should be similar enough to https://cs.opensource.google/go/go/+/refs/tags/go1.23.2:src/os/dir.go;l=126
		slices.SortFunc(fp.entries.Load().([]os.DirEntry), func(a, b fs.DirEntry) int {
			return strings.Compare(a.Name(), b.Name())
		})
	}

	switch fp.fen.config.SortBy {
	case SORT_ALPHABETICAL: // Since we already sort alphabetically above, we don't need to do anything
	case SORT_MODIFIED:
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
	case SORT_SIZE:
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
	case SORT_FILE_EXTENSION:
		// Also sorts folders based on file extension, kind of weird
		slices.SortStableFunc(fp.entries.Load().([]os.DirEntry), func(a, b fs.DirEntry) int {
			aExt := strings.ToLower(filepath.Ext(a.Name()))
			bExt := strings.ToLower(filepath.Ext(b.Name()))

			if aExt == bExt {
				return 0
			}

			if aExt < bExt {
				return -1
			}

			return 1
		})
	case SORT_NONE: // Does nothing, this has the side effect of making file events always show up at the bottom, until the entire folder is re-read
	default:
		fmt.Fprintln(os.Stderr, "Invalid sort_by value \""+fp.fen.config.SortBy+"\"")
		fmt.Fprintln(os.Stderr, "Valid values: "+strings.Join(ValidSortByValues[:], ", "))
		os.Exit(1)
	}

	if fp.fen.config.SortBy != SORT_NONE && fp.fen.config.SortReverse {
		slices.Reverse(fp.entries.Load().([]os.DirEntry))
	}

	if fp.fen.config.FoldersFirst {
		fp.entries.Store(FoldersAtBeginning(fp.entries.Load().([]os.DirEntry)))
	}
}

func (fp *FilesPane) keepSelectionInBounds() bool {
	// I think Load()ing entries multiple times like this could be unsafe, but might realistically be very rare
	if fp.selectedEntryIndex >= len(fp.entries.Load().([]os.DirEntry)) {
		if len(fp.entries.Load().([]os.DirEntry)) > 0 {
			fp.selectedEntryIndex = len(fp.entries.Load().([]os.DirEntry)) - 1
		} else {
			fp.selectedEntryIndex = 0
		}

		return true
	}

	return false
}

// Set the selected entry from entry name, on error it keeps the selection in bounds and adds the new current selection to the fen history
func (fp *FilesPane) SetSelectedEntryFromString(entryName string) error {
	for i, entry := range fp.entries.Load().([]os.DirEntry) {
		if entry.Name() == entryName {
			fp.selectedEntryIndex = i
			return nil
		}
	}

	fp.keepSelectionInBounds()
	fp.fen.history.AddToHistory(fp.GetSelectedPathFromIndex(fp.selectedEntryIndex))

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

func (fp *FilesPane) ClampEntryIndex(index int) int {
	return max(0, min(len(fp.entries.Load().([]os.DirEntry))-1, index))
}

func (fp *FilesPane) GetSelectedPathFromIndex(index int) string {
	entryFromIndex := fp.GetSelectedEntryFromIndex(index)
	if entryFromIndex == "" {
		return ""
	}

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

func (fp *FilesPane) CanOpenFile(path string) bool {
	// We let the Go garbage collector close the file, because manually calling .Close() on it can be really slow, atleast on Linux
	// It seems to only get up to about 7 duplicate file descriptors for a single path at a time
	_, readErr := os.OpenFile(path, os.O_RDONLY, 0)
	return readErr == nil
}

func (fp *FilesPane) Draw(screen tcell.Screen) {
	/*start := time.Now()
	defer func(){
		println(strconv.FormatInt(time.Since(start).Milliseconds(), 10) + "ms")
	}()*/

	if fp.Invisible {
		return
	}

	if fp.fen.config.UiBorders {
		// TODO: Make a custom border drawing so it runs faster
		fp.Box.DrawForSubclass(screen, fp)
	}

	x, y, w, h := fp.GetInnerRect()
	if fp.isRightFilesPane && fp.parentIsEmptyFolder || (!fp.isRightFilesPane && len(fp.entries.Load().([]os.DirEntry)) <= 0) && fp.folder != filepath.Dir(fp.folder) {
		tview.Print(screen, "[:red]empty", x, y, w, tview.AlignLeft, tcell.ColorDefault)
		return
	}

	// File previews
	stat, statErr := os.Stat(fp.fen.sel)
	if fp.isRightFilesPane && len(fp.fen.config.Preview) > 0 && statErr == nil && stat.Mode().IsRegular() && fp.CanOpenFile(fp.fen.sel) && len(fp.entries.Load().([]os.DirEntry)) <= 0 {
		filenameResolved, err := filepath.EvalSymlinks(fp.fen.sel)
		if err != nil {
			filenameResolved = fp.fen.sel
		}

		if fp.fen.config.PreviewSafetyBlocklist && PathMatchesListCaseInsensitive(filenameResolved, DefaultPreviewBlocklistCaseInsensitive) {
			text := "File not previewed, it matched the default preview safety blocklist"
			lines := tview.WordWrap(text, w)
			yOffset := h/2 - len(lines)/2
			i := 0
			for _, line := range lines {
				tview.Print(screen, line, x, y+yOffset+i, w, tview.AlignCenter, tcell.ColorDefault)
				i++
			}

			tview.Print(screen, "[::d]Set fen.preview_safety_blocklist = false to disable", x, y+yOffset+i, w, tview.AlignCenter, tcell.ColorRed)
			return
		}

		for _, previewWith := range fp.fen.config.Preview {
			matched := PathMatchesList(filenameResolved, previewWith.Match) && !PathMatchesList(filenameResolved, previewWith.DoNotMatch)
			if !matched {
				continue
			}

			if previewWith.Script != "" {
				L := lua.NewState()
				defer L.Close()

				fenLuaGlobal := &FenLuaGlobal{
					SelectedFile: filenameResolved,
					x:            x,
					y:            y,
					Width:        w,
					Height:       h,
					screen:       screen,
				}

				L.SetGlobal("fen", luar.New(L, fenLuaGlobal))
				err := L.DoFile(previewWith.Script)
				if err != nil {
					fp.Box.DrawForSubclass(screen, fp)
					tview.Print(screen, "File preview Lua error:", x, y, w, tview.AlignLeft, tcell.ColorRed)
					lines := tview.WordWrap(err.Error(), w)
					for i, line := range lines {
						tview.Print(fenLuaGlobal.screen, line, x, y+1+i, w, tview.AlignLeft, tcell.ColorDefault)
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

	gitRepoContainingPath, repoErr := fp.fen.gitStatusHandler.TryFindTrackedParentGitRepository(fp.folder)

	scrollOffset := fp.GetTopScreenEntryIndex()
	for i, entry := range fp.entries.Load().([]os.DirEntry)[scrollOffset:] {
		// We don't draw at the bottom row of the screen, since it's occupied by the bottomBar
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

		_, contains := fp.fen.selected[entryFullPath]

		if contains {
			spaceForSelected = " "
			style = style.Foreground(tcell.ColorYellow)
			style = style.Bold(false) // FileColor() makes folders and executables bold
		} else {
			// Show unstaged/untracked files in red
			if fp.fen.config.GitStatus && repoErr == nil {
				if entry.IsDir() {
					if fp.fen.gitStatusHandler.FolderContainsUnstagedOrUntrackedPath(entryFullPath, gitRepoContainingPath) {
						// Same color used in the git status command
						style = style.Foreground(tcell.ColorMaroon).Bold(false) // Unstaged/untracked file in a git directory, distinct from filetype colors
					}
				} else {
					if fp.fen.gitStatusHandler.PathIsUnstagedOrUntracked(entryFullPath, gitRepoContainingPath) {
						// Same color used in the git status command
						style = style.Foreground(tcell.ColorMaroon).Bold(false) // Unstaged/untracked file in a git directory, distinct from filetype colors
					}
				}
			}
		}

		_, entryInYankSelected := fp.fen.yankSelected[entryFullPath]
		if entryInYankSelected {
			style = style.Dim(true)
		}

		//styleStr := StyleToStyleTagString(style)

		entrySizePrintedSize := 0
		if fp.showEntrySizes {
			entrySizeText, err := EntrySizeText(entryInfo, entryFullPath, fp.fen.config.HiddenFiles)
			if err != nil {
				entrySizeText = "?"
			}

			entrySizeText = " " + entrySizeText + " "
			for j := 0; j < len(entrySizeText); j++ {
				screen.SetContent(x+w-len(entrySizeText)+j-1, y+i, rune(entrySizeText[j]), nil, style)
			}

			entrySizePrintedSize = len(entrySizeText)

			//_, entrySizePrintedSize = tview.Print(screen, styleStr+"[:default] "+tview.Escape(entrySizeText)+" ", x, y+i, w-1, tview.AlignRight, tcell.ColorDefault)
		}

		//tview.Print(screen, spaceForSelected+styleStr+" "+FilenameInvisibleCharactersAsCodeHighlighted(tview.Escape(entry.Name()), styleStr)+strings.Repeat(" ", w), x, y+i, w-1-entrySizePrintedSize, tview.AlignLeft, tcell.ColorDefault)
		xToUse := x
		if spaceForSelected != "" {
			screen.SetContent(xToUse, y+i, ' ', nil, tcell.StyleDefault)
			xToUse++
		}
		screen.SetContent(xToUse, y+i, ' ', nil, style)
		xToUse++
		leftSizePrinted := PrintFilenameInvisibleCharactersAsCodeHighlighted(screen, xToUse, y+i, w-1-entrySizePrintedSize, entry.Name(), style)

		widthOffset := 0
		if fp.isRightFilesPane {
			widthOffset = 1
		}
		for j := 0; j < w-1-leftSizePrinted-entrySizePrintedSize-(xToUse-x)+widthOffset; j++ {
			screen.SetContent(xToUse+leftSizePrinted+j, y+i, ' ', nil, style)
		}

		if entryInYankSelected {
			tview.Print(screen, "[::b]*", x, y+i, w, tview.AlignLeft, tcell.ColorWhite)
		}
	}
}
