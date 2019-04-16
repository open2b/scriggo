// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import "reflect"

const StackSize = 512

type VM struct {
	fp    [4]uint32       // frame pointers.
	st    [4]uint32       // stack tops.
	ok    bool            // ok flag.
	regs  registers       // registers.
	fn    *ScrigoFunction // current function.
	cvars []interface{}   // closure variables.
	calls []Call          // call stack.
	main  *Package        // package "main".
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
	vm.fn = nil
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
func (vm *VM) startScrigoGoroutine(pc uint32) bool {
	var fn *ScrigoFunction
	var vars []interface{}
	call := vm.fn.body[pc]
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
	pc++
	off := vm.fn.body[pc]
	copy(nvm.regs.Int, vm.regs.Int[vm.fp[0]+uint32(off.op):vm.fp[0]+127])
	copy(nvm.regs.Float, vm.regs.Float[vm.fp[1]+uint32(off.a):vm.fp[1]+127])
	copy(nvm.regs.String, vm.regs.String[vm.fp[2]+uint32(off.b):vm.fp[2]+127])
	copy(nvm.regs.General, vm.regs.General[vm.fp[3]+uint32(off.c):vm.fp[3]+127])
	go nvm.run()
	pc++
	return true
}
