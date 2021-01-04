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

import "unsafe"

func icall(i int, ptrto bool) interface{} {
	if i > max_icall_index {
		return nil
	}
	if ptrto {
		return icall_ptr[i]
	} else {
		return icall_typ[i]
	}
}

const max_icall_index = $max_index
`

var templ_fn = `	func(p, a unsafe.Pointer) { i_x($index, p, unsafe.Pointer(&a), $ptr) },
`

func main() {
	writeFile("./icall.go", 1024)
}

func writeFile(filename string, max_index int) {
	var buf bytes.Buffer
	r := strings.NewReplacer("$max_index", strconv.Itoa(max_index))
	buf.WriteString(r.Replace(head))

	fnWrite := func(name string, t string, ptr string) {
		buf.WriteString(fmt.Sprintf("\nvar %v = []interface{}{\n", name))
		for i := 0; i <= max_index; i++ {
			r := strings.NewReplacer("$index", strconv.Itoa(i), "$ptr", ptr)
			buf.WriteString(r.Replace(t))
		}
		buf.WriteString("}\n")
	}
	fnWrite("icall_typ", templ_fn, "false")
	fnWrite("icall_ptr", templ_fn, "true")

	ioutil.WriteFile(filename, buf.Bytes(), 0666)
}
