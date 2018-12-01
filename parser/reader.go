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
	"strings"
	"sync"
	"unicode/utf8"

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
	if !ValidDirReaderPath(path) {
		return nil, ErrInvalidPath
	}
	if path[0] != '/' {
		return nil, errors.New("template/parser: path for reader must be absolute")
	}
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

// DirLimitedReader implements a Reader that reads the source of a template
// from files in a directory limiting the maximum file size and the total
// bytes read from all files.
// It can be used concurrently.
type DirLimitedReader struct {
	dir       string
	maxFile   int
	remaining int
	mutex     sync.Mutex
}

// NewDirLimitedReader returns a DirLimitedReader that reads the file from
// directory dir limiting the file size to maxFile bytes and the total bytes
// read from all files to maxTotal.
// It panics if maxFile or maxTotal are negative.
func NewDirLimitedReader(dir string, maxFile, maxTotal int) *DirLimitedReader {
	if maxFile < 0 {
		panic("template/parser: negative max file")
	}
	if maxTotal < 0 {
		panic("template/parser: negative max total")
	}
	return &DirLimitedReader{dir, maxFile, maxTotal, sync.Mutex{}}
}

// testReader is set only for testing.
var testReader func(io.Reader) io.Reader

// Read implements the Read method of the Reader.
// If a limit is exceeded it returns the error ErrReadTooLarge.
func (dr *DirLimitedReader) Read(path string, ctx ast.Context) (*ast.Tree, error) {
	if !ValidDirReaderPath(path) {
		return nil, ErrInvalidPath
	}
	if path[0] != '/' {
		return nil, errors.New("template/parser: path for reader must be absolute")
	}
	// Opens the file.
	f, err := os.Open(filepath.Join(dr.dir, path))
	if err != nil {
		if os.IsNotExist(err) {
			err = ErrNotExist
		}
		return nil, err
	}
	defer f.Close()
	// Maximum number of byte to read.
	max := dr.maxFile
	dr.mutex.Lock()
	if max > dr.remaining {
		max = dr.remaining
	}
	dr.mutex.Unlock()
	// Tries to gets the file size.
	var size int64
	if fi, err := f.Stat(); err == nil {
		size = fi.Size()
		if size > int64(max) {
			return nil, ErrReadTooLarge
		}
	}
	err = nil
	if size == 0 {
		// File size is zero or it has failed to read the size.
		size = int64(max)
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
	for n < max && err == nil {
		if n == len(src) {
			// Grows the buffer.
			old := src
			size = int64(len(old)) * 2
			if size > int64(max) {
				size = int64(max)
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
				return nil, ErrReadTooLarge
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
		return nil, err
	}
	dr.mutex.Lock()
	defer dr.mutex.Unlock()
	if n > dr.remaining {
		return nil, ErrReadTooLarge
	}
	dr.remaining -= n
	return tree, err
}

// ValidDirReaderPath indicates whether path is valid as path for DirReader
// and DirLimitedReader.
func ValidDirReaderPath(path string) bool {
	// Must be a valid path
	if !validPath(path) {
		return false
	}
	// Splits the path in the various names.
	var names = strings.Split(path, "/")
	last := len(names) - 1
	for i, name := range names {
		// If the first name is empty, path starts with '/'.
		if i == 0 && name == "" {
			continue
		}
		if i < last && name == ".." {
			continue
		}
		// Cannot be long less than 256 characters.
		if utf8.RuneCountInString(name) >= 256 {
			return false
		}
		// Cannot be '.' and cannot contain '..'.
		if name == "." || strings.Contains(name, "..") {
			return false
		}
		// First and last character cannot be spaces.
		if name[0] == ' ' || name[len(name)-1] == ' ' {
			return false
		}
		// First and the last character cannot be a point.
		if name[0] == '.' || name[len(name)-1] == '.' {
			return false
		}
		if isWindowsReservedName(name) {
			return false
		}
	}
	return true
}

// isWindowsReservedName indicates if name is a reserved file name on Windows.
// See https://docs.microsoft.com/en-us/windows/desktop/fileio/naming-a-file
func isWindowsReservedName(name string) bool {
	const DEL = '\x7f'
	for i := 0; i < len(name); i++ {
		switch c := name[i]; c {
		case '"', '*', '/', ':', '<', '>', '?', '\\', '|', DEL:
			return true
		default:
			if c <= '\x1f' {
				return true
			}
		}
	}
	switch name {
	case "con", "prn", "aux", "nul",
		"com0", "com1", "com2", "com3", "com4", "com5", "com6", "com7", "com8",
		"com9", "lpt0", "lpt1", "lpt2", "lpt3", "lpt4", "lpt5", "lpt6", "lpt7",
		"lpt8", "lpt9":
		return true
	}
	if len(name) >= 4 {
		switch name[0:4] {
		case "con.", "prn.", "aux.", "nul.":
			return true
		}
		if len(name) >= 5 {
			switch name[0:5] {
			case "com0.", "com1.", "com2.", "com3.", "com4.", "com5.", "com6.",
				"com7.", "com8.", "com9.", "lpt0.", "lpt1.", "lpt2.", "lpt3.",
				"lpt4.", "lpt5.", "lpt6.", "lpt7.", "lpt8.", "lpt9.":
				return true
			}
		}
	}
	return false
}
