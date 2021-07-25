// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scriggo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"reflect"

	"github.com/open2b/scriggo/compiler"
	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/runtime"
)

// CompilerError represents an error returned by the compiler.
// The types that implement the CompilerError interface are four types of the
// compiler package
//
//  *compiler.SyntaxError
//  *compiler.CycleError
//  *compiler.CheckingError
//  *compiler.LimitExceededError
//
type CompilerError interface {
	error
	Position() ast.Position
	Path() string
	Message() string
}

type BuildOptions struct {
	DisallowGoStmt bool          // disallow "go" statement.
	Packages       PackageLoader // package loader used to load imported packages.
}

type RunOptions struct {
	Context   context.Context
	PrintFunc runtime.PrintFunc
}

type Program struct {
	fn      *runtime.Function
	types   runtime.Types
	globals []compiler.Global
}

var ErrTooManyGoFiles = compiler.ErrTooManyGoFiles
var ErrNoGoFiles = compiler.ErrNoGoFiles

// Build builds a Go program from the package in the root of fsys with the
// given options, loading the imported packages from packages.
//
// Current limitation: fsys can contain only one Go file in its root.
//
// If a compilation error occurs, it returns a CompilerError error.
func Build(fsys fs.FS, options *BuildOptions) (*Program, error) {
	co := compiler.Options{}
	if options != nil {
		co.DisallowGoStmt = options.DisallowGoStmt
		co.Packages = options.Packages
	}
	code, err := compiler.BuildProgram(fsys, co)
	if err != nil {
		return nil, err
	}
	return &Program{fn: code.Main, globals: code.Globals, types: code.Types}, nil
}

// Disassemble disassembles the package with the given path and returns its
// assembly code. Predefined packages can not be disassembled.
func (p *Program) Disassemble(pkgPath string) ([]byte, error) {
	assemblies := compiler.Disassemble(p.fn, p.globals, 0)
	asm, ok := assemblies[pkgPath]
	if !ok {
		return nil, errors.New("scriggo: package path does not exist")
	}
	return asm, nil
}

// Run starts the program and waits for it to complete.
func (p *Program) Run(options *RunOptions) (int, error) {
	vm := runtime.NewVM()
	if options != nil {
		if options.Context != nil {
			vm.SetContext(options.Context)
		}
		if options.PrintFunc != nil {
			vm.SetPrint(options.PrintFunc)
		}
	}
	return vm.Run(p.fn, p.types, initPackageLevelVariables(p.globals))
}

// MustRun is like Run but panics if the run fails.
func (p *Program) MustRun(options *RunOptions) int {
	code, err := p.Run(options)
	if err != nil {
		panic(err)
	}
	return code
}

// PrintFunc returns a function that print its argument to the writer w with
// the same format used by the builtin print to print to the standard error.
// The returned function can be used for the PrintFunc option.
func PrintFunc(w io.Writer) runtime.PrintFunc {
	return func(v interface{}) {
		r := reflect.ValueOf(v)
		switch r.Kind() {
		case reflect.Invalid, reflect.Array, reflect.Func, reflect.Interface, reflect.Ptr, reflect.Struct:
			_, _ = fmt.Fprintf(w, "%#x", reflect.ValueOf(&v).Elem().InterfaceData()[1])
		case reflect.Bool:
			_, _ = fmt.Fprintf(w, "%t", r.Bool())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			_, _ = fmt.Fprintf(w, "%d", r.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			_, _ = fmt.Fprintf(w, "%d", r.Uint())
		case reflect.Float32, reflect.Float64:
			_, _ = fmt.Fprintf(w, "%e", r.Float())
		case reflect.Complex64, reflect.Complex128:
			fmt.Printf("%e", r.Complex())
		case reflect.Chan, reflect.Map, reflect.UnsafePointer:
			_, _ = fmt.Fprintf(w, "%#x", r.Pointer())
		case reflect.Slice:
			_, _ = fmt.Fprintf(w, "[%d/%d] %#x", r.Len(), r.Cap(), r.Pointer())
		case reflect.String:
			_, _ = fmt.Fprint(w, r.String())
		}
	}
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

// Errorf formats according to a format specifier, and returns the string as a
// value that satisfies error.
//
// Unlike the function fmt.Errorf, Errorf does not recognize the %w verb in
// format.
func Errorf(env runtime.Env, format string, a ...interface{}) error {
	err := fmt.Sprintf(format, a...)
	return errors.New(err)
}
