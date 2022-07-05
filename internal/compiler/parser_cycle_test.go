// Copyright 2020 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"testing"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/fstest"
)

var cycleProgramTests = []struct {
	name    string
	program fstest.Files
	path    string
	pos     ast.Position
	msg     string
}{

	{
		name: "Package import cycle",
		program: fstest.Files{
			"go.mod":       "module cycle",
			"main.go":      "package main\nimport _ \"cycle/foo\"\nfunc main() {}",
			"foo/foo.go":   "package foo\nimport _ \"cycle/foo2\"",
			"foo2/foo2.go": "package foo2\nimport _ \"cycle/foo3\"",
			"foo3/foo3.go": "package foo3\nimport _ \"cycle/foo\"",
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
		program: fstest.Files{
			"go.mod":     "module cycle",
			"main.go":    "package main\nimport _ \"cycle/foo\"\nfunc main() {}",
			"foo/foo.go": "package foo\nimport _ \"cycle/foo\"",
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
			if test.msg != e.Error() {
				t.Fatalf("expecting error message %q, got %q", test.msg, e.Error())
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
	fsys fstest.Files
	path string
	pos  ast.Position
	msg  string
}{

	{
		name: "Template cycle",
		fsys: fstest.Files{
			"index.html":            `{% extends "/layout.html" %}`,
			"layout.html":           `{% import "/macros/macro.html" %}`,
			"partials/partial.html": `{% import "/macros/macro.html" %}`,
			"macros/macro.html":     `{% macro A() %}\n\t{{ render "/partials/partial.html" }}\n{% end %}`,
		},
		path: "macros/macro.html",
		pos:  ast.Position{Line: 1, Column: 23, Start: 22, End: 52},
		msg: `file index.html
	extends layout.html
	imports macros/macro.html
	renders partials/partial.html
	imports macros/macro.html: cycle not allowed`,
	},

	{
		name: "Template cycle on index file",
		fsys: fstest.Files{
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
		fsys: fstest.Files{
			"index.html":        `{% extends "/layout.html" %}`,
			"layout.html":       `{% import "/macros/macro.html" %}`,
			"macros/macro.html": `{% macro A() %}\n\t{{ render "/macros/macro.html" }}\n{% end %}`,
		},
		path: "macros/macro.html",
		pos:  ast.Position{Line: 1, Column: 23, Start: 22, End: 48},
		msg: `file index.html
	extends layout.html
	imports macros/macro.html
	renders macros/macro.html: cycle not allowed`,
	},

	{
		name: "Template file that extends itself",
		fsys: fstest.Files{
			"index.html": `{% extends "/index.html" %}`,
		},
		path: "index.html",
		pos:  ast.Position{Line: 1, Column: 4, Start: 3, End: 23},
		msg: `file index.html
	extends index.html: cycle not allowed`,
	},

	{
		name: "Template file that imports itself",
		fsys: fstest.Files{
			"index.html": `{% import "/index.html" %}`,
		},
		path: "index.html",
		pos:  ast.Position{Line: 1, Column: 11, Start: 10, End: 22},
		msg: `file index.html
	imports index.html: cycle not allowed`,
	},

	{
		name: "Template file that render itself",
		fsys: fstest.Files{
			"index.html": `{{ render "/index.html" }}`,
		},
		path: "index.html",
		pos:  ast.Position{Line: 1, Column: 4, Start: 3, End: 22},
		msg: `file index.html
	renders index.html: cycle not allowed`,
	},
}

func TestCyclicTemplates(t *testing.T) {
	for _, test := range cycleTemplateTests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseTemplate(test.fsys, "index.html", false)
			if err == nil {
				t.Fatal("expecting cycle error, got no error")
			}
			e, ok := err.(*CycleError)
			if !ok {
				t.Fatalf("expecting *CycleError value, got %T value", err)
			}
			if test.msg != e.Error() {
				t.Fatalf("expecting error message %q, got %q", test.msg, e.Error())
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
