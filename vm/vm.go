// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"bytes"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

const debug = true

type stackFrames int

const (
	deferState stackFrames = iota
	preDeferState
	scrigoState
	nativeState
	preRunState
)

var stackFramesStrings = [...]string{"defer", "pre defer", "scrigo", "native", "pre run"}

var panicCallStacks = sync.Map{}

const StackSize = 512

type VM struct {
	fp      [4]uint32 // frame pointers.
	st      [4]uint32 // stack tops.
	pc      uint32    // program counter.
	ok      bool      // ok flag.
	cpanic  bool      // capture panic?
	isFirst bool
	regs    registers       // registers.
	fn      *ScrigoFunction // current function.
	cvars   []interface{}   // closure variables.
	calls   []Call          // call stack.
	main    *Package        // package "main".
}

func New(main *Package) *VM {
	vm := &VM{}
	vm.main = main
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
	vm.isFirst = false
	vm.cvars = nil
	vm.calls = vm.calls[:]
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

type Call struct {
	fn    *ScrigoFunction // function.
	cvars []interface{}   // closure variables.
	fp    [4]uint32       // frame pointers.
	pc    uint32          // program counter.
	tail  bool            // tail call.
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
		nvm := New(nil)
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
		pkg := vm.fn.pkg
		if call.a != CurrentPackage {
			pkg = pkg.packages[uint8(call.a)]
		}
		fn = pkg.scrigoFunctions[uint8(call.b)]
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
	nvm := New(vm.main)
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
	msg    interface{}
	frames []interface{}
}

type stringer interface {
	String() string
}

func (p Panic) Error() string {
	pc, _, _, ok := runtime.Caller(2)
	if ok && runtime.FuncForPC(pc).Name() == "runtime.gopanic" {
		print("panic: ")
		switch msg := p.msg.(type) {
		case error:
			print(msg.Error())
		case stringer:
			print(msg.String())
		case bool:
			print(msg)
		case int:
			print(msg)
		case int8:
			print(msg)
		case int16:
			print(msg)
		case int32:
			print(msg)
		case int64:
			print(msg)
		case uint:
			print(msg)
		case uint8:
			print(msg)
		case uint16:
			print(msg)
		case uint32:
			print(msg)
		case uint64:
			print(msg)
		case float32:
			print(msg)
		case float64:
			print(msg)
		case string:
			print(msg)
		default:
			print("(", reflect.TypeOf(p.msg).String(), ") ", p.msg)
		}
		print("\n\ngoroutine 1 [running]:")
		for _, frame := range p.frames {
			switch frame := frame.(type) {
			case Call:
				fn := frame.fn
				var ppc uint32
				if frame.tail {
					ppc = frame.pc - 1
				} else {
					ppc = frame.pc - 2
				}
				print("\n", fn.pkg.name, ".", fn.name, "()\n\t")
				if fn.file != "" {
					print(fn.file)
				}
				print(":")
				if line, ok := fn.lines[ppc]; ok {
					print(line)
				}
			case runtime.Frame:
				print("\n", frame.Function, "()\n\t", frame.File, ":", frame.Line)
			}
		}
		print("\n\nnative ")
	}
	return "[msg]"
}

func (vm *VM) panic(err interface{}, fn *NativeFunction) {

	vm.calls = append(vm.calls, Call{fn: vm.fn, pc: vm.pc})
	gid := getGID()

	var calls [][]Call
	if c, ok := panicCallStacks.Load(gid); ok {
		calls = append(c.([][]Call), vm.calls)
	} else {
		calls = [][]Call{vm.calls}
	}
	panicCallStacks.Store(gid, calls)

	if !vm.isFirst {
		panic(err)
	}

	p := &Panic{msg: err}

	if fn == nil {
		p.frames = make([]interface{}, len(calls))
		for i, call := range calls[0] {
			p.frames[i] = call
		}
		calls = nil
	} else {
		var frames []interface{}
		var callers = make([]uintptr, 20)
		for {
			n := runtime.Callers(0, callers)
			if n < len(callers) {
				callers = callers[:n]
				break
			}
			callers = make([]uintptr, len(callers)*2)
		}
		callersFrame := runtime.CallersFrames(callers)
		var frame runtime.Frame
		more := true
		state := deferState
		dumpFrames := false
		for more {
			frame, more = callersFrame.Next()
			fun := frame.Function
			if debug {
				print(">>> ", fun, " (", stackFramesStrings[state], " -> ")
			}
			switch state {
			case deferState:
				if fun == "runtime.gopanic" {
					state = preDeferState
				}
			case preDeferState:
				if fun == "scrigo/vm.(*VM).panic" {
					for _, call := range calls[len(calls)-1] {
						frames = append(frames, call)
					}
					calls = calls[:len(calls)-1]
					state = scrigoState
				} else {
					frames = append(frames, frame)
					state = nativeState
				}
				dumpFrames = true
			case scrigoState:
				if !strings.HasPrefix(fun, "scrigo/vm.") && !strings.HasPrefix(fun, "reflect.") {
					frames = append(frames, frame)
					state = nativeState
					dumpFrames = true
				}
			case nativeState:
				if strings.HasPrefix(fun, "scrigo/vm.(*VM).") {
					if strings.HasSuffix(fun, "panic") {
						frames = frames[:len(frames)-1]
					} else {
						frames = frames[:len(frames)-2]
					}
					for _, call := range calls[len(calls)-1] {
						frames = append(frames, call)
					}
					calls = calls[:len(calls)-1]
					state = scrigoState
				} else {
					frames = append(frames, frame)
				}
				dumpFrames = true
			case preRunState:
				// Do nothing.
			}
			if debug {
				print(stackFramesStrings[state], ")\n")
				if dumpFrames {
					if len(frames) == 0 {
						print("\t(no frame)\n")
					} else {
						for _, frame := range frames {
							switch f := frame.(type) {
							case Call:
								print("\t", f.fn.pkg.name, ".", f.fn.name, "\n")
							case runtime.Frame:
								print("\t", f.Function, "\n")
							}
						}
					}
					dumpFrames = false
				}
			}
			if len(calls) == 0 {
				if debug {
					state = preRunState
				} else {
					break
				}
			}
		}
		p.frames = frames
	}

	panic(p)
}

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

//var bc = [64]byte{}
//
//func getGID() []byte {
//	b := bc[:]
//	b = b[:runtime.Stack(b, false)]
//	b = b[10:]
//	return b[:bytes.IndexByte(b, ' ')]
//}

//func (vm *VM) Stack(buf []byte, all bool) int {
//	// TODO(marco): implement all == true
//	if len(buf) == 0 {
//		return 0
//	}
//	b := buf[0:0:len(buf)]
//	write := func(s string) {
//		n := copy(b[len(b):cap(b)], s)
//		b = b[:len(b)+n]
//	}
//	write("scrigo goroutine 1 [running]:")
//	size := len(vm.calls)
//	for i := size; i >= 0; i-- {
//		var fn *ScrigoFunction
//		var ppc uint32
//		if i == size {
//			fn = vm.fn
//			ppc = vm.pc - 1
//		} else {
//			call := vm.calls[i]
//			fn = call.fn
//			if call.tail {
//				ppc = call.pc - 1
//			} else {
//				ppc = call.pc - 2
//			}
//		}
//		write("\n")
//		write(fn.pkg.name)
//		write(".")
//		write(fn.name)
//		write("()\n\t")
//		if fn.file != "" {
//			write(fn.file)
//		} else {
//			write("???")
//		}
//		write(":")
//		if line, ok := fn.lines[ppc]; ok {
//			write(strconv.Itoa(line))
//		} else {
//			write("???")
//		}
//		if len(b) == len(buf) {
//			break
//		}
//	}
//	return len(b)
//}
