// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/native"
)

var byteSliceType = reflect.TypeOf([]byte(nil))

// renderer is used by te Show and Text instructions to render template files.
type renderer struct {

	// env is the execution environment.
	env *env

	// out is the io.Writer to write to.
	out io.Writer

	// conv is the Markdown converter.
	conv Converter

	// inURL reports whether it is in a URL.
	inURL bool

	// query reports whether it is in the query string of an URL.
	query bool

	// addAmpersand reports whether an ampersand must be added before the next
	// written text. It can be true only if it is in a URL.
	addAmpersand bool

	// removeQuestionMark reports whether a question mark must be removed
	// before the next written text. It can be true only if it is in a URL.
	removeQuestionMark bool
}

// newRenderer returns a new renderer.
func newRenderer(env *env, out io.Writer, conv Converter) *renderer {
	return &renderer{env: env, out: out, conv: conv}
}

func (r *renderer) Close() error {
	if w, ok := r.out.(*markdownWriter); ok {
		return w.Close()
	}
	return nil
}

// Show shows v in the given context.
func (r *renderer) Show(v interface{}, context Context) error {

	ctx, inURL, _ := decodeRenderContext(context)

	// Check and eventually change the URL state.
	if r.inURL != inURL {
		if !inURL {
			r.endURL()
		}
		r.inURL = inURL
	}

	if inURL {
		return r.showInURL(v, ctx)
	}

	var err error

	switch ctx {
	case ast.ContextText:
		err = showInText(r.env, r.out, v)
	case ast.ContextHTML:
		err = showInHTML(r.env, r.out, r.conv, v)
	case ast.ContextTag:
		err = showInTag(r.env, r.out, v)
	case ast.ContextQuotedAttr:
		err = showInAttribute(r.env, r.out, v, true)
	case ast.ContextUnquotedAttr:
		err = showInAttribute(r.env, r.out, v, false)
	case ast.ContextCSS:
		err = showInCSS(r.env, r.out, v)
	case ast.ContextCSSString:
		err = showInCSSString(r.env, r.out, v)
	case ast.ContextJS:
		err = showInJS(r.env, r.out, v)
	case ast.ContextJSString:
		err = showInJSString(r.env, r.out, v)
	case ast.ContextJSON:
		err = showInJSON(r.env, r.out, v)
	case ast.ContextJSONString:
		err = showInJSONString(r.env, r.out, v)
	case ast.ContextMarkdown:
		err = showInMarkdown(r.env, r.out, v)
	case ast.ContextTabCodeBlock:
		err = showInMarkdownCodeBlock(r.env, r.out, v, false)
	case ast.ContextSpacesCodeBlock:
		err = showInMarkdownCodeBlock(r.env, r.out, v, true)
	default:
		panic("scriggo: unknown context")
	}

	return err
}

// Out returns the out writer.
func (r *renderer) Out() io.Writer {
	return r.out
}

// Text shows txt in the given context.
func (r *renderer) Text(txt []byte, inURL, isSet bool) error {

	// Check and eventually change the URL state.
	if r.inURL != inURL {
		if !inURL {
			r.endURL()
		}
		r.inURL = inURL
	}

	if inURL {
		if isSet && bytes.ContainsRune(txt, ',') {
			r.query = false
		} else if r.query {
			if r.removeQuestionMark && txt[0] == '?' {
				txt = txt[1:]
			}
			if r.addAmpersand && len(txt) > 0 && txt[0] != '&' {
				_, err := io.WriteString(r.out, "&amp;")
				if err != nil {
					return err
				}
			}
			r.removeQuestionMark = false
			r.addAmpersand = false
		} else {
			r.query = bytes.ContainsAny(txt, "?#")
		}
		_, err := r.out.Write(txt)
		return err
	}

	_, err := r.out.Write(txt)
	return err
}

func (r *renderer) WithConversion(from, to ast.Format) *renderer {
	if from == ast.FormatMarkdown && to == ast.FormatHTML {
		out := newMarkdownWriter(r.out, r.conv)
		return &renderer{env: r.env, out: out, conv: r.conv}
	}
	return &renderer{env: r.env, out: r.out, conv: r.conv}
}

func (r *renderer) WithOut(out io.Writer) *renderer {
	return &renderer{env: r.env, out: out, conv: r.conv}
}

// showInURL shows v in a URL in the given context.
func (r *renderer) showInURL(v interface{}, ctx ast.Context) error {

	var b strings.Builder
	err := showInHTML(r.env, &b, r.conv, v)
	if err != nil {
		return err
	}

	s := html.UnescapeString(b.String())
	out := newStringWriter(r.out)

	if r.query {
		if r.removeQuestionMark {
			c := s[len(s)-1]
			r.addAmpersand = c != '&'
			_, err := pathEscape(out, s, ctx == ast.ContextQuotedAttr)
			return err
		}
		_, err := queryEscape(out, s)
		return err
	}

	if strings.Contains(s, "?") {
		r.query = true
		r.removeQuestionMark = true
		if c := s[len(s)-1]; c != '&' && c != '?' {
			r.addAmpersand = true
		}
	}
	_, err = pathEscape(out, s, ctx == ast.ContextQuotedAttr)

	return err
}

// endURL is called when an URL ends.
func (r *renderer) endURL() {
	r.inURL = false
	r.query = false
	r.addAmpersand = false
	r.removeQuestionMark = false
}

// markdownWriter implements an io.WriteCloser that writes to the buffer buf.
// When the Close method is called, it converts the content in the buffer,
// using converter, from Markdown to HTML and writes it to out.
type markdownWriter struct {
	buf     bytes.Buffer
	convert Converter
	out     io.Writer
}

// newMarkdownWriter returns a *markdownWriter value that writes to out the
// Markdown code converted to HTML by converter.
func newMarkdownWriter(out io.Writer, converter Converter) *markdownWriter {
	var buf bytes.Buffer
	return &markdownWriter{buf, converter, out}
}

func (w *markdownWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *markdownWriter) Close() error {
	if w.convert == nil {
		return errors.New("no Markdown convert available")
	}
	return w.convert(w.buf.Bytes(), w.out)
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

func toString(env *env, i interface{}) (string, error) {
	v := valueOf(env, i)
	switch v.Kind() {
	case reflect.Invalid:
		return "", nil
	case reflect.Bool:
		if v.Bool() {
			return "true", nil
		}
		return "false", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10), nil
	case reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'f', -1, 32), nil
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64), nil
	case reflect.String:
		return v.String(), nil
	case reflect.Complex64:
		c := v.Complex()
		if c == 0 {
			return "0", nil
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
		return s, nil
	case reflect.Complex128:
		c := v.Complex()
		if c == 0 {
			return "0", nil
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
		return s, nil
	default:
		return "", fmt.Errorf("cannot show value of type %s", env.TypeOf(reflect.ValueOf(i)))
	}
}

// showInText shows value in the Text context.
func showInText(env *env, out io.Writer, value interface{}) error {
	var s string
	switch v := value.(type) {
	case fmt.Stringer:
		s = v.String()
	case native.EnvStringer:
		s = v.String(env)
	case error:
		s = v.Error()
	default:
		var err error
		s, err = toString(env, value)
		if err != nil {
			return err
		}
	}
	w := newStringWriter(out)
	_, err := w.WriteString(s)
	return err
}

// showInHTML shows value in HTML context.
func showInHTML(env *env, out io.Writer, conv Converter, value interface{}) error {
	w := newStringWriter(out)
	switch v := value.(type) {
	case native.HTML:
		_, err := w.WriteString(string(v))
		return err
	case native.HTMLStringer:
		_, err := w.WriteString(string(v.HTML()))
		return err
	case native.HTMLEnvStringer:
		_, err := w.WriteString(string(v.HTML(env)))
		return err
	case fmt.Stringer:
		return htmlEscape(w, v.String())
	case native.EnvStringer:
		return htmlEscape(w, v.String(env))
	case []byte:
		_, err := out.Write(v)
		return err
	case error:
		return htmlEscape(w, v.Error())
	case native.Markdown:
		if conv != nil {
			return conv([]byte(v), out)
		}
	}
	s, err := toString(env, value)
	if err != nil {
		return err
	}
	return htmlEscape(w, s)
}

// showInTag show value in Tag context.
func showInTag(env *env, out io.Writer, value interface{}) error {
	var s string
	switch v := value.(type) {
	case fmt.Stringer:
		s = v.String()
	case native.EnvStringer:
		s = v.String(env)
	case error:
		s = v.Error()
	default:
		var err error
		s, err = toString(env, value)
		if err != nil {
			return err
		}
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
func showInAttribute(env *env, out io.Writer, value interface{}, quoted bool) error {
	var s string
	var escapeEntities bool
	switch v := value.(type) {
	case fmt.Stringer:
		s = v.String()
		escapeEntities = true
	case native.EnvStringer:
		s = v.String(env)
		escapeEntities = true
	case native.HTML:
		s = string(v)
	case native.HTMLStringer:
		s = string(v.HTML())
	case native.HTMLEnvStringer:
		s = string(v.HTML(env))
	case error:
		s = v.Error()
		escapeEntities = true
	default:
		var err error
		s, err = toString(env, value)
		if err != nil {
			return err
		}
		escapeEntities = true
	}
	return attributeEscape(newStringWriter(out), s, escapeEntities, quoted)
}

// showInCSS shows value in CSS context.
func showInCSS(env *env, out io.Writer, value interface{}) error {
	w := newStringWriter(out)
	switch v := value.(type) {
	case native.CSS:
		_, err := w.WriteString(string(v))
		return err
	case native.CSSStringer:
		_, err := w.WriteString(string(v.CSS()))
		return err
	case native.CSSEnvStringer:
		_, err := w.WriteString(string(v.CSS(env)))
		return err
	case fmt.Stringer:
		value = v.String()
	case native.EnvStringer:
		value = v.String(env)
	case []byte:
		return escapeBytes(w, v, false)
	case error:
		value = v.Error()
	}
	v := valueOf(env, value)
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
		s, err := toString(env, value)
		if err != nil {
			return err
		}
		_, err = w.WriteString(s)
		return err
	}
}

// showInCSSString shows value in CSSString context.
func showInCSSString(env *env, out io.Writer, value interface{}) error {
	var s string
	switch value := value.(type) {
	case fmt.Stringer:
		s = value.String()
	case native.EnvStringer:
		s = value.String(env)
	case error:
		s = value.Error()
	default:
		v := reflect.ValueOf(value)
		if v.Type() == byteSliceType {
			w := newStringWriter(out)
			return escapeBytes(w, v.Interface().([]byte), false)
		}
		var err error
		s, err = toString(env, value)
		if err != nil {
			return err
		}
	}
	return cssStringEscape(newStringWriter(out), s)
}

// showInJS shows value in JavaScript context.
func showInJS(env *env, out io.Writer, value interface{}) error {

	w := newStringWriter(out)

	switch v := value.(type) {
	case nil:
		_, err := w.WriteString("null")
		return err
	case native.JS:
		_, err := w.WriteString(string(v))
		return err
	case native.JSStringer:
		_, err := w.WriteString(string(v.JS()))
		return err
	case native.JSEnvStringer:
		_, err := w.WriteString(string(v.JS(env)))
		return err
	case time.Time:
		_, err := w.WriteString(showTimeInJS(v))
		return err
	case error:
		value = v.Error()
	}

	v := valueOf(env, value)

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
					tagName, omitempty := parseTagValue(tag)
					if omitempty && isEmptyValue(value) {
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
			case native.EnvStringer:
				keyPairs[i].key = k.String(env)
			default:
				s, err := toString(env, k)
				if err != nil {
					return err
				}
				keyPairs[i].key = s
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
		t := env.TypeOf(reflect.ValueOf(value))
		s = fmt.Sprintf("undefined/* scriggo: cannot represent a %s value */", t)
	}

	_, err := w.WriteString(s)
	return err
}

// showInJSON shows value in JSON context.
func showInJSON(env *env, out io.Writer, value interface{}) error {

	w := newStringWriter(out)

	switch v := value.(type) {
	case nil:
		_, err := w.WriteString("null")
		return err
	case native.JSON:
		_, err := w.WriteString(string(v))
		return err
	case native.JSONStringer:
		_, err := w.WriteString(string(v.JSON()))
		return err
	case native.JSONEnvStringer:
		_, err := w.WriteString(string(v.JSON(env)))
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
					tagName, omitempty := parseTagValue(tag)
					if omitempty && isEmptyValue(value) {
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
			case native.EnvStringer:
				keyPairs[i].key = k.String(env)
			default:
				s, err := toString(env, k)
				if err != nil {
					return err
				}
				keyPairs[i].key = s
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
func showInJSString(env *env, out io.Writer, value interface{}) error {
	var s string
	switch v := value.(type) {
	case fmt.Stringer:
		s = v.String()
	case native.EnvStringer:
		s = v.String(env)
	case error:
		s = v.Error()
	default:
		var err error
		s, err = toString(env, value)
		if err != nil {
			return err
		}
	}
	return jsStringEscape(newStringWriter(out), s)
}

// showInJSONString shows value in JSONString context.
func showInJSONString(env *env, out io.Writer, value interface{}) error {
	return showInJSString(env, out, value)
}

// showInMarkdown shows value in the Markdown context.
func showInMarkdown(env *env, out io.Writer, value interface{}) error {
	w := newStringWriter(out)
	switch v := value.(type) {
	case native.Markdown:
		_, err := w.WriteString(string(v))
		return err
	case native.MarkdownStringer:
		_, err := w.WriteString(string(v.Markdown()))
		return err
	case native.MarkdownEnvStringer:
		_, err := w.WriteString(string(v.Markdown(env)))
		return err
	case native.HTML:
		return markdownEscape(w, string(v), true)
	case native.HTMLStringer:
		return markdownEscape(w, string(v.HTML()), true)
	case native.HTMLEnvStringer:
		return markdownEscape(w, string(v.HTML(env)), true)
	case fmt.Stringer:
		return markdownEscape(w, v.String(), false)
	case native.EnvStringer:
		return markdownEscape(w, v.String(env), false)
	case error:
		return markdownEscape(w, v.Error(), false)
	default:
		s, err := toString(env, value)
		if err != nil {
			return err
		}
		return markdownEscape(w, s, false)
	}
}

// showInMarkdownCodeBlock shows value in the Markdown code block context.
func showInMarkdownCodeBlock(env *env, out io.Writer, value interface{}, spaces bool) error {
	var s string
	switch v := value.(type) {
	case fmt.Stringer:
		s = v.String()
	case native.EnvStringer:
		s = v.String(env)
	case error:
		s = v.Error()
	default:
		var err error
		s, err = toString(env, value)
		if err != nil {
			return err
		}
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

// parseTagValue parses a 'json' tag value and returns its name and whether
// its options contain 'omitempty'.
func parseTagValue(tag string) (name string, omitempty bool) {
	i := strings.Index(tag, ",")
	if i == -1 {
		return tag, false
	}
	name, tag = tag[:i], tag[:i+1]
	for tag != "" {
		i = strings.Index(tag, ",")
		if i == -1 {
			return name, tag == "omitempty"
		}
		if tag[:i] == "omitempty" {
			return name, true
		}
		tag = tag[i+1:]
	}
	return name, false
}

// isEmptyValue reports whether v is an empty value for JS and JSON.
func isEmptyValue(v reflect.Value) (empty bool) {
	switch v.Kind() {
	case reflect.Bool:
		empty = !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		empty = v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		empty = v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		empty = v.Float() == 0
	case reflect.String, reflect.Map, reflect.Slice, reflect.Array:
		empty = v.Len() == 0
	case reflect.Interface, reflect.Ptr, reflect.UnsafePointer:
		empty = v.IsNil()
	}
	return
}

// valueOf returns the reflect value of i. If i is a proxy, it returns the
// underlying value.
func valueOf(env *env, i interface{}) reflect.Value {
	if i == nil {
		return reflect.Value{}
	}
	v := reflect.ValueOf(i)
	t := env.TypeOf(v)
	if w, ok := t.(ScriggoType); ok {
		v, _ = w.Unwrap(v)
	}
	return v
}

// decodeRenderContext decodes a runtime.Context.
// Keep in sync with the compiler.decodeRenderContext.
func decodeRenderContext(c Context) (ast.Context, bool, bool) {
	ctx := ast.Context(c & 0b00001111)
	inURL := c&0b10000000 != 0
	isURLSet := false
	if inURL {
		isURLSet = c&0b01000000 != 0
	}
	return ctx, inURL, isURLSet
}
