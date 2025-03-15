package main

import (
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestBytesToHumanReadableUnitString(t *testing.T) {
	expectedResults := map[uint64]string{
		0:                   "0 B",
		100:                 "100 B",
		999:                 "999 B",
		1000:                "1 kB",
		999999999:           "999.999 MB",
		1000000000:          "1 GB",
		1000000000000000000: "1 EB",
	}

	for byteCount, expected := range expectedResults {
		got := BytesToHumanReadableUnitString(byteCount, 3)
		if got != expected {
			t.Fatalf("Expected " + expected + ", but got " + got)
		}
	}
}

func TestTrimLastDecimals(t *testing.T) {
	expectedResults := map[string]string{
		"":           "",
		".":          ".",
		"0":          "0",
		"0.0":        "0.0",
		"0.00":       "0.00",
		"0.000":      "0.000",
		"0.0000":     "0.000",
		"12.3456789": "12.345",
	}

	for input, expected := range expectedResults {
		got := trimLastDecimals(input, 3)
		if got != expected {
			t.Fatalf("Expected " + expected + ", but got " + got)
		}
	}
}

func TestStyleToStyleTagString(t *testing.T) {
	// https://pkg.go.dev/github.com/gdamore/tcell/v2#AttrMask
	expectedResults := map[tcell.Style]string{
		tcell.StyleDefault: "[default:default]",
		tcell.StyleDefault.Dim(true).Foreground(tcell.ColorYellow):  "[yellow:default:d]",
		tcell.StyleDefault.Foreground(tcell.ColorYellow).Dim(true):  "[yellow:default:d]",
		tcell.StyleDefault.Background(tcell.ColorBlue):              "[default:blue]",
		tcell.StyleDefault.Foreground(tcell.NewRGBColor(0, 255, 0)): "[#00FF00:default]",
		tcell.StyleDefault.Attributes(0):                            "[default:default]",
		tcell.StyleDefault.Attributes(0b01111111):                   "[default:default:blrudis]",
	}

	for input, expected := range expectedResults {
		got := StyleToStyleTagString(input)
		if got != expected {
			t.Fatalf("Expected " + expected + ", but got " + got)
		}
	}
}

func TestRuneToPrintableCode(t *testing.T) {
	expectedResults := map[rune]string{
		'a':  "\\u61",
		'z':  "\\u7a",
		' ':  "\\u20",
		'🥺':  "\\u1f97a",
		'\a': "\\a",
		'\v': "\\v",
		'\t': "\\t",
	}

	for input, expected := range expectedResults {
		got := RuneToPrintableCode(input)
		if got != expected {
			t.Fatal("Expected " + expected + ", but got " + got)
		}
	}
}

func TestFilenameInvisibleCharactersAsCodeHighlighted(t *testing.T) {
	expectedResults := map[[2]string]string{
		{"file.txt", ""}:            "file.txt",
		{"file.txt", "[blue::b]"}:   "file.txt",
		{"file\n.txt", "[blue::b]"}: "file[:darkred]\\n[-:-:-:-][blue::b].txt",
		{" a a ", ""}:               "[:darkred]\\u20[-:-:-:-]a a[:darkred]\\u20[-:-:-:-]",
		{"●", ""}:                   "●",
		{"\u2800", ""}:              "[:darkred]\\u2800[-:-:-:-]",
		{"\U000e0100", ""}:          "[:darkred]\\ue0100[-:-:-:-]",
		{"\U000e01ef", ""}:          "[:darkred]\\ue01ef[-:-:-:-]",
	}

	for input, expected := range expectedResults {
		got := FilenameInvisibleCharactersAsCodeHighlighted(input[0], input[1])
		if got != expected {
			t.Fatalf("Expected " + expected + ", but got " + got)
		}
	}
}

func TestMapStringBoolKeys(t *testing.T) {
	theMap := map[string]bool{
		"1":     true,
		"2":     true,
		"3":     true,
		"hello": true,
		"":      true,
	}

	expectedValues := []string{
		"1",
		"2",
		"3",
		"hello",
		"",
	}

	keys := MapStringBoolKeys(theMap)

	if len(keys) != len(expectedValues) {
		t.Fatal("Expected a length of " + strconv.Itoa(len(expectedValues)) + ", but got " + strconv.Itoa(len(keys)))
	}

	for _, expectedValue := range expectedValues {
		if !slices.Contains(keys, expectedValue) {
			t.Fatal("Result did not contain " + expectedValue)
		}
	}
}

func TestStringSliceHasDuplicate(t *testing.T) {
	s := []string{"hello", "world", "", "hi"}
	_, err := StringSliceHasDuplicate(s)
	if err == nil {
		t.Fatal("Expected an error, but got nil")
	}

	s = []string{}
	_, err = StringSliceHasDuplicate(s)
	if err == nil {
		t.Fatal("Expected an error, but got nil")
	}

	s = []string{"", ""}
	found, err := StringSliceHasDuplicate(s)
	if err != nil {
		t.Fatal("Expected no error, but got: " + err.Error())
	}
	if found != "" {
		t.Fatal("Expected \"\", but got: " + found)
	}
}

func TestRandomStringPathSafe(t *testing.T) {
	r := RandomStringPathSafe(0)
	if r != "" {
		t.Fatal("Expected \"\", but got: " + r)
	}

	r = RandomStringPathSafe(1)
	if len(r) != 1 {
		t.Fatal("Expected len(r) == 1, but got: " + strconv.Itoa(len(r)))
	}

	r = RandomStringPathSafe(-1)
	if r != "" {
		t.Fatal("Expected \"\" (for -1 length), but got: " + r)
	}
}

func TestSplitPathTestable(t *testing.T) {
	type TestCase struct {
		pathInput string
		expected  []string
	}

	windowsTests := []TestCase{
		{
			pathInput: "C:\\Users\\user\\Desktop\\file.txt",
			expected:  []string{"C:\\", "C:\\Users", "C:\\Users\\user", "C:\\Users\\user\\Desktop", "C:\\Users\\user\\Desktop\\file.txt"},
		},
		{
			pathInput: "C:\\",
			expected:  []string{"C:\\"},
		},
		{
			pathInput: "", // Should be caught by filepath.IsAbs() anyway in SplitPath()
			expected:  []string{},
		},
		{
			pathInput: "C:\\Users",
			expected:  []string{"C:\\", "C:\\Users"},
		},
		{
			pathInput: "D:\\amogus\\video.mp4æ",
			expected:  []string{"D:\\", "D:\\amogus", "D:\\amogus\\video.mp4æ"},
		},
	}
	for _, test := range windowsTests {
		got := splitPathTestable(test.pathInput, '\\')
		if !reflect.DeepEqual(got, test.expected) {
			t.Fatal("Expected", test.expected, ", but got:", got)
		}
	}

	othersTests := []TestCase{
		{
			pathInput: "/home/user/file.txt",
			expected:  []string{"/", "/home", "/home/user", "/home/user/file.txt"},
		},
		{
			pathInput: "/",
			expected:  []string{"/"},
		},
		{
			pathInput: "", // Should be caught by filepath.IsAbs() anyway in SplitPath()
			expected:  []string{},
		},
		{
			pathInput: "/home",
			expected:  []string{"/", "/home"},
		},
		{
			pathInput: "/amogus/video.mp4æ",
			expected:  []string{"/", "/amogus", "/amogus/video.mp4æ"},
		},
	}
	for _, test := range othersTests {
		got := splitPathTestable(test.pathInput, '/')
		if !reflect.DeepEqual(got, test.expected) {
			t.Fatal("Expected", test.expected, ", but got:", got)
		}
	}
}
