package main

//lint:file-ignore ST1005 some user-visible messages are stored in error values and thus occasionally require capitalization

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

type History struct {
	history      []string
	historyMutex sync.Mutex
}

// Returns a full filepath (path joined with the next history entry), passing an empty path returns an error
func (h *History) GetHistoryEntryForPath(path string, hiddenFiles bool) (string, error) {
	if path == "" {
		return "", errors.New("The path argument passed was an empty string")
	}

	h.historyMutex.Lock()
	defer h.historyMutex.Unlock()

	for _, e := range h.history {
		if !hiddenFiles && strings.HasPrefix(filepath.Base(e), ".") {
			continue
		}

		sub, err := filepath.Rel(path, e)
		if err != nil {
			continue
		}

		// HACKY
		if strings.HasPrefix(sub, "..") {
			continue
		}

		if strings.HasPrefix(e, path) {
			if len(path) >= len(e) {
				continue
			}

			if theFSPathSeparator != '/' { // Windows host filesystem
				drivePath := filepath.VolumeName(path) + string(theFSPathSeparator)
				if path == drivePath {
					e = e[len(drivePath):]
				} else {
					e = e[len(path)+1:]
				}
			} else {
				if len(path) == 1 {
					e = e[1:]
				} else {
					e = e[len(path)+1:]
				}
			}
			nextSlashIdx := strings.Index(e, string(theFSPathSeparator))
			if nextSlashIdx == -1 {
				return filepath.Join(path, e), nil
			}

			return filepath.Join(path, e[:nextSlashIdx]), nil
		}
	}

	return "", errors.New("No entry found")
}

// Returns the full history filepath entry (guaranteed to be the furthest down the history), passing an empty path returns an error
func (h *History) GetHistoryFullPath(path string, hiddenFiles bool) (string, error) {
	if path == "" {
		return "", errors.New("The path argument passed was an empty string")
	}

	pathFurthestDownHistory := path
	for i := 0; ; i++ {
		pathFurtherDown, err := h.GetHistoryFirstFullPathFound(pathFurthestDownHistory, hiddenFiles)
		if i == 0 && err != nil {
			return "", errors.New("No entry found")
		}

		if err != nil || filepath.Clean(pathFurtherDown) == filepath.Clean(pathFurthestDownHistory) {
			break
		}
		pathFurthestDownHistory = pathFurtherDown
	}

	return pathFurthestDownHistory, nil
}

// Returns the first full history filepath entry found (not guaranteed to be the furthest down the history), passing an empty path returns an error
func (h *History) GetHistoryFirstFullPathFound(path string, hiddenFiles bool) (string, error) {
	if path == "" {
		return "", errors.New("The path argument passed was an empty string")
	}

	h.historyMutex.Lock()
	defer h.historyMutex.Unlock()

	for _, e := range h.history {
		if !hiddenFiles && strings.HasPrefix(filepath.Base(e), ".") {
			continue
		}

		sub, err := filepath.Rel(path, e)
		if err != nil {
			continue
		}

		// HACKY
		if strings.HasPrefix(sub, "..") {
			continue
		}

		if strings.HasPrefix(e, path) {
			if len(path) >= len(e) {
				continue
			}

			return e, nil
		}
	}

	return "", errors.New("No entry found")
}

// TODO: Disallow adding like, parents of existing stuff
func (h *History) AddToHistory(path string) {
	h.historyMutex.Lock()
	defer h.historyMutex.Unlock()

	if index := slices.Index(h.history, path); index != -1 {
		if h.history[0] == path {
			return
		}

		// Found in the history, but we need to move it to the front again
		h.history = append(h.history[:index], h.history[index+1:]...)
		h.history = append([]string{path}, h.history...)
		return
	}

	h.history = append([]string{path}, h.history...)
}

func (h *History) RemoveFromHistory(path string) error {
	h.historyMutex.Lock()
	defer h.historyMutex.Unlock()

	if index := slices.Index(h.history, path); index != -1 {
		h.history = append(h.history[:index], h.history[index+1:]...)
		return nil
	}

	return errors.New("path was not found in history")
}

func (h *History) ClearHistory() {
	h.historyMutex.Lock()
	defer h.historyMutex.Unlock()

	h.history = []string{}
}
