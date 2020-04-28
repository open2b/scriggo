// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package templates

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/open2b/scriggo/compiler"
)

// FileReader is implemented by values that can read files of a template.
// name, if not absolute, is relative to the root of the template. If the file
// with the given name does not exist, it returns nil and an os not found
// error.
type FileReader interface {
	ReadFile(name string) ([]byte, error)
}

// DirReader implements a FileReader that reads the files in a directory.
//
// To limit the size of read files, use DirLimitedReader instead.
type DirReader string

// ReadFile implements the ReadFile method of FileReader.
func (dir DirReader) ReadFile(name string) ([]byte, error) {
	if !ValidDirReaderPath(name) {
		return nil, ErrInvalidPath
	}
	return ioutil.ReadFile(filepath.Join(string(dir), name))
}

// DirLimitedReader implements a FileReader that reads a source from files in
// a directory limiting the maximum file size and the total bytes read from
// all reads.
//
// Use DirLimitedReader, instead of DirReader, when you do not have control
// of file sizes. As a Parser reads a file with a specific context only once,
// DirLimitedReader can be passed to a Parser to prevent it
// from allocating too much memory.
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
		panic("scriggo/parser: negative max file")
	}
	if maxTotal < 0 {
		panic("scriggo/parser: negative max total")
	}
	return &DirLimitedReader{dir, maxFile, maxTotal, sync.Mutex{}}
}

// testReader is set only for testing.
var testReader func(io.Reader) io.Reader

// ReadFile implements the ReadFile method of FileReader.
// If a limit is exceeded it returns the error ErrReadTooLarge.
func (dr *DirLimitedReader) ReadFile(name string) ([]byte, error) {
	if !ValidDirReaderPath(name) {
		return nil, ErrInvalidPath
	}
	// Opens the file.
	f, err := os.Open(filepath.Join(dr.dir, name))
	if err != nil {
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
	// Try to gets the file size.
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
	// Wrap the file reader in case of a test.
	var r io.Reader = f
	if testReader != nil {
		r = testReader(r)
	}
	// Read the source from the file.
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
		// Expect 0 and EOF from next read.
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
	return src[:n], nil
}

// MapReader implements a FileReader where sources are read from a map.
// Map keys are the file names. If a name is present both relative and
// absolute, the file of the absolute one is returned.
type MapReader map[string][]byte

// ReadFile implements the ReadFile method of FileReader.
func (r MapReader) ReadFile(name string) ([]byte, error) {
	if !compiler.ValidTemplatePath(name) {
		return nil, ErrInvalidPath
	}
	if name[0] != '/' {
		name = "/" + name
	}
	src, ok := r[name]
	if !ok {
		src, ok = r[name[1:]]
		if !ok {
			return nil, os.ErrNotExist
		}
	}
	return src, nil
}

// ValidDirReaderPath reports whether path is valid as name for DirReader
// and DirLimitedReader.
func ValidDirReaderPath(path string) bool {
	// Check if it is a valid template path.
	if !compiler.ValidTemplatePath(path) {
		return false
	}
	// Split the path in the various names.
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

// isWindowsReservedName reports whether name is a reserved file name on
// Windows.
//
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
