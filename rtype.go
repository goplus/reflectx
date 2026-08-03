//go:build !llgo

package reflectx

import (
	"fmt"
	"io"
	"path"
	"reflect"
	"strconv"
	"unsafe"
)

func funcTypeIn(t *funcType) []*rtype {
	return t.InSlice()
}

func funcTypeOut(t *funcType) []*rtype {
	return t.OutSlice()
}

type uncommonFuncType struct {
	funcType
	uncommonType
	args [1]*rtype
}

func uncommonFuncTypeArgs(rt *rtype, nargs int) []*rtype {
	f := (*uncommonFuncType)(unsafe.Pointer(rt))
	return (*[1 << 16]*rtype)(unsafe.Pointer(&f.args))[:nargs:nargs]
}

func SetUnderlying(typ reflect.Type, styp reflect.Type) {
	rt := totype(typ)
	ort := totype(styp)
	switch styp.Kind() {
	case reflect.Struct:
		st := (*structType)(unsafe.Pointer(rt))
		ost := (*structType)(unsafe.Pointer(ort))
		st.Fields = ost.Fields
	case reflect.Ptr:
		st := (*ptrType)(unsafe.Pointer(rt))
		ost := (*ptrType)(unsafe.Pointer(ort))
		st.Elem = ost.Elem
	case reflect.Slice:
		st := (*sliceType)(unsafe.Pointer(rt))
		ost := (*sliceType)(unsafe.Pointer(ort))
		st.Elem = ost.Elem
	case reflect.Array:
		st := (*arrayType)(unsafe.Pointer(rt))
		ost := (*arrayType)(unsafe.Pointer(ort))
		st.Elem = ost.Elem
		st.Slice = ost.Slice
		st.Len = ost.Len
	case reflect.Chan:
		st := (*chanType)(unsafe.Pointer(rt))
		ost := (*chanType)(unsafe.Pointer(ort))
		st.Elem = ost.Elem
		st.Dir = ost.Dir
	case reflect.Interface:
		st := (*interfaceType)(unsafe.Pointer(rt))
		ost := (*interfaceType)(unsafe.Pointer(ort))
		st.Methods = ost.Methods
	case reflect.Map:
		st := (*mapType)(unsafe.Pointer(rt))
		ost := (*mapType)(unsafe.Pointer(ort))
		cloneMap(st, ost)
	case reflect.Func:
		st := (*funcType)(unsafe.Pointer(rt))
		ost := (*funcType)(unsafe.Pointer(ort))
		st.InCount = ost.InCount
		st.OutCount = ost.OutCount
		numIn := typ.NumIn()
		numOut := typ.NumOut()
		narg := numIn + numOut
		if narg > 0 {
			args := uncommonFuncTypeArgs(rt, narg)
			var i int
			for i = 0; i < numIn; i++ {
				args[i] = totype(styp.In(i))
			}
			for j := 0; j < numOut; j++ {
				args[i+j] = totype(styp.Out(j))
			}
		}
	}
	rt.Size_ = ort.Size_
	rt.TFlag |= tflagUncommon | tflagExtraStar | tflagNamed
	rt.Kind_ = ort.Kind_
	rt.Align_ = ort.Align_
	rt.FieldAlign_ = ort.FieldAlign_
	rt.GCData = ort.GCData
	rt.PtrBytes = ort.PtrBytes
	rt.Equal = ort.Equal
	//rt.Str = resolveReflectName(rtype_nameOff(ort, ort.Str))
	if isRegularMemory(typ) {
		rt.TFlag |= tflagRegularMemory
	}
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
				Name: resolveReflectName(rtype_nameOff(ort, m.Name)),
				Typ:  resolveReflectType(rtype_typeOff(ort, m.Typ)),
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
		st.InCount = ost.InCount
		st.OutCount = ost.OutCount
		if narg > 0 {
			args := make([]*rtype, narg, narg)
			fnoff = uint32(unsafe.Sizeof((*rtype)(nil))) * uint32(narg)
			var i int
			for i = 0; i < numIn; i++ {
				args[i] = totype(styp.In(i))
			}
			for j := 0; j < numOut; j++ {
				args[i+j] = totype(styp.Out(j))
			}
			copy(tt.Elem().Field(2).Slice(0, narg).Interface().([]*rtype), args)
		}
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
	rt.Str = resolveReflectName(rtype_nameOff(ort, ort.Str))
	ut := (*uncommonType)(unsafe.Pointer(tt.Elem().Field(1).UnsafeAddr()))
	ut.Mcount = uint16(mcount)
	ut.Xcount = uint16(xcount)
	ut.Moff = uint32(unsafe.Sizeof(uncommonType{}))
	if skind == reflect.Interface {
		return rt, nil
	} else if skind == reflect.Func {
		ut.Moff += fnoff
		return rt, tt.Elem().Field(3).Slice(0, mcount).Interface().([]method)
	}
	return rt, tt.Elem().Field(2).Slice(0, mcount).Interface().([]method)
}

func NamedTypeOf(pkgpath string, name string, from reflect.Type) reflect.Type {
	rt, _ := newType(pkgpath, name, from, 0, 0)
	setTypeName(rt, pkgpath, name)
	return toType(rt)
}

//go:linkname typesByString reflect.typesByString
func typesByString(s string) []*rtype

//go:linkname typelinks reflect.typelinks
func typelinks() (sections []unsafe.Pointer, offset [][]int32)

//go:linkname rtypeOff reflect.rtypeOff
func rtypeOff(section unsafe.Pointer, off int32) *rtype

func TypeLinks() []reflect.Type {
	var r []reflect.Type
	sections, offset := typelinks()
	for i, offs := range offset {
		rodata := sections[i]
		for _, off := range offs {
			typ := (*rtype)(resolveTypeOff(unsafe.Pointer(rodata), off))
			r = append(r, toType(typ))
		}
	}
	return r
}

func TypesByString(s string) []reflect.Type {
	sections, offset := typelinks()
	var ret []reflect.Type

	for offsI, offs := range offset {
		section := sections[offsI]

		// We are looking for the first index i where the string becomes >= s.
		// This is a copy of sort.Search, with f(h) replaced by (*typ[h].String() >= s).
		i, j := 0, len(offs)
		for i < j {
			h := i + (j-i)/2 // avoid overflow when computing h
			// i ≤ h < j
			typ := toType(rtypeOff(section, offs[h]))
			if !(typ.String() >= s) {
				i = h + 1 // preserves f(i-1) == false
			} else {
				j = h // preserves f(j) == true
			}
		}
		// i == j, f(i-1) == false, and f(j) (= f(i)) == true  =>  answer is i.

		// Having found the first, linear scan forward to find the last.
		// We could do a second binary search, but the caller is going
		// to do a linear scan anyway.
		for j := i; j < len(offs); j++ {
			typ := toType(rtypeOff(section, offs[j]))
			if typ.String() != s {
				break
			}
			ret = append(ret, typ)
		}
	}
	return ret
}

func DumpType(w io.Writer, typ reflect.Type) {
	rt := totype(typ)
	fmt.Fprintf(w, "%#v\n", rt)
	for _, m := range rtypeMethods(rt) {
		fmt.Fprintf(w, "%v (%v)\t\t%#v\n",
			rtype_nameOff(rt, m.Name).Name(),
			toType(rtype_typeOff(rt, m.Mtyp)),
			m)
	}
}

func rtypeMethodX(t *rtype, i int) (m reflect.Method) {
	if reflect.Kind(t.Kind()) == reflect.Interface {
		return toType(t).Method(i)
	}
	methods := rtypeMethods(t)
	if i < 0 || i >= len(methods) {
		panic("reflect: Method index out of range")
	}
	p := methods[i]
	pname := rtype_nameOff(t, p.Name)
	m.Name = pname.Name()
	m.Index = i
	fl := flag(reflect.Func)
	if t.TFlag&tflagUserMethod != 0 {
		fl |= flagIndir
	}
	mtyp := rtype_typeOff(t, p.Mtyp)
	if mtyp == nil {
		return
	}
	ft := (*funcType)(unsafe.Pointer(mtyp))
	ins := funcTypeIn(ft)
	in := make([]reflect.Type, 0, 1+len(ins))
	in = append(in, toType(t))
	for _, arg := range ins {
		in = append(in, toType(arg))
	}
	outs := funcTypeOut(ft)
	out := make([]reflect.Type, 0, len(outs))
	for _, ret := range outs {
		out = append(out, toType(ret))
	}
	mt := reflect.FuncOf(in, out, ft.IsVariadic())
	m.Type = mt
	tfn := rtype_textOff(t, p.Tfn)
	fn := unsafe.Pointer(&tfn)
	m.Func = toValue(Value{totype(mt), fn, fl})
	return m
}

func rtypeMethodByNameX(t *rtype, name string) (m reflect.Method, ok bool) {
	if reflect.Kind(t.Kind()) == reflect.Interface {
		return toType(t).MethodByName(name)
	}
	if ut := t.Uncommon(); ut != nil {
		for i, p := range ut.Methods() {
			if rtype_nameOff(t, p.Name).Name() == name {
				return rtypeMethodX(t, i), true
			}
		}
	}
	return reflect.Method{}, false
}

func setTypeName(t *rtype, pkgpath string, name string) {
	if pkgpath == "" && name == "" {
		return
	}
	exported := isExported(name)
	if pkgpath != "" {
		_, f := path.Split(pkgpath)
		name = f + "." + name
	}
	t.TFlag |= tflagNamed | tflagExtraStar
	t.Str = resolveReflectName(newName("*"+name, "", exported))
	if t.TFlag&tflagUncommon == tflagUncommon {
		t.Uncommon().PkgPath = resolveReflectName(newName(pkgpath, "", false))
	}
	switch reflect.Kind(t.Kind()) {
	case reflect.Struct:
		st := (*structType)(toKindType(t))
		st.PkgPath = newName(pkgpath, "", false)
	case reflect.Interface:
		st := (*interfaceType)(toKindType(t))
		st.PkgPath = newName(pkgpath, "", false)
	}
}

func (ctx *Context) StructOf(fields []reflect.StructField) reflect.Type {
	var anonymous []int
	underscore := make(map[int]name)
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
				underscore[i] = newName("_", string(f.Tag), false)
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
		st.Fields[i].Name = n
	}
	str := typ.String()
	if v, ok := ctx.structLookupCache.Load(str); ok {
		ts := v.([]reflect.Type)
		for _, t := range ts {
			if haveIdenticalType(totype(t), totype(typ), true) {
				return t
			}
		}
		ctx.structLookupCache.Store(str, append(ts, typ))
	} else {
		ctx.structLookupCache.Store(str, []reflect.Type{typ})
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
	(*f.Name.Bytes) |= 1 << 3
}
