package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
)

type patchFunc func(*token.FileSet, *ast.File, versionData) (bool, error)

func applyAll(root string, v versionData, check bool) (int, error) {
	patches := []struct {
		rel string
		fn  patchFunc
	}{
		{"src/cmd/internal/obj/wasm/wasmobj.go", patchWasmobj},
		{"src/cmd/compile/internal/base/flag.go", patchCompileFlag},
		{"src/cmd/asm/internal/flags/flags.go", patchAsmFlags},
		{"src/cmd/go/internal/work/gc.go", patchGoGc},
		{"src/cmd/go/internal/work/init.go", patchGoInit},
		{"src/cmd/compile/internal/arm64/ssa.go", patchArm64SSA},
		{"src/cmd/compile/internal/amd64/ssa.go", patchAmd64SSA},
	}
	n := 0
	{
		rel := "src/cmd/internal/objabi/ifacefuncval.go"
		changed, err := writeGoSrc(filepath.Join(root, rel), v.goSrc("objabi_ifacefuncval.go"), check)
		if err != nil {
			return n, fmt.Errorf("%s: %w", rel, err)
		}
		if changed {
			n++
			if !check {
				fmt.Println(rel)
			}
		}
	}
	for _, p := range patches {
		path := filepath.Join(root, p.rel)
		changed, err := patchFile(path, p.fn, v, check)
		if err != nil {
			return n, fmt.Errorf("%s: %w", p.rel, err)
		}
		if changed {
			n++
			if !check {
				fmt.Println(p.rel)
			}
		}
	}
	for _, rel := range []string{"src/runtime/asm_arm64.s", "src/runtime/asm_amd64.s"} {
		var changed bool
		var err error
		if rel == "src/runtime/asm_arm64.s" {
			changed, err = patchAsmReplace(filepath.Join(root, rel), v.bytes("arm64_callfn_old.s"), v.bytes("arm64_callfn_new.s"), check)
		} else {
			changed, err = patchAsmReplace(filepath.Join(root, rel), v.bytes("amd64_callfn_old.s"), v.bytes("amd64_callfn_new.s"), check)
		}
		if err != nil {
			return n, fmt.Errorf("%s: %w", rel, err)
		}
		if changed {
			n++
			if !check {
				fmt.Println(rel)
			}
		}
	}
	return n, nil
}

func patchFile(path string, fn patchFunc, v versionData, check bool) (bool, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return false, err
	}
	changed, err := fn(fset, f, v)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	if check {
		return true, nil
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		return false, err
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(path, out, 0644)
}

func writeGoSrc(path string, want []byte, check bool) (bool, error) {
	old, err := os.ReadFile(path)
	if err == nil && bytes.Equal(old, want) {
		return false, nil
	}
	if check {
		return true, nil
	}
	return true, os.WriteFile(path, want, 0644)
}

func hasIdent(f *ast.File, name string) bool {
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if ok && id.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}

func importPath(spec *ast.ImportSpec) string {
	s, err := strconv.Unquote(spec.Path.Value)
	if err != nil {
		return ""
	}
	return s
}

func hasImport(f *ast.File, path string) bool {
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		for _, s := range gen.Specs {
			if importPath(s.(*ast.ImportSpec)) == path {
				return true
			}
		}
	}
	return false
}

func addImportAfter(f *ast.File, after, path string) {
	if hasImport(f, path) {
		return
	}
	spec := &ast.ImportSpec{Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(path)}}
	insert := func(gen *ast.GenDecl, i int) {
		specs := make([]ast.Spec, 0, len(gen.Specs)+1)
		specs = append(specs, gen.Specs[:i+1]...)
		specs = append(specs, spec)
		specs = append(specs, gen.Specs[i+1:]...)
		gen.Specs = specs
		f.Imports = append(f.Imports, spec)
	}
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		for i, s := range gen.Specs {
			if importPath(s.(*ast.ImportSpec)) != after {
				continue
			}
			insert(gen, i)
			return
		}
	}
	for i := len(f.Decls) - 1; i >= 0; i-- {
		gen, ok := f.Decls[i].(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		insert(gen, len(gen.Specs)-1)
		return
	}
}

func insertStmts(list []ast.Stmt, i int, extra ...ast.Stmt) []ast.Stmt {
	out := make([]ast.Stmt, 0, len(list)+len(extra))
	out = append(out, list[:i]...)
	out = append(out, extra...)
	out = append(out, list[i:]...)
	return out
}

func isSelector(e ast.Expr, pkg, name string) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == pkg
}

func isIdent(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}

func callName(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	}
	return ""
}

func isTYPE_NONE(cc *ast.CaseClause) bool {
	for _, e := range cc.List {
		if isSelector(e, "obj", "TYPE_NONE") {
			return true
		}
	}
	return false
}

func isUnwrapAssign(stmt ast.Stmt) bool {
	as, ok := stmt.(*ast.AssignStmt)
	if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
		return false
	}
	call, ok := as.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	return isIdent(call.Fun, "unwrapIfaceFuncvalPC")
}

func isAppendpAI64Const(stmt ast.Stmt) bool {
	as, ok := stmt.(*ast.AssignStmt)
	if !ok || len(as.Rhs) != 1 {
		return false
	}
	call, ok := as.Rhs[0].(*ast.CallExpr)
	if !ok || !isIdent(call.Fun, "appendp") || len(call.Args) < 2 {
		return false
	}
	return isIdent(call.Args[1], "AI64Const")
}

func patchWasmobj(_ *token.FileSet, f *ast.File, v versionData) (bool, error) {
	changed := false
	ast.Inspect(f, func(n ast.Node) bool {
		cc, ok := n.(*ast.CaseClause)
		if !ok || !isTYPE_NONE(cc) || len(cc.Body) == 0 {
			return true
		}
		if isUnwrapAssign(cc.Body[0]) {
			return true
		}
		if !isAppendpAI64Const(cc.Body[0]) {
			return true
		}
		unwrap := v.stmts("unwrap.go")[0]
		cc.Body = insertStmts(cc.Body, 0, unwrap)
		changed = true
		return true
	})
	if !hasImport(f, "cmd/internal/objabi") {
		addImportAfter(f, "cmd/internal/obj", "cmd/internal/objabi")
		changed = true
	}
	decls := v.decls("wasmobj.go")
	if fn := funcDecl(f, "unwrapIfaceFuncvalPC"); fn != nil {
		if !hasIdentExpr(fn, "objabi") {
			for _, d := range decls {
				if u, ok := d.(*ast.FuncDecl); ok && u.Name.Name == "unwrapIfaceFuncvalPC" {
					replaceFuncDecl(f, "unwrapIfaceFuncvalPC", u)
					changed = true
				}
			}
		}
	} else {
		f.Decls = append(f.Decls, decls...)
		changed = true
	}
	if removeVar(f, "EnableIfaceFuncval") {
		changed = true
	}
	return changed, nil
}

func removeVar(f *ast.File, name string) bool {
	out := f.Decls[:0]
	removed := false
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			out = append(out, d)
			continue
		}
		specs := gen.Specs[:0]
		for _, s := range gen.Specs {
			vs := s.(*ast.ValueSpec)
			keep := false
			for _, n := range vs.Names {
				if n.Name != name {
					keep = true
					break
				}
			}
			if keep {
				specs = append(specs, s)
			} else {
				removed = true
			}
		}
		if len(specs) > 0 {
			gen.Specs = specs
			out = append(out, gen)
		}
	}
	if removed {
		f.Decls = out
	}
	return removed
}

func removeImport(f *ast.File, path string) bool {
	removed := false
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		specs := gen.Specs[:0]
		for _, s := range gen.Specs {
			if importPath(s.(*ast.ImportSpec)) == path {
				removed = true
				continue
			}
			specs = append(specs, s)
		}
		gen.Specs = specs
	}
	return removed
}

func typeSpec(f *ast.File, name string) *ast.TypeSpec {
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, s := range gen.Specs {
			ts := s.(*ast.TypeSpec)
			if ts.Name.Name == name {
				return ts
			}
		}
	}
	return nil
}

func structType(ts *ast.TypeSpec) *ast.StructType {
	st, _ := ts.Type.(*ast.StructType)
	return st
}

func fieldNamed(fl *ast.FieldList, name string) bool {
	if fl == nil {
		return false
	}
	for _, field := range fl.List {
		for _, n := range field.Names {
			if n.Name == name {
				return true
			}
		}
	}
	return false
}

func patchCompileFlag(_ *token.FileSet, f *ast.File, v versionData) (bool, error) {
	changed := false
	if hasImport(f, "cmd/internal/obj/wasm") {
		if removeImport(f, "cmd/internal/obj/wasm") {
			changed = true
		}
	}
	ts := typeSpec(f, "CmdFlags")
	if ts == nil {
		return false, fmt.Errorf("CmdFlags not found")
	}
	st := structType(ts)
	if st == nil {
		return false, fmt.Errorf("CmdFlags is not a struct")
	}
	if !fieldNamed(st.Fields, "IfaceFuncval") {
		idx := -1
		for i, field := range st.Fields.List {
			if len(field.Names) == 1 && field.Names[0].Name == "Cfg" {
				idx = i
				break
			}
		}
		if idx < 0 {
			return false, fmt.Errorf("CmdFlags.Cfg not found")
		}
		field := &ast.Field{
			Names: []*ast.Ident{ast.NewIdent("IfaceFuncval")},
			Type:  ast.NewIdent("bool"),
			Tag:   &ast.BasicLit{Kind: token.STRING, Value: "`help:\"wasm: treat tagged itab.Fun as MakeFunc funcval (goplus.ifacefuncval)\"`"},
		}
		st.Fields.List = append(st.Fields.List[:idx], append([]*ast.Field{field}, st.Fields.List[idx:]...)...)
		changed = true
	}
	if insertAfterCountFlags(f, v) {
		changed = true
	}
	return changed, nil
}

func insertAfterCountFlags(f *ast.File, v versionData) bool {
	fn := funcDecl(f, "ParseFlags")
	if fn == nil || fn.Body == nil {
		return false
	}
	extra := v.stmts("compile.go")
	if hasIdentExpr(fn.Body, "EnableIfaceFuncval") {
		if hasIdentExpr(fn.Body, "objabi") && hasStringLit(fn.Body, "amd64") {
			return false
		}
		return replaceStmtWithIdent(fn.Body, "EnableIfaceFuncval", extra)
	}
	for i, stmt := range fn.Body.List {
		es, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := es.X.(*ast.CallExpr)
		if !ok || callName(call.Fun) != "CountFlags" {
			continue
		}
		fn.Body.List = insertStmts(fn.Body.List, i+1, extra...)
		return true
	}
	return false
}

func replaceStmtWithIdent(body *ast.BlockStmt, ident string, extra []ast.Stmt) bool {
	for i, stmt := range body.List {
		if hasIdentExpr(stmt, ident) {
			out := make([]ast.Stmt, 0, len(body.List)-1+len(extra))
			out = append(out, body.List[:i]...)
			out = append(out, extra...)
			out = append(out, body.List[i+1:]...)
			body.List = out
			return true
		}
	}
	return false
}

func hasStringLit(n ast.Node, s string) bool {
	found := false
	ast.Inspect(n, func(x ast.Node) bool {
		lit, ok := x.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		v, err := strconv.Unquote(lit.Value)
		if err == nil && v == s {
			found = true
			return false
		}
		return true
	})
	return found
}

func hasIdentExpr(n ast.Node, name string) bool {
	found := false
	ast.Inspect(n, func(x ast.Node) bool {
		id, ok := x.(*ast.Ident)
		if ok && id.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}

func funcDecl(f *ast.File, name string) *ast.FuncDecl {
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if ok && fn.Name.Name == name {
			return fn
		}
	}
	return nil
}

func patchAsmFlags(_ *token.FileSet, f *ast.File, v versionData) (bool, error) {
	changed := false
	if hasImport(f, "cmd/internal/obj/wasm") {
		if removeImport(f, "cmd/internal/obj/wasm") {
			changed = true
		}
	}
	if !hasImport(f, "internal/buildcfg") {
		addImportAfter(f, "fmt", "internal/buildcfg")
		changed = true
	}
	if !hasImport(f, "log") {
		addImportAfter(f, "internal/buildcfg", "log")
		changed = true
	}
	if !hasIdent(f, "IfaceFuncval") {
		if !addVarBoolFlag(f, "IfaceFuncval", "ifacefuncval", "wasm: treat tagged itab.Fun as MakeFunc funcval (goplus.ifacefuncval)") {
			return false, fmt.Errorf("Std flag var not found")
		}
		changed = true
	}
	fn := funcDecl(f, "Parse")
	if fn == nil || fn.Body == nil {
		return false, fmt.Errorf("Parse not found")
	}
	extra := v.stmts("asm.go")
	if hasIdentExpr(fn.Body, "EnableIfaceFuncval") {
		if !(hasIdentExpr(fn.Body, "objabi") && hasStringLit(fn.Body, "amd64")) {
			if replaceStmtWithIdent(fn.Body, "EnableIfaceFuncval", extra) {
				changed = true
			}
		}
	} else {
		idx := 0
		for i, stmt := range fn.Body.List {
			ifs, ok := stmt.(*ast.IfStmt)
			if !ok {
				continue
			}
			if hasIdentExpr(ifs.Cond, "NArg") {
				idx = i + 1
				break
			}
		}
		fn.Body.List = insertStmts(fn.Body.List, idx, extra...)
		changed = true
	}
	return changed, nil
}

func addVarBoolFlag(f *ast.File, name, flagName, help string) bool {
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for i, s := range gen.Specs {
			vs := s.(*ast.ValueSpec)
			if len(vs.Names) != 1 || vs.Names[0].Name != "Std" {
				continue
			}
			spec := &ast.ValueSpec{
				Names: []*ast.Ident{ast.NewIdent(name)},
				Values: []ast.Expr{
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{X: ast.NewIdent("flag"), Sel: ast.NewIdent("Bool")},
						Args: []ast.Expr{
							&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(flagName)},
							ast.NewIdent("false"),
							&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(help)},
						},
					},
				},
			}
			specs := make([]ast.Spec, 0, len(gen.Specs)+1)
			specs = append(specs, gen.Specs[:i+1]...)
			specs = append(specs, spec)
			specs = append(specs, gen.Specs[i+1:]...)
			gen.Specs = specs
			return true
		}
	}
	return false
}

func patchGoGc(_ *token.FileSet, f *ast.File, v versionData) (bool, error) {
	decls := v.decls("gc.go")
	if len(decls) == 0 {
		return false, fmt.Errorf("gc.go snippet has no decls")
	}
	if fn := funcDecl(f, "ifaceFuncvalEnabled"); fn != nil {
		if hasStringLit(fn, "amd64") {
			return false, nil
		}
		replaceFuncDecl(f, "ifaceFuncvalEnabled", decls[0])
		return true, nil
	}
	if funcDecl(f, "wasmIfaceFuncval") != nil {
		replaceFuncDecl(f, "wasmIfaceFuncval", decls[0])
		return true, nil
	}
	f.Decls = append(f.Decls, decls...)
	return true, nil
}

func renameIdent(n ast.Node, old, new string) {
	ast.Inspect(n, func(x ast.Node) bool {
		id, ok := x.(*ast.Ident)
		if ok && id.Name == old {
			id.Name = new
		}
		return true
	})
}

func replaceFuncDecl(f *ast.File, name string, d ast.Decl) {
	for i, old := range f.Decls {
		fn, ok := old.(*ast.FuncDecl)
		if ok && fn.Name.Name == name {
			f.Decls[i] = d
			return
		}
	}
}

func patchGoInit(_ *token.FileSet, f *ast.File, v versionData) (bool, error) {
	fn := funcDecl(f, "BuildInit")
	if fn == nil || fn.Body == nil {
		return false, fmt.Errorf("BuildInit not found")
	}
	if hasIdentExpr(fn.Body, "ifaceFuncvalEnabled") {
		return false, nil
	}
	if hasIdentExpr(fn.Body, "wasmIfaceFuncval") {
		renameIdent(fn.Body, "wasmIfaceFuncval", "ifaceFuncvalEnabled")
		return true, nil
	}
	extra := v.stmts("gc_flags.go")
	for i, stmt := range fn.Body.List {
		es, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := es.X.(*ast.CallExpr)
		if !ok || callName(call.Fun) != "buildModeInit" {
			continue
		}
		fn.Body.List = insertStmts(fn.Body.List, i+1, extra...)
		return true, nil
	}
	return false, fmt.Errorf("buildModeInit() not found in BuildInit")
}

func patchArm64SSA(_ *token.FileSet, f *ast.File, v versionData) (bool, error) {
	changed := false
	if hasImport(f, "cmd/internal/obj/wasm") {
		if removeImport(f, "cmd/internal/obj/wasm") {
			changed = true
		}
	}
	if !hasImport(f, "cmd/internal/objabi") {
		addImportAfter(f, "cmd/internal/obj/arm64", "cmd/internal/objabi")
		changed = true
	}
	addedHelper := false
	if funcDecl(f, "ssaGenIfaceFuncvalCall") == nil {
		f.Decls = append(f.Decls, v.decls("arm64_ssa.go")...)
		addedHelper = true
		changed = true
	}
	if splitARM64CallCase(f, "OpARM64CALLinter", "OpARM64CALLstatic", v.stmts("arm64_callinter.go")) {
		changed = true
	}
	if splitARM64CallCase(f, "OpARM64CALLtailinter", "OpARM64CALLtail", v.stmts("arm64_calltailinter.go")) {
		changed = true
	}
	if addedHelper {
		rewired := 0
		ast.Inspect(f, func(n ast.Node) bool {
			cc, ok := n.(*ast.CaseClause)
			if !ok {
				return true
			}
			if selectorInList(cc.List, "OpARM64CALLinter") && hasIdentExpr(cc, "ssaGenIfaceFuncvalCall") {
				rewired++
			}
			if selectorInList(cc.List, "OpARM64CALLtailinter") && hasIdentExpr(cc, "ssaGenIfaceFuncvalCall") {
				rewired++
			}
			return true
		})
		if rewired < 2 {
			return false, fmt.Errorf("arm64 CALL sites not rewired (found %d, want 2); ssaGenValue switch layout may have changed", rewired)
		}
	}
	return changed, nil
}

func splitARM64CallCase(f *ast.File, remove, keep string, body []ast.Stmt) bool {
	var did bool
	ast.Inspect(f, func(n ast.Node) bool {
		sw, ok := n.(*ast.SwitchStmt)
		if !ok || sw.Body == nil {
			return true
		}
		for i, stmt := range sw.Body.List {
			cc, ok := stmt.(*ast.CaseClause)
			if !ok {
				continue
			}
			if selectorInList(cc.List, remove) && selectorInList(cc.List, keep) {
				cc.List = filterSelector(cc.List, remove)
				neu := &ast.CaseClause{
					List: []ast.Expr{&ast.SelectorExpr{X: ast.NewIdent("ssa"), Sel: ast.NewIdent(remove)}},
					Body: body,
				}
				sw.Body.List = append(sw.Body.List[:i+1], append([]ast.Stmt{neu}, sw.Body.List[i+1:]...)...)
				did = true
				return false
			}
		}
		return true
	})
	return did
}

func selectorInList(list []ast.Expr, name string) bool {
	for _, e := range list {
		sel, ok := e.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == name {
			return true
		}
	}
	return false
}

func filterSelector(list []ast.Expr, name string) []ast.Expr {
	out := make([]ast.Expr, 0, len(list))
	for _, e := range list {
		sel, ok := e.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == name {
			continue
		}
		out = append(out, e)
	}
	return out
}

func patchAsmReplace(path string, old, new []byte, check bool) (bool, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if bytes.Contains(src, new) {
		return false, nil
	}
	if !bytes.Contains(src, old) {
		return false, fmt.Errorf("CALLFN indirect call sequence not found")
	}
	if check {
		return true, nil
	}
	out := bytes.Replace(src, old, new, 1)
	return true, os.WriteFile(path, out, 0644)
}

func patchAmd64SSA(_ *token.FileSet, f *ast.File, v versionData) (bool, error) {
	changed := false
	if hasImport(f, "cmd/internal/obj/wasm") {
		if removeImport(f, "cmd/internal/obj/wasm") {
			changed = true
		}
	}
	if !hasImport(f, "cmd/internal/objabi") {
		addImportAfter(f, "cmd/internal/obj/x86", "cmd/internal/objabi")
		changed = true
	}
	addedHelper := false
	if funcDecl(f, "ssaGenIfaceFuncvalCall") == nil {
		f.Decls = append(f.Decls, v.decls("amd64_ssa.go")...)
		addedHelper = true
		changed = true
	}
	if splitARM64CallCase(f, "OpAMD64CALLinter", "OpAMD64CALLclosure", v.stmts("amd64_callinter.go")) {
		changed = true
	}
	if splitARM64CallCase(f, "OpAMD64CALLtailinter", "OpAMD64CALLtail", v.stmts("amd64_calltailinter.go")) {
		changed = true
	}
	if addedHelper {
		rewired := 0
		ast.Inspect(f, func(n ast.Node) bool {
			cc, ok := n.(*ast.CaseClause)
			if !ok {
				return true
			}
			if selectorInList(cc.List, "OpAMD64CALLinter") && hasIdentExpr(cc, "ssaGenIfaceFuncvalCall") {
				rewired++
			}
			if selectorInList(cc.List, "OpAMD64CALLtailinter") && hasIdentExpr(cc, "ssaGenIfaceFuncvalTailCall") {
				rewired++
			}
			return true
		})
		if rewired < 2 {
			return false, fmt.Errorf("amd64 CALL sites not rewired (found %d, want 2); ssaGenValue switch layout may have changed", rewired)
		}
	}
	return changed, nil
}
