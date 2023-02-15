package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	output     = flag.String("o", "", "set output file path")
	pkgName    = flag.String("pkg", "icall", "set package name")
	presetSize = flag.Int("size", 0, "set methods preset size")
)

func main() {
	flag.Parse()
	if *output == "" || *pkgName == "" || *presetSize == 0 {
		flag.Usage()
		return
	}
	// write icall.go
	err := writeFile(*output, *pkgName, *presetSize)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	// write icall_regabi.go
	err = writeRegAbi(*output, *pkgName, *presetSize)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func writeFile(filename string, pkgName string, size int) error {
	dir, _ := filepath.Split(filename)
	if dir != "" {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			return fmt.Errorf("make dir %v error: %v", dir, err)
		}
	}

	var buf bytes.Buffer
	r := strings.NewReplacer("$pkgname", pkgName, "$max_size", strconv.Itoa(size))
	buf.WriteString(r.Replace(head))

	fnWrite := func(name string, t string) {
		buf.WriteString(fmt.Sprintf("\nvar %v = []interface{}{\n", name))
		for i := 0; i < size; i++ {
			r := strings.NewReplacer("$index", strconv.Itoa(i))
			buf.WriteString(r.Replace(t))
		}
		buf.WriteString("}\n")
	}
	fnWrite("icall_array", templ_fn)
	return ioutil.WriteFile(filename, buf.Bytes(), 0666)
}

var head = `//go:build (!go1.17 || (go1.17 && !go1.18 && !goexperiment.regabireflect) || (go1.18 && !go1.19 && !goexperiment.regabireflect && !amd64) || (go1.19 && !go1.20 && !goexperiment.regabiargs && !amd64 && !arm64 && !ppc64 && !ppc64le) || (go1.20 && !goexperiment.regabiargs && !amd64 && !arm64 && !ppc64 && !ppc64le && !riscv64)) && (!js || (js && wasm))
// +build !go1.17 go1.17,!go1.18,!goexperiment.regabireflect go1.18,!go1.19,!goexperiment.regabireflect,!amd64 go1.19,!go1.20,!goexperiment.regabiargs,!amd64,!arm64,!ppc64,!ppc64le go1.20,!goexperiment.regabiargs,!amd64,!arm64,!ppc64,!ppc64le,!riscv64
// +build !js js,wasm

package $pkgname

import (
	"reflect"
	"unsafe"

	"github.com/goplus/reflectx"
)

const capacity = $max_size

type provider struct {
	infos []*reflectx.MethodInfo
}

func (p *provider) Push(info *reflectx.MethodInfo) (ifn unsafe.Pointer) {
	fn := icall_array[len(p.infos)]
	p.infos = append(p.infos, info)
	return unsafe.Pointer(reflect.ValueOf(fn).Pointer())
}

func (p *provider) Len() int {
	return len(p.infos)
}

func (p *provider) Cap() int {
	return len(icall_array)
}

func (p *provider) Clear() {
	p.infos = nil
}

var (
	mp provider
)

func init() {
	reflectx.AddMethodProvider(&mp)
}

func i_x(index int, ptr unsafe.Pointer, p unsafe.Pointer) {
	info := mp.infos[index]
	var receiver reflect.Value
	if !info.Pointer && info.OnePtr {
		receiver = reflect.NewAt(info.Type, unsafe.Pointer(&ptr)).Elem() //.Elem().Field(0)
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
`

var templ_fn = `	func(p, a unsafeptr) { i_x($index, p, unsafeptr(&a)) },
`
