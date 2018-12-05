// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"errors"
	"io"

	"open2b/template/ast"
	"open2b/template/parser"
)

// Context indicates the type of source that has to be rendered and controls
// how to escape the values to render.
type Context int

const (
	ContextText   Context = Context(ast.ContextText)
	ContextHTML   Context = Context(ast.ContextHTML)
	ContextCSS    Context = Context(ast.ContextCSS)
	ContextScript Context = Context(ast.ContextScript)
)

var (
	// ErrInvalidPath is returned from a rendering function of method when the
	// path argument is not valid.
	ErrInvalidPath = errors.New("template: invalid path")

	// ErrNotExist is returned from a rendering function of method when the
	// path passed as argument does not exist.
	ErrNotExist = errors.New("template: path does not exist")
)

// Errors is an error that can be returned after a rendering and contains all
// errors on expressions and statements execution that has occurred during the
// rendering of the template.
//
// If the argument strict of a renderer function or Render type constructor is
// true, these types of errors stop the execution. If no other type of error
// has occurred, at the end, an Errors error is returned with all errors on
// expressions and statements execution that have occurred.
//
// Syntax errors, not-existing paths, write errors and invalid global
// variable definitions are all types of errors that always stop rendering.
//
// Operations on wrong types, out of bound on a slice, too few arguments in
// function calls are all types of errors that stop the execution only if
// strict is true.
type Errors []error

func (errs Errors) Error() string {
	var s string
	for _, err := range []error(errs) {
		if s != "" {
			s += "\n"
		}
		s += err.Error()
	}
	return s
}

// Renderer is the interface that is implemented by types that render template
// sources given a path.
type Renderer interface {
	Render(out io.Writer, path string, vars interface{}) error
}

// DirRenderer allows to render files located in a directory with the same
// context. Files are read and parsed only the first time that are rendered.
// Subsequents renderings are faster to execute.
type DirRenderer struct {
	parser *parser.Parser
	strict bool
	ctx    ast.Context
}

// NewDirRenderer returns a Dir that render files located in the directory dir
// in the context ctx. If strict is true, even errors on expressions and
// statements execution stop the rendering. See the type Errors for more
// details.
func NewDirRenderer(dir string, strict bool, ctx Context) *DirRenderer {
	var r = parser.DirReader(dir)
	return &DirRenderer{parser: parser.New(r), strict: strict, ctx: ast.Context(ctx)}
}

// Render renders the template file with the specified path, relative to the
// template directory, and writes the result to out. The variables in vars are
// defined as global variables.
//
// It is safe to call Render concurrently by more goroutines.
func (dr *DirRenderer) Render(out io.Writer, path string, vars interface{}) error {
	tree, err := dr.parser.Parse(path, dr.ctx)
	if err != nil {
		return convertError(err)
	}
	return RenderTree(out, tree, vars, dr.strict)
}

// MapRenderer allows to render sources as values of a map with the same
// context. Sources are parsed only the first time that are rendered.
// Subsequents renderings are faster to execute.
type MapRenderer struct {
	parser *parser.Parser
	strict bool
	ctx    ast.Context
}

// NewMapRenderer returns a Map that render sources as values of a map in the
// context ctx. If strict is true, even errors on expressions and statements
// execution stop the rendering. See the type Errors for more details.
func NewMapRenderer(sources map[string][]byte, strict bool, ctx Context) *MapRenderer {
	var r = parser.MapReader(sources)
	return &MapRenderer{parser: parser.New(r), strict: strict, ctx: ast.Context(ctx)}
}

// Render renders the template source with the specified path and writes
// the result to out. The variables in vars are defined as global variables.
//
// It is safe to call Render concurrently by more goroutines.
func (mr *MapRenderer) Render(out io.Writer, path string, vars interface{}) error {
	tree, err := mr.parser.Parse(path, mr.ctx)
	if err != nil {
		return convertError(err)
	}
	return RenderTree(out, tree, vars, mr.strict)
}

// RenderSource renders the template source src, in context ctx, and writes
// the result to out. The variables in vars are defined as global variables.
// If strict is true, even errors on expressions and statements execution stop
// the rendering. See the type Errors for more details.
//
// Statements "extend", "import" and "show <path>" cannot be used with
// RenderSource, use the function RenderTree or the method Render of a
// Renderer, as DirRenderer and MapRenderer, instead.
//
// It is safe to call RenderSource concurrently by more goroutines.
func RenderSource(out io.Writer, src []byte, vars interface{}, strict bool, ctx Context) error {
	tree, err := parser.ParseSource(src, ast.Context(ctx))
	if err != nil {
		return convertError(err)
	}
	return RenderTree(out, tree, vars, strict)
}

func convertError(err error) error {
	if err == parser.ErrInvalidPath {
		return ErrInvalidPath
	}
	if err == parser.ErrNotExist {
		return ErrNotExist
	}
	return err
}
