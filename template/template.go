// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"

	"scriggo"
	"scriggo/compiler"
	"scriggo/compiler/ast"
	"scriggo/runtime"
)

var (
	// ErrInvalidPath is returned from the Load function and a Reader when the
	// path argument is not valid.
	ErrInvalidPath = errors.New("scritto: invalid path")

	// ErrNotExist is returned from the Load function when the path does not
	// exist.
	ErrNotExist = errors.New("scritto: path does not exist")

	// ErrReadTooLarge is returned from the Load function when a limit is
	// exceeded reading a path.
	ErrReadTooLarge = errors.New("scritto: read too large")
)

// HTMStringer is implemented by values that are not escaped in HTML context.
type HTMLStringer interface {
	HTML() string
}

// CSSStringer is implemented by values that are not escaped in CSS context.
type CSSStringer interface {
	CSS() string
}

// JavaScriptStringer is implemented by values that are not escaped in
// JavaScript context.
type JavaScriptStringer interface {
	JavaScript() string
}

// HTML implements the HTMLStringer interface.
type HTML string

func (html HTML) HTML() string {
	return string(html)
}

// CSS implements the CSSStringer interface.
type CSS string

func (css CSS) CSS() string {
	return string(css)
}

// JavaScript implements the JavaScriptStringer interface.
type JavaScript string

func (js JavaScript) JavaScript() string {
	return string(js)
}

// Context indicates the type of source that has to be rendered and controls
// how to escape the values to render.
type Context int

const (
	ContextText       = Context(ast.ContextText)
	ContextHTML       = Context(ast.ContextHTML)
	ContextCSS        = Context(ast.ContextCSS)
	ContextJavaScript = Context(ast.ContextJavaScript)
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
	PrintFunc     runtime.PrintFunc
}

type Template struct {
	fn      *runtime.Function
	options *LoadOptions
	globals []compiler.Global
}

// CompilerError represents an error returned by the compiler.
type CompilerError interface {
	error
	Position() ast.Position
	Path() string
	Message() string
}

// Load loads a template given its path. Load calls the method Read of reader
// to read the files of the template. Package main declares constants, types,
// variables and functions that are accessible from the code in the template.
func Load(path string, reader Reader, main Package, ctx Context, options *LoadOptions) (*Template, error) {
	compileOpts := compiler.Options{}
	if options != nil {
		compileOpts.LimitMemorySize = options.LimitMemorySize
		compileOpts.TreeTransformer = options.TreeTransformer
	}
	var mainImporter scriggo.Packages
	if main != nil {
		mainImporter = scriggo.Packages{"main": main}
	}
	code, err := compiler.CompileTemplate(path, reader, mainImporter, ast.Context(ctx), compileOpts)
	if err != nil {
		if err == compiler.ErrInvalidPath {
			return nil, ErrInvalidPath
		}
		if err == compiler.ErrNotExist {
			return nil, ErrNotExist
		}
		return nil, err
	}
	return &Template{fn: code.Main, globals: code.Globals, options: options}, nil
}

var emptyVars = map[string]interface{}{}

// Render renders the template and write the output to out. vars contains the values for the
// variables of the main package.
func (t *Template) Render(out io.Writer, vars map[string]interface{}, options *RenderOptions) error {
	if options != nil {
		if options.MaxMemorySize > 0 && !t.options.LimitMemorySize {
			panic("scriggo: template not loaded with LimitMemorySize option")
		}
	}
	writeFunc := out.Write
	renderFunc := render
	uw := &urlEscaper{w: out}
	t.globals[0].Value = &out
	t.globals[1].Value = &writeFunc
	t.globals[2].Value = &renderFunc
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
	}
	return vm
}

// initGlobals initializes the global variables and returns the values. It
// panics if init is not valid.
//
// This function is a copy of the function in the scriggo package.
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
