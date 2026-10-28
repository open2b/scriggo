// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package misc

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/internal/fstest"
	"github.com/open2b/scriggo/native"
)

type Vars map[string]any

var htmlContextTests = []struct {
	src  string
	res  string
	vars Vars
}{
	{`""`, "", nil},
	{`"a"`, "a", nil},
	{`"<a>"`, "&lt;a&gt;", nil},
	{`"<div></div>"`, "&lt;div&gt;&lt;/div&gt;", nil},
	{`a`, "&lt;a&gt;", Vars{"a": "<a>"}},
	{`a`, "&#34;ab&#39;cd&#34;", Vars{"a": "\"ab'cd\""}},
	{`d`, "&lt;div&gt;&lt;/div&gt;", Vars{"d": "<div></div>"}},
	{`a`, "<a>", Vars{"a": native.HTML("<a>")}},
	{`d`, "<div></div>", Vars{"d": native.HTML("<div></div>")}},
	{`0`, "0", nil},
	{`25`, "25", nil},
	{`-25`, "-25", nil},
	{`0.0`, "0", nil},
	{`0.1`, "0.1", nil},
	{`0.01`, "0.01", nil},
	{`0.1111111`, "0.1111111", nil},
	{`0.1000000`, "0.1", nil},
	{`-0.1`, "-0.1", nil},
	{`-0.1111111`, "-0.1111111", nil},
	{`5479159814589435423678645745.523785742365476`, "5479159814589435000000000000", nil},
	{`0.0000000000000000000000000000000000000000000000000000000001`, "0.0000000000000000000000000000000000000000000000000000000001", nil},
	{`-0.1000000`, "-0.1", nil},
	{`true`, "true", nil},
	{`false`, "false", nil},
	{`s["a"]`, "", Vars{"s": map[string]string{}}},
	{`a`, "--- start Markdown ---\na--- end Markdown ---\n", Vars{"a": native.Markdown("a")}},
}

func TestHTMLContext(t *testing.T) {
	for _, expr := range htmlContextTests {
		fsys := fstest.Files{"index.html": "{{" + expr.src + "}}"}
		opts := &scriggo.BuildOptions{
			Globals:           asDeclarations(expr.vars),
			MarkdownConverter: markdownConverter,
		}
		template, err := scriggo.BuildTemplate(fsys, "index.html", opts)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = template.Run(b, expr.vars, nil)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var res = b.String()
		if res != expr.res {
			t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res, expr.res)
		}
	}
}

var quotedAttrContextTests = []struct {
	src  string
	res  string
	vars Vars
}{
	{`""`, "", nil},
	{`"a"`, "a", nil},
	{`"<a>"`, "&lt;a&gt;", nil},
	{`"<div></div>"`, "&lt;div&gt;&lt;/div&gt;", nil},
	{`a`, "&lt;a&gt;", Vars{"a": "<a>"}},
	{`d`, "&lt;div&gt;&lt;/div&gt;", Vars{"d": "<div></div>"}},
	{`a`, "&lt;a&gt;", Vars{"a": native.HTML("<a>")}},
	{`d`, "&lt;div&gt;&lt;/div&gt;", Vars{"d": native.HTML("<div></div>")}},
	{`a`, "&lt;a&gt;&#33;", Vars{"a": native.HTML("<a>&#33;")}},
	{`d`, "&lt;div&gt;&#33;&lt;/div&gt;", Vars{"d": native.HTML("<div>&#33;</div>")}},
	{`0`, "0", nil},
	{`25`, "25", nil},
	{`-25`, "-25", nil},
	{`0.0`, "0", nil},
	{`0.1`, "0.1", nil},
	{`0.01`, "0.01", nil},
	{`0.1111111`, "0.1111111", nil},
	{`0.1000000`, "0.1", nil},
	{`-0.1`, "-0.1", nil},
	{`-0.1111111`, "-0.1111111", nil},
	{`-0.1000000`, "-0.1", nil},
	{`true`, "true", nil},
	{`false`, "false", nil},
	{`s["a"]`, "", Vars{"s": map[any]any{}}},
}

func TestQuotedAttrContext(t *testing.T) {
	for _, expr := range quotedAttrContextTests {
		fsys := fstest.Files{"index.html": `<z x="{{` + expr.src + `}}">`}
		opts := &scriggo.BuildOptions{
			Globals: asDeclarations(expr.vars),
		}
		template, err := scriggo.BuildTemplate(fsys, "index.html", opts)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = template.Run(b, expr.vars, nil)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var res = b.String()
		if res[6:len(res)-2] != expr.res {
			t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res[6:len(res)-2], expr.res)
		}
	}
}

var unquotedAttrContextTests = []struct {
	src  string
	res  string
	vars Vars
}{
	{`a`, "&#32;a&#32;", Vars{"a": " a "}},
	{`a`, "&#09;&#10;&#13;&#12;&#32;a&#61;&#96;", Vars{"a": "\t\n\r\x0C a=`"}},
	{`s["a"]`, "", Vars{"s": map[any]any{}}},
	{`a`, "&lt;a&gt;", Vars{"a": "<a>"}},
	{`a`, "&lt;a&gt;", Vars{"a": native.HTML("<a>")}},
	{`a`, "&lt;a&gt;&#33;", Vars{"a": native.HTML("<a>&#33;")}},
}

func TestUnquotedAttrContext(t *testing.T) {
	for _, expr := range unquotedAttrContextTests {
		fsys := fstest.Files{"index.html": `<z x={{` + expr.src + `}}>`}
		opts := &scriggo.BuildOptions{
			Globals: asDeclarations(expr.vars),
		}
		template, err := scriggo.BuildTemplate(fsys, "index.html", opts)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = template.Run(b, expr.vars, nil)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var res = b.String()
		if res[5:len(res)-1] != expr.res {
			t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res[5:len(res)-1], expr.res)
		}
	}
}

var scriptContextTests = []struct {
	src  string
	res  string
	vars Vars
}{
	{`""`, `""`, nil},
	{`"a"`, `"a"`, nil},
	{`"<a>"`, `"\u003ca\u003e"`, nil},
	{`"<div></div>"`, `"\u003cdiv\u003e\u003c/div\u003e"`, nil},
	{`"\\"`, `"\\"`, nil},
	{`"\""`, `"\""`, nil},
	{`"\n"`, `"\n"`, nil},
	{`"\r"`, `"\r"`, nil},
	{`"\t"`, `"\t"`, nil},
	{`"\\\"\n\r\t\u2028\u2029\u0000\u0010"`, `"\\\"\n\r\t\u2028\u2029\u0000\u0010"`, nil},
	{`0`, "0", nil},
	{`25`, "25", nil},
	{`-25`, "-25", nil},
	{`0.0`, "0", nil},
	{`0.1`, "0.1", nil},
	{`0.1111111`, "0.1111111", nil},
	{`0.1000000`, "0.1", nil},
	{`-0.1`, "-0.1", nil},
	{`-0.1111111`, "-0.1111111", nil},
	{`-0.1000000`, "-0.1", nil},
	{`true`, "true", nil},
	{`false`, "false", nil},
	{`a`, `"a"`, Vars{"a": "a"}},
	{`a`, `"\u003c\u003e\""`, Vars{"a": "<>\""}},
	{`a`, "null", Vars{"a": []int(nil)}},
	{`a`, "[0,1,2,3,4,5]", Vars{"a": []int{0, 1, 2, 3, 4, 5}}},
	{`a`, "[-2,-1,0,1,2]", Vars{"a": []int{-2, -1, 0, 1, 2}}},
	{`a`, `"AAECAwQF"`, Vars{"a": []byte{0, 1, 2, 3, 4, 5}}},
	{`a`, "null", Vars{"a": []bool(nil)}},
	{`a`, "[true,false,true]", Vars{"a": []bool{true, false, true}}},
	{`a`, "null", Vars{"a": (*struct{})(nil)}},
	{`a`, "{}", Vars{"a": &struct{}{}}},
	{`a`, `{}`, Vars{"a": &struct{ a int }{a: 5}}},
	{`a`, `{"A":5}`, Vars{"a": &struct{ A int }{A: 5}}},
	{`a`, `{"A":5,"B":null}`, Vars{"a": &struct {
		A int
		B *struct{}
	}{A: 5, B: nil}}},
	{`a`, `{"A":5,"B":{}}`, Vars{"a": &struct {
		A int
		B *struct{}
	}{A: 5, B: &struct{}{}}}},
	{`a`, `{"A":5,"B":{"C":"C"}}`, Vars{"a": &struct {
		A int
		B *struct{ C string }
	}{A: 5, B: &struct{ C string }{C: "C"}}}},
	{`s["a"]`, "null", Vars{"s": map[any]any{}}},
	{`a`, `{"A":5,"B":2,"C":7,"D":3}`, Vars{"a": map[string]any{"A": 5, "B": 2, "C": 7, "D": 3}}},
	{`a`, `{"":"c","\"\u0027":5}`, Vars{"a": map[string]any{"\"'": 5, "": "c"}}},
	{`a`, `{"a\u0027b":3}`, Vars{"a": map[string]any{"a'b": 3}}},
	{`"$"`, `"\u0024"`, nil},
	{`"{}"`, `"\u007b}"`, nil},
	{`a`, `"\u0060"`, Vars{"a": "`"}},
	{`a`, `"\u0060\u0024\u007ba}\u0060"`, Vars{"a": "`${a}`"}},
	{`a`, `{"\u0024\u007ba}":1}`, Vars{"a": map[string]any{"${a}": 1}}},
}

func TestScriptContext(t *testing.T) {
	for _, typ := range []string{"text/javascript", "application/ld+json"} {
		for _, expr := range scriptContextTests {
			fsys := fstest.Files{"index.html": `<script type="` + typ + `">{{` + expr.src + `}}</script>`}
			opts := &scriggo.BuildOptions{
				Globals: asDeclarations(expr.vars),
			}
			template, err := scriggo.BuildTemplate(fsys, "index.html", opts)
			if err != nil {
				t.Errorf("type: %s, source: %q, %s\n", typ, expr.src, err)
				continue
			}
			var b = &bytes.Buffer{}
			err = template.Run(b, expr.vars, nil)
			if err != nil {
				t.Errorf("type: %s, source: %q, %s\n", typ, expr.src, err)
				continue
			}
			var res = b.String()
			if len(res) < 25+len(typ) || res[16+len(typ):len(res)-9] != expr.res {
				t.Errorf("type: %s, source: %q, unexpected %q, expecting %q\n", typ, expr.src, res[16+len(typ):len(res)-9], expr.res)
			}
		}
	}
}

var jsContextTests = []struct {
	src  string
	res  string
	vars Vars
}{
	{"t", `new Date("2016-01-02T15:04:05.000Z")`, Vars{"t": time.Date(2016, 1, 2, 15, 04, 05, 0, time.UTC)}},
}

func TestJSContext(t *testing.T) {
	for _, expr := range jsContextTests {
		fsys := fstest.Files{"index.html": "<script>{{" + expr.src + "}}</script>"}
		opts := &scriggo.BuildOptions{
			Globals: asDeclarations(expr.vars),
		}
		template, err := scriggo.BuildTemplate(fsys, "index.html", opts)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = template.Run(b, expr.vars, nil)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var res = b.String()
		if len(res) < 17 || res[8:len(res)-9] != expr.res {
			t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res[8:len(res)-9], expr.res)
		}
	}
}

func TestJSContextWithComments(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "single line comment",
			src: `<script>
// it's bad
{% x := "hello hey" %}
if ({{ x }}) { }   // AAA
if ('{{ x }}') { } // CCC
if ("{{ x }}") { } // DDD
</script>`,
		},
		{
			name: "single line comment with double quote",
			src: `<script>
// a double quote "
{% x := "hello hey" %}
if ({{ x }}) { }   // AAA
if ('{{ x }}') { } // CCC
if ("{{ x }}") { } // DDD
</script>`,
		},
		{
			name: "block comment",
			src: `<script>
/* it's bad */
{% x := "hello hey" %}
if ({{ x }}) { }   // AAA
if ('{{ x }}') { } // CCC
if ("{{ x }}") { } // DDD
</script>`,
		},
		{
			name: "block comment with double quote",
			src: `<script>
/* a double quote " */
{% x := "hello hey" %}
if ({{ x }}) { }   // AAA
if ('{{ x }}') { } // CCC
if ("{{ x }}") { } // DDD
</script>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.Files{"index.html": tt.src}
			template, err := scriggo.BuildTemplate(fsys, "index.html", nil)
			if err != nil {
				t.Fatalf("unexpected build error: %s", err)
			}

			var b bytes.Buffer
			err = template.Run(&b, nil, nil)
			if err != nil {
				t.Fatalf("unexpected run error: %s", err)
			}

			got := b.String()
			if !strings.Contains(got, `if ("hello hey") { }   // AAA`) {
				t.Fatalf("missing JS context rendering, output: %q", got)
			}
			if !strings.Contains(got, `if ('hello hey') { } // CCC`) {
				t.Fatalf("missing JS string context rendering, output: %q", got)
			}
			if !strings.Contains(got, `if ("hello hey") { } // DDD`) {
				t.Fatalf("missing quoted JS string context rendering, output: %q", got)
			}
			if strings.Contains(got, `if (hello hey) { }   // AAA`) {
				t.Fatalf("unexpected unquoted JS output: %q", got)
			}
			if strings.Contains(got, `if ('"hello hey"') { } // CCC`) {
				t.Fatalf("unexpected double-quoted JS string output: %q", got)
			}
		})
	}
}

// TestJSTemplateLiteralEscapes tests that values shown in a JavaScript
// template literal are escaped so that they cannot close it or start a
// substitution, also when they are shown through a rendered file or a macro.
func TestJSTemplateLiteralEscapes(t *testing.T) {
	const v = "`${a}`"
	const e = `\u0060\u0024\u007ba}\u0060` // escaped v
	// er is the escaped form of the JavaScript string literal with value v.
	er := `\"` + strings.ReplaceAll(e, `\`, `\\`) + `\"`
	tests := []struct {
		name     string
		files    fstest.Files
		expected string
	}{
		{
			name:     "script",
			files:    fstest.Files{"index.html": "<script>const s = `{{ v }}`;</script>"},
			expected: "<script>const s = `" + e + "`;</script>",
		},
		{
			name:     "JS file",
			files:    fstest.Files{"index.js": "const s = `{{ v }}`;"},
			expected: "const s = `" + e + "`;",
		},
		{
			name:     "dollar in template text",
			files:    fstest.Files{"index.html": "<script>const s = `It's ${{ v }}`;</script>"},
			expected: "<script>const s = `It's $" + e + "`;</script>",
		},
		{
			name:     "rendered JS file",
			files:    fstest.Files{"index.html": "<script>const s = `{{ render \"p.js\" }}`;</script>", "p.js": "{{ v }}"},
			expected: "<script>const s = `" + er + "`;</script>",
		},
		{
			name:     "rendered JSON file",
			files:    fstest.Files{"index.html": "<script>const s = `{{ render \"p.json\" }}`;</script>", "p.json": "{{ v }}"},
			expected: "<script>const s = `" + er + "`;</script>",
		},
		{
			name:     "macro",
			files:    fstest.Files{"index.js": "{% macro M(s string) %}{{ s }}{% end %}const s = `{{ M(v) }}`;"},
			expected: "const s = `" + er + "`;",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := "index.html"
			if _, ok := tt.files[name]; !ok {
				name = "index.js"
			}
			vars := Vars{"v": v}
			opts := &scriggo.BuildOptions{Globals: asDeclarations(vars)}
			template, err := scriggo.BuildTemplate(tt.files, name, opts)
			if err != nil {
				t.Fatalf("unexpected build error: %s", err)
			}
			var b bytes.Buffer
			err = template.Run(&b, vars, nil)
			if err != nil {
				t.Fatalf("unexpected run error: %s", err)
			}
			if got := b.String(); got != tt.expected {
				t.Fatalf("expecting %q, got %q", tt.expected, got)
			}
		})
	}
}

// TestStringEscapedBackslash tests that a string ending with an escaped
// backslash is closed by the following quote, so the values shown after it
// are not in the string context.
func TestStringEscapedBackslash(t *testing.T) {
	tests := []struct {
		name     string
		files    fstest.Files
		expected string
		err      string
	}{
		{
			name:     "JS single quoted string",
			files:    fstest.Files{"index.html": `<script>var s = '\\'; var r = {{ v }};</script>`},
			expected: `<script>var s = '\\'; var r = "a";</script>`,
		},
		{
			name:     "JS double quoted string",
			files:    fstest.Files{"index.html": `<script>var s = "\\"; var r = {{ v }};</script>`},
			expected: `<script>var s = "\\"; var r = "a";</script>`,
		},
		{
			name:     "JS file",
			files:    fstest.Files{"index.js": `var s = '\\'; var r = {{ v }};`},
			expected: `var s = '\\'; var r = "a";`,
		},
		{
			name:     "CSS string",
			files:    fstest.Files{"index.html": `<style>a::before { content: '\\'; font-family: {{ v }}; }</style>`},
			expected: `<style>a::before { content: '\\'; font-family: "a"; }</style>`,
		},
		{
			name:     "JSON string",
			files:    fstest.Files{"index.html": `<script type="application/ld+json">{"s": "\\", "r": {{ v }} }</script>`},
			expected: `<script type="application/ld+json">{"s": "\\", "r": "a" }</script>`,
		},
		{
			name:  "backslash before a show",
			files: fstest.Files{"index.html": `<script>var s = "\{{ v }}";</script>`},
			err:   "index.html:1:18: syntax error: unfinished escape sequence in JavaScript string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := "index.html"
			if _, ok := tt.files[name]; !ok {
				name = "index.js"
			}
			vars := Vars{"v": "a"}
			opts := &scriggo.BuildOptions{Globals: asDeclarations(vars)}
			template, err := scriggo.BuildTemplate(tt.files, name, opts)
			if err != nil {
				if tt.err == "" {
					t.Fatalf("unexpected build error: %s", err)
				}
				if err.Error() != tt.err {
					t.Fatalf("expecting build error %q, got %q", tt.err, err)
				}
				return
			}
			if tt.err != "" {
				t.Fatalf("expecting build error %q, got no error", tt.err)
			}
			var b bytes.Buffer
			err = template.Run(&b, vars, nil)
			if err != nil {
				t.Fatalf("unexpected run error: %s", err)
			}
			if got := b.String(); got != tt.expected {
				t.Fatalf("expecting %q, got %q", tt.expected, got)
			}
		})
	}
}

// TestRenderInStringAndAttributeContexts tests that the content of a file
// rendered in a tag, attribute, string or Markdown code block context is
// escaped, as it is when the rendered content is first assigned to a
// variable, while it is written as is in the HTML context.
func TestRenderInStringAndAttributeContexts(t *testing.T) {
	tests := []struct {
		name     string
		main     string
		src      string // {X} is replaced with the shown expression
		file     string
		content  string
		expected string
	}{
		{"quoted attribute", "index.html", `<a title="{X}">x</a>`, "p.html", `"><i>`, `<a title="&#34;&gt;&lt;i&gt;">x</a>`},
		{"unquoted attribute", "index.html", `<a title={X}>x</a>`, "p.html", `"><i>`, `<a title=&#34;&gt;&lt;i&gt;>x</a>`},
		{"URL attribute", "index.html", `<a href="{X}">x</a>`, "p.html", `"><i>`, `<a href="%22%3e%3ci%3e">x</a>`},
		{"tag", "index.html", `<a {X}>x</a>`, "p.html", `"><i>`, "<a \uFFFD\uFFFD<i\uFFFD>x</a>"},
		{"JS string", "index.html", `<script>var s = '{X}';</script>`, "p.js", `it's`, `<script>var s = 'it\u0027s';</script>`},
		{"CSS string", "index.html", `<style>a::before { content: "{X}"; }</style>`, "p.css", `a"b`, `<style>a::before { content: "a\22 b"; }</style>`},
		{"JSON string", "index.html", `<script type="application/ld+json">{"a": "{X}"}</script>`, "p.json", `a"b`, `<script type="application/ld+json">{"a": "a\"b"}</script>`},
		{"Markdown code block", "index.md", "text\n\n    x {X}\n", "p.md", "a\nb", "text\n\n    x a\n    b\n"},
		{"HTML", "index.html", `<div>{X}</div>`, "p.html", `<b>x</b>`, `<div><b>x</b></div>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sources := map[string]string{
				"render":          strings.Replace(tt.src, "{X}", `{{ render "`+tt.file+`" }}`, 1),
				"render variable": `{% var r = render "` + tt.file + `" %}` + strings.Replace(tt.src, "{X}", "{{ r }}", 1),
			}
			for name, src := range sources {
				fsys := fstest.Files{tt.main: src, tt.file: tt.content}
				template, err := scriggo.BuildTemplate(fsys, tt.main, nil)
				if err != nil {
					t.Fatalf("%s: unexpected build error: %s", name, err)
				}
				var b bytes.Buffer
				err = template.Run(&b, nil, nil)
				if err != nil {
					t.Fatalf("%s: unexpected run error: %s", name, err)
				}
				if got := b.String(); got != tt.expected {
					t.Fatalf("%s: expecting %q, got %q", name, tt.expected, got)
				}
			}
		})
	}
}

// TestJSTemplateLiteralAndRegExpContexts tests the values shown in
// template literal substitutions, in regular expressions and after them.
func TestJSTemplateLiteralAndRegExpContexts(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		v        string
		expected string
	}{
		{"substitution", "<script>const s = `${ {{ v }} }`;</script>", "a", "<script>const s = `${ \"a\" }`;</script>"},
		{"after a template literal", "<script>const s = `It's` + {{ v }};</script>", "a", "<script>const s = `It's` + \"a\";</script>"},
		{"after a regular expression", "<script>const s = /'/.test(t) ? {{ v }} : 0;</script>", "a", "<script>const s = /'/.test(t) ? \"a\" : 0;</script>"},
		{"after a macro", "{% macro M js %}a{% end %}<script>/'/.test(t); const s = {{ v }};</script>", "a", "<script>/'/.test(t); const s = \"a\";</script>"},
		{"regular expression", "<script>const r = /^{{ v }}$/;</script>", "a.b/c", "<script>const r = /^a\\.b\\/c$/;</script>"},
		{"empty regular expression", "<script>const r = /{{ v }}/;</script>", "", "<script>const r = /(?:)/;</script>"},
		{"regular expression class", "<script>const r = /[{{ v }}]/;</script>", "a-]", "<script>const r = /[a\\u002d\\]]/;</script>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := Vars{"v": tt.v}
			opts := &scriggo.BuildOptions{Globals: asDeclarations(vars)}
			fsys := fstest.Files{"index.html": tt.src}
			template, err := scriggo.BuildTemplate(fsys, "index.html", opts)
			if err != nil {
				t.Fatalf("unexpected build error: %s", err)
			}
			var b bytes.Buffer
			err = template.Run(&b, vars, nil)
			if err != nil {
				t.Fatalf("unexpected run error: %s", err)
			}
			if got := b.String(); got != tt.expected {
				t.Fatalf("expecting %q, got %q", tt.expected, got)
			}
		})
	}
}

// TestJSCommentContexts tests the values shown in JavaScript comments and
// after them.
func TestJSCommentContexts(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		v        string
		expected string
	}{
		{"value in a block comment", "<script>/* {{ v }} */</script>", "a*/b", "<script>/* \"a*\\/b\" */</script>"},
		{"value in a line comment", "<script>// {{ v }}\n</script>", "a\nb", "<script>// \"a\\nb\"\n</script>"},
		{"decrement and comparison before a regular expression", "<script>while (n-->0) r = /{{ v }}/;</script>", "a/b", "<script>while (n-->0) r = /a\\/b/;</script>"},
		{"decrement and comparison before a string", "<script>while (n-->0) s = '{{ v }}';</script>", "a", "<script>while (n-->0) s = 'a';</script>"},
		{"after an HTML-like comment", "<script><!-- it's\nconst s = {{ v }};</script>", "a", "<script><!-- it's\nconst s = \"a\";</script>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := Vars{"v": tt.v}
			opts := &scriggo.BuildOptions{Globals: asDeclarations(vars)}
			fsys := fstest.Files{"index.html": tt.src}
			template, err := scriggo.BuildTemplate(fsys, "index.html", opts)
			if err != nil {
				t.Fatalf("unexpected build error: %s", err)
			}
			var b bytes.Buffer
			err = template.Run(&b, vars, nil)
			if err != nil {
				t.Fatalf("unexpected run error: %s", err)
			}
			if got := b.String(); got != tt.expected {
				t.Fatalf("expecting %q, got %q", tt.expected, got)
			}
		})
	}
}

// TestStatementBranchContexts tests the contexts after the branches of
// statements and after macros.
func TestStatementBranchContexts(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		src      string
		expected string
		err      string
	}{
		{"if branch", "index.html", "<script>const r = {% if true %}'Admin: {% else %}'User: {% end %}' + {{ v }};</script>", "<script>const r = 'Admin: ' + \"a\";</script>", ""},
		{"else branch", "index.html", "<script>const r = {% if false %}'Admin: {% else %}'User: {% end %}' + {{ v }};</script>", "<script>const r = 'User: ' + \"a\";</script>", ""},
		{"after a macro", "index.js", "{% macro M %}a{% end %}/'/.test(x); var s = {{ v }};", "/'/.test(x); var s = \"a\";", ""},
		{"in a macro", "index.js", "var a = b{% macro M(s string) %}/'/.test(x) + {{ s }}{% end %};\n{{ M(v) }}", "var a = b;\n/'/.test(x) + \"a\"", ""},
		{"raw in a macro", "index.js", "{% macro M %}{% raw %}'{% end raw %}{% end %}var s = '{{ v }}';", "var s = 'a';", ""},
		{"fallthrough", "index.html", "<script>{% switch 1 %}{% case 1 %}var s = {{ v }}{% fallthrough %}{% case 2 %};{% end %}</script>", "<script>var s = \"a\";</script>", ""},
		{"loop re-entry", "index.html", "<script>var a = 1;{% for x in []int{1, 2} %}/'/.test(x) + '{{ v }}'; a{% end %}</script>", "", "index.html:1:45: syntax error: '/' could start a division or a regular expression"},
		{"different contexts", "index.html", "<script>{% if false %}'{% end %} var r = {{ v }};</script>", "", "index.html:1:27: syntax error: if branches end in different contexts"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := Vars{"v": "a"}
			opts := &scriggo.BuildOptions{Globals: asDeclarations(vars)}
			fsys := fstest.Files{tt.file: tt.src}
			template, err := scriggo.BuildTemplate(fsys, tt.file, opts)
			if err != nil {
				if tt.err == "" {
					t.Fatalf("unexpected build error: %s", err)
				}
				if err.Error() != tt.err {
					t.Fatalf("expecting build error %q, got %q", tt.err, err)
				}
				return
			}
			if tt.err != "" {
				t.Fatalf("expecting build error %q, got no error", tt.err)
			}
			var b bytes.Buffer
			err = template.Run(&b, vars, nil)
			if err != nil {
				t.Fatalf("unexpected run error: %s", err)
			}
			if got := b.String(); got != tt.expected {
				t.Fatalf("expecting %q, got %q", tt.expected, got)
			}
		})
	}
}

// TestTypeAttributeNotKnown tests that the content of a script or style
// element whose type attribute is not known at compile time is escaped as
// JavaScript or CSS.
func TestTypeAttributeNotKnown(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		expected string
	}{
		{"script with show", `<script type="{{ t }}">var s = {{ v }};</script>`, `<script type="module">var s = "a";</script>`},
		{"script with statement", `<script type="text/{% if true %}plain{% end %}">var s = {{ v }};</script>`, `<script type="text/plain">var s = "a";</script>`},
		{"style with statement", `<style type="text/{% if true %}css{% end %}">a { font-family: {{ v }}; }</style>`, `<style type="text/css">a { font-family: "a"; }</style>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := Vars{"t": "module", "v": "a"}
			opts := &scriggo.BuildOptions{Globals: asDeclarations(vars)}
			fsys := fstest.Files{"index.html": tt.src}
			template, err := scriggo.BuildTemplate(fsys, "index.html", opts)
			if err != nil {
				t.Fatalf("unexpected build error: %s", err)
			}
			var b bytes.Buffer
			err = template.Run(&b, vars, nil)
			if err != nil {
				t.Fatalf("unexpected run error: %s", err)
			}
			if got := b.String(); got != tt.expected {
				t.Fatalf("expecting %q, got %q", tt.expected, got)
			}
		})
	}
}

var jsonContextTests = []struct {
	src  string
	res  string
	vars Vars
}{
	{"t", `"2016-01-02T15:04:05Z"`, Vars{"t": time.Date(2016, 1, 2, 15, 04, 05, 0, time.UTC)}},
}

func TestJSONContext(t *testing.T) {
	for _, expr := range jsonContextTests {
		fsys := fstest.Files{"index.html": `<script type="application/ld+json">{{` + expr.src + `}}</script>`}
		opts := &scriggo.BuildOptions{
			Globals: asDeclarations(expr.vars),
		}
		template, err := scriggo.BuildTemplate(fsys, "index.html", opts)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = template.Run(b, expr.vars, nil)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var res = b.String()
		if len(res) < 44 || res[35:len(res)-9] != expr.res {
			t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res[35:len(res)-9], expr.res)
		}
	}
}

var jsStringContextTests = []struct {
	src  string
	res  string
	vars Vars
}{
	{`""`, ``, nil},
	{`"a"`, `a`, nil},
	{`"<a>"`, `\u003ca\u003e`, nil},
	{`"\\"`, `\\`, nil},
	{`"\""`, `\"`, nil},
	{`"'"`, `\u0027`, nil},
	{`"\n"`, `\n`, nil},
	{`"\r"`, `\r`, nil},
	{`"\t"`, `\t`, nil},
	{`"\\\"\n\r\t\u2028\u2029\u0000\u0010"`, `\\\"\n\r\t\u2028\u2029\u0000\u0010`, nil},
	{`25`, "25", nil},
	{`0.1`, "0.1", nil},
	{`a`, `a`, Vars{"a": "a"}},
	{`a`, `\u003c\u003e\"`, Vars{"a": "<>\""}},
	{`"$"`, `\u0024`, nil},
	{`"{}"`, `\u007b}`, nil},
	{`a`, `\u0060\u0024\u007ba}\u0060`, Vars{"a": "`${a}`"}},
}

func TestJSStringContext(t *testing.T) {
	for _, q := range []string{"\"", "'"} {
		for _, expr := range jsStringContextTests {
			fsys := fstest.Files{"index.html": "<script>" + q + "{{" + expr.src + "}}" + q + "</script>"}
			opts := &scriggo.BuildOptions{
				Globals: asDeclarations(expr.vars),
			}
			template, err := scriggo.BuildTemplate(fsys, "index.html", opts)
			if err != nil {
				t.Errorf("source: %q, %s\n", expr.src, err)
				continue
			}
			var b = &bytes.Buffer{}
			err = template.Run(b, expr.vars, nil)
			if err != nil {
				t.Errorf("source: %q, %s\n", expr.src, err)
				continue
			}
			var res = b.String()
			if len(res) < 19 || res[9:len(res)-10] != expr.res {
				t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res[8:len(res)-9], expr.res)
			}
		}
	}
}

var cssContextTests = []struct {
	src  string
	res  string
	vars Vars
}{
	{`""`, `""`, nil},
	{`"a"`, `"a"`, nil},
	{`"<a>"`, `"\3c a\3e "`, nil},
	{`a`, `"\3c a\3e "`, Vars{"a": native.HTML("<a>")}},
	{`5`, `5`, nil},
	{`5.2`, `5.2`, nil},
	{`a`, `AAECAwQF`, Vars{"a": []byte{0, 1, 2, 3, 4, 5}}},
}

func TestCSSContext(t *testing.T) {
	for _, expr := range cssContextTests {
		fsys := fstest.Files{"index.html": "<style>{{" + expr.src + "}}</style>"}
		opts := &scriggo.BuildOptions{
			Globals: asDeclarations(expr.vars),
		}
		template, err := scriggo.BuildTemplate(fsys, "index.html", opts)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = template.Run(b, expr.vars, nil)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var res = b.String()
		if len(res) < 15 || res[7:len(res)-8] != expr.res {
			t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res[7:len(res)-8], expr.res)
		}
	}
}

var cssStringContextTests = []struct {
	src  string
	res  string
	vars Vars
}{
	{`""`, ``, nil},
	{`"\u0000"`, `\0 `, nil},
	{`"\u000F"`, `\f `, nil},
	{`"\u001F"`, `\1f `, nil},
	{`"a"`, `a`, nil},
	{`"<a>"`, `\3c a\3e `, nil},
	{`a`, `\3c a\3e `, Vars{"a": native.HTML("<a>")}},
	{`"\\"`, `\\`, nil},
	{`"\""`, `\22 `, nil},
	{`"'"`, `\27 `, nil},
	{`"\n"`, `\a `, nil},
	{`"\r"`, `\d `, nil},
	{`"\t"`, `\9 `, nil},
	{`25`, "25", nil},
	{`0.1`, "0.1", nil},
	{`a`, `a`, Vars{"a": "a"}},
	{`a`, `\3c\3e\22 `, Vars{"a": "<>\""}},
	{`a`, `AAECAwQF`, Vars{"a": []byte{0, 1, 2, 3, 4, 5}}},
}

func TestCSSStringContext(t *testing.T) {
	for _, q := range []string{"\"", "'"} {
		for _, expr := range cssStringContextTests {
			fsys := fstest.Files{"index.html": "<style>" + q + "{{" + expr.src + "}}" + q + "</style>"}
			opts := &scriggo.BuildOptions{
				Globals: asDeclarations(expr.vars),
			}
			template, err := scriggo.BuildTemplate(fsys, "index.html", opts)
			if err != nil {
				t.Errorf("source: %q, %s\n", expr.src, err)
				continue
			}
			var b = &bytes.Buffer{}
			err = template.Run(b, expr.vars, nil)
			if err != nil {
				t.Errorf("source: %q, %s\n", expr.src, err)
				continue
			}
			var res = b.String()
			if len(res) < 17 || res[8:len(res)-9] != expr.res {
				t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res[8:len(res)-9], expr.res)
			}
		}
	}
}

func asDeclarations(vars Vars) native.Declarations {
	declarations := globals()
	for name, value := range vars {
		declarations[name] = reflect.Zero(reflect.PointerTo(reflect.TypeOf(value))).Interface()
	}
	return declarations
}
