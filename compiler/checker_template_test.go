// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"strings"
	"testing"

	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/internal/mapfs"
)

var templateCases = []struct {
	src      string
	expected string
}{

	// Misc.
	{
		src:      `Just test`,
		expected: ok,
	},
	{
		src:      `{{ a }}`,
		expected: "undefined: a",
	},
	// Macro definitions.
	{
		src:      `{% macro M %}{% end %}`,
		expected: ok,
	},
	{
		src:      `{% macro M() %}{% end %}`,
		expected: ok,
	},
	{
		src:      `{% macro M(a, b int) %}{% end %}`,
		expected: ok,
	},
	{
		src:      `{% macro M(int, int, string) %}{% end %}`,
		expected: ok,
	},
	{
		src:      `{% macro M(a) %}{% end %}`,
		expected: "undefined: a",
	},
	{
		src:      `{% macro M %}{% end %}{% macro M %}{% end %}`,
		expected: "M redeclared in this block\n\tprevious declaration at 1:10",
	},

	// Show macro.
	{
		src:      `{% macro M %}{% end %}         {% show M %}`,
		expected: ok,
	},
	{
		src:      `{% macro M %}{% end %}         {% show M(1) %}`,
		expected: "too many arguments in call to M\n\thave (number)\n\twant ()",
	},
	{
		src:      `{% macro M(int) %}{% end %}    {% show M("s") %}`,
		expected: "cannot use \"s\" (type string) as type int in argument to M",
	},

	{
		src:      `{% macro M %}{% end %}    {% show M() %}`,
		expected: ok,
	},

	{
		src:      `{% show M %}`,
		expected: `undefined: M`,
	},

	{
		src:      `{% a := 10 %}{% a %}`,
		expected: `a evaluated but not used`,
	},

	{
		src:      `{% a := 20 %}{{ a and a }}`,
		expected: ok,
	},

	{
		src:      `{% a := 20 %}{{ a or a }}`,
		expected: ok,
	},

	{
		src:      `{% a := 20 %}{% b := "" %}{{ a or b and (not b) }}`,
		expected: ok,
	},

	{
		src:      `{% a := 20 %}{{ 3 and a }}`,
		expected: ok,
	},

	{
		src:      `{% a := 20 %}{{ 3 or a }}`,
		expected: ok,
	},

	{
		src:      `{% const a = 20 %}{{ not a }}`,
		expected: ok,
	},

	{
		src:      `{% a := true %}{% b := true %}{{ a and b or b and b }}`,
		expected: ok,
	},

	{
		src:      `{% n := 10 %}{% var a bool = not n %}`,
		expected: ok,
	},

	{
		src:      `{% a := []int(nil) %}{% if a %}{% end %}`,
		expected: ``,
	},

	{
		src:      `{% if 20 %}{% end %}`,
		expected: ``,
	},

	{
		src:      `{{ true and nil }}`,
		expected: `invalid operation: true and nil (operator 'and' not defined on nil)`,
	},

	{
		src:      `{{ nil and false }}`,
		expected: `invalid operation: nil and false (operator 'and' not defined on nil)`,
	},

	{
		src:      `{{ true or nil }}`,
		expected: `invalid operation: true or nil (operator 'or' not defined on nil)`,
	},

	{
		src:      `{{ not nil }}`,
		expected: `invalid operation: not nil (operator 'not' not defined on nil)`,
	},

	{
		src:      `{% v := 10 %}{{ v and nil }}`,
		expected: `invalid operation: v and nil (operator 'and' not defined on nil)`,
	},

	{
		// Check that the 'and' operator returns an untyped bool even if its two
		// operands are both typed booleans. The same applies to the 'or' and
		// 'not' operators.
		src: `
			{% type Bool bool %}
			{% var _ bool = Bool(true) and Bool(false) %}
		`,
		expected: ok,
	},
}

func TestTemplate(t *testing.T) {
	type HTML string
	options := Options{
		FormatTypes: map[ast.Format]reflect.Type{
			ast.FormatHTML: reflect.TypeOf((*HTML)(nil)).Elem(),
		},
	}
	for _, cas := range templateCases {
		src := cas.src
		expected := cas.expected
		t.Run(src, func(t *testing.T) {
			fsys := mapfs.MapFS{"index.html": src}
			_, err := BuildTemplate(fsys, "index.html", options)
			switch {
			case expected == "" && err != nil:
				t.Fatalf("unexpected error: %q", err)
			case expected != "" && err == nil:
				t.Fatalf("expecting error %q, got nothing", expected)
			case expected != "" && err != nil && !strings.Contains(err.Error(), expected):
				t.Fatalf("expecting error %q, got %q", expected, err.Error())
			}
		})
	}
}
