// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package scripts allows the Go code to be executed as a script.
//
// Package scripts is EXPERIMENTAL. There is no guarantee that it will be
// included in the 1.0 version of Scriggo and could be removed at any time.
package scripts

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/compiler"
	"github.com/open2b/scriggo/runtime"
)

type BuildOptions struct {
	Globals        Declarations          // globals.
	DisallowGoStmt bool                  // disallow "go" statement.
	Packages       scriggo.PackageLoader // package loader used to load imported packages.
}

// Declarations.
type Declarations map[string]interface{}

type RunOptions struct {
	Context   context.Context
	PrintFunc runtime.PrintFunc
}

type Script struct {
	fn      *runtime.Function
	types   runtime.Types
	globals []compiler.Global
}

// Build builds a script with the given options, loading the imported packages
// from packages.
//
// If a compilation error occurs, it returns a CompilerError error.
func Build(src io.Reader, options *BuildOptions) (*Script, error) {
	co := compiler.Options{}
	if options != nil {
		co.Globals = compiler.Declarations(options.Globals)
		co.DisallowGoStmt = options.DisallowGoStmt
		co.Packages = options.Packages
	}
	code, err := compiler.BuildScript(src, co)
	if err != nil {
		return nil, err
	}
	return &Script{fn: code.Main, globals: code.Globals, types: code.Types}, nil
}

// Disassemble disassembles the script and returns its assembly code.
func (p *Script) Disassemble() []byte {
	assemblies := compiler.Disassemble(p.fn, p.globals, 0)
	return assemblies["main"]
}

// Run starts the script and waits for it to complete. vars contains the
// values of the global variables.
func (p *Script) Run(vars map[string]interface{}, options *RunOptions) (int, error) {
	vm := runtime.NewVM()
	if options != nil {
		if options.Context != nil {
			vm.SetContext(options.Context)
		}
		if options.PrintFunc != nil {
			vm.SetPrint(options.PrintFunc)
		}
	}
	return vm.Run(p.fn, p.types, initGlobalVariables(p.globals, vars))
}

// MustRun is like Run but panics if the run fails.
func (p *Script) MustRun(vars map[string]interface{}, options *RunOptions) int {
	code, err := p.Run(vars, options)
	if err != nil {
		panic(err)
	}
	return code
}

var emptyInit = map[string]interface{}{}

// initGlobalVariables initializes the global variables and returns their
// values. It panics if init is not valid.
func initGlobalVariables(variables []compiler.Global, init map[string]interface{}) []reflect.Value {
	n := len(variables)
	if n == 0 {
		return nil
	}
	if init == nil {
		init = emptyInit
	}
	values := make([]reflect.Value, n)
	for i, variable := range variables {
		if variable.Pkg == "main" {
			if value, ok := init[variable.Name]; ok {
				if variable.Value.IsValid() {
					panic(fmt.Sprintf("variable %q already initialized", variable.Name))
				}
				if value == nil {
					panic(fmt.Sprintf("variable initializer %q cannot be nil", variable.Name))
				}
				val := reflect.ValueOf(value)
				if typ := val.Type(); typ == variable.Type {
					v := reflect.New(typ).Elem()
					v.Set(val)
					values[i] = v
				} else {
					if typ.Kind() != reflect.Ptr || typ.Elem() != variable.Type {
						panic(fmt.Sprintf("variable initializer %q must have type %s or %s, but have %s",
							variable.Name, variable.Type, reflect.PtrTo(variable.Type), typ))
					}
					if val.IsNil() {
						panic(fmt.Sprintf("variable initializer %q cannot be a nil pointer", variable.Name))
					}
					values[i] = reflect.ValueOf(value).Elem()
				}
				continue
			}
		}
		if variable.Value.IsValid() {
			values[i] = variable.Value
		} else {
			values[i] = reflect.New(variable.Type).Elem()
		}
	}
	return values
}
