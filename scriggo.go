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
	"reflect"

	"scriggo/compiler"
	"scriggo/compiler/ast"
	"scriggo/runtime"
)

// CompilerError represents an error returned by the compiler.
type CompilerError interface {
	error
	Position() ast.Position
	Path() string
	Message() string
}

type LoadOptions struct {
	LimitMemory bool // limit the execution memory size.
	OutOfSpec   struct {
		AllowShebangLine bool         // allow shebang line; only for package-less programs.
		DisallowGoStmt   bool         // disallow "go" statement.
		PackageLess      bool         // enable the package-less syntax.
		Builtins         Declarations // builtins.
	}
}

// Declarations.
type Declarations map[string]interface{}

type RunOptions struct {
	Context       context.Context
	MemoryLimiter runtime.MemoryLimiter
	PrintFunc     runtime.PrintFunc
	OutOfSpec     struct {
		Builtins map[string]interface{}
	}
}

type Program struct {
	fn          *runtime.Function
	globals     []compiler.Global
	limitMemory bool
}

// Load loads a Go program with the given options, loading the imported
// packages from loader.
func Load(src io.Reader, loader PackageLoader, options *LoadOptions) (*Program, error) {
	co := compiler.Options{}
	if options != nil {
		if options.OutOfSpec.Builtins != nil && !options.OutOfSpec.PackageLess {
			panic("scriggo: PackageLess option is required for builtins")
		}
		co.AllowShebangLine = options.OutOfSpec.AllowShebangLine
		co.DisallowGoStmt = options.OutOfSpec.DisallowGoStmt
		co.LimitMemory = options.LimitMemory
		co.PackageLess = options.OutOfSpec.PackageLess
		co.Builtins = compiler.Declarations(options.OutOfSpec.Builtins)
	}
	code, err := compiler.CompileProgram(src, loader, co)
	if err != nil {
		return nil, err
	}
	return &Program{fn: code.Main, globals: code.Globals, limitMemory: co.LimitMemory}, nil
}

// Disassemble disassembles the package with the given path. Predefined
// packages can not be disassembled.
func (p *Program) Disassemble(w io.Writer, pkgPath string) (int64, error) {
	packages, err := compiler.Disassemble(p.fn, p.globals)
	if err != nil {
		return 0, err
	}
	asm, ok := packages[pkgPath]
	if !ok {
		return 0, errors.New("scriggo: package path does not exist")
	}
	n, err := io.WriteString(w, asm)
	return int64(n), err
}

// Run starts the program and waits for it to complete.
//
// Panics if the option MemoryLimiter is not nil but the program has not been
// loaded with option LimitMemory.
func (p *Program) Run(options *RunOptions) (int, error) {
	if options != nil && options.MemoryLimiter != nil && !p.limitMemory {
		panic("scriggo: program not loaded with LimitMemory option")
	}
	vm := newVM(options)
	if options != nil && options.OutOfSpec.Builtins != nil {
		return vm.Run(p.fn, initGlobals(p.globals, options.OutOfSpec.Builtins))
	}
	return vm.Run(p.fn, initGlobals(p.globals, nil))
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

// newVM returns a new vm with the given options.
func newVM(options *RunOptions) *runtime.VM {
	vm := runtime.NewVM()
	if options != nil {
		if options.Context != nil {
			vm.SetContext(options.Context)
		}
		vm.SetMemoryLimiter(options.MemoryLimiter)
		if options.PrintFunc != nil {
			vm.SetPrint(options.PrintFunc)
		}
	}
	return vm
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

// IsLimitExceeded reports whether the error is a limit exceeded compiler error.
//
// These limitations have been arbitrarily added to Scriggo to enhance
// performances:
//
// * 127 registers of a given type (integer, floating point, string or
// 	general) per function
// 	* 256 function literal declarations plus unique functions calls per
// 	function
// * 256 types available per function
// * 256 unique predefined functions per function
// * 256 integer values per function
// * 256 string values per function
// * 256 floating-point values per function
// * 256 general values per function
//
func IsLimitExceeded(err error) bool {
	_, ok := err.(*compiler.LimitExceededError)
	return ok
}

// Errorf formats according to a format specifier, reserves the memory for the
// string if the memory is limited and returns the string as a value that
// satisfies error.
//
// Unlike the function fmt.Errorf, Errorf does not recognize the %w verb in
// format.
func Errorf(env runtime.Env, format string, a ...interface{}) error {
	err := fmt.Sprintf(format, a...)
	runtime.ReserveMemory(env, 24+len(err))
	return errors.New(err)
}
