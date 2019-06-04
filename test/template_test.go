package test

import (
	"bytes"
	"scrigo/template"
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
}

func TestTemplate(t *testing.T) {
	for name, cas := range templateCases {
		t.Run(name, func(t *testing.T) {
			r := template.MapReader{"/main": []byte(cas.src)}
			templ, err := template.Load("/main", r, nil, template.ContextText, template.LoadOption(0))
			if err != nil {
				t.Fatalf("loading error: %s", err)
			}
			w := &bytes.Buffer{}
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

// TODO(Gianluca): should we allow empty template pages?
// "Empty": {
// 	src: ``,
// 	out: ``,
// },
