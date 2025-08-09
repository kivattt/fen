package main

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
	selectedFilenameIndex int
	cancel                   bool
	finishedLoading          bool
	lastDrawTime             time.Time
}

func NewSearchFilenames(fen *Fen) *SearchFilenames {
	s := SearchFilenames{
		Box:          tview.NewBox().SetBackgroundColor(tcell.ColorDefault),
		fen:          fen,
		lastDrawTime: time.Now(),
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
	// Unhandled error
	_ = filepath.WalkDir(pathInput, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}

		// Hide directories, and "." directory
		if d.IsDir() {
			return nil
		}

		s.mutex.Lock()
		if s.cancel {
			s.mutex.Unlock()
			return filepath.SkipAll
		}

		pathName, err := filepath.Rel(pathInput, path)
		if err != nil {
			s.mutex.Unlock()
			return nil
		}

		s.filenames = append(s.filenames, pathName)
		s.mutex.Unlock()

		if time.Since(s.lastDrawTime) > time.Duration(s.fen.config.FileEventIntervalMillis)*time.Millisecond {
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
	// PERF: Only allocate once, instead of every key-press...
	s.filenamesFilteredIndices = make([]int, len(s.filenames))
	j := 0

	for i, e := range s.filenames {
		if strings.Contains(e, text) {
			s.filenamesFilteredIndices[j] = i
			j += 1
		}
	}

	s.filenamesFilteredIndices = s.filenamesFilteredIndices[:j]
	s.selectedFilenameIndex = max(0, len(s.filenamesFilteredIndices) - 1)
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

	scrollOffset := max(0, min(len(s.filenamesFilteredIndices) - h + 1, s.selectedFilenameIndex - h/2))
	startY := y + max(0, h - len(s.filenamesFilteredIndices) - 1)

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
		if i == s.selectedFilenameIndex - scrollOffset {
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
				if j >= matchIndex && j < matchIndex + len(s.searchTerm) {
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
	tview.Print(screen, matchCountStr + " / " + filesTotalCountStr + " files", x, bottomY, w, tview.AlignLeft, color)

	var scrollPercentageStr string
	if len(s.filenamesFilteredIndices) < h {
		scrollPercentageStr = "All"
	} else {
		if s.selectedFilenameIndex == 0 { // Prevent divide-by-zero
			scrollPercentageStr = "Top"
		} else if s.selectedFilenameIndex == len(s.filenamesFilteredIndices) - 1 {
			scrollPercentageStr = "Bot"
		} else {
			scrollPercentageStr = strconv.FormatInt(int64(float32(s.selectedFilenameIndex) / float32(len(s.filenamesFilteredIndices) - 1) * 100), 10) + "%"
		}
	}
	tview.Print(screen, scrollPercentageStr, x, bottomY, w, tview.AlignRight, tcell.ColorDefault)
}

func (s *SearchFilenames) GoUp() {
	// Do we want the mutex lock/unlock here?
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.selectedFilenameIndex > 0 {
		s.selectedFilenameIndex -= 1
	}
}

func (s *SearchFilenames) GoDown() {
	// Do we want the mutex lock/unlock here?
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.selectedFilenameIndex < len(s.filenamesFilteredIndices) - 1 {
		s.selectedFilenameIndex += 1
	}
}

func (s *SearchFilenames) GoTop() {
	// Do we want the mutex lock/unlock here?
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.selectedFilenameIndex = 0
}

func (s *SearchFilenames) GoBottom() {
	// Do we want the mutex lock/unlock here?
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.selectedFilenameIndex = max(0, len(s.filenamesFilteredIndices) - 1)
}

func (s *SearchFilenames) PageUp() {
	// Do we want the mutex lock/unlock here?
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, _, _, height := s.Box.GetInnerRect()
	height = max(5, height-10) // Padding
	s.selectedFilenameIndex = max(0, s.selectedFilenameIndex - height)
}

func (s *SearchFilenames) PageDown() {
	// Do we want the mutex lock/unlock here?
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, _, _, height := s.Box.GetInnerRect()
	height = max(5, height-10) // Padding
	s.selectedFilenameIndex = max(0, min(len(s.filenamesFilteredIndices) - 1, s.selectedFilenameIndex + height))
}
