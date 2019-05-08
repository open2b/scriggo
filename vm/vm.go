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

const stackSize = 512
const maxInt8 = 128

type StackShift [4]int8

type Instruction struct {
	Op      Operation
	A, B, C int8
}

func DecodeAddr(a, b, c int8) uint32 {
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
	fp     [4]uint32            // frame pointers.
	st     [4]uint32            // stack tops.
	pc     uint32               // program counter.
	ok     bool                 // ok flag.
	regs   registers            // registers.
	fn     *ScrigoFunction      // current function.
	cvars  []interface{}        // closure variables.
	calls  []callFrame          // call stack frame.
	cases  []reflect.SelectCase // select cases.
	panics []Panic              // panics.
}

// New returns a new virtual machine.
func New() *VM {
	vm := &VM{}
	vm.regs.int = make([]int64, stackSize)
	vm.regs.float = make([]float64, stackSize)
	vm.regs.string = make([]string, stackSize)
	vm.regs.general = make([]interface{}, stackSize)
	vm.st[0] = stackSize
	vm.st[1] = stackSize
	vm.st[2] = stackSize
	vm.st[3] = stackSize
	vm.Reset()
	return vm
}

// Reset resets a virtual machine so that it is ready for a new call to Run.
func (vm *VM) Reset() {
	vm.fp = [4]uint32{0, 0, 0, 0}
	vm.pc = 0
	vm.fn = nil
	vm.cvars = nil
	if vm.calls != nil {
		vm.calls = vm.calls[:0]
	}
	if vm.cases != nil {
		vm.cases = vm.cases[:0]
	}
	if vm.panics != nil {
		vm.panics = vm.panics[:0]
	}
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

// callNative calls a native function. numVariadic is the number of actual
// variadic arguments, shift is the stack shift and newGoroutine reports
// whether a new goroutine must be started.
func (vm *VM) callNative(fn *NativeFunction, numVariadic int8, shift StackShift, newGoroutine bool) {
	fp := vm.fp
	vm.fp[0] += uint32(shift[0])
	vm.fp[1] += uint32(shift[1])
	vm.fp[2] += uint32(shift[2])
	vm.fp[3] += uint32(shift[3])
	if fn.Func != nil {
		if newGoroutine {
			switch f := fn.Func.(type) {
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
			default:
				fn.slow()
			}
		}
	}
	if fn.Func == nil {
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
	vm.calls = append(vm.calls, callFrame{fn: *fn, fp: vm.fp, pc: 0, status: deferred, variadics: numVariadic})
	if args[0] > 0 {
		stack := vm.regs.int[vm.fp[0]+1:]
		tot := shift[0] + args[0]
		copy(stack[shift[0]:], stack[:tot])
		copy(stack, stack[shift[0]:tot])
		vm.fp[0] += uint32(args[0])
	}
	if args[1] > 0 {
		stack := vm.regs.float[vm.fp[1]+1:]
		tot := shift[1] + args[1]
		copy(stack[shift[1]:], stack[:tot])
		copy(stack, stack[shift[1]:tot])
		vm.fp[1] += uint32(args[1])
	}
	if args[2] > 0 {
		stack := vm.regs.string[vm.fp[2]+1:]
		tot := shift[2] + args[2]
		copy(stack[shift[2]:], stack[:tot])
		copy(stack, stack[shift[2]:tot])
		vm.fp[2] += uint32(args[2])
	}
	if args[3] > 0 {
		stack := vm.regs.general[vm.fp[3]+1:]
		tot := shift[3] + args[3]
		copy(stack[shift[3]:], stack[:tot])
		copy(stack, stack[shift[3]:tot])
		vm.fp[3] += uint32(args[3])
	}
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
	var i int
	var call callFrame
	for i = len(vm.calls) - 1; i >= 0; i-- {
		call = vm.calls[i]
		switch call.status {
		case started:
			// A call is returned, continue with the previous call.
			// TODO(marco): call finalizer.
		case tailed:
			// A tail call is returned, continue with the previous call.
			// TODO(marco): call finalizer.
			continue
		case deferred:
			// A Scrigo call that has deferred calls is returned, its first
			// deferred call will be executed.
			call = vm.swapCall(call)
			vm.calls[i] = callFrame{fn: callable{scrigo: vm.fn}, fp: vm.fp, status: returned}
			if call.fn.scrigo != nil {
				break
			}
			vm.callNative(call.fn.native, call.variadics, StackShift{}, false)
			fallthrough
		case returned, recovered:
			// A deferred call is returned. If there is another deferred
			// call, it will be executed, otherwise the previous call will be
			// finalized.
			if i > 0 {
				if prev := vm.calls[i-1]; prev.status == deferred {
					call, vm.calls[i-1] = prev, call
					break
				}
			}
			// TODO(marco): call finalizer.
			if call.status == recovered {
				vm.panics = vm.panics[:len(vm.panics)-1]
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
// call at program counter pc. If the function is native, returns true.
func (vm *VM) startScrigoGoroutine() bool {
	var fn *ScrigoFunction
	var vars []interface{}
	call := vm.fn.Body[vm.pc]
	switch call.Op {
	case OpCall:
		fn = vm.fn.ScrigoFunctions[uint8(call.B)]
	case OpCallIndirect:
		f := vm.general(call.B).(*callable)
		if f.scrigo == nil {
			return true
		}
		fn = f.scrigo
		vars = f.vars
	default:
		return true
	}
	nvm := New()
	nvm.fn = fn
	nvm.cvars = vars
	vm.pc++
	off := vm.fn.Body[vm.pc]
	copy(nvm.regs.int, vm.regs.int[vm.fp[0]+uint32(off.Op):vm.fp[0]+127])
	copy(nvm.regs.float, vm.regs.float[vm.fp[1]+uint32(off.A):vm.fp[1]+127])
	copy(nvm.regs.string, vm.regs.string[vm.fp[2]+uint32(off.B):vm.fp[2]+127])
	copy(nvm.regs.general, vm.regs.general[vm.fp[3]+uint32(off.C):vm.fp[3]+127])
	go nvm.run()
	vm.pc++
	return false
}

func (vm *VM) swapCall(call callFrame) callFrame {
	if call.fp[0] < vm.fp[0] {
		a := uint32(vm.fp[0] - call.fp[0])
		b := uint32(vm.fn.RegNum[0])
		if vm.fp[0]+2*b > vm.st[0] {
			vm.moreIntStack()
		}
		s := vm.regs.int[call.fp[0]+1:]
		copy(s[a:], s[:a+b])
		copy(s, s[a+b:a+2*b])
		vm.fp[0] -= a
		call.fp[0] += b
	}
	if call.fp[1] < vm.fp[1] {
		a := uint32(vm.fp[1] - call.fp[1])
		b := uint32(vm.fn.RegNum[1])
		if vm.fp[1]+2*b > vm.st[1] {
			vm.moreFloatStack()
		}
		s := vm.regs.float[call.fp[1]+1:]
		copy(s[a:], s[:a+b])
		copy(s, s[a+b:a+2*b])
		vm.fp[1] -= a
		call.fp[1] += b
	}
	if call.fp[2] < vm.fp[2] {
		a := uint32(vm.fp[2] - call.fp[2])
		b := uint32(vm.fn.RegNum[2])
		if vm.fp[2]+2*b > vm.st[2] {
			vm.moreStringStack()
		}
		s := vm.regs.float[call.fp[2]+1:]
		copy(s[a:], s[:a+b])
		copy(s, s[a+b:a+2*b])
		vm.fp[2] -= a
		call.fp[2] += b
	}
	if call.fp[3] < vm.fp[3] {
		a := uint32(vm.fp[3] - call.fp[3])
		b := uint32(vm.fn.RegNum[3])
		if vm.fp[3]+2*b > vm.st[3] {
			vm.moreGeneralStack()
		}
		s := vm.regs.general[call.fp[3]+1:]
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
	Pkg    string
	Name   string
	Func   interface{}
	value  reflect.Value
	in     []Kind
	out    []Kind
	args   []reflect.Value
	outOff [4]int8
}

// Variable represents a global variable with a package, name and value.
// Value must a pointer to the variable value.
type Variable struct {
	Pkg   string
	Name  string
	Value interface{}
}

// ScrigoFunction represents a Scrigo function.
type ScrigoFunction struct {
	Pkg       string
	Name      string
	File      string
	Line      int
	Type      reflect.Type
	Parent    *ScrigoFunction
	CRefs     []int16
	Literals  []*ScrigoFunction
	Types     []reflect.Type
	RegNum    [4]uint8
	Constants struct {
		Int     []int64
		Float   []float64
		String  []string
		General []interface{}
	}
	Variables       []Variable
	ScrigoFunctions []*ScrigoFunction
	NativeFunctions []*NativeFunction
	Body            []Instruction
	Lines           map[uint32]int
}

func (fn *NativeFunction) slow() {
	if !fn.value.IsValid() {
		fn.value = reflect.ValueOf(fn.Func)
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
	fn.Func = nil
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

type callFrame struct {
	fn        callable   // function.
	fp        [4]uint32  // frame pointers.
	pc        uint32     // program counter.
	status    callStatus // status.
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
			c.native.value = reflect.ValueOf(c.native.Func)
		}
		c.value = c.native.value
		return c.value
	}
	// It is a Scrigo function.
	fn := c.scrigo
	cvars := c.vars
	c.value = reflect.MakeFunc(fn.Type, func(args []reflect.Value) []reflect.Value {
		nvm := New()
		nvm.fn = fn
		nvm.cvars = cvars
		nOut := fn.Type.NumOut()
		results := make([]reflect.Value, nOut)
		for i := 0; i < nOut; i++ {
			t := fn.Type.Out(i)
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
			t := fn.Type.Out(i)
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

type Operation int8

const (
	OpNone Operation = iota

	OpAddInt
	OpAddInt8
	OpAddInt16
	OpAddInt32
	OpAddFloat32
	OpAddFloat64

	OpAnd

	OpAndNot

	OpAssert
	OpAssertInt
	OpAssertFloat64
	OpAssertString

	OpAppend

	OpAppendSlice

	OpBind

	OpCall

	OpCallIndirect

	OpCallNative

	OpCap

	OpCase

	OpContinue

	OpConvert
	OpConvertInt
	OpConvertUint
	OpConvertFloat
	OpConvertString

	OpCopy

	OpConcat

	OpDefer

	OpDelete

	OpDivInt
	OpDivInt8
	OpDivInt16
	OpDivInt32
	OpDivUint8
	OpDivUint16
	OpDivUint32
	OpDivUint64
	OpDivFloat32
	OpDivFloat64

	OpFunc

	OpGetFunc

	OpGetVar

	OpGo

	OpGoto

	OpIf
	OpIfInt
	OpIfUint
	OpIfFloat
	OpIfString

	OpIndex

	OpLeftShift
	OpLeftShift8
	OpLeftShift16
	OpLeftShift32

	OpLen

	OpLoadNumber

	OpMakeChan

	OpMapIndex

	OpMakeMap

	OpMakeSlice

	OpMove

	OpMulInt
	OpMulInt8
	OpMulInt16
	OpMulInt32
	OpMulFloat32
	OpMulFloat64

	OpNew

	OpOr

	OpPanic

	OpPrint

	OpRange

	OpRangeString

	OpReceive

	OpRecover

	OpRemInt
	OpRemInt8
	OpRemInt16
	OpRemInt32
	OpRemUint8
	OpRemUint16
	OpRemUint32
	OpRemUint64

	OpReturn

	OpRightShift
	OpRightShiftU

	OpSelect

	OpSelector

	OpSend

	OpSetMap

	OpSetSlice

	OpSetVar

	OpSliceIndex

	OpStringIndex

	OpSubInt
	OpSubInt8
	OpSubInt16
	OpSubInt32
	OpSubFloat32
	OpSubFloat64

	OpSubInvInt
	OpSubInvInt8
	OpSubInvInt16
	OpSubInvInt32
	OpSubInvFloat32
	OpSubInvFloat64

	OpTailCall

	OpXor
)
