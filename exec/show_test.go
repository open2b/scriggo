//
// Copyright (c) 2017-2018 Open2b Software Snc. All Rights Reserved.
//

package exec

import (
	"bytes"
	"html"
	"io"
	"strconv"
	"testing"

	"open2b/template/ast"
	"open2b/template/parser"

	dec "github.com/shopspring/decimal"
)

type T struct {
	v int
}

func (t T) WriteTo(w io.Writer, ctx ast.Context) (int, error) {
	return io.WriteString(w, "t: "+strconv.Itoa(t.v))
}

func (t T) Number() dec.Decimal {
	return dec.New(int64(t.v), 0)
}

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
	{`a`, "t: 1", scope{"a": T{1}}},
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
		err = Execute(b, tree, "", expr.vars)
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
		err = Execute(b, tree, "", expr.vars)
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
