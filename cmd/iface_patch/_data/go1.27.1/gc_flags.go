//go:build ignore

package p

func _() {
	if wasmIfaceFuncval() {
		forcedGcflags = append(forcedGcflags, "-ifacefuncval")
		forcedAsmflags = append(forcedAsmflags, "-ifacefuncval")
	}
}
