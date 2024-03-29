package main

import (
	"errors"
	//	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

type History struct {
	history []string
}

func (h *History) GetHistoryEntryForPath(path string, ignoreHiddenFiles bool) (string, error) {
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

		//fmt.Println("sub: " + sub)

		/*		if len(path) >= len(sub) {
				continue
			}*/

		//		return filepath.Join(e, sub), nil

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
		h.history = append([]string{path}, h.history...)
		return
	}

	h.history = append([]string{path}, h.history...)
}

func (h *History) RemoveFromHistory(path string) {
	if index := slices.Index(h.history, path); index != -1 {
		h.history = append(h.history[:index], h.history[index+1:]...)
	}
}

func (h *History) ClearHistory() {
	h.history = []string{}
}
