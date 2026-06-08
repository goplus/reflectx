//go:build go1.23 && !linknamefix && !llgo
// +build go1.23,!linknamefix,!llgo

package reflectx

import (
	"unsafe"

	_ "github.com/goplus/reflectx/x/reflect"
)

//go:linkname interequal github.com/goplus/reflectx/x/reflect.interequal
func interequal(p, q unsafe.Pointer) bool

//go:linkname toUncommonType github.com/goplus/reflectx/x/reflect.toUncommonType
func toUncommonType(t *rtype) *uncommonType

//go:linkname haveIdenticalType github.com/goplus/reflectx/x/reflect.haveIdenticalType
func haveIdenticalType(T, V *rtype, cmpTags bool) bool
