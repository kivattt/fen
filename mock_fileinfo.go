package main

import (
	"os"
	"time"
)

type MockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (info MockFileInfo) Name() string {
	return info.name
}

func (info MockFileInfo) Size() int64 {
	return info.size
}

func (info MockFileInfo) Mode() os.FileMode {
	return info.mode
}

func (info MockFileInfo) ModTime() time.Time {
	return info.modTime
}

func (info MockFileInfo) IsDir() bool {
	return info.isDir
}

func (info MockFileInfo) Sys() any {
	return nil
}
