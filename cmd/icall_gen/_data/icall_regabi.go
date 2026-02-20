//go:build (go1.17 && goexperiment.regabireflect) || (go1.19 && goexperiment.regabiargs) || (go1.18 && amd64) || (go1.19 && arm64) || (go1.19 && ppc64) || (go1.19 && ppc64le) || (go1.20 && riscv64) || (go1.23 && loong64)
// +build go1.17,goexperiment.regabireflect go1.19,goexperiment.regabiargs go1.18,amd64 go1.19,arm64 go1.19,ppc64 go1.19,ppc64le go1.20,riscv64 go1.23,loong64

package $pkgname

import (
	"reflect"
	"unsafe"

	"github.com/goplus/reflectx/abi"
)

const capacity = $max_size

type methodUsed struct {
	fun reflect.Value
	ptr unsafe.Pointer
}

type provider struct {
	used map[int]*methodUsed
}

func i_x(c unsafe.Pointer, frame unsafe.Pointer, retValid *bool, r unsafe.Pointer, index int) {
	ptr := mp.used[index].ptr
	moveMakeFuncArgPtrs(ptr, r)
	callReflect(ptr, frame, retValid, r)
}

func spillArgs()
func unspillArgs()

func (p *provider) Insert(info *abi.MethodInfo) (unsafe.Pointer, int) {
	var index = -1
	for i := 0; i < capacity; i++ {
		if _, ok := p.used[i]; !ok {
			index = i
			break
		}
	}
	if index == -1 {
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
	icall := icall_fn[index]
	return unsafe.Pointer(reflect.ValueOf(icall).Pointer()), index
}

func (p *provider) Remove(indexs []int) {
	for _, n := range indexs {
		delete(p.used, n)
	}
}

func (p *provider) Available() int {
	return capacity - len(p.used)
}

func (p *provider) Used() int {
	return len(p.used)
}

func (p *provider) Cap() int {
	return capacity
}

func (p *provider) Clear() {
	p.used = make(map[int]*methodUsed)
}

var (
	mp = &provider{
		used: make(map[int]*methodUsed),
	}
)

func init() {
	abi.AddMethodProvider(mp)
}
