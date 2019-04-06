// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"fmt"
	"os"
	"reflect"
)

var DebugTraceExecution = true

type InterpretResult int

const (
	InterpretOK InterpretResult = iota
	InterpretCompileError
	InterpretRunTimeError
)

const noRegister = -128

func decodeAddr(a, b, c int8) uint32 {
	return uint32(uint8(a)) | uint32(uint8(b))<<8 | uint32(uint8(c))<<16
}

func (vm *VM) Run(funcname string) (InterpretResult, error) {

	var err error
	vm.fn, err = vm.pkg.Function(funcname)
	if err != nil {
		return 0, err
	}

	vm.run()

	return InterpretOK, nil
}

func (vm *VM) run() int {

	var op operation
	var a, b, c int8

	for {

		i := vm.fn.body[vm.pc]
		vm.pc++

		if DebugTraceExecution {
			n, _ := DisassembleInstruction(os.Stderr, vm.fn, i)
			for i := n; i < 20; i++ {
				_, _ = os.Stderr.WriteString(" ")
			}
		}

		op, a, b, c = i.op, i.a, i.b, i.c

		switch op {

		// Add
		case opAddInt:
			vm.setInt(c, vm.int(a)+vm.int(b))
		case -opAddInt:
			vm.setInt(c, vm.int(a)+int64(b))
		case opAddInt8, -opAddInt8:
			vm.setInt(c, int64(int8(vm.int(a)+vm.intk(b, op < 0))))
		case opAddInt16, -opAddInt16:
			vm.setInt(c, int64(int16(vm.int(a)+vm.intk(b, op < 0))))
		case opAddInt32, -opAddInt32:
			vm.setInt(c, int64(int32(vm.int(a)+vm.intk(b, op < 0))))
		case opAddFloat32, -opAddFloat32:
			vm.setFloat(c, float64(float32(vm.float(a)+vm.floatk(b, op < 0))))
		case opAddFloat64:
			vm.setFloat(c, vm.float(a)+vm.float(b))
		case -opAddFloat64:
			vm.setFloat(c, vm.float(a)+float64(b))

		// And
		case opAnd:
			vm.setInt(c, vm.int(a)&vm.intk(b, op < 0))

		// AndNot
		case opAndNot:
			vm.setInt(c, vm.int(a)&^vm.intk(b, op < 0))

		// Append
		case opAppend:
			//s := reflectValue.ValueOf(vm.popInterface())
			//t := s.Type()
			//n := int(vm.readByte())
			//var s2 reflectValue.Value
			//l, c := s.Len(), s.Cap()
			//p := 0
			//if l+n <= c {
			//	s2 = reflectValue.MakeSlice(t, n, n)
			//	s = s.Slice3(0, c, c)
			//} else {
			//	s2 = reflectValue.MakeSlice(t, l+n, l+n)
			//	reflectValue.Copy(s2, s)
			//	s = s2
			//	p = l
			//}
			//for i := 1; i < n; i++ {
			//	v := reflectValue.ValueOf(vm.popInterface())
			//	s2.Index(p + i - 1).Set(v)
			//}
			//if l+n <= c {
			//	reflectValue.Copy(s2.Slice(l, l+n+1), s2)
			//}
			//vm.pushInterface(s.Interface())

		// Assert
		case opAssert:
			v := reflect.ValueOf(vm.value(a))
			t := vm.fn.types[int(uint(b))]
			ok := v.Type() == t
			switch t.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				var n int64
				if ok {
					n = v.Int()
				}
				vm.setInt(c, n)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				var n int64
				if ok {
					n = int64(v.Uint())
				}
				vm.setInt(c, n)
			case reflect.Float32, reflect.Float64:
				var f float64
				if ok {
					f = v.Float()
				}
				vm.setFloat(c, f)
			case reflect.String:
				var s string
				if ok {
					s = v.String()
				}
				vm.setString(c, s)
			default:
				var i interface{}
				if ok {
					i = v.Interface()
				}
				vm.setValue(c, i)
			}
			vm.ok = ok
		case opAssertInt:
			i, ok := vm.value(a).(int)
			vm.setInt(c, int64(i))
			vm.ok = ok
		case opAssertFloat64:
			f, ok := vm.value(a).(float64)
			vm.setFloat(c, f)
			vm.ok = ok
		case opAssertString:
			s, ok := vm.value(a).(string)
			vm.setString(c, s)
			vm.ok = ok

		// Call
		case opCall:
			prev := Call{fp: vm.fp, pc: vm.pc, fn: vm.fn}
			//var uppers []interface{}
			vm.fn = vm.valuek(a, true).(*Function)
			for r := 0; r < 3; r++ {
				nr := uint32(vm.fn.numRegs[r])
				// Splits stack if necessary.
				if nr > 0 && (vm.fp[r]%StackSize)+nr > StackSize {
					vm.checkSplitStack(r, vm.fp[r]+nr)
				}
				// Increments frame pointer if necessary.
				na := vm.fn.numIn[r] + vm.fn.numOut[r]
				if nr > 0 || na > 0 {
					vm.fp[r] += uint32(prev.fn.numRegs[r] - na)
				}
			}
			vm.pc = 0
			vm.calls = append(vm.calls, prev)

		// CallNative
		case opCallNative:
			//fn := vm.iface(a).(reflectValue.Value)
			//ret := f.Call(args)
			//vm.pushValues(ret)

		// Cap
		case opCap:
			vm.setInt(c, int64(reflect.ValueOf(a).Cap()))

		//// Closure
		//case opClosure:
		//	closure := Closure{
		//		function: vm.fn.closures[a],
		//	}
		//	if vars := closure.function.uppers; vars != nil {
		//		closure.vars = make([]interface{}, len(vars))
		//		for i, v := range vars {
		//			switch v.typ {
		//			case TypeInt:
		//				closure.vars[i] = vm.getIntUpValue(v.index)
		//			case TypeFloat:
		//				closure.vars[i] = vm.floatUpValue(v.index)
		//			case TypeString:
		//				closure.vars[i] = vm.stringUpValue(v.index)
		//			case TypeIface:
		//				closure.vars[i] = vm.ifaceUpValue(v.index)
		//			}
		//		}
		//	}
		//	vm.setIface(b, &closure)

		// Continue
		case opContinue:
			return int(a)

		// Copy
		case opCopy:
			src := reflect.ValueOf(a)
			dst := reflect.ValueOf(b)
			vm.setInt(c, int64(reflect.Copy(src, dst)))

		// Concat
		case opConcat:
			vm.setString(c, vm.string(a)+vm.string(b))

		// Delete
		case opDelete:
			m := reflect.ValueOf(a)
			k := reflect.ValueOf(b)
			m.SetMapIndex(k, reflect.Value{})

		// Div
		case opDivInt:
			vm.setInt(c, vm.int(a)/vm.int(b))
		case -opDivInt:
			vm.setInt(c, vm.int(a)/int64(b))
		case opDivInt8, -opDivInt8:
			vm.setInt(c, int64(int8(vm.int(a))/int8(vm.intk(b, op < 0))))
		case opDivInt16, -opDivInt16:
			vm.setInt(c, int64(int16(vm.int(a))/int16(vm.intk(b, op < 0))))
		case opDivInt32, -opDivInt32:
			vm.setInt(c, int64(int32(vm.int(a))/int32(vm.intk(b, op < 0))))
		case opDivUint8, -opDivUint8:
			vm.setInt(c, int64(uint8(vm.int(a))/uint8(vm.intk(b, op < 0))))
		case opDivUint16, -opDivUint16:
			vm.setInt(c, int64(uint16(vm.int(a))/uint16(vm.intk(b, op < 0))))
		case opDivUint32, -opDivUint32:
			vm.setInt(c, int64(uint32(vm.int(a))/uint32(vm.intk(b, op < 0))))
		case opDivUint64, -opDivUint64:
			vm.setInt(c, int64(uint64(vm.int(a))/uint64(vm.intk(b, op < 0))))
		case opDivFloat32, -opDivFloat32:
			vm.setFloat(c, float64(float32(vm.float(a))/float32(vm.floatk(b, op < 0))))
		case opDivFloat64:
			vm.setFloat(c, vm.float(a)/vm.float(b))
		case -opDivFloat64:
			vm.setFloat(c, vm.float(a)/float64(b))

		// If
		case opIf:
			var cond bool
			v1 := vm.value(a)
			switch Condition(b) {
			case ConditionNil:
				cond = v1 == nil
			case ConditionNotNil:
				cond = v1 != nil
				if cond {
					vm.pc++
				}
			}
		case opIfInt, -opIfInt:
			var cond bool
			v1 := vm.int(a)
			v2 := int64(vm.intk(c, op < 0))
			switch Condition(b) {
			case ConditionEqual:
				cond = v1 == v2
			case ConditionNotEqual:
				cond = v1 != v2
			case ConditionLess:
				cond = v1 < v2
			case ConditionLessOrEqual:
				cond = v1 <= v2
			case ConditionGreater:
				cond = v1 > v2
			case ConditionGreaterOrEqual:
				cond = v1 >= v2
			}
			if cond {
				vm.pc++
			}
		case opIfUint, -opIfUint:
			var cond bool
			v1 := uint64(vm.int(a))
			v2 := uint64(vm.intk(c, op < 0))
			switch Condition(b) {
			case ConditionEqual:
				cond = v1 == v2
			case ConditionNotEqual:
				cond = v1 != v2
			case ConditionLess:
				cond = v1 < v2
			case ConditionLessOrEqual:
				cond = v1 <= v2
			case ConditionGreater:
				cond = v1 > v2
			case ConditionGreaterOrEqual:
				cond = v1 >= v2
			}
			if cond {
				vm.pc++
			}
		case opIfFloat, -opIfFloat:
			var cond bool
			v1 := vm.float(a)
			v2 := vm.floatk(c, op < 0)
			switch Condition(b) {
			case ConditionEqual:
				cond = v1 == v2
			case ConditionNotEqual:
				cond = v1 != v2
			case ConditionLess:
				cond = v1 < v2
			case ConditionLessOrEqual:
				cond = v1 <= v2
			case ConditionGreater:
				cond = v1 > v2
			case ConditionGreaterOrEqual:
				cond = v1 >= v2
			}
			if cond {
				vm.pc++
			}
		case opIfString, -opIfString:
			var cond bool
			v1 := vm.string(a)
			if Condition(b) < ConditionEqualLen {
				if op < 0 && c >= 0 {
					l := len(v1)
					switch Condition(b) {
					case ConditionEqual:
						cond = l == 1 && v1[0] == uint8(c)
					case ConditionNotEqual:
						cond = l != 1 || v1[0] != uint8(c)
					case ConditionLess:
						cond = l == 0 || l == 1 && v1[0] < uint8(c)
					case ConditionLessOrEqual:
						cond = l == 0 || l == 1 && v1[0] <= uint8(c)
					case ConditionGreater:
						cond = l > 1 || v1[0] > uint8(c)
					case ConditionGreaterOrEqual:
						cond = l > 1 || v1[0] >= uint8(c)
					}
				} else {
					v2 := vm.stringk(c, op < 0)
					switch Condition(b) {
					case ConditionEqual:
						cond = v1 == v2
					case ConditionNotEqual:
						cond = v1 != v2
					case ConditionLess:
						cond = v1 < v2
					case ConditionLessOrEqual:
						cond = v1 <= v2
					case ConditionGreater:
						cond = v1 > v2
					case ConditionGreaterOrEqual:
						cond = v1 >= v2
					}
				}
			} else {
				v2 := int(vm.intk(c, op < 0))
				switch Condition(b) {
				case ConditionEqualLen:
					cond = len(v1) == v2
				case ConditionNotEqualLen:
					cond = len(v1) != v2
				case ConditionLessLen:
					cond = len(v1) < v2
				case ConditionLessOrEqualLen:
					cond = len(v1) <= v2
				case ConditionGreaterLen:
					cond = len(v1) > v2
				case ConditionGreaterOrEqualLen:
					cond = len(v1) >= v2
				}
			}
			if cond {
				vm.pc++
			}

		// Goto
		case opGoto:
			vm.pc = decodeAddr(a, b, c)

		// Index
		case opIndex, -opIndex:
			vm.setValue(c, reflect.ValueOf(vm.value(a)).Index(int(vm.intk(b, op < 0))).Interface())

		// JmpOk
		case opJmpOk:
			if vm.ok {
				vm.pc = decodeAddr(a, b, c)
			}

		// JmpNotOk
		case opJmpNotOk:
			if !vm.ok {
				vm.pc = decodeAddr(a, b, c)
			}

		// Len
		case opLen:
			var length int
			if a == 0 {
				length = len(vm.string(b))
			} else {
				v := vm.value(b)
				switch int(a) {
				case 1:
					length = reflect.ValueOf(v).Len()
				case 2:
					length = len(v.([]byte))
				case 3:
					length = len(v.([]int))
				case 4:
					length = len(v.([]string))
				case 5:
					length = len(v.([]interface{}))
				case 6:
					length = len(v.(map[string]string))
				case 7:
					length = len(v.(map[string]int))
				case 8:
					length = len(v.(map[string]interface{}))
				}
			}
			vm.setInt(c, int64(length))

		// MakeMap
		case opMakeMap, -opMakeMap:
			t := vm.fn.types[int(uint8(a))]
			n := int(vm.intk(b, op < 0))
			vm.setValue(c, reflect.MakeMapWithSize(t, n))

		// MapIndex
		case opMapIndex, -opMapIndex:
			m := reflect.ValueOf(vm.value(a))
			t := m.Type()
			k := op < 0
			var key reflect.Value
			switch kind := t.Key().Kind(); {
			case reflect.Int <= kind && kind <= reflect.Uint64:
				key = reflect.ValueOf(vm.intk(b, k))
			case kind == reflect.Float64 || kind == reflect.Float32:
				key = reflect.ValueOf(vm.floatk(b, k))
			case kind == reflect.String:
				key = reflect.ValueOf(vm.stringk(b, k))
			case kind == reflect.Bool:
				key = reflect.ValueOf(vm.boolk(b, k))
			default:
				key = reflect.ValueOf(vm.value(b))
			}
			elem := m.MapIndex(key)
			switch kind := t.Elem().Kind(); {
			case reflect.Int <= kind && kind <= reflect.Int64:
				vm.setInt(c, elem.Int())
			case reflect.Uint <= kind && kind <= reflect.Uint64:
				vm.setInt(c, int64(elem.Uint()))
			case kind == reflect.Float64 || kind == reflect.Float32:
				vm.setFloat(c, elem.Float())
			case kind == reflect.String:
				vm.setString(c, elem.String())
			case kind == reflect.Bool:
				vm.setBool(c, elem.Bool())
			default:
				vm.setValue(c, elem.Interface())
			}

		// MapIndexStringInt
		case opMapIndexStringInt, -opMapIndexStringInt:
			vm.setInt(c, int64(vm.value(a).(map[string]int)[vm.stringk(b, op < 0)]))

		// MapIndexStringBool
		case opMapIndexStringBool, -opMapIndexStringBool:
			vm.setBool(c, vm.value(a).(map[string]bool)[vm.stringk(b, op < 0)])

		// MapIndexStringString
		case opMapIndexStringString, -opMapIndexStringString:
			vm.setString(c, vm.value(a).(map[string]string)[vm.stringk(b, op < 0)])

		// MapIndexStringInterface
		case opMapIndexStringInterface, -opMapIndexStringInterface:
			vm.setValue(c, vm.value(a).(map[string]interface{})[vm.stringk(b, op < 0)])

		// Move
		case opMove:
			vm.setValue(c, vm.value(b))
		case opMoveInt:
			vm.setInt(c, vm.int(b))
		case -opMoveInt:
			vm.setInt(c, int64(b))
		case opMoveFloat:
			vm.setFloat(c, vm.float(b))
		case -opMoveFloat:
			vm.setFloat(c, float64(b))
		case opMoveString:
			vm.setString(c, vm.string(b))
		case -opMoveString:
			vm.setString(c, vm.stringk(b, true))

		// Mul
		case opMulInt:
			vm.setInt(c, vm.int(a)*vm.int(b))
		case -opMulInt:
			vm.setInt(c, vm.int(a)*int64(b))
		case opMulInt8, -opMulInt8:
			vm.setInt(c, int64(int8(vm.int(a)*vm.intk(b, op < 0))))
		case opMulInt16, -opMulInt16:
			vm.setInt(c, int64(int16(vm.int(a)*vm.intk(b, op < 0))))
		case opMulInt32, -opMulInt32:
			vm.setInt(c, int64(int32(vm.int(a)*vm.intk(b, op < 0))))
		case opMulFloat32, -opMulFloat32:
			vm.setFloat(c, float64(float32(vm.float(a))*float32(vm.floatk(b, op < 0))))
		case opMulFloat64:
			vm.setFloat(c, vm.float(a)*vm.float(b))
		case -opMulFloat64:
			vm.setFloat(c, vm.float(a)*float64(b))

		// New
		case opNew:
			t := vm.fn.types[int(uint(b))]
			vm.setValue(c, reflect.New(t))

		// Or
		case opOr, -opOr:
			vm.setInt(c, vm.int(a)|vm.intk(b, op < 0))

		// Range
		case opRange:
			var cont int
			s := vm.value(c)
			switch s := s.(type) {
			case []int:
				for i, v := range s {
					if a != noRegister {
						vm.setInt(a, int64(i))
					}
					if b != noRegister {
						vm.setInt(b, int64(v))
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case []byte:
				for i, v := range s {
					if a != noRegister {
						vm.setInt(a, int64(i))
					}
					if b != noRegister {
						vm.setInt(b, int64(v))
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case []rune:
				for i, v := range s {
					if a != noRegister {
						vm.setInt(a, int64(i))
					}
					if b != noRegister {
						vm.setInt(b, int64(v))
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case []float64:
				for i, v := range s {
					if a != noRegister {
						vm.setInt(a, int64(i))
					}
					if b != noRegister {
						vm.setFloat(b, v)
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case []string:
				for i, v := range s {
					if a != noRegister {
						vm.setInt(a, int64(i))
					}
					if b != noRegister {
						vm.setString(b, v)
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case []interface{}:
				for i, v := range s {
					if a != noRegister {
						vm.setInt(a, int64(i))
					}
					if b != noRegister {
						vm.setValue(b, v)
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case map[string]int:
				for i, v := range s {
					if a != noRegister {
						vm.setString(a, i)
					}
					if b != noRegister {
						vm.setInt(b, int64(v))
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case map[string]bool:
				for i, v := range s {
					if a != noRegister {
						vm.setString(a, i)
					}
					if b != noRegister {
						vm.setBool(b, v)
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case map[string]string:
				for i, v := range s {
					if a != noRegister {
						vm.setString(a, i)
					}
					if b != noRegister {
						vm.setString(b, v)
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case map[string]interface{}:
				for i, v := range s {
					if a != noRegister {
						vm.setString(a, i)
					}
					if b != noRegister {
						vm.setValue(b, v)
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			default:
				rs := reflect.ValueOf(s)
				kind := rs.Kind()
				if kind == reflect.Map {
					iter := rs.MapRange()
					for iter.Next() {
						if a != noRegister {
							vm.set(a, iter.Key())
						}
						if b != noRegister {
							vm.set(b, iter.Value())
						}
						if cont = vm.run(); cont > 0 {
							break
						}
					}
				} else {
					if kind == reflect.Ptr {
						rs = rs.Elem()
					}
					length := rs.Len()
					for i := 0; i < length; i++ {
						if a != noRegister {
							vm.setInt(a, int64(i))
						}
						if b != noRegister {
							vm.set(b, rs.Index(i))
						}
						if cont = vm.run(); cont > 0 {
							break
						}
					}
				}
			}
			if cont > 1 {
				return cont - 1
			}

		// RangeString
		case opRangeString, -opRangeString:
			var cont int
			for i, e := range vm.stringk(c, op < 0) {
				if a != 0 {
					vm.setInt(a, int64(i))
				}
				if b != 0 {
					vm.setInt(b, int64(e))
				}
				if cont = vm.run(); cont > 0 {
					break
				}
			}
			if cont > 1 {
				return cont - 1
			}

		// Rem
		case opRemInt:
			vm.setInt(c, vm.int(a)%vm.int(b))
		case -opRemInt:
			vm.setInt(c, vm.int(a)%int64(b))
		case opRemInt8, -opRemInt8:
			vm.setInt(c, int64(int8(vm.int(a))%int8(vm.intk(b, op < 0))))
		case opRemInt16, -opRemInt16:
			vm.setInt(c, int64(int16(vm.int(a))%int16(vm.intk(b, op < 0))))
		case opRemInt32, -opRemInt32:
			vm.setInt(c, int64(int32(vm.int(a))%int32(vm.intk(b, op < 0))))
		case opRemUint8, -opRemUint8:
			vm.setInt(c, int64(uint8(vm.int(a))%uint8(vm.intk(b, op < 0))))
		case opRemUint16, -opRemUint16:
			vm.setInt(c, int64(uint16(vm.int(a))%uint16(vm.intk(b, op < 0))))
		case opRemUint32, -opRemUint32:
			vm.setInt(c, int64(uint32(vm.int(a))%uint32(vm.intk(b, op < 0))))
		case opRemUint64, -opRemUint64:
			vm.setInt(c, int64(uint64(vm.int(a))%uint64(vm.intk(b, op < 0))))

		// Return
		case opReturn:
			last := len(vm.calls) - 1
			if last < 0 {
				return maxInt8
			}
			prev := vm.calls[last]
			vm.fp = prev.fp
			vm.pc = prev.pc
			vm.fn = prev.fn
			vm.calls = vm.calls[0:last]

		// Selector
		case opSelector:
			v := reflect.ValueOf(vm.value(a)).Field(int(uint8(b)))
			switch v.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				vm.setInt(c, v.Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				vm.setInt(c, int64(v.Uint()))
			case reflect.Float64, reflect.Float32:
				vm.setFloat(c, v.Float())
			case reflect.Bool:
				vm.setBool(c, v.Bool())
			case reflect.String:
				vm.setString(c, v.String())
			default:
				vm.setValue(c, v.Interface())
			}

		// MakeSlice
		case opMakeSlice:
			//typ := vm.getType(a)
			//len := int(vm.int(b))
			//cap := int(vm.int(c))
			//vm.pushValue(reflectValue.MakeSlice(typ, len, cap))

		// SliceIndex
		case opSliceIndex, -opSliceIndex:
			v := vm.value(a)
			i := int(vm.intk(b, op < 0))
			switch v := v.(type) {
			case []int:
				vm.setInt(c, int64(v[i]))
			case []byte:
				vm.setInt(c, int64(v[i]))
			case []rune:
				vm.setInt(c, int64(v[i]))
			case []float64:
				vm.setInt(c, int64(v[i]))
			case []string:
				vm.setString(c, v[i])
			case []interface{}:
				vm.setValue(c, v[i])
			default:
				vm.setValue(c, reflect.ValueOf(v).Index(i).Interface())
			}

		// StringIndex
		case opStringIndex, -opStringIndex:
			vm.setInt(c, int64(vm.string(a)[int(vm.intk(b, op < 0))]))

		// Sub
		case opSubInt:
			vm.setInt(c, -(vm.int(a) - vm.int(b)))
		case -opSubInt:
			vm.setInt(c, -(vm.int(a) - int64(b)))
		case opSubInt8, -opSubInt8:
			vm.setInt(c, int64(int8(-(vm.int(a) - vm.intk(b, op < 0)))))
		case opSubInt16, -opSubInt16:
			vm.setInt(c, int64(int32(-(vm.int(a) - vm.intk(b, op < 0)))))
		case opSubInt32, -opSubInt32:
			vm.setInt(c, int64(int32(-(vm.int(a) - vm.intk(b, op < 0)))))
		case opSubFloat32, -opSubFloat32:
			vm.setFloat(c, float64(float32(-(float32(vm.float(a)) - float32(vm.floatk(b, op < 0))))))
		case opSubFloat64:
			vm.setFloat(c, -(vm.float(a) - vm.float(b)))
		case -opSubFloat64:
			vm.setFloat(c, -(vm.float(a) - float64(b)))

		// SubInv
		case opSubInvInt, -opSubInvInt:
			vm.setInt(c, vm.int(a)-vm.intk(b, op < 0))
		case opSubInvInt8, -opSubInvInt8:
			vm.setInt(c, int64(int8(vm.int(a)-vm.intk(b, op < 0))))
		case opSubInvInt16, -opSubInvInt16:
			vm.setInt(c, int64(int32(vm.int(a)-vm.intk(b, op < 0))))
		case opSubInvInt32, -opSubInvInt32:
			vm.setInt(c, int64(int32(vm.int(a)-vm.intk(b, op < 0))))
		case opSubInvFloat32, -opSubInvFloat32:
			vm.setFloat(c, float64(float32(float32(vm.float(a))-float32(vm.floatk(b, op < 0)))))
		case opSubInvFloat64:
			vm.setFloat(c, vm.float(a)-vm.float(b))
		case -opSubInvFloat64:
			vm.setFloat(c, vm.float(a)-float64(b))

		// opXor
		case opXor, -opXor:
			vm.setInt(c, vm.int(a)^vm.intk(b, op < 0))

		}

		if DebugTraceExecution {
			//fmt.Printf("\ti%v f%v s%v\n",
			_, _ = fmt.Fprintf(os.Stderr, "i%v f%v\n",
				vm.regs.t0[vm.fp[0]:vm.fp[0]+uint32(vm.fn.numRegs[0])],
				vm.regs.t1[vm.fp[1]:vm.fp[1]+uint32(vm.fn.numRegs[1])],
			//vm.regs.r2[vm.fp[2]:uint32(vm.fn.frameSize(2))]
			)
		}

	}

}

//func (vm *VM) execCallI() {

//n := int(vm.readByte())
//f := vm.popValue()
//fmt.Printf("\n%T %s\n", f, f)

//switch f.(type) {
//
//case func(string) int:
//	a := vm.popString()
//	_ = vm.popInterface()
//	vm.pushInt(int64(fn(a)))
//
//case func(string) string:
//	a := vm.popString()
//	_ = vm.popInterface()
//	vm.pushString((fn(a))
//
//case func(string, string) int:
//	a1 := vm.popString()
//	a2 := vm.popString()
//	_ = vm.popInterface()
//	vm.pushInt(int64(fn(a1, a2)))
//
//case func(string, string) bool:
//	a1 := vm.popString()
//	a2 := vm.popString()
//	_ = vm.popInterface()
//	b := fn(a1, a2)
//	if b {
//		vm.pushInt(1)
//	} else {
//		vm.pushInt(0)
//	}
//
//case func([]byte) []byte:
//	a := vm.popInterface().([]byte)
//	_ = vm.popInterface()
//	vm.pushInterface.push(fn(a))
//
//case func([]byte, []byte) int:
//	a1 := vm.popInterface().([]byte)
//	a2 := vm.popInterface().([]byte)
//	_ = vm.popInterface()
//	vm.pushInt(int64(fn(a1, a2)))
//
//case func([]byte, []byte) bool:
//	a1 := vm.popInterface().([]byte)
//	a2 := vm.popInterface().([]byte)
//	_ = vm.popInterface()
//	b := fn(a1, a2)
//	if b {
//		vm.pushInt(1)
//	} else {
//		vm.pushInt(0)
//	}

//default:
//var f = reflectValue.ValueOf(f)
//var t = f.Type()
//var args []reflectValue.Value
//if n == 0 {
//	vm.popInterface()
//} else {
//	var numIn = t.numIn()
//	var lastIn = numIn - 1
//	var in reflectValue.Type
//	args = make([]reflectValue.Value, numIn)
//	isVariadic := t.IsVariadic()
//	for i := 0; i < n; i++ {
//		var arg reflectValue.Value
//		if i < lastIn || !isVariadic {
//			in = t.in(i)
//		} else if i == lastIn {
//			in = t.in(lastIn).Elem()
//		}
//		switch in.Kind() {
//		case reflectValue.String:
//			arg = reflectValue.ValueOf(vm.popString())
//		case reflectValue.Int:
//			arg = reflectValue.ValueOf(int(vm.popInt()))
//		case reflectValue.Int64:
//			arg = reflectValue.ValueOf(vm.popInt())
//		case reflectValue.Int32:
//			arg = reflectValue.ValueOf(int32(vm.popInt()))
//		case reflectValue.Int16:
//			arg = reflectValue.ValueOf(int16(vm.popInt()))
//		case reflectValue.Int8:
//			arg = reflectValue.ValueOf(int8(vm.popInt()))
//		case reflectValue.Uint:
//			arg = reflectValue.ValueOf(uint(vm.popInt()))
//		case reflectValue.Uint64:
//			arg = reflectValue.ValueOf(uint64(vm.popInt()))
//		case reflectValue.Uint32:
//			arg = reflectValue.ValueOf(uint32(vm.popInt()))
//		case reflectValue.Uint16:
//			arg = reflectValue.ValueOf(uint16(vm.popInt()))
//		case reflectValue.Uint8:
//			arg = reflectValue.ValueOf(uint8(vm.popInt()))
//		case reflectValue.Float64:
//			arg = reflectValue.ValueOf(vm.popFloat())
//		case reflectValue.Float32:
//			arg = reflectValue.ValueOf(float32(vm.popFloat()))
//		case reflectValue.Bool:
//			if vm.popInt() == 0 {
//				arg = reflectValue.ValueOf(false)
//			} else {
//				arg = reflectValue.ValueOf(true)
//			}
//		default:
//			arg = reflectValue.ValueOf(vm.popInterface())
//		}
//		if i < lastIn || !isVariadic {
//			args[i] = arg
//		} else {
//			if i == lastIn {
//				args[i] = reflectValue.MakeSlice(in, n-numIn+1, n-numIn+1)
//			}
//			args[lastIn].Index(n - numIn + 1).Set(arg)
//		}
//	}
// Pop the fn.
//_ = vm.popInterface()
//}
//ret := f.Call(args)
//numOut := t.numOut()
//for i := 0; i < numOut; i++ {
//	switch t.out(i).Kind() {
//	case reflectValue.String:
//		vm.pushString(ret[i].String())
//	case reflectValue.Int, reflectValue.Int64, reflectValue.Int32, reflectValue.Int16, reflectValue.Int8:
//		vm.pushInt(ret[i].Int())
//	case reflectValue.Uint, reflectValue.Uint64, reflectValue.Uint32, reflectValue.Uint16, reflectValue.Uint8:
//		vm.pushInt(int64(ret[i].Uint()))
//	case reflectValue.Float64, reflectValue.Float32:
//		vm.pushFloat(ret[i].Float())
//	case reflectValue.Bool:
//		if ret[i].Bool() {
//			vm.pushInt(1)
//		} else {
//			vm.pushInt(0)
//		}
//	default:
//		vm.pushInterface(ret[i].Interface())
//	}
//}
//}

//}
