//go:build !go1.17 || (go1.17 && !go1.18 && !goexperiment.regabireflect) || (go1.18 && !go1.19 && !goexperiment.regabireflect && !amd64) || (go1.19 && !go1.20 && !goexperiment.regabiargs && !amd64 && !arm64 && !ppc64 && !ppc64le) || (go1.20 && !goexperiment.regabiargs && !amd64 && !arm64 && !ppc64 && !ppc64le && !riscv64) || (go1.22 && !goexperiment.regabiargs && !amd64 && !arm64 && !ppc64 && !ppc64le && !riscv64 && !loong64)
// +build !go1.17 go1.17,!go1.18,!goexperiment.regabireflect go1.18,!go1.19,!goexperiment.regabireflect,!amd64 go1.19,!go1.20,!goexperiment.regabiargs,!amd64,!arm64,!ppc64,!ppc64le go1.20,!goexperiment.regabiargs,!amd64,!arm64,!ppc64,!ppc64le,!riscv64 go1.22,!goexperiment.regabiargs,!amd64,!arm64,!ppc64,!ppc64le,!riscv64,!loong64

package $pkgname

import (
	"reflect"
	"unsafe"

	"github.com/goplus/reflectx/abi"
)

const capacity = $max_size

type provider struct {
	used map[int]*abi.MethodInfo
	free []int
}

func (p *provider) Insert(info *abi.MethodInfo) (ifn unsafe.Pointer, index int) {
	if len(p.free) == 0 {
		return nil, -1
	}
	index = p.free[len(p.free)-1]
	p.free = p.free[:len(p.free)-1]
	p.used[index] = info
	fn := icall_array[index]
	return unsafe.Pointer(reflect.ValueOf(fn).Pointer()), index
}

func (p *provider) Available() int {
	return len(p.free)
}

func (p *provider) Remove(indexs []int) {
	for _, n := range indexs {
		if _, ok := p.used[n]; !ok {
			continue // already freed; skip to avoid duplicating in p.free
		}
		delete(p.used, n)
		p.free = append(p.free, n)
	}
}

func (p *provider) Used() int {
	return len(p.used)
}

func (p *provider) Cap() int {
	return len(icall_array)
}

func (p *provider) Clear() {
	p.used = make(map[int]*abi.MethodInfo)
	p.free = initFreeList()
}

func initFreeList() []int {
	free := make([]int, capacity)
	for i := range free {
		free[i] = capacity - 1 - i
	}
	return free
}

var (
	mp = &provider{
		used: make(map[int]*abi.MethodInfo),
		free: initFreeList(),
	}
)

func init() {
	abi.AddMethodProvider(mp)
}

func i_x(index int, ptr unsafe.Pointer, p unsafe.Pointer) {
	info := mp.used[index]
	var receiver reflect.Value
	if !info.Pointer && info.OnePtr {
		receiver = reflect.NewAt(info.Type, unsafe.Pointer(&ptr)).Elem()
	} else {
		receiver = reflect.NewAt(info.Type, ptr)
		if !info.Pointer || info.Indirect {
			receiver = receiver.Elem()
		}
	}
	in := []reflect.Value{receiver}
	if inCount := info.Func.Type().NumIn(); inCount > 1 {
		sz := info.InTyp.Size()
		buf := make([]byte, sz, sz)
		if sz > info.InSize {
			sz = info.InSize
		}
		for i := uintptr(0); i < sz; i++ {
			buf[i] = *(*byte)(add(p, i, ""))
		}
		var inArgs reflect.Value
		if sz == 0 {
			inArgs = reflect.New(info.InTyp).Elem()
		} else {
			inArgs = reflect.NewAt(info.InTyp, unsafe.Pointer(&buf[0])).Elem()
		}
		for i := 1; i < inCount; i++ {
			in = append(in, inArgs.Field(i-1))
		}
	}
	var r []reflect.Value
	if info.Variadic {
		r = info.Func.CallSlice(in)
	} else {
		r = info.Func.Call(in)
	}
	if info.OutTyp.NumField() > 0 {
		out := reflect.New(info.OutTyp).Elem()
		for i, v := range r {
			out.Field(i).Set(v)
		}
		po := unsafe.Pointer(out.UnsafeAddr())
		for i := uintptr(0); i < info.OutSize; i++ {
			*(*byte)(add(p, info.InSize+i, "")) = *(*byte)(add(po, uintptr(i), ""))
		}
	}
}

func add(p unsafe.Pointer, x uintptr, whySafe string) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

type unsafeptr = unsafe.Pointer
