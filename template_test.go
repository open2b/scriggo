// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"open2b/template/ast"
	"open2b/template/parser"

	"github.com/shopspring/decimal"
)

type aNumber struct {
	v int
}

func (n aNumber) Render(w io.Writer) (int, error) {
	return io.WriteString(w, "t: "+strconv.Itoa(n.v))
}

func (n aNumber) Number() decimal.Decimal {
	return decimal.New(int64(n.v), 0)
}

type aString struct {
	v string
}

func (s aString) Render(w io.Writer) (int, error) {
	return io.WriteString(w, "t: "+s.v)
}

func (s aString) String() string {
	return s.v
}

type stringConvertible string

type aMap struct {
	v string
	H func() string `template:"G"`
}

func (s aMap) F() string {
	return s.v
}

func (s aMap) G() string {
	return s.v
}

type aStruct struct {
	a string
	B string `template:"b"`
	C string
}

var largeDecimal, _ = decimal.NewFromString("163095826571306923551828945029")

var rendererExprTests = []struct {
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
	{"true", "_true_", scope{"true": "_true_"}},
	{"false", "_false_", scope{"false": "_false_"}},
	{"2 - 3", "-1", nil},
	{"2 * 3", "6", nil},
	{"2.2 * 3", "6.6", nil},
	{"2 * 3.1", "6.2", nil},
	{"2.0 * 3.1", "6.2", nil},
	{"2 / 3", "0.66666666666666666667", nil},
	{"2.0 / 3", "0.66666666666666666667", nil},
	{"2 / 3.0", "0.66666666666666666667", nil},
	{"2.0 / 3.0", "0.66666666666666666667", nil},
	{"7 % 3", "1", nil},
	{"7.2 % 3.7", "3.5", nil},
	{"7 % 3.7", "3.3", nil},
	{"7.2 % 3", "1.2", nil},
	{"-2147483648 * -1", "2147483648", nil},                   // math.MinInt32 * -1
	{"-2147483649 * -1", "2147483649", nil},                   // (math.MinInt32-1) * -1
	{"2147483647 * -1", "-2147483647", nil},                   // math.MaxInt32 * -1
	{"2147483648 * -1", "-2147483648", nil},                   // (math.MaxInt32+1) * -1
	{"-9223372036854775808 * -1", "9223372036854775808", nil}, // math.MinInt64 * -1
	{"9223372036854775807 * -1", "-9223372036854775807", nil}, // math.MaxInt64 * -1
	{"-2147483648 / -1", "2147483648", nil},                   // math.MinInt32 / -1
	{"-2147483649 / -1", "2147483649", nil},                   // (math.MinInt32-1) / -1
	{"2147483647 / -1", "-2147483647", nil},                   // math.MaxInt32 / -1
	{"2147483648 / -1", "-2147483648", nil},                   // (math.MaxInt32+1) / -1
	{"-9223372036854775808 / -1", "9223372036854775808", nil}, // math.MinInt64 / -1
	{"9223372036854775807 / -1", "-9223372036854775807", nil}, // math.MaxInt64 / -1
	{"2147483647 + 2147483647", "4294967294", nil},            // math.MaxInt32 + math.MaxInt32
	{"-2147483648 + -2147483648", "-4294967296", nil},         // math.MinInt32 + math.MinInt32
	{"9223372036854775807 + 9223372036854775807", "18446744073709551614", nil},    // math.MaxInt64 + math.MaxInt64
	{"-9223372036854775808 + -9223372036854775808", "-18446744073709551616", nil}, // math.MinInt64 + math.MinInt64
	{"-1 + -2 * 6 / ( 6 - 1 - ( 5 * 3 ) + 2 ) * ( 1 + 2 ) * 3", "12.5", nil},
	{"a[1]", "y", scope{"a": []string{"x", "y", "z"}}},
	{"a[:]", "x, y, z", scope{"a": []string{"x", "y", "z"}}},
	{"a[1:]", "y, z", scope{"a": []string{"x", "y", "z"}}},
	{"a[:2]", "x, y", scope{"a": []string{"x", "y", "z"}}},
	{"a[1:2]", "y", scope{"a": []string{"x", "y", "z"}}},
	{"a[1:3]", "y, z", scope{"a": []string{"x", "y", "z"}}},
	{"a[0:3]", "x, y, z", scope{"a": []string{"x", "y", "z"}}},
	{"a[2:2]", "", scope{"a": []string{"x", "y", "z"}}},
	{"a[0]", "x", scope{"a": "x€z"}},
	{"a[1]", "€", scope{"a": "x€z"}},
	{"a[2]", "z", scope{"a": "x€z"}},
	{"a[2.2/1.1]", "z", scope{"a": []string{"x", "y", "z"}}},
	{"a[1]", "b", scope{"a": HTML("<b>")}},
	{"a[0]", "<", scope{"a": HTML("<b>")}},
	{`a[1]`, "b", scope{"a": aString{"abc"}}},
	{"a[1]", "b", scope{"a": stringConvertible("abc")}},
	{"a[:]", "x€z", scope{"a": "x€z"}},
	{"a[1:]", "€z", scope{"a": "x€z"}},
	{"a[:2]", "x€", scope{"a": "x€z"}},
	{"a[1:2]", "€", scope{"a": "x€z"}},
	{"a[1:3]", "€z", scope{"a": "x€z"}},
	{"a[0:3]", "x€z", scope{"a": "x€z"}},
	{"a[1:]", "xz", scope{"a": "€xz"}},
	{"a[:2]", "xz", scope{"a": "xz€"}},
	{"a[2:2]", "", scope{"a": "xz€"}},
	{"a[1:]", "b>", scope{"a": HTML("<b>")}},
	{`a[1:]`, "z€", scope{"a": aString{"xz€"}}},
	{"a[1:]", "z€", scope{"a": stringConvertible("xz€")}},
	{`a.(string)`, "abc", scope{"a": "abc"}},
	{`a.(string)`, "<b>", scope{"a": HTML("<b>")}},
	{`a.(html)`, "<b>", scope{"a": HTML("<b>")}},
	{`a.(number)`, "5.5", scope{"a": 5.5}},
	{`a.(number)`, "5", scope{"a": 5}},
	{`a.(int)`, "5", scope{"a": 5}},
	{`a.(bool)`, "true", scope{"a": true}},
	{`a.(map).B`, "b", scope{"a": &struct{ B string }{B: "b"}}},
	{`a.(slice)`, "1, 2, 3", scope{"a": []int{1, 2, 3}}},
	{`a.(error)`, "err", scope{"a": errors.New("err")}},

	// slice
	{"slice{}", "", nil},
	{"len(slice{})", "0", nil},
	{"slice{v}", "", map[string]interface{}{"v": []string(nil)}},
	{"len(slice{v})", "1", map[string]interface{}{"v": []string(nil)}},
	{"slice{v, v2}", ", ", map[string]interface{}{"v": []string(nil), "v2": []string(nil)}},
	{"slice{`a`}", "a", nil},
	{"slice{`a`, `b`, `c`}", "a, b, c", nil},
	{"slice{html(`<a>`), html(`<b>`), html(`<c>`)}", "<a>, <b>, <c>", nil},
	{"slice{4, 9, 3}", "4, 9, 3", nil},
	{"slice{4.2, 9.06, 3.7}", "4.2, 9.06, 3.7", nil},
	{"slice{false, false, true}", "false, false, true", nil},
	{"slice{`a`, 8, true, html(`<b>`)}", "a, 8, true, <b>", nil},
	{`slice{"a",2,3.6,html("<b>")}`, "a, 2, 3.6, <b>", nil},
	{`slice{slice{1,2},"/",slice{3,4}}`, "1, 2, /, 3, 4", nil},

	// map
	{"len(map{})", "0", nil},
	{"map{`a`:5}.a", "5", nil},
	{"map{`a`:5,``+`a`:7}.a", "7", nil},

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
	{"a == nil", "false", scope{"a": "b"}},
	{"a == nil", "false", scope{"a": 5}},
	{"5 == 5", "true", nil},
	{`a == "a"`, "true", scope{"a": "a"}},
	{`a == "a"`, "true", scope{"a": HTML("a")}},
	{`a != "b"`, "true", scope{"a": "a"}},
	{`a != "b"`, "true", scope{"a": HTML("a")}},
	{`a == "<a>"`, "true", scope{"a": "<a>"}},
	{`a == "<a>"`, "true", scope{"a": HTML("<a>")}},
	{`a != "<b>"`, "false", scope{"a": "<b>"}},
	{`a != "<b>"`, "false", scope{"a": HTML("<b>")}},
	{"map(nil) == nil", "true", nil},
	{"map(nil) == 5", "false", nil},
	{"map(nil) == map(nil)", "false", nil},
	{"map{} == nil", "false", nil},
	{"map{} == map{}", "false", nil},
	{"slice(nil) == nil", "true", nil},
	{"slice(nil) == 5", "false", nil},
	{"slice(nil) == slice(nil)", "false", nil},
	{"slice{} == nil", "false", nil},
	{"slice{} == slice{}", "false", nil},

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
	{`a + "b"`, "<a>b", scope{"a": "<a>"}},
	{`a + "b"`, "<a>b", scope{"a": HTML("<a>")}},
	{`a + "<b>"`, "<a><b>", scope{"a": "<a>"}},
	{`a + "<b>"`, "<a>&lt;b&gt;", scope{"a": HTML("<a>")}},
	{"a + b", "<a><b>", scope{"a": "<a>", "b": "<b>"}},
	{"a + b", "<a><b>", scope{"a": HTML("<a>"), "b": HTML("<b>")}},

	// call
	{"f()", "ok", scope{"f": func() string { return "ok" }}},
	{"f(5)", "5", scope{"f": func(i int) int { return i }}},
	{"f(5.4)", "5.4", scope{"f": func(n decimal.Decimal) decimal.Decimal { return n }}},
	{"f(5)", "5", scope{"f": func(n decimal.Decimal) decimal.Decimal { return n }}},
	{"f(`a`)", "a", scope{"f": func(s string) string { return s }}},
	{"f(html(`<a>`))", "<a>", scope{"f": func(s string) string { return s }}},
	{"f(true)", "true", scope{"f": func(t bool) bool { return t }}},
	{"f(5)", "5", scope{"f": func(v interface{}) interface{} { return v }}},
	{"f(`a`)", "a", scope{"f": func(v interface{}) interface{} { return v }}},
	{"f(html(`<a>`))", "<a>", scope{"f": func(s string) string { return s }}},
	{"f(true)", "true", scope{"f": func(v interface{}) interface{} { return v }}},
	{"f(nil)", "", scope{"f": func(v interface{}) interface{} { return v }}},
	{"f()", "", scope{"f": func(s ...string) string { return strings.Join(s, ",") }}},
	{"f(`a`)", "a", scope{"f": func(s ...string) string { return strings.Join(s, ",") }}},
	{"f(`a`, `b`)", "a,b", scope{"f": func(s ...string) string { return strings.Join(s, ",") }}},
	{"f(5)", "5 ", scope{"f": func(i int, s ...string) string { return strconv.Itoa(i) + " " + strings.Join(s, ",") }}},
	{"f(5, `a`, `b`)", "5 a,b", scope{"f": func(i int, s ...string) string { return strconv.Itoa(i) + " " + strings.Join(s, ",") }}},
	{"s.F()", "a", scope{"s": aMap{v: "a"}}},
	{"s.G()", "b", scope{"s": aMap{v: "a", H: func() string { return "b" }}}},

	// number types
	{"1+a", "3", scope{"a": int(2)}},
	{"1+a", "3", scope{"a": int8(2)}},
	{"1+a", "3", scope{"a": int16(2)}},
	{"1+a", "3", scope{"a": int32(2)}},
	{"1+a", "3", scope{"a": int64(2)}},
	{"1+a", "3", scope{"a": uint8(2)}},
	{"1+a", "3", scope{"a": uint16(2)}},
	{"1+a", "3", scope{"a": uint32(2)}},
	{"1+a", "3", scope{"a": uint64(2)}},
	{"1+a", "3.5", scope{"a": float32(2.5)}},
	{"1+a", "3.5", scope{"a": float64(2.5)}},
	{"f(a)", "3", scope{"f": func(n int) int { return n + 1 }, "a": int(2)}},
	{"f(a)", "3", scope{"f": func(n int) int { return n + 1 }, "a": int8(2)}},
	{"f(a)", "3", scope{"f": func(n int) int { return n + 1 }, "a": int16(2)}},
	{"f(a)", "3", scope{"f": func(n int) int { return n + 1 }, "a": int32(2)}},
	{"f(a)", "3", scope{"f": func(n int) int { return n + 1 }, "a": int64(2)}},
	{"f(a)", "3", scope{"f": func(n int) int { return n + 1 }, "a": byte(2)}},
	{"f(a)", "3", scope{"f": func(n int) int { return n + 1 }, "a": uint16(2)}},
	{"f(a)", "3", scope{"f": func(n int) int { return n + 1 }, "a": uint32(2)}},
	{"f(a)", "3", scope{"f": func(n int) int { return n + 1 }, "a": uint64(2)}},
	{"f(a)", "3", scope{"f": func(n int) int { return n + 1 }, "a": float32(2.0)}},
	{"f(a)", "3", scope{"f": func(n int) int { return n + 1 }, "a": float64(2.0)}},
}

var rendererStmtTests = []struct {
	src  string
	res  string
	vars scope
}{
	{"{% if true %}ok{% else %}no{% end %}", "ok", nil},
	{"{% if false %}no{% else %}ok{% end %}", "ok", nil},
	{"{% if a := true; a %}ok{% else %}no{% end %}", "ok", nil},
	{"{% if a := false; a %}no{% else %}ok{% end %}", "ok", nil},
	{"{% a := false %}{% if a = true; a %}ok{% else %}no{% end %}", "ok", nil},
	{"{% a := true %}{% if a = false; a %}no{% else %}ok{% end %}", "ok", nil},
	{"{% if a, ok := b; ok %}ok{% else %}no{% end %}", "ok", scope{"b": nil}},
	{"{% if a, ok := b; ok %}no{% else %}ok{% end %}", "ok", scope{}},
	{"{% if a, ok := b.c; ok %}ok{% else %}no{% end %}", "ok", scope{"b": map[string]interface{}{"c": true}}},
	{"{% if a, ok := b.d; ok %}no{% else %}ok{% end %}", "ok", scope{"b": map[string]interface{}{}}},
	{"{% if a, ok := b.c; a %}ok{% else %}no{% end %}", "ok", scope{"b": map[string]interface{}{"c": true}}},
	{"{% if a, ok := b.d; a %}no{% else %}ok{% end %}", "ok", scope{"b": map[string]interface{}{"d": false}}},
	{"{% if a, ok := b.c.d; a %}ok{% else %}no{% end %}", "ok", scope{"b": map[string]interface{}{"c": map[string]interface{}{"d": true}}}},
	{"{% if a, ok := b.c.d; a %}no{% else %}ok{% end %}", "ok", scope{"b": map[string]interface{}{"c": map[string]interface{}{"d": false}}}},
	{"{% if a, ok := b.c.d; ok %}no{% else %}ok{% end %}", "ok", scope{"b": map[string]interface{}{"c": map[string]interface{}{}}}},
	{"{% if a, ok := b[`c`]; ok %}ok{% else %}no{% end %}", "ok", scope{"b": map[string]interface{}{"c": true}}},
	{"{% if a, ok := b[`d`]; ok %}no{% else %}ok{% end %}", "ok", scope{"b": map[string]interface{}{}}},
	{"{% if a, ok := b[`c`]; a %}ok{% else %}no{% end %}", "ok", scope{"b": map[string]interface{}{"c": true}}},
	{"{% if a, ok := b[`d`]; a %}no{% else %}ok{% end %}", "ok", scope{"b": map[string]interface{}{"d": false}}},
	{"{% if a, ok := b[`c`].d; a %}ok{% else %}no{% end %}", "ok", scope{"b": map[string]interface{}{"c": map[string]interface{}{"d": true}}}},
	{"{% if a, ok := b[`c`].d; a %}no{% else %}ok{% end %}", "ok", scope{"b": map[string]interface{}{"c": map[string]interface{}{"d": false}}}},
	{"{% if a, ok := b[`c`].d; ok %}no{% else %}ok{% end %}", "ok", scope{"b": map[string]interface{}{"c": map[string]interface{}{}}}},
	{"{% if a, ok := b.c[`d`]; a %}ok{% else %}no{% end %}", "ok", scope{"b": map[string]interface{}{"c": map[string]interface{}{"d": true}}}},
	{"{% if a, ok := b.c[`d`]; a %}no{% else %}ok{% end %}", "ok", scope{"b": map[string]interface{}{"c": map[string]interface{}{"d": false}}}},
	{"{% if a, ok := b.c[`d`]; ok %}no{% else %}ok{% end %}", "ok", scope{"b": map[string]interface{}{"c": map[string]interface{}{}}}},
	{"{% if a, ok := b[`c`][`d`]; ok %}no{% else %}ok{% end %}", "ok", scope{"b": map[string]interface{}{"c": map[string]interface{}{}}}},
	{"{% if a, ok := b.(string); ok %}ok{% else %}no{% end %}", "ok", scope{"b": "abc"}},
	{"{% if a, ok := b.(string); ok %}no{% else %}ok{% end %}", "ok", scope{"b": 5}},
	{"{% if a, ok := b.(int); ok %}ok{% else %}no{% end %}", "ok", scope{"b": 5}},
	{"{% if a, ok := b.c.(int); ok %}ok{% else %}no{% end %}", "ok", scope{"b": map[string]interface{}{"c": 5}}},
	{"{% if a, ok := b.c.(int); ok %}no{% else %}ok{% end %}", "ok", scope{"b": map[string]interface{}{"c": true}}},
	{"{% if a, ok := b[`c`].(int); ok %}ok{% else %}no{% end %}", "ok", scope{"b": map[string]interface{}{"c": 5}}},
	{"{% if a, ok := b.(byte); ok %}ok{% else %}no{% end %}", "ok", scope{"b": 5}},
	{"{% if a, ok := b.c.(byte); ok %}ok{% else %}no{% end %}", "ok", scope{"b": map[string]interface{}{"c": 5}}},
	{"{% if a, ok := b.c.(byte); ok %}no{% else %}ok{% end %}", "ok", scope{"b": map[string]interface{}{"c": true}}},
	{"{% if a, ok := b[`c`].(byte); ok %}ok{% else %}no{% end %}", "ok", scope{"b": map[string]interface{}{"c": 5}}},
	{"{% b := map{html(`<b>`): true} %}{% if a, ok := b[html(`<b>`)]; ok %}ok{% else %}no{% end %}", "ok", nil},
	{"{% b := map{5.2: true} %}{% if a, ok := b[5.2]; ok %}ok{% else %}no{% end %}", "ok", nil},
	{"{% b := map{5: true} %}{% if a, ok := b[5]; ok %}ok{% else %}no{% end %}", "ok", nil},
	{"{% b := map{true: true} %}{% if a, ok := b[true]; ok %}ok{% else %}no{% end %}", "ok", nil},
	{"{% b := map{nil: true} %}{% if a, ok := b[nil]; ok %}ok{% else %}no{% end %}", "ok", nil},
	{"{% b := map{} %}{% if a, ok := b[map{}]; ok %}no{% else %}{{ a == nil }}{% end %}", "true", nil},
	{"{% a := 5 %}{% if true %}{% a = 7 %}{{ a }}{% end %}", "7", nil},
	{"{% a := 5 %}{% if true %}{% a := 7 %}{{ a }}{% end %}", "7", nil},
	{"{% a := 5 %}{% if true %}{% a := 7 %}{% a = 9 %}{{ a }}{% end %}", "9", nil},
	{"{% a := 5 %}{% if true %}{% a := 7 %}{% a, b := test2(1,2) %}{{ a }}{% end %}", "1", nil},
	{"{% a := 5 %}{% if true %}{% a, b := test2(7,8) %}{% a, b = test2(1,2) %}{{ a }}{% end %}", "1", nil},
	{"{% _ = 5 %}", "", nil},
	{"{% _, a := test2(4,5) %}{{ a }}", "5", nil},
	{"{% a := 3 %}{% _, a = test2(4,5) %}{{ a }}", "5", nil},
	{"{% a := slice{1,2,3} %}{% a[1] = 5 %}{{ a }}", "1, 5, 3", nil},
	{"{% a := map{`b`:1} %}{% a[`b`] = 5 %}{{ a[`b`] }}", "5", nil},
	{"{% a := map{`b`:1} %}{% a.b = 5 %}{{ a.b }}", "5", nil},
	{"{% a := 0 %}{% a, a = test2(1,2) %}{{ a }}", "2", nil},
	{"{% a := map{} %}{% a.b, a[`c`] = test2(1,2) %}{{ a.b }},{{ a.c }}", "1,2", nil},
	{"{% a, b := test2(1,2) %}{{ a }},{{ b }}", "1,2", nil},
	{"{% a := 0 %}{% a, b := test2(1,2) %}{{ a }},{{ b }}", "1,2", nil},
	{"{% b := 0 %}{% a, b := test2(1,2) %}{{ a }},{{ b }}", "1,2", nil},
	{"{% s := slice{1,2,3} %}{% s[0] = 5 %}{{ s[0] }}", "5", nil},
	{"{% s := slice{1,2,3} %}{% s2 := s[0:2] %}{% s2[0] = 5 %}{{ s2 }}", "5, 2", nil},
	{"{% for i, p := range products %}{{ i }}: {{ p }}\n{% end %}", "0: a\n1: b\n2: c\n",
		scope{"products": []string{"a", "b", "c"}}},
	{"{% for _, p := range products %}{{ p }}\n{% end %}", "a\nb\nc\n",
		scope{"products": []string{"a", "b", "c"}}},
	{"{% for _, p := range products %}a{% break %}b\n{% end %}", "a",
		scope{"products": []string{"a", "b", "c"}}},
	{"{% for _, p := range products %}a{% continue %}b\n{% end %}", "aaa",
		scope{"products": []string{"a", "b", "c"}}},
	{"{% for _, c := range \"\" %}{{ c }}{% end %}", "", nil},
	{"{% for _, c := range \"a\" %}({{ c }}){% end %}", "(a)", nil},
	{"{% for _, c := range \"aÈc\" %}({{ c }}){% end %}", "(a)(È)(c)", nil},
	{"{% for _, c := range html(\"<b>\") %}({{ c }}){% end %}", "(<)(b)(>)", nil},
	{"{% for _, i := range slice{ `a`, `b`, `c` } %}{{ i }}{% end %}", "abc", nil},
	{"{% for _, i := range slice{ html(`<`), html(`&`), html(`>`) } %}{{ i }}{% end %}", "<&>", nil},
	{"{% for _, i := range slice{1, 2, 3, 4, 5} %}{{ i }}{% end %}", "12345", nil},
	{"{% for _, i := range slice{1.3, 5.8, 2.5} %}{{ i }}{% end %}", "1.35.82.5", nil},
	{"{% for _, i := range bytes{ 0, 1, 2 } %}{{ i }}{% end %}", "012", nil},
	{"{% s := slice{} %}{% for k, v := range map{`a`: `1`, `b`: `2`} %}{% s = append(s, k+`:`+v) %}{% end %}{{ sort(s) }}", "a:1, b:2", nil},
	{"{% for k, v := range map{} %}{{ k }}:{{ v }},{% end %}", "", nil},
	{"{% s := slice{} %}{% for k, v := range m %}{% s = append(s, itoa(k)+`:`+itoa(v)) %}{% end %}{{ sort(s) }}", "1:1, 2:4, 3:9", scope{"m": map[int]int{1: 1, 2: 4, 3: 9}}},
	{"{% s := slice{} %}{% for k, v := range m %}{% s = append(s, k+`:`+v) %}{% end %}{{ sort(s) }}", "C:3, b:2", scope{"m": aStruct{a: "1", B: "2", C: "3"}}},
	{"{% for p in products %}{{ p }}\n{% end %}", "a\nb\nc\n",
		scope{"products": []string{"a", "b", "c"}}},
	{"{% i := 0 %}{% c := \"\" %}{% for i, c = range \"ab\" %}({{ c }}){% end %}{{ i }}", "(a)(b)1", nil},
	{"{% for range slice{ `a`, `b`, `c` } %}.{% end %}", "...", nil},
	{"{% for range bytes{ 1, 2, 3 } %}.{% end %}", "...", nil},
	{"{% for range slice{} %}.{% end %}", "", nil},
	{"{% for i := 0; i < 5; i++ %}{{ i }}{% end %}", "01234", nil},
	{"{% for i := 0; i < 5; i++ %}{{ i }}{% break %}{% end %}", "0", nil},
	{"{% for i := 0; ; i++ %}{{ i }}{% if i == 4 %}{% break %}{% end %}{% end %}", "01234", nil},
	{"{% for i := 0; i < 5; i++ %}{{ i }}{% if i == 4 %}{% continue %}{% end %},{% end %}", "0,1,2,3,4", nil},
	{"{% i := 0 %}{% c := true %}{% for c %}{% i++ %}{{ i }}{% c = i < 5 %}{% end %}", "12345", nil},
	{"{% i := 0 %}{% for ; ; %}{% i++ %}{{ i }}{% if i == 4 %}{% break %}{% end %},{% end %} {{ i }}", "1,2,3,4 4", nil},
	{"{% i := 5 %}{% i++ %}{{ i }}", "6", nil},
	{"{% i := a %}{% i++ %}{{ i }}", "6", scope{"a": aNumber{5}}},
	{"{% s := map{`a`: 5} %}{% s.a++ %}{{ s.a }}", "6", nil},
	{"{% s := slice{5} %}{% s[0]++ %}{{ s[0] }}", "6", nil},
	{"{% s := bytes{5} %}{% s[0]++ %}{{ s[0] }}", "6", nil},
	{"{% s := bytes{255} %}{% s[0]++ %}{{ s[0] }}", "0", nil},
	{"{% i := 5 %}{% i-- %}{{ i }}", "4", nil},
	{"{% i := a %}{% i++ %}{{ i }}", "6", scope{"a": aNumber{5}}},
	{"{% s := map{`a`: 5} %}{% s.a-- %}{{ s.a }}", "4", nil},
	{"{% s := slice{5} %}{% s[0]-- %}{{ s[0] }}", "4", nil},
	{"{% s := bytes{5} %}{% s[0]-- %}{{ s[0] }}", "4", nil},
	{"{% s := bytes{0} %}{% s[0]-- %}{{ s[0] }}", "255", nil},
	{"{# comment #}", "", nil},
	{"a{# comment #}b", "ab", nil},

	// conversion
	{`{% if s, ok := string("abc").(string); ok %}{{ s }}{% end %}`, "abc", nil},
	{`{% if s, ok := string(html("<b>")).(string); ok %}{{ s }}{% end %}`, "<b>", nil},
	{`{% if s, ok := string(87.5+0.5).(string); ok %}{{ s }}{% end %}`, "X", nil},
	{`{% if s, ok := string(88).(string); ok %}{{ s }}{% end %}`, "X", nil},
	{`{% if s, ok := string(88888888888).(string); ok %}{{ s }}{% end %}`, "\uFFFD", nil},
	{`{% if s, ok := string(slice(nil)).(string); ok %}{{ s }}{% end %}`, "", nil},
	{`{% if s, ok := string(slice{35, 8364}).(string); ok %}{{ s }}{% end %}`, "#€", nil},
	{`{% if s, ok := string(a).(string); ok %}{{ s }}{% end %}`, "#€", scope{"a": []int{35, 8364}}},
	{`{% if s, ok := string(bytes(nil)).(string); ok %}{{ s }}{% end %}`, "", nil},
	{`{% if s, ok := string(bytes{97, 226, 130, 172, 98}).(string); ok %}{{ s }}{% end %}`, "a€b", nil},
	{`{% if s, ok := string(bytes(nil)).(string); ok %}{{ s }}{% end %}`, "", nil},
	{`{% if s, ok := string(bytes{97, 226, 130, 172, 98}).(string); ok %}{{ s }}{% end %}`, "a€b", nil},
	{`{% if s, ok := string(a).(string); ok %}{{ s }}{% end %}`, "a€b", scope{"a": []byte{97, 226, 130, 172, 98}}},
	{`{% if s, ok := html(html("<b>")).(html); ok %}{{ s }}{% end %}`, "<b>", nil},
	{`{% if s, ok := html("<b>").(html); ok %}{{ s }}{% end %}`, "<b>", nil},
	{`{% if s, ok := html(87.5+0.5).(html); ok %}{{ s }}{% end %}`, "X", nil},
	{`{% if s, ok := html(88).(html); ok %}{{ s }}{% end %}`, "X", nil},
	{`{% if s, ok := html(88888888888).(html); ok %}{{ s }}{% end %}`, "\uFFFD", nil},
	{`{% if s, ok := html(slice(nil)).(html); ok %}{{ s }}{% end %}`, "", nil},
	{`{% if s, ok := html(slice{35, 8364}).(html); ok %}{{ s }}{% end %}`, "#€", nil},
	{`{% if s, ok := html(bytes(nil)).(html); ok %}{{ s }}{% end %}`, "", nil},
	{`{% if s, ok := html(bytes{97, 226, 130, 172, 98}).(html); ok %}{{ s }}{% end %}`, "a€b", nil},
	{`{% if s, ok := number(5.5).(number); ok %}{{ s }}{% end %}`, "5.5", nil},
	{`{% if s, ok := number(5).(number); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := int(5.5).(int); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := int(5).(int); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := byte(5).(byte); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := byte(260).(byte); ok %}{{ s }}{% end %}`, "4", nil},
	{`{% if s, ok := byte(a).(byte); ok %}{{ s }}{% end %}`, "133", scope{"a": largeDecimal}},
	{`{% if _, ok := map(nil).(map); ok %}ok{% end %}`, "ok", nil},
	{`{% if map(nil) == nil %}ok{% end %}`, "ok", nil},
	{`{% if _, ok := map(a).(map); ok %}ok{% end %}`, "ok", scope{"a": Map(nil)}},
	{`{% if map(a) == nil %}ok{% end %}`, "ok", scope{"a": Map(nil)}},
	{`{% if _, ok := map(a).(map); ok %}ok{% end %}`, "ok", scope{"a": map[string]int(nil)}},
	{`{% if map(a) == nil %}ok{% end %}`, "ok", scope{"a": map[string]int(nil)}},
	{`{% if map(a) != nil %}ok{% end %}`, "ok", scope{"a": map[string]int{"b": 2}}},
	{`{% m := map(a) %}{% keys := slice{} %}{% for k := range m %}{% keys = append(keys, k) %}{% end %}{{ sort(keys) }}`, "1, 2, 3", scope{"a": []int{1, 3, 2}}},
	{`{% m := map(a) %}{% keys := slice{} %}{% for range m %}.{% end %}`, ".", scope{"a": Slice{nil, nil, nil}}},
	{`{% m := map(a) %}{% if m == nil %}ok{% end %}`, "ok", scope{"a": Slice(nil)}},
	{`{% m := map(a) %}{% if m == nil %}ok{% end %}`, "ok", scope{"a": []int(nil)}},
	{`{% if _, ok := slice(nil).(slice); ok %}ok{% end %}`, "ok", nil},
	{`{% if slice(nil) == nil %}ok{% end %}`, "ok", nil},
	{`{% if _, ok := slice(a).(slice); ok %}ok{% end %}`, "ok", scope{"a": Slice(nil)}},
	{`{% if slice(a) == nil %}ok{% end %}`, "ok", scope{"a": Slice(nil)}},
	{`{% if _, ok := slice(a).(slice); ok %}ok{% end %}`, "ok", scope{"a": []int(nil)}},
	{`{% if slice(a) == nil %}ok{% end %}`, "ok", scope{"a": []int(nil)}},
	{`{% if slice(a) != nil %}ok{% end %}`, "ok", scope{"a": []int{1, 2}}},
	{`{% if _, ok := bytes(nil).(bytes); ok %}ok{% end %}`, "ok", nil},
	{`{% if bytes(nil) == nil %}ok{% end %}`, "ok", nil},
	{`{% if _, ok := bytes(a).(bytes); ok %}ok{% end %}`, "ok", scope{"a": Bytes(nil)}},
	{`{% if bytes(a) == nil %}ok{% end %}`, "ok", scope{"a": Bytes(nil)}},
	{`{% if _, ok := bytes(a).(bytes); ok %}ok{% end %}`, "ok", scope{"a": []byte(nil)}},
	{`{% if bytes(a) == nil %}ok{% end %}`, "ok", scope{"a": []byte(nil)}},
	{`{% if bytes(a) != nil %}ok{% end %}`, "ok", scope{"a": []byte{1, 2}}},
	{`{% if b, ok := bytes(a).(bytes); ok %}{{ b }}{% end %}`, "97, 226, 130, 172, 98", scope{"a": "a€b"}},
}

var rendererVarsToScope = []struct {
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

func TestRenderExpressions(t *testing.T) {
	for _, expr := range rendererExprTests {
		var tree, err = parser.ParseSource([]byte("{{"+expr.src+"}}"), ast.ContextText)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = RenderTree(b, tree, expr.vars, true)
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

func TestRenderStatements(t *testing.T) {
	builtins["test2"] = func(a, b int) (int, int) {
		return a, b
	}
	for _, stmt := range rendererStmtTests {
		var tree, err = parser.ParseSource([]byte(stmt.src), ast.ContextText)
		if err != nil {
			t.Errorf("source: %q, %s\n", stmt.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = RenderTree(b, tree, stmt.vars, true)
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
	for _, p := range rendererVarsToScope {
		res, err := varsToScope(p.vars)
		if err != nil {
			t.Errorf("vars: %#v, %q\n", p.vars, err)
			continue
		}
		if !reflect.DeepEqual(res, p.res) {
			t.Errorf("vars: %#v, unexpected %q, expecting %q\n", p.vars, res, p.res)
		}
	}
}

type RenderError struct{}

func (wr RenderError) Render(w io.Writer) (int, error) {
	return 0, errors.New("RenderTree error")
}

type RenderPanic struct{}

func (wr RenderPanic) Render(w io.Writer) (int, error) {
	panic("RenderTree panic")
}

func TestRenderErrors(t *testing.T) {
	tree := ast.NewTree("", []ast.Node{ast.NewValue(nil, ast.NewIdentifier(nil, "a"), ast.ContextText)}, ast.ContextText)
	err := RenderTree(ioutil.Discard, tree, scope{"a": RenderError{}}, true)
	if err == nil {
		t.Errorf("expecting not nil error\n")
	} else if err.Error() != "RenderTree error" {
		t.Errorf("unexpected error %q, expecting 'RenderTree error'\n", err)
	}

	err = RenderTree(ioutil.Discard, tree, scope{"a": RenderPanic{}}, true)
	if err == nil {
		t.Errorf("expecting not nil error\n")
	} else if err.Error() != "RenderTree panic" {
		t.Errorf("unexpected error %q, expecting 'RenderTree panic'\n", err)
	}
}
