// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builtin

import (
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/open2b/scriggo/native"
)

var sp = fmt.Sprint
var spf = fmt.Sprintf

var toHTML = Unsafeconv.Declarations["ToHTML"].(func(string) native.HTML)
var toCSS = Unsafeconv.Declarations["ToCSS"].(func(string) native.CSS)
var toJS = Unsafeconv.Declarations["ToJS"].(func(string) native.JS)
var toJSON = Unsafeconv.Declarations["ToJSON"].(func(string) native.JSON)
var toMarkdown = Unsafeconv.Declarations["ToMarkdown"].(func(string) native.Markdown)

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
	{spf("%d", Abs(math.MaxInt)), strconv.Itoa(math.MaxInt)},
	{spf("%d", Abs(math.MinInt)), strconv.Itoa(math.MinInt)},

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

	// formatFloat
	{spf(FormatFloat(0, "f", -1)), "0"},
	{spf(FormatFloat(5.2307, "f", -1)), "5.2307"},
	{spf(FormatFloat(-5.90361e100, "g", 2)), "-5.9e+100"},

	// formatInt
	{sp(FormatInt(0, 10)), "0"},
	{sp(FormatInt(22, 10)), "22"},
	{sp(FormatInt(-22, 10)), "-22"},
	{sp(FormatInt(334, 16)), "14e"},

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

	// hasPrefix
	{sp(HasPrefix("hello, world", "hello, ")), "true"},
	{sp(HasPrefix("hello, world", "hello!")), "false"},
	{sp(HasPrefix("", "hello!")), "false"},
	{sp(HasPrefix("", "")), "true"},
	{sp(HasPrefix("abc", "")), "true"},

	// hasSuffix
	{sp(HasSuffix("hello, world", ", world")), "true"},
	{sp(HasSuffix("hello, world", "hello")), "false"},
	{sp(HasSuffix("", "hello")), "false"},
	{sp(HasSuffix("abc", "")), "true"},

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

	// indexAny
	{sp(IndexAny("hello", "h")), "0"},
	{sp(IndexAny("hello", "j")), "-1"},
	{sp(IndexAny("aaaba", "b")), "3"},
	{sp(IndexAny("-aaaba", "a")), "1"},
	{sp(IndexAny("-aaèba", "è")), "3"},
	{sp(IndexAny("-ààèba", "è")), "5"},

	// indentJSON
	{string(IndentJSON("null", "", "")), "null"},
	{string(IndentJSON("\t\n 5\t \r\n", "", "")), "5"},
	{string(IndentJSON("\t\n true\t \r\n", " ", "\t")), "true"},
	{string(IndentJSON(`{ "a": true }`, "", "\t")), "{\n\t\"a\": true\n}"},
	{
		string(IndentJSON(`{ "a": true, "b": {"c":false, "d":[1,2, 3]}}`, " ", "\t")),
		"{\n \t\"a\": true,\n \t\"b\": {\n \t\t\"c\": false,\n \t\t\"d\": [\n \t\t\t1,\n \t\t\t2,\n \t\t\t3\n \t\t]\n \t}\n }",
	},

	// join
	{sp(Join([]string{"a", "b", "c"}, " ")), "a b c"},
	{sp(Join([]string{"a", "b", "c"}, "")), "abc"},
	{sp(Join([]string{"ab", "cd", "ef"}, "\n")), "ab\ncd\nef"},
	{sp(Join([]string{}, "")), ""},
	{sp(Join([]string{}, "something")), ""},
	{sp(Join([]string(nil), "")), ""},
	{sp(Join([]string(nil), "something")), ""},

	// marshalJSON
	{(func() string { s, _ := MarshalJSON(nil); return string(s) })(), "null"},
	{(func() string { s, _ := MarshalJSON(5); return string(s) })(), "5"},
	{(func() string { s, _ := MarshalJSON([]string{"red", "green"}); return string(s) })(), `["red","green"]`},

	// marshalJSONIndent
	{(func() string { s, _ := MarshalJSONIndent(nil, "", ""); return string(s) })(), "null"},
	{(func() string { s, _ := MarshalJSONIndent(5, " ", "\t"); return string(s) })(), "5"},
	{(func() string { s, _ := MarshalJSONIndent([]string{"red", "green"}, "\t", "  "); return string(s) })(), "[\n\t  \"red\",\n\t  \"green\"\n\t]"},

	// marshalYAML
	{(func() string { s, _ := MarshalYAML(nil); return s })(), "null\n"},
	{(func() string { s, _ := MarshalYAML(5); return s })(), "5\n"},
	{(func() string { s, _ := MarshalYAML([]string{"red", "green"}); return s })(), "- red\n- green\n"},

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
		t1 := NewTime(time.Now())
		t := Now()
		t2 := NewTime(time.Now())
		return (t.Equal(t1) || t.After(t1)) && (t.Equal(t2) || t.Before(t2))
	}()), "true"},

	// parseDuration
	{sp(ParseDuration("300ms")), "300ms <nil>"},
	{sp(ParseDuration("1h20m")), "1h20m0s <nil>"},
	{sp(ParseDuration(" 1h20m")), "0s parseDuration: invalid duration \" 1h20m\""},
	{sp(ParseDuration(" 1h 20m")), "0s parseDuration: invalid duration \" 1h 20m\""},

	// parseFloat
	{sp(ParseFloat("")), "0 parseFloat: parsing \"\": invalid syntax"},
	{sp(ParseFloat("Inf")), "0 parseFloat: parsing \"Inf\": invalid syntax"},
	{sp(ParseFloat("NaN")), "0 parseFloat: parsing \"NaN\": invalid syntax"},
	{sp(ParseFloat("0")), "0 <nil>"},
	{sp(ParseFloat("23.903")), "23.903 <nil>"},
	{sp(ParseFloat("-12.052")), "-12.052 <nil>"},
	{sp(ParseFloat("7.21e14")), "7.21e+14 <nil>"},
	{sp(ParseFloat("0x1.921fbe+01")), "0 parseFloat: parsing \"0x1.921fbe+01\": invalid syntax"},

	// parseInt
	{sp(ParseInt("", 10)), "0 parseInt: parsing \"\": invalid syntax"},
	{sp(ParseInt("a", 10)), "0 parseInt: parsing \"a\": invalid syntax"},
	{sp(ParseInt("0", 10)), "0 <nil>"},
	{sp(ParseInt("23", 10)), "23 <nil>"},
	{sp(ParseInt("-12", 10)), "-12 <nil>"},
	{sp(ParseInt("f6b", 16)), "3947 <nil>"},

	// pow
	{sp(Pow(0, 0)), "1"},
	{sp(Pow(0, 1)), "0"},
	{sp(Pow(1, 1)), "1"},
	{sp(Pow(5, 3)), "125"},
	{sp(Pow(-7, 4)), "2401"},
	{sp(Pow(12, 7)), "3.5831808e+07"},
	{sp(Pow(1<<31, 2)), "4.611686018427388e+18"},
	{sp(Pow(5.3, 2.8)), "106.65284254249674"},
	{sp(Pow(-2.89, 4.11)), "NaN"},
	{sp(Pow(12.6, 7.85)), "4.3441896761340076e+08"},

	// regexp
	{spf("%t", RegExp("(scriggo){2}").Match("scriggo")), "false"},
	{spf("%t", RegExp("(scriggo){2}").Match("scriggoscriggo")), "true"},
	{spf("%t", RegExp("(scriggo){2}").Match("scriggoscriggoscriggo")), "true"},
	{RegExp("foo.?").Find("seafood fool"), "food"},
	{RegExp("foo.?").Find("meat"), ""},
	{spf("%v", RegExp("a.").FindAll("paranormal", -1)), "[ar an al]"},
	{spf("%v", RegExp("a.").FindAll("paranormal", 2)), "[ar an]"},
	{spf("%v", RegExp("a.").FindAll("graal", -1)), "[aa]"},
	{spf("%v", RegExp("a.").FindAll("none", -1)), "[]"},
	{spf("%q", RegExp("a(x*)b").FindAllSubmatch("-ab-", -1)), `[["ab" ""]]`},
	{spf("%q", RegExp("a(x*)b").FindAllSubmatch("-axxb-", -1)), `[["axxb" "xx"]]`},
	{spf("%q", RegExp("a(x*)b").FindAllSubmatch("-ab-axb-", -1)), `[["ab" ""] ["axb" "x"]]`},
	{spf("%q", RegExp("a(x*)b").FindAllSubmatch("-axxb-ab-", -1)), `[["axxb" "xx"] ["ab" ""]]`},
	{spf("%q", RegExp("a(x*)b(y|z)c").FindSubmatch("-axxxbyc-")), `["axxxbyc" "xxx" "y"]`},
	{spf("%q", RegExp("a(x*)b(y|z)c").FindSubmatch("-abzc-")), `["abzc" "" "z"]`},
	{RegExp("a(x*)b").ReplaceAll("-ab-axxb-", "T"), `-T-T-`},
	{RegExp("a(x*)b").ReplaceAll("-ab-axxb-", "$1"), `--xx-`},
	{RegExp("a(x*)b").ReplaceAll("-ab-axxb-", "$1W"), `---`},
	{RegExp("a(x*)b").ReplaceAll("-ab-axxb-", "${1}W"), `-W-xxW-`},
	{RegExp("[^aeiou]").ReplaceAllFunc("seafood fool", ToUpper), `SeaFooD FooL`},
	{spf("%v", RegExp("a").Split("banana", -1)), `[b n n ]`},
	{spf("%v", RegExp("a").Split("banana", 0)), `[]`},
	{spf("%v", RegExp("a").Split("banana", 1)), `[banana]`},
	{spf("%v", RegExp("a").Split("banana", 2)), `[b nana]`},
	{spf("%v", RegExp("z+").Split("pizza", -1)), `[pi a]`},
	{spf("%v", RegExp("z+").Split("pizza", 0)), `[]`},
	{spf("%v", RegExp("z+").Split("pizza", 1)), `[pizza]`},
	{spf("%v", RegExp("z+").Split("pizza", 2)), `[pi a]`},

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
	{func() string { s := []native.HTML{`<b>`, `<a>`, `<c>`}; Reverse(s); return spf("%v", s) }(), "[<c> <a> <b>]"},

	// runeCount
	{sp(RuneCount("a")), "1"},
	{sp(RuneCount("abc")), "3"},
	{sp(RuneCount("")), "0"},
	{sp(RuneCount("eè")), "2"},

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
	{func() string { s := []rune{'b', 'a', 'c'}; Sort(s, nil); return spf("%v", s) }(), "[97 98 99]"},
	{func() string { s := []bool{true, false, true}; Sort(s, nil); return spf("%v", s) }(), "[false true true]"},
	{func() string { s := []native.HTML{`<b>`, `<a>`, `<c>`}; Sort(s, nil); return spf("%v", s) }(), "[<a> <b> <c>]"},
	{func() string { s := []interface{}{5, 8, 2}; Sort(s, nil); return spf("%v", s) }(), "[2 5 8]"},

	// split
	{sp(Split("a b c", " ")), "[a b c]"},
	{sp(Split("a-b-c", "-")), "[a b c]"},
	{sp(Split("ax-bx-cx", " ")), "[ax-bx-cx]"},
	{sp(Split("", "-")), "[]"},

	// splitAfter
	{sp(SplitAfter("a b c", " ")), "[a  b  c]"},
	{sp(SplitAfter("a-b-c", "-")), "[a- b- c]"},
	{sp(SplitAfter("ax-bx-cx", " ")), "[ax-bx-cx]"},
	{sp(SplitAfter("", "-")), "[]"},

	// sprint
	{Sprint("x"), "x"},
	{Sprint(10), "10"},
	{Sprint(20, 30), "20 30"},
	{Sprint(20, 30, 'f'), "20 30 102"},
	{Sprint(), ""},

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

	// unixTime
	{UnixTime(0, 0).UTC().Format(time.RFC3339Nano), "1970-01-01T00:00:00Z"},
	{UnixTime(1616964058, 0).UTC().Format(time.RFC3339Nano), "2021-03-28T20:40:58Z"},
	{UnixTime(1616964058, 136918407).UTC().Format(time.RFC3339Nano), "2021-03-28T20:40:58.136918407Z"},

	// unmarshalJSON
	{spf("%#v", (func() interface{} { var v map[string]interface{}; _ = UnmarshalJSON("null", &v); return v })()), "map[string]interface {}(nil)"},
	{spf("%#v", (func() interface{} { var v map[string]interface{}; _ = UnmarshalJSON(`{"a":"b"}`, &v); return v })()), `map[string]interface {}{"a":"b"}`},
	{spf("%#v", (func() interface{} { var v []int; _ = UnmarshalJSON("[1,2,3]", &v); return v })()), "[]int{1, 2, 3}"},
	{spf("%v", UnmarshalJSON("", nil)), "unmarshalJSON: cannot unmarshal into nil"},
	{spf("%v", UnmarshalJSON("", (*int)(nil))), "unmarshalJSON: cannot unmarshal into a nil pointer of type *int"},
	{spf("%v", UnmarshalJSON("", []int{})), "unmarshalJSON: cannot unmarshal into non-pointer value of type []int"},
	{spf("%v", UnmarshalJSON("5", &[]int{})), "unmarshalJSON: cannot unmarshal number into value of type []int"},

	// unmarshalYAML
	{spf("%#v", (func() interface{} { var v map[string]interface{}; _ = UnmarshalYAML("", &v); return v })()), "map[string]interface {}(nil)"},
	{spf("%#v", (func() interface{} { var v map[string]interface{}; _ = UnmarshalYAML(`{"a":"b"}`, &v); return v })()), `map[string]interface {}{"a":"b"}`},
	{spf("%#v", (func() interface{} { var v []int; _ = UnmarshalYAML("- 1\n- 2\n- 3\n", &v); return v })()), "[]int{1, 2, 3}"},
	{spf("%v", UnmarshalYAML("", nil)), "unmarshalYAML: cannot unmarshal into nil"},
	{spf("%v", UnmarshalYAML("", (*int)(nil))), "unmarshalYAML: cannot unmarshal into a nil pointer of type *int"},
	{spf("%v", UnmarshalYAML("", []int{})), "unmarshalYAML: cannot unmarshal into non-pointer value of type []int"},
	{spf("%v", UnmarshalYAML("5", &[]int{})), "unmarshalYAML: line 1: cannot unmarshal !!int `5` into []int"},

	// unsafeconv
	{string(toHTML("<a>")), "<a>"},
	{string(toCSS("#AAA")), "#AAA"},
	{string(toJS("= undefined;")), "= undefined;"},
	{string(toJSON("[ 1, 2, 3 ]")), "[ 1, 2, 3 ]"},
	{string(toMarkdown("# a title")), "# a title"},
}

func TestBuiltins(t *testing.T) {
	for _, expr := range tests {
		if expr.got != expr.expected {
			t.Errorf("source: %q, got %q, expecting %q\n", "", expr.got, expr.expected)
		}
	}
}

// TestOnlyJSONWhitespace tests the onlyJSONWhitespace function.
func TestOnlyJSONWhitespace(t *testing.T) {
	tests := []struct {
		in       string
		expected bool
	}{
		{"", true},
		{" \t\r\n", true},
		{"\n \t\r  \n", true},
		{" a", false},
		{"è ", false},
		{"\t\n\r x \t", false},
		{"\U0001F680", false},
	}
	for _, test := range tests {
		if got := onlyJSONWhitespace(test.in); got != test.expected {
			t.Errorf("onlyJSONWhitespace(%q): expected %v, got %v", test.in, test.expected, got)
		}
	}
}

// TestTrimJSONSpace tests the trimJSONSpace function.
func TestTrimJSONSpace(t *testing.T) {
	tests := []struct {
		in       native.JSON
		expected native.JSON
	}{
		{"", ""},
		{"\n\t {\"a\": 1}\r\t", "{\"a\": 1}"},
		{"\t{\"a\": [1,2,3]}", "{\"a\": [1,2,3]}"},
		{"{\"è\":true}\n\n", "{\"è\":true}"},
		{"\U0001F680{\"a\":1}\u00a0", "\U0001F680{\"a\":1}\u00a0"},
	}
	for _, test := range tests {
		if got := trimJSONSpace(test.in); got != test.expected {
			t.Errorf("trimJSONSpace(%q): expected %q, got %q", string(test.in), string(test.expected), string(got))
		}
	}
}
