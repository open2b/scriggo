// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package templates

import (
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/runtime"
)

var stringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
var envStringerType = reflect.TypeOf((*EnvStringer)(nil)).Elem()

var htmlStringerType = reflect.TypeOf((*HTMLStringer)(nil)).Elem()
var htmlEnvStringerType = reflect.TypeOf((*HTMLEnvStringer)(nil)).Elem()

var cssStringerType = reflect.TypeOf((*CSSStringer)(nil)).Elem()
var cssEnvStringerType = reflect.TypeOf((*CSSEnvStringer)(nil)).Elem()

var jsStringerType = reflect.TypeOf((*JSStringer)(nil)).Elem()
var jsEnvStringerType = reflect.TypeOf((*JSEnvStringer)(nil)).Elem()

var jsonStringerType = reflect.TypeOf((*JSONStringer)(nil)).Elem()
var jsonEnvStringerType = reflect.TypeOf((*JSONEnvStringer)(nil)).Elem()

var mdStringerType = reflect.TypeOf((*MarkdownStringer)(nil)).Elem()
var mdEnvStringerType = reflect.TypeOf((*MarkdownEnvStringer)(nil)).Elem()

var byteSliceType = reflect.TypeOf([]byte(nil))
var timeType = reflect.TypeOf(time.Time{})
var errorType = reflect.TypeOf((*error)(nil)).Elem()

// decodeRenderContext decodes a runtime.Renderer context.
// Keep in sync with the compiler.decodeRenderContext.
func decodeRenderContext(c uint8) (ast.Context, bool, bool) {
	ctx := ast.Context(c & 0b00001111)
	inURL := c&0b10000000 != 0
	isURLSet := false
	if inURL {
		isURLSet = c&0b01000000 != 0
	}
	return ctx, inURL, isURLSet
}

// buildRenderer implements the runtime.Renderer interface and it is used
// during the build of the template to type checks a show statement.
type buildRenderer struct{}

func (r buildRenderer) Enter(io.Writer, uint8) runtime.Renderer { return nil }

func (r buildRenderer) Exit() error { return nil }

func (r buildRenderer) Show(env runtime.Env, v interface{}, context uint8) {
	t := env.Types().TypeOf(v)
	ctx, _, _ := decodeRenderContext(context)
	kind := t.Kind()
	switch ctx {
	case ast.ContextText, ast.ContextTag, ast.ContextQuotedAttr, ast.ContextUnquotedAttr,
		ast.ContextCSSString, ast.ContextJSString, ast.ContextJSONString,
		ast.ContextTabCodeBlock, ast.ContextSpacesCodeBlock:
		switch {
		case kind == reflect.String:
		case reflect.Bool <= kind && kind <= reflect.Complex128:
		case ctx == ast.ContextCSSString && t == byteSliceType:
		case t.Implements(stringerType):
		case t.Implements(envStringerType):
		case t.Implements(errorType):
		default:
			env.Fatal(fmt.Errorf("cannot show type %s as %s", t, ctx))
		}
	case ast.ContextHTML:
		switch {
		case kind == reflect.String:
		case reflect.Bool <= kind && kind <= reflect.Complex128:
		case t == byteSliceType:
		case t.Implements(stringerType):
		case t.Implements(envStringerType):
		case t.Implements(htmlStringerType):
		case t.Implements(htmlEnvStringerType):
		case t.Implements(errorType):
		default:
			env.Fatal(fmt.Errorf("cannot show type %s as HTML", t))
		}
	case ast.ContextCSS:
		switch {
		case kind == reflect.String:
		case reflect.Int <= kind && kind <= reflect.Float64:
		case t == byteSliceType:
		case t.Implements(stringerType):
		case t.Implements(envStringerType):
		case t.Implements(cssStringerType):
		case t.Implements(cssEnvStringerType):
		case t.Implements(errorType):
		default:
			env.Fatal(fmt.Errorf("cannot show type %s as CSS", t))
		}
	case ast.ContextJS:
		err := shownAsJS(t)
		if err != nil {
			env.Fatal(err)
		}
	case ast.ContextJSON:
		err := shownAsJSON(t)
		if err != nil {
			env.Fatal(err)
		}
	case ast.ContextMarkdown:
		switch {
		case kind == reflect.String:
		case reflect.Bool <= kind && kind <= reflect.Complex128:
		case t.Implements(stringerType):
		case t.Implements(envStringerType):
		case t.Implements(mdStringerType):
		case t.Implements(mdEnvStringerType):
		case t.Implements(htmlStringerType):
		case t.Implements(htmlEnvStringerType):
		case t.Implements(errorType):
		default:
			env.Fatal(fmt.Errorf("cannot show type %s as Markdown", t))
		}
	default:
		panic("unexpected context")
	}
	return
}

func (r buildRenderer) Text(runtime.Env, []byte, uint8) {}

// shownAsJS reports whether a type can be shown as JavaScript. It returns
// an error if the type cannot be shown.
func shownAsJS(t reflect.Type) error {
	kind := t.Kind()
	if reflect.Bool <= kind && kind <= reflect.Float64 || kind == reflect.String ||
		t == timeType ||
		t.Implements(jsStringerType) ||
		t.Implements(jsEnvStringerType) ||
		t.Implements(errorType) {
		return nil
	}
	switch kind {
	case reflect.Array:
		if err := shownAsJS(t.Elem()); err != nil {
			return fmt.Errorf("cannot show array of %s as JavaScript", t.Elem())
		}
	case reflect.Interface:
	case reflect.Map:
		key := t.Key().Kind()
		switch {
		case key == reflect.String:
		case reflect.Bool <= key && key <= reflect.Complex128:
		case t.Implements(stringerType):
		case t.Implements(envStringerType):
		default:
			return fmt.Errorf("cannot show map with %s key as JavaScript", t.Key())
		}
		err := shownAsJS(t.Elem())
		if err != nil {
			return fmt.Errorf("cannot show map with %s element as JavaScript", t.Elem())
		}
	case reflect.Ptr, reflect.UnsafePointer:
		return shownAsJS(t.Elem())
	case reflect.Slice:
		if err := shownAsJS(t.Elem()); err != nil {
			return fmt.Errorf("cannot show slice of %s as JavaScript", t.Elem())
		}
	case reflect.Struct:
		n := t.NumField()
		for i := 0; i < n; i++ {
			field := t.Field(i)
			if field.PkgPath == "" {
				if err := shownAsJS(field.Type); err != nil {
					return fmt.Errorf("cannot show struct containing %s as JavaScript", field.Type)
				}
			}
		}
	default:
		return fmt.Errorf("cannot show type %s as JavaScript", t)
	}
	return nil
}

// shownAsJSON reports whether a type can be shown as JSON. It returns an
// error if the type cannot be shown.
func shownAsJSON(t reflect.Type) error {
	kind := t.Kind()
	if reflect.Bool <= kind && kind <= reflect.Float64 || kind == reflect.String ||
		t == timeType ||
		t.Implements(jsonStringerType) ||
		t.Implements(jsonEnvStringerType) ||
		t.Implements(errorType) {
		return nil
	}
	switch kind {
	case reflect.Array:
		if err := shownAsJSON(t.Elem()); err != nil {
			return fmt.Errorf("cannot show array of %s as JSON", t.Elem())
		}
	case reflect.Interface:
	case reflect.Map:
		key := t.Key().Kind()
		switch {
		case key == reflect.String:
		case reflect.Bool <= key && key <= reflect.Complex128:
		case t.Implements(stringerType):
		case t.Implements(envStringerType):
		default:
			return fmt.Errorf("cannot show map with %s key as JSON", t.Key())
		}
		err := shownAsJSON(t.Elem())
		if err != nil {
			return fmt.Errorf("cannot show map with %s element as JSON", t.Elem())
		}
	case reflect.Ptr, reflect.UnsafePointer:
		return shownAsJSON(t.Elem())
	case reflect.Slice:
		if err := shownAsJSON(t.Elem()); err != nil {
			return fmt.Errorf("cannot show slice of %s as JSON", t.Elem())
		}
	case reflect.Struct:
		n := t.NumField()
		for i := 0; i < n; i++ {
			field := t.Field(i)
			if field.PkgPath == "" {
				if err := shownAsJSON(field.Type); err != nil {
					return fmt.Errorf("cannot show struct containing %s as JSON", field.Type)
				}
			}
		}
	default:
		return fmt.Errorf("cannot show type %s as JSON", t)
	}
	return nil
}
