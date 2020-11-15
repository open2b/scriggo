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
	Globals   map[string]interface{}
}

type Script struct {
	fn      *runtime.Function
	typeof  runtime.TypeOfFunc
	globals []compiler.Global
}

// Load loads a script with the given options, loading the imported packages
// from loader.
func Load(src io.Reader, loader scriggo.PackageLoader, options *LoadOptions) (*Script, error) {
	co := compiler.Options{}
	if options != nil {
		co.Globals = compiler.Declarations(options.Globals)
		co.AllowShebangLine = options.AllowShebangLine
		co.DisallowGoStmt = options.DisallowGoStmt
	}
	code, err := compiler.CompileScript(src, loader, co)
	if err != nil {
		return nil, err
	}
	return &Script{fn: code.Main, globals: code.Globals, typeof: code.TypeOf}, nil
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

// Run starts the script and waits for it to complete.
func (p *Script) Run(options *RunOptions) (int, error) {
	vm := runtime.NewVM()
	var init map[string]interface{}
	if options != nil {
		if options.Context != nil {
			vm.SetContext(options.Context)
		}
		if options.PrintFunc != nil {
			vm.SetPrint(options.PrintFunc)
		}
		if options.Globals != nil {
			init = options.Globals
		}
	}
	return vm.Run(p.fn, p.typeof, initGlobals(p.globals, init))
}

// MustRun is like Run but panics if the run fails.
func (p *Script) MustRun(options *RunOptions) int {
	code, err := p.Run(options)
	if err != nil {
		panic(err)
	}
	return code
}

// initGlobals initializes the global variables and returns the values. It
// panics if init is not valid.
func initGlobals(globals []compiler.Global, init map[string]interface{}) []interface{} {
	n := len(globals)
	if n == 0 {
		return nil
	}
	values := make([]interface{}, n)
	for i, global := range globals {
		if global.Pkg == "main" {
			if value, ok := init[global.Name]; ok {
				if global.Value != nil {
					panic(fmt.Sprintf("variable %q already initialized", global.Name))
				}
				if value == nil {
					panic(fmt.Sprintf("variable initializer %q cannot be nil", global.Name))
				}
				val := reflect.ValueOf(value)
				if typ := val.Type(); typ == global.Type {
					v := reflect.New(typ).Elem()
					v.Set(val)
					values[i] = v.Addr().Interface()
				} else {
					if typ.Kind() != reflect.Ptr || typ.Elem() != global.Type {
						panic(fmt.Sprintf("variable initializer %q must have type %s or %s, but have %s",
							global.Name, global.Type, reflect.PtrTo(global.Type), typ))
					}
					if val.IsNil() {
						panic(fmt.Sprintf("variable initializer %q cannot be a nil pointer", global.Name))
					}
					values[i] = value
				}
				continue
			}
		}
		if global.Value == nil {
			values[i] = reflect.New(global.Type).Interface()
		} else {
			values[i] = global.Value
		}
	}
	return values
}
