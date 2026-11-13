// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"strings"
	"testing"
)

var mdEscapeCases = []struct {
	src       string
	allowHTML bool
	expected  string
}{
	{``, false, ``},
	{`a`, false, `a`},
	{`*`, false, `\*`},
	{`*+`, false, `\*\+`},
	{`\`, false, `\\`},
	{`\\`, false, `\\\\`},
	{"\\`*_{}[]()#+-.!|<&", false, "\\\\\\`\\*\\_\\{\\}\\[\\]\\(\\)\\#\\+\\-\\.\\!\\|\\<\\&"},
	{`a+è[]b\*c`, false, `a\+è\[\]b\\\*c`},
	{" ", false, "\u00a0"},
	{"  ", false, "\u00a0\u00a0"},
	{" a", false, "\u00a0a"},
	{"  a", false, "\u00a0 a"},
	{"a ", false, "a\u00a0"},
	{"a  ", false, "a\u00a0\u00a0"},
	{"\t", false, "\u00a0"},
	{"\t\t", false, "\u00a0\u00a0"},
	{"\ta", false, "\u00a0a"},
	{"\t\ta", false, "\u00a0\ta"},
	{"a\t", false, "a\u00a0"},
	{"a\t\t", false, "a\u00a0\u00a0"},
	{" \ta\t ", false, "\u00a0\ta\u00a0\u00a0"},

	{`<b>`, true, `<b>`},
	{`<b c="d">`, true, `<b c="d">`},
	{`<b c="#d">`, true, `<b c="#d">`},
	{`<b c='#d'>`, true, `<b c='#d'>`},
	{`<b c=#d>`, true, `<b c=#d>`},
	{`#a <b>#a</b> #a`, true, `\#a <b>\#a</b> \#a`},
	{`<!---->`, true, `<!---->`},
	{`#a <!-- #b --> #c <!-- #d -->`, true, `\#a <!-- #b --> \#c <!-- #d -->`},
	{`<![CDATA[]]>`, true, ``},
	{`#a <![CDATA[ #b <b>#c</b> ]]> #d`, true, "\\#a \u00a0\\#b \\<b\\>\\#c\\</b\\>\u00a0 \\#d"},
}

func TestMarkdownEscape(t *testing.T) {
	for _, cas := range mdEscapeCases {
		out := &strings.Builder{}
		err := markdownEscape(out, cas.src, cas.allowHTML)
		if err != nil {
			t.Fatalf("escape error: %s", err)
		}
		if out.String() != cas.expected {
			t.Fatalf("src: %q: expecting %q, got %q", cas.src, cas.expected, out.String())
		}
	}
}

func TestJSStringEscape(t *testing.T) {
	tests := []struct {
		src      string
		expected string
	}{
		{"", ""},
		{"a", "a"},
		{"a\u2028b&\u2029c", `a\u2028b\u0026\u2029c`},
		{`"'\`, `\u0022\u0027\\`},
		{"<>", `\u003c\u003e`},
		{"`", `\u0060`},
		{"$", `\u0024`},
		{"{}", `\u007b}`},
		{"`${a}`", `\u0060\u0024\u007ba}\u0060`},
		{"a/b", "a/b"},
		{"a*/b", "a*\\/b"},
		{"*//", "*\\//"},
		{"/*", "\\/*"},
		{"/script", "\\/script"},
		{"a//", "a//"},
		{"$\u2028{", `\u0024\u2028\u007b`},
	}
	for _, tt := range tests {
		var b strings.Builder
		err := jsStringEscape(&b, tt.src)
		if err != nil {
			t.Fatal(err)
		}
		if b.String() != tt.expected {
			t.Errorf("src: %q: expecting %q, got %q", tt.src, tt.expected, b.String())
		}
	}
}

// TestJSEscapesInMarkdown tests that an escaped string cannot close or
// interpolate a JavaScript string, template literal or regular expression
// when HTML code with a script element is shown in Markdown and the script is
// in an HTML block, where the Markdown converter does not remove the '\'
// added by markdownEscape.
func TestJSEscapesInMarkdown(t *testing.T) {
	// The sources are the ASCII characters, alone and after a '\' and a
	// '*', and the characters U+2028 and U+2029.
	var srcs []string
	for c := 0; c < 128; c++ {
		srcs = append(srcs, string(rune(c)), `\`+string(rune(c)), "*"+string(rune(c)))
	}
	srcs = append(srcs, "\u2028", "\u2029")
	for _, escaper := range []struct {
		name   string
		escape func(w strWriter, s string) error
		delims string // characters that close or interpolate the literal
	}{
		{"jsStringEscape", jsStringEscape, "\"'`$"},
		{"jsRegExpEscape", jsRegExpEscape, "/"},
	} {
		for _, src := range srcs {
			var escaped strings.Builder
			if err := escaper.escape(&escaped, src); err != nil {
				t.Fatal(err)
			}
			var b strings.Builder
			if err := markdownEscape(&b, escaped.String(), true); err != nil {
				t.Fatal(err)
			}
			out := b.String()
			for i := 0; i < len(out); i++ {
				if out[i] == '\\' {
					// It escapes the character that follows it.
					i++
				} else if strings.IndexByte(escaper.delims, out[i]) >= 0 {
					t.Errorf("%s: src: %q: %q in %q is not escaped", escaper.name, src, out[i], out)
					break
				}
			}
		}
	}
}

func TestCSSBytesEscape(t *testing.T) {
	tests := []struct {
		src      []byte
		expected string
	}{
		{nil, ""},
		{[]byte("a"), "YQ=="},
		{[]byte{0xff, 0xff}, "\\//8="},
		{[]byte{0xfe, 0xcb, 0x5a}, "\\/sta"},
		{[]byte{0x01, 0xff, 0xff}, "Af//"},
	}
	for _, tt := range tests {
		var b strings.Builder
		err := cssBytesEscape(&b, tt.src)
		if err != nil {
			t.Fatal(err)
		}
		if b.String() != tt.expected {
			t.Errorf("src: %v: expecting %q, got %q", tt.src, tt.expected, b.String())
		}
	}
}

func TestJSRegExpEscape(t *testing.T) {
	tests := []struct {
		src      string
		expected string
	}{
		{"", "(?:)"},
		{"a", "a"},
		{"a.b", "a\\u002eb"},
		{"^$.*+?()[]{}|/\\", "\\u005e\\u0024\\u002e\\u002a\\u002b\\u003f\\u0028\\u0029\\u005b\\u005d\\u007b\\u007d\\u007c\\u002f\\\\"},
		{"a-b", "a\\u002db"},
		{"\"'`&<>", "\\u0022\\u0027\\u0060\\u0026\\u003c\\u003e"},
		{"\b\t\n", "\\u0008\\u0009\\u000a"},
		{"a\u2028b\u2029c", "a\\u2028b\\u2029c"},
		{"\u00e8", "\u00e8"},
		{"!#%,:;=@~", "\\u0021\\u0023\\u0025\\u002c\\u003a\\u003b\\u003d\\u0040\\u007e"},
	}
	for _, tt := range tests {
		var b strings.Builder
		err := jsRegExpEscape(&b, tt.src)
		if err != nil {
			t.Fatal(err)
		}
		if b.String() != tt.expected {
			t.Errorf("src: %q: expecting %q, got %q", tt.src, tt.expected, b.String())
		}
	}
}
