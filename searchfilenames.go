package main

import (
	"io/fs"
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
		Box:          tview.NewBox().SetBackgroundColor(tcell.ColorBlack),
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

	for i, e := range s.filenamesFilteredIndices {
		if i >= h-1 {
			break
		}

		tview.Print(screen, s.filenames[e], x, y+i, w, tview.AlignLeft, tcell.ColorDefault)
	}

	bottomY := y + h - 1

	darkGray := tcell.NewRGBColor(20, 20, 22)
	green := tcell.NewRGBColor(0, 255, 0)
	for i := 0; i < w; i++ {
		screen.SetCell(x+i, bottomY, tcell.StyleDefault.Background(darkGray), ' ')
	}

	if s.searchTerm != "" {
		tview.Print(screen, strconv.FormatInt(int64(len(s.filenamesFilteredIndices)), 10)+" filenames matched", x, bottomY, w, tview.AlignLeft, green)
	}

	if s.finishedLoading {
		tview.Print(screen, strconv.FormatInt(int64(len(s.filenames)), 10)+" files", x, bottomY, w, tview.AlignRight, green)
	} else {
		tview.Print(screen, "Loading... "+strconv.FormatInt(int64(len(s.filenames)), 10)+" files", x, bottomY, w, tview.AlignRight, tcell.ColorWhite)
	}
}
