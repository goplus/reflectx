//go:build ignore

package runtime

import "unsafe"

// tflagIfaceFuncval matches github.com/goplus/reflectx tflagIfaceFuncval (1<<6).
const tflagIfaceFuncval = 1 << 6

func itabFuncval(ifn unsafe.Pointer, tflag uint8) uintptr {
	u := uintptr(ifn)
	if u != 0 && tflag&tflagIfaceFuncval != 0 {
		u |= 1
	}
	return u
}
