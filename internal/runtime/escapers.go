// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"encoding/base64"
	"errors"
	"strings"
)

const hexchars = "0123456789abcdef"

type strWriter interface {
	Write(b []byte) (int, error)
	WriteString(s string) (int, error)
}

// htmlEscape escapes the string s, so it can be placed inside HTML, and
// writes it on w.
func htmlEscape(w strWriter, s string) error {
	last := 0
	for i := 0; i < len(s); i++ {
		var esc string
		switch s[i] {
		case '"':
			esc = "&#34;"
		case '\'':
			esc = "&#39;"
		case '&':
			esc = "&amp;"
		case '<':
			esc = "&lt;"
		case '>':
			esc = "&gt;"
		default:
			continue
		}
		if last != i {
			_, err := w.WriteString(s[last:i])
			if err != nil {
				return err
			}
		}
		_, err := w.WriteString(esc)
		if err != nil {
			return err
		}
		last = i + 1
	}
	if last != len(s) {
		_, err := w.WriteString(s[last:])
		return err
	}
	return nil
}

// htmlNoEntitiesEscape escapes and writes to w the string s as htmlEscape
// does but without escaping the HTML entities.
func htmlNoEntitiesEscape(w strWriter, s string) error {
	last := 0
	for i := 0; i < len(s); i++ {
		var esc string
		switch s[i] {
		case '"':
			esc = "&#34;"
		case '\'':
			esc = "&#39;"
		case '<':
			esc = "&lt;"
		case '>':
			esc = "&gt;"
		default:
			continue
		}
		if last != i {
			_, err := w.WriteString(s[last:i])
			if err != nil {
				return err
			}
		}
		_, err := w.WriteString(esc)
		if err != nil {
			return err
		}
		last = i + 1
	}
	if last != len(s) {
		_, err := w.WriteString(s[last:])
		return err
	}
	return nil
}

// attributeEscape escapes the string s, so it can be placed inside an HTML
// attribute value, and write it to w. If escapeEntities is true it escapes
// also the HTML entities. quoted reports whether the attribute is quoted.
func attributeEscape(w strWriter, s string, escapeEntities, quoted bool) error {
	if quoted {
		if escapeEntities {
			return htmlEscape(w, s)
		}
		return htmlNoEntitiesEscape(w, s)
	}
	last := 0
	for i := 0; i < len(s); i++ {
		var esc string
		switch s[i] {
		case '<':
			esc = "&lt;"
		case '>':
			esc = "&gt;"
		case '&':
			if escapeEntities {
				esc = "&amp;"
			}
		case '\t':
			esc = "&#09;"
		case '\n':
			esc = "&#10;"
		case '\r':
			esc = "&#13;"
		case '\x0C':
			esc = "&#12;"
		case ' ':
			esc = "&#32;"
		case '"':
			esc = "&#34;"
		case '\'':
			esc = "&#39;"
		case '=':
			esc = "&#61;"
		case '`':
			esc = "&#96;"
		}
		if esc == "" {
			continue
		}
		if last != i {
			_, err := w.WriteString(s[last:i])
			if err != nil {
				return err
			}
		}
		_, err := w.WriteString(esc)
		if err != nil {
			return err
		}
		last = i + 1
	}
	if last != len(s) {
		_, err := w.WriteString(s[last:])
		return err
	}
	return nil
}

// prefixWithSpace reports whether the byte c, in a CSS string, must be
// preceded by a space when an escape sequence precedes it.
func prefixWithSpace(c byte) bool {
	switch c {
	case '\t', '\n', '\f', '\r', ' ':
		return true
	}
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'b' || 'A' <= c && c <= 'B'
}

var cssStringEscapes = []string{
	0:    `\0`,
	1:    `\1`,
	2:    `\2`,
	3:    `\3`,
	4:    `\4`,
	5:    `\5`,
	6:    `\6`,
	7:    `\7`,
	8:    `\8`,
	'\t': `\9`,
	'\n': `\a`,
	11:   `\b`,
	'\f': `\c`,
	'\r': `\d`,
	14:   `\e`,
	15:   `\f`,
	16:   `\10`,
	17:   `\11`,
	18:   `\12`,
	19:   `\13`,
	20:   `\14`,
	21:   `\15`,
	22:   `\16`,
	23:   `\17`,
	24:   `\18`,
	25:   `\19`,
	26:   `\1a`,
	27:   `\1b`,
	28:   `\1c`,
	29:   `\1d`,
	30:   `\1e`,
	31:   `\1f`,
	'"':  `\22`,
	'&':  `\26`,
	'\'': `\27`,
	'(':  `\28`,
	')':  `\29`,
	'+':  `\2b`,
	'/':  `\2f`,
	':':  `\3a`,
	';':  `\3b`,
	'<':  `\3c`,
	'>':  `\3e`,
	'\\': `\\`,
	'{':  `\7b`,
	'}':  `\7d`,
}

// cssStringEscape escapes the string s, so it can be placed inside a CSS
// string with single or double quotes, and write it to w.
func cssStringEscape(w strWriter, s string) error {
	last := 0
	for i := 0; i < len(s); i++ {
		var esc string
		c := s[i]
		if int(c) < len(cssStringEscapes) {
			esc = cssStringEscapes[c]
		}
		if esc == "" {
			continue
		}
		if last != i {
			_, err := w.WriteString(s[last:i])
			if err != nil {
				return err
			}
		}
		_, err := w.WriteString(esc)
		if err != nil {
			return err
		}
		if c != '\\' && (i == len(s)-1 || prefixWithSpace(s[i+1])) {
			_, err = w.WriteString(" ")
			if err != nil {
				return err
			}
		}
		last = i + 1
	}
	if last != len(s) {
		_, err := w.WriteString(s[last:])
		return err
	}
	return nil
}

// jsStringEscapes contains the runes that must be escaped when placed within
// a JavaScript and JSON string with single or double quotes, in addition to
// the runes U+2028 and U+2029.
var jsStringEscapes = []string{
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
func jsStringEscape(w strWriter, s string) error {
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
			_, err := w.WriteString(s[last:i])
			if err != nil {
				return err
			}
		}
		_, err := w.WriteString(esc)
		if err != nil {
			return err
		}
		if c == '\u2028' || c == '\u2029' {
			last = i + 3
		} else {
			last = i + 1
		}
	}
	if last != len(s) {
		_, err := w.WriteString(s[last:])
		return err
	}
	return nil
}

// jsonStringEscape escapes the string s so it can be placed within a JSON
// string, and write it to w.
//
// jsonStringEscape calls the jsStringEscape function that can also
// escape JSON.
func jsonStringEscape(w strWriter, s string) error {
	return jsStringEscape(w, s)
}

func isHexDigit(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

// pathEscape escapes the string s so it can be placed inside an attribute
// value as URL path, and write it to w. quoted reports whether the attribute
// is quoted. It returns the number of bytes written and the first error
// encountered.
//
// Note that url.PathEscape escapes '/' as '%2F' and ' ' as '%20'.
func pathEscape(w strWriter, s string, quoted bool) (int, error) {
	n := 0
	last := 0
	var buf []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' {
			continue
		}
		var esc string
		switch c {
		case '!', '#', '$', '*', ',', '-', '.', '/', ':', ';', '=', '?', '@', '[', ']', '_', '~':
			continue
		case '&':
			esc = "&amp;"
		case '+':
			esc = "&#43;"
		case ' ':
			if quoted {
				continue
			}
			esc = "&#32;"
		case '%':
			if i+2 < len(s) && isHexDigit(s[i+1]) && isHexDigit(s[i+2]) {
				continue
			}
			fallthrough
		default:
			if buf == nil {
				buf = make([]byte, 3)
				buf[0] = '%'
			}
			buf[1] = hexchars[c>>4]
			buf[2] = hexchars[c&0xF]
		}
		if last != i {
			nn, err := w.WriteString(s[last:i])
			n += nn
			if err != nil {
				return n, err
			}
		}
		var nn int
		var err error
		if esc == "" {
			nn, err = w.Write(buf)
		} else {
			nn, err = w.WriteString(esc)
		}
		n += nn
		if err != nil {
			return n, err
		}
		last = i + 1
	}
	if last != len(s) {
		nn, err := w.WriteString(s[last:])
		n += nn
		return n, err
	}
	return n, nil
}

// queryEscape escapes the string s, so it can be placed inside a URL query,
// and write it to w. It returns the number of bytes written and the first
// error encountered.
//
// Note that url.QueryEscape escapes ' ' as '+' and not as '%20'.
func queryEscape(w strWriter, s string) (int, error) {
	n := 0
	last := 0
	var buf []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' ||
			c == '-' || c == '.' || c == '_' {
			continue
		}
		if buf == nil {
			buf = make([]byte, 3)
			buf[0] = '%'
		}
		buf[1] = hexchars[c>>4]
		buf[2] = hexchars[c&0xF]
		if last != i {
			nn, err := w.WriteString(s[last:i])
			n += nn
			if err != nil {
				return n, err
			}
		}
		nn, err := w.Write(buf)
		n += nn
		if err != nil {
			return n, err
		}
		last = i + 1
	}
	if last != len(s) {
		nn, err := w.WriteString(s[last:])
		n += nn
		return n, err
	}
	return n, nil
}

// escapeBytes escapes b as Base64, so it can be placed inside JavaScript and
// CSS, and write it to w. addQuote indicates whether it should add quotes.
func escapeBytes(w strWriter, b []byte, addQuote bool) error {
	if addQuote {
		_, err := w.WriteString(`"`)
		if err != nil {
			return err
		}
	}
	encoder := base64.NewEncoder(base64.StdEncoding, w)
	_, _ = encoder.Write(b)
	err := encoder.Close()
	if addQuote && err == nil {
		_, err = w.WriteString(`"`)
	}
	return err
}

var slash = []byte(`\`)
var nbsp = []byte("\u00a0")

// isCDATA reports whether s at index p contains a isCDATA section.
func isCDATA(s string, p int) bool {
	if len(s) < p+9 {
		return false
	}
	for i := 0; i < 9; i++ {
		if s[p+i] != "<![CDATA["[i] {
			return false
		}
	}
	return true
}

// isHTMLComment reports whether s at index p contains a starting HTML
// comment.
func isHTMLComment(s string, p int) bool {
	if len(s) < p+4 {
		return false
	}
	for i := 0; i < 4; i++ {
		if s[p+i] != "<!--"[i] {
			return false
		}
	}
	return true
}

// markdownEscape escapes the string s, so it can be placed inside Markdown,
// and writes it to w. allowHTML indicates if HTML code is allowed and so it
// is not escaped.
func markdownEscape(w strWriter, s string, allowHTML bool) error {
	last := 0
	var esc []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			if allowHTML {
				if isHTMLComment(s, i) {
					// Don't escape the comment.
					i += 4
					p := strings.Index(s[i:], "-->")
					if p < 0 {
						return errors.New("not closed HTML comment")
					}
					i += p + 2
				} else if isCDATA(s, i) {
					// Replace the CDATA section with the escape of its content.
					if last != i {
						_, err := w.WriteString(s[last:i])
						if err != nil {
							return err
						}
					}
					i += 9
					p := strings.Index(s[i:], "]]>")
					if p < 0 {
						return errors.New("not closed CDATA section")
					}
					err := markdownEscape(w, s[i:i+p], false)
					if err != nil {
						return err
					}
					i += p + 2
					last = i + 1
				} else {
					// Don't escape the tag.
					var quote byte
					for ; i < len(s); i++ {
						c := s[i]
						if quote == 0 {
							if c == '>' {
								break
							} else if c == '"' || c == '\'' {
								quote = c
							}
						} else if c == quote {
							quote = 0
						}
					}
				}
				continue
			}
			esc = slash
		case '\\', '`', '*', '_', '{', '}', '[', ']', '(', ')', '#', '+', '-', '=', '.', '!', '|', '>', '~':
			esc = slash
		case '&':
			if !allowHTML {
				esc = slash
			}
		case ' ', '\t':
			if 0 < i && i < len(s)-1 {
				if c := s[i+1]; c != ' ' && c != '\t' {
					continue
				}
			}
			esc = nbsp
		default:
			continue
		}
		if last != i {
			_, err := w.WriteString(s[last:i])
			if err != nil {
				return err
			}
			last = i
		}
		_, err := w.Write(esc)
		if err != nil {
			return err
		}
		if s[i] == ' ' || s[i] == '\t' {
			last++
		}
	}
	if last != len(s) {
		_, err := w.WriteString(s[last:])
		return err
	}
	return nil
}

var tab = []byte{'\t'}
var fourSpaces = []byte{' ', ' ', ' ', ' '}

// markdownCodeBlockEscape escapes the string s, so it can be placed inside
// a Markdown code block, and writes it on w. spaces indicates if the line
// is indented with spaces instead of a tab.
func markdownCodeBlockEscape(w strWriter, s string, spaces bool) error {
	last := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if i+1 < len(s) && s[i+1] == '\r' {
				i++
			}
			_, err := w.WriteString(s[last : i+1])
			if err != nil {
				return err
			}
			if spaces {
				_, err = w.Write(fourSpaces)
			} else {
				_, err = w.Write(tab)
			}
			if err != nil {
				return err
			}
			last = i + 1
		}
	}
	if last != len(s) {
		_, err := w.WriteString(s[last:])
		return err
	}
	return nil
}
