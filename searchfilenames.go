package main

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
}

func (s *SearchFilenames) Draw(screen tcell.Screen) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Box.DrawForSubclass(screen, s)

	x, y, w, h := s.GetInnerRect()
	// 1 cell padding on left and right
	x += 1
	w -= 2

	for i, e := range s.filenamesFilteredIndices {
		if i >= h-1 {
			break
		}

		filename := s.filenames[e]

		// There is no strings.LastIndexRune() function, probably because it's slow.
		lastSlash := strings.LastIndexByte(filename, os.PathSeparator)

		matchIndices := FindSubstringAllStartIndices(filename, s.searchTerm)

		for j, c := range filename {
			if j >= w-1 {
				screen.SetCell(x+j, y+i, tcell.StyleDefault, missingSpaceRune)
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

			screen.SetCell(x+j, y+i, tcell.StyleDefault.Foreground(color), c)
		}
	}

	bottomY := y + h - 1

	green := tcell.NewRGBColor(0, 255, 0)

	color := tcell.ColorYellow
	if s.finishedLoading {
		color = green
	}
	matchCountStr := strconv.FormatInt(int64(len(s.filenamesFilteredIndices)), 10)
	filesTotalCountStr := strconv.FormatInt(int64(len(s.filenames)), 10)
	tview.Print(screen, matchCountStr + " / " + filesTotalCountStr + " files", x, bottomY, w, tview.AlignLeft, color)
}
