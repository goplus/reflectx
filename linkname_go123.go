//go:build go1.23
// +build go1.23

package reflectx

import (
	"unsafe"

	_ "github.com/goplus/reflectx/x/reflect"
)

//go:linkname interequal github.com/goplus/reflectx/x/reflect.interequal
func interequal(p, q unsafe.Pointer) bool

//go:linkname toUncommonType github.com/goplus/reflectx/x/reflect.toUncommonType
func toUncommonType(t *rtype) *uncommonType
