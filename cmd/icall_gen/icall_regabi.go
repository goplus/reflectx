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
	return nil
}

var icall_regabi = `//go:build go1.17 && goexperiment.regabireflect
// +build go1.17,goexperiment.regabireflect

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

//go:linkname moveMakeFuncArgPtrs reflect.moveMakeFuncArgPtrs
func moveMakeFuncArgPtrs(ctx unsafe.Pointer, r unsafe.Pointer)

var infos []*reflectx.MethodInfo
var funcs []reflect.Value
var fnptr []unsafe.Pointer

func i_x(index int, c unsafe.Pointer, frame unsafe.Pointer, retValid *bool, r unsafe.Pointer) {
	moveMakeFuncArgPtrs(fnptr[index], r)
	callReflect(fnptr[index], unsafe.Pointer(uintptr(frame)+ptrSize), retValid, r)
}

const ptrSize = (32 << (^uint(0) >> 63)) / 8

func spillArgs()
func unspillArgs()

func (p *provider) Push(info *reflectx.MethodInfo) (ifn unsafe.Pointer) {
	fn := icall_fn[len(infos)]
	infos = append(infos, info)

	ftyp := info.Func.Type()
	toPtr := (!info.Pointer && !info.OnePtr) || info.Indirect
	if toPtr {
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
		ftyp = reflect.FuncOf(in, out, ftyp.IsVariadic())
	}
	v := reflect.MakeFunc(ftyp, func(args []reflect.Value) []reflect.Value {
		if toPtr {
			args[0] = args[0].Elem()
		}
		if info.Variadic {
			return info.Func.CallSlice(args)
		}
		return info.Func.Call(args)
	})
	funcs = append(funcs, v)
	fnptr = append(fnptr, (*struct{ typ, ptr unsafe.Pointer })(unsafe.Pointer(&v)).ptr)

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
	funcs = nil
	fnptr = nil
}

var (
	mp provider
)

func init() {
	reflectx.AddMethodProvider(&mp)
}

`

var regabi_amd64 = `//go:build go1.17 && goexperiment.regabireflect
// +build go1.17,goexperiment.regabireflect

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
	LEAQ	argframe+0(FP), CX		\
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