package reflectx

import (
	"reflect"
)

var Default *Context = NewContext()

type Context struct {
	embedLookupCache    map[reflect.Type]reflect.Type
	structLookupCache   map[string][]reflect.Type
	interfceLookupCache map[string]reflect.Type
	methodIndexList     map[MethodProvider][]int
}

func NewContext() *Context {
	ctx := &Context{}
	ctx.embedLookupCache = make(map[reflect.Type]reflect.Type)
	ctx.structLookupCache = make(map[string][]reflect.Type)
	ctx.interfceLookupCache = make(map[string]reflect.Type)
	ctx.methodIndexList = make(map[MethodProvider][]int)
	return ctx
}

func (ctx *Context) Release() {
	for mp, list := range ctx.methodIndexList {
		mp.Remove(list)
	}
}
