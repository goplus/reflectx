//go:build (ppc64 || ppc64le) && !go1.27
// +build ppc64 ppc64le
// +build !go1.27

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"
#include "funcdata.h"
#include "asm_ppc64x.h"

// The frames of each of the two functions below contain two locals, at offsets
// that are known to the runtime.
//
// The first local is a bool called retValid with a whole pointer-word reserved
// for it on the stack. The purpose of this word is so that the runtime knows
// whether the stack-allocated return space contains valid values for stack
// scanning.
//
// The second local is an abi.RegArgs value whose offset is also known to the
// runtime, so that a stack map for it can be constructed, since it contains
// pointers visible to the GC.

#define LOCAL_RETVALID 32+FIXED_FRAME
#define LOCAL_REGARGS 40+FIXED_FRAME

// The frame size of the functions below is
// 32 (args of callReflect) + 8 (bool + padding) + 296 (abi.RegArgs) = 336.

// makeFuncStub is the code half of the function returned by MakeFunc.
// See the comment on the declaration of makeFuncStub in makefunc.go
// for more details.
// No arg size here, runtime pulls arg map out of the func value.
#define MAKE_FUNC_FN(NAME,INDEX)		\
TEXT NAME(SB),(NOSPLIT|WRAPPER),$336		\
	NO_LOCAL_POINTERS		\
	ADD	$LOCAL_REGARGS, R1, R20		\
	CALL	runtime·spillArgs(SB)		\
	MOVD	FIXED_FRAME+32(R1), R11			\
	MOVD	R11, FIXED_FRAME+0(R1)		\
	MOVD	$argframe+0(FP), R3		\
	MOVD	R3, FIXED_FRAME+8(R1)		\
	ADD	$LOCAL_RETVALID, R1, R3		\
	MOVB	R0, (R3)		\
	MOVD	R3, FIXED_FRAME+16(R1)			\
	ADD     $LOCAL_REGARGS, R1, R3		\
	MOVD	R3, FIXED_FRAME+24(R1)		\
	MOVD	$INDEX, R3		\
	MOVD	R3, FIXED_FRAME+32(R1)		\
	BL	·i_x(SB)		\
	ADD	$LOCAL_REGARGS, R1, R20		\
	CALL	runtime·unspillArgs(SB)		\
	RET
