// Copyright 2026 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
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
}

func TestMarkdownEscape(t *testing.T) {
	for _, cas := range mdEscapeCases {
		out := markdownEscape(cas.src)
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
