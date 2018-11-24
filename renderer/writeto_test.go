//
// Copyright (c) 2017-2018 Open2b Software Snc. All Rights Reserved.
//

package renderer

import (
	"bytes"
	"html"
	"testing"

	"open2b/template/ast"
	"open2b/template/parser"
)

var htmlContextTests = []struct {
	src  interface{}
	res  string
	vars scope
}{
	{`nil`, "", nil},
	{`""`, "", nil},
	{`"a"`, "a", nil},
	{`"<a>"`, html.EscapeString("<a>"), nil},
	{`"<div></div>"`, html.EscapeString("<div></div>"), nil},
	{`a`, html.EscapeString("<a>"), scope{"a": "<a>"}},
	{`d`, html.EscapeString("<div></div>"), scope{"d": "<div></div>"}},
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
	{`-0.1000000`, "-0.1", nil},
	{`true`, "true", nil},
	{`false`, "false", nil},
	{`a`, "0, 1, 2, 3, 4, 5", scope{"a": []int{0, 1, 2, 3, 4, 5}}},
	{`a`, "-2, -1, 0, 1, 2", scope{"a": []int{-2, -1, 0, 1, 2}}},
	{`a`, "true, false, true", scope{"a": []bool{true, false, true}}},
	{`a`, "t: 1", scope{"a": aNumber{1}}},
	{`a + 0`, "1", scope{"a": aNumber{1}}},
	{`a`, "t: a", scope{"a": aString{"a"}}},
	{`a + ""`, "a", scope{"a": aString{"a"}}},
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
		var tree, err = parser.Parse([]byte("{{"+src+"}}"), ast.ContextHTML)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = Render(b, tree, "", expr.vars, nil)
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

var attrContextTests = []struct {
	src  interface{}
	res  string
	vars scope
}{
	{`nil`, "", nil},
	{`""`, "", nil},
	{`"a"`, "a", nil},
	{`"<a>"`, html.EscapeString("<a>"), nil},
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
	{`a`, "t: 1", scope{"a": aNumber{1}}},
	{`a + 0`, "1", scope{"a": aNumber{1}}},
	{`a`, "t: a", scope{"a": aString{"a"}}},
	{`a + ""`, "a", scope{"a": aString{"a"}}},
}

func TestAttributeContext(t *testing.T) {
	for _, expr := range attrContextTests {
		var src string
		switch s := expr.src.(type) {
		case string:
			src = s
		case HTML:
			src = string(s)
		}
		var tree, err = parser.Parse([]byte("{{"+src+"}}"), ast.ContextAttribute)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = Render(b, tree, "", expr.vars, nil)
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

var urlContextTests = []struct {
	src  interface{}
	res  string
	vars scope
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
		var tree, err = parser.Parse([]byte(src), ast.ContextHTML)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = Render(b, tree, "", expr.vars, nil)
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
	src  string
	res  string
	vars scope
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
	{`a`, "null", scope{"a": []int(nil)}},
	{`a`, "[0,1,2,3,4,5]", scope{"a": []int{0, 1, 2, 3, 4, 5}}},
	{`a`, "[-2,-1,0,1,2]", scope{"a": []int{-2, -1, 0, 1, 2}}},
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
}

func TestJavaScriptContext(t *testing.T) {
	for _, expr := range scriptContextTests {
		var tree, err = parser.Parse([]byte("<script>{{"+expr.src+"}}</script>"), ast.ContextHTML)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = Render(b, tree, "", expr.vars, nil)
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
