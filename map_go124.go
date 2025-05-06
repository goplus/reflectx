//go:build go1.24
// +build go1.24

package reflectx

import "unsafe"

type mapType struct {
	rtype
	key   *rtype
	elem  *rtype
	group *rtype // internal type representing a slot group
	// function for hashing keys (ptr to key, seed) -> hash
	hasher    func(unsafe.Pointer, uintptr) uintptr
	groupSize uintptr // == Group.Size_
	slotSize  uintptr // size of key/elem slot
	elemOff   uintptr // offset of elem in key/elem slot
	flags     uint32
}

func cloneMap(st, ost *mapType) {
	st.key = ost.key
	st.elem = ost.elem
	st.group = ost.group
	st.hasher = ost.hasher
	st.groupSize = ost.groupSize
	st.slotSize = ost.slotSize
	st.elemOff = ost.elemOff
	st.flags = ost.flags
}
