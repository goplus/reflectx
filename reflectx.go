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

func NamedStructOf(pkgpath string, name string, fields []reflect.StructField) reflect.Type {
	return Default.NamedStructOf(pkgpath, name, fields)
}

func (ctx *Context) NamedStructOf(pkgpath string, name string, fields []reflect.StructField) reflect.Type {
	return NamedTypeOf(pkgpath, name, ctx.StructOf(fields))
}

func SetTypeName(typ reflect.Type, pkgpath string, name string) {
	setTypeName(totype(typ), pkgpath, name)
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

func isExported(name string) bool {
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

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
	return Default.StructOf(fields)
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
		st.Elem = totype(elem)
	case reflect.Slice:
		st := (*sliceType)(toKindType(rt))
		st.Elem = totype(elem)
	case reflect.Array:
		st := (*arrayType)(toKindType(rt))
		st.Elem = totype(elem)
	case reflect.Map:
		st := (*mapType)(toKindType(rt))
		st.Elem = totype(elem)
	case reflect.Chan:
		st := (*chanType)(toKindType(rt))
		st.Elem = totype(elem)
	default:
		panic("reflect: Elem of invalid type " + typ.String())
	}
}
