//go:build ignore

package p

func _() {
	if s.ABI == obj.ABI0 && v.Aux.(*ssa.AuxCall).Fn.ABI() == obj.ABIInternal {
		zeroX15(s)
		getgFromTLS(s, x86.REG_R14)
	}
	ssaGenIfaceFuncvalTailCall(s, v)
}
