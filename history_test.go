package main

import (
	"testing"
)

func TestGetHistoryEntryForPath(t *testing.T) {
	var h History
	h.AddToHistory("/home/user/folder1/folder2")

	e, _ := h.GetHistoryEntryForPath("/home")
	if e != "/home/user" {
		t.Fatalf("Expected " + "/home/user, but got: " + e)
	}
}
