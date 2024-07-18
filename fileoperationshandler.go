package main

import (
	"errors"
	"io"
	"os"
	"strconv"
	"sync"

	dirCopy "github.com/otiai10/copy"
)

type Operation int

const (
	Rename Operation = 0
	Delete           = 1
	Copy             = 2
)

type Status int

const (
	Queued    Status = 0
	Completed        = 1
	Failed           = 2
)

type FileOperation struct {
	operation Operation
	status    Status
	path      string
	newPath   string // For Rename, Copy and Cut
}

type FileOperationsHandler struct {
	fen *Fen // So we can access fen.config.NoWrite

	entries      [][]FileOperation
	entriesMutex sync.Mutex

	workCount      int
	workCountMutex sync.Mutex
}

func (handler *FileOperationsHandler) QueueOperations(batch []FileOperation) {
	if handler.fen.config.NoWrite {
		return
	}

	handler.entriesMutex.Lock()
	handler.entries = append(handler.entries, batch)
	batchIndex := len(handler.entries) - 1
	handler.entriesMutex.Unlock()

	handler.workCountMutex.Lock()
	handler.workCount += len(batch)
	handler.workCountMutex.Unlock()

	for i, e := range batch {
		handler.doOperation(e, batchIndex, i)
	}
}

func (handler *FileOperationsHandler) QueueOperation(fileOperation FileOperation) {
	handler.QueueOperations([]FileOperation{fileOperation})
}

func (handler *FileOperationsHandler) decrementWorkCount() {
	handler.workCountMutex.Lock()
	handler.workCount--
	if handler.workCount < 0 {
		panic("Tried to decrement work count to below 0: " + strconv.Itoa(handler.workCount))
	}
	handler.workCountMutex.Unlock()
}

func (handler *FileOperationsHandler) doOperation(fileOperation FileOperation, batchIndex, index int) error {
	var statusToSet Status = Failed
	defer func() {
		handler.entriesMutex.Lock()
		handler.entries[batchIndex][index].status = statusToSet
		handler.entriesMutex.Unlock()
	}()

	if handler.fen.config.NoWrite {
		handler.decrementWorkCount()
		return errors.New("NoWrite is enabled, will not do anything")
	}

	if fileOperation.status != Queued {
		panic("doOperation got a status that was not Queued")
	}

	if fileOperation.path == "" {
		handler.decrementWorkCount()
		return errors.New("Empty path")
	}

	_, err := os.Stat(fileOperation.path)
	if err != nil {
		handler.decrementWorkCount()
		return err
	}

	switch fileOperation.operation {
	case Rename:
		if fileOperation.newPath == "" {
			handler.decrementWorkCount()
			return errors.New("Empty newPath")
		}

		_, err := os.Stat(fileOperation.newPath)
		if err == nil {
			handler.decrementWorkCount()
			return errors.New("Can't rename to an existing file")
		}
		err = os.Rename(fileOperation.path, fileOperation.newPath)
		if err != nil {
			handler.decrementWorkCount()
			return err
		}
	case Delete:
		err := os.RemoveAll(fileOperation.path)
		if err != nil {
			handler.decrementWorkCount()
			return err
		}
	case Copy:
		fi, err := os.Stat(fileOperation.path)
		if err != nil {
			handler.decrementWorkCount()
			return err
		}

		if fi.IsDir() {
			err := os.Mkdir(fileOperation.newPath, 0755)
			if err != nil {
				handler.decrementWorkCount()
				return err
			}

			err = dirCopy.Copy(fileOperation.path, fileOperation.newPath)
			if err != nil {
				handler.decrementWorkCount()
				return err
			}
		} else if fi.Mode().IsRegular() {
			source, err := os.Open(fileOperation.path)
			if err != nil {
				handler.decrementWorkCount()
				return err
			}
			defer source.Close()

			destination, err := os.Create(fileOperation.newPath)
			if err != nil {
				handler.decrementWorkCount()
				return err
			}
			defer destination.Close()

			buf := make([]byte, 8*32*1024) // 8 times larger buffer size than io.Copy()
			_, err = io.CopyBuffer(destination, source, buf)
			if err != nil {
				handler.decrementWorkCount()
				return err
			}

			destination.Chmod(fi.Mode())
		}
	default:
		panic("doOperation got an invalid operation")
	}

	handler.decrementWorkCount()
	// This is only here to update the jobcount text in the bottombar with the correct workCount value
	// This update will probably be close in time with the file watcher update preceeding it, which can look bad (atleast on xterm...)
	handler.fen.app.QueueUpdateDraw(func() {})

	statusToSet = Completed

	return nil
}
