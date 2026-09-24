//go:build ignore

package arm64

import (
	"cmd/compile/internal/ssa"
	"cmd/compile/internal/ssagen"
	"cmd/internal/obj"
	"cmd/internal/obj/arm64"
	"cmd/internal/objabi"
)

func ssaGenIfaceFuncvalCall(s *ssagen.State, v *ssa.Value) {
	ssaGenIfaceFuncvalCallReg(s, v, false)
}

func ssaGenIfaceFuncvalTailCall(s *ssagen.State, v *ssa.Value) {
	ssaGenIfaceFuncvalCallReg(s, v, true)
}

func callOrTail(s *ssagen.State, v *ssa.Value, tail bool) {
	if tail {
		s.TailCall(v)
	} else {
		s.Call(v)
	}
}

// ssaGenIfaceFuncvalCallReg unwraps a tagged itab.Fun (makeFuncImpl*|1)
// then transfers through the stock Call/TailCall path when possible.
//
// Untagged (bit 0 clear): jump to s.Call/s.TailCall with the original
// register, same as unmodified gc.
// Tagged and Rn != R26: R26=CTXT, Rn=makeFuncStub, then s.Call/s.TailCall.
// Tagged and Rn == R26: stub cannot live in R26 (that is CTXT), so load it
// into R20 (same as CALLFN) and emit CALL/ARET (R20). TailCall(v) would
// target R26, which is now the impl pointer, not the code PC.
func ssaGenIfaceFuncvalCallReg(s *ssagen.State, v *ssa.Value, tail bool) {
	if !objabi.EnableIfaceFuncval {
		callOrTail(s, v, tail)
		return
	}
	rn := v.Args[0].Reg()
	tbz := s.Prog(arm64.ATBZ)
	tbz.From.Type = obj.TYPE_CONST
	tbz.From.Offset = 0
	tbz.Reg = rn
	tbz.To.Type = obj.TYPE_BRANCH
	if rn != arm64.REGCTXT {
		bic := s.Prog(arm64.ABIC)
		bic.From.Type = obj.TYPE_CONST
		bic.From.Offset = 1
		bic.Reg = rn
		bic.To.Type = obj.TYPE_REG
		bic.To.Reg = arm64.REGCTXT
		mov := s.Prog(arm64.AMOVD)
		mov.From.Type = obj.TYPE_MEM
		mov.From.Reg = arm64.REGCTXT
		mov.To.Type = obj.TYPE_REG
		mov.To.Reg = rn
		skip := s.Prog(obj.ANOP)
		tbz.To.SetTarget(skip)
		callOrTail(s, v, tail)
		return
	}
	bic := s.Prog(arm64.ABIC)
	bic.From.Type = obj.TYPE_CONST
	bic.From.Offset = 1
	bic.Reg = rn
	bic.To.Type = obj.TYPE_REG
	bic.To.Reg = arm64.REGCTXT
	load := s.Prog(arm64.AMOVD)
	load.From.Type = obj.TYPE_MEM
	load.From.Reg = arm64.REGCTXT
	load.To.Type = obj.TYPE_REG
	load.To.Reg = arm64.REG_R20
	jmpR20 := s.Prog(obj.AJMP)
	jmpR20.To.Type = obj.TYPE_BRANCH
	stock := s.Prog(obj.ANOP)
	tbz.To.SetTarget(stock)
	callOrTail(s, v, tail)
	jmpEnd := s.Prog(obj.AJMP)
	jmpEnd.To.Type = obj.TYPE_BRANCH
	callR20 := s.Prog(obj.ANOP)
	jmpR20.To.SetTarget(callR20)
	s.PrepareCall(v)
	p := s.Prog(obj.ACALL)
	p.To.Type = obj.TYPE_MEM
	p.To.Reg = arm64.REG_R20
	if tail {
		p.As = obj.ARET
	}
	end := s.Prog(obj.ANOP)
	jmpEnd.To.SetTarget(end)
}
