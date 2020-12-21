package reflectx

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"unsafe"
)

const (
	VerifyFieldName = "_reflectx_verify"
)

var (
	verifyFieldType = reflect.TypeOf(unsafe.Pointer(nil))
)

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

func MethodOf(styp reflect.Type, methods []Method, addVerifyField bool) reflect.Type {
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
	if addVerifyField && styp.Kind() == reflect.Struct {
		var fs []reflect.StructField
		var skip bool
		for i := 0; i < styp.NumField(); i++ {
			field := styp.Field(i)
			if field.Name == VerifyFieldName {
				skip = true
				break
			}
			fs = append(fs, field)
		}
		if !skip {
			fs = append(fs, reflect.StructField{
				Name:    VerifyFieldName,
				PkgPath: "main",
				Type:    verifyFieldType,
			})
			styp = NamedStructOf(styp.PkgPath(), styp.Name(), fs)
		}
	}
	rt, _ := premakeMethodType(styp, mcount, mcount)
	prt, _ := premakeMethodType(reflect.PtrTo(styp), pcount, pcount)
	rt.ptrToThis = resolveReflectType(prt)
	(*ptrType)(unsafe.Pointer(prt)).elem = rt
	typ := toType(rt)
	ptyp := reflect.PtrTo(typ)
	ms := rt.methods()
	pms := prt.methods()
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
		_, ifunc := icall(i, sz, m.Type.NumOut() > 0, true)
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
			_, ifunc := icall(index, int(sz), m.Type.NumOut() > 0, false)
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
	typInfoMap[typ] = infos
	typInfoMap[ptyp] = pinfos
	nt := &Named{Name: styp.Name(), PkgPath: styp.PkgPath(), Type: typ, Kind: TkStruct}
	ntypeMap[typ] = nt
	return typ
}

func toRealType(typ, orgtyp, mtyp reflect.Type) (in, out []reflect.Type, ntyp, inTyp, outTyp reflect.Type) {
	fn := func(t reflect.Type) reflect.Type {
		if t == orgtyp {
			return typ
		} else if t.Kind() == reflect.Ptr && t.Elem() == orgtyp {
			return reflect.PtrTo(typ)
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
	if IsNamed(typ) {
		storeValue(v)
	}
	return v
}

func toElem(typ reflect.Type) reflect.Type {
	if typ.Kind() == reflect.Ptr {
		return typ.Elem()
	}
	return typ
}

func storeValue(v reflect.Value) {
	ptr := tovalue(&v).ptr
	ptrTypeMap[ptr] = toElem(v.Type())

	// check verify field
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		item := FieldByName(v, VerifyFieldName)
		if item.IsValid() {
			item.SetPointer(ptr)
		}
	}
}

func foundTypeByPtr(ptr unsafe.Pointer, index int, insize int) reflect.Type {
	typ, ok := ptrTypeMap[ptr]
	if ok {
		return typ
	}
	checkMap := make(map[reflect.Type]bool)
	var matches []reflect.Type
	for p, typ := range ptrTypeMap {
		v2 := reflect.NewAt(typ, ptr).Elem()
		v1 := reflect.NewAt(typ, p).Elem()
		if reflect.DeepEqual(v1.Interface(), v2.Interface()) {
			if _, ok := typ.FieldByName(VerifyFieldName); ok {
				return typ
			}
			if !checkMap[typ] {
				checkMap[typ] = true
				infos := typInfoMap[typ]
				if len(infos) > index && int(infos[index].inTyp.Size()) == insize {
					matches = append(matches, typ)
				}
			}
		}
	}
	n := len(matches)
	if n == 1 {
		return matches[0]
	} else if n > 1 {
		log.Println("warring, multiple matches found, please add verify field.", matches)
	}
	return nil
}

func i_x(i int, this uintptr, p []byte, ptrto bool) []byte {
	ptr := unsafe.Pointer(this)
	typ := foundTypeByPtr(ptr, i, len(p))
	if typ == nil {
		log.Println("cannot found ptr type", ptr)
		return nil
	}
	if ptrto {
		typ = reflect.PtrTo(typ)
	}
	infos, ok := typInfoMap[typ]
	if !ok {
		log.Println("cannot found type info", typ)
	}
	info := infos[i]
	var method reflect.Method
	if ptrto && !info.pointer {
		method = MethodByIndex(typ.Elem(), info.index)
	} else {
		method = MethodByIndex(typ, info.index)
	}
	var in []reflect.Value
	var receiver reflect.Value
	if ptrto {
		receiver = reflect.NewAt(typ.Elem(), ptr)
		if !info.pointer {
			receiver = receiver.Elem()
		}
	} else {
		receiver = reflect.NewAt(typ, ptr).Elem()
	}
	in = append(in, receiver)
	inCount := method.Type.NumIn()
	if inCount > 1 {
		inArgs := reflect.NewAt(info.inTyp, unsafe.Pointer(&p[0])).Elem()
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
	if len(r) > 0 {
		out := reflect.New(info.outTyp).Elem()
		for i, v := range r {
			out.Field(i).Set(v)
		}
		return *(*[]byte)(tovalue(&out).ptr)
	}
	return nil
}
