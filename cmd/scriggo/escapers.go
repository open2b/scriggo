// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"io"
)

// jsStringEscapes contains the runes that must be escaped when placed within
// a JavaScript and JSON string with single or double quotes, in addition to
// the runes U+2028 and U+2029.
var jsStringEscapes = [...]string{
	0:    `\u0000`,
	1:    `\u0001`,
	2:    `\u0002`,
	3:    `\u0003`,
	4:    `\u0004`,
	5:    `\u0005`,
	6:    `\u0006`,
	7:    `\u0007`,
	'\b': `\b`,
	'\t': `\t`,
	'\n': `\n`,
	'\v': `\u000b`,
	'\f': `\f`,
	'\r': `\r`,
	14:   `\u000e`,
	15:   `\u000f`,
	16:   `\u0010`,
	17:   `\u0011`,
	18:   `\u0012`,
	19:   `\u0013`,
	20:   `\u0014`,
	21:   `\u0015`,
	22:   `\u0016`,
	23:   `\u0017`,
	24:   `\u0018`,
	25:   `\u0019`,
	26:   `\u001a`,
	27:   `\u001b`,
	28:   `\u001c`,
	29:   `\u001d`,
	30:   `\u001e`,
	31:   `\u001f`,
	'"':  `\"`,
	'&':  `\u0026`,
	'\'': `\u0027`,
	'<':  `\u003c`,
	'>':  `\u003e`,
	'\\': `\\`,
}

// jsStringEscape escapes the string s so it can be placed within a JavaScript
// and JSON string with single or double quotes, and write it to w.
func jsStringEscape(w io.Writer, s string) {
	last := 0
	for i, c := range s {
		var esc string
		switch {
		case int(c) < len(jsStringEscapes):
			esc = jsStringEscapes[c]
		case c == '\u2028':
			esc = `\u2028`
		case c == '\u2029':
			esc = `\u2029`
		}
		if esc == "" {
			continue
		}
		if last != i {
			_, _ = io.WriteString(w, s[last:i])
		}
		_, _ = io.WriteString(w, esc)
		if c == '\u2028' || c == '\u2029' {
			last = i + 3
		} else {
			last = i + 1
		}
	}
	if last != len(s) {
		_, _ = io.WriteString(w, s[last:])
	}
}
