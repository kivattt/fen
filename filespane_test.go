package main

import (
	"reflect"
	"testing"

	"github.com/fsnotify/fsnotify"
)

type TestCase struct {
	oldEvents []fsnotify.Event
	newEvent  fsnotify.Event
	expected  []fsnotify.Event
}

func TestAddEventToBatch(t *testing.T) {
	testCases := []TestCase{
		{
			oldEvents: []fsnotify.Event{
				{Op: fsnotify.Create, Name: "file.txt"},
				{Op: fsnotify.Write, Name: "file.txt"},
			},
			newEvent: fsnotify.Event{Op: fsnotify.Remove, Name: "file.txt"},
			expected: []fsnotify.Event{{Op: fsnotify.Remove, Name: "file.txt"}},
		},
		{
			oldEvents: []fsnotify.Event{
				{Op: fsnotify.Write, Name: "file.txt"},
				{Op: fsnotify.Write, Name: "file.txt"},
				{Op: fsnotify.Write, Name: "file.txt"},
			},
			newEvent: fsnotify.Event{Op: fsnotify.Write, Name: "file.txt"},
			expected: []fsnotify.Event{{Op: fsnotify.Write, Name: "file.txt"}},
		},
		{
			oldEvents: []fsnotify.Event{
				{Op: fsnotify.Create, Name: "file.txt"},
				{Op: fsnotify.Create, Name: "file.txt"},
				{Op: fsnotify.Create, Name: "file.txt"},
			},
			newEvent: fsnotify.Event{Op: fsnotify.Create, Name: "file.txt"},
			expected: []fsnotify.Event{{Op: fsnotify.Create, Name: "file.txt"}},
		},
	}

	for _, v := range testCases {
		got := AddEventToBatch(v.oldEvents, v.newEvent)
		if !reflect.DeepEqual(got, v.expected) {
			t.Fatalf("Expected %#v, but got %#v", v.expected, got)
		}
	}
}
