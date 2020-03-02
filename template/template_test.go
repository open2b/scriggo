// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"errors"
	"io"
	"math"
	"reflect"
	"strings"
	"testing"

	"scriggo"
	"scriggo/runtime"

	"github.com/google/go-cmp/cmp"
)

var rendererExprTests = []struct {
	src      string
	expected string
	globals  map[string]interface{}
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
	// {"-0.0", "0", nil}, // TODO(Gianluca).
	{"true", "true", nil},
	{"false", "false", nil},
	// {"true", "_true_", Vars{"true": "_true_"}},
	// {"false", "_false_", Vars{"false": "_false_"}},
	{"2 - 3", "-1", nil},
	{"2 * 3", "6", nil},
	{"1 + 2 * 3 + 1", "8", nil},
	{"2.2 * 3", "6.6", nil},
	{"2 * 3.1", "6.2", nil},
	{"2.0 * 3.1", "6.2", nil},
	// {"2 / 3", "0", nil},
	{"2.0 / 3", "0.6666666666666666", nil},
	{"2 / 3.0", "0.6666666666666666", nil},
	{"2.0 / 3.0", "0.6666666666666666", nil},
	// {"7 % 3", "1", nil},
	{"-2147483648 * -1", "2147483648", nil},                   // math.MinInt32 * -1
	{"-2147483649 * -1", "2147483649", nil},                   // (math.MinInt32-1) * -1
	{"2147483647 * -1", "-2147483647", nil},                   // math.MaxInt32 * -1
	{"2147483648 * -1", "-2147483648", nil},                   // (math.MaxInt32+1) * -1
	{"9223372036854775807 * -1", "-9223372036854775807", nil}, // math.MaxInt64 * -1
	{"-2147483648 / -1", "2147483648", nil},                   // math.MinInt32 / -1
	{"-2147483649 / -1", "2147483649", nil},                   // (math.MinInt32-1) / -1
	{"2147483647 / -1", "-2147483647", nil},                   // math.MaxInt32 / -1
	{"2147483648 / -1", "-2147483648", nil},                   // (math.MaxInt32+1) / -1
	{"9223372036854775807 / -1", "-9223372036854775807", nil}, // math.MaxInt64 / -1
	{"2147483647 + 2147483647", "4294967294", nil},            // math.MaxInt32 + math.MaxInt32
	{"-2147483648 + -2147483648", "-4294967296", nil},         // math.MinInt32 + math.MinInt32
	// {"-1 + -2 * 6 / ( 6 - 1 - ( 5 * 3 ) + 2 ) * ( 1 + 2 ) * 3", "8", nil},
	{"-1 + -2 * 6 / ( 6 - 1 - ( 5 * 3 ) + 2.0 ) * ( 1 + 2 ) * 3", "12.5", nil},
	// {"a[1]", "y", Vars{"a": []string{"x", "y", "z"}}},
	// {"a[:]", "x, y, z", Vars{"a": []string{"x", "y", "z"}}},
	// {"a[1:]", "y, z", Vars{"a": []string{"x", "y", "z"}}},
	// {"a[:2]", "x, y", Vars{"a": []string{"x", "y", "z"}}},
	// {"a[1:2]", "y", Vars{"a": []string{"x", "y", "z"}}},
	// {"a[1:3]", "y, z", Vars{"a": []string{"x", "y", "z"}}},
	// {"a[0:3]", "x, y, z", Vars{"a": []string{"x", "y", "z"}}},
	// {"a[2:2]", "", Vars{"a": []string{"x", "y", "z"}}},
	// {"a[0]", "120", Vars{"a": "x€z"}},
	// {"a[1]", "226", Vars{"a": "x€z"}},
	// {"a[2]", "130", Vars{"a": "x€z"}},
	// {"a[2.2/1.1]", "z", Vars{"a": []string{"x", "y", "z"}}},
	// {"a[1]", "98", Vars{"a": HTML("<b>")}},
	// {"a[0]", "60", Vars{"a": HTML("<b>")}},
	// {"a[1]", "98", Vars{"a": stringConvertible("abc")}},
	// {"a[:]", "x€z", Vars{"a": "x€z"}},
	// {"a[1:]", "€z", Vars{"a": "x€z"}},
	// {"a[:2]", "x\xe2", Vars{"a": "x€z"}},
	// {"a[1:2]", "\xe2", Vars{"a": "x€z"}},
	// {"a[1:3]", "\xe2\x82", Vars{"a": "x€z"}},
	// {"a[0:3]", "x\xe2\x82", Vars{"a": "x€z"}},
	// {"a[1:]", "\x82\xacxz", Vars{"a": "€xz"}},
	// {"a[:2]", "xz", Vars{"a": "xz€"}},
	// {"a[2:2]", "", Vars{"a": "xz€"}},
	// {"a[1:]", "b>", Vars{"a": HTML("<b>")}},
	// {"a[1:]", "z€", Vars{"a": stringConvertible("xz€")}},
	// {`interface{}(a).(string)`, "abc", Vars{"a": "abc"}},
	// {`interface{}(a).(string)`, "<b>", Vars{"a": HTML("<b>")}},
	// {`interface{}(a).(int)`, "5", Vars{"a": 5}},
	// {`interface{}(a).(int64)`, "5", Vars{"a": int64(5)}},
	// {`interface{}(a).(int32)`, "5", Vars{"a": int32(5)}},
	// {`interface{}(a).(int16)`, "5", Vars{"a": int16(5)}},
	// {`interface{}(a).(int8)`, "5", Vars{"a": int8(5)}},
	// {`interface{}(a).(uint)`, "5", Vars{"a": uint(5)}},
	// {`interface{}(a).(uint64)`, "5", Vars{"a": uint64(5)}},
	// {`interface{}(a).(uint32)`, "5", Vars{"a": uint32(5)}},
	// {`interface{}(a).(uint16)`, "5", Vars{"a": uint16(5)}},
	// {`interface{}(a).(uint8)`, "5", Vars{"a": uint8(5)}},
	// {`interface{}(a).(float64)`, "5.5", Vars{"a": 5.5}},
	// {`interface{}(a).(float32)`, "5.5", Vars{"a": float32(5.5)}},
	// {`interface{}((5)).(int)`, "5", nil},
	// {`interface{}((5.5)).(float64)`, "5.5", nil},
	// {`interface{}('a').(rune)`, "97", nil},
	// {`interface{}(a).(bool)`, "true", Vars{"a": true}},
	// {`interface{}(a).(error)`, "err", Vars{"a": errors.New("err")}}, // https://github.com/open2b/scriggo/issues/64.

	// slice
	// {"[]int{-3}[0]", "-3", nil},
	// {`[]string{"a","b","c"}[0]`, "a", nil},
	// {`[][]int{[]int{1,2}, []int{3,4,5}}[1][2]`, "5", nil},
	// {`len([]string{"a", "b", "c"})`, "3", nil},
	// {`[]string{0: "zero", 2: "two"}[2]`, "two", nil},
	// {`[]int{ 8: 64, 81, 5: 25,}[9]`, "81", nil},
	// {`[]byte{0, 4}[0]`, "0", nil},
	// {`[]byte{0, 124: 97}[124]`, "97", nil},
	// {"[]interface{}{}", "", nil},
	// {"len([]interface{}{})", "0", nil},
	// {"[]interface{}{v}", "", map[string]interface{}{"v": []string(nil)}},
	// {"len([]interface{}{v})", "1", map[string]interface{}{"v": []string(nil)}},
	// {"[]interface{}{v, v2}", ", ", map[string]interface{}{"v": []string(nil), "v2": []string(nil)}},
	// {"[]interface{}{`a`}", "a", nil},
	// {"[]interface{}{`a`, `b`, `c`}", "a, b, c", nil},
	// {"[]interface{}{HTML(`<a>`), HTML(`<b>`), HTML(`<c>`)}", "<a>, <b>, <c>", nil},
	// {"[]interface{}{4, 9, 3}", "4, 9, 3", nil},
	// {"[]interface{}{4.2, 9.06, 3.7}", "4.2, 9.06, 3.7", nil},
	// {"[]interface{}{false, false, true}", "false, false, true", nil},
	// {"[]interface{}{`a`, 8, true, HTML(`<b>`)}", "a, 8, true, <b>", nil},
	// {`[]interface{}{"a",2,3.6,HTML("<b>")}`, "a, 2, 3.6, <b>", nil},
	// {`[]interface{}{[]interface{}{1,2},"/",[]interface{}{3,4}}`, "1, 2, /, 3, 4", nil},
	// {`[]interface{}{0: "zero", 2: "two"}[2]`, "two", nil},
	// {`[]interface{}{2: "two", "three", "four"}[4]`, "four", nil},

	// array
	// {`[2]int{-30, 30}[0]`, "-30", nil},
	// {`[1][2]int{[2]int{-30, 30}}[0][1]`, "30", nil},
	// {`[4]string{0: "zero", 2: "two"}[2]`, "two", nil},
	// {`[...]int{4: 5}[4]`, "5", nil},

	// map
	// {"len(map[interface{}]interface{}{})", "0", nil},
	// {`map[interface{}]interface{}{1: 1, 2: 4, 3: 9}[2]`, "4", nil},
	// {`map[int]int{1: 1, 2: 4, 3: 9}[2]`, "4", nil},
	// {`10 + map[string]int{"uno": 1, "due": 2}["due"] * 3`, "16", nil},
	// {`len(map[interface{}]interface{}{1: 1, 2: 4, 3: 9})`, "3", nil},
	// {`s["a"]`, "3", Vars{"s": map[interface{}]int{"a": 3}}},
	// {`s[nil]`, "3", Vars{"s": map[interface{}]int{nil: 3}}},

	// struct
	// {`s{1, 2}.A`, "1", Vars{"s": reflect.TypeOf(struct{ A, B int }{})}},

	// composite literal with implicit type
	// {`[][]int{{1},{2,3}}[1][1]`, "3", nil},
	// {`[][]string{{"a", "b"}}[0][0]`, "a", nil},
	// {`map[string][]int{"a":{1,2}}["a"][1]`, "2", nil},
	// {`map[[2]int]string{{1,2}:"a"}[[2]int{1,2}]`, "a", nil},
	// {`[2][1]int{{1}, {5}}[1][0]`, "5", nil},
	// {`[]Point{{1,2}, {3,4}, {5,6}}[2].Y`, "6", Vars{"Point": reflect.TypeOf(struct{ X, Y float64 }{})}},
	// {`(*(([]*Point{{3,4}})[0])).X`, "3", Vars{"Point": reflect.TypeOf(struct{ X, Y float64 }{})}},

	// make
	// {`make([]int, 5)[0]`, "0", nil},
	// {`make([]int, 5, 10)[0]`, "0", nil},
	// {`make(map[string]int, 5)["key"]`, "0", nil},

	// selectors
	// {"a.B", "b", Vars{"a": &struct{ B string }{B: "b"}}},
	// TODO (Gianluca): field renaming is currently not supported by
	// type-checker.
	// {"a.b", "b", Vars{"a": &struct {
	// 	B string `scriggo:"b"`
	// }{B: "b"}}},
	// {"a.b", "b", Vars{"a": &struct {
	// 	C string `scriggo:"b"`
	// }{C: "b"}}},

	// ==, !=
	{"true == true", "true", nil},
	{"false == false", "true", nil},
	{"true == false", "false", nil},
	{"false == true", "false", nil},
	{"true != true", "false", nil},
	{"false != false", "false", nil},
	{"true != false", "true", nil},
	{"false != true", "true", nil},
	// {"a == nil", "true", Vars{"a": nil}},
	// {"a != nil", "false", Vars{"a": nil}},
	// {"nil == a", "true", Vars{"a": nil}},
	// {"nil != a", "false", Vars{"a": nil}},
	// {"a == nil", "false", Vars{"a": "b"}},
	// {"a == nil", "false", Vars{"a": 5}},
	{"5 == 5", "true", nil},
	// {`a == "a"`, "true", Vars{"a": "a"}},
	// {`a == "a"`, "true", Vars{"a": HTML("a")}},
	// {`a != "b"`, "true", Vars{"a": "a"}},
	// {`a != "b"`, "true", Vars{"a": HTML("a")}},
	// {`a == "<a>"`, "true", Vars{"a": "<a>"}},
	// {`a == "<a>"`, "true", Vars{"a": HTML("<a>")}},
	// {`a != "<b>"`, "false", Vars{"a": "<b>"}},
	// {`a != "<b>"`, "false", Vars{"a": HTML("<b>")}},

	// https://github.com/open2b/scriggo/issues/177.
	// {"[]interface{}{} == nil", "false", nil},
	// {"[]byte{} == nil", "false", nil},

	// &&
	// {"true && true", "true", nil},
	// {"true && false", "false", nil},
	// {"false && true", "false", nil},
	// {"false && false", "false", nil},
	// {"false && 0/a == 0", "false", Vars{"a": 0}},

	// ||
	{"true || true", "true", nil},
	{"true || false", "true", nil},
	{"false || true", "true", nil},
	{"false || false", "false", nil},
	// {"true || 0/a == 0", "true", Vars{"a": 0}},

	// +
	{"2 + 3", "5", nil},
	{`"a" + "b"`, "ab", nil},
	// {`a + "b"`, "ab", Vars{"a": "a"}},
	// {`a + "b"`, "ab", Vars{"a": HTML("a")}},
	// {`a + "b"`, "<a>b", Vars{"a": "<a>"}},
	// {`a + "b"`, "<a>b", Vars{"a": HTML("<a>")}},
	// {`a + "<b>"`, "<a><b>", Vars{"a": "<a>"}},
	// {`a + "<b>"`, "<a><b>", Vars{"a": HTML("<a>")}},
	// {"a + b", "<a><b>", Vars{"a": "<a>", "b": "<b>"}},
	// {"a + b", "<a><b>", Vars{"a": HTML("<a>"), "b": HTML("<b>")}},

	// call
	// {"f()", "ok", Vars{"f": func() string { return "ok" }}},
	// {"f(5)", "5", Vars{"f": func(i int) int { return i }}},
	// {"f(5.4)", "5.4", Vars{"f": func(n float64) float64 { return n }}},
	// {"f(5)", "5", Vars{"f": func(n int) int { return n }}},
	// {"f(`a`)", "a", Vars{"f": func(s string) string { return s }}},
	// {"f(HTML(`<a>`))", "<a>", Vars{"f": func(s string) string { return s }}},
	// {"f(true)", "true", Vars{"f": func(t bool) bool { return t }}},
	// {"f(5)", "5", Vars{"f": func(v interface{}) interface{} { return v }}},
	// {"f(`a`)", "a", Vars{"f": func(v interface{}) interface{} { return v }}},
	// {"f(HTML(`<a>`))", "<a>", Vars{"f": func(s string) string { return s }}},
	// {"f(true)", "true", Vars{"f": func(v interface{}) interface{} { return v }}},
	// {"f(nil)", "", Vars{"f": func(v interface{}) interface{} { return v }}},
	// {"f()", "", Vars{"f": func(s ...string) string { return strings.Join(s, ",") }}},
	// {"f(`a`)", "a", Vars{"f": func(s ...string) string { return strings.Join(s, ",") }}},
	// {"f(`a`, `b`)", "a,b", Vars{"f": func(s ...string) string { return strings.Join(s, ",") }}},
	// {"f(5)", "5 ", Vars{"f": func(i int, s ...string) string { return strconv.Itoa(i) + " " + strings.Join(s, ",") }}},
	// {"f(5, `a`, `b`)", "5 a,b", Vars{"f": func(i int, s ...string) string { return strconv.Itoa(i) + " " + strings.Join(s, ",") }}},
	// {"s.F()", "a", Vars{"s": aMap{v: "a"}}},
	// {"s.G()", "b", Vars{"s": aMap{v: "a", H: func() string { return "b" }}}},
	// {"f(5.2)", "5.2", Vars{"f": func(d float64) float64 { return d }}},

	// number types
	// {"1+a", "3", Vars{"a": int(2)}},
	// {"1+a", "3", Vars{"a": int8(2)}},
	// {"1+a", "3", Vars{"a": int16(2)}},
	// {"1+a", "3", Vars{"a": int32(2)}},
	// {"1+a", "3", Vars{"a": int64(2)}},
	// {"1+a", "3", Vars{"a": uint8(2)}},
	// {"1+a", "3", Vars{"a": uint16(2)}},
	// {"1+a", "3", Vars{"a": uint32(2)}},
	// {"1+a", "3", Vars{"a": uint64(2)}},
	// {"1+a", "3.5", Vars{"a": float32(2.5)}},
	// {"1+a", "3.5", Vars{"a": float64(2.5)}},
	// {"f(a)", "3", Vars{"f": func(n int) int { return n + 1 }, "a": int(2)}},
	// {"f(a)", "3", Vars{"f": func(n int8) int8 { return n + 1 }, "a": int8(2)}},
	// {"f(a)", "3", Vars{"f": func(n int16) int16 { return n + 1 }, "a": int16(2)}},
	// {"f(a)", "3", Vars{"f": func(n int32) int32 { return n + 1 }, "a": int32(2)}},
	// {"f(a)", "3", Vars{"f": func(n int64) int64 { return n + 1 }, "a": int64(2)}},
	// {"f(a)", "3", Vars{"f": func(n uint8) uint8 { return n + 1 }, "a": uint8(2)}},
	// {"f(a)", "3", Vars{"f": func(n uint16) uint16 { return n + 1 }, "a": uint16(2)}},
	// {"f(a)", "3", Vars{"f": func(n uint32) uint32 { return n + 1 }, "a": uint32(2)}},
	// {"f(a)", "3", Vars{"f": func(n uint64) uint64 { return n + 1 }, "a": uint64(2)}},
	// {"f(a)", "3", Vars{"f": func(n float32) float32 { return n + 1 }, "a": float32(2.0)}},
	// {"f(a)", "3", Vars{"f": func(n float64) float64 { return n + 1 }, "a": float64(2.0)}},
}

func TestRenderExpressions(t *testing.T) {
	for _, cas := range rendererExprTests {
		t.Run(cas.src, func(t *testing.T) {
			r := MapReader{"index.html": []byte("{{" + cas.src + "}}")}
			templ, err := Load("index.html", r, nil, LanguageText, nil)
			if err != nil {
				t.Fatalf("source %q: loading error: %s", cas.src, err)
			}
			b := &bytes.Buffer{}
			err = templ.Render(b, nil, nil)
			if err != nil {
				t.Fatalf("source %q: rendering error: %s", cas.src, err)
			}
			if cas.expected != b.String() {
				t.Fatalf("source %q: expecting %q, got %q", cas.src, cas.expected, b)
			}
		})
	}
}

var rendererStmtTests = []struct {
	src      string
	expected string
	globals  Vars
}{
	{"{% if true %}ok{% else %}no{% end %}", "ok", nil},
	{"{% if false %}no{% else %}ok{% end %}", "ok", nil},
	{"{% if a := true; a %}ok{% else %}no{% end %}", "ok", nil},
	{"{% if a := false; a %}no{% else %}ok{% end %}", "ok", nil},
	{"{% a := false %}{% if a = true; a %}ok{% else %}no{% end %}", "ok", nil},
	{"{% a := true %}{% if a = false; a %}no{% else %}ok{% end %}", "ok", nil},
	{"{% if x := 2; x == 2 %}x is 2{% else if x == 3 %}x is 3{% else %}?{% end %}", "x is 2", nil},
	{"{% if x := 3; x == 2 %}x is 2{% else if x == 3 %}x is 3{% else %}?{% end %}", "x is 3", nil},
	{"{% if x := 10; x == 2 %}x is 2{% else if x == 3 %}x is 3{% else %}?{% end %}", "?", nil},
	{"{% a := \"hi\" %}{% if a := 2; a == 3 %}{% else if a := false; a %}{% else %}{{ a }}{% end %}, {{ a }}", "false, hi", nil}, // https://play.golang.org/p/2OXyyKwCfS8
	{"{% if false %}{% else if true %}first true{% else if true %}second true{% else %}{% end %}", "first true", nil},
	{"{% x := 10 %}{% if false %}{% else if true %}{% if false %}{% else if true %}x is {% end %}{% else if false %}{% end %}{{ 10 }}", "x is 10", nil},
	{"{% a, b := 1, 2 %}{% if a == 1 && b == 2 %}ok{% end %}", "ok", nil},
	{"{% a, b, c := 1, 2, 3 %}{% if ( a == 1 && b == 2 ) && c == 3 %}ok{% end %}", "ok", nil},
	{"{% a, b, c, d := 1, 2, 3, 4 %}{% if ( a == 1 && b == 2 ) && ( c == 3 && d == 4 ) %}ok{% end %}", "ok", nil},
	{"{% a, b := 1, 2 %}{% a, b = b, a %}{% if a == 2 && b == 1 %}ok{% end %}", "ok", nil},
	// {"{% if a, ok := b[`c`]; ok %}ok{% else %}no{% end %}", "ok", Vars{"b": map[interface{}]interface{}{"c": true}}},
	// {"{% if a, ok := b[`d`]; ok %}no{% else %}ok{% end %}", "ok", Vars{"b": map[interface{}]interface{}{}}},
	// {"{% if a, ok := b[`c`]; a %}ok{% else %}no{% end %}", "ok", Vars{"b": map[interface{}]interface{}{"c": true}}},
	// {"{% if a, ok := b[`d`]; a %}no{% else %}ok{% end %}", "ok", Vars{"b": map[interface{}]interface{}{"d": false}}},
	// {"{% if a, ok := b[`c`][`d`]; ok %}no{% else %}ok{% end %}", "ok", Vars{"b": map[interface{}]interface{}{"c": map[interface{}]interface{}{}}}},
	// {"{% if a, ok := b.(string); ok %}ok{% else %}no{% end %}", "ok", Vars{"b": "abc"}},
	// {"{% if a, ok := b.(string); ok %}no{% else %}ok{% end %}", "ok", Vars{"b": 5}},
	// {"{% if a, ok := b.(int); ok %}ok{% else %}no{% end %}", "ok", Vars{"b": 5}},
	// {"{% if a, ok := b[`c`].(int); ok %}ok{% else %}no{% end %}", "ok", Vars{"b": map[interface{}]interface{}{"c": 5}}},
	// {"{% if a, ok := b.(byte); ok %}ok{% else %}no{% end %}", "ok", Vars{"b": byte(5)}},
	// {"{% if a, ok := b[`c`].(byte); ok %}ok{% else %}no{% end %}", "ok", Vars{"b": map[interface{}]interface{}{"c": byte(5)}}},
	// {"{% b := map[interface{}]interface{}{HTML(`<b>`): true} %}{% if a, ok := b[HTML(`<b>`)]; ok %}ok{% else %}no{% end %}", "ok", nil},
	// {"{% b := map[interface{}]interface{}{5.2: true} %}{% if a, ok := b[5.2]; ok %}ok{% else %}no{% end %}", "ok", nil},
	// {"{% b := map[interface{}]interface{}{5: true} %}{% if a, ok := b[5]; ok %}ok{% else %}no{% end %}", "ok", nil},
	// {"{% b := map[interface{}]interface{}{true: true} %}{% if a, ok := b[true]; ok %}ok{% else %}no{% end %}", "ok", nil},
	{"{% b := map[interface{}]interface{}{nil: true} %}{% if a, ok := b[nil]; ok %}ok{% else %}no{% end %}", "ok", nil},
	{"{% a := 5 %}{% if true %}{% a = 7 %}{{ a }}{% end %}", "7", nil},
	{"{% a := 5 %}{% if true %}{% a := 7 %}{{ a }}{% end %}", "7", nil},
	{"{% a := 5 %}{% if true %}{% a := 7 %}{% a = 9 %}{{ a }}{% end %}", "9", nil},
	// {"{% a := 5 %}{% if true %}{% a := 7 %}{% a, b := test2(1,2) %}{{ a }}{% end %}", "1", nil},
	// {"{% a := 5 %}{% if true %}{% a, b := test2(7,8) %}{% a, b = test2(1,2) %}{{ a }}{% end %}", "1", nil},
	{"{% _ = 5 %}", "", nil},
	// {"{% _, a := test2(4,5) %}{{ a }}", "5", nil},
	// {"{% a := 3 %}{% _, a = test2(4,5) %}{{ a }}", "5", nil},

	// https://github.com/open2b/scriggo/issues/324
	// {"{% a := []interface{}{1,2,3} %}{% a[1] = 5 %}{{ a }}", "1, 5, 3", nil},
	// {"{% s := []interface{}{1,2,3} %}{% s2 := s[0:2] %}{% s2[0] = 5 %}{{ s2 }}", "5, 2", nil},

	// {"{% a := map[interface{}]interface{}{`b`:1} %}{% a[`b`] = 5 %}{{ a[`b`] }}", "5", nil},
	// {"{% a := 0 %}{% a, a = test2(1,2) %}{{ a }}", "2", nil},
	// {"{% a, b := test2(1,2) %}{{ a }},{{ b }}", "1,2", nil},
	// {"{% a := 0 %}{% a, b := test2(1,2) %}{{ a }},{{ b }}", "1,2", nil},
	// {"{% b := 0 %}{% a, b := test2(1,2) %}{{ a }},{{ b }}", "1,2", nil},
	// {"{% s := []interface{}{1,2,3} %}{% s[0] = 5 %}{{ s[0] }}", "5", nil},
	{`{% x := []string{"a","c","b"} %}{{ x[0] }}{{ x[2] }}{{ x[1] }}`, "abc", nil},
	// {"{% for i, p := range products %}{{ i }}: {{ p }}\n{% end %}", "0: a\n1: b\n2: c\n",
	// 	Vars{"products": []string{"a", "b", "c"}}},
	// {"{% for _, p := range products %}{{ p }}\n{% end %}", "a\nb\nc\n",
	// 	Vars{"products": []string{"a", "b", "c"}}},
	// {"{% for _, p := range products %}a{% break %}b\n{% end %}", "a",
	// 	Vars{"products": []string{"a", "b", "c"}}},
	// {"{% for _, p := range products %}a{% continue %}b\n{% end %}", "aaa",
	// 	Vars{"products": []string{"a", "b", "c"}}},
	{"{% for _, c := range \"\" %}{{ c }}{% end %}", "", nil},
	{"{% for _, c := range \"a\" %}({{ c }}){% end %}", "(97)", nil},
	{"{% for _, c := range \"aÈc\" %}({{ c }}){% end %}", "(97)(200)(99)", nil},
	// {"{% for _, c := range HTML(\"<b>\") %}({{ c }}){% end %}", "(60)(98)(62)", nil},
	// {"{% for _, i := range []interface{}{ `a`, `b`, `c` } %}{{ i }}{% end %}", "abc", nil},
	// {"{% for _, i := range []interface{}{ HTML(`<`), HTML(`&`), HTML(`>`) } %}{{ i }}{% end %}", "<&>", nil},
	{"{% for _, i := range []interface{}{1, 2, 3, 4, 5} %}{{ i }}{% end %}", "12345", nil},
	{"{% for _, i := range []interface{}{1.3, 5.8, 2.5} %}{{ i }}{% end %}", "1.35.82.5", nil},
	{"{% for _, i := range []byte{ 0, 1, 2 } %}{{ i }}{% end %}", "012", nil},
	// {"{% s := []interface{}{} %}{% for k, v := range map[interface{}]interface{}{`a`: `1`, `b`: `2`} %}{% s = append(s, k+`:`+v) %}{% end %}{% sort(s) %}{{ s }}", "a:1, b:2", nil},
	{"{% for k, v := range map[interface{}]interface{}{} %}{{ k }}:{{ v }},{% end %}", "", nil},
	// {"{% s := []interface{}{} %}{% for k, v := range m %}{% s = append(s, itoa(k)+`:`+itoa(v)) %}{% end %}{% sort(s) %}{{ s }}", "1:1, 2:4, 3:9", Vars{"m": map[int]int{1: 1, 2: 4, 3: 9}}},
	// {"{% for p in products %}{{ p }}\n{% end %}", "a\nb\nc\n",
	// 	Vars{"products": []string{"a", "b", "c"}}},
	// {"{% i := 0 %}{% c := \"\" %}{% for i, c = range \"ab\" %}({{ c }}){% end %}{{ i }}", "(97)(98)1", nil},
	{"{% for range []interface{}{ `a`, `b`, `c` } %}.{% end %}", "...", nil},
	{"{% for range []byte{ 1, 2, 3 } %}.{% end %}", "...", nil},
	{"{% for range []interface{}{} %}.{% end %}", "", nil},
	{"{% for i := 0; i < 5; i++ %}{{ i }}{% end %}", "01234", nil},
	{"{% for i := 0; i < 5; i++ %}{{ i }}{% break %}{% end %}", "0", nil},
	{"{% for i := 0; ; i++ %}{{ i }}{% if i == 4 %}{% break %}{% end %}{% end %}", "01234", nil},
	// {"{% for i := 0; i < 5; i++ %}{{ i }}{% if i == 4 %}{% continue %}{% end %},{% end %}", "0,1,2,3,4", nil},
	{"{% switch %}{% end %}", "", nil},
	{"{% switch %}{% case true %}ok{% end %}", "ok", nil},
	{"{% switch ; %}{% case true %}ok{% end %}", "ok", nil},
	{"{% i := 2 %}{% switch i++; %}{% case true %}{{ i }}{% end %}", "3", nil},
	{"{% switch ; true %}{% case true %}ok{% end %}", "ok", nil},
	{"{% switch %}{% default %}default{% case true %}true{% end %}", "true", nil},
	// {"{% switch interface{}(\"hey\").(type) %}{% default %}default{% case string %}string{% end %}", "string", nil},
	// {"{% switch a := 5; a := a.(type) %}{% case int %}ok{% end %}", "ok", nil},
	{"{% switch 3 %}{% case 3 %}three{% end %}", "three", nil},
	{"{% switch 4 + 5 %}{% case 4 %}{% case 9 %}nine{% end %}", "nine", nil},
	{"{% switch x := 1; x + 1 %}{% case 1 %}one{% case 2 %}two{% end %}", "two", nil},
	{"{% switch %}{% case 7 < 10 %}7 < 10{% default %}other{% end %}", "7 < 10", nil},
	{"{% switch %}{% case 7 > 10 %}7 > 10{% default %}other{% end %}", "other", nil},
	{"{% switch %}{% case true %}ok{% end %}", "ok", nil},
	{"{% switch %}{% case false %}no{% end %}", "", nil},
	{"{% switch %}{% case true %}ab{% break %}c{% end %}", "ab", nil},
	// {"{% switch a, b := 2, 4; c < d %}{% case true %}{{ a }}{% case false %}{{ b }}{% end %}", "4", Vars{"c": 100, "d": 90}},
	{"{% switch a := 4; %}{% case 3 < 4 %}{{ a }}{% end %}", "4", nil},
	// {"{% switch a.(type) %}{% case string %}is a string{% case int %}is an int{% default %}is something else{% end %}", "is an int", Vars{"a": 3}},
	// {"{% switch (a + b).(type) %}{% case string %}{{ a + b }} is a string{% case int %}is an int{% default %}is something else{% end %}", "msgmsg2 is a string", Vars{"a": "msg", "b": "msg2"}},
	// {"{% switch x.(type) %}{% case string %}is a string{% default %}is something else{% case int %}is an int{% end %}", "is something else", Vars{"x": false}},
	// {"{% switch v := a.(type) %}{% case string %}{{ v }} is a string{% case int %}{{ v }} is an int{% default %}{{ v }} is something else{% end %}", "12 is an int", Vars{"a": 12}},
	{"{% switch %}{% case 4 < 10 %}4 < 10, {% fallthrough %}{% case 4 == 10 %}4 == 10{% end %}", "4 < 10, 4 == 10", nil},
	// {"{% switch a, b := 10, \"hey\"; (a + 20).(type) %}{% case string %}string{% case int %}int, msg: {{ b }}{% default %}def{% end %}", "int, msg: hey", nil},
	{"{% switch %}{% case true %}abc{% fallthrough %}{% case false %}def{% end %}", "abcdef", nil},
	{"{% switch %}{% case true %}abc{% fallthrough %}  {# #}  {# #} {% case false %}def{% end %}", "abc     def", nil},
	{"{% i := 0 %}{% c := true %}{% for c %}{% i++ %}{{ i }}{% c = i < 5 %}{% end %}", "12345", nil},
	{"{% i := 0 %}{% for ; ; %}{% i++ %}{{ i }}{% if i == 4 %}{% break %}{% end %},{% end %} {{ i }}", "1,2,3,4 4", nil},
	{"{% i := 5 %}{% i++ %}{{ i }}", "6", nil},
	// {"{% s := map[interface{}]interface{}{`a`: 5} %}{% s[`a`]++ %}{{ s[`a`] }}", "6", nil},
	// {"{% s := []interface{}{5} %}{% s[0]++ %}{{ s[0] }}", "6", nil},
	// {"{% s := []byte{5} %}{% s[0]++ %}{{ s[0] }}", "6", nil},
	// {"{% s := []byte{255} %}{% s[0]++ %}{{ s[0] }}", "0", nil},
	{"{% i := 5 %}{% i-- %}{{ i }}", "4", nil},
	// {"{% s := map[interface{}]interface{}{`a`: 5} %}{% s[`a`]-- %}{{ s[`a`] }}", "4", nil},
	// {"{% s := []interface{}{5} %}{% s[0]-- %}{{ s[0] }}", "4", nil},
	// {"{% s := []byte{5} %}{% s[0]-- %}{{ s[0] }}", "4", nil},
	// {"{% s := []byte{0} %}{% s[0]-- %}{{ s[0] }}", "255", nil},
	// {`{% a := [3]int{4,5,6} %}{% b := getref(a) %}{{ b[1] }}`, "5", Vars{"getref": func(s [3]int) *[3]int { return &s }}},
	// {`{% a := [3]int{4,5,6} %}{% b := getref(a) %}{% b[1] = 10 %}{{ (*b)[1] }}`, "10", Vars{"getref": func(s [3]int) *[3]int { return &s }}},
	// {`{% s := T{5, 6} %}{% if s.A == 5 %}ok{% end %}`, "ok", Vars{"T": reflect.TypeOf(struct{ A, B int }{})}},
	// {`{% s := interface{}(3) %}{% if s == 3 %}ok{% end %}`, "ok", nil},
	{"{% a := 12 %}{% a += 9 %}{{ a }}", "21", nil},
	// {"{% a := `ab` %}{% a += `c` %}{% if _, ok := a.(string); ok %}{{ a }}{% end %}", "abc", nil},
	// {"{% a := HTML(`ab`) %}{% a += `c` %}{% if _, ok := a.(string); ok %}{{ a }}{% end %}", "abc", nil},
	{"{% a := 12 %}{% a -= 3 %}{{ a }}", "9", nil},
	{"{% a := 12 %}{% a *= 2 %}{{ a }}", "24", nil},
	{"{% a := 12 %}{% a /= 4 %}{{ a }}", "3", nil},
	{"{% a := 12 %}{% a %= 5 %}{{ a }}", "2", nil},
	{"{% a := 12.3 %}{% a += 9.1 %}{{ a }}", "21.4", nil},
	{"{% a := 12.3 %}{% a -= 3.7 %}{{ a }}", "8.600000000000001", nil},
	{"{% a := 12.3 %}{% a *= 2.1 %}{{ a }}", "25.830000000000002", nil},
	{"{% a := 12.3 %}{% a /= 4.9 %}{{ a }}", "2.510204081632653", nil},
	// {`{% a := 5 %}{% b := getref(a) %}{{ *b }}`, "5", Vars{"getref": func(a int) *int { return &a }}},
	{`{% a := 1 %}{% b := &a %}{% *b = 5 %}{{ a }}`, "5", nil},
	// {`{% a := 2 %}{% f(&a) %}{{ a }}`, "3", Vars{"f": func(a *int) { *a++ }}},
	// {"{% b := &[]int{0,1,4,9}[1] %}{% *b = 5  %}{{ *b }}", "5", nil},
	// {"{% a := [ ]int{0,1,4,9} %}{% b := &a[1] %}{% *b = 5  %}{{ a[1] }}", "5", nil},
	// {"{% a := [4]int{0,1,4,9} %}{% b := &a[1] %}{% *b = 10 %}{{ a[1] }}", "10", nil},
	// {"{% p := Point{4.0, 5.0} %}{% px := &p.X %}{% *px = 8.6 %}{{ p.X }}", "8.6", Vars{"Point": reflect.TypeOf(struct{ X, Y float64 }{})}},
	// {`{% a := &A{3, 4} %}ok`, "ok", Vars{"A": reflect.TypeOf(struct{ X, Y int }{})}},
	// {`{% a := &A{3, 4} %}{{ (*a).X }}`, "3", Vars{"A": reflect.TypeOf(struct{ X, Y int }{})}},
	// {`{% a := &A{3, 4} %}{{ a.X }}`, "3", Vars{"A": reflect.TypeOf(struct{ X, Y int }{})}},
	// {`{% a := 2 %}{% c := &(*(&a)) %}{% *c = 5 %}{{ a }}`, "5", nil},
	{"{# comment #}", "", nil},
	{"a{# comment #}b", "ab", nil},
	{`{% switch %}{% case true %}{{ 5 }}{% end %}ok`, "5ok", nil},

	// conversions

	// string
	// {`{% if s, ok := string("abc").(string); ok %}{{ s }}{% end %}`, "abc", nil},
	// {`{% if s, ok := string(HTML("<b>")).(string); ok %}{{ s }}{% end %}`, "<b>", nil},
	// {`{% if s, ok := string(88).(string); ok %}{{ s }}{% end %}`, "X", nil},
	// {`{% if s, ok := string(88888888888).(string); ok %}{{ s }}{% end %}`, "\uFFFD", nil},
	//{`{% if s, ok := string(slice{}).(string); ok %}{{ s }}{% end %}`, "", nil},
	//{`{% if s, ok := string(slice{35, 8364}).(string); ok %}{{ s }}{% end %}`, "#€", nil},
	//{`{% if s, ok := string(a).(string); ok %}{{ s }}{% end %}`, "#€", Vars{"a": []int{35, 8364}}},
	// {`{% if s, ok := string([]byte{}).(string); ok %}{{ s }}{% end %}`, "", nil},
	// {`{% if s, ok := string([]byte{97, 226, 130, 172, 98}).(string); ok %}{{ s }}{% end %}`, "a€b", nil},
	// {`{% if s, ok := string(a).(string); ok %}{{ s }}{% end %}`, "a€b", Vars{"a": []byte{97, 226, 130, 172, 98}}},

	// int
	{`{% if s, ok := interface{}(int(5)).(int); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := interface{}(int(5.0)).(int); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := interface{}(int(2147483647)).(int); ok %}{{ s }}{% end %}`, "2147483647", nil},
	{`{% if s, ok := interface{}(int(-2147483648)).(int); ok %}{{ s }}{% end %}`, "-2147483648", nil},

	// float64
	{`{% if s, ok := interface{}(float64(5)).(float64); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := interface{}(float64(5.5)).(float64); ok %}{{ s }}{% end %}`, "5.5", nil},

	// float32
	{`{% if s, ok := interface{}(float32(5)).(float32); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := interface{}(float32(5.5)).(float32); ok %}{{ s }}{% end %}`, "5.5", nil},

	// rune
	{`{% if s, ok := interface{}(rune(5)).(rune); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := interface{}(rune(2147483647)).(rune); ok %}{{ s }}{% end %}`, "2147483647", nil},
	{`{% if s, ok := interface{}(rune(-2147483648)).(rune); ok %}{{ s }}{% end %}`, "-2147483648", nil},

	// byte
	{`{% if s, ok := interface{}(byte(5)).(byte); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := interface{}(byte(255)).(byte); ok %}{{ s }}{% end %}`, "255", nil},

	// map
	// {`{% if _, ok := map[interface{}]interface{}(a).(map[interface{}]interface{}); ok %}ok{% end %}`, "ok", Vars{"a": map[interface{}]interface{}{}}},
	// {`{% if map[interface{}]interface{}(a) != nil %}ok{% end %}`, "ok", Vars{"a": map[interface{}]interface{}{}}},
	// {`{% a := map[interface{}]interface{}(nil) %}ok`, "ok", nil},

	// slice
	// {`{% if _, ok := interface{}([]int{1,2,3}).([]int); ok %}ok{% end %}`, "ok", nil},
	// {`{% if _, ok := interface{}([]interface{}(a)).([]interface{}); ok %}ok{% end %}`, "ok", Vars{"a": []interface{}{}}},
	// {`{% if []interface{}(a) != nil %}ok{% end %}`, "ok", Vars{"a": []interface{}{}}}, // TODO (Gianluca): https://github.com/open2b/scriggo/issues/63.
}

func TestRenderStatements(t *testing.T) {
	for _, cas := range rendererStmtTests {
		t.Run(cas.src, func(t *testing.T) {
			r := MapReader{"index.html": []byte(cas.src)}
			templ, err := Load("index.html", r, nil, LanguageText, nil)
			if err != nil {
				t.Fatalf("source %q: loading error: %s", cas.src, err)
			}
			b := &bytes.Buffer{}
			err = templ.Render(b, nil, nil)
			if err != nil {
				t.Fatalf("source %q: rendering error: %s", cas.src, err)
			}
			if cas.expected != b.String() {
				t.Fatalf("source %q: expecting %q, got %q", cas.src, cas.expected, b)
			}
		})
	}
}

var rendererGlobalsToScope = []struct {
	globals interface{}
	res     Vars
}{
	{
		nil,
		Vars{},
	},
	{
		Vars{"a": 1, "b": "s"},
		Vars{"a": 1, "b": "s"},
	},
	{
		reflect.ValueOf(map[string]interface{}{"a": 1, "b": "s"}),
		Vars{"a": 1, "b": "s"},
	},
	{
		map[string]interface{}{"a": 1, "b": "s"},
		Vars{"a": 1, "b": "s"},
	},
	{
		map[string]string{"a": "t", "b": "s"},
		Vars{"a": "t", "b": "s"},
	},
	{
		map[string]int{"a": 1, "b": 2},
		Vars{"a": 1, "b": 2},
	},
	{
		reflect.ValueOf(map[string]interface{}{"a": 1, "b": "s"}),
		Vars{"a": 1, "b": "s"},
	},
	{
		struct {
			A int    `scriggo:"a"`
			B string `scriggo:"b"`
			C bool
		}{A: 1, B: "s", C: true},
		Vars{"a": 1, "b": "s", "C": true},
	},
	{
		&struct {
			A int    `scriggo:"a"`
			B string `scriggo:"b"`
			C bool
		}{A: 1, B: "s", C: true},
		Vars{"a": 1, "b": "s", "C": true},
	},
	{
		reflect.ValueOf(struct {
			A int    `scriggo:"a"`
			B string `scriggo:"b"`
			C bool
		}{A: 1, B: "s", C: true}),
		Vars{"a": 1, "b": "s", "C": true},
	},
}

// refToCopy returns a reference to a copy of v (not to v itself).
func refToCopy(v interface{}) reflect.Value {
	rv := reflect.New(reflect.TypeOf(v))
	rv.Elem().Set(reflect.ValueOf(v))
	return rv
}

func TestGlobalsToScope(t *testing.T) {

	t.Skip("(not runnable)")

	// for _, p := range rendererGlobalsToScope {
	// 	res, err := globalsToScope(p.globals)
	// 	if err != nil {
	// 		t.Errorf("vars: %#v, %q\n", p.globals, err)
	// 		continue
	// 	}
	// 	if !reflect.DeepEqual(res, p.res) {
	// 		t.Errorf("vars: %#v, unexpected %q, expecting %q\n", p.globals, res, p.res)
	// 	}
	// }
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

	t.Skip("(not runnable)")

	// TODO (Gianluca): what's the point of this test?
	// tree := ast.NewTree("", []ast.Node{ast.NewShow(nil, ast.NewIdentifier(nil, "a"), ast.ContextText)}, ast.ContextText)
	// err := RenderTree(ioutil.Discard, tree, Vars{"a": RenderError{}}, true)
	// if err == nil {
	// 	t.Errorf("expecting not nil error\n")
	// } else if err.Error() != "RenderTree error" {
	// 	t.Errorf("unexpected error %q, expecting 'RenderTree error'\n", err)
	// }

	// err = RenderTree(ioutil.Discard, tree, Vars{"a": RenderPanic{}}, true)
	// if err == nil {
	// 	t.Errorf("expecting not nil error\n")
	// } else if err.Error() != "RenderTree panic" {
	// 	t.Errorf("unexpected error %q, expecting 'RenderTree panic'\n", err)
	// }
}

type stringConvertible string

type aMap struct {
	v string
	H func() string `scriggo:"G"`
}

func (s aMap) F() string {
	return s.v
}

func (s aMap) G() string {
	return s.v
}

type aStruct struct {
	a string
	B string `scriggo:"b"`
	C string
}

var templateMultiPageCases = map[string]struct {
	sources    map[string]string
	expected   string                 // default to ""
	main       *scriggo.MapPackage    // default to nil
	vars       map[string]interface{} // default to nil
	lang       Language               // default to LanguageText
	entryPoint string                 // default to "index.html"
}{

	"Empty template": {
		sources: map[string]string{
			"index.html": ``,
		},
	},
	"Text only": {
		sources: map[string]string{
			"index.html": `Hello, world!`,
		},
		expected: `Hello, world!`,
	},

	"Template comments": {
		sources: map[string]string{
			"index.html": `{# this is a comment #}`,
		},
		expected: ``,
	},

	"Template comments with text": {
		sources: map[string]string{
			"index.html": `Text before comment{# comment #} text after comment{# another comment #}`,
		},
		expected: `Text before comment text after comment`,
	},

	"'Show' node only": {
		sources: map[string]string{
			"index.html": `{{ "i am a show" }}`,
		},
		expected: `i am a show`,
	},

	"Text and show": {
		sources: map[string]string{
			"index.html": `Hello, {{ "world" }}!!`,
		},
		expected: `Hello, world!!`,
	},

	"If statements - true": {
		sources: map[string]string{
			"index.html": `{% if true %}true{% else %}false{% end %}`,
		},
		expected: `true`,
	},

	"If statements - false": {
		sources: map[string]string{
			"index.html": `{% if !true %}true{% else %}false{% end %}`,
		},
		expected: `false`,
	},

	"Variable declarations": {
		sources: map[string]string{
			"index.html": `{% var a = 10 %}{% var b = 20 %}{{ a + b }}`,
		},
		expected: "30",
	},

	"For loop": {
		sources: map[string]string{
			"index.html": "For loop: {% for i := 0; i < 5; i++ %}{{ i }}, {% end %}",
		},
		expected: "For loop: 0, 1, 2, 3, 4, ",
	},

	"Template builtin - max": {
		sources: map[string]string{
			"index.html": `Maximum between 10 and -3 is {{ max(10, -3) }}`,
		},
		expected: `Maximum between 10 and -3 is 10`,
	},

	"Template builtin - sort": {
		sources: map[string]string{
			"index.html": `{% s := []string{"a", "c", "b"} %}{{ sprint(s) }} sorted is {% sort(s) %}{{ sprint(s) }}`,
		},
		expected: `[a c b] sorted is [a b c]`,
	},

	"Function call": {
		sources: map[string]string{
			"index.html": `{% func() { print(5) }() %}`,
		},
		expected: `5`,
	},

	"Multi rows": {
		sources: map[string]string{
			"index.html": `{%
	print(3) %}`,
		},
		expected: `3`,
	},

	"Multi rows 2": {
		sources: map[string]string{
			"index.html": `{%
	print(3)
%}`,
		},
		expected: `3`,
	},

	"Multi rows with comments": {
		sources: map[string]string{
			"index.html": `{%
// pre comment
/* pre comment */
	print(3)
/* post comment */
// post comment

%}`,
		},
		expected: `3`,
	},

	"Using a function declared in main": {
		sources: map[string]string{
			"index.html": `calling f: {{ f() }}, done!`,
		},
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"f": func() string { return "i'm f!" },
			},
		},
		expected: `calling f: i'm f!, done!`,
	},

	"Reading a variable declared in main": {
		sources: map[string]string{
			"index.html": `{{ mainVar }}`,
		},
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"mainVar": (*int)(nil),
			},
		},
		expected: `0`,
	},

	"Reading a variable declared in main and initialized with vars": {
		sources: map[string]string{
			"index.html": `{{ initMainVar }}`,
		},
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"initMainVar": (*int)(nil),
			},
		},
		vars: map[string]interface{}{
			"initMainVar": 42,
		},
		expected: `42`,
	},

	"Calling a builtin function": {
		sources: map[string]string{
			"index.html": `{{ lowercase("HellO ScrIgGo!") }}{% x := "A String" %}{{ lowercase(x) }}`,
		},
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"lowercase": func(s string) string {
					return strings.ToLower(s)
				},
			},
		},
		expected: `hello scriggo!a string`,
	},

	"Calling a function stored in a builtin variable": {
		sources: map[string]string{
			"index.html": `{{ lowercase("HellO ScrIgGo!") }}{% x := "A String" %}{{ lowercase(x) }}`,
		},
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"lowercase": (*func(string) string)(nil),
			},
		},
		vars: map[string]interface{}{
			"lowercase": func(s string) string {
				return strings.ToLower(s)
			},
		},
		expected: `hello scriggo!a string`,
	},

	"https://github.com/open2b/scriggo/issues/391": {
		sources: map[string]string{
			"index.html": `{{ a }}{{ b }}`,
		},
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"a": (*string)(nil),
				"b": (*string)(nil),
			},
		},
		vars: map[string]interface{}{
			"a": "AAA",
			"b": "BBB",
		},
		expected: `AAABBB`,
	},

	"Macro definition (no arguments)": {
		sources: map[string]string{
			"index.html": `Macro def: {% macro M %}M's body{% end %}end.`,
		},
		expected: `Macro def: end.`,
	},

	"Macro definition (no arguments) and show-macro": {
		sources: map[string]string{
			"index.html": `{% macro M %}body{% end %}{% show M %}`,
		},
		expected: `body`,
	},

	"Macro definition (with arguments)": {
		sources: map[string]string{
			"index.html": `{% macro M(v int) %}v is {{ v }}{% end %}`,
		},
	},

	"Macro definition (with one string argument) and show-macro": {
		sources: map[string]string{
			"index.html": `{% macro M(v string) %}v is {{ v }}{% end %}{% show M("msg") %}`,
		},
		expected: `v is msg`,
	},

	"Macro definition (with two string arguments) and show-macro": {
		sources: map[string]string{
			"index.html": `{% macro M(a, b string) %}a is {{ a }} and b is {{ b }}{% end %}{% show M("avalue", "bvalue") %}`,
		},
		expected: `a is avalue and b is bvalue`,
	},

	"Macro definition (with one int argument) and show-macro": {
		sources: map[string]string{
			"index.html": `{% macro M(v int) %}v is {{ v }}{% end %}{% show M(42) %}`,
		},
		expected: `v is 42`,
	},

	"Macro definition (with one []int argument) and show-macro": {
		sources: map[string]string{
			"index.html": `{% macro M(v []int) %}v is {{ sprint(v) }}{% end %}{% show M([]int{42}) %}`,
		},
		expected: `v is [42]`,
	},

	"Two macro definitions": {
		sources: map[string]string{
			"index.html": `{% macro M1 %}M1's body{% end %}{% macro M2(i int, s string) %}i: {{ i }}, s: {{ s }}{% end %}`,
		},
	},

	"Two macro definitions and three show-macro": {
		sources: map[string]string{
			"index.html": `{% macro M1 %}M1's body{% end %}{% macro M2(i int, s string) %}i: {{ i }}, s: {{ s }}{% end %}Show macro: {% show M1 %} {% show M2(-30, "hello") %} ... {% show M1 %}`,
		},
		expected: `Show macro: M1's body i: -30, s: hello ... M1's body`,
	},

	"Macro definition and show-macro without parameters": {
		sources: map[string]string{
			"index.html": `{% macro M %}ok{% end %}{% show M() %}`,
		},
		expected: `ok`,
	},

	"Macro definition and show-macro without parentheses": {
		sources: map[string]string{
			"index.html": `{% macro M %}ok{% end %}{% show M %}`,
		},
		expected: `ok`,
	},

	"Macro definition and show-macro variadic": {
		sources: map[string]string{
			"index.html": `{% macro M(v ...int) %}{% for _ , i := range v %}{{ i }}{% end for %}{% end macro %}{% show M([]int{1,2,3}...) %}`,
		},
		expected: `123`,
	},

	"Template builtin - title": {
		sources: map[string]string{
			"index.html": `{% s := "hello, world" %}{{ s }} converted to title is {{ title(s) }}`,
		},
		expected: `hello, world converted to title is Hello, World`,
	},

	"Label for": {
		sources: map[string]string{
			"index.html": `{% L: for %}a{% break L %}b{% end for %}`,
		},
		expected: `a`,
	},

	"Label switch": {
		sources: map[string]string{
			"index.html": `{% L: switch 1 %}{% case 1 %}a{% break L %}b{% end switch %}`,
		},
		expected: `a`,
	},

	"ShowMacro of a not-defined macro with 'or ignore' option": {
		sources: map[string]string{
			"index.html": `Ignored macro: {% show M or ignore %} ok.`,
		},
		expected: `Ignored macro:  ok.`,
	},

	"Include - Only text": {
		sources: map[string]string{
			"index.html":    `a{% include "/included.html" %}c`,
			"included.html": `b`,
		},
		expected: "abc",
	},

	"Include - Included file uses external variable": {
		sources: map[string]string{
			"index.html":    `{% var a = 10 %}a: {% include "/included.html" %}`,
			"included.html": `{{ a }}`,
		},
		expected: "a: 10",
	},

	"Include - File including uses included variable": {
		sources: map[string]string{
			"index.html":    `{% include "/included.html" %}included a: {{ a }}`,
			"included.html": `{% var a = 20 %}`,
		},
		expected: "included a: 20",
	},

	"Include - Including a file which includes another file": {
		sources: map[string]string{
			"index.html":              `indexstart,{% include "/dir1/included.html" %}indexend,`,
			"dir1/included.html":      `i1start,{% include "/dir1/dir2/included.html" %}i1end,`,
			"dir1/dir2/included.html": `i2,`,
		},
		expected: "indexstart,i1start,i2,i1end,indexend,",
	},

	"Import/Macro - Importing a macro defined in another page": {
		sources: map[string]string{
			"index.html": `{% import "/page.html" %}{% show M %}{% show M %}`,
			"page.html":  `{% macro M %}macro!{% end %}{% macro M2 %}macro 2!{% end %}`,
		},
		expected: "macro!macro!",
	},

	"Import/Macro - Importing a macro defined in another page, where a function calls a before-declared function": {
		sources: map[string]string{
			"index.html": `{% import "/page.html" %}{% show M %}{% show M %}`,
			"page.html": `
				{% macro M2 %}macro 2!{% end %}
				{% macro M %}{% show M2 %}{% end %}
			`,
		},
		expected: "macro 2!macro 2!",
	},

	"Import/Macro - Importing a macro defined in another page, where a function calls an after-declared function": {
		sources: map[string]string{
			"index.html": `{% import "/page.html" %}{% show M %}{% show M %}`,
			"page.html": `
				{% macro M %}{% show M2 %}{% end %}
				{% macro M2 %}macro 2!{% end %}
			`,
		},
		expected: "macro 2!macro 2!",
	},

	"Import/Macro - Importing a macro defined in another page, which imports a third page": {
		sources: map[string]string{
			"index.html": `{% import "/page1.html" %}index-start,{% show M1 %}index-end`,
			"page1.html": `{% import "/page2.html" %}{% macro M1 %}M1-start,{% show M2 %}M1-end,{% end %}`,
			"page2.html": `{% macro M2 %}M2,{% end %}`,
		},
		expected: "index-start,M1-start,M2,M1-end,index-end",
	},

	"Import/Macro - Importing a macro using an import statement with identifier": {
		sources: map[string]string{
			"index.html": `{% import pg "/page.html" %}{% show pg.M %}{% show pg.M %}`,
			"page.html":  `{% macro M %}macro!{% end %}`,
		},
		expected: "macro!macro!",
	},

	"Import/Macro - Importing a macro using an import statement with identifier (with comments)": {
		sources: map[string]string{
			"index.html": `{# a comment #}{% import pg "/page.html" %}{# a comment #}{% show pg.M %}{# a comment #}{% show pg.M %}{# a comment #}`,
			"page.html":  `{# a comment #}{% macro M %}{# a comment #}macro!{# a comment #}{% end %}{# a comment #}`,
		},
		expected: "macro!macro!",
	},

	"Extends - Empty page extends a page containing only text": {
		sources: map[string]string{
			"index.html": `{% extends "/page.html" %}`,
			"page.html":  `I'm page!`,
		},
		expected: "I'm page!",
	},

	"Extends - Extending a page that calls a macro defined on current page": {
		sources: map[string]string{
			"index.html": `{% extends "/page.html" %}{% macro E %}E's body{% end %}`,
			"page.html":  `{% show E %}`,
		},
		expected: "E's body",
	},

	"Extending an empty page": {
		sources: map[string]string{
			"index.html":    `{% extends "extended.html" %}`,
			"extended.html": ``,
		},
	},

	"Extending a page that imports another file": {
		sources: map[string]string{
			"index.html":    `{% extends "/extended.html" %}`,
			"extended.html": `{% import "/imported.html" %}`,
			"imported.html": `{% macro Imported %}Imported macro{% end macro %}`,
		},
	},

	"Extending a page (that imports another file) while declaring a macro": {
		sources: map[string]string{
			"index.html":    `{% extends "/extended.html" %}{% macro Index %}{% end macro %}`,
			"extended.html": `{% import "/imported.html" %}`,
			"imported.html": `{% macro Imported %}Imported macro{% end macro %}`,
		},
	},

	"Extends - Extending a page that calls two macros defined on current page": {
		sources: map[string]string{
			"index.html": `{% extends "/page.html" %}{% macro E1 %}E1's body{% end %}{% macro E2 %}E2's body{% end %}`,
			"page.html":  `{% show E1 %}{% show E2 %}`,
		},
		expected: "E1's bodyE2's body",
	},

	"Extends - Define a variable (with zero value) used in macro definition": {
		sources: map[string]string{
			"index.html": `{% extends "/page.html" %}{% var Local int %}{% macro E1 %}Local has value {{ Local }}{% end %}`,
			"page.html":  `{% show E1 %}`,
		},
		expected: "Local has value 0",
	},

	"Extends - Define a variable (with non-zero value) used in macro definition": {
		sources: map[string]string{
			"index.html": `{% extends "/page.html" %}{% var Local = 50 %}{% macro E1 %}Local has value {{ Local }}{% end %}`,
			"page.html":  `{% show E1 %}`,
		},
		expected: "Local has value 50",
	},

	"Extends - Extending a file which contains text and shows": {
		sources: map[string]string{
			"index.html": `{% extends "/page.html" %}`,
			"page.html":  `I am an {{ "extended" }} file.`,
		},
		expected: "I am an extended file.",
	},

	"File imported twice": {
		sources: map[string]string{
			"index.html": `{% import "/a.html" %}{% import "/b.html" %}`,
			"a.html":     `{% import "/b.html" %}`,
			"b.html":     `{% macro M %}I'm b{% end %}`,
		},
	},

	"https://github.com/open2b/scriggo/issues/392": {
		sources: map[string]string{
			"product.html": `{{ "" }}{% include "includes/products.html" %}
`, // this newline is intentional
			"includes/products.html": `{% macro M(s []int) %}{% end %}`,
		},
		expected:   "\n",
		lang:       LanguageHTML,
		entryPoint: "product.html",
	},

	"https://github.com/open2b/scriggo/issues/392 (minimal)": {
		sources: map[string]string{
			"index.html": `text{% macro M(s []int) %}{% end %}text`,
		},
		expected: `texttext`,
		lang:     LanguageHTML,
	},

	"https://github.com/open2b/scriggo/issues/392 (invalid memory address)": {
		sources: map[string]string{
			"index.html": `{% macro M(s []int) %}{% end %}text`,
		},
		expected: `text`,
		lang:     LanguageHTML,
	},

	"https://github.com/open2b/scriggo/issues/393": {
		sources: map[string]string{
			"product.html": `{% include "includes/products.html" %}
`, // this newline is intentional
			"includes/products.html": `{% macro M(s []int) %}{% end %}`,
		},
		expected:   "",
		lang:       LanguageHTML,
		entryPoint: "product.html",
	},
	"Auto imported packages - Function call": {
		sources: map[string]string{
			"index.html": `{{ strings.ToLower("HELLO") }}`,
		},
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"strings": &scriggo.MapPackage{
					PkgName: "strings",
					Declarations: map[string]interface{}{
						"ToLower": strings.ToLower,
					},
				},
			},
		},
		expected: "hello",
	},
	"Auto imported packages - Variable": {
		sources: map[string]string{
			"index.html": `{{ data.Name }} Holmes`,
		},
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"data": &scriggo.MapPackage{
					PkgName: "data",
					Declarations: map[string]interface{}{
						"Name": &[]string{"Sherlock"}[0],
					},
				},
			},
		},
		expected: "Sherlock Holmes",
	},
	"Auto imported packages - Type": {
		sources: map[string]string{
			"index.html": `{% b := &bytes.Buffer{} %}{% b.WriteString("oh!") %}{{ b.String() }}`,
		},
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"bytes": &scriggo.MapPackage{
					PkgName: "bytes",
					Declarations: map[string]interface{}{
						"Buffer": reflect.TypeOf(bytes.Buffer{}),
					},
				},
			},
		},
		expected: "oh!",
	},
	"Auto imported packages - Constants": {
		sources: map[string]string{
			"index.html": `{{ math.MaxInt8 }}`,
		},
		main: &scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"math": &scriggo.MapPackage{
					PkgName: "math",
					Declarations: map[string]interface{}{
						"MaxInt8": math.MaxInt8,
					},
				},
			},
		},
		expected: "127",
	},
}

func TestMultiPageTemplate(t *testing.T) {
	for name, cas := range templateMultiPageCases {
		t.Run(name, func(t *testing.T) {
			r := MapReader{}
			for p, src := range cas.sources {
				r[p] = []byte(src)
			}
			builtins := Builtins()
			if cas.main != nil {
				for k, v := range cas.main.Declarations {
					builtins[k] = v
				}
			}
			entryPoint := cas.entryPoint
			if entryPoint == "" {
				entryPoint = "index.html"
			}
			templ, err := Load(entryPoint, r, builtins, cas.lang, nil)
			if err != nil {
				t.Fatalf("loading error: %s", err)
			}
			w := &bytes.Buffer{}
			err = templ.Render(w, cas.vars, &RenderOptions{PrintFunc: scriggo.PrintFunc(w)})
			if err != nil {
				t.Fatalf("rendering error: %s", err)
			}
			if cas.expected != w.String() {
				t.Fatalf("expecting %q, got %q", cas.expected, w.String())
			}
		})
	}
}

func TestVars(t *testing.T) {
	var a int
	var b int
	var c int
	var d = func() {}
	var e = func() {}
	var f = 5
	var g = 7
	reader := MapReader{"example.txt": []byte(`{% _, _, _, _, _ = a, c, d, e, f %}`)}
	builtins := Declarations{
		"a": &a, // expected
		"b": &b,
		"c": c,
		"d": &d, // expected
		"e": &e, // expected
		"f": f,
		"g": g,
	}
	tmpl, err := Load("example.txt", reader, builtins, LanguageText, nil)
	if err != nil {
		t.Fatal(err)
	}
	vars := tmpl.Vars()
	if len(vars) != 3 {
		t.Fatalf("expecting 3 variable names, got %d", len(vars))
	}
	for _, v := range vars {
		switch v {
		case "a", "d", "e":
		default:
			t.Fatalf("expecting variable name \"a\", \"d\" or \"e\", got %q", v)
		}
	}
}

var envFilePathCases = []struct {
	name    string
	sources map[string]string
	want    string
}{

	{
		name: "Just one file",
		sources: map[string]string{
			"index.html": `{{ path() }}`,
		},
		want: "/index.html",
	},

	{
		name: "File including another file",
		sources: map[string]string{
			"index.html":    `{{ path() }}, {% include "included.html"%}, {{ path() }}`,
			"included.html": `{{ path() }}`,
		},
		want: `/index.html, /included.html, /index.html`,
	},

	{
		name: "File including another file in a sub-directory",
		sources: map[string]string{
			"index.html":              `{{ path() }}, {% include "includes/included1.html"%}, {{ path() }}`,
			"includes/included1.html": `{{ path() }}, {% include "included2.html" %}`,
			"includes/included2.html": `{{ path() }}`,
		},
		want: `/index.html, /includes/included1.html, /includes/included2.html, /index.html`,
	},

	{
		name: "File importing another file, which defines a macro",
		sources: map[string]string{
			"index.html":    `{% import "imported.html" %}{{ path() }}, {% show Path %}, {{ path() }}`,
			"imported.html": `{% macro Path %}{{ path() }}{% end %}`,
		},
		want: `/index.html, /imported.html, /index.html`,
	},

	{
		name: "File extending another file",
		sources: map[string]string{
			"index.html":    `{% extends "extended.html" %}{% macro Path %}{{ path() }}{% end %}`,
			"extended.html": `{{ path() }}, {% show Path %}`,
		},
		// X MARCO: extend paths does not have a leading /, while
		// imported/included paths have.
		want: `extended.html, /index.html`,
	},
}

func Test_envFilePath(t *testing.T) {
	builtins := Declarations{
		"path": func(env *runtime.Env) string { return env.FilePath() },
	}
	for _, cas := range envFilePathCases {
		t.Run(cas.name, func(t *testing.T) {
			r := MapReader{}
			for p, src := range cas.sources {
				r[p] = []byte(src)
			}
			template, err := Load("index.html", r, builtins, LanguageHTML, nil)
			if err != nil {
				t.Fatal(err)
			}
			w := &bytes.Buffer{}
			err = template.Render(w, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(cas.want, w.String()); diff != "" {
				t.Fatalf("(-want, +got):\n%s", diff)
			}
		})
	}
}
