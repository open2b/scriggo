// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"context"
	"errors"
	"io"
	"reflect"

	"scrigo"
	"scrigo/internal/compiler"
	"scrigo/internal/compiler/ast"
	"scrigo/vm"
)

type HTML string

// Context indicates the type of source that has to be rendered and controls
// how to escape the values to render.
type Context int

const (
	ContextText       Context = Context(ast.ContextText)
	ContextHTML       Context = Context(ast.ContextHTML)
	ContextCSS        Context = Context(ast.ContextCSS)
	ContextJavaScript Context = Context(ast.ContextJavaScript)
)

type LoadOption int

const (
	LimitMemorySize LoadOption = 1 << iota
)

type RenderOptions struct {
	Context       context.Context
	MaxMemorySize int
	DontPanic     bool
	Print         func(interface{})
	TraceFunc     vm.TraceFunc
}

type Template struct {
	main    *scrigo.Package
	fn      *vm.Function
	options LoadOption
}

// Load loads a template given its path. Load calls the method Read of reader
// to read the files of the template. Package main declares constants, types,
// variables and functions that are accessible from the code in the template.
// Context is the context in which the code is executed.
func Load(path string, reader Reader, main *scrigo.Package, ctx Context, options LoadOption) (*Template, error) {

	tree, err := compiler.ParseTemplate(path, reader, main, ast.Context(ctx))
	if err != nil {
		return nil, err
	}

	opts := &compiler.Options{
		IsPackage: false,
	}
	tci, err := compiler.Typecheck(opts, tree, map[string]*compiler.Package{"main": main}, nil, nil)
	if err != nil {
		return nil, err
	}

	alloc := options&LimitMemorySize != 0

	// TODO(Gianluca): pass "main" and "builtins" to emitter.
	// main contains user defined variables, while builtins contains template builtins.
	// // define something like "emitterBuiltins" in order to avoid converting at every compilation.

	mainFn := compiler.EmitSingle(tree, tci["main"].TypeInfo, tci["main"].IndirectVars, alloc)

	return &Template{main: main, fn: mainFn}, nil
}

// Render renders the template and write the output to out. vars contains the values for the
// variables of the main package.
func (t *Template) Render(out io.Writer, vars map[string]reflect.Value, options RenderOptions) error {
	// TODO: implement globals
	vmm := vm.New()
	if options.Context != nil {
		vmm.SetContext(options.Context)
	}
	if options.MaxMemorySize > 0 {
		if t.options&LimitMemorySize == 0 {
			return errors.New("program not loaded with LimitMemorySize option")
		}
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
	_, err := vmm.Run(t.fn)
	return err
}

// Options returns the options with which the template has been loaded.
func (t *Template) Options() LoadOption {
	return t.options
}

// Disassemble disassembles a template.
func (t *Template) Disassemble(w io.Writer) (int64, error) {
	return compiler.DisassembleFunction(w, t.fn)
}
