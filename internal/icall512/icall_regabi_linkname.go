//go:build (goexperiment.regabiargs || amd64 || arm64 || ppc64 || ppc64le || riscv64 || (go1.23 && loong64)) && (!go1.23 || linknamefix)
// +build goexperiment.regabiargs amd64 arm64 ppc64 ppc64le riscv64 go1.23,loong64
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
