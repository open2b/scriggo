// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"testing/iotest"
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
			dir := NewDirLimitedReader(filepath.Dir(f.Name()), file.max, file.max)
			_, err = dir.Read("/" + filepath.Base(f.Name()))
			if file.size <= file.max {
				if err == ErrReadTooLarge {
					t.Errorf("unexpected error %q", ErrReadTooLarge)
				} else if err != nil {
					t.Fatal(err)
				}
			} else {
				if err != ErrReadTooLarge {
					if err == nil {
						t.Errorf("unexpected no error, expecting error %q", ErrReadTooLarge)
					} else {
						t.Errorf("unexpected error %q, expecting %q", err, ErrReadTooLarge)
					}
				}
			}
		}()
	}
	testReader = nil
}

const long255chars = "È1234567890123456789012345678901234567890123456789012" +
	"È123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789" +
	"È123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.a"

const long256chars = "È12345678901234567890123456789012345678901234567890123" +
	"È123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789" +
	"È123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.a"

var validDirReaderFilePaths = []string{"/a.b", "/a/a.b", "a.b", "a/a.b", "../a.b", "a/../a.b", long255chars}

var invalidDirReaderFilePaths = []string{"a/", ".", "...", ".aa", "a/a..", "a/ab.", "./abc", " ", " abc",
	"abc ", "a\x00b.c", "a\x1fb.c", "a\"b.c", "a*ab.c", "a:b.c", "a<b.c", "a>b.c", "a?b.c", "a\\b.c",
	"a|b.c", "a\x7fb.c", "/con/ab.c", "/prn/ab.c", "/aux/ab.c", "/nul/ab.c", "com0/ab.c", "com9/ab.c",
	"lpt0/ab.c", "lpt9/ab.c", "con.a", "prn.a", "aux.a", "nul.a", "com0.a", "com9.a", "lpt0.a", "lpt9.a",
	long256chars,
}

func TestValidDirReaderPath(t *testing.T) {
	for _, p := range validDirReaderFilePaths {
		if !ValidDirReaderPath(p) {
			t.Errorf("path: %q, expected valid, but invalid\n", p)
		}
	}
	for _, p := range invalidDirReaderFilePaths {
		if ValidDirReaderPath(p) {
			t.Errorf("path: %q, expected invalid, but valid\n", p)
		}
	}
}
