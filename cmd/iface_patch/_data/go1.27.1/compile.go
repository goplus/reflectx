//go:build ignore

package p

func _() {
	if Flag.IfaceFuncval {
		if buildcfg.GOARCH != "wasm" {
			log.Fatal("-ifacefuncval is only supported on wasm")
		}
		wasm.EnableIfaceFuncval = true
	}
}
