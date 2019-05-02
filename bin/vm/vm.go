//+build ignore

// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"reflect"

	"scrigo/vm"
)

func main() {

	vm.DebugTraceExecution = true

	_, _ = fmt.Fprint(os.Stderr, "Call example:\n")
	call()

	_, _ = fmt.Fprint(os.Stderr, "\n\nRecursive call example:\n")
	callRec()

	_, _ = fmt.Fprint(os.Stderr, "\n\nClosure example:\n")
	closure()

	return
}

func call() {

	// x := 0
	// for i := i; i < 5; i++ {
	//     x = inc(x)
	// }

	pkg := vm.NewPackage("main")

	inc := pkg.NewFunction("inc", []vm.Type{vm.TypeInt}, []vm.Type{vm.TypeInt}, false)
	b := inc.Builder()
	b.Add(true, 1, 1, 0, reflect.Int)
	b.Return()
	b.End()

	main := pkg.NewFunction("main", nil, nil, false)

	b = main.Builder()
	b.Move(true, 0, 0, reflect.Int) // i := 0
	b.Move(true, 0, 1, reflect.Int) // x := 0
	b.SetLabel()
	b.If(true, 1, vm.ConditionLess, 5, reflect.Int)
	b.Goto(3)
	b.SetLabel()
	// x = inc(x)
	b.Move(false, 1, 3, reflect.Int)
	b.GetFunc(0, 0, 0)
	b.Call(vm.NoPackage, 0, vm.StackShift{2}) // inc()
	b.Move(false, 2, 1, reflect.Int)
	//
	b.Add(true, 0, 1, 0, reflect.Int) // i++
	//
	b.Goto(1)
	b.SetLabel()
	b.Return()
	b.End()

	_, err := vm.Disassemble(os.Stdout, pkg)
	if err != nil {
		panic(err)
	}
	vm1 := vm.New(pkg)
	_, err = vm1.Run("main")
	if err != nil {
		panic(err)
	}

}

func callRec() {

	// func fib(n, a, b int) int {
	//     if n == 0 {
	//         return a
	//     } else if n == 1 {
	//         return b
	//     }
	//     return fib(n-1, b, a+b)
	// }
	//
	// fib(35, 0, 1)

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
	fb.TailCall(vm.CurrentPackage, vm.CurrentFunction)
	fb.End()

	main := pkg.NewFunction("main", nil, nil, false)
	fb = main.Builder()
	fb.Move(true, 35, 1, reflect.Int) // n := fibN
	fb.Move(true, 0, 2, reflect.Int)  // a := 0
	fb.Move(true, 1, 3, reflect.Int)  // b := 1
	fb.GetFunc(0, 0, 0)
	fb.Call(vm.CurrentPackage, 0, vm.StackShift{0})
	fb.Return()
	fb.End()

	_, err := vm.Disassemble(os.Stdout, pkg)
	if err != nil {
		panic(err)
	}
	vm1 := vm.New(pkg)
	_, err = vm1.Run("main")
	if err != nil {
		panic(err)
	}
}

func closure() {

	// func main() {
	//    a := 5
	//    inc := func() {
	//         a += 1
	//    }
	//    inc()

	pkg := vm.NewPackage("main")

	intType := reflect.TypeOf(0)

	main := pkg.NewFunction("main", nil, nil, false)
	b := main.Builder()
	b.Alloc(intType, -1)              // a := 0
	inc := b.Func(1, nil, nil, false) // inc := func() { ... }

	inc.SetClosureRefs([]int16{-1})
	b2 := inc.Builder()
	b2.GetClosureVar(0, 0)
	b2.Add(true, 0, 1, 0, reflect.Int)
	b2.SetClosureVar(0, 0)
	b2.Return()
	b2.End()

	b.Call(vm.NoPackage, 1, vm.StackShift{1, 0, 0, 1})
	b.Return()
	b.End()

	_, err := vm.Disassemble(os.Stdout, pkg)
	if err != nil {
		panic(err)
	}
	vm1 := vm.New(pkg)
	_, err = vm1.Run("main")
	if err != nil {
		panic(err)
	}
}
