package reflectx

import (
	"reflect"
	"unsafe"

	_ "github.com/goplus/reflectx/x/reflect"
)

// memmove copies size bytes to dst from src. No write barriers are used.
//
//go:noescape
//go:linkname memmove reflect.memmove
func memmove(dst, src unsafe.Pointer, size uintptr)

// typedmemmove copies a value of type t to dst from src.
//
//go:noescape
//go:linkname typedmemmove reflect.typedmemmove
func typedmemmove(t *rtype, dst, src unsafe.Pointer)

// resolveTypeOff resolves an *rtype offset from a base type.
// The (*rtype).typeOff method is a convenience wrapper for this function.
// Implemented in the runtime package.
//
//go:linkname resolveTypeOff reflect.resolveTypeOff
func resolveTypeOff(rtype unsafe.Pointer, off int32) unsafe.Pointer

// resolveTextOff resolves a function pointer offset from a base type.
// The (*rtype).textOff method is a convenience wrapper for this function.
// Implemented in the runtime package.
//
//go:linkname resolveTextOff reflect.resolveTextOff
func resolveTextOff(rtype unsafe.Pointer, off int32) unsafe.Pointer

// addReflectOff adds a pointer to the reflection lookup map in the runtime.
// It returns a new ID that can be used as a typeOff or textOff, and will
// be resolved correctly. Implemented in the runtime package.
//
//go:linkname addReflectOff reflect.addReflectOff
func addReflectOff(ptr unsafe.Pointer) int32

// resolveReflectName adds a name to the reflection lookup map in the runtime.
// It returns a new nameOff that can be used to refer to the pointer.
func resolveReflectName(n name) nameOff {
	return nameOff(addReflectOff(unsafe.Pointer(n.bytes)))
}

//go:linkname resolveNameOff reflect.resolveNameOff
func resolveNameOff(ptrInModule unsafe.Pointer, off int32) unsafe.Pointer

//go:linkname toType reflect.toType
func toType(t *rtype) reflect.Type

func rtype_nameOff(t *rtype, off nameOff) name {
	return name{bytes: (*byte)(resolveNameOff(unsafe.Pointer(t), int32(off)))}
}

func rtype_typeOff(t *rtype, off typeOff) *rtype {
	return (*rtype)(resolveTypeOff(unsafe.Pointer(t), int32(off)))
}

func rtype_textOff(t *rtype, off textOff) unsafe.Pointer {
	return resolveTextOff(unsafe.Pointer(t), int32(off))
}

//go:linkname haveIdenticalType github.com/goplus/reflectx/x/reflect.haveIdenticalType
func haveIdenticalType(T, V *rtype, cmpTags bool) bool
