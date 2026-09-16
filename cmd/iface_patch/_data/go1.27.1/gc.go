//go:build ignore

package work

// wasmIfaceFuncval reports whether this wasm build requested the
// goplus.ifacefuncval extension (tagged MakeFunc funcval ifn).
func wasmIfaceFuncval() bool {
	if cfg.Goarch != "wasm" {
		return false
	}
	for _, tag := range cfg.BuildContext.BuildTags {
		if tag == "goplus.ifacefuncval" {
			return true
		}
	}
	return false
}
