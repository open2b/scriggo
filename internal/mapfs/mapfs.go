// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file is a copy of the "templates/mapfs.go" file. Keep in sync.

package mapfs

import (
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

// MapFS implements a file system that read the files from a map.
type MapFS map[string]string

func (fsys MapFS) Open(name string) (fs.File, error) {
	if fs.ValidPath(name) {
		if name == "." {
			return &mapDir{mapFile: mapFile{name: name, mode: fs.ModeDir}, fsys: fsys}, nil
		}
		data, ok := fsys[name]
		if ok {
			return &mapFile{name, data, 0, 0}, nil
		}
		prefix := name + "/"
		for n := range fsys {
			if strings.HasPrefix(n, prefix) {
				return &mapDir{mapFile: mapFile{name: name, mode: fs.ModeDir}, fsys: fsys}, nil
			}
		}
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

type mapDir struct {
	mapFile
	fsys map[string]string
	n    int
}

func (d *mapDir) ReadDir(n int) ([]fs.DirEntry, error) {
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
		entries[i] = &mapDirEntry{mapFileInfo{name: name}}
	}
	return entries, nil
}

// mapDirEntry implements fs.DirEntry.
type mapDirEntry struct {
	mapFileInfo
}

func (f *mapDirEntry) Type() fs.FileMode {
	return f.Mode()
}

func (f *mapDirEntry) Info() (fs.FileInfo, error) {
	return &f.mapFileInfo, nil
}

type mapFile struct {
	name   string
	data   string
	offset int
	mode   os.FileMode
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

func (i *mapFileInfo) Name() string       { return path.Base(i.name) }
func (i *mapFileInfo) Size() int64        { return int64(len(i.data)) }
func (i *mapFileInfo) Mode() os.FileMode  { return i.mode }
func (i *mapFileInfo) ModTime() time.Time { return time.Time{} }
func (i *mapFileInfo) IsDir() bool        { return i.mode&fs.ModeDir == fs.ModeDir }
func (i *mapFileInfo) Sys() interface{}   { return nil }
