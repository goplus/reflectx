//go:build go1.24 && !llgo

package reflectx

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
