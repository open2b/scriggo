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

// New ritorna un template i cui file sono nella directory dir.
// Se cache è true allora un file viene letto e messo in cache
// alla sua prima esecuzione e non sarà più riletto.
func New(dir string, cache bool) *Template {
	r := parser.FileReader(dir)
	if cache {
		r = parser.CacheReader(r)
	}
	return &Template{r, parser.NewParser(r)}
}

// Execute esegue un file del template al percorso path all'interno
// di dir e scrive il risualtato su out.
// Le variabili in vars saranno definite nell'ambiente di esecuzione.
func (t *Template) Execute(out io.Writer, path string, vars map[string]interface{}) error {
	tree, err := t.parser.Parse(path)
	if err != nil {
		return err
	}
	env := exec.NewEnv(tree, nil)
	return env.Execute(out, vars)
}
