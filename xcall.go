package reflectx

import (
	"reflect"
)

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
