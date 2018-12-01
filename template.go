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
	"open2b/template/renderer"
)

// HTML encapsulates a string containing an HTML code that have to be rendered
// without escape. It should be used with context HTML.
//
//   // example:
//   vars := map[string]interface{}{"link": template.HTML("<a href="/">go</a>")}
type HTML = renderer.HTML

// Makes an alias of Context and redefines the constants so it's not
// necessary to import the package "renderer".

// Context indicates the type of source that has to be rendered and controls
// how to escape the resulting value of the statement {{ expr }}.
type Context = ast.Context

//
const (
	ContextText   = ast.ContextText
	ContextHTML   = ast.ContextHTML
	ContextCSS    = ast.ContextCSS
	ContextScript = ast.ContextScript
)

var (
	// ErrInvalidPath is returned from Render when the path parameter is not valid.
	ErrInvalidPath = errors.New("template: invalid path")

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

// Template allows to render files located with same context and located in a
// directory. Files are read and parsed at the first rendering so subsequent
// renderings are fast to execute.
type Template struct {
	parser *parser.Parser
	ctx    ast.Context
}

// New returns a template whose files are read from the template directory
// dir and parsed in the context ctx.
func New(dir string, ctx Context) *Template {
	var r = parser.DirReader(dir)
	return &Template{parser: parser.NewParser(r), ctx: ctx}
}

// Render renders the template file with the specified path, relative to
// the template directory, and writes the result to out. The variables in
// vars are defined in the environment during rendering.
//
// In the event of an error during rendering, it continues and then returns
// a RenderErrors error with all errors that have occurred.
//
// It is safe to call Render concurrently by more goroutines.
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
// Statements "extend", "import" and "show <path>" cannot be used with
// RenderString, use Render instead.
//
// In the event of an error during rendering, it continues and then returns
// a RenderErrors error with all errors that have occurred.
//
// It is safe to call RenderString concurrently by more goroutines.
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
	if err == parser.ErrInvalidPath {
		return ErrInvalidPath
	}
	if err == parser.ErrNotExist {
		return ErrNotExist
	}
	return err
}
