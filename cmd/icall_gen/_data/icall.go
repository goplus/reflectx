//go:build !goexperiment.regabiargs && !amd64 && !arm64 && !ppc64 && !ppc64le && !riscv64 && !(go1.23 && loong64)
// +build !goexperiment.regabiargs
// +build !amd64
// +build !arm64
// +build !ppc64
// +build !ppc64le
// +build !riscv64
// +build !go1.23 !loong64

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
	used []atomic.Pointer[abi.MethodInfo]
	free []int
	n    atomic.Int64
}

func (p *provider) Insert(info *abi.MethodInfo) (ifn unsafe.Pointer, index int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := len(p.free)
	if n == 0 {
		return nil, -1
	}
	index = p.free[n-1]
	p.free = p.free[:n-1]
	p.used[index].Store(info)
	p.n.Add(1)
	fn := icall_array[index]
	return unsafe.Pointer(reflect.ValueOf(fn).Pointer()), index
}

func (p *provider) Available() int {
	return capacity - int(p.n.Load())
}

func (p *provider) Remove(indexs []int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, n := range indexs {
		if n < 0 || n >= capacity {
			continue
		}
		if p.used[n].Swap(nil) != nil {
			p.free = append(p.free, n)
			p.n.Add(-1)
		}
	}
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
	p := &provider{used: make([]atomic.Pointer[abi.MethodInfo], capacity)}
	p.resetFree()
	return p
}

var mp = newProvider()

func init() {
	abi.AddMethodProvider(mp)
}

func i_x(index int, ptr unsafe.Pointer, p unsafe.Pointer) {
	info := mp.used[index].Load()
	var receiver reflect.Value
	if !info.Pointer && info.OnePtr {
		receiver = reflect.NewAt(info.Type, unsafe.Pointer(&ptr)).Elem()
	} else {
		receiver = reflect.NewAt(info.Type, ptr)
		if !info.Pointer || info.Indirect {
			receiver = receiver.Elem()
		}
	}
	inCount := info.Func.Type().NumIn()
	in := make([]reflect.Value, inCount)
	in[0] = receiver
	if inCount > 1 {
		sz := info.InTyp.Size()
		var inArgs reflect.Value
		if sz == 0 {
			inArgs = reflect.New(info.InTyp).Elem()
		} else {
			buf := make([]byte, sz)
			if sz > info.InSize {
				sz = info.InSize
			}
			copy(buf, unsafe.Slice((*byte)(p), sz))
			inArgs = reflect.NewAt(info.InTyp, unsafe.Pointer(&buf[0])).Elem()
		}
		for i := 1; i < inCount; i++ {
			in[i] = inArgs.Field(i - 1)
		}
	}
	r := info.Call(in)
	if len(r) != info.OutTyp.NumField() {
		panic("reflect: wrong return count from function created by MakeFunc")
	}
	if info.OutTyp.NumField() > 0 {
		out := reflect.New(info.OutTyp).Elem()
		for i, v := range r {
			out.Field(i).Set(v)
		}
		po := unsafe.Pointer(out.UnsafeAddr())
		copy(unsafe.Slice((*byte)(add(p, info.InSize, "")), info.OutSize), unsafe.Slice((*byte)(po), info.OutSize))
	}
}

func add(p unsafe.Pointer, x uintptr, whySafe string) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

type unsafeptr = unsafe.Pointer
