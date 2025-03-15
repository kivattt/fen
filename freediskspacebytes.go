//go:build !windows && !freebsd

package main

import (
	"path/filepath"

	"golang.org/x/sys/unix"
)

func FreeDiskSpaceBytes(path string) (uint64, bool, error) {
	var stat unix.Statfs_t
	err := unix.Statfs(filepath.Dir(path), &stat)
	return stat.Bavail * uint64(stat.Bsize), false, err
}
