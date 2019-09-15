// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"testing"
	"time"
)

var loc, _ = time.LoadLocation("America/Los_Angeles")
var testTime, _ = time.Parse(time.RFC3339, "2006-01-02T15:04:05.123456Z")
var testTime1 = Time(time.Date(2006, 1, 02, 15, 4, 5, 123456789, time.UTC))
var testTime2 = Time(time.Date(12365, 3, 22, 15, 19, 5, 123456789, time.UTC))
var testTime3 = Time(time.Date(2006, 1, 02, 15, 4, 5, 123456789, loc))
var testTime4 = Time(time.Date(-12365, 3, 22, 15, 19, 5, 123456789, loc))

type builtinTest struct {
	src  string
	res  string
	vars Vars
}

var rendererBuiltinTestsInHTMLContext = []builtinTest{
	// Hasher
	{"{% var a Hasher %}{{ a }}", "0", nil},

	// MD5, SHA1, SHA256
	{"{% var h = MD5 %}{{ h }}", "0", nil},
	{"{% var h = SHA1 %}{{ h }}", "1", nil},
	{"{% var h = SHA256 %}{{ h }}", "2", nil},

	// Time
	{"{% var t Time %}{{ t }}", "Mon, 01 Jan 0001 00:00:00 UTC", nil},
	{"{{ t }}", "Mon, 02 Jan 2006 15:04:05 UTC", Vars{"t": Time(testTime)}},
	{"<div data-time=\"{{ t }}\">", "<div data-time=\"2006-01-02T15:04:05Z\">", Vars{"t": Time(testTime)}},
	{"<div data-time={{ t }}>", "<div data-time=2006-01-02T15:04:05Z>", Vars{"t": Time(testTime)}},

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

	// escapeHTML
	{"{{ escapeHTML(``) }}", "", nil},
	{"{{ escapeHTML(`a`) }}", "a", nil},
	{"{{ escapeHTML(`<a>`) }}", "&lt;a&gt;", nil},
	{"{{ escapeHTML(a) }}", "&lt;a&gt;", Vars{"a": "<a>"}},

	// escapeQuery
	{"{{ escapeQuery(``) }}", "", nil},
	{"{{ escapeQuery(`a`) }}", "a", nil},
	{"{{ escapeQuery(` `) }}", "%20", nil},
	{"{{ escapeQuery(`a/b+c?d#`) }}", "a%2fb%2bc%3fd%23", nil},

	// hash
	{"{{ hash(MD5, ``) }}", "d41d8cd98f00b204e9800998ecf8427e", nil},
	{"{{ hash(MD5, `hello world!`) }}", "fc3ff98e8c6a0d3087d515c0473f8677", nil},
	{"{{ hash(SHA1, ``) }}", "da39a3ee5e6b4b0d3255bfef95601890afd80709", nil},
	{"{{ hash(SHA1, `hello world!`) }}", "430ce34d020724ed75a196dfc2ad67c77772d169", nil},
	{"{{ hash(SHA256, ``) }}", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", nil},
	{"{{ hash(SHA256, `hello world!`) }}", "7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9", nil},

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

	// hmac
	{"{{ hmac(MD5, ``, ``) }}", "dOb3KYqcLRaJNfWMAButiA==", nil},
	{"{{ hmac(MD5, `hello world!`, ``) }}", "POUE2/xvWDT8UjcXJ4d/hQ==", nil},
	{"{{ hmac(MD5, ``, `secret`) }}", "XI2wPwTOwPQ7ywYAI5FBkA==", nil},
	{"{{ hmac(MD5, `hello world!`, `secret`) }}", "CgRh4Q6JUG18MaFFZjvtkw==", nil},
	{"{{ hmac(SHA1, ``, ``) }}", "+9sdGxiqbAgyS31ktx+3Y3BpDh0=", nil},
	{"{{ hmac(SHA1, `hello world!`, ``) }}", "Cs2Lo6MmqAmr0Qj3JXmz/wJnhDg=", nil},
	{"{{ hmac(SHA1, ``, `secret`) }}", "Ja9hdKD87MTTRmgKcrfOZEuaiOg=", nil},
	{"{{ hmac(SHA1, `hello world!`, `secret`) }}", "pN9fnSN6sMoyQfBCvPYFmk70kcQ=", nil},
	{"{{ hmac(SHA256, ``, ``) }}", "thNnmggU2ex3L5XXeMNfxf8Wl8STcVZTxscSFEKSxa0=", nil},
	{"{{ hmac(SHA256, `hello world!`, ``) }}", "7/WCWbmktkh3Gig/DI7JWORlJ0gUpKhebIYJG4iMxJw=", nil},
	{"{{ hmac(SHA256, ``, `secret`) }}", "+eZuF5tnR65UEI+C+K3os8Jddv0wr95sOVgixTAZYWk=", nil},
	{"{{ hmac(SHA256, `hello world!`, `secret`) }}", "cgaXMb8pG0Y67LIYvCJ6vOPUA9dtpn+u8tSNPLQ7L1Q=", nil},

	// HTML
	{"{{ HTML(``) }}", "", nil},
	{"{{ HTML(`a`) }}", "a", nil},
	{"{{ HTML(`<a>`) }}", "<a>", nil},
	{"{{ HTML(a) }}", "<a>", Vars{"a": "<a>"}},
	{"{{ HTML(a) }}", "<a>", Vars{"a": HTML("<a>")}},
	//{"{{ HTML(a) + HTML(b) }}", "<a><b>", Vars{"a": "<a>", "b": "<b>"}}, TODO: reflect: call of reflect.Value.Set on zero Value

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
	//{"{% sort(nil) %}", "", nil}, TODO: not implemented
	{"{% sort(s1) %}{{ s1 }}", "", Vars{"s1": []int{}}},
	{"{% sort(s2) %}{{ s2 }}", "1", Vars{"s2": []int{1}}},
	{"{% sort(s3) %}{{ s3 }}", "1, 2", Vars{"s3": []int{1, 2}}},
	{"{% sort(s4) %}{{ s4 }}", "1, 2", Vars{"s4": []int{2, 1}}},
	{"{% sort(s5) %}{{ s5 }}", "1, 2, 3", Vars{"s5": []int{3, 1, 2}}},
	{"{% sort(s6) %}{{ s6 }}", "a", Vars{"s6": []string{"a"}}},
	{"{% sort(s7) %}{{ s7 }}", "a, b", Vars{"s7": []string{"b", "a"}}},
	{"{% sort(s8) %}{{ s8 }}", "a, b, c", Vars{"s8": []string{"b", "a", "c"}}},
	{"{% sort(s9) %}{{ s9 }}", "false, true, true", Vars{"s9": []bool{true, false, true}}},
	{"{% s := []HTML{HTML(`<b>`), HTML(`<a>`), HTML(`<c>`)} %}{% sort(s) %}{{ s }}", "<a>, <b>, <c>", nil},

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

	// sprintf
	{"{{ sprintf(``) }}", "", nil},
	{"{{ sprintf(`a`) }}", "a", nil},
	{"{{ sprintf(`%s`, `a`) }}", "a", nil},
	{"{{ sprintf(`%d`, 5) }}", "5", nil},

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

	// trimLeft
	{"{{ trimLeft(``, ``) }}", "", nil},
	{"{{ trimLeft(` `, ``) }}", " ", nil},
	{"{{ trimLeft(` a`, ` `) }}", "a", nil},
	{"{{ trimLeft(`a `, ` `) }}", "a ", nil},
	{"{{ trimLeft(` a `, ` `) }}", "a ", nil},
	{"{{ trimLeft(` a b  `, ` `) }}", "a b  ", nil},
	{"{{ trimLeft(`a bb`, `b`) }}", "a bb", nil},
	{"{{ trimLeft(`bb a`, `b`) }}", " a", nil},
	{"{{ trimLeft(`a bb`, `b`) }}", "a bb", nil},
	{"{{ trimLeft(`bb a`, `a`) }}", "bb a", nil},

	// trimPrefix
	{"{{ trimPrefix(``, ``) }}", "", nil},
	{"{{ trimPrefix(` `, ``) }}", " ", nil},
	{"{{ trimPrefix(` a`, ` `) }}", "a", nil},
	{"{{ trimPrefix(`  a`, ` `) }}", " a", nil},
	{"{{ trimPrefix(`a `, ` `) }}", "a ", nil},
	{"{{ trimPrefix(` a `, ` `) }}", "a ", nil},
	{"{{ trimPrefix(`  a b `, ` `) }}", " a b ", nil},
	{"{{ trimPrefix(`a bb`, `b`) }}", "a bb", nil},
	{"{{ trimPrefix(`bb a`, `b`) }}", "b a", nil},
	{"{{ trimPrefix(`a bb`, `b`) }}", "a bb", nil},

	// trimRight
	{"{{ trimRight(``, ``) }}", "", nil},
	{"{{ trimRight(` `, ``) }}", " ", nil},
	{"{{ trimRight(` a`, ` `) }}", " a", nil},
	{"{{ trimRight(`a `, ` `) }}", "a", nil},
	{"{{ trimRight(` a `, ` `) }}", " a", nil},
	{"{{ trimRight(` a b  `, ` `) }}", " a b", nil},
	{"{{ trimRight(`a bb`, `b`) }}", "a ", nil},
	{"{{ trimRight(`bb a`, `b`) }}", "bb a", nil},
	{"{{ trimRight(`bb a`, `a`) }}", "bb ", nil},

	// trimSuffix
	{"{{ trimSuffix(``, ``) }}", "", nil},
	{"{{ trimSuffix(` `, ``) }}", " ", nil},
	{"{{ trimSuffix(`a `, ` `) }}", "a", nil},
	{"{{ trimSuffix(`a  `, ` `) }}", "a ", nil},
	{"{{ trimSuffix(`a `, ` `) }}", "a", nil},
	{"{{ trimSuffix(` a `, ` `) }}", " a", nil},
	{"{{ trimSuffix(`  a b `, ` `) }}", "  a b", nil},
	{"{{ trimSuffix(`a bb`, `b`) }}", "a b", nil},
	{"{{ trimSuffix(`bb a`, `b`) }}", "bb a", nil},
	{"{{ trimSuffix(`bb a`, `a`) }}", "bb ", nil},
}

func TestRenderBuiltinInHTMLContext(t *testing.T) {
	for _, expr := range rendererBuiltinTestsInHTMLContext {
		r := MapReader{"/index.html": []byte(expr.src)}
		tmpl, err := Load("/index.html", r, mainPackage(expr.vars), ContextHTML, &LoadOptions{LimitMemorySize: true})
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, &RenderOptions{MaxMemorySize: 1000})
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

var rendererBuiltinTestsInJavaScriptContext = []builtinTest{
	// Time
	{"var t = {{ t }};", "var t = new Date(\"2006-01-02T15:04:05.123Z\");", Vars{"t": Time(testTime)}},
	{"var t = new Date(\"{{ t }}\");", "var t = new Date(\"2006-01-02T15:04:05.123Z\");", Vars{"t": Time(testTime)}},
	{"var t = {{ t }};", "var t = new Date(\"+012365-03-22T15:19:05.123Z\");", Vars{"t": Time(testTime2)}},
	{"var t = new Date(\"{{ t }}\");", "var t = new Date(\"+012365-03-22T15:19:05.123Z\");", Vars{"t": Time(testTime2)}},
	{"var t = {{ t }};", "var t = new Date(\"2006-01-02T15:04:05.123-08:00\");", Vars{"t": Time(testTime3)}},
	{"var t = new Date(\"{{ t }}\");", "var t = new Date(\"2006-01-02T15:04:05.123-08:00\");", Vars{"t": Time(testTime3)}},
}

func TestRenderBuiltinInJavaScriptContext(t *testing.T) {
	for _, expr := range rendererBuiltinTestsInJavaScriptContext {
		r := MapReader{"/index.html": []byte(expr.src)}
		tmpl, err := Load("/index.html", r, mainPackage(expr.vars), ContextJavaScript, &LoadOptions{LimitMemorySize: true})
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		err = tmpl.Render(b, expr.vars, &RenderOptions{MaxMemorySize: 1000})
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

	// randFloat
	{"{{ randFloat() }}", 1, "0.6046602879796196", nil},
	{"{{ randFloat() }}", 2, "0.9405090880450124", nil},
	{"{{ randFloat() }}", 3, "0.6645600532184904", nil},
	{"{{ randFloat() }}", 4, "0.4377141871869802", nil},

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
		tmpl, err := Load("/index.html", r, mainPackage(expr.vars), ContextHTML, &LoadOptions{LimitMemorySize: true})
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		testSeed = expr.seed
		err = tmpl.Render(b, expr.vars, &RenderOptions{MaxMemorySize: 1000})
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
