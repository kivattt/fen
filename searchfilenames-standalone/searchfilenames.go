package main

// TODO: Add sorting based on fen.config.SortBy
// Make the file-loading thread insert at correct positions with a binary search.
// That way we avoid re-sorting every 200ms when we re-filter and re-draw the screen

/*
	+---------------------+
	| Search format ideas |
	+---------------------+

	Match "file"
	file

	Match all ending with ".go"
	*.go

	Match all starting with "file"
	file*

	Invert search
	!file

	+------------------+
	| Escape sequences |
	+------------------+

	Match "*.go"
	\*.go

	Match "file*"
	file\*

	Match "!file"
	\!file

	+-----------------------------------------+
	| Requiring whitespace-trimming / parsing |
	+-----------------------------------------+

	Match "a" OR "b"
	~a ~b

	Match *.go OR *.txt
	~*.go ~*.txt
*/

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchFilenames struct {
	*tview.Box
	fen *Fen

	mutex                    sync.Mutex
	wg                       sync.WaitGroup
	searchTerm               string
	lastSearchTerm           string
	filenames                []string
	filenamesFilteredIndices []int

	filenamesFilteredIndicesUnderlying []int
	selectedFilenameIndex              int
	selectedFilename                   string
	scrollLocked                       bool

	cancel               bool
	finishedLoading      bool
	lastDrawTime         time.Time
	firstDraw            bool
	selectLastOnNextDraw bool
}

func NewSearchFilenames(fen *Fen) *SearchFilenames {
	s := SearchFilenames{
		Box:                  tview.NewBox().SetBackgroundColor(tcell.ColorDefault),
		fen:                  fen,
		lastDrawTime:         time.Now(),
		firstDraw:            true, // This is used so we can have a shorter delay on the first draw and longer for later ones
		selectLastOnNextDraw: true, // Make sure the last element is selected on the first draw
	}

	s.wg.Add(1)
	go s.GatherFiles(fen.wd)
	go func() {
		s.wg.Wait()

		// All files have been loaded
		if !s.cancel {
			s.fen.app.QueueUpdateDraw(func() {
				s.mutex.Lock()
				{
					s.Filter(s.searchTerm)
					if s.scrollLocked {
						s.SetSelectedIndexToSelectedFilename()
					}
					s.scrollLocked = false
					s.finishedLoading = true
				}
				s.mutex.Unlock()
			})
		}
	}()

	return &s
}

// TODO: Measure the performance of this function.
// We might make it faster by searching around the previous selectedFilenameIndex
// (backwards, then forwards?)
func (s *SearchFilenames) SetSelectedIndexToSelectedFilename() {
	if s.selectedFilename == "" {
		s.scrollLocked = false
		return
	}

	if s.searchTerm == "" {
		found := slices.Index(s.filenames, s.selectedFilename)
		if found != -1 {
			s.selectedFilenameIndex = found
			return
		}
	} else {
		for i, e := range s.filenamesFilteredIndices {
			if s.filenames[e] == s.selectedFilename {
				s.selectedFilenameIndex = i
				return
			}
		}
	}

	// Didn't find it in the list
	s.scrollLocked = false
}

func (s *SearchFilenames) GetSelectedFilename() (string, error) {
	out := ""
	if s.searchTerm == "" {
		if s.selectedFilenameIndex < len(s.filenames) {
			out = s.filenames[s.selectedFilenameIndex]
			if out == "" {
				panic("Empty string selected in search filenames popup")
			}
		}
	} else {
		if s.selectedFilenameIndex < len(s.filenamesFilteredIndices) {
			out = s.filenames[s.filenamesFilteredIndices[s.selectedFilenameIndex]]
			if out == "" {
				panic("Empty string selected in search filenames popup")
			}
		}
	}

	if out == "" {
		return out, errors.New("No filename selected")
	} else {
		return out, nil
	}
}

func (s *SearchFilenames) SetSelectedIndexAndLockScrollIfLoading(index int) {
	s.selectedFilenameIndex = index

	if !s.finishedLoading {
		var err error
		s.selectedFilename, err = s.GetSelectedFilename()
		if err == nil {
			s.scrollLocked = true
		}
	}
}

func (s *SearchFilenames) GatherFiles(pathInput string) {
	// I forgot the difference between EvalSymlinks and directly stat-ing the symlink to get its path.
	// I think EvalSymlinks does so recursively and is slower.
	// We can afford it to be slow, this is only ran once when you open the search filenames popup.
	basePathSymlinkResolved, err := filepath.EvalSymlinks(pathInput)
	if err != nil {
		s.fen.bottomBar.TemporarilyShowTextInstead(err.Error())
		return
	}

	var basePathLength int
	if basePathSymlinkResolved == "." {
		basePathLength = 0
	} else {
		basePathLength = 1 + len(basePathSymlinkResolved)
	}

	if runtime.GOOS == "windows" {
		volumeName := filepath.VolumeName(pathInput) + "\\"
		if basePathSymlinkResolved == volumeName {
			basePathLength = len(volumeName)
		}
	} else {
		if basePathSymlinkResolved == "/" {
			basePathLength = 1
		}
	}

	// FIXME: Unfortunately, WalkDir doesn't resolve symlink directories. Do you think anyone will notice? :3

	// Unhandled error
	_ = filepath.WalkDir(basePathSymlinkResolved, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}

		// Hide files/folders starting with '.' if hidden files are hidden
		if !s.fen.config.HiddenFiles && d.Name()[0] == '.' {
			if d.IsDir() {
				return filepath.SkipDir
			} else {
				return nil
			}
		}

		// Hide directories, and "." directory
		if d.IsDir() {
			return nil
		}

		s.mutex.Lock()
		{
			// Are we cancelled?
			if s.cancel {
				s.mutex.Unlock()
				return filepath.SkipAll
			}

			pathName := path[basePathLength:]
			s.filenames = append(s.filenames, pathName)
		}
		s.mutex.Unlock()

		delay := 200 * time.Millisecond
		if s.firstDraw { // A mutex is not necessary here.
			// We use a shorter delay for the first draw so the user isn't left waiting 200ms for the first files to show up on-screen.
			delay = 10 * time.Millisecond
		}

		// If we've loaded atleast 100 files, don't bother waiting the whole 10 milliseconds for the first draw
		// TODO: Store the time it took to first draw, and show in some debug info in the UI
		if time.Since(s.lastDrawTime) > delay || (s.firstDraw && len(s.filenames) >= 100) {
			s.firstDraw = false

			s.fen.app.QueueUpdateDraw(func() {
				s.mutex.Lock()
				{
					s.Filter(s.searchTerm)
					if s.scrollLocked {
						s.SetSelectedIndexToSelectedFilename()
					}
				}
				s.mutex.Unlock()
			})
			s.lastDrawTime = time.Now()
		}

		return nil
	})

	s.wg.Done()
}

// You need to manually lock / unlock the mutex to use this function
func (s *SearchFilenames) Filter(text string) {
	s.lastSearchTerm = s.searchTerm
	s.searchTerm = text

	if s.searchTerm == "" {
		s.selectedFilenameIndex = max(0, len(s.filenames)-1)
		return
	}

	// On successive characters after the first, we only need to filter s.filenamesFilteredIndices
	if s.finishedLoading && len(s.lastSearchTerm) > 0 && (s.searchTerm != s.lastSearchTerm) && strings.HasPrefix(s.searchTerm, s.lastSearchTerm) {
		numGoroutines := runtime.NumCPU()
		arraySlices := SpreadArrayIntoSlicesForGoroutines(len(s.filenamesFilteredIndices), numGoroutines)

		resultsList := make([][]int, len(arraySlices))
		var wg sync.WaitGroup
		for goroutineIndex, slice := range arraySlices {
			wg.Add(1)
			go func(slice Slice, goroutineIndex int) {
				ourList := make([]int, 0, slice.length)

				for i := slice.start; i < slice.start+slice.length; i++ {
					filenameIndex := s.filenamesFilteredIndices[i]
					filename := s.filenames[filenameIndex]
					if strings.Contains(filename, s.searchTerm) {
						ourList = append(ourList, filenameIndex)
					}
				}

				resultsList[goroutineIndex] = ourList
				wg.Done()
			}(slice, goroutineIndex)
		}

		wg.Wait()

		// Merge the search results of all the goroutines
		s.filenamesFilteredIndices = []int{}
		for _, e := range resultsList {
			s.filenamesFilteredIndices = append(s.filenamesFilteredIndices, e...)
		}
	} else {
		// Let's grow the filenamesFilteredIndices by atleast 0.5 MB whenever we need to.
		// https://go.dev/wiki/SliceTricks
		if len(s.filenamesFilteredIndicesUnderlying) < len(s.filenames) {
			// 125000 * 4 bytes = 0.5MB
			grow := max(125000, len(s.filenames)-len(s.filenamesFilteredIndicesUnderlying))
			s.filenamesFilteredIndicesUnderlying = append(make([]int, len(s.filenamesFilteredIndicesUnderlying)+grow), s.filenamesFilteredIndicesUnderlying...)
		}

		numGoroutines := runtime.NumCPU()
		arraySlices := SpreadArrayIntoSlicesForGoroutines(len(s.filenames), numGoroutines)

		// FIXME: Prevent allocating on every keypress.
		// Do something similar to the array grow trick above.
		// We can do that, because runtime.NumCPU() is constant over the program execution
		resultsList := make([][]int, len(arraySlices))
		var wg sync.WaitGroup
		for goroutineIndex, slice := range arraySlices {
			wg.Add(1)
			go func(slice Slice, goroutineIndex int) {
				ourList := make([]int, 0, slice.length)

				for i := slice.start; i < slice.start+slice.length; i++ {
					filename := s.filenames[i]
					if strings.Contains(filename, s.searchTerm) {
						ourList = append(ourList, i)
					}
				}

				resultsList[goroutineIndex] = ourList
				wg.Done()
			}(slice, goroutineIndex)
		}

		wg.Wait()

		// Merge the search results of all the goroutines
		i := 0
		for _, e := range resultsList {
			copy(s.filenamesFilteredIndicesUnderlying[i:], e[:])
			i += len(e)
		}

		s.filenamesFilteredIndices = s.filenamesFilteredIndicesUnderlying[:i]
	}

	if !s.scrollLocked {
		s.selectedFilenameIndex = max(0, len(s.filenamesFilteredIndices)-1)
	}
}

func (s *SearchFilenames) Draw(screen tcell.Screen) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Box.DrawForSubclass(screen, s)

	x, y, w, h := s.GetInnerRect()
	// 1 cell padding on left and right
	x += 1
	w -= 2
	h -= 1

	filenamesLen := len(s.filenamesFilteredIndices)
	if s.searchTerm == "" {
		filenamesLen = len(s.filenames)
	}

	if s.selectLastOnNextDraw && !s.scrollLocked {
		s.selectedFilenameIndex = max(0, filenamesLen-1)
		s.selectLastOnNextDraw = false
	}

	scrollOffset := max(0, min(filenamesLen-h+1, s.selectedFilenameIndex-h/2))
	startY := y + max(0, h-filenamesLen-1)

	drawFilename := func(i int, filename string) {
		style := tcell.StyleDefault
		if i == s.selectedFilenameIndex-scrollOffset {
			style = style.Reverse(true)
		}

		// There is no strings.LastIndexRune() function, probably because it's slow.
		lastSlash := strings.LastIndexByte(filename, os.PathSeparator)

		// The search match indices in terms of bytes, not runes!
		matchIndices := FindSubstringAllStartIndices(filename, s.searchTerm)

		yPos := startY + i
		runeIndex := -1
		for byteIndex, c := range filename {
			runeIndex += 1
			if runeIndex >= w-1 {
				screen.SetContent(x+runeIndex, yPos, missingSpaceRune, nil, style)
				break
			}

			color := tcell.ColorBlue
			if runeIndex > lastSlash {
				color = tcell.ColorDefault
			}

			for _, matchIndex := range matchIndices {
				if byteIndex >= matchIndex && byteIndex < matchIndex+len(s.searchTerm) {
					color = tcell.ColorOrange
					break
				}
			}

			screen.SetContent(x+runeIndex, yPos, c, nil, style.Foreground(color))
		}
	}

	if s.searchTerm == "" {
		for i, filename := range s.filenames[scrollOffset:] {
			if i >= h-1 {
				break
			}
			drawFilename(i, filename)
		}
	} else {
		for i, e := range s.filenamesFilteredIndices[scrollOffset:] {
			if i >= h-1 {
				break
			}
			filename := s.filenames[e]
			drawFilename(i, filename)
		}
	}

	bottomY := y + h
	green := tcell.NewRGBColor(0, 255, 0)
	color := tcell.ColorYellow
	if s.finishedLoading {
		color = green
	}

	if filenamesLen == 0 {
		color = tcell.ColorGray
	}

	matchCountStr := strconv.FormatInt(int64(filenamesLen), 10)
	filesTotalCountStr := strconv.FormatInt(int64(len(s.filenames)), 10)
	tview.Print(screen, matchCountStr+" / "+filesTotalCountStr+" files", x, bottomY, w, tview.AlignLeft, color)

	var scrollPercentageStr string
	if filenamesLen < h {
		scrollPercentageStr = "All"
	} else {
		if s.selectedFilenameIndex == 0 { // Prevent divide-by-zero
			scrollPercentageStr = "Top"
		} else if s.selectedFilenameIndex == filenamesLen-1 {
			scrollPercentageStr = "Bot"
		} else {
			scrollPercentageStr = strconv.FormatInt(int64(float32(s.selectedFilenameIndex)/float32(filenamesLen-1)*100), 10) + "%"
		}
	}
	tview.Print(screen, scrollPercentageStr, x, bottomY, w, tview.AlignRight, tcell.ColorDefault)
}

func (s *SearchFilenames) GoUp() {
	if s.selectedFilenameIndex > 0 {
		s.SetSelectedIndexAndLockScrollIfLoading(s.selectedFilenameIndex - 1)
	}
}

func (s *SearchFilenames) GoDown() {
	filenamesLen := len(s.filenamesFilteredIndices)
	if s.searchTerm == "" {
		filenamesLen = len(s.filenames)
	}

	if s.selectedFilenameIndex < filenamesLen-1 {
		s.SetSelectedIndexAndLockScrollIfLoading(s.selectedFilenameIndex + 1)
	}
}

func (s *SearchFilenames) GoTop() {
	if s.selectedFilenameIndex != 0 {
		s.SetSelectedIndexAndLockScrollIfLoading(0)
	}
}

func (s *SearchFilenames) GoBottom() {
	filenamesLen := len(s.filenamesFilteredIndices)
	if s.searchTerm == "" {
		filenamesLen = len(s.filenames)
	}

	if s.selectedFilenameIndex != max(0, filenamesLen-1) {
		s.selectedFilenameIndex = max(0, filenamesLen-1)
		s.scrollLocked = false
	}
}

func (s *SearchFilenames) PageUp() {
	_, _, _, height := s.Box.GetInnerRect()
	height = max(5, height-10) // Padding
	if s.selectedFilenameIndex != 0 {
		s.SetSelectedIndexAndLockScrollIfLoading(max(0, s.selectedFilenameIndex-height))
	}
}

func (s *SearchFilenames) PageDown() {
	filenamesLen := len(s.filenamesFilteredIndices)
	if s.searchTerm == "" {
		filenamesLen = len(s.filenames)
	}

	_, _, _, height := s.Box.GetInnerRect()
	height = max(5, height-10) // Padding
	index := max(0, min(filenamesLen-1, s.selectedFilenameIndex+height))

	if s.selectedFilenameIndex != index {
		s.SetSelectedIndexAndLockScrollIfLoading(index)
	}
}
