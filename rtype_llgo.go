//go:build llgo

package reflectx

import (
	"path"
	"reflect"
	"strconv"
	"unsafe"
)

func NamedTypeOf(pkgpath string, name string, from reflect.Type) reflect.Type {
	if from.Kind() == reflect.Func {
		from = toType(closureOf(totype(from).FuncType()))
	}
	rt, _ := newType(pkgpath, name, from, 0, 0)
	setTypeName(rt, pkgpath, name)
	if rt.IsClosure() {
		ft := toFuncType(rt.StructType())
		rt = &ft.Type
	}
	return toType(rt)
}

func setTypeName(t *rtype, pkgpath string, name string) {
	if pkgpath == "" && name == "" {
		return
	}
	//exported := isExported(name)
	if pkgpath != "" {
		_, f := path.Split(pkgpath)
		name = f + "." + name
	}
	t.TFlag |= tflagNamed
	t.Str_ = name
	if t.TFlag&tflagUncommon == tflagUncommon {
		t.Uncommon().PkgPath_ = pkgpath
	}
	switch reflect.Kind(t.Kind()) {
	case reflect.Struct:
		st := (*structType)(toKindType(t))
		if !st.IsClosure() {
			st.PkgPath_ = pkgpath
		}
	case reflect.Interface:
		st := (*interfaceType)(toKindType(t))
		st.PkgPath_ = pkgpath
	}
}

func (ctx *Context) StructOf(fields []reflect.StructField) reflect.Type {
	var anonymous []int
	underscore := make(map[int]string)
	var underscoreCount int
	fs := make([]reflect.StructField, len(fields))
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if f.Anonymous {
			anonymous = append(anonymous, i)
			f.Anonymous = false
			if f.Name == "" {
				f.Name = typeName(f.Type)
			}
		} else if f.Name == "_" {
			if underscoreCount > 0 {
				underscore[i] = f.Name
				f.Name = "_gop_underscore_" + strconv.Itoa(i)
			}
			underscoreCount++
		}
		fs[i] = f
	}
	typ := reflect.StructOf(fs)
	rt := totype(typ)
	st := toStructType(rt)
	for _, i := range anonymous {
		setEmbedded(&st.Fields[i])
	}
	for i, n := range underscore {
		st.Fields[i].Name_ = n
	}
	str := typ.String()
	if ts, ok := ctx.structLookupCache[str]; ok {
		for _, t := range ts {
			if haveIdenticalType(totype(t), totype(typ), true) {
				return t
			}
		}
		ts = append(ts, typ)
	} else {
		ctx.structLookupCache[str] = []reflect.Type{typ}
	}
	// fix equal for blank fields and uncomparable type
	if rt.Equal != nil && underscoreCount > 0 {
		rt.Equal = func(p, q unsafe.Pointer) bool {
			for i, ft := range st.Fields {
				if fields[i].Name == "_" {
					continue
				}
				pi := add(p, ft.Offset, "&x.field safe")
				qi := add(q, ft.Offset, "&x.field safe")
				if !ft.Typ.Equal(pi, qi) {
					return false
				}
			}
			return true
		}
	}

	if rt.TFlag == 0 && isRegularMemory(typ) {
		rt.TFlag |= tflagRegularMemory
	}
	return typ
}

func setEmbedded(f *structField) {
	f.Embedded_ = true
}

func rtypeMethodByNameX(t *rtype, name string) (m reflect.Method, ok bool) {
	if reflect.Kind(t.Kind()) == reflect.Interface {
		return toType(t).MethodByName(name)
	}
	if ut := t.Uncommon(); ut != nil {
		for i, p := range ut.Methods() {
			if p.Name() == name {
				return rtypeMethodX(t, i), true
			}
		}
	}
	return reflect.Method{}, false
}

//go:linkname closureOf reflect.closureOf
func closureOf(ftyp *funcType) *rtype

//go:linkname toFuncType reflect.toFuncType
func toFuncType(ftyp *structType) *funcType

func rtypeMethodX(t *rtype, i int) (m reflect.Method) {
	if reflect.Kind(t.Kind()) == reflect.Interface {
		return toType(t).Method(i)
	}
	methods := rtypeMethods(t)
	if i < 0 || i >= len(methods) {
		panic("reflect: Method index out of range")
	}
	p := methods[i]
	m.Name = p.Name()
	fl := flag(reflect.Func)
	ft := p.Mtyp_
	in := make([]reflect.Type, 0, 1+len(ft.In))
	in = append(in, toType(t))
	for _, arg := range ft.In {
		in = append(in, toType(arg))
	}
	out := make([]reflect.Type, 0, len(ft.Out))
	for _, ret := range ft.Out {
		out = append(out, toType(ret))
	}
	mt := reflect.FuncOf(in, out, ft.Variadic())
	m.Type = mt
	mtfn := (*funcType)(unsafe.Pointer(totype(mt)))
	fv := &struct {
		fn  unsafe.Pointer
		env unsafe.Pointer
	}{p.Tfn_, nil}
	m.Func = toValue(Value{closureOf(mtfn), unsafe.Pointer(fv), fl | flagIndir})
	m.Index = i
	return m
}

func newType(pkg string, name string, styp reflect.Type, mcount int, xcount int) (*rtype, []method) {
	var rt *rtype
	var fnoff uint32
	var tt reflect.Value
	ort := totype(styp)
	skind := styp.Kind()
	switch skind {
	case reflect.Struct:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(structType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*structType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*structType)(unsafe.Pointer(ort))
		st.Fields = ost.Fields
	case reflect.Ptr:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(ptrType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*ptrType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		st.Elem = totype(styp.Elem())
	case reflect.Interface:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(interfaceType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
		}))
		st := (*interfaceType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*interfaceType)(unsafe.Pointer(ort))
		for _, m := range ost.Methods {
			st.Methods = append(st.Methods, imethod{
				Name_: m.Name_,
				Typ_:  m.Typ_,
			})
		}
	case reflect.Slice:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(sliceType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*sliceType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		st.Elem = totype(styp.Elem())
	case reflect.Array:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(arrayType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*arrayType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*arrayType)(unsafe.Pointer(ort))
		st.Elem = ost.Elem
		st.Slice = ost.Slice
		st.Len = ost.Len
	case reflect.Chan:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(chanType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*chanType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*chanType)(unsafe.Pointer(ort))
		st.Elem = ost.Elem
		st.Dir = ost.Dir
	case reflect.Func:
		numIn := styp.NumIn()
		numOut := styp.NumOut()
		narg := numIn + numOut
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(funcType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "N", Type: reflect.ArrayOf(narg, reflect.TypeOf((*rtype)(nil)))},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*funcType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*funcType)(unsafe.Pointer(ort))
		st.In = ost.In
		st.Out = ost.Out
	case reflect.Map:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(mapType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*mapType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*mapType)(unsafe.Pointer(ort))
		cloneMap(st, ost)
	default:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(rtype{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
	}
	rt = (*rtype)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
	rt.Size_ = ort.Size_
	rt.TFlag = ort.TFlag | tflagUncommon
	rt.Kind_ = ort.Kind_
	rt.Align_ = ort.Align_
	rt.FieldAlign_ = ort.FieldAlign_
	rt.GCData = ort.GCData
	rt.PtrBytes = ort.PtrBytes
	rt.Equal = ort.Equal
	rt.Str_ = ort.Str_
	ut := (*uncommonType)(unsafe.Pointer(tt.Elem().Field(1).UnsafeAddr()))
	ut.Mcount = uint16(mcount)
	ut.Xcount = uint16(xcount)
	ut.Moff = uint32(unsafe.Sizeof(uncommonType{}))
	if skind == reflect.Interface {
		return rt, nil
	} else if skind == reflect.Func {
		ut.Moff += fnoff
		//return rt, tt.Elem().Field(3).Slice(0, mcount).Interface().([]method)
		data := toWord(tt.Elem().Field(3).Slice(0, mcount).Interface())
		return rt, *(*[]method)(data)
	}
	//return rt, tt.Elem().Field(2).Slice(0, mcount).Interface().([]method)
	data := toWord(tt.Elem().Field(2).Slice(0, mcount).Interface())
	return rt, *(*[]method)(data)
}

func toWord(i interface{}) unsafe.Pointer {
	return (*emptyInterface)(unsafe.Pointer(&i)).word
}
