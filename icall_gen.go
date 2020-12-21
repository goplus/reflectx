// +build ignore

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

var head = `package reflectx

func icall(i int, bytes int, ret bool, ptrto bool) (index int, v interface{}) {
	if i > max_icall_index || bytes > max_icall_bytes {
		index = -1
		return
	}
	index = bytes/$size + i*(max_icall_bytes/$size+1)
	if ptrto {
		if ret {
			v = icall_ptr[index]
		} else {
			v = icall_ptr_n[index]
		}
	} else {
		if ret {
			v = icall_struct[index]
		} else {
			v = icall_struct_n[index]
		}
	}
	return
}
`

var templ_0 = `	func(p uintptr) []byte { return i_x($index, p, nil, $ptr) },
`
var templ = `	func(p uintptr, a [$bytes]byte) []byte { return i_x($index, p, a[:], $ptr) },
`
var templ_n_0 = `	func(p uintptr) { i_x($index, p, nil, $ptr) },
`
var templ_n = `	func(p uintptr, a [$bytes]byte) { i_x($index, p, a[:], $ptr) },
`

func main() {
	writeFile("./icall_386.go", 4, 128, 128)
	writeFile("./icall_amd64.go", 8, 128, 256)
}

func writeFile(filename string, size int, max_index int, max_bytes int) {
	var buf bytes.Buffer
	r := strings.NewReplacer("$size", strconv.Itoa(size))
	buf.WriteString(r.Replace(head))
	buf.WriteString(fmt.Sprintf("\nconst max_icall_index = %v\n", max_index))
	buf.WriteString(fmt.Sprintf("const max_icall_bytes = %v\n", max_bytes))

	fnWrite := func(name string, t string, t0 string, ptr string) {
		buf.WriteString(fmt.Sprintf("\nvar %v = []interface{}{\n", name))
		for i := 0; i <= max_index; i++ {
			for j := 0; j <= max_bytes; j += size {
				r := strings.NewReplacer("$index", strconv.Itoa(i), "$bytes", strconv.Itoa(j), "$ptr", ptr)
				if j == 0 {
					r.WriteString(&buf, t0)
				} else {
					r.WriteString(&buf, t)
				}
			}
		}
		buf.WriteString("}\n")
	}
	fnWrite("icall_struct", templ, templ_0, "false")
	fnWrite("icall_struct_n", templ_n, templ_n_0, "false")
	fnWrite("icall_ptr", templ, templ_0, "true")
	fnWrite("icall_ptr_n", templ_n, templ_n_0, "true")

	ioutil.WriteFile(filename, buf.Bytes(), 0666)
}
