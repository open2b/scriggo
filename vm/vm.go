// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

const StackSize = 512

type VM struct {
	fp    [4]uint32 // frame pointers.
	pc    uint32    // program counter.
	ok    bool      // ok flag.
	regs  values    // values.
	fn    *Function // function.
	calls []Call
	pkg   *Package
}

func New(pkg *Package) *VM {
	vm := &VM{}
	vm.pkg = pkg
	vm.regs.t0 = make([]int64, StackSize)
	vm.regs.t1 = make([]float64, StackSize)
	vm.regs.t2 = make([]string, StackSize)
	vm.regs.t3 = make([]interface{}, StackSize)
	vm.Reset()
	return vm
}

func (vm *VM) Reset() {
	vm.fp = [4]uint32{0, 0, 0, 0}
	vm.pc = 0
	vm.fn = nil
	vm.calls = vm.calls[:]
}

func (vm *VM) checkSplitStack(r int, num uint32) {
	switch r {
	case 0:
		if int(num) > len(vm.regs.t0) {
			stack := make([]int64, len(vm.regs.t0)*2)
			copy(stack, vm.regs.t0)
			vm.regs.t0 = stack
		}
	}
}

type Call struct {
	fp [4]uint32 // frame pointers.
	pc uint32    // program counter.
	fn *Function // function.
}
