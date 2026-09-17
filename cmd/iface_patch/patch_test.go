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

func TestPatchCompileFlagUpgradesOld(t *testing.T) {
	src := `package base

import (
	"cmd/internal/objabi"
	"log"
)

type CmdFlags struct {
	ErrorURL     bool "help:\"print url\""
	IfaceFuncval bool ` + "`help:\"treat tagged itab.Fun as MakeFunc funcval (goplus.ifacefuncval; wasm/arm64/amd64)\"`" + `
	Cfg          struct{}
}

func ParseFlags() {
	registerFlags()
	objabi.Flagparse(usage)
	counter.CountFlags("compile/flag:", *flag.CommandLine)
	if Flag.IfaceFuncval {
		if buildcfg.GOARCH != "wasm" && buildcfg.GOARCH != "arm64" && buildcfg.GOARCH != "amd64" {
			log.Fatal("-ifacefuncval is only supported on wasm, arm64, and amd64")
		}
		objabi.EnableIfaceFuncval = true
	}
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
		t.Fatal("expected upgrade")
	}
	fn := funcDecl(f, "ParseFlags")
	stmt := stmtWithIdent(fn.Body, "EnableIfaceFuncval")
	if stmt == nil || !ifaceFuncvalBlockCurrent(stmt) {
		t.Fatal("ParseFlags still missing 386")
	}
	if hasSelectorExpr(stmt, "wasm", "EnableIfaceFuncval") {
		t.Fatal("EnableIfaceFuncval must stay on objabi")
	}
	changed, err = patchCompileFlag(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected idempotent after upgrade")
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
	if funcDecl(f, "ifaceFuncvalEnabled") == nil {
		t.Fatal("missing ifaceFuncvalEnabled func")
	}
	changed, err = patchGoGc(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected idempotent")
	}
}

func TestPatchGoGcRenamesOld(t *testing.T) {
	src := `package work

func wasmIfaceFuncval() bool { return false }
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
		t.Fatal("expected rename")
	}
	if funcDecl(f, "wasmIfaceFuncval") != nil {
		t.Fatal("old name still present")
	}
	if funcDecl(f, "ifaceFuncvalEnabled") == nil {
		t.Fatal("missing ifaceFuncvalEnabled")
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

func TestPatchGoInitRenamesOld(t *testing.T) {
	src := `package work

func BuildInit() {
	buildModeInit()
	if wasmIfaceFuncval() {
		forcedGcflags = append(forcedGcflags, "-ifacefuncval")
	}
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
		t.Fatal("expected rename")
	}
	if hasIdentExpr(funcDecl(f, "BuildInit").Body, "wasmIfaceFuncval") {
		t.Fatal("old name still present")
	}
	if !hasIdentExpr(funcDecl(f, "BuildInit").Body, "ifaceFuncvalEnabled") {
		t.Fatal("missing ifaceFuncvalEnabled")
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

func TestPatchArm64SSAMissingLayout(t *testing.T) {
	src := `package arm64

import "cmd/internal/obj/arm64"

func ssaGenValue(s *ssagen.State, v *ssa.Value) {
	switch v.Op {
	case ssa.OpARM64CALLstatic:
		s.Call(v)
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "ssa.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	_, err = patchArm64SSA(fset, f, testVer())
	if err == nil {
		t.Fatal("expected error when CALL cases are missing")
	}
}

func TestPatchAsmArm64(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/asm_arm64.s"
	old := testVer().bytes("arm64_callfn_old.s")
	src := "TEXT x(SB), $0\n" + string(old) + "RET\n"
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	changed, err := patchAsmReplace(path, old, testVer().bytes("arm64_callfn_new.s"), false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	changed, err = patchAsmReplace(path, old, testVer().bytes("arm64_callfn_new.s"), false)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected idempotent")
	}
}

func TestPatchAmd64SSA(t *testing.T) {
	src := `package amd64

import (
	"cmd/internal/obj"
	"cmd/internal/obj/x86"
)

func ssaGenValue(s *ssagen.State, v *ssa.Value) {
	switch v.Op {
	case ssa.OpAMD64CALLstatic, ssa.OpAMD64CALLtail, ssa.OpAMD64CALLtailinter:
		s.Call(v)
	case ssa.OpAMD64CALLclosure, ssa.OpAMD64CALLinter:
		s.Call(v)
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "ssa.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := patchAmd64SSA(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	if funcDecl(f, "ssaGenIfaceFuncvalCall") == nil {
		t.Fatal("missing ssaGenIfaceFuncvalCall")
	}
	changed, err = patchAmd64SSA(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected idempotent")
	}
}

func TestPatchAmd64SSAMissingLayout(t *testing.T) {
	src := `package amd64

func ssaGenValue(s *ssagen.State, v *ssa.Value) {
	switch v.Op {
	case ssa.OpAMD64CALLstatic:
		s.Call(v)
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "ssa.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	_, err = patchAmd64SSA(fset, f, testVer())
	if err == nil {
		t.Fatal("expected error when CALL cases are missing")
	}
}

func TestPatchAsmAmd64(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/asm_amd64.s"
	old := testVer().bytes("amd64_callfn_old.s")
	src := "TEXT x(SB), $0\n" + string(old) + "RET\n"
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	changed, err := patchAsmReplace(path, old, testVer().bytes("amd64_callfn_new.s"), false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	changed, err = patchAsmReplace(path, old, testVer().bytes("amd64_callfn_new.s"), false)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected idempotent")
	}
}

func TestPatch386SSA(t *testing.T) {
	src := `package x86

import "cmd/internal/obj/x86"

func ssaGenValue(s *ssagen.State, v *ssa.Value) {
	switch v.Op {
	case ssa.Op386CALLstatic, ssa.Op386CALLclosure, ssa.Op386CALLinter:
		s.Call(v)
	case ssa.Op386CALLtail, ssa.Op386CALLtailinter:
		s.TailCall(v)
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "ssa.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := patch386SSA(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	if funcDecl(f, "ssaGenIfaceFuncvalCall") == nil {
		t.Fatal("missing ssaGenIfaceFuncvalCall")
	}
	changed, err = patch386SSA(fset, f, testVer())
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected idempotent")
	}
}

func TestPatch386SSAMissingLayout(t *testing.T) {
	src := `package x86

func ssaGenValue(s *ssagen.State, v *ssa.Value) {
	switch v.Op {
	case ssa.Op386CALLstatic:
		s.Call(v)
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "ssa.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	_, err = patch386SSA(fset, f, testVer())
	if err == nil {
		t.Fatal("expected error when CALL cases are missing")
	}
}

func TestPatchAsm386(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/asm_386.s"
	old := testVer().bytes("386_callfn_old.s")
	src := "TEXT x(SB), $0\n" + string(old) + "RET\n"
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	changed, err := patchAsmReplace(path, old, testVer().bytes("386_callfn_new.s"), false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	changed, err = patchAsmReplace(path, old, testVer().bytes("386_callfn_new.s"), false)
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
	if _, err := loadVersion("go1.26.8"); err != nil {
		t.Fatal(err)
	}
	if _, err := loadVersion("go1.25.14"); err != nil {
		t.Fatal(err)
	}
}

func mustVer(name string) versionData {
	v, err := loadVersion(name)
	if err != nil {
		panic(err)
	}
	return v
}

func noTailinterVers() []string {
	return []string{"go1.25.14", "go1.26.8"}
}

func TestPatchAmd64SSANoTailinter(t *testing.T) {
	src := `package amd64

import "cmd/internal/obj/x86"

func ssaGenValue(s *ssagen.State, v *ssa.Value) {
	switch v.Op {
	case ssa.OpAMD64CALLstatic, ssa.OpAMD64CALLtail:
		s.Call(v)
	case ssa.OpAMD64CALLclosure, ssa.OpAMD64CALLinter:
		s.Call(v)
	}
}
`
	for _, ver := range noTailinterVers() {
		t.Run(ver, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "ssa.go", src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			v := mustVer(ver)
			changed, err := patchAmd64SSA(fset, f, v)
			if err != nil {
				t.Fatal(err)
			}
			if !changed {
				t.Fatal("expected change")
			}
			if funcDecl(f, "ssaGenIfaceFuncvalCall") == nil {
				t.Fatal("missing helper")
			}
			if hasIdentExpr(f, "OpAMD64CALLtailinter") {
				t.Fatalf("%s must not introduce CALLtailinter", ver)
			}
			changed, err = patchAmd64SSA(fset, f, v)
			if err != nil {
				t.Fatal(err)
			}
			if changed {
				t.Fatal("expected idempotent")
			}
		})
	}
}

func TestPatchArm64SSANoTailinter(t *testing.T) {
	src := `package arm64

import "cmd/internal/obj/arm64"

func ssaGenValue(s *ssagen.State, v *ssa.Value) {
	switch v.Op {
	case ssa.OpARM64CALLstatic, ssa.OpARM64CALLclosure, ssa.OpARM64CALLinter:
		s.Call(v)
	case ssa.OpARM64CALLtail:
		s.TailCall(v)
	}
}
`
	for _, ver := range noTailinterVers() {
		t.Run(ver, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "ssa.go", src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			v := mustVer(ver)
			changed, err := patchArm64SSA(fset, f, v)
			if err != nil {
				t.Fatal(err)
			}
			if !changed {
				t.Fatal("expected change")
			}
			if hasIdentExpr(f, "OpARM64CALLtailinter") {
				t.Fatalf("%s must not introduce CALLtailinter", ver)
			}
			changed, err = patchArm64SSA(fset, f, v)
			if err != nil {
				t.Fatal(err)
			}
			if changed {
				t.Fatal("expected idempotent")
			}
		})
	}
}

func TestPatch386SSANoTailinter(t *testing.T) {
	src := `package x86

import "cmd/internal/obj/x86"

func ssaGenValue(s *ssagen.State, v *ssa.Value) {
	switch v.Op {
	case ssa.Op386CALLstatic, ssa.Op386CALLclosure, ssa.Op386CALLinter:
		s.Call(v)
	case ssa.Op386CALLtail:
		s.TailCall(v)
	}
}
`
	for _, ver := range noTailinterVers() {
		t.Run(ver, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "ssa.go", src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			v := mustVer(ver)
			changed, err := patch386SSA(fset, f, v)
			if err != nil {
				t.Fatal(err)
			}
			if !changed {
				t.Fatal("expected change")
			}
			if hasIdentExpr(f, "Op386CALLtailinter") {
				t.Fatalf("%s must not introduce CALLtailinter", ver)
			}
			changed, err = patch386SSA(fset, f, v)
			if err != nil {
				t.Fatal(err)
			}
			if changed {
				t.Fatal("expected idempotent")
			}
		})
	}
}

func TestPatchAsmFlagsNoStd(t *testing.T) {
	src := `package flags

import (
	"cmd/internal/obj"
	"cmd/internal/objabi"
	"flag"
	"fmt"
)

var (
	Shared  = flag.Bool("shared", false, "shared")
	Spectre = flag.String("spectre", "", "spectre")
)

func Parse() {
	objabi.Flagparse(Usage)
	if flag.NArg() == 0 {
		flag.Usage()
	}
}
`
	for _, ver := range noTailinterVers() {
		t.Run(ver, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "flags.go", src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			v := mustVer(ver)
			changed, err := patchAsmFlags(fset, f, v)
			if err != nil {
				t.Fatal(err)
			}
			if !changed {
				t.Fatal("expected change")
			}
			if !hasIdent(f, "IfaceFuncval") {
				t.Fatal("missing IfaceFuncval flag")
			}
			if !hasIdentExpr(funcDecl(f, "Parse").Body, "EnableIfaceFuncval") {
				t.Fatal("Parse missing EnableIfaceFuncval")
			}
			changed, err = patchAsmFlags(fset, f, v)
			if err != nil {
				t.Fatal(err)
			}
			if changed {
				t.Fatal("expected idempotent")
			}
		})
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
