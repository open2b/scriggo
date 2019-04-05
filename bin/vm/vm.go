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

var intType = reflect.TypeOf(0)

func main() {

	pkg := vm.NewPackage("main")

	inc := pkg.NewFunction("inc", []vm.Type{vm.TypeInt}, []vm.Type{vm.TypeInt}, false)
	b := inc.Builder()
	b.Add(0, 1, 0, reflect.Int)
	b.Move(0, 1, reflect.Int)
	b.Return()
	b.End()

	main := pkg.NewFunction("main", nil, nil, false)

	b = main.Builder()
	c0 := b.MakeIntConstant(0)
	b.Move(c0, 0, reflect.Int)
	b.Move(c0, 1, reflect.Int)
	b.SetLabel()
	c5 := b.MakeIntConstant(5)
	b.If(1, vm.ConditionLess, c5, reflect.Int)
	b.Goto(3)
	b.SetLabel()
	b.Add(1, 0, 0, reflect.Int)
	b.Move(1, 3, reflect.Int)
	cInc := b.MakeInterfaceConstant(inc)
	b.Call(cInc)
	b.Move(4, 1, reflect.Int)
	b.Goto(1)
	b.SetLabel()
	b.Return()
	b.End()

	vm.DebugTraceExecution = false

	_, err := vm.Disassemble(os.Stdout, pkg)
	if err != nil {
		panic(err)
	}
	//vm1 := vm.New(pkg)
	//_, err = vm1.Run("main")
	//if err != nil {
	//	panic(err)
	//}
	println()
	println()
}
