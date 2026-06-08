//go:build llgo
// +build llgo

package reflectx

import (
	"fmt"
	"reflect"
	"unsafe"
)

// Value mirrors the reflect.Value layout used by the llgo runtime.
// Flag constants are identical to standard Go.
type Value struct {
	typ_ unsafe.Pointer
	ptr  unsafe.Pointer
	flag
}

type flag uintptr

const (
	flagKindWidth        = 5
	flagKindMask    flag = 1<<flagKindWidth - 1
	flagStickyRO    flag = 1 << 5
	flagEmbedRO     flag = 1 << 6
	flagIndir       flag = 1 << 7
	flagAddr        flag = 1 << 8
	flagMethod      flag = 1 << 9
	flagMethodShift      = 10
	flagRO          flag = flagStickyRO | flagEmbedRO
)

func canSet(v *reflect.Value) {
	(*Value)(unsafe.Pointer(v)).flag &= ^flagRO
}

// CanSet clears read-only flags from v so that v.CanSet returns true.
func CanSet(v reflect.Value) reflect.Value {
	if !v.CanSet() {
		(*Value)(unsafe.Pointer(&v)).flag &= ^flagRO
	}
	return v
}

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

// NamedTypeOf returns a new named type based on from with the given pkgpath and name.
// Under llgo this only sets the type string and does not modify runtime metadata.
func NamedTypeOf(pkgpath string, name string, from reflect.Type) reflect.Type {
	return from
}

// SetTypeName is a no-op under llgo.
func SetTypeName(typ reflect.Type, pkgpath string, name string) {}

func StructOf(fields []reflect.StructField) reflect.Type {
	return Default.StructOf(fields)
}

func (ctx *Context) StructOf(fields []reflect.StructField) reflect.Type {
	var anonymous []int
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if f.Anonymous {
			anonymous = append(anonymous, i)
			if f.Name == "" {
				fields[i].Name = typeName(f.Type)
			}
		}
	}
	typ := reflect.StructOf(fields)
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

// SetElem is a no-op under llgo.
func SetElem(typ reflect.Type, elem reflect.Type) {}

func typeId(typ reflect.Type) string {
	var id string
	if path := typ.PkgPath(); path != "" {
		id = path + "."
	}
	return id + typ.Name()
}

// ReplaceType is a no-op under llgo.
func ReplaceType(pkg string, typ reflect.Type, m map[string]reflect.Type) (rtyp reflect.Type, changed bool) {
	return typ, false
}

// PtrTo returns the pointer type with element t.
func PtrTo(t reflect.Type) reflect.Type {
	return reflect.PointerTo(t)
}

// TypeLinks returns all types known to the runtime.
// Under llgo this returns an empty slice.
func TypeLinks() []reflect.Type {
	return nil
}

// TypesByString searches for types matching the given string.
// Under llgo this returns an empty slice.
func TypesByString(s string) []reflect.Type {
	return nil
}

// DumpType prints type information. Under llgo this is a no-op.
func DumpType(w interface{ Write([]byte) (int, error) }, typ reflect.Type) {}

// NumMethodX returns the number of methods including unexported.
func NumMethodX(typ reflect.Type) int {
	return typ.NumMethod()
}

// MethodX returns the i'th method of typ.
func MethodX(typ reflect.Type, i int) reflect.Method {
	return typ.Method(i)
}

// FieldX returns field i of struct value v with read-only flag cleared.
func FieldX(v reflect.Value, i int) reflect.Value {
	f := v.Field(i)
	canSet(&f)
	return f
}

// FieldByIndexX returns a field using a chain of indices.
func FieldByIndexX(v reflect.Value, index []int) reflect.Value {
	if len(index) == 1 {
		return FieldX(v, index[0])
	}
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

// FieldByNameX returns a named field with read-only flag cleared.
func FieldByNameX(v reflect.Value, name string) reflect.Value {
	if f, ok := v.Type().FieldByName(name); ok {
		return FieldByIndexX(v, f.Index)
	}
	return reflect.Value{}
}

// FieldByNameFuncX returns a named field with read-only flag cleared.
func FieldByNameFuncX(v reflect.Value, match func(string) bool) reflect.Value {
	if f, ok := v.Type().FieldByNameFunc(match); ok {
		return FieldByIndexX(v, f.Index)
	}
	return reflect.Value{}
}

// Method represents a reflectx method definition.
type Method struct {
	Name    string
	PkgPath string
	Pointer bool
	Type    reflect.Type
	Func    func([]reflect.Value) []reflect.Value
	FuncId  int
}

// MakeMethod creates a Method.
func MakeMethod(name string, pkgpath string, pointer bool, typ reflect.Type, fn func(args []reflect.Value) (result []reflect.Value)) Method {
	return Method{
		Name:    name,
		PkgPath: pkgpath,
		Pointer: pointer,
		Type:    typ,
		Func:    fn,
	}
}

// MethodByIndex returns the i'th method of typ.
func MethodByIndex(typ reflect.Type, index int) reflect.Method {
	return typ.Method(index)
}

// MethodByName returns the named method of typ.
func MethodByName(typ reflect.Type, name string) (m reflect.Method, ok bool) {
	return typ.MethodByName(name)
}

// MethodInfo carries metadata about a method being registered.
type MethodInfo struct {
	Name     string
	Func     reflect.Value
	Type     reflect.Type
	InTyp    reflect.Type
	OutTyp   reflect.Type
	InSize   uintptr
	OutSize  uintptr
	Pointer  bool
	Indirect bool
	Variadic bool
	OnePtr   bool
}

// AllocError is returned when no icall slots are available.
type AllocError struct {
	Typ reflect.Type
	Cap int
	Req int
}

func (p *AllocError) Error() string {
	return fmt.Sprintf("cannot alloc method %q, cap:%v req:%v", p.Typ, p.Cap, p.Req)
}

var DisableAllocateWarning bool

// Context holds reflectx state.
type Context struct{}

var Default *Context = NewContext()

func NewContext() *Context {
	return &Context{}
}

func (p *Context) SetHasImethod(hasImethod func(typ reflect.Type, method Method) bool) {}

func (p *Context) Reset() {}

func (p *Context) ResetAll() {}

func (p *Context) IcallAlloc() int { return 0 }

func Reset() {
	Default.Reset()
}

func ResetAll() {}

// IcallStat returns icall capacity info.
// Under llgo there is no icall mechanism; all values are 0.
func IcallStat() (capacity int, allocate int, aviable int) { return 0, 0, 0 }

// IcallCached returns the number of cached icalls.
func IcallCached() int { return 0 }

// NewMethodSet pre-allocates a method set. Under llgo returns a plain type.
func NewMethodSet(styp reflect.Type, maxmfunc, maxpfunc int) reflect.Type {
	return Default.NewMethodSet(styp, maxmfunc, maxpfunc)
}

func (ctx *Context) NewMethodSet(styp reflect.Type, maxmfunc, maxpfunc int) reflect.Type {
	return styp
}

// StructToMethodSet extracts methods from embedded struct fields.
// Under llgo returns the type unchanged.
func StructToMethodSet(styp reflect.Type) reflect.Type {
	return Default.StructToMethodSet(styp)
}

func (ctx *Context) StructToMethodSet(styp reflect.Type) reflect.Type {
	return styp
}

// SetMethodSet sets the method set for the given type.
// Under llgo this is not supported and returns an error.
func SetMethodSet(styp reflect.Type, methods []Method, extractStructEmbed bool) error {
	return Default.SetMethodSet(styp, methods, extractStructEmbed)
}

func (ctx *Context) SetMethodSet(styp reflect.Type, methods []Method, extractStructEmbed bool) error {
	return fmt.Errorf("reflectx.SetMethodSet: not supported under llgo")
}

func (ctx *Context) SetRawMethods(styp reflect.Type, methods []Method) error {
	return fmt.Errorf("reflectx.SetRawMethods: not supported under llgo")
}

// SetUnderlying is a no-op under llgo.
func SetUnderlying(typ reflect.Type, styp reflect.Type) {}

// UpdateField is a no-op under llgo.
func UpdateField(typ reflect.Type, rmap map[reflect.Type]reflect.Type) bool { return false }

// MakeEmptyInterface creates a named empty interface type.
// Under llgo this returns the plain empty interface type.
func MakeEmptyInterface(pkgpath string, name string) reflect.Type {
	return tyEmptyInterface
}

// NamedInterfaceOf creates a named interface type.
// Under llgo this falls back to reflect.InterfaceOf.
func NamedInterfaceOf(pkgpath string, name string, embedded []reflect.Type, methods []reflect.Method) reflect.Type {
	return InterfaceOf(embedded, methods)
}

// NewInterfaceType creates a new interface type.
// Under llgo this falls back to reflect.InterfaceOf.
func NewInterfaceType(pkgpath string, name string) reflect.Type {
	return tyEmptyInterface
}

// SetInterfaceType sets the methods on an interface type.
// Under llgo this is a no-op.
func SetInterfaceType(typ reflect.Type, embedded []reflect.Type, methods []reflect.Method) error {
	return nil
}

// InterfaceOf creates an interface type.
func InterfaceOf(embedded []reflect.Type, methods []reflect.Method) reflect.Type {
	return Default.InterfaceOf(embedded, methods)
}

func (ctx *Context) InterfaceOf(embedded []reflect.Type, methods []reflect.Method) reflect.Type {
	allMethods := make([]reflect.Method, 0, len(methods))
	for _, e := range embedded {
		if e.Kind() != reflect.Interface {
			panic(fmt.Sprintf("interface contains embedded non-interface %v", e))
		}
		for i := 0; i < e.NumMethod(); i++ {
			allMethods = append(allMethods, e.Method(i))
		}
	}
	allMethods = append(allMethods, methods...)
	return reflect.TypeOf((*interface{})(nil)).Elem()
}
