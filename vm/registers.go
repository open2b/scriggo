// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import "reflect"

type values struct {
	t0 []int64
	t1 []float64
	t2 []string
	t3 []interface{}
}

func (vm *VM) set(r int8, v reflect.Value) {
	k := v.Kind()
	if reflect.Int <= k && k <= reflect.Int64 {
		vm.setInt(r, v.Int())
	} else if reflect.Uint <= k && k <= reflect.Uint64 {
		vm.setInt(r, int64(v.Uint()))
	} else if k == reflect.String {
		vm.setString(r, v.String())
	} else if k == reflect.Float64 || k == reflect.Float32 {
		vm.setFloat(r, v.Float())
	} else {
		vm.setValue(r, v.Interface())
	}
}

func (vm *VM) int(r int8) int64 {
	if r >= 0 {
		return vm.regs.t0[vm.fp[0]+uint32(r)]
	}
	return *(vm.fn.refs.t0[-r-1])
}

func (vm *VM) intk(r int8, k bool) int64 {
	if k {
		if r >= 0 {
			return int64(r)
		}
		return vm.fn.constants.t0[-r]
	}
	if r >= 0 {
		return vm.regs.t0[vm.fp[0]+uint32(r)]
	}
	return *(vm.fn.refs.t0[-r-1])
}

func (vm *VM) bool(r int8) bool {
	if r >= 0 {
		return vm.regs.t0[vm.fp[0]+uint32(r)] > 0
	}
	return *(vm.fn.refs.t0[-r-1]) > 0
}

func (vm *VM) boolk(r int8, k bool) bool {
	if k {
		if r >= 0 {
			return r > 0
		}
		return vm.fn.constants.t0[-r] > 0
	}
	if r >= 0 {
		return vm.regs.t0[vm.fp[0]+uint32(r)] > 0
	}
	return *(vm.fn.refs.t0[-r-1]) > 0
}

func (vm *VM) setInt(r int8, i int64) {
	if r >= 0 {
		vm.regs.t0[vm.fp[0]+uint32(r)] = i
	} else {
		*(vm.fn.refs.t0[-r-1]) = i
	}
}

func (vm *VM) setBool(r int8, b bool) {
	v := int64(0)
	if b {
		v = 1
	}
	if r >= 0 {
		vm.regs.t0[vm.fp[0]+uint32(r)] = v
	} else {
		*(vm.fn.refs.t0[-r-1]) = v
	}
}

func (vm *VM) incInt(r int8) {
	if r >= 0 {
		vm.regs.t0[vm.fp[0]+uint32(r)]++
	} else {
		*(vm.fn.refs.t0[-r-1])++
	}
}

func (vm *VM) decInt(r int8) {
	if r >= 0 {
		vm.regs.t0[vm.fp[0]+uint32(r)]--
	} else {
		*(vm.fn.refs.t0[-r-1])--
	}
}

func (vm *VM) float(r int8) float64 {
	if r >= 0 {
		return vm.regs.t1[vm.fp[1]+uint32(r)]
	}
	return *(vm.fn.refs.t1[-r-1])
}

func (vm *VM) floatk(r int8, k bool) float64 {
	if k {
		if r >= 0 {
			return float64(r)
		}
		return vm.fn.constants.t1[-r]
	}
	if r >= 0 {
		return vm.regs.t1[vm.fp[1]+uint32(r)]
	}
	return *(vm.fn.refs.t1[-r-1])
}

func (vm *VM) setFloat(r int8, f float64) {
	if r >= 0 {
		vm.regs.t1[vm.fp[1]+uint32(r)] = f
	} else {
		*(vm.fn.refs.t1[-r-1]) = f
	}
}

func (vm *VM) incFloat(r int8) {
	if r >= 0 {
		vm.regs.t1[vm.fp[1]+uint32(r)]++
	} else {
		*(vm.fn.refs.t1[-r-1])++
	}
}

func (vm *VM) decFloat(r int8) {
	if r >= 0 {
		vm.regs.t1[vm.fp[1]+uint32(r)]--
	} else {
		*(vm.fn.refs.t1[-r-1])--
	}
}

func (vm *VM) string(r int8) string {
	if r >= 0 {
		return vm.regs.t2[vm.fp[2]+uint32(r)]
	}
	return *(vm.fn.refs.t2[-r-1])
}

func (vm *VM) stringk(r int8, k bool) string {
	if k {
		return vm.fn.constants.t2[-r]
	}
	if r >= 0 {
		return vm.regs.t2[vm.fp[2]+uint32(r)]
	}
	return *(vm.fn.refs.t2[-r-1])
}

func (vm *VM) setString(r int8, s string) {
	if r >= 0 {
		vm.regs.t2[vm.fp[2]+uint32(r)] = s
	} else {
		*(vm.fn.refs.t2[-r-1]) = s
	}
}

func (vm *VM) value(r int8) interface{} {
	if r >= 0 {
		return vm.regs.t3[vm.fp[3]+uint32(r)]
	}
	return *(vm.fn.refs.t3[-r-1])
}

func (vm *VM) setValue(r int8, i interface{}) {
	if r >= 0 {
		vm.regs.t3[vm.fp[3]+uint32(r)] = i
	} else {
		*(vm.fn.refs.t3[-r-1]) = i
	}
}

func (vm *VM) moveValue(r1, r2 int8) {
	if regs := vm.regs.t3; r1 >= 0 && r2 >= 0 {
		regs[vm.fp[3]+uint32(r2)] = regs[vm.fp[3]+uint32(r1)]
	} else if refs := vm.fn.refs.t3; r2 >= 0 {
		regs[vm.fp[3]+uint32(r2)] = *(refs[-r1-1])
	} else {
		dst := uint32(-r2 - 1)
		if r1 >= 0 {
			*(refs[dst]) = regs[vm.fp[3]+uint32(r1)]
		} else {
			*(refs[dst]) = *(refs[-r1-1])
		}
	}
}
