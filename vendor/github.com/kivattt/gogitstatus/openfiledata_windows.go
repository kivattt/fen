//go:build windows

package gogitstatus

import (
	"io"
	"os"
)

func openFileData(file *os.File, stat os.FileInfo) ([]byte, error) {
	if stat.Size() == 0 {
		return make([]byte, 0), nil
	}

	data := make([]byte, stat.Size())
	_, err := io.ReadFull(file, data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func closeFileData(data []byte) error {
	return nil
}
