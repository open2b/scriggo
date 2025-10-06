// Copyright 2020 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"io/fs"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// templateFS implements a file system that reads the files in a directory.
type templateFS struct {
	fsys    fs.FS
	watcher *fsnotify.Watcher
	changed chan string
	Errors  chan error

	sync.Mutex
	watched map[string]bool
}

func newTemplateFS(root string) (*templateFS, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	dir := &templateFS{
		fsys:    os.DirFS(root),
		watcher: watcher,
		watched: map[string]bool{},
		changed: make(chan string),
	}
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					dir.changed <- strings.ReplaceAll(event.Name, "\\", "/")
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				dir.Errors <- err
			}
		}
	}()
	return dir, nil
}

func (t *templateFS) Changed() chan string {
	return t.changed
}

func (t *templateFS) Close() error {
	return t.watcher.Close()
}

func (t *templateFS) Open(name string) (fs.File, error) {
	err := t.watch(name)
	if err != nil {
		return nil, err
	}
	return t.fsys.Open(name)
}

func (t *templateFS) ReadFile(name string) ([]byte, error) {
	err := t.watch(name)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(t.fsys, name)
}

func (t *templateFS) watch(name string) error {
	t.Lock()
	if !t.watched[name] {
		err := t.watcher.Add(name)
		if err != nil {
			t.Unlock()
			return err
		}
		t.watched[name] = true
	}
	t.Unlock()
	return nil
}

// recFS is a file system that reads files from a templateFS and records the
// names of the read files. The names of the read files can be retrieved by
// calling the RecordedNames method.
type recFS struct {
	*templateFS

	mu    sync.Mutex
	names []string
}

func newRecFS(fs *templateFS) *recFS {
	return &recFS{templateFS: fs}
}

func (fs *recFS) ReadFile(name string) ([]byte, error) {
	fs.append(name)
	return fs.templateFS.ReadFile(name)
}

func (fs *recFS) Open(name string) (fs.File, error) {
	fs.append(name)
	return fs.templateFS.Open(name)
}

func (fs *recFS) RecordedNames() []string {
	fs.mu.Lock()
	names := fs.names
	fs.names = nil
	fs.mu.Unlock()
	if names == nil {
		names = []string{}
	}
	return names
}

func (fs *recFS) append(name string) {
	fs.mu.Lock()
	if !slices.Contains(fs.names, name) {
		fs.names = append(fs.names, name)
	}
	fs.mu.Unlock()
}
