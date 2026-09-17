//go:build ignore

package p

func _() {
	if Flag.IfaceFuncval {
		if buildcfg.GOARCH != "wasm" && buildcfg.GOARCH != "arm64" && buildcfg.GOARCH != "amd64" && buildcfg.GOARCH != "386" {
			log.Fatal("-ifacefuncval is only supported on wasm, arm64, amd64, and 386")
		}
		objabi.EnableIfaceFuncval = true
	}
}
