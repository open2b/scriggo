// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"reflect"

	"scrigo/internal/compiler"
	"scrigo/internal/compiler/ast"
	"scrigo/vm"
)

const (
	LimitMemorySize  Option = 1 << iota // limit allocable memory size.
	AllowShebangLine                    // allow shebang line; only for scripts.
)

type Option int

type PredefinedPackage = compiler.PredefinedPackage

type PredefinedConstant = compiler.PredefinedConstant

// Constant returns a constant, given its type and value, that can be used as
// a declaration in a predefined package.
//
// For untyped constants the type is nil.
func Constant(typ reflect.Type, value interface{}) PredefinedConstant {
	return compiler.Constant(typ, value)
}

type Program struct {
	fn      *vm.Function
	globals []compiler.Global
	options Option
}

// TODO(Gianluca): add documentation.
type PackageImporter interface{}

// LoadProgram loads a program, reading "/main" from packages.
func LoadProgram(packages []PackageImporter, options Option) (*Program, error) {

	// Converts []PackageImporter in []compiler.PackageImporter.
	// Type alias is not reccomended as it would make documentation unclear.
	compilerPkgs := make([]compiler.PackageImporter, len(packages))
	for i := range packages {
		compilerPkgs[i] = packages[i]
	}

	tree, deps, predefinedPackages, err := compiler.ParseProgram(compilerPkgs)
	if err != nil {
		return nil, err
	}

	opts := &compiler.Options{
		IsPackage: true,
	}
	tci, err := compiler.Typecheck(opts, tree, nil, predefinedPackages, deps, nil)
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

	pkgMain := compiler.EmitPackageMain(tree.Nodes[0].(*ast.Package), predefinedPackages, typeInfos, tci["/main"].IndirectVars, alloc)
	globals := make([]compiler.Global, len(pkgMain.Globals))
	for i, global := range pkgMain.Globals {
		globals[i].Pkg = global.Pkg
		globals[i].Name = global.Name
		globals[i].Type = global.Type
		globals[i].Value = global.Value
	}
	return &Program{fn: pkgMain.Main, globals: globals, options: options}, nil
}

// Options returns the options with which the program has been loaded.
func (p *Program) Options() Option {
	return p.options
}

type RunOptions struct {
	Context       context.Context
	MaxMemorySize int
	DontPanic     bool
	TraceFunc     vm.TraceFunc
}

// Run starts the program and waits for it to complete.
func (p *Program) Run(options RunOptions) error {
	vmm := vm.New()
	if options.Context != nil {
		vmm.SetContext(options.Context)
	}
	if options.MaxMemorySize > 0 {
		if p.options&LimitMemorySize == 0 {
			return errors.New("program not loaded with LimitMemorySize option")
		}
		vmm.SetMaxMemory(options.MaxMemorySize)
	}
	if options.DontPanic {
		vmm.SetDontPanic(true)
	}
	if options.TraceFunc != nil {
		vmm.SetTraceFunc(options.TraceFunc)
	}
	if n := len(p.globals); n > 0 {
		globals := make([]interface{}, n)
		for i, global := range p.globals {
			if global.Value == nil {
				globals[i] = reflect.New(global.Type).Interface()
			} else {
				globals[i] = global.Value
			}
		}
		vmm.SetGlobals(globals)
	}
	_, err := vmm.Run(p.fn)
	return err
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
		return 0, errors.New("package path does not exist")
	}
	n, err := io.WriteString(w, asm)
	return int64(n), err
}

// Global represents a global variable with a package, name, type (only for
// not predefined globals) and value (only for predefined globals). Value, if
// present, must be a pointer to the variable value.
type Global struct {
	Name string
	Type reflect.Type
}

type Script struct {
	fn      *vm.Function
	globals []Global
	options Option
}

// LoadScript loads a script from a reader.
func LoadScript(src io.Reader, main *PredefinedPackage, options Option) (*Script, error) {

	alloc := options&LimitMemorySize != 0
	shebang := options&AllowShebangLine != 0

	// Parsing.
	buf, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	tree, _, err := compiler.ParseSource(buf, false, shebang)
	if err != nil {
		return nil, err
	}

	opts := &compiler.Options{
		IsPackage: false,
	}
	tci, err := compiler.Typecheck(opts, tree, main, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	// Emitting.
	// TODO(Gianluca): pass "main" to emitter.
	// main contains user defined variables.
	emitter := compiler.NewEmitter(nil, tci["main"].TypeInfo, tci["main"].IndirectVars)
	emitter.SetAlloc(alloc)
	fn := compiler.NewFunction("main", "main", reflect.FuncOf(nil, nil, false))
	emitter.CurrentFunction = fn
	emitter.FB = compiler.NewBuilder(emitter.CurrentFunction)
	emitter.FB.SetAlloc(alloc)
	emitter.FB.EnterScope()
	compiler.AddExplicitReturn(tree)
	emitter.EmitNodes(tree.Nodes)
	emitter.FB.ExitScope()
	emitter.FB.End()

	return &Script{fn: emitter.CurrentFunction, options: options}, nil
}

// Options returns the options with which the script has been loaded.
func (s *Script) Options() Option {
	return s.options
}

// Run starts a script with the specified global variables and waits for it to
// complete.
func (s *Script) Run(vars map[string]interface{}, options RunOptions) error {
	vmm := vm.New()
	if options.Context != nil {
		vmm.SetContext(options.Context)
	}
	if options.MaxMemorySize > 0 {
		if s.options&LimitMemorySize == 0 {
			return errors.New("script not loaded with LimitMemorySize option")
		}
		vmm.SetMaxMemory(options.MaxMemorySize)
	}
	if options.DontPanic {
		vmm.SetDontPanic(true)
	}
	if options.TraceFunc != nil {
		vmm.SetTraceFunc(options.TraceFunc)
	}
	if n := len(s.globals); n > 0 {
		globals := make([]interface{}, n)
		for i, global := range s.globals {
			if value, ok := vars[global.Name]; ok {
				if v, ok := value.(reflect.Value); ok {
					globals[i] = v.Addr().Interface()
				} else {
					rv := reflect.New(global.Type).Elem()
					rv.Set(reflect.ValueOf(v))
					globals[i] = rv.Interface()
				}
			} else {
				globals[i] = reflect.New(global.Type).Interface()
			}
		}
		vmm.SetGlobals(globals)
	}
	_, err := vmm.Run(s.fn)
	return err
}

// Disassemble disassembles a script.
func (s *Script) Disassemble(w io.Writer) (int64, error) {
	return compiler.DisassembleFunction(w, s.fn)
}
