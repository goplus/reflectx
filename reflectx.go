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

func SetTypeName(typ reflect.Type, pkgpath string, name string) {
	setTypeName(totype(typ), pkgpath, name)
}

func setTypeName(t *rtype, pkgpath string, name string) {
	exported := isExported(name)
	if pkgpath != "" {
		_, f := path.Split(pkgpath)
		name = f + "." + name
	}
	t.tflag |= tflagNamed | tflagExtraStar | tflagUncommon
	t.str = resolveReflectName(newName("*"+name, "", exported))
	if t.tflag&tflagUncommon == tflagUncommon {
		toUncommonType(t).pkgPath = resolveReflectName(newName(pkgpath, "", false))
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
	DisableStructOfExportAllField bool
)

func StructOf(fields []reflect.StructField) reflect.Type {
	var anonymous []int
	fs := make([]reflect.StructField, len(fields))
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if f.Anonymous {
			anonymous = append(anonymous, i)
			f.Anonymous = false
			if f.Name == "" {
				f.Name = typeName(f.Type)
			}
		}
		fs[i] = f
	}
	typ := reflect.StructOf(fs)
	rt := totype(typ)
	st := toStructType(rt)
	for _, i := range anonymous {
		st.fields[i].offsetEmbed |= 1
	}
	if !DisableStructOfExportAllField {
		for i := 0; i < len(fs); i++ {
			f := fs[i]
			st.fields[i].name = newName(f.Name, string(f.Tag), true)
		}
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
)
