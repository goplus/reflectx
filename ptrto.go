//go:build !go1.23 && !llgo
// +build !go1.23,!llgo

package reflectx

import (
	"reflect"
)

func PtrTo(t reflect.Type) reflect.Type {
	return reflect.PtrTo(t)
}
