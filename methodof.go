// +build !js js,wasm

package reflectx

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"unsafe"
)

var (
	typInfoMap   = make(map[reflect.Type][]*methodInfo)
	valueInfoMap = make(map[reflect.Value]typeInfo)
)

func isMethod(typ reflect.Type) (ok bool) {
	_, ok = typInfoMap[typ]
	return
}

type typeInfo struct {
	typ         reflect.Type
	oneFieldPtr bool
}

type methodInfo struct {
	inTyp    reflect.Type
	outTyp   reflect.Type
	name     string
	index    int
	isz      uintptr
	osz      uintptr
	pointer  bool
	variadic bool
}

func MethodByIndex(typ reflect.Type, index int) reflect.Method {
	m := typ.Method(index)
	if isMethod(typ) {
		tovalue(&m.Func).flag |= flagIndir
	}
	return m
}

func MethodByName(typ reflect.Type, name string) (m reflect.Method, ok bool) {
	m, ok = typ.MethodByName(name)
	if !ok {
		return
	}
	if isMethod(typ) {
		tovalue(&m.Func).flag |= flagIndir
	}
	return
}

func checkStoreMethodValue(v reflect.Value) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.IsValid() {
		return
	}
	typ := v.Type()
	if isMethod(typ) {
		valueInfoMap[v] = typeInfo{typ, checkOneFieldPtr(typ)}
	}
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			sf := v.Field(i)
			checkStoreMethodValue(sf)
		}
	}
}

func New(typ reflect.Type) reflect.Value {
	v := reflect.New(typ)
	checkStoreMethodValue(v)
	return v
}

func methodOf(styp reflect.Type, methods []Method) reflect.Type {
	sort.Slice(methods, func(i, j int) bool {
		n := strings.Compare(methods[i].Name, methods[j].Name)
		if n == 0 && methods[i].Type == methods[j].Type {
			panic(fmt.Sprintf("method redeclared: %v", methods[j].Name))
		}
		return n < 0
	})
	var mcount, pcount int
	pcount = len(methods)
	var mlist []string
	for _, m := range methods {
		if !m.Pointer {
			mlist = append(mlist, m.Name)
			mcount++
		}
	}
	orgtyp := styp
	rt, tt := newType("", "", styp, mcount, mcount)
	prt, ptt := newType("", "", reflect.PtrTo(styp), pcount, pcount)
	rt.ptrToThis = resolveReflectType(prt)

	(*ptrType)(unsafe.Pointer(prt)).elem = rt
	setTypeName(rt, styp.PkgPath(), styp.Name())
	typ := toType(rt)
	ptyp := reflect.PtrTo(typ)
	ms := make([]method, mcount, mcount)
	pms := make([]method, pcount, pcount)
	var infos []*methodInfo
	var pinfos []*methodInfo
	var index int
	for i, m := range methods {
		name := resolveReflectName(newName(m.Name, "", true))
		in, out, ntyp, inTyp, outTyp := toRealType(typ, orgtyp, m.Type)
		mtyp := resolveReflectType(totype(ntyp))
		var ftyp reflect.Type
		if m.Pointer {
			ftyp = reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, m.Type.IsVariadic())
		} else {
			ftyp = reflect.FuncOf(append([]reflect.Type{typ}, in...), out, m.Type.IsVariadic())
		}

		mfn := reflect.MakeFunc(ftyp, m.Func)
		ptr := tovalue(&mfn).ptr

		sz := int(inTyp.Size())
		ifunc := icall(i, true)
		var pifn, tfn, ptfn textOff
		if ifunc == nil {
			log.Printf("warning cannot wrapper method index:%v, size: %v\n", i, sz)
		} else {
			pifn = resolveReflectText(unsafe.Pointer(reflect.ValueOf(ifunc).Pointer()))
		}
		tfn = resolveReflectText(unsafe.Pointer(ptr))
		pindex := i
		if !m.Pointer {
			for i, s := range mlist {
				if s == m.Name {
					pindex = i
					break
				}
			}
			ctyp := reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, m.Type.IsVariadic())
			cv := reflect.MakeFunc(ctyp, func(args []reflect.Value) (results []reflect.Value) {
				return args[0].Elem().Method(pindex).Call(args[1:])
			})
			ptfn = resolveReflectText(tovalue(&cv).ptr)
		} else {
			ptfn = tfn
		}
		isz := argsTypeSize(inTyp, true)
		osz := argsTypeSize(outTyp, false)

		pms[i].name = name
		pms[i].mtyp = mtyp
		pms[i].tfn = ptfn
		pms[i].ifn = pifn
		pinfos = append(pinfos, &methodInfo{
			inTyp:    inTyp,
			outTyp:   outTyp,
			name:     m.Name,
			index:    pindex,
			isz:      isz,
			osz:      osz,
			pointer:  m.Pointer,
			variadic: m.Type.IsVariadic(),
		})
		if !m.Pointer {
			ifunc := icall(index, false)
			var ifn textOff
			if ifunc == nil {
				log.Printf("warning cannot wrapper method index:%v, size: %v\n", i, sz)
			} else {
				ifn = resolveReflectText(unsafe.Pointer(reflect.ValueOf(ifunc).Pointer()))
			}
			ms[index].name = name
			ms[index].mtyp = mtyp
			ms[index].tfn = tfn
			ms[index].ifn = ifn
			infos = append(infos, &methodInfo{
				inTyp:    inTyp,
				outTyp:   outTyp,
				name:     m.Name,
				index:    index,
				isz:      isz,
				osz:      osz,
				pointer:  m.Pointer,
				variadic: m.Type.IsVariadic(),
			})
			index++
		}
	}
	copy(tt, ms)
	copy(ptt, pms)
	typInfoMap[typ] = infos
	typInfoMap[ptyp] = pinfos
	return typ
}

func checkOneFieldPtr(typ reflect.Type) bool {
	return typ.Kind() == reflect.Struct &&
		typ.NumField() == 1 &&
		typ.Field(0).Type.Kind() == reflect.Ptr
}

const (
	uintptrAligin = unsafe.Sizeof(uintptr(0))
)

func argsTypeSize(typ reflect.Type, offset bool) (off uintptr) {
	numIn := typ.NumField()
	if numIn == 0 {
		return 0
	}
	for i := 0; i < numIn; i++ {
		t := typ.Field(i).Type
		targ := totype(t)
		a := uintptr(targ.align)
		off = (off + a - 1) &^ (a - 1)
		n := targ.size
		if n == 0 {
			continue
		}
		off += n
	}
	if offset {
		off = (off + uintptrAligin - 1) &^ (uintptrAligin - 1)
		if off == 0 {
			return uintptrAligin
		}
	}
	return
}

func i_x(i int, ptr unsafe.Pointer, p unsafe.Pointer, ptrto bool) bool {
	var receiver reflect.Value
	var typ reflect.Type
	if !ptrto {
		for v, t := range valueInfoMap {
			if t.oneFieldPtr {
				if ptr == unsafe.Pointer(*(**uintptr)(tovalue(&v).ptr)) {
					receiver = v
					typ = t.typ
					break
				}
			}
		}
	}
	if typ == nil {
		for v, t := range valueInfoMap {
			if ptr == tovalue(&v).ptr {
				receiver = v
				typ = t.typ
				break
			}
		}
	}
	if typ == nil {
		log.Panicln("cannot found ptr type", i, ptr)
		return false
	}
	if ptrto {
		typ = reflect.PtrTo(typ)
	}
	infos, ok := typInfoMap[typ]
	if !ok {
		log.Panicln("cannot found type info", typ)
		return false
	}
	info := infos[i]
	var method reflect.Method
	if ptrto && !info.pointer {
		method = MethodByIndex(typ.Elem(), info.index)
	} else {
		method = MethodByIndex(typ, info.index)
	}
	if ptrto && info.pointer {
		receiver = receiver.Addr()
	}
	in := []reflect.Value{receiver}
	if inCount := method.Type.NumIn(); inCount > 1 {
		sz := info.inTyp.Size()
		buf := make([]byte, sz, sz)
		if sz > info.isz {
			sz = info.isz
		}
		for i := uintptr(0); i < sz; i++ {
			buf[i] = *(*byte)(add(p, i, ""))
		}
		var inArgs reflect.Value
		if sz == 0 {
			inArgs = reflect.New(info.inTyp).Elem()
		} else {
			inArgs = reflect.NewAt(info.inTyp, unsafe.Pointer(&buf[0])).Elem()
		}
		if info.variadic {
			for i := 1; i < inCount-1; i++ {
				in = append(in, inArgs.Field(i-1))
			}
			slice := inArgs.Field(inCount - 2)
			for i := 0; i < slice.Len(); i++ {
				in = append(in, slice.Index(i))
			}
		} else {
			for i := 1; i < inCount; i++ {
				in = append(in, inArgs.Field(i-1))
			}
		}
	}
	r := method.Func.Call(in)
	if info.outTyp.NumField() > 0 {
		out := reflect.New(info.outTyp).Elem()
		for i, v := range r {
			out.Field(i).Set(v)
		}
		po := unsafe.Pointer(out.UnsafeAddr())
		for i := uintptr(0); i < info.osz; i++ {
			*(*byte)(add(p, info.isz+i, "")) = *(*byte)(add(po, uintptr(i), ""))
		}
	}
	return true
}
