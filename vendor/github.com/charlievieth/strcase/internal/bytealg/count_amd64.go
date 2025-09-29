// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

//go:build amd64
// +build amd64

package bytealg

// Make golangci-lint think these functions are accessed since it
// cannot see accesses in assembly.
var _ = countGeneric
var _ = countGenericString

// Backup implementation to use by assembly when POPCNT is not available.
func countGeneric(b []byte, c byte) int {
	if 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' {
		n := 0
		c |= ' '
		for _, cc := range b {
			if cc|' ' == c {
				n++
			}
		}
		return n
	}
	n := 0
	for _, x := range b {
		if x == c {
			n++
		}
	}
	return n
}

// Backup implementation to use by assembly when POPCNT is not available.
func countGenericString(s string, c byte) int {
	if 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' {
		n := 0
		c |= ' '
		for i := 0; i < len(s); i++ {
			if s[i]|' ' == c {
				n++
			}
		}
		return n
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			n++
		}
	}
	return n
}
