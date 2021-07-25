// Copyright (c) 2021 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scriggo

import (
	"io"
	"io/fs"
	"os"
)

// FileFS implements a file system with a single file.
type FileFS struct {
	name string
	data []byte
}

// NewFileFS returns a FileFS file system that contains only one file with the
// given name and data.
func NewFileFS(name string, data []byte) FileFS {
	return FileFS{name, data}
}

// Open opens the named file.
func (fsys FileFS) Open(name string) (fs.File, error) {
	if name == "." {
		return &fileReadDirFile{fileFile{name: fsys.name}, false}, nil
	}
	if name != fsys.name {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
	}
	return &fileFile{name: fsys.name, data: fsys.data}, nil
}

// fileFile implements fs.File.
type fileFile struct {
	name   string
	data   []byte
	offset int
}

func (f *fileFile) Stat() (os.FileInfo, error) {
	panic("Stat not implemented for FileFS")
}

func (f *fileFile) Read(p []byte) (int, error) {
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

func (f *fileFile) Close() error {
	f.offset = -1
	return nil
}

// fileReadDirFile implements fs.ReadDirFile.
type fileReadDirFile struct {
	fileFile
	eof bool
}

func (f *fileReadDirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if f.eof {
		return nil, io.EOF
	}
	f.eof = true
	return []fs.DirEntry{fileDirEntry(f.name)}, nil
}

// fileDirEntry implements DirEntry.
type fileDirEntry string

func (e fileDirEntry) Name() string {
	return string(e)
}

func (e fileDirEntry) IsDir() bool {
	return false
}

func (e fileDirEntry) Type() fs.FileMode {
	return 0
}

func (e fileDirEntry) Info() (fs.FileInfo, error) {
	panic("FileInfo not implemented for FileFS")
}
