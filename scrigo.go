// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"

	"scrigo/internal/compiler"
	"scrigo/internal/compiler/ast"
	"scrigo/vm"
)

const (
	LimitMemorySize  LoadOption = 1 << iota // limit allocable memory size.
	DisallowGoStmt                          // disallow "go" statement.
	AllowShebangLine                        // allow shebang line; only for scripts.
)

type LoadOption int

type Package = compiler.Package

type ConstantValue = compiler.ConstantValue

// Constant returns a constant, given its type and value, that can be used as
// a declaration in a predefined package.
//
// For untyped constants the type is nil.
func Constant(typ reflect.Type, value interface{}) ConstantValue {
	return compiler.Constant(typ, value)
}

type Program struct {
	fn      *vm.Function
	globals []vm.Global
	options LoadOption
}

// LoadProgram loads a program, reading package "main" from packages.
func LoadProgram(packages PackageLoader, options LoadOption) (*Program, error) {

	tree, deps, predefined, err := compiler.ParseProgram(packages)
	if err != nil {
		return nil, err
	}

	opts := &compiler.Options{
		IsPackage: true,
	}
	if options&DisallowGoStmt != 0 {
		opts.DisallowGoStmt = true
	}
	tci, err := compiler.Typecheck(opts, tree, nil, predefined, deps)
	if err != nil {
		return nil, err
	}
	typeInfos := map[ast.Node]*compiler.TypeInfo{}
	for _, pkgInfos := range tci {
		for node, ti := range pkgInfos.TypeInfo {
			typeInfos[node] = ti
		}
	}
	alloc := options&LimitMemorySize != 0

	pkgMain := compiler.EmitPackageMain(tree.Nodes[0].(*ast.Package), typeInfos, tci["main"].IndirectVars, alloc)

	return &Program{fn: pkgMain.Main, globals: pkgMain.Globals, options: options}, nil
}

// Options returns the options with which the program has been loaded.
func (p *Program) Options() LoadOption {
	return p.options
}

type RunOptions struct {
	Context       context.Context
	MaxMemorySize int
	DontPanic     bool
	Print         func(interface{})
	TraceFunc     vm.TraceFunc
}

// Run starts the program and waits for it to complete.
//
// Panics if the option MaxMemorySize is greater than zero but the program has
// not been loaded with option LimitMemorySize.
func (p *Program) Run(options RunOptions) error {
	if options.MaxMemorySize > 0 {
		if p.options&LimitMemorySize == 0 {
			panic("scrigo: program not loaded with LimitMemorySize option")
		}
	}
	vmm := newVM(p.globals, nil, options)
	_, err := vmm.Run(p.fn)
	return err
}

// Start starts the program in a new goroutine and returns its virtual machine
// execution environment.
//
// Panics if the option MaxMemorySize is greater than zero but the program has
// not been loaded with option LimitMemorySize.
func (p *Program) Start(options RunOptions) *vm.Env {
	if options.MaxMemorySize > 0 {
		if p.options&LimitMemorySize == 0 {
			panic("scrigo: program not loaded with LimitMemorySize option")
		}
	}
	vmm := newVM(p.globals, nil, options)
	go vmm.Run(p.fn)
	return vmm.Env()
}

// Disassemble disassembles the package with the given path. Predefined
// packages can not be disassembled.
func (p *Program) Disassemble(w io.Writer, pkgPath string) (int64, error) {
	packages, err := compiler.Disassemble(p.fn)
	if err != nil {
		return 0, err
	}
	asm, ok := packages[pkgPath]
	if !ok {
		return 0, errors.New("scrigo: package path does not exist")
	}
	n, err := io.WriteString(w, asm)
	return int64(n), err
}

type Script struct {
	fn      *vm.Function
	globals []vm.Global
	options LoadOption
}

// LoadScript loads a script from a reader.
func LoadScript(src io.Reader, loader PackageLoader, options LoadOption) (*Script, error) {

	alloc := options&LimitMemorySize != 0
	shebang := options&AllowShebangLine != 0

	tree, packages, err := compiler.ParseScript(src, loader, shebang)
	if err != nil {
		return nil, err
	}

	opts := &compiler.Options{
		IsPackage: false,
	}
	if options&DisallowGoStmt != 0 {
		opts.DisallowGoStmt = true
	}
	tci, err := compiler.Typecheck(opts, tree, packages, nil, nil)
	if err != nil {
		return nil, err
	}

	mainFn := compiler.EmitSingle(tree, tci["main"].TypeInfo, tci["main"].IndirectVars, alloc)

	return &Script{fn: mainFn, globals: mainFn.Globals, options: options}, nil
}

// Options returns the options with which the script has been loaded.
func (s *Script) Options() LoadOption {
	return s.options
}

var emptyInit = map[string]interface{}{}

// Run starts the script, with initialization values for the global variables,
// and waits for it to complete.
//
// Panics if the option MaxMemorySize is greater than zero but the script has
// not been loaded with option LimitMemorySize.
func (s *Script) Run(init map[string]interface{}, options RunOptions) error {
	if options.MaxMemorySize > 0 {
		if s.options&LimitMemorySize == 0 {
			panic("scrigo: script not loaded with LimitMemorySize option")
		}
	}
	if init == nil {
		init = emptyInit
	}
	vmm := newVM(s.globals, init, options)
	_, err := vmm.Run(s.fn)
	return err
}

// Start starts the script in a new goroutine, with initialization values for
// the global variables, and returns its virtual machine execution environment.
//
// Panics if the option MaxMemorySize is greater than zero but the script has
// not been loaded with option LimitMemorySize.
func (s *Script) Start(init map[string]interface{}, options RunOptions) *vm.Env {
	if options.MaxMemorySize > 0 {
		if s.options&LimitMemorySize == 0 {
			panic("scrigo: script not loaded with LimitMemorySize option")
		}
	}
	vmm := newVM(s.globals, init, options)
	go vmm.Run(s.fn)
	return vmm.Env()
}

// newVM returns a new vm with the given options.
func newVM(globals []vm.Global, init map[string]interface{}, options RunOptions) *vm.VM {
	vmm := vm.New()
	if options.Context != nil {
		vmm.SetContext(options.Context)
	}
	if options.MaxMemorySize > 0 {
		vmm.SetMaxMemory(options.MaxMemorySize)
	}
	if options.DontPanic {
		vmm.SetDontPanic(true)
	}
	if options.Print != nil {
		vmm.SetPrint(options.Print)
	}
	if options.TraceFunc != nil {
		vmm.SetTraceFunc(options.TraceFunc)
	}
	if n := len(globals); n > 0 {
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
							rv.Set(reflect.ValueOf(v))
							values[i] = rv.Interface()
						}
					} else {
						values[i] = reflect.New(global.Type).Interface()
					}
				} else {
					values[i] = global.Value
				}
			}
		}
		vmm.SetGlobals(values)
	}
	return vmm
}

// Disassemble disassembles a script.
func (s *Script) Disassemble(w io.Writer) (int64, error) {
	return compiler.DisassembleFunction(w, s.fn)
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
type Packages map[string]*Package

func (pp Packages) Load(path string) (interface{}, error) {
	if p, ok := pp[path]; ok {
		return p, nil
	}
	return nil, nil
}
