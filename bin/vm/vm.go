// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"reflect"
	"scrigo/vm"
)

func main() {

	callRec(35)
	return

	pkg := vm.NewPackage("main")

	inc := pkg.NewFunction("inc", []vm.Type{vm.TypeInt}, []vm.Type{vm.TypeInt}, false)
	b := inc.Builder()
	b.Add(true, 1, 1, 0, reflect.Int)
	b.Move(false, 1, 0, reflect.Int)
	b.Return()
	b.End()

	main := pkg.NewFunction("main", nil, nil, false)

	// x := 0
	// for i := i; i < 5; i++ {
	//     x = inc(x)
	// }

	b = main.Builder()
	cInc := b.MakeInterfaceConstant(inc)
	b.Move(true, 0, 0, reflect.Int) // i := 0
	b.Move(true, 0, 1, reflect.Int) // x := 0
	b.SetLabel()
	b.If(true, 1, vm.ConditionLess, 5, reflect.Int)
	b.Goto(3)
	b.SetLabel()
	// x = inc(x)
	b.Move(false, 1, 3, reflect.Int)
	b.Call(cInc, vm.StackShift{2})
	b.Move(false, 2, 1, reflect.Int)
	//
	b.Add(true, 0, 1, 0, reflect.Int) // i++
	//
	b.Goto(1)
	b.SetLabel()
	b.Return()
	b.End()

	vm.DebugTraceExecution = true

	_, err := vm.Disassemble(os.Stdout, pkg)
	if err != nil {
		panic(err)
	}
	vm1 := vm.New(pkg)
	_, err = vm1.Run("main")
	if err != nil {
		panic(err)
	}
	println()
	println()
}

func callRec(n int) {

	pkg := vm.NewPackage("main")

	fib := pkg.NewFunction("fib", []vm.Type{vm.TypeInt, vm.TypeInt, vm.TypeInt}, []vm.Type{vm.TypeInt}, false)
	fb := fib.Builder()

	fb.If(true, 1, vm.ConditionEqual, 0, reflect.Int) // if n == 0
	fb.Goto(1)
	fb.Move(false, 2, 0, reflect.Int) // return a
	fb.Return()
	fb.SetLabel()
	fb.If(true, 1, vm.ConditionEqual, 1, reflect.Int) // if n == 1
	fb.Goto(2)
	fb.Move(false, 3, 0, reflect.Int) // return b
	fb.Return()
	fb.SetLabel()
	fb.Sub(true, 1, 1, 1, reflect.Int)  // n = n-1
	fb.Move(false, 3, 4, reflect.Int)   // t = b
	fb.Add(false, 2, 3, 3, reflect.Int) // b = a+b
	fb.Move(false, 4, 2, reflect.Int)   // a = t
	fb.TailCall(vm.NoRegister)
	fb.End()

	main := pkg.NewFunction("main", nil, nil, false)
	fb = main.Builder()
	cFib := fb.MakeInterfaceConstant(fib)
	fb.Move(true, int8(n), 1, reflect.Int) // n := fibN
	fb.Move(true, 0, 2, reflect.Int)       // a := 0
	fb.Move(true, 1, 3, reflect.Int)       // b := 1
	fb.Call(cFib, vm.StackShift{0})
	fb.Return()
	fb.End()

	vm.DebugTraceExecution = true

	_, err := vm.Disassemble(os.Stdout, pkg)
	if err != nil {
		panic(err)
	}
	vm1 := vm.New(pkg)
	_, err = vm1.Run("main")
	if err != nil {
		panic(err)
	}
	println()
	println()
}
