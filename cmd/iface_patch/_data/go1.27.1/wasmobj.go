//go:build ignore

package wasm

import (
	"cmd/internal/obj"
	"cmd/internal/objabi"
)

// IfaceFuncvalBit marks an itab.Fun / indirect-call target as a
// MakeFunc funcval rather than a code PC. The stored value is
// makeFuncImpl* | IfaceFuncvalBit. Heap objects and function-entry
// PCs are both even, so the bit is free.
//
// Matches github.com/goplus/reflectx when built with
// -tags goplus.ifacefuncval on wasm, arm64, or amd64.
//
// At an indirect call the target is unwrapped: CTXT is set to the
// untagged pointer (the funcval) and the call uses the code pointer
// in the first word, which is makeFuncStub. Untagged targets are
// left alone, including CTXT, so ordinary methods and closure calls
// keep working.
const IfaceFuncvalBit = 1

func unwrapIfaceFuncvalPC(p *obj.Prog, appendp func(*obj.Prog, obj.As, ...obj.Addr) *obj.Prog) *obj.Prog {
	if !objabi.EnableIfaceFuncval {
		return p
	}
	// pc is on the wasm stack. Stash it in RET0 (not live at a call).
	p = appendp(p, ASet, regAddr(REG_RET0))
	p = appendp(p, AGet, regAddr(REG_RET0))
	p = appendp(p, AI64Const, constAddr(IfaceFuncvalBit))
	p = appendp(p, AI64And)
	p = appendp(p, AI32WrapI64)
	p = appendp(p, AIf)
	// ifacefuncval: CTXT = pc&^1, RET0 = CTXT.fn
	p = appendp(p, AGet, regAddr(REG_RET0))
	p = appendp(p, AI64Const, constAddr(^int64(IfaceFuncvalBit)))
	p = appendp(p, AI64And)
	p = appendp(p, ASet, regAddr(REG_CTXT))
	p = appendp(p, AGet, regAddr(REG_CTXT))
	p = appendp(p, AI32WrapI64)
	p = appendp(p, AI64Load, constAddr(0))
	p = appendp(p, ASet, regAddr(REG_RET0))
	p = appendp(p, AEnd)
	p = appendp(p, AGet, regAddr(REG_RET0))
	return p
}
