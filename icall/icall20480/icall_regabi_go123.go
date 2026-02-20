//go:build ((go1.17 && goexperiment.regabireflect) || (go1.19 && goexperiment.regabiargs) || (go1.18 && amd64) || (go1.19 && arm64) || (go1.19 && ppc64) || (go1.19 && ppc64le) || (go1.20 && riscv64) || (go1.23 && loong64)) && go1.23
// +build go1.17,goexperiment.regabireflect go1.19,goexperiment.regabiargs go1.18,amd64 go1.19,arm64 go1.19,ppc64 go1.19,ppc64le go1.20,riscv64 go1.23,loong64
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
