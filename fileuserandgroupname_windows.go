//go:build windows

package main

import (
	"errors"
	"os"
)

func FileUserAndGroupName(stat os.FileInfo) (string, string, error) {
	return "", "", errors.New("Unsupported on Windows")
}
