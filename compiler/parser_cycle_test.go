// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"testing"

	"github.com/open2b/scriggo/compiler/ast"

	"github.com/google/go-cmp/cmp"
)

var cyclicProgram = mapStringLoader{
	"main":       "package main\nimport _ \"cycle/foo\"\nfunc main() {}",
	"cycle/foo":  "package foo\nimport _ \"cycle/foo2\"",
	"cycle/foo2": "package foo2\nimport _ \"cycle/foo3\"",
	"cycle/foo3": "package foo3\nimport _ \"cycle/foo\"",
}

const cyclicProgramErrorMessage = `package main
	imports cycle/foo
	imports cycle/foo2
	imports cycle/foo3
	imports cycle/foo: import cycle not allowed`

func TestCyclicProgram(t *testing.T) {
	_, err := ParseProgram(cyclicProgram)
	if err == nil {
		t.Fatal("expecting cycle error, got no error")
	}
	if diff := cmp.Diff(cyclicProgramErrorMessage, err.Error()); diff != "" {
		t.Fatalf("unexpected error message (-want, +got):\n%s", diff)
	}
}

var cyclicTemplate = mapStringReader{
	"/index.html":            `{% extends "/layout.html" %}`,
	"/layout.html":           `{% import "/macros/macro.html" %}`,
	"/includes/include.html": `{% import "/macros/macro.html" %}`,
	"/macros/macro.html":     `{% macro A() %}\n\t{% include "/includes/include.html" %}\n{% end %}`,
}

const cyclicTemplateErrorMessage = `file /index.html
	extends  /layout.html
	imports  /macros/macro.html
	includes /includes/include.html
	imports  /macros/macro.html: cycle not allowed`

func TestTCyclicTemplate(t *testing.T) {
	_, err := ParseTemplate("index.html", cyclicTemplate, ast.LanguageHTML, false, nil)
	if err == nil {
		t.Fatal("expecting cycle error, got no error")
	}
	if diff := cmp.Diff(cyclicTemplateErrorMessage, err.Error()); diff != "" {
		t.Fatalf("unexpected error message (-want, +got):\n%s", diff)
	}
}

type mapStringReader map[string]string

func (r mapStringReader) ReadFile(name string) ([]byte, error) {
	src, ok := r[name]
	if !ok {
		panic("not existing")
	}
	return []byte(src), nil
}
