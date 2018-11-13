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
	// ErrInvalid è ritornato da Execute quando il parametro path non è valido.
	ErrInvalid = errors.New("template: invalid argument")

	// ErrNotExist è ritornato da Execute quando il path non esiste.
	ErrNotExist = errors.New("template: path does not exist")
)

// Un valore ExecErrors è ritornato da Execute quando si verificano uno o più
// errori di esecuzione. Riporta tutti gli errori di esecuzione nell'ordine
// in cui si sono verificati.
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

// New ritorna un template i cui file vengono letti dalla directory dir
// e parsati nel contesto ctx.
func New(dir string, ctx Context) *Template {
	var r parser.Reader = parser.DirReader(dir)
	return &Template{parser: parser.NewParser(r), ctx: ctx}
}

// Execute esegue il file del template con il path indicato, relativo
// alla directory del template, e scrive il risultato su out.
// Le variabili in vars sono definite nell'ambiente durante l'esecuzione.
//
// In caso di errore durante l'esecuzione, questa continua ugualmente per
// poi ritornare un ExecError con tutti gli errori che si sono verificati.
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
