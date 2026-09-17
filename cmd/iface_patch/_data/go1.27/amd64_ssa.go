//go:build ignore

package amd64

import (
	"cmd/compile/internal/ssa"
	"cmd/compile/internal/ssagen"
	"cmd/internal/obj"
	"cmd/internal/obj/x86"
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
// Tagged and Rn != DX: DX=CTXT, Rn=makeFuncStub, then s.Call/s.TailCall.
// Tagged and Rn == DX: stub cannot live in DX (that is CTXT), so load it
// into R12 and emit CALL/ARET R12. TailCall(v) would target DX, which is
// now the impl pointer, not the code PC. ARET REG is how TailCall already
// lowers a register-target interface tail call (see ssagen.TailCall).
func ssaGenIfaceFuncvalCallReg(s *ssagen.State, v *ssa.Value, tail bool) {
	if !objabi.EnableIfaceFuncval {
		callOrTail(s, v, tail)
		return
	}
	rn := v.Args[0].Reg()
	bt := s.Prog(x86.ABTQ)
	bt.From.Type = obj.TYPE_CONST
	bt.From.Offset = 0
	bt.To.Type = obj.TYPE_REG
	bt.To.Reg = rn
	jcc := s.Prog(x86.AJCC) // CF=0: even PC, use stock Call
	jcc.To.Type = obj.TYPE_BRANCH
	if rn != x86.REGCTXT {
		mov := s.Prog(x86.AMOVQ)
		mov.From.Type = obj.TYPE_REG
		mov.From.Reg = rn
		mov.To.Type = obj.TYPE_REG
		mov.To.Reg = x86.REGCTXT
	}
	and := s.Prog(x86.AANDQ)
	and.From.Type = obj.TYPE_CONST
	and.From.Offset = -2
	and.To.Type = obj.TYPE_REG
	and.To.Reg = x86.REGCTXT
	load := s.Prog(x86.AMOVQ)
	load.From.Type = obj.TYPE_MEM
	load.From.Reg = x86.REGCTXT
	if rn != x86.REGCTXT {
		load.To.Type = obj.TYPE_REG
		load.To.Reg = rn
		jmpStock := s.Prog(obj.AJMP)
		jmpStock.To.Type = obj.TYPE_BRANCH
		stock := s.Prog(obj.ANOP)
		jcc.To.SetTarget(stock)
		jmpStock.To.SetTarget(stock)
		callOrTail(s, v, tail)
		return
	}
	load.To.Type = obj.TYPE_REG
	load.To.Reg = x86.REG_R12
	jmpR12 := s.Prog(obj.AJMP)
	jmpR12.To.Type = obj.TYPE_BRANCH
	stock := s.Prog(obj.ANOP)
	jcc.To.SetTarget(stock)
	callOrTail(s, v, tail)
	jmpEnd := s.Prog(obj.AJMP)
	jmpEnd.To.Type = obj.TYPE_BRANCH
	callR12 := s.Prog(obj.ANOP)
	jmpR12.To.SetTarget(callR12)
	s.PrepareCall(v)
	p := s.Prog(obj.ACALL)
	p.To.Type = obj.TYPE_REG
	p.To.Reg = x86.REG_R12
	if tail {
		p.As = obj.ARET
	}
	end := s.Prog(obj.ANOP)
	jmpEnd.To.SetTarget(end)
}
