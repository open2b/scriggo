// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package template implements high level methods e functions to parse
// and render a template.
package template

import (
	"errors"
	"io"

	"open2b/template/ast"
	"open2b/template/parser"
	"open2b/template/renderer"
)

type HTML = renderer.HTML

// Does an alias of Context and the constants are redefined because it must be
// possible to define a renderer.WriterTo interface only by importing the
// "template" package and not necessarily also "renderer".
type Context = ast.Context

const (
	ContextText       = ast.ContextText
	ContextHTML       = ast.ContextHTML
	ContextTag        = ast.ContextTag
	ContextAttribute  = ast.ContextAttribute
	ContextCSS        = ast.ContextCSS
	ContextJavaScript = ast.ContextScript
)

var (
	// ErrInvalid is returned from Render when the path parameter is not valid.
	ErrInvalid = errors.New("template: invalid argument")

	// ErrNotExist is returned from Render when the path does not exist.
	ErrNotExist = errors.New("template: path does not exist")
)

// A RenderErrors value is returned from Render when one or more rendering
// errors occur. Reports all rendering errors in the order in which they occurred.
type RenderErrors []*RenderError

func (ee RenderErrors) Error() string {
	var s string
	for _, e := range ee {
		if s != "" {
			s += "\n"
		}
		s += e.Error()
	}
	return s
}

type RenderError = renderer.Error

type Template struct {
	parser *parser.Parser
	ctx    ast.Context
}

// New returns a template whose files are read from the dir directory and
// parsed in the ctx context.
func New(dir string, ctx Context) *Template {
	var r = parser.DirReader(dir)
	return &Template{parser: parser.NewParser(r), ctx: ctx}
}

// Render renders the template file with the specified path, relative to
// the template directory, and writes the result to out. The variables in
// vars are defined in the environment during rendering.
//
// In the event of an error during rendering, it continues and then returns
// an RenderErrors with all errors that have occurred.
func (t *Template) Render(out io.Writer, path string, vars interface{}) error {
	tree, err := t.parser.Parse(path, t.ctx)
	if err != nil {
		return convertError(err)
	}
	return render(out, tree, vars)
}

// RenderString renders the template source src, in context ctx, and writes
// the result to out. The variables in vars are defined in the environment
// during rendering.
//
// In the event of an error during rendering, it continues and then returns
// an RenderErrors with all errors that have occurred.
func RenderString(out io.Writer, src string, ctx Context, vars interface{}) error {
	tree, err := parser.Parse([]byte(src), ctx)
	if err != nil {
		return convertError(err)
	}
	return render(out, tree, vars)
}

// render renders tree and write the result to out. The variables in
// vars are defined in the environment during rendering.
func render(out io.Writer, tree *ast.Tree, vars interface{}) error {
	var errors RenderErrors
	err := renderer.Render(out, tree, vars, func(err error) bool {
		if e, ok := err.(*RenderError); ok {
			if errors == nil {
				errors = RenderErrors{e}
			} else {
				errors = append(errors, e)
			}
			return true
		}
		return false
	})
	if err != nil {
		return err
	}
	if errors != nil {
		return errors
	}
	return nil
}

func convertError(err error) error {
	if err == parser.ErrInvalid {
		return ErrInvalid
	}
	if err == parser.ErrNotExist {
		return ErrNotExist
	}
	return err
}
