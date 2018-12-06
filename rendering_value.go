// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"html"
	"io"
	"open2b/template/ast"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/shopspring/decimal"
)

// ValueRenderer can be implemented by the values of variables. When a value
// have to be rendered, if the value implements ValueRender its Render method
// is called.
//
// The method Render on a value is called only if the context in which the
// statement is rendered is the same context passed as argument to the
// renderer function or method.
//
// For example if this source is parsed in context HTML:
//
//  <a href="{{ expr }}">{{ expr }}</a>
//  <script>var b = {{ expr }};</script>
//
// the first statement has context Attribute, the second has context HTML and
// the third has context Script, so Render would only be called on the second
// statement.
//
// Render returns the number of bytes written to out and any error encountered
// during the write.
type ValueRenderer interface {
	Render(out io.Writer) (n int, err error)
}

// HTML is a ValueRenderer that encapsulates a string containing an HTML code
// that have to be rendered without escape.
//
// HTML values are safe to use in concatenation. An HTML value concatenated
// with a string become an HTML value with only the string escaped.
//
//  // For example, defining the variables "going" and "where" as:
//
//  vars := map[string]interface{}{
//      "going": renderer.HTML("<a href="/">going</a>"),
//      "where": ">> here & there",
//  }
//
//  // {{ going + " " + where }} is rendered as: <a href="/">going</a> &gt;&gt; here &amp; there
//
type HTML string

// Stringer is implemented by any value that behaves like a string.
type Stringer interface {
	String() string
}

// Numberer is implemented by any variable value that behaves like a number.
type Numberer interface {
	Number() decimal.Decimal
}

func (s HTML) Render(w io.Writer) (int, error) {
	return io.WriteString(w, string(s))
}

// renderValue renders value in the context of node and as a URL is urlstate
// is not nil.
func (s *rendering) renderValue(wr io.Writer, value interface{}, node *ast.Value, urlstate *urlState) error {

	if e, ok := value.(ValueRenderer); ok && node.Context == s.treeContext {

		err := func() (err error) {
			defer func() {
				if e := recover(); e != nil {
					err = s.errorf(node, "%s", e)
				}
			}()
			_, err = e.Render(wr)
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
			str, ok = renderInText(asBase(value))
		case ast.ContextHTML:
			str, ok = renderInHTML(asBase(value))
		case ast.ContextTag:
			str, ok = renderInTag(asBase(value))
		case ast.ContextAttribute:
			str, ok = renderInAttribute(asBase(value), urlstate, false)
		case ast.ContextUnquotedAttribute:
			str, ok = renderInAttribute(asBase(value), urlstate, true)
		case ast.ContextCSS:
			str, ok = renderInCSS(asBase(value))
		case ast.ContextCSSString:
			str, ok = renderInCSSString(asBase(value))
		case ast.ContextScript:
			str, ok = renderInScript(asBase(value))
		case ast.ContextScriptString:
			str, ok = renderInScriptString(asBase(value))
		default:
			panic("template: unknown context")
		}
		if !ok {
			err := s.errorf(node.Expr, "wrong type %s in value", typeof(value))
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

// renderInText renders value in the Text context.
func renderInText(value interface{}) (string, bool) {

	if value == nil {
		return "", true
	}

	var s string

	switch e := value.(type) {
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
		rv := reflect.ValueOf(value)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return "", false
		}
		if rv.IsNil() || rv.Len() == 0 {
			return "", true
		}
		buf := make([]string, rv.Len())
		for i := 0; i < len(buf); i++ {
			str, ok := renderInText(rv.Index(i).Interface())
			if !ok {
				return "", false
			}
			buf[i] = str
		}
		s = strings.Join(buf, ", ")
	}

	return s, true
}

// renderInHTML renders value in HTML context.
func renderInHTML(value interface{}) (string, bool) {

	if value == nil {
		return "", true
	}

	var s string

	switch e := value.(type) {
	case string:
		s = htmlEscape(e)
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
		rv := reflect.ValueOf(value)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return "", false
		}
		if rv.IsNil() || rv.Len() == 0 {
			return "", true
		}
		buf := make([]string, rv.Len())
		for i := 0; i < len(buf); i++ {
			str, ok := renderInHTML(rv.Index(i).Interface())
			if !ok {
				return "", false
			}
			buf[i] = str
		}
		s = strings.Join(buf, ", ")
	}

	return s, true
}

// renderInTag renders value in Tag context.
func renderInTag(value interface{}) (string, bool) {
	// TODO(marco): return a more explanatory error
	s, ok := renderInText(value)
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

// renderInAttribute renders value in Attribute context, as URL is urlstate is
// not nil and quoted or unquoted depending on unquoted value.
func renderInAttribute(value interface{}, urlstate *urlState, unquoted bool) (string, bool) {

	if value == nil {
		return "", true
	}

	var s string

	switch e := value.(type) {
	case string:
		s = e
		if urlstate == nil {
			s = attributeEscape(e, unquoted)
		}
	case HTML:
		s = string(e)
		if urlstate == nil {
			s = attributeEscape(html.UnescapeString(string(e)), unquoted)
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
		rv := reflect.ValueOf(value)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return "", false
		}
		if rv.IsNil() || rv.Len() == 0 {
			return "", true
		}
		buf := make([]string, rv.Len())
		for i := 0; i < len(buf); i++ {
			str, ok := renderInAttribute(rv.Index(i).Interface(), urlstate, unquoted)
			if !ok {
				return "", false
			}
			buf[i] = str
		}
		if unquoted {
			s = strings.Join(buf, ",&#32;")
		} else {
			s = strings.Join(buf, ", ")
		}
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
			s = pathEscape(s, unquoted)
		case urlstate.query:
			s = queryEscape(s)
		default:
			return "", false
		}
	}

	return s, true
}

// renderInCSS renders value in CSS context.
func renderInCSS(value interface{}) (string, bool) {

	if value == nil {
		return "", false
	}

	switch e := value.(type) {
	case string:
		return "\"" + cssStringEscape(e) + "\"", true
	case HTML:
		return "\"" + cssStringEscape(string(e)) + "\"", true
	case int:
		return strconv.Itoa(e), true
	case decimal.Decimal:
		return e.String(), true
	}

	return "", false
}

// renderInCSSString renders value in CSSString context.
func renderInCSSString(value interface{}) (string, bool) {

	if value == nil {
		return "", false
	}

	switch e := value.(type) {
	case string:
		return cssStringEscape(e), true
	case HTML:
		return cssStringEscape(string(e)), true
	case int:
		return strconv.Itoa(e), true
	case decimal.Decimal:
		return e.String(), true
	}

	return "", false
}

var mapStringToInterfaceType = reflect.TypeOf(map[string]interface{}{})

// renderInScript renders value in Script context.
func renderInScript(value interface{}) (string, bool) {

	if value == nil {
		return "null", true
	}

	switch e := value.(type) {
	case string:
		return "\"" + scriptStringEscape(e) + "\"", true
	case HTML:
		return "\"" + scriptStringEscape(string(e)) + "\"", true
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
		rv := reflect.ValueOf(value)
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
				s2, ok := renderInScript(rv.Index(i).Interface())
				if !ok {
					return "", false
				}
				s += s2
			}
			return s + "]", true
		case reflect.Struct:
			return renderValueAsScriptObject(rv)
		case reflect.Map:
			if rv.IsNil() {
				return "null", true
			}
			if !rv.Type().ConvertibleTo(mapStringToInterfaceType) {
				return "undefined", false
			}
			return renderMapAsScriptObject(rv.Convert(mapStringToInterfaceType).Interface().(map[string]interface{}))
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
			return renderValueAsScriptObject(rv)
		}
	}

	return "undefined", false
}

// renderInScriptString renders value in ScriptString context.
func renderInScriptString(value interface{}) (string, bool) {

	if value == nil {
		return "", false
	}

	switch e := value.(type) {
	case string:
		return scriptStringEscape(e), true
	case HTML:
		return scriptStringEscape(string(e)), true
	case int:
		return strconv.Itoa(e), true
	case decimal.Decimal:
		return e.String(), true
	}

	return "", false
}

// renderValueAsScriptObject returns value as a JavaScript object or undefined
// if it is not possible.
func renderValueAsScriptObject(value reflect.Value) (string, bool) {
	var s string
	fields := getStructFields(value)
	for _, name := range fields.names {
		if len(s) > 0 {
			s += ","
		}
		s += "\"" + scriptStringEscape(name) + "\":"
		s2, ok := renderInScript(value.Field(fields.indexOf[name]).Interface())
		if !ok {
			return "undefined", false
		}
		s += s2
	}
	return "{" + s + "}", true
}

// renderMapAsScriptObject returns value as a JavaScript object or undefined
// if it is not possible.
func renderMapAsScriptObject(e map[string]interface{}) (string, bool) {
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
		s += scriptStringEscape(n) + ":"
		s2, ok := renderInScript(e[n])
		if !ok {
			return "undefined", false
		}
		s += s2
	}
	return "{" + s + "}", true
}
