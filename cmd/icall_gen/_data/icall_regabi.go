//go:build goexperiment.regabiargs || amd64 || arm64 || ppc64 || ppc64le || riscv64 || (go1.23 && loong64)
// +build goexperiment.regabiargs amd64 arm64 ppc64 ppc64le riscv64 go1.23,loong64

package pkgname

import (
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/goplus/reflectx/abi"
)

const capacity = 1024

type provider struct {
	mu   sync.Mutex
	used []unsafe.Pointer
	free []int
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
	p.mu.Lock()
	defer p.mu.Unlock()
	n := len(p.free)
	if n == 0 {
		return nil, -1
	}
	index := p.free[n-1]
	p.free = p.free[:n-1]
	atomic.StorePointer(&p.used[index], ptr)
	p.n.Add(1)
	icall := icall_fn[index]
	return unsafe.Pointer(reflect.ValueOf(icall).Pointer()), index
}

func (p *provider) Remove(indexs []int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, n := range indexs {
		if n < 0 || n >= capacity {
			continue
		}
		if atomic.SwapPointer(&p.used[n], nil) != nil {
			p.free = append(p.free, n)
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
	p.mu.Lock()
	defer p.mu.Unlock()
	clear(p.used)
	p.resetFree()
	p.n.Store(0)
}

func (p *provider) resetFree() {
	if cap(p.free) < capacity {
		p.free = make([]int, capacity)
	} else {
		p.free = p.free[:capacity]
	}
	for i := range p.free {
		p.free[i] = capacity - 1 - i
	}
}

func newProvider() *provider {
	p := &provider{used: make([]unsafe.Pointer, capacity)}
	p.resetFree()
	return p
}

var mp = newProvider()

func init() {
	abi.AddMethodProvider(mp)
}
