// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package native provides types to implement native variables, constants,
// functions, types and packages that can be imported or used as builtins in
// programs and templates.
package native

import (
	"context"
	"reflect"
)

type (

	// HTML is the html type in templates.
	HTML string

	// CSS is the css type in templates.
	CSS string

	// JS is the js type in templates.
	JS string

	// JSON is the json type in templates.
	JSON string

	// Markdown is the markdown type in templates.
	Markdown string
)

// Env represents an execution environment.
//
// Each execution creates an Env value. This value is passed as the first
// argument to calls to native functions and methods that have Env as the type
// of the first parameter.
type Env interface {

	// Context returns the context of the execution.
	// It is the context passed as an option for execution.
	Context() context.Context

	// Exit exits the execution with the given status code. If the code is not
	// zero, the execution returns a scriggo.ExitError error with this code.
	// Deferred functions are not called and started goroutines are not
	// terminated.
	Exit(code int)

	// Fatal exits the execution and then panics with value v. Deferred
	// functions are not called and started goroutines are not terminated.
	Fatal(v interface{})

	// FilePath returns the path, relative to the root, of the current
	// executed file. If it is not called by the main goroutine, the
	// returned value is not significant.
	FilePath() string

	// Print calls the print built-in function with args as argument.
	Print(args ...interface{})

	// Println calls the println built-in function with args as argument.
	Println(args ...interface{})

	// TypeOf is like reflect.TypeOf but if v has a Scriggo type it returns
	// its Scriggo reflect type instead of the reflect type of the proxy.
	TypeOf(v reflect.Value) reflect.Type
}

type (

	// EnvStringer is like fmt.Stringer where the String method takes an native.Env
	// parameter.
	EnvStringer interface {
		String(Env) string
	}

	// HTMLStringer is implemented by values that are not escaped in HTML context.
	HTMLStringer interface {
		HTML() HTML
	}

	// HTMLEnvStringer is like HTMLStringer where the HTML method takes a
	// native.Env parameter.
	HTMLEnvStringer interface {
		HTML(Env) HTML
	}

	// CSSStringer is implemented by values that are not escaped in CSS context.
	CSSStringer interface {
		CSS() CSS
	}

	// CSSEnvStringer is like CSSStringer where the CSS method takes an Env
	// parameter.
	CSSEnvStringer interface {
		CSS(Env) CSS
	}

	// JSStringer is implemented by values that are not escaped in JavaScript
	// context.
	JSStringer interface {
		JS() JS
	}

	// JSEnvStringer is like JSStringer where the JS method takes an Env
	// parameter.
	JSEnvStringer interface {
		JS(Env) JS
	}

	// JSONStringer is implemented by values that are not escaped in JSON
	// context.
	JSONStringer interface {
		JSON() JSON
	}

	// JSONEnvStringer is like JSONStringer where the JSON method takes an Env
	// parameter.
	JSONEnvStringer interface {
		JSON(Env) JSON
	}

	// MarkdownStringer is implemented by values that are not escaped in
	// Markdown context.
	MarkdownStringer interface {
		Markdown() Markdown
	}

	// MarkdownEnvStringer is like MarkdownStringer where the Markdown method
	// takes a native.Env parameter.
	MarkdownEnvStringer interface {
		Markdown(Env) Markdown
	}
)

// Declaration represents a declaration.
//
//  for a variable: a pointer to the value of the variable
//  for a function: the function
//  for a type: its reflect.Type value
//  for a typed constant: its value as a string, boolean or numeric value
//  for an untyped constant: an UntypedStringConst, UntypedBooleanConst or UntypedNumericConst value
//  for a package: an ImportablePackage value (used only for template globals)
//
type Declaration interface{}

// Declarations represents a set of variables, constants, functions, types and
// packages declarations and can be used for template globals and package
// declarations.
//
// The key is the declaration's name and the element is its value.
type Declarations map[string]Declaration

type (
	// UntypedStringConst represents an untyped string constant.
	UntypedStringConst string

	// UntypedBooleanConst represents an untyped boolean constant.
	UntypedBooleanConst bool

	// UntypedNumericConst represents an untyped numeric constant.
	UntypedNumericConst string
)
