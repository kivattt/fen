package main

import (
	"fmt"
	"testing"
	"time"
)

func fillWithFilenames(s *SearchFilenames, howManyFilenames int) {
	s.filenames = make([]string, howManyFilenames)
	for i := 0; i < howManyFilenames; i++ {
		s.filenames[i] = "long_filename/folders/and/stuff/a_file.txt"
	}
}

// Actually a benchmark, but I don't know how to write it with the benchmark test API (b *testing.B)
func TestFilter(t *testing.T) {
	howManyFilenames := 2000000
	loopCount := 10
	searchTerms := []string{"", ".", ".t", ".tx", ".txt", "e.txt", "le.txt", "a_file", "folders", "long_filename/folders/and/stuff", "no match", "also not a match..."}
	fmt.Print("[Benchmark] Filtering ", howManyFilenames, " filenames ", loopCount*len(searchTerms), " times:")

	var s SearchFilenames

	fillWithFilenames(&s, howManyFilenames)


	totalStart := time.Now()

	for _, searchTerm := range searchTerms {
		for i := 0; i < loopCount; i++ {
			s.Filter(searchTerm)
		}
	}

	totalDuration := time.Since(totalStart)
	fmt.Println(" " + totalDuration.String())
}
