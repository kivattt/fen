package main

import "fmt"

type Operation int64

const (
	NoOperation Operation = iota
	NewFile
	NewFolder
	Delete
	Move
	Copy
)

type FileOperation struct {
	operation Operation
	files     [][2]string // List of [path, newPath (for Move, Copy)]
}

type Worker struct {
	entries []FileOperation
}

func (worker *Worker) Start(channel <-chan FileOperation) {
	for operation := range channel {
		switch operation.operation {
		case NoOperation:
			continue
		case NewFile:
			fmt.Println(operation.files)
		}
	}
}

func (worker *Worker) CreateFiles(paths []string, channel chan FileOperation) {
	operation := FileOperation{operation: NewFile}
	for _, path := range paths {
		operation.files = append(operation.files, [2]string{path, ""})
	}

	worker.entries = append(worker.entries, operation)
	channel <- operation
}
