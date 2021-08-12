// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package compiler implements parsing, type checking and emitting of sources.
//
// A program can be compiled using
//
//	BuildProgram
//
// while a template is compiled through
//
//  BuildTemplate
//
package compiler

import (
	"fmt"
	"io"
	"io/fs"
	"reflect"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/runtime"
)

// formatTypeName reports the type name for each format.
var formatTypeName = [...]string{"string", "html", "css", "js", "json", "markdown"}

// internalOperatorZero and internalOperatorNotZero are two internal operators
// that are inserted in the tree by the type checker and that are handled by the
// emitter as two unary operators that return true if the operand is,
// respectively, the zero or not the zero of its type.
//
// As a special case, if the operand is an interface type then its value is
// compared with the zero of the dynamic type of the interface.
const (
	internalOperatorZero = ast.OperatorExtendedNot + iota + 1
	internalOperatorNotZero
)

// Options represents a set of options used during the compilation.
type Options struct {
	DisallowGoStmt       bool
	NoParseShortShowStmt bool

	// DollarIdentifier, when true, keeps the backward compatibility by
	// supporting the dollar identifier.
	//
	// NOTE: the dollar identifier is deprecated and will be removed in a
	// future version of Scriggo.
	DollarIdentifier bool

	FormatTypes map[ast.Format]reflect.Type
	Globals     Declarations

	// Packages loads Scriggo packages and precompiled packages.
	//
	// For template files, Packages only loads precompiled packages; the template
	// files are read from a file system.
	Packages PackageLoader

	// MDConverter converts a Markdown source code to HTML.
	MDConverter Converter

	TreeTransformer func(*ast.Tree) error
}

// Declarations.
type Declarations map[string]interface{}

// BuildProgram builds a Go program from the package in the root of fsys with
// the given options, loading the imported packages from packages.
//
// Current limitation: fsys can contain only one Go file in its root.
//
// If a compilation error occurs, it returns a CompilerError error.
func BuildProgram(fsys fs.FS, opts Options) (*Code, error) {

	// Parse the source code.
	tree, err := ParseProgram(fsys)
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
		mod:            programMod,
		disallowGoStmt: opts.DisallowGoStmt,
		globals:        opts.Globals,
	}
	tci, err := typecheck(tree, opts.Packages, checkerOpts)
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
	code, err := emitProgram(tree.Nodes[0].(*ast.Package), typeInfos, tci["main"].IndirectVars)
	if err != nil {
		return nil, err
	}

	return code, nil
}

// BuildScript builds a script.
// Any error related to the compilation itself is returned as a CompilerError.
func BuildScript(r io.Reader, opts Options) (*Code, error) {
	var tree *ast.Tree

	// Parse the source code.
	var err error
	tree, err = ParseScript(r, opts.Packages)
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
		mod:            scriptMod,
		disallowGoStmt: opts.DisallowGoStmt,
		globals:        opts.Globals,
	}
	tci, err := typecheck(tree, opts.Packages, checkerOpts)
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
	code, err := emitScript(tree, typeInfos, tci["main"].IndirectVars)

	return code, err
}

// BuildTemplate builds the named template file rooted at the given file
// system. If fsys implements FormatFS, the file format is read from its
// Format method, otherwise it depends on the extension of the file name.
// Any error related to the compilation itself is returned as a CompilerError.
func BuildTemplate(fsys fs.FS, name string, opts Options) (*Code, error) {

	var tree *ast.Tree

	// Parse the source code.
	var err error
	tree, err = ParseTemplate(fsys, name, opts.Packages, opts.NoParseShortShowStmt, opts.DollarIdentifier)
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
		disallowGoStmt: opts.DisallowGoStmt,
		formatTypes:    opts.FormatTypes,
		globals:        opts.Globals,
		mdConverter:    opts.MDConverter,
		mod:            templateMod,
	}
	tci, err := typecheck(tree, opts.Packages, checkerOpts)
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
	code, err := emitTemplate(tree, typeInfos, tci["main"].IndirectVars)

	return code, err
}

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

// CheckingError records a type checking error with the path and the position
// where the error occurred.
type CheckingError struct {
	path string
	pos  ast.Position
	err  error
}

// Error returns a string representation of the type checking error.
func (e *CheckingError) Error() string {
	return fmt.Sprintf("%s:%s: %s", e.path, e.pos, e.err)
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
	Value reflect.Value
}

// Code is the result of a package emitting process.
type Code struct {
	// Globals is a slice of all globals used in Code.
	Globals []Global
	// Functions is a map of exported functions indexed by name.
	Functions map[string]*runtime.Function
	// Main is the Code entry point.
	Main *runtime.Function
	// TypeOf returns the type of a value, including new types defined in code.
	TypeOf runtime.TypeOfFunc
}

// emitProgram emits the code for a program given its ast node, the type info
// and indirect variables. emitProgram returns an emittedPackage  instance
// with the global variables and the main function.
func emitProgram(pkgMain *ast.Package, typeInfos map[ast.Node]*typeInfo, indirectVars map[*ast.Identifier]bool) (_ *Code, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(*LimitExceededError); ok {
				err = e
				return
			}
			panic(r)
		}
	}()
	e := newEmitter(typeInfos, indirectVars)
	functions, _, _ := e.emitPackage(pkgMain, false, "main")
	main, _ := e.fnStore.availableScriggoFn(pkgMain, "main")
	pkg := &Code{
		Globals:   e.varStore.getGlobals(),
		Functions: functions,
		Main:      main,
		TypeOf:    e.types.TypeOf,
	}
	return pkg, nil
}

// emitScript emits the code for a script given its tree, the type info and
// indirect variables. emitScript returns a function that is the entry point
// of the script and the global variables.
func emitScript(tree *ast.Tree, typeInfos map[ast.Node]*typeInfo, indirectVars map[*ast.Identifier]bool) (_ *Code, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(*LimitExceededError); ok {
				err = e
				return
			}
			panic(err)
		}
	}()
	e := newEmitter(typeInfos, indirectVars)
	e.fb = newBuilder(newFunction("main", "main", reflect.FuncOf(nil, nil, false), tree.Path, tree.Pos()), tree.Path)
	e.fb.enterScope()
	e.emitNodes(tree.Nodes)
	e.fb.exitScope()
	e.fb.end()
	return &Code{Main: e.fb.fn, TypeOf: e.types.TypeOf, Globals: e.varStore.getGlobals()}, nil
}

// emitTemplate emits the code for a template given its tree, the type info and
// indirect variables. emitTemplate returns a function that is the entry point
// of the template and the global variables.
func emitTemplate(tree *ast.Tree, typeInfos map[ast.Node]*typeInfo, indirectVars map[*ast.Identifier]bool) (_ *Code, err error) {

	// Recover and eventually return a LimitExceededError.
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(*LimitExceededError); ok {
				err = e
				return
			}
			panic(r)
		}
	}()

	e := newEmitter(typeInfos, indirectVars)
	e.pkg = &ast.Package{}
	e.isTemplate = true
	typ := reflect.FuncOf(nil, nil, false)
	e.fb = newBuilder(newMacro("main", "main", typ, tree.Format, tree.Path, tree.Pos()), tree.Path)
	e.fb.changePath(tree.Path)

	// If the template file is a package it means that such file extends
	// another file.
	if len(tree.Nodes) == 1 {
		if pkg, ok := tree.Nodes[0].(*ast.Package); ok {
			mainBuilder := e.fb
			// Macro declarations in the extending file must be accessed by the
			// extended file.
			for _, dec := range pkg.Declarations {
				if fn, ok := dec.(*ast.Func); ok && fn.Type.Macro {
					macro := newMacro("main", fn.Ident.Name, fn.Type.Reflect, fn.Format, e.fb.getPath(), fn.Pos())
					e.fnStore.makeAvailableScriggoFn(e.pkg, fn.Ident.Name, macro)
				}
			}
			// Emits extended file.
			backupPath := e.fb.getPath()
			extends := pkg.Declarations[0].(*ast.Extends)
			e.fb.changePath(extends.Tree.Path)
			e.fb.fn.Format = runtime.Format(extends.Tree.Format)
			e.fb.enterScope()
			// Reserves first index of Functions for the function that
			// initializes package variables. There is no guarantee that such
			// function will exist: it depends on the presence or the absence of
			// package variables.
			var initVarsIndex int8 = 0
			e.fb.fn.Functions = append(e.fb.fn.Functions, nil)
			e.fb.emitCallFunc(initVarsIndex, e.fb.currentStackShift(), nil)
			e.emitNodes(extends.Tree.Nodes)
			e.fb.end()
			e.fb.exitScope()
			e.fb.changePath(backupPath)
			// Emits extending file as a package.
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
				nopFunction := newFunction("main", "$nop", reflect.FuncOf(nil, nil, false), "", &ast.Position{})
				nopBuilder := newBuilder(nopFunction, tree.Path)
				nopBuilder.end()
				e.fb.fn.Functions[0] = nopFunction
			}
			return &Code{Main: e.fb.fn, Globals: e.varStore.getGlobals()}, nil
		}
	}

	// Default case: tree is a generic template file.
	e.fb.enterScope()
	e.emitNodes(tree.Nodes)
	e.fb.exitScope()
	e.fb.end()
	return &Code{Main: e.fb.fn, TypeOf: e.types.TypeOf, Globals: e.varStore.getGlobals()}, nil

}

// getExtends returns the 'extends' node contained in nodes, if exists. Note
// that such node can only be preceded by a comment node or a text node.
func getExtends(nodes []ast.Node) (*ast.Extends, bool) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.Comment, *ast.Text:
		case *ast.Extends:
			return n, true
		case *ast.Statements:
			if len(n.Nodes) > 0 {
				n, ok := n.Nodes[0].(*ast.Extends)
				return n, ok
			}
			return nil, false
		default:
			return nil, false
		}
	}
	return nil, false
}
