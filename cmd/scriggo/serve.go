// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/fs"
	"github.com/open2b/scriggo/templates"
	"github.com/open2b/scriggo/templates/builtin"

	"github.com/fsnotify/fsnotify"
	"github.com/yuin/goldmark"
)

// serve runs a web server and serves the template rooted at the current
// directory. metrics reports whether print the metrics. If asm is -1 or
// greater, serve prints the assembly code of the served file and the value of
// asm determines the maximum length, in runes, of disassembled Text
// instructions
//
//   asm > 0: at most asm runes; leading and trailing white space are removed
//   asm == 0: no text
//   asm == -1: all text
//
func serve(asm int, metrics bool) error {

	fsys, err := newTemplateFS(".")
	if err != nil {
		return err
	}
	defer fsys.Close()

	srv := &server{
		fsys:   fsys,
		static: http.FileServer(http.Dir(".")),
		buildOptions: &templates.BuildOptions{
			Globals: globals,
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
			case name := <-fsys.Changed:
				delete(srv.templates, name)
			case err := <-fsys.Errors:
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
	fsys         *templateFS
	static       http.Handler
	buildOptions *templates.BuildOptions
	runOptions   *templates.RunOptions
	asm          int

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
		template, err = templates.Build(srv.fsys, name, srv.buildOptions)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
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

	if srv.asm >= -1 {
		asm := template.Disassemble(srv.asm)
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

// templateFS implements a file system that reads the files in a directory.
type templateFS struct {
	fsys    fs.FS
	watcher *fsnotify.Watcher
	watched map[string]bool
	Errors  chan error

	sync.Mutex
	Changed chan string
}

func newTemplateFS(root string) (*templateFS, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	dir := &templateFS{
		fsys:    fs.DirFS(root),
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

func (t *templateFS) Close() error {
	return t.watcher.Close()
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

var globals = templates.Declarations{
	"Regexp":        reflect.TypeOf((*builtin.Regexp)(nil)).Elem(),
	"abbreviate":    builtin.Abbreviate,
	"abs":           builtin.Abs,
	"base64":        builtin.Base64,
	"capitalize":    builtin.Capitalize,
	"capitalizeAll": builtin.CapitalizeAll,
	"date":          builtin.Date,
	"hasPrefix":     builtin.HasPrefix,
	"hasSuffix":     builtin.HasSuffix,
	"hex":           builtin.Hex,
	"hmacSHA1":      builtin.HmacSHA1,
	"hmacSHA256":    builtin.HmacSHA256,
	"htmlEscape":    builtin.HtmlEscape,
	"index":         builtin.Index,
	"indexAny":      builtin.IndexAny,
	"join":          builtin.Join,
	"lastIndex":     builtin.LastIndex,
	"max":           builtin.Max,
	"md5":           builtin.Md5,
	"min":           builtin.Min,
	"now":           builtin.Now,
	"parseTime":     builtin.ParseTime,
	"queryEscape":   builtin.QueryEscape,
	"regexp":        builtin.RegExp,
	"replace":       builtin.Replace,
	"replaceAll":    builtin.ReplaceAll,
	"reverse":       builtin.Reverse,
	"runeCount":     builtin.RuneCount,
	"sha1":          builtin.Sha1,
	"sha256":        builtin.Sha256,
	"sort":          builtin.Sort,
	"split":         builtin.Split,
	"splitN":        builtin.SplitN,
	"sprint":        builtin.Sprint,
	"sprintf":       builtin.Sprintf,
	"time":          builtin.PackageTime,
	"toKebab":       builtin.ToKebab,
	"toLower":       builtin.ToLower,
	"toUpper":       builtin.ToUpper,
	"trim":          builtin.Trim,
	"trimLeft":      builtin.TrimLeft,
	"trimPrefix":    builtin.TrimPrefix,
	"trimRight":     builtin.TrimRight,
	"trimSuffix":    builtin.TrimSuffix,
	"unixTime":      builtin.UnixTime,
}
