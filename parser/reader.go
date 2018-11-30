// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"open2b/template/ast"
)

// ErrFileTooLarge is returned from DirLimitedReader's Read when the file
// to read is too large.
var ErrFileTooLarge = errors.New("template/parser: file too large")

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

// DirReader implements a Reader that reads the source of a template
// from files in a directory up to a maximum size. If the file has
// a size greater then Max returns the error ErrFileTooLarge.
type DirLimitedReader struct {
	Dir string
	Max int
}

// testReader is set only for testing.
var testReader func(io.Reader) io.Reader

// Read implements the Read method of the Reader.
func (dir DirLimitedReader) Read(path string, ctx ast.Context) (*ast.Tree, error) {
	if dir.Max < 0 {
		return nil, errors.New("template/parser: negative max size")
	}
	// Opens the file.
	f, err := os.Open(filepath.Join(dir.Dir, path))
	if err != nil {
		if os.IsNotExist(err) {
			err = ErrNotExist
		}
		return nil, err
	}
	defer f.Close()
	// Tries to gets the file size.
	var size int64
	if fi, err := f.Stat(); err == nil {
		size = fi.Size()
		if size > int64(dir.Max) {
			return nil, ErrFileTooLarge
		}
	}
	err = nil
	if size == 0 {
		// File size is zero or it has failed to read the size.
		size = int64(dir.Max)
		if size > 512 {
			size = 512
		}
	}
	// Wraps the file reader in case of a test.
	var r io.Reader = f
	if testReader != nil {
		r = testReader(r)
	}
	// Reads the source from the file.
	n := 0
	src := make([]byte, size)
	for n < dir.Max && err == nil {
		if n == len(src) {
			// Grows the buffer.
			old := src
			size = int64(len(old)) * 2
			if size > int64(dir.Max) {
				size = int64(dir.Max)
			}
			src = make([]byte, size)
			copy(src, old)
		}
		var nn int
		nn, err = r.Read(src[n:])
		n += nn
	}
	if err != io.EOF {
		if err != nil {
			return nil, err
		}
		// Expects 0 and EOF from next read.
		if nn, err2 := r.Read(make([]byte, 1)); nn != 0 || err2 != io.EOF {
			if nn != 0 {
				return nil, ErrFileTooLarge
			} else if err2 != nil {
				return nil, err2
			}
		}
	}
	err = f.Close()
	if err != nil {
		return nil, err
	}
	// Parses the tree.
	tree, err := Parse(src[:n], ctx)
	if err != nil {
		if err2, ok := err.(*Error); ok {
			err2.Path = path
		}
	}
	return tree, err
}
