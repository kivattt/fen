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
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
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
	filenames                []string
	filenamesFilteredIndices []int

	filenamesFilteredIndicesUnderlying []int
	selectedFilenameIndex              int
	cancel                             bool
	finishedLoading                    bool
	lastDrawTime                       time.Time
	firstDraw                          bool
}

func NewSearchFilenames(fen *Fen) *SearchFilenames {
	s := SearchFilenames{
		Box:          tview.NewBox().SetBackgroundColor(tcell.ColorDefault),
		fen:          fen,
		lastDrawTime: time.Now(),
		firstDraw:    true, // This is used so we can have a shorter delay on the first draw and longer for later ones
	}

	s.wg.Add(1)
	go s.GatherFiles(fen.wd)
	go func() {
		s.wg.Wait()

		if !s.cancel {
			s.fen.app.QueueUpdateDraw(func() {
				s.mutex.Lock()
				s.Filter(s.searchTerm)
				s.finishedLoading = true
				s.mutex.Unlock()
			})
		}
	}()

	return &s
}

func (s *SearchFilenames) GatherFiles(pathInput string) {
	// I forgot the difference between EvalSymlinks and directly stat-ing the symlink to get its path.
	// I think EvalSymlinks does so recursively and is slower.
	// We can afford it to be slow, this is only ran once when you open the search filenames popup.
	pathSymlinkResolved, err := filepath.EvalSymlinks(pathInput)
	if err != nil {
		s.fen.bottomBar.TemporarilyShowTextInstead(err.Error())
		return
	}

	// FIXME: Unfortunately, WalkDir doesn't resolve symlink directories. Do you think anyone will notice? :3

	// Unhandled error
	_ = filepath.WalkDir(pathSymlinkResolved, func(path string, d fs.DirEntry, err error) error {
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

			pathName, err := filepath.Rel(pathSymlinkResolved, path)
			if err != nil {
				s.mutex.Unlock()
				return nil
			}

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
				s.Filter(s.searchTerm)
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
	// Let's grow the filenamesFilteredIndices by 0.5 MB whenever we need to.
	// https://go.dev/wiki/SliceTricks
	grow := 125000 // 125000 * 4 bytes = 0.5 MB
	if len(s.filenamesFilteredIndicesUnderlying) < len(s.filenames) {
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
			var ourList []int

			for i := 0; i < slice.length; i++ {
				filename := s.filenames[slice.start+i]
				if strings.Contains(filename, text) {
					ourList = append(ourList, slice.start+i)
				}
			}

			resultsList[goroutineIndex] = ourList
			wg.Done()
		}(slice, goroutineIndex)
	}

	wg.Wait()

	// Merge the search results of all the goroutines
	j := 0
	for _, e := range resultsList {
		copy(s.filenamesFilteredIndicesUnderlying[j:], e[:])
		j += len(e)
	}

	s.filenamesFilteredIndices = s.filenamesFilteredIndicesUnderlying[:j]
	s.selectedFilenameIndex = max(0, len(s.filenamesFilteredIndices)-1)
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

	scrollOffset := max(0, min(len(s.filenamesFilteredIndices)-h+1, s.selectedFilenameIndex-h/2))
	startY := y + max(0, h-len(s.filenamesFilteredIndices)-1)

	/*debugString := fmt.Sprint("scrollOffset: ", scrollOffset, " startY: ", startY, " selected: ", s.selectedFilenameIndex)
	if s.selectedFilenameIndex > 0 {
		debugString += " " + s.filenames[s.filenamesFilteredIndices[s.selectedFilenameIndex]]
	}
	s.fen.bottomBar.TemporarilyShowTextInstead(debugString)*/

	for i, e := range s.filenamesFilteredIndices[scrollOffset:] {
		if i >= h-1 {
			break
		}

		filename := s.filenames[e]

		// There is no strings.LastIndexRune() function, probably because it's slow.
		lastSlash := strings.LastIndexByte(filename, os.PathSeparator)

		matchIndices := FindSubstringAllStartIndices(filename, s.searchTerm)

		style := tcell.StyleDefault
		if i == s.selectedFilenameIndex-scrollOffset {
			style = style.Reverse(true)
		}

		yPos := startY + i

		for j, c := range filename {
			if j >= w-1 {
				screen.SetCell(x+j, yPos, style, missingSpaceRune)
				break
			}

			color := tcell.ColorBlue
			if j > lastSlash {
				color = tcell.ColorDefault
			}

			for _, matchIndex := range matchIndices {
				if j >= matchIndex && j < matchIndex+len(s.searchTerm) {
					color = tcell.ColorOrange
					break
				}
			}

			screen.SetCell(x+j, yPos, style.Foreground(color), c)
		}
	}

	bottomY := y + h

	green := tcell.NewRGBColor(0, 255, 0)

	color := tcell.ColorYellow
	if s.finishedLoading {
		color = green
	}
	matchCountStr := strconv.FormatInt(int64(len(s.filenamesFilteredIndices)), 10)
	filesTotalCountStr := strconv.FormatInt(int64(len(s.filenames)), 10)
	tview.Print(screen, matchCountStr+" / "+filesTotalCountStr+" files", x, bottomY, w, tview.AlignLeft, color)

	var scrollPercentageStr string
	if len(s.filenamesFilteredIndices) < h {
		scrollPercentageStr = "All"
	} else {
		if s.selectedFilenameIndex == 0 { // Prevent divide-by-zero
			scrollPercentageStr = "Top"
		} else if s.selectedFilenameIndex == len(s.filenamesFilteredIndices)-1 {
			scrollPercentageStr = "Bot"
		} else {
			scrollPercentageStr = strconv.FormatInt(int64(float32(s.selectedFilenameIndex)/float32(len(s.filenamesFilteredIndices)-1)*100), 10) + "%"
		}
	}
	tview.Print(screen, scrollPercentageStr, x, bottomY, w, tview.AlignRight, tcell.ColorDefault)
}

func (s *SearchFilenames) GoUp() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.selectedFilenameIndex > 0 {
		s.selectedFilenameIndex -= 1
	}
}

func (s *SearchFilenames) GoDown() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.selectedFilenameIndex < len(s.filenamesFilteredIndices)-1 {
		s.selectedFilenameIndex += 1
	}
}

func (s *SearchFilenames) GoTop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.selectedFilenameIndex = 0
}

func (s *SearchFilenames) GoBottom() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.selectedFilenameIndex = max(0, len(s.filenamesFilteredIndices)-1)
}

func (s *SearchFilenames) PageUp() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, _, _, height := s.Box.GetInnerRect()
	height = max(5, height-10) // Padding
	s.selectedFilenameIndex = max(0, s.selectedFilenameIndex-height)
}

func (s *SearchFilenames) PageDown() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, _, _, height := s.Box.GetInnerRect()
	height = max(5, height-10) // Padding
	s.selectedFilenameIndex = max(0, min(len(s.filenamesFilteredIndices)-1, s.selectedFilenameIndex+height))
}
