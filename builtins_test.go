// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"bytes"
	"io/ioutil"
	"testing"

	"scrigo/ast"
	"scrigo/parser"

	"github.com/cockroachdb/apd"
)

func floatToDecimal(f float64) *apd.Decimal {
	d, err := apd.New(0, 0).SetFloat64(f)
	if err != nil {
		panic(err)
	}
	return d
}

var rendererBuiltinTests = []struct {
	src     string
	res     string
	globals scope
}{
	// abbreviate
	{"abbreviate(``,0)", "", nil},
	{"abbreviate(`abc`,0)", "", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,28)", "Lorem ipsum dolor sit amet.", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,27)", "Lorem ipsum dolor sit amet.", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,26)", "Lorem ipsum dolor sit...", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,10)", "Lorem...", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,8)", "Lorem...", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,7)", "...", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,6)", "...", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,5)", "...", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,4)", "...", nil},
	{"abbreviate(`Lorem ipsum d`,12)", "Lorem...", nil},
	{"abbreviate(`Lorem. Ipsum.`,9)", "Lorem...", nil},
	{"abbreviate(`Lorem, ipsum.`,9)", "Lorem...", nil},

	// abs
	{"abs(0)", "0", nil},
	{"abs(1)", "1", nil},
	{"abs(-1)", "1", nil},
	{"abs(3.56)", "3.56", nil},
	{"abs(-3.56)", "3.56", nil},

	// append
	{"append(s)", "a, b", scope{"s": []string{"a", "b"}}},
	{"append(s, v)", "a, b, ", scope{"s": []string{"a", "b"}, "v": ""}},
	{"append(s, `a`, `b`)", "a, b", scope{"s": []string(nil)}},
	{"append(s, `c`, `d`)", "a, b, c, d", scope{"s": []string{"a", "b"}}},
	{"append(s, html(`<c>`))", "<a>, <b>, <c>", scope{"s": []HTML{"<a>", "<b>"}}},
	{"append(s, 3, 4)", "1, 2, 3, 4", scope{"s": []int{1, 2}}},
	{"append(s, 3.5, 4.23)", "1.9, 2.32, 3.5, 4.23", scope{"s": []float64{1.9, 2.32}}},
	{"append(s, false, true)", "true, false, false, true", scope{"s": []bool{true, false}}},
	{"append(s, 7, true, html(`<b>`))", "a, false, 0, 7, true, <b>", scope{"s": []interface{}{"a", false, 0}}},
	{"append(slice{})", "", nil},
	{"append(slice{}, 1)", "1", nil},
	{"append(slice{1,2,3}, 4)", "1, 2, 3, 4", nil},
	{"append(slice{1,2,3}, 4, 5, 6)", "1, 2, 3, 4, 5, 6", nil},
	{"append(bytes{})", "", nil},
	{"append(bytes{}, 1)", "1", nil},
	{"append(bytes{1,2,3}, 4)", "1, 2, 3, 4", nil},
	{"append(bytes{1,2,3}, 4, 5, 6)", "1, 2, 3, 4, 5, 6", nil},

	// base64
	{"base64(``)", "", nil},
	{"base64(`hello world!`)", "aGVsbG8gd29ybGQh", nil},

	// contains
	{"contains(``,``)", "true", nil},
	{"contains(`a`,``)", "true", nil},
	{"contains(`abc`,`b`)", "true", nil},
	{"contains(`abc`,`e`)", "false", nil},

	// hasPrefix
	{"hasPrefix(``,``)", "true", nil},
	{"hasPrefix(`a`,``)", "true", nil},
	{"hasPrefix(`abc`,`a`)", "true", nil},
	{"hasPrefix(`abc`,`b`)", "false", nil},

	// hasSuffix
	{"hasSuffix(``,``)", "true", nil},
	{"hasSuffix(`a`,``)", "true", nil},
	{"hasSuffix(`abc`,`c`)", "true", nil},
	{"hasSuffix(`abc`,`b`)", "false", nil},

	// hex
	{"hex(``)", "", nil},
	{"hex(`hello world!`)", "68656c6c6f20776f726c6421", nil},

	// hmac
	{"hmac(`MD5`, ``, ``)", "dOb3KYqcLRaJNfWMAButiA==", nil},
	{"hmac(`MD5`, `hello world!`, ``)", "POUE2/xvWDT8UjcXJ4d/hQ==", nil},
	{"hmac(`MD5`, ``, `secret`)", "XI2wPwTOwPQ7ywYAI5FBkA==", nil},
	{"hmac(`MD5`, `hello world!`, `secret`)", "CgRh4Q6JUG18MaFFZjvtkw==", nil},
	{"hmac(`SHA-1`, ``, ``)", "+9sdGxiqbAgyS31ktx+3Y3BpDh0=", nil},
	{"hmac(`SHA-1`, `hello world!`, ``)", "Cs2Lo6MmqAmr0Qj3JXmz/wJnhDg=", nil},
	{"hmac(`SHA-1`, ``, `secret`)", "Ja9hdKD87MTTRmgKcrfOZEuaiOg=", nil},
	{"hmac(`SHA-1`, `hello world!`, `secret`)", "pN9fnSN6sMoyQfBCvPYFmk70kcQ=", nil},
	{"hmac(`SHA-256`, ``, ``)", "thNnmggU2ex3L5XXeMNfxf8Wl8STcVZTxscSFEKSxa0=", nil},
	{"hmac(`SHA-256`, `hello world!`, ``)", "7/WCWbmktkh3Gig/DI7JWORlJ0gUpKhebIYJG4iMxJw=", nil},
	{"hmac(`SHA-256`, ``, `secret`)", "+eZuF5tnR65UEI+C+K3os8Jddv0wr95sOVgixTAZYWk=", nil},
	{"hmac(`SHA-256`, `hello world!`, `secret`)", "cgaXMb8pG0Y67LIYvCJ6vOPUA9dtpn+u8tSNPLQ7L1Q=", nil},

	// index
	{"index(``,``)", "0", nil},
	{"index(`a`,``)", "0", nil},
	{"index(`ab€c`,`a`)", "0", nil},
	{"index(`ab€c`,`b`)", "1", nil},
	{"index(`ab€c`,`€`)", "2", nil},
	{"index(`ab€c`,`c`)", "3", nil},
	{"index(`ab€c`,`d`)", "-1", nil},
	{"index(`ab€c`,`ab`)", "0", nil},
	{"index(`ab€c`,`b€`)", "1", nil},
	{"index(`ab€c`,`bc`)", "-1", nil},

	// indexAny
	{"indexAny(``,``)", "-1", nil},
	{"indexAny(`a`,``)", "-1", nil},
	{"indexAny(`ab€c`,`a`)", "0", nil},
	{"indexAny(`ab€c`,`b`)", "1", nil},
	{"indexAny(`ab€c`,`€`)", "2", nil},
	{"indexAny(`ab€c`,`c`)", "3", nil},
	{"index(`ab€c`,`d`)", "-1", nil},
	{"indexAny(`ab€c`,`ab`)", "0", nil},
	{"indexAny(`ab€c`,`ac`)", "0", nil},
	{"indexAny(`ab€c`,`cb`)", "1", nil},
	{"indexAny(`ab€c`,`c€`)", "2", nil},
	{"indexAny(`ab€c`,`ef`)", "-1", nil},

	// int
	{"int(0)", "0", nil},
	{"int(1)", "1", nil},
	{"int(-1)", "-1", nil},

	// itoa
	{"itoa(0).(string)", "0", nil},
	{"itoa(1).(string)", "1", nil},
	{"itoa(-1).(string)", "-1", nil},

	// join
	{"join(a, ``)", "", scope{"a": []string(nil)}},
	{"join(a, ``)", "", scope{"a": []string{}}},
	{"join(a, ``)", "a", scope{"a": []string{"a"}}},
	{"join(a, ``)", "ab", scope{"a": []string{"a", "b"}}},
	{"join(a, `,`)", "a,b,c", scope{"a": []string{"a", "b", "c"}}},
	//{"join(slice{`a`, `b`, `c`}, `,`)", "a,b,c", nil},

	// lastIndex
	{"lastIndex(``,``)", "0", nil},
	{"lastIndex(`a`,``)", "1", nil},
	{"lastIndex(``,`a`)", "-1", nil},
	{"lastIndex(`ab€ac€`,`a`)", "3", nil},
	{"lastIndex(`ab€ac€`,`b`)", "1", nil},
	{"lastIndex(`ab€ac€`,`€`)", "5", nil},
	{"lastIndex(`ab€ac€`,`c`)", "4", nil},
	{"lastIndex(`ab€ac€`,`d`)", "-1", nil},
	{"lastIndex(`ab€ac€`,`ab`)", "0", nil},
	{"lastIndex(`ab€acb€`,`b€`)", "5", nil},
	{"lastIndex(`ab€acb€`,`bc`)", "-1", nil},

	// len
	{"len(``)", "0", nil},
	{"len(`a`)", "1", nil},
	{"len(`abc`)", "3", nil},
	{"len(`€`)", "3", nil},
	{"len(`€`)", "3", nil},
	{"len(a)", "1", scope{"a": "a"}},
	{"len(a)", "3", scope{"a": "<a>"}},
	{"len(a)", "3", scope{"a": HTML("<a>")}},
	{"len(a)", "3", scope{"a": []int{1, 2, 3}}},
	{"len(a)", "2", scope{"a": []string{"a", "b"}}},
	{"len(a)", "4", scope{"a": []interface{}{"a", 2, 3, 4}}},
	{"len(a)", "0", scope{"a": []int(nil)}},
	{"len(a)", "2", scope{"a": map[string]int{"a": 5, "b": 8}}},
	{"len(a)", "2", scope{"a": Map{"a": 5, "b": 8}}},

	// max
	{"max(0, 0)", "0", nil},
	{"max(5, 0)", "5", nil},
	{"max(0, 7)", "7", nil},
	{"max(5, 7)", "7", nil},
	{"max(7, 5)", "7", nil},
	{"max(-7, 5)", "5", nil},
	{"max(7, -5)", "7", nil},
	{"max(5.5, 7.5)", "7.5", nil},
	{"max(7.5, 5.5)", "7.5", nil},
	{"max(-7.5, 5.5)", "5.5", nil},
	{"max(7.5, -5.5)", "7.5", nil},
	{"max(0.0000000000000000000000000000001, 0.0000000000000000000000000000002)", "0.0000000000000000000000000000002", nil},

	// md5
	{"md5(``)", "d41d8cd98f00b204e9800998ecf8427e", nil},
	{"md5(`hello world!`)", "fc3ff98e8c6a0d3087d515c0473f8677", nil},

	// min
	{"min(0, 0)", "0", nil},
	{"min(5, 0)", "0", nil},
	{"min(0, 7)", "0", nil},
	{"min(5, 7)", "5", nil},
	{"min(7, 5)", "5", nil},
	{"min(-7, 5)", "-7", nil},
	{"min(7, -5)", "-5", nil},
	{"min(5.5, 7.5)", "5.5", nil},
	{"min(7.5, 5.5)", "5.5", nil},
	{"min(-7.5, 5.5)", "-7.5", nil},
	{"min(7.5, -5.5)", "-5.5", nil},
	{"min(0.0000000000000000000000000000001, 0.0000000000000000000000000000002)", "0.0000000000000000000000000000001", nil},

	// repeat
	{"repeat(`a`, 0)", "", nil},
	{"repeat(`a`, 1)", "a", nil},
	{"repeat(`a`, 5)", "aaaaa", nil},
	{"repeat(`€`, 3)", "€€€", nil},
	{"repeat(`€€`, 3)", "€€€€€€", nil},

	// replace
	{"replace(``, ``, ``, 1)", "", nil},
	{"replace(`abc`, `b`, `e`, 1)", "aec", nil},
	{"replace(`abc`, `b`, `€`, 1)", "a€c", nil},
	{"replace(`abcbcba`, `b`, `e`, 1)", "aecbcba", nil},
	{"replace(`abcbcba`, `b`, `e`, 2)", "aececba", nil},
	{"replace(`abcbcba`, `b`, `e`, -1)", "aececea", nil},

	// replaceAll
	{"replaceAll(``, ``, ``)", "", nil},
	{"replaceAll(`abc`, `b`, `e`)", "aec", nil},
	{"replaceAll(`abc`, `b`, `€`)", "a€c", nil},
	{"replaceAll(`abcbcba`, `b`, `e`)", "aececea", nil},

	// reverse
	{"reverse(s)", "", scope{"s": []int(nil)}},
	{"reverse(s)", "", scope{"s": []int{}}},
	{"reverse(s)", "1", scope{"s": []int{1}}},
	{"reverse(s)", "2, 1", scope{"s": []int{1, 2}}},
	{"reverse(s)", "3, 2, 1", scope{"s": []int{1, 2, 3}}},

	// round
	{"round(0)", "0", nil},
	{"round(0.0)", "0", nil},
	{"round(5.3752)", "5", nil},
	{"round(7.4999)", "7", nil},
	{"round(7.5)", "8", nil},

	// sha1
	{"sha1(``)", "da39a3ee5e6b4b0d3255bfef95601890afd80709", nil},
	{"sha1(`hello world!`)", "430ce34d020724ed75a196dfc2ad67c77772d169", nil},

	// sha256
	{"sha256(``)", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", nil},
	{"sha256(`hello world!`)", "7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9", nil},

	// split
	{"split(``, ``)", "", nil},
	{"split(`a`, ``)", "a", nil},
	{"split(`ab`, ``)", "a, b", nil},
	{"split(`a,b,c`, `,`)", "a, b, c", nil},
	{"split(`a,b,c,`, `,`)", "a, b, c, ", nil},

	// splitN
	{"splitN(``, ``, 0)", "", nil},
	{"splitN(`a`, ``, 0)", "", nil},
	{"splitN(`ab`, ``, 1)", "ab", nil},
	{"splitN(`a,b,c`, `,`, 2)", "a, b,c", nil},

	// title
	{"title(``)", "", nil},
	{"title(`a`)", "A", nil},
	{"title(`5`)", "5", nil},
	{"title(`€`)", "€", nil},
	{"title(`ab`)", "Ab", nil},
	{"title(`5a`)", "5a", nil},
	{"title(`ab cd`)", "Ab Cd", nil},

	// toLower
	{"toLower(``)", "", nil},
	{"toLower(`a`)", "a", nil},
	{"toLower(`A`)", "a", nil},
	{"toLower(`aB`)", "ab", nil},
	{"toLower(`aBCd`)", "abcd", nil},
	{"toLower(`èÈ`)", "èè", nil},

	// toTitle
	{"toTitle(``)", "", nil},
	{"toTitle(`a`)", "A", nil},
	{"toTitle(`5`)", "5", nil},
	{"toTitle(`€`)", "€", nil},
	{"toTitle(`ab`)", "AB", nil},
	{"toTitle(`5a`)", "5A", nil},
	{"toTitle(`ab cd`)", "AB CD", nil},

	// toUpper
	{"toUpper(``)", "", nil},
	{"toUpper(`A`)", "A", nil},
	{"toUpper(`a`)", "A", nil},
	{"toUpper(`Ab`)", "AB", nil},
	{"toUpper(`AbcD`)", "ABCD", nil},
	{"toUpper(`Èè`)", "ÈÈ", nil},

	// trim
	{"trim(``, ``)", "", nil},
	{"trim(` `, ``)", " ", nil},
	{"trim(` a`, ` `)", "a", nil},
	{"trim(`a `, ` `)", "a", nil},
	{"trim(` a `, ` `)", "a", nil},
	{"trim(` a b  `, ` `)", "a b", nil},
	{"trim(`a bb`, `b`)", "a ", nil},
	{"trim(`bb a`, `b`)", " a", nil},
}

var rendererBuiltinTestsInHTMLContext = []struct {
	src     string
	res     string
	globals scope
}{
	// html
	{"html(``)", "", nil},
	{"html(`a`)", "a", nil},
	{"html(`<a>`)", "<a>", nil},
	{"html(a)", "<a>", scope{"a": "<a>"}},
	{"html(a)", "<a>", scope{"a": HTML("<a>")}},
	{"html(a) + html(b)", "<a><b>", scope{"a": "<a>", "b": "<b>"}},
}

var statementBuiltinTests = []struct {
	src     string
	res     string
	globals scope
}{

	// atoi
	{"{% if n, err := atoi(``); err == nil %}{{ n }}{% else %}error{% end %}", "error", nil},
	{"{% if n, err := atoi(`0`); err == nil %}{{ n }}{% end %}", "0", nil},
	{"{% if n, err := atoi(`1`); err == nil %}{{ n }}{% end %}", "1", nil},
	{"{% if n, err := atoi(`-1`); err == nil %}{{ n }}{% end %}", "-1", nil},

	// copy
	{"{% n := copy(d, s) %}{{ d }}", "a, 2", scope{"d": []interface{}{nil, nil}, "s": []interface{}{"a", 2}}},
	{"{% n := copy(d, s) %}{{ d }}", "a, b", scope{"d": []string{"", ""}, "s": []string{"a", "b"}}},
	{"{% n := copy(d, s) %}{{ d }}", "1, 2", scope{"d": []int{0, 0}, "s": []int{1, 2}}},
	{"{% n := copy(d, s) %}{{ d }}", "1, 2", scope{"d": []byte{0, 0}, "s": []byte{1, 2}}},
	{"{% n := copy(d, s) %}{{ d[0][1] }}, {{ d[1][2] }}", "a, b", scope{"d": []Map{nil, nil}, "s": []Map{{1: "a", 2: "b"}, {1: "a", 2: "b"}}}},

	// delete
	{"{% delete(m,`a`) %}{% if _, ok := m[`a`]; ok %}no{% else %}ok{% end %}", "ok", scope{"m": Map{}}},
	{"{% delete(m,`a`) %}{% if _, ok := m[`a`]; ok %}no{% else %}ok{% end %}", "ok", scope{"m": Map{"a": 6}}},
	{"{% delete(m,`b`) %}{% if _, ok := m[`a`]; ok %}ok{% else %}no{% end %}", "ok", scope{"m": Map{"a": 6}}},
	{"{% delete(m,5) %}{% if _, ok := m[5]; ok %}no{% else %}ok{% end %}", "ok", scope{"m": Map{5: true}}},
	{"{% delete(m,5.0) %}{% if _, ok := m[5.0]; ok %}no{% else %}ok{% end %}", "ok", scope{"m": Map{5: true}}},

	// sort
	{"{% sort(nil) %}", "", nil},
	{"{% sort(s1) %}{{ s1 }}", "", scope{"s1": []int{}}},
	{"{% sort(s2) %}{{ s2 }}", "1", scope{"s2": []int{1}}},
	{"{% sort(s3) %}{{ s3 }}", "1, 2", scope{"s3": []int{1, 2}}},
	{"{% sort(s4) %}{{ s4 }}", "1, 2", scope{"s4": []int{2, 1}}},
	{"{% sort(s5) %}{{ s5 }}", "1, 2, 3", scope{"s5": []int{3, 1, 2}}},
	{"{% sort(s6) %}{{ s6 }}", "a", scope{"s6": []string{"a"}}},
	{"{% sort(s7) %}{{ s7 }}", "a, b", scope{"s7": []string{"b", "a"}}},
	{"{% sort(s8) %}{{ s8 }}", "a, b, c", scope{"s8": []string{"b", "a", "c"}}},
	{"{% sort(s9) %}{{ s9 }}", "false, true, true", scope{"s9": []bool{true, false, true}}},
}

var rendererRandomBuiltinTests = []struct {
	src     string
	seed    int64
	res     string
	globals scope
}{
	// rand
	{"rand(-1)", 1, "5577006791947779410", nil},
	{"rand(1)", 1, "0", nil},
	{"rand(1)", 2, "0", nil},
	{"rand(2)", 1, "1", nil},
	{"rand(2)", 2, "0", nil},
	{"rand(100)", 1, "81", nil},
	{"rand(100)", 2, "86", nil},
	{"rand(100)", 3, "8", nil},
	{"rand(100)", 4, "29", nil},

	// shuffle
	{"shuffle(s)", 1, "", scope{"s": []int{}}},
	{"shuffle(s)", 1, "1", scope{"s": []int{1}}},
	{"shuffle(s)", 1, "1, 2", scope{"s": []int{1, 2}}},
	{"shuffle(s)", 2, "2, 1", scope{"s": []int{1, 2}}},
	{"shuffle(s)", 1, "1, 2, 3", scope{"s": []int{1, 2, 3}}},
	{"shuffle(s)", 2, "3, 1, 2", scope{"s": []int{1, 2, 3}}},
	{"shuffle(s)", 3, "1, 3, 2", scope{"s": []int{1, 2, 3}}},
	{"shuffle(s)", 1, "a, b, c", scope{"s": []string{"a", "b", "c"}}},
	{"shuffle(s)", 2, "c, a, b", scope{"s": []string{"a", "b", "c"}}},
	{"shuffle(s)", 3, "a, c, b", scope{"s": []string{"a", "b", "c"}}},
}

func TestRenderBuiltin(t *testing.T) {
	for _, expr := range rendererBuiltinTests {
		var tree, err = parser.ParseSource([]byte("{{"+expr.src+"}}"), ast.ContextText)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = RenderTree(b, tree, expr.globals, true)
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

func TestRenderBuiltinInHTMLContext(t *testing.T) {
	for _, expr := range rendererBuiltinTestsInHTMLContext {
		var tree, err = parser.ParseSource([]byte("{{"+expr.src+"}}"), ast.ContextHTML)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = RenderTree(b, tree, expr.globals, true)
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

func TestStatementBuiltin(t *testing.T) {
	for _, expr := range statementBuiltinTests {
		var tree, err = parser.ParseSource([]byte(expr.src), ast.ContextText)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = RenderTree(b, tree, expr.globals, true)
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

func TestRenderErrorfBuiltin(t *testing.T) {
	src := "\n\n   {% errorf(`error %s %d`, `a`, 5) %}"
	var tree, err = parser.ParseSource([]byte(src), ast.ContextText)
	if err != nil {
		t.Errorf("source: %q, %s\n", src, err)
		return
	}
	err = RenderTree(ioutil.Discard, tree, nil, true)
	if err == nil {
		t.Errorf("source: %q, expecting error\n", src)
		return
	}
	if e := ":3:7: error a 5"; err.Error() != e {
		t.Errorf("source: %q, unexpected error %q, expecting error %q\n", src, err.Error(), e)
	}
}

func TestRenderRandomBuiltin(t *testing.T) {
	for _, expr := range rendererRandomBuiltinTests {
		var tree, err = parser.ParseSource([]byte("{{"+expr.src+"}}"), ast.ContextText)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		testSeed = expr.seed
		err = RenderTree(b, tree, expr.globals, true)
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
