//go:build !go1.24
// +build !go1.24

package reflectx

import (
	"github.com/goplus/reflectx/internal/abi"
)

// mapType is abi.MapType for the noswiss map implementation
type mapType = abi.MapType

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
