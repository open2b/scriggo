// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"context"
	"reflect"
	"strconv"
	"sync"
)

const NoVariadicArgs = -1
const CurrentFunction = -1

const stackSize = 512

var envType = reflect.TypeOf(&Env{})
var sliceByteType = reflect.TypeOf([]byte{})
var emptyInterfaceType = reflect.TypeOf(&[]interface{}{nil}[0]).Elem()
var emptyInterfaceNil = reflect.ValueOf(&[]interface{}{nil}[0]).Elem()

type StackShift [4]int8

type Instruction struct {
	Op      Operation
	A, B, C int8
}

func decodeInt16(a, b int8) int16 {
	return int16(int(a)<<8 | int(uint8(b)))
}

func decodeUint24(a, b, c int8) uint32 {
	return uint32(uint8(a))<<16 | uint32(uint8(b))<<8 | uint32(uint8(c))
}

// Sync with compiler.decodeFieldIndex.
func decodeFieldIndex(i int64) []int {
	if i <= 255 {
		return []int{int(i)}
	}
	s := []int{
		int(uint8(i >> 0)),
		int(uint8(i >> 8)),
		int(uint8(i >> 16)),
		int(uint8(i >> 24)),
		int(uint8(i >> 32)),
		int(uint8(i >> 40)),
		int(uint8(i >> 48)),
		int(uint8(i >> 56)),
	}
	ns := []int{}
	for i := 0; i < len(s); i++ {
		if i == len(s)-1 {
			ns = append(ns, s[i])
		} else {
			if s[i] > 0 {
				ns = append(ns, s[i]-1)
			}
		}
	}
	return ns
}

// VM represents a Scriggo virtual machine.
type VM struct {
	fp       [4]uint32            // frame pointers.
	st       [4]uint32            // stack tops.
	pc       uint32               // program counter.
	ok       bool                 // ok flag.
	regs     registers            // registers.
	fn       *Function            // running function.
	vars     []interface{}        // global and closure variables.
	env      *Env                 // execution environment.
	envArg   reflect.Value        // execution environment as argument.
	calls    []callFrame          // call stack frame.
	cases    []reflect.SelectCase // select cases.
	done     <-chan struct{}      // done.
	doneCase reflect.SelectCase   // done, as reflect case.
	panics   []Panic              // panics.
}

// New returns a new virtual machine.
func New() *VM {
	return create(&Env{})
}

// Env returns the execution environment of vm.
func (vm *VM) Env() *Env {
	return vm.env
}

// Reset resets a virtual machine so that it is ready for a new call to Run.
func (vm *VM) Reset() {
	vm.fp = [4]uint32{0, 0, 0, 0}
	vm.st[0] = uint32(len(vm.regs.int))
	vm.st[1] = uint32(len(vm.regs.float))
	vm.st[2] = uint32(len(vm.regs.string))
	vm.st[3] = uint32(len(vm.regs.general))
	vm.pc = 0
	vm.ok = false
	vm.fn = nil
	vm.vars = nil
	vm.env = &Env{}
	vm.envArg = reflect.ValueOf(vm.env)
	if vm.calls != nil {
		vm.calls = vm.calls[:0]
	}
	if vm.cases != nil {
		vm.cases = vm.cases[:0]
	}
	vm.done = nil
	vm.doneCase = reflect.SelectCase{}
	if vm.panics != nil {
		vm.panics = vm.panics[:0]
	}
}

// Run starts the execution of the function fn with the given global variables
// and waits for it to complete.
//
// During the execution if a panic occurs and has not been recovered, by
// default Run panics with the panic message.
//
// If a maximum available memory has been set and the memory is exhausted,
// Run returns immediately with the an OutOfMemoryError error.
//
// If a context has been set and the context is canceled, Run returns
// as soon as possible with the error returned by the Err method of the
// context.
func (vm *VM) Run(fn *Function, globals []interface{}) (code int, err error) {
	vm.env.globals = globals
	return vm.runFunc(fn, globals)
}

// SetContext sets the context.
//
// SetContext must not be called after vm has been started.
func (vm *VM) SetContext(ctx context.Context) {
	vm.env.ctx = ctx
	if ctx != nil {
		if done := ctx.Done(); done != nil {
			vm.done = done
			vm.doneCase = reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(done),
			}
			return
		}
	}
	vm.done = nil
	vm.doneCase = reflect.SelectCase{}
}

// SetMaxMemory sets the maximum available memory. Set bytes to zero or
// negative for no limits.
//
// SetMaxMemory must not be called after vm has been started.
func (vm *VM) SetMaxMemory(bytes int) {
	if bytes > 0 {
		vm.env.limitMemory = true
		vm.env.freeMemory = bytes
	} else {
		vm.env.limitMemory = false
		vm.env.freeMemory = 0
	}
}

// SetPrint sets the "print" builtin function.
//
// SetPrint must not be called after vm has been started.
func (vm *VM) SetPrint(p func(interface{})) {
	vm.env.print = p
}

// SetTraceFunc sets the trace stack function.
//
// SetTraceFunc must not be called after vm has been started.
func (vm *VM) SetTraceFunc(fn TraceFunc) {
	vm.env.trace = fn
}

// Stack returns the current stack trace.
func (vm *VM) Stack(buf []byte, all bool) int {
	// TODO(marco): implement all == true
	if len(buf) == 0 {
		return 0
	}
	b := buf[0:0:len(buf)]
	write := func(s string) {
		n := copy(b[len(b):cap(b)], s)
		b = b[:len(b)+n]
	}
	write("scriggo goroutine 1 [running]:")
	size := len(vm.calls)
	for i := size; i >= 0; i-- {
		var fn *Function
		var ppc uint32
		if i == size {
			fn = vm.fn
			ppc = vm.pc - 1
		} else {
			call := vm.calls[i]
			fn = call.cl.fn
			if call.status == tailed {
				ppc = call.pc - 1
			} else {
				ppc = call.pc - 2
			}
		}
		write("\n")
		write(packageName(fn.Pkg))
		write(".")
		write(fn.Name)
		write("()\n\t")
		if fn.File != "" {
			write(fn.File)
		} else {
			write("???")
		}
		write(":")
		if line, ok := fn.Lines[ppc]; ok {
			write(strconv.Itoa(line))
		} else {
			write("???")
		}
		if len(b) == len(buf) {
			break
		}
	}
	return len(b)
}

func (vm *VM) alloc() {
	if !vm.env.limitMemory {
		return
	}
	in := vm.fn.Body[vm.pc]
	op, a, b, c := in.Op, in.A, in.B, in.C
	k := op < 0
	if k {
		op = -op
	}
	var bytes int
	switch op {
	case OpAppend:
		var elemSize int
		var sc, sl int
		switch s := vm.general(c).(type) {
		case []int:
			elemSize = 8
			sl, sc = len(s), cap(s)
		case []byte:
			elemSize = 1
			sl, sc = len(s), cap(s)
		case []rune:
			elemSize = 4
			sl, sc = len(s), cap(s)
		case []float64:
			elemSize = 8
			sl, sc = len(s), cap(s)
		case []string:
			elemSize = 16
			sl, sc = len(s), cap(s)
		case []interface{}:
			elemSize = 16
			sl, sc = len(s), cap(s)
		default:
			rs := reflect.ValueOf(s)
			elemSize = int(rs.Type().Size())
			sl, sc = rs.Len(), rs.Cap()
		}
		l := int(b)
		if l > sc-sl {
			if sl+l < 0 {
				panic(OutOfMemoryError{vm.env})
			}
			capacity := appendCap(sc, sl, sl+l)
			bytes = capacity * elemSize
			if bytes/capacity != elemSize {
				panic(OutOfMemoryError{vm.env})
			}
		}
	case OpAppendSlice:
		var elemSize int
		var l, sc, sl int
		src := vm.general(a)
		switch s := vm.general(c).(type) {
		case []int:
			elemSize = 8
			l, sl, sc = len(src.([]int)), len(s), cap(s)
		case []byte:
			elemSize = 1
			l, sl, sc = len(src.([]byte)), len(s), cap(s)
		case []rune:
			elemSize = 4
			l, sl, sc = len(src.([]rune)), len(s), cap(s)
		case []float64:
			elemSize = 8
			l, sl, sc = len(src.([]float64)), len(s), cap(s)
		case []string:
			elemSize = 16
			l, sl, sc = len(src.([]string)), len(s), cap(s)
		case []interface{}:
			elemSize = 16
			l, sl, sc = len(src.([]interface{})), len(s), cap(s)
		default:
			sl, sc = reflect.ValueOf(src).Len(), reflect.ValueOf(s).Cap()
		}
		if l > sc-sl {
			nl := sl + l
			if nl < sl {
				panic(OutOfMemoryError{vm.env})
			}
			capacity := appendCap(sc, sl, nl)
			bytes = capacity * elemSize
			if bytes/capacity != elemSize {
				panic(OutOfMemoryError{vm.env})
			}
		}
	case OpConvertGeneral: // TODO(marco): implement in the builder.
		t := vm.fn.Types[uint8(b)]
		switch t.Kind() {
		case reflect.Func:
			call := vm.general(a).(*callable)
			if !call.value.IsValid() {
				// Approximated size based on makeFuncImpl in
				// https://golang.org/src/reflect/makefunc.go
				bytes = 100
			}
		default:
			bytes = int(reflect.TypeOf(vm.general(a)).Size())
		}
	case OpConvertString: // TODO(marco): implement in the builder.
		t := vm.fn.Types[uint8(b)]
		if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Int32 {
			length := len([]rune(vm.string(a)))
			bytes = length * 4
			if bytes/4 != length {
				panic(OutOfMemoryError{vm.env})
			}
		} else {
			bytes = len(vm.string(a))
		}
	case OpConcat:
		aLen := len(vm.string(a))
		bLen := len(vm.string(b))
		bytes = aLen + bLen
		if bytes < aLen {
			panic(OutOfMemoryError{vm.env})
		}
	case OpMakeChan:
		typ := vm.fn.Types[uint8(a)]
		capacity := int(vm.intk(b, k))
		ts := int(typ.Size())
		bytes = ts * capacity
		if bytes/ts != capacity {
			panic(OutOfMemoryError{vm.env})
		}
		bytes += 10 * 8
		if bytes < 0 {
			panic(OutOfMemoryError{vm.env})
		}
	case OpMakeMap:
		// The size is approximated. The actual size depend on the type and
		// architecture.
		n := int(vm.int(b))
		bytes = 50 * n
		if bytes/50 != n {
			panic(OutOfMemoryError{vm.env})
		}
		bytes += 24
		if bytes < 0 {
			panic(OutOfMemoryError{vm.env})
		}
	case OpMakeSlice:
		typ := vm.fn.Types[uint8(a)]
		capacity := int(vm.intk(vm.fn.Body[vm.pc+1].B, k))
		ts := int(typ.Elem().Size())
		bytes = ts * capacity
		if bytes/ts != capacity {
			panic(OutOfMemoryError{vm.env})
		}
		bytes += 24
		if bytes < 0 {
			panic(OutOfMemoryError{vm.env})
		}
	case OpNew:
		t := vm.fn.Types[uint8(b)]
		bytes = int(t.Size())
	case OpSetMap:
		m := vm.general(a)
		t := reflect.TypeOf(m)
		kSize := int(t.Key().Size())
		eSize := int(t.Elem().Size())
		bytes = kSize + eSize
		if bytes < 0 {
			panic(OutOfMemoryError{vm.env})
		}
	}
	if bytes != 0 {
		var free int
		vm.env.mu.Lock()
		free = vm.env.freeMemory
		if free >= 0 {
			free -= bytes
			vm.env.freeMemory = free
		}
		vm.env.mu.Unlock()
		if free < 0 {
			panic(OutOfMemoryError{vm.env})
		}
	}
	return
}

// callPredefined calls a predefined function. numVariadic is the number of
// variadic arguments, shift is the stack shift and asGoroutine reports
// whether the function must be started as a goroutine.
func (vm *VM) callPredefined(fn *PredefinedFunction, numVariadic int8, shift StackShift, asGoroutine bool) {

	// Make a copy of the frame pointer.
	fp := vm.fp

	// Shift the frame pointer.
	vm.fp[0] += uint32(shift[0])
	vm.fp[1] += uint32(shift[1])
	vm.fp[2] += uint32(shift[2])
	vm.fp[3] += uint32(shift[3])

	// Try to call the function without the reflect.
	if !asGoroutine {
		fn.mx.RLock()
		in := fn.in
		fn.mx.RUnlock()
		if in == nil {
			called := true
			switch f := fn.Func.(type) {
			case func(string) int:
				vm.setInt(1, int64(f(vm.string(1))))
			case func(string) string:
				vm.setString(1, f(vm.string(2)))
			case func(string, string) int:
				vm.setInt(1, int64(f(vm.string(1), vm.string(2))))
			case func(string, int) string:
				vm.setString(1, f(vm.string(2), int(vm.int(1))))
			case func(string, string) bool:
				vm.setBool(1, f(vm.string(1), vm.string(2)))
			case func([]byte) []byte:
				vm.setGeneral(1, f(vm.general(2).([]byte)))
			case func([]byte, []byte) int:
				vm.setInt(1, int64(f(vm.general(1).([]byte), vm.general(2).([]byte))))
			case func([]byte, []byte) bool:
				vm.setBool(1, f(vm.general(1).([]byte), vm.general(2).([]byte)))
			case func(interface{}, interface{}) interface{}:
				vm.setGeneral(1, f(vm.general(2), vm.general(3)))
			case func(interface{}) interface{}:
				vm.setGeneral(1, f(vm.general(2)))
			default:
				called = false
			}
			if called {
				vm.fp = fp // Restore the frame pointer.
				return
			}
		}
	}

	// Prepare the function to be be called with reflect.
	fn.mx.Lock()
	if fn.in == nil {
		fn.value = reflect.ValueOf(fn.Func)
		if fn.value.IsNil() {
			fn.mx.Unlock()
			panic(runtimeError("runtime error: invalid memory address or nil pointer dereference"))
		}
		typ := fn.value.Type()
		nIn := typ.NumIn()
		fn.in = make([]Kind, nIn)
		for i := 0; i < nIn; i++ {
			var k = typ.In(i).Kind()
			switch {
			case k == reflect.Bool:
				fn.in[i] = Bool
			case reflect.Int <= k && k <= reflect.Int64:
				fn.in[i] = Int
			case reflect.Uint <= k && k <= reflect.Uintptr:
				fn.in[i] = Uint
			case k == reflect.Float64 || k == reflect.Float32:
				fn.in[i] = Float64
			case k == reflect.String:
				fn.in[i] = String
			case k == reflect.Func:
				fn.in[i] = Func
			default:
				if i < 2 && typ.In(i) == envType {
					fn.in[i] = Environment
				} else {
					fn.in[i] = Interface
				}
			}
		}
		nOut := typ.NumOut()
		fn.out = make([]Kind, nOut)
		for i := 0; i < nOut; i++ {
			k := typ.Out(i).Kind()
			switch {
			case k == reflect.Bool:
				fn.out[i] = Bool
				fn.outOff[0]++
			case reflect.Int <= k && k <= reflect.Int64:
				fn.out[i] = Int
				fn.outOff[0]++
			case reflect.Uint <= k && k <= reflect.Uintptr:
				fn.out[i] = Uint
				fn.outOff[0]++
			case k == reflect.Float64 || k == reflect.Float32:
				fn.out[i] = Float64
				fn.outOff[1]++
			case k == reflect.String:
				fn.out[i] = String
				fn.outOff[2]++
			case k == reflect.Func:
				fn.out[i] = Func
				fn.outOff[3]++
			default:
				fn.out[i] = Interface
				fn.outOff[3]++
			}
		}
	}
	fn.mx.Unlock()

	// Call the function with reflect.
	var args []reflect.Value
	variadic := fn.value.Type().IsVariadic()

	if len(fn.in) > 0 {

		// Shift the frame pointer.
		vm.fp[0] += uint32(fn.outOff[0])
		vm.fp[1] += uint32(fn.outOff[1])
		vm.fp[2] += uint32(fn.outOff[2])
		vm.fp[3] += uint32(fn.outOff[3])

		// Get a slice of reflect.Value for the arguments.
		fn.mx.Lock()
		if len(fn.args) == 0 {
			fn.mx.Unlock()
			nIn := len(fn.in)
			typ := fn.value.Type()
			args = make([]reflect.Value, nIn)
			for i := 0; i < nIn; i++ {
				t := typ.In(i)
				args[i] = reflect.New(t).Elem()
			}
		} else {
			last := len(fn.args) - 1
			args = fn.args[last]
			fn.args = fn.args[:last]
			fn.mx.Unlock()
		}

		// Prepare the arguments.
		lastNonVariadic := len(fn.in)
		if variadic && numVariadic != NoVariadicArgs {
			lastNonVariadic--
		}
		for i, k := range fn.in {
			if i < lastNonVariadic {
				switch k {
				case Bool:
					args[i].SetBool(vm.bool(1))
					vm.fp[0]++
				case Int:
					args[i].SetInt(vm.int(1))
					vm.fp[0]++
				case Uint:
					args[i].SetUint(uint64(vm.int(1)))
					vm.fp[0]++
				case Float64:
					args[i].SetFloat(vm.float(1))
					vm.fp[1]++
				case String:
					args[i].SetString(vm.string(1))
					vm.fp[2]++
				case Func:
					f := vm.general(1).(*callable)
					args[i].Set(f.Value(vm.env))
					vm.fp[3]++
				case Environment:
					args[i].Set(vm.envArg)
				case Interface:
					if v := vm.general(1); v == nil {
						if t := args[i].Type(); t == emptyInterfaceType {
							args[i] = emptyInterfaceNil
						} else {
							args[i].Set(reflect.Zero(t))
						}
					} else {
						args[i].Set(reflect.ValueOf(v))
					}
					vm.fp[3]++
				default:
					args[i].Set(reflect.ValueOf(vm.general(1)))
					vm.fp[3]++
				}
			} else {
				sliceType := args[i].Type()
				slice := reflect.MakeSlice(sliceType, int(numVariadic), int(numVariadic))
				k := sliceType.Elem().Kind()
				switch k {
				case reflect.Bool:
					for j := 0; j < int(numVariadic); j++ {
						slice.Index(j).SetBool(vm.bool(int8(j + 1)))
					}
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					for j := 0; j < int(numVariadic); j++ {
						slice.Index(j).SetInt(vm.int(int8(j + 1)))
					}
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
					for j := 0; j < int(numVariadic); j++ {
						slice.Index(j).SetUint(uint64(vm.int(int8(j + 1))))
					}
				case reflect.Float32, reflect.Float64:
					for j := 0; j < int(numVariadic); j++ {
						slice.Index(j).SetFloat(vm.float(int8(j + 1)))
					}
				case reflect.Func:
					for j := 0; j < int(numVariadic); j++ {
						f := vm.general(int8(j + 1)).(*callable)
						slice.Index(j).Set(f.Value(vm.env))
					}
				case reflect.String:
					for j := 0; j < int(numVariadic); j++ {
						slice.Index(j).SetString(vm.string(int8(j + 1)))
					}
				case reflect.Interface:
					for j := 0; j < int(numVariadic); j++ {
						if v := vm.general(int8(j + 1)); v == nil {
							if t := slice.Index(j).Type(); t == emptyInterfaceType {
								slice.Index(j).Set(emptyInterfaceNil)
							} else {
								slice.Index(j).Set(reflect.Zero(t))
							}
						} else {
							slice.Index(j).Set(reflect.ValueOf(v))
						}
					}
				default:
					for j := 0; j < int(numVariadic); j++ {
						slice.Index(j).Set(reflect.ValueOf(vm.general(int8(j + 1))))
					}
				}
				args[i].Set(slice)
			}
		}

		// Shift the frame pointer.
		vm.fp[0] = fp[0] + uint32(shift[0])
		vm.fp[1] = fp[1] + uint32(shift[1])
		vm.fp[2] = fp[2] + uint32(shift[2])
		vm.fp[3] = fp[3] + uint32(shift[3])

	}

	if asGoroutine {

		// Start a goroutine.
		if variadic {
			go fn.value.CallSlice(args)
		} else {
			go fn.value.Call(args)
		}

	} else {

		// Call the function and get the results.
		var ret []reflect.Value
		if variadic {
			ret = fn.value.CallSlice(args)
		} else {
			ret = fn.value.Call(args)
		}
		for i, k := range fn.out {
			switch k {
			case Bool:
				vm.setBool(1, ret[i].Bool())
				vm.fp[0]++
			case Int:
				vm.setInt(1, ret[i].Int())
				vm.fp[0]++
			case Uint:
				vm.setInt(1, int64(ret[i].Uint()))
				vm.fp[0]++
			case Float64:
				vm.setFloat(1, ret[i].Float())
				vm.fp[1]++
			case String:
				vm.setString(1, ret[i].String())
				vm.fp[2]++
			case Func:

			default:
				vm.setGeneral(1, ret[i].Interface())
				vm.fp[3]++
			}
		}
		if args != nil {
			fn.mx.Lock()
			fn.args = append(fn.args, args)
			fn.mx.Unlock()
		}

	}

	vm.fp = fp // Restore the frame pointer.

	return
}

//go:noinline
func (vm *VM) invokeTraceFunc() {
	regs := Registers{
		Int:     vm.regs.int[vm.fp[0]+1 : vm.fp[0]+uint32(vm.fn.NumReg[0])+1],
		Float:   vm.regs.float[vm.fp[1]+1 : vm.fp[1]+uint32(vm.fn.NumReg[1])+1],
		String:  vm.regs.string[vm.fp[2]+1 : vm.fp[2]+uint32(vm.fn.NumReg[2])+1],
		General: vm.regs.general[vm.fp[3]+1 : vm.fp[3]+uint32(vm.fn.NumReg[3])+1],
	}
	vm.env.trace(vm.fn, vm.pc, regs)
}

func (vm *VM) moreIntStack() {
	top := len(vm.regs.int) * 2
	stack := make([]int64, top)
	copy(stack, vm.regs.int)
	vm.regs.int = stack
	vm.st[0] = uint32(top)
}

func (vm *VM) moreFloatStack() {
	top := len(vm.regs.float) * 2
	stack := make([]float64, top)
	copy(stack, vm.regs.float)
	vm.regs.float = stack
	vm.st[1] = uint32(top)
}

func (vm *VM) moreStringStack() {
	top := len(vm.regs.string) * 2
	stack := make([]string, top)
	copy(stack, vm.regs.string)
	vm.regs.string = stack
	vm.st[2] = uint32(top)
}

func (vm *VM) moreGeneralStack() {
	top := len(vm.regs.general) * 2
	stack := make([]interface{}, top)
	copy(stack, vm.regs.general)
	vm.regs.general = stack
	vm.st[3] = uint32(top)
}

func (vm *VM) nextCall() bool {
	for i := len(vm.calls) - 1; i >= 0; i-- {
		call := vm.calls[i]
		switch call.status {
		case started:
			// A call is returned, continue with the previous call.
			// TODO(marco): call finalizer.
		case tailed:
			// A tail call is returned, continue with the previous call.
			// TODO(marco): call finalizer.
			continue
		case deferred:
			// A call, that has deferred calls, is returned, its first
			// deferred call will be executed.
			current := callFrame{cl: callable{fn: vm.fn}, fp: vm.fp, status: returned}
			vm.swapStack(&call.fp, &current.fp, current.cl.fn.NumReg)
			vm.calls[i] = current
			i++
		case returned, recovered:
			// A deferred call is returned. If there is another deferred
			// call, it will be executed, otherwise the previous call will be
			// finalized.
			if i > 0 {
				prev := vm.calls[i-1]
				if prev.status == deferred {
					vm.swapStack(&prev.fp, &call.fp, call.cl.fn.NumReg)
					call, vm.calls[i-1] = prev, call
					break
				}
			}
			// TODO(marco): call finalizer.
			if call.status == recovered {
				numPanicked := 0
				for _, c := range vm.calls {
					if c.status == panicked {
						numPanicked++
					}
				}
				vm.panics = vm.panics[:numPanicked]
			}
			continue
		case panicked:
			// A call is panicked, the first deferred call in the call stack,
			// if there is one, will be executed.
			for i = i - 1; i >= 0; i-- {
				call = vm.calls[i]
				if call.status == deferred {
					vm.calls[i] = vm.calls[i+1]
					vm.calls[i].status = panicked
					if call.cl.fn != nil {
						i++
					}
					break
				}
			}
		}
		if i >= 0 {
			if call.cl.fn != nil {
				vm.calls = vm.calls[:i]
				vm.fp = call.fp
				vm.pc = call.pc
				vm.fn = call.cl.fn
				vm.vars = call.cl.vars
				return true
			}
			vm.fp = call.fp
			vm.callPredefined(call.cl.Predefined(), call.numVariadic, StackShift{}, false)
		}
	}
	return false
}

// create creates a new virtual machine with the execution environment env.
func create(env *Env) *VM {
	vm := &VM{
		st: [4]uint32{stackSize, stackSize, stackSize, stackSize},
		regs: registers{
			int:     make([]int64, stackSize),
			float:   make([]float64, stackSize),
			string:  make([]string, stackSize),
			general: make([]interface{}, stackSize),
		},
	}
	if env != nil {
		vm.env = env
		vm.envArg = reflect.ValueOf(env)
		vm.SetContext(env.ctx)
	}
	return vm
}

// startGoroutine starts a new goroutine to execute a function call at program
// counter pc. If the function is predefined, returns true.
func (vm *VM) startGoroutine() bool {
	var fn *Function
	var vars []interface{}
	call := vm.fn.Body[vm.pc]
	switch call.Op {
	case OpCall:
		fn = vm.fn.Functions[uint8(call.A)]
		vars = vm.env.globals
	case OpCallIndirect:
		f := vm.general(call.A).(*callable)
		if f.fn == nil {
			return true
		}
		fn = f.fn
		vars = f.vars
	default:
		return true
	}
	nvm := create(vm.env)
	vm.pc++
	off := vm.fn.Body[vm.pc]
	copy(nvm.regs.int, vm.regs.int[vm.fp[0]+uint32(off.Op):vm.fp[0]+127])
	copy(nvm.regs.float, vm.regs.float[vm.fp[1]+uint32(off.A):vm.fp[1]+127])
	copy(nvm.regs.string, vm.regs.string[vm.fp[2]+uint32(off.B):vm.fp[2]+127])
	copy(nvm.regs.general, vm.regs.general[vm.fp[3]+uint32(off.C):vm.fp[3]+127])
	go nvm.runFunc(fn, vars)
	vm.pc++
	return false
}

// swapStack swaps the stacks pointed by a and b. bSize is the size of the
// stack pointed by b. The stacks must be consecutive and a must precede b.
//
// A stack can have zero size, so a and b can point to the same index.
func (vm *VM) swapStack(a, b *[4]uint32, bSize StackShift) {

	// Swap int registers.
	as := b[0] - a[0]
	bs := uint32(bSize[0])
	if as > 0 && bs > 0 {
		tot := as + bs
		if a[0]+tot+bs > vm.st[0] {
			vm.moreIntStack()
		}
		s := vm.regs.int[a[0]+1:]
		copy(s[bs:], s[:tot])
		copy(s, s[tot:tot+bs])
	}
	b[0] = a[0]
	a[0] += bs

	// Swap float registers.
	as = b[1] - a[1]
	bs = uint32(bSize[1])
	if as > 0 && bs > 0 {
		tot := as + bs
		if a[1]+tot+bs > vm.st[1] {
			vm.moreFloatStack()
		}
		s := vm.regs.float[a[1]+1:]
		copy(s[bs:], s[:tot])
		copy(s, s[tot:tot+bs])
	}
	b[1] = a[1]
	a[1] += bs

	// Swap string registers.
	as = b[2] - a[2]
	bs = uint32(bSize[2])
	if as > 0 && bs > 0 {
		tot := as + bs
		if a[2]+tot+bs > vm.st[2] {
			vm.moreStringStack()
		}
		s := vm.regs.string[a[2]+1:]
		copy(s[bs:], s[:tot])
		copy(s, s[tot:tot+bs])
	}
	b[2] = a[2]
	a[2] += bs

	// Swap general registers.
	as = b[3] - a[3]
	bs = uint32(bSize[3])
	if as > 0 && bs > 0 {
		tot := as + bs
		if a[3]+tot+bs > vm.st[3] {
			vm.moreGeneralStack()
		}
		s := vm.regs.general[a[3]+1:]
		copy(s[bs:], s[:tot])
		copy(s, s[tot:tot+bs])
	}
	b[3] = a[3]
	a[3] += bs

}

type Registers struct {
	Int     []int64
	Float   []float64
	String  []string
	General []interface{}
}

type Kind uint8

const (
	Bool        = Kind(reflect.Bool)
	Int         = Kind(reflect.Int)
	Int8        = Kind(reflect.Int8)
	Int16       = Kind(reflect.Int16)
	Int32       = Kind(reflect.Int32)
	Int64       = Kind(reflect.Int64)
	Uint        = Kind(reflect.Uint)
	Uint8       = Kind(reflect.Uint8)
	Uint16      = Kind(reflect.Uint16)
	Uint32      = Kind(reflect.Uint32)
	Uint64      = Kind(reflect.Uint64)
	Uintptr     = Kind(reflect.Uintptr)
	Float32     = Kind(reflect.Float32)
	Float64     = Kind(reflect.Float64)
	String      = Kind(reflect.String)
	Func        = Kind(reflect.Func)
	Interface   = Kind(reflect.Interface)
	Unknown     = 254
	Environment = 255
)

type PredefinedFunction struct {
	Pkg    string
	Name   string
	Func   interface{}
	mx     sync.RWMutex // synchronize access to the following fields.
	in     []Kind
	out    []Kind
	args   [][]reflect.Value
	outOff [4]int8
	value  reflect.Value
}

// Function represents a function.
type Function struct {
	Pkg        string
	Name       string
	File       string
	Line       int
	Type       reflect.Type
	Parent     *Function
	VarRefs    []int16
	Literals   []*Function
	Types      []reflect.Type
	NumReg     [4]int8
	Constants  Registers
	Functions  []*Function
	Predefined []*PredefinedFunction
	Body       []Instruction
	Lines      map[uint32]int
	Data       [][]byte
}

type callStatus int8

const (
	started callStatus = iota
	tailed
	returned
	deferred
	panicked
	recovered
)

// Size of a CallFrame.
const CallFrameSize = 88

// If the size of callFrame changes, update the constant CallFrameSize.
type callFrame struct {
	cl          callable   // callable.
	fp          [4]uint32  // frame pointers.
	pc          uint32     // program counter.
	status      callStatus // status.
	numVariadic int8       // number of variadic arguments.
}

type callable struct {
	value      reflect.Value       // reflect value.
	fn         *Function           // function, if it is a Scriggo function.
	predefined *PredefinedFunction // predefined function.
	receiver   interface{}         // receiver, if it is a method value.
	method     string              // method name, if it is a method value.
	vars       []interface{}       // closure variables, if it is a closure.
}

// Predefined returns the predefined function of a callable.
func (c *callable) Predefined() *PredefinedFunction {
	if c.predefined != nil {
		return c.predefined
	}
	if !c.value.IsValid() {
		c.value = reflect.ValueOf(c.receiver).MethodByName(c.method)
		c.receiver = nil
		c.method = ""
	}
	c.predefined = &PredefinedFunction{
		Func:  c.value.Interface(),
		value: c.value,
	}
	return c.predefined
}

// Value returns a reflect Value of a callable, so it can be called from a
// predefined code and passed to a predefined code.
//
// TODO(marco): implement for variadic functions.
func (c *callable) Value(env *Env) reflect.Value {
	if c.value.IsValid() {
		return c.value
	}
	if c.predefined != nil {
		// It is a predefined function.
		c.value = reflect.ValueOf(c.predefined.Func)
		return c.value
	}
	if c.method == "" {
		// It is a Scriggo function.
		fn := c.fn
		vars := c.vars
		c.value = reflect.MakeFunc(fn.Type, func(args []reflect.Value) []reflect.Value {
			nvm := create(env)
			nOut := fn.Type.NumOut()
			results := make([]reflect.Value, nOut)
			var r = [4]int8{1, 1, 1, 1}
			for i := 0; i < nOut; i++ {
				typ := fn.Type.Out(i)
				results[i] = reflect.New(typ).Elem()
				t := kindToType(typ.Kind())
				r[t]++
			}
			for _, arg := range args {
				t := kindToType(arg.Kind())
				nvm.setFromReflectValue(r[t], arg)
				r[t]++
			}
			nvm.runFunc(fn, vars)
			r = [4]int8{1, 1, 1, 1}
			for _, result := range results {
				t := kindToType(result.Kind())
				nvm.getIntoReflectValue(r[t], result, false)
				r[t]++
			}
			return results
		})
	} else {
		// It is a method value.
		c.value = reflect.ValueOf(c.receiver).MethodByName(c.method)
	}
	return c.value
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

type Panic struct {
	Msg        interface{}
	Recovered  bool
	StackTrace []byte
}

func (err Panic) String() string {
	return panicToString(err.Msg)
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

func (err Panic) Error() string {
	b := make([]byte, 0, 100+len(err.StackTrace))
	//b = append(b, sprint(err.Msg)...) // TODO(marco): rewrite.
	b = append(b, "\n\n"...)
	b = append(b, err.StackTrace...)
	return string(b)
}

func packageName(pkg string) string {
	for i := len(pkg) - 1; i >= 0; i-- {
		if pkg[i] == '/' {
			return pkg[i+1:]
		}
	}
	return pkg
}

type Type int8

const (
	TypeInt Type = iota
	TypeFloat
	TypeString
	TypeGeneral
)

func (t Type) String() string {
	switch t {
	case TypeInt:
		return "int"
	case TypeFloat:
		return "float"
	case TypeString:
		return "string"
	case TypeGeneral:
		return "general"
	}
	panic("unknown type")
}

// kindToType returns the internal register type of a reflect kind.
func kindToType(k reflect.Kind) Type {
	switch k {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return TypeInt
	case reflect.Float32, reflect.Float64:
		return TypeFloat
	case reflect.String:
		return TypeString
	default:
		return TypeGeneral
	}
}

type Condition int8

const (
	ConditionEqual             Condition = iota // x == y
	ConditionNotEqual                           // x != y
	ConditionLess                               // x <  y
	ConditionLessOrEqual                        // x <= y
	ConditionGreater                            // x >  y
	ConditionGreaterOrEqual                     // x >= y
	ConditionEqualLen                           // len(x) == y
	ConditionNotEqualLen                        // len(x) != y
	ConditionLessLen                            // len(x) <  y
	ConditionLessOrEqualLen                     // len(x) <= y
	ConditionGreaterLen                         // len(x) >  y
	ConditionGreaterOrEqualLen                  // len(x) >= y
	ConditionInterfaceNil                       // x == nil
	ConditionInterfaceNotNil                    // x != nil
	ConditionNil                                // x == nil
	ConditionNotNil                             // x != nil
	ConditionOK                                 // [vm.ok]
	ConditionNotOK                              // ![vm.ok]
)

type Operation int8

const (
	OpNone Operation = iota

	OpAddInt8
	OpAddInt16
	OpAddInt32
	OpAddInt64
	OpAddFloat32
	OpAddFloat64

	OpAddr

	OpAlloc

	OpAnd

	OpAndNot

	OpAssert

	OpAppend

	OpAppendSlice

	OpBind

	OpBreak

	OpCall

	OpCallIndirect

	OpCallPredefined

	OpCap

	OpCase

	OpClose

	OpComplex64
	OpComplex128

	OpContinue

	OpConvertGeneral
	OpConvertInt
	OpConvertUint
	OpConvertFloat
	OpConvertString

	OpConcat

	OpCopy

	OpDefer

	OpDelete

	OpDivInt8
	OpDivInt16
	OpDivInt32
	OpDivInt64
	OpDivUint8
	OpDivUint16
	OpDivUint32
	OpDivUint64
	OpDivFloat32
	OpDivFloat64

	OpField

	OpFunc

	OpGetFunc

	OpGetVar

	OpGetVarAddr

	OpGo

	OpGoto

	OpIf
	OpIfInt
	OpIfUint
	OpIfFloat
	OpIfString

	OpIndex
	OpIndexMap
	OpIndexString

	OpLeftShift8
	OpLeftShift16
	OpLeftShift32
	OpLeftShift64

	OpLen

	OpLoadData

	OpLoadNumber

	OpMakeChan

	OpMakeMap

	OpMakeSlice

	OpMethodValue

	OpMove

	OpMulInt8
	OpMulInt16
	OpMulInt32
	OpMulInt64
	OpMulFloat32
	OpMulFloat64

	OpNew

	OpOr

	OpPanic

	OpPrint

	OpRange

	OpRangeString

	OpRealImag

	OpReceive

	OpRecover

	OpRemInt8
	OpRemInt16
	OpRemInt32
	OpRemInt64
	OpRemUint8
	OpRemUint16
	OpRemUint32
	OpRemUint64

	OpReturn

	OpRightShift
	OpRightShiftU

	OpSelect

	OpSend

	OpSetField
	OpSetMap

	OpSetSlice

	OpSetVar

	OpSlice
	OpSliceString

	OpSubInt8
	OpSubInt16
	OpSubInt32
	OpSubInt64
	OpSubFloat32
	OpSubFloat64

	OpSubInvInt8
	OpSubInvInt16
	OpSubInvInt32
	OpSubInvInt64
	OpSubInvFloat32
	OpSubInvFloat64

	OpTailCall

	OpTypify

	OpXor
)
