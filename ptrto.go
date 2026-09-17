//go:build !llgo && !go1.23
// +build !llgo,!go1.23

package reflectx

import (
	"reflect"
)

func PtrTo(t reflect.Type) reflect.Type {
	return reflect.PtrTo(t)
}
