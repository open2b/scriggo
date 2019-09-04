// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"reflect"
	"runtime"
	"scriggo/ast"
	"strconv"
	"strings"
)

var go112 bool

func init() {
	go112 = strings.HasPrefix(runtime.Version(), "go1.12")
}

var errNilPointer = runtimeError("runtime error: invalid memory address or nil pointer dereference")

// FatalError represents a fatal error. A fatal error cannot be recovered by
// the running program.
type FatalError struct {
	env  *Env
	msg  interface{}
	pos  *ast.Position
	path string
}

func (err *FatalError) Error() string {
	return "fatal error: " + panicToString(err.msg)
}

// runtimeError represents a runtime error.
type runtimeError string

func (err runtimeError) Error() string { return string(err) }
func (err runtimeError) RuntimeError() {}

// errTypeAssertion returns a runtime error for a failed type assertion. The
// Go runtime returns a runtime.TypeAssertionError, Scriggo cannot return an
// error with this type because it has unexported fields.
//
// See also https://github.com/golang/go/issues/14443
func errTypeAssertion(interfac, concrete, asserted reflect.Type, missingMethod string) runtimeError {
	s := "interface conversion: "
	if concrete == nil {
		return runtimeError(s + interfac.String() + " is nil, not " + asserted.String())
	}
	if missingMethod != "" {
		return runtimeError(s + concrete.String() + " is not " + asserted.String() +
			": missing method " + missingMethod)
	}
	s += interfac.String() + " is " + concrete.String() + ", not " + asserted.String()
	if concrete.String() != asserted.String() {
		return runtimeError(s)
	}
	s += " (types from different "
	if concrete.PkgPath() == interfac.PkgPath() {
		return runtimeError(s + "scopes)")
	}
	return runtimeError(s + "packages)")
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

// errIndexOutOfRange returns an index of range runtime error for the
// currently running virtual machine instruction.
func (vm *VM) errIndexOutOfRange() runtimeError {
	in := vm.fn.Body[vm.pc-1]
	s := "runtime error: index out of range ["
	switch in.Op {
	case OpAddr, OpIndex, -OpIndex, OpIndexString, -OpIndexString:
		s += strconv.Itoa(int(vm.intk(in.B, in.Op < 0)))
	case OpSetSlice, -OpSetSlice:
		s += strconv.Itoa(int(vm.int(in.C)))
	default:
		panic("unexpected operation")
	}
	s += "] with length " + strconv.Itoa(reflect.ValueOf(vm.general(in.A)).Len())
	return runtimeError(s)
}

// newPanic returns a new *Panic with the given error message.
func (vm *VM) newPanic(msg interface{}) *Panic {
	return &Panic{
		message:  msg,
		path:     vm.fn.Paths[vm.pc],
		position: vm.fn.Positions[vm.pc],
	}
}

// convertPanic converts a panic to an error.
func (vm *VM) convertPanic(msg interface{}) error {
	switch vm.fn.Body[vm.pc-1].Op {
	case OpAddr, OpIndex, -OpIndex, OpSetSlice, -OpSetSlice:
		switch err := msg.(type) {
		case runtime.Error:
			if s := err.Error(); strings.HasPrefix(s, "runtime error: index out of range") {
				if go112 {
					return vm.newPanic(vm.errIndexOutOfRange())
				}
				return vm.newPanic(runtimeError(s))
			}
		case string:
			if err == "reflect: slice index out of range" {
				return vm.newPanic(vm.errIndexOutOfRange())
			}
		}
	case OpAppendSlice:
		if err, ok := msg.(string); ok && err == "reflect.Append: slice overflow" {
			return vm.newPanic(runtimeError("append: out of memory"))
		}
	case OpCallIndirect:
		in := vm.fn.Body[vm.pc-1]
		if f, ok := vm.general(in.A).(*callable); ok && f.fn != nil {
			break
		}
		fallthrough
	case OpCallPredefined:
		switch msg := msg.(type) {
		case runtimeError:
			break
		case *FatalError:
			// TODO: check env.
			return msg
		case runtime.Error:
			// TODO: check env.
			break
		default:
			return vm.newPanic(msg)
		}
	case OpClose:
		if err, ok := msg.(runtime.Error); ok {
			switch s := err.Error(); s {
			case "close of closed channel", "close of nil channel":
				return vm.newPanic(runtimeError(s))
			}
		}
	case OpDivInt8, OpDivInt16, OpDivInt32, OpDivInt64, OpDivFloat32, OpDivFloat64, OpRemInt8,
		OpRemInt16, OpRemInt32, OpRemInt64, OpRemUint8, OpRemUint16, OpRemUint32, OpRemUint64:
		if err, ok := msg.(runtime.Error); ok {
			if s := err.Error(); s == "runtime error: integer divide by zero" {
				return vm.newPanic(runtimeError(s))
			}
		}
	case OpIf, -OpIf:
		if err, ok := msg.(runtime.Error); ok {
			if s := err.Error(); strings.HasPrefix(s, "runtime error: comparing uncomparable type ") {
				return vm.newPanic(runtimeError(s))
			}
		}
	case OpIndexString, -OpIndexString:
		if err, ok := msg.(runtime.Error); ok {
			if s := err.Error(); strings.HasPrefix(s, "runtime error: index out of range") {
				if go112 {
					return vm.newPanic(vm.errIndexOutOfRange())
				}
				return vm.newPanic(runtimeError(s))
			}
		}
	case OpMakeChan:
		if err, ok := msg.(string); ok && err == "reflect.MakeChan: negative buffer size" {
			return vm.newPanic(runtimeError("makechan: size out of range"))
		}
	case OpMakeSlice:
		if err, ok := msg.(string); ok {
			switch err {
			case "reflect.MakeSlice: negative len":
				return vm.newPanic(runtimeError("runtime error: makeslice: len out of range"))
			case "reflect.MakeSlice: negative cap", "reflect.MakeSlice: len > cap":
				return vm.newPanic(runtimeError("runtime error: makeslice: cap out of range"))
			}
		}
	case OpPanic:
		return vm.newPanic(msg)
	case OpSend, -OpSend:
		switch err := msg.(type) {
		case runtime.Error:
			if s := err.Error(); s == "send on closed channel" {
				return vm.newPanic(runtimeError(s))
			}
		case string:
			if err == "close of nil channel" {
				return vm.newPanic(runtimeError(err))
			}
		}
	case OpSetMap, -OpSetMap:
		if err, ok := msg.(runtime.Error); ok {
			s := err.Error()
			if s == "assignment to entry in nil map" ||
				strings.HasPrefix(s, "runtime error: hash of unhashable type ") {
				return vm.newPanic(runtimeError(s))
			}
		}
	case OpSlice, OpSliceString:
		// https://github.com/open2b/scriggo/issues/321
		switch err := msg.(type) {
		case runtime.Error:
			if s := err.Error(); strings.HasPrefix(s, "runtime error: slice bounds out of range") {
				return vm.newPanic(runtimeError("runtime error: slice bounds out of range"))
			}
		case string:
			if err == "reflect.Value.Slice3: slice index out of bounds" {
				return vm.newPanic(runtimeError("runtime error: slice bounds out of range"))
			}
		}
	}
	if _, ok := msg.(runtimeError); ok {
		return vm.newPanic(msg)
	}
	return &FatalError{msg: msg}
}

type Panic struct {
	message    interface{}
	recovered  bool
	stackTrace []byte
	next       *Panic
	path       string
	position   *ast.Position
}

func (p *Panic) Error() string {
	b := make([]byte, 0, 100+len(p.stackTrace))
	//b = append(b, sprint(err.message)...) // TODO(marco): rewrite.
	b = append(b, "\n\n"...)
	b = append(b, p.stackTrace...)
	return string(b)
}

// Message returns the message.
func (p *Panic) Message() interface{} {
	return p.message
}

// Next returns the next panic in the chain.
func (p *Panic) Next() *Panic {
	return p.next
}

// Recovered reports whether it has been recovered.
func (p *Panic) Recovered() bool {
	return p.recovered
}

// String returns the message as a string.
func (p *Panic) String() string {
	return panicToString(p.message)
}

// Path returns the path of the file that panicked.
func (p *Panic) Path() string {
	return p.path
}

// Position returns the position.
func (p *Panic) Position() *ast.Position {
	return p.position
}

func panicToString(msg interface{}) string {
	switch v := msg.(type) {
	case nil:
		return "nil"
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.Itoa(int(v))
	case int16:
		return strconv.Itoa(int(v))
	case int32:
		return strconv.Itoa(int(v))
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case uintptr:
		return strconv.FormatUint(uint64(v), 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'e', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'e', -1, 64)
	case complex64:
		return "(" + strconv.FormatFloat(float64(real(v)), 'e', -1, 32) +
			strconv.FormatFloat(float64(imag(v)), 'e', -1, 32) + ")"
	case complex128:
		return "(" + strconv.FormatFloat(real(v), 'e', 3, 64) +
			strconv.FormatFloat(imag(v), 'e', 3, 64) + ")"
	case string:
		return v
	case error:
		return v.Error()
	case stringer:
		return v.String()
	default:
		typ := reflect.TypeOf(v).String()
		iData := reflect.ValueOf(&v).Elem().InterfaceData()
		return "(" + typ + ") (" + hex(iData[0]) + "," + hex(iData[1]) + ")"
	}
}

type stringer interface {
	String() string
}

func hex(p uintptr) string {
	i := 20
	h := [20]byte{}
	for {
		i--
		h[i] = "0123456789abcdef"[p%16]
		p = p / 16
		if p == 0 {
			break
		}
	}
	h[i-1] = 'x'
	h[i-2] = '0'
	return string(h[i-2:])
}
