# iface_patch

Optional: patch a Go **1.25.x**, **1.26.x**, or **1.27.x** source tree so
`-tags goplus.ifacefuncval` can replace icall stubs with tagged MakeFunc
funcvals (`itab.Fun = makeFuncImpl*|1`).

Supported `GOARCH` values: **wasm**, **arm64**, **amd64**, **386** (any `GOOS`
that uses that backend).

The tag is **explicit** (not a ToolTag). Without it, the patched compiler
emits the same interface-call sequence as unmodified Go.

reflectx does not check whether the compiler can unwrap tagged ifn. An
unpatched gc still compiles with the tag; interface method calls then trap
at runtime (`uninitialized element` / invalid PC).

## Patch and rebuild

Install:

```
go install github.com/goplus/reflectx/cmd/iface_patch@latest
```

`iface_patch` only edits source. Rebuild with **`make.bash`** in that tree
(`go install cmd/asm cmd/compile cmd/go` is not enough):

```shell
iface_patch /path/to/go
iface_patch -check /path/to/go
cd /path/to/go/src && ./make.bash   # Windows: make.bat
export GOROOT=/path/to/go
export PATH="$GOROOT/bin:$PATH"
go version   # from $GOROOT
```

Re-running on an already patched tree is a no-op. Supported versions are
the directories under `_data/` (`iface_patch -h` lists them).

`go clean -cache` can clear the build cache; it is not required.

Do not mix this `GOROOT` with another `go` on `PATH`.

## Run tests

### Native (linux/darwin amd64, arm64, or 386)

```shell
export GOROOT=/path/to/go
export PATH="$GOROOT/bin:$PATH"
go test -tags goplus.ifacefuncval -v .
```

`make.bash` installs stdlib **without** `-ifacefuncval`. The tagged `go test`
rebuilds `fmt`/`runtime` into `GOCACHE` so their interface calls unwrap.

### wasip1

The host cannot run a wasip1 binary. Install [Wasmtime](https://wasmtime.dev/)
and pass `-exec wasmtime`:

```shell
export GOROOT=/path/to/go
export PATH="$GOROOT/bin:$PATH"
GOOS=wasip1 GOARCH=wasm go test -exec wasmtime -tags goplus.ifacefuncval -v .
```

This does **not** run the tests:

```shell
GOOS=wasip1 GOARCH=wasm go test -tags goplus.ifacefuncval .
```

### GOFLAGS

```shell
export GOFLAGS='-tags=goplus.ifacefuncval'
go test -v .                                          # native
GOOS=wasip1 GOARCH=wasm go test -exec wasmtime -v .    # wasip1
```

If `GOFLAGS` already has `-tags`, merge them:

```shell
export GOFLAGS='-tags=netgo,goplus.ifacefuncval'
```

Disable by omitting the tag. `GOFLAGS` cannot subtract a tag
(`-tags=-goplus.ifacefuncval` is not supported).

`GOFLAGS=-gcflags=-ifacefuncval` is not enough: that only unwraps calls.
Without the build tag, reflectx still compiles the icall path.

## What the patch changes

Unwrap runs **only** when `objabi.EnableIfaceFuncval` is set (`-ifacefuncval`).
Untagged builds must match stock gc.

| Location | Change |
|---|---|
| `cmd/internal/objabi/ifacefuncval.go` | `EnableIfaceFuncval` (shared; not wasm-only) |
| `cmd/compile`, `cmd/asm` | `-ifacefuncval` on wasm, arm64, amd64, 386 |
| `cmd/go` | if `GOARCH` is wasm/arm64/amd64/386 and the tag is set, add `-ifacefuncval` to `forcedGcflags`/`forcedAsmflags` |
| `cmd/internal/obj/wasm` | unwrap tagged PC at indirect `CALL` |
| `cmd/compile/internal/arm64` | unwrap before `CALLinter` / `CALLtailinter` (CTXT=R26) |
| `cmd/compile/internal/amd64` | unwrap before `CALLinter` / `CALLtailinter` (CTXT=DX, call via R12) |
| `cmd/compile/internal/x86` | unwrap before `CALLinter` / `CALLtailinter` (CTXT=DX, 32-bit) |
| `runtime/asm_arm64.s` | unwrap in `CALLFN` before `BL (R20)` |
| `runtime/asm_amd64.s` | unwrap in `CALLFN` before `CALL R12` |
| `runtime/asm_386.s` | unwrap in `CALLFN` before `CALL AX` |

A tagged `itab.Fun` is `makeFuncImpl* | 1`. Code PCs and heap pointers are
even, so bit 0 is free. The unwrap sets CTXT to the untagged pointer and
calls the first word (`makeFuncStub`).

## `_data/`

Matching uses `VERSION` as the directory name:

1. Exact `_data/go1.N.x/` if that directory exists
2. Else `_data/go1.N/` (`go1.25.14` → `go1.25`)

Current series dirs and the trees they were taken from:

| Dir | Source |
|---|---|
| `go1.25` | go1.25.14 |
| `go1.26` | go1.26.8 |
| `go1.27` | go1.27.1 |

Go fragments use `//go:build ignore`. Assembly CALLFN old/new pairs are
plain `.s` files used as exact text replacements.

| File | Role |
|---|---|
| `objabi_ifacefuncval.go` | written to `src/cmd/internal/objabi/ifacefuncval.go` |
| `compile.go` / `asm.go` | set `objabi.EnableIfaceFuncval` |
| `gc.go` / `gc_flags.go` | `ifaceFuncvalEnabled` + forced flags |
| `wasmobj.go` / `unwrap.go` | wasm indirect-call unwrap |
| `arm64_ssa.go`, `arm64_callinter.go`, `arm64_calltailinter.go` | arm64 compiler |
| `amd64_ssa.go`, `amd64_callinter.go`, `amd64_calltailinter.go` | amd64 compiler |
| `x86_ssa.go`, `x86_callinter.go`, `x86_calltailinter.go` | 386 compiler |
| `arm64_callfn_{old,new}.s` | `CALLFN` in `runtime/asm_arm64.s` |
| `amd64_callfn_{old,new}.s` | `CALLFN` in `runtime/asm_amd64.s` |
| `386_callfn_{old,new}.s` | `CALLFN` in `runtime/asm_386.s` |

To support another series, copy `_data/go1.27/` to `_data/go1.28/`.
To pin a patch release that diverges, copy to `_data/go1.27.2/` (exact
match wins). Adjust the snippets until `iface_patch` matches that tree.
