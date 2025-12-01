// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

//go:build amd64 || arm64 || s390x || ppc64le || ppc64

package bytealg

import (
	"runtime"
	_ "unsafe"
)

// NativeIndex is true if we have a fast native (assembly) implementation
// of Index and IndexString. Otherwise bytes.Index and strings.Index are used.
const NativeIndex = true

// TODO: remove if not used.
//
//go:linkname MaxBruteForce internal/bytealg.MaxBruteForce
var MaxBruteForce int

// Cutover reports the number of failures of IndexByte we should tolerate
// before switching over to Index.
// n is the number of bytes processed so far.
// See the bytes.Index implementation for details.
//
// Implemented here instead of linked to internal/bytealg.Cutover since this
// is a hot function and we want it to be inlined.
//
// NOTE: This needs to be kept up-to-date with internal/bytealg.Cutover.
func Cutover(n int) int {
	if runtime.GOARCH == "arm64" {
		// 1 error per 16 characters, plus a few slop to start.
		return 4 + n>>4
	}
	// 1 error per 8 characters, plus a few slop to start.
	return (n + 16) / 8
}

//go:noescape
//go:linkname Index internal/bytealg.Index
func Index(s, substr []byte) int

//go:noescape
//go:linkname IndexString internal/bytealg.IndexString
func IndexString(s, substr string) int
