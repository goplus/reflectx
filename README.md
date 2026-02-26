# reflectx
Golang reflect package hack tools

[![Build Status](https://github.com/goplus/reflectx/workflows/Go/badge.svg)](https://github.com/goplus/reflectx/workflows/Go/badge.svg)

### Go Version

- Go1.18 ~ Go1.26
- macOS Linux Windows WebAssembly

### ABI

- ABI0 stack-based ABI
- ABIInternal [register-based Go calling convention proposal](https://golang.org/design/40724-register-calling)

    - Go1.18: amd64 arm64 ppc64/ppc64le
    - Go1.19 ~ Go1.21: amd64 arm64 ppc64/ppc64le riscv64
    - Go1.22 ~ Go1.26: amd64 arm64 ppc64/ppc64le riscv64 loong64

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

### Method allocs
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

#### build linkname mode
```shell
go build -tags linknamefix -ldflags="-checklinkname=0"
```