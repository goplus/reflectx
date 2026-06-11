//go:build go1.26 && !llgo
// +build go1.26,!llgo

package abi

// IsDirectIface reports whether t is stored directly in an interface value.
func (t *Type) IsDirectIface() bool {
	return t.TFlag&TFlagDirectIface != 0
}

func (t *Type) Kind() Kind { return t.Kind_ }
