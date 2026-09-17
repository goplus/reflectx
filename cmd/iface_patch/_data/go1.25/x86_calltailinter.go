//go:build ignore

package p

func _() {
	// 386 has no amd64 ABI0→ABIInternal preamble (zeroX15 / getg in R14).
	ssaGenIfaceFuncvalTailCall(s, v)
}
