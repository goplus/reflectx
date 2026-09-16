//go:build goplus.ifacefuncval && !llgo && (wasm || arm64)

package reflectx

import (
	"reflect"
	"unsafe"

	"github.com/goplus/reflectx/abi"
)

// ifaceFuncvalBit marks an itab.Fun entry as a MakeFunc funcval.
// Must match cmd/internal/obj/wasm.IfaceFuncvalBit in the patched Go
// toolchain (wasm or arm64 -tags goplus.ifacefuncval → compile -ifacefuncval).
const ifaceFuncvalBit = 1

// ifaceFuncvalFns keeps MakeFunc funcvals reachable. itab.Fun stores the
// tagged pointer as a uintptr, so the GC will not scan it.
var ifaceFuncvalFns []reflect.Value

// clearIfaceFuncval drops those keep-alives. Call only when no live itab
// still references them (resetAll / process teardown). Context.Reset does
// not clear this slice; discarded methods stay pinned until ResetAll.
func clearIfaceFuncval() {
	ifaceFuncvalFns = nil
}

// ifaceFuncval returns a wasm itab.Fun entry for info.
// The patched Go wasm toolchain treats an odd function pointer as a
// funcval: CTXT is the untagged pointer and the call target is the
// first word (makeFuncStub).
func ifaceFuncval(info *abi.MethodInfo) unsafe.Pointer {
	fn := info.Func
	if (!info.Pointer && !info.OnePtr) || info.Indirect {
		ftyp := fn.Type()
		numIn := ftyp.NumIn()
		numOut := ftyp.NumOut()
		in := make([]reflect.Type, numIn)
		out := make([]reflect.Type, numOut)
		in[0] = reflect.PtrTo(info.Type)
		for i := 1; i < numIn; i++ {
			in[i] = ftyp.In(i)
		}
		for i := 0; i < numOut; i++ {
			out[i] = ftyp.Out(i)
		}
		call := info.Call
		fn = reflect.MakeFunc(reflect.FuncOf(in, out, info.Variadic), func(args []reflect.Value) []reflect.Value {
			args[0] = args[0].Elem()
			return call(args)
		})
	}
	ifaceFuncvalFns = append(ifaceFuncvalFns, fn)
	return unsafe.Pointer(uintptr(tovalue(&fn).ptr) | ifaceFuncvalBit)
}
