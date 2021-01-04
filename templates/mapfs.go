// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package templates

import (
	"io"
	"os"
	"time"

	"github.com/open2b/scriggo/fs"
)

// MapFS implements a file system that read the files from a map.
type MapFS map[string]string

func (fsys MapFS) Open(name string) (fs.File, error) {
	if fs.ValidPath(name) {
		data, ok := fsys[name]
		if ok {
			return &mapFile{name, data, 0}, nil
		}
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

type mapFile struct {
	name   string
	data   string
	offset int
}

func (f *mapFile) Stat() (os.FileInfo, error) {
	return (*mapFileInfo)(f), nil
}

func (f *mapFile) Read(p []byte) (int, error) {
	if f.offset < 0 {
		return 0, &os.PathError{Op: "read", Path: f.name, Err: os.ErrInvalid}
	}
	if f.offset == len(f.data) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.offset:])
	f.offset += n
	return n, nil
}

func (f *mapFile) Close() error {
	f.offset = -1
	return nil
}

type mapFileInfo mapFile

func (i *mapFileInfo) Name() string       { return i.name }
func (i *mapFileInfo) Size() int64        { return int64(len(i.data)) }
func (i *mapFileInfo) Mode() os.FileMode  { return 0 }
func (i *mapFileInfo) ModTime() time.Time { return time.Time{} }
func (i *mapFileInfo) IsDir() bool        { return false }
func (i *mapFileInfo) Sys() interface{}   { return nil }
