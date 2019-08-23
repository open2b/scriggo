package compiler_test

import (
	"strings"
	"testing"

	"scriggo/ast"
	"scriggo/internal/compiler"
	"scriggo/template"
)

var templateCases = []struct {
	src      string
	expected string
	opts     *compiler.CheckerOptions
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
		src:      `{% show M or error %}`,
		expected: `undefined: M`,
	},
	{
		src:      `{% show M or ignore %}`,
		expected: ok,
	},

	{
		src:      `{% a := 10 %}{% a %}`,
		expected: `a evaluated but not used`,
	},

	{
		src:      `{% show M or todo %}`,
		expected: ok,
		opts: &compiler.CheckerOptions{
			SyntaxType: compiler.TemplateSyntax,
			FailOnTODO: false,
		},
	},

	{
		src:      `{% show M or todo %}`,
		expected: `macro M is not defined: must be implemented`,
		opts: &compiler.CheckerOptions{
			SyntaxType: compiler.TemplateSyntax,
			FailOnTODO: true,
		},
	},
}

const ok = ""

func TestTemplate(t *testing.T) {
	for _, cas := range templateCases {
		src := cas.src
		expected := cas.expected
		t.Run(src, func(t *testing.T) {
			r := template.MapReader{"/index.html": []byte(src)}
			tree, err := compiler.ParseTemplate("/index.html", r, ast.ContextText)
			if err != nil {
				t.Fatalf("parsing error: %s", err)
			}
			opts := compiler.CheckerOptions{SyntaxType: compiler.TemplateSyntax}
			if cas.opts != nil {
				opts = *cas.opts
			}
			_, err = compiler.Typecheck(tree, nil, opts)
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
