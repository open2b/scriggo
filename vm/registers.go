// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"reflect"
)

type registers struct {
	int     []int64
	float   []float64
	string  []string
	general []interface{}
}

func (vm *VM) set(r int8, v reflect.Value) {
	k := v.Kind()
	if reflect.Int <= k && k <= reflect.Int64 {
		vm.setInt(r, v.Int())
	} else if reflect.Uint <= k && k <= reflect.Uintptr {
		vm.setInt(r, int64(v.Uint()))
	} else if k == reflect.String {
		vm.setString(r, v.String())
	} else if k == reflect.Float64 || k == reflect.Float32 {
		vm.setFloat(r, v.Float())
	} else {
		vm.setGeneral(r, v.Interface())
	}
}

func (vm *VM) int(r int8) int64 {
	if r > 0 {
		return vm.regs.int[vm.fp[0]+uint32(r)]
	}
	return vm.intIndirect(-r)
}

func (vm *VM) intk(r int8, k bool) int64 {
	if k {
		return int64(r)
	}
	if r > 0 {
		return vm.regs.int[vm.fp[0]+uint32(r)]
	}
	return vm.intIndirect(-r)
}

func (vm *VM) intIndirect(r int8) int64 {
	// TODO(Gianluca): add support bool values with defined type.
	v := vm.regs.general[vm.fp[3]+uint32(r)]
	if v, ok := v.(*int); ok {
		return int64(*v)
	}
	if v, ok := v.(*bool); ok {
		if *v {
			return 1 // true.
		}
		return 0 // false.
	}
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		panic(runtimeError("invalid memory address or nil pointer dereference"))
	}
	elem := rv.Elem()
	if k := elem.Kind(); reflect.Int <= k && k <= reflect.Int64 {
		return elem.Int()
	}
	return int64(elem.Uint())
}

func (vm *VM) setInt(r int8, i int64) {
	if r > 0 {
		vm.regs.int[vm.fp[0]+uint32(r)] = i
		return
	}
	vm.setIntIndirect(-r, i)
}

func (vm *VM) setIntIndirect(r int8, i int64) {
	v := vm.regs.general[vm.fp[3]+uint32(r)]
	if vi, ok := v.(*int); ok {
		*vi = int(i)
	} else if vb, ok := v.(*bool); ok {
		*vb = bool(i == 1)
	} else {
		elem := reflect.ValueOf(v).Elem()
		if k := elem.Kind(); reflect.Int <= k && k <= reflect.Int64 {
			elem.SetInt(i)
		} else {
			elem.SetUint(uint64(i))
		}
	}
}

func (vm *VM) bool(r int8) bool {
	if r > 0 {
		return vm.regs.int[vm.fp[0]+uint32(r)] > 0
	}
	return vm.boolIndirect(-r)
}

func (vm *VM) boolk(r int8, k bool) bool {
	if k {
		return r > 0
	}
	if r > 0 {
		return vm.regs.int[vm.fp[0]+uint32(r)] > 0
	}
	return vm.boolIndirect(-r)
}

func (vm *VM) boolIndirect(r int8) bool {
	v := vm.regs.general[vm.fp[3]+uint32(r)]
	if v, ok := v.(*bool); ok {
		return *v
	}
	return reflect.ValueOf(v).Elem().Bool()
}

func (vm *VM) setBool(r int8, b bool) {
	if r > 0 {
		v := int64(0)
		if b {
			v = 1
		}
		vm.regs.int[vm.fp[0]+uint32(r)] = v
		return
	}
	vm.setBoolIndirect(-r, b)
}

func (vm *VM) setBoolIndirect(r int8, b bool) {
	v := vm.regs.general[vm.fp[3]+uint32(r)]
	if v, ok := v.(*bool); ok {
		*v = b
	} else {
		reflect.ValueOf(v).Elem().SetBool(b)
	}
}

func (vm *VM) incInt(r int8) {
	if r > 0 {
		vm.regs.int[vm.fp[0]+uint32(r)]++
		return
	}
	vm.incIntIndirect(-r)
}

func (vm *VM) incIntIndirect(r int8) {
	v := vm.regs.general[vm.fp[3]+uint32(r)]
	if v, ok := v.(*int); ok {
		*v++
	} else {
		elem := reflect.ValueOf(v).Elem()
		if k := elem.Kind(); reflect.Int <= k && k <= reflect.Int64 {
			elem.SetInt(elem.Int() + 1)
		} else {
			elem.SetUint(elem.Uint() + 1)
		}
	}
}

func (vm *VM) decInt(r int8) {
	if r > 0 {
		vm.regs.int[vm.fp[0]+uint32(r)]--
		return
	}
	vm.decIntIndirect(-r)
}

func (vm *VM) decIntIndirect(r int8) {
	v := vm.regs.general[vm.fp[3]+uint32(r)]
	if v, ok := v.(*int); ok {
		*v--
	} else {
		elem := reflect.ValueOf(v).Elem()
		if k := elem.Kind(); reflect.Int <= k && k <= reflect.Int64 {
			elem.SetInt(elem.Int() - 1)
		} else {
			elem.SetUint(elem.Uint() - 1)
		}
	}
}

func (vm *VM) float(r int8) float64 {
	if r > 0 {
		return vm.regs.float[vm.fp[1]+uint32(r)]
	}
	return vm.floatIndirect(-r)
}

func (vm *VM) floatk(r int8, k bool) float64 {
	if k {
		return float64(r)
	}
	if r > 0 {
		return vm.regs.float[vm.fp[1]+uint32(r)]
	}
	return vm.floatIndirect(-r)
}

func (vm *VM) floatIndirect(r int8) float64 {
	v := vm.regs.general[vm.fp[3]+uint32(r)]
	if v, ok := v.(*float64); ok {
		return *v
	}
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		panic(runtimeError("invalid memory address or nil pointer dereference"))
	}
	return rv.Elem().Float()
}

func (vm *VM) setFloat(r int8, f float64) {
	if r > 0 {
		vm.regs.float[vm.fp[1]+uint32(r)] = f
		return
	}
	vm.setFloatIndirect(-r, f)
}

func (vm *VM) setFloatIndirect(r int8, f float64) {
	v := vm.regs.general[vm.fp[3]+uint32(r)]
	if v, ok := v.(*float64); ok {
		*v = f
	} else {
		reflect.ValueOf(v).Elem().SetFloat(f)
	}
}

func (vm *VM) incFloat(r int8) {
	if r > 0 {
		vm.regs.float[vm.fp[1]+uint32(r)]++
		return
	}
	vm.incFloatIndirect(-r)
}

func (vm *VM) incFloatIndirect(r int8) {
	v := vm.regs.general[vm.fp[3]+uint32(r)]
	if v, ok := v.(*float64); ok {
		*v++
	} else {
		elem := reflect.ValueOf(v).Elem()
		elem.SetFloat(elem.Float() + 1)
	}
}

func (vm *VM) decFloat(r int8) {
	if r > 0 {
		vm.regs.float[vm.fp[1]+uint32(r)]--
		return
	}
	vm.decFloatIndirect(-r)
}

func (vm *VM) decFloatIndirect(r int8) {
	v := vm.regs.general[vm.fp[3]+uint32(r)]
	if v, ok := v.(*float64); ok {
		*v--
	} else {
		elem := reflect.ValueOf(v).Elem()
		elem.SetFloat(elem.Float() - 1)
	}
}

func (vm *VM) string(r int8) string {
	if r > 0 {
		return vm.regs.string[vm.fp[2]+uint32(r)]
	}
	return vm.stringIndirect(-r)
}

func (vm *VM) stringk(r int8, k bool) string {
	if k {
		return vm.fn.Constants.String[uint8(r)]
	}
	if r > 0 {
		return vm.regs.string[vm.fp[2]+uint32(r)]
	}
	return vm.stringIndirect(-r)
}

func (vm *VM) stringIndirect(r int8) string {
	v := vm.regs.general[vm.fp[3]+uint32(r)]
	if v, ok := v.(*string); ok {
		return *v
	}
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		panic(runtimeError("invalid memory address or nil pointer dereference"))
	}
	return rv.Elem().String()
}

func (vm *VM) setString(r int8, s string) {
	if r > 0 {
		vm.regs.string[vm.fp[2]+uint32(r)] = s
		return
	}
	vm.setStringIndirect(-r, s)
}

func (vm *VM) setStringIndirect(r int8, s string) {
	v := vm.regs.general[vm.fp[3]+uint32(r)]
	if v, ok := v.(*string); ok {
		*v = s
	} else {
		reflect.ValueOf(v).Elem().SetString(s)
	}
}

func (vm *VM) general(r int8) interface{} {
	if r > 0 {
		return vm.regs.general[vm.fp[3]+uint32(r)]
	}
	return vm.generalIndirect(-r)
}

func (vm *VM) generalk(r int8, k bool) interface{} {
	if k {
		return vm.fn.Constants.General[uint8(r)]
	}
	if r > 0 {
		return vm.regs.general[vm.fp[3]+uint32(r)]
	}
	return vm.generalIndirect(-r)
}

func (vm *VM) generalIndirect(r int8) interface{} {
	rv := reflect.ValueOf(vm.regs.general[vm.fp[3]+uint32(r)])
	if rv.IsNil() {
		panic(runtimeError("invalid memory address or nil pointer dereference"))
	}
	elem := rv.Elem()
	switch elem.Kind() {
	case reflect.Func:
		return &callable{
			predefined: &PredefinedFunction{
				Func:  elem.Interface(),
				value: elem,
			},
		}
	case reflect.Array:
		return elem.Slice(0, elem.Len()).Interface()
	default:
		return elem.Interface()
	}
}

func (vm *VM) setGeneral(r int8, i interface{}) {
	if r > 0 {
		vm.regs.general[vm.fp[3]+uint32(r)] = i
		return
	}
	vm.setGeneralIndirect(-r, i)
}

func (vm *VM) setGeneralIndirect(r int8, i interface{}) {
	v := vm.regs.general[vm.fp[3]+uint32(r)]
	reflect.ValueOf(v).Elem().Set(reflect.ValueOf(i))
}

func (vm *VM) getIntoReflectValue(r int8, v reflect.Value, k bool) {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(vm.boolk(r, k))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(vm.intk(r, k))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v.SetUint(uint64(vm.intk(r, k)))
	case reflect.Float32, reflect.Float64:
		v.SetFloat(vm.floatk(r, k))
	case reflect.String:
		v.SetString(vm.stringk(r, k))
	case reflect.Func:
		v.Set(vm.generalk(r, k).(*callable).Value(vm.env))
	case reflect.Array:
		slice := reflect.ValueOf(vm.generalk(r, k))
		array := reflect.New(reflect.ArrayOf(slice.Len(), slice.Type().Elem())).Elem()
		reflect.Copy(array, slice)
		v.Set(array)
	case reflect.Interface:
		if g := vm.generalk(r, k); g == nil {
			if t := v.Type(); t == emptyInterfaceType {
				v.Set(emptyInterfaceNil)
			} else {
				v.Set(reflect.Zero(t))
			}
		} else {
			v.Set(reflect.ValueOf(g))
		}
	default:
		v.Set(reflect.ValueOf(vm.generalk(r, k)))
	}
}

func (vm *VM) setFromReflectValue(r int8, v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		vm.setBool(r, v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		vm.setInt(r, v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		vm.setInt(r, int64(v.Uint()))
	case reflect.Float32, reflect.Float64:
		vm.setFloat(r, v.Float())
	case reflect.String:
		vm.setString(r, v.String())
	case reflect.Func:
		c := &callable{
			predefined: &PredefinedFunction{
				Func:  v.Interface(),
				value: v,
			},
		}
		vm.setGeneral(r, c)
	case reflect.Array:
		vm.setGeneral(r, v.Slice(0, v.Len()).Interface())
	default:
		vm.setGeneral(r, v.Interface())
	}
}

func appendCap(c, ol, nl int) int {
	if c == 0 {
		return nl
	}
	for c < nl {
		if ol < 1024 {
			c += c
		} else {
			c += c / 4
		}
	}
	return c
}

func (vm *VM) appendSlice(first int8, length int, slice interface{}) interface{} {
	switch slice := slice.(type) {
	case []int:
		ol := len(slice)
		nl := ol + length
		if nl < ol {
			panic(OutOfMemoryError{vm.env})
		}
		if c := cap(slice); nl <= c {
			slice = slice[:nl]
		} else {
			old := slice
			c = appendCap(c, ol, nl)
			slice = make([]int, nl, c)
			copy(slice, old)
		}
		t := slice[ol:]
		regs := vm.regs.int[vm.fp[0]+uint32(first):]
		for i := 0; i < length; i++ {
			t[i] = int(regs[i])
		}
		return slice
	case []byte:
		ol := len(slice)
		nl := ol + length
		if nl < ol {
			panic(OutOfMemoryError{vm.env})
		}
		if c := cap(slice); nl <= c {
			slice = slice[:nl]
		} else {
			old := slice
			c = appendCap(c, ol, nl)
			slice = make([]byte, nl, c)
			copy(slice, old)
		}
		t := slice[ol:]
		s := vm.regs.int[vm.fp[0]+uint32(first):]
		for i := 0; i < length; i++ {
			t[i] = byte(s[i])
		}
		return slice
	case []rune:
		ol := len(slice)
		nl := ol + length
		if nl < ol {
			panic(OutOfMemoryError{vm.env})
		}
		if c := cap(slice); nl <= c {
			slice = slice[:nl]
		} else {
			old := slice
			c = appendCap(c, ol, nl)
			slice = make([]rune, nl, c)
			copy(slice, old)
		}
		t := slice[ol:]
		s := vm.regs.int[vm.fp[0]+uint32(first):]
		for i := 0; i < length; i++ {
			t[i] = rune(s[i])
		}
		return slice
	case []float64:
		i := int(vm.fp[1] + uint32(first))
		return append(slice, vm.regs.float[i:i+length]...)
	case []string:
		i := int(vm.fp[2] + uint32(first))
		return append(slice, vm.regs.string[i:i+length]...)
	case []interface{}:
		i := int(vm.fp[3] + uint32(first))
		return append(slice, vm.regs.general[i:i+length]...)
	default:
		s := reflect.ValueOf(slice)
		ol := s.Len()
		nl := ol + length
		if nl < ol {
			panic(OutOfMemoryError{vm.env})
		}
		if c := s.Cap(); nl <= c {
			s = s.Slice(0, nl)
		} else {
			old := s
			c = appendCap(c, ol, nl)
			s = reflect.MakeSlice(s.Type(), nl, c)
			if ol > 0 {
				reflect.Copy(s, old)
			}
		}
		switch s.Type().Elem().Kind() {
		case reflect.Bool:
			regs := vm.regs.int[vm.fp[0]+uint32(first):]
			for i, j := 0, ol; i < length; i, j = i+1, j+1 {
				s.Index(j).SetBool(regs[i] > 0)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			regs := vm.regs.int[vm.fp[0]+uint32(first):]
			for i, j := 0, ol; i < length; i, j = i+1, j+1 {
				s.Index(j).SetInt(regs[i])
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			regs := vm.regs.int[vm.fp[0]+uint32(first):]
			for i, j := 0, ol; i < length; i, j = i+1, j+1 {
				s.Index(j).SetUint(uint64(regs[i]))
			}
		case reflect.Float32, reflect.Float64:
			regs := vm.regs.float[vm.fp[1]+uint32(first):]
			for i, j := 0, ol; i < length; i, j = i+1, j+1 {
				s.Index(j).SetFloat(regs[i])
			}
		case reflect.String:
			regs := vm.regs.string[vm.fp[2]+uint32(first):]
			for i, j := 0, ol; i < length; i, j = i+1, j+1 {
				s.Index(j).SetString(regs[i])
			}
		default:
			regs := vm.regs.general[vm.fp[3]+uint32(first):]
			for i, j := 0, ol; i < length; i, j = i+1, j+1 {
				s.Index(j).Set(reflect.ValueOf(regs[i]))
			}
		}
		return s.Interface()
	}
}
