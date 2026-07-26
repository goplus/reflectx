//go:build (goexperiment.regabiargs || amd64 || arm64 || ppc64 || ppc64le || riscv64 || (go1.23 && loong64)) && go1.23 && !linknamefix
// +build goexperiment.regabiargs amd64 arm64 ppc64 ppc64le riscv64 go1.23,loong64
// +build go1.23
// +build !linknamefix

package pkgname

import (
	"unsafe"

	_ "github.com/goplus/reflectx/x/reflect"
)

//go:linkname callReflect github.com/goplus/reflectx/x/reflect.callReflect
func callReflect(ctxt unsafe.Pointer, frame unsafe.Pointer, retValid *bool, r unsafe.Pointer)

//go:linkname moveMakeFuncArgPtrs github.com/goplus/reflectx/x/reflect.moveMakeFuncArgPtrs
func moveMakeFuncArgPtrs(ctx unsafe.Pointer, r unsafe.Pointer)
