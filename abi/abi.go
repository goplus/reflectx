package abi

import (
	"reflect"
	"unsafe"
)

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

type MethodProvider interface {
	Insert(info *MethodInfo) (ifn unsafe.Pointer, index int) // insert method info
	Remove(index []int)                                      // remove method info
	Available() int                                          // available count
	Used() int                                               // methods used
	Cap() int                                                // methods capacity
	Clear()                                                  // clear all methods
}

type MethodProviderList struct {
	list   []MethodProvider
	maxCap int
}

func (p *MethodProviderList) List() []MethodProvider {
	return p.list
}

func (p *MethodProviderList) Clear() {
	for _, v := range p.list {
		v.Clear()
	}
}

func (p *MethodProviderList) Add(mp MethodProvider) {
	for _, v := range p.list {
		if v == mp {
			return
		}
	}
	p.list = append(p.list, mp)
	p.maxCap += mp.Cap()
}

func (p *MethodProviderList) Used() int {
	var n int
	for _, mp := range p.list {
		n += mp.Used()
	}
	return n
}

func (p *MethodProviderList) Available() int {
	var n int
	for _, mp := range p.list {
		n += mp.Available()
	}
	return n
}

func (p *MethodProviderList) Cap() int {
	return p.maxCap
}

func AddMethodProvider(mp MethodProvider) {
	Default.Add(mp)
}

var (
	Default = &MethodProviderList{}
)
