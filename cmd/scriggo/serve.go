// Copyright 2020 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/builtin"
	"github.com/open2b/scriggo/native"

	"github.com/fsnotify/fsnotify"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
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

	md := goldmark.New(
		goldmark.WithRendererOptions(html.WithUnsafe()),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithExtensions(extension.GFM))

	srv := &server{
		fsys:   fsys,
		static: http.FileServer(http.Dir(".")),
		mdConverter: func(src []byte, out io.Writer) error {
			return md.Convert(src, out)
		},
		templates:             map[string]*scriggo.Template{},
		templatesDependencies: map[string]map[string]struct{}{},
		asm:                   asm,
	}
	if metrics {
		srv.metrics.active = true
		srv.metrics.header = true
	}
	go func() {
		for {
			select {
			case name := <-fsys.Changed:
				srv.Lock()
				if _, ok := srv.templates[name]; ok {
					delete(srv.templates, name)
					for _, dependents := range srv.templatesDependencies {
						delete(dependents, name)
					}
				} else {
					var invalidatedFiles []string
					for dependency, dependents := range srv.templatesDependencies {
						if dependency == name {
							for d := range dependents {
								delete(srv.templates, d)
								invalidatedFiles = append(invalidatedFiles, d)
							}
						}
					}
					for _, invalidated := range invalidatedFiles {
						for _, dependents := range srv.templatesDependencies {
							delete(dependents, invalidated)
						}
					}
				}
				srv.Unlock()
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

func (srv *server) updateTemplateDependencies(tree *ast.Tree) error {

	var treeNavigation func(*ast.Tree, bool) []string

	treeNavigation = func(tree *ast.Tree, includeRoot bool) []string {
		var dependencies []string
		if includeRoot {
			dependencies = []string{tree.Path}
		}
		for _, n := range tree.Nodes {
			switch node := n.(type) {
			case *ast.Show:
				for _, e := range node.Expressions {
					switch expr := e.(type) {
					case *ast.Render:
						dependencies = append(dependencies, treeNavigation(expr.Tree, true)...)
					}
				}
			case *ast.Import:
				dependencies = append(dependencies, treeNavigation(node.Tree, true)...)
			case *ast.Extends:
				dependencies = append(dependencies, treeNavigation(node.Tree, true)...)
			default:
			}
		}
		return dependencies
	}

	dependencies := treeNavigation(tree, false)
	srv.Lock()
	for _, dependency := range dependencies {
		if _, ok := srv.templatesDependencies[dependency]; !ok {
			srv.templatesDependencies[dependency] = map[string]struct{}{}
		}
		srv.templatesDependencies[dependency][tree.Path] = struct{}{}
	}
	srv.Unlock()

	return nil
}

type server struct {
	fsys        *templateFS
	static      http.Handler
	mdConverter scriggo.Converter
	runOptions  *scriggo.RunOptions
	asm         int

	sync.Mutex
	templates             map[string]*scriggo.Template
	templatesDependencies map[string]map[string]struct{}
	metrics               struct {
		active bool
		header bool
	}
}

func (srv *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	name := r.URL.Path[1:]
	if name == "" || strings.HasSuffix(name, "/") {
		name += "index"
	}

	if ext := path.Ext(name); ext == "" {
		fi, err := srv.fsys.Open(name + ".html")
		if err == nil {
			name += ".html"
			_ = fi.Close()
		} else {
			if !errors.Is(err, os.ErrNotExist) {
				http.Error(w, "Internal Server Error", 500)
				return
			}
			fi, err = srv.fsys.Open(name + ".md")
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					http.NotFound(w, r)
				} else {
					http.Error(w, "Internal Server Error", 500)
				}
				return
			}
			name += ".md"
			_ = fi.Close()
		}
	} else if ext != ".html" && ext != ".md" {
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
		opts := scriggo.BuildOptions{
			AllowGoStmt:       true,
			MarkdownConverter: srv.mdConverter,
			Globals:           make(native.Declarations, len(globals)+1),
			TreeTransformer:   srv.updateTemplateDependencies,
		}
		for n, v := range globals {
			opts.Globals[n] = v
		}
		opts.Globals["filepath"] = strings.TrimSuffix(name, path.Ext(name))
		template, err = scriggo.BuildTemplate(srv.fsys, name, &opts)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.NotFound(w, r)
				return
			}
			if err, ok := err.(*scriggo.BuildError); ok {
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
	vars := map[string]interface{}{"form": builtin.NewFormData(r, 10)}
	err = template.Run(&b, vars, srv.runOptions)
	if err != nil {
		switch err {
		case builtin.ErrBadRequest:
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		case builtin.ErrRequestEntityTooLarge:
			http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		default:
			http.Error(w, err.Error(), 500)
		}
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
		fsys:    os.DirFS(root),
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
					dir.Changed <- strings.ReplaceAll(event.Name, "\\", "/")
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

var globals = native.Declarations{
	// crypto
	"hmacSHA1":   builtin.HmacSHA1,
	"hmacSHA256": builtin.HmacSHA256,
	"sha1":       builtin.Sha1,
	"sha256":     builtin.Sha256,

	// debug
	"version": version(),

	// encoding
	"base64":            builtin.Base64,
	"hex":               builtin.Hex,
	"marshalJSON":       builtin.MarshalJSON,
	"marshalJSONIndent": builtin.MarshalJSONIndent,
	"md5":               builtin.Md5,
	"unmarshalJSON":     builtin.UnmarshalJSON,

	// html
	"htmlEscape": builtin.HtmlEscape,

	// math
	"abs": builtin.Abs,
	"max": builtin.Max,
	"min": builtin.Min,
	"pow": builtin.Pow,

	// net
	"File":        reflect.TypeOf((*builtin.File)(nil)).Elem(),
	"FormData":    reflect.TypeOf(builtin.FormData{}),
	"form":        (*builtin.FormData)(nil),
	"queryEscape": builtin.QueryEscape,

	// regexp
	"Regexp": reflect.TypeOf(builtin.Regexp{}),
	"regexp": builtin.RegExp,

	// sort
	"reverse": builtin.Reverse,
	"sort":    builtin.Sort,

	// strconv
	"formatFloat": builtin.FormatFloat,
	"formatInt":   builtin.FormatInt,
	"parseFloat":  builtin.ParseFloat,
	"parseInt":    builtin.ParseInt,

	// strings
	"abbreviate":    builtin.Abbreviate,
	"capitalize":    builtin.Capitalize,
	"capitalizeAll": builtin.CapitalizeAll,
	"hasPrefix":     builtin.HasPrefix,
	"hasSuffix":     builtin.HasSuffix,
	"index":         builtin.Index,
	"indexAny":      builtin.IndexAny,
	"join":          builtin.Join,
	"lastIndex":     builtin.LastIndex,
	"replace":       builtin.Replace,
	"replaceAll":    builtin.ReplaceAll,
	"runeCount":     builtin.RuneCount,
	"split":         builtin.Split,
	"splitAfter":    builtin.SplitAfter,
	"splitAfterN":   builtin.SplitAfterN,
	"splitN":        builtin.SplitN,
	"sprint":        builtin.Sprint,
	"sprintf":       builtin.Sprintf,
	"toKebab":       builtin.ToKebab,
	"toLower":       builtin.ToLower,
	"toUpper":       builtin.ToUpper,
	"trim":          builtin.Trim,
	"trimLeft":      builtin.TrimLeft,
	"trimPrefix":    builtin.TrimPrefix,
	"trimRight":     builtin.TrimRight,
	"trimSuffix":    builtin.TrimSuffix,

	// time
	"Duration":      reflect.TypeOf(builtin.Duration(0)),
	"Hour":          time.Hour,
	"Microsecond":   time.Microsecond,
	"Millisecond":   time.Millisecond,
	"Minute":        time.Minute,
	"Nanosecond":    time.Nanosecond,
	"Second":        time.Second,
	"Time":          reflect.TypeOf(builtin.Time{}),
	"date":          builtin.Date,
	"now":           builtin.Now,
	"parseDuration": builtin.ParseDuration,
	"parseTime":     builtin.ParseTime,
	"unixTime":      builtin.UnixTime,

	// unsafeconv
	"unsafeconv": builtin.Unsafeconv,
}
