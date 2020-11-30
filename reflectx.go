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
	ntypeMap       = make(map[reflect.Type]*Named)
	typEmptyStruct = reflect.StructOf(nil)
)

type TypeKind int

const (
	TkInvalid TypeKind = iota
	TkStruct
	TkType
	TkInterface
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
	typ := StructOf(append(append([]reflect.StructField{}, fields...),
		reflect.StructField{
			Name: unusedName(),
			Type: typEmptyStruct,
		}))
	nt := &Named{Name: name, PkgPath: pkgpath, Type: typ, Kind: TkStruct}
	ntypeMap[typ] = nt
	rt := totype(typ)
	st := toStructType(rt)
	st.fields = st.fields[:len(st.fields)-1]
	setTypeName(rt, pkgpath, name)
	return typ
}

var (
	index int
)

func unusedName() string {
	index++
	return fmt.Sprintf("Gop_unused_%v", index)
}

func emptyType() reflect.Type {
	typ := reflect.StructOf([]reflect.StructField{
		reflect.StructField{
			Name: unusedName(),
			Type: typEmptyStruct,
		}})
	rt := totype(typ)
	st := toStructType(rt)
	st.fields = st.fields[:len(st.fields)-1]
	st.str = resolveReflectName(newName("unused", "", false))
	return typ
}

func NamedTypeOf(pkgpath string, name string, from reflect.Type) (typ reflect.Type) {
	switch from.Kind() {
	case reflect.Array:
		typ = reflect.ArrayOf(from.Len(), emptyType())
		dst := totype(typ)
		src := totype(from)
		copyType(dst, src)
		d := (*arrayType)(unsafe.Pointer(dst))
		s := (*arrayType)(unsafe.Pointer(src))
		d.elem = s.elem
		d.slice = s.slice
		d.len = s.len
		setTypeName(dst, pkgpath, name)
	case reflect.Slice:
		typ = reflect.SliceOf(emptyType())
		dst := totype(typ)
		src := totype(from)
		copyType(dst, src)
		d := (*sliceType)(unsafe.Pointer(dst))
		s := (*sliceType)(unsafe.Pointer(src))
		d.elem = s.elem
		setTypeName(dst, pkgpath, name)
	case reflect.Map:
		typ = reflect.MapOf(emptyType(), emptyType())
		dst := totype(typ)
		src := totype(from)
		copyType(dst, src)
		d := (*mapType)(unsafe.Pointer(dst))
		s := (*mapType)(unsafe.Pointer(src))
		d.key = s.key
		d.elem = s.elem
		d.bucket = s.bucket
		d.hasher = s.hasher
		d.keysize = s.keysize
		d.valuesize = s.valuesize
		d.bucketsize = s.bucketsize
		d.flags = s.flags
		dst.str = resolveReflectName(newName(name, "", isExported(name)))
		setTypeName(dst, pkgpath, name)
	case reflect.Ptr:
		typ = reflect.PtrTo(emptyType())
		dst := totype(typ)
		src := totype(from)
		copyType(dst, src)
		d := (*ptrType)(unsafe.Pointer(dst))
		s := (*ptrType)(unsafe.Pointer(src))
		d.elem = s.elem
		setTypeName(dst, pkgpath, name)
	case reflect.Chan:
		typ = reflect.ChanOf(from.ChanDir(), emptyType())
		dst := totype(typ)
		src := totype(from)
		copyType(dst, src)
		d := (*chanType)(unsafe.Pointer(dst))
		s := (*chanType)(unsafe.Pointer(src))
		d.elem = s.elem
		d.dir = s.dir
		setTypeName(dst, pkgpath, name)
	case reflect.Func:
		numIn := from.NumIn()
		in := make([]reflect.Type, numIn, numIn)
		for i := 0; i < numIn; i++ {
			in[i] = from.In(i)
		}
		numOut := from.NumOut()
		out := make([]reflect.Type, numOut, numOut)
		for i := 0; i < numOut; i++ {
			out[i] = from.Out(i)
		}
		out = append(out, emptyType())
		typ = reflect.FuncOf(in, out, from.IsVariadic())
		dst := totype(typ)
		src := totype(from)
		d := (*funcType)(unsafe.Pointer(dst))
		s := (*funcType)(unsafe.Pointer(src))
		d.inCount = s.inCount
		d.outCount = s.outCount
		setTypeName(dst, pkgpath, name)
	default:
		var fields []reflect.StructField
		if from.Kind() == reflect.Struct {
			for i := 0; i < from.NumField(); i++ {
				fields = append(fields, from.Field(i))
			}
		}
		fields = append(fields, reflect.StructField{
			Name: hashName(pkgpath, name),
			Type: typEmptyStruct,
		})
		typ = StructOf(fields)
		rt := totype(typ)
		st := toStructType(rt)
		st.fields = st.fields[:len(st.fields)-1]
		copyType(rt, totype(from))
		setTypeName(rt, pkgpath, name)
	}
	nt := &Named{Name: name, PkgPath: pkgpath, Type: typ, From: from, Kind: TkType}
	ntypeMap[typ] = nt
	return typ
}

func setTypeName(t *rtype, pkgpath string, name string) {
	exported := isExported(name)
	if pkgpath != "" {
		_, f := path.Split(pkgpath)
		name = f + "." + name
	}
	t.tflag |= tflagNamed | tflagExtraStar
	t.str = resolveReflectName(newName("*"+name, "", exported))
	switch t.Kind() {
	case reflect.Array:
	case reflect.Slice:
	case reflect.Map:
	case reflect.Ptr:
	case reflect.Func:
	case reflect.Chan:
	default:
		t.tflag |= tflagUncommon
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
}

func isExported(name string) bool {
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

func totype(typ reflect.Type) *rtype {
	v := reflect.Zero(typ)
	rt1 := (*Value)(unsafe.Pointer(&v)).typ
	return rt1
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

// fnv1 incorporates the list of bytes into the hash x using the FNV-1 hash function.
func fnv1(x uint32, list string) uint32 {
	for _, b := range list {
		x = x*16777619 ^ uint32(b)
	}
	return x
}

func hashName(pkgpath string, name string) string {
	return fmt.Sprintf("Gop_Named_%d_%d", fnv1(0, pkgpath), fnv1(0, name))
}
