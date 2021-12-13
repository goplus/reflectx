//go:build !js || (js && wasm)
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
	MethodList []*MethodInfo
)

func PresetMethods(array []interface{}) {
	icall_array = array
}

func icall(i int) interface{} {
	if i >= len(icall_array) {
		log.Printf("github.com/goplus/reflectx: must increase preset, cannot alloc method %v > %v\n", i, len(icall_array))
		return nil
	}
	return icall_array[i]
}

var (
	methodType  = make(map[reflect.Type]bool)
	icall_array []interface{}
)

func resetMethodList() {
	MethodList = nil
	methodType = make(map[reflect.Type]bool)
}

// register method info
func registerMethod(info *MethodInfo) (ifn unsafe.Pointer) {
	fn := icall(len(MethodList))
	if fn == nil {
		return nil
	}
	MethodList = append(MethodList, info)
	return unsafe.Pointer(reflect.ValueOf(fn).Pointer())
}

func isMethod(typ reflect.Type) (ok bool) {
	return methodType[typ]
}

type MethodInfo struct {
	Func     reflect.Value
	Type     reflect.Type
	InTyp    reflect.Type
	OutTyp   reflect.Type
	InSize   uintptr
	OutSize  uintptr
	Pointer  bool
	Indirect bool
	Variadic bool
	OnePtr   bool
}

func MethodByIndex(typ reflect.Type, index int) reflect.Method {
	//m := typ.Method(index)
	m := totype(typ).MethodX(index)
	if isMethod(typ) {
		tovalue(&m.Func).flag |= flagIndir
	}
	return m
}

func MethodByName(typ reflect.Type, name string) (m reflect.Method, ok bool) {
	//m, ok = typ.MethodByName(name)
	m, ok = totype(typ).MethodByNameX(name)
	if !ok {
		return
	}
	if isMethod(typ) {
		tovalue(&m.Func).flag |= flagIndir
	}
	return
}

func resizeMethod(typ reflect.Type, mcount int, xcount int) error {
	rt := totype(typ)
	ut := toUncommonType(rt)
	if ut == nil {
		return fmt.Errorf("not found uncommonType of %v", typ)
	}
	if uint16(mcount) > ut.mcount {
		return fmt.Errorf("too many methods of %v", typ)
	}
	ut.xcount = uint16(xcount)
	return nil
}

func createMethod(typ reflect.Type, ptyp reflect.Type, m Method, index int) (mfn reflect.Value, inTyp, outTyp reflect.Type, mtyp typeOff, tfn, ptfn textOff) {
	var in []reflect.Type
	var out []reflect.Type
	var ntyp reflect.Type
	in, out, ntyp, inTyp, outTyp = parserMethodType(m.Type, nil)
	mtyp = resolveReflectType(totype(ntyp))
	var ftyp reflect.Type
	if m.Pointer {
		ftyp = reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, m.Type.IsVariadic())
	} else {
		ftyp = reflect.FuncOf(append([]reflect.Type{typ}, in...), out, m.Type.IsVariadic())
	}

	mfn = reflect.MakeFunc(ftyp, m.Func)
	ptr := tovalue(&mfn).ptr

	tfn = resolveReflectText(unsafe.Pointer(ptr))
	if !m.Pointer {
		ctyp := reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, m.Type.IsVariadic())
		cv := reflect.MakeFunc(ctyp, func(args []reflect.Value) (results []reflect.Value) {
			return args[0].Elem().Method(index).Call(args[1:])
		})
		ptfn = resolveReflectText(tovalue(&cv).ptr)
	} else {
		ptfn = tfn
	}
	return
}

func setMethodSet(typ reflect.Type, methods []Method) error {
	sort.Slice(methods, func(i, j int) bool {
		n := strings.Compare(methods[i].Name, methods[j].Name)
		if n == 0 && methods[i].Type == methods[j].Type {
			panic(fmt.Sprintf("method redeclared: %v", methods[j].Name))
		}
		return n < 0
	})
	var mcount, pcount int
	var xcount, pxcount int
	pcount = len(methods)
	var mlist []string
	for _, m := range methods {
		isexport := methodIsExported(m.Name)
		if isexport {
			pxcount++
		}
		if !m.Pointer {
			if isexport {
				xcount++
			}
			mlist = append(mlist, m.Name)
			mcount++
		}
	}
	ptyp := reflect.PtrTo(typ)
	if err := resizeMethod(typ, mcount, xcount); err != nil {
		return err
	}
	if err := resizeMethod(ptyp, pcount, pxcount); err != nil {
		return err
	}
	rt := totype(typ)
	prt := totype(ptyp)

	ms := rt.methods()
	pms := prt.methods()

	var index int
	for i, m := range methods {
		isexport := methodIsExported(m.Name)
		nm := newNameEx(m.Name, "", isexport, !isexport)
		mname := resolveReflectName(nm)
		if !isexport {
			nm.setPkgPath(resolveReflectName(newName(m.PkgPath, "", false)))
		}
		mfn, inTyp, outTyp, mtyp, tfn, ptfn := createMethod(typ, ptyp, m, index)
		isz := argsTypeSize(inTyp, true)
		osz := argsTypeSize(outTyp, false)
		onePtr := checkOneFieldPtr(typ) || typ.Kind() == reflect.Func
		pinfo := &MethodInfo{
			Type:     typ,
			Func:     mfn,
			InTyp:    inTyp,
			OutTyp:   outTyp,
			InSize:   isz,
			OutSize:  osz,
			Pointer:  true,
			Indirect: !m.Pointer,
			Variadic: m.Type.IsVariadic(),
			OnePtr:   onePtr,
		}
		pifn := registerMethod(pinfo)
		pms[i].name = mname
		pms[i].mtyp = mtyp
		pms[i].tfn = ptfn
		pms[i].ifn = resolveReflectText(pifn)

		if !m.Pointer {
			info := &MethodInfo{
				Type:     typ,
				Func:     mfn,
				InTyp:    inTyp,
				OutTyp:   outTyp,
				InSize:   isz,
				OutSize:  osz,
				Variadic: m.Type.IsVariadic(),
				OnePtr:   onePtr,
			}
			ifn := registerMethod(info)
			ms[index].name = mname
			ms[index].mtyp = mtyp
			ms[index].tfn = tfn
			ms[index].ifn = resolveReflectText(ifn)
			index++
		}
	}
	methodType[typ] = true
	methodType[ptyp] = true
	return nil
}

func newMethodSet(styp reflect.Type, maxmfunc, maxpfunc int) reflect.Type {
	rt, _ := newType("", "", styp, maxmfunc, 0)
	prt, _ := newType("", "", reflect.PtrTo(styp), maxpfunc, 0)
	rt.ptrToThis = resolveReflectType(prt)
	(*ptrType)(unsafe.Pointer(prt)).elem = rt
	setTypeName(rt, styp.PkgPath(), styp.Name())
	prt.uncommon().pkgPath = resolveReflectName(newName(styp.PkgPath(), "", false))
	typ := toType(rt)
	if nt, ok := ntypeMap[styp]; ok {
		ntypeMap[typ] = &Named{Name: nt.Name, PkgPath: nt.PkgPath, Type: typ, From: nt.From, Kind: nt.Kind}
	}
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

type iparam struct {
	data []byte
}

// func i_y(itype int, index int, ptr unsafeptr, pin iparam, ptrto bool) (pout iparam) {
// 	typ := itypList[itype]
// 	otyp := typ
// 	if ptrto {
// 		typ = reflect.PtrTo(typ)
// 	}
// 	infos, ok := typInfoMap[typ]
// 	if !ok {
// 		log.Panicln("cannot found type info", typ)
// 		return
// 	}
// 	info := infos[index]
// 	var method reflect.Method
// 	if ptrto && !info.pointer {
// 		method.Func = info.Func
// 	} else {
// 		method = MethodByIndex(typ, info.index)
// 		method.Func = info.Func
// 	}
// 	var receiver reflect.Value
// 	if !ptrto && info.onePtr {
// 		receiver = reflect.NewAt(otyp, unsafe.Pointer(&ptr)).Elem() //.Elem().Field(0)
// 	} else {
// 		receiver = reflect.NewAt(otyp, ptr)
// 		if !ptrto || !info.pointer {
// 			receiver = receiver.Elem()
// 		}
// 	}
// 	in := []reflect.Value{receiver}
// 	if inCount := method.Func.Type().NumIn(); inCount > 1 {
// 		sz := info.inTyp.Size()
// 		var inArgs reflect.Value
// 		if sz == 0 {
// 			inArgs = reflect.New(info.inTyp).Elem()
// 		} else {
// 			inArgs = reflect.NewAt(info.inTyp, unsafe.Pointer(&pin)).Elem()
// 		}
// 		if info.variadic {
// 			for i := 1; i < inCount-1; i++ {
// 				in = append(in, inArgs.Field(i-1))
// 			}
// 			slice := inArgs.Field(inCount - 2)
// 			for i := 0; i < slice.Len(); i++ {
// 				in = append(in, slice.Index(i))
// 			}
// 		} else {
// 			for i := 1; i < inCount; i++ {
// 				in = append(in, inArgs.Field(i-1))
// 			}
// 		}
// 	}
// 	r := method.Func.Call(in)
// 	if info.outTyp.NumField() > 0 {
// 		out := reflect.New(info.outTyp).Elem()
// 		for i, v := range r {
// 			out.Field(i).Set(v)
// 		}
// 		pout.data = make([]byte, info.osz, info.osz)
// 		memmove(unsafe.Pointer(&pout), unsafe.Pointer(out.UnsafeAddr()), info.osz)
// 		// po := unsafe.Pointer(out.UnsafeAddr())
// 		// p := unsafe.Pointer(&pout)
// 		// for i := uintptr(0); i < info.osz; i++ {
// 		// 	*(*byte)(add(p, i, "")) = *(*byte)(add(po, uintptr(i), ""))
// 		// }
// 	}
// 	return
// }

func i_x(index int, ptr unsafe.Pointer, p unsafe.Pointer) {
	info := MethodList[index]
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
		if info.Variadic {
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
	r := info.Func.Call(in)
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

// func i_x_dyn(i int, ptr unsafe.Pointer, p unsafe.Pointer, ptrto bool) bool {
// 	var receiver reflect.Value
// 	var typ reflect.Type
// 	if !ptrto {
// 		for v, t := range valueInfoMap {
// 			if t.oneFieldPtr {
// 				if ptr == unsafe.Pointer(*(**uintptr)(tovalue(&v).ptr)) {
// 					receiver = v
// 					typ = t.typ
// 					break
// 				}
// 			}
// 		}
// 	}
// 	if typ == nil {
// 		for v, t := range valueInfoMap {
// 			if ptr == tovalue(&v).ptr {
// 				receiver = v
// 				typ = t.typ
// 				break
// 			}
// 		}
// 	}
// 	if typ == nil {
// 		log.Panicln("cannot found ptr type", i, ptr)
// 		return false
// 	}
// 	if ptrto {
// 		typ = reflect.PtrTo(typ)
// 	}
// 	infos, ok := typInfoMap[typ]
// 	if !ok {
// 		log.Panicln("cannot found type info", typ)
// 		return false
// 	}
// 	info := infos[i]
// 	var method reflect.Method
// 	if ptrto && !info.pointer {
// 		method = MethodByIndex(typ.Elem(), info.index)
// 	} else {
// 		method = MethodByIndex(typ, info.index)
// 	}
// 	if ptrto && info.pointer {
// 		receiver = receiver.Addr()
// 	}
// 	in := []reflect.Value{receiver}
// 	if inCount := method.Type.NumIn(); inCount > 1 {
// 		sz := info.inTyp.Size()
// 		buf := make([]byte, sz, sz)
// 		if sz > info.isz {
// 			sz = info.isz
// 		}
// 		for i := uintptr(0); i < sz; i++ {
// 			buf[i] = *(*byte)(add(p, i, ""))
// 		}
// 		var inArgs reflect.Value
// 		if sz == 0 {
// 			inArgs = reflect.New(info.inTyp).Elem()
// 		} else {
// 			inArgs = reflect.NewAt(info.inTyp, unsafe.Pointer(&buf[0])).Elem()
// 		}
// 		if info.variadic {
// 			for i := 1; i < inCount-1; i++ {
// 				in = append(in, inArgs.Field(i-1))
// 			}
// 			slice := inArgs.Field(inCount - 2)
// 			for i := 0; i < slice.Len(); i++ {
// 				in = append(in, slice.Index(i))
// 			}
// 		} else {
// 			for i := 1; i < inCount; i++ {
// 				in = append(in, inArgs.Field(i-1))
// 			}
// 		}
// 	}
// 	r := method.Func.Call(in)
// 	if info.outTyp.NumField() > 0 {
// 		out := reflect.New(info.outTyp).Elem()
// 		for i, v := range r {
// 			out.Field(i).Set(v)
// 		}
// 		po := unsafe.Pointer(out.UnsafeAddr())
// 		for i := uintptr(0); i < info.osz; i++ {
// 			*(*byte)(add(p, info.isz+i, "")) = *(*byte)(add(po, uintptr(i), ""))
// 		}
// 	}
// 	return true
// }
