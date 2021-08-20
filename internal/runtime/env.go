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

// Context represents a context in Show and Text instructions.
type Context byte

// The env type implements the native.Env interface.
type env struct {
	ctx     context.Context // context.
	globals []reflect.Value // global variables.
	print   PrintFunc       // custom print builtin.
	typeof  TypeOfFunc      // typeof function.

	// Only the filePath field can be changed after the vm has been started
	// and access to this field must be done with this mutex.
	mu       sync.Mutex
	filePath string // path of the file where the main goroutine is in.
}

func (env *env) Context() context.Context {
	return env.ctx
}

func (env *env) Exit(code int) {
	panic(exitError(code))
}

func (env *env) Fatal(v interface{}) {
	panic(&fatalError{env: env, msg: v})
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
	return env.typeof(v)
}

func typeOfFunc(v reflect.Value) reflect.Type {
	return v.Type()
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
