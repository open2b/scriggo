// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/templates"

	"github.com/fsnotify/fsnotify"
	"github.com/yuin/goldmark"
)

func serve(asm, metrics bool) error {

	dir, err := newDirReader(".")
	if err != nil {
		return err
	}
	defer dir.Close()

	srv := &server{
		files:  dir,
		static: http.FileServer(http.Dir(".")),
		runOptions: &templates.RunOptions{
			MarkdownConverter: func(src []byte, out io.Writer) error {
				return goldmark.Convert(src, out)
			},
		},
		templates: map[string]*templates.Template{},
		asm:       asm,
	}
	if metrics {
		srv.metrics.active = true
		srv.metrics.header = true
	}
	go func() {
		for {
			select {
			case name := <-dir.Changed:
				delete(srv.templates, name)
			case err := <-dir.Errors:
				srv.logf("%v", err)
			}
		}
	}()

	s := &http.Server{
		Addr:           ":8080",
		Handler:        srv,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	fmt.Fprintln(os.Stderr, "Web server is available at http://localhost:8080/")
	fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop\n\n")

	return s.ListenAndServe()
}

type server struct {
	files      *dirReader
	static     http.Handler
	runOptions *templates.RunOptions
	asm        bool

	sync.Mutex
	templates map[string]*templates.Template
	metrics   struct {
		active bool
		header bool
	}
}

func (srv *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	name := r.URL.Path[1:]
	if name == "" || strings.HasSuffix(name, "/") {
		name += "index.html"
	}

	if ext := path.Ext(name); ext != ".html" && ext != ".md" {
		srv.static.ServeHTTP(w, r)
		return
	}

	var err error
	var buildTime time.Duration
	srv.Lock()
	template, ok := srv.templates[name]
	srv.Unlock()
	start := time.Now()
	if !ok {
		template, err = templates.Build(name, srv.files, nil)
		if err != nil {
			if err == templates.ErrNotExist {
				http.NotFound(w, r)
				return
			}
			if err, ok := err.(scriggo.CompilerError); ok {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(500)
				fmt.Fprintf(w, "%s", err)
				return
			}
			http.Error(w, "Internal Server Error", 500)
			srv.logf("%s", err)
			return
		}
		buildTime = time.Since(start)
		srv.Lock()
		srv.templates[name] = template
		srv.Unlock()
		start = time.Now()
	}
	b := bytes.Buffer{}
	err = template.Run(&b, nil, srv.runOptions)
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		fmt.Fprintf(w, "%s", err)
		return
	}
	runTime := time.Since(start)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = b.WriteTo(w)
	if err != nil {
		srv.logf("%s", err)
	}

	if srv.metrics.active {
		var header bool
		srv.Lock()
		header = srv.metrics.header
		srv.Unlock()
		if header {
			fmt.Fprintf(os.Stderr, "     %12s  %12s  %12s  %s\n", "Build", "Run", "Total", "File")
			fmt.Fprintf(os.Stderr, "     %12s  %12s  %12s  %s\n", "-----", "---", "-----", "----")
			srv.Lock()
			srv.metrics.header = false
			srv.Unlock()
		}
		if buildTime == 0 {
			fmt.Fprintf(os.Stderr, "     %12s  %12s  %12s  %s\n", "-", runTime, buildTime+runTime, name)
		} else {
			fmt.Fprintf(os.Stderr, "     %12s  %12s  %12s  %s\n", buildTime, runTime, buildTime+runTime, name)
		}
	}

	if srv.asm {
		asm := template.Disassemble(-1)
		srv.logf("\n--- Assembler %s ---\n", name)
		_, _ = os.Stderr.Write(asm)
		srv.log("-----------------\n")
	}

	return
}

func (srv *server) log(a ...interface{}) {
	println()
	fmt.Fprint(os.Stderr, a...)
	println()
	srv.metrics.header = true
}

func (srv *server) logf(format string, a ...interface{}) {
	println()
	fmt.Fprintf(os.Stderr, format, a...)
	println()
	srv.metrics.header = true
}

// dirReader implements a templates.FileReader that reads the files in a
// directory.
type dirReader struct {
	files   templates.DirReader
	watcher *fsnotify.Watcher
	watched map[string]bool
	Errors  chan error

	sync.Mutex
	Changed chan string
}

func newDirReader(root string) (*dirReader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	dir := &dirReader{
		files:   templates.DirReader(root),
		watcher: watcher,
		watched: map[string]bool{},
		Changed: make(chan string),
	}
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					dir.Changed <- event.Name
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

func (dir *dirReader) Close() error {
	return dir.watcher.Close()
}

// ReadFile implements the ReadFile method of templates.FileReader.
func (dir *dirReader) ReadFile(name string) ([]byte, error) {
	dir.Lock()
	if !dir.watched[name] {
		err := dir.watcher.Add(name[1:])
		if err != nil {
			dir.Unlock()
			return nil, err
		}
		dir.watched[name] = true
	}
	dir.Unlock()
	return dir.files.ReadFile(name)
}

