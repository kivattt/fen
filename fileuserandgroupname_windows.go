//go:build windows
// +build windows

package main

import (
	"errors"
)

func FileUserAndGroupName(path string) (string, string, error) {
	return "", "", errors.New("Unsupported on Windows")
}
