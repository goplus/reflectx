//go:build go1.27 && !llgo

package reflectx

func cloneMap(st, ost *mapType) {
	st.Key = ost.Key
	st.Elem = ost.Elem
	st.Group = ost.Group
	st.Hasher = ost.Hasher
	st.GroupSize = ost.GroupSize
	st.KeysOff = ost.KeysOff
	st.KeyStride = ost.KeyStride
	st.ElemsOff = ost.ElemsOff
	st.ElemStride = ost.ElemStride
	st.ElemOff = ost.ElemOff
	st.Flags = ost.Flags
}
