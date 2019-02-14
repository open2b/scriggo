// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"scrigo/ast"
	"scrigo/parser"
)

type stringConvertible string

type aMap struct {
	v string
	H func() string `scrigo:"G"`
}

func (s aMap) F() string {
	return s.v
}

func (s aMap) G() string {
	return s.v
}

type aStruct struct {
	a string
	B string `scrigo:"b"`
	C string
}

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
	{"2 / 3", "0", nil},
	{"2.0 / 3", "0.6666666666666666", nil},
	{"2 / 3.0", "0.6666666666666666", nil},
	{"2.0 / 3.0", "0.6666666666666666", nil},
	{"7 % 3", "1", nil},
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
	{"-1 + -2 * 6 / ( 6 - 1 - ( 5 * 3 ) + 2 ) * ( 1 + 2 ) * 3", "8", nil},
	{"-1 + -2 * 6 / ( 6 - 1 - ( 5 * 3 ) + 2.0 ) * ( 1 + 2 ) * 3", "12.5", nil},
	{"a[1]", "y", scope{"a": []string{"x", "y", "z"}}},
	{"a[:]", "x, y, z", scope{"a": []string{"x", "y", "z"}}},
	{"a[1:]", "y, z", scope{"a": []string{"x", "y", "z"}}},
	{"a[:2]", "x, y", scope{"a": []string{"x", "y", "z"}}},
	{"a[1:2]", "y", scope{"a": []string{"x", "y", "z"}}},
	{"a[1:3]", "y, z", scope{"a": []string{"x", "y", "z"}}},
	{"a[0:3]", "x, y, z", scope{"a": []string{"x", "y", "z"}}},
	{"a[2:2]", "", scope{"a": []string{"x", "y", "z"}}},
	{"a[0]", "120", scope{"a": "x€z"}},
	{"a[1]", "226", scope{"a": "x€z"}},
	{"a[2]", "130", scope{"a": "x€z"}},
	{"a[2.2/1.1]", "z", scope{"a": []string{"x", "y", "z"}}},
	{"a[1]", "98", scope{"a": HTML("<b>")}},
	{"a[0]", "60", scope{"a": HTML("<b>")}},
	{"a[1]", "98", scope{"a": stringConvertible("abc")}},
	{"a[:]", "x€z", scope{"a": "x€z"}},
	{"a[1:]", "€z", scope{"a": "x€z"}},
	{"a[:2]", "x\xe2", scope{"a": "x€z"}},
	{"a[1:2]", "\xe2", scope{"a": "x€z"}},
	{"a[1:3]", "\xe2\x82", scope{"a": "x€z"}},
	{"a[0:3]", "x\xe2\x82", scope{"a": "x€z"}},
	{"a[1:]", "\x82\xacxz", scope{"a": "€xz"}},
	{"a[:2]", "xz", scope{"a": "xz€"}},
	{"a[2:2]", "", scope{"a": "xz€"}},
	{"a[1:]", "b>", scope{"a": HTML("<b>")}},
	//{"a[1:]", "z€", scope{"a": stringConvertible("xz€")}},
	{`a.(string)`, "abc", scope{"a": "abc"}},
	{`a.(string)`, "<b>", scope{"a": HTML("<b>")}},
	{`a.(int)`, "5", scope{"a": 5}},
	{`a.(int64)`, "5", scope{"a": int64(5)}},
	{`a.(int32)`, "5", scope{"a": int32(5)}},
	{`a.(int16)`, "5", scope{"a": int16(5)}},
	{`a.(int8)`, "5", scope{"a": int8(5)}},
	{`a.(uint)`, "5", scope{"a": uint(5)}},
	{`a.(uint64)`, "5", scope{"a": uint64(5)}},
	{`a.(uint32)`, "5", scope{"a": uint32(5)}},
	{`a.(uint16)`, "5", scope{"a": uint16(5)}},
	{`a.(uint8)`, "5", scope{"a": uint8(5)}},
	{`a.(float64)`, "5.5", scope{"a": 5.5}},
	{`a.(float32)`, "5.5", scope{"a": float32(5.5)}},
	{`(5).(int)`, "5", nil},
	{`(5.5).(float64)`, "5.5", nil},
	{`'a'.(rune)`, "'a'", nil},
	{`a.(bool)`, "true", scope{"a": true}},
	{`a.(error)`, "err", scope{"a": errors.New("err")}},

	// slice (builtin type Slice)
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
	{`slice{0: "zero", 2: "two"}[2]`, "two", nil},
	{`slice{2: "two", "three", "four"}[4]`, "four", nil},

	// slice
	{"[]int{-3}[0]", "-3", nil},
	{`[]string{"a","b","c"}[0]`, "a", nil},
	{`[][]int{[]int{1,2}, []int{3,4,5}}[1][2]`, "5", nil},
	{`len([]string{"a", "b", "c"})`, "3", nil},
	{`[]string{0: "zero", 2: "two"}[2]`, "two", nil},
	{`[]int{ 8: 64, 81, 5: 25,}[9]`, "81", nil},
	{`[]bytes{{97, 98}, {110, 67}}[1][0]`, "110", nil},

	// bytes ([]byte)
	{`bytes{0, 4}[0]`, "0", nil},
	{`bytes{0, 124: 97}[124]`, "97", nil},

	// array
	{`[2]int{-30, 30}[0]`, "-30", nil},
	{`[1][2]int{[2]int{-30, 30}}[0][1]`, "30", nil},
	{`[4]string{0: "zero", 2: "two"}[2]`, "two", nil},
	{`[...]int{4: 5}[4]`, "5", nil},

	// map (builtin type Map)
	{"len(map{})", "0", nil},
	{`map{1: 1, 2: 4, 3: 9}[2]`, "4", nil},

	// map
	{`map[int]int{1: 1, 2: 4, 3: 9}[2]`, "4", nil},
	{`10 + map[string]int{"uno": 1, "due": 2}["due"] * 3`, "16", nil},
	{`len(map{1: 1, 2: 4, 3: 9})`, "3", nil},
	{`s["a"]`, "3", scope{"s": map[interface{}]int{"a": 3}}},
	{`s[nil]`, "3", scope{"s": map[interface{}]int{nil: 3}}},

	// struct
	{`s{1, 2}.A`, "1", nil},

	// composite literal with implicit type
	{`[][]int{{1},{2,3}}[1][1]`, "3", nil},
	{`[][]string{{"a", "b"}}[0][0]`, "a", nil},
	{`map[string][]int{"a":{1,2}}["a"][1]`, "2", nil},
	{`map[[2]int]string{{1,2}:"a"}[[2]int{1,2}]`, "a", nil},
	{`[2][1]int{{1}, {5}}[1][0]`, "5", nil},
	{`[]Point{{1,2}, {3,4}, {5,6}}[2].Y`, "6", nil},
	{`(*(([]*Point{{3,4}})[0])).X`, "3", nil},

	// make
	{`make([]int, 5)[0]`, "0", nil},
	{`make([]int, 5, 10)[0]`, "0", nil},
	{`make(map[string]int, 5)["key"]`, "0", nil},

	// selectors
	{"a.B", "b", scope{"a": struct{ B string }{B: "b"}}},
	{"a.b", "b", scope{"a": struct {
		B string `scrigo:"b"`
	}{B: "b"}}},
	{"a.b", "b", scope{"a": struct {
		C string `scrigo:"b"`
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
	{"a == nil", "true", scope{"a": nil}},
	{"a != nil", "false", scope{"a": nil}},
	{"nil == a", "true", scope{"a": nil}},
	{"nil != a", "false", scope{"a": nil}},
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
	{"map{} == nil", "false", nil},
	{"map{} == map{}", "false", nil},
	{"slice{} == nil", "false", nil},
	{"slice{} == slice{}", "false", nil},
	{"bytes{} == nil", "false", nil},
	{"bytes{} == bytes{}", "false", nil},

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
	{`a + "<b>"`, "<a><b>", scope{"a": HTML("<a>")}},
	{"a + b", "<a><b>", scope{"a": "<a>", "b": "<b>"}},
	{"a + b", "<a><b>", scope{"a": HTML("<a>"), "b": HTML("<b>")}},

	// call
	{"f()", "ok", scope{"f": func() string { return "ok" }}},
	{"f(5)", "5", scope{"f": func(i int) int { return i }}},
	{"f(5.4)", "5.4", scope{"f": func(n float64) float64 { return n }}},
	{"f(5)", "5", scope{"f": func(n int) int { return n }}},
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
	{"f(5.2)", "5.2", scope{"f": func(d float64) float64 { return d }}},

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
	{"f(a)", "3", scope{"f": func(n int8) int8 { return n + 1 }, "a": int8(2)}},
	{"f(a)", "3", scope{"f": func(n int16) int16 { return n + 1 }, "a": int16(2)}},
	{"f(a)", "3", scope{"f": func(n int32) int32 { return n + 1 }, "a": int32(2)}},
	{"f(a)", "3", scope{"f": func(n int64) int64 { return n + 1 }, "a": int64(2)}},
	{"f(a)", "3", scope{"f": func(n uint8) uint8 { return n + 1 }, "a": uint8(2)}},
	{"f(a)", "3", scope{"f": func(n uint16) uint16 { return n + 1 }, "a": uint16(2)}},
	{"f(a)", "3", scope{"f": func(n uint32) uint32 { return n + 1 }, "a": uint32(2)}},
	{"f(a)", "3", scope{"f": func(n uint64) uint64 { return n + 1 }, "a": uint64(2)}},
	{"f(a)", "3", scope{"f": func(n float32) float32 { return n + 1 }, "a": float32(2.0)}},
	{"f(a)", "3", scope{"f": func(n float64) float64 { return n + 1 }, "a": float64(2.0)}},
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
	{"{% if a, ok := b[`c`]; ok %}ok{% else %}no{% end %}", "ok", scope{"b": Map{"c": true}}},
	{"{% if a, ok := b[`d`]; ok %}no{% else %}ok{% end %}", "ok", scope{"b": Map{}}},
	{"{% if a, ok := b[`c`]; a %}ok{% else %}no{% end %}", "ok", scope{"b": Map{"c": true}}},
	{"{% if a, ok := b[`d`]; a %}no{% else %}ok{% end %}", "ok", scope{"b": Map{"d": false}}},
	{"{% if a, ok := b[`c`][`d`]; ok %}no{% else %}ok{% end %}", "ok", scope{"b": Map{"c": Map{}}}},
	{"{% if a, ok := b.(string); ok %}ok{% else %}no{% end %}", "ok", scope{"b": "abc"}},
	{"{% if a, ok := b.(string); ok %}no{% else %}ok{% end %}", "ok", scope{"b": 5}},
	{"{% if a, ok := b.(int); ok %}ok{% else %}no{% end %}", "ok", scope{"b": 5}},
	{"{% if a, ok := b[`c`].(int); ok %}ok{% else %}no{% end %}", "ok", scope{"b": Map{"c": 5}}},
	{"{% if a, ok := b.(byte); ok %}ok{% else %}no{% end %}", "ok", scope{"b": byte(5)}},
	{"{% if a, ok := b[`c`].(byte); ok %}ok{% else %}no{% end %}", "ok", scope{"b": Map{"c": byte(5)}}},
	{"{% b := map{html(`<b>`): true} %}{% if a, ok := b[html(`<b>`)]; ok %}ok{% else %}no{% end %}", "ok", nil},
	{"{% b := map{5.2: true} %}{% if a, ok := b[5.2]; ok %}ok{% else %}no{% end %}", "ok", nil},
	{"{% b := map{5: true} %}{% if a, ok := b[5]; ok %}ok{% else %}no{% end %}", "ok", nil},
	{"{% b := map{true: true} %}{% if a, ok := b[true]; ok %}ok{% else %}no{% end %}", "ok", nil},
	{"{% b := map{nil: true} %}{% if a, ok := b[nil]; ok %}ok{% else %}no{% end %}", "ok", nil},
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
	{"{% a := 0 %}{% a, a = test2(1,2) %}{{ a }}", "2", nil},
	{"{% a, b := test2(1,2) %}{{ a }},{{ b }}", "1,2", nil},
	{"{% a := 0 %}{% a, b := test2(1,2) %}{{ a }},{{ b }}", "1,2", nil},
	{"{% b := 0 %}{% a, b := test2(1,2) %}{{ a }},{{ b }}", "1,2", nil},
	{"{% s := slice{1,2,3} %}{% s[0] = 5 %}{{ s[0] }}", "5", nil},
	{"{% s := slice{1,2,3} %}{% s2 := s[0:2] %}{% s2[0] = 5 %}{{ s2 }}", "5, 2", nil},
	{`{% x := []string{"a","c","b"} %}{{ x[0] }}{{ x[2] }}{{ x[1] }}`, "abc", nil},
	{"{% for i, p := range products %}{{ i }}: {{ p }}\n{% end %}", "0: a\n1: b\n2: c\n",
		scope{"products": []string{"a", "b", "c"}}},
	{"{% for _, p := range products %}{{ p }}\n{% end %}", "a\nb\nc\n",
		scope{"products": []string{"a", "b", "c"}}},
	{"{% for _, p := range products %}a{% break %}b\n{% end %}", "a",
		scope{"products": []string{"a", "b", "c"}}},
	{"{% for _, p := range products %}a{% continue %}b\n{% end %}", "aaa",
		scope{"products": []string{"a", "b", "c"}}},
	{"{% for _, c := range \"\" %}{{ c }}{% end %}", "", nil},
	{"{% for _, c := range \"a\" %}({{ c }}){% end %}", "(97)", nil},
	{"{% for _, c := range \"aÈc\" %}({{ c }}){% end %}", "(97)(200)(99)", nil},
	{"{% for _, c := range html(\"<b>\") %}({{ c }}){% end %}", "(60)(98)(62)", nil},
	{"{% for _, i := range slice{ `a`, `b`, `c` } %}{{ i }}{% end %}", "abc", nil},
	{"{% for _, i := range slice{ html(`<`), html(`&`), html(`>`) } %}{{ i }}{% end %}", "<&>", nil},
	{"{% for _, i := range slice{1, 2, 3, 4, 5} %}{{ i }}{% end %}", "12345", nil},
	{"{% for _, i := range slice{1.3, 5.8, 2.5} %}{{ i }}{% end %}", "1.35.82.5", nil},
	{"{% for _, i := range bytes{ 0, 1, 2 } %}{{ i }}{% end %}", "012", nil},
	{"{% s := slice{} %}{% for k, v := range map{`a`: `1`, `b`: `2`} %}{% s = append(s, k+`:`+v) %}{% end %}{% sort(s) %}{{ s }}", "a:1, b:2", nil},
	{"{% for k, v := range map{} %}{{ k }}:{{ v }},{% end %}", "", nil},
	{"{% s := slice{} %}{% for k, v := range m %}{% s = append(s, itoa(k)+`:`+itoa(v)) %}{% end %}{% sort(s) %}{{ s }}", "1:1, 2:4, 3:9", scope{"m": map[int]int{1: 1, 2: 4, 3: 9}}},
	{"{% for p in products %}{{ p }}\n{% end %}", "a\nb\nc\n",
		scope{"products": []string{"a", "b", "c"}}},
	{"{% i := 0 %}{% c := \"\" %}{% for i, c = range \"ab\" %}({{ c }}){% end %}{{ i }}", "(97)(98)1", nil},
	{"{% for range slice{ `a`, `b`, `c` } %}.{% end %}", "...", nil},
	{"{% for range bytes{ 1, 2, 3 } %}.{% end %}", "...", nil},
	{"{% for range slice{} %}.{% end %}", "", nil},
	{"{% for i := 0; i < 5; i++ %}{{ i }}{% end %}", "01234", nil},
	{"{% for i := 0; i < 5; i++ %}{{ i }}{% break %}{% end %}", "0", nil},
	{"{% for i := 0; ; i++ %}{{ i }}{% if i == 4 %}{% break %}{% end %}{% end %}", "01234", nil},
	{"{% for i := 0; i < 5; i++ %}{{ i }}{% if i == 4 %}{% continue %}{% end %},{% end %}", "0,1,2,3,4", nil},
	{"{% switch %}{% end %}", "", nil},
	{"{% switch %}{% case true %}ok{% end %}", "ok", nil},
	{"{% switch ; %}{% case true %}ok{% end %}", "ok", nil},
	{"{% i := 2 %}{% switch i++; %}{% case true %}{{ i }}{% end %}", "3", nil},
	{"{% switch ; true %}{% case true %}ok{% end %}", "ok", nil},
	{"{% switch %}{% default %}default{% case true %}true{% end %}", "true", nil},
	{"{% switch interface{}(\"hey\").(type) %}{% default %}default{% case string %}string{% end %}", "string", nil},
	{"{% switch a := 5; a := a.(type) %}{% case int %}ok{% end %}", "ok", nil},
	{"{% switch 3 %}{% case 3 %}three{% end %}", "three", nil},
	{"{% switch 4 + 5 %}{% case 4 %}{% case 9 %}nine{% case 9 %}second nine{% end %}", "nine", nil},
	{"{% switch x := 1; x + 1 %}{% case 1 %}one{% case 2 %}two{% end %}", "two", nil},
	{"{% switch %}{% case 4 < 2%}{% case 7 < 10 %}7 < 10{% default %}other{% end %}", "7 < 10", nil},
	{"{% switch %}{% case 4 < 2%}{% case 7 > 10 %}7 > 10{% default %}other{% end %}", "other", nil},
	{"{% switch %}{% case true %}ok{% end %}", "ok", nil},
	{"{% switch %}{% case false %}no{% end %}", "", nil},
	{"{% switch %}{% case true %}ab{% break %}c{% end %}", "ab", nil},
	{"{% switch a, b := 2, 4; c < d %}{% case true %}{{ a }}{% case false %}{{ b }}{% end %}", "4", scope{"c": 100, "d": 90}},
	{"{% switch a := 4; %}{% case 3 < 4 %}{{ a }}{% end %}", "4", nil},
	{"{% switch a.(type) %}{% case string %}is a string{% case int %}is an int{% default %}is something else{% end %}", "is an int", scope{"a": 3}},
	{"{% switch (a + b).(type) %}{% case string %}{{ a + b }} is a string{% case int %}is an int{% default %}is something else{% end %}", "msgmsg2 is a string", scope{"a": "msg", "b": "msg2"}},
	{"{% switch x.(type) %}{% case string %}is a string{% default %}is something else{% case int %}is an int{% end %}", "is something else", scope{"x": false}},
	{"{% switch v := a.(type) %}{% case string %}{{ v }} is a string{% case int %}{{ v }} is an int{% default %}{{ v }} is something else{% end %}", "12 is an int", scope{"a": 12}},
	{"{% switch %}{% case 4 < 10 %}4 < 10, {% fallthrough %}{% case 4 == 10 %}4 == 10{% end %}", "4 < 10, 4 == 10", nil},
	{"{% switch a, b := 10, \"hey\"; (a + 20).(type) %}{% case string %}string{% case int %}int, msg: {{ b }}{% default %}def{% end %}", "int, msg: hey", nil},
	{"{% i := 0 %}{% c := true %}{% for c %}{% i++ %}{{ i }}{% c = i < 5 %}{% end %}", "12345", nil},
	{"{% i := 0 %}{% for ; ; %}{% i++ %}{{ i }}{% if i == 4 %}{% break %}{% end %},{% end %} {{ i }}", "1,2,3,4 4", nil},
	{"{% i := 5 %}{% i++ %}{{ i }}", "6", nil},
	{"{% s := map{`a`: 5} %}{% s[`a`]++ %}{{ s[`a`] }}", "6", nil},
	{"{% s := slice{5} %}{% s[0]++ %}{{ s[0] }}", "6", nil},
	{"{% s := bytes{5} %}{% s[0]++ %}{{ s[0] }}", "6", nil},
	{"{% s := bytes{255} %}{% s[0]++ %}{{ s[0] }}", "0", nil},
	{"{% i := 5 %}{% i-- %}{{ i }}", "4", nil},
	{"{% s := map{`a`: 5} %}{% s[`a`]-- %}{{ s[`a`] }}", "4", nil},
	{"{% s := slice{5} %}{% s[0]-- %}{{ s[0] }}", "4", nil},
	{"{% s := bytes{5} %}{% s[0]-- %}{{ s[0] }}", "4", nil},
	{"{% s := bytes{0} %}{% s[0]-- %}{{ s[0] }}", "255", nil},
	{`{% a := [3]int{4,5,6} %}{% b := getref(a) %}{{ b[1] }}`, "5", scope{"getref": func(s [3]int) *[3]int { return &s }}},
	{`{% a := [3]int{4,5,6} %}{% b := getref(a) %}{% b[1] = 10 %}{{ (*b)[1] }}`, "10", scope{"getref": func(s [3]int) *[3]int { return &s }}},
	{`{% s := T{5, 6} %}{% if s.A == 5 %}ok{% end %}`, "ok", nil},
	{`{% s := interface{}(3) %}{% if s == 3 %}ok{% end %}`, "ok", nil},
	{"{% a := 12 %}{% a += 9 %}{{ a }}", "21", nil},
	{"{% a := `ab` %}{% a += `c` %}{% if _, ok := a.(string); ok %}{{ a }}{% end %}", "abc", nil},
	{"{% a := html(`ab`) %}{% a += `c` %}{% if _, ok := a.(string); ok %}{{ a }}{% end %}", "abc", nil},
	{"{% a := 12 %}{% a -= 3 %}{{ a }}", "9", nil},
	{"{% a := 12 %}{% a *= 2 %}{{ a }}", "24", nil},
	{"{% a := 12 %}{% a /= 4 %}{{ a }}", "3", nil},
	{"{% a := 12 %}{% a %= 5 %}{{ a }}", "2", nil},
	{"{% a := 12.3 %}{% a += 9.1 %}{{ a }}", "21.4", nil},
	{"{% a := 12.3 %}{% a -= 3.7 %}{{ a }}", "8.600000000000001", nil},
	{"{% a := 12.3 %}{% a *= 2.1 %}{{ a }}", "25.830000000000002", nil},
	{"{% a := 12.3 %}{% a /= 4.9 %}{{ a }}", "2.510204081632653", nil},
	{`{% a := 5 %}{% b := getref(a) %}{{ *b }}`, "5", scope{"getref": func(a int) *int { return &a }}},
	{`{% a := 1 %}{% b := &a %}{% *b = 5 %}{{ a }}`, "5", nil},
	{`{% a := 2 %}{% f(&a) %}{{ a }}`, "3", scope{"f": func(a *int) { *a++ }}},
	{"{% b := &[]int{0,1,4,9}[1] %}{% *b = 5  %}{{ *b }}", "5", nil},
	{"{% a := [ ]int{0,1,4,9} %}{% b := &a[1] %}{% *b = 5  %}{{ a[1] }}", "5", nil},
	{"{% a := [4]int{0,1,4,9} %}{% b := &a[1] %}{% *b = 10 %}{{ a[1] }}", "10", nil},
	{"{% p := Point{4.0, 5.0} %}{% px := &p.X %}{% *px = 8.6 %}{{ p.X }}", "8.6", nil},
	{`{% a := &A{3, 4} %}ok`, "ok", nil},
	{`{% a := &A{3, 4} %}{{ (*a).X }}`, "3", nil},
	{`{% a := &A{3, 4} %}{{ a.X }}`, "3", nil},
	{`{% a := 2 %}{% c := &(*(&a)) %}{% *c = 5 %}{{ a }}`, "5", nil},
	{"{# comment #}", "", nil},
	{"a{# comment #}b", "ab", nil},
	{`{% switch %}{% case true %}{{ 5 }}{% end %}ok`, "5ok", nil},

	// conversions

	// string
	{`{% if s, ok := string("abc").(string); ok %}{{ s }}{% end %}`, "abc", nil},
	{`{% if s, ok := string(html("<b>")).(string); ok %}{{ s }}{% end %}`, "<b>", nil},
	{`{% if s, ok := string(88).(string); ok %}{{ s }}{% end %}`, "X", nil},
	{`{% if s, ok := string(88888888888).(string); ok %}{{ s }}{% end %}`, "\uFFFD", nil},
	//{`{% if s, ok := string(slice{}).(string); ok %}{{ s }}{% end %}`, "", nil},
	//{`{% if s, ok := string(slice{35, 8364}).(string); ok %}{{ s }}{% end %}`, "#€", nil},
	//{`{% if s, ok := string(a).(string); ok %}{{ s }}{% end %}`, "#€", scope{"a": []int{35, 8364}}},
	{`{% if s, ok := string(bytes{}).(string); ok %}{{ s }}{% end %}`, "", nil},
	{`{% if s, ok := string(bytes{97, 226, 130, 172, 98}).(string); ok %}{{ s }}{% end %}`, "a€b", nil},
	{`{% if s, ok := string(a).(string); ok %}{{ s }}{% end %}`, "a€b", scope{"a": []byte{97, 226, 130, 172, 98}}},

	// int
	{`{% if s, ok := int(5).(int); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := int(5.0).(int); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := int(2147483647).(int); ok %}{{ s }}{% end %}`, "2147483647", nil},
	{`{% if s, ok := int(-2147483648).(int); ok %}{{ s }}{% end %}`, "-2147483648", nil},

	// float64
	{`{% if s, ok := float64(5).(float64); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := float64(5.5).(float64); ok %}{{ s }}{% end %}`, "5.5", nil},

	// float32
	{`{% if s, ok := float32(5).(float32); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := float32(5.5).(float32); ok %}{{ s }}{% end %}`, "5.5", nil},

	// rune
	{`{% if s, ok := rune(5).(rune); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := rune(2147483647).(rune); ok %}{{ s }}{% end %}`, "2147483647", nil},
	{`{% if s, ok := rune(-2147483648).(rune); ok %}{{ s }}{% end %}`, "-2147483648", nil},

	// byte
	{`{% if s, ok := byte(5).(byte); ok %}{{ s }}{% end %}`, "5", nil},
	{`{% if s, ok := byte(255).(byte); ok %}{{ s }}{% end %}`, "255", nil},

	// map
	{`{% if _, ok := map(a).(map); ok %}ok{% end %}`, "ok", scope{"a": Map{}}},
	{`{% if map(a) != nil %}ok{% end %}`, "ok", scope{"a": Map{}}},
	{`{% a := map(nil) %}ok`, "ok", nil},

	// slice (builtin)
	{`{% if _, ok := slice(a).(slice); ok %}ok{% end %}`, "ok", scope{"a": Slice{}}},
	{`{% if slice(a) != nil %}ok{% end %}`, "ok", scope{"a": Slice{}}},

	// slice
	{`{% if _, ok := []int{1,2,3}.([]int); ok %}ok{% end %}`, "ok", nil},

	// bytes
	{`{% if _, ok := bytes(a).(bytes); ok %}ok{% end %}`, "ok", scope{"a": []byte{}}},
	{`{% if bytes(a) != nil %}ok{% end %}`, "ok", scope{"a": []byte{}}},
	{`{% if _, ok := bytes(a).(bytes); ok %}ok{% end %}`, "ok", scope{"a": []byte(nil)}},
	{`{% if bytes(a) == nil %}ok{% end %}`, "ok", scope{"a": []byte(nil)}},
	{`{% if bytes(a) != nil %}ok{% end %}`, "ok", scope{"a": []byte{1, 2}}},
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
			A int    `scrigo:"a"`
			B string `scrigo:"b"`
			C bool
		}{A: 1, B: "s", C: true},
		scope{"a": 1, "b": "s", "C": true},
	},
	{
		&struct {
			A int    `scrigo:"a"`
			B string `scrigo:"b"`
			C bool
		}{A: 1, B: "s", C: true},
		scope{"a": 1, "b": "s", "C": true},
	},
	{
		reflect.ValueOf(struct {
			A int    `scrigo:"a"`
			B string `scrigo:"b"`
			C bool
		}{A: 1, B: "s", C: true}),
		scope{"a": 1, "b": "s", "C": true},
	},
}

func TestRenderExpressions(t *testing.T) {
	builtins["T"] = reflect.TypeOf(struct{ A, B int }{})
	builtins["A"] = reflect.TypeOf(struct{ X, Y int }{})
	builtins["s"] = reflect.TypeOf(struct{ A, B int }{})
	builtins["Point"] = reflect.TypeOf(struct{ X, Y float64 }{})
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

var rendererCallFuncTests = []struct {
	src  string
	res  string
	vars scope
}{
	{"func f() {}; f()", "", nil},
	{"func f(x int) int { return x }; print(f(2))", "2", nil},
	{"func f(_ int) { }; f(2)", "", nil},
	{"func f(int) {}; f(1)", "", nil},
	{"func f(x, y int) int { return x + y }; print(f(1, 2))", "3", nil},
	{"func f(_, _ int) { }; f(1, 2)", "", nil},
	{"func f(x int, y int) int { return x + y }; print(f(1, 2))", "3", nil},
	{"func f(_ int, y int) int { return y }; print(f(1, 2))", "2", nil},
	{"func f(...int) {}; f(1, 2, 3)", "", nil},
	{"func f(x ...int) int { s := 0; for _, i := range x { s += i }; return s }; print(f(1, 2, 3))", "6", nil},
	{"func f(_ ...int) { }; f(1, 2, 3)", "", nil},
	{"func f(x, y ...int) int { s := 0; for _, i := range y { s += i }; return s }; print(f(1, 2, 3, 4))", "9", nil},
	{"func f(_, _ ...int) { }; f(1, 2, 3, 4)", "", nil},
	{"func f(_, y ...int) int { s := 0; for _, i := range y { s += i }; return s }; print(f(1, 2, 3, 4))", "9", nil},
	{"func f(x, _ ...int) int { return x }; print(f(1, 2, 3, 4))", "1", nil},
	{"func f(x []int) int { s := 0; for _, i := range x { s += i }; return s }; print(f([]int{1, 2, 3, 4}))", "10", nil},
	{"func f(x, y []int) int { s := 0; for _, i := range y { s += i }; return s }; print(f([]int{1}, []int{2, 3, 4}))", "9", nil},
}

func TestRenderCallFunc(t *testing.T) {
	for _, stmt := range rendererCallFuncTests {
		var tree, err = parser.ParseSource([]byte(stmt.src), ast.ContextNone)
		if err != nil {
			t.Errorf("source: %q, %s\n", stmt.src, err)
			continue
		}
		r, w, err := os.Pipe()
		if err != nil {
			panic(err)
		}
		os.Stdout = w
		var b = &bytes.Buffer{}
		c := make(chan struct{})
		go func() {
			_, err = b.ReadFrom(r)
			if err != nil {
				panic(err)
			}
			c <- struct{}{}
		}()
		err = RenderTree(nil, tree, stmt.vars, true)
		if err != nil {
			t.Errorf("source: %q, %s\n", stmt.src, err)
			continue
		}
		err = os.Stdout.Close()
		if err != nil {
			panic(err)
		}
		_ = <-c
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
