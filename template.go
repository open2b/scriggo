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

// Makes an alias of Context and redefines the constants so it's not
// necessary to import the package "renderer".

// Context indicates the type of source that has to be rendered and controls
// how to escape the values to render.
type Context = ast.Context

const (
	ContextText   Context = ast.ContextText
	ContextHTML   Context = ast.ContextHTML
	ContextCSS    Context = ast.ContextCSS
	ContextScript Context = ast.ContextScript
)

var (
	// ErrInvalidPath is returned from the Render method of Renderer when the
	// path parameter is not valid.
	ErrInvalidPath = errors.New("template: invalid path")

	// ErrNotExist is returned from the Render method of Renderer when the
	// path does not exist.
	ErrNotExist = errors.New("template: path does not exist")
)

//// A Errors value is returned from a Render method when one or more rendering
//// errors occur. Reports all rendering errors in the order in which they
//// occurred.
//type Errors []*Error
//
//func (ee Errors) Error() string {
//	var s string
//	for _, e := range ee {
//		if s != "" {
//			s += "\n"
//		}
//		s += e.Error()
//	}
//	return s
//}


// Renderer is the interface that is implemented by types that render template sources given a path.
//
type Renderer interface {
	Render(out io.Writer, path string, vars interface{}, h ErrorHandle) error
}

// ErrorHandle is a function called during the rendering when an error occurs.
// The function receives the error and returns false if the rendering must be terminated or true

type ErrorHandle func(err error) bool

// DirRenderer allows to render files located in a directory with the same
// context. Files are read and parsed the first time that are rendered.
// Subsequents renderings are faster to execute.
type DirRenderer struct {
	parser *parser.Parser
	ctx    ast.Context
}

// NewDirRenderer returns a Dir that render files located in the directory dir
// in the context ctx.
func NewDirRenderer(dir string, ctx Context) *DirRenderer {
	var r = parser.DirReader(dir)
	return &DirRenderer{parser: parser.New(r), ctx: ctx}
}

// Render renders the template file with the specified path, relative to the
// template directory, and writes the result to out. The variables in vars are
// defined in the environment during rendering. In the event of an error
// during rendering, Render calls h to handle the error. If h il nil, Render
// stops and returns the error.
//
// It is safe to call Render concurrently by more goroutines.
func (d *DirRenderer) Render(out io.Writer, path string, vars interface{}, h ErrorHandle) error {
	tree, err := d.parser.Parse(path, d.ctx)
	if err != nil {
		return convertError(err)
	}
	return RenderTree(out, tree, vars, h)
}

// MapRenderer allows to render sources as values of a map with the same
// context. Files are read and parsed the first time that are rendered.
// Subsequents renderings are faster to execute.
type MapRenderer struct {
	parser *parser.Parser
	ctx    ast.Context
}

// NewMapRenderer returns a Map that render sources as values of a map in the
// context ctx.
func NewMapRenderer(sources map[string][]byte, ctx Context) *MapRenderer {
	var r = parser.MapReader(sources)
	return &MapRenderer{parser: parser.New(r), ctx: ctx}
}

// Render renders the template source with the specified path and writes
// the result to out. The variables in vars are defined in the environment
// during rendering. In the event of an error during rendering, Render calls
// h to handle the error. If h il nil, Render stops and returns the error.
//
// It is safe to call Render concurrently by more goroutines.
func (d *MapRenderer) Render(out io.Writer, path string, vars interface{}, h ErrorHandle) error {
	tree, err := d.parser.Parse(path, d.ctx)
	if err != nil {
		return convertError(err)
	}
	return RenderTree(out, tree, vars, h)
}

// RenderSource renders the template source src, in context ctx, and writes
// the result to out. The variables in vars are defined in the environment
// during rendering. In the event of an error during rendering, RenderSource
// calls h to handle the error. If h il nil, RenderSource stops and returns
// the error.
//
// Statements "extend", "import" and "show <path>" cannot be used with
// RenderSource, use the function RenderTree or the method Render of a
// Renderer, as DirRenderer and MapRenderer, instead.
//
// It is safe to call RenderSource concurrently by more goroutines.
func RenderSource(out io.Writer, src []byte, ctx Context, vars interface{}, h ErrorHandle) error {
	tree, err := parser.ParseSource(src, ctx)
	if err != nil {
		return convertError(err)
	}
	return RenderTree(out, tree, vars, h)
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
