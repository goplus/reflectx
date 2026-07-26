package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func writeRegAbi(filename string, pkgName string, size int) error {
	dir, f := filepath.Split(filename)
	if dir != "" {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("make dir %v error: %v", dir, err)
		}
	}
	type info struct {
		id   string
		abi  string
		file string
	}

	r := strings.NewReplacer("pkgname", pkgName, "1024", strconv.Itoa(size))
	gofiles := []info{
		{id: "", abi: ""},
		{id: "_linkname", abi: r.Replace(regabi_linkname)},
		{id: "_nolinkname", abi: r.Replace(regabi_nolinkname)},
	}
	for i := 0; i < len(gofiles); i++ {
		gofiles[i].file = filepath.Join(dir, strings.Replace(f, ".go", "_regabi"+gofiles[i].id+".go", 1))
	}
	var buf bytes.Buffer
	buf.WriteString(r.Replace(icall_regabi))
	buf.WriteString("\n")

	var ar []string
	for i := 0; i < size; i++ {
		buf.WriteString(fmt.Sprintf("func f%v()\n", i))
		ar = append(ar, fmt.Sprintf("f%v", i))
	}
	buf.WriteString(fmt.Sprintf(`
var (
	icall_fn = []func(){%v}
)
`, strings.Join(ar, ",")))

	for _, info := range gofiles {
		if info.id == "" {
			info.abi = buf.String()
		}
		err := ioutil.WriteFile(info.file, []byte(info.abi), 0644)
		if err != nil {
			return err
		}
	}

	infos := []info{
		{id: "amd64", abi: regabi_amd64},
		{id: "arm64", abi: regabi_arm64},
		{id: "ppc64x", abi: regabi_ppc64x},
		{id: "riscv64", abi: regabi_riscv64},
		{id: "loong64", abi: regabi_loong64},
	}
	for i := 0; i < len(infos); i++ {
		infos[i].file = filepath.Join(dir, strings.Replace(f, ".go", "_regabi_"+infos[i].id+".s", 1))
	}
	fnWrite := func(filename string, tmpl string, size int) error {
		var buf bytes.Buffer
		buf.WriteString(tmpl)
		buf.WriteString("\n")
		for i := 0; i < size; i++ {
			buf.WriteString(fmt.Sprintf("MAKE_FUNC_FN(·f%v,%v)\n", i, i))
		}
		return ioutil.WriteFile(filename, buf.Bytes(), 0644)
	}
	for _, info := range infos {
		err := fnWrite(info.file, info.abi, size)
		if err != nil {
			return err
		}
	}
	return nil
}

//go:embed _data/icall_regabi.go
var icall_regabi string

//go:embed _data/icall_regabi_amd64.s
var regabi_amd64 string

//go:embed _data/icall_regabi_arm64.s
var regabi_arm64 string

//go:embed _data/icall_regabi_ppc64x.s
var regabi_ppc64x string

//go:embed _data/icall_regabi_riscv64.s
var regabi_riscv64 string

//go:embed _data/icall_regabi_loong64.s
var regabi_loong64 string

//go:embed _data/icall_regabi_linkname.go
var regabi_linkname string

//go:embed _data/icall_regabi_nolinkname.go
var regabi_nolinkname string
