//go:build !llgo && go1.23
// +build !llgo,go1.23

package reflectx

import (
	"reflect"
	_ "unsafe"
)

//go:linkname toPointer reflect.(*rtype).ptrTo
func toPointer(t *rtype) *rtype

func PtrTo(t reflect.Type) reflect.Type {
	return toType(toPointer(totype(t)))
}
