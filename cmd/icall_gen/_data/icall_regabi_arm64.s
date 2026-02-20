//go:build (go1.18 && goexperiment.regabireflect) || (go1.19 && !go1.21)
// +build go1.18,goexperiment.regabireflect go1.19,!go1.21

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"
#include "funcdata.h"

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
#define LOCAL_RETVALID 40
#define LOCAL_REGARGS 48

// The frame size of the functions below is
// 32 (args of callReflect) + 8 (bool + padding) + 392 (abi.RegArgs) = 432.

// makeFuncStub is the code half of the function returned by MakeFunc.
// See the comment on the declaration of makeFuncStub in makefunc.go
// for more details.
// No arg size here, runtime pulls arg map out of the func value.
#define MAKE_FUNC_FN(NAME,INDEX)		\
TEXT NAME(SB),(NOSPLIT|WRAPPER),$432		\
	NO_LOCAL_POINTERS		\
	ADD	$LOCAL_REGARGS, RSP, R20		\
	CALL	runtime·spillArgs(SB)		\
	MOVD	32(RSP), R26		\
	MOVD	R26, 8(RSP)		\
	MOVD	$argframe+0(FP), R3		\
	MOVD	R3, 16(RSP)		\
	MOVB	$0, LOCAL_RETVALID(RSP)		\
	ADD	$LOCAL_RETVALID, RSP, R3		\
	MOVD	R3, 24(RSP)		\
	ADD	$LOCAL_REGARGS, RSP, R3		\
	MOVD	R3, 32(RSP)		\
	MOVD	$INDEX, R3		\
	MOVD	R3, 40(RSP)		\
	CALL	·i_x(SB)		\
	ADD	$LOCAL_REGARGS, RSP, R20		\
	CALL	runtime·unspillArgs(SB)		\
	RET
