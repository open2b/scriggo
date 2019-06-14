// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"testing"
)

var rendererBuiltinTestsInHTMLContext = []struct {
	src     string
	res     string
	globals scope
}{
	// html
	{"html(``)", "", nil},
	{"html(`a`)", "a", nil},
	{"html(`<a>`)", "<a>", nil},
	{"html(a)", "<a>", scope{"a": "<a>"}},
	{"html(a)", "<a>", scope{"a": HTML("<a>")}},
	{"html(a) + html(b)", "<a><b>", scope{"a": "<a>", "b": "<b>"}},
}

func TestRenderBuiltinInHTMLContext(t *testing.T) {
	t.Skip("(not runnable)")
	// TODO(Gianluca):
	// for _, expr := range rendererBuiltinTestsInHTMLContext {
	// 	var tree, err = compiler.ParseSource([]byte("{{"+expr.src+"}}"), ast.ContextHTML)
	// 	if err != nil {
	// 		t.Errorf("source: %q, %s\n", expr.src, err)
	// 		continue
	// 	}
	// 	var b = &bytes.Buffer{}
	// 	err = RenderTree(b, tree, expr.globals, true)
	// 	if err != nil {
	// 		t.Errorf("source: %q, %s\n", expr.src, err)
	// 		continue
	// 	}
	// 	var res = b.String()
	// 	if res != expr.res {
	// 		t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res, expr.res)
	// 	}
	// }
}
