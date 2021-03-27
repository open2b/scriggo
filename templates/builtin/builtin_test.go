// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builtin

import (
	"fmt"
	"testing"
	"time"

	"github.com/open2b/scriggo/templates"
)

var spf = fmt.Sprintf

var tests = []struct {
	got      string
	expected string
}{

	// abbreviate
	{Abbreviate(``, 0), ""},
	{Abbreviate(`abc`, 0), ""},
	{Abbreviate(`Lorem ipsum dolor sit amet.`, 28), "Lorem ipsum dolor sit amet."},
	{Abbreviate(`Lorem ipsum dolor sit amet.`, 27), "Lorem ipsum dolor sit amet."},
	{Abbreviate(`Lorem ipsum dolor sit amet.`, 26), "Lorem ipsum dolor sit..."},
	{Abbreviate(`Lorem ipsum dolor sit amet.`, 10), "Lorem..."},
	{Abbreviate(`Lorem ipsum dolor sit amet.`, 8), "Lorem..."},
	{Abbreviate(`Lorem ipsum dolor sit amet.`, 7), "..."},
	{Abbreviate(`Lorem ipsum dolor sit amet.`, 6), "..."},
	{Abbreviate(`Lorem ipsum dolor sit amet.`, 5), "..."},
	{Abbreviate(`Lorem ipsum dolor sit amet.`, 4), "..."},
	{Abbreviate(`Lorem ipsum d`, 12), "Lorem..."},
	{Abbreviate(`Lorem. Ipsum.`, 9), "Lorem..."},
	{Abbreviate(`Lorem, ipsum.`, 9), "Lorem..."},

	// abs
	{spf("%d", Abs(0)), "0"},
	{spf("%d", Abs(1)), "1"},
	{spf("%d", Abs(-1)), "1"},
	{spf("%d", Abs(22)), "22"},
	{spf("%d", Abs(-22)), "22"},

	// base64
	{Base64(``), ""},
	{Base64(`hello world!`), "aGVsbG8gd29ybGQh"},

	// capitalize
	{Capitalize(``), ""},
	{Capitalize(`a`), "A"},
	{Capitalize(`5`), "5"},
	{Capitalize(`€`), "€"},
	{Capitalize(`ab`), "Ab"},
	{Capitalize(`5a`), "5a"},
	{Capitalize(`ab cd`), "Ab cd"},
	{Capitalize(` ab,cd`), " Ab,cd"},
	{Capitalize(` Ab,cd`), " Ab,cd"},

	// capitalizeAll
	{CapitalizeAll(``), ""},
	{CapitalizeAll(`a`), "A"},
	{CapitalizeAll(`5`), "5"},
	{CapitalizeAll(`€`), "€"},
	{CapitalizeAll(`ab`), "Ab"},
	{CapitalizeAll(`5a`), "5a"},
	{CapitalizeAll(`ab cd`), "Ab Cd"},
	{CapitalizeAll(` ab,cd`), " Ab,Cd"},
	{CapitalizeAll(` Ab,cd`), " Ab,Cd"},

	// date
	{func() string {
		t, _ := Date(2009, 11, 10, 12, 15, 32, 680327414, "UTC")
		return t.Format(time.RFC3339Nano)
	}(), "2009-11-10T12:15:32.680327414Z"},
	{func() string {
		t, _ := Date(2021, 3, 27, 17, 18, 51, 0, "CET")
		return t.Format(time.RFC3339Nano)
	}(), "2021-03-27T17:18:51+01:00"},

	// htmlEscape
	{spf("%s", HtmlEscape(``)), ""},
	{spf("%s", HtmlEscape(`a`)), "a"},
	{spf("%s", HtmlEscape(`<a>`)), "&lt;a&gt;"},
	{spf("%s", HtmlEscape("<a>")), "&lt;a&gt;"},

	// queryEscape
	{QueryEscape(``), ""},
	{QueryEscape(`a`), "a"},
	{QueryEscape(` `), "%20"},
	{QueryEscape(`a/b+c?d#`), "a%2fb%2bc%3fd%23"},

	// md5
	{Md5(``), "d41d8cd98f00b204e9800998ecf8427e"},
	{Md5(`hello world!`), "fc3ff98e8c6a0d3087d515c0473f8677"},

	// hex
	{Hex(``), ""},
	{Hex(`hello world!`), "68656c6c6f20776f726c6421"},

	// hmacSHA1
	{HmacSHA1(``, ``), "+9sdGxiqbAgyS31ktx+3Y3BpDh0="},
	{HmacSHA1(`hello world!`, ``), "Cs2Lo6MmqAmr0Qj3JXmz/wJnhDg="},
	{HmacSHA1(``, `secret`), "Ja9hdKD87MTTRmgKcrfOZEuaiOg="},
	{HmacSHA1(`hello world!`, `secret`), "pN9fnSN6sMoyQfBCvPYFmk70kcQ="},

	// hmacSHA256
	{HmacSHA256(``, ``), "thNnmggU2ex3L5XXeMNfxf8Wl8STcVZTxscSFEKSxa0="},
	{HmacSHA256(`hello world!`, ``), "7/WCWbmktkh3Gig/DI7JWORlJ0gUpKhebIYJG4iMxJw="},
	{HmacSHA256(``, `secret`), "+eZuF5tnR65UEI+C+K3os8Jddv0wr95sOVgixTAZYWk="},
	{HmacSHA256(`hello world!`, `secret`), "cgaXMb8pG0Y67LIYvCJ6vOPUA9dtpn+u8tSNPLQ7L1Q="},

	// max
	{spf("%d", Max(0, 0)), "0"},
	{spf("%d", Max(5, 0)), "5"},
	{spf("%d", Max(0, 7)), "7"},
	{spf("%d", Max(5, 7)), "7"},
	{spf("%d", Max(7, 5)), "7"},
	{spf("%d", Max(-7, 5)), "5"},
	{spf("%d", Max(7, -5)), "7"},

	// min
	{spf("%d", Min(0, 0)), "0"},
	{spf("%d", Min(5, 0)), "0"},
	{spf("%d", Min(0, 7)), "0"},
	{spf("%d", Min(5, 7)), "5"},
	{spf("%d", Min(7, 5)), "5"},
	{spf("%d", Min(-7, 5)), "-7"},
	{spf("%d", Min(7, -5)), "-5"},

	// now
	{spf("%t", func() bool {
		t, _ := TimeArguments(Now())
		return t.Sub(time.Now()) < time.Millisecond
	}()), "true"},

	// regexp
	{spf("%t", RegExp(nil, "(scriggo){2}").Match("scriggo")), "false"},
	{spf("%t", RegExp(nil, "(scriggo){2}").Match("scriggoscriggo")), "true"},
	{spf("%t", RegExp(nil, "(scriggo){2}").Match("scriggoscriggoscriggo")), "true"},
	{RegExp(nil, "foo.?").Find("seafood fool"), "food"},
	{RegExp(nil, "foo.?").Find("meat"), ""},
	{spf("%v", RegExp(nil, "a.").FindAll("paranormal", -1)), "[ar an al]"},
	{spf("%v", RegExp(nil, "a.").FindAll("paranormal", 2)), "[ar an]"},
	{spf("%v", RegExp(nil, "a.").FindAll("graal", -1)), "[aa]"},
	{spf("%v", RegExp(nil, "a.").FindAll("none", -1)), "[]"},
	{spf("%q", RegExp(nil, "a(x*)b").FindAllSubmatch("-ab-", -1)), `[["ab" ""]]`},
	{spf("%q", RegExp(nil, "a(x*)b").FindAllSubmatch("-axxb-", -1)), `[["axxb" "xx"]]`},
	{spf("%q", RegExp(nil, "a(x*)b").FindAllSubmatch("-ab-axb-", -1)), `[["ab" ""] ["axb" "x"]]`},
	{spf("%q", RegExp(nil, "a(x*)b").FindAllSubmatch("-axxb-ab-", -1)), `[["axxb" "xx"] ["ab" ""]]`},
	{spf("%q", RegExp(nil, "a(x*)b(y|z)c").FindSubmatch("-axxxbyc-")), `["axxxbyc" "xxx" "y"]`},
	{spf("%q", RegExp(nil, "a(x*)b(y|z)c").FindSubmatch("-abzc-")), `["abzc" "" "z"]`},
	{RegExp(nil, "a(x*)b").ReplaceAll("-ab-axxb-", "T"), `-T-T-`},
	{RegExp(nil, "a(x*)b").ReplaceAll("-ab-axxb-", "$1"), `--xx-`},
	{RegExp(nil, "a(x*)b").ReplaceAll("-ab-axxb-", "$1W"), `---`},
	{RegExp(nil, "a(x*)b").ReplaceAll("-ab-axxb-", "${1}W"), `-W-xxW-`},
	{RegExp(nil, "[^aeiou]").ReplaceAllFunc("seafood fool", ToUpper), `SeaFooD FooL`},
	{spf("%v", RegExp(nil, "a").Split("banana", -1)), `[b n n ]`},
	{spf("%v", RegExp(nil, "a").Split("banana", 0)), `[]`},
	{spf("%v", RegExp(nil, "a").Split("banana", 1)), `[banana]`},
	{spf("%v", RegExp(nil, "a").Split("banana", 2)), `[b nana]`},
	{spf("%v", RegExp(nil, "z+").Split("pizza", -1)), `[pi a]`},
	{spf("%v", RegExp(nil, "z+").Split("pizza", 0)), `[]`},
	{spf("%v", RegExp(nil, "z+").Split("pizza", 1)), `[pizza]`},
	{spf("%v", RegExp(nil, "z+").Split("pizza", 2)), `[pi a]`},

	// reverse
	{func() string { Reverse(nil); return "" }(), ""},
	{func() string { s := []int{}; Reverse(s); return spf("%v", s) }(), "[]"},
	{func() string { s := []int{1}; Reverse(s); return spf("%v", s) }(), "[1]"},
	{func() string { s := []int{1, 2}; Reverse(s); return spf("%v", s) }(), "[2 1]"},
	{func() string { s := []int{2, 1}; Reverse(s); return spf("%v", s) }(), "[1 2]"},
	{func() string { s := []int{3, 1, 2}; Reverse(s); return spf("%v", s) }(), "[2 1 3]"},
	{func() string { s := []string{"a"}; Reverse(s); return spf("%v", s) }(), "[a]"},
	{func() string { s := []string{"b", "a"}; Reverse(s); return spf("%v", s) }(), "[a b]"},
	{func() string { s := []string{"b", "a", "c"}; Reverse(s); return spf("%v", s) }(), "[c a b]"},
	{func() string { s := []bool{false, false, true}; Reverse(s); return spf("%v", s) }(), "[true false false]"},
	{func() string { s := []templates.HTML{`<b>`, `<a>`, `<c>`}; Reverse(s); return spf("%v", s) }(), "[<c> <a> <b>]"},

	// sha1
	{Sha1(``), "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
	{Sha1(`hello world!`), "430ce34d020724ed75a196dfc2ad67c77772d169"},

	// sha256
	{Sha256(``), "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
	{Sha256(`hello world!`), "7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9"},

	// sort
	{func() string { Sort(nil, nil); return "" }(), ""},
	{func() string { s := []int{}; Sort(s, nil); return spf("%v", s) }(), "[]"},
	{func() string { s := []int{1}; Sort(s, nil); return spf("%v", s) }(), "[1]"},
	{func() string { s := []int{1, 2}; Sort(s, nil); return spf("%v", s) }(), "[1 2]"},
	{func() string { s := []int{2, 1}; Sort(s, nil); return spf("%v", s) }(), "[1 2]"},
	{func() string { s := []int{3, 1, 2}; Sort(s, nil); return spf("%v", s) }(), "[1 2 3]"},
	{func() string { s := []string{"a"}; Sort(s, nil); return spf("%v", s) }(), "[a]"},
	{func() string { s := []string{"b", "a"}; Sort(s, nil); return spf("%v", s) }(), "[a b]"},
	{func() string { s := []string{"b", "a", "c"}; Sort(s, nil); return spf("%v", s) }(), "[a b c]"},
	{func() string { s := []bool{true, false, true}; Sort(s, nil); return spf("%v", s) }(), "[false true true]"},
	{func() string { s := []templates.HTML{`<b>`, `<a>`, `<c>`}; Sort(s, nil); return spf("%v", s) }(), "[<a> <b> <c>]"},

	// toKebab
	{ToKebab(""), ""},
	{ToKebab("AaBbCc"), "aa-bb-cc"},
	{ToKebab("aBc"), "a-bc"},
	{ToKebab("aBC"), "a-bc"},
	{ToKebab("abC"), "ab-c"},
	{ToKebab("aaBBBcc"), "aa-bb-bcc"},
	{ToKebab("AAbb"), "a-abb"},
	{ToKebab("a5"), "a5"},
	{ToKebab("A5"), "a5"},
	{ToKebab("5a"), "5a"},
	{ToKebab("5A"), "5a"},
	{ToKebab("-"), ""},
	{ToKebab("---"), ""},
	{ToKebab("a-b-c"), "a-b-c"},
	{ToKebab("A-B-C"), "a-b-c"},
	{ToKebab("AB5B-C"), "ab5b-c"},
	{ToKebab("Aa Bbb C"), "aa-bbb-c"},
	{ToKebab("AA BBB C"), "aa-bbb-c"},
	{ToKebab(" \t\n\r"), ""},
	{ToKebab(" \t\n\na"), "a"},
	{ToKebab("a \t\n\n"), "a"},
	{ToKebab("eÈè"), "e-èè"},
	{ToKebab("eÈÈè"), "e-è-èè"},
	{ToKebab("eÈÈÈè"), "e-èè-èè"},
	{ToKebab("Aaa, Bb"), "aaa-bb"},
	{ToKebab("A€B"), "a-b"},
	{ToKebab("A€€B"), "a-b"},
	{ToKebab("€€AB"), "ab"},
	{ToKebab("AB€€"), "ab"},
}

func TestBuiltins(t *testing.T) {
	for _, expr := range tests {
		if expr.got != expr.expected {
			t.Errorf("source: %q, got %q, expecting %q\n", "", expr.got, expr.expected)
		}
	}
}

var parseTimeTests = []struct {
	value    string
	expected string
}{
	{"2021-03-27T11:21:14.964553+01:00", "2021-03-27T11:21:14.964553+01:00"},
	{"2021-03-27T11:21:14+01:00", "2021-03-27T11:21:14+01:00"},
	{"2021-03-27T11:21:14", "2021-03-27T11:21:14Z"},
	{"Sat, 27 Mar 2021 11:21:14 +0100", "2021-03-27T11:21:14+01:00"},
	{"Sat, 27 Mar 2021 11:21:14 CET", "2021-03-27T11:21:14+01:00"},
	{"27 Mar 21 11:21 +0100", "2021-03-27T11:21:00+01:00"},
	{"27 Mar 21 11:21 CET", "2021-03-27T11:21:00+01:00"},
	{"Saturday, 27-Mar-21 11:21:14 CET", "2021-03-27T11:21:14+01:00"},
	{"Sat Mar 27 11:21:14 2021", "2021-03-27T11:21:14Z"},
	{"Sat Mar 27 11:21:14 CET 2021", "2021-03-27T11:21:14+01:00"},
	{"Sat Mar 27 11:21:14 +0100 2021", "2021-03-27T11:21:14+01:00"},
	{"2021-03-27 11:21:14.964553 +0100 CET", "2021-03-27T11:21:14.964553+01:00"},
	{"2021-03-27", "2021-03-27T00:00:00Z"},
	{"27 Mar 2021", "2021-03-27T00:00:00Z"},
	{"2021-03-27T11:21:14+0100", "2021-03-27T11:21:14+01:00"},
	{"2021-03-27 11:21:14 +01:00", "2021-03-27T11:21:14+01:00"},
	{"2021-03-27 11:21:14 +0100", "2021-03-27T11:21:14+01:00"},
	{"2021-03-27 11:21:14+01:00", "2021-03-27T11:21:14+01:00"},
	{"2021-03-27 11:21:14+0100", "2021-03-27T11:21:14+01:00"},
	{"2021-03-27 11:21:14", "2021-03-27T11:21:14Z"},
	{"11:21AM", "0000-01-01T11:21:00Z"},
	{"Mar 27 11:21:14", "0000-03-27T11:21:14Z"},
	{"Mar 27 11:21:14.964", "0000-03-27T11:21:14.964Z"},
	{"Mar 27 11:21:14.964553", "0000-03-27T11:21:14.964553Z"},
	{"Mar 27 11:21:14.964553000", "0000-03-27T11:21:14.964553Z"},
}

func TestParseTime(t *testing.T) {
	for _, cas := range parseTimeTests {
		tm, err := ParseTime(cas.value)
		if err != nil {
			t.Fatalf("source: %q, got error %q, expecting %q\n", cas.value, err, cas.expected)
		}
		if got := tm.t.Format(time.RFC3339Nano); got != cas.expected {
			t.Errorf("source: %q, got %q, expecting %q\n", cas.value, got, cas.expected)
		}
	}
}

var t1, _ = ParseTime("2021-03-27T11:21:14.964553+01:00")
var t2, _ = ParseTime("2021-02-12T10:16:29.370149+01:00")

var sptime = func(t Time) string { return t.t.Format(time.RFC3339Nano) }

var timeTests = []struct {
	got      string
	expected string
}{
	{sptime(t1.AddClock(18, 45, 13)), "2021-03-28T07:06:27.964553+02:00"},
	{sptime(t1.AddDate(2, 3, 7)), "2023-07-04T11:21:14.964553+02:00"},
	{sptime(t1.AddFraction(150, 350, 23)), "2021-03-27T11:21:15.114903023+01:00"},
	{spf("%t", t1.After(t2)), "true"},
	{spf("%t", t1.Before(t2)), "false"},
	{spf("%v", func() []int { h, m, s := t1.Clock(); return []int{h, m, s} }()), "[11 21 14]"},
	{spf("%v", func() []int { y, m, d := t1.Date(); return []int{y, m, d} }()), "[2021 3 27]"},
	{spf("%d", t1.Day()), "27"},
	{spf("%t", t1.Equal(t1)), "true"},
	{spf("%t", t1.Equal(t2)), "false"},
	{spf("%s", t1.Format("")), "2021-03-27T11:21:14.964553+01:00"},
	{spf("%s", t1.Format("Monday, 02-Jan-06 15:04:05 MST")), "Saturday, 27-Mar-21 11:21:14 CET"},
	{spf("%v", func() []int { ms, us, ns := t1.Fraction(); return []int{ms, us, ns} }()), "[964 964553 964553000]"},
	{spf("%d", t1.Hour()), "11"},
	{spf("%s", t1.JS()), `new Date("2021-03-27T11:21:14.964+01:00")`},
	{spf("%s", t1.JSON()), `"2021-03-27T11:21:14+01:00"`},
	{spf("%d", t1.Minute()), "21"},
	{spf("%d", t1.Month()), "3"},
	{spf("%d", t1.Second()), "14"},
	{spf("%s", t1.String()), "2021-03-27T11:21:14.964553+01:00"},
	{spf("%v", func() []int { h, m, s, ns := t1.Sub(t2); return []int{h, m, s, ns} }()), "[1033 4 45 594404000]"},
	{spf("%s", t1.UTC()), "2021-03-27T10:21:14.964553Z"},
	{spf("%d", t1.Unix()), "1616840474"},
	{spf("%d", t1.Weekday()), "6"},
	{spf("%d", t1.Year()), "2021"},
}

func TestTime(t *testing.T) {
	for _, expr := range timeTests {
		if expr.got != expr.expected {
			t.Errorf("got %q, expecting %q\n", expr.got, expr.expected)
		}
	}
}
