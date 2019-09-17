// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"
	"io"
	"reflect"
	_sort "sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"scriggo/ast"
	"scriggo/internal/compiler"
	"scriggo/runtime"
)

var byteSliceType = reflect.TypeOf([]byte(nil))

type PrintTypeError struct {
	t reflect.Type
}

func (err PrintTypeError) Error() string {
	return fmt.Sprintf("cannot print value of type %s", err.t)
}
func (err PrintTypeError) RuntimeError() {}

// render renders value in the context ctx and writes to out.
func render(_ *runtime.Env, out io.Writer, value interface{}, ctx ast.Context) {

	var err error

	switch ctx {
	case ast.ContextText:
		err = renderInText(out, value)
	case ast.ContextHTML:
		err = renderInHTML(out, value)
	case ast.ContextTag:
		err = renderInTag(out, value)
	case ast.ContextAttribute:
		err = renderInAttribute(out, value, true)
	case ast.ContextUnquotedAttribute:
		err = renderInAttribute(out, value, false)
	case ast.ContextCSS:
		err = renderInCSS(out, value)
	case ast.ContextCSSString:
		err = renderInCSSString(out, value)
	case ast.ContextJavaScript:
		err = renderInJavaScript(out, value)
	case ast.ContextJavaScriptString:
		err = renderInJavaScriptString(out, value)
	default:
		panic("scriggo: unknown context")
	}

	if err != nil {
		panic(err)
	}

	return
}

type strWriterWrapper struct {
	w io.Writer
}

func (wr strWriterWrapper) Write(b []byte) (int, error) {
	return wr.w.Write(b)
}

func (wr strWriterWrapper) WriteString(s string) (int, error) {
	return wr.w.Write([]byte(s))
}

func newStringWriter(wr io.Writer) strWriter {
	if sw, ok := wr.(strWriter); ok {
		return sw
	}
	return strWriterWrapper{wr}
}

func toString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Invalid:
		return ""
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'f', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.String:
		return v.String()
	case reflect.Complex64:
		c := v.Complex()
		if c == 0 {
			return "0"
		}
		var s string
		if r := real(c); r != 0 {
			s = strconv.FormatFloat(r, 'f', -1, 32)
		}
		if i := imag(c); i != 0 {
			if s != "" && i > 0 {
				s += " "
			}
			s = strconv.FormatFloat(i, 'f', -1, 32) + "i"
		}
		return s
	case reflect.Complex128:
		c := v.Complex()
		if c == 0 {
			return "0"
		}
		var s string
		if r := real(c); r != 0 {
			s = strconv.FormatFloat(r, 'f', -1, 64)
		}
		if i := imag(c); i != 0 {
			if s != "" && i > 0 {
				s += "+"
			}
			s = strconv.FormatFloat(i, 'f', -1, 64) + "i"
		}
		return s
	default:
		panic(PrintTypeError{t: v.Type()})
	}
}

// renderInText renders value in the Text context.
func renderInText(out io.Writer, value interface{}) error {
	var s string
	if v, ok := value.(fmt.Stringer); ok {
		s = v.String()
	} else {
		s = toString(reflect.ValueOf(value))
	}
	w := newStringWriter(out)
	_, err := w.WriteString(s)
	return err
}

// renderInHTML renders value in HTML context.
func renderInHTML(out io.Writer, value interface{}) error {
	w := newStringWriter(out)
	switch v := value.(type) {
	case HTML:
		_, err := w.WriteString(string(v))
		return err
	case compiler.HTMLStringer:
		_, err := w.WriteString(string(v.HTML()))
		return err
	case fmt.Stringer:
		return htmlEscape(w, v.String())
	default:
		return htmlEscape(w, toString(reflect.ValueOf(value)))
	}
}

// renderInTag renders value in Tag context.
func renderInTag(out io.Writer, value interface{}) error {
	var s string
	if v, ok := value.(fmt.Stringer); ok {
		s = v.String()
	} else {
		s = toString(reflect.ValueOf(value))
	}
	i := 0
	var s2 *strings.Builder
	for j, c := range s {
		if (c == utf8.RuneError && j == i+1) ||
			c <= 0x1F || c == '"' || c == '\'' || c == '>' || c == '/' || c == '=' ||
			0x7F <= c && c <= 0x9F || unicode.Is(unicode.Noncharacter_Code_Point, c) {
			if s2 == nil {
				s2 = &strings.Builder{}
				s2.WriteString(s[0:j])
			}
			c = unicode.ReplacementChar
		}
		if s2 != nil {
			s2.WriteRune(c)
		}
	}
	if s2 != nil {
		s = s2.String()
	}
	w := newStringWriter(out)
	_, err := w.WriteString(s)
	return err
}

// renderInAttribute renders value in Attribute context quoted or unquoted
// depending on quoted value.
func renderInAttribute(out io.Writer, value interface{}, quoted bool) error {
	var s string
	if v, ok := value.(fmt.Stringer); ok {
		s = v.String()
	} else {
		s = toString(reflect.ValueOf(value))
	}
	return attributeEscape(newStringWriter(out), s, quoted)
}

// renderInCSS renders value in CSS context.
func renderInCSS(out io.Writer, value interface{}) error {
	w := newStringWriter(out)
	switch v := value.(type) {
	case CSS:
		_, err := w.WriteString(string(v))
		return err
	case compiler.CSSStringer:
		_, err := w.WriteString(string(v.CSS()))
		return err
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		_, err := w.WriteString(`"`)
		if err == nil {
			err = cssStringEscape(w, v.String())
		}
		if err == nil {
			_, err = w.WriteString(`"`)
		}
		return err
	case reflect.Slice:
		return escapeBytes(w, v.Interface().([]byte), false)
	default:
		_, err := w.WriteString(toString(v))
		return err
	}
}

// renderInCSSString renders value in CSSString context.
func renderInCSSString(out io.Writer, value interface{}) error {
	var s string
	if v, ok := value.(fmt.Stringer); ok {
		s = v.String()
	} else {
		v := reflect.ValueOf(value)
		if v.Type() == byteSliceType {
			w := newStringWriter(out)
			return escapeBytes(w, v.Interface().([]byte), false)
		}
		s = toString(reflect.ValueOf(value))
	}
	return cssStringEscape(newStringWriter(out), s)
}

// renderInJavaScript renders value in JavaScript context.
func renderInJavaScript(out io.Writer, value interface{}) error {

	w := newStringWriter(out)

	switch v := value.(type) {
	case JavaScript:
		_, err := w.WriteString(string(v))
		return err
	case compiler.JavaScriptStringer:
		_, err := w.WriteString(string(v.JavaScript()))
		return err
	}

	v := reflect.ValueOf(value)

	var s string

	switch v.Kind() {
	case reflect.Invalid:
		s = "null"
	case reflect.Bool:
		s = "false"
		if v.Bool() {
			s = "true"
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		s = strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		s = strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32:
		s = strconv.FormatFloat(v.Float(), 'f', -1, 32)
	case reflect.Float64:
		s = strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.String:
		_, err := w.WriteString("\"")
		if err == nil {
			err = javaScriptStringEscape(w, v.String())
		}
		if err == nil {
			_, err = w.WriteString("\"")
		}
		return err
	case reflect.Slice:
		if v.Type() == byteSliceType {
			w := newStringWriter(out)
			return escapeBytes(w, v.Interface().([]byte), true)
		}
		if v.IsNil() {
			s = "null"
			break
		}
		fallthrough
	case reflect.Array:
		if v.Len() == 0 {
			s = "[]"
			break
		}
		_, err := w.WriteString("[")
		for i := 0; i < v.Len(); i++ {
			if err != nil {
				return err
			}
			if i > 0 {
				_, err = w.WriteString(",")
			}
			if err == nil {
				err = renderInJavaScript(out, v.Index(i).Interface())
			}
		}
		if err == nil {
			_, err = w.WriteString("]")
		}
		return err
	case reflect.Ptr, reflect.UnsafePointer:
		if v.IsNil() {
			s = "null"
			break
		}
		return renderInJavaScript(out, v.Elem().Interface())
	case reflect.Struct:
		t := v.Type()
		n := t.NumField()
		first := true
		_, err := w.WriteString(`{`)
		for i := 0; i < n; i++ {
			if err != nil {
				return err
			}
			if field := t.Field(i); field.PkgPath == "" {
				if first {
					_, err = w.WriteString(`"`)
				} else {
					_, err = w.WriteString(`,"`)
				}
				if err == nil {
					err = javaScriptStringEscape(w, field.Name)
				}
				if err == nil {
					_, err = w.WriteString(`":`)
				}
				if err == nil {
					err = renderInJavaScript(w, v.Field(i).Interface())
				}
				first = false
			}
		}
		if err == nil {
			_, err = w.WriteString(`}`)
		}
		return err
	case reflect.Map:
		if v.IsNil() {
			s = "null"
			break
		}
		type keyPair struct {
			key string
			val interface{}
		}
		keyPairs := make([]keyPair, v.Len())
		iter := v.MapRange()
		for i := 0; iter.Next(); i++ {
			key := iter.Key()
			if k, ok := key.Interface().(fmt.Stringer); ok {
				keyPairs[i].key = k.String()
			} else {
				keyPairs[i].key = toString(key)
			}
			keyPairs[i].val = iter.Value().Interface()
		}
		_sort.Slice(keyPairs, func(i, j int) bool {
			return keyPairs[i].key < keyPairs[i].key
		})
		_, err := w.WriteString("{")
		for i, keyPair := range keyPairs {
			if err != nil {
				return err
			}
			if i > 0 {
				_, err = w.WriteString(",")
			}
			if err == nil {
				err = javaScriptStringEscape(w, keyPair.key)
			}
			if err == nil {
				_, err = w.WriteString(":")
			}
			if err == nil {
				err = renderInJavaScript(out, keyPair.val)
			}
		}
		if err == nil {
			_, err = w.WriteString("}")
		}
		return err
	default:
		s = fmt.Sprintf("undefined/* scriggo: cannot represent a %s value */", v.Type())
	}

	_, err := w.WriteString(s)
	return err
}

// renderInJavaScriptString renders value in JavaScriptString context.
func renderInJavaScriptString(out io.Writer, value interface{}) error {
	var s string
	if v, ok := value.(fmt.Stringer); ok {
		s = v.String()
	} else {
		s = toString(reflect.ValueOf(value))
	}
	return javaScriptStringEscape(newStringWriter(out), s)
}
