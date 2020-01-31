package compiler_test

import (
	"strings"
	"testing"

	"scriggo/compiler"
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
		expected: `non-bool constant 3 not allowed with operator and`,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{% a := 20 %}{{ 3 or a }}`,
		expected: `non-bool constant 3 not allowed with operator or`,
		opts: &compiler.Options{
			RelaxedBoolean: true,
		},
	},

	{
		src:      `{% const a = 20 %}{{ not a }}`,
		expected: `non-bool constant a not allowed with operator not`,
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
		src:      `{% if 20 %}{% end %}`,
		expected: `non-bool constant 20 cannot be used as if condition`,
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
			_, err := compiler.CompileTemplate("/index.html", r, nil, 0, compileOpts)
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

func (r mapReader) Read(path string) ([]byte, error) {
	src, ok := r[path]
	if !ok {
		panic("not existing")
	}
	return src, nil
}
