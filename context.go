//go:build !llgo
// +build !llgo

package reflectx

import (
	"fmt"
	"reflect"
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
	methodIndexList     map[int][]int
	fnHasImethod        func(typ reflect.Type, method Method) bool
	nAllocateError      int
}

func NewContext() *Context {
	ctx := &Context{}
	ctx.embedLookupCache = make(map[reflect.Type]reflect.Type)
	ctx.structLookupCache = make(map[string][]reflect.Type)
	ctx.interfceLookupCache = make(map[string]reflect.Type)
	ctx.methodIndexList = make(map[int][]int)
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
