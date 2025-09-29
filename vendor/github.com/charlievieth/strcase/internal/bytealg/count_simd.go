// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

//go:build s390x || ppc64
// +build s390x ppc64

// NOTE(cev): See the comment in indexbyte_simd.go for how the list of GOARCH
// build tags was created (note: wasm is excluded here because it lacks an
// assembly implementation of Count).

package bytealg

import (
	"bytes"
	"strings"
)

func Count(b []byte, c byte) int {
	s := []byte{c}
	if !isAlpha(c) {
		return bytes.Count(b, s)
	}
	n := bytes.Count(b, s)
	s[0] ^= ' ' // swap case
	return n + bytes.Count(b, s)
}

func CountString(s string, c byte) int {
	if !isAlpha(c) {
		return strings.Count(s, string(c))
	}
	return strings.Count(s, string(c)) +
		strings.Count(s, string(c^' ')) // swap case
}
