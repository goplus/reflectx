package reflectx

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"unsafe"
)

// memmove copies size bytes to dst from src. No write barriers are used.
//go:linkname memmove reflect.memmove
func memmove(dst, src unsafe.Pointer, size uintptr)

type Method struct {
	Name    string        // method Name
	Type    reflect.Type  // method type without receiver
	Func    reflect.Value // func with receiver as first argument
	Pointer bool          // receiver is pointer
}

// MakeMethod returns a new Method of the given Type
// that wraps the function fn.
//
//	- name: method name
//	- pointer: flag receiver struct or pointer
//	- typ: method func type without receiver
//	- fn: func with receiver as first argument
func MakeMethod(name string, pointer bool, typ reflect.Type, fn func(args []reflect.Value) (result []reflect.Value)) Method {
	return Method{
		Name:    name,
		Type:    typ,
		Func:    reflect.MakeFunc(typ, fn),
		Pointer: pointer,
	}
}

func MethodOf(styp reflect.Type, methods []Method) reflect.Type {
	sort.Slice(methods, func(i, j int) bool {
		n := strings.Compare(methods[i].Name, methods[j].Name)
		if n == 0 {
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
	rt, tt := premakeMethodType(styp, mcount, mcount)
	prt, ptt := premakeMethodType(reflect.PtrTo(styp), pcount, pcount)
	rt.ptrToThis = resolveReflectType(prt)
	(*ptrType)(unsafe.Pointer(prt)).elem = rt
	typ := toType(rt)
	ptyp := reflect.PtrTo(typ)
	ms := make([]method, mcount, mcount)
	pms := make([]method, pcount, pcount)
	var infos []*methodInfo
	var pinfos []*methodInfo
	var index int
	for i, m := range methods {
		ptr := tovalue(&m.Func).ptr
		name := resolveReflectName(newName(m.Name, "", true))
		in, out, ntyp, inTyp, outTyp := toRealType(typ, orgtyp, m.Type)
		mtyp := resolveReflectType(totype(ntyp))
		var ftyp reflect.Type
		if m.Pointer {
			ftyp = reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, m.Type.IsVariadic())
		} else {
			ftyp = reflect.FuncOf(append([]reflect.Type{typ}, in...), out, m.Type.IsVariadic())
		}
		funcImpl := (*makeFuncImpl)(tovalue(&m.Func).ptr)
		funcImpl.ftyp = (*funcType)(unsafe.Pointer(totype(ftyp)))
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

		pms[i].name = name
		pms[i].mtyp = mtyp
		pms[i].tfn = ptfn
		pms[i].ifn = pifn
		pinfos = append(pinfos, &methodInfo{
			inTyp:    inTyp,
			outTyp:   outTyp,
			name:     m.Name,
			index:    pindex,
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
				pointer:  m.Pointer,
				variadic: m.Type.IsVariadic(),
			})
			index++
		}
	}
	copy(tt.Elem().Field(2).Slice(0, len(ms)).Interface().([]method), ms)
	copy(ptt.Elem().Field(2).Slice(0, len(pms)).Interface().([]method), pms)
	typInfoMap[typ] = infos
	typInfoMap[ptyp] = pinfos
	nt := &Named{Name: styp.Name(), PkgPath: styp.PkgPath(), Type: typ, Kind: TkMethod}
	ntypeMap[typ] = nt
	return typ
}

func toRealType(typ, orgtyp, mtyp reflect.Type) (in, out []reflect.Type, ntyp, inTyp, outTyp reflect.Type) {
	var fnx func(t reflect.Type) (reflect.Type, bool)
	fnx = func(t reflect.Type) (reflect.Type, bool) {
		if t == orgtyp {
			return typ, true
		}
		switch t.Kind() {
		case reflect.Ptr:
			if e, ok := fnx(t.Elem()); ok {
				return reflect.PtrTo(e), true
			}
		case reflect.Slice:
			if e, ok := fnx(t.Elem()); ok {
				return reflect.SliceOf(e), true
			}
		case reflect.Array:
			if e, ok := fnx(t.Elem()); ok {
				return reflect.ArrayOf(t.Len(), e), true
			}
		case reflect.Map:
			k, ok1 := fnx(t.Key())
			v, ok2 := fnx(t.Elem())
			if ok1 || ok2 {
				return reflect.MapOf(k, v), true
			}
		}
		return t, false
	}
	fn := func(t reflect.Type) reflect.Type {
		if r, ok := fnx(t); ok {
			return r
		}
		return t
	}
	var inFields []reflect.StructField
	var outFields []reflect.StructField
	for i := 0; i < mtyp.NumIn(); i++ {
		t := fn(mtyp.In(i))
		in = append(in, t)
		inFields = append(inFields, reflect.StructField{
			Name: fmt.Sprintf("Arg%v", i),
			Type: t,
		})
	}
	for i := 0; i < mtyp.NumOut(); i++ {
		t := fn(mtyp.Out(i))
		out = append(out, t)
		outFields = append(outFields, reflect.StructField{
			Name: fmt.Sprintf("Out%v", i),
			Type: t,
		})
	}
	ntyp = reflect.FuncOf(in, out, mtyp.IsVariadic())
	inTyp = reflect.StructOf(inFields)
	outTyp = reflect.StructOf(outFields)
	return
}

func premakeMethodType(styp reflect.Type, mcount int, xcount int) (rt *rtype, tt reflect.Value) {
	ort := totype(styp)
	switch styp.Kind() {
	case reflect.Struct:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(structType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*structType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := toStructType(ort)
		st.fields = ost.fields
		rt = (*rtype)(unsafe.Pointer(st))
	case reflect.Ptr:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(ptrType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*ptrType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		rt = (*rtype)(unsafe.Pointer(st))
	case reflect.Slice:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(sliceType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*sliceType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		st.elem = totype(styp.Elem())
		rt = (*rtype)(unsafe.Pointer(st))
	case reflect.Array:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(arrayType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*arrayType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*arrayType)(unsafe.Pointer(ort))
		st.elem = ost.elem
		st.slice = ost.slice
		st.len = ost.len
		rt = (*rtype)(unsafe.Pointer(st))
	default:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(rtype{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		rt = (*rtype)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
	}
	ut := (*uncommonType)(unsafe.Pointer(tt.Elem().Field(1).UnsafeAddr()))
	// copy(tt.Elem().Field(2).Slice(0, len(methods)).Interface().([]method), methods)
	ut.mcount = uint16(mcount)
	ut.xcount = uint16(xcount)
	ut.moff = uint32(unsafe.Sizeof(uncommonType{}))

	rt.size = ort.size
	rt.tflag = ort.tflag | tflagUncommon
	rt.kind = ort.kind
	rt.align = ort.align
	rt.fieldAlign = ort.fieldAlign
	rt.gcdata = ort.gcdata
	rt.ptrdata = ort.ptrdata
	rt.str = resolveReflectName(ort.nameOff(ort.str))
	return
}

var (
	typInfoMap = make(map[reflect.Type][]*methodInfo)
	ptrTypeMap = make(map[unsafe.Pointer]reflect.Type)
)

type methodInfo struct {
	inTyp    reflect.Type
	outTyp   reflect.Type
	name     string
	index    int
	pointer  bool
	variadic bool
}

func MethodByIndex(typ reflect.Type, index int) reflect.Method {
	m := typ.Method(index)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if _, ok := ntypeMap[typ]; ok {
		tovalue(&m.Func).flag |= flagIndir
	}
	return m
}

func MethodByName(typ reflect.Type, name string) (m reflect.Method, ok bool) {
	m, ok = typ.MethodByName(name)
	if !ok {
		return
	}
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if _, ok := ntypeMap[typ]; ok {
		tovalue(&m.Func).flag |= flagIndir
	}
	return
}

func New(typ reflect.Type) reflect.Value {
	v := reflect.New(typ)
	if IsMethod(typ) {
		storeMethodValue(v)
	}
	return v
}

func Interface(v reflect.Value) interface{} {
	i := v.Interface()
	if i != nil && IsMethod(v.Type()) {
		storeMethodValue(reflect.ValueOf(i))
	}
	return i
}

func toElem(typ reflect.Type) reflect.Type {
	if typ.Kind() == reflect.Ptr {
		return typ.Elem()
	}
	return typ
}

func storeMethodValue(v reflect.Value) {
	ptr := tovalue(&v).ptr
	ptrTypeMap[ptr] = toElem(v.Type())
}

const (
	uintptrAligin = unsafe.Sizeof(uintptr(0))
)

func methodArgsOffsize(typ reflect.Type) (off uintptr) {
	for i := 1; i < typ.NumIn(); i++ {
		t := typ.In(i)
		targ := totype(t)
		a := uintptr(targ.align)
		off = (off + a - 1) &^ (a - 1)
		n := targ.size
		if n == 0 {
			continue
		}
		off += n
	}
	off = (off + uintptrAligin - 1) &^ (uintptrAligin - 1)
	if off == 0 {
		return uintptrAligin
	}
	return
}

func methodReturnSize(typ reflect.Type) (off uintptr) {
	for i := 0; i < typ.NumOut(); i++ {
		t := typ.Out(i)
		targ := totype(t)
		a := uintptr(targ.align)
		off = (off + a - 1) &^ (a - 1)
		n := targ.size
		if n == 0 {
			continue
		}
		off += n
	}
	return
}

func i_x(i int, ptr unsafe.Pointer, p unsafe.Pointer, ptrto bool) bool {
	typ, ok := ptrTypeMap[ptr]
	if !ok {
		log.Println("cannot found ptr type", ptr)
		return false
	}
	if ptrto {
		typ = reflect.PtrTo(typ)
	}
	infos, ok := typInfoMap[typ]
	if !ok {
		log.Println("cannot found type info", typ)
		return false
	}
	info := infos[i]
	var method reflect.Method
	if ptrto && !info.pointer {
		method = MethodByIndex(typ.Elem(), info.index)
	} else {
		method = MethodByIndex(typ, info.index)
	}
	var receiver reflect.Value
	if ptrto {
		receiver = reflect.NewAt(typ.Elem(), ptr)
		if !info.pointer {
			receiver = receiver.Elem()
		}
	} else {
		receiver = reflect.NewAt(typ, ptr).Elem()
	}
	in := []reflect.Value{receiver}

	var ioff uintptr
	if inCount := method.Type.NumIn(); inCount > 1 {
		ioff = methodArgsOffsize(method.Type)
		isz := info.inTyp.Size()
		buf := make([]byte, isz, isz)
		if isz > ioff {
			isz = ioff
		}
		for i := uintptr(0); i < isz; i++ {
			buf[i] = *(*byte)(add(p, i, ""))
		}
		var inArgs reflect.Value
		if isz == 0 {
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
		osz := methodReturnSize(method.Type)
		po := unsafe.Pointer(out.UnsafeAddr())
		for i := uintptr(0); i < osz; i++ {
			*(*byte)(add(p, ioff+i, "")) = *(*byte)(add(po, uintptr(i), ""))
		}
	}
	return true
}
