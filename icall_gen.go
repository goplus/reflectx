// +build ignore

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

var head = `// +build !js js,wasm

package reflectx

import (
	"log"
	"unsafe"
)

func icall(t int, i int, ptrto bool) interface{} {
	if t >= max_itype_index {
		log.Println("warning, not support too many types interface call", t)
		return nil
	}
	if i >= max_icall_index {
		log.Println("warning, not support too many methods interface call", i)
		return nil
	}
	if ptrto {
		return icall_ptr[t*$max_index+i]
	} else {
		return icall_typ[t*$max_index+i]
	}
}

const max_itype_index = $max_itype
const max_icall_index = $max_index
`

var templ_fn = `	func(p, a unsafe.Pointer) { i_x($itype, $index, p, unsafe.Pointer(&a), $ptr) },
`

func main() {
	writeFile("./icall.go", 64, 128)
}

func writeFile(filename string, max_itype int, max_index int) {
	var buf bytes.Buffer
	r := strings.NewReplacer("$max_itype", strconv.Itoa(max_itype),
		"$max_index", strconv.Itoa(max_index))
	buf.WriteString(r.Replace(head))

	fnWrite := func(name string, t string, ptr string) {
		buf.WriteString(fmt.Sprintf("\nvar %v = []interface{}{\n", name))
		for i := 0; i < max_itype; i++ {
			for j := 0; j < max_index; j++ {
				r := strings.NewReplacer("$itype", strconv.Itoa(i),
					"$index", strconv.Itoa(j),
					"$ptr", ptr)
				buf.WriteString(r.Replace(t))
			}
		}
		buf.WriteString("}\n")
	}
	fnWrite("icall_typ", templ_fn, "false")
	fnWrite("icall_ptr", templ_fn, "true")

	ioutil.WriteFile(filename, buf.Bytes(), 0666)
}
