// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"path"
	"testing"

	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/internal/mapfs"

	"github.com/google/go-cmp/cmp"
)

var cycleProgramTests = []struct {
	name    string
	program mapStringLoader
	path    string
	pos     ast.Position
	msg     string
}{

	{
		name: "Package import cycle",
		program: mapStringLoader{
			"main":       "package main\nimport _ \"cycle/foo\"\nfunc main() {}",
			"cycle/foo":  "package foo\nimport _ \"cycle/foo2\"",
			"cycle/foo2": "package foo2\nimport _ \"cycle/foo3\"",
			"cycle/foo3": "package foo3\nimport _ \"cycle/foo\"",
		},
		path: "cycle/foo",
		pos:  ast.Position{Line: 2, Column: 8, Start: 20, End: 32},
		msg: `package main
	imports cycle/foo
	imports cycle/foo2
	imports cycle/foo3
	imports cycle/foo: import cycle not allowed`,
	},

	{
		name: "Package that imports itself",
		program: mapStringLoader{
			"main":      "package main\nimport _ \"cycle/foo\"\nfunc main() {}",
			"cycle/foo": "package foo\nimport _ \"cycle/foo\"",
		},
		path: "cycle/foo",
		pos:  ast.Position{Line: 2, Column: 8, Start: 19, End: 31},
		msg: `package main
	imports cycle/foo
	imports cycle/foo: import cycle not allowed`,
	},
}

func TestCyclicPrograms(t *testing.T) {
	for _, test := range cycleProgramTests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseProgram(test.program)
			if err == nil {
				t.Fatal("expecting cycle error, got no error")
			}
			e, ok := err.(*CycleError)
			if !ok {
				t.Fatalf("expecting *CycleError value, got %T value", err)
			}
			if diff := cmp.Diff(test.msg, e.Error()); diff != "" {
				t.Fatalf("unexpected error message (-want, +got):\n%s", diff)
			}
			if e.path != test.path {
				t.Fatalf(`expecting path %q, got %q`, test.path, e.path)
			}
			if e.pos != test.pos {
				t.Fatalf(`expecting %#v, got %#v`, test.pos, e.pos)
			}
		})
	}
}

var cycleTemplateTests = []struct {
	name string
	fsys mapfs.MapFS
	path string
	pos  ast.Position
	msg  string
}{

	{
		name: "Template cycle",
		fsys: mapfs.MapFS{
			"index.html":            `{% extends "/layout.html" %}`,
			"layout.html":           `{% import "/macros/macro.html" %}`,
			"partials/partial.html": `{% import "/macros/macro.html" %}`,
			"macros/macro.html":     `{% macro A() %}\n\t{{ render "/partials/partial.html" }}\n{% end %}`,
		},
		path: "macros/macro.html",
		pos:  ast.Position{Line: 1, Column: 23, Start: 22, End: 53},
		msg: `file index.html
	extends layout.html
	imports macros/macro.html
	renders partials/partial.html
	imports macros/macro.html: cycle not allowed`,
	},

	{
		name: "Template cycle on index file",
		fsys: mapfs.MapFS{
			"index.html":        `{% extends "/layout.html" %}`,
			"layout.html":       `{% import "/macros/macro.html" %}`,
			"macros/macro.html": `{% macro A() %}\n\t{{ render "/index.html" }}\n{% end %}`,
		},
		path: "index.html",
		pos:  ast.Position{Line: 1, Column: 4, Start: 3, End: 24},
		msg: `file index.html
	extends layout.html
	imports macros/macro.html
	renders index.html: cycle not allowed`,
	},

	{
		name: "Template cycle on last rendered file",
		fsys: mapfs.MapFS{
			"index.html":        `{% extends "/layout.html" %}`,
			"layout.html":       `{% import "/macros/macro.html" %}`,
			"macros/macro.html": `{% macro A() %}\n\t{{ render "/macros/macro.html" }}\n{% end %}`,
		},
		path: "macros/macro.html",
		pos:  ast.Position{Line: 1, Column: 23, Start: 22, End: 49},
		msg: `file index.html
	extends layout.html
	imports macros/macro.html
	renders macros/macro.html: cycle not allowed`,
	},

	{
		name: "Template file that extends itself",
		fsys: mapfs.MapFS{
			"index.html": `{% extends "/index.html" %}`,
		},
		path: "index.html",
		pos:  ast.Position{Line: 1, Column: 4, Start: 3, End: 23},
		msg: `file index.html
	extends index.html: cycle not allowed`,
	},

	{
		name: "Template file that imports itself",
		fsys: mapfs.MapFS{
			"index.html": `{% import "/index.html" %}`,
		},
		path: "index.html",
		pos:  ast.Position{Line: 1, Column: 11, Start: 10, End: 22},
		msg: `file index.html
	imports index.html: cycle not allowed`,
	},

	{
		name: "Template file that render itself",
		fsys: mapfs.MapFS{
			"index.html": `{{ render "/index.html" }}`,
		},
		path: "index.html",
		pos:  ast.Position{Line: 1, Column: 4, Start: 3, End: 23},
		msg: `file index.html
	renders index.html: cycle not allowed`,
	},
}

func TestCyclicTemplates(t *testing.T) {
	for _, test := range cycleTemplateTests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseTemplate(test.fsys, "index.html", nil)
			if err == nil {
				t.Fatal("expecting cycle error, got no error")
			}
			e, ok := err.(*CycleError)
			if !ok {
				t.Fatalf("expecting *CycleError value, got %T value", err)
			}
			if diff := cmp.Diff(test.msg, e.Error()); diff != "" {
				t.Fatalf("unexpected error message (-want, +got):\n%s", diff)
			}
			if e.path != test.path {
				t.Fatalf(`expecting path %q, got %q`, test.path, e.path)
			}
			if e.pos != test.pos {
				t.Fatalf(`expecting %#v, got %#v`, test.pos, e.pos)
			}
		})
	}
}

type mapStringReader map[string]string

func (r mapStringReader) ReadFile(name string) ([]byte, ast.Format, error) {
	src, ok := r[name]
	if !ok {
		panic("not existing")
	}
	format := ast.FormatText
	switch path.Ext(name) {
	case ".html":
		format = ast.FormatHTML
	case ".css":
		format = ast.FormatCSS
	case ".js":
		format = ast.FormatJS
	case ".json":
		format = ast.FormatJSON
	}
	return []byte(src), format, nil
}
