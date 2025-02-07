// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package misc

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/internal/fstest"
	"github.com/open2b/scriggo/native"
)

type Vars map[string]interface{}

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
	{`s["a"]`, "", Vars{"s": map[interface{}]interface{}{}}},
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
	{`s["a"]`, "", Vars{"s": map[interface{}]interface{}{}}},
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
	{`s["a"]`, "null", Vars{"s": map[interface{}]interface{}{}}},
	{`a`, `{"A":5,"B":2,"C":7,"D":3}`, Vars{"a": map[string]interface{}{"A": 5, "B": 2, "C": 7, "D": 3}}},
	{`a`, `{"":"c","\"\u0027":5}`, Vars{"a": map[string]interface{}{"\"'": 5, "": "c"}}},
	{`a`, `{"a\u0027b":3}`, Vars{"a": map[string]interface{}{"a'b": 3}}},
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
