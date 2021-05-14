package reflectx

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// MakeMethod make reflect.Method for MethodOf
// - name: method name
// - pointer: flag receiver struct or pointer
// - typ: method func type without receiver
// - fn: func with receiver as first argument
func MakeMethod(name string, pointer bool, typ reflect.Type, fn func(args []reflect.Value) (result []reflect.Value)) Method {
	return Method{
		Name:    name,
		Pointer: pointer,
		Type:    typ,
		Func:    fn,
	}
}

// Method struct for MethodOf
// - name: method name
// - pointer: flag receiver struct or pointer
// - typ: method func type without receiver
// - fn: func with receiver as first argument
type Method struct {
	Name    string
	Pointer bool
	Type    reflect.Type
	Func    func([]reflect.Value) []reflect.Value
}

func extraFieldMethod(ifield int, typ reflect.Type, skip map[string]bool) (methods []Method) {
	isPtr := typ.Kind() == reflect.Ptr
	for i := 0; i < typ.NumMethod(); i++ {
		m := MethodByIndex(typ, i)
		if skip[m.Name] {
			continue
		}
		in, out := parserFuncIO(m.Type)
		mtyp := reflect.FuncOf(in[1:], out, m.Type.IsVariadic())
		var fn func(args []reflect.Value) []reflect.Value
		if isPtr {
			fn = func(args []reflect.Value) []reflect.Value {
				args[0] = args[0].Elem().Field(ifield).Addr()
				return m.Func.Call(args)
			}
		} else {
			fn = func(args []reflect.Value) []reflect.Value {
				args[0] = args[0].Field(ifield)
				if mtyp.IsVariadic() {
					return m.Func.CallSlice(args)
				}
				return m.Func.Call(args)
			}
		}
		methods = append(methods, Method{
			Name:    m.Name,
			Pointer: in[0].Kind() == reflect.Ptr,
			Type:    mtyp,
			Func:    fn,
		})
	}
	return
}

func parserFuncIO(typ reflect.Type) (in, out []reflect.Type) {
	for i := 0; i < typ.NumIn(); i++ {
		in = append(in, typ.In(i))
	}
	for i := 0; i < typ.NumOut(); i++ {
		out = append(out, typ.Out(i))
	}
	return
}

func extraPtrFieldMethod(ifield int, typ reflect.Type) (methods []Method) {
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		in, out := parserFuncIO(m.Type)
		mtyp := reflect.FuncOf(in[1:], out, m.Type.IsVariadic())
		imethod := i
		methods = append(methods, Method{
			Name: m.Name,
			Type: mtyp,
			Func: func(args []reflect.Value) []reflect.Value {
				var recv = args[0]
				if mtyp.IsVariadic() {
					return recv.Field(ifield).Method(imethod).CallSlice(args[1:])
				}
				return recv.Field(ifield).Method(imethod).Call(args[1:])
			},
		})
	}
	return
}

func extraInterfaceFieldMethod(ifield int, typ reflect.Type) (methods []Method) {
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		in, out := parserFuncIO(m.Type)
		mtyp := reflect.FuncOf(in, out, m.Type.IsVariadic())
		imethod := i
		methods = append(methods, Method{
			Name: m.Name,
			Type: mtyp,
			Func: func(args []reflect.Value) []reflect.Value {
				var recv = args[0]
				return recv.Field(ifield).Method(imethod).Call(args[1:])
			},
		})
	}
	return
}

func extractEmbedMethod(styp reflect.Type) []Method {
	var methods []Method
	for i := 0; i < styp.NumField(); i++ {
		sf := styp.Field(i)
		if !sf.Anonymous {
			continue
		}
		switch sf.Type.Kind() {
		case reflect.Interface:
			ms := extraInterfaceFieldMethod(i, sf.Type)
			methods = append(methods, ms...)
		case reflect.Ptr:
			ms := extraPtrFieldMethod(i, sf.Type)
			methods = append(methods, ms...)
		default:
			skip := make(map[string]bool)
			ms := extraFieldMethod(i, sf.Type, skip)
			for _, m := range ms {
				skip[m.Name] = true
			}
			pms := extraFieldMethod(i, reflect.PtrTo(sf.Type), skip)
			methods = append(methods, ms...)
			methods = append(methods, pms...)
		}
	}
	// ambiguous selector check
	chk := make(map[string]int)
	for _, m := range methods {
		chk[m.Name]++
	}
	var ms []Method
	for _, m := range methods {
		if chk[m.Name] == 1 {
			ms = append(ms, m)
		}
	}
	return ms
}

func UpdateField(typ reflect.Type, rmap map[reflect.Type]reflect.Type) bool {
	if rmap == nil || typ.Kind() != reflect.Struct {
		return false
	}
	rt := totype(typ)
	st := toStructType(rt)
	for i := 0; i < len(st.fields); i++ {
		t := replaceType(toType(st.fields[i].typ), rmap)
		st.fields[i].typ = totype(t)
	}
	return true
}

func UpdateMethod(typ reflect.Type, methods []Method, rmap map[reflect.Type]reflect.Type) bool {
	chk := make(map[string]int)
	for _, m := range methods {
		chk[m.Name]++
		if chk[m.Name] > 1 {
			panic(fmt.Sprintf("method redeclared: %v", m.Name))
		}
	}
	if typ.Kind() == reflect.Struct {
		ms := extractEmbedMethod(typ)
		for _, m := range ms {
			if chk[m.Name] == 1 {
				continue
			}
			methods = append(methods, m)
		}
	}
	return updateMethod(typ, methods, rmap)
}

func Reset() {
	resetTypeList()
	ntypeMap = make(map[reflect.Type]*Named)
}

func MethodOf(styp reflect.Type, maxmfunc, maxpfunc int) reflect.Type {
	typ := methodOf(styp, maxmfunc, maxpfunc)
	if n, ok := ntypeMap[styp]; ok {
		ntypeMap[typ] = &Named{Name: n.Name, PkgPath: n.PkgPath, Type: typ, From: n.From, Kind: TkType}
	}
	return typ
}

func SetMethods(styp reflect.Type, methods []Method) bool {
	chk := make(map[string]int)
	for _, m := range methods {
		chk[m.Name]++
		if chk[m.Name] > 1 {
			panic(fmt.Sprintf("method redeclared: %v", m.Name))
		}
	}
	if styp.Kind() == reflect.Struct {
		ms := extractEmbedMethod(styp)
		for _, m := range ms {
			if chk[m.Name] == 1 {
				continue
			}
			methods = append(methods, m)
		}
	}
	return setMethods(styp, methods)
}

func MakeEmptyInterface(pkgpath string, name string) reflect.Type {
	return NamedTypeOf(pkgpath, name, tyEmptyInterface)
}

func NamedInterfaceOf(pkgpath string, name string, embedded []reflect.Type, methods []Method) reflect.Type {
	typ := InterfaceOf(embedded, methods)
	return NamedTypeOf(pkgpath, name, typ)
}

var interfceLookupCache sync.Map

func InterfaceOf(embedded []reflect.Type, methods []Method) reflect.Type {
	for _, e := range embedded {
		if e.Kind() != reflect.Interface {
			panic(fmt.Errorf("interface contains embedded non-interface %v", e))
		}
		for i := 0; i < e.NumMethod(); i++ {
			m := e.Method(i)
			methods = append(methods, Method{
				Name: m.Name,
				Type: m.Type,
			})
		}
	}
	sort.Slice(methods, func(i, j int) bool {
		n := strings.Compare(methods[i].Name, methods[j].Name)
		if n == 0 && methods[i].Type != methods[j].Type {
			panic(fmt.Sprintf("duplicate method %v", methods[j].Name))
		}
		return n < 0
	})
	rt, _ := newType("", "", tyEmptyInterface, 0, 0)
	st := (*interfaceType)(toKindType(rt))
	st.methods = nil
	var info []string
	var lastname string
	for _, m := range methods {
		if m.Name == lastname {
			continue
		}
		lastname = m.Name
		st.methods = append(st.methods, imethod{
			name: resolveReflectName(newName(m.Name, "", true)),
			typ:  resolveReflectType(totype(m.Type)),
		})
		info = append(info, methodStr(m.Name, m.Type))
	}
	var str string
	if len(info) > 0 {
		str = fmt.Sprintf("*interface { %v }", strings.Join(info, "; "))
	} else {
		str = "*interface {}"
	}
	if t, ok := interfceLookupCache.Load(str); ok {
		return t.(reflect.Type)
	}
	rt.str = resolveReflectName(newName(str, "", false))
	ti, _ := interfceLookupCache.LoadOrStore(str, toType(rt))
	return ti.(reflect.Type)
}

func methodStr(name string, typ reflect.Type) string {
	return strings.Replace(typ.String(), "func", name, 1)
}

func toElem(typ reflect.Type) reflect.Type {
	if typ.Kind() == reflect.Ptr {
		return typ.Elem()
	}
	return typ
}

func toElemValue(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		return v.Elem()
	}
	return v
}

func replaceType(typ reflect.Type, rmap map[reflect.Type]reflect.Type) reflect.Type {
	var fnx func(t reflect.Type) (reflect.Type, bool)
	fnx = func(t reflect.Type) (reflect.Type, bool) {
		for k, v := range rmap {
			if k.String() == t.String() {
				return v, true
			}
		}
		switch t.Kind() {
		case reflect.Ptr:
			if e, ok := fnx(t.Elem()); ok {
				return reflect.PtrTo(e), true
			}
		case reflect.Slice:
			if e, ok := fnx(t.Elem()); ok {
				return reflect.SliceOf(e), true
			}
		case reflect.Array:
			if e, ok := fnx(t.Elem()); ok {
				return reflect.ArrayOf(t.Len(), e), true
			}
		case reflect.Map:
			k, ok1 := fnx(t.Key())
			v, ok2 := fnx(t.Elem())
			if ok1 || ok2 {
				return reflect.MapOf(k, v), true
			}
		}
		return t, false
	}
	if r, ok := fnx(typ); ok {
		return r
	}
	return typ
}

func parserMethodType(mtyp reflect.Type, rmap map[reflect.Type]reflect.Type) (in, out []reflect.Type, ntyp, inTyp, outTyp reflect.Type) {
	var inFields []reflect.StructField
	var outFields []reflect.StructField
	for i := 0; i < mtyp.NumIn(); i++ {
		t := mtyp.In(i)
		if rmap != nil {
			t = replaceType(t, rmap)
		}
		in = append(in, t)
		inFields = append(inFields, reflect.StructField{
			Name: fmt.Sprintf("Arg%v", i),
			Type: t,
		})
	}
	for i := 0; i < mtyp.NumOut(); i++ {
		t := mtyp.Out(i)
		if rmap != nil {
			t = replaceType(t, rmap)
		}
		out = append(out, t)
		outFields = append(outFields, reflect.StructField{
			Name: fmt.Sprintf("Out%v", i),
			Type: t,
		})
	}
	if rmap == nil {
		ntyp = mtyp
	} else {
		ntyp = reflect.FuncOf(in, out, mtyp.IsVariadic())
	}
	inTyp = reflect.StructOf(inFields)
	outTyp = reflect.StructOf(outFields)
	return
}
