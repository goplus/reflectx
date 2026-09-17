//go:build ignore

package arm64

import (
	"cmd/compile/internal/ssa"
	"cmd/compile/internal/ssagen"
	"cmd/internal/obj"
	"cmd/internal/obj/arm64"
	"cmd/internal/objabi"
)

// ssaGenIfaceFuncvalCall rewrites a tagged itab.Fun (makeFuncImpl*|1)
// into CTXT + makeFuncStub before an indirect interface call.
// Only when -ifacefuncval is set; otherwise CALLinter is a plain BL.
func ssaGenIfaceFuncvalCall(s *ssagen.State, v *ssa.Value) {
	if !objabi.EnableIfaceFuncval {
		return
	}
	rn := v.Args[0].Reg()
	tbz := s.Prog(arm64.ATBZ)
	tbz.From.Type = obj.TYPE_CONST
	tbz.From.Offset = 0
	tbz.Reg = rn
	tbz.To.Type = obj.TYPE_BRANCH
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
}
