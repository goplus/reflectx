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

// ssaGenIfaceFuncvalCallReg loads the call target into R12. If the pointer
// is a tagged MakeFunc funcval, DX (CTXT) is the untagged impl and R12 is
// makeFuncStub. R12 is caller-save and unused as an integer arg, so it is
// safe to clobber immediately before CALL.
func ssaGenIfaceFuncvalCallReg(s *ssagen.State, v *ssa.Value, tail bool) {
	if !objabi.EnableIfaceFuncval {
		if tail {
			s.TailCall(v)
		} else {
			s.Call(v)
		}
		return
	}
	rn := v.Args[0].Reg()
	bt := s.Prog(x86.ABTQ)
	bt.From.Type = obj.TYPE_CONST
	bt.From.Offset = 0
	bt.To.Type = obj.TYPE_REG
	bt.To.Reg = rn
	jcc := s.Prog(x86.AJCC) // CF=0: bit 0 clear, already a code PC
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
	load.To.Type = obj.TYPE_REG
	load.To.Reg = x86.REG_R12
	jmp := s.Prog(obj.AJMP)
	jmp.To.Type = obj.TYPE_BRANCH
	untagged := s.Prog(x86.AMOVQ)
	untagged.From.Type = obj.TYPE_REG
	untagged.From.Reg = rn
	untagged.To.Type = obj.TYPE_REG
	untagged.To.Reg = x86.REG_R12
	jcc.To.SetTarget(untagged)
	done := s.Prog(obj.ANOP)
	jmp.To.SetTarget(done)
	s.PrepareCall(v)
	p := s.Prog(obj.ACALL)
	p.To.Type = obj.TYPE_REG
	p.To.Reg = x86.REG_R12
	if tail {
		p.As = obj.ARET
	}
}
