package reflectx

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

var (
	// disable unable allocate warning
	DisableAllocateWarning bool
)

var Default *Context = NewContext()

type Context struct {
	embedLookupCache    sync.Map // map[reflect.Type]reflect.Type
	structLookupCache   sync.Map // map[string][]reflect.Type
	interfceLookupCache sync.Map // map[string]reflect.Type
	methodIndexList     map[int][]int
	methodIndexLock     sync.Mutex
	fnHasImethod        func(typ reflect.Type, method Method) bool
	nAllocateError      atomic.Int64
}

func NewContext() *Context {
	ctx := &Context{}
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
