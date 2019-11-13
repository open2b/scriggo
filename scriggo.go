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

	"scriggo/ast"
	"scriggo/internal/compiler"
	"scriggo/runtime"
)

type LoadOptions struct {
	LimitMemorySize  bool // limit allocable memory size.
	DisallowGoStmt   bool // disallow "go" statement.
	AllowShebangLine bool // allow shebang line; only for scripts.
}

// UntypedConstant represents an untyped constant.
type UntypedConstant = compiler.UntypedConstant

type Program struct {
	fn      *runtime.Function
	globals []compiler.Global
	options *LoadOptions
}

// CompilerError represents an error returned by the compiler.
type CompilerError interface {
	error
	Position() ast.Position
	Path() string
	Message() string
}

// Load loads a Go program with the given options, loading the main package and
// the imported packages from loader. A main package have path "main".
func Load(loader PackageLoader, options *LoadOptions) (*Program, error) {

	tree, err := compiler.ParseProgram(loader)
	if err != nil {
		return nil, err
	}

	checkerOpts := compiler.CheckerOptions{SyntaxType: compiler.ProgramSyntax}
	emitterOpts := compiler.EmitterOptions{}
	if options != nil {
		checkerOpts.DisallowGoStmt = options.DisallowGoStmt
		emitterOpts.MemoryLimit = options.LimitMemorySize
	}

	tci, err := compiler.Typecheck(tree, loader, checkerOpts)
	if err != nil {
		return nil, err
	}
	typeInfos := map[ast.Node]*compiler.TypeInfo{}
	for _, pkgInfos := range tci {
		for node, ti := range pkgInfos.TypeInfos {
			typeInfos[node] = ti
		}
	}

	pkgMain := compiler.EmitPackageMain(tree.Nodes[0].(*ast.Package), typeInfos, tci["main"].IndirectVars, emitterOpts)

	return &Program{fn: pkgMain.Main, globals: pkgMain.Globals, options: options}, nil
}

// Options returns the options with which the program has been loaded.
func (p *Program) Options() *LoadOptions {
	return p.options
}

type RunOptions struct {
	Context       context.Context
	MaxMemorySize int
	DontPanic     bool
	PrintFunc     runtime.PrintFunc
	TraceFunc     runtime.TraceFunc
}

// Run starts the program and waits for it to complete.
//
// Panics if the option MaxMemorySize is greater than zero but the program has
// not been loaded with option LimitMemorySize.
func (p *Program) Run(options *RunOptions) error {
	if options != nil && options.MaxMemorySize > 0 && !p.options.LimitMemorySize {
		panic("scriggo: program not loaded with LimitMemorySize option")
	}
	vm := newVM(options)
	return vm.Run(p.fn, initGlobals(p.globals, nil))
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

type Script struct {
	fn      *runtime.Function
	globals []compiler.Global
	options *LoadOptions
}

// LoadScript loads a script from src with the given options, loading the
// imported packages from loader.
func LoadScript(src io.Reader, loader PackageLoader, options *LoadOptions) (*Script, error) {

	checkerOpts := compiler.CheckerOptions{
		SyntaxType: compiler.ScriptSyntax,
	}
	emitterOpts := compiler.EmitterOptions{}
	shebang := false
	if options != nil {
		shebang = options.AllowShebangLine
		checkerOpts.DisallowGoStmt = options.DisallowGoStmt
		emitterOpts.MemoryLimit = options.LimitMemorySize
	}

	tree, err := compiler.ParseScript(src, loader, shebang)
	if err != nil {
		return nil, err
	}

	tci, err := compiler.Typecheck(tree, loader, checkerOpts)
	if err != nil {
		return nil, err
	}

	code := compiler.EmitScript(tree, tci["main"].TypeInfos, tci["main"].IndirectVars, emitterOpts)

	return &Script{fn: code.Main, globals: code.Globals, options: options}, nil
}

// Options returns the options with which the script has been loaded.
func (s *Script) Options() *LoadOptions {
	return s.options
}

var emptyInit = map[string]interface{}{}

// Run starts the script, with initialization values for the global variables,
// and waits for it to complete.
//
// Panics if the option MaxMemorySize is greater than zero but the script has
// not been loaded with option LimitMemorySize.
func (s *Script) Run(init map[string]interface{}, options *RunOptions) error {
	if options != nil && options.MaxMemorySize > 0 && !s.options.LimitMemorySize {
		panic("scriggo: script not loaded with LimitMemorySize option")
	}
	if init == nil {
		init = emptyInit
	}
	vm := newVM(options)
	return vm.Run(s.fn, initGlobals(s.globals, init))
}

// newVM returns a new vm with the given options.
func newVM(options *RunOptions) *runtime.VM {
	vm := runtime.NewVM()
	if options != nil {
		if options.Context != nil {
			vm.SetContext(options.Context)
		}
		if options.MaxMemorySize > 0 {
			vm.SetMaxMemory(options.MaxMemorySize)
		}
		if options.PrintFunc != nil {
			vm.SetPrint(options.PrintFunc)
		}
		if options.TraceFunc != nil {
			vm.SetTraceFunc(options.TraceFunc)
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

// Disassemble disassembles a script.
func (s *Script) Disassemble(w io.Writer) (int64, error) {
	return compiler.DisassembleFunction(w, s.fn, s.globals)
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
