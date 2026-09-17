//go:build ignore

package p

func _() {
	if *IfaceFuncval {
		if buildcfg.GOARCH != "wasm" && buildcfg.GOARCH != "arm64" && buildcfg.GOARCH != "amd64" {
			log.Fatal("-ifacefuncval is only supported on wasm, arm64, and amd64")
		}
		objabi.EnableIfaceFuncval = true
	}
}
