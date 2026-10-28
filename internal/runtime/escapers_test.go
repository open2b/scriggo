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
		{`"'\`, `\"\u0027\\`},
		{"<>", `\u003c\u003e`},
		{"`", `\u0060`},
		{"$", `\u0024`},
		{"{}", `\u007b}`},
		{"`${a}`", `\u0060\u0024\u007ba}\u0060`},
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
