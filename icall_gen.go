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
// +build !go1.17 go1.17,!goexperiment.regabireflect

package reflectx

func icall(i int) interface{} {
	return icall_array[i]
}
`

var templ_fn = `	func(p, a unsafeptr) { i_x($index, p, unsafeptr(&a)) },
`

func main() {
	writeFile("./icall.go", 1024)
}

func writeFile(filename string, max_index int) {
	var buf bytes.Buffer
	r := strings.NewReplacer("$max_index", strconv.Itoa(max_index))
	buf.WriteString(r.Replace(head))

	fnWrite := func(name string, t string) {
		buf.WriteString(fmt.Sprintf("\nvar %v = []interface{}{\n", name))
		for j := 0; j < max_index; j++ {
			r := strings.NewReplacer("$index", strconv.Itoa(j))
			buf.WriteString(r.Replace(t))
		}
		buf.WriteString("}\n")
	}
	fnWrite("icall_array", templ_fn)

	ioutil.WriteFile(filename, buf.Bytes(), 0666)
}
