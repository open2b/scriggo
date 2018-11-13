//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package template

import (
	"errors"
	"io"

	"open2b/template/ast"
	"open2b/template/exec"
	"open2b/template/parser"
)

type HTML = exec.HTML

type Context = ast.Context

const (
	ContextText Context = iota
	ContextHTML
	ContextCSS
	ContextJavaScript
)

var (
	// ErrInvalid is returned from Execute when the path parameter is not valid.
	ErrInvalid = errors.New("template: invalid argument")

	// ErrNotExist is returned from Execute when the path does not exist.
	ErrNotExist = errors.New("template: path does not exist")
)

// An ExecErrors value is returned from Execute when one or more execution
// errors occur. Reports all execution errors in the order in which they occurred.
type ExecErrors []*ExecError

func (ee ExecErrors) Error() string {
	var s string
	for _, e := range ee {
		if s != "" {
			s += "\n"
		}
		s += e.Error()
	}
	return s
}

type ExecError = exec.Error

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

// Execute executes the template file with the specified path, relative to
// the template directory, and writes the result to out. The variables in
// vars are defined in the environment during execution.
//
// In the event of an error during execution, it continues and then returns
// an ExecError with all errors that have occurred.
func (t *Template) Execute(out io.Writer, path string, vars interface{}) error {
	tree, err := t.parser.Parse(path, t.ctx)
	if err != nil {
		return convertError(err)
	}
	var errors ExecErrors
	err = exec.Execute(out, tree, "", vars, func(err error) bool {
		if e, ok := err.(*ExecError); ok {
			if errors == nil {
				errors = ExecErrors{e}
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
