//
// Copyright (c) 2017 Open2b Software Snc. All Rights Reserved.
//

package exec

import (
	"bytes"
	"html"
	"testing"

	"open2b/template/parser"
)

var htmlContextTests = []struct {
	src  interface{}
	res  string
	vars map[string]interface{}
}{
	{`""`, "", nil},
	{`"a"`, "a", nil},
	{`"<a>"`, html.EscapeString("<a>"), nil},
	{`"<div></div>"`, html.EscapeString("<div></div>"), nil},
	{`a`, html.EscapeString("<a>"), map[string]interface{}{"a": "<a>"}},
	{`d`, html.EscapeString("<div></div>"), map[string]interface{}{"d": "<div></div>"}},
	{`a`, "<a>", map[string]interface{}{"a": HTML("<a>")}},
	{`d`, "<div></div>", map[string]interface{}{"d": HTML("<div></div>")}},
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
	{`a`, "0, 1, 2, 3, 4, 5", map[string]interface{}{"a": []int{0, 1, 2, 3, 4, 5}}},
	{`a`, "-2, -1, 0, 1, 2", map[string]interface{}{"a": []int{-2, -1, 0, 1, 2}}},
	{`a`, "true, false, true", map[string]interface{}{"a": []bool{true, false, true}}},
}

func TestHTMLContext(t *testing.T) {
	for _, expr := range htmlContextTests {
		var src string
		switch s := expr.src.(type) {
		case string:
			src = s
		case HTMLer:
			src = s.HTML()
		}
		var tree, err = parser.Parse([]byte("{{" + src + "}}"))
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		var env = NewEnv(tree, nil)
		err = env.Execute(b, expr.vars)
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
	vars map[string]interface{}
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
	{`a`, "[0,1,2,3,4,5]", map[string]interface{}{"a": []int{0, 1, 2, 3, 4, 5}}},
	{`a`, "[-2,-1,0,1,2]", map[string]interface{}{"a": []int{-2, -1, 0, 1, 2}}},
	{`a`, "[true,false,true]", map[string]interface{}{"a": []bool{true, false, true}}},
}

func TestJavaScriptContext(t *testing.T) {
	for _, expr := range javaScriptContextTests {
		var tree, err = parser.Parse([]byte("<script>{{" + expr.src + "}}</script>"))
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		var env = NewEnv(tree, nil)
		err = env.Execute(b, expr.vars)
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
