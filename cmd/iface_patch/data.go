package main

import (
	"embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
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
	dir := path.Join("_data", ver)
	if _, err := fs.Stat(patchData, dir); err != nil {
		return versionData{}, fmt.Errorf("unsupported Go version %q (have %s)", ver, strings.Join(supportedVersions(), ", "))
	}
	return versionData{ver: ver}, nil
}

func (v versionData) read(name string) string {
	b, err := patchData.ReadFile(path.Join("_data", v.ver, name))
	if err != nil {
		panic(err)
	}
	return string(b)
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
