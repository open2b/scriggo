package compiler_test

import (
	"strings"
	"testing"

	"github.com/open2b/scriggo/compiler"
	"github.com/open2b/scriggo/compiler/ast"
)

var templateCases = []struct {
	src      string
	expected string
	opts     *compiler.Options
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

	// https://github.com/open2b/scriggo/issues/560

	// {
	// 	src:      `{% show M or error %}`,
	// 	expected: `undefined: M`,
	// },
	// {
	// 	src:      `{% show M or ignore %}`,
	// 	expected: ok,
	// },

	{
		src:      `{% a := 10 %}{% a %}`,
		expected: `a evaluated but not used`,
	},

	// https://github.com/open2b/scriggo/issues/560

	// {
	// 	src:      `{% show M or todo %}`,
	// 	expected: ok,
	// 	opts: &compiler.Options{
	// 		TemplateFailOnTODO: false,
	// 	},
	// },

	// {
	// 	src:      `{% show M or todo %}`,
	// 	expected: `macro M is not defined: must be implemented`,
	// 	opts: &compiler.Options{
	// 		TemplateFailOnTODO: true,
	// 	},
	// },

	{
		src:      `{% a := 20 %}{{ a and a }}`,
		expected: ok,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{% a := 20 %}{{ a or a }}`,
		expected: ok,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{% a := 20 %}{% b := "" %}{{ a or b and (not b) }}`,
		expected: ok,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{% a := 20 %}{{ 3 and a }}`,
		expected: ok,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{% a := 20 %}{{ 3 or a }}`,
		expected: ok,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{% const a = 20 %}{{ not a }}`,
		expected: ok,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{% a := true %}{% b := true %}{{ a and b or b and b }}`,
		expected: ok,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{% n := 10 %}{% var a bool = not n %}`,
		expected: ok,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{% a := []int(nil) %}{% if a %}{% end %}`,
		expected: ``,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{% a := []int(nil) %}{% if a %}{% end %}`,
		expected: `non-bool a (type []int) used as if condition`,
	},

	{
		src:      `{% if 20 %}{% end %}`,
		expected: ``,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{{ true and nil }}`,
		expected: `invalid operation: true and nil (operator 'and' not defined on nil)`,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{{ nil and false }}`,
		expected: `invalid operation: nil and false (operator 'and' not defined on nil)`,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{{ true or nil }}`,
		expected: `invalid operation: true or nil (operator 'or' not defined on nil)`,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{{ not nil }}`,
		expected: `invalid operation: not nil (operator 'not' not defined on nil)`,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{% v := 10 %}{{ v and nil }}`,
		expected: `invalid operation: v and nil (operator 'and' not defined on nil)`,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
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
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},
}

const ok = ""

func TestTemplate(t *testing.T) {
	for _, cas := range templateCases {
		src := cas.src
		expected := cas.expected
		t.Run(src, func(t *testing.T) {
			r := mapReader{"/index.html": []byte(src)}
			var compileOpts compiler.Options
			if cas.opts != nil {
				compileOpts = *cas.opts
			}
			_, err := compiler.CompileTemplate("/index.html", r, ast.LanguageHTML, compileOpts)
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

type mapReader map[string][]byte

func (r mapReader) ReadFile(name string) ([]byte, error) {
	src, ok := r[name]
	if !ok {
		panic("not existing")
	}
	return src, nil
}
