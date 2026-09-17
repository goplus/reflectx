//go:build ignore

package work

// ifaceFuncvalEnabled reports whether this build requested the
// goplus.ifacefuncval extension (tagged MakeFunc funcval ifn)
// on wasm or arm64.
func ifaceFuncvalEnabled() bool {
	switch cfg.Goarch {
	case "wasm", "arm64":
	default:
		return false
	}
	for _, tag := range cfg.BuildContext.BuildTags {
		if tag == "goplus.ifacefuncval" {
			return true
		}
	}
	return false
}
