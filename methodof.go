// +build !js js,wasm

package reflectx

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"sync"
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
	onePtr   bool
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

func resizeMethod(typ reflect.Type, count int) error {
	rt := totype(typ)
	ut := toUncommonType(rt)
	if ut == nil {
		return fmt.Errorf("not found uncommonType of %v", typ)
	}
	if uint16(count) > ut.mcount {
		return fmt.Errorf("too many methods of %v", typ)
	}
	ut.xcount = uint16(count)
	return nil
}

func updateMethod(typ reflect.Type, methods []Method, rmap map[reflect.Type]reflect.Type) bool {
	ptyp := reflect.PtrTo(typ)
	pinfos, ok := typInfoMap[ptyp]
	if !ok && ptyp.NumMethod() > 0 {
		log.Printf("warning cannot found type info: %v\n", ptyp)
		return false
	}
	infos, ok := typInfoMap[typ]
	if !ok && typ.NumMethod() > 0 {
		log.Printf("warning cannot found type info: %v\n", typ)
		return false
	}
	rt := totype(typ)
	prt := totype(ptyp)
	ms := toUncommonType(rt).exportedMethods()
	pms := toUncommonType(prt).exportedMethods()
	itype := itypeIndex(typ)
	for _, m := range methods {
		var i int
		var index int
		f, ok := ptyp.MethodByName(m.Name)
		if !ok {
			log.Printf("warning cannot found method: (%v).%v\n", ptyp, m.Name)
			continue
		}
		i = f.Index
		if !m.Pointer {
			f, ok := typ.MethodByName(m.Name)
			if !ok {
				log.Printf("warning cannot found method: (%v).%v\n", typ, m.Name)
			}
			index = f.Index
		}
		inTyp, outTyp, mtyp, tfn, ifn, ptfn, pifn := createMethod(itype, typ, ptyp, m, i, index, rmap)
		isz := argsTypeSize(inTyp, true)
		osz := argsTypeSize(outTyp, false)
		pindex := i
		if !m.Pointer {
			pindex = index
		}
		pms[i].mtyp = mtyp
		pms[i].tfn = ptfn
		pms[i].ifn = pifn
		onePtr := checkOneFieldPtr(typ)
		pinfos[i] = &methodInfo{
			inTyp:    inTyp,
			outTyp:   outTyp,
			name:     m.Name,
			index:    pindex,
			isz:      isz,
			osz:      osz,
			pointer:  m.Pointer,
			variadic: m.Type.IsVariadic(),
			onePtr:   onePtr,
		}
		if !m.Pointer {
			ms[index].mtyp = mtyp
			ms[index].tfn = tfn
			ms[index].ifn = ifn
			infos[index] = &methodInfo{
				inTyp:    inTyp,
				outTyp:   outTyp,
				name:     m.Name,
				index:    index,
				isz:      isz,
				osz:      osz,
				pointer:  m.Pointer,
				variadic: m.Type.IsVariadic(),
				onePtr:   onePtr,
			}
		}
	}
	return true
}

func createMethod(itype int, typ reflect.Type, ptyp reflect.Type, m Method, i int, index int, rmap map[reflect.Type]reflect.Type) (inTyp, outTyp reflect.Type, mtyp typeOff, tfn, ifn, ptfn, pifn textOff) {
	var in []reflect.Type
	var out []reflect.Type
	var ntyp reflect.Type
	in, out, ntyp, inTyp, outTyp = parserMethodType(m.Type, rmap)
	mtyp = resolveReflectType(totype(ntyp))
	var ftyp reflect.Type
	if m.Pointer {
		ftyp = reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, m.Type.IsVariadic())
	} else {
		ftyp = reflect.FuncOf(append([]reflect.Type{typ}, in...), out, m.Type.IsVariadic())
	}

	mfn := reflect.MakeFunc(ftyp, m.Func)
	ptr := tovalue(&mfn).ptr

	sz := int(inTyp.Size())
	ifunc := icall(itype, i, true)

	if ifunc == nil {
		log.Printf("warning cannot wrapper method index:%v, size: %v\n", i, sz)
	} else {
		pifn = resolveReflectText(unsafe.Pointer(reflect.ValueOf(ifunc).Pointer()))
	}
	tfn = resolveReflectText(unsafe.Pointer(ptr))
	if !m.Pointer {
		ctyp := reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, m.Type.IsVariadic())
		cv := reflect.MakeFunc(ctyp, func(args []reflect.Value) (results []reflect.Value) {
			return args[0].Elem().Method(index).Call(args[1:])
		})
		ptfn = resolveReflectText(tovalue(&cv).ptr)
		ifunc := icall(itype, index, false)
		if ifunc == nil {
			log.Printf("warning cannot wrapper method index:%v, size: %v\n", i, sz)
		} else {
			ifn = resolveReflectText(unsafe.Pointer(reflect.ValueOf(ifunc).Pointer()))
		}
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
	pcount = len(methods)
	var mlist []string
	for _, m := range methods {
		if !m.Pointer {
			mlist = append(mlist, m.Name)
			mcount++
		}
	}
	ptyp := reflect.PtrTo(typ)
	if err := resizeMethod(typ, mcount); err != nil {
		return err
	}
	if err := resizeMethod(ptyp, pcount); err != nil {
		return err
	}
	rt := totype(typ)
	prt := totype(ptyp)

	ms := rt.exportedMethods()
	pms := prt.exportedMethods()

	infos := make([]*methodInfo, mcount, mcount)
	pinfos := make([]*methodInfo, pcount, pcount)
	itype := itypeIndex(typ)
	var index int
	for i, m := range methods {
		name := resolveReflectName(newName(m.Name, "", true))
		inTyp, outTyp, mtyp, tfn, ifn, ptfn, pifn := createMethod(itype, typ, ptyp, m, i, index, nil)
		isz := argsTypeSize(inTyp, true)
		osz := argsTypeSize(outTyp, false)
		pindex := i
		if !m.Pointer {
			pindex = index
		}
		onePtr := checkOneFieldPtr(typ)
		pms[i].name = name
		pms[i].mtyp = mtyp
		pms[i].tfn = ptfn
		pms[i].ifn = pifn
		pinfos[i] = &methodInfo{
			inTyp:    inTyp,
			outTyp:   outTyp,
			name:     m.Name,
			index:    pindex,
			isz:      isz,
			osz:      osz,
			pointer:  m.Pointer,
			variadic: m.Type.IsVariadic(),
			onePtr:   onePtr,
		}
		if !m.Pointer {
			ms[index].name = name
			ms[index].mtyp = mtyp
			ms[index].tfn = tfn
			ms[index].ifn = ifn
			infos[index] = &methodInfo{
				inTyp:    inTyp,
				outTyp:   outTyp,
				name:     m.Name,
				index:    index,
				isz:      isz,
				osz:      osz,
				pointer:  m.Pointer,
				variadic: m.Type.IsVariadic(),
				onePtr:   onePtr,
			}
			index++
		}
	}
	typInfoMap[typ] = infos
	typInfoMap[ptyp] = pinfos
	return nil
}

func newMethodSet(styp reflect.Type, maxmfunc, maxpfunc int) reflect.Type {
	rt, _ := newType("", "", styp, maxmfunc, 0)
	prt, _ := newType("", "", reflect.PtrTo(styp), maxpfunc, 0)
	rt.ptrToThis = resolveReflectType(prt)
	(*ptrType)(unsafe.Pointer(prt)).elem = rt
	setTypeName(rt, styp.PkgPath(), styp.Name())
	typ := toType(rt)
	if nt, ok := ntypeMap[styp]; ok {
		ntypeMap[typ] = &Named{Name: nt.Name, PkgPath: nt.PkgPath, Type: typ, From: nt.From, Kind: nt.Kind}
	}
	return typ
}

func _methodOf(styp reflect.Type, methods []Method) reflect.Type {
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
	rt, tt := newType("", "", styp, mcount, mcount)
	prt, ptt := newType("", "", reflect.PtrTo(styp), mcount, pcount)
	rt.ptrToThis = resolveReflectType(prt)

	(*ptrType)(unsafe.Pointer(prt)).elem = rt
	setTypeName(rt, styp.PkgPath(), styp.Name())
	typ := toType(rt)
	ptyp := reflect.PtrTo(typ)
	ms := make([]method, mcount, mcount)
	pms := make([]method, pcount, pcount)
	infos := make([]*methodInfo, mcount, mcount)
	pinfos := make([]*methodInfo, pcount, pcount)
	rmap := make(map[reflect.Type]reflect.Type)
	rmap[styp] = typ
	itype := itypeIndex(typ)
	var index int
	for i, m := range methods {
		name := resolveReflectName(newName(m.Name, "", true))
		inTyp, outTyp, mtyp, tfn, ifn, ptfn, pifn := createMethod(itype, typ, ptyp, m, i, index, rmap)
		isz := argsTypeSize(inTyp, true)
		osz := argsTypeSize(outTyp, false)
		pindex := i
		if !m.Pointer {
			pindex = index
		}
		onePtr := checkOneFieldPtr(typ)
		pms[i].name = name
		pms[i].mtyp = mtyp
		pms[i].tfn = ptfn
		pms[i].ifn = pifn
		pinfos[i] = &methodInfo{
			inTyp:    inTyp,
			outTyp:   outTyp,
			name:     m.Name,
			index:    pindex,
			isz:      isz,
			osz:      osz,
			pointer:  m.Pointer,
			variadic: m.Type.IsVariadic(),
			onePtr:   onePtr,
		}
		if !m.Pointer {
			ms[index].name = name
			ms[index].mtyp = mtyp
			ms[index].tfn = tfn
			ms[index].ifn = ifn
			infos[index] = &methodInfo{
				inTyp:    inTyp,
				outTyp:   outTyp,
				name:     m.Name,
				index:    index,
				isz:      isz,
				osz:      osz,
				pointer:  m.Pointer,
				variadic: m.Type.IsVariadic(),
				onePtr:   onePtr,
			}
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

func resetTypeList() {
	itypList = nil
}

var (
	itypList []reflect.Type
)

var (
	mu sync.Mutex
)

func isUserType(typ reflect.Type) bool {
	for _, t := range itypList {
		if t == typ {
			return true
		}
	}
	return false
}

func itypeIndex(typ reflect.Type) int {
	mu.Lock()
	defer mu.Unlock()
	for i, t := range itypList {
		if t == typ {
			return i
		}
	}
	itypList = append(itypList, typ)
	return len(itypList) - 1
}

func i_x(itype int, index int, ptr unsafe.Pointer, p unsafe.Pointer, ptrto bool) bool {
	typ := itypList[itype]
	otyp := typ
	if ptrto {
		typ = reflect.PtrTo(typ)
	}
	infos, ok := typInfoMap[typ]
	if !ok {
		log.Panicln("cannot found type info", typ)
		return false
	}
	info := infos[index]
	var method reflect.Method
	if ptrto && !info.pointer {
		method = MethodByIndex(typ.Elem(), info.index)
	} else {
		method = MethodByIndex(typ, info.index)
	}
	var receiver reflect.Value
	if !ptrto && info.onePtr {
		receiver = reflect.NewAt(otyp, unsafe.Pointer(&ptr)).Elem() //.Elem().Field(0)
	} else {
		receiver = reflect.NewAt(otyp, ptr)
		if !ptrto || !info.pointer {
			receiver = receiver.Elem()
		}
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

func i_x_dyn(i int, ptr unsafe.Pointer, p unsafe.Pointer, ptrto bool) bool {
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
