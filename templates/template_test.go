// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package templates

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/runtime"

	"github.com/google/go-cmp/cmp"
)

func globals() Declarations {
	return Declarations{
		"max": func(x, y int) int {
			if x < y {
				return y
			}
			return x
		},
		"sort": func(slice interface{}) {
			// no reflect
			switch s := slice.(type) {
			case nil:
			case []string:
				sort.Strings(s)
			case []rune:
				sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
			case []byte:
				sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
			case []HTML:
				sort.Slice(s, func(i, j int) bool { return string(s[i]) < string(s[j]) })
			case []int:
				sort.Ints(s)
			case []float64:
				sort.Float64s(s)
			}
			// reflect
			sortSlice(slice)
		},
		"sprint": func(a ...interface{}) string {
			return fmt.Sprint(a...)
		},
		"title": func(env runtime.Env, s string) string {
			return strings.Title(s)
		},
	}
}

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
			fsys := MapFS{"index.html": "{{" + cas.src + "}}"}
			template, err := Build(fsys, "index.html", nil)
			if err != nil {
				t.Fatalf("source %q: loading error: %s", cas.src, err)
			}
			b := &bytes.Buffer{}
			err = template.Run(b, nil, nil)
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
			fsys := MapFS{"index.html": cas.src}
			template, err := Build(fsys, "index.html", nil)
			if err != nil {
				t.Fatalf("source %q: loading error: %s", cas.src, err)
			}
			b := &bytes.Buffer{}
			err = template.Run(b, nil, nil)
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
	sources          map[string]string
	expectedBuildErr string                 // default to empty string (no build error). Mutually exclusive with expectedOut.
	expectedOut      string                 // default to "". Mutually exclusive with expectedBuildErr.
	main             scriggo.MapPackage     // default to nil
	vars             map[string]interface{} // default to nil
	entryPoint       string                 // default to "index.html"
	packages         scriggo.PackageLoader  // default to nil
}{

	"Empty template": {
		sources: map[string]string{
			"index.txt": ``,
		},
	},
	"Text only": {
		sources: map[string]string{
			"index.txt": `Hello, world!`,
		},
		expectedOut: `Hello, world!`,
	},

	"Template comments": {
		sources: map[string]string{
			"index.txt": `{# this is a comment #}`,
		},
		expectedOut: ``,
	},

	"Template comments with text": {
		sources: map[string]string{
			"index.txt": `Text before comment{# comment #} text after comment{# another comment #}`,
		},
		expectedOut: `Text before comment text after comment`,
	},

	"'Show' node only": {
		sources: map[string]string{
			"index.txt": `{{ "i am a show" }}`,
		},
		expectedOut: `i am a show`,
	},

	"Text and show": {
		sources: map[string]string{
			"index.txt": `Hello, {{ "world" }}!!`,
		},
		expectedOut: `Hello, world!!`,
	},

	"If statements - true": {
		sources: map[string]string{
			"index.txt": `{% if true %}true{% else %}false{% end %}`,
		},
		expectedOut: `true`,
	},

	"If statements - false": {
		sources: map[string]string{
			"index.txt": `{% if !true %}true{% else %}false{% end %}`,
		},
		expectedOut: `false`,
	},

	"Variable declarations": {
		sources: map[string]string{
			"index.txt": `{% var a = 10 %}{% var b = 20 %}{{ a + b }}`,
		},
		expectedOut: "30",
	},

	"For loop": {
		sources: map[string]string{
			"index.txt": "For loop: {% for i := 0; i < 5; i++ %}{{ i }}, {% end %}",
		},
		expectedOut: "For loop: 0, 1, 2, 3, 4, ",
	},

	"Template global - max": {
		sources: map[string]string{
			"index.txt": `Maximum between 10 and -3 is {{ max(10, -3) }}`,
		},
		expectedOut: `Maximum between 10 and -3 is 10`,
	},

	"Template global - sort": {
		sources: map[string]string{
			"index.txt": `{% s := []string{"a", "c", "b"} %}{{ sprint(s) }} sorted is {% sort(s) %}{{ sprint(s) }}`,
		},
		expectedOut: `[a c b] sorted is [a b c]`,
	},

	"Function literal": {
		sources: map[string]string{
			"index.txt": `{% func() {} %}`,
		},
		expectedBuildErr: "func literal evaluated but not used",
	},

	"Function call": {
		sources: map[string]string{
			"index.txt": `{% func() { print(5) }() %}`,
		},
		expectedOut: `5`,
	},

	"Multi rows": {
		sources: map[string]string{
			"index.txt": `{%
	print(3) %}`,
		},
		expectedOut: `3`,
	},

	"Multi rows 2": {
		sources: map[string]string{
			"index.txt": `{%
	print(3)
%}`,
		},
		expectedOut: `3`,
	},

	"Multi rows with comments": {
		sources: map[string]string{
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
		sources: map[string]string{
			"index.txt": `calling f: {{ f() }}, done!`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"f": func() string { return "i'm f!" },
			},
		},
		expectedOut: `calling f: i'm f!, done!`,
	},

	"Reading a variable declared in main": {
		sources: map[string]string{
			"index.txt": `{{ mainVar }}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"mainVar": (*int)(nil),
			},
		},
		expectedOut: `0`,
	},

	"Reading a variable declared in main and initialized with vars": {
		sources: map[string]string{
			"index.txt": `{{ initMainVar }}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"initMainVar": (*int)(nil),
			},
		},
		vars: map[string]interface{}{
			"initMainVar": 42,
		},
		expectedOut: `42`,
	},

	"Calling a global function": {
		sources: map[string]string{
			"index.txt": `{{ lowercase("HellO ScrIgGo!") }}{% x := "A String" %}{{ lowercase(x) }}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"lowercase": func(s string) string {
					return strings.ToLower(s)
				},
			},
		},
		expectedOut: `hello scriggo!a string`,
	},

	"Calling a function stored in a global variable": {
		sources: map[string]string{
			"index.txt": `{{ lowercase("HellO ScrIgGo!") }}{% x := "A String" %}{{ lowercase(x) }}`,
		},
		main: scriggo.MapPackage{
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
		expectedOut: `hello scriggo!a string`,
	},

	"https://github.com/open2b/scriggo/issues/391": {
		sources: map[string]string{
			"index.txt": `{{ a }}{{ b }}`,
		},
		main: scriggo.MapPackage{
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
		expectedOut: `AAABBB`,
	},

	"Macro definition (no arguments)": {
		sources: map[string]string{
			"index.txt": `Macro def: {% macro M %}M's body{% end %}end.`,
		},
		expectedOut: `Macro def: end.`,
	},

	"Macro definition (no arguments) and show-macro": {
		sources: map[string]string{
			"index.txt": `{% macro M %}body{% end %}{% show M() %}`,
		},
		expectedOut: `body`,
	},

	"Macro definition (with arguments)": {
		sources: map[string]string{
			"index.txt": `{% macro M(v int) %}v is {{ v }}{% end %}`,
		},
	},

	"Macro definition (with one string argument) and show-macro": {
		sources: map[string]string{
			"index.txt": `{% macro M(v string) %}v is {{ v }}{% end %}{% show M("msg") %}`,
		},
		expectedOut: `v is msg`,
	},

	"Macro definition (with two string arguments) and show-macro": {
		sources: map[string]string{
			"index.txt": `{% macro M(a, b string) %}a is {{ a }} and b is {{ b }}{% end %}{% show M("avalue", "bvalue") %}`,
		},
		expectedOut: `a is avalue and b is bvalue`,
	},

	"Macro definition (with one int argument) and show-macro": {
		sources: map[string]string{
			"index.txt": `{% macro M(v int) %}v is {{ v }}{% end %}{% show M(42) %}`,
		},
		expectedOut: `v is 42`,
	},

	"Macro definition (with one []int argument) and show-macro": {
		sources: map[string]string{
			"index.txt": `{% macro M(v []int) %}v is {{ sprint(v) }}{% end %}{% show M([]int{42}) %}`,
		},
		expectedOut: `v is [42]`,
	},

	"Two macro definitions": {
		sources: map[string]string{
			"index.txt": `{% macro M1 %}M1's body{% end %}{% macro M2(i int, s string) %}i: {{ i }}, s: {{ s }}{% end %}`,
		},
	},

	"Two macro definitions and three show-macro": {
		sources: map[string]string{
			"index.txt": `{% macro M1 %}M1's body{% end %}{% macro M2(i int, s string) %}i: {{ i }}, s: {{ s }}{% end %}Show macro: {% show M1() %} {% show M2(-30, "hello") %} ... {% show M1() %}`,
		},
		expectedOut: `Show macro: M1's body i: -30, s: hello ... M1's body`,
	},

	"Macro definition and show-macro without parameters": {
		sources: map[string]string{
			"index.txt": `{% macro M %}ok{% end %}{% show M() %}`,
		},
		expectedOut: `ok`,
	},

	"Macro definition and show-macro without parentheses": {
		sources: map[string]string{
			"index.txt": `{% macro M %}ok{% end %}{% show M() %}`,
		},
		expectedOut: `ok`,
	},

	"Macro definition and show-macro variadic": {
		sources: map[string]string{
			"index.txt": `{% macro M(v ...int) %}{% for _ , i := range v %}{{ i }}{% end for %}{% end macro %}{% show M([]int{1,2,3}...) %}`,
		},
		expectedOut: `123`,
	},

	"Template global - title": {
		sources: map[string]string{
			"index.txt": `{% s := "hello, world" %}{{ s }} converted to title is {{ title(s) }}`,
		},
		expectedOut: `hello, world converted to title is Hello, World`,
	},

	"Label for": {
		sources: map[string]string{
			"index.txt": `{% L: for %}a{% break L %}b{% end for %}`,
		},
		expectedOut: `a`,
	},

	"Label switch": {
		sources: map[string]string{
			"index.txt": `{% L: switch 1 %}{% case 1 %}a{% break L %}b{% end switch %}`,
		},
		expectedOut: `a`,
	},

	"Render - Only text": {
		sources: map[string]string{
			"index.txt":   `a{{ render "/partial.txt" }}c`,
			"partial.txt": `b`,
		},
		expectedOut: "abc",
	},

	"Render - Render file that uses external variable": {
		sources: map[string]string{
			"index.txt":   `{% var a = 10 %}a: {{ render "/partial.txt" }}`,
			"partial.txt": `{{ a }}`,
		},
		expectedBuildErr: "undefined: a",
	},

	"Render - File with a render expression try to use a variable declared in the rendered file": {
		sources: map[string]string{
			"index.txt":   `{{ render "/partial.txt" }}partial a: {{ a }}`,
			"partial.txt": `{% var a = 20 %}`,
		},
		expectedBuildErr: "undefined: a",
	},

	"Render - File renders a file which renders another file": {
		sources: map[string]string{
			"index.txt":             `indexstart,{{ render "/dir1/partial.txt" }}indexend,`,
			"dir1/partial.txt":      `i1start,{{ render "/dir1/dir2/partial.txt" }}i1end,`,
			"dir1/dir2/partial.txt": `i2,`,
		},
		expectedOut: "indexstart,i1start,i2,i1end,indexend,",
	},

	"Import/Macro - Importing a macro defined in another page": {
		sources: map[string]string{
			"index.txt": `{% import "/page.txt" %}{% show M() %}{% show M() %}`,
			"page.txt":  `{% macro M %}macro!{% end %}{% macro M2 %}macro 2!{% end %}`,
		},
		expectedOut: "macro!macro!",
	},

	"Import/Macro - Importing a macro defined in another page, where a function calls a before-declared function": {
		sources: map[string]string{
			"index.txt": `{% import "/page.txt" %}{% show M() %}{% show M() %}`,
			"page.txt": `
				{% macro M2 %}macro 2!{% end %}
				{% macro M %}{% show M2() %}{% end %}
			`,
		},
		expectedOut: "macro 2!macro 2!",
	},

	"Import/Macro - Importing a macro defined in another page, where a function calls an after-declared function": {
		sources: map[string]string{
			"index.txt": `{% import "/page.txt" %}{% show M() %}{% show M() %}`,
			"page.txt": `
				{% macro M %}{% show M2() %}{% end %}
				{% macro M2 %}macro 2!{% end %}
			`,
		},
		expectedOut: "macro 2!macro 2!",
	},

	"Import/Macro - Importing a macro defined in another page, which imports a third page": {
		sources: map[string]string{
			"index.txt": `{% import "/page1.txt" %}index-start,{% show M1() %}index-end`,
			"page1.txt": `{% import "/page2.txt" %}{% macro M1 %}M1-start,{% show M2() %}M1-end,{% end %}`,
			"page2.txt": `{% macro M2 %}M2,{% end %}`,
		},
		expectedOut: "index-start,M1-start,M2,M1-end,index-end",
	},

	"Import/Macro - Importing a macro using an import statement with identifier": {
		sources: map[string]string{
			"index.txt": `{% import pg "/page.txt" %}{% show pg.M() %}{% show pg.M() %}`,
			"page.txt":  `{% macro M %}macro!{% end %}`,
		},
		expectedOut: "macro!macro!",
	},

	"Import/Macro - Importing a macro using an import statement with identifier (with comments)": {
		sources: map[string]string{
			"index.txt": `{# a comment #}{% import pg "/page.txt" %}{# a comment #}{% show pg.M() %}{# a comment #}{% show pg.M() %}{# a comment #}`,
			"page.txt":  `{# a comment #}{% macro M %}{# a comment #}macro!{# a comment #}{% end %}{# a comment #}`,
		},
		expectedOut: "macro!macro!",
	},

	"Extends - Empty page extends a page containing only text": {
		sources: map[string]string{
			"index.txt": `{% extends "/page.txt" %}`,
			"page.txt":  `I'm page!`,
		},
		expectedOut: "I'm page!",
	},

	"Extends - Extending a page that calls a macro defined on current page": {
		sources: map[string]string{
			"index.txt": `{% extends "/page.txt" %}{% macro E %}E's body{% end %}`,
			"page.txt":  `{% show E() %}`,
		},
		expectedOut: "E's body",
	},

	"Extending an empty page": {
		sources: map[string]string{
			"index.txt":    `{% extends "extended.txt" %}`,
			"extended.txt": ``,
		},
	},

	"Extending a page that imports another file": {
		sources: map[string]string{
			"index.txt":    `{% extends "/extended.txt" %}`,
			"extended.txt": `{% import "/imported.txt" %}`,
			"imported.txt": `{% macro Imported %}Imported macro{% end macro %}`,
		},
	},

	"Extending a page (that imports another file) while declaring a macro": {
		sources: map[string]string{
			"index.txt":    `{% extends "/extended.txt" %}{% macro Index %}{% end macro %}`,
			"extended.txt": `{% import "/imported.txt" %}`,
			"imported.txt": `{% macro Imported %}Imported macro{% end macro %}`,
		},
	},

	"Extends - Extending a page that calls two macros defined on current page": {
		sources: map[string]string{
			"index.txt": `{% extends "/page.txt" %}{% macro E1 %}E1's body{% end %}{% macro E2 %}E2's body{% end %}`,
			"page.txt":  `{% show E1() %}{% show E2() %}`,
		},
		expectedOut: "E1's bodyE2's body",
	},

	"Extends - Define a variable (with zero value) used in macro definition": {
		sources: map[string]string{
			"index.txt": `{% extends "/page.txt" %}{% var Local int %}{% macro E1 %}Local has value {{ Local }}{% end %}`,
			"page.txt":  `{% show E1() %}`,
		},
		expectedOut: "Local has value 0",
	},

	"Extends - Define a variable (with non-zero value) used in macro definition": {
		sources: map[string]string{
			"index.txt": `{% extends "/page.txt" %}{% var Local = 50 %}{% macro E1 %}Local has value {{ Local }}{% end %}`,
			"page.txt":  `{% show E1() %}`,
		},
		expectedOut: "Local has value 50",
	},

	"Extends - Extending a file which contains text and shows": {
		sources: map[string]string{
			"index.txt": `{% extends "/page.txt" %}`,
			"page.txt":  `I am an {{ "extended" }} file.`,
		},
		expectedOut: "I am an extended file.",
	},

	"File imported twice": {
		sources: map[string]string{
			"index.txt": `{% import "/a.txt" %}{% import "/b.txt" %}`,
			"a.txt":     `{% import "/b.txt" %}`,
			"b.txt":     `{% macro M %}I'm b{% end %}`,
		},
	},

	"File imported twice - Variable declaration": {
		sources: map[string]string{
			"index.txt": `{% import "b.txt" %}{% import "c.txt" %}`,
			"b.txt":     `{% import "c.txt" %}`,
			"c.txt":     `{% var V int %}`,
		},
	},

	"https://github.com/open2b/scriggo/issues/392": {
		sources: map[string]string{
			"product.html": `{{ "" }}{{ render "partials/products.html" }}
`, // this newline is intentional
			"partials/products.html": `{% macro M(s []int) %}{% end %}`,
		},
		expectedOut: "\n",
		entryPoint:  "product.html",
	},

	"https://github.com/open2b/scriggo/issues/392 (minimal)": {
		sources: map[string]string{
			"index.html": `text{% macro M(s []int) %}{% end %}text`,
		},
		expectedOut: `texttext`,
	},

	"https://github.com/open2b/scriggo/issues/392 (invalid memory address)": {
		sources: map[string]string{
			"index.html": `{% macro M(s []int) %}{% end %}text`,
		},
		expectedOut: `text`,
	},

	"https://github.com/open2b/scriggo/issues/393": {
		sources: map[string]string{
			"product.html": `{{ render "partials/products.html" }}
`, // this newline is intentional
			"partials/products.html": `{% macro M(s []int) %}{% end %}`,
		},
		expectedOut: "",
		entryPoint:  "product.html",
	},
	"Auto imported packages - Function call": {
		sources: map[string]string{
			"index.txt": `{{ strings.ToLower("HELLO") }}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"strings": scriggo.MapPackage{
					PkgName: "strings",
					Declarations: map[string]interface{}{
						"ToLower": strings.ToLower,
					},
				},
			},
		},
		expectedOut: "hello",
	},
	"Auto imported packages - Variable": {
		sources: map[string]string{
			"index.txt": `{{ data.Name }} Holmes`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"data": scriggo.MapPackage{
					PkgName: "data",
					Declarations: map[string]interface{}{
						"Name": &[]string{"Sherlock"}[0],
					},
				},
			},
		},
		expectedOut: "Sherlock Holmes",
	},
	"Auto imported packages - Type": {
		sources: map[string]string{
			"index.txt": `{% b := &bytes.Buffer{} %}{% b.WriteString("oh!") %}{{ b.String() }}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"bytes": scriggo.MapPackage{
					PkgName: "bytes",
					Declarations: map[string]interface{}{
						"Buffer": reflect.TypeOf(bytes.Buffer{}),
					},
				},
			},
		},
		expectedOut: "oh!",
	},
	"Auto imported packages - Constants": {
		sources: map[string]string{
			"index.txt": `{{ math.MaxInt8 }}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"math": scriggo.MapPackage{
					PkgName: "math",
					Declarations: map[string]interface{}{
						"MaxInt8": math.MaxInt8,
					},
				},
			},
		},
		expectedOut: "127",
	},

	"Syntax {{ f() }} where 'f' returns a value and a nil error": {
		sources: map[string]string{
			"index.txt": `{{ atoi("42") }}`,
		},
		main:        functionReturningErrorPackage,
		expectedOut: "42",
	},

	"Syntax {{ f() }} where 'f' returns a zero value and an error": {
		sources: map[string]string{
			"index.txt": `{{ atoi("what?") }}`,
		},
		main:        functionReturningErrorPackage,
		expectedOut: "",
	},

	"Undefined variable error": {
		sources: map[string]string{
			"index.txt": `Name is {{ name }}`,
		},
		expectedBuildErr: "undefined: name",
	},

	"Render file tries to overwrite a variable of the file that renders it": {
		// The emitter must use another scope when emitting a rendered file,
		// otherwise such file can overwrite the variables of the file that
		// renders it.
		sources: map[string]string{
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
		sources: map[string]string{
			"index.txt":   `{% v := "showing" %}{{ render "partial.txt" }}, {{ v }}`,
			"partial.txt": "{{ v }}",
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"v": &globalVariable,
			},
		},
		expectedOut: "global variable, showing",
	},

	"A partial file defines a macro, which should not be accessible from the file that renders the partial": {
		sources: map[string]string{
			"index.txt":   `{{ render "partial.txt" }}{% show MacroInRenderFile() %}`,
			"partial.txt": `{% macro MacroInRenderFile %}{% end macro %}`,
		},
		expectedBuildErr: "undefined: MacroInRenderFile",
	},

	"The file with a render expression defines a macro, which should not be accessible from the rendered file": {
		sources: map[string]string{
			"index.txt":   `{% macro Macro %}{% end macro %}{{ render "partial.txt" }}`,
			"partial.txt": `{% show Macro() %}`,
		},
		expectedBuildErr: "undefined: Macro",
	},

	"Byte slices are rendered as they are in context HTML": {
		sources: map[string]string{
			"index.html": `{{ sb1 }}{{ sb2 }}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"sb1": &[]byte{97, 98, 99},                      // abc
				"sb2": &[]byte{60, 104, 101, 108, 108, 111, 62}, // <hello>
			},
		},
		expectedOut: `abc<hello>`,
	},

	"Cannot show byte slices in text context": {
		sources: map[string]string{
			"index.txt": `{{ sb1 }}{{ sb2 }}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"sb1": &[]byte{97, 98, 99},                      // abc
				"sb2": &[]byte{60, 104, 101, 108, 108, 111, 62}, // <hello>
			},
		},
		expectedBuildErr: `cannot show sb1 (cannot show type []uint8 as text)`,
	},

	"Using the precompiled package 'fmt'": {
		sources: map[string]string{
			"index.txt": `{% import "fmt" %}{{ fmt.Sprint(10, 20) }}`,
		},
		packages:    testPackages,
		expectedOut: "10 20",
	},

	"Using the precompiled package 'fmt' from a file that extends another file": {
		sources: map[string]string{
			"index.txt":    `{% extends "extended.txt" %}{% import "fmt" %}{% macro M %}{{ fmt.Sprint(321, 11) }}{% end macro %}`,
			"extended.txt": `{% show M() %}`,
		},
		packages:    testPackages,
		expectedOut: "321 11",
	},

	"Using the precompiled packages 'fmt' and 'math'": {
		sources: map[string]string{
			"index.txt": `{% import "fmt" %}{% import m "math" %}{{ fmt.Sprint(-42, m.Abs(-42)) }}`,
		},
		packages:    testPackages,
		expectedOut: "-42 42",
	},

	"Importing the precompiled package 'fmt' with '.'": {
		sources: map[string]string{
			"index.txt": `{% import . "fmt" %}{{ Sprint(50, 70) }}`,
		},
		packages:    testPackages,
		expectedOut: "50 70",
	},

	"Trying to import a precompiled package that is not available in the loader": {
		sources: map[string]string{
			"index.txt": `{% import "mypackage" %}{{ mypackage.F() }}`,
		},
		packages:         testPackages,
		expectedBuildErr: "index.txt:1:11: syntax error: cannot find package \"mypackage\"",
	},

	"Trying to access a precompiled function 'SuperPrint' that is not available in the package 'fmt'": {
		sources: map[string]string{
			"index.txt": `{% import "fmt" %}{{ fmt.SuperPrint(42) }}`,
		},
		packages:         testPackages,
		expectedBuildErr: "index.txt:1:25: undefined: fmt.SuperPrint",
	},

	"Using the precompiled package 'fmt' without importing it returns an error": {
		sources: map[string]string{
			"index.txt": `{{ fmt.Sprint(10, 20) }}`,
		},
		packages:         testPackages,
		expectedBuildErr: "index.txt:1:4: undefined: fmt",
	},

	"Check if a value that has a method 'IsZero() bool' is zero or not": {
		sources: map[string]string{
			"index.txt": "{% if (NeverZero{}) %}OK{% else %}BUG{% end %}\n" +
				"{% if (AlwaysZero{}) %}BUG{% else %}OK{% end %}\n" +
				"{% if (struct{}{}) %}BUG{% else %}OK{% end %}\n" +
				"{% if (struct{Value int}{}) %}BUG{% else %}OK{% end %}\n" +
				"{% if (struct{Value int}{Value: 42}) %}OK{% else %}BUG{% end %}\n" +
				"{% if (ZeroIf42{}) %}OK{% else %}BUG{% end %}\n" +
				"{% if (ZeroIf42{Value: 42}) %}BUG{% else %}OK{% end %}\n" +
				"{% if (NotImplIsZero{}) %}BUG{% else %}OK{% end %}",
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"NeverZero":     reflect.TypeOf((*testNeverZero)(nil)).Elem(),
				"AlwaysZero":    reflect.TypeOf((*testAlwaysZero)(nil)).Elem(),
				"ZeroIf42":      reflect.TypeOf((*testZeroIf42)(nil)).Elem(),
				"NotImplIsZero": reflect.TypeOf((*testNotImplementIsZero)(nil)).Elem(),
			},
		},
		expectedOut: "OK\nOK\nOK\nOK\nOK\nOK\nOK\nOK",
	},

	// https://github.com/open2b/scriggo/issues/640
	"Importing a file that imports a file that declares a variable": {
		sources: map[string]string{
			"index.html":     `{% import "imported1.html" %}`,
			"imported1.html": `{% import "imported2.html" %}`,
			"imported2.html": `{% var X = 0 %}`,
		},
	},

	// https://github.com/open2b/scriggo/issues/640
	"Importing a file that imports a file that declares a macro": {
		sources: map[string]string{
			"index.html":     `{% import "imported1.html" %}{% show M1(42) %}`,
			"imported1.html": `{% import "imported2.html" %}{% macro M1(a int) %}{% show M2(a) %}{% end macro %}`,
			"imported2.html": `{% macro M2(b int) %}b is {{ b }}{% end macro %}`,
		},
		expectedOut: "b is 42",
	},

	// https://github.com/open2b/scriggo/issues/641
	"File imported by two files - test compilation": {
		sources: map[string]string{
			"index.html":   `{% import "/v.html" %}{{ render "/partial.html" }}`,
			"partial.html": `{% import "/v.html" %}`,
			"v.html":       `{% var V int %}`,
		},
	},

	// https://github.com/open2b/scriggo/issues/642
	"Macro imported twice - test compilation": {
		sources: map[string]string{
			"index.html":    `{% import "/imported.html" %}{% import "/macro.html" %}{% show M() %}`,
			"imported.html": `{% import "/macro.html" %}`,
			"macro.html":    `{% macro M %}{% end macro %}`,
		},
	},

	// https://github.com/open2b/scriggo/issues/642
	"Macro imported twice - test output": {
		sources: map[string]string{
			"index.html":    `{% import "/imported.html" %}{% import "/macro.html" %}{% show M(42) %}`,
			"imported.html": `{% import "/macro.html" %}`,
			"macro.html":    `{% macro M(a int) %}a is {{ a }}{% end macro %}`,
		},
		expectedOut: "a is 42",
	},

	// https://github.com/open2b/scriggo/issues/643
	"Invalid variable value when imported": {
		sources: map[string]string{
			"index.html": `{% import "/v.html" %}{{ V }}`,
			"v.html":     `{% var V = 42 %}`,
		},
		expectedOut: "42",
	},

	// https://github.com/open2b/scriggo/issues/643
	"Invalid variable value with multiple imports": {
		sources: map[string]string{
			"index.html":   `{% import "/v.html" %}{{ render "/partial.html" }}V is {{ V }}`,
			"partial.html": `{% import "/v.html" %}`,
			"v.html":       `{% var V = 42 %}`,
		},
		expectedOut: "V is 42",
	},

	// https://github.com/open2b/scriggo/issues/643
	"Init function called more than once": {
		sources: map[string]string{
			"index.html":   `{% import "v.html" %}{{ render "/partial.html" }}{{ V }}`,
			"partial.html": `{% import "/v.html" %}`,
			"v.html":       `{% var V = GetValue() %}`,
		},
		expectedOut: "42",
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
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

	"Can access to unexported struct field declared in the same page - struct literal": {
		sources: map[string]string{
			"index.txt": `{% var s struct { a int } %}{% s.a = 42 %}{{ s.a }}
			{% s2 := &s %}{{ s2.a }}`,
		},
		expectedOut: "42\n\t\t\t42",
	},

	"Can access to unexported struct field declared in the same page - defined type": {
		sources: map[string]string{
			"index.txt": `{% type t struct { a int } %}{% var s t %}{% s.a = 84 %}{{ s.a }}
			{% s2 := &s %}{{ s2.a }}`,
		},
		expectedOut: "84\n\t\t\t84",
	},

	"Cannot access to unexported struct fields of a precompiled value (struct)": {
		sources: map[string]string{
			"index.txt": `{{ s.foo }}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"s": structWithUnexportedFields,
			},
		},
		expectedBuildErr: `s.foo undefined (cannot refer to unexported field or method foo)`,
	},

	"Cannot access to unexported struct fields of a precompiled value (*struct)": {
		sources: map[string]string{
			"index.txt": `{{ s.foo }}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"s": &structWithUnexportedFields,
			},
		},
		expectedBuildErr: `s.foo undefined (cannot refer to unexported field or method foo)`,
	},

	"Cannot access to an unexported field declared in another page (struct)": {
		sources: map[string]string{
			// Note the statement: {% type _ struct { bar int } %}: we try to
			// deceive the type checker into thinking that the type `struct {
			// field int }` can be fully accessed because is the same declared
			// in this package.
			"index.txt":    `{% import "imported.txt" %}{% type _ struct { bar int } %}{{ S.bar }}`,
			"imported.txt": `{% var S struct { bar int } %}`,
		},
		expectedBuildErr: `S.bar undefined (cannot refer to unexported field or method bar)`,
	},

	"Cannot access to an unexported field declared in another page (*struct)": {
		sources: map[string]string{
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
		sources: map[string]string{
			"index.txt": `{% macro M %}{{ globalVariable }}{% end %}{% show M() %}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"globalVariable": &([]string{"<b>global</b>"}[0]),
			},
		},
		expectedOut: "<b>global</b>",
	},
	"Double type checking of render expression": {
		sources: map[string]string{
			"index.txt":   `{{ render "/partial.txt" }}{{ render "/partial.txt" }}`,
			"partial.txt": `{% var v int %}`,
		},
	},
	"https://github.com/open2b/scriggo/issues/661": {
		sources: map[string]string{
			"index.txt": `{% extends "extended.txt" %}
{% macro M %}
{{ render "/partial.txt" }}
{% end macro %}`,
			"extended.txt": `{{ render "/partial.txt" }}`,
			"partial.txt":  `{% var v int %}`,
		},
	},
	"https://github.com/open2b/scriggo/issues/660": {
		sources: map[string]string{
			"index.txt":   `{% macro M() %}{{ render "partial.txt" }}{% end macro %}`,
			"partial.txt": `{% var v int %}{% _ = v %}`,
		},
	},

	// https://github.com/open2b/scriggo/issues/659
	"Accessing global variable from function literal's body": {
		sources: map[string]string{
			"index.txt": `{%
				func(){
					_ = globalVariable
				}() 
			%}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"globalVariable": (*int)(nil),
			},
		},
	},

	// https://github.com/open2b/scriggo/issues/659
	"Accessing global variable from function literal's body - nested": {
		sources: map[string]string{
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
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"globalVariable": (*int)(nil),
			},
		},
	},

	"Macro declaration inside implicit blocks": {
		sources: map[string]string{
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

	"Dollar identifier - Referencing to a global variable that does not exist": {
		sources: map[string]string{
			"index.txt": `{% var _ interface{} = $notExisting %}{{ $notExisting2 == nil }}`,
		},
		expectedOut: "true",
	},

	"Dollar identifier - Referencing to a global variable that exists": {
		sources: map[string]string{
			"index.txt": `{{ $forthyTwo }}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"forthyTwo": &([]int8{42}[0]),
			},
		},
		expectedOut: "42",
	},

	"Dollar identifier - Type assertion on a global variable that exists (1)": {
		sources: map[string]string{
			"index.txt": `{{ $forthyThree.(int) }}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"forthyThree": &([]int{43}[0]),
			},
		},
		expectedOut: "43",
	},

	"Dollar identifier - Type assertion on a global variable that exists (2)": {
		sources: map[string]string{
			"index.txt": `{% var n, ok = $forthyThree.(int) %}{{ n * 32 }}{{ ok }}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"forthyThree": &([]int{42}[0]),
			},
		},
		expectedOut: "1344true",
	},

	"Dollar identifier - Cannot use an type": {
		sources: map[string]string{
			"index.txt": `{% _ = $int %}`,
		},
		expectedBuildErr: `unexpected type in dollar identifier`,
	},

	"Dollar identifier - Cannot use a builtin": {
		sources: map[string]string{
			"index.txt": `{% _ = $println %}`,
		},
		expectedBuildErr: `use of builtin println not in function call`,
	},

	"Dollar identifier - Cannot use a local identifier": {
		sources: map[string]string{
			"index.txt": `{% var local = 10 %}{% _ = $local %}`,
		},
		expectedBuildErr: `use of local identifier within dollar identifier`,
	},

	"Dollar identifier - Cannot take the address (variable exists)": {
		sources: map[string]string{
			"index.txt": `{% _ = &($fortyTwo) %}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"forthyTwo": &([]int8{42}[0]),
			},
		},
		expectedBuildErr: `cannot take the address of $fortyTwo`,
	},

	"Dollar identifier - Cannot take the address (variable does not exist)": {
		sources: map[string]string{
			"index.txt": `{% _ = &($notExisting) %}`,
		},
		expectedBuildErr: `cannot take the address of $notExisting`,
	},

	"Dollar identifier - Cannot assign to dollar identifier (variable exists)": {
		sources: map[string]string{
			"index.txt": `{% $fortyTwo = 43 %}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"forthyTwo": &([]int8{42}[0]),
			},
		},
		expectedBuildErr: `cannot assign to $fortyTwo`,
	},

	"Dollar identifier - Cannot assign to dollar identifier (variable does not exist)": {
		sources: map[string]string{
			"index.txt": `{% $notExisting = 43 %}`,
		},
		expectedBuildErr: `cannot assign to $notExisting`,
	},

	"Dollar identifier - Referencing to a constant returns a non-constant": {
		sources: map[string]string{
			"index.txt": `{% const _ = $constant %}`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"constant": 42,
			},
		},
		expectedBuildErr: `const initializer $constant is not a constant`,
	},

	"https://github.com/open2b/scriggo/issues/679 (1)": {
		sources: map[string]string{
			"index.txt": `{% global := interface{}(global) %}ok`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"global": &[]string{"ciao"},
			},
		},
		expectedOut: "ok",
	},

	"https://github.com/open2b/scriggo/issues/679 (2)": {
		sources: map[string]string{
			"index.txt": `{% var global = interface{}(global) %}ok`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"global": &[]string{},
			},
		},
		expectedOut: "ok",
	},

	"https://github.com/open2b/scriggo/issues/679 (3)": {
		sources: map[string]string{
			"index.txt": `{% _ = []int{} %}{% global := interface{}(global) %}ok`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"global": &[]string{},
			},
		},
		expectedOut: "ok",
	},

	"Dollar identifier referring to package declaration in imported file": {
		sources: map[string]string{
			"index.txt":    `{% import "imported.txt" %}`,
			"imported.txt": `{% var X = 10 %}{% var _ = $X %}`,
		},
		expectedBuildErr: `use of top-level identifier within dollar identifier`,
	},

	"Dollar identifier referring to package declaration in extending file": {
		sources: map[string]string{
			"index.txt":    `{% extends "extended.txt" %}{% var X = 10 %}{% var _ = $X %}`,
			"extended.txt": ``,
		},
		expectedBuildErr: `use of top-level identifier within dollar identifier`,
	},

	"https://github.com/open2b/scriggo/issues/680 - Import": {
		sources: map[string]string{
			"index.txt":    `{% import "imported.txt" %}`,
			"imported.txt": `{% var x = $global %}`,
		},
	},

	"https://github.com/open2b/scriggo/issues/680 - Extends": {
		sources: map[string]string{
			"index.txt":    `{% extends "extended.txt" %}{% var x = $global %}`,
			"extended.txt": ``,
		},
	},

	"Panic after importing file that declares a variable in general register (1)": {
		sources: map[string]string{
			"index.txt":    `before{% import "imported.txt" %}after`,
			"imported.txt": `{% var a []int %}`,
		},
		expectedOut: "beforeafter",
	},

	"Panic after importing file that declares a variable in general register (2)": {
		sources: map[string]string{
			"index.txt":     `a{% import "imported1.txt" %}{% import "imported2.txt" %}b`,
			"imported1.txt": `{% var X []int %}`,
			"imported2.txt": `{% var Y []string %}`,
		},
		expectedOut: "ab",
	},

	"https://github.com/open2b/scriggo/issues/686": {
		sources: map[string]string{
			"index.txt":    `{% extends "extended.txt" %}{% var _ = $global %}`,
			"extended.txt": `text`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"global": (*int)(nil),
			},
		},
		expectedOut: "text",
	},

	"https://github.com/open2b/scriggo/issues/686 (2)": {
		sources: map[string]string{
			"index.txt":    `{% extends "extended.txt" %}{% var _ = interface{}(global) %}`,
			"extended.txt": `text`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"global": (*int)(nil),
			},
		},
		expectedOut: "text",
	},

	"https://github.com/open2b/scriggo/issues/687": {
		sources: map[string]string{
			"index.html": `{% extends "extended.html" %}
			
				{% import "imported.html" %}`,

			"extended.html": `
				<head>
				<script>....
				{{ design.Base }}		
				{{ design.Open2b }}		
				fef`,

			"imported.html": `
				{% var filters, _ = $filters.([]int) %}
			`,
		},
		main: scriggo.MapPackage{
			PkgName: "main",
			Declarations: map[string]interface{}{
				"design": &struct {
					Base   string
					Open2b string
				}{},
			},
		},
		expectedOut: "\n\t\t\t\t<head>\n\t\t\t\t<script>....\n\t\t\t\t\"\"\t\t\n\t\t\t\t\"\"\t\t\n\t\t\t\tfef",
	},

	"https://github.com/open2b/scriggo/issues/655": {
		sources: map[string]string{
			"index.html":  "{% extends \"layout.html\" %}\n{% var _ = func() { } %}",
			"layout.html": `<a href="a">`,
		},
		expectedOut: "<a href=\"a\">",
	},

	"https://github.com/open2b/scriggo/issues/656": {
		sources: map[string]string{
			"index.txt":  "{% extends \"layout.txt\" %}\n{% var _ = func() { } %}",
			"layout.txt": `abc`,
		},
		expectedOut: "abc",
	},

	"Show of a previously imported file": {
		sources: map[string]string{
			"index.txt": `{% import "file.txt" %}{{ render "file.txt" }}`,
			"file.txt":  ``,
		},
		expectedBuildErr: `syntax error: render of file imported at index.txt:1:11`,
	},

	"Show of a previously extended file": {
		sources: map[string]string{
			"index.txt": `{% extends "file.txt" %}{% macro A %}{{ render "file.txt" }}{% end %}`,
			"file.txt":  ``,
		},
		expectedBuildErr: `syntax error: render of file extended at index.txt:1:4`,
	},

	"Import of a previously extended file": {
		sources: map[string]string{
			"index.txt": `{% extends "file.txt" %}{% import "file.txt" %}`,
			"file.txt":  ``,
		},
		expectedBuildErr: `syntax error: import of file extended at index.txt:1:4`,
	},

	"Import of a partial file": {
		sources: map[string]string{
			"index.txt": `{{ render "file1.txt" }}{{ render "file2.txt" }}`,
			"file1.txt": ``,
			"file2.txt": `{% import "file1.txt" %}`,
		},
		expectedBuildErr: `syntax error: import of file rendered at index.txt:1:4`,
	},

	"Not only spaces in a page that extends": {
		sources: map[string]string{
			"index.txt":  `{% extends "layout.txt" %}abc`,
			"layout.txt": ``,
		},
		expectedBuildErr: "syntax error: unexpected text in file with extends",
	},

	"Not only spaces in an imported file": {
		sources: map[string]string{
			"index.txt":    `{% import "imported.txt" %}`,
			"imported.txt": `abc`,
		},
		expectedBuildErr: "syntax error: unexpected text in imported file",
	},

	"Extends preceded by not empty text": {
		sources: map[string]string{
			"index.txt":  `abc{% extends "layout.txt" %}`,
			"layout.txt": ``,
		},
		expectedBuildErr: "syntax error: extends is not at the beginning of the file",
	},

	"Extends preceded by another statement": {
		sources: map[string]string{
			"index.txt":  `{% var a = 5 %}{% extends "layout.txt" %}`,
			"layout.txt": ``,
		},
		expectedBuildErr: "syntax error: extends is not at the beginning of the file",
	},

	"Extends preceded by comment": {
		sources: map[string]string{
			"index.txt":  `{# comment #}{% extends "layout.txt" %}`,
			"layout.txt": `abc`,
		},
		expectedOut: "abc",
	},

	"EOF after {%": {
		sources: map[string]string{
			"index.txt": `{%`,
		},
		expectedBuildErr: "syntax error: unexpected EOF, expecting %}",
	},

	"EOF after {%%": {
		sources: map[string]string{
			"index.txt": `{%%`,
		},
		expectedBuildErr: "syntax error: unexpected EOF, expecting %%}",
	},

	"EOF after {{": {
		sources: map[string]string{
			"index.txt": `{{`,
		},
		expectedBuildErr: "syntax error: unexpected EOF, expecting }}",
	},

	"Multi line statements #1": {
		sources: map[string]string{
			"index.txt": `{%%
				extends "extended.txt"
			%%}{% var x = $global %}`,
			"extended.txt": ``,
		},
	},

	"Multi line statements #2": {
		sources: map[string]string{
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
		sources: map[string]string{
			"index.txt":    `{%% import "imported.txt" %%}`,
			"imported.txt": `{% var x = $global %}`,
		},
	},

	"Multi line statements #4": {
		sources: map[string]string{
			"index.txt":    `{% import "imported.txt" %}`,
			"imported.txt": `{%% var x = $global %%}`,
		},
	},

	"Multiline statements #5": {
		sources: map[string]string{
			"index.txt":    `{%% extends "extended.txt" %%}{% import "fmt" %}{% macro M %}{{ fmt.Sprint(321, 11) }}{% end macro %}`,
			"extended.txt": `{% show M() %}`,
		},
		packages:    testPackages,
		expectedOut: "321 11",
	},

	"Multiline statements #6": {
		sources: map[string]string{
			"index.txt": `{%%
				import "fmt"
				import m "math"
			%%}
			{{ fmt.Sprint(-42, m.Abs(-42)) }}`,
		},
		packages:    testPackages,
		expectedOut: "\t\t\t-42 42",
	},

	"Multi line statements #7": {
		sources: map[string]string{
			"index.txt":    `{% import "imported.txt" %}{% type _ struct { bar int } %}{{ S.bar }}`,
			"imported.txt": `{%% var S struct { bar int } %%}`,
		},
		expectedBuildErr: `S.bar undefined (cannot refer to unexported field or method bar)`,
	},

	"Multi line statements #8": {
		sources: map[string]string{
			"index.txt":    `{% import "imported.txt" %}{% type _ *struct { bar int } %}{{ S.bar }}`,
			"imported.txt": `{%% var S *struct { bar int } %%}`,
		},
		expectedBuildErr: `S.bar undefined (cannot refer to unexported field or method bar)`,
	},

	"https://github.com/open2b/scriggo/issues/694": {
		sources: map[string]string{
			"index.txt":   `{{ render "partial.txt" }}`,
			"partial.txt": `{% var a int %}{% func() { a = 20 }() %}`,
		},
		expectedOut: ``,
	},

	"Error positioned in first non space character": {
		sources: map[string]string{
			"index.txt":    `{% import "imported.txt" %}`,
			"imported.txt": "\n \n\té",
		},
		expectedBuildErr: `3:2: syntax error: unexpected text in imported file`,
	},

	"Show a Scriggo defined type value": {
		sources: map[string]string{
			"index.txt": `{% type Bool bool %}{{ Bool(true) }}`,
		},
		expectedOut: `true`,
	},

	"https://github.com/open2b/scriggo/issues/708 (1)": {
		sources: map[string]string{
			"index.txt":    `{% extends "extended.txt" %}{% macro M %}{%% a := 10 %%}{% end macro %}`,
			"extended.txt": `a`,
		},
		expectedOut: `a`,
	},

	"https://github.com/open2b/scriggo/issues/708 (2)": {
		sources: map[string]string{
			"index.txt":    `{% import "imported.txt" %}`,
			"imported.txt": `{% macro M %}{%% a := 20 %%}{% end %}`,
		},
	},

	"Endless macro declaration": {
		sources: map[string]string{
			"index.html":  `{% extends "layout.html" %}{% Article %}content`,
			"layout.html": `{% show Article() %}`,
		},
		expectedOut: `content`,
	},

	"Endless macro declaration (2)": {
		sources: map[string]string{
			"index.html":  `{% extends "layout.html" %}{% Article %}{% Content %}`,
			"layout.html": `{% show Article() %}`,
		},
		expectedBuildErr: `undefined: Content`,
	},

	"Endless macro declaration (3)": {
		sources: map[string]string{
			"index.html":  `{% extends "layout.html" %}{% Article %}{% end macro %}`,
			"layout.html": `{% show Article() %}`,
		},
		expectedBuildErr: `syntax error: unexpected end`,
	},

	"Endless macro declaration (4)": {
		sources: map[string]string{
			"index.html":  `{% extends "layout.html" %}{% article %}{% end %}`,
			"layout.html": `{% show Article() %}`,
		},
		expectedBuildErr: `syntax error: unexpected article, expecting declaration statement`,
	},

	"Endless macro declaration (5)": {
		sources: map[string]string{
			"index.html":    `{% import "imported.html" %}{% show Article() %}`,
			"imported.html": `{% Article %}`,
		},
		expectedBuildErr: `syntax error: unexpected Article, expecting declaration statement`,
	},

	"Endless macro declaration (6)": {
		sources: map[string]string{
			"index.html":   `{{ render "partial.html" }}`,
			"partial.html": `{% Article %}`,
		},
		expectedBuildErr: `undefined: Article`,
	},

	"Macro in tab code block context": {
		sources: map[string]string{
			"index.md": "\t{% macro A %}{% end %}",
		},
		expectedBuildErr: `syntax error: macro not allowed in tab code block`,
	},

	"Macro in spaces code block context": {
		sources: map[string]string{
			"index.md": `    {% macro A %}{% end %}`,
		},
		expectedBuildErr: `syntax error: macro not allowed in spaces code block`,
	},

	"Macro used in function call - an empty string is returned": {
		sources: map[string]string{
			"index.html": `{% macro M %}{% end %}{% var str = M() %}{{ len(str) }}`,
		},
		expectedOut: `0`,
	},

	"Macro used in function call - a non-empty string is returned (1)": {
		sources: map[string]string{
			"index.html": `{% macro M %}hello{% end %}{% var str = M() %}{{ len(str) }}`,
		},
		expectedOut: `5`,
	},

	"Macro used in function call - a non-empty string is returned (2)": {
		sources: map[string]string{
			"index.html": `{% macro M %}hello{% end %}{% var str = M() %}len(str): {{ len(str) }}, output of macro: {{ M() }}`,
		},
		expectedOut: `len(str): 5, output of macro: hello`,
	},

	"Show as render (TO BE REMOVED)": {
		sources: map[string]string{
			"index.html":   `{% show "partial.html" %}`,
			"partial.html": `partial`,
		},
		expectedOut: `partial`,
	},

	// TODO: in a future version these tests should fail because the
	// transformation must be reverted. This is intended just to make easier to
	// upgrade Scriggo templates.
	"{% show Macro %} should be transformed to {% show Macro() %} for the current version": {
		sources: map[string]string{
			`index.html`: `{% macro M %}hello{% end %}{% show M %}`,
		},
		expectedOut: `hello`,
	},
	"{% show Macro %} should be transformed to {% show Macro() %} for the current version (2)": {
		sources: map[string]string{
			`index.html`:    `{% import "imported.html" %}{% show Macro %}`,
			`imported.html`: `{% macro Macro %}imported macro{% end %}`,
		},
		expectedOut: `imported macro`,
	},

	"Render - Expression": {
		sources: map[string]string{
			"index.txt": `{% page := render "page.txt" %}page.txt has a length of {{ len(page) }}`,
			"page.txt":  `ciao`,
		},
		expectedOut: "page.txt has a length of 4",
	},

	"Render - Rendering the same file twice": {
		sources: map[string]string{
			"index.txt": `{% p1 := render "page.txt" %}{% p2 := render "page.txt" %}p1 is {{ p1 }} (len = {{ len(p1) }}), p2 is {{ p2 }}`,
			"page.txt":  `ciao`,
		},
		expectedOut: "p1 is ciao (len = 4), p2 is ciao",
	},

	"Convert a markdown value to an html value": {
		sources: map[string]string{
			"index.txt": `{% var m markdown = "# title" %}{% h := html(m) %}{{ string(h) }}`,
		},
		expectedOut: "--- start Markdown ---\n# title--- end Markdown ---\n",
	},

	"Convert a markdown value to an html value - Indirect": {
		sources: map[string]string{
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
		sources: map[string]string{
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
		sources: map[string]string{
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

	"https://github.com/open2b/scriggo/issues/723 - Short variable declaration: rendered file exists": {
		sources: map[string]string{
			"index.txt": `{% s, ok := render "file.txt" %}ok: {{ ok }}, s: {{ s }}`,
			"file.txt":  `this is file.txt`,
		},
		expectedOut: "ok: true, s: this is file.txt",
	},

	"https://github.com/open2b/scriggo/issues/723 - Short variable declaration: rendered file does not exist": {
		sources: map[string]string{
			"index.txt": `{% s, ok := render "not-existing.txt" %}ok: {{ ok }}, s: {{ s }}`,
		},
		expectedOut: "ok: false, s: ",
	},

	"https://github.com/open2b/scriggo/issues/723 - Simple assignment: rendered file exists": {
		sources: map[string]string{
			"index.txt": `{% var s string %}{% var ok bool %}{% s, ok = render "file.txt" %}ok: {{ ok }}, s: {{ s }}`,
			"file.txt":  `this is file.txt`,
		},
		expectedOut: "ok: true, s: this is file.txt",
	},

	"https://github.com/open2b/scriggo/issues/723 - Simple assignment: rendered file does not exist": {
		sources: map[string]string{
			"index.txt": `{% var s string %}{% var ok bool %}{% s, ok = render "not-existing.txt" %}ok: {{ ok }}, s: {{ s }}`,
		},
		expectedOut: "ok: false, s: ",
	},

	"https://github.com/open2b/scriggo/issues/725 - Variable declaration: rendered file exists": {
		sources: map[string]string{
			"index.txt": `{% var s, ok = render "file.txt" %}ok: {{ ok }}, s: {{ s }}`,
			"file.txt":  `this is file.txt`,
		},
		expectedOut: "ok: true, s: this is file.txt",
	},

	"https://github.com/open2b/scriggo/issues/725 - Variable declaration: rendered file does not exist": {
		sources: map[string]string{
			"index.txt": `{% var s, ok = render "not-existing.txt" %}ok: {{ ok }}, s: {{ s }}`,
		},
		expectedOut: "ok: false, s: ",
	},
}

var structWithUnexportedFields = &struct {
	foo int
}{foo: 100}

// testGetValueCalled is used in a test.
// See https://github.com/open2b/scriggo/issues/643
var testGetValueCalled = false

// testAlwaysZero is always considered zero.
type testAlwaysZero struct{}

func (testAlwaysZero) IsZero() bool {
	return true
}

// testNeverZero is never considered zero.
type testNeverZero struct{}

func (testNeverZero) IsZero() bool {
	return false
}

// testZeroIf42 is zero only if its field Value is 42.
type testZeroIf42 struct {
	Value int
}

func (s testZeroIf42) IsZero() bool {
	return s.Value == 42
}

// testNotImplementIsZero has as method called 'IsZero', but its type is
// 'IsZero() int' instead of 'IsZero() bool' so it cannot be used to check if a
// value of its type is zero. This is not an error: simply such method will be
// ignored by the Scriggo runtime.
type testNotImplementIsZero struct{}

func (testNotImplementIsZero) IsZero() int {
	panic("BUG: this method should never be called")
}

var testPackages = scriggo.Packages{
	"fmt": scriggo.MapPackage{
		PkgName: "fmt",
		Declarations: map[string]interface{}{
			"Sprint": fmt.Sprint,
		},
	},
	"math": scriggo.MapPackage{
		PkgName: "math",
		Declarations: map[string]interface{}{
			"Abs": math.Abs,
		},
	},
}

var globalVariable = "global variable"

var functionReturningErrorPackage = scriggo.MapPackage{
	PkgName: "main",
	Declarations: map[string]interface{}{
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

func TestMultiPageTemplate(t *testing.T) {
	for name, cas := range templateMultiPageCases {
		if cas.expectedOut != "" && cas.expectedBuildErr != "" {
			panic("invalid test: " + name)
		}
		t.Run(name, func(t *testing.T) {
			entryPoint := cas.entryPoint
			fsys := MapFS{}
			for p, src := range cas.sources {
				fsys[p] = src
				if entryPoint == "" {
					if strings.TrimSuffix(p, path.Ext(p)) == "index" {
						entryPoint = p
					}
				}
			}
			globals := globals()
			if cas.main.PkgName != "" {
				for k, v := range cas.main.Declarations {
					globals[k] = v
				}
			}
			opts := &BuildOptions{
				Globals:           globals,
				Packages:          cas.packages,
				MarkdownConverter: markdownConverter,
			}
			template, err := Build(fsys, entryPoint, opts)
			switch {
			case err == nil && cas.expectedBuildErr == "":
				// Ok, no errors expected: continue with the test.
			case err != nil && cas.expectedBuildErr == "":
				t.Fatalf("unexpected loading error: %q", err)
			case err == nil && cas.expectedBuildErr != "":
				t.Fatalf("expected error %q but not errors have been returned by Build", cas.expectedBuildErr)
			case err != nil && cas.expectedBuildErr != "":
				if strings.Contains(err.Error(), cas.expectedBuildErr) {
					// Ok, the error returned by Build contains the expected error.
					return // this test is end.
				} else {
					t.Fatalf("expected error %q, got %q", cas.expectedBuildErr, err)
				}
			}
			w := &bytes.Buffer{}
			err = template.Run(w, cas.vars, &RunOptions{PrintFunc: scriggo.PrintFunc(w)})
			if err != nil {
				t.Fatalf("rendering error: %s", err)
			}
			if cas.expectedOut != w.String() {
				t.Fatalf("expecting %q, got %q", cas.expectedOut, w.String())
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
	fsys := MapFS{"example.txt": `{% _, _, _, _, _ = a, c, d, e, f %}`}
	globals := Declarations{
		"a": &a, // expected
		"b": &b,
		"c": c,
		"d": &d, // expected
		"e": &e, // expected
		"f": f,
		"g": g,
	}
	opts := &BuildOptions{
		Globals: globals,
	}
	template, err := Build(fsys, "example.txt", opts)
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
		want: "index.html",
	},

	{
		name: "File rendering another file",
		sources: map[string]string{
			"index.html":   `{{ path() }}, {{ render "partial.html" }}, {{ path() }}`,
			"partial.html": `{{ path() }}`,
		},
		want: `index.html, partial.html, index.html`,
	},

	{
		name: "File rendering a file in a sub-directory",
		sources: map[string]string{
			"index.html":             `{{ path() }}, {{ render "partials/partial1.html" }}, {{ path() }}`,
			"partials/partial1.html": `{{ path() }}, {{ render "partial2.html" }}`,
			"partials/partial2.html": `{{ path() }}`,
		},
		want: `index.html, partials/partial1.html, partials/partial2.html, index.html`,
	},

	{
		name: "File importing another file, which defines a macro",
		sources: map[string]string{
			"index.html":    `{% import "imported.html" %}{{ path() }}, {% show Path() %}, {{ path() }}`,
			"imported.html": `{% macro Path %}{{ path() }}{% end %}`,
		},
		want: `index.html, imported.html, index.html`,
	},

	{
		name: "File extending another file",
		sources: map[string]string{
			"index.html":    `{% extends "extended.html" %}{% macro Path %}{{ path() }}{% end %}`,
			"extended.html": `{{ path() }}, {% show Path() %}`,
		},
		want: `extended.html, index.html`,
	},
}

func Test_envFilePath(t *testing.T) {
	globals := Declarations{
		"path": func(env runtime.Env) string { return env.FilePath() },
	}
	for _, cas := range envFilePathCases {
		t.Run(cas.name, func(t *testing.T) {
			fsys := MapFS{}
			for p, src := range cas.sources {
				fsys[p] = src
			}
			opts := &BuildOptions{
				Globals: globals,
			}
			template, err := Build(fsys, "index.html", opts)
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

var mdStart = []byte("--- start Markdown ---\n")
var mdEnd = []byte("--- end Markdown ---\n")

// markdownConverter is a templates.Converter that it used to check that the
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
