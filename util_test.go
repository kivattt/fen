package main

import "testing"

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
