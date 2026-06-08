// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package reflect implements run-time reflection, allowing a program to
// manipulate objects with arbitrary types. The typical use is to take a value
// with static type interface{} and extract its dynamic type information by
// calling TypeOf, which returns a Type.
//
// A call to ValueOf returns a Value representing the run-time data.
// Zero takes a Type and returns a Value representing a zero value
// for that type.
//
// See "The Laws of Reflection" for an introduction to reflection in Go:
// https://golang.org/doc/articles/laws_of_reflection.html

package reflectx

import (
	"reflect"
	"unsafe"

	"github.com/goplus/reflectx/internal/abi"
)

// Type aliases to internal/abi types
type rtype = abi.Type
type nameOff = abi.NameOff
type typeOff = abi.TypeOff
type textOff = abi.TextOff
type tflag = abi.TFlag
type method = abi.Method
type name = abi.Name
type imethod = abi.Imethod
type uncommonType = abi.UncommonType
type structField = abi.StructField
type arrayType = abi.ArrayType
type chanType = abi.ChanType
type interfaceType = abi.InterfaceType
type ptrType = abi.PtrType
type sliceType = abi.SliceType
type structType = abi.StructType
type funcType = abi.FuncType

type structTypeUncommon struct {
	structType
	u uncommonType
}

const (
	tflagUncommon      = abi.TFlagUncommon
	tflagExtraStar     = abi.TFlagExtraStar
	tflagNamed         = abi.TFlagNamed
	tflagRegularMemory = abi.TFlagRegularMemory
	tflagUserMethod    tflag = 1 << 7
)

// add returns p+x.
//
// The whySafe string is ignored, so that the function still inlines
// as efficiently as p+x, but all call sites should use the string to
// record why the addition is safe, which is to say why the addition
// does not cause x to advance to the very end of p's allocation
// and therefore point incorrectly at the next block in memory.
func add(p unsafe.Pointer, x uintptr, whySafe string) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

// stringHeader is a safe version of StringHeader used within this package.
type stringHeader struct {
	Data unsafe.Pointer
	Len  int
}

// ChanDir represents a channel type's direction.
type ChanDir int

const (
	RecvDir ChanDir             = 1 << iota // <-chan
	SendDir                                 // chan<-
	BothDir = RecvDir | SendDir             // chan
)

// go/src/cmd/compile/internal/gc/alg.go#algtype1
// IsRegularMemory reports whether t can be compared/hashed as regular memory.
func isRegularMemory(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice, reflect.String, reflect.Interface:
		return false
	case reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return false
	case reflect.Array:
		b := isRegularMemory(t.Elem())
		if b {
			return true
		}
		if t.Len() == 0 {
			return true
		}
		return b
	case reflect.Struct:
		n := t.NumField()
		switch n {
		case 0:
			return true
		case 1:
			f := t.Field(0)
			if f.Name == "_" {
				return false
			}
			return isRegularMemory(f.Type)
		default:
			for i := 0; i < n; i++ {
				f := t.Field(i)
				if f.Name == "_" || !isRegularMemory(f.Type) || ispaddedfield(t, i) {
					return false
				}
			}
		}
	}
	return true
}

// ispaddedfield reports whether the i'th field of struct type t is followed
// by padding.
func ispaddedfield(t reflect.Type, i int) bool {
	end := t.Size()
	if i+1 < t.NumField() {
		end = t.Field(i + 1).Offset
	}
	fd := t.Field(i)
	return fd.Offset+fd.Type.Size() != end
}
