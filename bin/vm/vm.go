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

	pkg := vm.NewPackage("main")

	inc := pkg.NewFunction("inc", []vm.Type{vm.TypeInt}, []vm.Type{vm.TypeInt}, false)
	b := inc.Builder()
	b.Add(true, 0, 1, 0, reflect.Int)
	b.Move(false, 0, 1, reflect.Int)
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
	b.Add(true, 0, 1, 0, reflect.Int) // i++
	// x = inc(x)
	b.Move(false, 1, 3, reflect.Int)
	b.Call(cInc)
	b.Move(false, 4, 1, reflect.Int)
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
