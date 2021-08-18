// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scriggo

import (
	"github.com/open2b/scriggo/native"
)

// HTMLEscape escapes s, replacing the characters <, >, &, " and ' and returns
// the escaped string as HTML type.
//
// Use HTMLEscape to put a trusted or untrusted string into an HTML element
// content or in a quoted attribute value. But don't use it with complex
// attributes like href, src, style, or any of the event handlers like
// onmouseover.
func HTMLEscape(s string) native.HTML {
	n := 0
	j := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"', '\'', '&':
			n += 4
		case '<', '>':
			n += 3
		default:
			continue
		}
		if n <= 4 {
			j = i
		}
	}
	if n == 0 {
		return native.HTML(s)
	}
	b := make([]byte, len(s)+n)
	if j > 0 {
		copy(b[:j], s[:j])
	}
	for i := j; i < len(s); i++ {
		switch c := s[i]; c {
		case '"':
			copy(b[j:], "&#34;")
			j += 5
		case '\'':
			copy(b[j:], "&#39;")
			j += 5
		case '&':
			copy(b[j:], "&amp;")
			j += 5
		case '<':
			copy(b[j:], "&lt;")
			j += 4
		case '>':
			copy(b[j:], "&gt;")
			j += 4
		default:
			b[j] = c
			j++
			continue
		}
		if j == i+n {
			copy(b[j:], s[i:])
			break
		}
	}
	return native.HTML(b)
}
