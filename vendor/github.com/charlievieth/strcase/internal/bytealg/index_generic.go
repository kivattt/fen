// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

//go:build !amd64 && !arm64 && !s390x && !ppc64le && !ppc64

package bytealg

import _ "unsafe"

// NativeIndex is true if we have a fast native (assembly) implementation
// of Index and IndexString. Otherwise, bytes.Index and strings.Index are used.
const NativeIndex = false

//go:linkname MaxBruteForce internal/bytealg.MaxBruteForce
var MaxBruteForce int

// Cutover reports the number of failures of IndexByte we should tolerate
// before switching over to Index.
// n is the number of bytes processed so far.
// See the bytes.Index implementation for details.
func Cutover(n int) int {
	panic("unimplemented")
}

func Index(s, sep []byte) int {
	panic("unimplemented")
}

func IndexString(s, substr string) int {
	panic("unimplemented")
}
