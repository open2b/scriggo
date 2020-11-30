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
	"github.com/open2b/scriggo/runtime"
)

var (
	// ErrInvalidPath is returned from the Load function and a FileReader when
	// the path argument is not valid.
	ErrInvalidPath = errors.New("scriggo: invalid path")

	// ErrNotExist is returned from the Load function when the path does not
	// exist.
	ErrNotExist = errors.New("scriggo: path does not exist")

	// ErrReadTooLarge is returned from the Load function when a limit is
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

// A Language represents a source language.
type Language int

const (
	LanguageText Language = iota
	LanguageHTML
	LanguageCSS
	LanguageJS
	LanguageJSON
)

func (language Language) String() string {
	return ast.Language(language).String()
}

type LoadOptions struct {
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

type RenderOptions struct {
	Context   context.Context
	PrintFunc runtime.PrintFunc
}

type Template struct {
	fn      *runtime.Function
	typeof  runtime.TypeOfFunc
	globals []compiler.Global
}

// CompilerError represents an error returned by the compiler.
type CompilerError interface {
	error
	Position() ast.Position
	Path() string
	Message() string
}

// Load loads a template given its file name. Load calls the method ReadFile of
// files to read the files of the template.
func Load(name string, files FileReader, lang Language, options *LoadOptions) (*Template, error) {
	co := compiler.Options{ShowFunc: show}
	if options != nil {
		co.Globals = compiler.Declarations(options.Globals)
		co.TreeTransformer = options.TreeTransformer
		co.DisallowGoStmt = options.DisallowGoStmt
		co.Packages = options.Packages
	}
	code, err := compiler.CompileTemplate(name, files, ast.Language(lang), co)
	if err != nil {
		if err == compiler.ErrInvalidPath {
			return nil, ErrInvalidPath
		}
		if err == compiler.ErrNotExist {
			return nil, ErrNotExist
		}
		return nil, err
	}
	return &Template{fn: code.Main, typeof: code.TypeOf, globals: code.Globals}, nil
}

// Render renders the template and write the output to out. vars contains the
// values of the global variables.
func (t *Template) Render(out io.Writer, vars map[string]interface{}, options *RenderOptions) error {
	writeFunc := out.Write
	showFunc := show
	uw := &urlEscaper{w: out}
	t.globals[0].Value = &out
	t.globals[1].Value = &writeFunc
	t.globals[2].Value = &showFunc
	t.globals[3].Value = &uw
	vm := newVM(options)
	_, err := vm.Run(t.fn, t.typeof, initGlobalVariables(t.globals, vars))
	return err
}

// MustRender is like Render but panics if the rendering fails.
func (t *Template) MustRender(out io.Writer, vars map[string]interface{}, options *RenderOptions) {
	err := t.Render(out, vars, options)
	if err != nil {
		panic(err)
	}
}

// Disassemble disassembles a template.
func (t *Template) Disassemble(w io.Writer) (int64, error) {
	return compiler.DisassembleFunction(w, t.fn, t.globals)
}

// Vars returns the names of the template builtin variables that are used in
// the template.
func (t *Template) Vars() []string {
	vars := make([]string, len(t.globals)-4)
	for i, global := range t.globals[4:] {
		vars[i] = global.Name
	}
	sort.Strings(vars)
	return vars
}

// newVM returns a new vm with the given options.
func newVM(options *RenderOptions) *runtime.VM {
	vm := runtime.NewVM()
	if options != nil {
		if options.Context != nil {
			vm.SetContext(options.Context)
		}
		if options.PrintFunc != nil {
			vm.SetPrint(options.PrintFunc)
		}
	}
	return vm
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
