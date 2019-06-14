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
	//{`nil`, "", nil}, TODO: runtime error: invalid memory address or nil pointer dereference
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
	{`a`, "0, 1, 2, 3, 4, 5", Vars{"a": []int{0, 1, 2, 3, 4, 5}}},
	{`a`, "-2, -1, 0, 1, 2", Vars{"a": []int{-2, -1, 0, 1, 2}}},
	{`a`, "true, false, true", Vars{"a": []bool{true, false, true}}},
	//{`s["a"]`, "", Vars{"s": map[string]string{}}}, TODO: invalid operation: s["a"] (type interface {} does not support indexing)
}

func TestHTMLContext(t *testing.T) {
	for _, expr := range htmlContextTests {
		r := MapReader{"/index.html": []byte("{{" + expr.src + "}}")}
		tmpl, err := Load("/index.html", r, mainPackage(expr.vars), ContextHTML, LimitMemorySize)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, RenderOptions{MaxMemorySize: 1000})
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
	//{`nil`, "", nil}, TODO: runtime error: invalid memory address or nil pointer dereference
	{`""`, "", nil},
	{`"a"`, "a", nil},
	{`"<a>"`, "&lt;a&gt;", nil},
	{`"<div></div>"`, "&lt;div&gt;&lt;/div&gt;", nil},
	{`a`, "&lt;a&gt;", Vars{"a": "<a>"}},
	{`d`, "&lt;div&gt;&lt;/div&gt;", Vars{"d": "<div></div>"}},
	{`a`, "&lt;a&gt;", Vars{"a": HTML("<a>")}},
	{`a`, "&amp;", Vars{"a": HTML("&amp;")}},
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
	{`a`, "0, 1, 2, 3, 4, 5", Vars{"a": []int{0, 1, 2, 3, 4, 5}}},
	{`a`, "-2, -1, 0, 1, 2", Vars{"a": []int{-2, -1, 0, 1, 2}}},
	{`a`, "true, false, true", Vars{"a": []bool{true, false, true}}},
	//{`s["a"]`, "", Vars{"s": map[interface{}]interface{}{}}},  TODO: invalid operation: s["a"] (type interface {} does not support indexing)
}

func TestAttributeContext(t *testing.T) {
	for _, expr := range attributeContextTests {
		r := MapReader{"/index.html": []byte(`<z x="{{` + expr.src + `}}">`)}
		tmpl, err := Load("/index.html", r, mainPackage(expr.vars), ContextHTML, LimitMemorySize)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, RenderOptions{MaxMemorySize: 1000})
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
	{`a`, "0,&#32;1,&#32;2", Vars{"a": []int{0, 1, 2}}},
	//{`s["a"]`, "", Vars{"s": map[interface{}]interface{}{}}},  TODO: invalid operation: s["a"] (type interface {} does not support indexing)
}

func TestUnquotedAttributeContext(t *testing.T) {
	for _, expr := range unquotedAttributeContextTests {
		r := MapReader{"/index.html": []byte(`<z x={{` + expr.src + `}}>`)}
		tmpl, err := Load("/index.html", r, mainPackage(expr.vars), ContextHTML, LimitMemorySize)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, RenderOptions{MaxMemorySize: 1000})
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

var urlContextTests = []struct {
	src  string
	res  string
	vars Vars
}{
	{`<a href="">`, `<a href="">`, nil},
	{`<a href="abc">`, `<a href="abc">`, nil},
	{`<a href="本">`, `<a href="本">`, nil},
	{`<a href="{{b}}">`, `<a href="b">`, Vars{"b": "b"}},
	{`<a href="{{b}}">`, `<a href="/">`, Vars{"b": "/"}},
	{`<a href="{{b}}">`, `<a href="http://www.site.com/">`, Vars{"b": "http://www.site.com/"}},
	{`<a href="{{b}}">`, `<a href=" http://www.site.com/ ">`, Vars{"b": " http://www.site.com/ "}},
	{`<a href="http://s/{{a}}/">`, `<a href="http://s/a%c3%a0%e6%9c%ac/">`, Vars{"a": "aà本"}},
	{`<a href="{{a}}{{b}}">`, `<a href="a%e6%9c%ac">`, Vars{"a": "a", "b": "本"}},
	{`<a href="{{a}}?b={{b}}">`, `<a href="a?b=%20">`, Vars{"a": "a", "b": " "}},
	{`<a href="{{a}}?b={{b}}">`, `<a href="a?b=%3d">`, Vars{"a": "a", "b": "="}},
	{`<a href="{{a}}?b={{b}}">`, `<a href="=?b=%3d">`, Vars{"a": "=", "b": "="}},
	{`<a href="{{a}}?b={{b}}">`, `<a href="p?b=%3d">`, Vars{"a": "p?", "b": "="}},
	{`<a href="{{a}}?b={{b}}">`, `<a href="p?&amp;b=%3d">`, Vars{"a": "p?&", "b": "="}},
	{`<a href="{{a}}?b={{b}}">`, `<a href="p?q&amp;b=%3d">`, Vars{"a": "p?q", "b": "="}},
	{`<a href="{{a}}?b={{b}}">`, `<a href="?b=%3d">`, Vars{"a": "?", "b": "="}},
	{`<a href="{{a}}?b={{b}}&c={{c}}">`, `<a href="/a/b/c?b=%3d&c=6">`, Vars{"a": "/a/b/c", "b": "=", "c": "6"}},
	{`<a href="{{a}}?b={{b}}&amp;c={{c}}">`, `<a href="/a/b/c?b=%3d&amp;c=6">`, Vars{"a": "/a/b/c", "b": "=", "c": "6"}},
	{`<a href="{{a}}?{{b}}">`, `<a href="?">`, Vars{"a": "", "b": ""}},
	{`<a href="?{{b}}">`, `<a href="?b">`, Vars{"b": "b"}},
	{`<a href="?{{b}}">`, `<a href="?%3d">`, Vars{"b": "="}},
	{`<a href="#">`, `<a href="#">`, nil},
	{`<a href="#{{b}}">`, `<a href="#%3d">`, Vars{"b": "="}},
	{`<a href="{{a}}#{{b}}">`, `<a href="=#%3d">`, Vars{"a": "=", "b": "="}},
	{`<a href="{{a}}?{{b}}#{{c}}">`, `<a href="=?%3d#%3d">`, Vars{"a": "=", "b": "=", "c": "="}},
	{`<a href="{{a}}?b=6#{{c}}">`, `<a href="=?b=6#%3d">`, Vars{"a": "=", "c": "="}},
	{`<a href="{{a}}?{{b}}">`, `<a href=",?%2c">`, Vars{"a": ",", "b": ","}},
	{`<img srcset="{{a}} 1024w, {{b}} 640w,{{c}} 320w">`, `<img srcset="large.jpg 1024w, medium.jpg 640w,small.jpg 320w">`, Vars{"a": "large.jpg", "b": "medium.jpg", "c": "small.jpg"}},
	{`<img srcset="{{a}} 1024w, {{b}} 640w">`, `<img srcset="large.jpg?s=1024 1024w, medium.jpg 640w">`, Vars{"a": "large.jpg?s=1024", "b": "medium.jpg"}},
	{`<img srcset="{{a}} 1024w, {{b}} 640w">`, `<img srcset="large.jpg?s=1024 1024w, medium=.jpg 640w">`, Vars{"a": "large.jpg?s=1024", "b": "medium=.jpg"}},
	{`<a href="{% if t %}{{ a }}{% else %}?{{ a }}{% end %}">`, `<a href="=">`, Vars{"a": "=", "t": true}},
	{`<a href="{% if t %}{{ a }}{% else %}?{{ a }}{% end %}">`, `<a href="?%3d">`, Vars{"a": "=", "t": false}},
	{`<input {{ a }}>`, `<input disabled>`, Vars{"a": "disabled"}},
	{`<a href={{b}}>`, `<a href=b>`, Vars{"b": "b"}},
	{`<a href={{b}}>`, `<a href=&#32;b&#32;>`, Vars{"b": " b "}},
	{`<a href= {{b}} >`, `<a href= &#32;b&#32; >`, Vars{"b": " b "}},
	{`<a href= {{b}} >`, `<a href= %09%0a%0d%0c&#32;b=%60 >`, Vars{"b": "\t\n\r\x0C b=`"}},
	{`<a href="{{ s["a"] }}">`, "<a href=\"\">", Vars{"s": map[interface{}]interface{}{}}},
}

func TestURLContext(t *testing.T) {

	// TODO: unexpected node 1:10 (type *ast.URL)
	t.Skip("(not runnable)")

	for _, expr := range urlContextTests {
		r := MapReader{"/index.html": []byte(expr.src)}
		tmpl, err := Load("/index.html", r, mainPackage(expr.vars), ContextHTML, LimitMemorySize)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, RenderOptions{MaxMemorySize: 1000})
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

var javaScriptContextTests = []struct {
	src  string
	res  string
	vars Vars
}{
	//{`nil`, `null`, nil}, TODO: untime error: invalid memory address or nil pointer dereference
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
	//{`s["a"]`, "null", Vars{"s": map[interface{}]interface{}{}}}, TODO: invalid operation: s["a"] (type interface {} does not support indexing)
}

func TestScriptContext(t *testing.T) {
	for _, expr := range javaScriptContextTests {
		r := MapReader{"/index.html": []byte("<script>{{" + expr.src + "}}</script>")}
		tmpl, err := Load("/index.html", r, mainPackage(expr.vars), ContextHTML, LimitMemorySize)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, RenderOptions{MaxMemorySize: 1000})
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
			r := MapReader{"/index.html": []byte("<script>" + q + "{{" + expr.src + "}}" + q + "</script>")}
			tmpl, err := Load("/index.html", r, mainPackage(expr.vars), ContextHTML, LimitMemorySize)
			if err != nil {
				t.Errorf("source: %q, %s\n", expr.src, err)
				continue
			}
			var b = &bytes.Buffer{}
			err = tmpl.Render(b, expr.vars, RenderOptions{MaxMemorySize: 1000})
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
	{`html("<a>")`, `"\3c a\3e "`, nil},
	{`5`, `5`, nil},
	{`5.2`, `5.2`, nil},
	{`a`, `AAECAwQF`, Vars{"a": []byte{0, 1, 2, 3, 4, 5}}},
}

func TestCSSContext(t *testing.T) {
	for _, expr := range cssContextTests {
		r := MapReader{"/index.html": []byte("<style>{{" + expr.src + "}}</style>")}
		tmpl, err := Load("/index.html", r, mainPackage(expr.vars), ContextHTML, LimitMemorySize)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, RenderOptions{MaxMemorySize: 1000})
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
	{`html("<a>")`, `\3c a\3e `, nil},
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
			r := MapReader{"/index.html": []byte("<style>" + q + "{{" + expr.src + "}}" + q + "</style>")}
			tmpl, err := Load("/index.html", r, mainPackage(expr.vars), ContextHTML, LimitMemorySize)
			if err != nil {
				t.Errorf("source: %q, %s\n", expr.src, err)
				continue
			}
			var b = &bytes.Buffer{}
			err = tmpl.Render(b, expr.vars, RenderOptions{MaxMemorySize: 1000})
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

func mainPackage(vars Vars) *scriggo.Package {
	main := Builtins()
	for name, value := range vars {
		main.Declarations[name] = reflect.Zero(reflect.PtrTo(reflect.TypeOf(value))).Interface()
	}
	return main
}
