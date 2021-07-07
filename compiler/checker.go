// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/compiler/types"
	"github.com/open2b/scriggo/runtime"
)

// checkingMod represents the checking modality.
type checkingMod int

const (
	programMod checkingMod = iota + 1
	scriptMod
	templateMod
)

// typecheck makes a type check on tree.
// This is the entry point for the type checker.
// Note that tree may be altered during the type checking.
func typecheck(tree *ast.Tree, packages PackageLoader, opts checkerOptions) (map[string]*packageInfo, error) {

	if opts.mod == 0 {
		panic("unspecified modality")
	}

	// Type check a program.
	if opts.mod == programMod {
		pkg := tree.Nodes[0].(*ast.Package)
		if pkg.Name != "main" {
			return nil, &CheckingError{path: tree.Path, pos: *pkg.Pos(), err: errors.New("package name must be main")}
		}
		compilation := newCompilation()
		err := checkPackage(compilation, pkg, tree.Path, packages, opts, nil)
		if err != nil {
			return nil, err
		}
		return compilation.pkgInfos, nil
	}

	// Prepare the type checking for scripts and templates.
	var globalScope typeCheckerScope
	if opts.globals != nil {
		globals := &mapPackage{
			PkgName:      "main",
			Declarations: opts.globals,
		}
		globalScope = toTypeCheckerScope(globals, opts.mod, true, 0)
	}

	// Add the global "exit" to script global scope.
	if opts.mod == scriptMod {
		exit := scopeElement{t: &typeInfo{Properties: propertyUniverse}}
		if globalScope == nil {
			globalScope = typeCheckerScope{"exit": exit}
		} else if _, ok := globalScope["exit"]; !ok {
			globalScope["exit"] = exit
		}
	}

	compilation := newCompilation()
	tc := newTypechecker(compilation, tree.Path, opts, globalScope, packages)

	// Type check a template file which extends another file.
	if extends, ok := getExtends(tree.Nodes); ok {
		// First: all macro definitions in extending files are declared but not
		// initialized. This is necessary because the extended file can refer
		// to macro defined in the extending one, but these macro can contain
		// references to variables defined outside them.
		for _, d := range tree.Nodes[1:] {
			if m, ok := d.(*ast.Func); ok && m.Type.Macro {
				tc.makeMacroResultExplicit(m)
				ti := &typeInfo{
					Type:       tc.checkType(m.Type).Type,
					Properties: propertyIsMacroDeclaration | propertyMacroDeclaredInFileWithExtends,
				}
				tc.filePackageBlock[m.Ident.Name] = scopeElement{t: ti}
			}
		}
		// Second: type check the extended file in a new scope.
		currentPath := tc.path
		tc.path = extends.Tree.Path
		tc.inExtendedFile = true
		var err error
		extends.Tree.Nodes, err = tc.checkNodesInNewScopeError(extends.Tree.Nodes)
		if err != nil {
			return nil, err
		}
		tc.inExtendedFile = false
		tc.path = currentPath
		// Third: extending file is converted to a "package", that means that
		// out of order initialization is allowed.
		tc.templateFileToPackage(tree)
		err = checkPackage(compilation, tree.Nodes[0].(*ast.Package), tree.Path, packages, opts, tc.globalScope)
		if err != nil {
			return nil, err
		}
		// Collect data from the type checker and return it.
		mainPkgInfo := &packageInfo{}
		mainPkgInfo.IndirectVars = tc.compilation.indirectVars
		mainPkgInfo.TypeInfos = tc.compilation.typeInfos
		for _, pkgInfo := range compilation.pkgInfos {
			for k, v := range pkgInfo.TypeInfos {
				mainPkgInfo.TypeInfos[k] = v
			}
			for k, v := range pkgInfo.IndirectVars {
				mainPkgInfo.IndirectVars[k] = v
			}
		}
		return map[string]*packageInfo{"main": mainPkgInfo}, nil
	}

	// Type check a template file or a script.
	var err error
	tree.Nodes, err = tc.checkNodesInNewScopeError(tree.Nodes)
	if err != nil {
		return nil, err
	}
	mainPkgInfo := &packageInfo{}
	mainPkgInfo.IndirectVars = tc.compilation.indirectVars
	mainPkgInfo.TypeInfos = tc.compilation.typeInfos
	return map[string]*packageInfo{"main": mainPkgInfo}, nil
}

// checkerOptions contains the options for the type checker.
type checkerOptions struct {

	// mod is the checking modality.
	mod checkingMod

	// disallowGoStmt disables the "go" statement.
	disallowGoStmt bool

	// format types.
	formatTypes map[ast.Format]reflect.Type

	// global declarations.
	globals Declarations

	// renderer.
	renderer runtime.Renderer
}

// typechecker represents the state of the type checking.
type typechecker struct {

	// compilation holds the state of a single compilation across multiple
	// instances of 'typechecker'.
	compilation *compilation

	path string

	precompiledPkgs PackageLoader

	// universe is the outermost scope.
	universe typeCheckerScope

	// A globalScope is a scope between the universe and the file/package
	// block. In Go there is not an equivalent concept. In scripts and
	// templates, the declarations of the predefined package 'main' are added
	// to this scope; this makes possible, in templates, to access such
	// declarations from every template file, including imported and extended
	// ones.
	globalScope typeCheckerScope

	// filePackageBlock is a scope that holds the declarations from both the
	// file block and the package block.
	filePackageBlock typeCheckerScope

	// scopes holds the local scopes.
	scopes []typeCheckerScope

	// ancestors is the current list of ancestors. See the documentation of the
	// ancestor type for further details.
	ancestors []*ancestor

	// terminating reports whether current statement is terminating. In a
	// context other than ContextGo, the type checker does not check the
	// termination so the value of terminating is not significant. For
	// further details see https://golang.org/ref/spec#Terminating_statements.
	terminating bool

	// hasBreak reports whether a given statement node has a 'break' that refers
	// to it. This is necessary to determine if a 'breakable' statement (for,
	// switch or select) can be terminating or not.
	hasBreak map[ast.Node]bool

	// unusedVars keeps track of all declared but not used variables.
	unusedVars []*scopeVariable

	// unusedImports keeps track of all imported but not used packages.
	//
	// The key of the first map is the name of a package while the key of the
	// second map is the name of the imported declaration.
	//
	// When an identifier imported from a package is used, then such package is
	// removed from unusedImports. Doing so, when the type checking of a
	// file/program ends if some packages remain in unusedImports then it is a
	// type checking error.
	unusedImports map[string]map[string]*typeInfo

	// opts holds the options that define the behavior of the type checker.
	opts checkerOptions

	// iota holds the current iota value.
	iota int

	// Text structures for Goto and Labels checking.
	gotos           []string
	storedGotos     []string
	nextValidGoto   int
	storedValidGoto int
	labels          [][]string

	// scriptFuncOrMacroDecl reports whether the type checker is currently
	// checking a function declaration in a script or a macro declaration in a
	// template that has been transformed into an assignment node.
	scriptFuncOrMacroDecl bool

	// types refers the types of the current compilation and it is used to
	// create and manipulate types and values, both predefined and defined only
	// by Scriggo.
	types *types.Types

	// env is passed to the showFunc function and implements only the TypeOf method.
	env *env

	// structDeclPkg contains, for every struct literal and defined type with
	// underlying type 'struct' denoted in Scriggo, the package in which it has
	// been denoted.
	//
	// TODO: in theory we should keep track of the package in which the field
	// identifier (and not the struct type) has been declared, because the Go
	// type checker checks the package where the identifier has been declared
	// to see if it accessible or not. This has become relevant since embedded
	// struct fields have been implemented in Scriggo, so the package in which
	// a struct type is declared may be different from the package in which one
	// of its fields have been declared.
	structDeclPkg map[reflect.Type]string

	// inExtendedFile reports whether the type checker is type checking an
	// extended file.
	inExtendedFile bool
}

// newTypechecker creates a new type checker. A global scope may be provided
// for scripts and templates.
func newTypechecker(compilation *compilation, path string, opts checkerOptions, globalScope typeCheckerScope, precompiledPkgs PackageLoader) *typechecker {
	tt := types.NewTypes()
	tc := typechecker{
		compilation:      compilation,
		path:             path,
		filePackageBlock: typeCheckerScope{},
		globalScope:      globalScope,
		hasBreak:         map[ast.Node]bool{},
		universe:         universe,
		unusedImports:    map[string]map[string]*typeInfo{},
		opts:             opts,
		iota:             -1,
		types:            tt,
		env:              &env{tt.Runtime(), nil},
		structDeclPkg:    map[reflect.Type]string{},
		precompiledPkgs:  precompiledPkgs,
	}
	if len(opts.formatTypes) > 0 {
		tc.universe = typeCheckerScope{}
		for name, scope := range universe {
			tc.universe[name] = scope
		}
		for format, typ := range opts.formatTypes {
			name := formatTypeName[format]
			tc.universe[name] = scopeElement{t: &typeInfo{
				Type:       typ,
				Properties: propertyIsType | propertyIsFormatType | propertyUniverse,
			}}
		}
	}
	return &tc
}

// enterScope enters into a new empty scope.
func (tc *typechecker) enterScope() {
	tc.scopes = append(tc.scopes, typeCheckerScope{})
	tc.labels = append(tc.labels, []string{})
	tc.storedGotos = tc.gotos
	tc.gotos = []string{}
}

// exitScope exits from the current scope.
func (tc *typechecker) exitScope() {
	// Check if some variables declared in the closing scope are still unused.
	if tc.opts.mod != templateMod {
		unused := []struct {
			node  ast.Node
			ident string
		}{}
		cut := len(tc.unusedVars)
		for i := len(tc.unusedVars) - 1; i >= 0; i-- {
			v := tc.unusedVars[i]
			if v.scopeLevel < len(tc.scopes)-1 {
				break
			}
			if v.node != nil {
				unused = append(unused, struct {
					node  ast.Node
					ident string
				}{v.node, v.ident})
			}
			cut = i
		}
		if len(unused) > 0 {
			panic(tc.errorf(unused[len(unused)-1].node, "%s declared but not used", unused[len(unused)-1].ident))
		}
		tc.unusedVars = tc.unusedVars[:cut]
	}
	tc.scopes = tc.scopes[:len(tc.scopes)-1]
	tc.labels = tc.labels[:len(tc.labels)-1]
	tc.gotos = append(tc.storedGotos, tc.gotos...)
	tc.nextValidGoto = tc.storedValidGoto
}

// lookupScopesElem looks up name in the scopes. Returns the declaration of the
// name or false if the name does not exist. If justCurrentScope is true,
// lookupScopesElem looks up only in the current scope.
func (tc *typechecker) lookupScopesElem(name string, justCurrentScope bool) (scopeElement, bool) {
	// Iterating over scopes, from inside.
	for i := len(tc.scopes) - 1; i >= 0; i-- {
		elem, ok := tc.scopes[i][name]
		if ok {
			return elem, true
		}
		if justCurrentScope && i == len(tc.scopes)-1 {
			return scopeElement{}, false
		}
	}
	if len(tc.scopes) >= 1 && justCurrentScope {
		return scopeElement{}, false
	}
	// Package + file block.
	if elem, ok := tc.filePackageBlock[name]; ok {
		return elem, true
	}
	if justCurrentScope {
		return scopeElement{}, false
	}
	// Global scope.
	if elem, ok := tc.globalScope[name]; ok {
		return elem, true
	}
	// Universe.
	if elem, ok := tc.universe[name]; ok {
		return elem, true
	}
	return scopeElement{}, false
}

// lookupScopes looks up name in the scopes. Returns the type info of the name or
// false if the name does not exist. If justCurrentScope is true, lookupScopes
// looks up only in the current scope.
func (tc *typechecker) lookupScopes(name string, justCurrentScope bool) (*typeInfo, bool) {
	elem, ok := tc.lookupScopesElem(name, justCurrentScope)
	if !ok {
		return nil, false
	}
	return elem.t, ok
}

// assignScope assigns value to name in the current scope. If there are no
// scopes, value is assigned to the file/package block.
//
// assignScope panics a type checking error "redeclared in this block" if name
// is already defined in the current scope.
func (tc *typechecker) assignScope(name string, value *typeInfo, declNode *ast.Identifier) {

	if tc.declaredInThisBlock(name) {
		if tc.scriptFuncOrMacroDecl {
			switch tc.opts.mod {
			case scriptMod:
				panic(tc.errorf(declNode, "%s already declared in this program", declNode))
			case templateMod:
				panic(tc.errorf(declNode, "%s already declared in this template scope", declNode))
			}
		}
		previousDecl, _ := tc.lookupScopesElem(name, true)
		s := name + " redeclared in this block"
		if decl := previousDecl.decl; decl != nil {
			if pos := decl.Pos(); pos != nil {
				s += "\n\tprevious declaration at " + pos.String()
			}
		}
		panic(tc.errorf(declNode, s))
	}

	if len(tc.scopes) == 0 {
		tc.filePackageBlock[name] = scopeElement{t: value, decl: declNode}
	} else {
		tc.scopes[len(tc.scopes)-1][name] = scopeElement{t: value, decl: declNode}
	}

}

// An ancestor is an AST node with a scope level associated. The type checker
// holds a list of ancestors to keep track of the current position and depth
// inside the full AST tree.
//
// Note that an ancestor must be an AST node that can hold statements inside its
// body. For example an *ast.For node should be set as ancestor before checking
// it's body, while an *ast.Assignment node should not.
type ancestor struct {
	scopeLevel int
	node       ast.Node
}

// addToAncestors adds a node as ancestor.
func (tc *typechecker) addToAncestors(n ast.Node) {
	tc.ancestors = append(tc.ancestors, &ancestor{len(tc.scopes), n})
}

// removeLastAncestor removes current ancestor node.
func (tc *typechecker) removeLastAncestor() {
	tc.ancestors = tc.ancestors[:len(tc.ancestors)-1]
}

// currentFunction returns the current function and the related scope level.
// If it is called when not in a function body, returns nil and 0.
func (tc *typechecker) currentFunction() (*ast.Func, int) {
	for i := len(tc.ancestors) - 1; i >= 0; i-- {
		if f, ok := tc.ancestors[i].node.(*ast.Func); ok {
			return f, tc.ancestors[i].scopeLevel
		}
	}
	return nil, 0
}

// isUpVar checks if name is an upvar, that is a variable declared outside
// current function.
// TODO: is this function correct? It is correct to check the filePackageBlock first?
func (tc *typechecker) isUpVar(name string) bool {

	// Check if name is a package variable.
	if elem, ok := tc.filePackageBlock[name]; ok {
		// Elem must be a variable.
		return elem.t.Addressable()
	}

	// Check if name is a global variable in a template or script.
	if tc.opts.mod == templateMod || tc.opts.mod == scriptMod {
		if elem, ok := tc.globalScope[name]; ok {
			return elem.t.Addressable()
		}
	}

	// name is not a package variable; check if has been declared outside
	// current function.
	_, funcBound := tc.currentFunction()
	for i := len(tc.scopes) - 1; i >= 0; i-- {
		for n, elem := range tc.scopes[i] {
			if n != name {
				continue
			}
			if i < funcBound-1 { // out of current function scope.
				if elem.t.Addressable() { // elem must be a variable.
					tc.compilation.indirectVars[tc.scopes[i][n].decl] = true
					return true
				}
			}
			return false
		}
	}

	return false
}

// scopeLevelOf returns the scope level in which name is declared, and a
// boolean which reports whether name has been imported from another
// package/template file or not.
func (tc *typechecker) scopeLevelOf(name string) (int, bool) {
	// Iterating over scopes, from inside.
	for i := len(tc.scopes) - 1; i >= 0; i-- {
		if _, ok := tc.scopes[i][name]; ok {
			// If name is declared in a local scope then it's impossible that
			// its value has been imported.
			return i + 1, false
		}
	}
	return -1, tc.filePackageBlock[name].decl == nil
}

// getDeclarationNode returns the declaration node which declares name.
func (tc *typechecker) getDeclarationNode(name string) ast.Node {
	// Iterating over scopes, from inside.
	for i := len(tc.scopes) - 1; i >= 0; i-- {
		elem, ok := tc.scopes[i][name]
		if ok {
			return elem.decl
		}
	}
	// Package + file block.
	if elem, ok := tc.filePackageBlock[name]; ok {
		return elem.decl
	}
	// Global scope.
	if elem, ok := tc.globalScope[name]; ok {
		return elem.decl
	}
	// Universe.
	if elem, ok := tc.universe[name]; ok {
		return elem.decl
	}
	panic(fmt.Sprintf("BUG: trying to get scope level of %s, but any scope, package block, file block or universe contains it", name)) // remove.
}

// programImportError returns an error that states that it's impossible to
// import a program (package main).
func (tc *typechecker) programImportError(imp *ast.Import) error {
	return tc.errorf(imp, "import \"%s\" is a program, not a in importable package", imp.Path)
}

// getNestedFuncs returns an ordered list of all nested functions, starting from
// the function declaration that is a sibling of name declaration to the
// innermost function, which is the current.
//
// For example, given the source code:
//
// 		func f() {
// 			A := 10
// 			func g() {
// 				func h() {
// 					_ = A
// 				}
// 			}
// 		}
// calling getNestedFuncs("A") returns [G, H].
//
func (tc *typechecker) getNestedFuncs(name string) []*ast.Func {
	declLevel, imported := tc.scopeLevelOf(name)
	// If name has been imported, function chain does not exist.
	if imported {
		return nil
	}
	funcs := []*ast.Func{}
	for _, anc := range tc.ancestors {
		if fun, ok := anc.node.(*ast.Func); ok {
			if declLevel < anc.scopeLevel {
				funcs = append(funcs, fun)
			}
		}
	}
	return funcs
}

// nestedFuncs returns an ordered list of the nested functions, starting from
// the outermost function declaration to the innermost function, which is the
// current.
func (tc *typechecker) nestedFuncs() []*ast.Func {
	funcs := []*ast.Func{}
	for _, anc := range tc.ancestors {
		if fun, ok := anc.node.(*ast.Func); ok {
			funcs = append(funcs, fun)
		}
	}
	return funcs
}

// errorf builds and returns a type checking error. This method is used
// internally by the type checker: when it finds an error, instead of returning
// it the type checker panics with a CheckingError argument; this type of panics
// are recovered an converted into well-formatted errors before being returned
// by the compiler.
//
// For example:
//
//		if bad(node) {
//			panic(tc.errorf(node, "bad node"))
//		}
//
func (tc *typechecker) errorf(nodeOrPos interface{}, format string, args ...interface{}) error {
	var pos *ast.Position
	if node, ok := nodeOrPos.(ast.Node); ok {
		pos = node.Pos()
		if pos == nil {
			return fmt.Errorf(format, args...)
		}
	} else {
		pos = nodeOrPos.(*ast.Position)
	}
	var err = &CheckingError{
		path: tc.path,
		pos: ast.Position{
			Line:   pos.Line,
			Column: pos.Column,
			Start:  pos.Start,
			End:    pos.End,
		},
		err: fmt.Errorf(format, args...),
	}
	return err
}

type mapPackage struct {
	// Package name.
	PkgName string
	// Package declarations.
	Declarations Declarations
}

func (p *mapPackage) Name() string {
	return p.PkgName
}

func (p *mapPackage) Lookup(declName string) interface{} {
	return p.Declarations[declName]
}

func (p *mapPackage) DeclarationNames() []string {
	declarations := make([]string, 0, len(p.Declarations))
	for name := range p.Declarations {
		declarations = append(declarations, name)
	}
	return declarations
}
