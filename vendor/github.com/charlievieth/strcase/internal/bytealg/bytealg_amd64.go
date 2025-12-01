// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

package bytealg

import (
	"unsafe"

	"golang.org/x/sys/cpu"
)

// Offsets into internal/cpu records for use in assembly.
const (
	offsetX86HasAVX2   = unsafe.Offsetof(cpu.X86.HasAVX2)
	offsetX86HasPOPCNT = unsafe.Offsetof(cpu.X86.HasPOPCNT)
)

// Make golangci-lint think these constants are accessed since it
// cannot see accesses in assembly.
const _ = offsetX86HasAVX2
const _ = offsetX86HasPOPCNT

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
