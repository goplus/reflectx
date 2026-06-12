//go:build (!go1.23 || linknamefix) && !llgo
// +build !go1.23 linknamefix
// +build !llgo

package reflectx

import (
	_ "reflect"
	_ "runtime"
	"unsafe"
)

//go:linkname interequal runtime.interequal
func interequal(p, q unsafe.Pointer) bool

//go:linkname haveIdenticalType reflect.haveIdenticalType
func haveIdenticalType(T, V *rtype, cmpTags bool) bool
