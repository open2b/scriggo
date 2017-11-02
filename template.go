//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

package template

import (
	"errors"
	"io"
	"sync"

	"open2b/template/exec"
	"open2b/template/parser"
)

type HTML = exec.HTML

var (
	// ErrInvalid è ritornato da Execute quando il parametro path non è valido.
	ErrInvalid = errors.New("template: invalid argument")

	// ErrNotExist è ritornato da Execute quando il path non esiste.
	ErrNotExist = errors.New("template: path does not exist")
)

type Template struct {
	read   parser.ReadFunc
	parser *parser.Parser
	envs   map[string]*exec.Env
	sync.RWMutex
}

// New ritorna un template i cui file vengono letti dalla directory dir.
// Se cache è true i file vengono letti e parsati una sola volta al
// momento della loro prima esecuzione.
func New(dir string, cache bool) *Template {
	var envs map[string]*exec.Env
	r := parser.FileReader(dir)
	if cache {
		envs = map[string]*exec.Env{}
		r = parser.CacheReader(r)
	}
	return &Template{read: r, parser: parser.NewParser(r), envs: envs}
}

// Execute esegue il file del template con il path indicato, relativo
// alla directory del template, e scrive il risultato su out.
// Le variabili in vars sono definite nell'ambiente durante l'esecuzione.
func (t *Template) Execute(out io.Writer, path string, vars map[string]interface{}) error {
	var env *exec.Env
	if t.envs == nil {
		// senza cache
		tree, err := t.parser.Parse(path)
		if err != nil {
			return convertError(err)
		}
		env = exec.NewEnv(tree)
	} else {
		// con cache
		t.RLock()
		env = t.envs[path]
		t.RUnlock()
		if env == nil {
			// parsa l'albero di path e crea l'ambiente
			tree, err := t.parser.Parse(path)
			if err != nil {
				return convertError(err)
			}
			env = exec.NewEnv(tree)
			t.Lock()
			t.envs[path] = env
			t.Unlock()
		}
	}
	return env.Execute(out, vars)
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
