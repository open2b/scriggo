// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"bytes"
	"testing"

	"scrigo/ast"
	"scrigo/parser"
)

var htmlContextTests = []struct {
	src     interface{}
	res     string
	globals scope
}{
	{`nil`, "", nil},
	{`""`, "", nil},
	{`"a"`, "a", nil},
	{`"<a>"`, "&lt;a&gt;", nil},
	{`"<div></div>"`, "&lt;div&gt;&lt;/div&gt;", nil},
	{`a`, "&lt;a&gt;", scope{"a": "<a>"}},
	{`a`, "&#34;ab&#39;cd&#34;", scope{"a": "\"ab'cd\""}},
	{`d`, "&lt;div&gt;&lt;/div&gt;", scope{"d": "<div></div>"}},
	{`a`, "<a>", scope{"a": HTML("<a>")}},
	{`d`, "<div></div>", scope{"d": HTML("<div></div>")}},
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
	{`5479159814589435423678645745.523785742365476`, "5.479159814589435e+27", nil},
	{`0.0000000000000000000000000000000000000000000000000000000001`, "1e-58", nil},
	{`-0.1000000`, "-0.1", nil},
	{`true`, "true", nil},
	{`false`, "false", nil},
	{`a`, "0, 1, 2, 3, 4, 5", scope{"a": []int{0, 1, 2, 3, 4, 5}}},
	{`a`, "-2, -1, 0, 1, 2", scope{"a": []int{-2, -1, 0, 1, 2}}},
	{`a`, "true, false, true", scope{"a": []bool{true, false, true}}},
	{`s["a"]`, "", scope{"s": Map{}}},
}

func TestHTMLContext(t *testing.T) {
	for _, expr := range htmlContextTests {
		var src string
		switch s := expr.src.(type) {
		case string:
			src = s
		case HTML:
			src = string(s)
		}
		var tree, err = parser.ParseSource([]byte("{{"+src+"}}"), ast.ContextHTML)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = RenderTree(b, tree, expr.globals, false)
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
	src     interface{}
	res     string
	globals scope
}{
	{`nil`, "", nil},
	{`""`, "", nil},
	{`"a"`, "a", nil},
	{`"<a>"`, "&lt;a&gt;", nil},
	{`"<div></div>"`, "&lt;div&gt;&lt;/div&gt;", nil},
	{`a`, "&lt;a&gt;", scope{"a": "<a>"}},
	{`d`, "&lt;div&gt;&lt;/div&gt;", scope{"d": "<div></div>"}},
	{`a`, "&lt;a&gt;", scope{"a": HTML("<a>")}},
	{`a`, "&amp;", scope{"a": HTML("&amp;")}},
	{`d`, "&lt;div&gt;&lt;/div&gt;", scope{"d": HTML("<div></div>")}},
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
	{`a`, "0, 1, 2, 3, 4, 5", scope{"a": []int{0, 1, 2, 3, 4, 5}}},
	{`a`, "-2, -1, 0, 1, 2", scope{"a": []int{-2, -1, 0, 1, 2}}},
	{`a`, "true, false, true", scope{"a": []bool{true, false, true}}},
	{`s["a"]`, "", scope{"s": Map{}}},
}

func TestAttributeContext(t *testing.T) {
	for _, expr := range attributeContextTests {
		var src string
		switch s := expr.src.(type) {
		case string:
			src = s
		case HTML:
			src = string(s)
		}
		var tree, err = parser.ParseSource([]byte(`<z x="{{`+src+`}}">`), ast.ContextHTML)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = RenderTree(b, tree, expr.globals, false)
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
	src     interface{}
	res     string
	globals scope
}{
	{`a`, "&#32;a&#32;", scope{"a": " a "}},
	{`a`, "&#09;&#10;&#13;&#12;&#32;a&#61;&#96;", scope{"a": "\t\n\r\x0C a=`"}},
	{`a`, "0,&#32;1,&#32;2", scope{"a": []int{0, 1, 2}}},
	{`s["a"]`, "", scope{"s": Map{}}},
}

func TestUnquotedAttributeContext(t *testing.T) {
	for _, expr := range unquotedAttributeContextTests {
		var src string

		switch s := expr.src.(type) {
		case string:
			src = s
		case HTML:
			src = string(s)
		}
		var tree, err = parser.ParseSource([]byte(`<z x={{`+src+`}}>`), ast.ContextHTML)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = RenderTree(b, tree, expr.globals, false)
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
	src     interface{}
	res     string
	globals scope
}{
	{`<a href="">`, `<a href="">`, nil},
	{`<a href="abc">`, `<a href="abc">`, nil},
	{`<a href="本">`, `<a href="本">`, nil},
	{`<a href="{{b}}">`, `<a href="b">`, scope{"b": "b"}},
	{`<a href="{{b}}">`, `<a href="/">`, scope{"b": "/"}},
	{`<a href="{{b}}">`, `<a href="http://www.site.com/">`, scope{"b": "http://www.site.com/"}},
	{`<a href="{{b}}">`, `<a href=" http://www.site.com/ ">`, scope{"b": " http://www.site.com/ "}},
	{`<a href="http://s/{{a}}/">`, `<a href="http://s/a%c3%a0%e6%9c%ac/">`, scope{"a": "aà本"}},
	{`<a href="{{a}}{{b}}">`, `<a href="a%e6%9c%ac">`, scope{"a": "a", "b": "本"}},
	{`<a href="{{a}}?b={{b}}">`, `<a href="a?b=%20">`, scope{"a": "a", "b": " "}},
	{`<a href="{{a}}?b={{b}}">`, `<a href="a?b=%3d">`, scope{"a": "a", "b": "="}},
	{`<a href="{{a}}?b={{b}}">`, `<a href="=?b=%3d">`, scope{"a": "=", "b": "="}},
	{`<a href="{{a}}?b={{b}}">`, `<a href="p?b=%3d">`, scope{"a": "p?", "b": "="}},
	{`<a href="{{a}}?b={{b}}">`, `<a href="p?&amp;b=%3d">`, scope{"a": "p?&", "b": "="}},
	{`<a href="{{a}}?b={{b}}">`, `<a href="p?q&amp;b=%3d">`, scope{"a": "p?q", "b": "="}},
	{`<a href="{{a}}?b={{b}}">`, `<a href="?b=%3d">`, scope{"a": "?", "b": "="}},
	{`<a href="{{a}}?b={{b}}&c={{c}}">`, `<a href="/a/b/c?b=%3d&c=6">`, scope{"a": "/a/b/c", "b": "=", "c": "6"}},
	{`<a href="{{a}}?b={{b}}&amp;c={{c}}">`, `<a href="/a/b/c?b=%3d&amp;c=6">`, scope{"a": "/a/b/c", "b": "=", "c": "6"}},
	{`<a href="{{a}}?{{b}}">`, `<a href="?">`, scope{"a": "", "b": ""}},
	{`<a href="?{{b}}">`, `<a href="?b">`, scope{"b": "b"}},
	{`<a href="?{{b}}">`, `<a href="?%3d">`, scope{"b": "="}},
	{`<a href="#">`, `<a href="#">`, nil},
	{`<a href="#{{b}}">`, `<a href="#%3d">`, scope{"b": "="}},
	{`<a href="{{a}}#{{b}}">`, `<a href="=#%3d">`, scope{"a": "=", "b": "="}},
	{`<a href="{{a}}?{{b}}#{{c}}">`, `<a href="=?%3d#%3d">`, scope{"a": "=", "b": "=", "c": "="}},
	{`<a href="{{a}}?b=6#{{c}}">`, `<a href="=?b=6#%3d">`, scope{"a": "=", "c": "="}},
	{`<a href="{{a}}?{{b}}">`, `<a href=",?%2c">`, scope{"a": ",", "b": ","}},
	{`<img srcset="{{a}} 1024w, {{b}} 640w,{{c}} 320w">`, `<img srcset="large.jpg 1024w, medium.jpg 640w,small.jpg 320w">`, scope{"a": "large.jpg", "b": "medium.jpg", "c": "small.jpg"}},
	{`<img srcset="{{a}} 1024w, {{b}} 640w">`, `<img srcset="large.jpg?s=1024 1024w, medium.jpg 640w">`, scope{"a": "large.jpg?s=1024", "b": "medium.jpg"}},
	{`<img srcset="{{a}} 1024w, {{b}} 640w">`, `<img srcset="large.jpg?s=1024 1024w, medium=.jpg 640w">`, scope{"a": "large.jpg?s=1024", "b": "medium=.jpg"}},
	{`<a href="{% if t %}{{ a }}{% else %}?{{ a }}{% end %}">`, `<a href="=">`, scope{"a": "=", "t": true}},
	{`<a href="{% if t %}{{ a }}{% else %}?{{ a }}{% end %}">`, `<a href="?%3d">`, scope{"a": "=", "t": false}},
	{`<input {{ a }}>`, `<input disabled>`, scope{"a": "disabled"}},
	{`<a href={{b}}>`, `<a href=b>`, scope{"b": "b"}},
	{`<a href={{b}}>`, `<a href=&#32;b&#32;>`, scope{"b": " b "}},
	{`<a href= {{b}} >`, `<a href= &#32;b&#32; >`, scope{"b": " b "}},
	{`<a href= {{b}} >`, `<a href= %09%0a%0d%0c&#32;b=%60 >`, scope{"b": "\t\n\r\x0C b=`"}},
	{`<a href="{{ s["a"] }}">`, "<a href=\"\">", scope{"s": Map{}}},
}

func TestURLContext(t *testing.T) {
	for _, expr := range urlContextTests {
		var src string
		switch s := expr.src.(type) {
		case string:
			src = s
		case HTML:
			src = string(s)
		}
		var tree, err = parser.ParseSource([]byte(src), ast.ContextHTML)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = RenderTree(b, tree, expr.globals, false)
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

var scriptContextTests = []struct {
	src     string
	res     string
	globals scope
}{
	{`nil`, `null`, nil},
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
	{`a`, `"a"`, scope{"a": "a"}},
	{`a`, `"\x3c\x3e\""`, scope{"a": "<>\""}},
	{`a`, "null", scope{"a": []int(nil)}},
	{`a`, "[0,1,2,3,4,5]", scope{"a": []int{0, 1, 2, 3, 4, 5}}},
	{`a`, "[-2,-1,0,1,2]", scope{"a": []int{-2, -1, 0, 1, 2}}},
	{`a`, `"AAECAwQF"`, scope{"a": Bytes{0, 1, 2, 3, 4, 5}}},
	{`a`, `"AAECAwQF"`, scope{"a": []byte{0, 1, 2, 3, 4, 5}}},
	{`a`, "null", scope{"a": []bool(nil)}},
	{`a`, "[true,false,true]", scope{"a": []bool{true, false, true}}},
	{`a`, "null", scope{"a": (*struct{})(nil)}},
	{`a`, "{}", scope{"a": &struct{}{}}},
	{`a`, `{}`, scope{"a": &struct{ a int }{a: 5}}},
	{`a`, `{"A":5}`, scope{"a": &struct{ A int }{A: 5}}},
	{`a`, `{"A":5,"B":null}`, scope{"a": &struct {
		A int
		B *struct{}
	}{A: 5, B: nil}}},
	{`a`, `{"A":5,"B":{}}`, scope{"a": &struct {
		A int
		B *struct{}
	}{A: 5, B: &struct{}{}}}},
	{`a`, `{"A":5,"B":{"C":"C"}}`, scope{"a": &struct {
		A int
		B *struct{ C string }
	}{A: 5, B: &struct{ C string }{C: "C"}}}},
	{`s["a"]`, "null", scope{"s": Map{}}},
}

func TestScriptContext(t *testing.T) {
	for _, expr := range scriptContextTests {
		var tree, err = parser.ParseSource([]byte("<script>{{"+expr.src+"}}</script>"), ast.ContextHTML)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = RenderTree(b, tree, expr.globals, false)
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

var scriptStringContextTests = []struct {
	src     string
	res     string
	globals scope
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
	{`a`, `a`, scope{"a": "a"}},
	{`a`, `\x3c\x3e\"`, scope{"a": "<>\""}},
}

func TestScriptStringContext(t *testing.T) {
	for _, q := range []string{"\"", "'"} {
		for _, expr := range scriptStringContextTests {
			var tree, err = parser.ParseSource([]byte("<script>"+q+"{{"+expr.src+"}}"+q+"</script>"), ast.ContextHTML)
			if err != nil {
				t.Errorf("source: %q, %s\n", expr.src, err)
				continue
			}
			var b = &bytes.Buffer{}
			err = RenderTree(b, tree, expr.globals, false)
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
	src     string
	res     string
	globals scope
}{
	{`""`, `""`, nil},
	{`"a"`, `"a"`, nil},
	{`"<a>"`, `"\3c a\3e "`, nil},
	{`html("<a>")`, `"\3c a\3e "`, nil},
	{`5`, `5`, nil},
	{`5.2`, `5.2`, nil},
	{`a`, `AAECAwQF`, scope{"a": Bytes{0, 1, 2, 3, 4, 5}}},
	{`a`, `AAECAwQF`, scope{"a": []byte{0, 1, 2, 3, 4, 5}}},
}

func TestCSSContext(t *testing.T) {
	for _, expr := range cssContextTests {
		var tree, err = parser.ParseSource([]byte("<style>{{"+expr.src+"}}</style>"), ast.ContextHTML)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = RenderTree(b, tree, expr.globals, false)
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
	src     string
	res     string
	globals scope
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
	{`a`, `a`, scope{"a": "a"}},
	{`a`, `\3c\3e\22 `, scope{"a": "<>\""}},
	{`a`, `AAECAwQF`, scope{"a": Bytes{0, 1, 2, 3, 4, 5}}},
	{`a`, `AAECAwQF`, scope{"a": []byte{0, 1, 2, 3, 4, 5}}},
}

func TestCSSStringContext(t *testing.T) {
	for _, q := range []string{"\"", "'"} {
		for _, expr := range cssStringContextTests {
			var tree, err = parser.ParseSource([]byte("<style>"+q+"{{"+expr.src+"}}"+q+"</style>"), ast.ContextHTML)
			if err != nil {
				t.Errorf("source: %q, %s\n", expr.src, err)
				continue
			}
			var b = &bytes.Buffer{}
			err = RenderTree(b, tree, expr.globals, false)
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
