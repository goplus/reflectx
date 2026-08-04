//go:build goexperiment.regabiargs || amd64 || arm64 || ppc64 || ppc64le || riscv64 || (go1.23 && loong64)
// +build goexperiment.regabiargs amd64 arm64 ppc64 ppc64le riscv64 go1.23,loong64

package pkgname

import (
	"reflect"
	"sync/atomic"
	"unsafe"

	"github.com/goplus/reflectx/abi"
)

const capacity = 1024

type provider struct {
	used []unsafe.Pointer
	n    atomic.Int64
}

//go:nosplit
func i_x(c unsafe.Pointer, frame unsafe.Pointer, retValid *bool, r unsafe.Pointer, index int) {
	ptr := atomic.LoadPointer(&mp.used[index])
	moveMakeFuncArgPtrs(ptr, r)
	callReflect(ptr, frame, retValid, r)
}

func spillArgs()
func unspillArgs()

func (p *provider) Insert(info *abi.MethodInfo) (unsafe.Pointer, int) {
	var fn reflect.Value
	if (!info.Pointer && !info.OnePtr) || info.Indirect {
		ftyp := info.Func.Type()
		numIn := ftyp.NumIn()
		numOut := ftyp.NumOut()
		in := make([]reflect.Type, numIn, numIn)
		out := make([]reflect.Type, numOut, numOut)
		in[0] = reflect.PtrTo(info.Type)
		for i := 1; i < numIn; i++ {
			in[i] = ftyp.In(i)
		}
		for i := 0; i < numOut; i++ {
			out[i] = ftyp.Out(i)
		}
		ftyp = reflect.FuncOf(in, out, info.Variadic)
		fn = reflect.MakeFunc(ftyp, func(args []reflect.Value) []reflect.Value {
			args[0] = args[0].Elem()
			return info.Call(args)
		})
	} else {
		fn = info.Func
	}
	ptr := (*struct{ typ, ptr unsafe.Pointer })(unsafe.Pointer(&fn)).ptr
	for i := 0; i < capacity; i++ {
		if atomic.CompareAndSwapPointer(&p.used[i], nil, ptr) {
			p.n.Add(1)
			icall := icall_fn[i]
			return unsafe.Pointer(reflect.ValueOf(icall).Pointer()), i
		}
	}
	return nil, -1
}

func (p *provider) Remove(indexs []int) {
	for _, n := range indexs {
		if n < 0 || n >= capacity {
			continue
		}
		if atomic.SwapPointer(&p.used[n], nil) != nil {
			p.n.Add(-1)
		}
	}
}

func (p *provider) Available() int {
	return capacity - int(p.n.Load())
}

func (p *provider) Used() int {
	return int(p.n.Load())
}

func (p *provider) Cap() int {
	return capacity
}

func (p *provider) Clear() {
	clear(p.used)
	p.n.Store(0)
}

var (
	mp = &provider{
		used: make([]unsafe.Pointer, capacity),
	}
)

func init() {
	abi.AddMethodProvider(mp)
}
