// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"context"
	"errors"
	"reflect"
	"sync"
)

type PrintFunc func(interface{})

// Env represents an execution environment.
type Env interface {
	Context() context.Context
	Exit(code int)
	Exited() bool
	ExitFunc(f func())
	Fatal(v interface{})
	MemoryLimiter() MemoryLimiter
	FilePath() string
	Print(args ...interface{})
	Println(args ...interface{})
	ReleaseMemory(bytes int)
	ReserveMemory(bytes int)
}

// The env type implements the Env interface.
type env struct {
	ctx     context.Context // context.
	globals []interface{}   // global variables.
	print   PrintFunc       // custom print builtin.
	memory  MemoryLimiter   // memory limiter.

	// Only exited, exits and filePath fields can be changed after the vm has
	// been started and access to these three fields must be done with this
	// mutex.
	mu       sync.Mutex
	exited   bool     // reports whether it is exited.
	exits    []func() // exit functions.
	filePath string   // path of the file where the main goroutine is in.
}

// Context returns the context of the environment.
func (env *env) Context() context.Context {
	return env.ctx
}

// Exit exits the environment with the given status code. Deferred functions
// are not run.
func (env *env) Exit(code int) {
	panic(&ExitError{env, code})
}

// Exited reports whether the environment is exited.
func (env *env) Exited() bool {
	var exited bool
	env.mu.Lock()
	exited = env.exited
	env.mu.Unlock()
	return exited
}

// ExitFunc calls f in its own goroutine after the execution of the
// environment is terminated.
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

// Fatal calls panic() with a FatalError error.
func (env *env) Fatal(v interface{}) {
	panic(&FatalError{env: env, msg: v})
}

// MemoryLimiter returns the memory limiter.
func (env *env) MemoryLimiter() MemoryLimiter {
	return env.memory
}

// FilePath can be called from a builtin function to get the absolute path of
// the file where such builtin was called. If the builtin function was not
// called by the main virtual machine goroutine, the returned value is not
// significant.
func (env *env) FilePath() string {
	env.mu.Lock()
	filePath := env.filePath
	env.mu.Unlock()
	return filePath
}

// Print calls the print built-in function with args as argument.
func (env *env) Print(args ...interface{}) {
	for _, arg := range args {
		env.doPrint(arg)
	}
}

// Println calls the println built-in function with args as argument.
func (env *env) Println(args ...interface{}) {
	for i, arg := range args {
		if i > 0 {
			env.doPrint(" ")
		}
		env.doPrint(arg)
	}
	env.doPrint("\n")
}

// ReleaseMemory releases a previously reserved memory. It panics if bytes is
// negative.
func (env *env) ReleaseMemory(bytes int) {
	if bytes < 0 {
		panic(errors.New("scriggo: release of negative bytes"))
	}
	if env.memory != nil {
		env.memory.Release(env, bytes)
	}
}

// ReserveMemory reserves memory. If the memory can not be reserved, it panics
// with an OutOfMemory error. It panics if bytes is negative.
func (env *env) ReserveMemory(bytes int) {
	if bytes < 0 {
		panic(errors.New("scriggo: reserve of negative bytes"))
	}
	if env.memory != nil {
		err := env.memory.Reserve(env, bytes)
		if err != nil {
			panic(OutOfMemoryError{env, err})
		}
	}
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
