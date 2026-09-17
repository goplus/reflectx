//go:build ignore

package p

func _() {
	if ifaceFuncvalEnabled() {
		forcedGcflags = append(forcedGcflags, "-ifacefuncval")
		forcedAsmflags = append(forcedAsmflags, "-ifacefuncval")
	}
}
