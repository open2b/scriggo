// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package env

import (
	"context"
	"reflect"
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
