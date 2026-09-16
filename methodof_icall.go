//go:build !llgo && (!wasm || !goplus.ifacefuncval)

package reflectx

import (
	"unsafe"

	"github.com/goplus/reflectx/abi"
	_ "github.com/goplus/reflectx/internal/icall512"
)

func clearIfaceFuncval() {}

func ifaceFuncval(info *abi.MethodInfo) unsafe.Pointer {
	return nil
}
