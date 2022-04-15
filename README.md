# reflectx
Golang reflect package hack tools

[![Go1.14](https://github.com/goplus/reflectx/workflows/Go1.14/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go114.yml)
[![Go1.15](https://github.com/goplus/reflectx/workflows/Go1.15/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go115.yml)
[![Go1.16](https://github.com/goplus/reflectx/workflows/Go1.16/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go116.yml)
[![Go1.17](https://github.com/goplus/reflectx/workflows/Go1.17/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go117.yml)
[![Go1.18](https://github.com/goplus/reflectx/workflows/Go1.18/badge.svg)](https://github.com/goplus/reflectx/actions/workflows/go118.yml)

### ABI

support ABI0 and ABIInternal

- ABI0 stack-based ABI
- ABIInternal [register-based Go calling convention proposal](https://golang.org/design/40724-register-calling)

	- Go1.17: amd64
	- Go1.18: amd64 arm64 ppc64/ppc64le



### Field
* reflectx.CanSet
* reflectx.Field
* reflectx.FieldByIndex
* reflectx.FieldByName
* reflectx.FieldByNameFunc

### Named
* reflectx.StructOf(fs)
* reflectx.NamedTypeOf
* reflectx.IsNamed
* reflectx.ToNamed

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


### methods preset
* preset
```
import _ "github.com/goplus/reflectx/icall/icall[2^n]"
```
* install icall_gen
```
go get github.com/goplus/reflectx/cmd/icall_gen
```
```
icall_gen -o icall1024.go -pkg main -size 1024
```
