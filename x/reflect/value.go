package reflect

import (
	"runtime"
	"unsafe"

	"github.com/goplus/reflectx/internal/abi"
)

// Value is the reflection interface to a Go value.
//
// Not all methods apply to all kinds of values. Restrictions,
// if any, are noted in the documentation for each method.
// Use the Kind method to find out the kind of value before
// calling kind-specific methods. Calling a method
// inappropriate to the kind of type causes a run time panic.
//
// The zero Value represents no value.
// Its [Value.IsValid] method returns false, its Kind method returns [Invalid],
// its String method returns "<invalid Value>", and all other methods panic.
// Most functions and methods never return an invalid value.
// If one does, its documentation states the conditions explicitly.
//
// A Value can be used concurrently by multiple goroutines provided that
// the underlying Go value can be used concurrently for the equivalent
// direct operations.
//
// To compare two Values, compare the results of the Interface method.
// Using == on two Values does not compare the underlying values
// they represent.
type Value struct {
	// typ_ holds the type of the value represented by a Value.
	// Access using the typ method to avoid escape of v.
	typ_ *abi.Type

	// Pointer-valued data or, if flagIndir is set, pointer to data.
	// Valid when either flagIndir is set or typ.pointers() is true.
	ptr unsafe.Pointer

	// flag holds metadata about the value.
	//
	// The lowest five bits give the Kind of the value, mirroring typ.Kind().
	//
	// The next set of bits are flag bits:
	//	- flagStickyRO: obtained via unexported not embedded field, so read-only
	//	- flagEmbedRO: obtained via unexported embedded field, so read-only
	//	- flagIndir: val holds a pointer to the data
	//	- flagAddr: v.CanAddr is true (implies flagIndir and ptr is non-nil)
	//	- flagMethod: v is a method value.
	// If ifaceIndir(typ), code can assume that flagIndir is set.
	//
	// The remaining 22+ bits give a method number for method values.
	// If flag.kind() != Func, code can assume that flagMethod is unset.
	flag

	// A method value represents a curried method invocation
	// like r.Read for some receiver r. The typ+val+flag bits describe
	// the receiver r, but the flag's Kind bits say Func (methods are
	// functions), and the top bits of the flag give the method number
	// in r's type's method table.
}

// typ returns the *abi.Type stored in the Value. This method is fast,
// but it doesn't always return the correct type for the Value.
// See abiType and Type, which do return the correct type.
func (v Value) typ() *abi.Type {
	// Types are either static (for compiler-created types) or
	// heap-allocated but always reachable (for reflection-created
	// types, held in the central map). So there is no need to
	// escape types. noescape here help avoid unnecessary escape
	// of v.
	return (*abi.Type)(abi.NoEscape(unsafe.Pointer(v.typ_)))
}

type flag uintptr

const (
	flagKindWidth        = 5 // there are 27 kinds
	flagKindMask    flag = 1<<flagKindWidth - 1
	flagStickyRO    flag = 1 << 5
	flagEmbedRO     flag = 1 << 6
	flagIndir       flag = 1 << 7
	flagAddr        flag = 1 << 8
	flagMethod      flag = 1 << 9
	flagMethodShift      = 10
	flagRO          flag = flagStickyRO | flagEmbedRO
)

func (f flag) kind() Kind {
	return Kind(f & flagKindMask)
}

func (f flag) ro() flag {
	if f&flagRO != 0 {
		return flagStickyRO
	}
	return 0
}

// This structure must be kept in sync with runtime.reflectMethodValue.
// Any changes should be reflected in all both.
type makeFuncCtxt struct {
	fn      uintptr
	stack   *bitVector // ptrmap for both stack args and results
	argLen  uintptr    // just args
	regPtrs abi.IntArgRegBitmap
}

//go:nosplit
func moveMakeFuncArgPtrs(ctxt *makeFuncCtxt, args *abi.RegArgs) {
	for i, arg := range args.Ints {
		// Avoid write barriers! Because our write barrier enqueues what
		// was there before, we might enqueue garbage.
		if ctxt.regPtrs.Get(i) {
			*(*uintptr)(unsafe.Pointer(&args.Ptrs[i])) = arg
		} else {
			// We *must* zero this space ourselves because it's defined in
			// assembly code and the GC will scan these pointers. Otherwise,
			// there will be garbage here.
			*(*uintptr)(unsafe.Pointer(&args.Ptrs[i])) = 0
		}
	}
}

// makeFuncImpl is the closure value implementing the function
// returned by MakeFunc.
// The first three words of this type must be kept in sync with
// methodValue and runtime.reflectMethodValue.
// Any changes should be reflected in all three.
type makeFuncImpl struct {
	makeFuncCtxt
	ftyp *funcType
	fn   func([]Value) []Value
}

// align returns the result of rounding x up to a multiple of n.
// n must be a power of two.
func align(x, n uintptr) uintptr {
	return (x + n - 1) &^ (n - 1)
}

// callReflect is the call implementation used by a function
// returned by MakeFunc. In many ways it is the opposite of the
// method Value.call above. The method above converts a call using Values
// into a call of a function with a concrete argument frame, while
// callReflect converts a call of a function with a concrete argument
// frame into a call using Values.
// It is in this file so that it can be next to the call method above.
// The remainder of the MakeFunc implementation is in makefunc.go.
//
// NOTE: This function must be marked as a "wrapper" in the generated code,
// so that the linker can make it work correctly for panic and recover.
// The gc compilers know to do that for the name "reflect.callReflect".
//
// ctxt is the "closure" generated by MakeFunc.
// frame is a pointer to the arguments to that closure on the stack.
// retValid points to a boolean which should be set when the results
// section of frame is set.
//
// regs contains the argument values passed in registers and will contain
// the values returned from ctxt.fn in registers.
func callReflect(ctxt *makeFuncImpl, frame unsafe.Pointer, retValid *bool, regs *abi.RegArgs) {
	ftyp := ctxt.ftyp
	f := ctxt.fn

	_, _, abid := funcLayout(ftyp, nil)

	// Copy arguments into Values.
	ptr := frame
	in := make([]Value, 0, int(ftyp.InCount))
	for i, typ := range ftyp.InSlice() {
		if typ.Size() == 0 {
			in = append(in, Zero(typ))
			continue
		}
		v := Value{typ, nil, flag(typ.Kind())}
		steps := abid.call.stepsForValue(i)
		if st := steps[0]; st.kind == abiStepStack {
			if !typ.IsDirectIface() {
				// value cannot be inlined in interface data.
				// Must make a copy, because f might keep a reference to it,
				// and we cannot let f keep a reference to the stack frame
				// after this function returns, not even a read-only reference.
				v.ptr = unsafe_New(typ)
				if typ.Size() > 0 {
					typedmemmove(typ, v.ptr, add(ptr, st.stkOff, "typ.size > 0"))
				}
				v.flag |= flagIndir
			} else {
				v.ptr = *(*unsafe.Pointer)(add(ptr, st.stkOff, "1-ptr"))
			}
		} else {
			if !typ.IsDirectIface() {
				// All that's left is values passed in registers that we need to
				// create space for the values.
				v.flag |= flagIndir
				v.ptr = unsafe_New(typ)
				for _, st := range steps {
					switch st.kind {
					case abiStepIntReg:
						offset := add(v.ptr, st.offset, "precomputed value offset")
						intFromReg(regs, st.ireg, st.size, offset)
					case abiStepPointer:
						s := add(v.ptr, st.offset, "precomputed value offset")
						*((*unsafe.Pointer)(s)) = regs.Ptrs[st.ireg]
					case abiStepFloatReg:
						offset := add(v.ptr, st.offset, "precomputed value offset")
						floatFromReg(regs, st.freg, st.size, offset)
					case abiStepStack:
						panic("register-based return value has stack component")
					default:
						panic("unknown ABI part kind")
					}
				}
			} else {
				// Pointer-valued data gets put directly
				// into v.ptr.
				if steps[0].kind != abiStepPointer {
					print("kind=", steps[0].kind, ", type=", stringFor(typ), "\n")
					panic("mismatch between ABI description and types")
				}
				v.ptr = regs.Ptrs[steps[0].ireg]
			}
		}
		in = append(in, v)
	}

	// Call underlying function.
	out := f(in)
	numOut := ftyp.NumOut()
	if len(out) != numOut {
		panic("reflect: wrong return count from function created by MakeFunc")
	}

	// Copy results back into argument frame and register space.
	if numOut > 0 {
		for i, typ := range ftyp.OutSlice() {
			v := out[i]
			if v.typ() == nil {
				panic("reflect: function created by MakeFunc using " + funcName(f) +
					" returned zero Value")
			}
			if v.flag&flagRO != 0 {
				panic("reflect: function created by MakeFunc using " + funcName(f) +
					" returned value obtained from unexported field")
			}
			if typ.Size() == 0 {
				continue
			}

			// Convert v to type typ if v is assignable to a variable
			// of type t in the language spec.
			// See issue 28761.
			//
			//
			// TODO(mknyszek): In the switch to the register ABI we lost
			// the scratch space here for the register cases (and
			// temporarily for all the cases).
			//
			// If/when this happens, take note of the following:
			//
			// We must clear the destination before calling assignTo,
			// in case assignTo writes (with memory barriers) to the
			// target location used as scratch space. See issue 39541.
			v = assignTo(v, "reflect.MakeFunc", typ, nil)
		stepsLoop:
			for _, st := range abid.ret.stepsForValue(i) {
				switch st.kind {
				case abiStepStack:
					// Copy values to the "stack."
					addr := add(ptr, st.stkOff, "precomputed stack arg offset")
					// Do not use write barriers. The stack space used
					// for this call is not adequately zeroed, and we
					// are careful to keep the arguments alive until we
					// return to makeFuncStub's caller.
					if v.flag&flagIndir != 0 {
						memmove(addr, v.ptr, st.size)
					} else {
						// This case must be a pointer type.
						*(*uintptr)(addr) = uintptr(v.ptr)
					}
					// There's only one step for a stack-allocated value.
					break stepsLoop
				case abiStepIntReg, abiStepPointer:
					// Copy values to "integer registers."
					if v.flag&flagIndir != 0 {
						offset := add(v.ptr, st.offset, "precomputed value offset")
						intToReg(regs, st.ireg, st.size, offset)
					} else {
						// Only populate the Ints space on the return path.
						// This is safe because out is kept alive until the
						// end of this function, and the return path through
						// makeFuncStub has no preemption, so these pointers
						// are always visible to the GC.
						regs.Ints[st.ireg] = uintptr(v.ptr)
					}
				case abiStepFloatReg:
					// Copy values to "float registers."
					if v.flag&flagIndir == 0 {
						panic("attempted to copy pointer to FP register")
					}
					offset := add(v.ptr, st.offset, "precomputed value offset")
					floatToReg(regs, st.freg, st.size, offset)
				default:
					panic("unknown ABI part kind")
				}
			}
		}
	}

	// Announce that the return values are valid.
	// After this point the runtime can depend on the return values being valid.
	*retValid = true

	// We have to make sure that the out slice lives at least until
	// the runtime knows the return values are valid. Otherwise, the
	// return values might not be scanned by anyone during a GC.
	// (out would be dead, and the return slots not yet alive.)
	runtime.KeepAlive(out)

	// runtime.getArgInfo expects to be able to find ctxt on the
	// stack when it finds our caller, makeFuncStub. Make sure it
	// doesn't get garbage collected.
	runtime.KeepAlive(ctxt)
}
