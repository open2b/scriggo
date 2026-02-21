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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/builtin"
	"github.com/open2b/scriggo/native"

	"github.com/yuin/goldmark"
)

const (
	defaultHost  = "localhost"
	defaultHPort = "8080"
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
func serve(asm int, metrics bool, disableLiveReload bool, httpValue string, httpFlagSet bool, consts []string) error {

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
	if len(consts) > 0 {
		srv.consts = make(native.Declarations, len(consts))
		for _, c := range consts {
			err := parseConstants(c, srv.consts)
			if err != nil {
				return err
			}
		}
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

	var addr string
	if httpFlagSet {
		addr, err = parseAddr(httpValue)
		if err != nil {
			return err
		}
	} else {
		addr = defaultHost + ":" + defaultHPort
	}

	s := &http.Server{
		Addr:           addr,
		Handler:        srv,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	fmt.Fprintf(os.Stderr, "Web server is available at http://%s/\n", addr)
	fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop\n\n")

	return s.ListenAndServe()
}

type server struct {
	fsys        *templateFS
	static      http.Handler
	mdConverter scriggo.Converter
	runOptions  *scriggo.RunOptions
	asm         int
	consts      native.Declarations

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

// parseAddr returns listen addr or an error.
// Empty host => defaultHost, empty port => defaultPort.
func parseAddr(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("invalid -http value: missing host and port")
	}
	if strings.Count(s, ":") > 1 {
		return "", fmt.Errorf("invalid -http value %q: too many ':' characters", s)
	}

	host, port, found := strings.Cut(s, ":")
	if found {
		host = strings.TrimSpace(host)
		port = strings.TrimSpace(port)
	} else {
		host = s
	}

	if host == "" && port == "" {
		return "", fmt.Errorf("invalid -http value: missing host and port")
	}
	if host == "" {
		host = defaultHost
	}
	if port == "" {
		return host + ":" + defaultHPort, nil
	}

	for _, r := range port {
		if '0' <= r && r <= '9' {
			continue
		}
		return "", fmt.Errorf("invalid -http port %q: must be numeric", port)
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return "", fmt.Errorf("invalid -http port %q: must be between 1 and 65535", port)
	}

	return host + ":" + port, nil
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
	_, ok := srv.templates[file+".html"]
	if !ok {
		_, ok = srv.templates[file+".md"]
	}
	if !ok {
		_, ok = srv.templates[file+"/index.html"]
		if !ok {
			_, ok = srv.templates[file+"/index.md"]
		}
		if ok {
			lr.file += "/index"
		}
	}
	srv.liveReloads[lr] = struct{}{}
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
		var err error
		name, err = srv.resolvePath(name, true)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.NotFound(w, r)
			} else {
				http.Error(w, "Internal Server Error", 500)
			}
			return
		}
	} else if ext != ".html" {
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
			Globals:           make(native.Declarations, len(globals)+len(srv.consts)+1),
		}
		for n, v := range globals {
			opts.Globals[n] = v
		}
		for n, v := range srv.consts {
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if template.Format() == scriggo.FormatMarkdown {
		md := goldmark.New(goldmarkOptions...)
		var body bytes.Buffer
		err = md.Convert(b.Bytes(), &body)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
		var urlPath string
		if srv.liveReloads != nil {
			urlPath = r.URL.Path
		}
		writeWrappedBody(w, urlPath, name, &body)
	} else {
		if srv.liveReloads == nil {
			_, _ = b.WriteTo(w)
		} else {
			s := b.Bytes()
			i := indexEndBody(s)
			if i == -1 {
				i = len(s)
			}
			_, _ = w.Write(s[:i])
			_, _ = writeLiveReloadScript(w, r.URL.Path)
			_, _ = w.Write(s[i:])
		}
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

// resolvePath resolves a URL path to a local filesystem path. If descend is
// true and the target is a directory, it also checks for an "index" file within
// that directory.
func (srv *server) resolvePath(name string, descend bool) (string, error) {
	fi, err := srv.fsys.Open(name + ".html")
	if err == nil {
		_ = fi.Close()
		return name + ".html", nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	fi, err = srv.fsys.Open(name + ".md")
	if err == nil {
		_ = fi.Close()
		return name + ".md", nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if !descend {
		return "", os.ErrNotExist
	}
	return srv.resolvePath(name+"/index", false)
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

// writeWrappedBody writes a complete HTML page to w. It inserts body between
// the <body> tags. If urlPath is non-empty, it includes a live reload script
// using urlPath. The title parameter specifies the page title.
func writeWrappedBody(w io.Writer, urlPath, title string, body *bytes.Buffer) {
	_, _ = io.WriteString(w, `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>`)
	_, _ = io.WriteString(w, title)
	_, _ = io.WriteString(w, `</title>
  <style>
    body {
      margin: 0 auto;
      max-width: 70ch;
      padding: 0 16px;
      font-family: system-ui, -apple-system, sans-serif;
      line-height: 1.6;
    }
    body > :first-child {
        margin-top: 16px;
    }
    h1,h2,h3 { line-height: 1.25; margin-top: 1.4em; }
    pre {
      background: #f3f4f6;
      padding: 1rem;
      overflow: auto;
      border-radius: 8px;
    }
    code {
      background: #f3f4f6;
      padding: 0.2em 0.4em;
      border-radius: 4px;
    }
    table {
      border-collapse: collapse;
      width: 100%;
    }
    th, td {
      border: 1px solid #ddd;
      padding: 0.5rem;
    }
  </style>
</head>
<body>
`)
	_, _ = body.WriteTo(w)
	if urlPath != "" {
		_, _ = writeLiveReloadScript(w, urlPath)
	}
	_, _ = io.WriteString(w, `
</body>
</html>`)
}

// writeLiveReloadScript writes a live reload script for urlPath to w.
// It returns the number of bytes written and any error.
func writeLiveReloadScript(w io.Writer, urlPath string) (int, error) {
	n, err := io.WriteString(w, `<script>
	// Scriggo 	LiveReload.
	(function () {
		if (typeof EventSource !== 'function') return;
		const es = new EventSource('`)

	jsStringEscape(w, urlPath)
	_, _ = io.WriteString(w, `');
		es.onmessage = function(e) {
			if (e.data === 'reload') {
				es.close();
				location.reload();
			}
		};
	})();
</script>`)
	return n, err
}
