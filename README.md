# reflectx
Golang reflect package hack tools

[![Go1.14](https://github.com/goplus/reflectx/workflows/Go1.14/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go114.yml)
[![Go1.15](https://github.com/goplus/reflectx/workflows/Go1.15/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go115.yml)
[![Go1.16](https://github.com/goplus/reflectx/workflows/Go1.16/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go116.yml)
[![Go1.17](https://github.com/goplus/reflectx/workflows/Go1.17/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go117.yml)
[![Go1.18](https://github.com/goplus/reflectx/workflows/Go1.18/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go118.yml)
[![Go1.19](https://github.com/goplus/reflectx/workflows/Go1.19/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go119.yml)
[![Go1.20](https://github.com/goplus/reflectx/workflows/Go1.20/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go120.yml)
[![Go1.21](https://github.com/goplus/reflectx/workflows/Go1.21/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go121.yml)
[![Go1.22](https://github.com/goplus/reflectx/workflows/Go1.22/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go122.yml)
[![Go1.23](https://github.com/goplus/reflectx/workflows/Go1.23/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go123.yml)

### Build

- Go1.14 ~ Go1.22

  `go build`

- Go1.23

  `go build -ldflags="-checklinkname=0"`

### ABI

support ABI0 and ABIInternal

- ABI0 stack-based ABI
- ABIInternal [register-based Go calling convention proposal](https://golang.org/design/40724-register-calling)

	- Go1.17: amd64
	- Go1.18: amd64 arm64 ppc64/ppc64le
	- Go1.19~Go1.23: amd64 arm64 ppc64/ppc64le riscv64

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
go get github.com/goplus/reflectx/cmd/icall_gen
```
```
icall_gen -o icall1024.go -pkg main -size 1024
```
