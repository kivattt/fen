package main

import (
	"fmt"
	"testing"
	"time"
)

func fillWithFilenames(s *SearchFilenames, howManyFilenames int, uppercase bool) {
	s.filenames = make([]string, howManyFilenames)
	if uppercase {
		for i := 0; i < howManyFilenames; i++ {
			s.filenames[i] = "LONG_FILENAME/FOLDERS/AND/STUFF/A_FILE.TXT"
		}
	} else {
		for i := 0; i < howManyFilenames; i++ {
			s.filenames[i] = "long_filename/folders/and/stuff/a_file.txt"
		}
	}
}

// Actually a benchmark, but I don't know how to write it with the benchmark test API (b *testing.B)
func TestFilter(t *testing.T) {
	howManyFilenames := 1000000
	loopCount := 4
	searchTerms := []string{"", ".", ".t", ".tx", ".txt", "e.txt", "le.txt", "a_file", "folders", "long_filename/folders/and/stuff", "no match", "also not a match..."}
	fmt.Print("[Benchmark] Filtering ", howManyFilenames, " filenames ", loopCount*2*len(searchTerms), " times:")

	var s SearchFilenames

	totalStart := time.Now()

	fillWithFilenames(&s, howManyFilenames, false)
	for _, searchTerm := range searchTerms {
		for i := 0; i < loopCount; i++ {
			s.Filter(searchTerm)
		}
	}
	fillWithFilenames(&s, howManyFilenames, true)
	for _, searchTerm := range searchTerms {
		for i := 0; i < loopCount; i++ {
			s.Filter(searchTerm)
		}
	}

	totalDuration := time.Since(totalStart)
	fmt.Println(" " + totalDuration.String())
}
