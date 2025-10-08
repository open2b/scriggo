// Copyright 2020 The Scriggo Authors. All rights reserved.
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
	"strings"
	"sync"
	"time"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/builtin"
	"github.com/open2b/scriggo/native"

	"github.com/yuin/goldmark"
)

// serve runs a web server and serves the template rooted at the current
// directory. metrics reports whether print the metrics. If asm is -1 or
// greater, serve prints the assembly code of the served file and the value of
// asm determines the maximum length, in runes, of disassembled Text
// instructions. If disableLiveReload is true, it disables LiveReload.
//
//	asm > 0: at most asm runes; leading and trailing white space are removed
//	asm == 0: no text
//	asm == -1: all text
func serve(asm int, metrics bool, disableLiveReload bool) error {

	fsys, err := newTemplateFS(".")
	if err != nil {
		return err
	}
	defer fsys.Close()

	md := goldmark.New(goldmarkOptions...)

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
	if !disableLiveReload {
		srv.liveReloads = map[*liveReload]struct{}{}
	}
	if metrics {
		srv.metrics.active = true
		srv.metrics.header = true
	}
	go func() {
		for {
			select {
			case name := <-fsys.Changed():
				srv.Lock()
				invalidated := map[string]bool{}
				if _, ok := srv.templates[name]; ok {
					delete(srv.templates, name)
					for _, dependents := range srv.templatesDependencies {
						delete(dependents, name)
					}
					invalidated[name] = true
				} else {
					for dependency, dependents := range srv.templatesDependencies {
						if dependency == name {
							for d := range dependents {
								delete(srv.templates, d)
								invalidated[d] = true
							}
						}
					}
					for invalidated := range invalidated {
						for _, dependents := range srv.templatesDependencies {
							delete(dependents, invalidated)
						}
					}
				}
				if len(invalidated) > 0 {
					for r := range srv.liveReloads {
						if invalidated[r.file+".html"] || invalidated[r.file+".md"] {
							go func() { r.reload() }()
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

type server struct {
	fsys        *templateFS
	static      http.Handler
	mdConverter scriggo.Converter
	runOptions  *scriggo.RunOptions
	asm         int

	sync.Mutex
	templates             map[string]*scriggo.Template
	templatesDependencies map[string]map[string]struct{}
	liveReloads           map[*liveReload]struct{}
	metrics               struct {
		active bool
		header bool
	}
}

func (srv *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Accept") == "text/event-stream" {
		srv.serveLiveReload(w, r)
		return
	}
	srv.serveTemplate(w, r)
}

type liveReload struct {
	file string // file path without extension

	sync.Mutex
	w io.Writer
}

var reload = []byte("data: reload\n\n")

func (lr *liveReload) reload() {
	lr.Lock()
	_, _ = lr.w.Write(reload)
	lr.w.(http.Flusher).Flush()
	lr.Unlock()
}

func (srv *server) serveLiveReload(w http.ResponseWriter, r *http.Request) {

	if _, ok := w.(http.Flusher); !ok || srv.liveReloads == nil {
		http.Error(w, "Live reload is not supported", http.StatusUnsupportedMediaType)
		return
	}

	file := r.URL.Path[1:]
	if file == "" || strings.HasSuffix(file, "/") {
		file += "index"
	}
	if ext := path.Ext(file); ext != "" {
		file = file[:len(file)-len(ext)]
	}

	lr := &liveReload{
		file: file,
		w:    w,
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(200)

	srv.Lock()
	srv.liveReloads[lr] = struct{}{}
	_, ok := srv.templates[file+".html"]
	if !ok {
		_, ok = srv.templates[file+".md"]
	}
	srv.Unlock()

	if !ok {
		lr.reload()
	}

	<-r.Context().Done()

	srv.Lock()
	delete(srv.liveReloads, lr)
	srv.Unlock()

}

func (srv *server) serveTemplate(w http.ResponseWriter, r *http.Request) {

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
		fs := newRecFS(srv.fsys)
		opts := scriggo.BuildOptions{
			AllowGoStmt:       true,
			MarkdownConverter: srv.mdConverter,
			Globals:           make(native.Declarations, len(globals)+1),
		}
		for n, v := range globals {
			opts.Globals[n] = v
		}
		opts.Globals["filepath"] = strings.TrimSuffix(name, path.Ext(name))
		template, err = scriggo.BuildTemplate(fs, name, &opts)
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
		for _, dependency := range fs.RecordedNames() {
			if _, ok := srv.templatesDependencies[dependency]; !ok {
				srv.templatesDependencies[dependency] = map[string]struct{}{}
			}
			srv.templatesDependencies[dependency][name] = struct{}{}
		}
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

	s := b.Bytes()
	i := indexEndBody(s)
	if i == -1 {
		i = len(s)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if srv.liveReloads == nil {
		_, _ = w.Write(s)
	} else {
		_, _ = w.Write(s[:i])
		_, _ = io.WriteString(w, `<script>
	// Scriggo 	LiveReload.
	(function () {
		if (typeof EventSource !== 'function') return;
		const es = new EventSource('`)
		jsStringEscape(w, r.URL.Path)
		_, _ = io.WriteString(w, `');
		es.onmessage = function(e) {
			if (e.data === 'reload') {
				es.close();
				location.reload();
			}
		};
	})();
</script>`)
		_, _ = w.Write(s[i:])
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

// indexEndBody locates the closing "</body>" tag in s, ignoring case and
// permitting spaces before '>'. If the tag is found, the function returns its
// index, unless the tag's line contains only whitespaces before it; in that
// case, it returns the index of the newline ('\n') at the start of the line.
// If no closing tag is found, it returns -1.
func indexEndBody(s []byte) int {
	for {
		i := bytes.LastIndex(s, []byte("</"))
		if i == -1 {
			return -1
		}
		if isBodyTag(s[i+2:]) {
			for j := i - 1; j >= 0; j-- {
				switch s[j] {
				case ' ', '\t', '\r':
					continue
				case '\n':
					if j > 0 && s[j-1] == '\r' {
						j--
					}
					return j
				}
				break
			}
			return i
		}
		s = s[:i]
	}
}

// isBodyTag checks if s starts with "body>", ignoring case and allowing spaces
// before '>'.
func isBodyTag(s []byte) bool {
	if len(s) < 5 {
		return false
	}
	if !bytes.EqualFold(s[:4], []byte("body")) {
		return false
	}
	for i := 4; i < len(s); i++ {
		switch s[i] {
		case '>':
			return true
		case ' ', '\t', '\n', '\r':
			continue
		}
		break
	}
	return false
}
