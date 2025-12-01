// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

//go:build s390x || wasm || ppc64
// +build s390x wasm ppc64

// The below functions assume their corresponding functions in the standard
// library can search multiple bytes simultaneously (SIMD or whatever).
// Otherwise, a simple for loop should be used.
//
// NOTE(cev): The arch build tags included here were picked by me browsing the
// assembly implementations of indexbye_{GOARCH}.s and if it looked fancy and
// complicated I assume it used SIMD so included it here - so could be wrong.
//
// This list should be revisited when new versions of Go are released, or by
// anyone who either has access to the hardware or understands the assembly.

package bytealg

import (
	"bytes"
	"strings"
)

func isAlpha(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

func IndexByte(s []byte, c byte) int {
	if len(s) == 0 {
		return -1
	}
	n := bytes.IndexByte(s, c)
	if n == 0 || !isAlpha(c) {
		return n
	}

	c ^= ' ' // swap case
	if s[0] == c {
		return 0
	}

	// TODO: calculate the optimal cutoff
	if n > 0 && len(s) >= 16 {
		s = s[:n] // limit search space
	}

	if o := bytes.IndexByte(s, c); n == -1 || (o != -1 && o < n) {
		n = o
	}
	return n
}

func IndexByteString(s string, c byte) int {
	if len(s) == 0 {
		return -1
	}
	n := strings.IndexByte(s, c)
	if n == 0 || !isAlpha(c) {
		return n
	}

	c ^= ' ' // swap case
	if s[0] == c {
		return 0
	}

	// TODO: calculate the optimal cutoff
	if n > 0 && len(s) >= 16 {
		s = s[:n] // limit search space
	}

	if o := strings.IndexByte(s, c); n == -1 || (o != -1 && o < n) {
		n = o
	}
	return n
}
