// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	_sort "sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"scriggo/vm"
)

// ErrNoRenderInContext indicates that a value, for its type, cannot be
// rendered in a context.
var ErrNoRenderInContext = errors.New("cannot render type in context")

// DefaultRenderFunc is the default RenderFunc used by Render method if the
// option RenderFunc is nil.
var DefaultRenderFunc = render

// ValueRenderer is called by the RenderFunc and is implemented by the types
// that know how to render their values. Render renders the value in the
// context ctx and writes to out.
//
// If the value can not be rendered in context ctx, Render returns the
// ErrNoRenderInContext error.
type ValueRenderer interface {
	Render(out io.Writer, ctx Context) error
}

// Render renders value in the context ctx and writes to out.
func render(_ *vm.Env, out io.Writer, value interface{}, ctx Context) {

	// TODO: pass url state to render.

	if e, ok := value.(ValueRenderer); ok {
		err := e.Render(out, ctx)
		if err != nil {
			if err == ErrNoRenderInContext {
				panic(fmt.Errorf("cannot render %T in %s context", value, ctx))
			}
			panic(err)
		}
		return
	}

	w := newStringWriter(out)

	if e, ok := value.(error); ok {
		value = e.Error()
	}

	var err error

	switch ctx {
	case ContextText:
		err = renderInText(w, value)
	case ContextHTML:
		err = renderInHTML(w, value)
	case ContextTag:
		err = renderInTag(w, value)
	case ContextAttribute:
		//if urlstate == nil {
		err = renderInAttribute(w, value, true)
		//} else {
		//	err = r.renderInAttributeURL(w, value, node, urlstate, true)
		//}
	case ContextUnquotedAttribute:
		//if urlstate == nil {
		err = renderInAttribute(w, value, false)
		//} else {
		//	err = r.renderInAttributeURL(w, value, node, urlstate, false)
		//}
	case ContextCSS:
		err = renderInCSS(w, value)
	case ContextCSSString:
		err = renderInCSSString(w, value)
	case ContextJavaScript:
		err = renderInJavaScript(w, value)
	case ContextJavaScriptString:
		err = renderInJavaScriptString(w, value)
	default:
		panic("scriggo: unknown context")
	}
	if err != nil {
		if err == ErrNoRenderInContext {
			panic(fmt.Errorf("cannot render %T in %s context", value, ctx))
		}
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

// renderInText renders value in the Text context.
func renderInText(w strWriter, value interface{}) error {

	if value == nil {
		return nil
	}

	var s string

	switch v := withConcreteType(value).(type) {
	case bool:
		s = "false"
		if v {
			s = "true"
		}
	case int64:
		s = strconv.FormatInt(v, 10)
	case uint64:
		s = strconv.FormatUint(v, 10)
	case float32:
		s = strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		s = strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		s = v
	default:
		rv := reflect.ValueOf(value)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return ErrNoRenderInContext
		}
		if rv.IsNil() || rv.Len() == 0 {
			return nil
		}
		var err error
		for i, l := 0, rv.Len(); i < l; i++ {
			if i > 0 {
				_, err := w.WriteString(", ")
				if err != nil {
					return err
				}
			}
			v := rv.Index(i).Interface()
			if e, ok := v.(ValueRenderer); ok {
				err = e.Render(w, ContextText)
			} else {
				err = renderInText(w, v)
			}
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
func renderInHTML(w strWriter, value interface{}) error {

	if value == nil {
		return nil
	}

	var s string

	switch v := withConcreteType(value).(type) {
	case bool:
		s = "false"
		if v {
			s = "true"
		}
	case int64:
		s = strconv.FormatInt(v, 10)
	case uint64:
		s = strconv.FormatUint(v, 10)
	case float32:
		s = strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		s = strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return htmlEscape(w, v)
	default:
		rv := reflect.ValueOf(value)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return ErrNoRenderInContext
		}
		if rv.IsNil() || rv.Len() == 0 {
			return nil
		}
		var err error
		for i, l := 0, rv.Len(); i < l; i++ {
			if i > 0 {
				_, err := w.WriteString(", ")
				if err != nil {
					return err
				}
			}
			v := rv.Index(i).Interface()
			if e, ok := v.(ValueRenderer); ok {
				err = e.Render(w, ContextHTML)
			} else {
				err = renderInHTML(w, v)
			}
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
func renderInTag(w strWriter, value interface{}) error {
	buf := strings.Builder{}
	err := renderInText(&buf, value)
	if err != nil {
		return err
	}
	s := buf.String()
	i := 0
	for j, c := range s {
		if c == utf8.RuneError && j == i+1 {
			return errors.New("not valid unicode in tag context")
		}
		const DEL = 0x7F
		if c <= 0x1F || c == '"' || c == '\'' || c == '>' || c == '/' || c == '=' || c == DEL ||
			0x7F <= c && c <= 0x9F || unicode.Is(unicode.Noncharacter_Code_Point, c) {
			return fmt.Errorf("not allowed character %s in tag context", string(c))
		}
	}
	_, err = w.WriteString(s)
	return err
}

// renderInAttribute renders value in Attribute context quoted or unquoted
// depending on quoted value.
func renderInAttribute(w strWriter, value interface{}, quoted bool) error {

	if value == nil {
		return nil
	}

	var s string

	switch v := withConcreteType(value).(type) {
	case bool:
		s = "false"
		if v {
			s = "true"
		}
	case int64:
		s = strconv.FormatInt(v, 10)
	case uint64:
		s = strconv.FormatUint(v, 10)
	case float32:
		s = strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		s = strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return attributeEscape(w, v, quoted)
	default:
		rv := reflect.ValueOf(value)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return ErrNoRenderInContext
		}
		if rv.IsNil() || rv.Len() == 0 {
			return nil
		}
		var err error
		for i, l := 0, rv.Len(); i < l; i++ {
			if i > 0 {
				if quoted {
					_, err = w.WriteString(", ")
				} else {
					_, err = w.WriteString(",&#32;")
				}
				if err != nil {
					return err
				}
			}
			v := rv.Index(i).Interface()
			if e, ok := v.(ValueRenderer); ok {
				err = e.Render(w, ContextAttribute)
			} else {
				err = renderInAttribute(w, v, quoted)
			}
			if err != nil {
				return err
			}
		}
	}

	_, err := w.WriteString(s)

	return err
}

// renderInAttributeURL renders value as an URL in Attribute context, quoted
// or unquoted depending on quoted value.
func renderInAttributeURL(w strWriter, value interface{}, inQuery, quoted bool) error {

	if value == nil {
		return nil
	}

	var s string

	switch v := withConcreteType(value).(type) {
	case bool:
		s = "false"
		if v {
			s = "true"
		}
	case int64:
		s = strconv.FormatInt(v, 10)
	case uint64:
		s = strconv.FormatUint(v, 10)
	case float32:
		s = strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		s = strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		s = v
	default:
		rv := reflect.ValueOf(value)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return ErrNoRenderInContext
		}
		if rv.IsNil() || rv.Len() == 0 {
			return nil
		}
		for i, l := 0, rv.Len(); i < l; i++ {
			if i > 0 {
				if quoted {
					s += ", "
				} else {
					s += ",&#32;"
				}
			}
			switch v := withConcreteType(rv.Index(i).Interface()).(type) {
			case bool:
				if v {
					s += "true"
				} else {
					s += "false"
				}
			case int64:
				s = strconv.FormatInt(v, 10)
			case uint64:
				s = strconv.FormatUint(v, 10)
			case float32:
				s = strconv.FormatFloat(float64(v), 'f', -1, 32)
			case float64:
				s = strconv.FormatFloat(v, 'f', -1, 64)
			case string:
				s += v
			default:
				return ErrNoRenderInContext
			}
		}
	}

	if s == "" {
		return nil
	}

	if inQuery {
		return queryEscape(w, s)
	}

	return pathEscape(w, s, quoted)
}

// renderInCSS renders value in CSS context.
func renderInCSS(w strWriter, value interface{}) error {

	if value == nil {
		return ErrNoRenderInContext
	}

	var s string

	switch v := withConcreteType(value).(type) {
	case int64:
		s = strconv.FormatInt(v, 10)
	case uint64:
		s = strconv.FormatUint(v, 10)
	case float32:
		s = strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		s = strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		_, err := w.WriteString(`"`)
		if err != nil {
			return err
		}
		err = cssStringEscape(w, v)
		if err != nil {
			return err
		}
		_, err = w.WriteString(`"`)
		return err
	case []byte:
		return escapeBytes(w, v, false)
	default:
		return ErrNoRenderInContext
	}

	_, err := w.WriteString(s)

	return err
}

// renderInCSSString renders value in CSSString context.
func renderInCSSString(w strWriter, value interface{}) error {

	if value == nil {
		return ErrNoRenderInContext
	}

	var s string

	switch v := withConcreteType(value).(type) {
	case int64:
		s = strconv.FormatInt(v, 10)
	case uint64:
		s = strconv.FormatUint(v, 10)
	case float32:
		s = strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		s = strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return cssStringEscape(w, v)
	case []byte:
		return escapeBytes(w, v, false)
	default:
		return ErrNoRenderInContext
	}

	_, err := w.WriteString(s)

	return err
}

var mapStringToInterfaceType = reflect.TypeOf(map[string]interface{}{})

// renderInJavaScript renders value in JavaScript context.
func renderInJavaScript(w strWriter, value interface{}) error {

	switch v := withConcreteType(value).(type) {
	case nil:
		_, err := w.WriteString("null")
		return err
	case bool:
		s := "false"
		if v {
			s = "true"
		}
		_, err := w.WriteString(s)
		return err
	case int64:
		_, err := w.WriteString(strconv.FormatInt(v, 10))
		return err
	case uint64:
		_, err := w.WriteString(strconv.FormatUint(v, 10))
		return err
	case float32:
		_, err := w.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 32))
		return err
	case float64:
		_, err := w.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
		return err
	case string:
		_, err := w.WriteString("\"")
		if err != nil {
			return err
		}
		err = javaScriptStringEscape(w, v)
		if err != nil {
			return err
		}
		_, err = w.WriteString("\"")
		return err
	case []byte:
		return escapeBytes(w, v, true)
	default:
		var err error
		rv := reflect.ValueOf(value)
		if !rv.IsValid() {
			_, err = w.WriteString("undefined")
			if err != nil {
				return err
			}
			return ErrNoRenderInContext
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
				v := rv.Index(i).Interface()
				if e, ok := v.(ValueRenderer); ok {
					err = e.Render(w, ContextJavaScript)
				} else {
					err = renderInJavaScript(w, v)
				}
				if err != nil {
					return err
				}
			}
			_, err = w.WriteString("]")
			return err
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
				return ErrNoRenderInContext
			}
			return renderMapAsJavaScriptObject(w, rv.Convert(mapStringToInterfaceType).Interface().(map[string]interface{}))
		case reflect.Struct, reflect.Ptr:
			return renderValueAsJavaScriptObject(w, rv)
		}
	}

	_, err := w.WriteString("undefined")
	if err != nil {
		return err
	}
	return ErrNoRenderInContext
}

// renderInJavaScriptString renders value in JavaScriptString context.
func renderInJavaScriptString(w strWriter, value interface{}) error {
	switch v := withConcreteType(value).(type) {
	case nil:
		return ErrNoRenderInContext
	case int64:
		_, err := w.WriteString(strconv.FormatInt(v, 10))
		return err
	case uint64:
		_, err := w.WriteString(strconv.FormatUint(v, 10))
		return err
	case float32:
		_, err := w.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 32))
		return err
	case float64:
		_, err := w.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
		return err
	case string:
		err := javaScriptStringEscape(w, v)
		return err
	}
	return ErrNoRenderInContext
}

// renderValueAsJavaScriptObject returns value as a JavaScript object or
// undefined if it is not possible.
func renderValueAsJavaScriptObject(w strWriter, rv reflect.Value) error {

	if rv.IsNil() {
		_, err := w.WriteString("null")
		return err
	}

	keys := structKeys(rv)
	if keys == nil {
		return ErrNoRenderInContext
	}

	if len(keys) == 0 {
		_, err := w.WriteString(`{}`)
		return err
	}

	names := make([]string, 0, len(keys))
	for name := range keys {
		names = append(names, name)
	}
	_sort.Strings(names)

	for i, name := range names {
		sep := `,"`
		if i == 0 {
			sep = `{"`
		}
		_, err := w.WriteString(sep)
		if err != nil {
			return err
		}
		err = javaScriptStringEscape(w, name)
		if err != nil {
			return err
		}
		_, err = w.WriteString(`":`)
		if err != nil {
			return err
		}
		err = renderInJavaScript(w, keys[name].value(rv))
		if err != nil {
			return err
		}
	}

	_, err := w.WriteString(`}`)

	return err
}

// renderMapAsJavaScriptObject returns value as a JavaScript object or
// undefined if it is not possible.
func renderMapAsJavaScriptObject(w strWriter, value map[string]interface{}) error {

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
	_sort.Strings(names)

	for i, n := range names {
		if i > 0 {
			_, err = w.WriteString(",")
			if err != nil {
				return err
			}
		}
		err = javaScriptStringEscape(w, n)
		if err != nil {
			return err
		}
		_, err = w.WriteString(":")
		if err != nil {
			return err
		}
		err = renderInJavaScript(w, value[n])
		if err != nil {
			return err
		}
	}

	_, err = w.WriteString("}")

	return err
}

func withConcreteType(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	typ := reflect.TypeOf(v)
	if typ.Name() == "" {
		return v
	}
	r := reflect.ValueOf(v)
	switch typ.Kind() {
	case reflect.Bool:
		return r.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return r.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return r.Uint()
	case reflect.Float32, reflect.Float64:
		return r.Float()
	case reflect.String:
		return r.String()
	}
	return nil
}

// structs maintains the association between the field names of a struct,
// as they are called in the template, and the field index in the struct.
var structs = struct {
	keys map[reflect.Type]map[string]structKey
	sync.RWMutex
}{map[reflect.Type]map[string]structKey{}, sync.RWMutex{}}

// structKey represents the fields of a struct.
type structKey struct {
	isMethod bool
	index    int
}

func (sk structKey) value(st reflect.Value) interface{} {
	st = reflect.Indirect(st)
	if sk.isMethod {
		return st.Method(sk.index).Interface()
	}
	return st.Field(sk.index).Interface()
}

func structKeys(st reflect.Value) map[string]structKey {
	st = reflect.Indirect(st)
	if st.Kind() != reflect.Struct {
		return nil
	}
	structs.RLock()
	keys, ok := structs.keys[st.Type()]
	structs.RUnlock()
	if ok {
		return keys
	}
	typ := st.Type()
	structs.Lock()
	keys, ok = structs.keys[st.Type()]
	if ok {
		structs.Unlock()
		return keys
	}
	keys = map[string]structKey{}
	n := typ.NumField()
	for i := 0; i < n; i++ {
		fieldType := typ.Field(i)
		if fieldType.PkgPath != "" {
			continue
		}
		name := fieldType.Name
		keys[name] = structKey{index: i}
	}
	n = typ.NumMethod()
	for i := 0; i < n; i++ {
		name := typ.Method(i).Name
		if _, ok := keys[name]; !ok {
			keys[name] = structKey{index: i, isMethod: true}
		}
	}
	structs.keys[st.Type()] = keys
	structs.Unlock()
	return keys
}
