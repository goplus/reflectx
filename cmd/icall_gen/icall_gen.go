package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	output     = flag.String("o", "", "set output file path")
	pkgName    = flag.String("pkg", "icall", "set package name")
	presetSize = flag.Int("size", 0, "set methods preset size")
)

func main() {
	flag.Parse()
	if *output == "" || *pkgName == "" || *presetSize == 0 {
		flag.Usage()
		return
	}
	// write icall.go
	err := writeFile(*output, *pkgName, *presetSize)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	// write icall_regabi.go
	err = writeRegAbi(*output, *pkgName, *presetSize)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func writeFile(filename string, pkgName string, size int) error {
	dir, _ := filepath.Split(filename)
	if dir != "" {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			return fmt.Errorf("make dir %v error: %v", dir, err)
		}
	}

	var buf bytes.Buffer
	r := strings.NewReplacer("pkgname", pkgName, "1024", strconv.Itoa(size))
	buf.WriteString(r.Replace(head))
	buf.WriteString("\n")

	fnWrite := func(name string, t string) {
		buf.WriteString(fmt.Sprintf("\nvar %v = []interface{}{\n", name))
		for i := 0; i < size; i++ {
			r := strings.NewReplacer("$index", strconv.Itoa(i))
			buf.WriteString(r.Replace(t))
		}
		buf.WriteString("}\n")
	}
	fnWrite("icall_array", templ_fn)
	return ioutil.WriteFile(filename, buf.Bytes(), 0666)
}

//go:embed _data/icall.go
var head string

var templ_fn = `	func(p, a unsafeptr) { i_x($index, p, unsafeptr(&a)) },
`
