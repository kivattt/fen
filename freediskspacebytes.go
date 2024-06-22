//go:build !windows
// +build !windows

package main

import (
	"path/filepath"

	"golang.org/x/sys/unix"
)

func FreeDiskSpaceBytes(path string) (uint64, error) {
	var stat unix.Statfs_t
	err := unix.Statfs(filepath.Dir(path), &stat)
	// This first uint64 cast is necessary for building on FreeBSD (go version go1.21.9 freebsd/amd64)
	return uint64(stat.Bavail) * uint64(stat.Bsize), err
}
