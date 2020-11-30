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
)

var emptyInterfaceType = reflect.TypeOf(&[]interface{}{interface{}(nil)}[0]).Elem()

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

var timeType = reflect.TypeOf(time.Time{})

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// printedAs reports whether a type can be printed in context ctx.
func printedAs(t reflect.Type, ctx ast.Context) error {
	kind := t.Kind()
	switch ctx {
	case ast.ContextText, ast.ContextTag, ast.ContextQuotedAttr, ast.ContextUnquotedAttr,
		ast.ContextCSSString, ast.ContextJSString, ast.ContextJSONString:
		switch {
		case kind == reflect.String:
		case reflect.Bool <= kind && kind <= reflect.Complex128:
		case ctx == ast.ContextCSSString && t == byteSliceType:
		case t.Implements(stringerType):
		case t.Implements(envStringerType):
		case t.Implements(errorType):
		default:
			return fmt.Errorf("type %s cannot be printed as text", t)
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
			return fmt.Errorf("type %s cannot be printed as HTML", t)
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
			return fmt.Errorf("type %s cannot be printed as CSS", t)
		}
	case ast.ContextJS:
		return printedAsJS(t)
	case ast.ContextJSON:
		return printedAsJSON(t)
	default:
		panic("unexpected context")
	}
	return nil
}

// printedAsJS reports whether a type can be printed as JavaScript. It returns
// an error if the type cannot be printed.
func printedAsJS(t reflect.Type) error {
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
		if err := printedAsJS(t.Elem()); err != nil {
			return fmt.Errorf("array of %s cannot be printed as JavaScript", t.Elem())
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
			return fmt.Errorf("map with %s key cannot be printed as JavaScript", t.Key())
		}
		err := printedAsJS(t.Elem())
		if err != nil {
			return fmt.Errorf("map with %s element cannot be printed as JavaScript", t.Elem())
		}
	case reflect.Ptr, reflect.UnsafePointer:
		return printedAsJS(t.Elem())
	case reflect.Slice:
		if err := printedAsJS(t.Elem()); err != nil {
			return fmt.Errorf("slice of %s cannot be printed as JavaScript", t.Elem())
		}
	case reflect.Struct:
		n := t.NumField()
		for i := 0; i < n; i++ {
			field := t.Field(i)
			if field.PkgPath == "" {
				if err := printedAsJS(field.Type); err != nil {
					return fmt.Errorf("struct containing %s cannot be printed as JavaScript", field.Type)
				}
			}
		}
	default:
		return fmt.Errorf("type %s cannot be printed as JavaScript", t)
	}
	return nil
}

// printedAsJSON reports whether a type can be printed as JSON. It returns an
// error if the type cannot be printed.
func printedAsJSON(t reflect.Type) error {
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
		if err := printedAsJSON(t.Elem()); err != nil {
			return fmt.Errorf("array of %s cannot be printed as JSON", t.Elem())
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
			return fmt.Errorf("map with %s key cannot be printed as JSON", t.Key())
		}
		err := printedAsJSON(t.Elem())
		if err != nil {
			return fmt.Errorf("map with %s element cannot be printed as JSON", t.Elem())
		}
	case reflect.Ptr, reflect.UnsafePointer:
		return printedAsJSON(t.Elem())
	case reflect.Slice:
		if err := printedAsJSON(t.Elem()); err != nil {
			return fmt.Errorf("slice of %s cannot be printed as JSON", t.Elem())
		}
	case reflect.Struct:
		n := t.NumField()
		for i := 0; i < n; i++ {
			field := t.Field(i)
			if field.PkgPath == "" {
				if err := printedAsJSON(field.Type); err != nil {
					return fmt.Errorf("struct containing %s cannot be printed as JSON", field.Type)
				}
			}
		}
	default:
		return fmt.Errorf("type %s cannot be printed as JSON", t)
	}
	return nil
}
