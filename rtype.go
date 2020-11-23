// +build !js js,wasm

package reflectx

import (
	"unsafe"
)

func toStructType(typ *rtype) *structType {
	return (*structType)(unsafe.Pointer(typ))
}

func toUncommonType(typ *rtype) *uncommonType {
	return &(*structTypeUncommon)(unsafe.Pointer(typ)).u
}

func setUncommonTypePkgPath(typ *rtype, n nameOff) {
	ut := toUncommonType(typ)
	ut.pkgPath = n
	typ.tflag |= tflagUncommon
}

// uncommonType is present only for defined types or types with methods
// (if T is a defined type, the uncommonTypes for T and *T have methods).
// Using a pointer to this struct reduces the overall size required
// to describe a non-defined type with no methods.
type uncommonType struct {
	pkgPath nameOff // import path; empty for built-in types like int, string
	mcount  uint16  // number of methods
	xcount  uint16  // number of exported methods
	moff    uint32  // offset from this uncommontype to [mcount]method
	_       uint32  // unused
}
