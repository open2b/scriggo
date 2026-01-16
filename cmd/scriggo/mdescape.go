// Copyright 2026 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"strings"
)

var slash = []byte(`\`)
var nbsp = []byte("\u00a0")

// markdownEscape escapes the string s, so it can be placed inside Markdown.
func markdownEscape(s string) string {
	var b strings.Builder
	last := 0
	var esc []byte
	for i := 0; i < len(s); i++ {
		if isMarkdownEscapable(s[i]) {
			esc = slash
		} else if s[i] == ' ' || s[i] == '\t' {
			if 0 < i && i < len(s)-1 {
				if c := s[i+1]; c != ' ' && c != '\t' {
					continue
				}
			}
			esc = nbsp
		} else {
			continue
		}
		if last != i {
			b.WriteString(s[last:i])
			last = i
		}
		_, _ = b.Write(esc)
		if s[i] == ' ' || s[i] == '\t' {
			last++
		}
	}
	if last != len(s) {
		b.WriteString(s[last:])
	}
	return b.String()
}

// markdownUnescape unescapes a Markdown-escaped string.
func markdownUnescape(s []byte) (string, error) {
	var b strings.Builder
	last := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' {
			if i+1 < len(s) && isMarkdownEscapable(s[i+1]) {
				if last != i {
					b.Write(s[last:i])
				}
				b.Write(s[i+1 : i+2])
				i++
				last = i + 1
				continue
			}
		}
		if s[i] == 0xc2 && i+1 < len(s) && s[i+1] == 0xa0 {
			if last != i {
				_, _ = b.Write(s[last:i])
			}
			b.WriteByte(' ')
			i++
			last = i + 1
		}
	}
	if last != len(s) {
		b.Write(s[last:])
	}
	return b.String(), nil
}

func isMarkdownEscapable(c byte) bool {
	switch c {
	case '\\', '`', '*', '_', '{', '}', '[', ']', '(', ')', '#', '+', '-', '=', '.', '!', '|', '<', '>', '~', '&':
		return true
	}
	return false
}
