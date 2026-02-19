//go:build go1.19
// +build go1.19

package reflectx

import (
	_ "reflect"
	_ "unsafe"
)

func newName(n, tag string, exported bool) name {
	return newNameEx(n, tag, exported, false)
}
