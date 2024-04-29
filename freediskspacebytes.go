//go:build !windows
// +build !windows

package main

import (
	"golang.org/x/sys/unix"
)

func FreeDiskSpaceBytes(path string) uint64 {
	var stat unix.Statfs_t
	unix.Statfs(path, &stat)
	return stat.Bavail * uint64(stat.Bsize)
}
