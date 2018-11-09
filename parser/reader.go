//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package parser

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"open2b/template/ast"
)

// Reader definisce un tipo che consente di leggere i sorgenti di un template.
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

// CacheReader implementa un Reader che legge e mette in una cache
// i sorgenti del template letti da un altro reader.
type CacheReader struct {
	reader Reader
	trees  map[cacheReaderEntry]*ast.Tree
	sync.Mutex
}

// cacheReaderEntry implementa una entry di CacheReader.
type cacheReaderEntry struct {
	path string
	ctx  ast.Context
}

// NewCacheReader ritorna un CacheReader che legge i sorgenti dal reader r.
func NewCacheReader(r Reader) *CacheReader {
	return &CacheReader{
		reader: r,
		trees:  map[cacheReaderEntry]*ast.Tree{},
	}
}

// Read implementa il metodo Read del Reader.
func (r *CacheReader) Read(path string, ctx ast.Context) (*ast.Tree, error) {
	var err error
	var entry = cacheReaderEntry{path, ctx}
	r.Lock()
	tree, ok := r.trees[entry]
	r.Unlock()
	if !ok {
		tree, err = r.reader.Read(path, ctx)
		if err == nil {
			r.Lock()
			r.trees[entry] = tree
			r.Unlock()
		}
	}
	return tree, err
}
