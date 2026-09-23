//go:build ignore

package runtime

import "unsafe"

// taggedTextOffs maps addReflectOff IDs to tagged MakeFunc Ifn values
// (makeFuncImpl*|1). The ID is allocated from a unique heap key so Tfn
// can keep the untagged impl in reflectOffs.m. itab.Fun and
// reflect.Value.Method use the tagged uintptr, which the GC does not scan.
var taggedTextOffs map[int32]uintptr

//go:linkname reflect_registerTaggedTextOff
func reflect_registerTaggedTextOff(id int32, tagged uintptr) {
	if id >= 0 || tagged == 0 {
		return
	}
	reflectOffsLock()
	if taggedTextOffs == nil {
		taggedTextOffs = make(map[int32]uintptr)
	}
	taggedTextOffs[id] = tagged
	reflectOffsUnlock()
}

func itabFuncval(ifn unsafe.Pointer) uintptr {
	return uintptr(ifn)
}

//go:nocheckptr
func taggedTextOff(off int32, untagged unsafe.Pointer) unsafe.Pointer {
	reflectOffsLock()
	tagged, ok := taggedTextOffs[off]
	reflectOffsUnlock()
	if ok {
		return unsafe.Pointer(tagged)
	}
	return untagged
}
