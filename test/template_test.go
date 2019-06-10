package test

import (
	"bytes"
	"testing"

	"scriggo"
	"scriggo/builtins"
	"scriggo/template"
)

var templateCases = map[string]struct {
	src  string
	out  string
	main *scriggo.Package
	vars map[string]interface{}
}{
	"Text only": {
		src: `Hello, world!`,
		out: `Hello, world!`,
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
		out: `[a c b] sorted is [a b c]`,
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
		main: &scriggo.Package{
			Name: "main",
			Declarations: map[string]interface{}{
				"f": func() string { return "i'm f!" },
			},
		},
		out: `calling f: i'm f!, done!`,
	},

	"Reading a variable declared in main": {
		src: `{{ mainVar }}`,
		main: &scriggo.Package{
			Name: "main",
			Declarations: map[string]interface{}{
				"mainVar": (*int)(nil),
			},
		},
		out: `0`,
	},

	"Reading a variable declared in main and initialized with vars": {
		src: `{{ initMainVar }}`,
		main: &scriggo.Package{
			Name: "main",
			Declarations: map[string]interface{}{
				"initMainVar": (*int)(nil),
			},
		},
		vars: map[string]interface{}{
			"initMainVar": 42,
		},
		out: `42`,
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
		out: `v is [42]`,
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

	// TODO(Gianluca): reflect: call of reflect.Value.Len on struct Value
	//"Macro definition and show-macro variadic": {
	//	src: `{% macro M(v ...int) %}{% for _ , i := range v %}{{ i }}{% end for %}{% end macro %}{% show M([]int{1,2,3}...) %}`,
	//	out: `123`,
	//},

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
}

func TestTemplate(t *testing.T) {
	for name, cas := range templateCases {
		t.Run(name, func(t *testing.T) {
			r := template.MapReader{"/main": []byte(cas.src)}
			main := builtins.Main()
			if cas.main != nil {
				for k, v := range cas.main.Declarations {
					main.Declarations[k] = v
				}
			}
			templ, err := template.Load("/main", r, main, template.ContextText, template.LoadOption(0))
			if err != nil {
				t.Fatalf("loading error: %s", err)
			}
			w := &bytes.Buffer{}
			err = templ.Render(w, cas.vars, template.RenderOptions{PrintFunc: scriggo.PrintFunc(w)})
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

	"Extends - Empty page extends a page containing only text": {
		sources: map[string]string{
			"/index.html": `{% extends "/page.html" %}`,
			"/page.html":  `I'm page!`,
		},
		expected: "I'm page!",
	},

	"Extends - Extending a page that calls a macro defined in current page": {
		sources: map[string]string{
			"/index.html": `{% extends "/page.html" %}{% macro E %}E's body{% end %}`,
			"/page.html":  `{% show E %}`,
		},
		expected: "E's body",
	},
}

func TestMultiPageTemplate(t *testing.T) {
	for name, cas := range templateMultiPageCases {
		t.Run(name, func(t *testing.T) {
			r := template.MapReader{}
			for p, src := range cas.sources {
				r[p] = []byte(src)
			}
			templ, err := template.Load("/index.html", r, builtins.Main(), template.ContextText, template.LoadOption(0))
			if err != nil {
				t.Fatalf("loading error: %s", err)
			}
			w := &bytes.Buffer{}
			err = templ.Render(w, nil, template.RenderOptions{})
			if err != nil {
				t.Fatalf("rendering error: %s", err)
			}
			if cas.expected != w.String() {
				t.Fatalf("expecting %q, got %q", cas.expected, w.String())
			}
		})
	}
}
