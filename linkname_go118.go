//go:build !go1.23
// +build !go1.23

package reflectx

import (
	_ "reflect"
	_ "runtime"
	"unsafe"

	_ "github.com/goplus/reflectx/x/reflect"
)

//go:linkname interequal runtime.interequal
func interequal(p, q unsafe.Pointer) bool

//go:linkname toUncommonType reflect.(*rtype).uncommon
func toUncommonType(t *rtype) *uncommonType
