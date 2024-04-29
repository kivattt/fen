//go:build windows
// +build windows

package main

import "golang.org/x/sys/windows"

func FreeDiskSpaceBytes(path string) (uint64, error) {
	var freeBytes uint64
	var totalBytes uint64
	var totalFreeBytes uint64
	err := windows.GetDiskFreeSpaceEx(windows.StringToUTF16Ptr(path), &freeBytes, &totalBytes, &totalFreeBytes)
	return freeBytes, err
}
