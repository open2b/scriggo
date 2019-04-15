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

const NoVariadic = -1
const NoPackage = -2
const CurrentPackage = -1
const CurrentFunction = -1

func decodeAddr(a, b, c int8) uint32 {
	return uint32(uint8(a)) | uint32(uint8(b))<<8 | uint32(uint8(c))<<16
}

func (vm *VM) Run(funcname string) (InterpretResult, error) {

	var err error
	vm.fn, err = vm.main.Function(funcname)
	if err != nil {
		return 0, err
	}

	vm.run()

	return InterpretOK, nil
}

func (vm *VM) run() int {

	var pc uint32

	var startNativeGoroutine bool

	var op operation
	var a, b, c int8

	for {

		in := vm.fn.body[pc]

		if DebugTraceExecution {
			funcName := vm.fn.name
			if funcName != "" {
				funcName += ":"
			}
			_, _ = fmt.Fprintf(os.Stderr, "i%v f%v\t%s\t",
				vm.regs.Int[vm.fp[0]+1:vm.fp[0]+uint32(vm.fn.regnum[0])+1],
				vm.regs.Float[vm.fp[1]+1:vm.fp[1]+uint32(vm.fn.regnum[1])+1],
				funcName)
			_, _ = DisassembleInstruction(os.Stderr, vm.fn, pc)
			println()
		}

		pc++
		op, a, b, c = in.op, in.a, in.b, in.c

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
			v := reflect.ValueOf(vm.general(a))
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
				vm.setGeneral(c, i)
			}
			vm.ok = ok
		case opAssertInt:
			i, ok := vm.general(a).(int)
			vm.setInt(c, int64(i))
			vm.ok = ok
		case opAssertFloat64:
			f, ok := vm.general(a).(float64)
			vm.setFloat(c, f)
			vm.ok = ok
		case opAssertString:
			s, ok := vm.general(a).(string)
			vm.setString(c, s)
			vm.ok = ok

		// Bind
		case opBind:
			vm.setGeneral(c, vm.cvars[uint8(b)])

		// Call
		case opCall:
			pkg := vm.fn.pkg
			if a != CurrentPackage {
				pkg = pkg.packages[uint8(a)]
			}
			fn := pkg.scrigoFunctions[uint8(b)]
			off := vm.fn.body[pc]
			call := Call{fn: vm.fn, cvars: vm.cvars, fp: vm.fp, pc: pc + 1}
			vm.fp[0] += uint32(off.op)
			if vm.fp[0]+uint32(fn.regnum[0]) > vm.st[0] {
				vm.moreIntStack()
			}
			if vm.fp[1]+uint32(fn.regnum[1]) > vm.st[1] {
				vm.moreFloatStack()
			}
			vm.fp[2] += uint32(off.b)
			if vm.fp[2]+uint32(fn.regnum[2]) > vm.st[2] {
				vm.moreStringStack()
			}
			vm.fp[3] += uint32(off.c)
			if vm.fp[3]+uint32(fn.regnum[3]) > vm.st[3] {
				vm.moreGeneralStack()
			}
			vm.fn = fn
			vm.cvars = nil
			vm.calls = append(vm.calls, call)
			pc = 0

		// CallIndirect
		case opCallIndirect:
			f := vm.general(b).(*callable)
			if f.scrigo != nil {
				fn := f.scrigo
				vm.cvars = f.vars
				off := vm.fn.body[pc]
				call := Call{fn: vm.fn, cvars: vm.cvars, fp: vm.fp, pc: pc + 1}
				vm.fp[0] += uint32(off.op)
				if vm.fp[0]+uint32(fn.regnum[0]) > vm.st[0] {
					vm.moreIntStack()
				}
				if vm.fp[1]+uint32(fn.regnum[1]) > vm.st[1] {
					vm.moreFloatStack()
				}
				vm.fp[2] += uint32(off.b)
				if vm.fp[2]+uint32(fn.regnum[2]) > vm.st[2] {
					vm.moreStringStack()
				}
				vm.fp[3] += uint32(off.c)
				if vm.fp[3]+uint32(fn.regnum[3]) > vm.st[3] {
					vm.moreGeneralStack()
				}
				vm.fn = fn
				vm.calls = append(vm.calls, call)
				pc = 0
				continue
			}
			fallthrough

		// CallFunc, CallMethod, CallIndirect
		case opCallFunc, opCallMethod:
			var fn *NativeFunction
			switch op {
			case opCallMethod:
				t := vm.fn.types[int(uint8(a))]
				m := t.Method(int(uint8(b)))
				fn = &NativeFunction{name: m.Name, value: m.Func}
				fn.slow()
				if i, ok := vm.fn.pkg.AddNativeFunction(fn); ok {
					vm.fn.body[pc-1] = instruction{op: opCallFunc, a: CurrentPackage, b: int8(i)}
				}
			case opCallFunc:
				pkg := vm.fn.pkg
				if a != CurrentPackage {
					pkg = pkg.packages[uint8(a)]
				}
				fn = pkg.nativeFunctions[uint8(b)]
			case opCall:
				fn = vm.general(b).(*callable).native
			}
			fp := vm.fp
			off := vm.fn.body[pc]
			vm.fp[0] += uint32(off.op)
			vm.fp[1] += uint32(off.a)
			vm.fp[2] += uint32(off.b)
			vm.fp[3] += uint32(off.c)
			if fn.fast != nil {
				if startNativeGoroutine {
					startNativeGoroutine = false
					switch f := fn.fast.(type) {
					case func(string) int:
						go f(vm.string(1))
					case func(string) string:
						go f(vm.string(2))
					case func(string, string) int:
						go f(vm.string(1), vm.string(2))
					case func(string, int) string:
						go f(vm.string(2), int(vm.int(1)))
					case func(string, string) bool:
						go f(vm.string(1), vm.string(2))
					case func([]byte) []byte:
						go f(vm.general(2).([]byte))
					case func([]byte, []byte) int:
						go f(vm.general(1).([]byte), vm.general(2).([]byte))
					case func([]byte, []byte) bool:
						go f(vm.general(1).([]byte), vm.general(2).([]byte))
					default:
						startNativeGoroutine = true
						fn.slow()
					}
				} else {
					switch f := fn.fast.(type) {
					case func(string) int:
						vm.setInt(1, int64(f(vm.string(1))))
					case func(string) string:
						vm.setString(1, f(vm.string(2)))
					case func(string, string) int:
						vm.setInt(1, int64(f(vm.string(1), vm.string(2))))
					case func(string, int) string:
						vm.setString(1, f(vm.string(2), int(vm.int(1))))
					case func(string, string) bool:
						vm.setBool(1, f(vm.string(1), vm.string(2)))
					case func([]byte) []byte:
						vm.setGeneral(1, f(vm.general(2).([]byte)))
					case func([]byte, []byte) int:
						vm.setInt(1, int64(f(vm.general(1).([]byte), vm.general(2).([]byte))))
					case func([]byte, []byte) bool:
						vm.setBool(1, f(vm.general(1).([]byte), vm.general(2).([]byte)))
					default:
						fn.slow()
					}
				}
			}
			if fn.fast == nil {
				variadic := fn.value.Type().IsVariadic()
				if len(fn.in) > 0 {
					vm.fp[0] += uint32(fn.outOff[0])
					vm.fp[1] += uint32(fn.outOff[1])
					vm.fp[2] += uint32(fn.outOff[2])
					vm.fp[3] += uint32(fn.outOff[3])
					lastNonVariadic := len(fn.in)
					if variadic && c != NoVariadic {
						lastNonVariadic--
					}
					for i, k := range fn.in {
						if i < lastNonVariadic {
							switch k {
							case Bool:
								fn.args[i].SetBool(vm.bool(1))
								vm.fp[0]++
							case Int:
								fn.args[i].SetInt(vm.int(1))
								vm.fp[0]++
							case Uint:
								fn.args[i].SetUint(uint64(vm.int(1)))
								vm.fp[0]++
							case Float64:
								fn.args[i].SetFloat(vm.float(1))
								vm.fp[1]++
							case String:
								fn.args[i].SetString(vm.string(1))
								vm.fp[2]++
							case Func:
								f := vm.general(1).(*callable)
								fn.args[i].Set(f.reflectValue())
								vm.fp[3]++
							default:
								fn.args[i].Set(reflect.ValueOf(vm.general(1)))
								vm.fp[3]++
							}
						} else {
							sliceType := fn.args[i].Type()
							slice := reflect.MakeSlice(sliceType, int(c), int(c))
							k := sliceType.Elem().Kind()
							switch {
							case k == reflect.Bool:
								for j := 0; j < int(c); j++ {
									slice.Index(j).SetBool(vm.bool(int8(j + 1)))
								}
							case reflect.Int <= k && k <= reflect.Int64:
								for j := 0; j < int(c); j++ {
									slice.Index(j).SetInt(vm.int(int8(j + 1)))
								}
							case reflect.Uint <= k && k <= reflect.Uint64:
								for j := 0; j < int(c); j++ {
									slice.Index(j).SetUint(uint64(vm.int(int8(j + 1))))
								}
							case k == reflect.Float64 || k == reflect.Float32:
								for j := 0; j < int(c); j++ {
									slice.Index(j).SetFloat(vm.float(int8(j + 1)))
								}
							case k == reflect.String:
								for j := 0; j < int(c); j++ {
									slice.Index(j).SetString(vm.string(int8(j + 1)))
								}
							case k == reflect.Func:
								for j := 0; j < int(c); j++ {
									f := vm.general(int8(j + 1)).(*callable)
									slice.Index(j).Set(f.reflectValue())
								}
							default:
								for j := 0; j < int(c); j++ {
									slice.Index(j).Set(reflect.ValueOf(vm.general(int8(j + 1))))
								}
							}
							fn.args[i].Set(slice)
						}
					}
					vm.fp[0] = fp[0] + uint32(off.op)
					vm.fp[1] = fp[1] + uint32(off.a)
					vm.fp[2] = fp[2] + uint32(off.b)
					vm.fp[3] = fp[3] + uint32(off.c)
				}
				if startNativeGoroutine {
					startNativeGoroutine = false
					if variadic {
						go fn.value.CallSlice(fn.args)
					} else {
						go fn.value.Call(fn.args)
					}
				} else {
					var ret []reflect.Value
					if variadic {
						ret = fn.value.CallSlice(fn.args)
					} else {
						ret = fn.value.Call(fn.args)
					}
					for i, k := range fn.out {
						switch k {
						case Bool:
							vm.setBool(1, ret[i].Bool())
							vm.fp[0]++
						case Int:
							vm.setInt(1, ret[i].Int())
							vm.fp[0]++
						case Uint:
							vm.setInt(1, int64(ret[i].Uint()))
							vm.fp[0]++
						case Float64:
							vm.setFloat(1, ret[i].Float())
							vm.fp[1]++
						case String:
							vm.setString(1, ret[i].String())
							vm.fp[2]++
						case Func:

						default:
							vm.setGeneral(1, ret[i].Interface())
							vm.fp[3]++
						}
					}
				}
			}
			vm.fp = fp
			pc++

		// Cap
		case opCap:
			vm.setInt(c, int64(reflect.ValueOf(a).Cap()))

		// Continue
		case opContinue:
			return int(a)

		// Convert
		case opConvert:
			t := vm.fn.types[uint8(b)]
			vm.setGeneral(c, reflect.ValueOf(vm.general(a)).Convert(t).Interface())
		case opConvertInt:
			t := vm.fn.types[uint8(b)]
			vm.setGeneral(c, reflect.ValueOf(vm.int(a)).Convert(t).Interface())
		case opConvertUint:
			t := vm.fn.types[uint8(b)]
			vm.setGeneral(c, reflect.ValueOf(uint64(vm.int(a))).Convert(t).Interface())
		case opConvertFloat:
			t := vm.fn.types[uint8(b)]
			vm.setGeneral(c, reflect.ValueOf(vm.float(a)).Convert(t).Interface())
		case opConvertString:
			t := vm.fn.types[uint8(b)]
			vm.setGeneral(c, reflect.ValueOf(vm.string(a)).Convert(t).Interface())

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

		// Func
		case opFunc:
			fn := vm.fn.literals[uint8(b)]
			var vars []interface{}
			if fn.crefs != nil {
				vars = make([]interface{}, len(fn.crefs))
				for i, ref := range fn.crefs {
					if ref < 0 {
						vars[i] = vm.general(int8(-ref))
					} else {
						vars[i] = vm.cvars[ref]
					}
				}
			}
			vm.setGeneral(c, &callable{scrigo: fn, vars: vars})

		// GetFunc
		case opGetFunc:
			var fn interface{}
			pkg := vm.fn.pkg
			if a != CurrentPackage {
				pkg = pkg.packages[uint8(a)]
			}
			if pkg.scrigoFunctions == nil {
				fn = &callable{native: pkg.nativeFunctions[uint8(b)]}
			} else {
				fn = &callable{scrigo: pkg.scrigoFunctions[uint8(b)]}
			}
			vm.setGeneral(c, fn)

		// GetVar
		case opGetVar:
			pkg := vm.fn.pkg
			if a > 1 {
				pkg = pkg.packages[uint8(a)-2]
			}
			v := pkg.variables[uint8(b)]
			switch v := v.(type) {
			case *int:
				vm.setInt(c, int64(*v))
			case *float64:
				vm.setFloat(c, *v)
			case *string:
				vm.setString(c, *v)
			default:
				rv := reflect.ValueOf(v).Elem()
				switch k := rv.Kind(); {
				case reflect.Int < k && k <= reflect.Int64:
					vm.setInt(c, rv.Int())
				case reflect.Uint <= k && k <= reflect.Uint64:
					vm.setInt(c, int64(rv.Uint()))
				case k == reflect.Float32:
					vm.setFloat(c, rv.Float())
				default:
					vm.setGeneral(c, rv.Interface())
				}
			}

		// Go
		case opGo:
			if !vm.startScrigoGoroutine(pc) {
				startNativeGoroutine = true
			}

		// Goto
		case opGoto:
			pc = decodeAddr(a, b, c)

		// If
		case opIf:
			var cond bool
			v1 := vm.general(a)
			switch Condition(b) {
			case ConditionNil:
				cond = v1 == nil
			case ConditionNotNil:
				cond = v1 != nil
				if cond {
					pc++
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
				pc++
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
				pc++
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
				pc++
			}
		case opIfString, -opIfString:
			var cond bool
			v1 := vm.string(a)
			if Condition(b) < ConditionEqualLen {
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
				pc++
			}

		// Index
		case opIndex, -opIndex:
			vm.setGeneral(c, reflect.ValueOf(vm.general(a)).Index(int(vm.intk(b, op < 0))).Interface())

		// JmpOk
		case opJmpOk:
			if vm.ok {
				pc = decodeAddr(a, b, c)
			}

		// JmpNotOk
		case opJmpNotOk:
			if !vm.ok {
				pc = decodeAddr(a, b, c)
			}

		// Len
		case opLen:
			var length int
			if a == 0 {
				length = len(vm.string(b))
			} else {
				v := vm.general(b)
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

		// MakeChan
		case opMakeChan, -opMakeChan:
			vm.setGeneral(c, reflect.MakeChan(vm.fn.types[uint8(a)], int(vm.intk(b, op < 0))).Interface())

		// MakeMap
		case opMakeMap, -opMakeMap:
			t := vm.fn.types[int(uint8(a))]
			n := int(vm.intk(b, op < 0))
			vm.setGeneral(c, reflect.MakeMapWithSize(t, n))

		// MakeSlice
		case opMakeSlice:
			//typ := vm.getType(a)
			//len := int(vm.int(b))
			//cap := int(vm.int(c))
			//vm.pushValue(reflectValue.MakeSlice(typ, len, cap))

		// MapIndex
		case opMapIndex, -opMapIndex:
			m := reflect.ValueOf(vm.general(a))
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
				key = reflect.ValueOf(vm.general(b))
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
				vm.setGeneral(c, elem.Interface())
			}

		// MapIndexStringInt
		case opMapIndexStringInt, -opMapIndexStringInt:
			vm.setInt(c, int64(vm.general(a).(map[string]int)[vm.stringk(b, op < 0)]))

		// MapIndexStringBool
		case opMapIndexStringBool, -opMapIndexStringBool:
			vm.setBool(c, vm.general(a).(map[string]bool)[vm.stringk(b, op < 0)])

		// MapIndexStringString
		case opMapIndexStringString, -opMapIndexStringString:
			vm.setString(c, vm.general(a).(map[string]string)[vm.stringk(b, op < 0)])

		// MapIndexStringInterface
		case opMapIndexStringInterface, -opMapIndexStringInterface:
			vm.setGeneral(c, vm.general(a).(map[string]interface{})[vm.stringk(b, op < 0)])

		// Move
		case opMove:
			vm.setGeneral(c, vm.general(b))
		case -opMove:
			vm.setGeneral(c, vm.generalk(b, true))
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
			var v interface{}
			switch t.Kind() {
			case reflect.Int:
				v = new(int)
			case reflect.Float64:
				v = new(float64)
			case reflect.String:
				v = new(string)
			default:
				v = reflect.New(t).Interface()
			}
			vm.setGeneral(c, v)

		// Or
		case opOr, -opOr:
			vm.setInt(c, vm.int(a)|vm.intk(b, op < 0))

		case opPrint:
			print(vm.general(a))

		// Range
		case opRange:
			var cont int
			s := vm.general(c)
			switch s := s.(type) {
			case []int:
				for i, v := range s {
					if a != 0 {
						vm.setInt(a, int64(i))
					}
					if b != 0 {
						vm.setInt(b, int64(v))
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case []byte:
				for i, v := range s {
					if a != 0 {
						vm.setInt(a, int64(i))
					}
					if b != 0 {
						vm.setInt(b, int64(v))
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case []rune:
				for i, v := range s {
					if a != 0 {
						vm.setInt(a, int64(i))
					}
					if b != 0 {
						vm.setInt(b, int64(v))
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case []float64:
				for i, v := range s {
					if a != 0 {
						vm.setInt(a, int64(i))
					}
					if b != 0 {
						vm.setFloat(b, v)
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case []string:
				for i, v := range s {
					if a != 0 {
						vm.setInt(a, int64(i))
					}
					if b != 0 {
						vm.setString(b, v)
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case []interface{}:
				for i, v := range s {
					if a != 0 {
						vm.setInt(a, int64(i))
					}
					if b != 0 {
						vm.setGeneral(b, v)
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case map[string]int:
				for i, v := range s {
					if a != 0 {
						vm.setString(a, i)
					}
					if b != 0 {
						vm.setInt(b, int64(v))
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case map[string]bool:
				for i, v := range s {
					if a != 0 {
						vm.setString(a, i)
					}
					if b != 0 {
						vm.setBool(b, v)
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case map[string]string:
				for i, v := range s {
					if a != 0 {
						vm.setString(a, i)
					}
					if b != 0 {
						vm.setString(b, v)
					}
					if cont = vm.run(); cont > 0 {
						break
					}
				}
			case map[string]interface{}:
				for i, v := range s {
					if a != 0 {
						vm.setString(a, i)
					}
					if b != 0 {
						vm.setGeneral(b, v)
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
						if a != 0 {
							vm.set(a, iter.Key())
						}
						if b != 0 {
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
						if a != 0 {
							vm.setInt(a, int64(i))
						}
						if b != 0 {
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

		// Receive
		case opReceive:
			ch := vm.general(a)
			switch ch := ch.(type) {
			case chan struct{}:
				vm.setGeneral(c, <-ch)
			case chan int:
				vm.setGeneral(c, <-ch)
			case chan bool:
				vm.setGeneral(c, <-ch)
			case chan string:
				vm.setGeneral(c, <-ch)
			case chan rune:
				vm.setGeneral(c, <-ch)
			default:
				x, ok := reflect.ValueOf(ch).Recv()
				vm.ok = ok
				if b != 0 {
					vm.setBool(b, ok)
				}
				if c != 0 {
					vm.setGeneral(c, x.Interface())
				}
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
			var call Call
			i := len(vm.calls)
			if i == 0 {
				return maxInt8
			}
			for {
				i--
				call = vm.calls[i]
				if !call.tail {
					break
				}
			}
			vm.calls = vm.calls[:i]
			vm.fp = call.fp
			pc = call.pc
			vm.fn = call.fn
			vm.cvars = call.cvars

		// Selector
		case opSelector:
			v := reflect.ValueOf(vm.general(a)).Field(int(uint8(b)))
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
				vm.setGeneral(c, v.Interface())
			}

		// Send
		case opSend:
			ch := vm.general(c)
			switch ch := ch.(type) {
			case chan struct{}:
				ch <- struct{}{}
			case chan int:
				ch <- vm.general(a).(int)
			case chan bool:
				ch <- vm.general(a).(bool)
			case chan string:
				ch <- vm.general(a).(string)
			case chan rune:
				ch <- vm.general(a).(rune)
			default:
				reflect.ValueOf(ch).Send(reflect.ValueOf(vm.general(a)))
			}

		// SetVar
		case opSetVar, -opSetVar:
			pkg := vm.fn.pkg
			if a > 1 {
				pkg = pkg.packages[uint8(b)-2]
			}
			v := pkg.variables[uint8(c)]
			switch v := v.(type) {
			case *int:
				*v = int(vm.intk(a, op < 0))
			case *float64:
				*v = vm.floatk(a, op < 0)
			case *string:
				*v = vm.stringk(a, op < 0)
			default:
				rv := reflect.ValueOf(v).Elem()
				switch k := rv.Kind(); {
				case reflect.Int < k && k <= reflect.Int64:
					rv.SetInt(vm.intk(a, op < 0))
				case reflect.Uint <= k && k <= reflect.Uint64:
					rv.SetUint(uint64(vm.intk(a, op < 0)))
				case k == reflect.Float32:
					rv.SetFloat(vm.floatk(a, op < 0))
				default:
					rv.Set(reflect.ValueOf(vm.general(a)))
				}
			}

		// SliceIndex
		case opSliceIndex, -opSliceIndex:
			v := vm.general(a)
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
				vm.setGeneral(c, v[i])
			default:
				vm.setGeneral(c, reflect.ValueOf(v).Index(i).Interface())
			}

		// StringIndex
		case opStringIndex, -opStringIndex:
			vm.setInt(c, int64(vm.string(a)[int(vm.intk(b, op < 0))]))

		// Sub
		case opSubInt:
			vm.setInt(c, vm.int(a)-vm.int(b))
		case -opSubInt:
			vm.setInt(c, vm.int(a)-int64(b))
		case opSubInt8, -opSubInt8:
			vm.setInt(c, int64(int8(vm.int(a)-vm.intk(b, op < 0))))
		case opSubInt16, -opSubInt16:
			vm.setInt(c, int64(int32(vm.int(a)-vm.intk(b, op < 0))))
		case opSubInt32, -opSubInt32:
			vm.setInt(c, int64(int32(vm.int(a)-vm.intk(b, op < 0))))
		case opSubFloat32, -opSubFloat32:
			vm.setFloat(c, float64(float32(float32(vm.float(a))-float32(vm.floatk(b, op < 0)))))
		case opSubFloat64:
			vm.setFloat(c, vm.float(a)-vm.float(b))
		case -opSubFloat64:
			vm.setFloat(c, vm.float(a)-float64(b))

		// SubInv
		case opSubInvInt, -opSubInvInt:
			vm.setInt(c, vm.intk(b, op < 0)-vm.int(a))
		case opSubInvInt8, -opSubInvInt8:
			vm.setInt(c, int64(int8(vm.intk(b, op < 0)-vm.int(a))))
		case opSubInvInt16, -opSubInvInt16:
			vm.setInt(c, int64(int32(vm.intk(b, op < 0)-vm.int(a))))
		case opSubInvInt32, -opSubInvInt32:
			vm.setInt(c, int64(int32(vm.intk(b, op < 0)-vm.int(a))))
		case opSubInvFloat32, -opSubInvFloat32:
			vm.setFloat(c, float64(float32(float32(vm.floatk(b, op < 0))-float32(vm.float(a)))))
		case opSubInvFloat64:
			vm.setFloat(c, vm.float(b)-vm.float(a))
		case -opSubInvFloat64:
			vm.setFloat(c, float64(b)-vm.float(a))

		// TailCall
		case opTailCall:
			vm.calls = append(vm.calls, Call{fn: vm.fn, cvars: vm.cvars, pc: pc, tail: true})
			pc = 0
			if b != CurrentFunction {
				var fn *ScrigoFunction
				if a == NoPackage {
					closure := vm.general(b).(*callable)
					fn = closure.scrigo
					vm.cvars = closure.vars
				} else {
					pkg := vm.fn.pkg
					if a != CurrentPackage {
						pkg = pkg.packages[uint8(a)]
					}
					fn = pkg.scrigoFunctions[uint8(b)]
					vm.cvars = nil
				}
				if vm.fp[0]+uint32(fn.regnum[0]) > vm.st[0] {
					vm.moreIntStack()
				}
				if vm.fp[1]+uint32(fn.regnum[1]) > vm.st[1] {
					vm.moreFloatStack()
				}
				if vm.fp[2]+uint32(fn.regnum[2]) > vm.st[2] {
					vm.moreStringStack()
				}
				if vm.fp[3]+uint32(fn.regnum[3]) > vm.st[3] {
					vm.moreGeneralStack()
				}
				vm.fn = fn
			}

		// opXor
		case opXor, -opXor:
			vm.setInt(c, vm.int(a)^vm.intk(b, op < 0))

		}

	}

}
