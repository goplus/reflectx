//go:build go1.23
// +build go1.23

package icall

import (
	"unsafe"

	_ "github.com/goplus/reflectx/x/reflect"
)

//go:linkname callReflect github.com/goplus/reflectx/x/reflect.callReflect
func callReflect(ctxt unsafe.Pointer, frame unsafe.Pointer, retValid *bool, r unsafe.Pointer)

//go:linkname moveMakeFuncArgPtrs github.com/goplus/reflectx/x/reflect.moveMakeFuncArgPtrs
func moveMakeFuncArgPtrs(ctx unsafe.Pointer, r unsafe.Pointer)
