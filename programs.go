// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scriggo

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"reflect"

	"github.com/open2b/scriggo/internal/compiler"
	"github.com/open2b/scriggo/internal/runtime"
	"github.com/open2b/scriggo/native"
)

type BuildOptions struct {
	DisallowGoStmt bool                 // disallow "go" statement.
	Packages       native.PackageLoader // package loader used to load imported packages.
}

type PrintFunc func(interface{})

type RunOptions struct {
	Context   context.Context
	PrintFunc PrintFunc
}

type Program struct {
	fn      *runtime.Function
	typeof  runtime.TypeOfFunc
	globals []compiler.Global
}

var ErrTooManyGoFiles = compiler.ErrTooManyGoFiles
var ErrNoGoFiles = compiler.ErrNoGoFiles

// Build builds a Go program from the package in the root of fsys with the
// given options, loading the imported packages from packages.
//
// Current limitation: fsys can contain only one Go file in its root.
//
// If a build error occurs, it returns a *BuildError error.
func Build(fsys fs.FS, options *BuildOptions) (*Program, error) {
	co := compiler.Options{}
	if options != nil {
		co.DisallowGoStmt = options.DisallowGoStmt
		co.Packages = options.Packages
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
	code, err := vm.Run(p.fn, p.typeof, initPackageLevelVariables(p.globals))
	if p, ok := err.(*runtime.Panic); ok {
		err = &Panic{p}
	}
	return code, err
}

// MustRun is like Run but panics if the run fails.
func (p *Program) MustRun(options *RunOptions) int {
	code, err := p.Run(options)
	if err != nil {
		panic(err)
	}
	return code
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
func Errorf(env native.Env, format string, a ...interface{}) error {
	err := fmt.Sprintf(format, a...)
	return errors.New(err)
}
