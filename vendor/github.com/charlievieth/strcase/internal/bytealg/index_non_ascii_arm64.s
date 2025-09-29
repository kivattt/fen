// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

// The following code is modified version of the standard library's
// internal/bytealg/indexbyte_armd64.s the Go LICENSE can be found in
// the go.LICENSE file.

#include "textflag.h"

TEXT ·IndexNonASCII(SB), NOSPLIT, $0-24
	MOVD s_base+0(FP), R0
	MOVD s_len+8(FP), R2
	MOVD $ret+16(FP), R8

	B indexByteBodyNonASCII<>(SB)

TEXT ·IndexByteNonASCII(SB), NOSPLIT, $0-32
	MOVD b_base+0(FP), R0
	MOVD b_len+8(FP), R2
	MOVD $ret+24(FP), R8

	B indexByteBodyNonASCII<>(SB)

// input:
//   R0: data
//   R2: data len
//   R8: address to put result
TEXT indexByteBodyNonASCII<>(SB), NOSPLIT, $0
	CBZ  R2, fail
	MOVD R0, R11

	// Magic constant 0x40100401 allows us to identify
	// which lane matches the requested byte.
	// 0x40100401 = ((1<<0) + (4<<8) + (16<<16) + (64<<24))
	// Different bytes have different bit masks (i.e: 1, 4, 16, 64)
	MOVD $0x40100401, R5

	// Bit mask for RuneSelf (0x80)
	VMOVQ $0x8080808080808080, $0x8080808080808080, V0

	// Work with aligned 32-byte chunks
	BIC  $0x1f, R0, R3
	VMOV R5, V5.S4
	ANDS $0x1f, R0, R9
	AND  $0x1f, R2, R10
	BEQ  loop

	// Input string is not 32-byte aligned. We calculate the
	// syndrome value for the aligned 32 bytes block containing
	// the first bytes and mask off the irrelevant part.
	VLD1.P (R3), [V1.B16, V2.B16]
	VAND   V0.B16, V1.B16, V1.B16 // Check for RuneSelf
	VAND   V0.B16, V2.B16, V2.B16 // Check for RuneSelf
	SUB    $0x20, R9, R4
	ADDS   R4, R2, R2
	VCMEQ  V0.B16, V1.B16, V3.B16
	VCMEQ  V0.B16, V2.B16, V4.B16
	VAND   V5.B16, V3.B16, V3.B16
	VAND   V5.B16, V4.B16, V4.B16
	VADDP  V4.B16, V3.B16, V6.B16 // 256->128
	VADDP  V6.B16, V6.B16, V6.B16 // 128->64
	VMOV   V6.D[0], R6

	// Clear the irrelevant lower bits
	LSL $1, R9, R4
	LSR R4, R6, R6
	LSL R4, R6, R6

	// The first block can also be the last
	BLS masklast

	// Have we found something already?
	CBNZ R6, tail

loop:
	VLD1.P (R3), [V1.B16, V2.B16]
	SUBS   $0x20, R2, R2
	VAND   V0.B16, V1.B16, V1.B16 // Check for RuneSelf
	VAND   V0.B16, V2.B16, V2.B16 // Check for RuneSelf
	VCMEQ  V0.B16, V1.B16, V3.B16
	VCMEQ  V0.B16, V2.B16, V4.B16

	// If we're out of data we finish regardless of the result
	BLS end

	// Use a fast check for the termination condition
	VORR  V4.B16, V3.B16, V6.B16
	VADDP V6.D2, V6.D2, V6.D2
	VMOV  V6.D[0], R6

	// We're not out of data, loop if we haven't found the character
	CBZ R6, loop

end:
	// Termination condition found, let's calculate the syndrome value
	VAND  V5.B16, V3.B16, V3.B16
	VAND  V5.B16, V4.B16, V4.B16
	VADDP V4.B16, V3.B16, V6.B16
	VADDP V6.B16, V6.B16, V6.B16
	VMOV  V6.D[0], R6

	// Only do the clear for the last possible block with less than 32 bytes
	// Condition flags come from SUBS in the loop
	BHS tail

masklast:
	// Clear the irrelevant upper bits
	ADD R9, R10, R4
	AND $0x1f, R4, R4
	SUB $0x20, R4, R4
	NEG R4<<1, R4
	LSL R4, R6, R6
	LSR R4, R6, R6

tail:
	// Check that we have found a character
	CBZ R6, fail

	// Count the trailing zeros using bit reversing
	RBIT R6, R6

	// Compensate the last post-increment
	SUB $0x20, R3, R3

	// And count the leading zeros
	CLZ R6, R6

	// R6 is twice the offset into the fragment
	ADD R6>>1, R3, R0

	// Compute the offset result
	SUB  R11, R0, R0
	MOVD R0, (R8)
	RET

fail:
	MOVD $-1, R0
	MOVD R0, (R8)
	RET
