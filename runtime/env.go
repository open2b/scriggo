// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"context"
	"reflect"
	"sync"
)

type PrintFunc func(interface{})

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

	// TypeOf returns the reflect type of a reflect value. TypeOf is like the
	// reflect.TypeOf function, but for values with a Scriggo type it returns
	// its Scriggo reflect type instead of the reflect type of the proxy.
	// TypeOf returns nil if v is the zero reflect.Value.
	TypeOf(v reflect.Value) reflect.Type
}

// The env type implements the Env interface.
type env struct {
	ctx     context.Context // context.
	globals []interface{}   // global variables.
	print   PrintFunc       // custom print builtin.
	typeof  TypeOfFunc      // typeof function.

	// Only exited, exits and filePath fields can be changed after the vm has
	// been started and access to these three fields must be done with this
	// mutex.
	mu       sync.Mutex
	exited   bool     // reports whether it is exited.
	exits    []func() // exit functions.
	filePath string   // path of the file where the main goroutine is in.
}

func (env *env) Context() context.Context {
	return env.ctx
}

func (env *env) Exit(code int) {
	panic(&ExitError{env, code})
}

func (env *env) Exited() bool {
	var exited bool
	env.mu.Lock()
	exited = env.exited
	env.mu.Unlock()
	return exited
}

func (env *env) ExitFunc(f func()) {
	env.mu.Lock()
	if env.exited {
		go f()
	} else {
		env.exits = append(env.exits, f)
	}
	env.mu.Unlock()
	return
}

func (env *env) Fatal(v interface{}) {
	panic(&FatalError{env: env, msg: v})
}

func (env *env) FilePath() string {
	env.mu.Lock()
	filePath := env.filePath
	env.mu.Unlock()
	return filePath
}

func (env *env) Print(args ...interface{}) {
	for _, arg := range args {
		env.doPrint(arg)
	}
}

func (env *env) Println(args ...interface{}) {
	for i, arg := range args {
		if i > 0 {
			env.doPrint(" ")
		}
		env.doPrint(arg)
	}
	env.doPrint("\n")
}

func (env *env) TypeOf(v reflect.Value) reflect.Type {
	if v.IsValid() {
		if env.typeof == nil {
			return v.Type()
		}
		return env.typeof(v)
	}
	return nil
}

func (env *env) doPrint(arg interface{}) {
	if env.print != nil {
		env.print(arg)
		return
	}
	r := reflect.ValueOf(arg)
	switch r.Kind() {
	case reflect.Invalid, reflect.Array, reflect.Func, reflect.Struct:
		print(hex(reflect.ValueOf(&arg).Elem().InterfaceData()[1]))
	case reflect.Bool:
		print(r.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		print(r.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		print(r.Uint())
	case reflect.Float32, reflect.Float64:
		print(r.Float())
	case reflect.Complex64, reflect.Complex128:
		print(r.Complex())
	case reflect.Chan, reflect.Map, reflect.UnsafePointer:
		print(hex(r.Pointer()))
	case reflect.Interface, reflect.Ptr:
		print(arg)
	case reflect.Slice:
		print("[", r.Len(), "/", r.Cap(), "]", hex(r.Pointer()))
	case reflect.String:
		print(r.String())
	}
}

func (env *env) exit() {
	env.mu.Lock()
	if !env.exited {
		for _, f := range env.exits {
			go f()
		}
		env.exits = nil
		env.exited = true
	}
	env.mu.Unlock()
}
