//go:build !llgo && (!goplus.ifacefuncval || (!wasm && !arm64))

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
