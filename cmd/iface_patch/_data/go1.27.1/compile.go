//go:build ignore

package p

func _() {
	if Flag.IfaceFuncval {
		if buildcfg.GOARCH != "wasm" && buildcfg.GOARCH != "arm64" {
			log.Fatal("-ifacefuncval is only supported on wasm and arm64")
		}
		wasm.EnableIfaceFuncval = true
	}
}
