// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

//go:build !s390x && !ppc64 && !amd64 && !arm64
// +build !s390x,!ppc64,!amd64,!arm64

// NOTE(cev): See the comment in indexbyte_simd.go for how the list of GOARCH
// build tags was created (note: wasm is included here because it lacks an
// assembly implementation of Count).

package bytealg

func Count(b []byte, c byte) int {
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

func CountString(s string, c byte) int {
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
