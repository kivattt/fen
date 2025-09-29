// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

package bytealg

//go:noescape
func IndexByte(b []byte, c byte) int

//go:noescape
func IndexByteString(s string, c byte) int

//go:noescape
func IndexNonASCII(s string) int

//go:noescape
func IndexByteNonASCII(b []byte) int

//go:noescape
func Count(b []byte, c byte) int

//go:noescape
func CountString(s string, c byte) int
