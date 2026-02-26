//go:build ((go1.17 && goexperiment.regabireflect) || (go1.19 && goexperiment.regabiargs) || (go1.18 && amd64) || (go1.19 && arm64) || (go1.19 && ppc64) || (go1.19 && ppc64le) || (go1.20 && riscv64) || (go1.23 && loong64)) && (!go1.23 || linknamefix)
// +build go1.17,goexperiment.regabireflect go1.19,goexperiment.regabiargs go1.18,amd64 go1.19,arm64 go1.19,ppc64 go1.19,ppc64le go1.20,riscv64 go1.23,loong64
// +build !go1.23 linknamefix

package icall

import (
	_ "reflect"
	"unsafe"
)

//go:linkname callReflect reflect.callReflect
func callReflect(ctxt unsafe.Pointer, frame unsafe.Pointer, retValid *bool, r unsafe.Pointer)

//go:linkname moveMakeFuncArgPtrs reflect.moveMakeFuncArgPtrs
func moveMakeFuncArgPtrs(ctx unsafe.Pointer, r unsafe.Pointer)
