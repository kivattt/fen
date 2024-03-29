package main

import (
	"os"
	"strconv"
)

func EntrySize(path string) (string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if !stat.IsDir() {
		return strconv.FormatInt(stat.Size(), 10) + " B", nil
	} else {
		files, err := os.ReadDir(path)
		if err != nil {
			return "", err
		}
		return strconv.Itoa(len(files)), nil
	}
}
