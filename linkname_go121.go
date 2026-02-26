//go:build go1.21 && (!go1.23 || linknamefix)
// +build go1.21
// +build !go1.23 linknamefix

package reflectx

import (
	_ "reflect"
	_ "runtime"
	"unsafe"
)

//go:linkname interequal runtime.interequal
func interequal(p, q unsafe.Pointer) bool

//go:linkname toUncommonType reflect.(*rtype).uncommon
func toUncommonType(t *rtype) *uncommonType

//go:linkname haveIdenticalType reflect.haveIdenticalType
func haveIdenticalType(T, V *rtype, cmpTags bool) bool
