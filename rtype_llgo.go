//go:build llgo

package reflectx

import (
	"path"
	"reflect"
	"strconv"
	"unsafe"
)

func NamedTypeOf(pkgpath string, name string, from reflect.Type) reflect.Type {
	panic("TODO: NamedTypeOf")
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
	t.TFlag |= tflagNamed | tflagExtraStar
	t.Str_ = "*" + name
	if t.TFlag&tflagUncommon == tflagUncommon {
		t.Uncommon().PkgPath_ = pkgpath
	}
	switch reflect.Kind(t.Kind()) {
	case reflect.Struct:
		st := (*structType)(toKindType(t))
		st.PkgPath_ = pkgpath
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
			if p.Name_ == name {
				return rtypeMethodX(t, i), true
			}
		}
	}
	return reflect.Method{}, false
}

//go:linkname closureOf reflect.closureOf
func closureOf(ftyp *funcType) *rtype

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
