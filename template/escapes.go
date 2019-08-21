// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	_base64 "encoding/base64"
	"html"
	"io"
	"strings"
)

const hexchars = "0123456789abcdef"

// TODO(Gianluca): this definition is a copy-paste from "value.go", which has
// been excluded from building. See "value.go" for further details.
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

// attributeEscape escapes the string s, so it can be placed inside an HTML
// attribute value, and write it to w. quoted reports whether the attribute is
// quoted.
func attributeEscape(w strWriter, s string, quoted bool) error {
	if quoted {
		return htmlEscape(w, s)
	}
	last := 0
	for i := 0; i < len(s); i++ {
		var esc string
		switch s[i] {
		case '<':
			esc = "&gt;"
		case '>':
			esc = "&lt;"
		case '&':
			esc = "&amp;"
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

var javaScriptStringEscapes = []string{
	0:    `\x00`,
	1:    `\x01`,
	2:    `\x02`,
	3:    `\x03`,
	4:    `\x04`,
	5:    `\x05`,
	6:    `\x06`,
	7:    `\x07`,
	8:    `\x08`,
	'\t': `\t`,
	'\n': `\n`,
	11:   `\x0b`,
	'\f': `\x0c`,
	'\r': `\r`,
	14:   `\x0e`,
	15:   `\x0f`,
	16:   `\x10`,
	17:   `\x11`,
	18:   `\x12`,
	19:   `\x13`,
	20:   `\x14`,
	21:   `\x15`,
	22:   `\x16`,
	23:   `\x17`,
	24:   `\x18`,
	25:   `\x19`,
	26:   `\x1a`,
	27:   `\x1b`,
	28:   `\x1c`,
	29:   `\x1d`,
	30:   `\x1e`,
	31:   `\x1f`,
	'"':  `\"`,
	'&':  `\x26`,
	'\'': `\'`,
	'<':  `\x3c`,
	'>':  `\x3e`,
	'\\': `\\`,
}

// javaScriptStringEscape escapes the string s so it can be placed inside a
// JavaScript and JSON string with single or double quotes, and write it to w.
func javaScriptStringEscape(w strWriter, s string) error {
	last := 0
	var buf []byte
	for i, c := range s {
		var esc string
		switch {
		case int(c) < len(javaScriptStringEscapes) && javaScriptStringEscapes[c] != "":
			esc = javaScriptStringEscapes[c]
		case c == '\u2028':
			esc = `\u2028`
		case c == '\u2029':
			esc = `\u2029`
		default:
			continue
		}
		if last != i {
			_, err := w.WriteString(s[last:i])
			if err != nil {
				return err
			}
		}
		var err error
		if esc == "" {
			_, err = w.Write(buf)
		} else {
			_, err = w.WriteString(esc)
		}
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
		case '!', '#', '$', '*', ',', '-', '.', '/', ':', ';', '=', '?', '@', '[', ']', '_':
			continue
		case '&':
			esc = "&amp;"
		case '+':
			esc = "&#34;"
		case ' ':
			if quoted {
				continue
			}
			esc = "&#32;"
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
	encoder := _base64.NewEncoder(_base64.StdEncoding, w)
	_, _ = encoder.Write(b)
	err := encoder.Close()
	if addQuote && err == nil {
		_, err = w.WriteString(`"`)
	}
	return err
}

// urlEscaper implements an io.Writer that escapes a URL or a set of URLs and
// writes it to another writer. An urlEscaper is passed to a RenderFunc
// function and to a ValueRenderer Render function when the context is
// ContextAttribute or ContextUnquotedAttribute and the value of the
// attribute is an URL or a set of URL.
type urlEscaper struct {
	// path reports whether it is currently in the path.
	path bool
	// query reports whether it is currently in the query string.
	query bool
	// addAmp reports whether an ampersand must be added before the next
	// written text.
	addAmp bool
	// isSet reports whether the attribute is a set of URLs.
	isSet bool
	// quoted reports whether the attribute is quoted.
	quoted bool
	// w is the io.Writer to write to.
	w io.Writer
}

// StartURL starts the escaping of a URL. It is called whenever a URL or a set
// of URLs is to be escaped. quoted reports whether the attribute is quoted
// and isSet reports whether the value of the attribute is a set of URLs.
func (w *urlEscaper) StartURL(quoted, isSet bool) {
	w.path = true
	w.query = false
	w.addAmp = false
	w.quoted = quoted
	w.isSet = isSet
}

// WriteText is called for each template {{ value }} in a URL attribute value.
func (w *urlEscaper) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	s := html.UnescapeString(string(p)) // TODO(Gianluca): optimize.
	sw := newStringWriter(w.w)
	if w.path {
		if strings.Contains(s, "?") {
			w.path = false
			w.addAmp = s[len(s)-1] != '?' && s[len(s)-1] != '&'
		}
		return pathEscape(sw, s, w.quoted)
	}
	if w.query {
		return queryEscape(sw, s)
	}
	panic("not w.path and not w.query...")
	return 0, nil
}

// WriteText is called for each template text in a URL attribute value.
func (w *urlEscaper) WriteText(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	n := 0
	if !w.query {
		if bytes.ContainsAny(p, "?#") {
			if p[0] == '?' && !w.path {
				if w.addAmp {
					nn, err := io.WriteString(w.w, "&amp;")
					n += nn
					if err != nil {
						return n, err
					}
				}
				p = p[1:]
			}
			w.path = false
			w.query = true
		}
		if w.isSet && bytes.ContainsRune(p, ',') {
			w.path = true
			w.query = false
		}
	}
	nn, err := w.w.Write(p)
	n += nn
	return n, err
}
