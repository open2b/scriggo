// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"reflect"
	"testing"

	"scriggo"
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
	{`a`, "<a>", Vars{"a": HTML("<a>")}},
	{`d`, "<div></div>", Vars{"d": HTML("<div></div>")}},
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
}

func TestHTMLContext(t *testing.T) {
	for _, expr := range htmlContextTests {
		r := MapReader{"index.html": []byte("{{" + expr.src + "}}")}
		tmpl, err := Load("index.html", r, mainPackage(expr.vars), ContextHTML, &LoadOptions{LimitMemorySize: true})
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, &RenderOptions{MaxMemorySize: 1000})
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

var attributeContextTests = []struct {
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
	{`a`, "&lt;a&gt;", Vars{"a": HTML("<a>")}},
	{`d`, "&lt;div&gt;&lt;/div&gt;", Vars{"d": HTML("<div></div>")}},
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

func TestAttributeContext(t *testing.T) {
	for _, expr := range attributeContextTests {
		r := MapReader{"index.html": []byte(`<z x="{{` + expr.src + `}}">`)}
		tmpl, err := Load("index.html", r, mainPackage(expr.vars), ContextHTML, &LoadOptions{LimitMemorySize: true})
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, &RenderOptions{MaxMemorySize: 1000})
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

var unquotedAttributeContextTests = []struct {
	src  string
	res  string
	vars Vars
}{
	{`a`, "&#32;a&#32;", Vars{"a": " a "}},
	{`a`, "&#09;&#10;&#13;&#12;&#32;a&#61;&#96;", Vars{"a": "\t\n\r\x0C a=`"}},
	{`s["a"]`, "", Vars{"s": map[interface{}]interface{}{}}},
}

func TestUnquotedAttributeContext(t *testing.T) {
	for _, expr := range unquotedAttributeContextTests {
		r := MapReader{"index.html": []byte(`<z x={{` + expr.src + `}}>`)}
		tmpl, err := Load("index.html", r, mainPackage(expr.vars), ContextHTML, &LoadOptions{LimitMemorySize: true})
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, &RenderOptions{MaxMemorySize: 1000})
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

var javaScriptContextTests = []struct {
	src  string
	res  string
	vars Vars
}{
	{`""`, `""`, nil},
	{`"a"`, `"a"`, nil},
	{`"<a>"`, `"\x3ca\x3e"`, nil},
	{`"<div></div>"`, `"\x3cdiv\x3e\x3c/div\x3e"`, nil},
	{`"\\"`, `"\\"`, nil},
	{`"\""`, `"\""`, nil},
	{`"\n"`, `"\n"`, nil},
	{`"\r"`, `"\r"`, nil},
	{`"\t"`, `"\t"`, nil},
	{`"\\\"\n\r\t\u2028\u2029\u0000\u0010"`, `"\\\"\n\r\t\u2028\u2029\x00\x10"`, nil},
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
	{`a`, `"\x3c\x3e\""`, Vars{"a": "<>\""}},
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
}

func TestScriptContext(t *testing.T) {
	for _, expr := range javaScriptContextTests {
		r := MapReader{"index.html": []byte("<script>{{" + expr.src + "}}</script>")}
		tmpl, err := Load("index.html", r, mainPackage(expr.vars), ContextHTML, &LoadOptions{LimitMemorySize: true})
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, &RenderOptions{MaxMemorySize: 1000})
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

var javaScriptStringContextTests = []struct {
	src  string
	res  string
	vars Vars
}{
	{`""`, ``, nil},
	{`"a"`, `a`, nil},
	{`"<a>"`, `\x3ca\x3e`, nil},
	{`"\\"`, `\\`, nil},
	{`"\""`, `\"`, nil},
	{`"'"`, `\'`, nil},
	{`"\n"`, `\n`, nil},
	{`"\r"`, `\r`, nil},
	{`"\t"`, `\t`, nil},
	{`"\\\"\n\r\t\u2028\u2029\u0000\u0010"`, `\\\"\n\r\t\u2028\u2029\x00\x10`, nil},
	{`25`, "25", nil},
	{`0.1`, "0.1", nil},
	{`a`, `a`, Vars{"a": "a"}},
	{`a`, `\x3c\x3e\"`, Vars{"a": "<>\""}},
}

func TestJavaScriptStringContext(t *testing.T) {
	for _, q := range []string{"\"", "'"} {
		for _, expr := range javaScriptStringContextTests {
			r := MapReader{"index.html": []byte("<script>" + q + "{{" + expr.src + "}}" + q + "</script>")}
			tmpl, err := Load("index.html", r, mainPackage(expr.vars), ContextHTML, &LoadOptions{LimitMemorySize: true})
			if err != nil {
				t.Errorf("source: %q, %s\n", expr.src, err)
				continue
			}
			var b = &bytes.Buffer{}
			err = tmpl.Render(b, expr.vars, &RenderOptions{MaxMemorySize: 1000})
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
	{`HTML("<a>")`, `"\3c a\3e "`, nil},
	{`5`, `5`, nil},
	{`5.2`, `5.2`, nil},
	{`a`, `AAECAwQF`, Vars{"a": []byte{0, 1, 2, 3, 4, 5}}},
}

func TestCSSContext(t *testing.T) {
	for _, expr := range cssContextTests {
		r := MapReader{"index.html": []byte("<style>{{" + expr.src + "}}</style>")}
		tmpl, err := Load("index.html", r, mainPackage(expr.vars), ContextHTML, &LoadOptions{LimitMemorySize: true})
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, &RenderOptions{MaxMemorySize: 1000})
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
	{`HTML("<a>")`, `\3c a\3e `, nil},
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
			r := MapReader{"index.html": []byte("<style>" + q + "{{" + expr.src + "}}" + q + "</style>")}
			tmpl, err := Load("index.html", r, mainPackage(expr.vars), ContextHTML, &LoadOptions{LimitMemorySize: true})
			if err != nil {
				t.Errorf("source: %q, %s\n", expr.src, err)
				continue
			}
			var b = &bytes.Buffer{}
			err = tmpl.Render(b, expr.vars, &RenderOptions{MaxMemorySize: 1000})
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

func mainPackage(vars Vars) scriggo.Package {
	p := scriggo.MapPackage{
		PkgName:      "main",
		Declarations: map[string]interface{}{},
	}
	for _, name := range main.DeclarationNames() {
		p.Declarations[name] = main.Lookup(name)
	}
	for name, value := range vars {
		p.Declarations[name] = reflect.Zero(reflect.PtrTo(reflect.TypeOf(value))).Interface()
	}
	return &p
}
