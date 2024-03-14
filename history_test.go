package main

import (
	"testing"
)

func TestGetHistoryEntryForPath(t *testing.T) {
	var h History
	h.AddToHistory("/home/user/folder1/folder2")

	e, _ := h.GetHistoryEntryForPath("/home")
	if e != "/home/user" {
		t.Fatalf("Expected /home/user, but got: " + e)
	}

	h.AddToHistory("/home/user/test/something")
	h.AddToHistory("/home/user/test.txt")

	e, _ = h.GetHistoryEntryForPath("/home/user/test")
	if e != "/home/user/test/something" {
		t.Fatalf("Expected /home/user/test/something, but got: " + e)
	}
}
