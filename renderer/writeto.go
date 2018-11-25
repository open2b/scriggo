// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package renderer

import (
	"html"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"open2b/template/ast"

	"github.com/shopspring/decimal"
)

func (s *state) writeTo(wr io.Writer, expr interface{}, node *ast.Value, urlstate *urlState) error {

	if e, ok := expr.(WriterTo); ok {

		err := func() (err error) {
			defer func() {
				if e := recover(); e != nil {
					err = s.errorf(node, "%s", e)
				}
			}()
			_, err = e.WriteTo(wr, node.Context)
			return err
		}()
		if err != nil {
			return err
		}

	} else {

		var ok = true
		var str string
		switch node.Context {
		case ast.ContextText:
			str, ok = interfaceToText(asBase(expr))
		case ast.ContextHTML:
			str, ok = interfaceToHTML(asBase(expr))
		case ast.ContextTag:
			str, ok = interfaceToTag(asBase(expr))
		case ast.ContextAttribute:
			str, ok = interfaceToAttribute(asBase(expr), urlstate)
		case ast.ContextCSS:
			str, ok = interfaceToCSS(asBase(expr))
		case ast.ContextCSSString:
			str, ok = interfaceToCSSString(asBase(expr))
		case ast.ContextScript:
			str, ok = interfaceToScript(asBase(expr))
		case ast.ContextScriptString:
			str, ok = interfaceToScriptString(asBase(expr))
		default:
			panic("template/renderer: unknown context")
		}
		if !ok {
			err := s.errorf(node.Expr, "wrong type %s in value", typeof(expr))
			if !s.handleError(err) {
				return err
			}
		}
		_, err := io.WriteString(wr, str)
		if err != nil {
			return err
		}

	}

	return nil
}

func interfaceToText(expr interface{}) (string, bool) {

	if expr == nil {
		return "", true
	}

	var s string

	switch e := expr.(type) {
	case string:
		s = e
	case HTML:
		s = string(e)
	case int:
		s = strconv.Itoa(e)
	case decimal.Decimal:
		s = e.String()
	case bool:
		s = "false"
		if e {
			s = "true"
		}
	default:
		rv := reflect.ValueOf(expr)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return "", false
		}
		if rv.IsNil() || rv.Len() == 0 {
			return "", true
		}
		buf := make([]string, rv.Len())
		for i := 0; i < len(buf); i++ {
			str, ok := interfaceToText(rv.Index(i).Interface())
			if !ok {
				return "", false
			}
			buf[i] = str
		}
		s = strings.Join(buf, ", ")
	}

	return s, true
}

func interfaceToHTML(expr interface{}) (string, bool) {

	if expr == nil {
		return "", true
	}

	var s string

	switch e := expr.(type) {
	case string:
		s = html.EscapeString(e)
	case HTML:
		s = string(e)
	case int:
		s = strconv.Itoa(e)
	case decimal.Decimal:
		s = e.String()
	case bool:
		s = "false"
		if e {
			s = "true"
		}
	default:
		rv := reflect.ValueOf(expr)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return "", false
		}
		if rv.IsNil() || rv.Len() == 0 {
			return "", true
		}
		buf := make([]string, rv.Len())
		for i := 0; i < len(buf); i++ {
			str, ok := interfaceToHTML(rv.Index(i).Interface())
			if !ok {
				return "", false
			}
			buf[i] = str
		}
		s = strings.Join(buf, ", ")
	}

	return s, true
}

func interfaceToTag(expr interface{}) (string, bool) {
	// TODO(marco): return a more explanatory error
	s, ok := interfaceToText(expr)
	if !ok {
		return "", false
	}
	i := 0
	for j, c := range s {
		if c == utf8.RuneError && j == i+1 {
			return "", false
		}
		const DEL = 0x7F
		if c <= 0x1F || c == '"' || c == '\'' || c == '>' || c == '/' || c == '=' || c == DEL ||
			0x7F <= c && c <= 0x9F || unicode.Is(unicode.Noncharacter_Code_Point, c) {
			return "", false
		}
		i = j
	}
	return s, true
}

func interfaceToAttribute(expr interface{}, urlstate *urlState) (string, bool) {

	if expr == nil {
		return "", true
	}

	var s string

	switch e := expr.(type) {
	case string:
		s = e
		if urlstate == nil {
			s = html.EscapeString(e)
		}
	case HTML:
		s = string(e)
		if urlstate == nil {
			s = html.EscapeString(html.UnescapeString(string(e)))
		}
	case int:
		s = strconv.Itoa(e)
	case decimal.Decimal:
		s = e.String()
	case bool:
		s = "false"
		if e {
			s = "true"
		}
	default:
		rv := reflect.ValueOf(expr)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return "", false
		}
		if rv.IsNil() || rv.Len() == 0 {
			return "", true
		}
		buf := make([]string, rv.Len())
		for i := 0; i < len(buf); i++ {
			str, ok := interfaceToAttribute(rv.Index(i).Interface(), urlstate)
			if !ok {
				return "", false
			}
			buf[i] = str
		}
		s = strings.Join(buf, ", ")
	}

	if s == "" {
		return "", true
	}

	if urlstate != nil {
		switch {
		case urlstate.path:
			if strings.Contains(s, "?") {
				urlstate.path = false
				urlstate.addAmp = s[len(s)-1] != '?' && s[len(s)-1] != '&'
			}
			s = pathEscape(s)
		case urlstate.query:
			s = queryEscape(s)
		default:
			return "", false
		}
	}

	return s, true
}

func interfaceToCSS(expr interface{}) (string, bool) {

	if expr == nil {
		return "", false
	}

	switch e := expr.(type) {
	case string:
		return "\"" + stringToCSS(e) + "\"", true
	case HTML:
		return "\"" + stringToCSS(string(e)) + "\"", true
	case int:
		return strconv.Itoa(e), true
	case decimal.Decimal:
		return e.String(), true
	}

	return "", false
}

func interfaceToCSSString(expr interface{}) (string, bool) {

	if expr == nil {
		return "", false
	}

	switch e := expr.(type) {
	case string:
		return stringToCSS(e), true
	case HTML:
		return stringToCSS(string(e)), true
	case int:
		return strconv.Itoa(e), true
	case decimal.Decimal:
		return e.String(), true
	}

	return "", false
}

func prefixWithSpace(c byte) bool {
	switch c {
	case '\t', '\n', '\f', '\r', ' ':
		return true
	}
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'b' || 'A' <= c && c <= 'B'
}

func stringToCSS(s string) string {
	more := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if i > 0 && s[i-1] != '\\' && prefixWithSpace(c) {
			more++
		}
		switch c {
		case '"', '&', '\'', '(', ')', '+', '/', ':', ';', '<', '>', '{', '}':
			more += 2
		default:
			if c <= 0x1F {
				more += 1
			}
		}
	}
	if more == 0 {
		return s
	}
	b := make([]byte, len(s)+more)
	j := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if i > 0 && s[i-1] != '\\' && prefixWithSpace(c) {
			b[j] = ' '
			j++
		}
		switch c {
		case '"', '&', '\'', '(', ')', '+', '/', ':', ';', '<', '>', '{', '}':
			b[j] = '\\'
			b[j+1] = hexchars[c>>4]
			b[j+2] = hexchars[c&0xF]
			j += 3
		default:
			if c <= 0x1F {
				b[j] = '\\'
				b[j+1] = hexchars[c&0xF]
				j += 2
			} else {
				b[j] = c
				j++
			}
		}
	}
	return string(b)
}

var mapStringToInterfaceType = reflect.TypeOf(map[string]interface{}{})

func interfaceToScript(expr interface{}) (string, bool) {

	if expr == nil {
		return "null", true
	}

	switch e := expr.(type) {
	case string:
		return "\"" + stringToScript(e) + "\"", true
	case HTML:
		return "\"" + stringToScript(string(e)) + "\"", true
	case int:
		return strconv.Itoa(e), true
	case decimal.Decimal:
		return e.String(), true
	case bool:
		if e {
			return "true", true
		}
		return "false", true
	default:
		rv := reflect.ValueOf(expr)
		if !rv.IsValid() {
			return "undefined", false
		}
		switch rv.Kind() {
		case reflect.Slice:
			if rv.IsNil() {
				return "null", true
			}
			if rv.Len() == 0 {
				return "[]", true
			}
			s := "["
			for i := 0; i < rv.Len(); i++ {
				if i > 0 {
					s += ","
				}
				s2, ok := interfaceToScript(rv.Index(i).Interface())
				if !ok {
					return "", false
				}
				s += s2
			}
			return s + "]", true
		case reflect.Struct:
			return structToScript(rv)
		case reflect.Map:
			if rv.IsNil() {
				return "null", true
			}
			if !rv.Type().ConvertibleTo(mapStringToInterfaceType) {
				return "undefined", false
			}
			return mapToScript(rv.Convert(mapStringToInterfaceType).Interface().(map[string]interface{}))
		case reflect.Ptr:
			if rv.IsNil() {
				return "null", true
			}
			rt := rv.Type().Elem()
			if rt.Kind() != reflect.Struct {
				return "undefined", false
			}
			rv = rv.Elem()
			if !rv.IsValid() {
				return "undefined", false
			}
			return structToScript(rv)
		}
	}

	return "undefined", false
}

func interfaceToScriptString(expr interface{}) (string, bool) {

	if expr == nil {
		return "", false
	}

	switch e := expr.(type) {
	case string:
		return stringToScript(e), true
	case HTML:
		return stringToScript(string(e)), true
	case int:
		return strconv.Itoa(e), true
	case decimal.Decimal:
		return e.String(), true
	}

	return "", false
}

const hexchars = "0123456789abcdef"

func stringToScript(s string) string {
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

func structToScript(v reflect.Value) (string, bool) {
	var s string
	fields := getStructFields(v)
	for _, name := range fields.names {
		if len(s) > 0 {
			s += ","
		}
		s += "\"" + stringToScript(name) + "\":"
		s2, ok := interfaceToScript(v.Field(fields.indexOf[name]).Interface())
		if !ok {
			return "undefined", false
		}
		s += s2
	}
	return "{" + s + "}", true
}

func mapToScript(e map[string]interface{}) (string, bool) {
	if e == nil {
		return "null", true
	}
	var s string
	names := make([]string, 0, len(e))
	for name := range e {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, n := range names {
		if len(s) > 0 {
			s += ","
		}
		s += stringToScript(n) + ":"
		s2, ok := interfaceToScript(e[n])
		if !ok {
			return "undefined", false
		}
		s += s2
	}
	return "{" + s + "}", true
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
