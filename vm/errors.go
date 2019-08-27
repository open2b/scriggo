// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"reflect"
	"strconv"
)

// runtimeError represents a runtime error.
type runtimeError string

func (err runtimeError) Error() string { return "runtime error: " + string(err) }
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

// runtimeIndex returns the v's i'th element. If i is out of range, it panics
// with a runtimeError error.
func runtimeIndex(v reflect.Value, i int) reflect.Value {
	defer func() {
		if err := recover(); err != nil {
			if _, ok := err.(string); ok {
				err = runtimeError("index out of range")
			}
			panic(err)
		}
	}()
	return v.Index(i)
}

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
