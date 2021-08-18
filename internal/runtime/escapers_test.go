// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

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
	b := strings.Builder{}
	s := "a\u2028b&\u2029c"
	err := jsStringEscape(&b, s)
	if err != nil {
		t.Fatal(err)
	}
	if b.String() != `a\u2028b\u0026\u2029c` {
		t.Errorf("unexpected %q, expecting %q\n", b.String(), s)
	}
}
