//go:build ppc64 || ppc64le || riscv64 || s390x

package reflect

func archFloat32FromReg(reg uint64) float32
func archFloat32ToReg(val float32) uint64
