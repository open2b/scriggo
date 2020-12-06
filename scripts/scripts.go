// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

type LoadOptions struct {
	Globals          Declarations // globals.
	AllowShebangLine bool         // allow shebang line
	DisallowGoStmt   bool         // disallow "go" statement.
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

// Load loads a script with the given options, loading the imported packages
// from packages.
func Load(src io.Reader, packages scriggo.PackageLoader, options *LoadOptions) (*Script, error) {
	co := compiler.Options{}
	if options != nil {
		co.Globals = compiler.Declarations(options.Globals)
		co.AllowShebangLine = options.AllowShebangLine
		co.DisallowGoStmt = options.DisallowGoStmt
	}
	code, err := compiler.CompileScript(src, packages, co)
	if err != nil {
		return nil, err
	}
	return &Script{fn: code.Main, globals: code.Globals, types: code.Types}, nil
}

// Disassemble disassembles the script.
func (p *Script) Disassemble(w io.Writer) (int64, error) {
	packages, err := compiler.Disassemble(p.fn, p.globals)
	if err != nil {
		return 0, err
	}
	n, err := io.WriteString(w, packages["main"])
	return int64(n), err
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
func initGlobalVariables(variables []compiler.Global, init map[string]interface{}) []interface{} {
	n := len(variables)
	if n == 0 {
		return nil
	}
	if init == nil {
		init = emptyInit
	}
	values := make([]interface{}, n)
	for i, variable := range variables {
		if variable.Pkg == "main" {
			if value, ok := init[variable.Name]; ok {
				if variable.Value != nil {
					panic(fmt.Sprintf("variable %q already initialized", variable.Name))
				}
				if value == nil {
					panic(fmt.Sprintf("variable initializer %q cannot be nil", variable.Name))
				}
				val := reflect.ValueOf(value)
				if typ := val.Type(); typ == variable.Type {
					v := reflect.New(typ).Elem()
					v.Set(val)
					values[i] = v.Addr().Interface()
				} else {
					if typ.Kind() != reflect.Ptr || typ.Elem() != variable.Type {
						panic(fmt.Sprintf("variable initializer %q must have type %s or %s, but have %s",
							variable.Name, variable.Type, reflect.PtrTo(variable.Type), typ))
					}
					if val.IsNil() {
						panic(fmt.Sprintf("variable initializer %q cannot be a nil pointer", variable.Name))
					}
					values[i] = value
				}
				continue
			}
		}
		if variable.Value == nil {
			values[i] = reflect.New(variable.Type).Interface()
		} else {
			values[i] = variable.Value
		}
	}
	return values
}
