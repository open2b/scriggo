// Copyright 2020 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"io/fs"
	"os"
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
				if event.Op&fsnotify.Write == fsnotify.Write {
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
