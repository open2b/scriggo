//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package template

import (
	"errors"
	"io"
	"sync"

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

type Template struct {
	parser *parser.Parser
	errorHandler func(error) bool
	trees  map[string]*ast.Tree
	sync.RWMutex
}

// New ritorna un template i cui file vengono letti dalla directory dir.
// Se cache è true i file vengono letti e parsati una sola volta al
// momento della loro prima esecuzione.
func New(dir string, cache bool) *Template {
	var trees map[string]*ast.Tree
	var r parser.Reader = parser.DirReader(dir)
	if cache {
		trees = map[string]*ast.Tree{}
		r = parser.NewCacheReader(r)
	}
	return &Template{parser: parser.NewParser(r), trees: trees}
}

// Execute esegue il file del template con il path indicato, relativo
// alla directory del template, e scrive il risultato su out.
// Le variabili in vars sono definite nell'ambiente durante l'esecuzione.
//
// Il contesto dipende dall'estensione del file, HTML per ".html", CSS per ".css",
// JavaScript per ".js", Text in tutti gli altri casi.
func (t *Template) Execute(out io.Writer, path string, vars interface{}) error {
	var ctx ast.Context
	switch ext(path) {
	case "html":
		ctx = ast.ContextHTML
	case "css":
		ctx = ast.ContextCSS
	case "js":
		ctx = ast.ContextJavaScript
	default:
		ctx = ast.ContextText
	}
	var err error
	var tree *ast.Tree
	if t.trees == nil {
		// senza cache
		tree, err = t.parser.Parse(path, ctx)
		if err != nil {
			return convertError(err)
		}
	} else {
		// con cache
		t.RLock()
		tree = t.trees[path]
		t.RUnlock()
		if tree == nil {
			// parsa l'albero di path e crea l'ambiente
			tree, err = t.parser.Parse(path, ctx)
			if err != nil {
				return convertError(err)
			}
			t.Lock()
			t.trees[path] = tree
			t.Unlock()
		}
	}
	return exec.Execute(out, tree, "", vars, t.errorHandler)
}

func (t *Template) SetErrorHandler(h func(error) bool) {
	t.errorHandler = h
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

// ext ritorna l'estensione di path.
func ext(path string) string {
	for i := len(path) - 1; i >= 0 && path[i] != '/'; i-- {
		if path[i] == '.' {
			return path[i+1:]
		}
	}
	return ""
}
