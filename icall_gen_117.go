//go:build ignore
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
// +build go1.17,goexperiment.regabireflect

package reflectx

import (
	"log"
)

var (
	check_max_itype = true
	check_max_index = true
)

func icall(t int, i int, max int, ptrto bool, output bool) interface{} {
	if t >= max_itype_index {
		if check_max_itype {
			check_max_itype = false
			log.Println("warning, too many types interface call >", t)
		}
		return func(p, a unsafeptr) {}
	}
	if i >= max_icall_index {
		if check_max_index {
			check_max_index = false
			log.Println("warning, too many methods interface call >", i)
		}
		return func(p, a unsafeptr) {}
	}
	if ptrto {
		if output {
			return icall_ptr_output[t*max_icall_index+i]
		}
		return icall_ptr[t*max_icall_index+i]
	} else {
		if output {
			return icall_typ_output[t*max_icall_index+i]
		}
		return icall_typ[t*max_icall_index+i]
	}
}

const max_itype_index = $max_itype
const max_icall_index = $max_index
`

var templ_fn = `	func(p unsafeptr, a iparam) { i_y($itype, $index, p, a, $ptr) },
`
var templ_fn_output = `	func(p unsafeptr, a iparam) iparam { return i_y($itype, $index, p, a, $ptr) },
`

func main() {
	writeFile("./icall_go117.go", 64, 256)
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
	fnWrite("icall_typ_output", templ_fn_output, "false")
	fnWrite("icall_ptr", templ_fn, "true")
	fnWrite("icall_ptr_output", templ_fn_output, "true")

	ioutil.WriteFile(filename, buf.Bytes(), 0666)
}
