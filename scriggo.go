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
	"strings"

	"scriggo/ast"
	"scriggo/internal/compiler"
	"scriggo/vm"
)

type LoadOptions struct {
	LimitMemorySize  bool // limit allocable memory size.
	DisallowGoStmt   bool // disallow "go" statement.
	AllowShebangLine bool // allow shebang line; only for scripts.
}

// UntypedConstant represents an untyped constant.
type UntypedConstant = compiler.UntypedConstant

// Package represents a predefined package.
type Package interface {

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

type Program struct {
	fn      *vm.Function
	globals []compiler.Global
	options *LoadOptions
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
	PrintFunc     vm.PrintFunc
	TraceFunc     vm.TraceFunc
}

// Run starts the program and waits for it to complete.
//
// Panics if the option MaxMemorySize is greater than zero but the program has
// not been loaded with option LimitMemorySize.
func (p *Program) Run(options *RunOptions) error {
	if options != nil && options.MaxMemorySize > 0 && !p.options.LimitMemorySize {
		panic("scriggo: program not loaded with LimitMemorySize option")
	}
	vmm := newVM(options)
	_, err := vmm.Run(p.fn, initGlobals(p.globals, nil))
	return err
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
	fn      *vm.Function
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
	vmm := newVM(options)
	_, err := vmm.Run(s.fn, initGlobals(s.globals, init))
	return err
}

// newVM returns a new vm with the given options.
func newVM(options *RunOptions) *vm.VM {
	vmm := vm.New()
	if options != nil {
		if options.Context != nil {
			vmm.SetContext(options.Context)
		}
		if options.MaxMemorySize > 0 {
			vmm.SetMaxMemory(options.MaxMemorySize)
		}
		if options.PrintFunc != nil {
			vmm.SetPrint(options.PrintFunc)
		}
		if options.TraceFunc != nil {
			vmm.SetTraceFunc(options.TraceFunc)
		}
	}
	return vmm
}

// initGlobals initializes the global variables and returns the values.
func initGlobals(globals []compiler.Global, init map[string]interface{}) []interface{} {
	n := len(globals)
	if n == 0 {
		return nil
	}
	values := make([]interface{}, n)
	for i, global := range globals {
		if init == nil { // Program.
			if global.Value == nil {
				values[i] = reflect.New(global.Type).Interface()
			} else {
				values[i] = global.Value
			}
		} else { // Script and template.
			if global.Pkg == "main" {
				if value, ok := init[global.Name]; ok {
					if v, ok := value.(reflect.Value); ok {
						values[i] = v.Addr().Interface()
					} else {
						rv := reflect.New(global.Type).Elem()
						rv.Set(reflect.ValueOf(value))
						values[i] = rv.Addr().Interface()
					}
				} else {
					values[i] = reflect.New(global.Type).Interface()
				}
			} else {
				values[i] = global.Value
			}
		}
	}
	return values
}

// Disassemble disassembles a script.
func (s *Script) Disassemble(w io.Writer) (int64, error) {
	return compiler.DisassembleFunction(w, s.fn, s.globals)
}

// PackageLoader is implemented by package loaders. Given a package path, Load
// returns a *Package value or a package source as io.Reader.
//
// If the package does not exist it returns nil and nil.
// If the package exists but there was an error while loading the package, it
// returns nil and the error.
//
// If Load returns an io.Reader that implements io.Closer, the Close method
// will be called after a Read returns either EOF or an error.
type PackageLoader interface {
	Load(path string) (interface{}, error)
}

// MapStringLoader implements PackageLoader that returns the source of a
// package. Package paths and sources are respectively the keys and the values
// of the map.
type MapStringLoader map[string]string

func (r MapStringLoader) Load(path string) (interface{}, error) {
	if src, ok := r[path]; ok {
		return strings.NewReader(src), nil
	}
	return nil, nil
}

// CombinedLoaders combines more loaders in one loader. Load calls in order
// the Load methods of each loader and returns as soon as a loader returns
// a package.
type CombinedLoaders []PackageLoader

func (loaders CombinedLoaders) Load(path string) (interface{}, error) {
	for _, loader := range loaders {
		p, err := loader.Load(path)
		if p != nil || err != nil {
			return p, err
		}
	}
	return nil, nil
}

// Loaders returns a CombinedLoaders that combine loaders.
func Loaders(loaders ...PackageLoader) PackageLoader {
	return CombinedLoaders(loaders)
}

// Packages is a Loader that load packages from a map where the key is a
// package path and the value is a *Package value.
type Packages map[string]Package

func (pp Packages) Load(path string) (interface{}, error) {
	if p, ok := pp[path]; ok {
		return p, nil
	}
	return nil, nil
}

// PrintFunc returns a function that print its argument to the writer w with
// the same format used by the builtin print to print to the standard error.
// The returned function can be used for the PrintFunc option.
func PrintFunc(w io.Writer) vm.PrintFunc {
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

type MapPackage struct {
	// Package name.
	PkgName string
	// Package declarations.
	Declarations map[string]interface{}
}

func (p *MapPackage) Name() string {
	return p.PkgName
}

func (p *MapPackage) Lookup(declName string) interface{} {
	return p.Declarations[declName]
}

func (p *MapPackage) DeclarationNames() []string {
	declarations := make([]string, 0, len(p.Declarations))
	for name := range p.Declarations {
		declarations = append(declarations, name)
	}
	return declarations
}
