//go:build windows

package main

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

func FreeDiskSpaceBytes(path string) (uint64, bool, error) {
	var freeBytes uint64
	var totalBytes uint64
	var totalFreeBytes uint64
	err := windows.GetDiskFreeSpaceEx(windows.StringToUTF16Ptr(filepath.Dir(path)), &freeBytes, &totalBytes, &totalFreeBytes)
	return freeBytes, false, err
}
