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
	"path"
	"reflect"
	"sort"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/compiler"
	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/runtime"
)

var (
	// ErrInvalidPath is returned from the Build function and a FileReader when
	// the path argument is not valid.
	ErrInvalidPath = errors.New("scriggo: invalid path")

	// ErrNotExist is returned from the Build function when the path does not
	// exist.
	ErrNotExist = errors.New("scriggo: path does not exist")

	// ErrReadTooLarge is returned from the Build function when a limit is
	// exceeded reading a path.
	ErrReadTooLarge = errors.New("scriggo: read too large")
)

// EnvStringer is like fmt.Stringer where the String method takes a runtime.Env
// parameter.
type EnvStringer interface {
	String(runtime.Env) string
}

// HTMLStringer is implemented by values that are not escaped in HTML context.
type HTMLStringer interface {
	HTML() string
}

// HTMLEnvStringer is like HTMLStringer where the HTML method takes a
// runtime.Env parameter.
type HTMLEnvStringer interface {
	HTML(runtime.Env) string
}

// CSSStringer is implemented by values that are not escaped in CSS context.
type CSSStringer interface {
	CSS() string
}

// CSSEnvStringer is like CSSStringer where the CSS method takes a runtime.Env
// parameter.
type CSSEnvStringer interface {
	CSS(runtime.Env) string
}

// JSStringer is implemented by values that are not escaped in JavaScript
// context.
type JSStringer interface {
	JS() string
}

// JSEnvStringer is like JSStringer where the JS method takes a runtime.Env
// parameter.
type JSEnvStringer interface {
	JS(runtime.Env) string
}

// JSONStringer is implemented by values that are not escaped in JSON context.
type JSONStringer interface {
	JSON() string
}

// JSONEnvStringer is like JSONStringer where the JSON method takes a
// runtime.Env parameter.
type JSONEnvStringer interface {
	JSON(runtime.Env) string
}

// MarkdownStringer is implemented by values that are not escaped in Markdown
// context.
type MarkdownStringer interface {
	Markdown() string
}

// MarkdownEnvStringer is like MarkdownStringer where the Markdown method
// takes a runtime.Env parameter.
type MarkdownEnvStringer interface {
	Markdown(runtime.Env) string
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

// JS implements the JSStringer interface.
type JS string

func (js JS) JS() string {
	return string(js)
}

// JSON implements the JSONStringer interface.
type JSON string

func (json JSON) JSON() string {
	return string(json)
}

// Markdown implements the MarkdownStringer interface.
type Markdown string

func (md Markdown) Markdown() string {
	return string(md)
}

// A Format represents a content format.
type Format int

const (
	FormatText Format = iota
	FormatHTML
	FormatCSS
	FormatJS
	FormatJSON
	FormatMarkdown
)

func (format Format) String() string {
	return ast.Format(format).String()
}

type BuildOptions struct {
	DisallowGoStmt  bool
	TreeTransformer func(*ast.Tree) error // if not nil transforms tree after parsing.

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

	// MarkdownConverter converts a Markdown source code to HTML.
	MarkdownConverter Converter
}

type Template struct {
	fn      *runtime.Function
	types   runtime.Types
	globals []compiler.Global
}

// CompilerError represents an error returned by the compiler.
type CompilerError interface {
	error
	Position() ast.Position
	Path() string
	Message() string
}

type fileFormatReader struct {
	r FileReader
}

func (f fileFormatReader) ReadFile(name string) ([]byte, ast.Format, error) {
	src, err := f.r.ReadFile(name)
	if err != nil {
		return nil, 0, err
	}
	if files, ok := f.r.(interface {
		Format(string) (Format, error)
	}); ok {
		format, err := files.Format(name)
		if err != nil {
			return nil, 0, err
		}
		if format < FormatText || format > FormatMarkdown {
			return nil, 0, fmt.Errorf("unkonwn format %d", format)
		}
		return src, ast.Format(format), nil
	}
	format := ast.FormatText
	switch path.Ext(name) {
	case ".html":
		format = ast.FormatHTML
	case ".css":
		format = ast.FormatCSS
	case ".js":
		format = ast.FormatJS
	case ".json":
		format = ast.FormatJSON
	case ".md", ".markdown":
		format = ast.FormatMarkdown
	}
	return src, format, nil
}

// Build builds a template given its file name. Build calls the method
// ReadFile of files to read the files of the template.
//
// The format of the file depends on the extension of name or, if files has
// the method 'Format(string) (Format, error)', Build gets the format
// from this method. If this method returns an error, this error is returned.
func Build(name string, files FileReader, options *BuildOptions) (*Template, error) {
	co := compiler.Options{Renderer: buildRenderer{}}
	if options != nil {
		co.Globals = compiler.Declarations(options.Globals)
		co.TreeTransformer = options.TreeTransformer
		co.DisallowGoStmt = options.DisallowGoStmt
		co.Packages = options.Packages
	}
	code, err := compiler.BuildTemplate(name, fileFormatReader{files}, co)
	if err != nil {
		if err == compiler.ErrInvalidPath {
			return nil, ErrInvalidPath
		}
		if err == compiler.ErrNotExist {
			return nil, ErrNotExist
		}
		return nil, err
	}
	return &Template{fn: code.Main, types: code.Types, globals: code.Globals}, nil
}

// Run runs the template and write the rendered code to out. vars contains
// the values of the global variables.
func (t *Template) Run(out io.Writer, vars map[string]interface{}, options *RunOptions) error {
	vm := runtime.NewVM()
	var mdConverter Converter
	if options != nil {
		if options.Context != nil {
			vm.SetContext(options.Context)
		}
		if options.PrintFunc != nil {
			vm.SetPrint(options.PrintFunc)
		}
		mdConverter = options.MarkdownConverter
	}
	renderer := newRenderer(out, ast.Format(t.fn.Format), mdConverter)
	vm.SetRenderer(renderer)
	_, err := vm.Run(t.fn, t.types, initGlobalVariables(t.globals, vars))
	return err
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

// Vars returns the names of the template builtin variables that are used in
// the template.
func (t *Template) Vars() []string {
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
