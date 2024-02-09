package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func writeRegAbi(filename string, pkgName string, size int) error {
	dir, f := filepath.Split(filename)
	if dir != "" {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("make dir %v error: %v", dir, err)
		}
	}
	gofile := filepath.Join(dir, strings.Replace(f, ".go", "_regabi.go", 1))
	var buf bytes.Buffer
	r := strings.NewReplacer("$pkgname", pkgName, "$max_size", strconv.Itoa(size))
	buf.WriteString(r.Replace(icall_regabi))

	var ar []string
	for i := 0; i < size; i++ {
		buf.WriteString(fmt.Sprintf("func f%v()\n", i))
		ar = append(ar, fmt.Sprintf("f%v", i))
	}
	buf.WriteString(fmt.Sprintf(`
var (
	icall_fn = []func(){%v}
)
`, strings.Join(ar, ",")))

	err := ioutil.WriteFile(gofile, buf.Bytes(), 0644)
	if err != nil {
		return err
	}

	asm_amd64_file := filepath.Join(dir, strings.Replace(f, ".go", "_regabi_amd64.s", 1))
	asm_arm64_file := filepath.Join(dir, strings.Replace(f, ".go", "_regabi_arm64.s", 1))
	asm_ppc64x_file := filepath.Join(dir, strings.Replace(f, ".go", "_regabi_ppc64x.s", 1))
	asm_riscv64_file := filepath.Join(dir, strings.Replace(f, ".go", "_regabi_riscv64.s", 1))
	asm_go121_amd64_file := filepath.Join(dir, strings.Replace(f, ".go", "_regabi_go121_amd64.s", 1))
	asm_go121_arm64_file := filepath.Join(dir, strings.Replace(f, ".go", "_regabi_go121_arm64.s", 1))
	fnWrite := func(filename string, tmpl string, size int) error {
		var buf bytes.Buffer
		buf.WriteString(tmpl)
		for i := 0; i < size; i++ {
			buf.WriteString(fmt.Sprintf("MAKE_FUNC_FN(·f%v,%v)\n", i, i))
		}
		return ioutil.WriteFile(filename, buf.Bytes(), 0644)
	}
	err = fnWrite(asm_amd64_file, regabi_amd64, size)
	if err != nil {
		return err
	}
	err = fnWrite(asm_arm64_file, regabi_arm64, size)
	if err != nil {
		return err
	}
	err = fnWrite(asm_go121_amd64_file, regabi_go121_amd64, size)
	if err != nil {
		return err
	}
	err = fnWrite(asm_go121_arm64_file, regabi_go121_arm64, size)
	if err != nil {
		return err
	}
	err = fnWrite(asm_ppc64x_file, regabi_ppc64x, size)
	if err != nil {
		return err
	}
	err = fnWrite(asm_riscv64_file, regabi_riscv64, size)
	if err != nil {
		return err
	}
	return nil
}

var icall_regabi = `//go:build ((go1.17 && goexperiment.regabireflect) || (go1.19 && goexperiment.regabiargs) || (go1.18 && amd64) || (go1.19 && arm64) || (go1.19 && ppc64) || (go1.19 && ppc64le) || (go1.20 && riscv64)) && (!js || (js && wasm))
// +build go1.17,goexperiment.regabireflect go1.19,goexperiment.regabiargs go1.18,amd64 go1.19,arm64 go1.19,ppc64 go1.19,ppc64le go1.20,riscv64
// +build !js js,wasm

package $pkgname

import (
	"reflect"
	"unsafe"

	"github.com/goplus/reflectx/abi"
)

const capacity = $max_size

type methodUsed struct {
	fun reflect.Value
	ptr unsafe.Pointer
}

type provider struct {
	used map[int]*methodUsed
}

//go:linkname callReflect reflect.callReflect
func callReflect(ctxt unsafe.Pointer, frame unsafe.Pointer, retValid *bool, r unsafe.Pointer)

//go:linkname moveMakeFuncArgPtrs reflect.moveMakeFuncArgPtrs
func moveMakeFuncArgPtrs(ctx unsafe.Pointer, r unsafe.Pointer)

func i_x(c unsafe.Pointer, frame unsafe.Pointer, retValid *bool, r unsafe.Pointer, index int) {
	ptr := mp.used[index].ptr
	moveMakeFuncArgPtrs(ptr, r)
	callReflect(ptr, frame, retValid, r)
}

func spillArgs()
func unspillArgs()

func (p *provider) Insert(info *abi.MethodInfo) (unsafe.Pointer, int) {
	var index = -1
	for i := 0; i < capacity; i++ {
		if _, ok := p.used[i]; !ok {
			index = i
			break
		}
	}
	if index == -1 {
		return nil, -1
	}
	var fn reflect.Value
	if (!info.Pointer && !info.OnePtr) || info.Indirect {
		ftyp := info.Func.Type()
		numIn := ftyp.NumIn()
		numOut := ftyp.NumOut()
		in := make([]reflect.Type, numIn, numIn)
		out := make([]reflect.Type, numOut, numOut)
		in[0] = reflect.PtrTo(info.Type)
		for i := 1; i < numIn; i++ {
			in[i] = ftyp.In(i)
		}
		for i := 0; i < numOut; i++ {
			out[i] = ftyp.Out(i)
		}
		ftyp = reflect.FuncOf(in, out, info.Variadic)
		if info.Variadic {
			fn = reflect.MakeFunc(ftyp, func(args []reflect.Value) []reflect.Value {
				args[0] = args[0].Elem()
				return info.Func.CallSlice(args)
			})
		} else {
			fn = reflect.MakeFunc(ftyp, func(args []reflect.Value) []reflect.Value {
				args[0] = args[0].Elem()
				return info.Func.Call(args)
			})
		}
	} else {
		fn = info.Func
	}
	p.used[index] = &methodUsed{
		fun: fn,
		ptr: (*struct{ typ, ptr unsafe.Pointer })(unsafe.Pointer(&fn)).ptr,
	}
	icall := icall_fn[index]
	return unsafe.Pointer(reflect.ValueOf(icall).Pointer()), index
}

func (p *provider) Remove(indexs []int) {
	for _, n := range indexs {
		delete(p.used, n)
	}
}

func (p *provider) Available() int {
	return capacity - len(p.used)
}

func (p *provider) Used() int {
	return len(p.used)
}

func (p *provider) Cap() int {
	return capacity
}

func (p *provider) Clear() {
	p.used = make(map[int]*methodUsed)
}

var (
	mp = &provider{
		used: make(map[int]*methodUsed),
	}
)

func init() {
	abi.AddMethodProvider(mp)
}

`

var regabi_go121_amd64 = `//go:build go1.21
// +build go1.21

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"
#include "funcdata.h"
#include "go_asm.h"

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
#define LOCAL_RETVALID 32
#define LOCAL_REGARGS 40

// makeFuncStub is the code half of the function returned by MakeFunc.
// See the comment on the declaration of makeFuncStub in makefunc.go
// for more details.
// No arg size here; runtime pulls arg map out of the func value.
// This frame contains two locals. See the comment above LOCAL_RETVALID.
// amd64 argframe+8(FP) offset to func from method
#define MAKE_FUNC_FN(NAME,INDEX)		\
TEXT NAME(SB),(NOSPLIT|WRAPPER),$312		\
	NO_LOCAL_POINTERS		\
	LEAQ	LOCAL_REGARGS(SP), R12		\
	CALL	runtime·spillArgs(SB)		\
	MOVQ	24(SP), DX		\
	MOVQ	DX, 0(SP)		\
	LEAQ	argframe+16(FP), CX		\
	MOVQ	CX, 8(SP)		\
	MOVB	$0, LOCAL_RETVALID(SP)		\
	LEAQ	LOCAL_RETVALID(SP), AX		\
	MOVQ	AX, 16(SP)		\
	LEAQ	LOCAL_REGARGS(SP), AX		\
	MOVQ	AX, 24(SP)		\
	MOVQ	$INDEX, AX		\
	MOVQ	AX, 32(SP)		\
	CALL	·i_x(SB)		\
	LEAQ	LOCAL_REGARGS(SP), R12		\
	CALL	runtime·unspillArgs(SB)		\
	RET

`

var regabi_go121_arm64 = `//go:build go1.21
// +build go1.21

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
	MOVD	R26, 16(RSP)		\
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

`

var regabi_amd64 = `//go:build (go1.17 && goexperiment.regabireflect) || (go1.18 && !go1.21)
// +build go1.17,goexperiment.regabireflect go1.18,!go1.21

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"
#include "funcdata.h"
#include "go_asm.h"

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
#define LOCAL_RETVALID 32
#define LOCAL_REGARGS 40

TEXT ·spillArgs(SB),NOSPLIT,$0-0
	MOVQ AX, 0(R12)
	MOVQ BX, 8(R12)
	MOVQ CX, 16(R12)
	MOVQ DI, 24(R12)
	MOVQ SI, 32(R12)
	MOVQ R8, 40(R12)
	MOVQ R9, 48(R12)
	MOVQ R10, 56(R12)
	MOVQ R11, 64(R12)
	MOVQ X0, 72(R12)
	MOVQ X1, 80(R12)
	MOVQ X2, 88(R12)
	MOVQ X3, 96(R12)
	MOVQ X4, 104(R12)
	MOVQ X5, 112(R12)
	MOVQ X6, 120(R12)
	MOVQ X7, 128(R12)
	MOVQ X8, 136(R12)
	MOVQ X9, 144(R12)
	MOVQ X10, 152(R12)
	MOVQ X11, 160(R12)
	MOVQ X12, 168(R12)
	MOVQ X13, 176(R12)
	MOVQ X14, 184(R12)
	RET

// unspillArgs loads args into registers from a *internal/abi.RegArgs in R12.
TEXT ·unspillArgs(SB),NOSPLIT,$0-0
	MOVQ 0(R12), AX
	MOVQ 8(R12), BX
	MOVQ 16(R12), CX
	MOVQ 24(R12), DI
	MOVQ 32(R12), SI
	MOVQ 40(R12), R8
	MOVQ 48(R12), R9
	MOVQ 56(R12), R10
	MOVQ 64(R12), R11
	MOVQ 72(R12), X0
	MOVQ 80(R12), X1
	MOVQ 88(R12), X2
	MOVQ 96(R12), X3
	MOVQ 104(R12), X4
	MOVQ 112(R12), X5
	MOVQ 120(R12), X6
	MOVQ 128(R12), X7
	MOVQ 136(R12), X8
	MOVQ 144(R12), X9
	MOVQ 152(R12), X10
	MOVQ 160(R12), X11
	MOVQ 168(R12), X12
	MOVQ 176(R12), X13
	MOVQ 184(R12), X14
	RET

// makeFuncStub is the code half of the function returned by MakeFunc.
// See the comment on the declaration of makeFuncStub in makefunc.go
// for more details.
// No arg size here; runtime pulls arg map out of the func value.
// This frame contains two locals. See the comment above LOCAL_RETVALID.
#define MAKE_FUNC_FN(NAME,INDEX)		\
TEXT NAME(SB),(NOSPLIT|WRAPPER),$312		\
	NO_LOCAL_POINTERS		\
	LEAQ	LOCAL_REGARGS(SP), R12		\
	CALL	·spillArgs(SB)		\
	MOVQ	24(SP), DX		\
	MOVQ	DX, 0(SP)		\
	LEAQ	argframe+8(FP), CX		\
	MOVQ	CX, 8(SP)		\
	MOVB	$0, LOCAL_RETVALID(SP)		\
	LEAQ	LOCAL_RETVALID(SP), AX		\
	MOVQ	AX, 16(SP)		\
	LEAQ	LOCAL_REGARGS(SP), AX		\
	MOVQ	AX, 24(SP)		\
	MOVQ	$INDEX, AX		\
	MOVQ	AX, 32(SP)		\
	CALL	·i_x(SB)		\
	LEAQ	LOCAL_REGARGS(SP), R12		\
	CALL	·unspillArgs(SB)		\
	RET

`

var regabi_arm64 = `//go:build (go1.18 && goexperiment.regabireflect) || (go1.19 && !go1.21)
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

`

var regabi_ppc64x = `//go:build ((go1.18 && goexperiment.regabireflect) || go1.19) && (ppc64 || ppc64le)
// +build go1.18,goexperiment.regabireflect go1.19
// +build ppc64 ppc64le

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

`

var regabi_riscv64 = `//go:build (go1.19 && goexperiment.regabiargs) || go1.20
// +build go1.19,goexperiment.regabiargs go1.20

// Copyright 2019 The Go Authors. All rights reserved.
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
// 32 (args of callReflect/callMethod) + (8 bool with padding) + 392 (abi.RegArgs) = 432.

// makeFuncStub is the code half of the function returned by MakeFunc.
// See the comment on the declaration of makeFuncStub in makefunc.go
// for more details.
// No arg size here, runtime pulls arg map out of the func value.
#define MAKE_FUNC_FN(NAME,INDEX)		\
TEXT NAME(SB),(NOSPLIT|WRAPPER),$432	\
	NO_LOCAL_POINTERS	\
	ADD	$LOCAL_REGARGS, SP, X25 	\
	CALL	runtime·spillArgs(SB)	\
	MOV	32(SP), CTXT 		\
	MOV	CTXT, 8(SP)		\
	MOV	$argframe+0(FP), T0		\
	MOV	T0, 16(SP)		\
	MOV	ZERO, LOCAL_RETVALID(SP)		\
	ADD	$LOCAL_RETVALID, SP, T1		\
	MOV	T1, 24(SP)		\
	ADD	$LOCAL_REGARGS, SP, T1		\
	MOV	T1, 32(SP)		\
	MOV	$INDEX, T1		\
	MOV	T1, 40(SP)		\
	CALL	·i_x(SB)		\
	ADD	$LOCAL_REGARGS, SP, X25 		\
	CALL	runtime·unspillArgs(SB)		\
	RET

`
