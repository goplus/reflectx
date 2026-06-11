//go:build !go1.26 && !llgo
// +build !go1.26,!llgo

package abi

const (
	KindMask Kind = (1 << 5) - 1
)

// IsDirectIface reports whether t is stored directly in an interface value.
func (t *Type) IsDirectIface() bool {
	return t.Kind_&KindDirectIface != 0
}

func (t *Type) Kind() Kind { return t.Kind_ & KindMask }
