//go:build freebsd

package main

import (
	"path/filepath"

	"golang.org/x/sys/unix"
)

// Returns free disk space in bytes, and whether its negative
func FreeDiskSpaceBytes(path string) (uint64, bool, error) {
	var stat unix.Statfs_t
	err := unix.Statfs(filepath.Dir(path), &stat)

	abs := func(n int64) int64 {
		if n < 0 {
			return -n
		}

		return n
	}

	// This first uint64 cast is necessary for building on FreeBSD (go version go1.21.9 freebsd/amd64)
	free := uint64(abs(stat.Bavail)) * stat.Bsize
	negative := stat.Bavail < 0
	return free, negative, err
}
