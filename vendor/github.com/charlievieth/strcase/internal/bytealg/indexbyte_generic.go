// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

//go:build !s390x && !wasm && !ppc64 && !amd64 && !arm64
// +build !s390x,!wasm,!ppc64,!amd64,!arm64

// Simple implementations for arch's where the standard library does not
// appear to use SIMD for IndexByte.
//
// NOTE(cev): See the comment in indexbyte_simd.go for how the list of GOARCH
// build tags was created.

package bytealg

import (
	"bytes"
	"strings"
)

func isAlpha(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

func IndexByte(s []byte, c byte) int {
	if !isAlpha(c) {
		return bytes.IndexByte(s, c)
	}
	c |= ' '
	for i, cc := range s {
		if cc|' ' == c {
			return i
		}
	}
	return -1
}

func IndexByteString(s string, c byte) int {
	if !isAlpha(c) {
		return strings.IndexByte(s, c)
	}
	c |= ' '
	for i := 0; i < len(s); i++ {
		if s[i]|' ' == c {
			return i
		}
	}
	return -1
}
