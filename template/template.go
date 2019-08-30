// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"context"
	"io"
	"reflect"

	"scriggo"
	"scriggo/ast"
	"scriggo/internal/compiler"
	"scriggo/runtime"
)

// Context indicates the type of source that has to be rendered and controls
// how to escape the values to render.
type Context int

const (
	ContextText              = Context(ast.ContextText)
	ContextHTML              = Context(ast.ContextHTML)
	ContextTag               = Context(ast.ContextTag)
	ContextAttribute         = Context(ast.ContextAttribute)
	ContextUnquotedAttribute = Context(ast.ContextUnquotedAttribute)
	ContextCSS               = Context(ast.ContextCSS)
	ContextCSSString         = Context(ast.ContextCSSString)
	ContextJavaScript        = Context(ast.ContextJavaScript)
	ContextJavaScriptString  = Context(ast.ContextJavaScriptString)
)

func (ctx Context) String() string {
	return ast.Context(ctx).String()
}

type LoadOptions struct {
	LimitMemorySize bool
	TreeTransformer func(*ast.Tree) error // if not nil transforms tree after parsing.
}

type RenderOptions struct {
	Context       context.Context
	MaxMemorySize int
	DontPanic     bool
	RenderFunc    RenderFunc
	PrintFunc     runtime.PrintFunc
	TraceFunc     runtime.TraceFunc
}

type Template struct {
	fn      *runtime.Function
	options *LoadOptions
	globals []compiler.Global
}

// Load loads a template given its path. Load calls the method Read of reader
// to read the files of the template. Package main declares constants, types,
// variables and functions that are accessible from the code in the template.
// Context is the context in which the code is executed.
func Load(path string, reader Reader, main scriggo.Package, ctx Context, options *LoadOptions) (*Template, error) {
	tree, err := compiler.ParseTemplate(path, reader, ast.Context(ctx))
	if err != nil {
		return nil, err
	}
	if options == nil {
		options = &LoadOptions{}
	}
	checkerOpts := compiler.CheckerOptions{
		SyntaxType:   compiler.TemplateSyntax,
		AllowNotUsed: true,
	}
	var pkgs scriggo.Packages
	if main != nil {
		pkgs = scriggo.Packages{"main": main}
	}
	if options.TreeTransformer != nil {
		err := options.TreeTransformer(tree)
		if err != nil {
			return nil, err
		}
	}
	tci, err := compiler.Typecheck(tree, pkgs, checkerOpts)
	if err != nil {
		return nil, err
	}
	typeInfos := map[ast.Node]*compiler.TypeInfo{}
	for _, pkgInfos := range tci {
		for node, ti := range pkgInfos.TypeInfos {
			typeInfos[node] = ti
		}
	}
	emitterOpts := compiler.EmitterOptions{
		MemoryLimit: options.LimitMemorySize,
	}
	code := compiler.EmitTemplate(tree, typeInfos, tci["main"].IndirectVars, emitterOpts)
	return &Template{fn: code.Main, globals: code.Globals, options: options}, nil
}

// A RenderFunc renders value in the context ctx and writes the result to out.
// A RenderFunc is called by the Render method to render the value resulting
// from the evaluation of an expression between "{{" and "}}".
type RenderFunc func(env *runtime.Env, out io.Writer, value interface{}, ctx Context)

var emptyVars = map[string]interface{}{}

// Render renders the template and write the output to out. vars contains the values for the
// variables of the main package.
func (t *Template) Render(out io.Writer, vars map[string]interface{}, options *RenderOptions) error {
	render := DefaultRenderFunc
	if options != nil {
		if options.MaxMemorySize > 0 && !t.options.LimitMemorySize {
			panic("scrigoo: template not loaded with LimitMemorySize option")
		}
		if options.RenderFunc != nil {
			render = options.RenderFunc
		}
	}
	write := out.Write
	uw := &urlEscaper{w: out}
	t.globals[0].Value = &out
	t.globals[1].Value = &write
	t.globals[2].Value = &render
	t.globals[3].Value = &uw
	if vars == nil {
		vars = emptyVars
	}
	vm := newVM(options)
	_, err := vm.Run(t.fn, initGlobals(t.globals, vars))
	return err
}

// Options returns the options with which the template has been loaded.
func (t *Template) Options() *LoadOptions {
	return t.options
}

// Disassemble disassembles a template.
func (t *Template) Disassemble(w io.Writer) (int64, error) {
	return compiler.DisassembleFunction(w, t.fn, t.globals)
}

// newVM returns a new vm with the given options.
func newVM(options *RenderOptions) *runtime.VM {
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

// initGlobals initializes the global variables and returns the values.
func initGlobals(globals []compiler.Global, init map[string]interface{}) []interface{} {
	n := len(globals)
	if n == 0 {
		return nil
	}
	values := make([]interface{}, n)
	for i, global := range globals {
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
	return values
}
