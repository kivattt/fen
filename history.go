package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

type History struct {
	history      []string
	historyMutex sync.Mutex
}

func (h *History) GetHistoryEntryForPath(path string, ignoreHiddenFiles bool) (string, error) {
	h.historyMutex.Lock()
	defer h.historyMutex.Unlock()

	for _, e := range h.history {
		if ignoreHiddenFiles && strings.HasPrefix(filepath.Base(e), ".") {
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

			e = e[len(path)+1:]
			nextSlashIdx := strings.Index(e, string(os.PathSeparator))
			if nextSlashIdx == -1 {
				return filepath.Join(path, e), nil
			}

			return filepath.Join(path, e[:nextSlashIdx]), nil
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

func (h *History) RemoveFromHistory(path string) {
	h.historyMutex.Lock()
	defer h.historyMutex.Unlock()

	if index := slices.Index(h.history, path); index != -1 {
		h.history = append(h.history[:index], h.history[index+1:]...)
	}
}

func (h *History) ClearHistory() {
	h.historyMutex.Lock()
	defer h.historyMutex.Unlock()

	h.history = []string{}
}
