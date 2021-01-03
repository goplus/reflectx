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

//go:linkname typedmemmove reflect.typedmemmove
func typedmemmove(typ *rtype, dst, src unsafe.Pointer)

// MakeMethod make reflect.Method for MethodOf
// - name: method name
// - pointer: flag receiver struct or pointer
// - typ: method func type without receiver
// - fn: func with receiver as first argument
func MakeMethod(name string, pointer bool, typ reflect.Type, fn func(args []reflect.Value) (result []reflect.Value)) reflect.Method {
	var in []reflect.Type
	var out []reflect.Type
	if pointer {
		in = append(in, tyEmptyInterfacePtr)
	} else {
		in = append(in, tyEmptyInterface)
	}
	for i := 0; i < typ.NumIn(); i++ {
		in = append(in, typ.In(i))
	}
	for i := 0; i < typ.NumOut(); i++ {
		out = append(out, typ.Out(i))
	}
	return reflect.Method{
		Name: name,
		Type: reflect.FuncOf(in, out, typ.IsVariadic()),
		Func: reflect.MakeFunc(typ, fn),
	}
}

func extraFieldMethod(ifield int, typ reflect.Type, skip map[string]bool) (methods []reflect.Method) {
	isPtr := typ.Kind() == reflect.Ptr
	for i := 0; i < typ.NumMethod(); i++ {
		m := MethodByIndex(typ, i)
		if skip[m.Name] {
			continue
		}
		var fn func(args []reflect.Value) []reflect.Value
		if isPtr {
			fn = func(args []reflect.Value) []reflect.Value {
				args[0] = args[0].Elem().Field(ifield).Addr()
				return m.Func.Call(args)
			}
		} else {
			fn = func(args []reflect.Value) []reflect.Value {
				args[0] = args[0].Field(ifield)
				return m.Func.Call(args)
			}
		}
		methods = append(methods, reflect.Method{
			Name:    m.Name,
			PkgPath: m.PkgPath,
			Type:    m.Type,
			Func:    reflect.MakeFunc(m.Type, fn),
		})
	}
	return
}

func parserFuncIO(typ reflect.Type) (in, out []reflect.Type) {
	for i := 0; i < typ.NumIn(); i++ {
		in = append(in, typ.In(i))
	}
	for i := 0; i < typ.NumOut(); i++ {
		out = append(out, typ.Out(i))
	}
	return
}

func extraPtrFieldMethod(ifield int, typ reflect.Type) (methods []reflect.Method) {
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		in, out := parserFuncIO(m.Type)
		in[0] = tyEmptyInterface
		mtyp := reflect.FuncOf(in, out, m.Type.IsVariadic())
		imethod := i
		methods = append(methods, reflect.Method{
			Name:    m.Name,
			PkgPath: m.PkgPath,
			Type:    mtyp,
			Func: reflect.MakeFunc(
				mtyp,
				func(args []reflect.Value) []reflect.Value {
					var recv = args[0]
					return recv.Field(ifield).Method(imethod).Call(args[1:])
				},
			),
		})
	}
	return
}

func extraInterfaceFieldMethod(ifield int, typ reflect.Type) (methods []reflect.Method) {
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		in, out := parserFuncIO(m.Type)
		in = append([]reflect.Type{tyEmptyInterface}, in...)
		mtyp := reflect.FuncOf(in, out, m.Type.IsVariadic())
		imethod := i
		methods = append(methods, reflect.Method{
			Name:    m.Name,
			PkgPath: m.PkgPath,
			Type:    mtyp,
			Func: reflect.MakeFunc(
				mtyp,
				func(args []reflect.Value) []reflect.Value {
					var recv = args[0]
					return recv.Field(ifield).Method(imethod).Call(args[1:])
				},
			),
		})
	}
	return
}

func extractEmbedMethod(styp reflect.Type) []reflect.Method {
	var methods []reflect.Method
	for i := 0; i < styp.NumField(); i++ {
		sf := styp.Field(i)
		if !sf.Anonymous {
			continue
		}
		switch sf.Type.Kind() {
		case reflect.Interface:
			ms := extraInterfaceFieldMethod(i, sf.Type)
			methods = append(methods, ms...)
		case reflect.Ptr:
			ms := extraPtrFieldMethod(i, sf.Type)
			methods = append(methods, ms...)
		default:
			skip := make(map[string]bool)
			ms := extraFieldMethod(i, sf.Type, skip)
			for _, m := range ms {
				skip[m.Name] = true
			}
			pms := extraFieldMethod(i, reflect.PtrTo(sf.Type), skip)
			methods = append(methods, ms...)
			methods = append(methods, pms...)
		}
	}
	// ambiguous selector check
	chk := make(map[string]int)
	for _, m := range methods {
		chk[m.Name]++
	}
	var ms []reflect.Method
	for _, m := range methods {
		if chk[m.Name] == 1 {
			ms = append(ms, m)
		}
	}
	return ms
}

func MethodOf(styp reflect.Type, methods []reflect.Method) reflect.Type {
	chk := make(map[string]int)
	for _, m := range methods {
		chk[m.Name]++
		if chk[m.Name] > 1 {
			panic(fmt.Sprintf("method redeclared: %v", m.Name))
		}
	}
	if styp.Kind() == reflect.Struct {
		ms := extractEmbedMethod(styp)
		for _, m := range ms {
			if chk[m.Name] == 1 {
				continue
			}
			methods = append(methods, m)
		}
	}
	return methodOf(styp, methods)
}

func methodOf(styp reflect.Type, methods []reflect.Method) reflect.Type {
	sort.Slice(methods, func(i, j int) bool {
		n := strings.Compare(methods[i].Name, methods[j].Name)
		if n == 0 && methods[i].Type == methods[j].Type {
			panic(fmt.Sprintf("method redeclared: %v", methods[j].Name))
		}
		return n < 0
	})
	isPointer := func(m reflect.Method) bool {
		return m.Type.In(0).Kind() == reflect.Ptr
	}
	var mcount, pcount int
	pcount = len(methods)
	var mlist []string
	for _, m := range methods {
		if !isPointer(m) {
			mlist = append(mlist, m.Name)
			mcount++
		}
	}
	orgtyp := styp
	rt, tt := newType(styp, mcount, mcount)
	prt, ptt := newType(reflect.PtrTo(styp), pcount, pcount)
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
		ptr := tovalue(&m.Func).ptr
		name := resolveReflectName(newName(m.Name, "", true))
		in, out, ntyp, inTyp, outTyp := toRealType(typ, orgtyp, m.Type)
		mtyp := resolveReflectType(totype(ntyp))
		pointer := isPointer(m)
		var ftyp reflect.Type
		if pointer {
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
		if !pointer {
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
			pointer:  pointer,
			variadic: m.Type.IsVariadic(),
		})
		if !pointer {
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
				pointer:  pointer,
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
	for i := 1; i < mtyp.NumIn(); i++ {
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

func newType(styp reflect.Type, mcount int, xcount int) (rt *rtype, tt reflect.Value) {
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
		st.elem = totype(styp.Elem())
		rt = (*rtype)(unsafe.Pointer(st))
	case reflect.Interface:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(interfaceType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
		}))
		st := (*interfaceType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*interfaceType)(unsafe.Pointer(ort))
		for _, m := range ost.methods {
			st.methods = append(st.methods, imethod{
				name: resolveReflectName(ost.nameOff(m.name)),
				typ:  resolveReflectType(ost.typeOff(m.typ)),
			})
		}
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
	case reflect.Chan:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(chanType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*chanType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*chanType)(unsafe.Pointer(ort))
		st.elem = ost.elem
		st.dir = ost.dir
		rt = (*rtype)(unsafe.Pointer(st))
	case reflect.Func:
		narg := styp.NumIn() + styp.NumOut()
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(funcType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(narg, reflect.TypeOf((*rtype)(nil)))},
		}))
		st := (*funcType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*funcType)(unsafe.Pointer(ort))
		st.inCount = ost.inCount
		st.outCount = ost.outCount
		if narg > 0 {
			args := make([]*rtype, narg, narg)
			for i := 0; i < styp.NumIn(); i++ {
				args[i] = totype(styp.In(i))
			}
			index := styp.NumIn()
			for i := 0; i < styp.NumOut(); i++ {
				args[index+i] = totype(styp.Out(i))
			}
			copy(tt.Elem().Field(2).Slice(0, narg).Interface().([]*rtype), args)
		}
		rt = (*rtype)(unsafe.Pointer(st))
	case reflect.Map:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(mapType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*mapType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ort := (*mapType)(unsafe.Pointer(ort))
		st.key = ort.key
		st.elem = ort.elem
		st.bucket = ort.bucket
		st.hasher = ort.hasher
		st.keysize = ort.keysize
		st.valuesize = ort.valuesize
		st.bucketsize = ort.bucketsize
		st.flags = ort.flags
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
	typInfoMap   = make(map[reflect.Type][]*methodInfo)
	valueInfoMap = make(map[reflect.Value]typeInfo)
)

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
	if IsMethod(typ) {
		tovalue(&m.Func).flag |= flagIndir
	}
	return m
}

func MethodByName(typ reflect.Type, name string) (m reflect.Method, ok bool) {
	m, ok = typ.MethodByName(name)
	if !ok {
		return
	}
	if IsMethod(typ) {
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

func Interface(v reflect.Value) interface{} {
	i := v.Interface()
	if i != nil {
		checkStoreMethodValue(reflect.ValueOf(i))
	}
	return i
}

func MakeEmptyInterface(pkgpath string, name string) reflect.Type {
	return NamedTypeOf(pkgpath, name, tyEmptyInterface)
}

func NamedInterfaceOf(pkgpath string, name string, embedded []reflect.Type, methods []reflect.Method) reflect.Type {
	styp := NamedTypeOf(pkgpath, name, tyEmptyInterface)
	return InterfaceOf(styp, embedded, methods)
}

func InterfaceOf(styp reflect.Type, embedded []reflect.Type, methods []reflect.Method) reflect.Type {
	if styp.Kind() != reflect.Interface {
		panic(fmt.Errorf("non-interface %v", styp))
	}
	for _, e := range embedded {
		if e.Kind() != reflect.Interface {
			panic(fmt.Errorf("interface contains embedded non-interface %v", e))
		}
		for i := 0; i < e.NumMethod(); i++ {
			m := e.Method(i)
			methods = append(methods, reflect.Method{
				Name: m.Name,
				Type: m.Type,
			})
		}
	}
	sort.Slice(methods, func(i, j int) bool {
		n := strings.Compare(methods[i].Name, methods[j].Name)
		if n == 0 && methods[i].Type != methods[j].Type {
			panic(fmt.Sprintf("duplicate method %v", methods[j].Name))
		}
		return n < 0
	})
	rt, _ := newType(styp, 0, 0)
	st := (*interfaceType)(unsafe.Pointer(rt))
	st.methods = nil
	var lastname string
	for _, m := range methods {
		if m.Name == lastname {
			continue
		}
		lastname = m.Name
		st.methods = append(st.methods, imethod{
			name: resolveReflectName(newName(m.Name, "", isExported(m.Name))),
			typ:  resolveReflectType(totype(m.Type)),
		})
	}
	return toType(rt)
}

func toElem(typ reflect.Type) reflect.Type {
	if typ.Kind() == reflect.Ptr {
		return typ.Elem()
	}
	return typ
}

func toElemValue(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		return v.Elem()
	}
	return v
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
