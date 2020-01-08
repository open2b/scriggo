// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package compiler implements parsing, type checking and emitting of sources.
//
// A program can be compiled using
//
//	CompileProgram
//
// while a template is compiled through
//
//  CompileTemplate
//
package compiler

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"

	"scriggo/compiler/ast"
	"scriggo/runtime"
)

// Options represents a set of options used during the compilation.
type Options struct {
	AllowShebangLine   bool
	DisallowGoStmt     bool
	LimitMemorySize    bool
	PackageLess        bool
	TemplateContext    ast.Context
	TemplateFailOnTODO bool
	TreeTransformer    func(*ast.Tree) error
}

// CompileProgram compiles a program.
func CompileProgram(r io.Reader, importer PackageLoader, opts Options) (*Code, error) {
	var tree *ast.Tree

	// Parse the source code.
	if opts.PackageLess {
		var err error
		tree, err = ParsePackageLessProgram(r, importer, opts.AllowShebangLine)
		if err != nil {
			return nil, err
		}
	} else {
		mainSrc, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		tree, err = ParseProgram(mainCombiner{mainSrc, importer})
		if err != nil {
			return nil, err
		}
	}

	// Transform the tree.
	if opts.TreeTransformer != nil {
		err := opts.TreeTransformer(tree)
		if err != nil {
			return nil, err
		}
	}

	// Type check the tree.
	checkerOpts := checkerOptions{PackageLess: opts.PackageLess}
	checkerOpts.DisallowGoStmt = opts.DisallowGoStmt
	checkerOpts.SyntaxType = ProgramSyntax
	tci, err := typecheck(tree, importer, checkerOpts)
	if err != nil {
		return nil, err
	}
	typeInfos := map[ast.Node]*typeInfo{}
	for _, pkgInfos := range tci {
		for node, ti := range pkgInfos.TypeInfos {
			typeInfos[node] = ti
		}
	}

	// Emit the code.
	var code *Code
	emitterOpts := emitterOptions{}
	emitterOpts.MemoryLimit = opts.LimitMemorySize
	if opts.PackageLess {
		code = emitPackageLessProgram(tree, typeInfos, tci["main"].IndirectVars, emitterOpts)
	} else {
		code = emitPackageMain(tree.Nodes[0].(*ast.Package), typeInfos, tci["main"].IndirectVars, emitterOpts)
	}

	return code, nil
}

// CompileTemplate compiles a template.
func CompileTemplate(path string, r Reader, main PackageLoader, opts Options) (*Code, error) {

	var tree *ast.Tree

	// Parse the source code.
	var err error
	tree, err = ParseTemplate(path, r, ast.Context(opts.TemplateContext))
	if err != nil {
		return nil, err
	}

	// Transform the tree.
	if opts.TreeTransformer != nil {
		err := opts.TreeTransformer(tree)
		if err != nil {
			return nil, err
		}
	}

	// Type check the tree.
	checkerOpts := checkerOptions{
		AllowNotUsed:   true,
		DisallowGoStmt: opts.DisallowGoStmt,
		FailOnTODO:     opts.TemplateFailOnTODO,
		PackageLess:    opts.PackageLess,
		SyntaxType:     TemplateSyntax,
	}
	tci, err := typecheck(tree, main, checkerOpts)
	if err != nil {
		return nil, err
	}
	typeInfos := map[ast.Node]*typeInfo{}
	for _, pkgInfos := range tci {
		for node, ti := range pkgInfos.TypeInfos {
			typeInfos[node] = ti
		}
	}

	// Emit the code.
	emitterOpts := emitterOptions{}
	emitterOpts.MemoryLimit = opts.LimitMemorySize
	code := emitTemplate(tree, typeInfos, tci["main"].IndirectVars, emitterOpts)

	return code, nil
}

type mainCombiner struct {
	mainSrc      []byte
	otherImports PackageLoader
}

func (ml mainCombiner) Load(path string) (interface{}, error) {
	if path == "main" {
		return bytes.NewReader(ml.mainSrc), nil
	}
	if ml.otherImports != nil {
		return ml.otherImports.Load(path)
	}
	return nil, nil
}

// UntypedConst represents an untyped constant.
type UntypedConstant string

// predefinedPackage represents a predefined package.
type predefinedPackage interface {

	// Name returns the package's name.
	Name() string

	// Lookup searches for an exported declaration, named declName, in the
	// package. If the declaration does not exist, it returns nil.
	//
	// For a variable returns a pointer to the variable, for a function
	// returns the function, for a type returns the reflect.Type and for a
	// constant returns its value or a Constant.
	Lookup(declName string) interface{}

	// DeclarationNames returns the exported declaration names in the package.
	DeclarationNames() []string
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

// emitterOptions contains the options for the emitter.
type emitterOptions struct {

	// MemoryLimit adds Alloc instructions during compilation.
	MemoryLimit bool
}

// CheckingError records a type checking error with the path and the position
// where the error occurred.
type CheckingError struct {
	path    string
	parents string
	pos     ast.Position
	err     error
}

// Error returns a string representation of the type checking error.
func (e *CheckingError) Error() string {
	return fmt.Sprintf("%s:%s: %s%s", e.path, e.pos, e.err, e.parents)
}

// Message returns the message of the type checking error, without position and
// path.
func (e *CheckingError) Message() string {
	return e.err.Error()
}

// Path returns the path of the type checking error.
func (e *CheckingError) Path() string {
	return e.path
}

// Position returns the position of the checking error.
func (e *CheckingError) Position() ast.Position {
	return e.pos
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
	Functions map[string]*runtime.Function
	// Main is the Code entry point.
	Main *runtime.Function
}

// emitPackageMain emits the code for a package main given its ast node, the
// type info and indirect variables. alloc reports whether Alloc instructions
// must be emitted. emitPackageMain returns an emittedPackage instance with
// the global variables and the main function.
func emitPackageMain(pkgMain *ast.Package, typeInfos map[ast.Node]*typeInfo, indirectVars map[*ast.Identifier]bool, opts emitterOptions) *Code {
	e := newEmitter(typeInfos, indirectVars, opts)
	functions, _, _ := e.emitPackage(pkgMain, false, "main")
	main, _ := e.fnStore.availableScriggoFn(pkgMain, "main")
	pkg := &Code{
		Globals:   e.varStore.getGlobals(),
		Functions: functions,
		Main:      main,
	}
	return pkg
}

// emitPackageLessProgram emits the code for a package-less program given its
// tree, the type info and indirect variables. alloc reports whether Alloc
// instructions must be emitted. emitPackageLessProgram returns a function that
// is the entry point of the package-less program and the global variables.
func emitPackageLessProgram(tree *ast.Tree, typeInfos map[ast.Node]*typeInfo, indirectVars map[*ast.Identifier]bool, opts emitterOptions) *Code {
	e := newEmitter(typeInfos, indirectVars, opts)
	e.fb = newBuilder(newFunction("main", "main", reflect.FuncOf(nil, nil, false)), tree.Path)
	e.fb.emitSetAlloc(opts.MemoryLimit)
	e.fb.enterScope()
	e.emitNodes(tree.Nodes)
	e.fb.exitScope()
	e.fb.end()
	return &Code{Main: e.fb.fn, Globals: e.varStore.getGlobals()}
}

// emitTemplate emits the code for a template given its tree, the type info and
// indirect variables. alloc reports whether Alloc instructions must be
// emitted. emitTemplate returns a function that is the entry point of the
// template and the global variables.
func emitTemplate(tree *ast.Tree, typeInfos map[ast.Node]*typeInfo, indirectVars map[*ast.Identifier]bool, opts emitterOptions) *Code {

	e := newEmitter(typeInfos, indirectVars, opts)
	e.pkg = &ast.Package{}
	e.isTemplate = true
	e.fb = newBuilder(newFunction("main", "main", reflect.FuncOf(nil, nil, false)), tree.Path)
	e.fb.changePath(tree.Path)

	// Globals.
	_ = e.varStore.createScriggoPackageVar(e.pkg, newGlobal("$template", "$io.Writer", emptyInterfaceType, nil))
	_ = e.varStore.createScriggoPackageVar(e.pkg, newGlobal("$template", "$Write", reflect.FuncOf(nil, nil, false), nil))
	_ = e.varStore.createScriggoPackageVar(e.pkg, newGlobal("$template", "$Render", reflect.FuncOf(nil, nil, false), nil))
	_ = e.varStore.createScriggoPackageVar(e.pkg, newGlobal("$template", "$urlWriter", reflect.TypeOf(&struct{}{}), nil))
	e.fb.emitSetAlloc(opts.MemoryLimit)

	// If page is a package, then page extends another page.
	if len(tree.Nodes) == 1 {
		if pkg, ok := tree.Nodes[0].(*ast.Package); ok {
			mainBuilder := e.fb
			// Macro declarations in extending page must be accessed by the extended page.
			for _, dec := range pkg.Declarations {
				if fun, ok := dec.(*ast.Func); ok {
					fn := newFunction("main", fun.Ident.Name, fun.Type.Reflect)
					e.fnStore.makeAvailableScriggoFn(e.pkg, fun.Ident.Name, fn)
				}
			}
			// Emits extended page.
			backupPath := e.fb.getPath()
			extends, _ := getExtends(pkg.Declarations)
			e.fb.changePath(extends.Path)
			e.fb.enterScope()
			e.reserveTemplateRegisters()
			// Reserves first index of Functions for the function that
			// initializes package variables. There is no guarantee that such
			// function will exist: it depends on the presence or the absence of
			// package variables.
			var initVarsIndex int8 = 0
			e.fb.fn.Functions = append(e.fb.fn.Functions, nil)
			e.fb.emitCall(initVarsIndex, runtime.StackShift{}, nil)
			e.emitNodes(extends.Tree.Nodes)
			e.fb.end()
			e.fb.exitScope()
			e.fb.changePath(backupPath)
			// Emits extending page as a package.
			e.fb.changePath(tree.Path)
			_, _, inits := e.emitPackage(pkg, true, tree.Path)
			e.fb = mainBuilder
			// Just one init is supported: the implicit one (the one that
			// initializes variables).
			if len(inits) == 1 {
				e.fb.fn.Functions[0] = inits[0]
			} else {
				// If there are no variables to initialize, a nop function is
				// created because space has already been reserved for it.
				nopFunction := newFunction("main", "$nop", reflect.FuncOf(nil, nil, false))
				nopBuilder := newBuilder(nopFunction, tree.Path)
				nopBuilder.end()
				e.fb.fn.Functions[0] = nopFunction
			}
			return &Code{Main: e.fb.fn, Globals: e.varStore.getGlobals()}
		}
	}

	// Default case: tree is a generic template page.
	e.fb.enterScope()
	e.reserveTemplateRegisters()
	e.emitNodes(tree.Nodes)
	e.fb.exitScope()
	e.fb.end()
	return &Code{Main: e.fb.fn, Globals: e.varStore.getGlobals()}

}

// getExtends returns the 'extends' node contained in nodes, if exists. Note
// that such node can only be preceded by a comment node or a text node.
func getExtends(nodes []ast.Node) (*ast.Extends, bool) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.Comment, *ast.Text:
		case *ast.Extends:
			return n, true
		default:
			return nil, false
		}
	}
	return nil, false
}
