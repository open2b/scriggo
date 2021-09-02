// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/native"
)

// parseNumericCost parses an expression representing an untyped number
// constant and return the represented constant, it type.
func parseNumericConst(s string) (constant, reflect.Type, error) {
	if s[0] == '\'' {
		r, err := parseBasicLiteral(ast.RuneLiteral, s)
		if err != nil {
			return nil, nil, err
		}
		return r, runeType, nil
	}
	r, _, tail, err := strconv.UnquoteChar(s[1:], '\'')
	if err == nil && tail == "'" {
		return int64Const(r), int32Type, nil
	}
	if last := len(s) - 1; s[last] == 'i' {
		p := strings.LastIndexAny(s, "+-")
		if p == -1 {
			p = 0
		}
		im, err := parseBasicLiteral(ast.FloatLiteral, s[p:last])
		if err != nil {
			return nil, nil, err
		}
		var re constant
		if p > 0 {
			re, err = parseBasicLiteral(ast.FloatLiteral, s[0:p])
			if err != nil {
				return nil, nil, err
			}
		} else {
			re = newFloat64Const(0)
		}
		return newComplexConst(re, im), complex128Type, nil
	}
	if strings.ContainsAny(s, "/.") {
		n, err := parseBasicLiteral(ast.FloatLiteral, s)
		if err != nil {
			return nil, nil, err
		}
		return n, float64Type, nil
	}
	n, err := parseBasicLiteral(ast.IntLiteral, s)
	if err != nil {
		return nil, nil, err
	}
	return n, intType, nil
}

// toTypeCheckerScope generates a type checker scope given a native package.
// depth must be 0 unless toTypeCheckerScope is called recursively.
func toTypeCheckerScope(pkg native.ImportablePackage, mod checkingMod, global bool, depth int) map[string]scopeName {
	scope := map[string]scopeName{}
	_ = pkg.LookupFunc(func(ident string, decl native.Declaration) error {
		ti := &typeInfo{NativePackageName: pkg.PackageName()}
		if global {
			ti.Properties = propertyGlobal
		}
		switch v := pkg.Lookup(ident).(type) {
		default:
			rv := reflect.ValueOf(v)
			switch rv.Kind() {
			case reflect.Ptr:
				// Import a variable.
				ti.Type = rv.Type().Elem()
				elem := rv.Elem()
				// Note that 'elem' may be an invalid reflect.Value. That
				// indicates that such variable has not been initialized.
				ti.value = &elem
				ti.Properties |= propertyAddressable | propertyIsNative | propertyHasValue
			case reflect.Func:
				// Import a function.
				ti.Type = removeEnvArg(rv.Type(), false)
				ti.value = rv
				ti.Properties |= propertyIsNative | propertyHasValue
			default:
				// Import a typed constant.
				ti.Constant = convertToConstant(rv)
				if ti.Constant == nil {
					panic(fmt.Errorf("scriggo: importing package %s, declaration %q is not valid", pkg.PackageName(), ident))
				}
				ti.Type = rv.Type()
			}
		case native.ImportablePackage:
			// Import an auto-imported package. This is supported in scripts and templates only.
			if mod == programMod {
				panic(fmt.Errorf("scriggo: auto-imported packages are supported only for scripts and templates"))
			}
			if depth > 0 {
				panic(fmt.Errorf("scriggo: cannot have an auto-imported package inside another auto-imported package"))
			}
			pkg := &packageInfo{
				Name:         v.PackageName(),
				Declarations: map[string]*typeInfo{},
			}
			for n, d := range toTypeCheckerScope(v, mod, global, depth+1) {
				pkg.Declarations[n] = d.ti
			}
			ti.value = pkg
			ti.Properties |= propertyIsPackage | propertyHasValue
			ti.NativePackageName = ""
		case reflect.Type:
			// Import a type.
			ti.Type = v
			ti.Properties |= propertyIsType | propertyIsNative
		case native.UntypedBooleanConst:
			// Import an untyped boolean constant.
			ti.Type = boolType
			ti.Properties |= propertyUntyped
			ti.Constant = boolConst(v)
		case native.UntypedStringConst:
			// Import an untyped string constant.
			ti.Type = stringType
			ti.Properties |= propertyUntyped
			ti.Constant = stringConst(v)
		case native.UntypedNumericConst:
			// Import an untyped numeric constant.
			var err error
			ti.Constant, ti.Type, err = parseNumericConst(string(v))
			if err != nil {
				panic(fmt.Errorf("scriggo: invalid untyped constant %q for %s.%s", v, pkg.PackageName(), ident))
			}
			ti.Properties |= propertyUntyped
		}
		scope[ident] = scopeName{ti: ti}
		return nil
	})
	return scope
}

type packageInfo struct {
	Name         string
	Declarations map[string]*typeInfo
	IndirectVars map[*ast.Identifier]bool
	TypeInfos    map[ast.Node]*typeInfo
}

func depsOf(name string, deps packageDeclsDeps) []*ast.Identifier {
	for g, d := range deps {
		if g.Name == name {
			return d
		}
	}
	return nil
}

func checkDepsPath(path []*ast.Identifier, deps packageDeclsDeps) []*ast.Identifier {
	last := path[len(path)-1]
	for _, dep := range depsOf(last.Name, deps) {
		for _, p := range path {
			if p.Name == dep.Name {
				return append(path, dep)
			}
		}
		loopPath := checkDepsPath(append(path, dep), deps)
		if loopPath != nil {
			return loopPath
		}
	}
	return nil
}

// An initLoopError is a error representing an initialization loop.
type initLoopError struct {
	node ast.Node
	msg  string
}

func (loopErr initLoopError) Error() string {
	return loopErr.node.Pos().String() + ": " + loopErr.msg
}

func detectConstantsLoop(consts []*ast.Const, deps packageDeclsDeps) error {
	for _, c := range consts {
		path := []*ast.Identifier{c.Lhs[0]}
		loopPath := checkDepsPath(path, deps)
		if loopPath != nil {
			msg := "constant definition loop\n"
			for i := 0; i < len(loopPath)-1; i++ {
				msg += "\t" + loopPath[i].Pos().String() + ": "
				msg += loopPath[i].String() + " uses " + loopPath[i+1].String()
				msg += "\n"
			}
			return initLoopError{node: c, msg: msg}
		}
	}
	return nil
}

func detectVarsLoop(vars []*ast.Var, deps packageDeclsDeps) error {
	for _, v := range vars {
		for _, left := range v.Lhs {
			path := []*ast.Identifier{left}
			loopPath := checkDepsPath(path, deps)
			if loopPath != nil {
				msg := "typechecking loop involving " + v.String() + "\n"
				for _, p := range loopPath {
					msg += "\t" + p.Pos().String() + ": " + p.String() + "\n"
				}
				return initLoopError{node: v, msg: msg}
			}
		}
	}
	return nil
}

func detectTypeLoop(types []*ast.TypeDeclaration, deps packageDeclsDeps) error {
	for _, t := range types {
		// TODO: currently the initialization loop check for type definitions
		// is the same as for type alias declarations. Is this correct?
		path := []*ast.Identifier{t.Ident}
		loopPath := checkDepsPath(path, deps)
		if loopPath == nil {
			continue // Ok, no loops.
		}
		msg := "invalid recursive type"
		if t.IsAliasDeclaration {
			msg += " alias"
		}
		msg += " " + t.Ident.String() + "\n"
		for _, p := range loopPath {
			msg += "\t" + p.Pos().String() + ": " + p.String() + "\n"
		}
		return initLoopError{node: t, msg: msg}
	}
	return nil
}

func sortDeclarations(pkg *ast.Package) error {
	var extends *ast.Extends
	types := []*ast.TypeDeclaration{}
	consts := []*ast.Const{}
	vars := []*ast.Var{}
	imports := []*ast.Import{}
	funcs := []*ast.Func{}

	// Fragments global declarations.
	for _, decl := range pkg.Declarations {
		switch decl := decl.(type) {
		case *ast.Extends:
			extends = decl
		case *ast.Import:
			imports = append(imports, decl)
		case *ast.Func:
			funcs = append(funcs, decl)
		case *ast.Const:
			if len(decl.Rhs) == 0 {
				for i := range decl.Lhs {
					consts = append(consts, ast.NewConst(decl.Pos(), decl.Lhs[i:i+1], decl.Type, nil, decl.Index))
				}
			} else {
				for i := range decl.Lhs {
					consts = append(consts, ast.NewConst(decl.Pos(), decl.Lhs[i:i+1], decl.Type, decl.Rhs[i:i+1], decl.Index))
				}
			}
		case *ast.TypeDeclaration:
			types = append(types, decl)
		case *ast.Var:
			if len(decl.Lhs) == len(decl.Rhs) {
				for i := range decl.Lhs {
					vars = append(vars, ast.NewVar(decl.Pos(), decl.Lhs[i:i+1], decl.Type, decl.Rhs[i:i+1]))
				}
			} else {
				vars = append(vars, decl)
			}
		}
	}

	// Removes from deps all non-global dependencies.
	deps := analyzeTree(pkg)
	for decl, ds := range deps {
		newDs := []*ast.Identifier{}
		for _, d := range ds {
			// Is d a global identifier?
			for _, c := range consts {
				if c.Lhs[0].Name == d.Name {
					newDs = append(newDs, d)
				}
			}
			for _, t := range types {
				if t.Ident.Name == d.Name {
					newDs = append(newDs, d)
				}
			}
			for _, f := range funcs {
				if f.Ident.Name == d.Name {
					newDs = append(newDs, d)
				}
			}
			for _, v := range vars {
				for _, left := range v.Lhs {
					if left.Name == d.Name {
						newDs = append(newDs, d)
					}
				}
			}
		}
		deps[decl] = newDs
	}

	err := detectConstantsLoop(consts, deps)
	if err != nil {
		return err
	}
	err = detectVarsLoop(vars, deps)
	if err != nil {
		return err
	}
	err = detectTypeLoop(types, deps)
	if err != nil {
		return err
	}

	// Sort types declarations.
	sortedTypes := []*ast.TypeDeclaration{}
typesLoop:
	for len(types) > 0 {
		// Searches for next type declaration with resolved deps.
		for i, t := range types {
			depsOk := true
			for _, dep := range deps[t.Ident] {
				found := false
				for _, resolvedT := range sortedTypes {
					if dep.Name == resolvedT.Ident.Name {
						// This dependency has been resolved: move
						// on checking for next one.
						found = true
						break
					}
				}
				if !found {
					// A dependency of t cannot be found.
					depsOk = false
					break
				}
			}
			if depsOk {
				// All dependencies of types are resolved.
				sortedTypes = append(sortedTypes, t)
				types = append(types[:i], types[i+1:]...)
				continue typesLoop
			}
		}
		// All remaining constants are added as "sorted"; a later stage of
		// the type checking will report the error (usually a not defined
		// symbol).
		sortedTypes = append(sortedTypes, types...)
		types = nil
	}

	// Sorts constants.
	sortedConsts := []*ast.Const{}
constsLoop:
	for len(consts) > 0 {
		// Searches for next constant with resolved deps.
		for i, c := range consts {
			depsOk := true
			for _, dep := range deps[c.Lhs[0]] {
				found := false
				for _, resolvedC := range sortedConsts {
					if dep.Name == resolvedC.Lhs[0].Name {
						// This dependency has been resolved: move
						// on checking for next one.
						found = true
						break
					}
				}
				for _, t := range sortedTypes {
					if dep.Name == t.Ident.Name {
						// This dependency has been resolved: move
						// on checking for next one.
						found = true
						break
					}
				}
				if !found {
					// A dependency of c cannot be found.
					depsOk = false
					break
				}
			}
			if depsOk {
				// All dependencies of consts are resolved.
				sortedConsts = append(sortedConsts, c)
				consts = append(consts[:i], consts[i+1:]...)
				continue constsLoop
			}
		}
		if true {
			// All remaining constants are added as "sorted"; a later stage of
			// the type checking will report the error (usually a not defined
			// symbol).
			sortedConsts = append(sortedConsts, consts...)
			consts = nil
		}
	}

	// Sorts variables.
	sortedVars := []*ast.Var{}
varsLoop:
	for len(vars) > 0 {
		// Searches for next variable with resolved deps.
		for i, v := range vars {
			depsOk := true
			for _, dep := range deps[v.Lhs[0]] {
				found := false
			resolvedLoop:
				for _, resolvedV := range sortedVars {
					for _, left := range resolvedV.Lhs {
						if dep.Name == left.Name {
							// This dependency has been resolved: move
							// on checking for next one.
							found = true
							break resolvedLoop
						}
					}
				}
				for _, resolvedC := range sortedConsts {
					if dep.Name == resolvedC.Lhs[0].Name {
						// This dependency has been resolved: move
						// on checking for next one.
						found = true
						break
					}
				}
				for _, f := range funcs {
					if dep.Name == f.Ident.Name {
						// This dependency has been resolved: move
						// on checking for next one.
						found = true
						break
					}
				}
				for _, t := range sortedTypes {
					if dep.Name == t.Ident.Name {
						// This dependency has been resolved: move
						// on checking for next one.
						found = true
						break
					}
				}
				if !found {
					// A dependency of v cannot be found.
					depsOk = false
					break
				}
			}
			if depsOk {
				// All dependencies of vars are resolved.
				sortedVars = append(sortedVars, v)
				vars = append(vars[:i], vars[i+1:]...)
				continue varsLoop
			}
		}
		// All remaining vars are added as "sorted"; a later stage of the
		// type checking will report the error (usually a not defined
		// symbol).
		sortedVars = append(sortedVars, vars...)
		vars = nil
	}

	sorted := []ast.Node{}
	if extends != nil {
		sorted = append(sorted, extends)
	}
	for _, imp := range imports {
		sorted = append(sorted, imp)
	}
	for _, t := range sortedTypes {
		sorted = append(sorted, t)
	}
	for _, c := range sortedConsts {
		sorted = append(sorted, c)
	}
	for _, v := range sortedVars {
		sorted = append(sorted, v)
	}
	for _, f := range funcs {
		sorted = append(sorted, f)
	}
	pkg.Declarations = sorted

	return nil

}

// checkPackage type checks a package.
//
// extendingFile indicates whether the package pkg was originally a template
// file that extended another template file.
func checkPackage(compilation *compilation, pkg *ast.Package, path string, importer native.Importer, opts checkerOptions, extendingFile bool) (err error) {

	// If the package has already been checked just return.
	if _, ok := compilation.pkgInfos[path]; ok {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(*CheckingError); ok {
				err = rerr
			} else {
				panic(r)
			}
		}
	}()

	tc := newTypechecker(compilation, path, opts, importer)

	// Check package level names for "init" and "main"
	// and check that constant declarations are balanced.
	for _, decl := range pkg.Declarations {
		switch decl := decl.(type) {
		case *ast.Var:
			for _, d := range decl.Lhs {
				if d.Name == "init" || d.Name == "main" {
					panic(tc.errorf(d.Pos(), "cannot declare %s - must be func", d.Name))
				}
			}
		case *ast.Const:
			tc.checkBalancedConstantDeclaration(decl)
			for _, d := range decl.Lhs {
				if d.Name == "init" || d.Name == "main" {
					panic(tc.errorf(d.Pos(), "cannot declare %s - must be func", d.Name))
				}
			}
		case *ast.TypeDeclaration:
			if name := decl.Ident.Name; name == "init" || name == "main" {
				panic(tc.errorf(decl.Pos(), "cannot declare %s - must be func", name))
			}
		case *ast.Import:
			if decl.Ident != nil && decl.Ident.Name == "init" {
				panic(tc.errorf(decl.Pos(), "cannot import package as init - init must be a func"))
			}
		}
	}

	// Sort the declarations in the package 'pkg' if it has not already been
	// sorted.
	if !compilation.alreadySortedPkgs[pkg] {
		err := sortDeclarations(pkg)
		if err != nil {
			loopErr := err.(initLoopError)
			return tc.errorf(loopErr.node, loopErr.msg)
		}
		compilation.alreadySortedPkgs[pkg] = true
	}

	// First: import packages.
	for _, d := range pkg.Declarations {
		if d, ok := d.(*ast.Import); ok {
			err := tc.checkImport(d)
			if err != nil {
				return err
			}
		}
	}

	// Second: check all type declarations.
	for _, d := range pkg.Declarations {
		if td, ok := d.(*ast.TypeDeclaration); ok {
			name, ti := tc.checkTypeDeclaration(td)
			if ti != nil {
				tc.assignScope(name, ti, td.Ident)
			}
		}
	}

	// Defines functions in file/package block before checking all
	// declarations.
	for _, d := range pkg.Declarations {
		if f, ok := d.(*ast.Func); ok {
			if f.Body == nil {
				return tc.errorf(f.Ident.Pos(), "missing function body")
			}
			if f.Ident.Name == "init" || f.Ident.Name == "main" {
				if len(f.Type.Parameters) > 0 || len(f.Type.Result) > 0 {
					return tc.errorf(f.Ident, "func %s must have no arguments and no return values", f.Ident.Name)
				}
			}
			if f.Type.Macro && len(f.Type.Result) == 0 {
				tc.makeMacroResultExplicit(f)
			}
			// Function type must be checked for every function, including
			// 'init's functions.
			funcType := tc.checkType(f.Type).Type
			if f.Ident.Name == "init" || isBlankIdentifier(f.Ident) {
				// Do not add 'init' and '_' functions to the file/package block.
				continue
			}
			if _, ok := tc.scopes.FilePackage(f.Ident.Name); ok {
				return tc.errorf(f.Ident, "%s redeclared in this block", f.Ident.Name)
			}
			ti := &typeInfo{Type: funcType}
			if f.Type.Macro {
				ti.Properties |= propertyIsMacroDeclaration
				if extendingFile {
					ti.Properties |= propertyMacroDeclaredInFileWithExtends
				}
			}
			tc.scopes.Declare(f.Ident.Name, ti, f.Ident)
		}
	}

	// Type check and defined functions, variables and constants.
	for _, d := range pkg.Declarations {
		switch d := d.(type) {
		case *ast.Func:
			tc.checkFunc(d)
		case *ast.Const:
			tc.checkConstantDeclaration(d)
		case *ast.Var:
			tc.checkVariableDeclaration(d)
		}
	}

	if tc.opts.mod != templateMod {
		// Check that the imported packages have been used.
		if node := tc.scopes.UnusedImport(); node != nil {
			var s string
			if node.Ident == nil || node.Ident.Name == "." {
				s = fmt.Sprintf("%q", node.Path)
			} else {
				s = fmt.Sprintf("%q as %s", node.Path, node.Ident)
			}
			return tc.errorf(node, "imported and not used: %s", s)
		}
	}

	if pkg.Name == "main" {
		if _, ok := tc.scopes.FilePackage("main"); !ok {
			return tc.errorf(new(ast.Position), "function main is undeclared in the main package")
		}
	}

	// Create a package info and store it into the compilation.
	pkgInfo := &packageInfo{
		Name:         pkg.Name,
		Declarations: make(map[string]*typeInfo, len(pkg.Declarations)),
		TypeInfos:    tc.compilation.typeInfos,
	}
	pkgInfo.Declarations = make(map[string]*typeInfo)
	for _, name := range tc.scopes.FilePackageNames() {
		isDummyMacroForRender := strings.HasPrefix(name, `"`) && strings.HasSuffix(name, `"`)
		if isExported(name) || isDummyMacroForRender {
			pkgInfo.Declarations[name], _ = tc.scopes.FilePackage(name)
		}
	}
	pkgInfo.IndirectVars = tc.compilation.indirectVars
	compilation.pkgInfos[path] = pkgInfo

	err = compilation.finalizeUsingStatements(tc)
	if err != nil {
		return err
	}

	return nil
}
