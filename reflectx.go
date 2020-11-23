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
	"fmt"
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
	namedMagic     = "Gop_Temp_"
	named          = make(map[string]reflect.Type)
	typEmptyStruct = reflect.StructOf(nil)
)

// fnv1 incorporates the list of bytes into the hash x using the FNV-1 hash function.
func fnv1(x uint32, list string) uint32 {
	for _, b := range list {
		x = x*16777619 ^ uint32(b)
	}
	return x
}

func hashName(pkgpath string, name string) string {
	return fmt.Sprintf("Gop_Unused_%d_%d", fnv1(0, pkgpath), fnv1(0, name))
}

func NamedStructOf(pkgpath string, name string, fields []reflect.StructField) reflect.Type {
	typ := StructOf(append(append([]reflect.StructField{}, fields...),
		reflect.StructField{
			Name: hashName(pkgpath, name),
			Type: typEmptyStruct,
		}))
	str := typ.String()
	if t, ok := named[str]; ok {
		return t
	}
	named[str] = typ
	v := reflect.Zero(typ)
	rt := (*Value)(unsafe.Pointer(&v)).typ
	st := toStructType(rt)
	st.fields = st.fields[:len(st.fields)-1]
	rt.str = toNameOff(pkgpath, name, "")
	rt.tflag |= tflagNamed
	setUncommonTypePkgPath(rt, toNameOff("", pkgpath, ""))
	return typ
}

func toNameOff(pkgpath string, name string, tag string) nameOff {
	exported := isExported(name)
	if pkgpath != "" {
		_, f := path.Split(pkgpath)
		name = f + "." + name
	}
	return resolveReflectName(newName(name, tag, exported))
}

func isExported(name string) bool {
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

func totype(typ reflect.Type) *rtype {
	v := reflect.Zero(typ)
	rt := (*Value)(unsafe.Pointer(&v)).typ
	return rt
}

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
	v := reflect.Zero(typ)
	rt := (*Value)(unsafe.Pointer(&v)).typ
	st := toStructType(rt)
	for _, i := range anonymous {
		st.fields[i].offsetEmbed |= 1
	}
	return typ
}
