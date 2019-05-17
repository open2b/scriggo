// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"testing"

	"scrigo/internal/compiler"
	"scrigo/internal/compiler/ast"

	"github.com/cockroachdb/apd"
)

var errorTests = []struct {
	src     string
	res     string
	globals scope
}{
	{`{% len = 5 %}{{ "ok" }}`, `ok`, nil},
	{`{% a := "a" %}{% a := "b" %}{{ "ok" }}`, `ok`, nil},
	{`{% for a in false %}{{ "no" }}{% end %}{{ "ok" }}`, `ok`, nil},
	{`{{ "5" + 5 }}{{ "ok" }}`, `ok`, nil},
	{`{% if len() >= 0 %}{{ "no" }}{% else %}{{ "ok" }}{% end %}`, `ok`, nil},
	{`{% if len(nil) >= 0 %}{{ "no" }}{% else %}{{ "ok" }}{% end %}`, `ok`, nil},
	{`{% if len("a", "b") >= 0 %}{{ "no" }}{% else %}{{ "ok" }}{% end %}`, `ok`, nil},
	{`{% nil = 5 %}{{ "ok" }}`, `ok`, nil},
	{`{{ f(nil) }}{{ "ok" }}`, `ok`, map[string]interface{}{"f": func(s string) string { return "no" }}},
	{`{{ f(1) }}{{ "ok" }}`, `ok`, map[string]interface{}{"f": func() string { return "no" }}},
	{`{{ f(1, 2) }}{{ "ok" }}`, `ok`, map[string]interface{}{"f": func(i int) string { return "no" }}},
	{`{{ f(2) }}{{ "ok" }}`, `ok`, map[string]interface{}{"f": func(s string) string { return "no" }}},
	{`{{ f("2") }}{{ "ok" }}`, `ok`, map[string]interface{}{"f": func(n *apd.Decimal) string { return "no" }}},
	{`{% b := map[interface{}]interface{}{} %}{% b[map[interface{}]interface{}{}] = 5 %}{{ "ok" }}`, `ok`, nil},
	{`{% b := []interface{}{} %}{% b[3] = 5 %}{{ "ok" }}`, `ok`, nil},
	{`{% a := nil() %}{{ "ok" }}`, `ok`, nil},
	{`{{ f() }}{{ "ok" }}`, "ok", scope{"f": (func() int)(nil)}},
	{`{{ s["a"] + s["b"] }}ok`, "ok", scope{"s": map[interface{}]interface{}{}}},
	{`{% s["a"] += s["b"] %}ok`, "ok", scope{"s": map[interface{}]interface{}{}}},
	{`{{ 2 / s["a"] }}ok`, "ok", scope{"s": map[interface{}]interface{}{}}},
	{`{{ 2.5 / s["a"] }}ok`, "ok", scope{"s": map[interface{}]interface{}{}}},
	{`{{ s["a"] / s["b"] }}ok`, "ok", scope{"s": map[interface{}]interface{}{}}},
	{`{% a := 3 %}{% a /= s["a"] %}ok`, "ok", scope{"s": map[interface{}]interface{}{}}},
	{`{% s["a"] /= s["b"] %}ok`, "ok", scope{"s": map[interface{}]interface{}{}}},
	{`{{ 7.0 / 0 }}ok`, "ok", nil},
	{`{{ 7 / 0.0 }}ok`, "ok", nil},
	{`{{ 7 % 0 }}ok`, "ok", nil},
	{`{{ 7.0 % 0 }}ok`, "ok", nil},
	{`{{ 7 % 0.0 }}ok`, "ok", nil},
	{`{{ -9223372036854775808 * -1 }}ok`, "ok", nil},                   // math.MinInt64 * -1
	{`{{ 9223372036854775807 + 9223372036854775807 }}ok`, "ok", nil},   // math.MaxInt64 + math.MaxInt64
	{`{{ -9223372036854775808 + -9223372036854775808 }}ok`, "ok", nil}, // math.MinInt64 + math.MinInt64
	{"{% delete(m,map[interface{}]interface{}{}) %}ok", "ok", scope{"m": map[interface{}]interface{}{}}},
	{`{% m := map[int]int{1:1, 2:4} %}{% v := m["string"] %}ok`, "ok", nil},
	{`{% m := map[int]int{1:1, 2:4} %}{% m["string"] = 5 %}ok`, "ok", nil},
	{`{% switch %}{% case true %}{% a := 5 %}{% fallthrough %}{% case false %}{{ a }}{% end %}ok`, "ok", nil},
	{"{% a := int(nil) %}ok", "ok", nil},
}

// TODO (Gianluca): error checking in renderer is no longer supported. Consider
// removing these tests or moving them to the typechecker.
func NoTestErrors(t *testing.T) {
	for _, expr := range errorTests {
		var tree, err = compiler.ParseSource([]byte(expr.src), ast.ContextHTML)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = RenderTree(b, tree, expr.globals, false)
		if err == nil {
			t.Errorf("source: %q, expecting error\n", expr.src)
			continue
		}
		if errs, ok := err.(Errors); ok {
			if len(errs) > 1 {
				t.Errorf("source: %q, unexpected %d errors, expecting 1 error\n", expr.src, len(errs))
				continue
			}
			if res := b.String(); res != expr.res {
				t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res, expr.res)
			}
		} else {
			t.Errorf("source: %q, %s\n", expr.src, err)
		}
	}
}
