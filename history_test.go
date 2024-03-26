package main

import (
	//	"fmt"
	"testing"
)

func TestGetHistoryEntryForPath(t *testing.T) {
	var h History
	h.AddToHistory("/home/user/folder1/folder2")

	e, _ := h.GetHistoryEntryForPath("/home", false)
	if e != "/home/user" {
		t.Fatalf("Expected /home/user, but got: " + e)
	}

	h.AddToHistory("/home/user/test/something")
	h.AddToHistory("/home/user/test.txt")

	e, _ = h.GetHistoryEntryForPath("/home/user/test", false)
	if e != "/home/user/test/something" {
		t.Fatalf("Expected /home/user/test/something, but got: " + e)
	}
	/*
	   h.ClearHistory()
	   h.AddToHistory("/home/user/folder/file.txt")
	   h.RemoveFromHistory("/home/user/folder/file.txt")
	   h.AddToHistory("/home/user/folder")

	   fmt.Println(h.history)

	   e, _ = h.GetHistoryEntryForPath("/home/user/folder")

	   	if e != "/home/user/folder" {
	   		t.Fatalf("Expected /home/user/folder, but got: " + e)
	   	}
	*/
}
