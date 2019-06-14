// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"testing"
)

var rendererBuiltinTestsInHTMLContext = []struct {
	src  string
	res  string
	vars Vars
}{
	{"{{ html(``) }}", "", nil},
	{"{{ html(`a`) }}", "a", nil},
	{"{{ html(`<a>`) }}", "<a>", nil},
	{"{{ html(a) }}", "<a>", Vars{"a": "<a>"}},
	{"{{ html(a) }}", "<a>", Vars{"a": HTML("<a>")}},
	//{"{{ html(a) + html(b) }}", "<a><b>", Vars{"a": "<a>", "b": "<b>"}}, TODO: reflect: call of reflect.Value.Set on zero Value

	{"{{ escape(``) }}", "", nil},
	{"{{ escape(`a`) }}", "a", nil},
	{"{{ escape(`<a>`) }}", "&lt;a&gt;", nil},
	{"{{ escape(a) }}", "&lt;a&gt;", Vars{"a": "<a>"}},

	{"{% s := []html{html(`<b>`), html(`<a>`), html(`<c>`)} %}{% sort(s) %}{{ s }}", "<a>, <b>, <c>", nil},
}

func TestRenderBuiltinInHTMLContext(t *testing.T) {
	for _, expr := range rendererBuiltinTestsInHTMLContext {
		r := MapReader{"/index.html": []byte(expr.src)}
		tmpl, err := Load("/index.html", r, mainPackage(expr.vars), ContextHTML, LimitMemorySize)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, RenderOptions{MaxMemorySize: 1000})
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var res = b.String()
		if res != expr.res {
			t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res, expr.res)
		}
	}
}
