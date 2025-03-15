package main

import (
	"runtime"
	"testing"
)

func TestGetHistoryEntryForPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		var h History
		_, err := h.GetHistoryEntryForPath("C:\\", true)
		if err == nil {
			t.Fatal("Expected an error on empty history, but got nil")
		}

		h.AddToHistory("C:\\Users\\user\\test\\something")
		h.AddToHistory("C:\\Users\\user\\test.txt")

		e, _ := h.GetHistoryEntryForPath("C:\\Users\\user\\test", true)
		if e != "C:\\Users\\user\\test\\something" {
			t.Fatal("Expected C:\\Users\\user\\test\\something, but got: " + e)
		}

		e, _ = h.GetHistoryEntryForPath("C:\\", true)
		if e != "C:\\Users" {
			t.Fatal("Expected C:\\Users, but got: " + e)
		}

		_, err = h.GetHistoryEntryForPath("D:\\", true)
		if err == nil {
			t.Fatal("Expected an error on empty history, but got nil")
		}
	} else {
		var h History
		h.AddToHistory("/home/user/folder1/folder2")

		e, _ := h.GetHistoryEntryForPath("/home", true)
		if e != "/home/user" {
			t.Fatal("Expected /home/user, but got: " + e)
		}

		h.AddToHistory("/home/user/test/something")
		h.AddToHistory("/home/user/test.txt")

		e, _ = h.GetHistoryEntryForPath("/home/user/test", true)
		if e != "/home/user/test/something" {
			t.Fatal("Expected /home/user/test/something, but got: " + e)
		}

		e, _ = h.GetHistoryEntryForPath("/", true)
		if e != "/home" {
			t.Fatal("Expected /home, but got: " + e)
		}

		_, err := h.GetHistoryEntryForPath("", true)
		if err == nil {
			t.Fatal("Passing an empty path did not error")
		}

		// Multiple directories at root
		h.ClearHistory()
		h.AddToHistory("/home/user/folder/file")
		h.AddToHistory("/home/user/folder/file2")
		h.AddToHistory("/mnt/something/else/here")
		h.AddToHistory("/mnt/something/else/here2")

		e, _ = h.GetHistoryFullPath("/home", true)
		if e != "/home/user/folder/file2" {
			t.Fatal("Expected /home/user/folder/file2, but got: " + e)
		}

		e, _ = h.GetHistoryFullPath("/mnt", true)
		if e != "/mnt/something/else/here2" {
			t.Fatal("Expected /mnt/something/else/here2, but got: " + e)
		}

		// Hidden files
		h.ClearHistory()
		h.AddToHistory("/home/user/nothidden")
		h.AddToHistory("/home/user/.hidden")
		e, _ = h.GetHistoryEntryForPath("/home/user", false)
		if e != "/home/user/nothidden" {
			t.Fatal("Expected /home/user/nothidden, but got: " + e)
		}

		e, _ = h.GetHistoryEntryForPath("/home/user", true)
		if e != "/home/user/.hidden" {
			t.Fatal("Expected /home/user/.hidden, but got: " + e)
		}
	}
}
