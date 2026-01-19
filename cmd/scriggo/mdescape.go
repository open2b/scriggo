// Copyright 2026 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"strings"
)

// markdownURLEscape escapes the string s, so it can be placed inside Markdown
// destination link.
func markdownURLEscape(s string) string {
	var b strings.Builder
	for s != "" {
		i := strings.IndexByte(s, '\\')
		if i == -1 {
			break
		}
		b.WriteString(s[:i+1])
		if i == len(s)-1 || isMarkdownEscapable(s[i+1]) {
			b.WriteByte('\\')
		}
		s = s[i+1:]
	}
	if b.Len() == 0 {
		return s
	}
	b.WriteString(s)
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
