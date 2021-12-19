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
		err := os.MkdirAll(dir, 0777)
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
	for i := 0; i < size; i++ {
		buf.WriteString(fmt.Sprintf(`func x%v(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(%v, c, f, v, r)
}
`, i, i))
	}

	err := ioutil.WriteFile(gofile, buf.Bytes(), 0666)
	if err != nil {
		return err
	}

	asm117file := filepath.Join(dir, strings.Replace(f, ".go", "_regabi_go117_amd64.s", 1))
	asm118file := filepath.Join(dir, strings.Replace(f, ".go", "_regabi_go118_amd64.s", 1))
	fnWrite := func(filename string, tmpl string, size int) error {
		var buf bytes.Buffer
		buf.WriteString(tmpl)
		for i := 0; i < size; i++ {
			buf.WriteString(fmt.Sprintf("MAKE_FUNC_FN(·f%v,·x%v)\n", i, i))
		}
		return ioutil.WriteFile(filename, buf.Bytes(), 0666)
	}
	err = fnWrite(asm117file, regabi_go117_amd64, size)
	if err != nil {
		return err
	}
	err = fnWrite(asm118file, regabi_go118_amd64, size)
	if err != nil {
		return err
	}
	return nil
}

var icall_regabi = `//go:build (go1.17 && goexperiment.regabireflect) || (go1.18 && amd64) || (go1.18 && goexperiment.regabireflect)
// +build go1.17,goexperiment.regabireflect go1.18,amd64 go1.18,goexperiment.regabireflect

package $pkgname

import (
	"reflect"
	"unsafe"

	"github.com/goplus/reflectx"
)

const capacity = $max_size

type provider struct {
}

//go:linkname callReflect reflect.callReflect
func callReflect(ctxt unsafe.Pointer, frame unsafe.Pointer, retValid *bool, r unsafe.Pointer)

var infos []*reflectx.MethodInfo

func i_x(index int, frame unsafe.Pointer, retValid *bool, r unsafe.Pointer) {
	info := infos[index]
	this := reflect.NewAt(info.Type, unsafe.Pointer(*(**uintptr)(r)))
	if !info.Pointer || info.Indirect {
		this = this.Elem()
	}
	v := reflect.MakeFunc(info.Func.Type(), func(args []reflect.Value) []reflect.Value {
		args[0] = this
		if info.Variadic {
			return info.Func.CallSlice(args)
		}
		return info.Func.Call(args)
	})
	callReflect(tovalue(&v).ptr, frame, retValid, r)
}

func (p *provider) Push(info *reflectx.MethodInfo) (ifn unsafe.Pointer) {
	fn := icall_fn[len(infos)]
	infos = append(infos, info)
	return unsafe.Pointer(reflect.ValueOf(fn).Pointer())
}

func (p *provider) Len() int {
	return len(infos)
}

func (p *provider) Cap() int {
	return capacity
}

func (p *provider) Clear() {
	infos = nil
}

var (
	mp provider
)

func init() {
	reflectx.AddMethodProvider(&mp)
}

type Value struct {
	typ  unsafe.Pointer
	ptr  unsafe.Pointer
	flag uintptr
}

func tovalue(v *reflect.Value) *Value {
	return (*Value)(unsafe.Pointer(v))
}

type unsafeptr = unsafe.Pointer

`

var regabi_go117_amd64 = `//go:build go1.17 && !go1.18
// +build go1.17,!go1.18

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
#define MAKE_FUNC_FN(NAME,FNCALL)		\
TEXT NAME(SB),(NOSPLIT|WRAPPER),$312		\
	NO_LOCAL_POINTERS		\
	LEAQ	LOCAL_REGARGS(SP), R12		\
	CALL	·spillArgs(SB)		\
	MOVQ	DX, 24(SP)		\
	MOVQ	DX, 0(SP)		\
	MOVQ	R12, 8(SP)		\
	CALL	reflect·moveMakeFuncArgPtrs(SB)		\
	MOVQ	24(SP), DX		\
	MOVQ	DX, 0(SP)		\
	LEAQ	argframe+0(FP), CX		\
	MOVQ	CX, 8(SP)		\
	MOVB	$0, LOCAL_RETVALID(SP)		\
	LEAQ	LOCAL_RETVALID(SP), AX		\
	MOVQ	AX, 16(SP)		\
	LEAQ	LOCAL_REGARGS(SP), AX		\
	MOVQ	AX, 24(SP)		\
	CALL	FNCALL(SB)		\
	LEAQ	LOCAL_REGARGS(SP), R12		\
	CALL	·unspillArgs(SB)		\
	RET
`

var regabi_go118_amd64 = `//go:build go1.18
// +build go1.18


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
#define MAKE_FUNC_FN(NAME,FNCALL)		\
TEXT NAME(SB),(NOSPLIT|WRAPPER),$312		\
	NO_LOCAL_POINTERS		\
	LEAQ	LOCAL_REGARGS(SP), R12		\
	CALL	runtime·spillArgs(SB)		\
	MOVQ	DX, 24(SP)		\
	MOVQ	DX, 0(SP)		\
	MOVQ	R12, 8(SP)		\
	CALL	reflect·moveMakeFuncArgPtrs(SB)		\
	MOVQ	24(SP), DX		\
	MOVQ	DX, 0(SP)		\
	LEAQ	argframe+0(FP), CX		\
	MOVQ	CX, 8(SP)		\
	MOVB	$0, LOCAL_RETVALID(SP)		\
	LEAQ	LOCAL_RETVALID(SP), AX		\
	MOVQ	AX, 16(SP)		\
	LEAQ	LOCAL_REGARGS(SP), AX		\
	MOVQ	AX, 24(SP)		\
	CALL	FNCALL(SB)		\
	LEAQ	LOCAL_REGARGS(SP), R12		\
	CALL	runtime·unspillArgs(SB)		\
	RET
`
