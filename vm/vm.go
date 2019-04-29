// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"reflect"
	"strconv"
)

const StackSize = 512

type VM struct {
	fp     [4]uint32       // frame pointers.
	st     [4]uint32       // stack tops.
	pc     uint32          // program counter.
	ok     bool            // ok flag.
	regs   registers       // registers.
	fn     *ScrigoFunction // current function.
	cvars  []interface{}   // closure variables.
	calls  []Call          // call stack.
	panics []Panic         // panics.
}

func New() *VM {
	vm := &VM{}
	vm.regs.Int = make([]int64, StackSize)
	vm.regs.Float = make([]float64, StackSize)
	vm.regs.String = make([]string, StackSize)
	vm.regs.General = make([]interface{}, StackSize)
	vm.st[0] = StackSize
	vm.st[1] = StackSize
	vm.st[2] = StackSize
	vm.st[3] = StackSize
	vm.Reset()
	return vm
}

func (vm *VM) Reset() {
	vm.fp = [4]uint32{0, 0, 0, 0}
	vm.pc = 0
	vm.fn = nil
	vm.cvars = nil
	vm.calls = vm.calls[:0]
	vm.panics = vm.panics[:0]
}

func (vm *VM) moreIntStack() {
	top := len(vm.regs.Int) * 2
	stack := make([]int64, top)
	copy(stack, vm.regs.Int)
	vm.regs.Int = stack
	vm.st[0] = uint32(top)
}

func (vm *VM) moreFloatStack() {
	top := len(vm.regs.Float) * 2
	stack := make([]float64, top)
	copy(stack, vm.regs.Float)
	vm.regs.Float = stack
	vm.st[1] = uint32(top)
}

func (vm *VM) moreStringStack() {
	top := len(vm.regs.String) * 2
	stack := make([]string, top)
	copy(stack, vm.regs.String)
	vm.regs.String = stack
	vm.st[2] = uint32(top)
}

func (vm *VM) moreGeneralStack() {
	top := len(vm.regs.General) * 2
	stack := make([]interface{}, top)
	copy(stack, vm.regs.General)
	vm.regs.General = stack
	vm.st[3] = uint32(top)
}

type CallStatus int8

const (
	Started CallStatus = iota
	Tailed
	Returned
	Deferred
	Panicked
	Recovered
)

type Call struct {
	fn        callable   // function.
	fp        [4]uint32  // frame pointers.
	pc        uint32     // program counter.
	status    CallStatus // status.
	variadics int8       // number of variadic arguments.
}

type callable struct {
	scrigo *ScrigoFunction // Scrigo function.
	native *NativeFunction // native function.
	value  reflect.Value   // reflect value.
	vars   []interface{}   // closure variables, if it is a closure.
}

// reflectValue returns a Reflect Value of a callable, so it can be called
// from a native code and passed to a native code.
func (c *callable) reflectValue() reflect.Value {
	if c.value.IsValid() {
		return c.value
	}
	if c.native != nil {
		// It is a native function.
		if !c.native.value.IsValid() {
			c.native.value = reflect.ValueOf(c.native.fast)
		}
		c.value = c.native.value
		return c.value
	}
	// It is a Scrigo function.
	fn := c.scrigo
	cvars := c.vars
	c.value = reflect.MakeFunc(fn.typ, func(args []reflect.Value) []reflect.Value {
		nvm := New()
		nvm.fn = fn
		nvm.cvars = cvars
		nOut := fn.typ.NumOut()
		results := make([]reflect.Value, nOut)
		for i := 0; i < nOut; i++ {
			t := fn.typ.Out(i)
			results[i] = reflect.New(t).Elem()
			k := t.Kind()
			switch k {
			case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				nvm.fp[0]++
			case reflect.Float32, reflect.Float64:
				nvm.fp[1]++
			case reflect.String:
				nvm.fp[2]++
			default:
				nvm.fp[3]++
			}
		}
		var r int8 = 1
		for _, arg := range args {
			k := arg.Kind()
			switch k {
			case reflect.Bool:
				nvm.setBool(r, arg.Bool())
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				nvm.setInt(r, arg.Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				nvm.setInt(r, int64(arg.Uint()))
			case reflect.Float32, reflect.Float64:
				nvm.setFloat(r, arg.Float())
			case reflect.String:
				nvm.setString(r, arg.String())
			default:
				nvm.setGeneral(r, arg.Interface())
			}
			r++
		}
		nvm.fp[0] = 0
		nvm.fp[1] = 0
		nvm.fp[2] = 0
		nvm.fp[3] = 0
		nvm.run()
		r = 1
		for i, result := range results {
			t := fn.typ.Out(i)
			k := t.Kind()
			switch k {
			case reflect.Bool:
				result.SetBool(nvm.bool(r))
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				result.SetInt(nvm.int(r))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				result.SetUint(uint64(nvm.int(r)))
			case reflect.Float32, reflect.Float64:
				result.SetFloat(nvm.float(r))
			case reflect.String:
				result.SetString(nvm.string(r))
			default:
				result.Set(reflect.ValueOf(nvm.general(r)))
			}
			r++
		}
		return results
	})
	return c.value
}

// startScrigoGoroutine starts a new goroutine to execute a Scrigo function
// call at program counter pc. If the function is native, returns false.
func (vm *VM) startScrigoGoroutine() bool {
	var fn *ScrigoFunction
	var vars []interface{}
	call := vm.fn.body[vm.pc]
	switch call.op {
	case opCall:
		fn = vm.fn.scrigoFunctions[uint8(call.b)]
	case opCallIndirect:
		f := vm.general(call.b).(*callable)
		if f.scrigo == nil {
			return false
		}
		fn = f.scrigo
		vars = f.vars
	default:
		return false
	}
	nvm := New()
	nvm.fn = fn
	nvm.cvars = vars
	vm.pc++
	off := vm.fn.body[vm.pc]
	copy(nvm.regs.Int, vm.regs.Int[vm.fp[0]+uint32(off.op):vm.fp[0]+127])
	copy(nvm.regs.Float, vm.regs.Float[vm.fp[1]+uint32(off.a):vm.fp[1]+127])
	copy(nvm.regs.String, vm.regs.String[vm.fp[2]+uint32(off.b):vm.fp[2]+127])
	copy(nvm.regs.General, vm.regs.General[vm.fp[3]+uint32(off.c):vm.fp[3]+127])
	go nvm.run()
	vm.pc++
	return true
}

type Panic struct {
	Msg        interface{}
	Recovered  bool
	StackTrace []byte
}

type stringer interface {
	String() string
}

func (err *Panic) Error() string {
	b := make([]byte, 0, 100+len(err.StackTrace))
	switch v := err.Msg.(type) {
	case nil:
		b = append(b, "nil"...)
	case error:
		b = append(b, v.Error()...)
	case stringer:
		b = append(b, v.String()...)
	case bool:
		b = strconv.AppendBool(b, v)
	case int:
		b = strconv.AppendInt(b, int64(v), 10)
	case int8:
		b = strconv.AppendInt(b, int64(v), 10)
	case int16:
		b = strconv.AppendInt(b, int64(v), 10)
	case int32:
		b = strconv.AppendInt(b, int64(v), 10)
	case int64:
		b = strconv.AppendInt(b, v, 10)
	case uint:
		b = strconv.AppendUint(b, uint64(v), 10)
	case uint8:
		b = strconv.AppendUint(b, uint64(v), 10)
	case uint16:
		b = strconv.AppendUint(b, uint64(v), 10)
	case uint32:
		b = strconv.AppendUint(b, uint64(v), 10)
	case uint64:
		b = strconv.AppendUint(b, v, 10)
	case uintptr:
		b = strconv.AppendUint(b, uint64(v), 10)
	case float32:
		b = strconv.AppendFloat(b, float64(v), 'e', 6, 32)
	case float64:
		b = strconv.AppendFloat(b, v, 'e', 6, 64)
	case string:
		b = append(b, v...)
	default:
		if v2, ok := v.(*callable); ok {
			v = v2.reflectValue().Interface()
		}
		b = append(b, "("...)
		b = append(b, reflect.TypeOf(v).String()...)
		b = append(b, ")"...)
		// TODO(marco): implement print of value
	}
	b = append(b, "\n\n"...)
	b = append(b, err.StackTrace...)
	return string(b)
}

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
	write("scrigo goroutine 1 [running]:")
	size := len(vm.calls)
	for i := size; i >= 0; i-- {
		var fn *ScrigoFunction
		var ppc uint32
		if i == size {
			fn = vm.fn
			ppc = vm.pc - 1
		} else {
			call := vm.calls[i]
			fn = call.fn.scrigo
			if call.status == Tailed {
				ppc = call.pc - 1
			} else {
				ppc = call.pc - 2
			}
		}
		write("\n")
		write(packageName(fn.pkg))
		write(".")
		write(fn.name)
		write("()\n\t")
		if fn.file != "" {
			write(fn.file)
		} else {
			write("???")
		}
		write(":")
		if line, ok := fn.lines[ppc]; ok {
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

func (vm *VM) callNative(fn *NativeFunction, numVariadic int8, shift StackShift, newGoroutine bool) {
	fp := vm.fp
	vm.fp[0] += uint32(shift[0])
	vm.fp[1] += uint32(shift[1])
	vm.fp[2] += uint32(shift[2])
	vm.fp[3] += uint32(shift[3])
	if fn.fast != nil {
		if newGoroutine {
			switch f := fn.fast.(type) {
			case func(string) int:
				go f(vm.string(1))
			case func(string) string:
				go f(vm.string(2))
			case func(string, string) int:
				go f(vm.string(1), vm.string(2))
			case func(string, int) string:
				go f(vm.string(2), int(vm.int(1)))
			case func(string, string) bool:
				go f(vm.string(1), vm.string(2))
			case func([]byte) []byte:
				go f(vm.general(2).([]byte))
			case func([]byte, []byte) int:
				go f(vm.general(1).([]byte), vm.general(2).([]byte))
			case func([]byte, []byte) bool:
				go f(vm.general(1).([]byte), vm.general(2).([]byte))
			default:
				fn.slow()
			}
		} else {
			switch f := fn.fast.(type) {
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
			default:
				fn.slow()
			}
		}
	}
	if fn.fast == nil {
		variadic := fn.value.Type().IsVariadic()
		if len(fn.in) > 0 {
			vm.fp[0] += uint32(fn.outOff[0])
			vm.fp[1] += uint32(fn.outOff[1])
			vm.fp[2] += uint32(fn.outOff[2])
			vm.fp[3] += uint32(fn.outOff[3])
			lastNonVariadic := len(fn.in)
			if variadic && numVariadic != NoVariadic {
				lastNonVariadic--
			}
			for i, k := range fn.in {
				if i < lastNonVariadic {
					switch k {
					case Bool:
						fn.args[i].SetBool(vm.bool(1))
						vm.fp[0]++
					case Int:
						fn.args[i].SetInt(vm.int(1))
						vm.fp[0]++
					case Uint:
						fn.args[i].SetUint(uint64(vm.int(1)))
						vm.fp[0]++
					case Float64:
						fn.args[i].SetFloat(vm.float(1))
						vm.fp[1]++
					case String:
						fn.args[i].SetString(vm.string(1))
						vm.fp[2]++
					case Func:
						f := vm.general(1).(*callable)
						fn.args[i].Set(f.reflectValue())
						vm.fp[3]++
					default:
						fn.args[i].Set(reflect.ValueOf(vm.general(1)))
						vm.fp[3]++
					}
				} else {
					sliceType := fn.args[i].Type()
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
					case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
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
							slice.Index(j).Set(f.reflectValue())
						}
					case reflect.String:
						for j := 0; j < int(numVariadic); j++ {
							slice.Index(j).SetString(vm.string(int8(j + 1)))
						}
					default:
						for j := 0; j < int(numVariadic); j++ {
							slice.Index(j).Set(reflect.ValueOf(vm.general(int8(j + 1))))
						}
					}
					fn.args[i].Set(slice)
				}
			}
			vm.fp[0] = fp[0] + uint32(shift[0])
			vm.fp[1] = fp[1] + uint32(shift[1])
			vm.fp[2] = fp[2] + uint32(shift[2])
			vm.fp[3] = fp[3] + uint32(shift[3])
		}
		if newGoroutine {
			if variadic {
				go fn.value.CallSlice(fn.args)
			} else {
				go fn.value.Call(fn.args)
			}
		} else {
			var ret []reflect.Value
			if variadic {
				ret = fn.value.CallSlice(fn.args)
			} else {
				ret = fn.value.Call(fn.args)
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
		}
	}
	vm.fp = fp
	vm.pc++
}

func (vm *VM) deferCall(fn *callable, numVariadic int8, shift, args StackShift) {
	vm.calls = append(vm.calls, Call{fn: *fn, fp: vm.fp, pc: 0, status: Deferred, variadics: numVariadic})
	if args[0] > 0 {
		stack := vm.regs.Int[vm.fp[0]+1:]
		tot := shift[0] + args[0]
		copy(stack[shift[0]:], stack[:tot])
		copy(stack, stack[shift[0]:tot])
		vm.fp[0] += uint32(args[0])
	}
	if args[1] > 0 {
		stack := vm.regs.Float[vm.fp[1]+1:]
		tot := shift[1] + args[1]
		copy(stack[shift[1]:], stack[:tot])
		copy(stack, stack[shift[1]:tot])
		vm.fp[1] += uint32(args[1])
	}
	if args[2] > 0 {
		stack := vm.regs.String[vm.fp[2]+1:]
		tot := shift[2] + args[2]
		copy(stack[shift[2]:], stack[:tot])
		copy(stack, stack[shift[2]:tot])
		vm.fp[2] += uint32(args[2])
	}
	if args[3] > 0 {
		stack := vm.regs.General[vm.fp[3]+1:]
		tot := shift[3] + args[3]
		copy(stack[shift[3]:], stack[:tot])
		copy(stack, stack[shift[3]:tot])
		vm.fp[3] += uint32(args[3])
	}
}

func (vm *VM) swapCall(call Call) Call {
	if call.fp[0] < vm.fp[0] {
		a := uint32(vm.fp[0] - call.fp[0])
		b := uint32(vm.fn.regnum[0])
		if vm.fp[0]+2*b > vm.st[0] {
			vm.moreIntStack()
		}
		s := vm.regs.Int[call.fp[0]+1:]
		copy(s[a:], s[:a+b])
		copy(s, s[a+b:a+2*b])
		vm.fp[0] -= a
		call.fp[0] += b
	}
	if call.fp[1] < vm.fp[1] {
		a := uint32(vm.fp[1] - call.fp[1])
		b := uint32(vm.fn.regnum[1])
		if vm.fp[1]+2*b > vm.st[1] {
			vm.moreFloatStack()
		}
		s := vm.regs.Float[call.fp[1]+1:]
		copy(s[a:], s[:a+b])
		copy(s, s[a+b:a+2*b])
		vm.fp[1] -= a
		call.fp[1] += b
	}
	if call.fp[2] < vm.fp[2] {
		a := uint32(vm.fp[2] - call.fp[2])
		b := uint32(vm.fn.regnum[2])
		if vm.fp[2]+2*b > vm.st[2] {
			vm.moreStringStack()
		}
		s := vm.regs.Float[call.fp[2]+1:]
		copy(s[a:], s[:a+b])
		copy(s, s[a+b:a+2*b])
		vm.fp[2] -= a
		call.fp[2] += b
	}
	if call.fp[3] < vm.fp[3] {
		a := uint32(vm.fp[3] - call.fp[3])
		b := uint32(vm.fn.regnum[3])
		if vm.fp[3]+2*b > vm.st[3] {
			vm.moreGeneralStack()
		}
		s := vm.regs.General[call.fp[3]+1:]
		copy(s[a:], s[:a+b])
		copy(s, s[a+b:a+2*b])
		vm.fp[3] -= a
		call.fp[3] += b
	}
	return call
}

func (vm *VM) nextCall() bool {
	var i int
	var call Call
	for i = len(vm.calls) - 1; i >= 0; i-- {
		call = vm.calls[i]
		switch call.status {
		case Started:
			// A call is returned, continue with the previous call.
			// TODO(marco): call finalizer.
		case Tailed:
			// A tail call is returned, continue with the previous call.
			// TODO(marco): call finalizer.
			continue
		case Deferred:
			// A Scrigo call that has deferred calls is returned, its first
			// deferred call will be executed.
			call = vm.swapCall(call)
			vm.calls[i] = Call{fn: callable{scrigo: vm.fn}, fp: vm.fp, status: Returned}
			if call.fn.scrigo != nil {
				break
			}
			vm.callNative(call.fn.native, call.variadics, StackShift{}, false)
			fallthrough
		case Returned, Recovered:
			// A deferred call is returned. If there is another deferred
			// call, it will be executed, otherwise the previous call will be
			// finalized.
			if i > 0 {
				if prev := vm.calls[i-1]; prev.status == Deferred {
					call, vm.calls[i-1] = prev, call
					break
				}
			}
			// TODO(marco): call finalizer.
			if call.status == Recovered {
				vm.panics = vm.panics[:len(vm.panics)-1]
			}
			continue
		case Panicked:
			// A call is panicked, the first deferred call in the call stack,
			// if there is one, will be executed.
			for i = i - 1; i >= 0; i-- {
				call = vm.calls[i]
				if call.status == Deferred {
					vm.calls[i] = vm.calls[i+1]
					vm.calls[i].status = Panicked
					if call.fn.scrigo != nil {
						i++
						break
					}
					vm.callNative(call.fn.native, call.variadics, StackShift{}, false)
				}
			}
		}
		break
	}
	if i >= 0 {
		vm.calls = vm.calls[:i]
		vm.fp = call.fp
		vm.pc = call.pc
		vm.fn = call.fn.scrigo
		vm.cvars = call.fn.vars
		return true
	}
	return false
}
