// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"reflect"
	"runtime"
	"strconv"
)

// FatalError represents a fatal error. A fatal error cannot be recovered by
// the running program.
type FatalError struct {
	env *Env
	msg interface{}
}

func (err FatalError) Error() string {
	return "fatal error: " + panicToString(err.msg)
}

// runtimeError represents a runtime error.
type runtimeError string

func (err runtimeError) Error() string { return string(err) }
func (err runtimeError) RuntimeError() {}

type TypeAssertionError struct {
	interfac      reflect.Type
	concrete      reflect.Type
	asserted      reflect.Type
	missingMethod string
}

func (e TypeAssertionError) Error() string {
	s := "interface conversion: "
	if e.concrete == nil {
		return s + e.interfac.String() + " is nil, not " + e.asserted.String()
	}
	if e.missingMethod != "" {
		return s + e.concrete.String() + " is not " + e.asserted.String() +
			": missing method " + e.missingMethod
	}
	s += e.interfac.String() + " is " + e.concrete.String() + ", not " + e.asserted.String()
	if e.concrete.String() != e.asserted.String() {
		return s
	}
	s += " (types from different "
	if e.concrete.PkgPath() == e.interfac.PkgPath() {
		return s + "scopes)"
	}
	return s + "packages)"
}

func (e TypeAssertionError) RuntimeError() {}

// OutOfTimeError represents a runtime out of time error.
type OutOfTimeError struct {
	env *Env
}

func (err OutOfTimeError) Error() string {
	return "runtime error: out of time: " + err.env.ctx.Err().Error()
}

func (err OutOfTimeError) RuntimeError() {}

// OutOfMemoryError represents a runtime out of memory error.
type OutOfMemoryError struct {
	env *Env
}

func (err OutOfMemoryError) Error() string {
	return "runtime error: out of memory: cannot allocate " +
		strconv.Itoa(-err.env.freeMemory) + " bytes"
}

func (err OutOfMemoryError) RuntimeError() {}

// convertInternalError converts an internal error, from a panic, to a Go
// error.
func (vm *VM) convertInternalError(msg interface{}) error {
	var op Operation
	if vm.pc > 1 && vm.fn.Body[vm.pc-2].Op == OpMakeSlice {
		op = OpMakeSlice
	} else {
		op = vm.fn.Body[vm.pc-1].Op
	}
	switch op {
	case OpAddr, OpIndex, -OpIndex, OpSetSlice, -OpSetSlice:
		switch err := msg.(type) {
		case runtime.Error:
			if err.Error() == "runtime error: index out of range" {
				return runtimeError("runtime error: index out of range")
			}
		case string:
			if err == "reflect: slice index out of range" {
				return runtimeError("runtime error: index out of range")
			}
		}
	case OpAppendSlice:
		if err, ok := msg.(string); ok && err == "reflect.Append: slice overflow" {
			return runtimeError("append: out of memory")
		}
	case OpClose:
		if err, ok := msg.(runtime.Error); ok && err.Error() == "close of closed channel" {
			return runtimeError("close of closed channel")
		}
	case OpIndexString, -OpIndexString:
		if err, ok := msg.(runtime.Error); ok && err.Error() == "runtime error: index out of range" {
			return runtimeError("runtime error: index out of range")
		}
	case OpMakeChan:
		if err, ok := msg.(string); ok && err == "reflect.MakeChan: negative buffer size" {
			return runtimeError("makechan: size out of range")
		}
	case OpMakeSlice:
		if err, ok := msg.(string); ok {
			switch err {
			case "reflect.MakeSlice: negative len":
				return runtimeError("runtime error: makeslice: len out of range")
			case "reflect.MakeSlice: negative cap", "reflect.MakeSlice: len > cap":
				return runtimeError("runtime error: makeslice: cap out of range")
			}
		}
	case OpSend, -OpSend:
		if err, ok := msg.(string); ok {
			switch err {
			case "close of nil channel", "send on closed channel":
				return runtimeError(err)
			}
		}
	}
	return nil
}
