// Copyright 2026 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"net/url"
	"testing"
)

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	return u
}

// TestLinkDestinationReplacer exercises link rewriting across inline links,
// reference definitions, and contexts that must be ignored.
func TestLinkDestinationReplacer(t *testing.T) {
	base := mustParseURL(t, "https://example.com/base")
	replacer := linkDestinationReplacer{base: base, dir: "docs"}

	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "inlineRelative",
			src:  "[API](api)",
			want: "[API](" + markdownEscape("https://example.com/base/docs/api.md") + ")",
		},
		{
			name: "inlineHTML",
			src:  "[API](api.html)",
			want: "[API](" + markdownEscape("https://example.com/base/docs/api.md") + ")",
		},
		{
			name: "inlineMD",
			src:  "[Read](readme.md)",
			want: "[Read](" + markdownEscape("https://example.com/base/docs/readme.md") + ")",
		},
		{
			name: "inlineOtherExt",
			src:  "[Pic](img/logo.png)",
			want: "[Pic](" + markdownEscape("https://example.com/base/docs/img/logo.png") + ")",
		},
		{
			name: "absolutePath",
			src:  "[Abs](/guide)",
			want: "[Abs](" + markdownEscape("https://example.com/base/guide.md") + ")",
		},
		{
			name: "trailingSlash",
			src:  "[Dir](guide/)",
			want: "[Dir](" + markdownEscape("https://example.com/base/docs/guide/") + ")",
		},
		{
			name: "queryFragment",
			src:  "[Q](api?x=1#y)",
			want: "[Q](" + markdownEscape("https://example.com/base/docs/api.md?x=1#y") + ")",
		},
		{
			name: "schemeRelative",
			src:  "[CDN](//cdn.example.com/lib.js)",
			want: "[CDN](" + markdownEscape("https://cdn.example.com/lib.js") + ")",
		},
		{
			name: "absoluteURL",
			src:  "[Go](https://golang.org/doc)",
			want: "[Go](https://golang.org/doc)",
		},
		{
			name: "queryOnly",
			src:  "[q](?x=1)",
			want: "[q](?x=1)",
		},
		{
			name: "fragmentOnly",
			src:  "[f](#sec)",
			want: "[f](#sec)",
		},
		{
			name: "referenceDefinition",
			src:  "[ref]: api \"Title\"\nSee [ref].",
			want: "[ref]: " + markdownEscape("https://example.com/base/docs/api.md") + " \"Title\"\nSee [ref].",
		},
		{
			name: "inlineCode",
			src:  "`[API](api)`",
			want: "`[API](api)`",
		},
		{
			name: "fencedCode",
			src:  "```\n[API](api)\n```\n",
			want: "```\n[API](api)\n```\n",
		},
		{
			name: "indentedCode",
			src:  "    [API](api)",
			want: "    [API](api)",
		},
		{
			name: "angleDestination",
			src:  "[t](<api.html>)",
			want: "[t](<" + markdownEscape("https://example.com/base/docs/api.md") + ">)",
		},
		{
			name: "escapedParens",
			src:  "[t](a\\(b\\).html)",
			want: "[t](" + markdownEscape("https://example.com/base/docs/a%28b%29.md") + ")",
		},
		{
			name: "multipleLinks",
			src:  "[a](a) [b](b.html)",
			want: "[a](" + markdownEscape("https://example.com/base/docs/a.md") + ") [b](" + markdownEscape("https://example.com/base/docs/b.md") + ")",
		},
		{
			name: "dotPath",
			src:  "[here](./guide)",
			want: "[here](" + markdownEscape("https://example.com/base/docs/guide.md") + ")",
		},
		{
			name: "dotDotPath",
			src:  "[up](../guide)",
			want: "[up](" + markdownEscape("https://example.com/base/guide.md") + ")",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dst bytes.Buffer
			err := replacer.replace(&dst, []byte(tt.src))
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			got := dst.String()
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

// TestLinkDestinationReplacerErrors checks nil handling in the replacer API.
func TestLinkDestinationReplacerErrors(t *testing.T) {
	base := mustParseURL(t, "https://example.com")
	replacer := linkDestinationReplacer{base: base, dir: "docs"}

	if err := replacer.replace(nil, []byte("[x](y)")); err == nil {
		t.Fatalf("expected error, got nil")
	}
	var dst bytes.Buffer
	if err := replacer.replace(&dst, nil); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
