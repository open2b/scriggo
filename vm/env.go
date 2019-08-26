// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"context"
	"reflect"
	"sync"
)

type TraceFunc func(fn *Function, pc uint32, regs Registers)
type PrintFunc func(interface{})

// Env represents an execution environment.
type Env struct {

	// Only freeMemory, exited and exits fields can be changed after the vm
	// has been started and access to these three fields must be done with
	// this mutex.
	mu sync.Mutex

	ctx         context.Context // context.
	globals     []interface{}   // global variables.
	trace       TraceFunc       // trace function.
	print       PrintFunc       // custom print builtin.
	freeMemory  int             // free memory.
	limitMemory bool            // reports whether memory is limited.
	dontPanic   bool            // don't panic.
	exited      bool            // reports whether it is exited.
	exits       []func()        // exit functions.

}

// Alloc allocates, or if bytes is negative, deallocates memory. Alloc does
// nothing if there is no memory limit. If there is no free memory, Alloc
// panics with the OutOfMemory error.
func (env *Env) Alloc(bytes int) {
	if env.limitMemory {
		env.mu.Lock()
		free := env.freeMemory
		if free >= 0 {
			free -= int(bytes)
			env.freeMemory = free
		}
		env.mu.Unlock()
		if free < 0 {
			panic(errOutOfMemory)
		}
	}
}

// Context returns the context of the environment.
func (env *Env) Context() context.Context {
	return env.ctx
}

// Exited reports whether the environment is exited.
func (env *Env) Exited() bool {
	var exited bool
	env.mu.Lock()
	exited = env.exited
	env.mu.Unlock()
	return exited
}

// ExitFunc calls f in its own goroutine after the execution of the
// environment is terminated.
func (env *Env) ExitFunc(f func()) {
	env.mu.Lock()
	if env.exited {
		go f()
	} else {
		env.exits = append(env.exits, f)
	}
	env.mu.Unlock()
	return
}

// FreeMemory returns the current free memory in bytes and true if the maximum
// memory has been limited. Otherwise returns zero and false.
//
// A negative value means that an out of memory error has been occurred and in
// this case bytes represents the number of bytes that were not available.
func (env *Env) FreeMemory() (bytes int, limitedMemory bool) {
	if env.limitMemory {
		env.mu.Lock()
		free := env.freeMemory
		env.mu.Unlock()
		return free, true
	}
	return 0, false
}

// Print calls the print built-in function with args as argument.
func (env *Env) Print(args ...interface{}) {
	for _, arg := range args {
		env.doPrint(arg)
	}
}

// Println calls the println built-in function with args as argument.
func (env *Env) Println(args ...interface{}) {
	for i, arg := range args {
		if i > 0 {
			env.doPrint(" ")
		}
		env.doPrint(arg)
	}
	env.doPrint("\n")
}

func (env *Env) doPrint(arg interface{}) {
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
