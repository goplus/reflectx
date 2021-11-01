package reflectx

import (
	"reflect"
	"unsafe"
)

func AllNumMethod(typ reflect.Type) int {
	return totype(typ).AllNumMethod()
}

func AllMethod(typ reflect.Type, i int) reflect.Method {
	return totype(typ).AllMethod(i)
}

func (t *rtype) AllNumMethod() int {
	return len(t.methods())
}

func (t *rtype) AllMethod(i int) (m reflect.Method) {
	if t.Kind() == reflect.Interface {
		return toType(t).Method(i)
	}
	methods := t.methods()
	if i < 0 || i >= len(methods) {
		panic("reflect: Method index out of range")
	}
	p := methods[i]
	pname := t.nameOff(p.name)
	m.Name = pname.name()
	m.Index = i
	fl := flag(reflect.Func)
	mtyp := t.typeOff(p.mtyp)
	if mtyp == nil {
		return
	}
	ft := (*funcType)(unsafe.Pointer(mtyp))
	in := make([]reflect.Type, 0, 1+len(ft.in()))
	in = append(in, toType(t))
	for _, arg := range ft.in() {
		in = append(in, toType(arg))
	}
	out := make([]reflect.Type, 0, len(ft.out()))
	for _, ret := range ft.out() {
		out = append(out, toType(ret))
	}
	mt := reflect.FuncOf(in, out, ft.IsVariadic())
	m.Type = mt
	tfn := t.textOff(p.tfn)
	fn := unsafe.Pointer(&tfn)
	m.Func = toValue(Value{totype(mt), fn, fl})
	return m
}

func (t *rtype) AllMethodByName(name string) (m reflect.Method, ok bool) {
	if t.Kind() == reflect.Interface {
		return toType(t).MethodByName(name)
	}
	ut := t.uncommon()
	if ut == nil {
		return reflect.Method{}, false
	}
	for i, p := range ut.methods() {
		if t.nameOff(p.name).name() == name {
			return t.AllMethod(i), true
		}
	}
	return reflect.Method{}, false
}
