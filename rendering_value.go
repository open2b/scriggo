// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

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

// ValueRenderer can be implemented by the values of variables. When a value
// have to be rendered, if the value implements ValueRender, its Render method
// is called.
//
// The method Render on a value is called only if the context in which the
// statement is rendered is the same context passed as argument to the
// renderer function or method.
//
// For example if this source is parsed in context HTML:
//
//  <a href="{{ expr1 }}">{{ expr2 }}</a>
//  <script>var b = {{ expr3 }};</script>
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
//  // {{ going + " " + where }} is rendered as:
//  // <a href="/">going</a> &gt;&gt; here &amp; there
//
type HTML string

// Stringer is implemented by any value that behaves like a string.
type Stringer interface {
	String() string
}

// Numberer is implemented by any value that behaves like a number.
type Numberer interface {
	Number() decimal.Decimal
}

func (s HTML) Render(w io.Writer) (int, error) {
	return io.WriteString(w, string(s))
}

type stringWriter interface {
	Write(b []byte) (int, error)
	WriteString(s string) (int, error)
}

type stringWriterWrapper struct {
	w io.Writer
}

func (wr stringWriterWrapper) Write(b []byte) (int, error) {
	return wr.w.Write(b)
}

func (wr stringWriterWrapper) WriteString(s string) (int, error) {
	return wr.w.Write([]byte(s))
}

func newStringWriter(wr io.Writer) stringWriter {
	if sw, ok := wr.(stringWriter); ok {
		return sw
	}
	return stringWriterWrapper{wr}
}

// renderValue renders value in the context of node and as a URL is urlstate
// is not nil.
func (r *rendering) renderValue(wr io.Writer, value interface{}, node *ast.Value, urlstate *urlState) error {

	if e, ok := value.(ValueRenderer); ok && node.Context == r.treeContext {

		err := func() (err error) {
			defer func() {
				if e := recover(); e != nil {
					err = r.errorf(node, "%s", e)
				}
			}()
			_, err = e.Render(wr)
			return err
		}()
		if err != nil {
			return err
		}

	} else {

		w := newStringWriter(wr)

		var err error

		switch node.Context {
		case ast.ContextText:
			err = r.renderInText(w, asBase(value), node)
		case ast.ContextHTML:
			err = r.renderInHTML(w, asBase(value), node)
		case ast.ContextTag:
			err = r.renderInTag(w, asBase(value), node)
		case ast.ContextAttribute:
			if urlstate == nil {
				err = r.renderInAttribute(w, asBase(value), node, false)
			} else {
				err = r.renderInAttributeURL(w, asBase(value), node, urlstate, false)
			}
		case ast.ContextUnquotedAttribute:
			if urlstate == nil {
				err = r.renderInAttribute(w, asBase(value), node, true)
			} else {
				err = r.renderInAttributeURL(w, asBase(value), node, urlstate, true)
			}
		case ast.ContextCSS:
			err = r.renderInCSS(w, asBase(value), node)
		case ast.ContextCSSString:
			err = r.renderInCSSString(w, asBase(value), node)
		case ast.ContextScript:
			err = r.renderInScript(w, asBase(value), node)
		case ast.ContextScriptString:
			err = r.renderInScriptString(w, asBase(value), node)
		default:
			panic("template: unknown context")
		}
		if err != nil && !r.handleError(err) {
			return err
		}

	}

	return nil
}

// renderInText renders value in the Text context.
func (r *rendering) renderInText(w stringWriter, value interface{}, node *ast.Value) error {

	if value == nil {
		return nil
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
			return r.errorf(node, "no-render type %s", typeof(value))
		}
		if rv.IsNil() || rv.Len() == 0 {
			return nil
		}
		for i, l := 0, rv.Len(); i < l; i++ {
			if i > 0 {
				_, err := w.WriteString(", ")
				if err != nil {
					return err
				}
			}
			err := r.renderInText(w, rv.Index(i).Interface(), node)
			if err != nil {
				return err
			}
		}
		return nil
	}

	_, err := w.WriteString(s)

	return err
}

// renderInHTML renders value in HTML context.
func (r *rendering) renderInHTML(w stringWriter, value interface{}, node *ast.Value) error {

	if value == nil {
		return nil
	}

	var s string

	switch e := value.(type) {
	case string:
		return htmlEscape(w, e)
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
			return r.errorf(node, "no-render type %s", typeof(value))
		}
		if rv.IsNil() || rv.Len() == 0 {
			return nil
		}
		for i, l := 0, rv.Len(); i < l; i++ {
			if i > 0 {
				_, err := w.WriteString(", ")
				if err != nil {
					return err
				}
			}
			err := r.renderInHTML(w, rv.Index(i).Interface(), node)
			if err != nil {
				return err
			}
		}
		return nil
	}

	_, err := w.WriteString(s)

	return err
}

// renderInTag renders value in Tag context.
func (r *rendering) renderInTag(w stringWriter, value interface{}, node *ast.Value) error {
	buf := strings.Builder{}
	err := r.renderInText(&buf, value, node)
	if err != nil {
		return err
	}
	s := buf.String()
	i := 0
	for j, c := range s {
		if c == utf8.RuneError && j == i+1 {
			return r.errorf(node, "not valid unicode in tag context")
		}
		const DEL = 0x7F
		if c <= 0x1F || c == '"' || c == '\'' || c == '>' || c == '/' || c == '=' || c == DEL ||
			0x7F <= c && c <= 0x9F || unicode.Is(unicode.Noncharacter_Code_Point, c) {
			return r.errorf(node, "not allowed character %s in tag context", string(c))
		}
	}
	_, err = w.WriteString(s)
	return err
}

// renderInAttribute renders value in Attribute context quoted or unquoted
// depending on unquoted value.
func (r *rendering) renderInAttribute(w stringWriter, value interface{}, node *ast.Value, unquoted bool) error {

	if value == nil {
		return nil
	}

	var s string

	switch e := value.(type) {
	case string:
		return attributeEscape(w, e, unquoted)
	case HTML:
		return attributeEscape(w, html.UnescapeString(string(e)), unquoted)
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
			return r.errorf(node, "no-render type %s", typeof(value))
		}
		if rv.IsNil() || rv.Len() == 0 {
			return nil
		}
		for i, l := 0, rv.Len(); i < l; i++ {
			var err error
			if i > 0 {
				if unquoted {
					_, err = w.WriteString(",&#32;")
				} else {
					_, err = w.WriteString(", ")
				}
			}
			err = r.renderInAttribute(w, rv.Index(i).Interface(), node, unquoted)
			if err != nil {
				return err
			}
		}
	}

	_, err := w.WriteString(s)

	return err
}

// renderInAttributeURL renders value as an URL in Attribute context quoted or
// unquoted depending on unquoted value.
func (r *rendering) renderInAttributeURL(w stringWriter, value interface{}, node *ast.Value, urlstate *urlState, unquoted bool) error {

	if value == nil {
		return nil
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
			return r.errorf(node, "no-render type %s", typeof(value))
		}
		if rv.IsNil() || rv.Len() == 0 {
			return nil
		}
		for i, l := 0, rv.Len(); i < l; i++ {
			if i > 0 {
				if unquoted {
					s += ",&#32;"
				} else {
					s += ", "
				}
			}
			switch e := rv.Index(i).Interface().(type) {
			case string:
				s += e
			case HTML:
				s += string(e)
			case int:
				s += strconv.Itoa(e)
			case decimal.Decimal:
				s += e.String()
			case bool:
				if e {
					s += "true"
				} else {
					s += "false"
				}
			default:
				return r.errorf(node, "no-render type %s", typeof(value))
			}
		}
	}

	if s == "" {
		return nil
	}

	if urlstate.path {
		if strings.Contains(s, "?") {
			urlstate.path = false
			urlstate.addAmp = s[len(s)-1] != '?' && s[len(s)-1] != '&'
		}
		return pathEscape(w, s, unquoted)
	} else if urlstate.query {
		return queryEscape(w, s)
	}

	// TODO(marco): return the correct error
	return r.errorf(node, "no-render type %s", typeof(value))
}

// renderInCSS renders value in CSS context.
func (r *rendering) renderInCSS(w stringWriter, value interface{}, node *ast.Value) error {

	if value == nil {
		return r.errorf(node, "no-render type %s", typeof(value))
	}

	var s string

	switch e := value.(type) {
	case string:
		_, err := w.WriteString(`"`)
		if err != nil {
			return err
		}
		err = cssStringEscape(w, e)
		if err != nil {
			return err
		}
		_, err = w.WriteString(`"`)
		return err
	case HTML:
		_, err := w.WriteString(`"`)
		if err != nil {
			return err
		}
		err = cssStringEscape(w, string(e))
		if err != nil {
			return err
		}
		_, err = w.WriteString(`"`)
		return err
	case int:
		s = strconv.Itoa(e)
	case decimal.Decimal:
		s = e.String()
	default:
		return r.errorf(node, "no-render type %s", typeof(value))
	}

	_, err := w.WriteString(s)

	return err
}

// renderInCSSString renders value in CSSString context.
func (r *rendering) renderInCSSString(w stringWriter, value interface{}, node *ast.Value) error {

	if value == nil {
		return r.errorf(node, "no-render type %s", typeof(value))
	}

	var s string

	switch e := value.(type) {
	case string:
		return cssStringEscape(w, e)
	case HTML:
		return cssStringEscape(w, string(e))
	case int:
		s = strconv.Itoa(e)
	case decimal.Decimal:
		s = e.String()
	default:
		return r.errorf(node, "no-render type %s", typeof(value))
	}

	_, err := w.WriteString(s)

	return err
}

var mapStringToInterfaceType = reflect.TypeOf(map[string]interface{}{})

// renderInScript renders value in Script context.
func (r *rendering) renderInScript(w stringWriter, value interface{}, node *ast.Value) error {

	if value == nil {
		_, err := w.WriteString("null")
		return err
	}

	switch e := value.(type) {
	case string:
		_, err := w.WriteString("\"")
		if err != nil {
			return err
		}
		err = scriptStringEscape(w, e)
		if err != nil {
			return err
		}
		_, err = w.WriteString("\"")
		return err
	case HTML:
		_, err := w.WriteString("\"")
		if err != nil {
			return err
		}
		err = scriptStringEscape(w, string(e))
		if err != nil {
			return err
		}
		_, err = w.WriteString("\"")
		return err
	case int:
		_, err := w.WriteString(strconv.Itoa(e))
		return err
	case decimal.Decimal:
		_, err := w.WriteString(e.String())
		return err
	case bool:
		s := "false"
		if e {
			s = "true"
		}
		_, err := w.WriteString(s)
		return err
	default:
		var err error
		rv := reflect.ValueOf(value)
		if !rv.IsValid() {
			_, err = w.WriteString("undefined")
			if err != nil {
				return err
			}
			return r.errorf(node, "no-render type %s", typeof(value))
		}
		switch rv.Kind() {
		case reflect.Slice:
			if rv.IsNil() {
				_, err = w.WriteString("null")
				return err
			}
			if rv.Len() == 0 {
				_, err = w.WriteString("[]")
				return err
			}
			_, err = w.WriteString("[")
			if err != nil {
				return err
			}
			for i := 0; i < rv.Len(); i++ {
				if i > 0 {
					_, err = w.WriteString(",")
					if err != nil {
						return err
					}
				}
				err = r.renderInScript(w, rv.Index(i).Interface(), node)
				if err != nil {
					return err
				}
			}
			_, err = w.WriteString("]")
			return err
		case reflect.Struct:
			return r.renderValueAsScriptObject(w, rv, node)
		case reflect.Map:
			if rv.IsNil() {
				_, err = w.WriteString("null")
				return err
			}
			if !rv.Type().ConvertibleTo(mapStringToInterfaceType) {
				_, err = w.WriteString("undefined")
				if err != nil {
					return err
				}
				return r.errorf(node, "no-render type %s", typeof(value))
			}
			return r.renderMapAsScriptObject(w, rv.Convert(mapStringToInterfaceType).Interface().(map[string]interface{}), node)
		case reflect.Ptr:
			if rv.IsNil() {
				_, err = w.WriteString("null")
				return err
			}
			rt := rv.Type().Elem()
			if rt.Kind() != reflect.Struct {
				_, err = w.WriteString("undefined")
				if err != nil {
					return err
				}
				return r.errorf(node, "no-render type %s", typeof(value))
			}
			rv = rv.Elem()
			if !rv.IsValid() {
				_, err = w.WriteString("undefined")
				if err != nil {
					return err
				}
				return r.errorf(node, "no-render type %s", typeof(value))
			}
			return r.renderValueAsScriptObject(w, rv, node)
		}
	}

	_, err := w.WriteString("undefined")
	if err != nil {
		return err
	}
	return r.errorf(node, "no-render type %s", typeof(value))
}

// renderInScriptString renders value in ScriptString context.
func (r *rendering) renderInScriptString(w stringWriter, value interface{}, node *ast.Value) error {

	if value == nil {
		return r.errorf(node, "no-render type %s", typeof(value))
	}

	switch e := value.(type) {
	case string:
		err := scriptStringEscape(w, e)
		return err
	case HTML:
		err := scriptStringEscape(w, string(e))
		return err
	case int:
		_, err := w.WriteString(strconv.Itoa(e))
		return err
	case decimal.Decimal:
		_, err := w.WriteString(e.String())
		return err
	}

	return r.errorf(node, "no-render type %s", typeof(value))
}

// renderValueAsScriptObject returns value as a JavaScript object or undefined
// if it is not possible.
func (r *rendering) renderValueAsScriptObject(w stringWriter, value reflect.Value, node *ast.Value) error {

	var err error

	fields := getStructFields(value)

	if len(fields.names) == 0 {
		_, err = w.WriteString(`{}`)
		return err
	}

	for i, name := range fields.names {
		sep := `,"`
		if i == 0 {
			sep = `{"`
		}
		_, err = w.WriteString(sep)
		if err != nil {
			return err
		}
		err = scriptStringEscape(w, name)
		if err != nil {
			return err
		}
		_, err = w.WriteString(`":`)
		if err != nil {
			return err
		}
		err = r.renderInScript(w, value.Field(fields.indexOf[name]).Interface(), node)
		if err != nil {
			return err
		}
	}

	_, err = w.WriteString(`}`)

	return err
}

// renderMapAsScriptObject returns value as a JavaScript object or undefined
// if it is not possible.
func (r *rendering) renderMapAsScriptObject(w stringWriter, value map[string]interface{}, node *ast.Value) error {

	if value == nil {
		_, err := w.WriteString("null")
		return err
	}

	_, err := w.WriteString("{")
	if err != nil {
		return err
	}

	names := make([]string, 0, len(value))
	for name := range value {
		names = append(names, name)
	}
	sort.Strings(names)

	for i, n := range names {
		if i > 0 {
			_, err = w.WriteString(",")
			if err != nil {
				return err
			}
		}
		err = scriptStringEscape(w, n)
		if err != nil {
			return err
		}
		_, err = w.WriteString(":")
		if err != nil {
			return err
		}
		err = r.renderInScript(w, value[n], node)
		if err != nil {
			return err
		}
	}

	_, err = w.WriteString("}")

	return err
}
