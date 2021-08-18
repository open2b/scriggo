// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package native provides types to implement native values, constants,
// functions, types and packages that can be imported or used as builtins in
// programs and templates.
package native

import (
	"context"
	"reflect"
)

// Declarations contains variable, constant, function, type and package
// declarations.
type Declarations map[string]interface{}

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
	// takes an native.Env parameter.
	MarkdownEnvStringer interface {
		Markdown(Env) Markdown
	}
)

type (

	// UntypedStringConst represents an untyped string constant.
	UntypedStringConst string

	// UntypedBooleanConst represents an untyped boolean constant.
	UntypedBooleanConst bool

	// UntypedNumericConst represents an untyped numeric constant.
	UntypedNumericConst string
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

	// Exit exits the execution with the given status code.
	// Deferred functions are not run.
	Exit(code int)

	// Exited reports whether the execution is terminated.
	Exited() bool

	// ExitFunc calls f in its own goroutine after the execution is
	// terminated.
	ExitFunc(f func())

	// Fatal calls the panic built-in function with a FatalError error.
	Fatal(v interface{})

	// FilePath returns the absolute path of the current executed file. If it
	// is not called by the main goroutine, the returned value is not
	// significant.
	FilePath() string

	// Print calls the print built-in function with args as argument.
	Print(args ...interface{})

	// Println calls the println built-in function with args as argument.
	Println(args ...interface{})

	// TypeOf is like reflect.TypeOf but if v has a Scriggo type it returns
	// its Scriggo reflect type instead of the reflect type of the proxy.
	TypeOf(v reflect.Value) reflect.Type
}
