//go:build !go1.24
// +build !go1.24

package reflectx

import "unsafe"

// mapType represents a map type.
type mapType struct {
	rtype
	key    *rtype // map key type
	elem   *rtype // map element (value) type
	bucket *rtype // internal bucket structure
	// function for hashing keys (ptr to key, seed) -> hash
	hasher     func(unsafe.Pointer, uintptr) uintptr
	keysize    uint8  // size of key slot
	valuesize  uint8  // size of value slot
	bucketsize uint16 // size of bucket
	flags      uint32
}

func cloneMap(st, ost *mapType) {
	st.key = ost.key
	st.elem = ost.elem
	st.bucket = ost.bucket
	st.hasher = ost.hasher
	st.keysize = ost.keysize
	st.valuesize = ost.valuesize
	st.bucketsize = ost.bucketsize
	st.flags = ost.flags
}
