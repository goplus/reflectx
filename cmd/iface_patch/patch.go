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
	"strings"
)

type patchFunc func(*token.FileSet, *ast.File, versionData) (bool, error)

func applyAll(root string, v versionData, check bool) (int, error) {
	operations := patchOperations(v)
	if check {
		return applyPatchOperations(root, operations, true, nil)
	}

	backups, err := snapshotPatchFiles(root, operations)
	if err != nil {
		return 0, fmt.Errorf("prepare rollback: %w", err)
	}
	var changedFiles []string
	n, err := applyPatchOperations(root, operations, false, func(path string) {
		changedFiles = append(changedFiles, path)
	})
	if err == nil {
		var remaining int
		remaining, err = applyPatchOperations(root, operations, true, nil)
		if err == nil && remaining != 0 {
			err = fmt.Errorf("post-patch validation failed: %d changes still required", remaining)
		}
	}
	if err == nil {
		for _, path := range changedFiles {
			fmt.Println(path)
		}
		return n, nil
	}
	if rollbackErr := restorePatchFiles(backups); rollbackErr != nil {
		return n, fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
	}
	return n, fmt.Errorf("%w (all changes rolled back)", err)
}

type fileBackup struct {
	path   string
	data   []byte
	mode   os.FileMode
	exists bool
}

func snapshotPatchFiles(root string, operations []patchOperation) ([]fileBackup, error) {
	backups := make([]fileBackup, 0, len(operations))
	seen := make(map[string]bool)
	for _, operation := range operations {
		rel := operation.rel
		if seen[rel] {
			continue
		}
		seen[rel] = true
		path := filepath.Join(root, rel)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			backups = append(backups, fileBackup{path: path})
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		backups = append(backups, fileBackup{path: path, data: data, mode: info.Mode(), exists: true})
	}
	return backups, nil
}

func restorePatchFiles(backups []fileBackup) error {
	var errs []string
	for _, backup := range backups {
		var err error
		if backup.exists {
			err = os.WriteFile(backup.path, backup.data, backup.mode.Perm())
			if err == nil {
				err = os.Chmod(backup.path, backup.mode.Perm())
			}
		} else if removeErr := os.Remove(backup.path); removeErr != nil && !os.IsNotExist(removeErr) {
			err = removeErr
		}
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", backup.path, err))
		}
	}
	if len(errs) != 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

type patchOperation struct {
	rel   string
	apply func(path string, check bool) (bool, error)
}

func patchOperations(v versionData) []patchOperation {
	type textPatch struct {
		rel   string
		pairs [][2]string
	}
	sourcePatches := []struct {
		rel, data string
	}{
		{"src/cmd/internal/objabi/ifacefuncval.go", "objabi_ifacefuncval.go"},
		{"src/runtime/iface_funcval.go", "iface_funcval.go"},
	}
	astPatches := []struct {
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
		{"src/cmd/compile/internal/x86/ssa.go", patch386SSA},
	}
	textPatches := []textPatch{
		{"src/runtime/iface.go", [][2]string{{"iface_fun0_old.txt", "iface_fun0_new.txt"}}},
		{"src/runtime/iface.go", [][2]string{{"iface_fun0store_old.txt", "iface_fun0store_new.txt"}}},
		{"src/runtime/iface.go", [][2]string{
			{"iface_ifn_old.txt", "iface_ifn_new.txt"},
			{"iface_ifn_ptr_old.txt", "iface_ifn_ptr_new.txt"},
		}},
		{"src/reflect/value.go", [][2]string{{"reflect_method_old.txt", "reflect_method_new.txt"}}},
	}
	for _, arch := range []string{"arm64", "amd64", "386"} {
		textPatches = append(textPatches, textPatch{
			"src/runtime/asm_" + arch + ".s",
			[][2]string{{arch + "_callfn_old.s", arch + "_callfn_new.s"}},
		})
	}

	operations := make([]patchOperation, 0, len(sourcePatches)+len(astPatches)+len(textPatches))
	for _, patch := range sourcePatches {
		operations = append(operations, goSourceOperation(patch.rel, v.goSrc(patch.data)))
	}
	for _, patch := range astPatches {
		operations = append(operations, astOperation(patch.rel, patch.fn, v))
	}
	for _, patch := range textPatches {
		pairs := make([][2][]byte, len(patch.pairs))
		for i, pair := range patch.pairs {
			pairs[i] = [2][]byte{v.bytes(pair[0]), v.bytes(pair[1])}
		}
		operations = append(operations, textOperation(patch.rel, pairs))
	}
	return operations
}

func goSourceOperation(rel string, src []byte) patchOperation {
	return patchOperation{rel: rel, apply: func(path string, check bool) (bool, error) {
		return writeGoSrc(path, src, check)
	}}
}

func astOperation(rel string, fn patchFunc, v versionData) patchOperation {
	return patchOperation{rel: rel, apply: func(path string, check bool) (bool, error) {
		return patchFile(path, fn, v, check)
	}}
}

func textOperation(rel string, pairs [][2][]byte) patchOperation {
	return patchOperation{rel: rel, apply: func(path string, check bool) (bool, error) {
		return patchAsmReplaceAny(path, pairs, check)
	}}
}

func applyPatchOperations(root string, operations []patchOperation, check bool, report func(string)) (int, error) {
	n := 0
	for _, operation := range operations {
		changed, err := operation.apply(filepath.Join(root, operation.rel), check)
		if err != nil {
			return n, fmt.Errorf("%s: %w", operation.rel, err)
		}
		if changed {
			n++
			if report != nil {
				report(operation.rel)
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

func hasSelectorExpr(n ast.Node, pkg, name string) bool {
	found := false
	ast.Inspect(n, func(x ast.Node) bool {
		sel, ok := x.(*ast.SelectorExpr)
		if ok && isSelector(sel, pkg, name) {
			found = true
			return false
		}
		return true
	})
	return found
}

const ifaceFuncvalHelp = "treat tagged itab.Fun as MakeFunc funcval (goplus.ifacefuncval; wasm/arm64/amd64/386)"

func ifaceFuncvalBlockCurrent(stmt ast.Stmt) bool {
	return hasSelectorExpr(stmt, "objabi", "EnableIfaceFuncval") && hasStringLit(stmt, "386")
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
	callSites := 0
	ast.Inspect(f, func(n ast.Node) bool {
		cc, ok := n.(*ast.CaseClause)
		if !ok || !isTYPE_NONE(cc) || len(cc.Body) == 0 {
			return true
		}
		if isUnwrapAssign(cc.Body[0]) {
			callSites++
			return true
		}
		if !isAppendpAI64Const(cc.Body[0]) {
			return true
		}
		unwrap := v.stmts("unwrap.go")[0]
		cc.Body = insertStmts(cc.Body, 0, unwrap)
		callSites++
		changed = true
		return true
	})
	if callSites != 2 {
		return false, fmt.Errorf("wasm indirect CALL sites: found %d, want 2", callSites)
	}
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
			Tag:   &ast.BasicLit{Kind: token.STRING, Value: "`help:\"" + ifaceFuncvalHelp + "\"`"},
		}
		st.Fields.List = append(st.Fields.List[:idx], append([]*ast.Field{field}, st.Fields.List[idx:]...)...)
		changed = true
	} else if updateIfaceFuncvalFieldHelp(st) {
		changed = true
	}
	if insertAfterCountFlags(f, v) {
		changed = true
	} else if fn := funcDecl(f, "ParseFlags"); fn == nil || fn.Body == nil || stmtWithIdent(fn.Body, "EnableIfaceFuncval") == nil {
		return false, fmt.Errorf("CountFlags call not found in ParseFlags")
	}
	return changed, nil
}

func updateIfaceFuncvalFieldHelp(st *ast.StructType) bool {
	want := "`help:\"" + ifaceFuncvalHelp + "\"`"
	for _, field := range st.Fields.List {
		if len(field.Names) != 1 || field.Names[0].Name != "IfaceFuncval" || field.Tag == nil {
			continue
		}
		if field.Tag.Value == want {
			return false
		}
		field.Tag = &ast.BasicLit{Kind: token.STRING, Value: want}
		return true
	}
	return false
}

func insertAfterCountFlags(f *ast.File, v versionData) bool {
	fn := funcDecl(f, "ParseFlags")
	if fn == nil || fn.Body == nil {
		return false
	}
	extra := v.stmts("compile.go")
	for _, stmt := range fn.Body.List {
		if !hasIdentExpr(stmt, "EnableIfaceFuncval") {
			continue
		}
		if ifaceFuncvalBlockCurrent(stmt) {
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
		if !addVarBoolFlag(f, "IfaceFuncval", "ifacefuncval", ifaceFuncvalHelp) {
			return false, fmt.Errorf("Std/Spectre flag var not found")
		}
		changed = true
	} else if updateVarBoolFlagHelp(f, "IfaceFuncval", ifaceFuncvalHelp) {
		changed = true
	}
	fn := funcDecl(f, "Parse")
	if fn == nil || fn.Body == nil {
		return false, fmt.Errorf("Parse not found")
	}
	extra := v.stmts("asm.go")
	if stmt := stmtWithIdent(fn.Body, "EnableIfaceFuncval"); stmt != nil {
		if !ifaceFuncvalBlockCurrent(stmt) {
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

func stmtWithIdent(body *ast.BlockStmt, ident string) ast.Stmt {
	for _, stmt := range body.List {
		if hasIdentExpr(stmt, ident) {
			return stmt
		}
	}
	return nil
}

func updateVarBoolFlagHelp(f *ast.File, name, help string) bool {
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, s := range gen.Specs {
			vs, ok := s.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || vs.Names[0].Name != name || len(vs.Values) != 1 {
				continue
			}
			call, ok := vs.Values[0].(*ast.CallExpr)
			if !ok || len(call.Args) < 3 {
				continue
			}
			lit, ok := call.Args[2].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			cur, err := strconv.Unquote(lit.Value)
			if err != nil || cur == help {
				return false
			}
			call.Args[2] = &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(help)}
			return true
		}
	}
	return false
}

func addVarBoolFlag(f *ast.File, name, flagName, help string) bool {
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		idx := -1
		for _, after := range []string{"Std", "Spectre"} {
			for i, s := range gen.Specs {
				vs, ok := s.(*ast.ValueSpec)
				if !ok || len(vs.Names) != 1 || vs.Names[0].Name != after {
					continue
				}
				idx = i
				break
			}
			if idx >= 0 {
				break
			}
		}
		if idx < 0 {
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
		specs = append(specs, gen.Specs[:idx+1]...)
		specs = append(specs, spec)
		specs = append(specs, gen.Specs[idx+1:]...)
		gen.Specs = specs
		return true
	}
	return false
}

func patchGoGc(_ *token.FileSet, f *ast.File, v versionData) (bool, error) {
	changed := false
	decls := v.decls("gc.go")
	if len(decls) == 0 {
		return false, fmt.Errorf("gc.go snippet has no decls")
	}
	if fn := funcDecl(f, "ifaceFuncvalEnabled"); fn != nil {
		if !hasStringLit(fn, "386") {
			replaceFuncDecl(f, "ifaceFuncvalEnabled", decls[0])
			changed = true
		}
	} else if funcDecl(f, "wasmIfaceFuncval") != nil {
		replaceFuncDecl(f, "wasmIfaceFuncval", decls[0])
		changed = true
	} else {
		f.Decls = append(f.Decls, decls...)
		changed = true
	}
	return changed, nil
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
	extra := v.stmts("gc_flags.go")
	if hasIdentExpr(fn.Body, "ifaceFuncvalEnabled") {
		if hasIdentExpr(fn.Body, "forcedAsmflags") {
			return false, nil
		}
		if !replaceStmtWithIdent(fn.Body, "ifaceFuncvalEnabled", extra) {
			return false, fmt.Errorf("ifaceFuncvalEnabled block not replaced")
		}
		return true, nil
	}
	if hasIdentExpr(fn.Body, "wasmIfaceFuncval") {
		if !replaceStmtWithIdent(fn.Body, "wasmIfaceFuncval", extra) {
			return false, fmt.Errorf("wasmIfaceFuncval block not replaced")
		}
		return true, nil
	}
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
	if !arm64HelperCurrent(f) {
		removeFuncDecls(f, "ssaGenIfaceFuncvalCall", "ssaGenIfaceFuncvalTailCall", "ssaGenIfaceFuncvalCallReg", "callOrTail")
		f.Decls = append(f.Decls, v.decls("arm64_ssa.go")...)
		changed = true
	}
	if splitCallCase(f, "OpARM64CALLinter", "OpARM64CALLstatic", v.stmts("arm64_callinter.go")) {
		changed = true
	}
	want := 1
	if hasIdentExpr(f, "OpARM64CALLtailinter") {
		if splitCallCase(f, "OpARM64CALLtailinter", "OpARM64CALLtail", v.stmts("arm64_calltailinter.go")) {
			changed = true
		}
		want = 2
	}
	if syncCallCaseBody(f, "OpARM64CALLinter", v.stmts("arm64_callinter.go")) {
		changed = true
	}
	if want == 2 && syncCallCaseBody(f, "OpARM64CALLtailinter", v.stmts("arm64_calltailinter.go")) {
		changed = true
	}
	rewired := countCallRewired(f, [2]string{"OpARM64CALLinter", "ssaGenIfaceFuncvalCall"}, [2]string{"OpARM64CALLtailinter", "ssaGenIfaceFuncvalTailCall"})
	if rewired < want {
		return false, fmt.Errorf("arm64 CALL sites not rewired (found %d, want %d); ssaGenValue switch layout may have changed", rewired, want)
	}
	return changed, nil
}

func arm64HelperCurrent(f *ast.File) bool {
	return funcDecl(f, "ssaGenIfaceFuncvalCallReg") != nil
}

func removeFuncDecls(f *ast.File, names ...string) {
	drop := make(map[string]bool, len(names))
	for _, name := range names {
		drop[name] = true
	}
	out := f.Decls[:0]
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if ok && drop[fn.Name.Name] {
			continue
		}
		out = append(out, d)
	}
	f.Decls = out
}

func syncCallCaseBody(f *ast.File, op string, body []ast.Stmt) bool {
	helper := firstCallName(body)
	did := false
	ast.Inspect(f, func(n ast.Node) bool {
		cc, ok := n.(*ast.CaseClause)
		if !ok || len(cc.List) != 1 || !selectorInList(cc.List, op) {
			return true
		}
		if helper != "" && hasIdentExpr(cc, helper) && !hasIdentExpr(cc, "Call") && !hasIdentExpr(cc, "TailCall") {
			return true
		}
		cc.Body = body
		did = true
		return true
	})
	return did
}

func firstCallName(body []ast.Stmt) string {
	if len(body) == 0 {
		return ""
	}
	es, ok := body[0].(*ast.ExprStmt)
	if !ok {
		return ""
	}
	call, ok := es.X.(*ast.CallExpr)
	if !ok {
		return ""
	}
	return callName(call.Fun)
}

func splitCallCase(f *ast.File, remove, keep string, body []ast.Stmt) bool {
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
	oldCount := bytes.Count(src, old)
	newCount := bytes.Count(src, new)
	if oldCount == 0 && newCount == 1 {
		return false, nil
	}
	if oldCount != 1 || newCount != 0 {
		return false, fmt.Errorf("ambiguous patch state: old sequence occurs %d times, new sequence occurs %d times", oldCount, newCount)
	}
	if check {
		return true, nil
	}
	out := bytes.Replace(src, old, new, 1)
	return true, os.WriteFile(path, out, 0644)
}

func patchAsmReplaceAny(path string, pairs [][2][]byte, check bool) (bool, error) {
	var last error
	for _, pair := range pairs {
		changed, err := patchAsmReplace(path, pair[0], pair[1], check)
		if err == nil {
			return changed, nil
		}
		last = err
	}
	if last == nil {
		return false, fmt.Errorf("patch sequence not found")
	}
	return false, last
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
	if funcDecl(f, "ssaGenIfaceFuncvalCall") == nil {
		f.Decls = append(f.Decls, v.decls("amd64_ssa.go")...)
		changed = true
	}
	if splitCallCase(f, "OpAMD64CALLinter", "OpAMD64CALLclosure", v.stmts("amd64_callinter.go")) {
		changed = true
	}
	want := 1
	if hasIdentExpr(f, "OpAMD64CALLtailinter") {
		if splitCallCase(f, "OpAMD64CALLtailinter", "OpAMD64CALLtail", v.stmts("amd64_calltailinter.go")) {
			changed = true
		}
		want = 2
	}
	rewired := countCallRewired(f, [2]string{"OpAMD64CALLinter", "ssaGenIfaceFuncvalCall"}, [2]string{"OpAMD64CALLtailinter", "ssaGenIfaceFuncvalTailCall"})
	if rewired < want {
		return false, fmt.Errorf("amd64 CALL sites not rewired (found %d, want %d); ssaGenValue switch layout may have changed", rewired, want)
	}
	return changed, nil
}

func patch386SSA(_ *token.FileSet, f *ast.File, v versionData) (bool, error) {
	changed := false
	if !hasImport(f, "cmd/internal/objabi") {
		addImportAfter(f, "cmd/internal/obj/x86", "cmd/internal/objabi")
		changed = true
	}
	if funcDecl(f, "ssaGenIfaceFuncvalCall") == nil {
		f.Decls = append(f.Decls, v.decls("x86_ssa.go")...)
		changed = true
	}
	if splitCallCase(f, "Op386CALLinter", "Op386CALLstatic", v.stmts("x86_callinter.go")) {
		changed = true
	}
	want := 1
	if hasIdentExpr(f, "Op386CALLtailinter") {
		if splitCallCase(f, "Op386CALLtailinter", "Op386CALLtail", v.stmts("x86_calltailinter.go")) {
			changed = true
		}
		want = 2
	}
	rewired := countCallRewired(f, [2]string{"Op386CALLinter", "ssaGenIfaceFuncvalCall"}, [2]string{"Op386CALLtailinter", "ssaGenIfaceFuncvalTailCall"})
	if rewired < want {
		return false, fmt.Errorf("386 CALL sites not rewired (found %d, want %d); ssaGenValue switch layout may have changed", rewired, want)
	}
	return changed, nil
}

func countCallRewired(f *ast.File, pairs ...[2]string) int {
	rewired := 0
	ast.Inspect(f, func(n ast.Node) bool {
		cc, ok := n.(*ast.CaseClause)
		if !ok {
			return true
		}
		for _, p := range pairs {
			if selectorInList(cc.List, p[0]) && hasIdentExpr(cc, p[1]) {
				rewired++
			}
		}
		return true
	})
	return rewired
}
