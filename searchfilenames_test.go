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
	loopCount := 2
	searchTerms := []string{"", ".", ".t", ".tx", ".txt", "e.txt", "le.txt", "a_file", "folders", "long_filename/folders/and/stuff", "no match", "also not a match..."}
	var s SearchFilenames

	fmt.Print("[Benchmark] Filtering (case-sensitive) ", howManyFilenames, " filenames ", loopCount*2*len(searchTerms), " times:")
	totalStart := time.Now()
	fillWithFilenames(&s, howManyFilenames, false)
	for _, searchTerm := range searchTerms {
		for i := 0; i < loopCount; i++ {
			s.Filter(searchTerm, CASE_SENSITIVE)
		}
	}
	fillWithFilenames(&s, howManyFilenames, true)
	for _, searchTerm := range searchTerms {
		for i := 0; i < loopCount; i++ {
			s.Filter(searchTerm, CASE_SENSITIVE)
		}
	}
	totalDuration := time.Since(totalStart)
	fmt.Println(" " + totalDuration.String())

	fmt.Print("[Benchmark] Filtering (case-insensitive) ", howManyFilenames, " filenames ", loopCount*2*len(searchTerms), " times:")
	totalStart = time.Now()
	fillWithFilenames(&s, howManyFilenames, false)
	for _, searchTerm := range searchTerms {
		for i := 0; i < loopCount; i++ {
			s.Filter(searchTerm, CASE_INSENSITIVE)
		}
	}
	fillWithFilenames(&s, howManyFilenames, true)
	for _, searchTerm := range searchTerms {
		for i := 0; i < loopCount; i++ {
			s.Filter(searchTerm, CASE_INSENSITIVE)
		}
	}
	totalDuration = time.Since(totalStart)
	fmt.Println(" " + totalDuration.String())
}
