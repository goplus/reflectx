//go:build !go1.21
// +build !go1.21

package reflectx

import (
	"reflect"
	_ "runtime"
	"unsafe"
)

//go:linkname interequal runtime.interequal
func interequal(p, q unsafe.Pointer) bool

//go:linkname toUncommonType reflect.(*rtype).uncommon
func toUncommonType(t *rtype) *uncommonType

//go:linkname _haveIdenticalType reflect.haveIdenticalType
func _haveIdenticalType(T, V reflect.Type, cmpTags bool) bool

func haveIdenticalType(T, V *rtype, cmpTags bool) bool {
	return _haveIdenticalType(toType(T), toType(V), cmpTags)
}
