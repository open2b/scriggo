package compiler_test

import (
	"strings"
	"testing"

	"scriggo/ast"
	"scriggo/internal/compiler"
	"scriggo/template"
)

var templateCases = map[string]string{

	// Misc.
	`Just test`: ok,
	`{{ a }}`:   "undefined: a",

	// Macro definitions.
	`{% macro M %}{% end %}`:                   ok,
	`{% macro M() %}{% end %}`:                 ok,
	`{% macro M(a, b int) %}{% end %}`:         ok,
	`{% macro M(int, int, string) %}{% end %}`: ok,
	`{% macro M(a) %}{% end %}`:                "undefined: a",

	// Show macro.
	`{% macro M %}{% end %}         {% show M %}`:      ok,
	`{% macro M %}{% end %}         {% show M(1) %}`:   "too many arguments in call to M\n\thave (number)\n\twant ()",
	`{% macro M(int) %}{% end %}    {% show M("s") %}`: "cannot use \"s\" (type string) as type int in argument to M",

	// "{% macro M %}{% end %}    {% show M() %}":  ok, // TODO(Gianluca): See issue #136.

	`{% show M %}`:           `undefined: M`,
	`{% show M or error %}`:  `undefined: M`,
	`{% show M or ignore %}`: ok,

	`{% a := 10 %}{% a %}`: `a evaluated but not used`,

	// TODO(Gianluca): result of this test depends on the type checking options.
	// `{% show M or todo %}`: ok,
}

const ok = ""

func TestTemplate(t *testing.T) {
	for src, expected := range templateCases {
		t.Run(src, func(t *testing.T) {
			r := template.MapReader{"/index.html": []byte(src)}
			tree, err := compiler.ParseTemplate("/index.html", r, ast.ContextText)
			if err != nil {
				t.Fatalf("parsing error: %s", err)
			}
			_, err = compiler.Typecheck(tree, nil, compiler.CheckerOptions{SyntaxType: compiler.TemplateSyntax})
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
