// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"context"
	"reflect"
)

// Format types.
type (
	HTML     string // the html type in templates.
	CSS      string // the css type in templates.
	JS       string // the js type in templates.
	JSON     string // the json type in templates.
	Markdown string // the markdown type in templates.
)

type (

	// EnvStringer is like fmt.Stringer where the String method takes an types.Env
	// parameter.
	EnvStringer interface {
		String(Env) string
	}

	// HTMLStringer is implemented by values that are not escaped in HTML context.
	HTMLStringer interface {
		HTML() HTML
	}

	// HTMLEnvStringer is like HTMLStringer where the HTML method takes a
	// types.Env parameter.
	HTMLEnvStringer interface {
		HTML(Env) HTML
	}

	// CSSStringer is implemented by values that are not escaped in CSS context.
	CSSStringer interface {
		CSS() CSS
	}

	// CSSEnvStringer is like CSSStringer where the CSS method takes an types.Env
	// parameter.
	CSSEnvStringer interface {
		CSS(Env) CSS
	}

	// JSStringer is implemented by values that are not escaped in JavaScript
	// context.
	JSStringer interface {
		JS() JS
	}

	// JSEnvStringer is like JSStringer where the JS method takes an types.Env
	// parameter.
	JSEnvStringer interface {
		JS(Env) JS
	}

	// JSONStringer is implemented by values that are not escaped in JSON context.
	JSONStringer interface {
		JSON() JSON
	}

	// JSONEnvStringer is like JSONStringer where the JSON method takes an types.Env
	// parameter.
	JSONEnvStringer interface {
		JSON(Env) JSON
	}

	// MarkdownStringer is implemented by values that are not escaped in Markdown
	// context.
	MarkdownStringer interface {
		Markdown() Markdown
	}

	// MarkdownEnvStringer is like MarkdownStringer where the Markdown method
	// takes an types.Env parameter.
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
type Env interface {

	// Context returns the context of the environment.
	Context() context.Context

	// Exit exits the environment with the given status code. Deferred functions
	// are not run.
	Exit(code int)

	// Exited reports whether the environment is exited.
	Exited() bool

	// ExitFunc calls f in its own goroutine after the execution of the
	// environment is terminated.
	ExitFunc(f func())

	// Fatal calls panic() with a FatalError error.
	Fatal(v interface{})

	// FilePath can be called from a global function to get the absolute path
	// of the file where such global was called. If the global function was
	// not called by the main virtual machine goroutine, the returned value is
	// not significant.
	FilePath() string

	// Print calls the print built-in function with args as argument.
	Print(args ...interface{})

	// Println calls the println built-in function with args as argument.
	Println(args ...interface{})

	// TypeOf is like reflect.TypeOf but if v has a Scriggo type it returns
	// its Scriggo reflect type instead of the reflect type of the proxy.
	TypeOf(v reflect.Value) reflect.Type
}
