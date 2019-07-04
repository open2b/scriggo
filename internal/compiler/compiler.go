// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package compiler implements parsing, type-checking and emitting of sources.
//
// Parsing
//
// Parsing is done using
//
//	ParseTemplate(..)
//	ParseProgram(..)
//	ParseScript(..)
//
// Typechecking
//
// When parsing is done, tree can be type-checked by:
//
// 	Typecheck(...)
//
// Emitting
//
// To emit a type-checked tree, use:
//
//  EmitTemplate(..)
//  EmitPackageMain(..)
//  EmitScript(..)
//
package compiler

import (
	"reflect"

	"scriggo/internal/compiler/ast"
	"scriggo/vm"
)

// Package represents a predefined package.
// A package can contain a function, a variable, a constant or a type.
//
//		Function: assign function value to Declarations as is.
//		Variable: assign the address of value to Declarations.
//		Constant: TODO(Gianluca).
//		Type:     assign the reflect.TypeOf of type to Declarations.
type Package struct {
	// Package name.
	Name string
	// Package declarations.
	Declarations map[string]interface{}
}

type Constant struct {
	value   interface{}
	literal string
	typ     reflect.Type // nil for untyped constants.
}

// ConstLiteral returns a constant, given its type and its literal
// representation, that can be used as a declaration in a predefined package.
//
// For untyped constants the type is nil.
func ConstLiteral(typ reflect.Type, literal string) Constant {
	return Constant{literal: literal, typ: typ}
}

// ConstValue returns a constant given its value.
func ConstValue(v interface{}) Constant {
	return Constant{value: v}
}

// PackageLoader is implemented by package loaders. Load returns a predefined
// package as *Package or the source of a non predefined package as
// an io.Reader.
//
// If the package does not exist it returns nil and nil.
// If the package exists but there was an error while loading the package, it
// returns nil and the error.
type PackageLoader interface {
	Load(pkgPath string) (interface{}, error)
}

// Options represents compilation options.
type Options struct {

	// AllowImports makes import statements available.
	AllowImports bool

	// NotUsedError returns a checking error if a variable is declared and not
	// used or a package is imported and not used.
	NotUsedError bool

	IsProgram, IsTemplate, IsScript bool

	// DisallowGoStmt disables the "go" statement.
	DisallowGoStmt bool

	// FailOnTODO makes compilation fail when a ShowMacro statement with "or
	// todo" option cannot be resolved.
	FailOnTODO bool

	// MemoryLimit adds Alloc instructions during compilation.
	MemoryLimit bool
}

// Typecheck typechecks tree. A map of predefined packages may be provided. deps
// must contain dependencies in case of package initialization (program or
// template import/extend).
// tree may be altered during typechecking.
func Typecheck(tree *ast.Tree, predefinedPkgs map[string]*Package, opts Options) (map[string]*PackageInfo, error) {
	deps := depsAnalysis(tree, opts)
	if opts.IsProgram {
		pkgInfos := map[string]*PackageInfo{}
		err := checkPackage(tree.Nodes[0].(*ast.Package), tree.Path, deps, predefinedPkgs, pkgInfos, opts.IsTemplate, opts.DisallowGoStmt)
		if err != nil {
			return nil, err
		}
		return pkgInfos, nil
	}
	tc := newTypechecker(tree.Path, opts)
	if main, ok := predefinedPkgs["main"]; ok {
		tc.Scopes = append(tc.Scopes, ToTypeCheckerScope(main))
	}
	if opts.IsTemplate {
		if extends, ok := tree.Nodes[0].(*ast.Extends); ok {
			for _, d := range tree.Nodes[1:] {
				if m, ok := d.(*ast.Macro); ok {
					f := macroToFunc(m)
					tc.filePackageBlock[f.Ident.Name] = scopeElement{t: &TypeInfo{Type: tc.typeof(f.Type, noEllipses).Type}}
				}
			}
			err := tc.checkNodesInNewScopeError(extends.Tree.Nodes)
			if err != nil {
				return nil, err
			}
			err = tc.templateToPackage(tree)
			if err != nil {
				return nil, err
			}
			pkgInfos := map[string]*PackageInfo{}
			err = checkPackage(tree.Nodes[0].(*ast.Package), tree.Path, deps, nil, pkgInfos, true, true)
			if err != nil {
				return nil, err
			}
			mainPkgInfo := &PackageInfo{}
			mainPkgInfo.IndirectVars = tc.IndirectVars
			mainPkgInfo.TypeInfo = tc.TypeInfo
			for _, pkgInfo := range pkgInfos {
				for k, v := range pkgInfo.TypeInfo {
					mainPkgInfo.TypeInfo[k] = v
				}
				for k, v := range pkgInfo.IndirectVars {
					mainPkgInfo.IndirectVars[k] = v
				}
			}
			return map[string]*PackageInfo{"main": mainPkgInfo}, nil
		}
	}
	tc.predefinedPkgs = predefinedPkgs
	err := tc.checkNodesInNewScopeError(tree.Nodes)
	if err != nil {
		return nil, err
	}
	mainPkgInfo := &PackageInfo{}
	mainPkgInfo.IndirectVars = tc.IndirectVars
	mainPkgInfo.TypeInfo = tc.TypeInfo
	return map[string]*PackageInfo{"main": mainPkgInfo}, nil
}

// Global represents a global variable with a package, name, type (only for
// not predefined globals) and value (only for predefined globals). Value, if
// present, must be a pointer to the variable value.
type Global struct {
	Pkg   string
	Name  string
	Type  reflect.Type
	Value interface{}
}

// Code is the result of a package emitting process.
type Code struct {
	// Globals is a slice of all globals used in Code.
	Globals []Global
	// Functions is a map of exported functions indexed by name.
	Functions map[string]*vm.Function
	// Main is the Code entry point.
	Main *vm.Function
}

// EmitPackageMain emits the code for a package main given its ast node, the
// type info and indirect variables. alloc reports whether Alloc instructions
// must be emitted. EmitPackageMain returns an emittedPackage instance with
// the global variables and the main function.
func EmitPackageMain(pkgMain *ast.Package, typeInfos map[ast.Node]*TypeInfo, indirectVars map[*ast.Identifier]bool, opts Options) *Code {
	e := newEmitter(typeInfos, indirectVars, opts)
	funcs, _, _ := e.emitPackage(pkgMain, false)
	main := e.availableFuncs[pkgMain]["main"]
	pkg := &Code{
		Globals:   e.globals,
		Functions: funcs,
		Main:      main,
	}
	return pkg
}

// EmitScript emits the code for a script given its tree, the type info and
// indirect variables. alloc reports whether Alloc instructions must be
// emitted. EmitScript returns a function that is the entry point of the
// script and the global variables.
func EmitScript(tree *ast.Tree, typeInfos map[ast.Node]*TypeInfo, indirectVars map[*ast.Identifier]bool, opts Options) *Code {
	e := newEmitter(typeInfos, indirectVars, opts)
	e.fb = newBuilder(newFunction("main", "main", reflect.FuncOf(nil, nil, false)))
	e.fb.SetAlloc(opts.MemoryLimit)
	e.fb.EnterScope()
	e.EmitNodes(tree.Nodes)
	e.fb.ExitScope()
	e.fb.End()
	return &Code{Main: e.fb.fn, Globals: e.globals}
}

// EmitTemplate emits the code for a template given its tree, the type info and
// indirect variables. alloc reports whether Alloc instructions must be
// emitted. EmitTemplate returns a function that is the entry point of the
// template and the global variables.
func EmitTemplate(tree *ast.Tree, typeInfos map[ast.Node]*TypeInfo, indirectVars map[*ast.Identifier]bool, opts Options) *Code {

	e := newEmitter(typeInfos, indirectVars, opts)
	e.pkg = &ast.Package{}
	e.isTemplate = true
	e.fb = newBuilder(newFunction("main", "main", reflect.FuncOf(nil, nil, false)))

	// Globals.
	e.globals = append(e.globals, Global{Pkg: "$template", Name: "$io.Writer", Type: emptyInterfaceType})
	e.globals = append(e.globals, Global{Pkg: "$template", Name: "$Write", Type: reflect.FuncOf(nil, nil, false)})
	e.globals = append(e.globals, Global{Pkg: "$template", Name: "$Render", Type: reflect.FuncOf(nil, nil, false)})
	e.fb.SetAlloc(opts.MemoryLimit)

	// If page is a package, then page extends another page.
	if len(tree.Nodes) == 1 {
		if pkg, ok := tree.Nodes[0].(*ast.Package); ok {
			mainBuilder := e.fb
			// Macro declarations in extending page must be accessed by extended page.
			e.availableFuncs[e.pkg] = map[string]*vm.Function{}
			for _, dec := range pkg.Declarations {
				if fun, ok := dec.(*ast.Func); ok {
					fn := newFunction("main", fun.Ident.Name, fun.Type.Reflect)
					e.availableFuncs[e.pkg][fun.Ident.Name] = fn
				}
			}
			// Emits extended page.
			extends := pkg.Declarations[0].(*ast.Extends)
			e.fb.EnterScope()
			e.reserveTemplateRegisters()
			// Reserves first index of Functions for the function that
			// initializes package variables. There is no guarantee that such
			// function will exist: it depends on the presence or the absence of
			// package variables.
			var initVarsIndex int8 = 0
			e.fb.fn.Functions = append(e.fb.fn.Functions, nil)
			e.fb.Call(initVarsIndex, vm.StackShift{}, 0)
			e.EmitNodes(extends.Tree.Nodes)
			e.fb.End()
			e.fb.ExitScope()
			// Emits extending page as a package.
			_, _, inits := e.emitPackage(pkg, true)
			e.fb = mainBuilder
			// Just one init is supported: the implicit one (the one that
			// initializes variables).
			if len(inits) == 1 {
				e.fb.fn.Functions[0] = inits[0]
			} else {
				// If there are no variables to initialize, a nop function is
				// created because space has already been reserved for it.
				nopFunction := newFunction("main", "$nop", reflect.FuncOf(nil, nil, false))
				nopBuilder := newBuilder(nopFunction)
				nopBuilder.End()
				e.fb.fn.Functions[0] = nopFunction
			}
			return &Code{Main: e.fb.fn, Globals: e.globals}
		}
	}

	// Default case: tree is a generic template page.
	e.fb.EnterScope()
	e.reserveTemplateRegisters()
	e.EmitNodes(tree.Nodes)
	e.fb.ExitScope()
	e.fb.End()
	return &Code{Main: e.fb.fn, Globals: e.globals}

}
