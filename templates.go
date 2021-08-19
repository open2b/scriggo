// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scriggo

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"reflect"
	"sort"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/compiler"
	"github.com/open2b/scriggo/internal/runtime"
	"github.com/open2b/scriggo/native"
)

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

// String returns the name of the format.
func (format Format) String() string {
	return ast.Format(format).String()
}

// Converter is implemented by format converters.
type Converter func(src []byte, out io.Writer) error

// Template is a template compiled with the BuildTemplate function.
type Template struct {
	fn          *runtime.Function
	typeof      runtime.TypeOfFunc
	globals     []compiler.Global
	mdConverter Converter
}

// FormatFS is the interface implemented by a file system that can determine
// the file format from a path name.
type FormatFS interface {
	fs.FS
	Format(name string) (Format, error)
}

// formatTypes contains the format types added to the universe block.
var formatTypes = map[ast.Format]reflect.Type{
	ast.FormatHTML:     reflect.TypeOf((*native.HTML)(nil)).Elem(),
	ast.FormatCSS:      reflect.TypeOf((*native.CSS)(nil)).Elem(),
	ast.FormatJS:       reflect.TypeOf((*native.JS)(nil)).Elem(),
	ast.FormatJSON:     reflect.TypeOf((*native.JSON)(nil)).Elem(),
	ast.FormatMarkdown: reflect.TypeOf((*native.Markdown)(nil)).Elem(),
}

// BuildTemplate builds the named template file rooted at the given file
// system. Imported, rendered and extended files are read from fsys.
//
// If fsys implements FormatFS, file formats are read with its Format method,
// otherwise it depends on the file name extension
//
//   HTML       : .html
//   CSS        : .css
//   JavaScript : .js
//   JSON       : .json
//   Markdown   : .md .mkd .mkdn .mdown .markdown
//   Text       : all other extensions
//
// If the named file does not exist, BuildTemplate returns an error satisfying
// errors.Is(err, fs.ErrNotExist).
//
// If a build error occurs, it returns a *BuildError.
func BuildTemplate(fsys fs.FS, name string, options *BuildOptions) (*Template, error) {
	co := compiler.Options{
		FormatTypes: formatTypes,
	}
	var mdConverter Converter
	if options != nil {
		co.Globals = options.Globals
		co.TreeTransformer = options.TreeTransformer
		co.DisallowGoStmt = options.DisallowGoStmt
		co.NoParseShortShowStmt = options.NoParseShortShowStmt
		co.DollarIdentifier = options.DollarIdentifier
		co.Packages = options.Packages
		co.MDConverter = compiler.Converter(options.MarkdownConverter)
		mdConverter = options.MarkdownConverter
	}
	code, err := compiler.BuildTemplate(fsys, name, co)
	if err != nil {
		if e, ok := err.(compiler.Error); ok {
			err = &BuildError{err: e}
		}
		return nil, err
	}
	return &Template{fn: code.Main, typeof: code.TypeOf, globals: code.Globals, mdConverter: mdConverter}, nil
}

// Run runs the template and write the rendered code to out. vars contains
// the values of the global variables.
//
// If the executed template panics or the Panic method of native.Env is
// called, and the panic is not recovered, Run returns a *PanicError.
//
// If the Exit method of native.Env is called with a non-zero code, Run
// returns a *ExitError with the exit code.
//
// If the Fatal method of native.Env is called with argument v, Run panics
// with the value v.
func (t *Template) Run(out io.Writer, vars map[string]interface{}, options *RunOptions) error {
	if out == nil {
		return errors.New("invalid nil out")
	}
	vm := runtime.NewVM()
	if options != nil {
		if options.Context != nil {
			vm.SetContext(options.Context)
		}
		if options.Print != nil {
			vm.SetPrint(options.Print)
		}
	}
	vm.SetOutput(out, runtime.Converter(t.mdConverter))
	code, err := vm.Run(t.fn, t.typeof, initGlobalVariables(t.globals, vars))
	if err != nil {
		if p, ok := err.(*runtime.Panic); ok {
			err = &PanicError{p}
		}
		return err
	}
	if code != 0 {
		return ExitError(code)
	}
	return err
}

// MustRun is like Run but panics with the returned error if the run fails.
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
func initGlobalVariables(variables []compiler.Global, init map[string]interface{}) []reflect.Value {
	n := len(variables)
	if n == 0 {
		return nil
	}
	if init == nil {
		init = emptyInit
	}
	values := make([]reflect.Value, n)
	for i, variable := range variables {
		if variable.Pkg == "main" {
			if value, ok := init[variable.Name]; ok {
				if variable.Value.IsValid() {
					panic(fmt.Sprintf("variable %q already initialized", variable.Name))
				}
				if value == nil {
					panic(fmt.Sprintf("variable initializer %q cannot be nil", variable.Name))
				}
				val := reflect.ValueOf(value)
				if typ := val.Type(); typ == variable.Type {
					v := reflect.New(typ).Elem()
					v.Set(val)
					values[i] = v
				} else {
					if typ.Kind() != reflect.Ptr || typ.Elem() != variable.Type {
						panic(fmt.Sprintf("variable initializer %q must have type %s or %s, but have %s",
							variable.Name, variable.Type, reflect.PtrTo(variable.Type), typ))
					}
					if val.IsNil() {
						panic(fmt.Sprintf("variable initializer %q cannot be a nil pointer", variable.Name))
					}
					values[i] = reflect.ValueOf(value).Elem()
				}
				continue
			}
		}
		if variable.Value.IsValid() {
			values[i] = variable.Value
		} else {
			values[i] = reflect.New(variable.Type).Elem()
		}
	}
	return values
}
