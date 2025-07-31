package main

import (
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchFilenames struct {
	*tview.Box
	fen *Fen

	mutex sync.Mutex
	wg sync.WaitGroup
	searchTerm string
	filenames []string
	filenamesFilteredIndices []int
	cancel bool
}

func NewSearchFilenames(fen *Fen) *SearchFilenames {
	s := SearchFilenames{
		Box: tview.NewBox().SetBackgroundColor(tcell.ColorBlack),
		fen: fen,
	}

	s.wg.Add(1)
	go s.GatherFiles("..")
	go func() {
		s.wg.Wait()
		s.fen.app.QueueUpdateDraw(func() {
			s.Filter("")
			// TODO: Only show if it took > 1 second or something
			s.fen.bottomBar.TemporarilyShowTextInstead("Finished loading files")
		})
	}()

	return &s
}

func (s *SearchFilenames) GatherFiles(path string) {
	// Unhandled error
	_ = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		s.mutex.Lock()
		if s.cancel {
			s.mutex.Unlock()
			return filepath.SkipAll
		}

		s.filenames = append(s.filenames, path)

		//s.Filter(s.searchTerm)

		s.mutex.Unlock()
		return nil
	})

	s.wg.Done()
}

func (s *SearchFilenames) Filter(text string) {
	// Necessary? Do we want this?
	s.mutex.Lock()
	defer s.mutex.Unlock()

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

/*func (s *SearchFilenames) InputHandler() func(ek *tcell.EventKey, f func(p tview.Primitive)) {
	return s.WrapInputHandler(func(ek *tcell.EventKey, f func(p tview.Primitive)) {
		if ek.Rune() == 'a' {
			s.filenamesFilteredIndices = slices.Delete(s.filenamesFilteredIndices, 0, 1)
		}
	})
}*/

func (s *SearchFilenames) Draw(screen tcell.Screen) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Box.DrawForSubclass(screen, s)

	x, y, w, h := s.GetInnerRect()

	//s.filenames = []string{"file1", "file2", "file3", "file4", "file5", "file6", "file7"}
	//s.filenamesFilteredIndices = []int{0, 2, 4, 5, 6}

	for i, e := range s.filenamesFilteredIndices {
		if i >= h {
			break
		}

		tview.Print(screen, s.filenames[e], x, y+i, w, tview.AlignLeft, tcell.ColorDefault)
		//tview.PrintSimple(screen, s.filenames[e], x, y+i)
	}
}
