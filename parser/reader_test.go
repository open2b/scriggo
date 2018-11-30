// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"testing/iotest"

	"open2b/template/ast"
)

var space = []byte{' '}

var dirLimitedFiles = []struct {
	size int
	max  int
	read func(io.Reader) io.Reader
}{
	{0, 0, nil},
	{1, 0, nil},
	{0, 1, nil},
	{511, 511, nil},
	{512, 511, nil},
	{511, 512, nil},
	{512, 512, nil},
	{513, 512, nil},
	{512, 513, nil},
	{513, 513, nil},
	{2789, 2903, nil},
	{2903, 2789, nil},
	{2903, 2903, nil},
	{763, 1000, iotest.OneByteReader},
	{100, 763, iotest.OneByteReader},
	{1892, 2000, iotest.HalfReader},
	{2000, 1892, iotest.HalfReader},
	{1892, 2000, iotest.DataErrReader},
	{2000, 1892, iotest.DataErrReader},
}

func TestDirLimitedReader(t *testing.T) {
	for _, file := range dirLimitedFiles {
		func() {
			f, err := ioutil.TempFile("", "test-dir-limited-reader-")
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = f.Close()
				_ = os.Remove(f.Name())
			}()
			if file.size > 0 {
				_, err = f.WriteAt(space, int64(file.size)-1)
				if err != nil {
					t.Fatal(err)
				}
			}
			testReader = file.read
			dir := DirLimitedReader{filepath.Dir(f.Name()), file.max}
			_, err = dir.Read(filepath.Base(f.Name()), ast.ContextText)
			if file.size <= file.max {
				if err == ErrFileTooLarge {
					t.Errorf("unexpected error %q", ErrFileTooLarge)
				} else if err != nil {
					t.Fatal(err)
				}
			} else {
				if err != ErrFileTooLarge {
					if err == nil {
						t.Errorf("unexpected no error, expecting error %q", ErrFileTooLarge)
					} else {
						t.Errorf("unexpected error %q, expecting %q", err, ErrFileTooLarge)
					}
				}
			}
		}()
	}
	testReader = nil
}
