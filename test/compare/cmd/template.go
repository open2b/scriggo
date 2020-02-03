// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	utf8 "unicode/utf8"

	"scriggo"
	"scriggo/compiler"
	"scriggo/compiler/ast"
	"scriggo/runtime"
)

type mapReader map[string][]byte

func (r mapReader) Read(path string) ([]byte, error) {
	src, ok := r[path]
	if !ok {
		panic("not existing")
	}
	return src, nil
}

type dirReader string

func (dir dirReader) Read(path string) ([]byte, error) {
	src, err := ioutil.ReadFile(filepath.Join(string(dir), path))
	if err != nil {
		if os.IsNotExist(err) {
			panic("not existing")
		}
		return nil, err
	}
	return src, nil
}

type compiledTemplate struct {
	fn      *runtime.Function
	options *compiler.Options
	globals []compiler.Global
}

func compileTemplate(reader compiler.Reader, limitMemorySize bool) (*compiledTemplate, error) {
	opts := compiler.Options{
		LimitMemorySize: limitMemorySize,
		RelaxedBoolean: true,
	}
	mainImporter := scriggo.Packages{"main": templateMain}
	code, err := compiler.CompileTemplate("/index.html", reader, mainImporter, ast.LanguageHTML, opts)
	if err != nil {
		return nil, err
	}
	return &compiledTemplate{fn: code.Main, globals: code.Globals, options: &opts}, nil
}

func (t *compiledTemplate) render(ctx context.Context, maxMemoriSize int) error {
	out := os.Stdout
	writeFunc := out.Write
	renderFunc := render
	uw := &urlEscaper{w: out}
	t.globals[0].Value = &out
	t.globals[1].Value = &writeFunc
	t.globals[2].Value = &renderFunc
	t.globals[3].Value = &uw
	vm := runtime.NewVM()
	vm.SetContext(ctx)
	vm.SetMaxMemory(maxMemoriSize)
	_, err := vm.Run(t.fn, initGlobals(t.globals))
	return err
}

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
	default:
		panic("scriggo: unknown context")
	}
	if err != nil {
		panic(err)
	}
	return
}

func initGlobals(globals []compiler.Global) []interface{} {
	n := len(globals)
	if n == 0 {
		return nil
	}
	values := make([]interface{}, n)
	for i, global := range globals {
		if global.Value == nil {
			values[i] = reflect.New(global.Type).Interface()
		} else {
			values[i] = global.Value
		}
	}
	return values
}

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
		panic("toString does not support this kind")
	}
}

func renderInHTML(out io.Writer, value interface{}) error {
	w := newStringWriter(out)
	switch v := value.(type) {
	case compiler.HTMLStringer:
		_, err := w.WriteString(v.HTML())
		return err
	case fmt.Stringer:
		return htmlEscape(w, v.String())
	default:
		return htmlEscape(w, toString(reflect.ValueOf(value)))
	}
}

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

type strWriter interface {
	Write(b []byte) (int, error)
	WriteString(s string) (int, error)
}

type urlEscaper struct{ w io.Writer }

func (w *urlEscaper) StartURL(quoted, isSet bool) {}

func (w *urlEscaper) Write(p []byte) (int, error) { return 0, nil }

func (w *urlEscaper) WriteText(p []byte) (int, error) { return 0, nil }

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

func renderInAttribute(out io.Writer, value interface{}, quoted bool) error {
	var s string
	if v, ok := value.(fmt.Stringer); ok {
		s = v.String()
	} else {
		s = toString(reflect.ValueOf(value))
	}
	return attributeEscape(newStringWriter(out), s, quoted)
}

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
