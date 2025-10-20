package reflectx

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/goplus/reflectx/abi"
)

var (
	// disable unable allocate warning
	DisableAllocateWarning bool
)

var Default *Context = NewContext()

type Context struct {
	embedLookupCache    map[reflect.Type]reflect.Type
	structLookupCache   map[string][]reflect.Type
	interfceLookupCache map[string]reflect.Type
	methodIndexList     map[abi.MethodProvider][]int
	methodIfnCache      map[ifnKey]unsafe.Pointer
	fnHasImethod        func(typ reflect.Type, method Method) bool
	nAllocateError      int
}

type ifnKey struct {
	name     string
	funcId   uintptr
	inTyp    reflect.Type
	outTyp   reflect.Type
	pointer  bool
	indirect bool
	oneptr   bool
}

func NewContext() *Context {
	ctx := &Context{}
	ctx.embedLookupCache = make(map[reflect.Type]reflect.Type)
	ctx.structLookupCache = make(map[string][]reflect.Type)
	ctx.interfceLookupCache = make(map[string]reflect.Type)
	ctx.methodIndexList = make(map[abi.MethodProvider][]int)
	ctx.methodIfnCache = make(map[ifnKey]unsafe.Pointer)
	return ctx
}

func (p *Context) SetHasImethod(hasImethod func(typ reflect.Type, method Method) bool) {
	p.fnHasImethod = hasImethod
}

type AllocError struct {
	Typ reflect.Type
	Cap int
	Req int
}

func (p *AllocError) Error() string {
	return fmt.Sprintf("cannot alloc method %q, cap:%v req:%v",
		p.Typ, p.Cap, p.Req)
}
