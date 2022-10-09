//go:build js && !wasm
// +build js,!wasm

package reflectx

import (
	"log"
	_ "unsafe"
)

//go:linkname name_nameLen reflect.name.nameLen
func name_nameLen(name) int

//go:linkname name_isExported reflect.name.isExported
func name_isExported(name) bool

//go:linkname name_name reflect.name.name
func name_name(name) string

//go:linkname name_tag reflect.name.tag
func name_tag(name) string

//go:linkname name_tag reflect.name.pkgPath
func name_pkgPath(name) string

type name struct {
	bytes *byte
}

func (n name) isExported() bool {
	return name_isExported(n)
}

func (n name) nameLen() int {
	return name_nameLen(n)
}

func (n name) name() (s string) {
	return name_name(n)
}

func (n name) tag() (s string) {
	return name_tag(n)
}

func (n name) pkgPath() string {
	return name_pkgPath(n)
}

func (n name) setPkgPath(pkgpath nameOff) bool {
	log.Println("not impl setPkgPath")
	return true
}

func newNameEx(n, tag string, exported bool, pkgpath bool) name {
	return newName(n, tag, exported)
}
