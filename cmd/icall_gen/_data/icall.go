//go:build !goexperiment.regabiargs && !amd64 && !arm64 && !ppc64 && !ppc64le && !riscv64 && !(go1.23 && loong64)
// +build !goexperiment.regabiargs
// +build !amd64
// +build !arm64
// +build !ppc64
// +build !ppc64le
// +build !riscv64
// +build !go1.23 !loong64

package $pkgname

import (
	"reflect"
	"unsafe"

	"github.com/goplus/reflectx/abi"
)

const capacity = $max_size

type provider struct {
	used []*abi.MethodInfo
	n    int
}

func (p *provider) Insert(info *abi.MethodInfo) (ifn unsafe.Pointer, index int) {
	for i := 0; i < capacity; i++ {
		if p.used[i] == nil {
			p.n++
			p.used[i] = info
			fn := icall_array[i]
			return unsafe.Pointer(reflect.ValueOf(fn).Pointer()), i
		}
	}
	return nil, -1
}

func (p *provider) Available() int {
	return capacity - p.n
}

func (p *provider) Remove(indexs []int) {
	for _, n := range indexs {
		if n < capacity && p.used[n] != nil {
			p.used[n] = nil
			p.n--
		}
	}
}

func (p *provider) Used() int {
	return p.n
}

func (p *provider) Cap() int {
	return capacity
}

func (p *provider) Clear() {
	clear(p.used)
	p.n = 0
}

var (
	mp = &provider{
		used: make([]*abi.MethodInfo, capacity),
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