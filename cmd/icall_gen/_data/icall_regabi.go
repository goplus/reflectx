//go:build goexperiment.regabiargs || amd64 || arm64 || ppc64 || ppc64le || riscv64 || (go1.23 && loong64)
// +build goexperiment.regabiargs amd64 arm64 ppc64 ppc64le riscv64 go1.23,loong64

package $pkgname

import (
	"reflect"
	"runtime"
	"sync"
	"unsafe"

	"github.com/goplus/reflectx/abi"
)

const capacity = $max_size

type methodUsed struct {
	fun reflect.Value
	ptr unsafe.Pointer
}

type provider struct {
	mu   sync.RWMutex
	used []*methodUsed
	n    int
}

func i_x(c unsafe.Pointer, frame unsafe.Pointer, retValid *bool, r unsafe.Pointer, index int) {
	method := mp.lookup(index)
	ptr := method.ptr
	moveMakeFuncArgPtrs(ptr, r)
	callReflect(ptr, frame, retValid, r)
	// Context.Reset may clear the slot after lookup. Keep the copied method alive
	// until callReflect returns without holding the provider lock over user code.
	runtime.KeepAlive(method)
}

func spillArgs()
func unspillArgs()

func (p *provider) Insert(info *abi.MethodInfo) (unsafe.Pointer, int) {
	p.mu.Lock()
	var index = -1
	for i := 0; i < capacity; i++ {
		if p.used[i] == nil {
			index = i
			break
		}
	}
	if index == -1 {
		p.mu.Unlock()
		return nil, -1
	}
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
		if info.Variadic {
			fn = reflect.MakeFunc(ftyp, func(args []reflect.Value) []reflect.Value {
				args[0] = args[0].Elem()
				return info.Func.CallSlice(args)
			})
		} else {
			fn = reflect.MakeFunc(ftyp, func(args []reflect.Value) []reflect.Value {
				args[0] = args[0].Elem()
				return info.Func.Call(args)
			})
		}
	} else {
		fn = info.Func
	}
	p.used[index] = &methodUsed{
		fun: fn,
		ptr: (*struct{ typ, ptr unsafe.Pointer })(unsafe.Pointer(&fn)).ptr,
	}
	p.n++
	icall := icall_fn[index]
	p.mu.Unlock()
	return unsafe.Pointer(reflect.ValueOf(icall).Pointer()), index
}

func (p *provider) Remove(indexs []int) {
	p.mu.Lock()
	for _, n := range indexs {
		if n < capacity && p.used[n] != nil {
			p.used[n] = nil
			p.n--
		}
	}
	p.mu.Unlock()
}

func (p *provider) Available() int {
	p.mu.RLock()
	n := capacity - p.n
	p.mu.RUnlock()
	return n
}

func (p *provider) Used() int {
	p.mu.RLock()
	n := p.n
	p.mu.RUnlock()
	return n
}

func (p *provider) Cap() int {
	return capacity
}

func (p *provider) Clear() {
	p.mu.Lock()
	clear(p.used)
	p.n = 0
	p.mu.Unlock()
}

func (p *provider) lookup(index int) *methodUsed {
	p.mu.RLock()
	method := p.used[index]
	p.mu.RUnlock()
	return method
}

var (
	mp = &provider{
		used: make([]*methodUsed, capacity),
	}
)

func init() {
	abi.AddMethodProvider(mp)
}
