//go:build !go1.19
// +build !go1.19

package reflectx

import (
	_ "reflect"
	_ "unsafe"
)

//go:linkname newName reflect.newName
func newName(n, tag string, exported bool) name
