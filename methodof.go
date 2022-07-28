//go:build !js || (js && wasm)
// +build !js js,wasm

package reflectx

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"unsafe"
)

type MethodProvider interface {
	Push(info *MethodInfo) (ifn unsafe.Pointer) // push method info
	Len() int                                   // methods len
	Cap() int                                   // methods capacity
	Clear()                                     // clear all methods
}

type mpList struct {
	list     []MethodProvider
	cur      MethodProvider
	mpIndex  int
	curCap   int
	curIndex int
	maxCap   int
	allIndex int
}

func (p *mpList) Clear() {
	p.mpIndex = 0
	p.allIndex = 0
	p.curIndex = 0
	for _, v := range p.list {
		v.Clear()
	}
	if len(p.list) >= 1 {
		p.cur = p.list[0]
		p.curCap = p.cur.Cap()
	}
}

func (p *mpList) Add(mp MethodProvider) {
	for _, v := range p.list {
		if v == mp {
			return
		}
	}
	p.list = append(p.list, mp)
	p.maxCap += mp.Cap()
	if len(p.list) == 1 {
		p.cur = p.list[0]
		p.curCap = p.cur.Cap()
	}
}

func (p *mpList) Push(info *MethodInfo) (ifn unsafe.Pointer) {
	p.allIndex++
	p.curIndex++
	if p.curIndex >= p.curCap {
		p.mpIndex++
		if p.mpIndex >= len(p.list) {
			log.Printf("warning, cannot alloc method %v > %v, import _ %q\n",
				p.allIndex+1, p.maxCap, "github.com/goplus/reflectx/icall/icall[2^n]")
			return nil
		}
		p.cur = p.list[p.mpIndex]
		p.curIndex = 0
		p.curCap = p.cur.Cap()
	}
	return p.cur.Push(info)
}

func AddMethodProvider(mp MethodProvider) {
	mps.Add(mp)
}

var (
	mps mpList
)

func resetMethodList() {
	mps.Clear()
}

// register method info
func registerMethod(info *MethodInfo) (ifn unsafe.Pointer) {
	return mps.Push(info)
}

// func isMethod(typ reflect.Type) (ok bool) {
// 	return totype(typ).tflag&tflagUserMethod != 0
// }

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

func MethodByIndex(typ reflect.Type, index int) reflect.Method {
	return totype(typ).MethodX(index)
}

func MethodByName(typ reflect.Type, name string) (m reflect.Method, ok bool) {
	m, ok = totype(typ).MethodByNameX(name)
	return
}

func resizeMethod(typ reflect.Type, mcount int, xcount int) error {
	rt := totype(typ)
	ut := toUncommonType(rt)
	if ut == nil {
		return fmt.Errorf("not found uncommonType of %v", typ)
	}
	if uint16(mcount) > ut.mcount {
		return fmt.Errorf("too many methods of %v", typ)
	}
	ut.xcount = uint16(xcount)
	return nil
}

func createMethod(typ reflect.Type, ptyp reflect.Type, m Method, index int) (mfn reflect.Value, inTyp, outTyp reflect.Type, mtyp typeOff, tfn, ptfn textOff) {
	var in []reflect.Type
	var out []reflect.Type
	var ntyp reflect.Type
	in, out, ntyp, inTyp, outTyp = parserMethodType(m.Type, nil)
	mtyp = resolveReflectType(totype(ntyp))
	var ftyp reflect.Type
	if m.Pointer {
		ftyp = reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, m.Type.IsVariadic())
	} else {
		ftyp = reflect.FuncOf(append([]reflect.Type{typ}, in...), out, m.Type.IsVariadic())
	}

	mfn = reflect.MakeFunc(ftyp, m.Func)
	ptr := tovalue(&mfn).ptr

	tfn = resolveReflectText(unsafe.Pointer(ptr))
	if !m.Pointer {
		variadic := m.Type.IsVariadic()
		ctyp := reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, variadic)
		var cv reflect.Value
		if variadic {
			cv = reflect.MakeFunc(ctyp, func(args []reflect.Value) (results []reflect.Value) {
				return args[0].Elem().Method(index).CallSlice(args[1:])
			})
		} else {
			cv = reflect.MakeFunc(ctyp, func(args []reflect.Value) (results []reflect.Value) {
				return args[0].Elem().Method(index).Call(args[1:])
			})
		}
		ptfn = resolveReflectText(tovalue(&cv).ptr)
	} else {
		ptfn = tfn
	}
	return
}

func setMethodSet(typ reflect.Type, methods []Method) error {
	sort.Slice(methods, func(i, j int) bool {
		n := strings.Compare(methods[i].Name, methods[j].Name)
		if n == 0 && methods[i].PkgPath == methods[j].PkgPath {
			panic(fmt.Sprintf("method redeclared: %v", methods[j].Name))
		}
		return n < 0
	})
	var mcount, pcount int
	var xcount, pxcount int
	pcount = len(methods)
	var mlist []string
	for _, m := range methods {
		isexport := methodIsExported(m.Name)
		if isexport {
			pxcount++
		}
		if !m.Pointer {
			if isexport {
				xcount++
			}
			mlist = append(mlist, m.Name)
			mcount++
		}
	}
	ptyp := reflect.PtrTo(typ)
	if err := resizeMethod(typ, mcount, xcount); err != nil {
		return err
	}
	if err := resizeMethod(ptyp, pcount, pxcount); err != nil {
		return err
	}
	rt := totype(typ)
	prt := totype(ptyp)

	ms := rt.methods()
	pms := prt.methods()

	var onePtr bool
	switch typ.Kind() {
	case reflect.Func, reflect.Chan, reflect.Map:
		onePtr = true
	case reflect.Struct:
		onePtr = typ.NumField() == 1 && typ.Field(0).Type.Kind() == reflect.Ptr
	}
	var index int
	for i, m := range methods {
		isexport := methodIsExported(m.Name)
		nm := newNameEx(m.Name, "", isexport, !isexport)
		mname := resolveReflectName(nm)
		if !isexport {
			nm.setPkgPath(resolveReflectName(newName(m.PkgPath, "", false)))
		}
		mfn, inTyp, outTyp, mtyp, tfn, ptfn := createMethod(typ, ptyp, m, index)
		isz := argsTypeSize(inTyp, true)
		osz := argsTypeSize(outTyp, false)
		pinfo := &MethodInfo{
			Name:     m.Name,
			Type:     typ,
			Func:     mfn,
			InTyp:    inTyp,
			OutTyp:   outTyp,
			InSize:   isz,
			OutSize:  osz,
			Pointer:  true,
			Indirect: !m.Pointer,
			Variadic: m.Type.IsVariadic(),
			OnePtr:   onePtr,
		}
		pifn := registerMethod(pinfo)
		pms[i].name = mname
		pms[i].mtyp = mtyp
		pms[i].tfn = ptfn
		pms[i].ifn = resolveReflectText(pifn)

		if !m.Pointer {
			info := &MethodInfo{
				Name:     m.Name,
				Type:     typ,
				Func:     mfn,
				InTyp:    inTyp,
				OutTyp:   outTyp,
				InSize:   isz,
				OutSize:  osz,
				Variadic: m.Type.IsVariadic(),
				OnePtr:   onePtr,
			}
			ifn := registerMethod(info)
			ms[index].name = mname
			ms[index].mtyp = mtyp
			ms[index].tfn = tfn
			ms[index].ifn = resolveReflectText(ifn)
			index++
		}
	}
	rt.tflag |= tflagUserMethod
	prt.tflag |= tflagUserMethod
	return nil
}

func newMethodSet(styp reflect.Type, maxmfunc, maxpfunc int) reflect.Type {
	rt, _ := newType("", "", styp, maxmfunc, 0)
	prt, _ := newType("", "", reflect.PtrTo(styp), maxpfunc, 0)
	rt.ptrToThis = resolveReflectType(prt)
	(*ptrType)(unsafe.Pointer(prt)).elem = rt
	setTypeName(rt, styp.PkgPath(), styp.Name())
	prt.uncommon().pkgPath = resolveReflectName(newName(styp.PkgPath(), "", false))
	typ := toType(rt)
	if nt, ok := ntypeMap[styp]; ok {
		ntypeMap[typ] = &Named{Name: nt.Name, PkgPath: nt.PkgPath, Type: typ, From: nt.From, Kind: nt.Kind}
	}
	return typ
}

const (
	uintptrAligin = unsafe.Sizeof(uintptr(0))
)

func argsTypeSize(typ reflect.Type, offset bool) (off uintptr) {
	numIn := typ.NumField()
	if numIn == 0 {
		return 0
	}
	for i := 0; i < numIn; i++ {
		t := typ.Field(i).Type
		targ := totype(t)
		a := uintptr(targ.align)
		off = (off + a - 1) &^ (a - 1)
		n := targ.size
		if n == 0 {
			continue
		}
		off += n
	}
	if offset {
		off = (off + uintptrAligin - 1) &^ (uintptrAligin - 1)
		if off == 0 {
			return uintptrAligin
		}
	}
	return
}
