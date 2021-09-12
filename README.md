# reflectx
Golang reflect package hack tools

[![Go1.14](https://github.com/goplus/reflectx/workflows/Go1.14/badge.svg)](https://github.com/goplus/reflectx/actions?query=workflow%3AGo1.14)
[![Go1.15](https://github.com/goplus/reflectx/workflows/Go1.15/badge.svg)](https://github.com/goplus/reflectx/actions?query=workflow%3AGo1.15)
[![Go1.16](https://github.com/goplus/reflectx/workflows/Go1.16/badge.svg)](https://github.com/goplus/reflectx/actions?query=workflow%3AGo1.16)

**Go1.17**

set env

`GOEXPERIMENT=noregabi`


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

### SetElem
* reflectx.SetElem
