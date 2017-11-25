//
// Copyright (c) 2017 Open2b Software Snc. All Rights Reserved.
//

package exec

import (
	"bytes"
	"reflect"
	"testing"

	"open2b/template/parser"
)

var execExprTests = []struct {
	src  string
	res  string
	vars scope
}{
	{`"a"`, "a", nil},
	{"`a`", "a", nil},
	{"3", "3", nil},
	{`"3"`, "3", nil},
	{"-3", "-3", nil},
	{"3.56", "3.56", nil},
	{"3.560", "3.56", nil},
	{"3.50", "3.5", nil},
	{"-3.50", "-3.5", nil},
	{"3.0", "3", nil},
	{"0.0", "0", nil},
	{"-0.0", "0", nil},
	{"true", "true", nil},
	{"false", "false", nil},
	{"_", "5", scope{"_": "5"}},
	{"_", "5", scope{"_": 5}},
	{"true", "_true_", scope{"true": "_true_"}},
	{"false", "_false_", scope{"false": "_false_"}},
	{"2 - 3", "-1", nil},
	{"2 * 3", "6", nil},
	{"2 / 3", "0.6666666666666666666666666667", nil},
	{"7 % 3", "1", nil},
	{"7.2 % 3.7", "3.5", nil},
	{"7 % 3.7", "3.3", nil},
	{"7.2 % 3", "1.2", nil},
	{"-2147483648 * -1", "2147483648", nil},                                       // math.MinInt32 * -1
	{"-2147483649 * -1", "2147483649", nil},                                       // (math.MinInt32-1) * -1
	{"2147483647 * -1", "-2147483647", nil},                                       // math.MaxInt32 * -1
	{"2147483648 * -1", "-2147483648", nil},                                       // (math.MaxInt32+1) * -1
	{"-9223372036854775808 * -1", "9223372036854775808", nil},                     // math.MinInt64 * -1
	{"-9223372036854775809 * -1", "9223372036854775809", nil},                     // (math.MinInt64-1) * -1
	{"9223372036854775807 * -1", "-9223372036854775807", nil},                     // math.MaxInt64 * -1
	{"9223372036854775808 * -1", "-9223372036854775808", nil},                     // (math.MaxInt64+1) * -1
	{"-2147483648 / -1", "2147483648", nil},                                       // math.MinInt32 / -1
	{"-2147483649 / -1", "2147483649", nil},                                       // (math.MinInt32-1) / -1
	{"2147483647 / -1", "-2147483647", nil},                                       // math.MaxInt32 / -1
	{"2147483648 / -1", "-2147483648", nil},                                       // (math.MaxInt32+1) / -1
	{"-9223372036854775808 / -1", "9223372036854775808", nil},                     // math.MinInt64 / -1
	{"-9223372036854775809 / -1", "9223372036854775809", nil},                     // (math.MinInt64-1) / -1
	{"9223372036854775807 / -1", "-9223372036854775807", nil},                     // math.MaxInt64 / -1
	{"9223372036854775808 / -1", "-9223372036854775808", nil},                     // (math.MaxInt64+1) / -1
	{"2147483647 + 2147483647", "4294967294", nil},                                // math.MaxInt32 + math.MaxInt32
	{"-2147483648 + -2147483648", "-4294967296", nil},                             // math.MinInt32 + math.MinInt32
	{"9223372036854775807 + 9223372036854775807", "18446744073709551614", nil},    // math.MaxInt64 + math.MaxInt64
	{"-9223372036854775808 + -9223372036854775808", "-18446744073709551616", nil}, // math.MinInt64 + math.MinInt64
	{"-1 + -2 * 6 / ( 6 - 1 - ( 5 * 3 ) + 2 ) * ( 1 + 2 ) * 3", "12.5", nil},
	{"433937734937734969526500969526500", "433937734937734969526500969526500", nil},
	{"a[1]", "y", scope{"a": []string{"x", "y", "z"}}},
	{"a[:]", "x, y, z", scope{"a": []string{"x", "y", "z"}}},
	{"a[1:]", "y, z", scope{"a": []string{"x", "y", "z"}}},
	{"a[:2]", "x, y", scope{"a": []string{"x", "y", "z"}}},
	{"a[1:2]", "y", scope{"a": []string{"x", "y", "z"}}},
	{"a[1:3]", "y, z", scope{"a": []string{"x", "y", "z"}}},
	{"a[0:3]", "x, y, z", scope{"a": []string{"x", "y", "z"}}},
	{"a[-1:1]", "x", scope{"a": []string{"x", "y", "z"}}},
	{"a[1:10]", "y, z", scope{"a": []string{"x", "y", "z"}}},
	{"a[2:2]", "", scope{"a": []string{"x", "y", "z"}}},
	{"a[2:1]", "", scope{"a": []string{"x", "y", "z"}}},
	{"a[:]", "x€z", scope{"a": "x€z"}},
	{"a[1:]", "€z", scope{"a": "x€z"}},
	{"a[:2]", "x€", scope{"a": "x€z"}},
	{"a[1:2]", "€", scope{"a": "x€z"}},
	{"a[1:3]", "€z", scope{"a": "x€z"}},
	{"a[0:3]", "x€z", scope{"a": "x€z"}},
	{"a[1:]", "xz", scope{"a": "€xz"}},
	{"a[:2]", "xz", scope{"a": "xz€"}},
	{"a[-1:1]", "x", scope{"a": "xz€"}},
	{"a[1:10]", "z€", scope{"a": "xz€"}},
	{"a[2:2]", "", scope{"a": "xz€"}},
	{"a[2:1]", "", scope{"a": "xz€"}},

	// selectors
	{"a.b", "b", scope{"a": map[string]interface{}{"b": "b"}}},
	{"a.B", "b", scope{"a": map[string]interface{}{"B": "b"}}},
	{"a.B", "b", scope{"a": struct{ B string }{B: "b"}}},
	{"a.b", "b", scope{"a": struct {
		B string `template:"b"`
	}{B: "b"}}},
	{"a.b", "b", scope{"a": struct {
		C string `template:"b"`
	}{C: "b"}}},

	// ==, !=
	{"true == true", "true", nil},
	{"false == false", "true", nil},
	{"true == false", "false", nil},
	{"false == true", "false", nil},
	{"true != true", "false", nil},
	{"false != false", "false", nil},
	{"true != false", "true", nil},
	{"false != true", "true", nil},
	{"a == nil", "true", scope{"a": scope(nil)}},
	{"a != nil", "false", scope{"a": scope(nil)}},
	{"nil == a", "true", scope{"a": scope(nil)}},
	{"nil != a", "false", scope{"a": scope(nil)}},
	{`a == "a"`, "true", scope{"a": "a"}},
	{`a == "a"`, "true", scope{"a": HTML("a")}},
	{`a != "b"`, "true", scope{"a": "a"}},
	{`a != "b"`, "true", scope{"a": HTML("a")}},
	{`a == "<a>"`, "true", scope{"a": "<a>"}},
	{`a == "<a>"`, "true", scope{"a": HTML("<a>")}},
	{`a != "<b>"`, "false", scope{"a": "<b>"}},
	{`a != "<b>"`, "false", scope{"a": HTML("<b>")}},

	// &&
	{"true && true", "true", nil},
	{"true && false", "false", nil},
	{"false && true", "false", nil},
	{"false && false", "false", nil},
	{"false && 0/a == 0", "false", scope{"a": 0}},

	// ||
	{"true || true", "true", nil},
	{"true || false", "true", nil},
	{"false || true", "true", nil},
	{"false || false", "false", nil},
	{"true || 0/a == 0", "true", scope{"a": 0}},

	// +
	{"2 + 3", "5", nil},
	{`"a" + "b"`, "ab", nil},
	{`a + "b"`, "ab", scope{"a": "a"}},
	{`a + "b"`, "ab", scope{"a": HTML("a")}},
	{`a + "b"`, "&lt;a&gt;b", scope{"a": "<a>"}},
	{`a + "b"`, "<a>b", scope{"a": HTML("<a>")}},
	{`a + "<b>"`, "&lt;a&gt;&lt;b&gt;", scope{"a": "<a>"}},
	{`a + "<b>"`, "<a>&lt;b&gt;", scope{"a": HTML("<a>")}},
	{"a + b", "&lt;a&gt;&lt;b&gt;", scope{"a": "<a>", "b": "<b>"}},
	{"a + b", "<a><b>", scope{"a": HTML("<a>"), "b": HTML("<b>")}},

	// len
	{"len()", "0", nil},
	{"len(``)", "0", nil},
	{"len(`a`)", "1", nil},
	{"len(`abc`)", "3", nil},
	{"len(`€`)", "1", nil},
	{"len(`€`)", "1", nil},
	{"len(a)", "1", scope{"a": "a"}},
	{"len(a)", "3", scope{"a": "<a>"}},
	{"len(a)", "3", scope{"a": HTML("<a>")}},
}

var execStmtTests = []struct {
	src  string
	res  string
	vars scope
}{
	{"{% for p in products %}{{ p }}\n{% end %}", "a\nb\nc\n",
		scope{"products": []string{"a", "b", "c"}}},
	{"{% for i, p in products %}{{ i }}: {{ p }}\n{% end %}", "0: a\n1: b\n2: c\n",
		scope{"products": []string{"a", "b", "c"}}},
	{"{% for p in products %}a{% break %}b\n{% end %}", "a",
		scope{"products": []string{"a", "b", "c"}}},
	{"{% for p in products %}a{% continue %}b\n{% end %}", "aaa",
		scope{"products": []string{"a", "b", "c"}}},
	{"{% for i in 1..5 %}{{ i }}{% end %}", "12345", nil},
	{"{% for i in 5..1 %}{{ i }}{% end %}", "54321", nil},
	{"{% for i in 0..0 %}{{ i }}{% end %}", "0", nil},
	{"{% for i in -10..-5 %}{{ i }}{% end %}", "-10-9-8-7-6-5", nil},
	{"{% for c in \"\" %}{{ c }}{% end %}", "", nil},
	{"{% for c in \"a\" %}({{ c }}){% end %}", "(a)", nil},
	{"{% for c in \"aÈc\" %}({{ c }}){% end %}", "(a)(È)(c)", nil},
	{"{# comment #}", "", nil},
	{"a{# comment #}b", "ab", nil},
}

var execVarsToScope = []struct {
	vars interface{}
	res  scope
}{
	{
		nil,
		scope{},
	},
	{
		scope{"a": 1, "b": "s"},
		scope{"a": 1, "b": "s"},
	},
	{
		reflect.ValueOf(map[string]interface{}{"a": 1, "b": "s"}),
		scope{"a": 1, "b": "s"},
	},
	{
		map[string]interface{}{"a": 1, "b": "s"},
		scope{"a": 1, "b": "s"},
	},
	{
		map[string]string{"a": "t", "b": "s"},
		scope{"a": "t", "b": "s"},
	},
	{
		map[string]int{"a": 1, "b": 2},
		scope{"a": 1, "b": 2},
	},
	{
		reflect.ValueOf(map[string]interface{}{"a": 1, "b": "s"}),
		scope{"a": 1, "b": "s"},
	},
	{
		struct {
			A int    `template:"a"`
			B string `template:"b"`
			C bool
		}{A: 1, B: "s", C: true},
		scope{"a": 1, "b": "s", "C": true},
	},
	{
		&struct {
			A int    `template:"a"`
			B string `template:"b"`
			C bool
		}{A: 1, B: "s", C: true},
		scope{"a": 1, "b": "s", "C": true},
	},
	{
		reflect.ValueOf(struct {
			A int    `template:"a"`
			B string `template:"b"`
			C bool
		}{A: 1, B: "s", C: true}),
		scope{"a": 1, "b": "s", "C": true},
	},
}

func TestExecExpressions(t *testing.T) {
	for _, expr := range execExprTests {
		var tree, err = parser.Parse([]byte("{{" + expr.src + "}}"))
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		var env = NewEnv(tree, "")
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

func TestExecStatements(t *testing.T) {
	for _, stmt := range execStmtTests {
		var tree, err = parser.Parse([]byte(stmt.src))
		if err != nil {
			t.Errorf("source: %q, %s\n", stmt.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		var env = NewEnv(tree, "")
		err = env.Execute(b, stmt.vars)
		if err != nil {
			t.Errorf("source: %q, %s\n", stmt.src, err)
			continue
		}
		var res = b.String()
		if res != stmt.res {
			t.Errorf("source: %q, unexpected %q, expecting %q\n", stmt.src, res, stmt.res)
		}
	}
}

func TestVarsToScope(t *testing.T) {
	for _, p := range execVarsToScope {
		res, err := varsToScope(p.vars, "")
		if err != nil {
			t.Errorf("vars: %#v, %q\n", p.vars, err)
			continue
		}
		if !reflect.DeepEqual(res, p.res) {
			t.Errorf("vars: %#v, unexpected %q, expecting %q\n", p.vars, res, p.res)
		}
	}
}
