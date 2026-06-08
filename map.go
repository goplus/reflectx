//go:build !go1.24
// +build !go1.24

package reflectx

func cloneMap(st, ost *mapType) {
	st.Key = ost.Key
	st.Elem = ost.Elem
	st.Bucket = ost.Bucket
	st.Hasher = ost.Hasher
	st.KeySize = ost.KeySize
	st.ValueSize = ost.ValueSize
	st.BucketSize = ost.BucketSize
	st.Flags = ost.Flags
}
