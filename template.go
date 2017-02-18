//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

package template

import (
	"io"

	"open2b/template/exec"
	"open2b/template/parser"
)

type Template struct {
	read   parser.ReadFunc
	parser *parser.Parser
}

// New ritorna un template i cui file vengono letti dalla directory dir.
// Se cache è true i file vengono letti e parsati una sola volta al
// momento della loro prima esecuzione.
func New(dir string, cache bool) *Template {
	r := parser.FileReader(dir)
	if cache {
		r = parser.CacheReader(r)
	}
	return &Template{r, parser.NewParser(r)}
}

// Execute esegue il file del template con il path indicato, relativo
// alla directory del template, e scrive il risultato su out.
// Le variabili in vars sono definite nell'ambiente durante l'esecuzione.
func (t *Template) Execute(out io.Writer, path string, vars map[string]interface{}) error {
	tree, err := t.parser.Parse(path)
	if err != nil {
		return err
	}
	env := exec.NewEnv(tree, nil)
	return env.Execute(out, vars)
}
