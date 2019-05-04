// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"reflect"
	"strconv"
)

const NoVariadic = -1
const CurrentFunction = -1

const StackSize = 512
const maxInt8 = 128

type StackShift [4]int8

type instruction struct {
	op      operation
	a, b, c int8
}

func decodeAddr(a, b, c int8) uint32 {
	return uint32(uint8(a)) | uint32(uint8(b))<<8 | uint32(uint8(c))<<16
}

type MoveType int8

const (
	IntInt MoveType = iota
	FloatFloat
	StringString
	GeneralGeneral
	IntGeneral
	FloatGeneral
	StringGeneral
)

// VM represents a Scrigo virtual machine.
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

// New returns a new virtual machine.
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

// Reset resets a virtual machine so that it is ready for a new call to Run.
func (vm *VM) Reset() {
	vm.fp = [4]uint32{0, 0, 0, 0}
	vm.pc = 0
	vm.fn = nil
	vm.cvars = nil
	vm.calls = vm.calls[:0]
	vm.panics = vm.panics[:0]
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

// callNative calls a native function. numVariadic is the number of actual
// variadic arguments, shift is the stack shift and newGoroutine reports
// whether a new goroutine must be started.
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

type Kind uint8

const (
	Bool      = Kind(reflect.Bool)
	Int       = Kind(reflect.Int)
	Int8      = Kind(reflect.Int8)
	Int16     = Kind(reflect.Int16)
	Int32     = Kind(reflect.Int32)
	Int64     = Kind(reflect.Int64)
	Uint      = Kind(reflect.Uint)
	Uint8     = Kind(reflect.Uint8)
	Uint16    = Kind(reflect.Uint16)
	Uint32    = Kind(reflect.Uint32)
	Uint64    = Kind(reflect.Uint64)
	Float32   = Kind(reflect.Float32)
	Float64   = Kind(reflect.Float64)
	String    = Kind(reflect.String)
	Func      = Kind(reflect.Func)
	Interface = Kind(reflect.Interface)
)

type NativeFunction struct {
	pkg    string
	name   string
	fast   interface{}
	value  reflect.Value
	in     []Kind
	out    []Kind
	args   []reflect.Value
	outOff [4]int8
}

// variable represents a global variable with a package, name and value.
// value must a pointer to the variable value.
type variable struct {
	pkg   string
	name  string
	value interface{}
}

// NewVariable returns a new variable.
func NewVariable(pkg, name string, value interface{}) variable {
	return variable{pkg, name, value}
}

// ScrigoFunction represents a Scrigo function.
type ScrigoFunction struct {
	pkg             string
	name            string
	file            string
	line            int
	typ             reflect.Type
	parent          *ScrigoFunction
	crefs           []int16           // opFunc
	literals        []*ScrigoFunction // opFunc
	types           []reflect.Type    // opAlloc, opAssert, opMakeMap, opMakeSlice, opNew
	regnum          [4]uint8          // opCall, opCallDirect
	constants       registers
	variables       []variable
	scrigoFunctions []*ScrigoFunction
	nativeFunctions []*NativeFunction
	body            []instruction // run, opCall, opCallDirect
	lines           map[uint32]int
}

// NewScrigoFunction returns a new Scrigo function with a given package, name
// and type.
func NewScrigoFunction(pkg, name string, typ reflect.Type) *ScrigoFunction {
	return &ScrigoFunction{pkg: pkg, name: name, typ: typ}
}

func (fn *ScrigoFunction) AddLine(pc uint32, line int) {
	if fn.lines == nil {
		fn.lines = map[uint32]int{pc: line}
	} else {
		fn.lines[pc] = line
	}
}

// AddNativeFunction adds a native function to a Scrigo function.
func (fn *ScrigoFunction) AddNativeFunction(f *NativeFunction) uint8 {
	r := len(fn.nativeFunctions)
	if r > 255 {
		panic("native functions limit reached")
	}
	fn.nativeFunctions = append(fn.nativeFunctions, f)
	return uint8(r)
}

// AddScrigoFunction adds a Scrigo function to a Scrigo function.
// TODO(Gianluca): is this method obsolete?
func (fn *ScrigoFunction) AddScrigoFunction(f *ScrigoFunction) uint8 {
	r := len(fn.scrigoFunctions)
	if r > 255 {
		panic("Scrigo functions limit reached")
	}
	fn.scrigoFunctions = append(fn.scrigoFunctions, f)
	return uint8(r)
}

// AddType adds a type to a Scrigo function.
func (fn *ScrigoFunction) AddType(typ reflect.Type) uint8 {
	index := len(fn.types)
	if index > 255 {
		panic("types limit reached")
	}
	for i, t := range fn.types {
		if t == typ {
			return uint8(i)
		}
	}
	fn.types = append(fn.types, typ)
	return uint8(index)
}

// AddVariable adds a variable to a Scrigo function.
func (fn *ScrigoFunction) AddVariable(v variable) uint8 {
	r := len(fn.variables)
	if r > 255 {
		panic("variables limit reached")
	}
	fn.variables = append(fn.variables, v)
	return uint8(r)
}

func (fn *ScrigoFunction) SetClosureRefs(refs []int16) {
	fn.crefs = refs
}

// SetFileLine sets the file name and line number of a Scrigo function.
func (fn *ScrigoFunction) SetFileLine(file string, line int) {
	fn.file = file
	fn.line = line
}

// NewNativeFunction returns a new native function with a given package, name
// and implementation. fn must be a function type.
func NewNativeFunction(pkg, name string, fn interface{}) *NativeFunction {
	return &NativeFunction{pkg: pkg, name: name, fast: fn}
}

func (fn *NativeFunction) slow() {
	if !fn.value.IsValid() {
		fn.value = reflect.ValueOf(fn.fast)
	}
	typ := fn.value.Type()
	nIn := typ.NumIn()
	fn.in = make([]Kind, nIn)
	fn.args = make([]reflect.Value, nIn)
	for i := 0; i < nIn; i++ {
		t := typ.In(i)
		k := t.Kind()
		switch {
		case k == reflect.Bool:
			fn.in[i] = Bool
		case reflect.Int <= k && k <= reflect.Int64:
			fn.in[i] = Int
		case reflect.Uint <= k && k <= reflect.Uint64:
			fn.in[i] = Uint
		case k == reflect.Float64 || k == reflect.Float32:
			fn.in[i] = Float64
		case k == reflect.String:
			fn.in[i] = String
		case k == reflect.Func:
			fn.in[i] = Func
		default:
			fn.in[i] = Interface
		}
		fn.args[i] = reflect.New(t).Elem()
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
		case reflect.Uint <= k && k <= reflect.Uint64:
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
	fn.fast = nil
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

type Panic struct {
	Msg        interface{}
	Recovered  bool
	StackTrace []byte
}

type stringer interface {
	String() string
}

// TODO(Gianluca): this code is wrong, needs a refactoring.
func sprint(v interface{}) []byte {
	switch v := v.(type) {
	case nil:
		return []byte("nil")
	case error:
		return []byte(v.Error())
	case stringer:
		return []byte(v.String())
	case bool:
		return strconv.AppendBool([]byte{}, v)
	case int:
		return strconv.AppendInt([]byte{}, int64(v), 10)
	case int8:
		return strconv.AppendInt([]byte{}, int64(v), 10)
	case int16:
		return strconv.AppendInt([]byte{}, int64(v), 10)
	case int32:
		return strconv.AppendInt([]byte{}, int64(v), 10)
	case int64:
		return strconv.AppendInt([]byte{}, v, 10)
	case uint:
		return strconv.AppendUint([]byte{}, uint64(v), 10)
	case uint8:
		return strconv.AppendUint([]byte{}, uint64(v), 10)
	case uint16:
		return strconv.AppendUint([]byte{}, uint64(v), 10)
	case uint32:
		return strconv.AppendUint([]byte{}, uint64(v), 10)
	case uint64:
		return strconv.AppendUint([]byte{}, v, 10)
	case uintptr:
		return strconv.AppendUint([]byte{}, uint64(v), 10)
	case float32:
		return strconv.AppendFloat([]byte{}, float64(v), 'e', 6, 32)
	case float64:
		return strconv.AppendFloat([]byte{}, v, 'e', 6, 64)
	case string:
		return append([]byte{}, v...)
	default:
		if v2, ok := v.(*callable); ok {
			v = v2.reflectValue().Interface()
		}
		b := append([]byte{}, "("...)
		b = append([]byte{}, reflect.TypeOf(v).String()...)
		b = append([]byte{}, ")"...)
		return b
		// TODO(marco): implement print of value
	}
}

func (err *Panic) Error() string {
	b := make([]byte, 0, 100+len(err.StackTrace))
	b = append(b, sprint(err.Msg)...)
	b = append(b, "\n\n"...)
	b = append(b, err.StackTrace...)
	return string(b)
}

type Type int8

const (
	TypeInt Type = iota
	TypeFloat
	TypeString
	TypeIface
)

func (t Type) String() string {
	switch t {
	case TypeInt:
		return "int"
	case TypeFloat:
		return "float"
	case TypeString:
		return "string"
	case TypeIface:
		return "iface"
	}
	panic("unknown type")
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
	ConditionNil                                // x == nil
	ConditionNotNil                             // x != nil
	ConditionOK                                 // [vm.ok]
	ConditionNotOK                              // ![vm.ok]
)

func (c Condition) String() string {
	switch c {
	case ConditionEqual:
		return "Equal"
	case ConditionNotEqual:
		return "NotEqual"
	case ConditionLess:
		return "Less"
	case ConditionLessOrEqual:
		return "LessOrEqual"
	case ConditionGreater:
		return "Greater"
	case ConditionGreaterOrEqual:
		return "GreaterOrEqual"
	case ConditionEqualLen:
		return "EqualLen"
	case ConditionNotEqualLen:
		return "NotEqualLen"
	case ConditionLessLen:
		return "LessLen"
	case ConditionLessOrEqualLen:
		return "LessOrEqualLen"
	case ConditionGreaterLen:
		return "GreaterOrEqualLen"
	case ConditionGreaterOrEqualLen:
		return "GreaterOrEqualLen"
	case ConditionNil:
		return "Nil"
	case ConditionNotNil:
		return "NotNil"
	case ConditionOK:
		return "OK"
	case ConditionNotOK:
		return "NotOK"
	}
	panic("unknown condition")
}

type operation int8

const (
	opNone operation = iota

	opAddInt
	opAddInt8
	opAddInt16
	opAddInt32
	opAddFloat32
	opAddFloat64

	opAnd

	opAndNot

	opAssert
	opAssertInt
	opAssertFloat64
	opAssertString

	opAppend

	opAppendSlice

	opBind

	opCall

	opCallIndirect

	opCallNative

	opCap

	opContinue

	opConvert
	opConvertInt
	opConvertUint
	opConvertFloat
	opConvertString

	opCopy

	opConcat

	opDefer

	opDelete

	opDivInt
	opDivInt8
	opDivInt16
	opDivInt32
	opDivUint8
	opDivUint16
	opDivUint32
	opDivUint64
	opDivFloat32
	opDivFloat64

	opFunc

	opGetFunc

	opGetVar

	opGo

	opGoto

	opIf
	opIfInt
	opIfUint
	opIfFloat
	opIfString

	opIndex

	opLen

	opMakeChan

	opMakeMap

	opMakeSlice

	opMapIndex

	opMapIndexStringBool

	opMapIndexStringInt

	opMapIndexStringInterface

	opMapIndexStringString

	opMove

	opMulInt
	opMulInt8
	opMulInt16
	opMulInt32
	opMulFloat32
	opMulFloat64

	opNew

	opOr

	opPanic

	opPrint

	opRange

	opRangeString

	opReceive

	opRecover

	opRemInt
	opRemInt8
	opRemInt16
	opRemInt32
	opRemUint8
	opRemUint16
	opRemUint32
	opRemUint64

	opReturn

	opSelector

	opSend

	opSetMap

	opSetSlice

	opSetVar

	opSliceIndex

	opStringIndex

	opSubInt
	opSubInt8
	opSubInt16
	opSubInt32
	opSubFloat32
	opSubFloat64

	opSubInvInt
	opSubInvInt8
	opSubInvInt16
	opSubInvInt32
	opSubInvFloat32
	opSubInvFloat64

	opTailCall

	opXor
)

func (op operation) String() string {
	return operationName[op]
}

var operationName = [...]string{

	opNone: "Nop", // TODO(Gianluca): review.

	opAddInt:     "Add",
	opAddInt8:    "Add8",
	opAddInt16:   "Add16",
	opAddInt32:   "Add32",
	opAddFloat32: "Add32",
	opAddFloat64: "Add",

	opAnd: "And",

	opAndNot: "AndNot",

	opAppend: "Append",

	opAppendSlice: "AppendSlice",

	opAssert:        "Assert",
	opAssertInt:     "Assert",
	opAssertFloat64: "Assert",
	opAssertString:  "Assert",

	opBind: "Bind",

	opCall: "Call",

	opCallIndirect: "CallIndirect",

	opCallNative: "CallNative",

	opCap: "Cap",

	opContinue: "Continue",

	opConvert:       "Convert",
	opConvertInt:    "Convert",
	opConvertUint:   "ConvertU",
	opConvertFloat:  "Convert",
	opConvertString: "Convert",

	opCopy: "Copy",

	opConcat: "concat",

	opDefer: "Defer",

	opDelete: "delete",

	opDivInt:     "Div",
	opDivInt8:    "Div8",
	opDivInt16:   "Div16",
	opDivInt32:   "Div32",
	opDivUint8:   "DivU8",
	opDivUint16:  "DivU16",
	opDivUint32:  "DivU32",
	opDivUint64:  "DivU64",
	opDivFloat32: "Div32",
	opDivFloat64: "Div",

	opFunc: "Func",

	opGetFunc: "GetFunc",

	opGetVar: "GetVar",

	opGo: "Go",

	opGoto: "Goto",

	opIf:       "If",
	opIfInt:    "If",
	opIfUint:   "IfU",
	opIfFloat:  "If",
	opIfString: "If",

	opIndex: "Index",

	opLen: "Len",

	opMakeChan: "MakeChan",

	opMakeMap: "MakeMap",

	opMakeSlice: "MakeSlice",

	opMapIndex: "MapIndex",

	opMapIndexStringBool: "MapIndexStringBool",

	opMapIndexStringInt: "MapIndexStringInt",

	opMapIndexStringInterface: "MapIndexStringInterface",

	opMapIndexStringString: "MapIndexStringString",

	opMove: "Move",

	opMulInt:     "Mul",
	opMulInt8:    "Mul8",
	opMulInt16:   "Mul16",
	opMulInt32:   "Mul32",
	opMulFloat32: "Mul32",
	opMulFloat64: "Mul",

	opNew: "New",

	opOr: "Or",

	opPanic: "Panic",

	opPrint: "Print",

	opRange: "Range",

	opRangeString: "Range",

	opReceive: "Receive",

	opRecover: "Recover",

	opRemInt:    "Rem",
	opRemInt8:   "Rem8",
	opRemInt16:  "Rem16",
	opRemInt32:  "Rem32",
	opRemUint8:  "RemU8",
	opRemUint16: "RemU16",
	opRemUint32: "RemU32",
	opRemUint64: "RemU64",

	opReturn: "Return",

	opSelector: "Selector",

	opSend: "Send",

	opSetMap: "SetMap",

	opSetSlice: "SetSlice",

	opSetVar: "SetPackageVar",

	opSliceIndex: "SliceIndex",

	opStringIndex: "StringIndex",

	opSubInt:     "Sub",
	opSubInt8:    "Sub8",
	opSubInt16:   "Sub16",
	opSubInt32:   "Sub32",
	opSubFloat32: "Sub32",
	opSubFloat64: "Sub",

	opSubInvInt:     "SubInv",
	opSubInvInt8:    "SubInv8",
	opSubInvInt16:   "SubInv16",
	opSubInvInt32:   "SubInv32",
	opSubInvFloat32: "SubInv32",
	opSubInvFloat64: "SubInv",

	opTailCall: "TailCall",

	opXor: "Xor",
}
