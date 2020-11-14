// +build !js js,wasm

package reflectx

import (
	"unsafe"
)

func toStructType(typ *rtype) *structType {
	return (*structType)(unsafe.Pointer(typ))
}
