/*
 Copyright 2020 The GoPlus Authors (goplus.org)

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package reflectx

import (
	"path"
	"reflect"
	"strconv"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

func Field(s reflect.Value, i int) reflect.Value {
	v := s.Field(i)
	canSet(&v)
	return v
}

func FieldByIndex(s reflect.Value, index []int) reflect.Value {
	v := s.FieldByIndex(index)
	canSet(&v)
	return v
}

func FieldByName(s reflect.Value, name string) reflect.Value {
	v := s.FieldByName(name)
	canSet(&v)
	return v
}

func FieldByNameFunc(s reflect.Value, match func(name string) bool) reflect.Value {
	v := s.FieldByNameFunc(match)
	canSet(&v)
	return v
}

func canSet(v *reflect.Value) {
	(*Value)(unsafe.Pointer(v)).flag &= ^flagRO
}

func CanSet(v reflect.Value) reflect.Value {
	if !v.CanSet() {
		(*Value)(unsafe.Pointer(&v)).flag &= ^flagRO
	}
	return v
}

func typeName(typ reflect.Type) string {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ.Name()
}

var (
	ntypeMap = make(map[reflect.Type]*Named)
)

type TypeKind int

const (
	TkInvalid TypeKind = 1 << iota
	TkType
	TkMethod
)

type Named struct {
	Name    string
	PkgPath string
	Type    reflect.Type
	From    reflect.Type
	Kind    TypeKind
}

func IsNamed(typ reflect.Type) bool {
	_, ok := ntypeMap[typ]
	return ok
}

func ToNamed(typ reflect.Type) (t *Named, ok bool) {
	t, ok = ntypeMap[typ]
	return
}

func NamedStructOf(pkgpath string, name string, fields []reflect.StructField) reflect.Type {
	return NamedTypeOf(pkgpath, name, StructOf(fields))
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
	t.tflag |= tflagNamed | tflagExtraStar
	t.str = resolveReflectName(newName("*"+name, "", exported))
	if t.tflag&tflagUncommon == tflagUncommon {
		toUncommonType(t).pkgPath = resolveReflectName(newName(pkgpath, "", false))
	}
	switch t.Kind() {
	case reflect.Struct:
		st := (*structType)(toKindType(t))
		st.pkgPath = newName(pkgpath, "", false)
	case reflect.Interface:
		st := (*interfaceType)(toKindType(t))
		st.pkgPath = newName(pkgpath, "", false)
	}
}

func copyType(dst *rtype, src *rtype) {
	dst.size = src.size
	dst.kind = src.kind
	dst.equal = src.equal
	dst.align = src.align
	dst.fieldAlign = src.fieldAlign
	dst.tflag = src.tflag
	dst.gcdata = src.gcdata
	dst.ptrdata = src.ptrdata
}

func isExported(name string) bool {
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

var (
	EnableStructOfExportAllField bool
)

var (
	structLookupCache = make(map[string][]reflect.Type)
)

func checkFields(t1, t2 reflect.Type) bool {
	n1 := t1.NumField()
	n2 := t2.NumField()
	if n1 != n2 {
		return false
	}
	for i := 0; i < n1; i++ {
		f1 := t1.Field(i)
		f2 := t2.Field(i)
		if f1.Name != f2.Name ||
			f1.PkgPath != f2.PkgPath ||
			f1.Anonymous != f2.Anonymous ||
			f1.Type != f2.Type ||
			f1.Offset != f2.Offset {
			return false
		}
	}
	return true
}

func StructOf(fields []reflect.StructField) reflect.Type {
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
		st.fields[i].offsetEmbed |= 1
	}
	for i, n := range underscore {
		st.fields[i].name = n
	}
	if EnableStructOfExportAllField {
		for i := 0; i < len(fs); i++ {
			f := fs[i]
			st.fields[i].name = newName(f.Name, string(f.Tag), true)
		}
	}
	str := typ.String()
	if ts, ok := structLookupCache[str]; ok {
		for _, t := range ts {
			if haveIdenticalType(t, typ, true) {
				return t
			}
		}
		ts = append(ts, typ)
	} else {
		structLookupCache[str] = []reflect.Type{typ}
	}
	return typ
}

// fnv1 incorporates the list of bytes into the hash x using the FNV-1 hash function.
func fnv1(x uint32, list string) uint32 {
	for _, b := range list {
		x = x*16777619 ^ uint32(b)
	}
	return x
}

func SetValue(v reflect.Value, x reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(x.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(x.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(x.Uint())
	case reflect.Uintptr:
		v.SetUint(x.Uint())
	case reflect.Float32, reflect.Float64:
		v.SetFloat(x.Float())
	case reflect.Complex64, reflect.Complex128:
		v.SetComplex(x.Complex())
	case reflect.String:
		v.SetString(x.String())
	case reflect.UnsafePointer:
		v.SetPointer(unsafe.Pointer(x.Pointer()))
	default:
		v.Set(x)
	}
}

var (
	tyEmptyInterface    = reflect.TypeOf((*interface{})(nil)).Elem()
	tyEmptyInterfacePtr = reflect.TypeOf((*interface{})(nil))
	tyEmptyStruct       = reflect.TypeOf((*struct{})(nil)).Elem()
	tyErrorInterface    = reflect.TypeOf((*error)(nil)).Elem()
)

func SetElem(typ reflect.Type, elem reflect.Type) {
	rt := totype(typ)
	switch typ.Kind() {
	case reflect.Ptr:
		st := (*ptrType)(toKindType(rt))
		st.elem = totype(elem)
	case reflect.Slice:
		st := (*sliceType)(toKindType(rt))
		st.elem = totype(elem)
	case reflect.Array:
		st := (*arrayType)(toKindType(rt))
		st.elem = totype(elem)
	case reflect.Map:
		st := (*mapType)(toKindType(rt))
		st.elem = totype(elem)
	case reflect.Chan:
		st := (*chanType)(toKindType(rt))
		st.elem = totype(elem)
	default:
		panic("reflect: Elem of invalid type " + typ.String())
	}
}

func typeId(typ reflect.Type) string {
	var id string
	if path := typ.PkgPath(); path != "" {
		id = path + "."
	}
	return id + typ.Name()
}

type replaceTypeContext struct {
	checking map[reflect.Type]bool
}

func ReplaceType(pkg string, typ reflect.Type, m map[string]reflect.Type) (rtyp reflect.Type, changed bool) {
	ctx := &replaceTypeContext{make(map[reflect.Type]bool)}
	return ctx.replace(pkg, typ, m)
}

func (ctx *replaceTypeContext) replace(pkg string, typ reflect.Type, m map[string]reflect.Type) (rtyp reflect.Type, changed bool) {
	if ctx.checking[typ] {
		return
	}
	ctx.checking[typ] = true
	rt := totype(typ)
	switch typ.Kind() {
	case reflect.Struct:
		if typ.PkgPath() != pkg {
			return
		}
		st := (*structType)(toKindType(rt))
		for i := 0; i < len(st.fields); i++ {
			et := toType(st.fields[i].typ)
			if t, ok := m[typeId(et)]; ok {
				st.fields[i].typ = totype(t)
				changed = true
			} else {
				if rtyp, ok := ctx.replace(pkg, et, m); ok {
					changed = true
					st.fields[i].typ = totype(rtyp)
				}
			}
		}
		if changed {
			return toType(rt), true
		}
	case reflect.Ptr:
		st := (*ptrType)(toKindType(rt))
		et := toType(st.elem)
		if t, ok := m[typeId(et)]; ok {
			st.elem = totype(t)
			return reflect.PtrTo(t), true
		} else {
			if rtyp, ok := ctx.replace(pkg, et, m); ok {
				return reflect.PtrTo(rtyp), true
			}
		}
	case reflect.Slice:
		st := (*sliceType)(toKindType(rt))
		et := toType(st.elem)
		if t, ok := m[typeId(et)]; ok {
			st.elem = totype(t)
			return reflect.SliceOf(t), true
		} else {
			if rtyp, ok := ctx.replace(pkg, et, m); ok {
				return reflect.SliceOf(rtyp), true
			}
		}
	case reflect.Array:
		st := (*arrayType)(toKindType(rt))
		et := toType(st.elem)
		if t, ok := m[typeId(et)]; ok {
			st.elem = totype(t)
			return reflect.ArrayOf(int(st.len), t), true
		} else {
			if rtyp, ok := ctx.replace(pkg, et, m); ok {
				return reflect.ArrayOf(int(st.len), rtyp), true
			}
		}
	case reflect.Map:
		st := (*mapType)(toKindType(rt))
		kt := toType(st.key)
		et := toType(st.elem)
		if t, ok := m[typeId(kt)]; ok {
			kt = t
			changed = true
		} else {
			if rtyp, ok := ctx.replace(pkg, kt, m); ok {
				kt = rtyp
				changed = true
			}
		}
		if t, ok := m[typeId(et)]; ok {
			et = t
			changed = true
		} else {
			if rtyp, ok := ctx.replace(pkg, et, m); ok {
				et = rtyp
				changed = true
			}
		}
		if changed {
			return reflect.MapOf(kt, et), true
		}
	case reflect.Chan:
		st := (*chanType)(toKindType(rt))
		et := toType(st.elem)
		if t, ok := m[typeId(et)]; ok {
			st.elem = totype(t)
			return reflect.ChanOf(typ.ChanDir(), t), true
		} else {
			if rtyp, ok := ctx.replace(pkg, et, m); ok {
				return reflect.ChanOf(typ.ChanDir(), rtyp), true
			}
		}
	case reflect.Func:
		st := (*funcType)(toKindType(rt))
		in := st.in()
		out := st.out()
		for i := 0; i < len(in); i++ {
			et := toType(in[i])
			if t, ok := m[typeId(et)]; ok {
				in[i] = totype(t)
				changed = true
			} else {
				if rtyp, ok := ctx.replace(pkg, et, m); ok {
					in[i] = totype(rtyp)
					changed = true
				}
			}
		}
		for i := 0; i < len(out); i++ {
			et := toType(out[i])
			if t, ok := m[typeId(et)]; ok {
				out[i] = totype(t)
				changed = true
			} else {
				if rtyp, ok := ctx.replace(pkg, et, m); ok {
					out[i] = totype(rtyp)
					changed = true
				}
			}
		}
		if changed {
			ins := make([]reflect.Type, len(in))
			for i := 0; i < len(in); i++ {
				ins[i] = toType(in[i])
			}
			outs := make([]reflect.Type, len(out))
			for i := 0; i < len(out); i++ {
				outs[i] = toType(out[i])
			}
			return reflect.FuncOf(ins, outs, typ.IsVariadic()), true
		}
	case reflect.Interface:
		if typ.PkgPath() != pkg {
			return
		}
		if typ == tyErrorInterface {
			return
		}
		st := (*interfaceType)(toKindType(rt))
		for i := 0; i < len(st.methods); i++ {
			tt := typ.Method(i).Type
			if t, ok := m[typeId(tt)]; ok {
				st.methods[i].typ = resolveReflectType(totype(t))
				changed = true
			} else if rtyp, ok := ctx.replace(pkg, tt, m); ok {
				st.methods[i].typ = resolveReflectType(totype(rtyp))
				changed = true
			}
		}
		if changed {
			return toType(rt), true
		}
	}
	return nil, false
}
