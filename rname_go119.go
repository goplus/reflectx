//go:build go1.19
// +build go1.19

package reflectx

import (
	_ "reflect"
	_ "unsafe"
)

//go:linkname _newName reflect.newName
func _newName(n, tag string, exported, embedded bool) name

func newName(n, tag string, exported bool) name {
	return _newName(n, tag, exported, false)
}
