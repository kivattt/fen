package main

import "testing"

func TestGetHistoryEntryForPath(t *testing.T) {
	var h History
	h.AddToHistory("/home/user/folder1/folder2")

	e, _ := h.GetHistoryEntryForPath("/home", true)
	if e != "/home/user" {
		t.Fatalf("Expected /home/user, but got: " + e)
	}

	h.AddToHistory("/home/user/test/something")
	h.AddToHistory("/home/user/test.txt")

	e, _ = h.GetHistoryEntryForPath("/home/user/test", true)
	if e != "/home/user/test/something" {
		t.Fatalf("Expected /home/user/test/something, but got: " + e)
	}

	e, _ = h.GetHistoryEntryForPath("/", true)
	if e != "/home" {
		t.Fatalf("Expected /home, but got: " + e)
	}

	e, err := h.GetHistoryEntryForPath("", true)
	if err == nil {
		t.Fatal("Passing an empty path did not error")
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
