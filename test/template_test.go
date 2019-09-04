// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"bytes"
	"strings"
	"testing"

	"scriggo"
	"scriggo/template"
)

var templateCases = map[string]struct {
	src  string
	out  string
	main *scriggo.MapPackage
	vars map[string]interface{}
}{
	"Empty template": {
		src: ``,
	},
	"Text only": {
		src: `Hello, world!`,
		out: `Hello, world!`,
	},

	"Template comments": {
		src: `{# this is a comment #}`,
		out: ``,
	},

	"Template comments with text": {
		src: `Text before comment{# comment #} text after comment{# another comment #}`,
		out: `Text before comment text after comment`,
	},

	"'Show' node only": {
		src: `{{ "i am a show" }}`,
		out: `i am a show`,
	},

	"Text and show": {
		src: `Hello, {{ "world" }}!!`,
		out: `Hello, world!!`,
	},

	"If statements - true": {
		src: `{% if true %}true{% else %}false{% end %}`,
		out: `true`,
	},

	"If statements - false": {
		src: `{% if !true %}true{% else %}false{% end %}`,
		out: `false`,
	},

	"Variable declarations": {
		src: `{% var a = 10 %}{% var b = 20 %}{{ a + b }}`,
		out: "30",
	},

	"For loop": {
		src: "For loop: {% for i := 0; i < 5; i++ %}{{ i }}, {% end %}",
		out: "For loop: 0, 1, 2, 3, 4, ",
	},

	"Template builtin - max": {
		src: `Maximum between 10 and -3 is {{ max(10, -3) }}`,
		out: `Maximum between 10 and -3 is 10`,
	},

	"Template builtin - sort": {
		src: `{% s := []string{"a", "c", "b"} %}{{ s }} sorted is {% sort(s) %}{{ s }}`,
		out: `a, c, b sorted is a, b, c`,
	},

	"Function call": {
		src: `{% func() { print(5) }() %}`,
		out: `5`,
	},

	"Multi rows": {
		src: `{%
	print(3) %}`,
		out: `3`,
	},

	"Multi rows 2": {
		src: `{%
	print(3)
%}`,
		out: `3`,
	},

	"Multi rows with comments": {
		src: `{%
// pre comment
/* pre comment */
	print(3)
/* post comment */
// post comment

%}`,
		out: `3`,
	},

	"Using a function declared in main": {
		src: `calling f: {{ f() }}, done!`,
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"f": func() string { return "i'm f!" },
			},
		},
		out: `calling f: i'm f!, done!`,
	},

	"Reading a variable declared in main": {
		src: `{{ mainVar }}`,
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"mainVar": (*int)(nil),
			},
		},
		out: `0`,
	},

	"Reading a variable declared in main and initialized with vars": {
		src: `{{ initMainVar }}`,
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"initMainVar": (*int)(nil),
			},
		},
		vars: map[string]interface{}{
			"initMainVar": 42,
		},
		out: `42`,
	},

	"Calling a builtin function": {
		src: `{{ lowercase("HellO ScrIgGo!") }}{% x := "A String" %}{{ lowercase(x) }}`,
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"lowercase": func(s string) string {
					return strings.ToLower(s)
				},
			},
		},
		out: `hello scriggo!a string`,
	},

	"Calling a function stored in a builtin variable": {
		src: `{{ lowercase("HellO ScrIgGo!") }}{% x := "A String" %}{{ lowercase(x) }}`,
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"lowercase": (*func(string) string)(nil),
			},
		},
		vars: map[string]interface{}{
			"lowercase": func(s string) string {
				return strings.ToLower(s)
			},
		},
		out: `hello scriggo!a string`,
	},

	"Macro definition (no arguments)": {
		src: `Macro def: {% macro M %}M's body{% end %}end.`,
		out: `Macro def: end.`,
	},

	"Macro definition (no arguments) and show-macro": {
		src: `{% macro M %}body{% end %}{% show M %}`,
		out: `body`,
	},

	"Macro definition (with arguments)": {
		src: `{% macro M(v int) %}v is {{ v }}{% end %}`,
	},

	"Macro definition (with one string argument) and show-macro": {
		src: `{% macro M(v string) %}v is {{ v }}{% end %}{% show M("msg") %}`,
		out: `v is msg`,
	},

	"Macro definition (with two string arguments) and show-macro": {
		src: `{% macro M(a, b string) %}a is {{ a }} and b is {{ b }}{% end %}{% show M("avalue", "bvalue") %}`,
		out: `a is avalue and b is bvalue`,
	},

	"Macro definition (with one int argument) and show-macro": {
		src: `{% macro M(v int) %}v is {{ v }}{% end %}{% show M(42) %}`,
		out: `v is 42`,
	},

	"Macro definition (with one []int argument) and show-macro": {
		src: `{% macro M(v []int) %}v is {{ v }}{% end %}{% show M([]int{42}) %}`,
		out: `v is 42`,
	},

	"Two macro definitions": {
		src: `{% macro M1 %}M1's body{% end %}{% macro M2(i int, s string) %}i: {{ i }}, s: {{ s }}{% end %}`,
	},

	"Two macro definitions and three show-macro": {
		src: `{% macro M1 %}M1's body{% end %}{% macro M2(i int, s string) %}i: {{ i }}, s: {{ s }}{% end %}Show macro: {% show M1 %} {% show M2(-30, "hello") %} ... {% show M1 %}`,
		out: `Show macro: M1's body i: -30, s: hello ... M1's body`,
	},

	"Macro definition and show-macro without parameters": {
		src: `{% macro M %}ok{% end %}{% show M() %}`,
		out: `ok`,
	},

	"Macro definition and show-macro without parentheses": {
		src: `{% macro M %}ok{% end %}{% show M %}`,
		out: `ok`,
	},

	"Macro definition and show-macro variadic": {
		src: `{% macro M(v ...int) %}{% for _ , i := range v %}{{ i }}{% end for %}{% end macro %}{% show M([]int{1,2,3}...) %}`,
		out: `123`,
	},

	"Template builtin - title": {
		src: `{% s := "hello, world" %}{{ s }} converted to title is {{ title(s) }}`,
		out: `hello, world converted to title is Hello, World`,
	},

	"Label for": {
		src: `{% L: for %}a{% break L %}b{% end for %}`,
		out: `a`,
	},

	"Label switch": {
		src: `{% L: switch 1 %}{% case 1 %}a{% break L %}b{% end switch %}`,
		out: `a`,
	},

	"ShowMacro of a not-defined macro with 'or ignore' option": {
		src: `Ignored macro: {% show M or ignore %} ok.`,
		out: `Ignored macro:  ok.`,
	},
}

func TestTemplate(t *testing.T) {
	for name, cas := range templateCases {
		t.Run(name, func(t *testing.T) {
			r := template.MapReader{"/main": []byte(cas.src)}
			builtins := template.Builtins()
			if cas.main != nil {
				b := scriggo.MapPackage{
					PkgName:      "main",
					Declarations: map[string]interface{}{},
				}
				for _, name := range builtins.DeclarationNames() {
					b.Declarations[name] = builtins.Lookup(name)
				}
				for k, v := range cas.main.Declarations {
					b.Declarations[k] = v
				}
				builtins = &b
			}
			templ, err := template.Load("/main", r, builtins, template.ContextText, nil)
			if err != nil {
				t.Fatalf("loading error: %s", err)
			}
			w := &bytes.Buffer{}
			err = templ.Render(w, cas.vars, &template.RenderOptions{PrintFunc: scriggo.PrintFunc(w)})
			if err != nil {
				t.Fatalf("rendering error: %s", err)
			}
			if cas.out != w.String() {
				t.Fatalf("expecting %q, got %q", cas.out, w.String())
			}
		})
	}
}

var templateMultiPageCases = map[string]struct {
	sources  map[string]string
	expected string
}{

	"Include - Only text": {
		sources: map[string]string{
			"/index.html":    `a{% include "/included.html" %}c`,
			"/included.html": `b`,
		},
		expected: "abc",
	},

	"Include - Included file uses external variable": {
		sources: map[string]string{
			"/index.html":    `{% var a = 10 %}a: {% include "/included.html" %}`,
			"/included.html": `{{ a }}`,
		},
		expected: "a: 10",
	},

	"Include - File including uses included variable": {
		sources: map[string]string{
			"/index.html":    `{% include "/included.html" %}included a: {{ a }}`,
			"/included.html": `{% var a = 20 %}`,
		},
		expected: "included a: 20",
	},

	"Include - Including a file which includes another file": {
		sources: map[string]string{
			"/index.html":              `indexstart,{% include "/dir1/included.html" %}indexend,`,
			"/dir1/included.html":      `i1start,{% include "/dir1/dir2/included.html" %}i1end,`,
			"/dir1/dir2/included.html": `i2,`,
		},
		expected: "indexstart,i1start,i2,i1end,indexend,",
	},

	"Import/Macro - Importing a macro defined in another page": {
		sources: map[string]string{
			"/index.html": `{% import "/page.html" %}{% show M %}{% show M %}`,
			"/page.html":  `{% macro M %}macro!{% end %}{% macro M2 %}macro 2!{% end %}`,
		},
		expected: "macro!macro!",
	},

	"Import/Macro - Importing a macro defined in another page, where a function calls a before-declared function": {
		sources: map[string]string{
			"/index.html": `{% import "/page.html" %}{% show M %}{% show M %}`,
			"/page.html": `
				{% macro M2 %}macro 2!{% end %}
				{% macro M %}{% show M2 %}{% end %}
			`,
		},
		expected: "macro 2!macro 2!",
	},

	"Import/Macro - Importing a macro defined in another page, where a function calls an after-declared function": {
		sources: map[string]string{
			"/index.html": `{% import "/page.html" %}{% show M %}{% show M %}`,
			"/page.html": `
				{% macro M %}{% show M2 %}{% end %}
				{% macro M2 %}macro 2!{% end %}
			`,
		},
		expected: "macro 2!macro 2!",
	},

	"Import/Macro - Importing a macro defined in another page, which imports a third page": {
		sources: map[string]string{
			"/index.html": `{% import "/page1.html" %}index-start,{% show M1 %}index-end`,
			"/page1.html": `{% import "/page2.html" %}{% macro M1 %}M1-start,{% show M2 %}M1-end,{% end %}`,
			"/page2.html": `{% macro M2 %}M2,{% end %}`,
		},
		expected: "index-start,M1-start,M2,M1-end,index-end",
	},

	"Import/Macro - Importing a macro using an import statement with identifier": {
		sources: map[string]string{
			"/index.html": `{% import pg "/page.html" %}{% show pg.M %}{% show pg.M %}`,
			"/page.html":  `{% macro M %}macro!{% end %}`,
		},
		expected: "macro!macro!",
	},

	"Import/Macro - Importing a macro using an import statement with identifier (with comments)": {
		sources: map[string]string{
			"/index.html": `{# a comment #}{% import pg "/page.html" %}{# a comment #}{% show pg.M %}{# a comment #}{% show pg.M %}{# a comment #}`,
			"/page.html":  `{# a comment #}{% macro M %}{# a comment #}macro!{# a comment #}{% end %}{# a comment #}`,
		},
		expected: "macro!macro!",
	},

	"Extends - Empty page extends a page containing only text": {
		sources: map[string]string{
			"/index.html": `{% extends "/page.html" %}`,
			"/page.html":  `I'm page!`,
		},
		expected: "I'm page!",
	},

	"Extends - Extending a page that calls a macro defined on current page": {
		sources: map[string]string{
			"/index.html": `{% extends "/page.html" %}{% macro E %}E's body{% end %}`,
			"/page.html":  `{% show E %}`,
		},
		expected: "E's body",
	},

	"Extending an empty page": {
		sources: map[string]string{
			"/index.html":    `{% extends "extended.html" %}`,
			"/extended.html": ``,
		},
	},

	"Extending a page that imports another file": {
		sources: map[string]string{
			"/index.html":    `{% extends "/extended.html" %}`,
			"/extended.html": `{% import "/imported.html" %}`,
			"/imported.html": `{% macro Imported %}Imported macro{% end macro %}`,
		},
	},

	"Extending a page (that imports another file) while declaring a macro": {
		sources: map[string]string{
			"/index.html":    `{% extends "/extended.html" %}{% macro Index %}{% end macro %}`,
			"/extended.html": `{% import "/imported.html" %}`,
			"/imported.html": `{% macro Imported %}Imported macro{% end macro %}`,
		},
	},

	"Extends - Extending a page that calls two macros defined on current page": {
		sources: map[string]string{
			"/index.html": `{% extends "/page.html" %}{% macro E1 %}E1's body{% end %}{% macro E2 %}E2's body{% end %}`,
			"/page.html":  `{% show E1 %}{% show E2 %}`,
		},
		expected: "E1's bodyE2's body",
	},

	"Extends - Define a variable (with zero value) used in macro definition": {
		sources: map[string]string{
			"/index.html": `{% extends "/page.html" %}{% var Local int %}{% macro E1 %}Local has value {{ Local }}{% end %}`,
			"/page.html":  `{% show E1 %}`,
		},
		expected: "Local has value 0",
	},

	"Extends - Define a variable (with non-zero value) used in macro definition": {
		sources: map[string]string{
			"/index.html": `{% extends "/page.html" %}{% var Local = 50 %}{% macro E1 %}Local has value {{ Local }}{% end %}`,
			"/page.html":  `{% show E1 %}`,
		},
		expected: "Local has value 50",
	},

	"Extends - Extending a file which contains text and shows": {
		sources: map[string]string{
			"/index.html": `{% extends "/page.html" %}`,
			"/page.html":  `I am an {{ "extended" }} file.`,
		},
		expected: "I am an extended file.",
	},

	"File imported twice": {
		sources: map[string]string{
			"/index.html": `{% import "/a.html" %}{% import "/b.html" %}`,
			"/a.html":     `{% import "/b.html" %}`,
			"/b.html":     `{% macro M %}I'm b{% end %}`,
		},
	},
}

func TestMultiPageTemplate(t *testing.T) {
	for name, cas := range templateMultiPageCases {
		t.Run(name, func(t *testing.T) {
			r := template.MapReader{}
			for p, src := range cas.sources {
				r[p] = []byte(src)
			}
			templ, err := template.Load("/index.html", r, template.Builtins(), template.ContextText, nil)
			if err != nil {
				t.Fatalf("loading error: %s", err)
			}
			w := &bytes.Buffer{}
			err = templ.Render(w, nil, nil)
			if err != nil {
				t.Fatalf("rendering error: %s", err)
			}
			if cas.expected != w.String() {
				t.Fatalf("expecting %q, got %q", cas.expected, w.String())
			}
		})
	}
}
