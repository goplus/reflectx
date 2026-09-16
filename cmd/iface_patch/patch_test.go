package main

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func TestPatchWasmobjIdempotent(t *testing.T) {
	src := `package wasm

import "cmd/internal/obj"

func f(p *obj.Prog, appendp func(*obj.Prog, obj.As, ...obj.Addr) *obj.Prog) {
	switch call.To.Type {
	case obj.TYPE_MEM:
		p = appendp(p, ACall, call.To)
	case obj.TYPE_NONE:
		p = appendp(p, AI64Const, constAddr(16))
		p = appendp(p, ACallIndirect)
	}
	switch jmp.To.Type {
	case obj.TYPE_NONE:
		p = appendp(p, AI64Const, constAddr(16))
		p = appendp(p, ACallIndirect)
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "wasmobj.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := patchWasmobj(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected first patch to change file")
	}
	changed, err = patchWasmobj(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected second patch to be idempotent")
	}
}

func TestPatchCompileFlagInserts(t *testing.T) {
	src := `package base

import (
	"cmd/internal/obj"
	"cmd/internal/objabi"
	"log"
)

type CmdFlags struct {
	ErrorURL bool "help:\"print url\""
	Cfg      struct{}
}

func ParseFlags() {
	registerFlags()
	objabi.Flagparse(usage)
	counter.CountFlags("compile/flag:", *flag.CommandLine)
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "flag.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := patchCompileFlag(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	changed, err = patchCompileFlag(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected idempotent")
	}
}

func TestPatchGoGcInserts(t *testing.T) {
	src := `package work

func (gcToolchain) gc() {}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "gc.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := patchGoGc(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	if funcDecl(f, "wasmIfaceFuncval") == nil {
		t.Fatal("missing wasmIfaceFuncval func")
	}
	changed, err = patchGoGc(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected idempotent")
	}
}

func TestPatchGoInitInserts(t *testing.T) {
	src := `package work

func BuildInit() {
	modload.Init(ld)
	instrumentInit()
	buildModeInit()
	initCompilerConcurrencyPool()
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "init.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := patchGoInit(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	if !hasIdentExpr(funcDecl(f, "BuildInit").Body, "forcedGcflags") {
		t.Fatal("missing forcedGcflags")
	}
	changed, err = patchGoInit(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected idempotent")
	}
}

func testVer() versionData {
	v, err := loadVersion("go1.27.1")
	if err != nil {
		panic(err)
	}
	return v
}

func TestGoVersion(t *testing.T) {
	if _, err := goVersion("/no/such"); err == nil {
		t.Fatal("expected error")
	}
}

func TestPatchArm64SSA(t *testing.T) {
	src := `package arm64

import (
	"cmd/internal/obj"
	"cmd/internal/obj/arm64"
)

func ssaGenValue(s *ssagen.State, v *ssa.Value) {
	switch v.Op {
	case ssa.OpARM64CALLstatic, ssa.OpARM64CALLclosure, ssa.OpARM64CALLinter:
		s.Call(v)
	case ssa.OpARM64CALLtail, ssa.OpARM64CALLtailinter:
		s.TailCall(v)
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "ssa.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := patchArm64SSA(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	if funcDecl(f, "ssaGenIfaceFuncvalCall") == nil {
		t.Fatal("missing ssaGenIfaceFuncvalCall")
	}
	changed, err = patchArm64SSA(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected idempotent")
	}
}

func TestPatchAsmArm64(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/asm_arm64.s"
	src := "TEXT x(SB), $0\n" + arm64CallFNOld + "RET\n"
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	changed, err := patchAsmArm64(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	changed, err = patchAsmArm64(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected idempotent")
	}
}

func TestLoadVersion(t *testing.T) {
	if _, err := loadVersion("go1.0.0"); err == nil {
		t.Fatal("expected unsupported version")
	}
	if _, err := loadVersion("go1.27.1"); err != nil {
		t.Fatal(err)
	}
}

func TestHasStringLit(t *testing.T) {
	src := `package p
func f() { x := base.Tool("asm") }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "t.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !hasStringLit(f, "asm") {
		t.Fatal("want string lit asm")
	}
	if strings.Contains(src, "ifacefuncval") {
		t.Fatal("fixture should be unpatched")
	}
}
