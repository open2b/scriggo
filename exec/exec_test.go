//
// Copyright (c) 2017 Open2b Software Snc. All Rights Reserved.
//

package exec

import (
	"bytes"
	"testing"

	"open2b/template/parser"
)

var execExprTests = []struct {
	src  string
	res  string
	vars map[string]interface{}
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
	{"_", "5", map[string]interface{}{"_": "5"}},
	{"_", "5", map[string]interface{}{"_": 5}},
	{"true", "_true_", map[string]interface{}{"true": "_true_"}},
	{"false", "_false_", map[string]interface{}{"false": "_false_"}},
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
	{"a[1]", "y", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[:]", "x, y, z", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[1:]", "y, z", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[:2]", "x, y", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[1:2]", "y", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[1:3]", "y, z", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[0:3]", "x, y, z", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[-1:1]", "x", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[1:10]", "y, z", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[2:2]", "", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[2:1]", "", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[:]", "x€z", map[string]interface{}{"a": "x€z"}},
	{"a[1:]", "€z", map[string]interface{}{"a": "x€z"}},
	{"a[:2]", "x€", map[string]interface{}{"a": "x€z"}},
	{"a[1:2]", "€", map[string]interface{}{"a": "x€z"}},
	{"a[1:3]", "€z", map[string]interface{}{"a": "x€z"}},
	{"a[0:3]", "x€z", map[string]interface{}{"a": "x€z"}},
	{"a[1:]", "xz", map[string]interface{}{"a": "€xz"}},
	{"a[:2]", "xz", map[string]interface{}{"a": "xz€"}},
	{"a[-1:1]", "x", map[string]interface{}{"a": "xz€"}},
	{"a[1:10]", "z€", map[string]interface{}{"a": "xz€"}},
	{"a[2:2]", "", map[string]interface{}{"a": "xz€"}},
	{"a[2:1]", "", map[string]interface{}{"a": "xz€"}},

	// selectors
	{"a.B", "b", map[string]interface{}{"a": map[string]interface{}{"B": "b"}}},
	{"a.B", "b", map[string]interface{}{"a": struct{ B string }{B: "b"}}},

	// ==, !=
	{"true == true", "true", nil},
	{"false == false", "true", nil},
	{"true == false", "false", nil},
	{"false == true", "false", nil},
	{"true != true", "false", nil},
	{"false != false", "false", nil},
	{"true != false", "true", nil},
	{"false != true", "true", nil},
	{"a == nil", "true", map[string]interface{}{"a": map[string]interface{}(nil)}},
	{"a != nil", "false", map[string]interface{}{"a": map[string]interface{}(nil)}},
	{"nil == a", "true", map[string]interface{}{"a": map[string]interface{}(nil)}},
	{"nil != a", "false", map[string]interface{}{"a": map[string]interface{}(nil)}},
	{`a == "a"`, "true", map[string]interface{}{"a": "a"}},
	{`a == "a"`, "true", map[string]interface{}{"a": HTML("a")}},
	{`a != "b"`, "true", map[string]interface{}{"a": "a"}},
	{`a != "b"`, "true", map[string]interface{}{"a": HTML("a")}},
	{`a == "<a>"`, "true", map[string]interface{}{"a": "<a>"}},
	{`a == "<a>"`, "true", map[string]interface{}{"a": HTML("<a>")}},
	{`a != "<b>"`, "false", map[string]interface{}{"a": "<b>"}},
	{`a != "<b>"`, "false", map[string]interface{}{"a": HTML("<b>")}},

	// +
	{"2 + 3", "5", nil},
	{`"a" + "b"`, "ab", nil},
	{`a + "b"`, "ab", map[string]interface{}{"a": "a"}},
	{`a + "b"`, "ab", map[string]interface{}{"a": HTML("a")}},
	{`a + "b"`, "&lt;a&gt;b", map[string]interface{}{"a": "<a>"}},
	{`a + "b"`, "<a>b", map[string]interface{}{"a": HTML("<a>")}},
	{`a + "<b>"`, "&lt;a&gt;&lt;b&gt;", map[string]interface{}{"a": "<a>"}},
	{`a + "<b>"`, "<a>&lt;b&gt;", map[string]interface{}{"a": HTML("<a>")}},
	{"a + b", "&lt;a&gt;&lt;b&gt;", map[string]interface{}{"a": "<a>", "b": "<b>"}},
	{"a + b", "<a><b>", map[string]interface{}{"a": HTML("<a>"), "b": HTML("<b>")}},

	// len
	{"len()", "0", nil},
	{"len(``)", "0", nil},
	{"len(`a`)", "1", nil},
	{"len(`abc`)", "3", nil},
	{"len(`€`)", "1", nil},
	{"len(`€`)", "1", nil},
	{"len(a)", "1", map[string]interface{}{"a": "a"}},
	{"len(a)", "3", map[string]interface{}{"a": "<a>"}},
	{"len(a)", "3", map[string]interface{}{"a": HTML("<a>")}},
}

var execStmtTests = []struct {
	src  string
	res  string
	vars map[string]interface{}
}{
	{"{% for p in products %}{{ p }}\n{% end %}", "a\nb\nc\n",
		map[string]interface{}{"products": []string{"a", "b", "c"}}},
	{"{% for i, p in products %}{{ i }}: {{ p }}\n{% end %}", "0: a\n1: b\n2: c\n",
		map[string]interface{}{"products": []string{"a", "b", "c"}}},
	{"{% for p in products %}a{% break %}b\n{% end %}", "a",
		map[string]interface{}{"products": []string{"a", "b", "c"}}},
	{"{% for p in products %}a{% continue %}b\n{% end %}", "aaa",
		map[string]interface{}{"products": []string{"a", "b", "c"}}},
	{"{# comment #}", "", nil},
	{"a{# comment #}b", "ab", nil},
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
