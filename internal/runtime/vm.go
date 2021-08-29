// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"context"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/native"
)

const NoVariadicArgs = -1
const CurrentFunction = -1
const ReturnString = -1

const maxUint32 = 1<<31 - 1

const stackSize = 512

var envType = reflect.TypeOf((*native.Env)(nil)).Elem()
var emptyInterfaceType = reflect.TypeOf(&[]interface{}{nil}[0]).Elem()
var emptyInterfaceNil = reflect.ValueOf(&[]interface{}{nil}[0]).Elem()

// Converter is implemented by format converters.
type Converter func(src []byte, out io.Writer) error

// A TypeOfFunc function returns a type of a value.
type TypeOfFunc func(reflect.Value) reflect.Type

// A ScriggoType represents a type compiled by Scriggo from a type definition
// or a composite type literal with at least one element with a Scriggo type.
type ScriggoType interface {
	reflect.Type

	// Wrap wraps a value with a Scriggo type putting into a proxy that exposes
	// methods to Go.
	Wrap(reflect.Value) reflect.Value

	// Unwrap unwraps a value that has been read from Go. If the value given as
	// parameter can be unwrapped using the unwrapper's type, the unwrapped
	// value is returned and the method returns true.
	Unwrap(reflect.Value) (reflect.Value, bool)

	// GoType returns the Go type of a Scriggo type. Note that the
	// implementation of the reflect.Type returned by GoType is the
	// implementation of the package 'reflect', so it's safe to pass the
	// returned value to reflect functions and methods as argument.
	GoType() reflect.Type
}

// isZeroType is the reflect.Type representing the interface:
//
//     interface {
//         IsZero() bool
//     }
//
// If a value implements this interface, the method 'IsZero() bool' is used by
// the virtual machine to determine if a value is zero or not.
var isZeroType = reflect.TypeOf((*interface{ IsZero() bool })(nil)).Elem()

type StackShift [4]int8

type Instruction struct {
	Op      Operation
	A, B, C int8
}

func decodeInt16(a, b int8) int16 {
	return int16(int(a)<<8 | int(uint8(b)))
}

func decodeUint16(a, b int8) uint16 {
	return uint16(uint8(a))<<8 | uint16(uint8(b))
}

func decodeUint24(a, b, c int8) uint32 {
	return uint32(uint8(a))<<16 | uint32(uint8(b))<<8 | uint32(uint8(c))
}

func decodeValueIndex(a, b int8) (t registerType, i int) {
	return registerType(uint8(a) >> 6), int(decodeUint16(a, b) &^ (3 << 14))
}

// VM represents a Scriggo virtual machine.
type VM struct {
	fp       [4]Addr              // frame pointers.
	st       [4]Addr              // stack tops.
	pc       Addr                 // program counter.
	ok       bool                 // ok flag.
	regs     registers            // registers.
	fn       *Function            // running function.
	vars     []reflect.Value      // global and closure variables.
	env      *env                 // execution environment.
	envArg   reflect.Value        // execution environment as argument.
	renderer *renderer            // renderer
	calls    []callFrame          // call stack frame.
	cases    []reflect.SelectCase // select cases.
	panic    *PanicError          // panic.
	main     bool                 // reports whether this VM is executing the the main goroutine.
}

// NewVM returns a new virtual machine.
func NewVM() *VM {
	vm := create(&env{})
	vm.main = true
	return vm
}

// Reset resets a virtual machine so that it is ready for a new call to Run.
func (vm *VM) Reset() {
	vm.fp = [4]Addr{0, 0, 0, 0}
	vm.st[0] = Addr(len(vm.regs.int))
	vm.st[1] = Addr(len(vm.regs.float))
	vm.st[2] = Addr(len(vm.regs.string))
	vm.st[3] = Addr(len(vm.regs.general))
	vm.pc = 0
	vm.ok = false
	vm.fn = nil
	vm.vars = nil
	vm.env = &env{}
	vm.envArg = reflect.ValueOf(vm.env)
	vm.renderer = nil
	if vm.calls != nil {
		vm.calls = vm.calls[:0]
	}
	if vm.cases != nil {
		vm.cases = vm.cases[:0]
	}
	vm.panic = nil
}

// Run starts the execution of the function fn with the given global variables
// and waits for it to complete.
//
// During the execution if a panic occurs and has not been recovered, by
// default Run panics with the panic message.
//
// If a context has been set and the context is canceled, Run returns
// as soon as possible with the error returned by the Err method of the
// context.
func (vm *VM) Run(fn *Function, typeof TypeOfFunc, globals []reflect.Value) (int, error) {
	if typeof == nil {
		typeof = typeOfFunc
	}
	vm.env.typeof = typeof
	vm.env.globals = globals
	err := vm.runFunc(fn, globals)
	if err != nil {
		switch e := err.(type) {
		case *PanicError:
			if outErr, ok := e.message.(outError); ok {
				err = outErr.err
			}
		case *fatalError:
			panic(e.msg)
		case exitError:
			return int(e), nil
		}
		return 0, err
	}
	return 0, nil
}

// SetContext sets the context.
//
// SetContext must not be called after vm has been started.
func (vm *VM) SetContext(ctx context.Context) {
	vm.env.ctx = ctx
}

// SetRenderer sets template output and markdown converter.
//
// SetRenderer must not be called after vm has been started.
func (vm *VM) SetRenderer(out io.Writer, conv Converter) {
	vm.renderer = newRenderer(vm.env, out, conv)
}

// SetPrint sets the "print" builtin function.
//
// SetPrint must not be called after vm has been started.
func (vm *VM) SetPrint(p PrintFunc) {
	vm.env.print = p
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
		var ppc Addr
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
		if debugInfo, ok := fn.DebugInfo[ppc]; ok {
			write(strconv.Itoa(debugInfo.Position.Line))
		} else {
			write("???")
		}
		if len(b) == len(buf) {
			break
		}
	}
	return len(b)
}

// callNative calls a native function. numVariadic is the number of variadic
// arguments, shift is the stack shift and asGoroutine reports whether the
// function must be started as a goroutine.
//
// When callNative is called, vm.pc must be the address of the call
// instruction plus one.
func (vm *VM) callNative(fn *NativeFunction, numVariadic int8, shift StackShift, asGoroutine bool) {

	if fn.value.IsNil() {
		panic(errNilPointer)
	}

	// Make a copy of the frame pointer.
	fp := vm.fp

	// Shift the frame pointer.
	vm.fp[0] += Addr(shift[0])
	vm.fp[1] += Addr(shift[1])
	vm.fp[2] += Addr(shift[2])
	vm.fp[3] += Addr(shift[3])

	// Call the function without the reflect.
	if !fn.reflectCall {
		if asGoroutine {
			switch f := fn.function.(type) {
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
			default:
				panic("unexpected")
			}
		} else {
			switch f := fn.function.(type) {
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
			default:
				panic("unexpected")
			}
		}
		vm.fp = fp
		return
	}

	// Call the function with reflect.
	var args []reflect.Value

	typ := fn.value.Type()
	nunIn := typ.NumIn()
	variadic := typ.IsVariadic()

	if nunIn > 0 {

		// Shift the frame pointer.
		vm.fp[0] += Addr(fn.outOff[0])
		vm.fp[1] += Addr(fn.outOff[1])
		vm.fp[2] += Addr(fn.outOff[2])
		vm.fp[3] += Addr(fn.outOff[3])

		// Get a slice of reflect.Value for the arguments.
		args = fn.argsPool.Get().([]reflect.Value)

		// Prepare the arguments.
		lastNonVariadic := nunIn
		if variadic && numVariadic != NoVariadicArgs {
			lastNonVariadic--
		}
		for i := 0; i < nunIn; i++ {
			if i < lastNonVariadic {
				if i < 2 && typ.In(i) == envType {
					// Set the path of the file that contains the call.
					if vm.main {
						env := vm.env
						env.mu.Lock()
						env.filePath = vm.fn.DebugInfo[vm.pc-1].Path
						env.mu.Unlock()
					}
					args[i].Set(vm.envArg)
				} else {
					r := vm.getIntoReflectValue(1, args[i], false)
					vm.fp[r]++
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
						f := vm.general(int8(j + 1)).Interface().(*callable)
						slice.Index(j).Set(f.Value(vm.renderer, vm.env))
					}
				case reflect.String:
					for j := 0; j < int(numVariadic); j++ {
						slice.Index(j).SetString(vm.string(int8(j + 1)))
					}
				case reflect.Interface:
					for j := 0; j < int(numVariadic); j++ {
						if v := vm.general(int8(j + 1)); !v.IsValid() {
							if t := slice.Index(j).Type(); t == emptyInterfaceType {
								slice.Index(j).Set(emptyInterfaceNil)
							} else {
								slice.Index(j).Set(reflect.Zero(t))
							}
						} else {
							slice.Index(j).Set(v)
						}
					}
				default:
					for j := 0; j < int(numVariadic); j++ {
						slice.Index(j).Set(vm.general(int8(j + 1)))
					}
				}
				args[i].Set(slice)
			}
		}

		// Shift the frame pointer.
		vm.fp[0] = fp[0] + Addr(shift[0])
		vm.fp[1] = fp[1] + Addr(shift[1])
		vm.fp[2] = fp[2] + Addr(shift[2])
		vm.fp[3] = fp[3] + Addr(shift[3])

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
		var out []reflect.Value
		if variadic {
			out = fn.value.CallSlice(args)
		} else {
			out = fn.value.Call(args)
		}
		for _, arg := range out {
			r := vm.setFromReflectValue(1, arg)
			vm.fp[r]++
		}

		if args != nil {
			fn.argsPool.Put(args)
		}

	}

	vm.fp = fp // Restore the frame pointer.

	return
}

// equals reports whether x and y are equal.
// It panics if x and y are not comparable.
//
// If x and y are not interfaces and are guaranteed to have the same type,
// x.Interface() == y.Interface() can be used as a faster alternative.
func (vm *VM) equals(x, y reflect.Value) bool {
	if x.IsValid() != y.IsValid() {
		return false
	}
	if !x.IsValid() {
		return true
	}
	tx := vm.env.typeof(x)
	ty := vm.env.typeof(y)
	if tx != ty {
		return false
	}
	if t, ok := tx.(ScriggoType); ok {
		if !tx.Comparable() {
			panic("runtime error: comparing uncomparable type " + tx.String())
		}
		x, _ = t.Unwrap(x)
		y, _ = t.Unwrap(y)
	}
	return x.Interface() == y.Interface()
}

// fieldByIndex returns the field of the struct s at index i.
// s can be a pointer to a struct value.
//
// It panics with errNilPointer if s represents a nil pointer or accessing
// to a nil embedded struct.
func (vm *VM) fieldByIndex(s reflect.Value, i uint8) reflect.Value {
	v := s
	for _, x := range vm.fn.FieldIndexes[i] {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				panic(errNilPointer)
			}
			v = v.Elem()
		}
		v = v.Field(x)
	}
	return v
}

func (vm *VM) finalize(regs [][2]int8) {
	for _, reg := range regs {
		vm.setFromReflectValue(reg[1], vm.generalIndirect(reg[0]))
	}
}

func (vm *VM) moreIntStack() {
	top := len(vm.regs.int) * 2
	stack := make([]int64, top)
	copy(stack, vm.regs.int)
	vm.regs.int = stack
	vm.st[0] = Addr(top)
}

func (vm *VM) moreFloatStack() {
	top := len(vm.regs.float) * 2
	stack := make([]float64, top)
	copy(stack, vm.regs.float)
	vm.regs.float = stack
	vm.st[1] = Addr(top)
}

func (vm *VM) moreStringStack() {
	top := len(vm.regs.string) * 2
	stack := make([]string, top)
	copy(stack, vm.regs.string)
	vm.regs.string = stack
	vm.st[2] = Addr(top)
}

func (vm *VM) moreGeneralStack() {
	top := len(vm.regs.general) * 2
	stack := make([]reflect.Value, top)
	copy(stack, vm.regs.general)
	vm.regs.general = stack
	vm.st[3] = Addr(top)
}

func (vm *VM) nextCall() bool {
	for i := len(vm.calls) - 1; i >= 0; i-- {
		call := vm.calls[i]
		switch call.status {
		case started:
			// A call is returned, continue with the previous call.
		case tailed:
			// A tail call is returned, continue with the previous call.
			continue
		case deferred:
			// A call, that has deferred calls, is returned, its first
			// deferred call will be executed.
			current := callFrame{cl: callable{fn: vm.fn}, renderer: vm.renderer, fp: vm.fp, status: returned}
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
			if regs := call.cl.fn.FinalRegs; regs != nil {
				vm.fp = call.fp
				vm.finalize(regs)
			}
			if call.status == recovered {
				numPanicked := 0
				for _, c := range vm.calls {
					if c.status == panicked {
						numPanicked++
					}
				}
				num := 0
				for p := vm.panic; p != nil; p = p.next {
					num++
				}
				for p := vm.panic; num > numPanicked; num-- {
					p = p.next
					vm.panic = p
				}
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
				vm.renderer = call.renderer
				return true
			}
			vm.fp = call.fp
			vm.callNative(call.cl.Native(), call.numVariadic, StackShift{}, false)
		}
	}
	return false
}

// create creates a new virtual machine with the execution environment env.
func create(env *env) *VM {
	vm := &VM{
		st: [4]Addr{stackSize, stackSize, stackSize, stackSize},
		regs: registers{
			int:     make([]int64, stackSize),
			float:   make([]float64, stackSize),
			string:  make([]string, stackSize),
			general: make([]reflect.Value, stackSize),
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
// counter pc. If the function is native, returns true.
func (vm *VM) startGoroutine() bool {
	var fn *Function
	var vars []reflect.Value
	call := vm.fn.Body[vm.pc]
	switch call.Op {
	case OpCallFunc:
		fn = vm.fn.Functions[uint8(call.A)]
		vars = vm.env.globals
	case OpCallIndirect:
		f := vm.general(call.A).Interface().(*callable)
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
	copy(nvm.regs.int, vm.regs.int[vm.fp[0]+Addr(off.Op):vm.fp[0]+127])
	copy(nvm.regs.float, vm.regs.float[vm.fp[1]+Addr(off.A):vm.fp[1]+127])
	copy(nvm.regs.string, vm.regs.string[vm.fp[2]+Addr(off.B):vm.fp[2]+127])
	copy(nvm.regs.general, vm.regs.general[vm.fp[3]+Addr(off.C):vm.fp[3]+127])
	go nvm.runFunc(fn, vars)
	vm.pc++
	return false
}

// swapStack swaps the stacks pointed by a and b. bSize is the size of the
// stack pointed by b. The stacks must be consecutive and a must precede b.
//
// A stack can have zero size, so a and b can point to the same index.
func (vm *VM) swapStack(a, b *[4]Addr, bSize StackShift) {

	// Swap int registers.
	as := b[0] - a[0]
	bs := Addr(bSize[0])
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
	bs = Addr(bSize[1])
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
	bs = Addr(bSize[2])
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
	bs = Addr(bSize[3])
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
	General []reflect.Value
}

// macroOutBuffer is used in CallMacro and CallIndirect instructions to buffer
// the out of a macro call and convert it to a string in Return instructions.
type macroOutBuffer struct {
	strings.Builder
}

type NativeFunction struct {
	pkg         string        // package.
	name        string        // name.
	function    interface{}   // value.
	outOff      [4]int8       // offset of out arguments.
	value       reflect.Value // reflect value.
	reflectCall bool          // reports whether it can be called only with reflect.
	argsPool    *sync.Pool    // pool of arguments for reflect.Call and reflect.CallSlice.
}

// NewNativeFunction returns a new native function given its package and name
// and the function value or its reflect value. pkg and name can be empty
// strings.
func NewNativeFunction(pkg, name string, function interface{}) *NativeFunction {
	fn := &NativeFunction{
		pkg:  pkg,
		name: name,
	}
	if value, ok := function.(reflect.Value); ok {
		fn.function = value.Interface()
		fn.value = value
	} else {
		fn.function = function
		fn.value = reflect.ValueOf(function)
	}
	typ := fn.value.Type()
	nOut := typ.NumOut()
	for i := 0; i < nOut; i++ {
		k := typ.Out(i).Kind()
		fn.outOff[kindToType[k]]++
	}
	switch fn.function.(type) {
	case func(string) int:
	case func(string) string:
	case func(string, string) int:
	case func(string, int) string:
	case func(string, string) bool:
	default:
		fn.reflectCall = true
		if numIn := typ.NumIn(); numIn > 0 {
			fn.argsPool = &sync.Pool{
				New: func() interface{} {
					args := make([]reflect.Value, numIn)
					for i := 0; i < numIn; i++ {
						t := typ.In(i)
						args[i] = reflect.New(t).Elem()
					}
					return args
				},
			}
		}
	}
	return fn
}

func (fn *NativeFunction) Package() string {
	return fn.pkg
}

func (fn *NativeFunction) Name() string {
	return fn.name
}

func (fn *NativeFunction) Func() interface{} {
	return fn.function
}

// Function represents a function.
type Function struct {
	Pkg             string
	Name            string
	File            string
	Pos             *Position // position of the function declaration.
	Type            reflect.Type
	Parent          *Function
	VarRefs         []int16
	Types           []reflect.Type
	NumReg          [4]int8
	FinalRegs       [][2]int8 // [indirect -> return parameter registers]
	Macro           bool
	Format          ast.Format
	Values          Registers
	FieldIndexes    [][]int
	Functions       []*Function
	NativeFunctions []*NativeFunction
	Body            []Instruction
	Text            [][]byte
	DebugInfo       map[Addr]DebugInfo
}

// Position represents a source position.
type Position struct {
	Line   int // line starting from 1
	Column int // column in characters starting from 1
	Start  int // index of the first byte
	End    int // index of the last byte
}

func (p Position) String() string {
	return strconv.Itoa(p.Line) + ":" + strconv.Itoa(p.Column)
}

// DebugInfo represents a set of debug information associated to a given
// instruction. None of the fields below is mandatory.
type DebugInfo struct {
	Position    Position        // position of the instruction in the source code.
	Path        string          // path of the source code where the instruction is located in.
	OperandKind [3]reflect.Kind // kind of operands A, B and C.
	FuncType    reflect.Type    // type of the function that is called; only for call instructions.
}

type Addr uint32

type callStatus int8

const (
	started callStatus = iota
	tailed
	returned
	deferred
	panicked
	recovered
)

// If the size of callFrame changes, update the constant CallFrameSize.
type callFrame struct {
	cl          callable   // callable.
	renderer    *renderer  // renderer
	fp          [4]Addr    // frame pointers.
	pc          Addr       // program counter.
	status      callStatus // status.
	numVariadic int8       // number of variadic arguments.
}

type callable struct {
	value  reflect.Value   // reflect value.
	fn     *Function       // function, if it is a Scriggo function.
	native *NativeFunction // native function.
	vars   []reflect.Value // non-local (global and closure) variables.
}

// Native returns the native function of a callable.
func (c *callable) Native() *NativeFunction {
	if c.native != nil {
		return c.native
	}
	c.native = NewNativeFunction("", "", c.value)
	return c.native
}

// Value returns a reflect Value of a callable, so it can be called from a
// native code and passed to a native code.
func (c *callable) Value(renderer *renderer, env *env) reflect.Value {
	if c.value.IsValid() {
		return c.value
	}
	if c.native != nil {
		// It is a native function.
		c.value = reflect.ValueOf(c.native.function)
		return c.value
	}
	// It is a Scriggo function.
	fn := c.fn
	vars := c.vars
	c.value = reflect.MakeFunc(fn.Type, func(args []reflect.Value) []reflect.Value {
		nvm := create(env)
		if fn.Macro {
			renderer = renderer.WithOut(&macroOutBuffer{})
		}
		nvm.renderer = renderer
		nOut := fn.Type.NumOut()
		results := make([]reflect.Value, nOut)
		var r = [4]int8{1, 1, 1, 1}
		for i := 0; i < nOut; i++ {
			typ := fn.Type.Out(i)
			results[i] = reflect.New(typ).Elem()
			t := kindToType[typ.Kind()]
			r[t]++
		}
		for _, arg := range args {
			t := kindToType[arg.Kind()]
			nvm.setFromReflectValue(r[t], arg)
			r[t]++
		}
		err := nvm.runFunc(fn, vars)
		if err != nil {
			if p, ok := err.(*PanicError); ok {
				var msg string
				for ; p != nil; p = p.next {
					msg = "\n" + msg
					if p.recovered {
						msg = " [recovered]" + msg
					}
					msg = p.String() + msg
					if p.next != nil {
						msg = "\tpanic: " + msg
					}
				}
				err = &fatalError{msg: msg}
			}
			panic(err)
		}
		if fn.Macro {
			b := renderer.Out().(*macroOutBuffer)
			nvm.setString(1, b.String())
			err := renderer.Close()
			if err != nil {
				panic(&fatalError{env: env, msg: err})
			}
		}
		r = [4]int8{1, 1, 1, 1}
		for _, result := range results {
			t := kindToType[result.Kind()]
			nvm.getIntoReflectValue(r[t], result, false)
			r[t]++
		}
		return results
	})
	return c.value
}

func packageName(pkg string) string {
	for i := len(pkg) - 1; i >= 0; i-- {
		if pkg[i] == '/' {
			return pkg[i+1:]
		}
	}
	return pkg
}

type registerType int8

const (
	intRegister registerType = iota
	floatRegister
	stringRegister
	generalRegister
)

// kindToType maps a reflect kind to the corresponding internal register type.
var kindToType = [...]registerType{
	reflect.Float32:       floatRegister,
	reflect.Float64:       floatRegister,
	reflect.Complex64:     generalRegister,
	reflect.Complex128:    generalRegister,
	reflect.Array:         generalRegister,
	reflect.Chan:          generalRegister,
	reflect.Func:          generalRegister,
	reflect.Interface:     generalRegister,
	reflect.Map:           generalRegister,
	reflect.Ptr:           generalRegister,
	reflect.Slice:         generalRegister,
	reflect.String:        stringRegister,
	reflect.Struct:        generalRegister,
	reflect.UnsafePointer: generalRegister,
}

type Condition int8

const (
	ConditionZero                 Condition = iota // x == 0
	ConditionNotZero                               // x != 0
	ConditionEqual                                 // x == y
	ConditionNotEqual                              // x != y
	ConditionInterfaceEqual                        // x == y
	ConditionInterfaceNotEqual                     // x != y
	ConditionLess                                  // x <  y
	ConditionLessEqual                             // x <= y
	ConditionGreater                               // x >  y
	ConditionGreaterEqual                          // x >= y
	ConditionLessU                                 // x <  y (unsigned)
	ConditionLessEqualU                            // x <= y (unsigned)
	ConditionGreaterU                              // x >  y (unsigned)
	ConditionGreaterEqualU                         // x >= y (unsigned)
	ConditionLenEqual                              // len(x) == y
	ConditionLenNotEqual                           // len(x) != y
	ConditionLenLess                               // len(x) <  y
	ConditionLenLessEqual                          // len(x) <= y
	ConditionLenGreater                            // len(x) >  y
	ConditionLenGreaterEqual                       // len(x) >= y
	ConditionInterfaceNil                          // x == nil
	ConditionInterfaceNotNil                       // x != nil
	ConditionNil                                   // x == nil
	ConditionNotNil                                // x != nil
	ConditionContainsSubstring                     // x contains y
	ConditionContainsRune                          // x contains y
	ConditionContainsElement                       // x contains y
	ConditionContainsKey                           // x contains y
	ConditionContainsNil                           // x contains nil
	ConditionNotContainsSubstring                  // x not contains y
	ConditionNotContainsRune                       // x not contains y
	ConditionNotContainsElement                    // x not contains y
	ConditionNotContainsKey                        // x not contains y
	ConditionNotContainsNil                        // x not contains nil
	ConditionOK                                    // [vm.ok]
	ConditionNotOK                                 // ![vm.ok]
)

type Operation int8

const (
	OpNone Operation = iota

	OpAdd
	OpAddInt
	OpAddFloat64

	OpAddr

	OpAnd

	OpAndNot

	OpAssert

	OpAppend

	OpAppendSlice

	OpBreak

	OpCallFunc
	OpCallIndirect
	OpCallMacro
	OpCallNative

	OpCap

	OpCase

	OpClose

	OpComplex64
	OpComplex128

	OpConcat

	OpContinue

	OpConvert
	OpConvertInt
	OpConvertUint
	OpConvertFloat
	OpConvertString

	OpCopy

	OpDefer

	OpDelete

	OpDiv
	OpDivInt
	OpDivFloat64

	OpField

	OpGetVar

	OpGetVarAddr

	OpGo

	OpGoto

	OpIf
	OpIfInt
	OpIfFloat
	OpIfString

	OpIndex
	OpIndexString

	OpIndexRef

	OpLen

	OpLoad

	OpLoadFunc

	OpMakeArray

	OpMakeChan

	OpMakeMap

	OpMakeSlice

	OpMakeStruct

	OpMapIndex

	OpMethodValue

	OpMove

	OpMul
	OpMulInt
	OpMulFloat64

	OpNeg

	OpNew

	OpOr

	OpPanic

	OpPrint

	OpRange

	OpRangeString

	OpRealImag

	OpReceive

	OpRecover

	OpRem
	OpRemInt

	OpReturn

	OpSelect

	OpSend

	OpSetField

	OpSetMap

	OpSetSlice

	OpSetVar

	OpShl
	OpShlInt

	OpShow

	OpShr
	OpShrInt

	OpSlice

	OpStringSlice

	OpSub
	OpSubInt
	OpSubFloat64

	OpSubInv
	OpSubInvInt
	OpSubInvFloat64

	OpTailCall

	OpText

	OpTypify

	OpXor

	OpZero
)
