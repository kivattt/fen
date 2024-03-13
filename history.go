package main

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
)

type History struct {
	history []string
}

func (h *History) GetHistoryEntryForPath(path string) (string, error) {
	for _, e := range h.history {
		if strings.HasPrefix(e, path) {
			if len(path) >= len(e) {
				continue
			}

			e = e[len(path)+1:]
			nextSlashIdx := strings.Index(e, "/")
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
	if index := slices.Index(h.history, path); index != -1 {
		if h.history[0] == path {
			return
		}

		// Found in the history, but we need to move it to the front again
		h.history = append(h.history[:index], h.history[index+1:]...)
		h.history = slices.Concat([]string{path}, h.history)
		return
	}

	h.history = slices.Concat([]string{path}, h.history)
}
