package reflect

import (
	"sync"
	"unsafe"

	"github.com/goplus/reflectx/internal/abi"
	"github.com/goplus/reflectx/internal/goarch"
)

type Kind uint

const (
	Invalid Kind = iota
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	Array
	Chan
	Func
	Interface
	Map
	Pointer
	Slice
	String
	Struct
	UnsafePointer
)

// Ptr is the old name for the [Pointer] kind.
const Ptr = Pointer

type layoutKey struct {
	ftyp *funcType // function signature
	rcvr *abi.Type // receiver type, or nil if none
}

type layoutType struct {
	t         *abi.Type
	framePool *sync.Pool
	abid      abiDesc
}

var layoutCache sync.Map // map[layoutKey]layoutType

// funcLayout computes a struct type representing the layout of the
// stack-assigned function arguments and return values for the function
// type t.
// If rcvr != nil, rcvr specifies the type of the receiver.
// The returned type exists only for GC, so we only fill out GC relevant info.
// Currently, that's just size and the GC program. We also fill in
// the name for possible debugging use.
func funcLayout(t *funcType, rcvr *abi.Type) (frametype *abi.Type, framePool *sync.Pool, abid abiDesc) {
	if t.Kind() != abi.Func {
		panic("reflect: funcLayout of non-func type " + stringFor(&t.Type))
	}
	if rcvr != nil && rcvr.Kind() == abi.Interface {
		panic("reflect: funcLayout with interface receiver " + stringFor(rcvr))
	}
	k := layoutKey{t, rcvr}
	if lti, ok := layoutCache.Load(k); ok {
		lt := lti.(layoutType)
		return lt.t, lt.framePool, lt.abid
	}

	// Compute the ABI layout.
	abid = newAbiDesc(t, rcvr)

	// build dummy rtype holding gc program
	x := &abi.Type{
		Align_: goarch.PtrSize,
		// Don't add spill space here; it's only necessary in
		// reflectcall's frame, not in the allocated frame.
		// TODO(mknyszek): Remove this comment when register
		// spill space in the frame is no longer required.
		Size_:    align(abid.retOffset+abid.ret.stackBytes, goarch.PtrSize),
		PtrBytes: uintptr(abid.stackPtrs.n) * goarch.PtrSize,
	}
	if abid.stackPtrs.n > 0 {
		x.GCData = &abid.stackPtrs.data[0]
	}

	var s string
	if rcvr != nil {
		s = "methodargs(" + stringFor(rcvr) + ")(" + stringFor(&t.Type) + ")"
	} else {
		s = "funcargs(" + stringFor(&t.Type) + ")"
	}
	x.Str = resolveReflectName(newName(s, "", false, false))

	// cache result for future callers
	framePool = &sync.Pool{New: func() interface{} {
		return unsafe_New(x)
	}}
	lti, _ := layoutCache.LoadOrStore(k, layoutType{
		t:         x,
		framePool: framePool,
		abid:      abid,
	})
	lt := lti.(layoutType)
	return lt.t, lt.framePool, lt.abid
}

// nonEmptyInterface is the header for an interface value with methods.
type nonEmptyInterface struct {
	itab *abi.ITab
	word unsafe.Pointer
}

type funcType = abi.FuncType
type arrayType = abi.ArrayType
type structType = abi.StructType
type sliceType = abi.SliceType
type interfaceType = abi.InterfaceType
