// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"testing"
)

var rendererBuiltinTestsInHTMLContext = []struct {
	src  string
	res  string
	vars Vars
}{
	// abbreviate
	{"{{ abbreviate(``,0) }}", "", nil},
	{"{{ abbreviate(`abc`,0) }}", "", nil},
	{"{{ abbreviate(`Lorem ipsum dolor sit amet.`,28) }}", "Lorem ipsum dolor sit amet.", nil},
	{"{{ abbreviate(`Lorem ipsum dolor sit amet.`,27) }}", "Lorem ipsum dolor sit amet.", nil},
	{"{{ abbreviate(`Lorem ipsum dolor sit amet.`,26) }}", "Lorem ipsum dolor sit...", nil},
	{"{{ abbreviate(`Lorem ipsum dolor sit amet.`,10) }}", "Lorem...", nil},
	{"{{ abbreviate(`Lorem ipsum dolor sit amet.`,8) }}", "Lorem...", nil},
	{"{{ abbreviate(`Lorem ipsum dolor sit amet.`,7) }}", "...", nil},
	{"{{ abbreviate(`Lorem ipsum dolor sit amet.`,6) }}", "...", nil},
	{"{{ abbreviate(`Lorem ipsum dolor sit amet.`,5) }}", "...", nil},
	{"{{ abbreviate(`Lorem ipsum dolor sit amet.`,4) }}", "...", nil},
	{"{{ abbreviate(`Lorem ipsum d`,12) }}", "Lorem...", nil},
	{"{{ abbreviate(`Lorem. Ipsum.`,9) }}", "Lorem...", nil},
	{"{{ abbreviate(`Lorem, ipsum.`,9) }}", "Lorem...", nil},

	// abs
	{"{{ abs(0) }}", "0", nil},
	{"{{ abs(1) }}", "1", nil},
	{"{{ abs(-1) }}", "1", nil},
	{"{{ abs(22) }}", "22", nil},
	{"{{ abs(-22) }}", "22", nil},

	// base64
	{"{{ base64(``) }}", "", nil},
	{"{{ base64(`hello world!`) }}", "aGVsbG8gd29ybGQh", nil},

	// contains
	{"{{ contains(``, ``) }}", "true", nil},
	{"{{ contains(`a`, ``) }}", "true", nil},
	{"{{ contains(`abc`, `b`) }}", "true", nil},
	{"{{ contains(`abc`, `e`) }}", "false", nil},

	// hasPrefix
	{"{{ hasPrefix(``, ``) }}", "true", nil},
	{"{{ hasPrefix(`a`, ``) }}", "true", nil},
	{"{{ hasPrefix(`abc`, `a`) }}", "true", nil},
	{"{{ hasPrefix(`abc`, `b`) }}", "false", nil},

	// hasSuffix
	{"{{ hasSuffix(``, ``) }}", "true", nil},
	{"{{ hasSuffix(`a`, ``) }}", "true", nil},
	{"{{ hasSuffix(`abc`, `c`) }}", "true", nil},
	{"{{ hasSuffix(`abc`, `b`) }}", "false", nil},

	// hex
	{"{{ hex(``) }}", "", nil},
	{"{{ hex(`hello world!`) }}", "68656c6c6f20776f726c6421", nil},

	// hmac TODO: invalid memory address or nil pointer dereference
	//{"{{ hmac(MD5, ``, ``) }}", "dOb3KYqcLRaJNfWMAButiA==", nil},
	//{"{{ hmac(MD5, `hello world!`, ``) }}", "POUE2/xvWDT8UjcXJ4d/hQ==", nil},
	//{"{{ hmac(MD5, ``, `secret`) }}", "XI2wPwTOwPQ7ywYAI5FBkA==", nil},
	//{"{{ hmac(MD5, `hello world!`, `secret`) }}", "CgRh4Q6JUG18MaFFZjvtkw==", nil},
	//{"{{ hmac(SHA1, ``, ``) }}", "+9sdGxiqbAgyS31ktx+3Y3BpDh0=", nil},
	//{"{{ hmac(SHA1, `hello world!`, ``) }}", "Cs2Lo6MmqAmr0Qj3JXmz/wJnhDg=", nil},
	//{"{{ hmac(SHA1, ``, `secret`) }}", "Ja9hdKD87MTTRmgKcrfOZEuaiOg=", nil},
	//{"{{ hmac(SHA1, `hello world!`, `secret`) }}", "pN9fnSN6sMoyQfBCvPYFmk70kcQ=", nil},
	//{"{{ hmac(SHA256, ``, ``) }}", "thNnmggU2ex3L5XXeMNfxf8Wl8STcVZTxscSFEKSxa0=", nil},
	//{"{{ hmac(SHA256, `hello world!`, ``) }}", "7/WCWbmktkh3Gig/DI7JWORlJ0gUpKhebIYJG4iMxJw=", nil},
	//{"{{ hmac(SHA256, ``, `secret`) }}", "+eZuF5tnR65UEI+C+K3os8Jddv0wr95sOVgixTAZYWk=", nil},
	//{"{{ hmac(SHA256, `hello world!`, `secret`) }}", "cgaXMb8pG0Y67LIYvCJ6vOPUA9dtpn+u8tSNPLQ7L1Q=", nil},

	// html
	{"{{ html(``) }}", "", nil},
	{"{{ html(`a`) }}", "a", nil},
	{"{{ html(`<a>`) }}", "<a>", nil},
	{"{{ html(a) }}", "<a>", Vars{"a": "<a>"}},
	{"{{ html(a) }}", "<a>", Vars{"a": HTML("<a>")}},
	//{"{{ html(a) + html(b) }}", "<a><b>", Vars{"a": "<a>", "b": "<b>"}}, TODO: reflect: call of reflect.Value.Set on zero Value

	// index
	{"{{ index(``,``) }}", "0", nil},
	{"{{ index(`a`,``) }}", "0", nil},
	{"{{ index(`ab€c`,`a`) }}", "0", nil},
	{"{{ index(`ab€c`,`b`) }}", "1", nil},
	{"{{ index(`ab€c`,`€`) }}", "2", nil},
	{"{{ index(`ab€c`,`c`) }}", "3", nil},
	{"{{ index(`ab€c`,`d`) }}", "-1", nil},
	{"{{ index(`ab€c`,`ab`) }}", "0", nil},
	{"{{ index(`ab€c`,`b€`) }}", "1", nil},
	{"{{ index(`ab€c`,`bc`) }}", "-1", nil},

	// indexAny
	{"{{ indexAny(``,``) }}", "-1", nil},
	{"{{ indexAny(`a`,``) }}", "-1", nil},
	{"{{ indexAny(`ab€c`,`a`) }}", "0", nil},
	{"{{ indexAny(`ab€c`,`b`) }}", "1", nil},
	{"{{ indexAny(`ab€c`,`€`) }}", "2", nil},
	{"{{ indexAny(`ab€c`,`c`) }}", "3", nil},
	{"{{ index(`ab€c`,`d`) }}", "-1", nil},
	{"{{ indexAny(`ab€c`,`ab`) }}", "0", nil},
	{"{{ indexAny(`ab€c`,`ac`) }}", "0", nil},
	{"{{ indexAny(`ab€c`,`cb`) }}", "1", nil},
	{"{{ indexAny(`ab€c`,`c€`) }}", "2", nil},
	{"{{ indexAny(`ab€c`,`ef`) }}", "-1", nil},

	// itoa
	{"{{ itoa(0) }}", "0", nil},
	{"{{ itoa(1) }}", "1", nil},
	{"{{ itoa(-1) }}", "-1", nil},
	{"{{ itoa(523) }}", "523", nil},

	// escapeHTML
	{"{{ escapeHTML(``) }}", "", nil},
	{"{{ escapeHTML(`a`) }}", "a", nil},
	{"{{ escapeHTML(`<a>`) }}", "&lt;a&gt;", nil},
	{"{{ escapeHTML(a) }}", "&lt;a&gt;", Vars{"a": "<a>"}},

	// join
	{"{{ join(a, ``) }}", "", Vars{"a": []string(nil)}},
	{"{{ join(a, ``) }}", "", Vars{"a": []string{}}},
	{"{{ join(a, ``) }}", "a", Vars{"a": []string{"a"}}},
	{"{{ join(a, ``) }}", "ab", Vars{"a": []string{"a", "b"}}},
	{"{{ join(a, `,`) }}", "a,b,c", Vars{"a": []string{"a", "b", "c"}}},
	{"{{ join([]string{`a`, `b`, `c`}, `,`) }}", "a,b,c", nil},

	// lastIndex
	{"{{ lastIndex(``,``) }}", "0", nil},
	{"{{ lastIndex(`a`,``) }}", "1", nil},
	{"{{ lastIndex(``,`a`) }}", "-1", nil},
	{"{{ lastIndex(`ab€ac€`,`a`) }}", "3", nil},
	{"{{ lastIndex(`ab€ac€`,`b`) }}", "1", nil},
	{"{{ lastIndex(`ab€ac€`,`€`) }}", "5", nil},
	{"{{ lastIndex(`ab€ac€`,`c`) }}", "4", nil},
	{"{{ lastIndex(`ab€ac€`,`d`) }}", "-1", nil},
	{"{{ lastIndex(`ab€ac€`,`ab`) }}", "0", nil},
	{"{{ lastIndex(`ab€acb€`,`b€`) }}", "5", nil},
	{"{{ lastIndex(`ab€acb€`,`bc`) }}", "-1", nil},

	// max
	{"{{ max(0, 0) }}", "0", nil},
	{"{{ max(5, 0) }}", "5", nil},
	{"{{ max(0, 7) }}", "7", nil},
	{"{{ max(5, 7) }}", "7", nil},
	{"{{ max(7, 5) }}", "7", nil},
	{"{{ max(-7, 5) }}", "5", nil},
	{"{{ max(7, -5) }}", "7", nil},

	// min
	{"{{ min(0, 0) }}", "0", nil},
	{"{{ min(5, 0) }}", "0", nil},
	{"{{ min(0, 7) }}", "0", nil},
	{"{{ min(5, 7) }}", "5", nil},
	{"{{ min(7, 5) }}", "5", nil},
	{"{{ min(-7, 5) }}", "-7", nil},
	{"{{ min(7, -5) }}", "-5", nil},

	// repeat
	{"{{ repeat(`a`, 0) }}", "", nil},
	{"{{ repeat(`a`, 1) }}", "a", nil},
	{"{{ repeat(`a`, 5) }}", "aaaaa", nil},
	{"{{ repeat(`€`, 3) }}", "€€€", nil},
	{"{{ repeat(`€€`, 3) }}", "€€€€€€", nil},

	// replace
	{"{{ replace(``, ``, ``, 1) }}", "", nil},
	{"{{ replace(`abc`, `b`, `e`, 1) }}", "aec", nil},
	{"{{ replace(`abc`, `b`, `€`, 1) }}", "a€c", nil},
	{"{{ replace(`abcbcba`, `b`, `e`, 1) }}", "aecbcba", nil},
	{"{{ replace(`abcbcba`, `b`, `e`, 2) }}", "aececba", nil},
	{"{{ replace(`abcbcba`, `b`, `e`, -1) }}", "aececea", nil},

	// replaceAll
	{"{{ replaceAll(``, ``, ``) }}", "", nil},
	{"{{ replaceAll(`abc`, `b`, `e`) }}", "aec", nil},
	{"{{ replaceAll(`abc`, `b`, `€`) }}", "a€c", nil},
	{"{{ replaceAll(`abcbcba`, `b`, `e`) }}", "aececea", nil},

	// reverse
	{"{{ reverse(s) }}", "", Vars{"s": []int(nil)}},
	{"{{ reverse(s) }}", "", Vars{"s": []int{}}},
	{"{{ reverse(s) }}", "1", Vars{"s": []int{1}}},
	{"{{ reverse(s) }}", "2, 1", Vars{"s": []int{1, 2}}},
	{"{{ reverse(s) }}", "3, 2, 1", Vars{"s": []int{1, 2, 3}}},

	// round
	{"{{ round(0) }}", "0", nil},
	{"{{ round(0.0) }}", "0", nil},
	{"{{ round(5.3752) }}", "5", nil},
	{"{{ round(7.4999) }}", "7", nil},
	{"{{ round(7.5) }}", "8", nil},

	// sort
	//{"{% sort(nil) %}", "", nil}, TODO: invalid memory address or nil pointer dereference
	{"{% sort(s1) %}{{ s1 }}", "", Vars{"s1": []int{}}},
	{"{% sort(s2) %}{{ s2 }}", "1", Vars{"s2": []int{1}}},
	{"{% sort(s3) %}{{ s3 }}", "1, 2", Vars{"s3": []int{1, 2}}},
	{"{% sort(s4) %}{{ s4 }}", "1, 2", Vars{"s4": []int{2, 1}}},
	{"{% sort(s5) %}{{ s5 }}", "1, 2, 3", Vars{"s5": []int{3, 1, 2}}},
	{"{% sort(s6) %}{{ s6 }}", "a", Vars{"s6": []string{"a"}}},
	{"{% sort(s7) %}{{ s7 }}", "a, b", Vars{"s7": []string{"b", "a"}}},
	{"{% sort(s8) %}{{ s8 }}", "a, b, c", Vars{"s8": []string{"b", "a", "c"}}},
	{"{% sort(s9) %}{{ s9 }}", "false, true, true", Vars{"s9": []bool{true, false, true}}},
	{"{% s := []html{html(`<b>`), html(`<a>`), html(`<c>`)} %}{% sort(s) %}{{ s }}", "<a>, <b>, <c>", nil},

	// split
	{"{{ split(``, ``) }}", "", nil},
	{"{{ split(`a`, ``) }}", "a", nil},
	{"{{ split(`ab`, ``) }}", "a, b", nil},
	{"{{ split(`a,b,c`, `,`) }}", "a, b, c", nil},
	{"{{ split(`a,b,c,`, `,`) }}", "a, b, c, ", nil},

	// splitN
	{"{{ splitN(``, ``, 0) }}", "", nil},
	{"{{ splitN(`a`, ``, 0) }}", "", nil},
	{"{{ splitN(`ab`, ``, 1) }}", "ab", nil},
	{"{{ splitN(`a,b,c`, `,`, 2) }}", "a, b,c", nil},

	// title
	{"{{ title(``) }}", "", nil},
	{"{{ title(`a`) }}", "A", nil},
	{"{{ title(`5`) }}", "5", nil},
	{"{{ title(`€`) }}", "€", nil},
	{"{{ title(`ab`) }}", "Ab", nil},
	{"{{ title(`5a`) }}", "5a", nil},
	{"{{ title(`ab cd`) }}", "Ab Cd", nil},

	// toLower
	{"{{ toLower(``) }}", "", nil},
	{"{{ toLower(`a`) }}", "a", nil},
	{"{{ toLower(`A`) }}", "a", nil},
	{"{{ toLower(`aB`) }}", "ab", nil},
	{"{{ toLower(`aBCd`) }}", "abcd", nil},
	{"{{ toLower(`èÈ`) }}", "èè", nil},

	// toTitle
	{"{{ toTitle(``) }}", "", nil},
	{"{{ toTitle(`a`) }}", "A", nil},
	{"{{ toTitle(`5`) }}", "5", nil},
	{"{{ toTitle(`€`) }}", "€", nil},
	{"{{ toTitle(`ab`) }}", "AB", nil},
	{"{{ toTitle(`5a`) }}", "5A", nil},
	{"{{ toTitle(`ab cd`) }}", "AB CD", nil},

	// toUpper
	{"{{ toUpper(``) }}", "", nil},
	{"{{ toUpper(`A`) }}", "A", nil},
	{"{{ toUpper(`a`) }}", "A", nil},
	{"{{ toUpper(`Ab`) }}", "AB", nil},
	{"{{ toUpper(`AbcD`) }}", "ABCD", nil},
	{"{{ toUpper(`Èè`) }}", "ÈÈ", nil},

	// trim
	{"{{ trim(``, ``) }}", "", nil},
	{"{{ trim(` `, ``) }}", " ", nil},
	{"{{ trim(` a`, ` `) }}", "a", nil},
	{"{{ trim(`a `, ` `) }}", "a", nil},
	{"{{ trim(` a `, ` `) }}", "a", nil},
	{"{{ trim(` a b  `, ` `) }}", "a b", nil},
	{"{{ trim(`a bb`, `b`) }}", "a ", nil},
	{"{{ trim(`bb a`, `b`) }}", " a", nil},
}

func TestRenderBuiltinInHTMLContext(t *testing.T) {
	for _, expr := range rendererBuiltinTestsInHTMLContext {
		r := MapReader{"/index.html": []byte(expr.src)}
		tmpl, err := Load("/index.html", r, mainPackage(expr.vars), ContextHTML, LimitMemorySize)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, RenderOptions{MaxMemorySize: 1000})
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

var rendererRandomBuiltinTests = []struct {
	src  string
	seed int64
	res  string
	vars Vars
}{
	// rand
	{"{{ rand(-1) }}", 1, "5577006791947779410", nil},
	{"{{ rand(1) }}", 1, "0", nil},
	{"{{ rand(1) }}", 2, "0", nil},
	{"{{ rand(2) }}", 1, "1", nil},
	{"{{ rand(2) }}", 2, "0", nil},
	{"{{ rand(100) }}", 1, "81", nil},
	{"{{ rand(100) }}", 2, "86", nil},
	{"{{ rand(100) }}", 3, "8", nil},
	{"{{ rand(100) }}", 4, "29", nil},

	// shuffle
	{"{% shuffle(s) %}{{ s }}", 1, "", Vars{"s": []int{}}},
	{"{% shuffle(s) %}{{ s }}", 1, "1", Vars{"s": []int{1}}},
	{"{% shuffle(s) %}{{ s }}", 1, "1, 2", Vars{"s": []int{1, 2}}},
	{"{% shuffle(s) %}{{ s }}", 2, "2, 1", Vars{"s": []int{1, 2}}},
	{"{% shuffle(s) %}{{ s }}", 1, "1, 2, 3", Vars{"s": []int{1, 2, 3}}},
	{"{% shuffle(s) %}{{ s }}", 2, "3, 1, 2", Vars{"s": []int{1, 2, 3}}},
	{"{% shuffle(s) %}{{ s }}", 3, "1, 3, 2", Vars{"s": []int{1, 2, 3}}},
	{"{% shuffle(s) %}{{ s }}", 1, "a, b, c", Vars{"s": []string{"a", "b", "c"}}},
	{"{% shuffle(s) %}{{ s }}", 2, "c, a, b", Vars{"s": []string{"a", "b", "c"}}},
	{"{% shuffle(s) %}{{ s }}", 3, "a, c, b", Vars{"s": []string{"a", "b", "c"}}},
}

func TestRenderRandomBuiltin(t *testing.T) {
	for _, expr := range rendererRandomBuiltinTests {
		r := MapReader{"/index.html": []byte(expr.src)}
		tmpl, err := Load("/index.html", r, mainPackage(expr.vars), ContextHTML, LimitMemorySize)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		testSeed = expr.seed
		err = tmpl.Render(b, expr.vars, RenderOptions{MaxMemorySize: 1000})
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
