// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"reflect"
)

type registers struct {
	Int    []int64
	Float  []float64
	String []string
	Misc   []interface{}
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
		return vm.regs.Int[vm.fp[0]+uint32(r)]
	}
	return *(vm.regs.Misc[vm.fp[3]+uint32(-r-1)].(*int64))
}

func (vm *VM) intk(r int8, k bool) int64 {
	if k {
		return int64(r)
	}
	if r >= 0 {
		return vm.regs.Int[vm.fp[0]+uint32(r)]
	}
	return *(vm.regs.Misc[vm.fp[3]+uint32(-r-1)].(*int64))
}

func (vm *VM) bool(r int8) bool {
	if r >= 0 {
		return vm.regs.Int[vm.fp[0]+uint32(r)] > 0
	}
	return *(vm.regs.Misc[vm.fp[3]+uint32(-r-1)].(*bool))
}

func (vm *VM) boolk(r int8, k bool) bool {
	if k {
		return r > 0
	}
	if r >= 0 {
		return vm.regs.Int[vm.fp[0]+uint32(r)] > 0
	}
	return *(vm.regs.Misc[-r-1].(*bool))
}

func (vm *VM) setInt(r int8, i int64) {
	if r >= 0 {
		vm.regs.Int[vm.fp[0]+uint32(r)] = i
	} else {
		*(vm.regs.Misc[vm.fp[3]+uint32(-r-1)].(*int64)) = i
	}
}

func (vm *VM) setBool(r int8, b bool) {
	v := int64(0)
	if b {
		v = 1
	}
	if r >= 0 {
		vm.regs.Int[vm.fp[0]+uint32(r)] = v
	} else {
		*(vm.regs.Misc[vm.fp[3]+uint32(-r-1)].(*bool)) = b
	}
}

func (vm *VM) incInt(r int8) {
	if r >= 0 {
		vm.regs.Int[vm.fp[0]+uint32(r)]++
	} else {
		*(vm.regs.Misc[vm.fp[3]+uint32(-r-1)].(*int64))++
	}
}

func (vm *VM) decInt(r int8) {
	if r >= 0 {
		vm.regs.Int[vm.fp[0]+uint32(r)]--
	} else {
		*(vm.regs.Misc[vm.fp[3]+uint32(-r-1)].(*int64))--
	}
}

func (vm *VM) float(r int8) float64 {
	if r >= 0 {
		return vm.regs.Float[vm.fp[1]+uint32(r)]
	}
	return *(vm.regs.Misc[-r-1].(*float64))
}

func (vm *VM) floatk(r int8, k bool) float64 {
	if k {
		return float64(r)
	}
	if r >= 0 {
		return vm.regs.Float[vm.fp[1]+uint32(r)]
	}
	return *(vm.regs.Misc[vm.fp[3]+uint32(-r-1)].(*float64))

}

func (vm *VM) setFloat(r int8, f float64) {
	if r >= 0 {
		vm.regs.Float[vm.fp[1]+uint32(r)] = f
	} else {
		*(vm.regs.Misc[vm.fp[3]+uint32(-r-1)].(*float64)) = f
	}
}

func (vm *VM) incFloat(r int8) {
	if r >= 0 {
		vm.regs.Float[vm.fp[1]+uint32(r)]++
	} else {
		*(vm.regs.Misc[vm.fp[3]+uint32(-r-1)].(*float64))++
	}
}

func (vm *VM) decFloat(r int8) {
	if r >= 0 {
		vm.regs.Float[vm.fp[1]+uint32(r)]--
	} else {
		*(vm.regs.Misc[vm.fp[3]+uint32(-r-1)].(*float64))--
	}
}

func (vm *VM) string(r int8) string {
	if r >= 0 {
		return vm.regs.String[vm.fp[2]+uint32(r)]
	}
	return *(vm.regs.Misc[vm.fp[3]+uint32(-r-1)].(*string))
}

func (vm *VM) stringk(r int8, k bool) string {
	if k {
		return vm.fn.constants.String[-r]
	}
	if r >= 0 {
		return vm.regs.String[vm.fp[2]+uint32(r)]
	}
	return *(vm.regs.Misc[vm.fp[3]+uint32(-r-1)].(*string))
}

func (vm *VM) setString(r int8, s string) {
	if r >= 0 {
		vm.regs.String[vm.fp[2]+uint32(r)] = s
	} else {
		*(vm.regs.Misc[vm.fp[3]+uint32(-r-1)].(*string)) = s
	}
}

func (vm *VM) value(r int8) interface{} {
	if r >= 0 {
		return vm.regs.Misc[vm.fp[3]+uint32(r)]
	}
	return reflect.ValueOf(vm.regs.Misc[vm.fp[3]+uint32(-r-1)]).Elem().Interface()
}

func (vm *VM) valuek(r int8, k bool) interface{} {
	if k {
		return vm.fn.constants.Misc[-r-1]
	}
	if r >= 0 {
		return vm.regs.Misc[vm.fp[3]+uint32(r)]
	}
	return reflect.ValueOf(vm.regs.Misc[vm.fp[3]+uint32(-r-1)]).Elem().Interface()
}

func (vm *VM) setValue(r int8, i interface{}) {
	if r >= 0 {
		vm.regs.Misc[vm.fp[3]+uint32(r)] = i
		return
	}
	reflect.ValueOf(vm.regs.Misc[vm.fp[3]+uint32(-r-1)]).Elem().Set(reflect.ValueOf(i))
}
