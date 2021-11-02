package reflectx

import (
	"reflect"
	"unsafe"
)

func NumMethodX(typ reflect.Type) int {
	return totype(typ).NumMethodX()
}

func MethodX(typ reflect.Type, i int) reflect.Method {
	return totype(typ).MethodX(i)
}

func (t *rtype) NumMethodX() int {
	return len(t.methods())
}

func (t *rtype) MethodX(i int) (m reflect.Method) {
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

func (t *rtype) MethodByNameX(name string) (m reflect.Method, ok bool) {
	if t.Kind() == reflect.Interface {
		return toType(t).MethodByName(name)
	}
	ut := t.uncommon()
	if ut == nil {
		return reflect.Method{}, false
	}
	for i, p := range ut.methods() {
		if t.nameOff(p.name).name() == name {
			return t.MethodX(i), true
		}
	}
	return reflect.Method{}, false
}

// Field returns the i'th field of the struct v.
// It panics if v's Kind is not Struct or i is out of range.
func FieldX(v reflect.Value, i int) reflect.Value {
	mustBe("reflect.Value.Field", v, reflect.Struct)
	rv := tovalue(&v)
	tt := (*structType)(unsafe.Pointer(rv.typ))
	if uint(i) >= uint(len(tt.fields)) {
		panic("reflect: Field index out of range")
	}
	field := &tt.fields[i]
	typ := field.typ

	// Inherit permission bits from v, but clear flagEmbedRO.
	fl := rv.flag&(flagStickyRO|flagIndir|flagAddr) | flag(typ.Kind())
	// Using an unexported field forces flagRO.
	// if !field.name.isExported() {
	// 	if field.embedded() {
	// 		fl |= flagEmbedRO
	// 	} else {
	// 		fl |= flagStickyRO
	// 	}
	// }
	// Either flagIndir is set and v.ptr points at struct,
	// or flagIndir is not set and v.ptr is the actual struct data.
	// In the former case, we want v.ptr + offset.
	// In the latter case, we must have field.offset = 0,
	// so v.ptr + field.offset is still the correct address.
	ptr := add(rv.ptr, field.offset(), "same as non-reflect &v.field")
	return toValue(Value{typ, ptr, fl})
}

func FieldByIndexX(v reflect.Value, index []int) reflect.Value {
	if len(index) == 1 {
		return FieldX(v, index[0])
	}
	mustBe("reflect.Value.FieldByIndex", v, reflect.Struct)
	for i, x := range index {
		if i > 0 {
			if v.Kind() == reflect.Ptr && v.Type().Elem().Kind() == reflect.Struct {
				if v.IsNil() {
					panic("reflect: indirection through nil pointer to embedded struct")
				}
				v = v.Elem()
			}
		}
		v = FieldX(v, x)
	}
	return v
}

func mustBe(method string, v reflect.Value, kind reflect.Kind) {
	if v.Kind() != kind {
		panic(&reflect.ValueError{method, v.Kind()})
	}
}

func FieldByNameX(v reflect.Value, name string) reflect.Value {
	mustBe("reflect.Value.FieldByName", v, reflect.Struct)
	if f, ok := v.Type().FieldByName(name); ok {
		return FieldByIndexX(v, f.Index)
	}
	return reflect.Value{}
}

// FieldByNameFunc returns the struct field with a name
// that satisfies the match function.
// It panics if v's Kind is not struct.
// It returns the zero Value if no field was found.
func FieldByNameFuncX(v reflect.Value, match func(string) bool) reflect.Value {
	if f, ok := v.Type().FieldByNameFunc(match); ok {
		return FieldByIndexX(v, f.Index)
	}
	return reflect.Value{}
}
