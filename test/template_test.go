package test

import (
	"bytes"
	"scrigo/template"
	"scrigo/template/builtins"
	"testing"
)

var templateCases = map[string]struct {
	src string
	out string
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

	// TODO(Gianluca): out of memory.
	// "Template builtin - title": {
	// 	src: `{% s := "hello, world" %}{{ s }} converted to title is {{ title(s) }}`,
	// 	out: ``,
	// },
}

func TestTemplate(t *testing.T) {
	for name, cas := range templateCases {
		t.Run(name, func(t *testing.T) {
			r := template.MapReader{"/main": []byte(cas.src)}
			templ, err := template.Load("/main", r, builtins.Main(), template.ContextText, template.LoadOption(0))
			if err != nil {
				t.Fatalf("loading error: %s", err)
			}
			w := &bytes.Buffer{}
			templ.SetRenderFunc(template.DefaultRender)
			err = templ.Render(w, nil, template.RenderOptions{})
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
			templ.SetRenderFunc(template.DefaultRender)
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
