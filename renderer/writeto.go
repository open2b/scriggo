//
// Copyright (c) 2017-2018 Open2b Software Snc. All Rights Reserved.
//

package renderer

import (
	"bytes"
	"html"
	"io"
	"reflect"
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
			str, ok = interfaceToText(asBase(expr), s.version)
		case ast.ContextHTML:
			str, ok = interfaceToHTML(asBase(expr), s.version)
		case ast.ContextTag:
			str, ok = interfaceToTag(asBase(expr), s.version)
		case ast.ContextAttribute:
			str, ok = interfaceToAttribute(asBase(expr), s.version, urlstate)
		case ast.ContextCSS:
			str, ok = interfaceToCSS(asBase(expr), s.version)
		case ast.ContextJavaScript:
			str, ok = interfaceToJavaScript(asBase(expr), s.version)
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

func interfaceToText(expr interface{}, version string) (string, bool) {

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
			str, ok := interfaceToText(rv.Index(i).Interface(), version)
			if !ok {
				return "", false
			}
			buf[i] = str
		}
		s = strings.Join(buf, ", ")
	}

	return s, true
}

func interfaceToHTML(expr interface{}, version string) (string, bool) {

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
			str, ok := interfaceToHTML(rv.Index(i).Interface(), version)
			if !ok {
				return "", false
			}
			buf[i] = str
		}
		s = strings.Join(buf, ", ")
	}

	return s, true
}

func interfaceToTag(expr interface{}, version string) (string, bool) {
	// TODO(marco): return a more explanatory error
	s, ok := interfaceToText(expr, version)
	if !ok {
		return "", false
	}
	i := 0
	for j, c := range s {
		if c == utf8.RuneError && j == i+1 {
			return "", false
		}
		const DEL = 0x7F
		if 0x00 <= c && c <= 0x1F || c == '"' || c == '\'' || c == '>' || c == '/' || c == '=' || c == DEL ||
			0x7F <= c && c <= 0x9F || unicode.Is(unicode.Noncharacter_Code_Point, c) {
			return "", false
		}
		i = j
	}
	return s, true
}

func interfaceToAttribute(expr interface{}, version string, urlstate *urlState) (string, bool) {

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
			str, ok := interfaceToAttribute(rv.Index(i).Interface(), version, urlstate)
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

func interfaceToCSS(expr interface{}, version string) (string, bool) {
	return interfaceToText(expr, version)
}

var mapStringToInterfaceType = reflect.TypeOf(map[string]interface{}{})

func interfaceToJavaScript(expr interface{}, version string) (string, bool) {

	if expr == nil {
		return "null", true
	}

	switch e := expr.(type) {
	case string:
		return stringToJavaScript(e), true
	case HTML:
		return stringToJavaScript(string(e)), true
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
				s2, ok := interfaceToJavaScript(rv.Index(i).Interface(), version)
				if !ok {
					return "", false
				}
				s += s2
			}
			return s + "]", true
		case reflect.Struct:
			return structToJavaScript(rv.Type(), rv, version)
		case reflect.Map:
			if rv.IsNil() {
				return "null", true
			}
			if !rv.Type().ConvertibleTo(mapStringToInterfaceType) {
				return "undefined", false
			}
			return mapToJavaScript(rv.Convert(mapStringToInterfaceType).Interface().(map[string]interface{}), version)
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
			return structToJavaScript(rt, rv, version)
		}
	}

	return "undefined", false
}

const hexchars = "0123456789abcdef"

func stringToJavaScript(s string) string {
	if len(s) == 0 {
		return "\"\""
	}
	var b bytes.Buffer
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString("\\\\")
		case '"':
			b.WriteString("\\\"")
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
			if 0 <= r && r <= 31 || r == '<' || r == '>' || r == '&' {
				b.WriteString("\\x")
				b.WriteByte(hexchars[r>>4])
				b.WriteByte(hexchars[r&0xF])
			} else {
				b.WriteRune(r)
			}
		}
	}
	return "\"" + b.String() + "\""
}

func structToJavaScript(t reflect.Type, v reflect.Value, version string) (string, bool) {
	var s string
	for _, field := range getStructFields(v) {
		if field.version == "" || field.version == version {
			if len(s) > 0 {
				s += ","
			}
			s += stringToJavaScript(field.name) + ":"
			s2, ok := interfaceToJavaScript(v.Field(field.index).Interface(), version)
			if !ok {
				return "undefined", false
			}
			s += s2
		}
	}
	return "{" + s + "}", true
}

func mapToJavaScript(e map[string]interface{}, version string) (string, bool) {
	if e == nil {
		return "null", true
	}
	var s string
	for k, v := range e {
		if len(s) > 0 {
			s += ","
		}
		s += stringToJavaScript(k) + ":"
		s2, ok := interfaceToJavaScript(v, version)
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
