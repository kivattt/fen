//go:build !windows

package gogitstatus

import (
	"os"
	"syscall"
)

// TODO: Keep track of opened files to see if we forgot to close one
// Keep it behind a `gogitstatus_debug_forgot_to_close_file` flag

func openFileData(file *os.File, stat os.FileInfo) ([]byte, error) {
	if stat.Size() == 0 {
		return make([]byte, 0), nil
	}

	data, err := syscall.Mmap(int(file.Fd()), 0, int(stat.Size()), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func closeFileData(data []byte) error {
	err := syscall.Munmap(data)
	if err != nil {
		return err
	}

	return nil
}
