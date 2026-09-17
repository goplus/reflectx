//go:build ignore

package objabi

// EnableIfaceFuncval is set by compile/asm -ifacefuncval (cmd/go adds
// that flag for wasm, arm64, amd64, and 386 when -tags goplus.ifacefuncval).
// It is not wasm-specific; the name lives in objabi because several
// backends read it.
var EnableIfaceFuncval bool
