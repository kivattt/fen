//go:build amd64 && !go1.22
// +build amd64,!go1.22

// Copyright 2023 Charlie Vieth. All rights reserved.
// Use of this source code is governed by the MIT license.

// The following code is modified version of the standard library's
// internal/bytealg/indexbyte_amd64.s the Go LICENSE can be found in
// the go.LICENSE file.

#include "go_asm.h"
#include "textflag.h"

TEXT ·IndexByte(SB), NOSPLIT, $0-40
	MOVQ b_base+0(FP), SI
	MOVQ b_len+8(FP), BX
	MOVB c+24(FP), AL
	LEAQ ret+32(FP), R8

	LEAL -65(AX), CX // Check if the byte is a ASCII letter
	CMPB CL, $25
	JLS  index_case  // Byte sought is a ASCII letter
	ADDL $-97, AX
	CMPB AL, $25
	JHI  index

index_case:
	MOVB c+24(FP), AL
	JMP  indexbytebodyCase<>(SB)

index:
	MOVB c+24(FP), AL
	JMP  indexbytebody<>(SB)

TEXT ·IndexByteString(SB), NOSPLIT, $0-32
	MOVQ s_base+0(FP), SI
	MOVQ s_len+8(FP), BX
	MOVB c+16(FP), AL
	LEAQ ret+24(FP), R8

	LEAL -65(AX), CX // Check if the byte is a ASCII letter
	CMPB CL, $25
	JLS  index_case  // Byte sought is a ASCII letter
	ADDL $-97, AX
	CMPB AL, $25
	JHI  index

index_case:
	MOVB c+16(FP), AL
	JMP  indexbytebodyCase<>(SB)

index:
	MOVB c+16(FP), AL
	JMP  indexbytebody<>(SB)

// indexbytebodyCase is a case insensitive version indexbytebody
// the byte being sought *must* be an ASCII letter.
//
// input:
//   SI: data
//   BX: data len
//   AL: byte sought
//   R8: address to put result
TEXT indexbytebodyCase<>(SB), NOSPLIT, $0
	// Shuffle X0 around so that each byte contains
	// the character we're looking for.
	ORL       $32, AX    // Convert byte to lowercase
	MOVD      AX, X0
	PUNPCKLBW X0, X0
	PUNPCKLBW X0, X0
	PSHUFL    $0, X0, X0

	// Add space (' ') mask to X2
	MOVQ      $32, CX
	MOVQ      CX, X2
	PUNPCKLBW X2, X2
	PUNPCKLBW X2, X2
	PSHUFL    $0, X2, X2

	CMPQ BX, $16
	JLT  small

	MOVQ SI, DI

	CMPQ BX, $32
	JA   avx2

sse:
	LEAQ -16(SI)(BX*1), AX // AX = address of last 16 bytes
	JMP  sseloopentry

sseloop:
	// Move the next 16-byte chunk of the data into X1.
	MOVOU (DI), X1

	// Logical OR to convert data to lowercase
	POR X2, X1

	// Compare bytes in X0 to X1.
	PCMPEQB X0, X1

	// Take the top bit of each byte in X1 and put the result in DX.
	PMOVMSKB X1, DX

	// Find first set bit, if any.
	BSFL DX, DX
	JNZ  ssesuccess

	// Advance to next block.
	ADDQ $16, DI

sseloopentry:
	CMPQ DI, AX
	JB   sseloop

	// Search the last 16-byte chunk. This chunk may overlap with the
	// chunks we've already searched, but that's ok.
	MOVQ     AX, DI
	MOVOU    (AX), X1
	POR      X2, X1     // Convert data to lowercase
	PCMPEQB  X0, X1
	PMOVMSKB X1, DX
	BSFL     DX, DX
	JNZ      ssesuccess

failure:
	MOVQ $-1, (R8)
	RET

// We've found a chunk containing the byte.
// The chunk was loaded from DI.
// The index of the matching byte in the chunk is DX.
// The start of the data is SI.
ssesuccess:
	SUBQ SI, DI   // Compute offset of chunk within data.
	ADDQ DX, DI   // Add offset of byte within chunk.
	MOVQ DI, (R8)
	RET

// handle for lengths < 16
small:
	TESTQ BX, BX
	JEQ   failure

	// Check if we'll load across a page boundary.
	LEAQ  16(SI), AX
	TESTW $0xff0, AX
	JEQ   endofpage

	MOVOU    (SI), X1 // Load data
	POR      X2, X1   // Convert data to lowercase
	PCMPEQB  X0, X1   // Compare target byte with each byte in data.
	PMOVMSKB X1, DX   // Move result bits to integer register.
	BSFL     DX, DX   // Find first set bit.
	JZ       failure  // No set bit, failure.
	CMPL     DX, BX
	JAE      failure  // Match is past end of data.
	MOVQ     DX, (R8)
	RET

endofpage:
	MOVOU    -16(SI)(BX*1), X1 // Load data into the high end of X1.
	POR      X2, X1            // Convert data to lowercase
	PCMPEQB  X0, X1            // Compare target byte with each byte in data.
	PMOVMSKB X1, DX            // Move result bits to integer register.
	MOVL     BX, CX
	SHLL     CX, DX
	SHRL     $16, DX           // Shift desired bits down to bottom of register.
	BSFL     DX, DX            // Find first set bit.
	JZ       failure           // No set bit, failure.
	MOVQ     DX, (R8)
	RET

avx2:
#ifndef hasAVX2
	CMPB golang·org∕x∕sys∕cpu·X86+const_offsetX86HasAVX2(SB), $1
	JNE  sse

#endif
	// Create a mask in Y4 that converts text to upper case.
	VPBROADCASTB X2, Y4 // space ' ' already stored in X2

	MOVD         AX, X0
	LEAQ         -32(SI)(BX*1), R11
	VPBROADCASTB X0, Y1

avx2_loop:
	VMOVDQU  (DI), Y2
	VPOR     Y4, Y2, Y2  // Convert data to lowercase
	VPCMPEQB Y1, Y2, Y3
	VPTEST   Y3, Y3
	JNZ      avx2success
	ADDQ     $32, DI
	CMPQ     DI, R11
	JLT      avx2_loop
	MOVQ     R11, DI
	VMOVDQU  (DI), Y2
	VPOR     Y4, Y2, Y2  // Convert data to lowercase
	VPCMPEQB Y1, Y2, Y3
	VPTEST   Y3, Y3
	JNZ      avx2success
	VZEROUPPER
	MOVQ     $-1, (R8)
	RET

avx2success:
	VPMOVMSKB Y3, DX
	BSFL      DX, DX
	SUBQ      SI, DI
	ADDQ      DI, DX
	MOVQ      DX, (R8)
	VZEROUPPER
	RET

// indexbytebody is the same as internal/bytealg/indexbyte_amd64.s (go1.20)
//
// input:
//   SI: data
//   BX: data len
//   AL: byte sought
//   R8: address to put result
TEXT indexbytebody<>(SB), NOSPLIT, $0
	// Shuffle X0 around so that each byte contains
	// the character we're looking for.
	MOVD      AX, X0
	PUNPCKLBW X0, X0
	PUNPCKLBW X0, X0
	PSHUFL    $0, X0, X0

	CMPQ BX, $16
	JLT  small

	MOVQ SI, DI

	CMPQ BX, $32
	JA   avx2

sse:
	LEAQ -16(SI)(BX*1), AX // AX = address of last 16 bytes
	JMP  sseloopentry

sseloop:
	// Move the next 16-byte chunk of the data into X1.
	MOVOU (DI), X1

	// Compare bytes in X0 to X1.
	PCMPEQB X0, X1

	// Take the top bit of each byte in X1 and put the result in DX.
	PMOVMSKB X1, DX

	// Find first set bit, if any.
	BSFL DX, DX
	JNZ  ssesuccess

	// Advance to next block.
	ADDQ $16, DI

sseloopentry:
	CMPQ DI, AX
	JB   sseloop

	// Search the last 16-byte chunk. This chunk may overlap with the
	// chunks we've already searched, but that's ok.
	MOVQ     AX, DI
	MOVOU    (AX), X1
	PCMPEQB  X0, X1
	PMOVMSKB X1, DX
	BSFL     DX, DX
	JNZ      ssesuccess

failure:
	MOVQ $-1, (R8)
	RET

// We've found a chunk containing the byte.
// The chunk was loaded from DI.
// The index of the matching byte in the chunk is DX.
// The start of the data is SI.
ssesuccess:
	SUBQ SI, DI   // Compute offset of chunk within data.
	ADDQ DX, DI   // Add offset of byte within chunk.
	MOVQ DI, (R8)
	RET

// handle for lengths < 16
small:
	TESTQ BX, BX
	JEQ   failure

	// Check if we'll load across a page boundary.
	LEAQ  16(SI), AX
	TESTW $0xff0, AX
	JEQ   endofpage

	MOVOU    (SI), X1 // Load data
	PCMPEQB  X0, X1   // Compare target byte with each byte in data.
	PMOVMSKB X1, DX   // Move result bits to integer register.
	BSFL     DX, DX   // Find first set bit.
	JZ       failure  // No set bit, failure.
	CMPL     DX, BX
	JAE      failure  // Match is past end of data.
	MOVQ     DX, (R8)
	RET

endofpage:
	MOVOU    -16(SI)(BX*1), X1 // Load data into the high end of X1.
	PCMPEQB  X0, X1            // Compare target byte with each byte in data.
	PMOVMSKB X1, DX            // Move result bits to integer register.
	MOVL     BX, CX
	SHLL     CX, DX
	SHRL     $16, DX           // Shift desired bits down to bottom of register.
	BSFL     DX, DX            // Find first set bit.
	JZ       failure           // No set bit, failure.
	MOVQ     DX, (R8)
	RET

avx2:
#ifndef hasAVX2
	CMPB golang·org∕x∕sys∕cpu·X86+const_offsetX86HasAVX2(SB), $1
	JNE  sse

#endif
	MOVD         AX, X0
	LEAQ         -32(SI)(BX*1), R11
	VPBROADCASTB X0, Y1

avx2_loop:
	VMOVDQU  (DI), Y2
	VPCMPEQB Y1, Y2, Y3
	VPTEST   Y3, Y3
	JNZ      avx2success
	ADDQ     $32, DI
	CMPQ     DI, R11
	JLT      avx2_loop
	MOVQ     R11, DI
	VMOVDQU  (DI), Y2
	VPCMPEQB Y1, Y2, Y3
	VPTEST   Y3, Y3
	JNZ      avx2success
	VZEROUPPER
	MOVQ     $-1, (R8)
	RET

avx2success:
	VPMOVMSKB Y3, DX
	BSFL      DX, DX
	SUBQ      SI, DI
	ADDQ      DI, DX
	MOVQ      DX, (R8)
	VZEROUPPER
	RET
