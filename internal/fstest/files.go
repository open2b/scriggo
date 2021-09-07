// Copyright 2020 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fstest

import (
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

// Files implements a file system that read the files from a map.
type Files map[string]string

func (fsys Files) Open(name string) (fs.File, error) {
	if fs.ValidPath(name) {
		if name == "." {
			return &filesDir{filesFile: filesFile{name: name, mode: fs.ModeDir}, fsys: fsys}, nil
		}
		data, ok := fsys[name]
		if ok {
			return &filesFile{name, data, 0, 0}, nil
		}
		prefix := name + "/"
		for n := range fsys {
			if strings.HasPrefix(n, prefix) {
				return &filesDir{filesFile: filesFile{name: name, mode: fs.ModeDir}, fsys: fsys}, nil
			}
		}
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

type filesDir struct {
	filesFile
	fsys map[string]string
	n    int
}

func (d *filesDir) ReadDir(n int) ([]fs.DirEntry, error) {
	var dir string
	if d.name != "." {
		dir = d.name + "/"
	}
	var names []string
	hasDir := map[string]bool{}
	for name := range d.fsys {
		if !strings.HasPrefix(name, dir) {
			continue
		}
		if i := strings.IndexByte(name[len(dir):], '/'); i > 0 {
			name = name[:len(dir)+i]
			if hasDir[name] {
				continue
			}
			hasDir[name] = true
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if n > 0 {
		if len(names) <= d.n {
			return nil, io.EOF
		}
		names = names[d.n:]
		if len(names) > n {
			names = names[:n]
		}
		d.n += len(names)
	}
	entries := make([]fs.DirEntry, len(names))
	for i, name := range names {
		entries[i] = &mapDirEntry{filesFileInfo{name: name}}
	}
	return entries, nil
}

// mapDirEntry implements fs.DirEntry.
type mapDirEntry struct {
	filesFileInfo
}

func (f *mapDirEntry) Type() fs.FileMode {
	return f.Mode()
}

func (f *mapDirEntry) Info() (fs.FileInfo, error) {
	return &f.filesFileInfo, nil
}

type filesFile struct {
	name   string
	data   string
	offset int
	mode   os.FileMode
}

func (f *filesFile) Stat() (os.FileInfo, error) {
	return (*filesFileInfo)(f), nil
}

func (f *filesFile) Read(p []byte) (int, error) {
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

func (f *filesFile) Close() error {
	f.offset = -1
	return nil
}

type filesFileInfo filesFile

func (i *filesFileInfo) Name() string       { return path.Base(i.name) }
func (i *filesFileInfo) Size() int64        { return int64(len(i.data)) }
func (i *filesFileInfo) Mode() os.FileMode  { return i.mode }
func (i *filesFileInfo) ModTime() time.Time { return time.Time{} }
func (i *filesFileInfo) IsDir() bool        { return i.mode&fs.ModeDir == fs.ModeDir }
func (i *filesFileInfo) Sys() interface{}   { return nil }
