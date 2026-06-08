//go:build go1.24
// +build go1.24

package reflectx

import (
	"github.com/goplus/reflectx/internal/abi"
)

// mapType is abi.MapType for the swiss map implementation
type mapType = abi.MapType

func cloneMap(st, ost *mapType) {
	st.Key = ost.Key
	st.Elem = ost.Elem
	st.Group = ost.Group
	st.Hasher = ost.Hasher
	st.GroupSize = ost.GroupSize
	st.SlotSize = ost.SlotSize
	st.ElemOff = ost.ElemOff
	st.Flags = ost.Flags
}
