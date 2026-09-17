package main

import (
	"embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"strconv"
	"strings"
)

//go:embed _data
var patchData embed.FS

type versionData struct {
	ver string
}

func supportedVersions() []string {
	entries, err := fs.ReadDir(patchData, "_data")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "go") {
			out = append(out, e.Name())
		}
	}
	return out
}

func loadVersion(ver string) (versionData, error) {
	ver = strings.TrimSpace(ver)
	if dir := matchDataDir(ver); dir != "" {
		return versionData{ver: dir}, nil
	}
	return versionData{}, fmt.Errorf("unsupported Go version %q (have %s)", ver, strings.Join(supportedVersions(), ", "))
}

// matchDataDir prefers _data/<VERSION>/, else _data/go1.N/.
func matchDataDir(ver string) string {
	if _, err := fs.Stat(patchData, path.Join("_data", ver)); err == nil {
		return ver
	}
	series := seriesOf(ver)
	if series == "" || series == ver {
		return ""
	}
	if _, err := fs.Stat(patchData, path.Join("_data", series)); err == nil {
		return series
	}
	return ""
}

func seriesOf(ver string) string {
	if !strings.HasPrefix(ver, "go") {
		return ""
	}
	parts := strings.Split(ver, ".")
	switch len(parts) {
	case 2:
		return ver
	case 3:
		if _, err := strconv.Atoi(parts[2]); err != nil {
			return ""
		}
		return parts[0] + "." + parts[1]
	default:
		return ""
	}
}

func (v versionData) read(name string) string {
	b, err := patchData.ReadFile(path.Join("_data", v.ver, name))
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (v versionData) goSrc(name string) []byte {
	s := v.read(name)
	s = strings.TrimPrefix(s, "//go:build ignore\n\n")
	return []byte(s)
}

func (v versionData) bytes(name string) []byte {
	s := strings.ReplaceAll(v.read(name), "\r\n", "\n")
	s = strings.TrimRight(s, "\n") + "\n"
	return []byte(s)
}

func (v versionData) stmts(name string) []ast.Stmt {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, v.read(name), 0)
	if err != nil {
		panic(err)
	}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if ok && fn.Body != nil {
			return fn.Body.List
		}
	}
	panic(name + ": no func body")
}

func (v versionData) decls(name string) []ast.Decl {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, v.read(name), parser.ParseComments)
	if err != nil {
		panic(err)
	}
	var out []ast.Decl
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if ok && gen.Tok == token.IMPORT {
			continue
		}
		out = append(out, d)
	}
	return out
}
