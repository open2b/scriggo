// Copyright 2026 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"
)

var mdURLEscapeCases = []struct {
	src      string
	expected string
}{
	{``, ``},
	{`a`, `a`},
	{`*`, `*`},
	{`*+`, `*+`},
	{`\`, `\\`},
	{`\\`, `\\\\`},
	{`\a`, `\a`},
	{"\\`", "\\\\`"},
	{`abc\`, `abc\\`},
	{`abc\\`, `abc\\\\`},
	{`\*\_\{\}\[\]\(\)\#\+\-\.\!\|\<\&`, `\\*\\_\\{\\}\\[\\]\\(\\)\\#\\+\\-\\.\\!\\|\\<\\&`},
	{`https://example.com/p?a=b#c`, `https://example.com/p?a=b#c`},
	{`https://example.com/p?a=b#\`, `https://example.com/p?a=b#\\`},
	{`https://example.com/p.html\?a=b#c`, `https://example.com/p.html\?a=b#c`},
}

func TestMarkdownEscape(t *testing.T) {
	for _, cas := range mdURLEscapeCases {
		out := markdownURLEscape(cas.src)
		if cas.expected != out {
			t.Fatalf("src: %q: expecting %q, got %q", cas.src, cas.expected, out)
		}
	}
}

var mdUnescapeCases = []struct {
	src      string
	expected string
}{
	{``, ``},
	{`a`, `a`},
	{`\*`, `*`},
	{`\*\+`, `*+`},
	{`\\`, `\`},
	{`\\\\`, `\\`},
	{`\\\*`, `\*`},
	{"\\\\\\`\\*\\_\\{\\}\\[\\]\\(\\)\\#\\+\\-\\.\\!\\|\\<\\&", "\\`*_{}[]()#+-.!|<&"},
	{"a\\+è\\[\\]b\\\\\\*c", "a+è[]b\\*c"},
	{"\u00a0", " "},
	{"\u00a0\u00a0", "  "},
	{"\u00a0a", " a"},
	{"\u00a0 a", "  a"},
	{"a\u00a0", "a "},
	{"a\u00a0\u00a0", "a  "},
	{`a\-b\-c\.html`, "a-b-c.html"},
}

func TestMarkdownUnescape(t *testing.T) {
	for _, cas := range mdUnescapeCases {
		out, err := markdownUnescape([]byte(cas.src))
		if err != nil {
			t.Fatalf("unescape error: %s", err)
		}
		if cas.expected != out {
			t.Fatalf("src: %q: expecting %q, got %q", cas.src, cas.expected, out)
		}
	}
}
