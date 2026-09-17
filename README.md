# reflectx
Golang reflect package hack tools

[![Build Status](https://github.com/goplus/reflectx/workflows/Go/badge.svg)](https://github.com/goplus/reflectx/workflows/Go/badge.svg)

### Go Version

- Go 1.21 ~ 1.27

### Compilers

- Go (gc)
- [LLGo](https://github.com/xgo-dev/llgo)

### Platforms

- macOS
- Linux
- Windows
- WebAssembly

### Go ABI

- ABI0 stack-based ABI
- ABIInternal [register-based Go calling convention proposal](https://golang.org/design/40724-register-calling)
    - Go1.21+: amd64 arm64 ppc64/ppc64le riscv64
    - Go1.23+: amd64 arm64 ppc64/ppc64le riscv64 loong64

### Field
* reflectx.CanSet
* reflectx.Field
* reflectx.FieldByIndex
* reflectx.FieldByName
* reflectx.FieldByNameFunc

### Named
* reflectx.StructOf(fs)
* reflectx.NamedTypeOf

* SetUnderlying
* SetTypeName

### Method
* reflectx.Method
* reflectx.MakeMethod

* reflectx.NewMethodSet
* reflectx.SetMethodSet

* reflectx.StructToMethodSet

### Interface
* reflectx.InterfaceOf
* reflectx.NamedInterfaceOf
* reflectx.NewInterfaceType
* reflectx.SetInterfaceType

### Context
* reflectx.NewContext()

### Method allocs (icall)
Default gc path: each installed method needs a unique ifn stub.

* allocs
```
import _ "github.com/goplus/reflectx/icall/icall[N]"
```
* install icall_gen
```
go install github.com/goplus/reflectx/cmd/icall_gen@latest
```
```
icall_gen -o icall1024.go -pkg main -size 1024
```

### ifacefuncval (optional)
Optional alternative to icall: `itab.Fun` is a tagged MakeFunc funcval
(`makeFuncImpl*|1`). No stub table; methods share `makeFuncStub`.

Needs a **patched Go 1.25.x, 1.26.x, or 1.27.x** and an explicit tag.
Supported `GOARCH`: wasm, arm64, amd64, 386.

* install iface_patch
```
go install github.com/goplus/reflectx/cmd/iface_patch@latest
```
```shell
iface_patch /path/to/go   # 1.25.x, 1.26.x, or 1.27.x
cd /path/to/go/src && ./make.bash
export GOROOT=/path/to/go
export PATH="$GOROOT/bin:$PATH"
go test -tags goplus.ifacefuncval .
# wasip1: GOOS=wasip1 GOARCH=wasm go test -exec wasmtime -tags goplus.ifacefuncval .
```

Without the tag, a patched compiler matches official gc. An unpatched
compiler still accepts the tag; interface method calls then trap.

See [cmd/iface_patch/README.md](cmd/iface_patch/README.md).

#### build linkname mode
```shell
go build -tags linknamefix -ldflags="-checklinkname=0"
```