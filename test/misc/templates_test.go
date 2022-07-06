// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package misc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/fstest"
	"github.com/open2b/scriggo/native"

	"github.com/google/go-cmp/cmp"
)

func globals() native.Declarations {
	var I = 5
	return native.Declarations{
		"max": func(x, y int) int {
			if x < y {
				return y
			}
			return x
		},
		"sprint": func(a ...interface{}) string {
			return fmt.Sprint(a...)
		},
		"sprintf": func(format string, a ...interface{}) string {
			return fmt.Sprintf(format, a...)
		},
		"title": func(env native.Env, s string) string {
			return strings.Title(s)
		},
		"I": &I,
		"C": 8,
	}
}

var rendererExprTests = []struct {
	src      string
	expected string
	globals  native.Declarations
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
	// {"[]interface{}{v}", "", native.Declarations{"v": []string(nil)}},
	// {"len([]interface{}{v})", "1", native.Declarations{"v": []string(nil)}},
	// {"[]interface{}{v, v2}", ", ", native.Declarations{"v": []string(nil), "v2": []string(nil)}},
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
			fsys := fstest.Files{"index.html": "{{" + cas.src + "}}"}
			template, err := scriggo.BuildTemplate(fsys, "index.html", nil)
			if err != nil {
				t.Fatalf("source %q: build error: %s", cas.src, err)
			}
			b := &bytes.Buffer{}
			err = template.Run(b, nil, nil)
			if err != nil {
				t.Fatalf("source %q: run error: %s", cas.src, err)
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
	{"{% a := \"hi\" %}{% if a := 2; a == 3 %}{% else if a := false; a %}{% else %}{{ a }}{% end %}, {{ a }}", "false, hi", nil}, // https://go.dev/play/p/2OXyyKwCfS8
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
			fsys := fstest.Files{"index.html": cas.src}
			template, err := scriggo.BuildTemplate(fsys, "index.html", nil)
			if err != nil {
				t.Fatalf("source %q: build error: %s", cas.src, err)
			}
			b := &bytes.Buffer{}
			err = template.Run(b, nil, nil)
			if err != nil {
				t.Fatalf("source %q: run error: %s", cas.src, err)
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

var templateMultiFileCases = map[string]struct {
	sources          fstest.Files
	expectedBuildErr string                 // default to empty string (no build error). Mutually exclusive with expectedOut.
	expectedOut      string                 // default to "". Mutually exclusive with expectedBuildErr.
	main             native.Package         // default to nil
	vars             map[string]interface{} // default to nil
	entryPoint       string                 // default to "index.html"
	importer         native.Importer        // default to nil
	noParseShow      bool
}{

	"Empty template": {
		sources: fstest.Files{
			"index.txt": ``,
		},
	},
	"Text only": {
		sources: fstest.Files{
			"index.txt": `Hello, world!`,
		},
		expectedOut: `Hello, world!`,
	},

	"Template comments": {
		sources: fstest.Files{
			"index.txt": `{# this is a comment #}`,
		},
		expectedOut: ``,
	},

	"Template comments with text": {
		sources: fstest.Files{
			"index.txt": `Text before comment{# comment #} text after comment{# another comment #}`,
		},
		expectedOut: `Text before comment text after comment`,
	},

	"'Show' node only": {
		sources: fstest.Files{
			"index.txt": `{{ "i am a show" }}`,
		},
		expectedOut: `i am a show`,
	},

	"Text and show": {
		sources: fstest.Files{
			"index.txt": `Hello, {{ "world" }}!!`,
		},
		expectedOut: `Hello, world!!`,
	},

	"If statements - true": {
		sources: fstest.Files{
			"index.txt": `{% if true %}true{% else %}false{% end %}`,
		},
		expectedOut: `true`,
	},

	"If statements - false": {
		sources: fstest.Files{
			"index.txt": `{% if !true %}true{% else %}false{% end %}`,
		},
		expectedOut: `false`,
	},

	"Variable declarations": {
		sources: fstest.Files{
			"index.txt": `{% var a = 10 %}{% var b = 20 %}{{ a + b }}`,
		},
		expectedOut: "30",
	},

	"For loop": {
		sources: fstest.Files{
			"index.txt": "For loop: {% for i := 0; i < 5; i++ %}{{ i }}, {% end %}",
		},
		expectedOut: "For loop: 0, 1, 2, 3, 4, ",
	},

	"Template global - max": {
		sources: fstest.Files{
			"index.txt": `Maximum between 10 and -3 is {{ max(10, -3) }}`,
		},
		expectedOut: `Maximum between 10 and -3 is 10`,
	},

	"Function literal": {
		sources: fstest.Files{
			"index.txt": `{% func() {} %}`,
		},
		expectedBuildErr: "func literal evaluated but not used",
	},

	"Function call": {
		sources: fstest.Files{
			"index.txt": `{% func() { print(5) }() %}`,
		},
		expectedOut: `5`,
	},

	"Multi rows": {
		sources: fstest.Files{
			"index.txt": `{%
	print(3) %}`,
		},
		expectedOut: `3`,
	},

	"Multi rows 2": {
		sources: fstest.Files{
			"index.txt": `{%
	print(3)
%}`,
		},
		expectedOut: `3`,
	},

	"Multi rows with comments": {
		sources: fstest.Files{
			"index.txt": `{%
// pre comment
/* pre comment */
	print(3)
/* post comment */
// post comment

%}`,
		},
		expectedOut: `3`,
	},

	"Using a function declared in main": {
		sources: fstest.Files{
			"index.txt": `calling f: {{ f() }}, done!`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"f": func() string { return "i'm f!" },
			},
		},
		expectedOut: `calling f: i'm f!, done!`,
	},

	"Reading a variable declared in main": {
		sources: fstest.Files{
			"index.txt": `{{ mainVar }}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"mainVar": (*int)(nil),
			},
		},
		expectedOut: `0`,
	},

	"Reading a variable declared in main and initialized with vars": {
		sources: fstest.Files{
			"index.txt": `{{ initMainVar }}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"initMainVar": (*int)(nil),
			},
		},
		vars: map[string]interface{}{
			"initMainVar": 42,
		},
		expectedOut: `42`,
	},

	"Calling a global function": {
		sources: fstest.Files{
			"index.txt": `{{ lowercase("HellO ScrIgGo!") }}{% x := "A String" %}{{ lowercase(x) }}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"lowercase": func(s string) string {
					return strings.ToLower(s)
				},
			},
		},
		expectedOut: `hello scriggo!a string`,
	},

	"Calling a function stored in a global variable": {
		sources: fstest.Files{
			"index.txt": `{{ lowercase("HellO ScrIgGo!") }}{% x := "A String" %}{{ lowercase(x) }}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"lowercase": (*func(string) string)(nil),
			},
		},
		vars: map[string]interface{}{
			"lowercase": func(s string) string {
				return strings.ToLower(s)
			},
		},
		expectedOut: `hello scriggo!a string`,
	},

	"https://github.com/open2b/scriggo/issues/391": {
		sources: fstest.Files{
			"index.txt": `{{ a }}{{ b }}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"a": (*string)(nil),
				"b": (*string)(nil),
			},
		},
		vars: map[string]interface{}{
			"a": "AAA",
			"b": "BBB",
		},
		expectedOut: `AAABBB`,
	},

	"Macro definition (no arguments)": {
		sources: fstest.Files{
			"index.txt": `Macro def: {% macro M %}M's body{% end %}end.`,
		},
		expectedOut: `Macro def: end.`,
	},

	"Macro definition (no arguments) and show-macro": {
		sources: fstest.Files{
			"index.txt": `{% macro M %}body{% end %}{% show M() %}`,
		},
		expectedOut: `body`,
	},

	"Macro definition (with arguments)": {
		sources: fstest.Files{
			"index.txt": `{% macro M(v int) %}v is {{ v }}{% end %}`,
		},
	},

	"Macro definition (with one string argument) and show-macro": {
		sources: fstest.Files{
			"index.txt": `{% macro M(v string) %}v is {{ v }}{% end %}{% show M("msg") %}`,
		},
		expectedOut: `v is msg`,
	},

	"Macro definition (with two string arguments) and show-macro": {
		sources: fstest.Files{
			"index.txt": `{% macro M(a, b string) %}a is {{ a }} and b is {{ b }}{% end %}{% show M("avalue", "bvalue") %}`,
		},
		expectedOut: `a is avalue and b is bvalue`,
	},

	"Macro definition (with one int argument) and show-macro": {
		sources: fstest.Files{
			"index.txt": `{% macro M(v int) %}v is {{ v }}{% end %}{% show M(42) %}`,
		},
		expectedOut: `v is 42`,
	},

	"Macro definition (with one []int argument) and show-macro": {
		sources: fstest.Files{
			"index.txt": `{% macro M(v []int) %}v is {{ sprint(v) }}{% end %}{% show M([]int{42}) %}`,
		},
		expectedOut: `v is [42]`,
	},

	"Two macro definitions": {
		sources: fstest.Files{
			"index.txt": `{% macro M1 %}M1's body{% end %}{% macro M2(i int, s string) %}i: {{ i }}, s: {{ s }}{% end %}`,
		},
	},

	"Two macro definitions and three show-macro": {
		sources: fstest.Files{
			"index.txt": `{% macro M1 %}M1's body{% end %}{% macro M2(i int, s string) %}i: {{ i }}, s: {{ s }}{% end %}Show macro: {% show M1() %} {% show M2(-30, "hello") %} ... {% show M1() %}`,
		},
		expectedOut: `Show macro: M1's body i: -30, s: hello ... M1's body`,
	},

	"Macro definition and show-macro without parameters": {
		sources: fstest.Files{
			"index.txt": `{% macro M %}ok{% end %}{% show M() %}`,
		},
		expectedOut: `ok`,
	},

	"Macro definition and show-macro without parentheses": {
		sources: fstest.Files{
			"index.txt": `{% macro M %}ok{% end %}{% show M() %}`,
		},
		expectedOut: `ok`,
	},

	"Macro definition and show-macro variadic": {
		sources: fstest.Files{
			"index.txt": `{% macro M(v ...int) %}{% for _ , i := range v %}{{ i }}{% end for %}{% end macro %}{% show M([]int{1,2,3}...) %}`,
		},
		expectedOut: `123`,
	},

	"Template global - title": {
		sources: fstest.Files{
			"index.txt": `{% s := "hello, world" %}{{ s }} converted to title is {{ title(s) }}`,
		},
		expectedOut: `hello, world converted to title is Hello, World`,
	},

	"Label for": {
		sources: fstest.Files{
			"index.txt": `{% L: for %}a{% break L %}b{% end for %}`,
		},
		expectedOut: `a`,
	},

	"Label switch": {
		sources: fstest.Files{
			"index.txt": `{% L: switch 1 %}{% case 1 %}a{% break L %}b{% end switch %}`,
		},
		expectedOut: `a`,
	},

	"Render - Only text": {
		sources: fstest.Files{
			"index.txt":   `a{{ render "/partial.txt" }}c`,
			"partial.txt": `b`,
		},
		expectedOut: "abc",
	},

	"Render - Render file that uses external variable": {
		sources: fstest.Files{
			"index.txt":   `{% var a = 10 %}a: {{ render "/partial.txt" }}`,
			"partial.txt": `{{ a }}`,
		},
		expectedBuildErr: "undefined: a",
	},

	"Render - File with a render expression try to use a variable declared in the rendered file": {
		sources: fstest.Files{
			"index.txt":   `{{ render "/partial.txt" }}partial a: {{ a }}`,
			"partial.txt": `{% var a = 20 %}`,
		},
		expectedBuildErr: "undefined: a",
	},

	"Render - File renders a file which renders another file": {
		sources: fstest.Files{
			"index.txt":             `indexstart,{{ render "/dir1/partial.txt" }}indexend,`,
			"dir1/partial.txt":      `i1start,{{ render "/dir1/dir2/partial.txt" }}i1end,`,
			"dir1/dir2/partial.txt": `i2,`,
		},
		expectedOut: "indexstart,i1start,i2,i1end,indexend,",
	},

	"Import/Macro - Importing a macro defined in another file": {
		sources: fstest.Files{
			"index.txt": `{% import "/file.txt" %}{% show M() %}{% show M() %}`,
			"file.txt":  `{% macro M %}macro!{% end %}{% macro M2 %}macro 2!{% end %}`,
		},
		expectedOut: "macro!macro!",
	},

	"Import/Macro - Importing a macro defined in another file, where a function calls a before-declared function": {
		sources: fstest.Files{
			"index.txt": `{% import "/file.txt" %}{% show M() %}{% show M() %}`,
			"file.txt": `
				{% macro M2 %}macro 2!{% end %}
				{% macro M %}{% show M2() %}{% end %}
			`,
		},
		expectedOut: "macro 2!macro 2!",
	},

	"Import/Macro - Importing a macro defined in another file, where a function calls an after-declared function": {
		sources: fstest.Files{
			"index.txt": `{% import "/file.txt" %}{% show M() %}{% show M() %}`,
			"file.txt": `
				{% macro M %}{% show M2() %}{% end %}
				{% macro M2 %}macro 2!{% end %}
			`,
		},
		expectedOut: "macro 2!macro 2!",
	},

	"Import/Macro - Importing a macro defined in another file, which imports a third file": {
		sources: fstest.Files{
			"index.txt": `{% import "/file1.txt" %}index-start,{% show M1() %}index-end`,
			"file1.txt": `{% import "/file2.txt" %}{% macro M1 %}M1-start,{% show M2() %}M1-end,{% end %}`,
			"file2.txt": `{% macro M2 %}M2,{% end %}`,
		},
		expectedOut: "index-start,M1-start,M2,M1-end,index-end",
	},

	"Import/Macro - Importing a macro using an import statement with identifier": {
		sources: fstest.Files{
			"index.txt": `{% import pg "/file.txt" %}{% show pg.M() %}{% show pg.M() %}`,
			"file.txt":  `{% macro M %}macro!{% end %}`,
		},
		expectedOut: "macro!macro!",
	},

	"Import/Macro - Importing a macro using an import statement with identifier (with comments)": {
		sources: fstest.Files{
			"index.txt": `{# a comment #}{% import pg "/file.txt" %}{# a comment #}{% show pg.M() %}{# a comment #}{% show pg.M() %}{# a comment #}`,
			"file.txt":  `{# a comment #}{% macro M %}{# a comment #}macro!{# a comment #}{% end %}{# a comment #}`,
		},
		expectedOut: "macro!macro!",
	},

	"Extends - Empty file extends a file containing only text": {
		sources: fstest.Files{
			"index.txt": `{% extends "/file.txt" %}`,
			"file.txt":  `I'm file!`,
		},
		expectedOut: "I'm file!",
	},

	"Extends - Extending a file that calls a macro defined on current file": {
		sources: fstest.Files{
			"index.txt": `{% extends "/file.txt" %}{% macro E %}E's body{% end %}`,
			"file.txt":  `{% show E() %}`,
		},
		expectedOut: "E's body",
	},

	"Extending an empty file": {
		sources: fstest.Files{
			"index.txt":    `{% extends "extended.txt" %}`,
			"extended.txt": ``,
		},
	},

	"Extending a file that imports another file": {
		sources: fstest.Files{
			"index.txt":    `{% extends "/extended.txt" %}`,
			"extended.txt": `{% import "/imported.txt" %}`,
			"imported.txt": `{% macro Imported %}Imported macro{% end macro %}`,
		},
	},

	"Extending a file (that imports another file) while declaring a macro": {
		sources: fstest.Files{
			"index.txt":    `{% extends "/extended.txt" %}{% macro Index %}{% end macro %}`,
			"extended.txt": `{% import "/imported.txt" %}`,
			"imported.txt": `{% macro Imported %}Imported macro{% end macro %}`,
		},
	},

	"Extends - Extending a file that calls two macros defined on current file": {
		sources: fstest.Files{
			"index.txt": `{% extends "/file.txt" %}{% macro E1 %}E1's body{% end %}{% macro E2 %}E2's body{% end %}`,
			"file.txt":  `{% show E1() %}{% show E2() %}`,
		},
		expectedOut: "E1's bodyE2's body",
	},

	"Extends - Define a variable (with zero value) used in macro definition": {
		sources: fstest.Files{
			"index.txt": `{% extends "/file.txt" %}{% var Local int %}{% macro E1 %}Local has value {{ Local }}{% end %}`,
			"file.txt":  `{% show E1() %}`,
		},
		expectedOut: "Local has value 0",
	},

	"Extends - Define a variable (with non-zero value) used in macro definition": {
		sources: fstest.Files{
			"index.txt": `{% extends "/file.txt" %}{% var Local = 50 %}{% macro E1 %}Local has value {{ Local }}{% end %}`,
			"file.txt":  `{% show E1() %}`,
		},
		expectedOut: "Local has value 50",
	},

	"Extends - Extending a file which contains text and shows": {
		sources: fstest.Files{
			"index.txt": `{% extends "/file.txt" %}`,
			"file.txt":  `I am an {{ "extended" }} file.`,
		},
		expectedOut: "I am an extended file.",
	},

	"File imported twice": {
		sources: fstest.Files{
			"index.txt": `{% import "/a.txt" %}{% import "/b.txt" %}`,
			"a.txt":     `{% import "/b.txt" %}`,
			"b.txt":     `{% macro M %}I'm b{% end %}`,
		},
	},

	"File imported twice - Variable declaration": {
		sources: fstest.Files{
			"index.txt": `{% import "b.txt" %}{% import "c.txt" %}`,
			"b.txt":     `{% import "c.txt" %}`,
			"c.txt":     `{% var V int %}`,
		},
	},

	"https://github.com/open2b/scriggo/issues/392": {
		sources: fstest.Files{
			"product.html": `{{ "" }}{{ render "partials/products.html" }}
`, // this newline is intentional
			"partials/products.html": `{% macro M(s []int) %}{% end %}`,
		},
		expectedOut: "\n",
		entryPoint:  "product.html",
	},

	"https://github.com/open2b/scriggo/issues/392 (minimal)": {
		sources: fstest.Files{
			"index.html": `text{% macro M(s []int) %}{% end %}text`,
		},
		expectedOut: `texttext`,
	},

	"https://github.com/open2b/scriggo/issues/392 (invalid memory address)": {
		sources: fstest.Files{
			"index.html": `{% macro M(s []int) %}{% end %}text`,
		},
		expectedOut: `text`,
	},

	"https://github.com/open2b/scriggo/issues/393": {
		sources: fstest.Files{
			"product.html": `{{ render "partials/products.html" }}
`, // this newline is intentional
			"partials/products.html": `{% macro M(s []int) %}{% end %}`,
		},
		expectedOut: "",
		entryPoint:  "product.html",
	},
	"Auto imported packages - Function call": {
		sources: fstest.Files{
			"index.txt": `{{ strings.ToLower("HELLO") }}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"strings": native.Package{
					Name: "strings",
					Declarations: native.Declarations{
						"ToLower": strings.ToLower,
					},
				},
			},
		},
		expectedOut: "hello",
	},
	"Auto imported packages - Variable": {
		sources: fstest.Files{
			"index.txt": `{{ data.Name }} Holmes`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"data": native.Package{
					Name: "data",
					Declarations: native.Declarations{
						"Name": &[]string{"Sherlock"}[0],
					},
				},
			},
		},
		expectedOut: "Sherlock Holmes",
	},
	"Auto imported packages - Type": {
		sources: fstest.Files{
			"index.txt": `{% b := &bytes.Buffer{} %}{% b.WriteString("oh!") %}{{ b.String() }}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"bytes": native.Package{
					Name: "bytes",
					Declarations: native.Declarations{
						"Buffer": reflect.TypeOf(bytes.Buffer{}),
					},
				},
			},
		},
		expectedOut: "oh!",
	},
	"Auto imported packages - Constants": {
		sources: fstest.Files{
			"index.txt": `{{ math.MaxInt8 }}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"math": native.Package{
					Name: "math",
					Declarations: native.Declarations{
						"MaxInt8": math.MaxInt8,
					},
				},
			},
		},
		expectedOut: "127",
	},

	"Syntax {{ f() }} where 'f' returns a value and a nil error": {
		sources: fstest.Files{
			"index.txt": `{{ atoi("42") }}`,
		},
		main:        functionReturningErrorPackage,
		expectedOut: "42",
	},

	"Syntax {{ f() }} where 'f' returns a zero value and an error": {
		sources: fstest.Files{
			"index.txt": `{{ atoi("what?") }}`,
		},
		main:        functionReturningErrorPackage,
		expectedOut: "",
	},

	"Undefined variable error": {
		sources: fstest.Files{
			"index.txt": `Name is {{ name }}`,
		},
		expectedBuildErr: "undefined: name",
	},

	"Render file tries to overwrite a variable of the file that renders it": {
		// The emitter must use another scope when emitting a rendered file,
		// otherwise such file can overwrite the variables of the file that
		// renders it.
		sources: fstest.Files{
			"index.txt":   `{% v := "showing" %}{{ render "partial.txt" }}{{ v }}`,
			"partial.txt": `{% v := "partial" %}`,
		},
		expectedOut: "showing",
	},

	"The partial file must see the global variable 'v', not the local variable 'v' of the file that renders it": {
		// If the partial file refers to a global symbol with the same name of a
		// local variable in the scope of the file that renders it, then the
		// emitter emits the code for such variable instead of such global
		// variable. This happens because the emitter gives the precedence to
		// local variables respect to global variables. For this reason the
		// emitter must hide the scopes to the partial file (as the type checker
		// does).
		sources: fstest.Files{
			"index.txt":   `{% v := "showing" %}{{ render "partial.txt" }}, {{ v }}`,
			"partial.txt": "{{ v }}",
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"v": &globalVariable,
			},
		},
		expectedOut: "global variable, showing",
	},

	"A partial file defines a macro, which should not be accessible from the file that renders the partial": {
		sources: fstest.Files{
			"index.txt":   `{{ render "partial.txt" }}{% show MacroInRenderFile() %}`,
			"partial.txt": `{% macro MacroInRenderFile %}{% end macro %}`,
		},
		expectedBuildErr: "undefined: MacroInRenderFile",
	},

	"The file with a render expression defines a macro, which should not be accessible from the rendered file": {
		sources: fstest.Files{
			"index.txt":   `{% macro Macro %}{% end macro %}{{ render "partial.txt" }}`,
			"partial.txt": `{% show Macro() %}`,
		},
		expectedBuildErr: "undefined: Macro",
	},

	"Byte slices are rendered as they are in context HTML": {
		sources: fstest.Files{
			"index.html": `{{ sb1 }}{{ sb2 }}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"sb1": &[]byte{97, 98, 99},                      // abc
				"sb2": &[]byte{60, 104, 101, 108, 108, 111, 62}, // <hello>
			},
		},
		expectedOut: `abc<hello>`,
	},

	"Cannot show byte slices in text context": {
		sources: fstest.Files{
			"index.txt": `{{ sb1 }}{{ sb2 }}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"sb1": &[]byte{97, 98, 99},                      // abc
				"sb2": &[]byte{60, 104, 101, 108, 108, 111, 62}, // <hello>
			},
		},
		expectedBuildErr: `cannot show sb1 (cannot show type []uint8 as text)`,
	},

	"Using the precompiled package 'fmt'": {
		sources: fstest.Files{
			"index.txt": `{% import "fmt" %}{{ fmt.Sprint(10, 20) }}`,
		},
		importer:    testPackages,
		expectedOut: "10 20",
	},

	"Using the precompiled package 'fmt' from a file that extends another file": {
		sources: fstest.Files{
			"index.txt":    `{% extends "extended.txt" %}{% import "fmt" %}{% macro M %}{{ fmt.Sprint(321, 11) }}{% end macro %}`,
			"extended.txt": `{% show M() %}`,
		},
		importer:    testPackages,
		expectedOut: "321 11",
	},

	"Using the precompiled packages 'fmt' and 'math'": {
		sources: fstest.Files{
			"index.txt": `{% import "fmt" %}{% import m "math" %}{{ fmt.Sprint(-42, m.Abs(-42)) }}`,
		},
		importer:    testPackages,
		expectedOut: "-42 42",
	},

	"Importing the precompiled package 'fmt' with '.'": {
		sources: fstest.Files{
			"index.txt": `{% import . "fmt" %}{{ Sprint(50, 70) }}`,
		},
		importer:    testPackages,
		expectedOut: "50 70",
	},

	"Trying to import a precompiled package that is not available in the importer": {
		sources: fstest.Files{
			"index.txt": `{% import "mypackage" %}{{ mypackage.F() }}`,
		},
		importer:         testPackages,
		expectedBuildErr: "index.txt:1:11: cannot find package \"mypackage\"",
	},

	"Trying to access a precompiled function 'SuperPrint' that is not available in the package 'fmt'": {
		sources: fstest.Files{
			"index.txt": `{% import "fmt" %}{{ fmt.SuperPrint(42) }}`,
		},
		importer:         testPackages,
		expectedBuildErr: "index.txt:1:25: undefined: fmt.SuperPrint",
	},

	"Using the precompiled package 'fmt' without importing it returns an error": {
		sources: fstest.Files{
			"index.txt": `{{ fmt.Sprint(10, 20) }}`,
		},
		importer:         testPackages,
		expectedBuildErr: "index.txt:1:4: undefined: fmt",
	},

	"Check if a value that has a method 'IsTrue() bool' is true or not": {
		sources: fstest.Files{
			"index.txt": "{% if (True{true}) %}OK{% else %}BUG{% end %}\n" +
				"{% if (True{false}) %}BUG{% else %}OK{% end %}\n" +
				"{% if (struct{}{}) %}BUG{% else %}OK{% end %}\n" +
				"{% if (struct{Value int}{}) %}BUG{% else %}OK{% end %}\n" +
				"{% if (struct{Value int}{Value: 42}) %}OK{% else %}BUG{% end %}\n" +
				"{% if (TrueIf42{}) %}BUG{% else %}OK{% end %}\n" +
				"{% if (TrueIf42{Value: 42}) %}OK{% else %}BUG{% end %}\n" +
				"{% if (NotImplIsTrue{}) %}BUG{% else %}OK{% end %}",
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"True":          reflect.TypeOf((*testTrue)(nil)).Elem(),
				"TrueIf42":      reflect.TypeOf((*testTrueIf42)(nil)).Elem(),
				"NotImplIsTrue": reflect.TypeOf((*testNotImplementIsTrue)(nil)).Elem(),
			},
		},
		expectedOut: "OK\nOK\nOK\nOK\nOK\nOK\nOK\nOK",
	},

	// https://github.com/open2b/scriggo/issues/640
	"Importing a file that imports a file that declares a variable": {
		sources: fstest.Files{
			"index.html":     `{% import "imported1.html" %}`,
			"imported1.html": `{% import "imported2.html" %}`,
			"imported2.html": `{% var X = 0 %}`,
		},
	},

	// https://github.com/open2b/scriggo/issues/640
	"Importing a file that imports a file that declares a macro": {
		sources: fstest.Files{
			"index.html":     `{% import "imported1.html" %}{% show M1(42) %}`,
			"imported1.html": `{% import "imported2.html" %}{% macro M1(a int) %}{% show M2(a) %}{% end macro %}`,
			"imported2.html": `{% macro M2(b int) %}b is {{ b }}{% end macro %}`,
		},
		expectedOut: "b is 42",
	},

	// https://github.com/open2b/scriggo/issues/641
	"File imported by two sources - test compilation": {
		sources: fstest.Files{
			"index.html":   `{% import "/v.html" %}{{ render "/partial.html" }}`,
			"partial.html": `{% import "/v.html" %}`,
			"v.html":       `{% var V int %}`,
		},
	},

	// https://github.com/open2b/scriggo/issues/642
	"Macro imported twice - test compilation": {
		sources: fstest.Files{
			"index.html":    `{% import "/imported.html" %}{% import "/macro.html" %}{% show M() %}`,
			"imported.html": `{% import "/macro.html" %}`,
			"macro.html":    `{% macro M %}{% end macro %}`,
		},
	},

	// https://github.com/open2b/scriggo/issues/642
	"Macro imported twice - test output": {
		sources: fstest.Files{
			"index.html":    `{% import "/imported.html" %}{% import "/macro.html" %}{% show M(42) %}`,
			"imported.html": `{% import "/macro.html" %}`,
			"macro.html":    `{% macro M(a int) %}a is {{ a }}{% end macro %}`,
		},
		expectedOut: "a is 42",
	},

	// https://github.com/open2b/scriggo/issues/643
	"Invalid variable value when imported": {
		sources: fstest.Files{
			"index.html": `{% import "/v.html" %}{{ V }}`,
			"v.html":     `{% var V = 42 %}`,
		},
		expectedOut: "42",
	},

	// https://github.com/open2b/scriggo/issues/643
	"Invalid variable value with multiple imports": {
		sources: fstest.Files{
			"index.html":   `{% import "/v.html" %}{{ render "/partial.html" }}V is {{ V }}`,
			"partial.html": `{% import "/v.html" %}`,
			"v.html":       `{% var V = 42 %}`,
		},
		expectedOut: "V is 42",
	},

	// https://github.com/open2b/scriggo/issues/643
	"Init function called more than once": {
		sources: fstest.Files{
			"index.html":   `{% import "v.html" %}{{ render "/partial.html" }}{{ V }}`,
			"partial.html": `{% import "/v.html" %}`,
			"v.html":       `{% var V = GetValue() %}`,
		},
		expectedOut: "42",
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"GetValue": func() int {
					if testGetValueCalled {
						panic("already called!")
					}
					testGetValueCalled = true
					return 42
				},
			},
		},
	},

	"Can access to unexported struct field declared in the same file - struct literal": {
		sources: fstest.Files{
			"index.txt": `{% var s struct { a int } %}{% s.a = 42 %}{{ s.a }}
			{% s2 := &s %}{{ s2.a }}`,
		},
		expectedOut: "42\n\t\t\t42",
	},

	"Can access to unexported struct field declared in the same file - defined type": {
		sources: fstest.Files{
			"index.txt": `{% type t struct { a int } %}{% var s t %}{% s.a = 84 %}{{ s.a }}
			{% s2 := &s %}{{ s2.a }}`,
		},
		expectedOut: "84\n\t\t\t84",
	},

	"Cannot access to unexported struct fields of a precompiled value (struct)": {
		sources: fstest.Files{
			"index.txt": `{{ s.foo }}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"s": structWithUnexportedFields,
			},
		},
		expectedBuildErr: `s.foo undefined (cannot refer to unexported field or method foo)`,
	},

	"Cannot access to unexported struct fields of a precompiled value (*struct)": {
		sources: fstest.Files{
			"index.txt": `{{ s.foo }}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"s": &structWithUnexportedFields,
			},
		},
		expectedBuildErr: `s.foo undefined (cannot refer to unexported field or method foo)`,
	},

	"Cannot access to an unexported field declared in another file (struct)": {
		sources: fstest.Files{
			// Note the statement: {% type _ struct { bar int } %}: we try to
			// deceive the type checker into thinking that the type `struct {
			// field int }` can be fully accessed because is the same declared
			// in this package.
			"index.txt":    `{% import "imported.txt" %}{% type _ struct { bar int } %}{{ S.bar }}`,
			"imported.txt": `{% var S struct { bar int } %}`,
		},
		expectedBuildErr: `S.bar undefined (cannot refer to unexported field or method bar)`,
	},

	"Cannot access to an unexported field declared in another file (*struct)": {
		sources: fstest.Files{
			// Note the statement: {% type _ struct { bar int } %}: we try to
			// deceive the type checker into thinking that the type `struct {
			// field int }` can be fully accessed because is the same declared
			// in this package.
			"index.txt":    `{% import "imported.txt" %}{% type _ *struct { bar int } %}{{ S.bar }}`,
			"imported.txt": `{% var S *struct { bar int } %}`,
		},
		expectedBuildErr: `S.bar undefined (cannot refer to unexported field or method bar)`,
	},

	"Accessing global variable from macro's body": {
		sources: fstest.Files{
			"index.txt": `{% macro M %}{{ globalVariable }}{% end %}{% show M() %}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"globalVariable": &([]string{"<b>global</b>"}[0]),
			},
		},
		expectedOut: "<b>global</b>",
	},
	"Double type checking of render expression": {
		sources: fstest.Files{
			"index.txt":   `{{ render "/partial.txt" }}{{ render "/partial.txt" }}`,
			"partial.txt": `{% var v int %}`,
		},
	},
	"https://github.com/open2b/scriggo/issues/661": {
		sources: fstest.Files{
			"index.txt": `{% extends "extended.txt" %}
{% macro M %}
{{ render "/partial.txt" }}
{% end macro %}`,
			"extended.txt": `{{ render "/partial.txt" }}`,
			"partial.txt":  `{% var v int %}`,
		},
	},
	"https://github.com/open2b/scriggo/issues/660": {
		sources: fstest.Files{
			"index.txt":   `{% macro M() %}{{ render "partial.txt" }}{% end macro %}`,
			"partial.txt": `{% var v int %}{% _ = v %}`,
		},
	},

	// https://github.com/open2b/scriggo/issues/659
	"Accessing global variable from function literal's body": {
		sources: fstest.Files{
			"index.txt": `{%
				func(){
					_ = globalVariable
				}() 
			%}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"globalVariable": (*int)(nil),
			},
		},
	},

	// https://github.com/open2b/scriggo/issues/659
	"Accessing global variable from function literal's body - nested": {
		sources: fstest.Files{
			"index.txt": `{%
				func(){
					func() {
						func() {
							_ = globalVariable
						}()
					}()
				}()
			%}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"globalVariable": (*int)(nil),
			},
		},
	},

	"Macro declaration inside implicit blocks": {
		sources: fstest.Files{
			"index.txt": `
				{% macro M1 %}
					{% if true %}
						{% macro M2 %}m2{% end macro %}
						{% show M2() %}
					{% end %}
				{% end macro %}
				{% show M1() %}
			`,
		},
		expectedOut: "\n\t\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t\tm2\n\n\t\t\t",
	},

	"https://github.com/open2b/scriggo/issues/679 (1)": {
		sources: fstest.Files{
			"index.txt": `{% global := interface{}(global) %}ok`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"global": &[]string{"ciao"},
			},
		},
		expectedOut: "ok",
	},

	"https://github.com/open2b/scriggo/issues/679 (2)": {
		sources: fstest.Files{
			"index.txt": `{% var global = interface{}(global) %}ok`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"global": &[]string{},
			},
		},
		expectedOut: "ok",
	},

	"https://github.com/open2b/scriggo/issues/679 (3)": {
		sources: fstest.Files{
			"index.txt": `{% _ = []int{} %}{% global := interface{}(global) %}ok`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"global": &[]string{},
			},
		},
		expectedOut: "ok",
	},

	"Panic after importing file that declares a variable in general register (1)": {
		sources: fstest.Files{
			"index.txt":    `before{% import "imported.txt" %}after`,
			"imported.txt": `{% var a []int %}`,
		},
		expectedOut: "beforeafter",
	},

	"Panic after importing file that declares a variable in general register (2)": {
		sources: fstest.Files{
			"index.txt":     `a{% import "imported1.txt" %}{% import "imported2.txt" %}b`,
			"imported1.txt": `{% var X []int %}`,
			"imported2.txt": `{% var Y []string %}`,
		},
		expectedOut: "ab",
	},

	"https://github.com/open2b/scriggo/issues/686": {
		sources: fstest.Files{
			"index.txt":    `{% extends "extended.txt" %}{% var _ = interface{}(global) %}`,
			"extended.txt": `text`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"global": (*int)(nil),
			},
		},
		expectedOut: "text",
	},

	"https://github.com/open2b/scriggo/issues/687": {
		sources: fstest.Files{
			"index.html": `{% extends "extended.html" %}
			
				{% import "imported.html" %}`,

			"extended.html": `
				<head>
				<script>....
				{{ design.Base }}		
				{{ design.Open2b }}		
				fef`,

			"imported.html": `
				{% var f, _ = interface{}(filters).([]int) %}
			`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"design": &struct {
					Base   string
					Open2b string
				}{},
				"filters": &[]int{1, 2, 3},
			},
		},
		expectedOut: "\n\t\t\t\t<head>\n\t\t\t\t<script>....\n\t\t\t\t\"\"\t\t\n\t\t\t\t\"\"\t\t\n\t\t\t\tfef",
	},

	"https://github.com/open2b/scriggo/issues/655": {
		sources: fstest.Files{
			"index.html":  "{% extends \"layout.html\" %}\n{% var _ = func() { } %}",
			"layout.html": `<a href="a">`,
		},
		expectedOut: "<a href=\"a\">",
	},

	"https://github.com/open2b/scriggo/issues/656": {
		sources: fstest.Files{
			"index.txt":  "{% extends \"layout.txt\" %}\n{% var _ = func() { } %}",
			"layout.txt": `abc`,
		},
		expectedOut: "abc",
	},

	"Show of a previously imported file": {
		sources: fstest.Files{
			"index.txt": `{% import "file.txt" %}{{ render "file.txt" }}`,
			"file.txt":  ``,
		},
		expectedBuildErr: `syntax error: render of file imported at index.txt:1:11`,
	},

	"Show of a previously extended file": {
		sources: fstest.Files{
			"index.txt": `{% extends "file.txt" %}{% macro A %}{{ render "file.txt" }}{% end %}`,
			"file.txt":  ``,
		},
		expectedBuildErr: `syntax error: render of file extended at index.txt:1:4`,
	},

	"Import of a previously extended file": {
		sources: fstest.Files{
			"index.txt": `{% extends "file.txt" %}{% import "file.txt" %}`,
			"file.txt":  ``,
		},
		expectedBuildErr: `syntax error: import of file extended at index.txt:1:4`,
	},

	"Import of a partial file": {
		sources: fstest.Files{
			"index.txt": `{{ render "file1.txt" }}{{ render "file2.txt" }}`,
			"file1.txt": ``,
			"file2.txt": `{% import "file1.txt" %}`,
		},
		expectedBuildErr: `syntax error: import of file rendered at index.txt:1:4`,
	},

	"Not only spaces in a file that extends": {
		sources: fstest.Files{
			"index.txt":  "{% extends \"layout.html\" %}\n\n\n\tboo",
			"layout.txt": ``,
		},
		expectedBuildErr: "index.txt:4:2: syntax error: unexpected text in file with extends",
	},

	"Not only spaces in an imported file": {
		sources: fstest.Files{
			"index.txt":    `{% import "imported.txt" %}`,
			"imported.txt": `abc`,
		},
		expectedBuildErr: "syntax error: unexpected text in imported file",
	},

	"Extends preceded by not empty text": {
		sources: fstest.Files{
			"index.txt":  `abc{% extends "layout.txt" %}`,
			"layout.txt": ``,
		},
		expectedBuildErr: "syntax error: extends is not at the beginning of the file",
	},

	"Extends preceded by another statement": {
		sources: fstest.Files{
			"index.txt":  `{% var a = 5 %}{% extends "layout.txt" %}`,
			"layout.txt": ``,
		},
		expectedBuildErr: "syntax error: extends is not at the beginning of the file",
	},

	"Extends preceded by comment": {
		sources: fstest.Files{
			"index.txt":  `{# comment #}{% extends "layout.txt" %}`,
			"layout.txt": `abc`,
		},
		expectedOut: "abc",
	},

	"EOF after {%": {
		sources: fstest.Files{
			"index.txt": `{%`,
		},
		expectedBuildErr: "syntax error: unexpected EOF, expecting %}",
	},

	"EOF after {%%": {
		sources: fstest.Files{
			"index.txt": `{%%`,
		},
		expectedBuildErr: "syntax error: unexpected EOF, expecting %%}",
	},

	"EOF after {{": {
		sources: fstest.Files{
			"index.txt": `{{`,
		},
		expectedBuildErr: "syntax error: unexpected EOF, expecting }}",
	},

	"Multi line statements #1": {
		sources: fstest.Files{
			"index.txt": `{%%
				extends "extended.txt"
			%%}{% var x = I %}`,
			"extended.txt": ``,
		},
	},

	"Multi line statements #2": {
		sources: fstest.Files{
			"index.txt": `before{%%
	import "imported.txt"
	%%}after`,
			"imported.txt": `{%%
				var a []int
			%%}`,
		},
		expectedOut: "beforeafter",
	},

	"Multi line statements #3": {
		sources: fstest.Files{
			"index.txt":    `{%% import "imported.txt" %%}`,
			"imported.txt": `{% var x = I %}`,
		},
	},

	"Multi line statements #4": {
		sources: fstest.Files{
			"index.txt":    `{% import "imported.txt" %}`,
			"imported.txt": `{%% var x = I %%}`,
		},
	},

	"Multiline statements #5": {
		sources: fstest.Files{
			"index.txt":    `{%% extends "extended.txt" %%}{% import "fmt" %}{% macro M %}{{ fmt.Sprint(321, 11) }}{% end macro %}`,
			"extended.txt": `{% show M() %}`,
		},
		importer:    testPackages,
		expectedOut: "321 11",
	},

	"Multiline statements #6": {
		sources: fstest.Files{
			"index.txt": `{%%
				import "fmt"
				import m "math"
			%%}
			{{ fmt.Sprint(-42, m.Abs(-42)) }}`,
		},
		importer:    testPackages,
		expectedOut: "\t\t\t-42 42",
	},

	"Multi line statements #7": {
		sources: fstest.Files{
			"index.txt":    `{% import "imported.txt" %}{% type _ struct { bar int } %}{{ S.bar }}`,
			"imported.txt": `{%% var S struct { bar int } %%}`,
		},
		expectedBuildErr: `S.bar undefined (cannot refer to unexported field or method bar)`,
	},

	"Multi line statements #8": {
		sources: fstest.Files{
			"index.txt":    `{% import "imported.txt" %}{% type _ *struct { bar int } %}{{ S.bar }}`,
			"imported.txt": `{%% var S *struct { bar int } %%}`,
		},
		expectedBuildErr: `S.bar undefined (cannot refer to unexported field or method bar)`,
	},

	"https://github.com/open2b/scriggo/issues/694": {
		sources: fstest.Files{
			"index.txt":   `{{ render "partial.txt" }}`,
			"partial.txt": `{% var a int %}{% func() { a = 20 }() %}`,
		},
		expectedOut: ``,
	},

	"Error positioned in first non space character": {
		sources: fstest.Files{
			"index.txt":    `{% import "imported.txt" %}`,
			"imported.txt": "\n \n\té",
		},
		expectedBuildErr: `3:2: syntax error: unexpected text in imported file`,
	},

	"Show a Scriggo defined type value": {
		sources: fstest.Files{
			"index.txt": `{% type Bool bool %}{{ Bool(true) }}`,
		},
		expectedOut: `true`,
	},

	"https://github.com/open2b/scriggo/issues/708 (1)": {
		sources: fstest.Files{
			"index.txt":    `{% extends "extended.txt" %}{% macro M %}{%% a := 10 %%}{% end macro %}`,
			"extended.txt": `a`,
		},
		expectedOut: `a`,
	},

	"https://github.com/open2b/scriggo/issues/708 (2)": {
		sources: fstest.Files{
			"index.txt":    `{% import "imported.txt" %}`,
			"imported.txt": `{% macro M %}{%% a := 20 %%}{% end %}`,
		},
	},

	"Distraction free macro declaration": {
		sources: fstest.Files{
			"index.html":  `{% extends "layout.html" %}{% Article %}content`,
			"layout.html": `{% show Article() %}`,
		},
		expectedOut: `content`,
	},

	"Distraction free macro declaration (2)": {
		sources: fstest.Files{
			"index.html":  `{% extends "layout.html" %}{% Article %}{% Content %}`,
			"layout.html": `{% show Article() %}`,
		},
		expectedBuildErr: `undefined: Content`,
	},

	"Distraction free macro declaration (3)": {
		sources: fstest.Files{
			"index.html":  `{% extends "layout.html" %}{% Article %}{% end macro %}`,
			"layout.html": `{% show Article() %}`,
		},
		expectedBuildErr: `syntax error: unexpected end`,
	},

	"Distraction free macro declaration (4)": {
		sources: fstest.Files{
			"index.html":  `{% extends "layout.html" %}{% article %}{% end %}`,
			"layout.html": `{% show Article() %}`,
		},
		expectedBuildErr: `syntax error: unexpected article, expecting declaration statement`,
	},

	"Distraction free macro declaration (5)": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" %}{% show Article() %}`,
			"imported.html": `{% Article %}`,
		},
		expectedBuildErr: `syntax error: unexpected Article, expecting declaration statement`,
	},

	"Distraction free macro declaration (6)": {
		sources: fstest.Files{
			"index.html":   `{{ render "partial.html" }}`,
			"partial.html": `{% Article %}`,
		},
		expectedBuildErr: `undefined: Article`,
	},

	"Macro in tab code block context": {
		sources: fstest.Files{
			"index.md": "\t{% macro A %}{% end %}",
		},
		expectedBuildErr: `syntax error: macro declaration not allowed in tab code block`,
	},

	"Macro in spaces code block context": {
		sources: fstest.Files{
			"index.md": `    {% macro A %}{% end %}`,
		},
		expectedBuildErr: `syntax error: macro declaration not allowed in spaces code block`,
	},

	"Macro used in function call - an empty string is returned": {
		sources: fstest.Files{
			"index.html": `{% macro M %}{% end %}{% var str = M() %}{{ len(str) }}`,
		},
		expectedOut: `0`,
	},

	"Macro used in function call - a non-empty string is returned (1)": {
		sources: fstest.Files{
			"index.html": `{% macro M %}hello{% end %}{% var str = M() %}{{ len(str) }}`,
		},
		expectedOut: `5`,
	},

	"Macro used in function call - a non-empty string is returned (2)": {
		sources: fstest.Files{
			"index.html": `{% macro M %}hello{% end %}{% var str = M() %}len(str): {{ len(str) }}, output of macro: {{ M() }}`,
		},
		expectedOut: `len(str): 5, output of macro: hello`,
	},

	"Show - String literal": {
		sources: fstest.Files{
			"index.html": `{% show "partial.html" %}`,
		},
		expectedOut: `partial.html`,
	},

	"Render - Expression": {
		sources: fstest.Files{
			"index.txt": `{% file := render "file.txt" %}file.txt has a length of {{ len(file) }}`,
			"file.txt":  `ciao`,
		},
		expectedOut: "file.txt has a length of 4",
	},

	"Render - Rendering the same file twice": {
		sources: fstest.Files{
			"index.txt": `{% p1 := render "file.txt" %}{% p2 := render "file.txt" %}p1 is {{ p1 }} (len = {{ len(p1) }}), p2 is {{ p2 }}`,
			"file.txt":  `ciao`,
		},
		expectedOut: "p1 is ciao (len = 4), p2 is ciao",
	},

	"Convert a markdown value to an html value": {
		sources: fstest.Files{
			"index.txt": `{% var m markdown = "# title" %}{% h := html(m) %}{{ string(h) }}`,
		},
		expectedOut: "--- start Markdown ---\n# title--- end Markdown ---\n",
	},

	"Convert a markdown value to an html value - Indirect": {
		sources: fstest.Files{
			"index.txt": `{%%
				var m markdown = "# title"
				var h html
				var hRef *html = &h
				h = html(m)
				show string(h)
			%%}`,
		},
		expectedOut: "--- start Markdown ---\n# title--- end Markdown ---\n",
	},

	"Convert a markdown value to an html value - Interface": {
		sources: fstest.Files{
			"index.txt": `{%%
				var m markdown = "# title"
				var i interface{}
				i = html(m)
				show string(i.(html))
			%%}`,
		},
		expectedOut: "--- start Markdown ---\n# title--- end Markdown ---\n",
	},

	"Convert a markdown value to an html value - Closure": {
		sources: fstest.Files{
			"index.txt": `{%%
				var m markdown = "# title"
				var h html
				func () {
					h = html(m)
				}()
				show string(h)
			%%}`,
		},
		expectedOut: "--- start Markdown ---\n# title--- end Markdown ---\n",
	},

	"https://github.com/open2b/scriggo/issues/728: Text instruction merging error": {
		sources: fstest.Files{
			"index.txt": `{% if false %}{% for false %}{% end %}<d>{% end %}<e>`,
		},
		expectedOut: "<e>",
	},

	"Macro declarations inside macro declarations": {
		sources: fstest.Files{
			"index.html": `
			{% macro External1() %}
				{% macro internal1 %}internal1 (1){% end %}
				{% macro internal2 string %}internal2 (1){% end %}
				External1's body: {{ internal1() }} {{ internal2() }}
			{% end %}

			{% macro External2() %}
				{% macro internal1 %}internal1 (2){% end %}
				{% macro internal2 string %}internal2 (2){% end %}
				External2's body: {{ internal1() }} {{ internal2() }}
			{% end %}
			
			External1: {{ External1() }}
			External2: {{ External2() }}`,
		},
		expectedOut: "\n\n\t\t\t\n\t\t\tExternal1: \t\t\t\t\n\t\t\t\t\n\t\t\t\tExternal1's body: internal1 (1) internal2 (1)\n\n\t\t\tExternal2: \t\t\t\t\n\t\t\t\t\n\t\t\t\tExternal2's body: internal1 (2) internal2 (2)\n",
	},

	"Internal function declaration accessing a variable declared in the external function declaration": {
		sources: fstest.Files{
			"index.html": `
		{%% External := func() string {
			var n int = 42
			internal := func() string { return sprintf("n has value: %d", n) }
			return internal()
		} %%}
		{{ External() }}`,
		},
		expectedOut: "\n\t\tn has value: 42",
	},

	"Trying to assign to a macro declared in the file/package block": {
		sources: fstest.Files{
			"index.html": `
					{% macro M %}{% end %}
					{% M = func() string { return "" } %}
				`,
		},
		expectedBuildErr: "cannot assign to M",
	},

	"Trying to assign to a macro declared inside another macro": {
		sources: fstest.Files{
			"index.html": `
			{% macro External %}
				{% macro M %}{% end %}
				{% M = func() string { return "" } %}
			{% end macro %}
		`,
		},
		expectedBuildErr: "cannot assign to M",
	},

	"When a macro is assigned to a variable, such variable can be reassigned without returning error 'cannot assign to'": {
		sources: fstest.Files{
			"index.txt": `
			{% macro M %}{% end %}
			{% var N = M %}
			{% N = func() string { return "hi" } %}`,
		},
		expectedOut: "\n\t\t\t\n\t\t\t\n\t\t\t",
	},

	"Internal macro accessing a variable declared in the external macro": {
		sources: fstest.Files{
			"index.html": `
			{% macro External %}
				This is External
				{% var n int = 42 %}
				{% macro internal %}n has value: {{ n }}{% end %}
				internal: {{ internal() }}
			{% end %}
			Showing External: {{ External() }}`,
		},
		expectedOut: "\n\t\t\tShowing External: \t\t\t\tThis is External\n\t\t\t\t\n\t\t\t\t\n\t\t\t\tinternal: n has value: 42\n",
	},

	"Shadowing an identifier used as macro result parameter": {
		sources: fstest.Files{
			"index.html": `{% var css string %}{% macro A css %}{% end %}`,
		},
		expectedBuildErr: `css is not a type`,
	},

	"Redeclaration of a macro within the same scope": {
		sources: fstest.Files{
			"index.html": `
			{% macro M %}
				{% macro Inner %}{% end %}
				{% macro Inner %}{% end %}
			{% end macro %}`,
		},
		expectedBuildErr: "4:14: Inner redeclared in this block\n\tprevious declaration at 3:14",
	},

	"Redeclaration of an identifier within the same scope": {
		sources: fstest.Files{
			"index.html": `
			{% macro M %}
				{% macro Inner %}{% end %}
				{% var Inner = 2 %}
			{% end macro %}`,
		},
		expectedBuildErr: "4:12: Inner redeclared in this block\n\tprevious declaration at 3:14",
	},

	"https://github.com/open2b/scriggo/issues/701": {
		sources: fstest.Files{
			"index.html":   `{{ render "partial.html" }}`,
			"partial.html": "{% var a int %}{% macro b %}{{ a }}{% end %}{{ b() }}",
		},
		expectedOut: `0`,
	},

	"https://github.com/open2b/scriggo/issues/739 (import)": {
		sources: fstest.Files{
			"index.txt":    `{% import "imported.txt" %}`,
			"imported.txt": `{%% a := 1 %%}`,
		},
		expectedBuildErr: `unexpected a, expecting declaration statement`,
	},

	"https://github.com/open2b/scriggo/issues/739 (extends)": {
		sources: fstest.Files{
			"index.txt":    `{% extends "extended.txt" %}{%% a := 1 %%}`,
			"extended.txt": ``,
		},
		expectedBuildErr: `unexpected a, expecting declaration statement`,
	},

	"https://github.com/open2b/scriggo/issues/741 - non pointer": {
		sources: fstest.Files{
			"index.txt": `{%%
				t := T{}
				t.A = "hello"
				show t.A
			%%}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"T": reflect.TypeOf(struct{ A string }{}),
			},
		},
		expectedOut: "hello",
	},

	"Imported file that imported a precompiled package": {
		sources: fstest.Files{
			"index.txt":    `{% import "imported.txt" %}{{ A }}, len is {{ len(A) }}`,
			"imported.txt": `{% import "fmt" %}{% var A = fmt.Sprint(42) %}`,
		},
		importer:    testPackages,
		expectedOut: "42, len is 2",
	},

	"Importing and not using a precompiled package should not return error": {
		sources: fstest.Files{
			"index.txt": `{% import "fmt" %}that's ok`,
		},
		importer:    testPackages,
		expectedOut: "that's ok",
	},

	"https://github.com/open2b/scriggo/issues/727 - Macro (1)": {
		sources: fstest.Files{
			"index.html": `
			{% macro M(param int) %}
				{% macro localMacro %}
					{{ param }}
				{% end %}
			{% end %}`,
		},
		expectedOut: "\n",
	},

	"https://github.com/open2b/scriggo/issues/727 - Macro (2)": {
		sources: fstest.Files{
			"index.html": `
			{% macro M(param int) %}
				{% macro localMacro %}{{ param }}{% end %}
				{{ localMacro() }}
			{% end %}
			{{ M(42) }}`,
		},
		expectedOut: "\n\t\t\t\t\t\t\t\n\t\t\t\t42\n",
	},

	"https://github.com/open2b/scriggo/issues/727 - Function literal (1)": {
		sources: fstest.Files{
			"index.html": `
			{%%
				M := func(param int) int {
					localFunc := func() int {
						return param
					}
					return 0
				}
			%%}`,
		},
		expectedOut: "\n\t\t\t",
	},

	"https://github.com/open2b/scriggo/issues/727 - Function literal (2)": {
		sources: fstest.Files{
			"index.html": `
			{%%
				M := func(param int) int {
					localFunc := func() int {
						return param
					}
					return localFunc()
				}
			%%}
			{{ M(439) }}`,
		},
		expectedOut: "\n\t\t\t439",
	},

	"https://github.com/open2b/scriggo/issues/740 - Call of unexported macro declared in imported file": {
		sources: fstest.Files{
			"index.txt":    `{% import "imported.txt" %}{{ m() }}`,
			"imported.txt": `{% macro m %}{% end %}`,
		},
		expectedBuildErr: "undefined: m",
	},

	"Missing end for statement": {
		sources: fstest.Files{
			"index.txt": `{% for %}`,
		},
		expectedBuildErr: "unexpected EOF, expecting {% end %} or {% end for %}",
	},

	"Missing end if statement": {
		sources: fstest.Files{
			"index.txt": `{% if a %}`,
		},
		expectedBuildErr: "unexpected EOF, expecting {% end %} or {% end if %}",
	},

	"Missing end if else statement": {
		sources: fstest.Files{
			"index.txt": `{% if a %}{% else %}`,
		},
		expectedBuildErr: "unexpected EOF, expecting {% end %} or {% end if %}",
	},

	"Missing end macro statement": {
		sources: fstest.Files{
			"index.txt": `{% macro a %}`,
		},
		expectedBuildErr: "unexpected EOF, expecting {% end %} or {% end macro %}",
	},

	"Missing end raw statement with marker": {
		sources: fstest.Files{
			"index.txt": "{% raw code %}",
		},
		expectedBuildErr: "unexpected EOF, expecting {% end raw code %}",
	},

	"Missing end raw statement without marker": {
		sources: fstest.Files{
			"index.txt": "{% raw %}",
		},
		expectedBuildErr: "unexpected EOF, expecting {% end %} or {% end raw %}",
	},

	"Missing end switch statement": {
		sources: fstest.Files{
			"index.txt": `{% switch %}`,
		},
		expectedBuildErr: "unexpected EOF, expecting {% end %} or {% end switch %}",
	},

	"Missing end select statement": {
		sources: fstest.Files{
			"index.txt": `{% select %}`,
		},
		expectedBuildErr: "unexpected EOF, expecting {% end %} or {% end select %}",
	},

	"Raw statement": {
		sources: fstest.Files{
			"index.txt": "a\n{% raw %}\nb\n{% end %}\nc",
		},
		expectedOut: "a\nb\nc",
	},

	"Raw statement with marker": {
		sources: fstest.Files{
			"index.txt": "a\n{% raw code %}\nb\n{% end raw code %}\nc",
		},
		expectedOut: "a\nb\nc",
	},

	"Missing marker in end raw statement": {
		sources: fstest.Files{
			"index.txt": "{% raw code %}{% end raw %}",
		},
		expectedBuildErr: "unexpected EOF, expecting {% end raw code %}",
	},

	"Invalid bytes in an end raw statement": {
		sources: fstest.Files{
			"index.txt": "{% raw %}{% end \x00 %}",
		},
		expectedBuildErr: "unexpected NUL in input",
	},

	"Raw statement in statements": {
		sources: fstest.Files{
			"index.txt": "{%% raw %%}",
		},
		expectedBuildErr: "cannot use raw between {%% and %%}",
	},

	"Raw statement in imported sources": {
		sources: fstest.Files{
			"index.txt":     "{% import \"imported1.txt\" %}{% import \"imported2.txt\" %}",
			"imported1.txt": "{% macro a %}{% raw %}{% end %}{% end %}",
			"imported2.txt": "{% raw %}{% end %}",
		},
		expectedBuildErr: "imported2.txt:1:4: syntax error: unexpected raw, expecting declaration statement",
	},

	"https://github.com/open2b/scriggo/issues/770": {
		sources: fstest.Files{
			"index.txt":    `{% import "imported.txt" %}`,
			"imported.txt": `{% macro m %}{% end %}{{ m() }}`,
		},
		expectedBuildErr: "unexpected {{, expecting declaration statement",
	},

	"https://github.com/open2b/scriggo/issues/770 (2)": {
		sources: fstest.Files{
			"index.txt":  `{% extends "layout.txt" %}{% macro m %}{% end %}{{ m() }}`,
			"layout.txt": ``,
		},
		expectedBuildErr: "unexpected {{, expecting declaration statement",
	},

	"Do not parse short show statement": {
		sources: fstest.Files{
			"index.txt": "{% show 5 %} == {{ 5 }}",
		},
		noParseShow: true,
		expectedOut: "5 == {{ 5 }}",
	},

	"Default variable declaration": {
		sources: fstest.Files{
			"index.txt": `{% var i, j = I default 10, J default 3 %}{{ i }},{{ j }}`,
		},
		expectedOut: `5,3`,
	},

	"Default short declaration": {
		sources: fstest.Files{
			"index.txt": `{% i, j := I default 10, J default 3 %}{{ i }},{{ j }}`,
		},
		expectedOut: `5,3`,
	},

	"Default constant declaration": {
		sources: fstest.Files{
			"index.txt": `{% const i, j int = C default 10, J default 3 %}{{ i }},{{ j }}`,
		},
		expectedOut: `8,3`,
	},

	"Default assignment": {
		sources: fstest.Files{
			"index.txt": `{% var i, j int %}{% i, j = I default 10, J default 3 %}{{ i }},{{ j }}`,
		},
		expectedOut: `5,3`,
	},

	"Show default": {
		sources: fstest.Files{
			"index.html": `{{ I default 10 }},{{ J default 3 }}`,
		},
		expectedOut: `5,3`,
	},

	"Default show macro": {
		sources: fstest.Files{
			"index.html":  `{% extends "layout.html" %}{% macro M %}i'm a macro{% end %}`,
			"layout.html": `{% show M() default 42 %}; {% show N() default "no macro" %}`,
		},
		expectedOut: `i'm a macro; no macro`,
	},

	"Default short show macro": {
		sources: fstest.Files{
			"index.html":  `{% extends "layout.html" %}{% macro M %}i'm a macro{% end %}`,
			"layout.html": `{{ M() default 42 }}; {{ N() default "no macro" }}`,
		},
		expectedOut: `i'm a macro; no macro`,
	},

	"Default: cannot use non-macro in call form": {
		sources: fstest.Files{
			"index.html":  `{% extends "layout.html" %}`,
			"layout.html": `{% M := 32 %}{{ M() default 42 }}`,
		},
		expectedBuildErr: `cannot use M (type int) as macro`,
	},

	"Default: macro not declared in file with extends": {
		sources: fstest.Files{
			"index.html":    `{% extends "extended.html" %}`,
			"extended.html": `{% macro M %}{% end %}{{ M() default "" }}`,
		},
		expectedBuildErr: "macro not declared in file with extends",
	},

	"Use of default with call in non-extended file": {
		sources: fstest.Files{
			"index.html": `
				{% extends "extended.html" %}
				{% macro M %}{% end %}
				{% macro N %}{{ M() default "" }}{% end macro %}`,
			"extended.html": ``,
		},
		expectedBuildErr: "use of default with call in non-extended file",
	},

	"Default show macro with blank identifier": {
		sources: fstest.Files{
			"index.html":  `{% extends "layout.html" %}`,
			"layout.html": `{{ _() default "" }}`,
		},
		expectedBuildErr: `cannot use _ as value`,
	},

	"Default declaration with macro": {
		sources: fstest.Files{
			"index.html":  `{% extends "layout.html" %}{% macro M %}i'm a macro{% end %}`,
			"layout.html": `{% var m, n = M() default html(""), N() default "no macro" %}{{ m }}; {{ n }}`,
		},
		expectedOut: `i'm a macro; no macro`,
	},

	"Default declaration with iota": {
		sources: fstest.Files{
			"index.html": `{% var v = iota default 5 %}{% const ( c1 = iota; c2 = iota default 5 ) %}{{ v }}; {{ c2 }}`,
		},
		expectedOut: `5; 1`,
	},

	"Default declaration with not existent macro": {
		sources: fstest.Files{
			"index.html":  `{% extends "layout.html" %}`,
			"layout.html": `{% var m = M(5, nil, struct{}{}, []int{}...) default "no macro" %}{{ m }}`,
		},
		expectedOut: `no macro`,
	},

	"Default declaration with not existent macro (2)": {
		sources: fstest.Files{
			"index.html":  `{% extends "layout.html" %}`,
			"layout.html": `{% s := "s" %}{% var m = M(5, s...) default "no macro" %}`,
		},
		expectedBuildErr: `cannot use s (type string) as variadic argument`,
	},

	"Default declaration with not existent macro (3)": {
		sources: fstest.Files{
			"index.html":  `{% extends "layout.html" %}`,
			"layout.html": `{% var m = M() default 6 %}`,
		},
		expectedBuildErr: `mismatched format type and int type`,
	},

	"Default show with render": {
		sources: fstest.Files{
			"index.html":   `{% show render "partial.html" default "ops" %}; {% show render "no-partial.html" default "no partial" %}`,
			"partial.html": `i'm a partial`,
		},
		expectedOut: `i'm a partial; no partial`,
	},

	"Default show with double render": {
		sources: fstest.Files{
			"index.html":    `{% show render "partial1.html" default render "partial2.html" %}; {% show render "partial3.html" default render "partial4.html" %}`,
			"partial1.html": `i'm partial 1`,
			"partial2.html": `i'm partial 2`,
			"partial4.html": `i'm partial 4`,
		},
		expectedOut: `i'm partial 1; i'm partial 4`,
	},

	"Default short show with render": {
		sources: fstest.Files{
			"index.html":   `{{ render "partial.html" default "ops" }}; {{ render "no-partial.html" default "no partial" }}`,
			"partial.html": `i'm a partial`,
		},
		expectedOut: `i'm a partial; no partial`,
	},

	"Default declaration with render": {
		sources: fstest.Files{
			"index.html":   `{% var s html = render "partial.html" default "ops" %}{% t := render "no-partial.html" default html("no partial") %}{{ s }}; {{ t }}`,
			"partial.html": `i'm a partial`,
		},
		expectedOut: `i'm a partial; no partial`,
	},

	"Default declaration with render (2)": {
		sources: fstest.Files{
			"index.html":   `{% var s = render "partial.html" default "" %}`,
			"partial.html": `i'm a partial`,
		},
		expectedBuildErr: `cannot use render "partial.html" (type native.HTML) as type string in assignment`,
	},

	"Default declaration with render (3)": {
		sources: fstest.Files{
			"index.html":   `{% const s html = render "partial.html" default "" %}`,
			"partial.html": `i'm a partial`,
		},
		expectedBuildErr: `const initializer render "partial.html" is not a constant`,
	},

	"Removed special render assignment form": {
		sources: fstest.Files{
			"index.html":   `{% var s string %}{% var ok bool %}{% s, ok = render "partial.html" %}`,
			"partial.html": `i'm a partial`,
		},
		expectedBuildErr: `assignment mismatch: 2 variables but 1 values`,
	},

	"Removed special render declaration form": {
		sources: fstest.Files{
			"index.html":   `{% var s, ok = render "partial.html" %}`,
			"partial.html": `i'm a partial`,
		},
		expectedBuildErr: `assignment mismatch: 2 variables but 1 values`,
	},

	"Use of default in invalid context": {
		sources: fstest.Files{
			"index.html": `{% if a default true %}{% end %}`,
		},
		expectedBuildErr: `cannot use default expression in this context`,
	},

	"https://github.com/open2b/scriggo/issues/572 (1)": {
		sources: fstest.Files{
			"index.html":  `{% extends "layout.html" %}`,
			"layout.html": `{% a = 5 %}`,
		},
		expectedBuildErr: `layout.html:1:4: undefined: a`,
	},

	"https://github.com/open2b/scriggo/issues/572 (2)": {
		sources: fstest.Files{
			"index.html": `{% extends "layout.html" %}{% a = 5 %}`,
		},
		expectedBuildErr: `index.html:1:31: syntax error: unexpected a, expecting declaration statement`,
	},

	"https://github.com/open2b/scriggo/issues/768 (1)": {
		sources: fstest.Files{
			"index.html":   `{% _ = render "partial.html" %}`,
			"partial.html": `{% macro m %}{% _ = page %}{% end %}{{ m() }}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"page": &([]string{"a"})[0],
			},
		},
		expectedOut: "",
	},

	"https://github.com/open2b/scriggo/issues/768 (2)": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" %}{% r := M() %}{{ r }}`,
			"imported.html": `{% macro M %}{% _ = global %}{% end %}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"global": &([]string{"a"})[0],
			},
		},
	},

	"https://github.com/open2b/scriggo/issues/768 (3)": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" %}{{ M() }}`,
			"imported.html": `{% macro M %}{% macro m %}{% _ = global %}{% end %}{{ m() }}{% end %}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"global": &([]int{0})[0],
			},
		},
	},

	"https://github.com/open2b/scriggo/issues/768 (4)": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" %}{{ M() }}`,
			"imported.html": `{% macro M %}{% f := func() { _ = global } %}{% f() %}{% end %}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"global": &([]int{0})[0],
			},
		},
	},

	"https://github.com/open2b/scriggo/issues/768 (5)": {
		sources: fstest.Files{
			"index.html": `{% import "imported.html" %}{% M() %}`,
			"imported.html": `{% var M = func() {
				f := func() { _ = global }
				f()
			} %}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"global": &([]int{0})[0],
			},
		},
	},

	"https://github.com/open2b/scriggo/issues/768 (6)": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" %}{{ M() }}`,
			"imported.html": `{% macro M %}{% func() { _ = global }() %}{% end %}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"global": &([]int{0})[0],
			},
		},
	},

	"https://github.com/open2b/scriggo/issues/768 (7)": {
		sources: fstest.Files{
			"index.html":   `{{ render "partial.html" }}`,
			"partial.html": `{% macro m %}{{ page }}{% end %}{{ m() }}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"page": &([]string{"a"})[0],
			},
		},
		expectedOut: `a`,
	},

	"https://github.com/open2b/scriggo/issues/768 (8)": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" %}{{ M() }}`,
			"imported.html": `{% macro M %}{{ 2 * func() int { return global }() }}{% end %}`,
		},
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"global": &([]int{21})[0],
			},
		},
		expectedOut: `42`,
	},

	"Macro called as native": {
		sources: fstest.Files{
			"index.html": `{% macro a %}hey{% end %}{% var s = func() html { return a() }() %}{{ s }}`,
		},
		expectedOut: "hey",
	},

	"Invalid memory address or nil pointer dereference": {
		sources: fstest.Files{
			"index.html": `

		{% macro M1(m macro() html) %}{{ m() }}{% end %}
		{% macro M2(title, buttonText string) %}
			{% macro content %}
			{% end %}
			{{ M1(content) }}
		{% end %}
		
		{{ M2("Press the button dialog", "Play") }}`,
		},
		expectedOut: "\n\n\t\t\n\t\t\n\t\t\t\t\t\n",
	},

	"Using - show": {
		sources: fstest.Files{
			"index.html": `{% show itea; using html %}foo{% end using %}`,
		},
		expectedOut: "foo",
	},

	"Using - show (implicit type)": {
		sources: fstest.Files{
			"index.html": `{% show itea; using %}foo{% end using %}`,
		},
		expectedOut: "foo",
	},

	"Using - show - Two using statement": {
		sources: fstest.Files{
			"index.txt": `{% show itea; using %}foo{% end using %}{% show itea; using %}bar{% end using %}`,
		},
		expectedOut: "foobar",
	},

	"Using - 'itea' is not defined outside": {
		sources: fstest.Files{
			"index.html": `{% show itea; using html %}foo{% end using %}{{ itea }}`,
		},
		expectedBuildErr: "undefined: itea",
	},

	"Using - 'itea' is not defined outside (implicit type)": {
		sources: fstest.Files{
			"index.html": `{% show itea; using %}foo{% end using %}{{ itea }}`,
		},
		expectedBuildErr: "undefined: itea",
	},

	"Using - assignment with ':='": {
		sources: fstest.Files{
			"index.html": `{% x := itea; using html %}hello, how are you{% end using %}{{ x }}, len: {{ len(x) }}`,
		},
		expectedOut: "hello, how are you, len: 18",
	},

	"Using - assignment with ':=' (implicit type)": {
		sources: fstest.Files{
			"index.html": `{% x := itea; using %}hello, how are you{% end using %}{{ x }}, len: {{ len(x) }}`,
		},
		expectedOut: "hello, how are you, len: 18",
	},

	"Using - assignment with 'var'": {
		sources: fstest.Files{
			"index.html": `{% var date, days = itea, 5; using html %}
			<span>{{ now() }}</span>
		  {% end using %}
		  Date is {{ date }}`,
		},
		expectedOut: "\t\t  Date is \n\t\t\t<span>1999-01-19</span>\n",
		main: native.Package{
			Name: "main",
			Declarations: native.Declarations{
				"now": func() string { return "1999-01-19" },
			},
		},
	},

	"Using - macro (without parameters)": {
		sources: fstest.Files{
			"index.txt": `{% show itea(); using macro() string %}macro content{% end using %}`,
		},
		expectedOut: "macro content",
	},

	"Using - macro (with parameters)": {
		sources: fstest.Files{
			"index.txt": `{% show itea(4.2); using macro(f float64) string %}f / 2 = {{ f / 2 }}.{% end using %}`,
		},
		expectedOut: "f / 2 = 2.1.",
	},

	"Using - function literal 1": {
		sources: fstest.Files{
			"index.txt": `{% show func() string { _ = itea ; var itea = "ok"; return itea }(); using %}no{% end using %}`,
		},
		expectedOut: "ok",
	},

	"Using - function literal 2": {
		sources: fstest.Files{
			"index.txt": `{% show func() string { return itea }(); using %}ok{% end using %}`,
		},
		expectedOut: "ok",
	},

	"Using - package level var declaration ": {
		sources: fstest.Files{
			"index.html": `{% import "file.html" %}`,
			"file.html":  `{% var _ = itea; using %}hey{% end using %}`,
		},
	},

	"Using - package level var declaration (2)": {
		sources: fstest.Files{
			"index.html": `{% import "file.html" %}{{ V }}, len: {{ len(V) }}`,
			"file.html":  `{% var V = itea; using %}hey{% end using %}`,
		},
		expectedOut: "hey, len: 3",
	},

	"Using - package level var declaration (3)": {
		sources: fstest.Files{
			"index.html": `{% import "file.html" %}V is {{ V }}`,
			"file.html":  `{% var V = len(itea); using %}hey my friend{% end using %}`,
		},
		expectedOut: "V is 13",
	},

	"Using - package level var declaration (4)": {
		sources: fstest.Files{
			"index.html": `{% import "file.html" %}{{ V1 }}, {{ V2 }}`,
			"file.html":  `{% var V1, V2 = itea, len(itea); using %}hey oh{% end using %}`,
		},
		expectedOut: "hey oh, 6",
	},

	"Using - package level var declaration (5)": {
		sources: fstest.Files{
			"index.html": `
				{% extends "extended.html" %}
				{% var itea = "shadowed" %}
				{% var V = itea; using %}content...{% end using %}
				{% macro M %}V is {{ V }}{% end macro %}
			`,
			"extended.html": `{{ M () }}`,
		},
		expectedBuildErr: "predeclared identifier itea not used",
	},

	"Using - package level var declaration (5) - simplified": {
		sources: fstest.Files{
			"index.html": `
				{% extends "extended.html" %}
				{% var itea = "shadowed" %}
				{% var _ = itea; using %}{% end using %}
			`,
			"extended.html": ``,
		},
		expectedBuildErr: "predeclared identifier itea not used",
	},

	"Using - itea shadowed by a package name at package level": {
		sources: fstest.Files{
			"index.html": `
				{% extends "extended.html" %}
				{% import itea "imported.html" %}
				{% var V = itea.A; using %}content...{% end using %}
				{% macro M %}V is {{ V }}{% end macro %}
			`,
			"extended.html": `{{ M () }}`,
			"imported.html": `{% var A = 5 %}`,
		},
		expectedBuildErr: "predeclared identifier itea not used",
	},

	"Using - itea shadowed by a 'var' declaration inside a multiline statement": {
		sources: fstest.Files{
			"index.html": `
				{% extends "extended.html" %}
				{%%
					var (
						something = 43982
						itea = "shadowed"
						somethingElse = 43289
					)
				%%}
				{% var V = itea; using %}content...{% end using %}
				{% macro M %}V is {{ V }}{% end macro %}
			`,
			"extended.html": `{{ M () }}`,
		},
		expectedBuildErr: "predeclared identifier itea not used",
	},

	"Using - assigning from using body": {
		sources: fstest.Files{
			"index.html": `
	            {% f := func() html { return html("") } %}
	            {% _ = itea; using %}
	        		{% f = itea; using macro() html %}x{% end %}
	            {% end %}
				{{ f() }}
			`,
		},
		expectedOut: "\n\t\t\t\tx\n\t\t\t",
	},

	"Nested using statements": {
		sources: fstest.Files{
			"index.html": `
	           {% var f func(html) html %}
	           {% show f(itea); using %}
	           2 {% f = itea; using macro(s html) html %}1 {{ s }} 4{% end %} 3
	           {% end %}
			`,
		},
		expectedOut: "\n\t           \n\t           1 \n\t           2  3\n 4\t\t\t",
	},

	"Using - nested using statements (1)": {
		sources: fstest.Files{
			"index.html": `
	            {% _ = itea; using %}
	      	    	{% _ = itea; using %}{% end %}
	            {% end %}
			`,
		},
		expectedOut: "\n\t\t\t",
	},

	"Using - nested using statements (2)": {
		sources: fstest.Files{
			"index.html": `
	            {% show itea; using %}
					External using-start
	      	    	{% show itea; using %}internal using{% end %}
					External using-end
	            {% end %}
			`,
		},
		expectedOut: "\n\t            \n\t\t\t\t\tExternal using-start\n\t      \t    \tinternal using\n\t\t\t\t\tExternal using-end\n\t\t\t",
	},

	"Using - nested using statements (3)": {
		sources: fstest.Files{
			"index.html": `
	            {% _ = itea; using %}
	      	    	{% _ = itea; using %}{% end %}
					{% _ = itea; using %}{% end %}
	            {% end %}
				{% _ = itea; using %}
	      	    	{% _ = itea; using %}
					  {% _ = itea; using %}{% end %}
					  {% _ = itea; using %}{% end %}
					{% end %}
					{% _ = itea; using %}{% end %}
	            {% end %}
				{% _ = itea; using %}
	      	    	{% _ = itea; using %}{% end %}
	            {% end %}
			`,
		},
		expectedOut: "\n\t\t\t",
	},

	"Using - the type has been shadowed at package-level": {
		sources: fstest.Files{
			"index.html": `{% import "imported.html" %}{{ A }}`,
			"imported.html": `
				{% type html int %}
				{% var A = itea; using html %}OPS{% end %}
			`,
		},
		expectedBuildErr: `invalid using type html`,
	},

	"Using - expression statement": {
		sources: fstest.Files{
			"index.txt": `
				{% var V int %}
				{% f := func(s string) { V = len(s) } %}
				{% f(itea); using %}hello{% end using %}
				V is {{ V }}
			`,
		},
		expectedOut: "\n\t\t\t\t\n\t\t\t\t\n\t\t\t\tV is 5\n\t\t\t",
	},

	"Using - send statement": {
		sources: fstest.Files{
			"index.txt": `
				{% ch := make(chan string, 1) %}
				{% ch <- itea; using %}how are you?{% end %}
				Message is: {{ <-ch }}
			`,
		},
		expectedOut: "\n\t\t\t\t\n\t\t\t\tMessage is: how are you?\n\t\t\t",
	},

	"Using - escaping string in html context": {
		sources: fstest.Files{
			"index.html": `{% show itea; using string %}<b>{% end using %}`,
		},
		expectedOut: "&lt;b&gt;",
	},

	"Using - in macro": {
		sources: fstest.Files{
			"index.html": `
				{% extends "layout.html" %}
				{% macro Body %}
					{% var a = itea; using %}a{% end using %}
				{% end macro %}
			`,
			"layout.html": `{{ Body() }}`,
		},
		expectedOut: "\t\t\t\t\t\n",
	},

	"Using - in macro (2)": {
		sources: fstest.Files{
			"index.html": `
				{% extends "imported.html" %}
				{% macro M %}
					{% var a = itea; using %}content{% end using %}
					{{ a }}
				{% end macro %}
			`,
			"imported.html": `{{ M() }}`,
		},
		expectedOut: "\t\t\t\t\t\n\t\t\t\t\tcontent\n",
	},

	"Using - error if 'itea' is unused": {
		sources: fstest.Files{
			"index.html": `
				{% show 4; using %}Something{% end using %}
			`,
		},
		expectedBuildErr: "index.html:2:16: predeclared identifier itea not used",
	},

	"Using - error if 'itea' is unused (package level)": {
		sources: fstest.Files{
			"index.html": `{% import "imported.html" %}`,
			"imported.html": `
				{% var _ = 4; using %}Something{% end using %}
			`,
		},
		expectedBuildErr: "imported.html:2:19: predeclared identifier itea not used",
	},

	"Using - itea on right side of default (evaluated)": {
		sources: fstest.Files{
			"index.html":    `{% extends "extended.html" %}`,
			"extended.html": `{% show Undef() default itea; using %}Something{% end using %}`,
		},
		expectedOut: "Something",
	},

	"Using - itea on right side of default (evaluated, package level)": {
		sources: fstest.Files{
			"index.html": `{% show undef default itea; using %}Something{% end using %}`,
		},
		expectedOut: "Something",
	},

	"Using - itea on right side of default ('itea' not referenced, content of 'using' must not be evaluated)": {
		sources: fstest.Files{
			"index.html":    `{% extends "extended.html" %}{% macro M %}{% end %}`,
			"extended.html": `{% show M() default itea; using %}{{ []int{}[1000] }}{% end using %}`,
		},
	},

	"Using - taking address of 'itea'": {
		sources: fstest.Files{
			"index.html": `
				{% var ref1, ref2, ref3, ref4 *html %}
				{% func() { ref1, ref2 = &itea, &itea }(); using %}content..{% end %}
				{% func() { ref3, ref4 = &itea, &itea }(); using %}content..{% end %}
				{{ ref1 == ref2 }}{{ ref2 == ref3 }}{{ ref3 == ref4 }}
			`,
		},
		expectedOut: "\n\t\t\t\t\n\t\t\t\t\n\t\t\t\t\n\t\t\t\ttruefalsetrue\n\t\t\t",
	},

	"Using - assign to 'itea'": {
		sources: fstest.Files{
			"index.html": `{% show func() html { itea = html("hey"); return itea }(); using %}content..{% end %}`,
		},
		expectedOut: "hey",
	},

	"Using - cannot use 'itea' on left side of default": {
		sources: fstest.Files{
			"index.html": `{% show itea default 4; using %}...{% end %}`,
		},
		expectedBuildErr: "use of predeclared identifier itea",
	},

	"Using - cannot use 'itea' on left side of default - package level": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" %}`,
			"imported.html": `{% var _ = itea default 4; using %}...{% end %}`,
		},
		expectedBuildErr: "use of predeclared identifier itea",
	},

	"Using - cannot use 'itea()' on left side of default": {
		sources: fstest.Files{
			"index.html":    `{% extends "extended.html" %}`,
			"extended.html": `{% show itea() default 4; using %}...{% end %}`,
		},
		expectedBuildErr: "use of predeclared identifier itea",
	},

	"Using - can assign to 'itea', even if it contains a macro": {
		sources: fstest.Files{
			"index.html": `{% func() { itea = func() html { return "x" } }(); using macro() %}content..{% end %}`,
		},
	},

	"Using - bad type (is a variable instead of a format type) (block)": {
		sources: fstest.Files{
			"index.html": `
				{% var html = 32 %}
				{% var _ = itea; using html %}...{% end using %}
			`,
		},
		expectedBuildErr: "html is not a type",
	},

	"Using - bad type (is a variable instead of a format type) (package-level)": {
		sources: fstest.Files{
			"index.html": `{% import "imported.html" %}`,
			"imported.html": `
				{% var html = 32 %}
				{% var _ = itea; using html %}...{% end using %}
			`,
		},
		expectedBuildErr: "html is not a type",
	},

	"Using - bad type (is a type but not a format type) (block)": {
		sources: fstest.Files{
			"index.html": `
				{% type html int %}
				{% var _ = itea; using html %}...{% end using %}
			`,
		},
		expectedBuildErr: `index.html:3:28: invalid using type html`,
	},

	"Using - bad type (is a type but not a format type) (package-level)": {
		sources: fstest.Files{
			"index.html": `{% import "imported.html" %}`,
			"imported.html": `
				{% type html int %}
				{% var _ = itea; using html %}...{% end using %}
			`,
		},
		expectedBuildErr: `imported.html:3:28: invalid using type html`,
	},

	"Using - implicit type": {
		sources: fstest.Files{
			"index.md": `{% var a markdown = itea; using %}# Scriggo{% end %}`,
		},
	},

	"Using - implicit macro type": {
		sources: fstest.Files{
			"index.css": `{% var a css = itea(); using macro %} div { color: red; }{% end %}`,
		},
	},

	"https://github.com/open2b/scriggo/issues/780": {
		sources: fstest.Files{
			"index.html":    `{% extends "extended.html" %}{% macro M %}{% end %}`,
			"extended.html": `{% show M default 0 %}`,
		},
		expectedBuildErr: "extended.html:1:9: use of non-builtin M on left side of default",
	},

	"Import without identifier": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" %}{{ V }}`,
			"imported.html": `{% var V = 10 %}`,
		},
		expectedOut: `10`,
	},

	"Import with . in templates": {
		sources: fstest.Files{
			"index.html":     `{% import . "imported1.html" %}{% import . "imported2.html" %}{{ V1 }}, {{ V2 }}`,
			"imported1.html": `{% var V1 = 10 %}`,
			"imported2.html": `{% var V2 = 20 %}`,
		},
		expectedOut: `10, 20`,
	},

	"import with 'for' - just one identifier": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" for V %}{{ V }}`,
			"imported.html": `{% var V = 10 %}`,
		},
		expectedOut: `10`,
	},

	"import with 'for' - more than one identifier": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" for V, T %}{{ V }}, {{ len(T{2, 3}) }}`,
			"imported.html": `{% var V = 10 %}{% type T []int %}`,
		},
		expectedOut: `10, 2`,
	},

	"import with 'for' - trying to import a not existing declaration": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" for NotExists %}{{ NotExists }}`,
			"imported.html": `{% var V = 10 %}`,
		},
		expectedBuildErr: "undefined: NotExists",
	},

	"import with 'for' - referring to a declaration not imported by 'for'": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" for V1 %}{{ V2 }}`,
			"imported.html": `{% var V1, V2 = 10, 20 %}`,
		},
		expectedBuildErr: "index.html:1:39: undefined: V2",
	},

	"import with 'for' - importing a declaration from a native package": {
		sources: fstest.Files{
			"index.txt": `{% import "fmt" for Sprint %}{{ Sprint(10, 20) }}`,
		},
		importer:    testPackages,
		expectedOut: "10 20",
	},

	"import with 'for' - importing a macro declaration": {
		sources: fstest.Files{
			"index.txt":   `{% import "macros.html" for M %}{{ M("hello") }}`,
			"macros.html": `{% macro M(s string) %}{{ s }}, world!{% end macro %}`,
		},
		expectedOut: "hello, world!",
	},

	"import with 'for' - trying to import a not exported declaration": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" for x %}{{ x }}`,
			"imported.html": `{% var x = 10 %}`,
		},
		expectedBuildErr: "cannot refer to unexported name x",
	},

	"import with 'for' - trying to import a not exported and not-existing declaration": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" for x %}{{ x }}`,
			"imported.html": `{% var y = 10 %}`,
		},
		expectedBuildErr: "cannot refer to unexported name x",
	},

	"Show markdown in HTML context": {
		sources: fstest.Files{
			"index.html": `{% md := markdown("# title") %}{{ md }}`,
		},
		expectedOut: "--- start Markdown ---\n# title--- end Markdown ---\n",
	},

	"Show string macro in HTML context": {
		sources: fstest.Files{
			"index.html": `
			{% macro M() string %}<b>ciao</b>{% end macro %}
			{% show M() %}
		`,
		},
		expectedOut: "\n\t\t\t\n\t\t\t&lt;b&gt;ciao&lt;/b&gt;\n\t\t",
	},

	// https://github.com/open2b/scriggo/issues/842
	"Macro in extending file refers to a type defined in the same file": {
		sources: fstest.Files{
			"index.html": `
			{% extends "extended.html" %}
			{% type T int %}
			{% macro M(T) %}{% end macro %}
		`,
			"extended.html": ``,
		},
	},

	"Changing the tree with extends does not impact paths of rendered and imported sources": {
		sources: fstest.Files{
			"index.html":           `{% extends "subdir/extended.html" %}`,
			"subdir/extended.html": `{% import "i.html" for V %}{{ render "r.html" }}{{ V }}`,
			"subdir/r.html":        ` rendered `,
			"subdir/i.html":        `{% var V = " imported " %}`,
		},
		expectedOut: " rendered  imported ",
	},

	"Extended file accessing to variables declared in extending file": {
		sources: fstest.Files{
			"index.html":    `{% extends "extended.html" %}{% var V = 42 %}`,
			"extended.html": `{{ V }}`,
		},
		expectedOut: "42",
	},

	"Taking the address of an imported variable": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" %}{% pv := &V %}{% *pv = 32 %}{{ V }}`,
			"imported.html": `{% var V int %}`,
		},
		expectedOut: "32",
	},

	"https://github.com/open2b/scriggo/issues/849": {
		sources: fstest.Files{
			"index.html":    `{% extends "extended.html" %}{% var V = 1 %}`,
			"extended.html": `{% var V = 2 %}`,
		},
		expectedBuildErr: "V redeclared in this block\n\textended.html:<nil>: previous declaration during import . \"index.html\"",
	},

	"https://github.com/open2b/scriggo/issues/849 (2)": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" %}{% var V = 32 %}`,
			"imported.html": `{% var V = 2 %}`,
		},
		expectedBuildErr: "V redeclared in this block\n\tindex.html:1:11: previous declaration during import . \"imported.html\"",
	},

	"https://github.com/open2b/scriggo/issues/855": {
		sources: fstest.Files{
			"index.html":     `{% import "imported1.html" %}{{ V2 }}`,
			"imported1.html": `{% import "imported2.html" %}`,
			"imported2.html": `{% var V2 = 2 %}`,
		},
		expectedBuildErr: "index.html:1:33: undefined: V2",
	},

	"https://github.com/open2b/scriggo/issues/851": {
		sources: fstest.Files{
			"index.html": `{% var x = { "one": 1, "two": 2, "three": 3 } %}`,
		},
		expectedBuildErr: "syntax error: unexpected {, expecting expression",
	},

	"https://github.com/open2b/scriggo/issues/850": {
		sources: fstest.Files{
			"index.txt": `{{ struct{T}{T{true}} }}`,
		},
		expectedBuildErr: "undefined: T",
	},

	"Multiple extends - simple case": {
		sources: fstest.Files{
			"index.html":     `{% extends "extended1.html" %}`,
			"extended1.html": `{% extends "extended2.html" %}`,
			"extended2.html": `extends 2`,
		},
		expectedOut: "extends 2",
	},

	"Multiple extends - redeclaration error": {
		sources: fstest.Files{
			"index.html":     `{% extends "extended1.html" %}`,
			"extended1.html": `{% extends "extended2.html" %}`,
			"extended2.html": `{% extends "extended3.html" %}`,
			"extended3.html": `{% extends "extended4.html" %}{% var V4 = 4 %}`,
			"extended4.html": `{% extends "extended5.html" %}{% var V4 = 4 %}`,
			"extended5.html": `{{ V4 }}`,
		},
		expectedBuildErr: "extended4.html:1:38: V4 redeclared in this block\n\textended4.html:<nil>: previous declaration during import . \"extended3.html\"",
	},

	"Multiple extends - many extended files": {
		sources: fstest.Files{
			"index.html":     `{% extends "extended1.html" %}{% var S0 = "0" %}`,
			"extended1.html": `{% extends "extended2.html" %}{% var S1 = S0 + "1" %}`,
			"extended2.html": `{% extends "extended3.html" %}{% var S2 = S1 + "2" %}`,
			"extended3.html": `{% extends "extended4.html" %}{% var S3 = S2 + "3" %}`,
			"extended4.html": `{% extends "extended5.html" %}{% var S4 = S3 + "4" %}`,
			"extended5.html": `{{ S4 }}`,
		},
		expectedOut: "01234",
	},

	"Multiple extends - error when referring to undefined name": {
		sources: fstest.Files{
			"index.html":     `{% extends "extended1.html" %}`,
			"extended1.html": `{% extends "extended2.html" %}`,
			"extended2.html": `{% extends "extended3.html" %}`,
			"extended3.html": `{% extends "extended4.html" %}{% var V3 = 3 %}`,
			"extended4.html": `{% extends "extended5.html" %}{% var V4 = 4 %}`,
			"extended5.html": `{{ V3 }}{{ V4 }}`,
		},
		expectedBuildErr: "extended5.html:1:4: undefined: V3",
	},

	"Multiple extends - with imports": {
		sources: fstest.Files{
			"index.html":     `{% extends "extended1.html" %}{% import "imported.html" %}{% var S0 = "0" + I %}`,
			"extended1.html": `{% extends "extended2.html" %}{% var S1 = S0 + "1" %}`,
			"extended2.html": `{% extends "extended3.html" %}{% import "imported.html" %}{% var S2 = S1 + "2" + I %}`,
			"extended3.html": `{% extends "extended4.html" %}{% var S3 = S2 + "3" %}`,
			"extended4.html": `{% extends "extended5.html" %}{% var S4 = S3 + "4" %}`,
			"extended5.html": `{% import "imported.html" %}{{ S4 }}{{ I }}`,
			"imported.html":  `{% var I = "imported" %}`,
		},
		expectedOut: "0imported12imported34imported",
	},

	"Multiple extends - using 'default' in extended file that extends another file": {
		sources: fstest.Files{
			"index.html":     `{% extends "extended1.html" %}`,
			"extended1.html": `{% extends "extended2.html" %}{% macro M %}{{ Undef() default "hello" }}{% end macro %}`,
			"extended2.html": `M: {{ M() }}`,
		},
		expectedOut: "M: hello",
	},

	"Import a file with an extends statement": {
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" %}`,
			"imported.html": `{% extends "extended.html" %}`,
			"extended.html": ``,
		},
		expectedBuildErr: "imported and rendered files can not have extends",
	},

	"Render a file with an extends statement": {
		sources: fstest.Files{
			"index.html":    `{{ render "partial.html" }}`,
			"partial.html":  `{% extends "extended.html" %}`,
			"extended.html": ``,
		},
		expectedBuildErr: "imported and rendered files can not have extends",
	},

	"Cyclic extends is not allowed": {
		sources: fstest.Files{
			"index.html": `
				{% extends "extended1.html" %}
			`,
			"extended1.html": `
				{% extends "extended2.html" %}
			`,
			"extended2.html": `
				{% extends "extended1.html" %}
			`,
		},
		expectedBuildErr: "file index.html\n\textends extended1.html\n\textends extended2.html\n\textends extended1.html: cycle not allowed",
	},

	"https://github.com/open2b/scriggo/issues/857": {
		sources: fstest.Files{
			"index.html": `{% import "fmt" for Sprint, Fprint %}{% _ = Sprint %}{% _ = Fprint %}`,
		},
		importer: testPackages,
	},

	"https://github.com/open2b/scriggo/issues/852 (1)": {
		sources: fstest.Files{
			"index.html": `
				{% import "imported.html" %}
				{% macro M %}
					{{ V }}
				{% end macro %}
			`,
			"imported.html": `{% var V int %}`,
		},
		expectedOut: "\n\t\t\t",
	},

	"https://github.com/open2b/scriggo/issues/852 (2)": {
		sources: fstest.Files{
			"index.html": `
				{% import pkg "imported.html" %}
				{% macro M %}
					{{ pkg.V }}
				{% end macro %}
			`,
			"imported.html": `{% var V int %}`,
		},
		expectedOut: "\n\t\t\t",
	},

	"https://github.com/open2b/scriggo/issues/852 (3)": {
		sources: fstest.Files{
			"index.html": `
				{% import "imported.html" %}
				{% ref := &V %}
				{% _ = ref %}
			`,
			"imported.html": `{% var V int %}`,
		},
		expectedOut: "\n\t\t\t",
	},

	"https://github.com/open2b/scriggo/issues/852 (4)": {
		sources: fstest.Files{
			"index.html": `
				{% import pkg "imported.html" %}
				{% ref := &pkg.V %}
				{% _ = ref %}
			`,
			"imported.html": `{% var V int %}`,
		},
		expectedOut: "\n\t\t\t",
	},

	"https://github.com/open2b/scriggo/issues/852 (5)": {
		sources: fstest.Files{
			"index.html": `
				{% import . "imported.html" %}
				{% macro M %}
					{{ V }}
				{% end macro %}
			`,
			"imported.html": `{% var V int %}`,
		},
		expectedOut: "\n\t\t\t",
	},

	"https://github.com/open2b/scriggo/issues/852 (6)": {
		sources: fstest.Files{
			"index.html": `
				{% import "imported.html" %}
				{% macro M1 %}
					{% macro M2 %}
						{% macro M3 %}
							{{ V }}
						{% end macro %}
					{% end macro %}
				{% end macro %}
			`,
			"imported.html": `{% var V int %}`,
		},
		expectedOut: "\n\t\t\t",
	},

	"https://github.com/open2b/scriggo/issues/888": {
		// The emitter used to emit two Convert instructions for every
		// conversion in this code before fixing #888.
		sources: fstest.Files{
			"index.html": `{%%
				var s1 html     = "1"
				var s2 css      = "2"
				var s3 js       = "3"
				var s4 json     = "4"
				var s5 markdown = "5"
				show string(s1)
				show string(s2)
				show string(s3)
				show string(s4)
				show string(s5)
			%%}`,
		},
		expectedOut: "12345",
	},

	"Shebang": {
		sources: fstest.Files{
			"index.txt":  "#! /usr/bin/scriggo\n{% extends \"layout.txt\" %}{% import \"import.txt\" %}{% macro M %}{{ A() }}{% end %}",
			"layout.txt": "#!/usr/bin/env scriggo\n{{ render \"render.txt\" }}{{ M() }}",
			"import.txt": "{% macro A %}a{% end %}",
			"render.txt": "#! /usr/bin/scriggo\n{{ \"b\" }}",
		},
		expectedOut: "ba",
	},

	"For-else -- else not executed": {
		sources: fstest.Files{
			"index.txt": `{% var xs = []int{10, 20, 30} %}
			{% for x in xs %}{{ x }} {% else %}NOT EXPECTED (1){% end for %}`,
		},
		expectedOut: "\n\t\t\t10 20 30 ",
	},
	"For-else -- else executed": {
		sources: fstest.Files{
			"index.txt": `{% var xs = []int{} %}
			{% for x in xs %}NOT EXPECTED{% else %}i'm the else block{% end for %}`,
		},
		expectedOut: "\n\t\t\ti'm the else block",
	},
	"For-else on string -- else not executed": {
		sources: fstest.Files{
			"index.txt": `{% var xs = "this is a string" %}
			{% for x in xs %}{{ x }} {% else %}NOT EXPECTED (1){% end for %}`,
		},
		expectedOut: "\n\t\t\t116 104 105 115 32 105 115 32 97 32 115 116 114 105 110 103 ",
	},
	"For-else on string -- else executed": {
		sources: fstest.Files{
			"index.txt": `{% var xs = "" %}
			{% for x in xs %}NOT EXPECTED{% else %}i'm the else block{% end for %}`,
		},
		expectedOut: "\n\t\t\ti'm the else block",
	},
	"For-else -- else not executed (break)": {
		sources: fstest.Files{
			"index.txt": `{% var xs = []int{10, 20, 30} %}
			{% for x in xs %}{% break %}{% else %}NOT EXPECTED (1){% end for %}`,
		},
		expectedOut: "\n\t\t\t",
	},
	"For-else -- else not executed (multiline statement)": {
		sources: fstest.Files{
			"index.txt": `{% var xs = []int{10, 20, 30} %}
			{%%
				for x in xs {
					show x, " "
				} else {
					show "NOT EXPECTED (1)"
				}
			%%}`,
		},
		expectedOut: "\n\t\t\t10 20 30 ",
	},
	"For-else -- else executed (multiline statement)": {
		sources: fstest.Files{
			"index.txt": `{% var xs = []int{} %}
			{%% for x in xs {
				show "NOT EXPECTED"
			} else {
				show "i'm the else block"
			} %%}`,
		},
		expectedOut: "\n\t\t\ti'm the else block",
	},
	"For-else channel -- else not executed": {
		sources: fstest.Files{
			"index.txt": `{%%
            var ch = make(chan int, 1)
            ch <- 5
			close(ch)
            for x in ch {
                show x
			} else {
				show "NOT EXPECTED"
			}
            %%}`,
		},
		expectedOut: "5",
	},
	"For-else channel -- else executed": {
		sources: fstest.Files{
			"index.txt": `{%%
            var ch = make(chan int)
            close(ch)
            for x in ch {
                show "NOT EXPECTED"
			} else {
				show "i'm the else block"
			}
            %%}`,
		},
		expectedOut: "i'm the else block",
	},
	"Key selector": {
		sources: fstest.Files{
			"index.txt": `{%% 
			m := map[string]interface{}{"a":6}
			show m.a
			m.a = 1
			show m.a
			n := map[interface{}]interface{}{"a":3,'a':1}
			show n.a
			n.a = 2
			show n.a
            %%}`,
		},
		expectedOut: "6132",
	},
}

var structWithUnexportedFields = &struct {
	foo int
}{foo: 100}

// testGetValueCalled is used in a test.
// See https://github.com/open2b/scriggo/issues/643
var testGetValueCalled = false

// testTrue is true if the field Value is true.
type testTrue struct {
	Value bool
}

func (tt testTrue) IsTrue() bool {
	return tt.Value
}

// testTrueIf42 is true only if its field Value is 42.
type testTrueIf42 struct {
	Value int
}

func (s testTrueIf42) IsTrue() bool {
	return s.Value == 42
}

// testNotImplementIsTrue has as method called 'IsTrue', but its type is
// 'IsTrue() int' instead of 'IsTrue() bool' so it cannot be used to check if a
// value of its type is true. This is not an error: simply such method will be
// ignored by the Scriggo runtime.
type testNotImplementIsTrue struct{}

func (testNotImplementIsTrue) IsTrue() int {
	panic("BUG: this method should never be called")
}

var testPackages = native.Packages{
	"fmt": native.Package{
		Name: "fmt",
		Declarations: native.Declarations{
			"Sprint": fmt.Sprint,
			"Fprint": fmt.Fprint,
		},
	},
	"math": native.Package{
		Name: "math",
		Declarations: native.Declarations{
			"Abs": math.Abs,
		},
	},
}

var globalVariable = "global variable"

var functionReturningErrorPackage = native.Package{
	Name: "main",
	Declarations: native.Declarations{
		"atoi": func(v string) (int, error) { return strconv.Atoi(v) },
		"uitoa": func(n int) (string, error) {
			if n < 0 {
				return "", errors.New("uitoa requires a positive integer as argument")
			}
			return strconv.Itoa(n), nil
		},
		"baderror": func() (int, error) {
			return 0, errors.New("i'm a bad error -->")
		},
	},
}

func TestMultiFileTemplate(t *testing.T) {
	for name, cas := range templateMultiFileCases {
		if cas.expectedOut != "" && cas.expectedBuildErr != "" {
			panic("invalid test: " + name)
		}
		t.Run(name, func(t *testing.T) {
			entryPoint := cas.entryPoint
			if entryPoint == "" {
				for p := range cas.sources {
					if strings.TrimSuffix(p, path.Ext(p)) == "index" {
						entryPoint = p
					}
				}
			}
			globals := globals()
			for k, v := range cas.main.Declarations {
				globals[k] = v
			}
			opts := &scriggo.BuildOptions{
				Globals:              globals,
				Packages:             cas.importer,
				MarkdownConverter:    markdownConverter,
				NoParseShortShowStmt: cas.noParseShow,
			}
			template, err := scriggo.BuildTemplate(cas.sources, entryPoint, opts)
			switch {
			case err == nil && cas.expectedBuildErr == "":
				// Ok, no errors expected: continue with the test.
			case err != nil && cas.expectedBuildErr == "":
				t.Fatalf("unexpected build error: %q", err)
			case err == nil && cas.expectedBuildErr != "":
				t.Fatalf("expected error %q but not errors have been returned by Build", cas.expectedBuildErr)
			case err != nil && cas.expectedBuildErr != "":
				if strings.Contains(err.Error(), cas.expectedBuildErr) {
					// Ok, the error returned by Build contains the expected error.
					return // this test is end.
				}
				t.Fatalf("expected error %q, got %q", cas.expectedBuildErr, err)
			}
			w := &bytes.Buffer{}
			err = template.Run(w, cas.vars, &scriggo.RunOptions{Print: printFunc(w)})
			if err != nil {
				t.Fatalf("run error: %s", err)
			}
			if cas.expectedOut != w.String() {
				t.Fatalf("expecting %q, got %q", cas.expectedOut, w.String())
			}
		})
	}
}

// printFunc returns a function that print its argument to the writer w with
// the same format used by the builtin print to print to the standard error.
// The returned function can be used for the PrintFunc option.
func printFunc(w io.Writer) scriggo.PrintFunc {
	return func(v interface{}) {
		r := reflect.ValueOf(v)
		switch r.Kind() {
		case reflect.Invalid, reflect.Array, reflect.Func, reflect.Interface, reflect.Ptr, reflect.Struct:
			_, _ = fmt.Fprintf(w, "%#x", reflect.ValueOf(&v).Elem().InterfaceData()[1])
		case reflect.Bool:
			_, _ = fmt.Fprintf(w, "%t", r.Bool())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			_, _ = fmt.Fprintf(w, "%d", r.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			_, _ = fmt.Fprintf(w, "%d", r.Uint())
		case reflect.Float32, reflect.Float64:
			_, _ = fmt.Fprintf(w, "%e", r.Float())
		case reflect.Complex64, reflect.Complex128:
			fmt.Printf("%e", r.Complex())
		case reflect.Chan, reflect.Map, reflect.UnsafePointer:
			_, _ = fmt.Fprintf(w, "%#x", r.Pointer())
		case reflect.Slice:
			_, _ = fmt.Fprintf(w, "[%d/%d] %#x", r.Len(), r.Cap(), r.Pointer())
		case reflect.String:
			_, _ = fmt.Fprint(w, r.String())
		}
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
	fsys := fstest.Files{"example.txt": `{% _, _, _, _, _ = a, c, d, e, f %}`}
	globals := native.Declarations{
		"a": &a, // expected
		"b": &b,
		"c": c,
		"d": &d, // expected
		"e": &e, // expected
		"f": f,
		"g": g,
	}
	opts := &scriggo.BuildOptions{
		Globals: globals,
	}
	template, err := scriggo.BuildTemplate(fsys, "example.txt", opts)
	if err != nil {
		t.Fatal(err)
	}
	vars := template.UsedVars()
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

var envCallPathCases = []struct {
	name    string
	sources fstest.Files
	want    string
}{

	{
		name: "Just one file",
		sources: fstest.Files{
			"index.html": `{{ path() }}`,
		},
		want: "index.html",
	},

	{
		name: "File rendering another file",
		sources: fstest.Files{
			"index.html":   `{{ path() }}, {{ render "partial.html" }}, {{ path() }}`,
			"partial.html": `{{ path() }}`,
		},
		want: `index.html, partial.html, index.html`,
	},

	{
		name: "File rendering a file in a sub-directory",
		sources: fstest.Files{
			"index.html":             `{{ path() }}, {{ render "partials/partial1.html" }}, {{ path() }}`,
			"partials/partial1.html": `{{ path() }}, {{ render "partial2.html" }}`,
			"partials/partial2.html": `{{ path() }}`,
		},
		want: `index.html, partials/partial1.html, partials/partial2.html, index.html`,
	},

	{
		name: "File importing another file, which defines a macro",
		sources: fstest.Files{
			"index.html":    `{% import "imported.html" %}{{ path() }}, {% show Path() %}, {{ path() }}`,
			"imported.html": `{% macro Path %}{{ path() }}{% end %}`,
		},
		want: `index.html, imported.html, index.html`,
	},

	{
		name: "File extending another file",
		sources: fstest.Files{
			"index.html":    `{% extends "extended.html" %}{% macro Path %}{{ path() }}{% end %}`,
			"extended.html": `{{ path() }}, {% show Path() %}`,
		},
		want: `extended.html, index.html`,
	},
}

func Test_envCallPath(t *testing.T) {
	globals := native.Declarations{
		"path": func(env native.Env) string { return env.CallPath() },
	}
	for _, cas := range envCallPathCases {
		t.Run(cas.name, func(t *testing.T) {
			opts := &scriggo.BuildOptions{
				Globals: globals,
			}
			template, err := scriggo.BuildTemplate(cas.sources, "index.html", opts)
			if err != nil {
				t.Fatal(err)
			}
			w := &bytes.Buffer{}
			err = template.Run(w, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(cas.want, w.String()); diff != "" {
				t.Fatalf("(-want, +got):\n%s", diff)
			}
		})
	}
}

func Test_treeTransformer(t *testing.T) {
	stdout := &strings.Builder{}
	fsys := fstest.Files{"index.html": `{% w := "hi, " %}{{ w }}world!`}
	opts := &scriggo.BuildOptions{
		TreeTransformer: func(tree *ast.Tree) error {
			assignment := tree.Nodes[0].(*ast.Assignment)
			assignment.Rhs[0].(*ast.BasicLiteral).Value = `"hello, "`
			text := tree.Nodes[2].(*ast.Text)
			text.Text = []byte("scriggo!")
			return nil
		},
	}
	template, err := scriggo.BuildTemplate(fsys, "index.html", opts)
	if err != nil {
		t.Fatal(err)
	}
	err = template.Run(stdout, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	expectedOutput := "hello, scriggo!"
	if stdout.String() != expectedOutput {
		t.Fatalf("expecting output %q, got %q", expectedOutput, stdout.String())
	}
}

var mdStart = []byte("--- start Markdown ---\n")
var mdEnd = []byte("--- end Markdown ---\n")

// markdownConverter is a scriggo.Converter that it used to check that the
// markdown converter is called. To do this, markdownConverter does not
// convert but only wraps the Markdown code.
func markdownConverter(src []byte, out io.Writer) error {
	_, err := out.Write(mdStart)
	if err == nil {
		_, err = out.Write(src)
	}
	if err == nil {
		_, err = out.Write(mdEnd)
	}
	return err
}

// testErrorWriter is a writer that always return an error.
type testErrorWriter struct {
	err error
}

func (t testErrorWriter) Write(p []byte) (int, error) { return 0, t.err }

// TestOutError tests that in case of an out error, the Run method returns it.
func TestOutError(t *testing.T) {
	fsys := fstest.Files{"index.txt": "a"}
	template, err := scriggo.BuildTemplate(fsys, "index.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	out := testErrorWriter{errors.New("out error")}
	panicked := true
	var rec interface{}
	func() {
		defer func() {
			rec = recover()
		}()
		err = template.Run(out, nil, nil)
		panicked = false
	}()
	if panicked {
		t.Fatalf("expecting error, got panic: %v", rec)
	}
	if err == nil {
		t.Fatalf("expecting error, got no error")
	}
	if err != out.err {
		t.Fatalf("expecting out error, got error %q", err)
	}
}

// TestRecoveredOutError tests that in case of an out error, it can be
// recovered.
func TestRecoveredOutError(t *testing.T) {
	fsys := fstest.Files{"index.txt": "{% defer func() { recover() }() %}a"}
	template, err := scriggo.BuildTemplate(fsys, "index.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	out := testErrorWriter{errors.New("out error")}
	panicked := true
	var rec interface{}
	func() {
		defer func() {
			rec = recover()
		}()
		err = template.Run(out, nil, nil)
		panicked = false
	}()
	if panicked {
		t.Fatalf("expecting no error, got panic: %v", rec)
	}
	if err != nil {
		t.Fatalf("expecting no error, got error %v", err)
	}
}
