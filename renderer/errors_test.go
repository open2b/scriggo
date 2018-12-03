// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package renderer

import (
	"bytes"
	"testing"

	"open2b/template/ast"
	"open2b/template/parser"

	"github.com/shopspring/decimal"
)

var errorTests = []struct {
	src  string
	res  string
	vars scope
}{
	{`{% len = 5 %}{{ "ok" }}`, `ok`, nil},
	{`{% var a = "a" %}{% var a = "b" %}{{ "ok" }}`, `ok`, nil},
	{`{% if "a" == 5 %}{{ "no" }}{% end %}{{ "ok" }}`, `ok`, nil},
	{`{% if "a" == 5 %}{{ "no" }}{% else %}{{ "ok" }}{% end %}`, `ok`, nil},
	{`{% for a in false %}{{ "no" }}{% end %}{{ "ok" }}`, `ok`, nil},
	{`{% for a in false..10 %}{{ "no" }}{% end %}{{ "ok" }}`, `ok`, nil},
	{`{% for a in 1..false %}{{ "no" }}{% end %}{{ "ok" }}`, `ok`, nil},
	{`{{ "5" + 5 }}{{ "ok" }}`, `ok`, nil},
	{`{% if len() >= 0 %}{{ "no" }}{% else %}{{ "ok" }}{% end %}`, `ok`, nil},
	{`{% if len(nil) >= 0 %}{{ "no" }}{% else %}{{ "ok" }}{% end %}`, `ok`, nil},
	{`{% if len("a", "b") >= 0 %}{{ "no" }}{% else %}{{ "ok" }}{% end %}`, `ok`, nil},
	{`{% nil = 5 %}{{ "ok" }}`, `ok`, nil},
	{`{{ f(nil) }}{{ "ok" }}`, `ok`, map[string]interface{}{"f": func(s string) string { return "no" }}},
	{`{{ f(1) }}{{ "ok" }}`, `ok`, map[string]interface{}{"f": func() string { return "no" }}},
	{`{{ f(1, 2) }}{{ "ok" }}`, `ok`, map[string]interface{}{"f": func(i int) string { return "no" }}},
	{`{{ f(2) }}{{ "ok" }}`, `ok`, map[string]interface{}{"f": func(s string) string { return "no" }}},
	{`{{ f("2") }}{{ "ok" }}`, `ok`, map[string]interface{}{"f": func(n decimal.Decimal) string { return "no" }}},
}

func TestErrors(t *testing.T) {
	for _, expr := range errorTests {
		var tree, err = parser.ParseSource([]byte(expr.src), ast.ContextHTML)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		var e error
		err = RenderTree(b, tree, expr.vars, func(err error) bool {
			e = err
			t.Log(err)
			return true
		})
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		if e == nil {
			t.Errorf("source: %q, expecting error\n", expr.src)
		}
		var res = b.String()
		if res != expr.res {
			t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res, expr.res)
		}
	}
}
