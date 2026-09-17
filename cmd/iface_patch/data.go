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
	if _, err := fs.Stat(patchData, path.Join("_data", ver)); err == nil {
		return versionData{ver: ver}, nil
	}
	for _, series := range []string{"go1.25", "go1.26", "go1.27"} {
		if alias := resolveSeries(ver, series); alias != "" {
			return versionData{ver: alias}, nil
		}
	}
	return versionData{}, fmt.Errorf("unsupported Go version %q (have %s)", ver, strings.Join(supportedVersions(), ", "))
}

func isSeriesVersion(ver, series string) bool {
	if ver == series {
		return true
	}
	rest, ok := strings.CutPrefix(ver, series+".")
	if !ok || rest == "" {
		return false
	}
	_, err := strconv.Atoi(rest)
	return err == nil
}

// resolveSeries maps go1.N / go1.N.x onto the highest _data/go1.N.* dir.
func resolveSeries(ver, series string) string {
	if !isSeriesVersion(ver, series) {
		return ""
	}
	best := ""
	bestPatch := -1
	prefix := series + "."
	for _, d := range supportedVersions() {
		if d != series && !strings.HasPrefix(d, prefix) {
			continue
		}
		patch := 0
		if strings.HasPrefix(d, prefix) {
			p, err := strconv.Atoi(d[len(prefix):])
			if err != nil {
				continue
			}
			patch = p
		}
		if patch > bestPatch {
			bestPatch = patch
			best = d
		}
	}
	return best
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
