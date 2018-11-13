//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package parser

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"open2b/template/ast"
)

// Reader definisce un tipo che consente di leggere i sorgenti di un template.
//
// Read deve ritornare sempre un nuovo albero in quanto il chiamante può
// modificare l'albero ritornato.
type Reader interface {
	Read(path string, ctx ast.Context) (*ast.Tree, error)
}

// DirReader implementa un Reader che legge i sorgenti
// di un template dai file in una directory.
type DirReader string

// Read implementa il metodo Read del Reader.
func (dir DirReader) Read(path string, ctx ast.Context) (*ast.Tree, error) {
	src, err := ioutil.ReadFile(filepath.Join(string(dir), path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotExist
		}
		return nil, err
	}
	tree, err := Parse(src, ctx)
	if err != nil {
		if err2, ok := err.(*Error); ok {
			err2.Path = path
		}
		return nil, err
	}
	tree.Path = path
	return tree, nil
}
