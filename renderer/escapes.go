// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package renderer

import "strings"

const hexchars = "0123456789abcdef"

// prefixWithSpace indicates if the byte c, in a CSS string, must be preceded
// by a space when an escape sequence precedes it.
func prefixWithSpace(c byte) bool {
	switch c {
	case '\t', '\n', '\f', '\r', ' ':
		return true
	}
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'b' || 'A' <= c && c <= 'B'
}

// cssStringEscape escapes the string s so it can be places inside a CSS
// string with single or double quotes.
func cssStringEscape(s string) string {
	more := 0
MORE:
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"', '&', '\'', '(', ')', '+', '/', ':', ';', '<', '>', '{', '}':
			more += 2
		default:
			if c <= 0x0F {
				more += 1
			} else if c <= 0x1F {
				more += 2
			} else {
				continue MORE
			}
		}
		if c != '\\' && (i == len(s)-1 || prefixWithSpace(s[i+1])) {
			more++
		}
	}
	if more == 0 {
		return s
	}
	b := make([]byte, len(s)+more)
	j := 0
ESCAPE:
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"', '&', '\'', '(', ')', '+', '/', ':', ';', '<', '>', '{', '}':
			b[j] = '\\'
			b[j+1] = hexchars[c>>4]
			b[j+2] = hexchars[c&0xF]
			j += 3
		default:
			if c <= 0x0F {
				b[j] = '\\'
				b[j+1] = hexchars[c&0xF]
				j += 2
			} else if c <= 0x1F {
				b[j] = '\\'
				b[j+1] = hexchars[c>>4]
				b[j+2] = hexchars[c&0xF]
				j += 3
			} else {
				b[j] = c
				j++
				continue ESCAPE
			}
		}
		if c != '\\' && (i == len(s)-1 || prefixWithSpace(s[i+1])) {
			b[j] = ' '
			j++
		}
	}
	return string(b)
}

// scriptStringEscape escapes the string s so it can be places inside a
// JavaScript and JSON string with single or double quotes.
func scriptStringEscape(s string) string {
	if len(s) == 0 {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString("\\\\")
		case '"':
			b.WriteString("\\\"")
		case '\'':
			b.WriteString("\\'")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		case '\u2028':
			b.WriteString("\\u2028")
		case '\u2029':
			b.WriteString("\\u2029")
		default:
			if r <= 31 || r == '<' || r == '>' || r == '&' {
				b.WriteString("\\x")
				b.WriteByte(hexchars[r>>4])
				b.WriteByte(hexchars[r&0xF])
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// pathEscape escapes the string s so it can be placed inside a URL path.
// Note that url.PathEscape escapes '/' as '%2F' and ' ' as '%20'.
func pathEscape(s string) string {
	more := 0
	for i := 0; i < len(s); i++ {
		if c := s[i]; !('0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
			switch c {
			case ' ', '!', '#', '$', '*', ',', '-', '.', '/', ':', ';', '=', '?', '@', '[', ']', '_':
			case '&', '+':
				more += 4
			default:
				more += 2
			}
		}
	}
	if more == 0 {
		return s
	}
	b := make([]byte, len(s)+more)
	for i, j := 0, 0; i < len(s); i++ {
		c := s[i]
		if '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' {
			b[j] = c
			j++
			continue
		}
		switch c {
		case ' ', '!', '#', '$', '*', ',', '-', '.', '/', ':', ';', '=', '?', '@', '[', ']', '_':
			b[j] = c
			j++
		case '&':
			b[j] = '&'
			b[j+1] = 'a'
			b[j+2] = 'm'
			b[j+3] = 'p'
			b[j+4] = ';'
			j += 5
		case '+':
			b[j] = '&'
			b[j+1] = '#'
			b[j+2] = '4'
			b[j+3] = '3'
			b[j+4] = ';'
			j += 5
		default:
			b[j] = '%'
			b[j+1] = hexchars[c>>4]
			b[j+2] = hexchars[c&0xF]
			j += 3
		}
	}
	return string(b)
}

// queryEscape escapes the string s so it can be placed inside a URL query.
// Note that url.QueryEscape escapes ' ' as '+' and not as '%20'.
func queryEscape(s string) string {
	more := 0
	for i := 0; i < len(s); i++ {
		if c := s[i]; !('0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' ||
			c == '-' || c == '.' || c == '_') {
			more += 2
		}
	}
	if more == 0 {
		return s
	}
	b := make([]byte, len(s)+more)
	for i, j := 0, 0; i < len(s); i++ {
		c := s[i]
		if '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' ||
			c == '-' || c == '.' || c == '_' {
			b[j] = c
			j++
		} else {
			b[j] = '%'
			b[j+1] = hexchars[c>>4]
			b[j+2] = hexchars[c&0xF]
			j += 3
		}
	}
	return string(b)
}
