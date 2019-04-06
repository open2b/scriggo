// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

const StackSize = 512

type VM struct {
	fp    [4]uint32 // frame pointers.
	ok    bool      // ok flag.
	regs  registers // registers.
	fn    *Function // function.
	calls []Call
	pkg   *Package
}

func New(pkg *Package) *VM {
	vm := &VM{}
	vm.pkg = pkg
	vm.regs.Int = make([]int64, StackSize)
	vm.regs.Float = make([]float64, StackSize)
	vm.regs.String = make([]string, StackSize)
	vm.regs.Misc = make([]interface{}, StackSize)
	vm.Reset()
	return vm
}

func (vm *VM) Reset() {
	vm.fp = [4]uint32{0, 0, 0, 0}
	vm.fn = nil
	vm.calls = vm.calls[:]
}

func (vm *VM) checkSplitStack(r int, num uint32) {
	switch r {
	case 0:
		if int(num) > len(vm.regs.Int) {
			stack := make([]int64, len(vm.regs.Int)*2)
			copy(stack, vm.regs.Int)
			vm.regs.Int = stack
		}
	}
}

type Call struct {
	fp [4]uint32 // frame pointers.
	pc uint32    // program counter.
	fn *Function // function.
}
