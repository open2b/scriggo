// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package templates

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/open2b/scriggo/compiler"
	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/runtime"
)

var byteSliceType = reflect.TypeOf([]byte(nil))

type ShowTypeError struct {
	t reflect.Type
}

func (err ShowTypeError) Error() string {
	return fmt.Sprintf("cannot show value of type %s", err.t)
}
func (err ShowTypeError) RuntimeError() {}

// show shows value in the context ctx and writes to out. If out is nil, it
// does not show the value but only checks that the type of value can be
// shown.
//
// show has type scriggo/compiler.ShowFunc.
func show(env runtime.Env, out io.Writer, value interface{}, ctx ast.Context) {

	var err error

	if out == nil {
		err = shownAs(env.Types().TypeOf(value), ctx)
		if err != nil {
			panic(compiler.ShowTypeError(err))
		}
		return
	}

	switch ctx {
	case ast.ContextText:
		err = showInText(env, out, value)
	case ast.ContextHTML:
		err = showInHTML(env, out, value)
	case ast.ContextTag:
		err = showInTag(env, out, value)
	case ast.ContextQuotedAttr:
		err = showInAttribute(env, out, value, true)
	case ast.ContextUnquotedAttr:
		err = showInAttribute(env, out, value, false)
	case ast.ContextCSS:
		err = showInCSS(env, out, value)
	case ast.ContextCSSString:
		err = showInCSSString(env, out, value)
	case ast.ContextJS:
		err = showInJS(env, out, value)
	case ast.ContextJSString:
		err = showInJSString(env, out, value)
	case ast.ContextJSON:
		err = showInJSON(env, out, value)
	case ast.ContextJSONString:
		err = showInJSONString(env, out, value)
	case ast.ContextMarkdown:
		err = showInMarkdown(env, out, value)
	case ast.ContextTabCodeBlock:
		err = showInMarkdownCodeBlock(env, out, value, false)
	case ast.ContextSpacesCodeBlock:
		err = showInMarkdownCodeBlock(env, out, value, true)
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

func toString(env runtime.Env, i interface{}) string {
	types := env.Types()
	v := types.ValueOf(i)
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
		panic(ShowTypeError{t: types.TypeOf(i)})
	}
}

// showInText shows value in the Text context.
func showInText(env runtime.Env, out io.Writer, value interface{}) error {
	var s string
	switch v := value.(type) {
	case fmt.Stringer:
		s = v.String()
	case EnvStringer:
		s = v.String(env)
	case error:
		s = v.Error()
	default:
		s = toString(env, value)
	}
	w := newStringWriter(out)
	_, err := w.WriteString(s)
	return err
}

// showInHTML shows value in HTML context.
func showInHTML(env runtime.Env, out io.Writer, value interface{}) error {
	w := newStringWriter(out)
	switch v := value.(type) {
	case HTML:
		_, err := w.WriteString(string(v))
		return err
	case HTMLStringer:
		_, err := w.WriteString(v.HTML())
		return err
	case HTMLEnvStringer:
		_, err := w.WriteString(v.HTML(env))
		return err
	case fmt.Stringer:
		return htmlEscape(w, v.String())
	case EnvStringer:
		return htmlEscape(w, v.String(env))
	case []byte:
		_, err := out.Write(v)
		return err
	case error:
		return htmlEscape(w, v.Error())
	default:
		return htmlEscape(w, toString(env, value))
	}
}

// showInTag show value in Tag context.
func showInTag(env runtime.Env, out io.Writer, value interface{}) error {
	var s string
	switch v := value.(type) {
	case fmt.Stringer:
		s = v.String()
	case EnvStringer:
		s = v.String(env)
	case error:
		s = v.Error()
	default:
		s = toString(env, value)
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

// showInAttribute shows value in the attribute context quoted or unquoted
// depending on quoted value.
func showInAttribute(env runtime.Env, out io.Writer, value interface{}, quoted bool) error {
	var s string
	var escapeEntities bool
	switch v := value.(type) {
	case fmt.Stringer:
		s = v.String()
		escapeEntities = true
	case EnvStringer:
		s = v.String(env)
		escapeEntities = true
	case HTML:
		s = string(v)
	case HTMLStringer:
		s = v.HTML()
	case HTMLEnvStringer:
		s = v.HTML(env)
	case error:
		s = v.Error()
		escapeEntities = true
	default:
		s = toString(env, value)
		escapeEntities = true
	}
	return attributeEscape(newStringWriter(out), s, escapeEntities, quoted)
}

// showInCSS shows value in CSS context.
func showInCSS(env runtime.Env, out io.Writer, value interface{}) error {
	w := newStringWriter(out)
	switch v := value.(type) {
	case CSS:
		_, err := w.WriteString(string(v))
		return err
	case CSSStringer:
		_, err := w.WriteString(v.CSS())
		return err
	case CSSEnvStringer:
		_, err := w.WriteString(v.CSS(env))
		return err
	case fmt.Stringer:
		value = v.String()
	case EnvStringer:
		value = v.String(env)
	case []byte:
		return escapeBytes(w, v, false)
	case error:
		value = v.Error()
	}
	v := env.Types().ValueOf(value)
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
	default:
		_, err := w.WriteString(toString(env, value))
		return err
	}
}

// showInCSSString shows value in CSSString context.
func showInCSSString(env runtime.Env, out io.Writer, value interface{}) error {
	var s string
	switch value := value.(type) {
	case fmt.Stringer:
		s = value.String()
	case EnvStringer:
		s = value.String(env)
	case error:
		s = value.Error()
	default:
		v := reflect.ValueOf(value)
		if v.Type() == byteSliceType {
			w := newStringWriter(out)
			return escapeBytes(w, v.Interface().([]byte), false)
		}
		s = toString(env, value)
	}
	return cssStringEscape(newStringWriter(out), s)
}

// showInJS shows value in JavaScript context.
func showInJS(env runtime.Env, out io.Writer, value interface{}) error {

	w := newStringWriter(out)

	switch v := value.(type) {
	case nil:
		_, err := w.WriteString("null")
		return err
	case JS:
		_, err := w.WriteString(string(v))
		return err
	case JSStringer:
		_, err := w.WriteString(v.JS())
		return err
	case JSEnvStringer:
		_, err := w.WriteString(v.JS(env))
		return err
	case time.Time:
		_, err := w.WriteString(showTimeInJS(v))
		return err
	case error:
		value = v.Error()
	}

	types := env.Types()
	v := types.ValueOf(value)

	var s string

	switch v.Kind() {
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
			err = jsStringEscape(w, v.String())
		}
		if err == nil {
			_, err = w.WriteString("\"")
		}
		return err
	case reflect.Slice:
		if b, ok := value.([]byte); ok {
			w := newStringWriter(out)
			return escapeBytes(w, b, true)
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
				err = showInJS(env, out, v.Index(i).Interface())
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
		return showInJS(env, out, v.Elem().Interface())
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
				name := field.Name
				value := v.Field(i)
				if tag := field.Tag.Get("json"); tag != "" {
					if tag == "-" {
						continue
					}
					tagName, tagOptions := parseTag(tag)
					if tagOptions.Contains("omitempty") && isEmptyJSONValue(value) {
						continue
					}
					if tagName != "" {
						name = tagName
					}
				}
				if first {
					_, err = w.WriteString(`"`)
				} else {
					_, err = w.WriteString(`,"`)
				}
				if err == nil {
					err = jsStringEscape(w, name)
				}
				if err == nil {
					_, err = w.WriteString(`":`)
				}
				if err == nil {
					err = showInJS(env, w, value.Interface())
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
			switch k := key.Interface().(type) {
			case fmt.Stringer:
				keyPairs[i].key = k.String()
			case EnvStringer:
				keyPairs[i].key = k.String(env)
			default:
				keyPairs[i].key = toString(env, k)
			}
			keyPairs[i].val = iter.Value().Interface()
		}
		sort.Slice(keyPairs, func(i, j int) bool {
			return keyPairs[i].key < keyPairs[j].key
		})
		_, err := w.WriteString("{")
		for i, keyPair := range keyPairs {
			if err != nil {
				return err
			}
			if i == 0 {
				_, err = w.WriteString(`"`)
			} else {
				_, err = w.WriteString(`,"`)
			}
			if err == nil {
				err = jsStringEscape(w, keyPair.key)
			}
			if err == nil {
				_, err = w.WriteString(`":`)
			}
			if err == nil {
				err = showInJS(env, out, keyPair.val)
			}
		}
		if err == nil {
			_, err = w.WriteString("}")
		}
		return err
	default:
		t := types.TypeOf(value)
		s = fmt.Sprintf("undefined/* scriggo: cannot represent a %s value */", t)
	}

	_, err := w.WriteString(s)
	return err
}

// showInJSON shows value in JSON context.
func showInJSON(env runtime.Env, out io.Writer, value interface{}) error {

	w := newStringWriter(out)

	switch v := value.(type) {
	case nil:
		_, err := w.WriteString("null")
		return err
	case JSON:
		_, err := w.WriteString(string(v))
		return err
	case JSONStringer:
		_, err := w.WriteString(v.JSON())
		return err
	case JSONEnvStringer:
		_, err := w.WriteString(v.JSON(env))
		return err
	case time.Time:
		_, err := w.WriteString("\"")
		if err == nil {
			_, err = w.WriteString(v.Format(time.RFC3339))
		}
		if err == nil {
			_, err = w.WriteString("\"")
		}
		return err
	case error:
		value = v.Error()
	}

	v := reflect.ValueOf(value)

	var s string

	switch v.Kind() {
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
			err = jsonStringEscape(w, v.String())
		}
		if err == nil {
			_, err = w.WriteString("\"")
		}
		return err
	case reflect.Slice:
		if b, ok := value.([]byte); ok {
			w := newStringWriter(out)
			return escapeBytes(w, b, true)
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
				err = showInJSON(env, out, v.Index(i).Interface())
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
		return showInJSON(env, out, v.Elem().Interface())
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
				name := field.Name
				value := v.Field(i)
				if tag := field.Tag.Get("json"); tag != "" {
					if tag == "-" {
						continue
					}
					tagName, tagOptions := parseTag(tag)
					if tagOptions.Contains("omitempty") && isEmptyJSONValue(value) {
						continue
					}
					if tagName != "" {
						name = tagName
					}
				}
				if first {
					_, err = w.WriteString(`"`)
				} else {
					_, err = w.WriteString(`,"`)
				}
				if err == nil {
					err = jsonStringEscape(w, name)
				}
				if err == nil {
					_, err = w.WriteString(`":`)
				}
				if err == nil {
					err = showInJSON(env, w, value.Interface())
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
			switch k := key.Interface().(type) {
			case fmt.Stringer:
				keyPairs[i].key = k.String()
			case EnvStringer:
				keyPairs[i].key = k.String(env)
			default:
				keyPairs[i].key = toString(env, k)
			}
			keyPairs[i].val = iter.Value().Interface()
		}
		sort.Slice(keyPairs, func(i, j int) bool {
			return keyPairs[i].key < keyPairs[j].key
		})
		_, err := w.WriteString("{")
		for i, keyPair := range keyPairs {
			if err != nil {
				return err
			}
			if i == 0 {
				_, err = w.WriteString(`"`)
			} else {
				_, err = w.WriteString(`,"`)
			}
			if err == nil {
				err = jsonStringEscape(w, keyPair.key)
			}
			if err == nil {
				_, err = w.WriteString(`":`)
			}
			if err == nil {
				err = showInJSON(env, out, keyPair.val)
			}
		}
		if err == nil {
			_, err = w.WriteString("}")
		}
		return err
	default:
		s = "null"
	}

	_, err := w.WriteString(s)
	return err
}

// showInJSString shows value in JSString context.
func showInJSString(env runtime.Env, out io.Writer, value interface{}) error {
	var s string
	switch v := value.(type) {
	case fmt.Stringer:
		s = v.String()
	case EnvStringer:
		s = v.String(env)
	case error:
		s = v.Error()
	default:
		s = toString(env, value)
	}
	return jsStringEscape(newStringWriter(out), s)
}

// showInJSONString shows value in JSONString context.
func showInJSONString(env runtime.Env, out io.Writer, value interface{}) error {
	return showInJSString(env, out, value)
}

// showInMarkdown shows value in the Markdown context.
func showInMarkdown(env runtime.Env, out io.Writer, value interface{}) error {
	w := newStringWriter(out)
	switch v := value.(type) {
	case Markdown:
		_, err := w.WriteString(string(v))
		return err
	case MarkdownStringer:
		_, err := w.WriteString(v.Markdown())
		return err
	case MarkdownEnvStringer:
		_, err := w.WriteString(v.Markdown(env))
		return err
	case fmt.Stringer:
		return markdownEscape(w, v.String())
	case EnvStringer:
		return markdownEscape(w, v.String(env))
	case error:
		return markdownEscape(w, v.Error())
	default:
		return markdownEscape(w, toString(env, value))
	}
}

// showInMarkdownCodeBlock shows value in the Markdown code block context.
func showInMarkdownCodeBlock(env runtime.Env, out io.Writer, value interface{}, spaces bool) error {
	var s string
	switch v := value.(type) {
	case fmt.Stringer:
		s = v.String()
	case EnvStringer:
		s = v.String(env)
	case error:
		s = v.Error()
	default:
		s = toString(env, value)
	}
	w := newStringWriter(out)
	return markdownCodeBlockEscape(w, s, spaces)
}

// showTimeInJS shows a value of type time.Time in a JavaScript context.
func showTimeInJS(tt time.Time) string {
	y := tt.Year()
	if y < -999999 || y > 999999 {
		panic("not representable year in JavaScript")
	}
	ms := int64(tt.Nanosecond()) / int64(time.Millisecond)
	name, offset := tt.Zone()
	if name == "UTC" {
		format := `new Date("%0.4d-%0.2d-%0.2dT%0.2d:%0.2d:%0.2d.%0.3dZ")`
		if y < 0 || y > 9999 {
			format = `new Date("%+0.6d-%0.2d-%0.2dT%0.2d:%0.2d:%0.2d.%0.3dZ")`
		}
		return fmt.Sprintf(format, y, tt.Month(), tt.Day(), tt.Hour(), tt.Minute(), tt.Second(), ms)
	}
	zone := offset / 60
	h, m := zone/60, zone%60
	if m < 0 {
		m = -m
	}
	format := `new Date("%0.4d-%0.2d-%0.2dT%0.2d:%0.2d:%0.2d.%0.3d%+0.2d:%0.2d")`
	if y < 0 || y > 9999 {
		format = `new Date("%+0.6d-%0.2d-%0.2dT%0.2d:%0.2d:%0.2d.%0.3d%+0.2d:%0.2d")`
	}
	return fmt.Sprintf(format, y, tt.Month(), tt.Day(), tt.Hour(), tt.Minute(), tt.Second(), ms, h, m)
}
