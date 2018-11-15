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

// Reader defines a type that lets you read the source of a template.
//
// Read must always return a new tree because the caller can modify
// the returned tree.
type Reader interface {
	Read(path string, ctx ast.Context) (*ast.Tree, error)
}

// DirReader implements a Reader that reads the source of a template
// from files in a directory.
type DirReader string

// Read implements the Read method of the Reader.
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
	return tree, nil
}
