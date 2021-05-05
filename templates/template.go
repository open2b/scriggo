// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package templates

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/compiler"
	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/fs"
	"github.com/open2b/scriggo/runtime"
)

// EnvStringer is like fmt.Stringer where the String method takes a runtime.Env
// parameter.
type EnvStringer interface {
	String(runtime.Env) string
}

// HTMLStringer is implemented by values that are not escaped in HTML context.
type HTMLStringer interface {
	HTML() HTML
}

// HTMLEnvStringer is like HTMLStringer where the HTML method takes a
// runtime.Env parameter.
type HTMLEnvStringer interface {
	HTML(runtime.Env) HTML
}

// CSSStringer is implemented by values that are not escaped in CSS context.
type CSSStringer interface {
	CSS() CSS
}

// CSSEnvStringer is like CSSStringer where the CSS method takes a runtime.Env
// parameter.
type CSSEnvStringer interface {
	CSS(runtime.Env) CSS
}

// JSStringer is implemented by values that are not escaped in JavaScript
// context.
type JSStringer interface {
	JS() JS
}

// JSEnvStringer is like JSStringer where the JS method takes a runtime.Env
// parameter.
type JSEnvStringer interface {
	JS(runtime.Env) JS
}

// JSONStringer is implemented by values that are not escaped in JSON context.
type JSONStringer interface {
	JSON() JSON
}

// JSONEnvStringer is like JSONStringer where the JSON method takes a
// runtime.Env parameter.
type JSONEnvStringer interface {
	JSON(runtime.Env) JSON
}

// MarkdownStringer is implemented by values that are not escaped in Markdown
// context.
type MarkdownStringer interface {
	Markdown() Markdown
}

// MarkdownEnvStringer is like MarkdownStringer where the Markdown method
// takes a runtime.Env parameter.
type MarkdownEnvStringer interface {
	Markdown(runtime.Env) Markdown
}

// Format types.
type (
	HTML     string // the html type in templates.
	CSS      string // the css type in templates.
	JS       string // the js type in templates.
	JSON     string // the json type in templates.
	Markdown string // the markdown type in templates.
)

// A Format represents a content format.
type Format = ast.Format

const (
	FormatText     = ast.FormatText
	FormatHTML     = ast.FormatHTML
	FormatCSS      = ast.FormatCSS
	FormatJS       = ast.FormatJS
	FormatJSON     = ast.FormatJSON
	FormatMarkdown = ast.FormatMarkdown
)

type BuildOptions struct {
	DisallowGoStmt  bool
	TreeTransformer func(*ast.Tree) error // if not nil transforms tree after parsing.

	// MarkdownConverter converts a Markdown source code to HTML.
	MarkdownConverter Converter

	// Globals declares constants, types, variables and functions that are
	// accessible from the code in the template.
	Globals Declarations

	// Packages is a PackageLoader that makes precompiled packages available
	// in the template through the 'import' statement.
	//
	// Note that an import statement refers to a precompiled package read from
	// Packages if its path has no extension.
	//
	//     {%  import  "my/package"   %}    Import a precompiled package.
	//     {%  import  "my/file.html  %}    Import a template file.
	//
	Packages scriggo.PackageLoader
}

// Declarations.
type Declarations map[string]interface{}

// Converter is implemented by format converters.
type Converter func(src []byte, out io.Writer) error

type RunOptions struct {
	Context   context.Context
	PrintFunc runtime.PrintFunc
}

type Template struct {
	fn          *runtime.Function
	types       runtime.Types
	globals     []compiler.Global
	mdConverter Converter
}

// CompilerError represents an error returned by the compiler.
type CompilerError interface {
	error
	Position() ast.Position
	Path() string
	Message() string
}

// FormatFS is the interface implemented by a file system that can determine
// the file format from a path name.
type FormatFS interface {
	fs.FS
	Format(name string) (Format, error)
}

// formatTypes contains the format types added to the universe block.
var formatTypes = map[ast.Format]reflect.Type{
	ast.FormatHTML:     reflect.TypeOf((*HTML)(nil)).Elem(),
	ast.FormatCSS:      reflect.TypeOf((*CSS)(nil)).Elem(),
	ast.FormatJS:       reflect.TypeOf((*JS)(nil)).Elem(),
	ast.FormatJSON:     reflect.TypeOf((*JSON)(nil)).Elem(),
	ast.FormatMarkdown: reflect.TypeOf((*Markdown)(nil)).Elem(),
}

// Build builds the named template file rooted at the given file system.
//
// If fsys implements FormatFS, the file format is read from its Format
// method, otherwise it depends on the file name extension
//
//   HTML       : .html
//   CSS        : .css
//   JavaScript : .js
//   JSON       : .json
//   Markdown   : .md .mkd .mkdn .mdown .markdown
//   Text       : all other extensions
//
// If the named file does not exist, Build returns an error satisfying
// errors.Is(err, fs.ErrNotExist). If a compilation error occurs, it returns
// a CompilerError error.
func Build(fsys fs.FS, name string, options *BuildOptions) (*Template, error) {
	co := compiler.Options{
		FormatTypes: formatTypes,
	}
	var mdConverter Converter
	if options != nil {
		co.Globals = compiler.Declarations(options.Globals)
		co.TreeTransformer = options.TreeTransformer
		co.DisallowGoStmt = options.DisallowGoStmt
		co.Packages = options.Packages
		mdConverter = options.MarkdownConverter
	}
	co.Renderer = newRenderer(nil, mdConverter)
	code, err := compiler.BuildTemplate(fsys, name, co)
	if err != nil {
		return nil, err
	}
	return &Template{fn: code.Main, types: code.Types, globals: code.Globals, mdConverter: mdConverter}, nil
}

// Run runs the template and write the rendered code to out. vars contains
// the values of the global variables.
func (t *Template) Run(out io.Writer, vars map[string]interface{}, options *RunOptions) error {
	if out == nil {
		return errors.New("invalid nil out")
	}
	vm := runtime.NewVM()
	if options != nil {
		if options.Context != nil {
			vm.SetContext(options.Context)
		}
		if options.PrintFunc != nil {
			vm.SetPrint(options.PrintFunc)
		}
	}
	renderer := newRenderer(out, t.mdConverter)
	vm.SetRenderer(renderer)
	_, err := vm.Run(t.fn, t.types, ifacesToRvalues(initGlobalVariables(t.globals, vars)))
	return err
}

// REVIEW: remove.
func ifacesToRvalues(ifaces []interface{}) []reflect.Value {
	rvs := make([]reflect.Value, len(ifaces))
	for i, iface := range ifaces {
		rv := reflect.ValueOf(iface)
		rvs[i] = rv
	}
	return rvs
}

// MustRun is like Run but panics if the execution fails.
func (t *Template) MustRun(out io.Writer, vars map[string]interface{}, options *RunOptions) {
	err := t.Run(out, vars, options)
	if err != nil {
		panic(err)
	}
}

// Disassemble disassembles a template and returns its assembly code.
//
// n determines the maximum length, in runes, of a disassembled text:
//
//   n > 0: at most n runes; leading and trailing white space are removed
//   n == 0: no text
//   n < 0: all text
//
func (t *Template) Disassemble(n int) []byte {
	assemblies := compiler.Disassemble(t.fn, t.globals, n)
	return assemblies["main"]
}

// UsedVars returns the names of the global variables used in the template.
func (t *Template) UsedVars() []string {
	vars := make([]string, len(t.globals))
	for i, global := range t.globals {
		vars[i] = global.Name
	}
	sort.Strings(vars)
	return vars
}

var emptyInit = map[string]interface{}{}

// initGlobalVariables initializes the global variables and returns their
// values. It panics if init is not valid.
//
// This function is a copy of the function in the scripts package.
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
