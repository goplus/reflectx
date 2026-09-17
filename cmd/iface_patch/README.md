# iface_patch

Patch a **Go 1.27.1 source tree** so wasm and **darwin/linux arm64** can use
`-tags goplus.ifacefuncval`: interface method `ifn` is a tagged MakeFunc
funcval instead of an icall stub.

The tag is **explicit**. It is not added as a ToolTag. Without the tag, the
patched toolchain matches unmodified Go (icall path).

## Rebuild after patching

`iface_patch` only edits Go source. You must rebuild **from that tree**:

```shell
go run ./cmd/iface_patch /path/to/go1.27.1
go run ./cmd/iface_patch -check /path/to/go1.27.1
cd /path/to/go1.27.1/src && ./make.bash   # Windows: make.bat
```

`go install cmd/asm cmd/compile cmd/go` is not enough: `cmd/go` and the
compiler must come from this `make.bash`. Then use that tree as `GOROOT`:

```shell
export GOROOT=/path/to/go1.27.1
export PATH="$GOROOT/bin:$PATH"
go version   # should print go1.27.1 from $GOROOT
```

Re-running `iface_patch` on an already patched tree is a no-op. Supported
Go versions are the directory names under `_data/` (`iface_patch -h` lists
them).

After `make.bash` (or after upgrading the patch), clear the Go build cache:

```shell
go clean -cache
```

Otherwise a wasip1 `runtime` compiled **without** `-ifacefuncval` can be
reused by a tagged test. That mismatch traps:

```text
wasm trap: uninitialized element
```

## Run tests (darwin/arm64)

After `make.bash` on Apple Silicon, native tests do not need wasmtime:

```shell
export GOROOT=/path/to/go1.27.1
export PATH="$GOROOT/bin:$PATH"
go clean -cache
go test -tags goplus.ifacefuncval -v .
```

## Run tests (wasip1)

`GOOS=wasip1` produces a wasm binary. The host cannot execute it, so
`go test` **must** use `-exec wasmtime`. Install
[Wasmtime](https://wasmtime.dev/) first.

```shell
export GOROOT=/path/to/go1.27.1
export PATH="$GOROOT/bin:$PATH"
go clean -cache
GOOS=wasip1 GOARCH=wasm go test -exec wasmtime -tags goplus.ifacefuncval -v .
```

This does **not** work (no runner):

```shell
GOOS=wasip1 GOARCH=wasm go test -tags goplus.ifacefuncval .
```

Same tag via `GOFLAGS` (still need `-exec wasmtime`):

```shell
export GOFLAGS='-tags=goplus.ifacefuncval'
go clean -cache
GOOS=wasip1 GOARCH=wasm go test -exec wasmtime -v .
```

If `GOFLAGS` already has `-tags`, merge them:

```shell
export GOFLAGS='-tags=netgo,goplus.ifacefuncval'
```

Disable by omitting the tag. `GOFLAGS` cannot subtract a tag
(`-tags=-goplus.ifacefuncval` is not supported).

## What not to do

- `-tags goplus.ifacefuncval` does not check whether the compiler can
  unwrap tagged ifn. With an **official** gc, interface method calls trap
  at runtime (`uninitialized element` / invalid PC). Use a patched
  toolchain (wasm or arm64).
- `GOFLAGS=-gcflags=-ifacefuncval` is not enough: that only unwraps calls.
  Without the build tag, reflectx still compiles the icall path.
- Do not mix the patched `GOROOT` with another `go` on `PATH`.

## What the patch changes

* `cmd/internal/obj/wasm`: unwrap tagged `itab.Fun` (`makeFuncImpl*|1`) at
  indirect calls when `-ifacefuncval` is set
* `cmd/compile/internal/arm64`: same unwrap before `CALLinter` / `CALLtailinter`
* `runtime/asm_arm64.s`: unwrap in `CALLFN` before `BL (R20)`
* `cmd/compile` and `cmd/asm`: accept `-ifacefuncval` on wasm and arm64
* `cmd/go`: when `GOARCH` is `wasm` or `arm64` and `-tags goplus.ifacefuncval`
  is set (including via `GOFLAGS`), add `-ifacefuncval` to **`forcedGcflags`
  and `forcedAsmflags`** so it is part of the compile action ID

Replacement sources are in `_data/<VERSION>/` (that directory list is the
supported-version list). To add a Go version, copy `_data/go1.27.1/` to
`_data/go1.xx.y/` and adjust the snippets.
