// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

var errNilPointer = runtimeError("runtime error: invalid memory address or nil pointer dereference")

// fatalError represents a fatal error. A fatal error cannot be recovered by
// the running program.
type fatalError struct {
	env  *env
	msg  interface{}
	pos  Position
	path string
}

func (err *fatalError) Error() string {
	return "fatal error: " + panicToString(err.msg)
}

// runtimeError represents a runtime error.
type runtimeError string

func (err runtimeError) Error() string { return string(err) }
func (err runtimeError) RuntimeError() {}

// outError represents an error occurred calling the Write method of a
// template output.
type outError struct {
	err error
}

func (err outError) Error() string {
	return "out error: " + err.err.Error()
}

// errTypeAssertion returns a runtime error for a failed type assertion x.(T).
// interfaceType is the type of x, dynamicType is dynamic type of x or nil if
// x is nil and assertedType is the type T. If T is an interface type,
// missingMethod is the missing method name.
//
// The Go runtime returns a runtime.TypeAssertionError, Scriggo cannot return
// an error with this type because it has unexported fields. See also:
// https://github.com/golang/go/issues/14443
func errTypeAssertion(interfaceType, dynamicType, assertedType reflect.Type, missingMethod string) runtimeError {
	s := "interface conversion: "
	if dynamicType == nil {
		if assertedType.Kind() == reflect.Interface {
			return runtimeError(s + "interface is nil, not " + assertedType.String())
		}
		return runtimeError(s + interfaceType.String() + " is nil, not " + assertedType.String())
	}
	if missingMethod != "" {
		return runtimeError(s + dynamicType.String() + " is not " + assertedType.String() +
			": missing method " + missingMethod)
	}
	s += interfaceType.String() + " is " + dynamicType.String() + ", not " + assertedType.String()
	if dynamicType.String() != assertedType.String() {
		return runtimeError(s)
	}
	s += " (types from different "
	if dynamicType.PkgPath() == interfaceType.PkgPath() {
		return runtimeError(s + "scopes)")
	}
	return runtimeError(s + "packages)")
}

// exitError represents an exit error.
type exitError int

func (err exitError) Error() string {
	return "exit code " + strconv.Itoa(int(err))
}

// errIndexOutOfRange returns an index of range runtime error for the
// currently running virtual machine instruction.
func (vm *VM) errIndexOutOfRange() runtimeError {
	in := vm.fn.Body[vm.pc-1]
	var index, length int
	switch in.Op {
	case OpAddr, OpIndex, -OpIndex, OpIndexRef, -OpIndexRef:
		index = int(vm.intk(in.B, in.Op < 0))
		length = vm.general(in.A).Len()
	case OpIndexString, -OpIndexString:
		index = int(vm.intk(in.B, in.Op < 0))
		length = len(vm.string(in.A))
	case OpSetSlice, -OpSetSlice:
		index = int(vm.int(in.C))
		length = vm.general(in.B).Len()
	default:
		panic("unexpected operation")
	}
	s := "runtime error: index out of range [" + strconv.Itoa(index) + "] with length " + strconv.Itoa(length)
	return runtimeError(s)
}

// newPanic returns a new *PanicError with the given error message.
func (vm *VM) newPanic(msg interface{}) *PanicError {
	return &PanicError{
		message:  msg,
		path:     vm.fn.DebugInfo[vm.pc].Path,
		position: vm.fn.DebugInfo[vm.pc].Position,
	}
}

// convertPanic converts a panic to an error.
func (vm *VM) convertPanic(msg interface{}) error {
	switch err := msg.(type) {
	case exitError:
		return err
	case outError:
		return vm.newPanic(err)
	}
	switch op := vm.fn.Body[vm.pc-1].Op; op {
	case OpAddr, OpIndex, -OpIndex, OpIndexRef, -OpIndexRef, OpSetSlice, -OpSetSlice:
		switch err := msg.(type) {
		case runtime.Error:
			if s := err.Error(); strings.HasPrefix(s, "runtime error: index out of range") {
				return vm.newPanic(runtimeError(s))
			}
		case string:
			if err == "reflect: slice index out of range" || err == "reflect: array index out of range" {
				return vm.newPanic(vm.errIndexOutOfRange())
			}
		}
	case OpAppendSlice:
		if err, ok := msg.(string); ok && err == "reflect.Append: slice overflow" {
			return vm.newPanic(runtimeError("append: out of memory"))
		}
	case OpCallIndirect:
		in := vm.fn.Body[vm.pc-1]
		v := vm.general(in.A)
		if !v.IsValid() || !v.CanInterface() {
			break
		}
		if f, ok := v.Interface().(*callable); !ok || f.fn != nil {
			break
		}
		fallthrough
	case OpCallNative:
		switch msg := msg.(type) {
		case runtimeError:
			break
		case *fatalError:
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
	case OpConvert:
		if err, ok := msg.(string); ok && strings.HasPrefix(err, "reflect: cannot convert slice with length") {
			return vm.newPanic(runtimeError("runtime error:" + err[len("reflect:"):]))
		}
	case OpDelete:
		if err, ok := msg.(runtime.Error); ok {
			if s := err.Error(); strings.HasPrefix(s, "runtime error: hash of unhashable type ") {
				return vm.newPanic(runtimeError(s))
			}
		}
	case OpDivInt, OpDiv, OpRemInt, OpRem:
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
				return vm.newPanic(runtimeError(s))
			}
		}
	case OpMakeChan, -OpMakeChan:
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
	case OpSlice, OpStringSlice:
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
	return &fatalError{msg: msg}
}

type PanicError struct {
	message    interface{}
	recovered  bool
	stackTrace []byte
	next       *PanicError
	path       string
	position   Position
}

// Error returns all currently active panics as a string.
//
// To print only the message, use the String method instead.
func (p *PanicError) Error() string {
	var s string
	for p != nil {
		s = "\n" + s
		if p.Recovered() {
			s = " [recovered]" + s
		}
		s = p.String() + s
		if p.Next() != nil {
			s = "\tpanic: " + s
		}
		p = p.Next()
	}
	return s
}

// Message returns the message.
func (p *PanicError) Message() interface{} {
	return p.message
}

// Next returns the next panic in the chain.
func (p *PanicError) Next() *PanicError {
	return p.next
}

// Recovered reports whether it has been recovered.
func (p *PanicError) Recovered() bool {
	return p.recovered
}

// String returns the message as a string.
func (p *PanicError) String() string {
	return panicToString(p.message)
}

// Path returns the path of the file that panicked.
func (p *PanicError) Path() string {
	return p.path
}

// Position returns the position.
func (p *PanicError) Position() Position {
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
		rv := reflect.ValueOf(v)
		rt := rv.Type().String()
		var s string
		switch rv.Kind() {
		case reflect.Bool:
			s = "false"
			if rv.Bool() {
				s = "true"
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			s = strconv.FormatInt(rv.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			s = strconv.FormatUint(rv.Uint(), 10)
		case reflect.Float32:
			s = strconv.FormatFloat(rv.Float(), 'e', -1, 32)
		case reflect.Float64:
			s = strconv.FormatFloat(rv.Float(), 'e', -1, 64)
		case reflect.Complex64:
			c := rv.Complex()
			s = strconv.FormatFloat(real(c), 'e', -1, 32) +
				strconv.FormatFloat(imag(c), 'e', -1, 32)
		case reflect.Complex128:
			c := rv.Complex()
			s = strconv.FormatFloat(real(c), 'e', 3, 64) +
				strconv.FormatFloat(imag(c), 'e', 3, 64)
		case reflect.String:
			s = `"` + rv.String() + `"`
		default:
			iData := reflect.ValueOf(&v).Elem().InterfaceData()
			return "(" + rt + ") (" + hex(iData[0]) + "," + hex(iData[1]) + ")"
		}
		return rt + "(" + s + ")"
	}
}

// missingMethod returns a method in iface and not in typ.
func missingMethod(typ reflect.Type, iface reflect.Type) string {
	num := iface.NumMethod()
	for i := 0; i < num; i++ {
		mi := iface.Method(i)
		mt, ok := typ.MethodByName(mi.Name)
		if !ok {
			return mi.Name
		}
		numIn := mi.Type.NumIn()
		numOut := mi.Type.NumOut()
		if mt.Type.NumIn()-1 != numIn || mt.Type.NumOut() != numOut {
			return mi.Name
		}
		for j := 0; j < numIn; j++ {
			if mt.Type.In(j+1) != mi.Type.In(j) {
				return mi.Name
			}
		}
		for j := 0; j < numOut; j++ {
			if mt.Type.Out(j) != mi.Type.Out(j) {
				return mi.Name
			}
		}
	}
	return ""
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
