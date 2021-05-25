// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package templates

import (
	"fmt"
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

// decodeRenderContext decodes a runtime.Context.
// Keep in sync with the compiler.decodeRenderContext.
func decodeRenderContext(c runtime.Context) (ast.Context, bool, bool) {
	ctx := ast.Context(c & 0b00001111)
	inURL := c&0b10000000 != 0
	isURLSet := false
	if inURL {
		isURLSet = c&0b01000000 != 0
	}
	return ctx, inURL, isURLSet
}

// checkShow type checks the show of a value of type t in context ctx.
func checkShow(t reflect.Type, ctx ast.Context) error {
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
			return fmt.Errorf("cannot show type %s as %s", t, ctx)
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
			return fmt.Errorf("cannot show type %s as HTML", t)
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
			return fmt.Errorf("cannot show type %s as CSS", t)
		}
	case ast.ContextJS:
		err := checkShowJS(t, nil)
		if err != nil {
			return err
		}
	case ast.ContextJSON:
		err := checkShowJSON(t, nil)
		if err != nil {
			return err
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
			return fmt.Errorf("cannot show type %s as Markdown", t)
		}
	default:
		panic("unexpected context")
	}
	return nil
}

// checkShowJS reports whether a type can be shown as JavaScript. It returns
// an error if the type cannot be shown.
func checkShowJS(t reflect.Type, types []reflect.Type) error {
	for _, typ := range types {
		if t == typ {
			return nil
		}
	}
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
		if err := checkShowJS(t.Elem(), append(types, t)); err != nil {
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
		te := t.Elem()
		err := checkShowJS(te, append(types, t))
		if err != nil {
			return fmt.Errorf("cannot show map with %s element as JavaScript", t.Elem())
		}
	case reflect.Ptr, reflect.UnsafePointer:
		return checkShowJS(t.Elem(), append(types, t))
	case reflect.Slice:
		if err := checkShowJS(t.Elem(), append(types, t)); err != nil {
			return fmt.Errorf("cannot show slice of %s as JavaScript", t.Elem())
		}
	case reflect.Struct:
		n := t.NumField()
		for i := 0; i < n; i++ {
			field := t.Field(i)
			if field.PkgPath == "" {
				if err := checkShowJS(field.Type, append(types, t)); err != nil {
					return fmt.Errorf("cannot show struct containing %s as JavaScript", field.Type)
				}
			}
		}
	default:
		return fmt.Errorf("cannot show type %s as JavaScript", t)
	}
	return nil
}

// checkShowJSON reports whether a type can be shown as JSON. It returns an
// error if the type cannot be shown.
func checkShowJSON(t reflect.Type, types []reflect.Type) error {
	for _, typ := range types {
		if t == typ {
			return nil
		}
	}
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
		if err := checkShowJSON(t.Elem(), append(types, t)); err != nil {
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
		err := checkShowJSON(t.Elem(), append(types, t))
		if err != nil {
			return fmt.Errorf("cannot show map with %s element as JSON", t.Elem())
		}
	case reflect.Ptr, reflect.UnsafePointer:
		return checkShowJSON(t.Elem(), append(types, t))
	case reflect.Slice:
		if err := checkShowJSON(t.Elem(), append(types, t)); err != nil {
			return fmt.Errorf("cannot show slice of %s as JSON", t.Elem())
		}
	case reflect.Struct:
		n := t.NumField()
		for i := 0; i < n; i++ {
			field := t.Field(i)
			if field.PkgPath == "" {
				if err := checkShowJSON(field.Type, append(types, t)); err != nil {
					return fmt.Errorf("cannot show struct containing %s as JSON", field.Type)
				}
			}
		}
	default:
		return fmt.Errorf("cannot show type %s as JSON", t)
	}
	return nil
}
