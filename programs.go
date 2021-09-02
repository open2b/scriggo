// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scriggo

import (
	"context"
	"errors"
	"io/fs"
	"reflect"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/compiler"
	"github.com/open2b/scriggo/internal/runtime"
	"github.com/open2b/scriggo/native"
)

// BuildOptions contains options for building programs and templates.
type BuildOptions struct {

	// AllowGoStmt, when true, allows the use of the go statement.
	AllowGoStmt bool

	// Packages is a package importer that makes native packages available
	// in programs and templates through the import statement.
	Packages native.Importer

	// TreeTransformer is a function that transforms a tree. If it is not nil,
	// it is called before the type checking.
	//
	// Used for templates only.
	TreeTransformer func(tree *ast.Tree) error

	// NoParseShortShowStmt, when true, don't parse the short show statements.
	//
	// Used for templates only.
	NoParseShortShowStmt bool

	// MarkdownConverter converts a Markdown source code to HTML.
	//
	// Used for templates only.
	MarkdownConverter Converter

	// Globals declares constants, types, variables, functions and packages
	// that are accessible from the code in the template.
	//
	// Used for templates only.
	Globals native.Declarations

	// DollarIdentifier, when true, keeps the backward compatibility by
	// supporting the dollar identifier.
	//
	// NOTE: the dollar identifier is deprecated and will be removed in a
	// future version of Scriggo.
	//
	// Used for templates only.
	DollarIdentifier bool
}

// PrintFunc represents a function that prints the arguments of the print and
// println builtins.
type PrintFunc func(interface{})

// RunOptions are the run options.
type RunOptions struct {

	// Context is a context that can be read by native functions and methods
	// via the Context method of native.Env.
	Context context.Context

	// Print is called by the print and println builtins to print values.
	// If it is nil, the print and println builtins format their arguments as
	// expected and write the result to standard error.
	Print PrintFunc
}

// Program is a program compiled with the Build function.
type Program struct {
	fn      *runtime.Function
	typeof  runtime.TypeOfFunc
	globals []compiler.Global
}

// Build builds a program from the package in the root of fsys with the given
// options.
//
// Current limitation: fsys can contain only one Go file in its root.
//
// If a build error occurs, it returns a *BuildError.
func Build(fsys fs.FS, options *BuildOptions) (*Program, error) {
	co := compiler.Options{}
	if options != nil {
		co.AllowGoStmt = options.AllowGoStmt
		co.Importer = options.Packages
	}
	code, err := compiler.BuildProgram(fsys, co)
	if err != nil {
		if e, ok := err.(compiler.Error); ok {
			err = &BuildError{err: e}
		}
		return nil, err
	}
	return &Program{fn: code.Main, globals: code.Globals, typeof: code.TypeOf}, nil
}

// Disassemble disassembles the package with the given path and returns its
// assembly code. Native packages can not be disassembled.
func (p *Program) Disassemble(pkgPath string) ([]byte, error) {
	assemblies := compiler.Disassemble(p.fn, p.globals, 0)
	asm, ok := assemblies[pkgPath]
	if !ok {
		return nil, errors.New("scriggo: package path does not exist")
	}
	return asm, nil
}

// Run starts the program and waits for it to complete.
//
// If the executed program panics or the Panic method of native.Env is called,
// and the executed code does not recover the panic, Run returns a
// *PanicError.
//
// If the Exit method of native.Env is called with a non-zero code, Run
// returns a *ExitError with the exit code.
//
// If the Fatal method of native.Env is called with argument v, Run panics
// with the value v.
func (p *Program) Run(options *RunOptions) error {
	vm := runtime.NewVM()
	if options != nil {
		if options.Context != nil {
			vm.SetContext(options.Context)
		}
		if options.Print != nil {
			vm.SetPrint(runtime.PrintFunc(options.Print))
		}
	}
	code, err := vm.Run(p.fn, p.typeof, initPackageLevelVariables(p.globals))
	if err != nil {
		if p, ok := err.(*runtime.PanicError); ok {
			err = &PanicError{p}
		}
		return err
	}
	if code != 0 {
		return ExitError(code)
	}
	return nil
}

// initPackageLevelVariables initializes the package level variables and
// returns the values.
func initPackageLevelVariables(globals []compiler.Global) []reflect.Value {
	n := len(globals)
	if n == 0 {
		return nil
	}
	values := make([]reflect.Value, n)
	for i, global := range globals {
		if global.Value.IsValid() {
			values[i] = global.Value
		} else {
			values[i] = reflect.New(global.Type).Elem()
		}
	}
	return values
}
