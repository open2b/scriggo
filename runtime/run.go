// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"reflect"
)

func (vm *VM) runFunc(fn *Function, vars []interface{}) error {
	vm.fn = fn
	vm.vars = vars
	for {
		err := vm.runRecoverable()
		if err == nil {
			break
		}
		p, ok := err.(*Panic)
		if !ok {
			return err
		}
		p.next = vm.panic
		vm.panic = p
		if len(vm.calls) == 0 {
			break
		}
		vm.calls = append(vm.calls, callFrame{cl: callable{fn: vm.fn}, fp: vm.fp, status: panicked})
		vm.fn = nil
	}
	if vm.panic != nil {
		return vm.panic
	}
	return nil
}

func (vm *VM) runRecoverable() (err error) {
	panicking := true
	defer func() {
		if panicking {
			msg := recover()
			err = vm.convertPanic(msg)
		}
	}()
	if vm.fn != nil || vm.nextCall() {
		vm.run()
	}
	panicking = false
	return nil
}

func (vm *VM) run() (Addr, bool) {

	var startPredefinedGoroutine bool
	var hasDefaultCase bool

	var op Operation
	var a, b, c int8

	for {

		if vm.done != nil {
			select {
			case <-vm.done:
				panic(OutOfTimeError{vm.env})
			default:
			}
		}

		if vm.env.trace != nil {
			vm.invokeTraceFunc()
		}

		in := vm.fn.Body[vm.pc]

		vm.pc++
		op, a, b, c = in.Op, in.A, in.B, in.C

		// If an instruction needs to change the program counter,
		// it must be changed, if possible, at the end of the instruction execution.

		switch op {

		// Add
		case OpAddInt8, -OpAddInt8:
			vm.setInt(c, int64(int8(vm.int(a)+vm.intk(b, op < 0))))
		case OpAddInt16, -OpAddInt16:
			vm.setInt(c, int64(int16(vm.int(a)+vm.intk(b, op < 0))))
		case OpAddInt32, -OpAddInt32:
			vm.setInt(c, int64(int32(vm.int(a)+vm.intk(b, op < 0))))
		case OpAddInt64:
			vm.setInt(c, vm.int(a)+vm.int(b))
		case -OpAddInt64:
			vm.setInt(c, vm.int(a)+int64(b))
		case OpAddFloat32, -OpAddFloat32:
			vm.setFloat(c, float64(float32(vm.float(a)+vm.floatk(b, op < 0))))
		case OpAddFloat64:
			vm.setFloat(c, vm.float(a)+vm.float(b))
		case -OpAddFloat64:
			vm.setFloat(c, vm.float(a)+float64(b))

		// Addr
		case OpAddr:
			rv := reflect.ValueOf(vm.general(a))
			switch rv.Kind() {
			case reflect.Slice:
				i := int(vm.int(b))
				vm.setFromReflectValue(c, rv.Index(i).Addr())
			case reflect.Ptr:
				i := decodeFieldIndex(vm.fn.Constants.Int[uint8(b)])
				v := rv.Elem().FieldByIndex(i).Addr()
				vm.setFromReflectValue(c, v)
			}

		// Alloc
		case OpAlloc:
			vm.alloc()
		case -OpAlloc:
			if vm.env.limitMemory {
				bytes := decodeUint24(a, b, c)
				var free int
				vm.env.mu.Lock()
				free = vm.env.freeMemory
				if free >= 0 {
					free -= int(bytes)
					vm.env.freeMemory = free
				}
				vm.env.mu.Unlock()
				if free < 0 {
					panic(OutOfMemoryError{vm.env})
				}
			}

		// And
		case OpAnd, -OpAnd:
			vm.setInt(c, vm.int(a)&vm.intk(b, op < 0))

		// AndNot
		case OpAndNot, -OpAndNot:
			vm.setInt(c, vm.int(a)&^vm.intk(b, op < 0))

		// Append
		case OpAppend:
			vm.setGeneral(c, vm.appendSlice(a, int(b-a), vm.general(c)))

		// AppendSlice
		case OpAppendSlice:
			src := vm.general(a)
			dst := vm.general(c)
			switch s := src.(type) {
			case []int:
				vm.setGeneral(c, append(dst.([]int), s...))
			case []byte:
				vm.setGeneral(c, append(dst.([]byte), s...))
			case []rune:
				vm.setGeneral(c, append(dst.([]rune), s...))
			case []float64:
				vm.setGeneral(c, append(dst.([]float64), s...))
			case []string:
				vm.setGeneral(c, append(dst.([]string), s...))
			case []interface{}:
				vm.setGeneral(c, append(dst.([]interface{}), s...))
			default:
				sv := reflect.ValueOf(src)
				dv := reflect.ValueOf(dst)
				vm.setGeneral(c, reflect.AppendSlice(dv, sv).Interface())
			}

		// Assert
		case OpAssert:
			v := reflect.ValueOf(vm.general(a))
			t := vm.fn.Types[uint8(b)]
			var ok bool
			if v.IsValid() {
				if t.Kind() == reflect.Interface {
					ok = v.Type().Implements(t)
				} else {
					ok = v.Type() == t
				}
			}
			vm.ok = ok
			if !ok {
				in := vm.fn.Body[vm.pc]
				if in.Op == OpPanic {
					var concrete reflect.Type
					if v.IsValid() {
						concrete = v.Type()
					}
					var method string
					if t.Kind() == reflect.Interface {
						method = missingMethod(concrete, t)
					}
					panic(errTypeAssertion(vm.fn.Types[uint8(in.C)], concrete, t, method))
				}
			}
			if c != 0 {
				switch t.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					var n int64
					if ok {
						n = v.Int()
					}
					vm.setInt(c, n)
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
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
					rv := reflect.New(t).Elem()
					if ok {
						rv.Set(v)
					}
					vm.setFromReflectValue(c, rv)
				}
			}
			if ok {
				vm.pc++
			}

		// Break
		case OpBreak:
			return Addr(decodeUint24(a, b, c)), true

		// Call
		case OpCall:
			call := callFrame{cl: callable{fn: vm.fn, vars: vm.vars}, fp: vm.fp, pc: vm.pc + 1}
			fn := vm.fn.Functions[uint8(a)]
			off := vm.fn.Body[vm.pc]
			vm.fp[0] += Addr(off.Op)
			if vm.fp[0]+Addr(fn.NumReg[0]) > vm.st[0] {
				vm.moreIntStack()
			}
			vm.fp[1] += Addr(off.A)
			if vm.fp[1]+Addr(fn.NumReg[1]) > vm.st[1] {
				vm.moreFloatStack()
			}
			vm.fp[2] += Addr(off.B)
			if vm.fp[2]+Addr(fn.NumReg[2]) > vm.st[2] {
				vm.moreStringStack()
			}
			vm.fp[3] += Addr(off.C)
			if vm.fp[3]+Addr(fn.NumReg[3]) > vm.st[3] {
				vm.moreGeneralStack()
			}
			vm.fn = fn
			vm.vars = vm.env.globals
			vm.calls = append(vm.calls, call)
			vm.pc = 0

		// CallIndirect
		case OpCallIndirect:
			f := vm.general(a).(*callable)
			if f.fn == nil {
				off := vm.fn.Body[vm.pc]
				vm.callPredefined(f.Predefined(), c, StackShift{int8(off.Op), off.A, off.B, off.C}, startPredefinedGoroutine)
				startPredefinedGoroutine = false
				vm.pc++
			} else {
				call := callFrame{cl: callable{fn: vm.fn, vars: vm.vars}, fp: vm.fp, pc: vm.pc + 1}
				fn := f.fn
				off := vm.fn.Body[vm.pc]
				vm.fp[0] += Addr(off.Op)
				if vm.fp[0]+Addr(fn.NumReg[0]) > vm.st[0] {
					vm.moreIntStack()
				}
				vm.fp[1] += Addr(off.A)
				if vm.fp[1]+Addr(fn.NumReg[1]) > vm.st[1] {
					vm.moreFloatStack()
				}
				vm.fp[2] += Addr(off.B)
				if vm.fp[2]+Addr(fn.NumReg[2]) > vm.st[2] {
					vm.moreStringStack()
				}
				vm.fp[3] += Addr(off.C)
				if vm.fp[3]+Addr(fn.NumReg[3]) > vm.st[3] {
					vm.moreGeneralStack()
				}
				vm.fn = fn
				vm.vars = f.vars
				vm.calls = append(vm.calls, call)
				vm.pc = 0
			}

		// CallPredefined
		case OpCallPredefined:
			fn := vm.fn.Predefined[uint8(a)]
			off := vm.fn.Body[vm.pc]
			vm.callPredefined(fn, c, StackShift{int8(off.Op), off.A, off.B, off.C}, startPredefinedGoroutine)
			startPredefinedGoroutine = false
			vm.pc++

		// Cap
		case OpCap:
			vm.setInt(c, int64(reflect.ValueOf(vm.general(a)).Cap()))

		// Case
		case OpCase, -OpCase:
			dir := reflect.SelectDir(a)
			i := len(vm.cases)
			if i == cap(vm.cases) {
				vm.cases = append(vm.cases, reflect.SelectCase{Dir: dir})
			} else {
				vm.cases = vm.cases[:i+1]
				vm.cases[i].Dir = dir
				if dir == reflect.SelectDefault {
					vm.cases[i].Chan = reflect.Value{}
				}
				if dir != reflect.SelectSend {
					vm.cases[i].Send = reflect.Value{}
				}
			}
			if dir == reflect.SelectDefault {
				hasDefaultCase = true
			} else {
				vm.cases[i].Chan = reflect.ValueOf(vm.general(c))
				if dir == reflect.SelectSend {
					t := vm.cases[i].Chan.Type().Elem()
					if !vm.cases[i].Send.IsValid() || t != vm.cases[i].Send.Type() {
						vm.cases[i].Send = reflect.New(t).Elem()
					}
					vm.getIntoReflectValue(b, vm.cases[i].Send, op < 0)
				}
			}
			vm.pc++

		// Close
		case OpClose:
			reflect.ValueOf(vm.general(a)).Close()

		// Complex
		case OpComplex64:
			var re, im float32
			if a > 0 {
				re = float32(vm.float(a))
			}
			if b > 0 {
				im = float32(vm.float(b))
			}
			vm.setGeneral(c, complex(re, im))
		case OpComplex128:
			var re, im float64
			if a > 0 {
				re = vm.float(a)
			}
			if b > 0 {
				im = vm.float(b)
			}
			vm.setGeneral(c, complex(re, im))

		// Continue
		case OpContinue:
			return Addr(decodeUint24(a, b, c)), false

		// Convert
		case OpConvert:
			t := vm.fn.Types[uint8(b)]
			if t == sliceByteType {
				vm.setString(c, string(vm.general(a).([]byte)))
			} else {
				switch t.Kind() {
				case reflect.String:
					vm.setString(c, reflect.ValueOf(vm.general(a)).Convert(t).String())
				default:
					vm.setGeneral(c, reflect.ValueOf(vm.general(a)).Convert(t).Interface())
				}
			}
		case OpConvertInt:
			t := vm.fn.Types[uint8(b)]
			v := vm.int(a)
			switch t.Kind() {
			case reflect.Int:
				vm.setInt(c, int64(int(v)))
			case reflect.Int8:
				vm.setInt(c, int64(int8(v)))
			case reflect.Int16:
				vm.setInt(c, int64(int16(v)))
			case reflect.Int32:
				vm.setInt(c, int64(int32(v)))
			case reflect.Int64:
				vm.setInt(c, v)
			case reflect.Uint, reflect.Uintptr:
				vm.setInt(c, int64(uint(v)))
			case reflect.Uint8:
				vm.setInt(c, int64(uint8(v)))
			case reflect.Uint16:
				vm.setInt(c, int64(uint16(v)))
			case reflect.Uint32:
				vm.setInt(c, int64(uint32(v)))
			case reflect.Uint64:
				vm.setInt(c, v)
			case reflect.Float32:
				vm.setFloat(c, float64(float32(v)))
			case reflect.Float64:
				vm.setFloat(c, float64(v))
			case reflect.String:
				vm.setString(c, string(v))
			}
		case OpConvertUint:
			t := vm.fn.Types[uint8(b)]
			v := uint64(vm.int(a))
			switch t.Kind() {
			case reflect.Int:
				vm.setInt(c, int64(int(v)))
			case reflect.Int8:
				vm.setInt(c, int64(int8(v)))
			case reflect.Int16:
				vm.setInt(c, int64(int16(v)))
			case reflect.Int32:
				vm.setInt(c, int64(int32(v)))
			case reflect.Int64:
				vm.setInt(c, int64(v))
			case reflect.Uint, reflect.Uintptr:
				vm.setInt(c, int64(uint(v)))
			case reflect.Uint8:
				vm.setInt(c, int64(uint8(v)))
			case reflect.Uint16:
				vm.setInt(c, int64(uint16(v)))
			case reflect.Uint32:
				vm.setInt(c, int64(uint32(v)))
			case reflect.Uint64:
				vm.setInt(c, int64(v))
			case reflect.Float32:
				vm.setFloat(c, float64(float32(v)))
			case reflect.Float64:
				vm.setFloat(c, float64(v))
			case reflect.String:
				vm.setString(c, string(v))
			}
		case OpConvertFloat:
			t := vm.fn.Types[uint8(b)]
			v := vm.float(a)
			switch t.Kind() {
			case reflect.Int:
				vm.setInt(c, int64(int(v)))
			case reflect.Int8:
				vm.setInt(c, int64(int8(v)))
			case reflect.Int16:
				vm.setInt(c, int64(int16(v)))
			case reflect.Int32:
				vm.setInt(c, int64(int32(v)))
			case reflect.Int64:
				vm.setInt(c, int64(v))
			case reflect.Uint, reflect.Uintptr:
				vm.setInt(c, int64(uint(v)))
			case reflect.Uint8:
				vm.setInt(c, int64(uint8(v)))
			case reflect.Uint16:
				vm.setInt(c, int64(uint16(v)))
			case reflect.Uint32:
				vm.setInt(c, int64(uint32(v)))
			case reflect.Uint64:
				vm.setInt(c, int64(uint64(v)))
			case reflect.Float32:
				vm.setFloat(c, float64(float32(v)))
			case reflect.Float64:
				vm.setFloat(c, v)
			}
		case OpConvertString:
			t := vm.fn.Types[uint8(b)]
			if t == sliceByteType {
				vm.setGeneral(c, []byte(vm.string(a)))
			} else {
				vm.setGeneral(c, reflect.ValueOf(vm.string(a)).Convert(t).Interface())
			}

		// Concat
		case OpConcat:
			vm.setString(c, vm.string(a)+vm.string(b))

		// Copy
		case OpCopy:
			src := reflect.ValueOf(vm.general(a))
			dst := reflect.ValueOf(vm.general(c))
			n := reflect.Copy(dst, src)
			if b != 0 {
				vm.setInt(b, int64(n))
			}

		// Defer
		case OpDefer:
			cl := vm.general(a).(*callable)
			off := vm.fn.Body[vm.pc]
			arg := vm.fn.Body[vm.pc+1]
			fp := [4]Addr{
				vm.fp[0] + Addr(off.Op),
				vm.fp[1] + Addr(off.A),
				vm.fp[2] + Addr(off.B),
				vm.fp[3] + Addr(off.C),
			}
			vm.swapStack(&vm.fp, &fp, StackShift{int8(arg.Op), arg.A, arg.B, arg.C})
			vm.calls = append(vm.calls, callFrame{cl: *cl, fp: fp, pc: 0, status: deferred, numVariadic: c})
			vm.pc += 2

		// Delete
		case OpDelete:
			m := reflect.ValueOf(vm.general(a))
			k := reflect.ValueOf(vm.general(b))
			m.SetMapIndex(k, reflect.Value{})

		// Div
		case OpDivInt8, -OpDivInt8:
			vm.setInt(c, int64(int8(vm.int(a))/int8(vm.intk(b, op < 0))))
		case OpDivInt16, -OpDivInt16:
			vm.setInt(c, int64(int16(vm.int(a))/int16(vm.intk(b, op < 0))))
		case OpDivInt32, -OpDivInt32:
			vm.setInt(c, int64(int32(vm.int(a))/int32(vm.intk(b, op < 0))))
		case OpDivInt64:
			vm.setInt(c, vm.int(a)/vm.int(b))
		case -OpDivInt64:
			vm.setInt(c, vm.int(a)/int64(b))
		case OpDivUint8, -OpDivUint8:
			vm.setInt(c, int64(uint8(vm.int(a))/uint8(vm.intk(b, op < 0))))
		case OpDivUint16, -OpDivUint16:
			vm.setInt(c, int64(uint16(vm.int(a))/uint16(vm.intk(b, op < 0))))
		case OpDivUint32, -OpDivUint32:
			vm.setInt(c, int64(uint32(vm.int(a))/uint32(vm.intk(b, op < 0))))
		case OpDivUint64, -OpDivUint64:
			vm.setInt(c, int64(uint64(vm.int(a))/uint64(vm.intk(b, op < 0))))
		case OpDivFloat32, -OpDivFloat32:
			vm.setFloat(c, float64(float32(vm.float(a))/float32(vm.floatk(b, op < 0))))
		case OpDivFloat64:
			vm.setFloat(c, vm.float(a)/vm.float(b))
		case -OpDivFloat64:
			vm.setFloat(c, vm.float(a)/float64(b))

		// Field
		case OpField:
			i := decodeFieldIndex(vm.fn.Constants.Int[uint8(b)])
			v := reflect.ValueOf(vm.general(a)).Elem().FieldByIndex(i)
			vm.setFromReflectValue(c, v)

		// GetVar
		case OpGetVar:
			v := vm.vars[decodeInt16(a, b)]
			switch v := v.(type) {
			case *bool:
				vm.setBool(c, *v)
			case *int:
				vm.setInt(c, int64(*v))
			case *float64:
				vm.setFloat(c, *v)
			case *string:
				vm.setString(c, *v)
			default:
				rv := reflect.ValueOf(v).Elem()
				vm.setFromReflectValue(c, rv)
			}

		// GetVarAddr
		case OpGetVarAddr:
			v := vm.vars[decodeInt16(a, b)]
			vm.setFromReflectValue(c, reflect.ValueOf(v))

		// Go
		case OpGo:
			wasPredefined := vm.startGoroutine()
			if wasPredefined {
				startPredefinedGoroutine = true
			}

		// Goto
		case OpGoto:
			vm.pc = Addr(decodeUint24(a, b, c))

		// If
		case OpIf:
			var cond bool
			switch Condition(b) {
			case ConditionOK, ConditionNotOK:
				cond = vm.ok
			case ConditionInterfaceNil, ConditionInterfaceNotNil:
				cond = vm.general(a) == nil
			case ConditionNil, ConditionNotNil:
				switch v := vm.general(a).(type) {
				case []int:
					cond = v == nil
				case []string:
					cond = v == nil
				case []byte:
					cond = v == nil
				default:
					cond = reflect.ValueOf(v).IsNil()
				}
			case ConditionEqual, ConditionNotEqual:
				cond = vm.general(a) == vm.generalk(c, op < 0)
			}
			switch Condition(b) {
			case ConditionNotOK, ConditionInterfaceNotNil, ConditionNotNil, ConditionNotEqual:
				cond = !cond
			}
			if cond {
				vm.pc++
			}
		case OpIfInt, -OpIfInt:
			var cond bool
			if ConditionLessU <= Condition(b) && Condition(b) <= ConditionGreaterOrEqualU {
				v1 := uint64(vm.int(a))
				v2 := uint64(vm.intk(c, op < 0))
				switch Condition(b) {
				case ConditionLessU:
					cond = v1 < v2
				case ConditionLessOrEqualU:
					cond = v1 <= v2
				case ConditionGreaterU:
					cond = v1 > v2
				case ConditionGreaterOrEqualU:
					cond = v1 >= v2
				}
			} else {
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
			}
			if cond {
				vm.pc++
			}
		case OpIfFloat, -OpIfFloat:
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
		case OpIfString, -OpIfString:
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
				vm.pc++
			}

		// Index
		case OpIndex, -OpIndex:
			s := vm.general(a)
			i := int(vm.intk(b, op < 0))
			switch v := s.(type) {
			case []int:
				vm.setInt(c, int64(v[i]))
			case []byte:
				vm.setInt(c, int64(v[i]))
			case []rune:
				vm.setInt(c, int64(v[i]))
			case []float64:
				vm.setFloat(c, v[i])
			case []string:
				vm.setString(c, v[i])
			case []interface{}:
				vm.setGeneral(c, v[i])
			default:
				vm.setFromReflectValue(c, reflect.ValueOf(v).Index(i))
			}
		case OpIndexString, -OpIndexString:
			vm.setInt(c, int64(vm.string(a)[int(vm.intk(b, op < 0))]))

		// LeftShift
		case OpLeftShift8, -OpLeftShift8:
			vm.setInt(c, int64(int8(vm.int(a))<<uint(vm.intk(b, op < 0))))
		case OpLeftShift16, -OpLeftShift16:
			vm.setInt(c, int64(int16(vm.int(a))<<uint(vm.intk(b, op < 0))))
		case OpLeftShift32, -OpLeftShift32:
			vm.setInt(c, int64(int32(vm.int(a))<<uint(vm.intk(b, op < 0))))
		case OpLeftShift64, -OpLeftShift64:
			vm.setInt(c, vm.int(a)<<uint(vm.intk(b, op < 0)))

		// Len
		case OpLen:
			// TODO(marco): add other cases
			// TODO(marco): declare a new type for the condition
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

		// LoadData
		case OpLoadData:
			vm.setGeneral(c, vm.fn.Data[decodeInt16(a, b)])

		// OpLoadFunc
		case OpLoadFunc:
			if a == 1 {
				fn := callable{}
				fn.predefined = vm.fn.Predefined[uint8(b)]
				vm.setGeneral(c, &fn)
			} else {
				fn := vm.fn.Functions[uint8(b)]
				var vars []interface{}
				if fn.VarRefs != nil {
					vars = make([]interface{}, len(fn.VarRefs))
					for i, ref := range fn.VarRefs {
						if ref < 0 {
							vars[i] = vm.general(int8(-ref))
						} else {
							vars[i] = vm.vars[ref]
						}
					}
				} else {
					vars = vm.env.globals
				}
				vm.setGeneral(c, &callable{fn: fn, vars: vars})
			}

		// LoadNumber.
		case OpLoadNumber:
			switch a {
			case 0:
				vm.setInt(c, vm.fn.Constants.Int[uint8(b)])
			case 1:
				vm.setFloat(c, vm.fn.Constants.Float[uint8(b)])
			}

		// MakeChan
		case OpMakeChan, -OpMakeChan:
			typ := vm.fn.Types[uint8(a)]
			buffer := int(vm.intk(b, op < 0))
			vm.setGeneral(c, reflect.MakeChan(typ, buffer).Interface())

		// MakeMap
		case OpMakeMap, -OpMakeMap:
			typ := vm.fn.Types[uint8(a)]
			n := int(vm.intk(b, op < 0))
			if n > 0 {
				vm.setGeneral(c, reflect.MakeMapWithSize(typ, n).Interface())
			} else {
				vm.setGeneral(c, reflect.MakeMap(typ).Interface())
			}

		// MakeSlice
		case OpMakeSlice:
			typ := vm.fn.Types[uint8(a)]
			var len, cap int
			if b > 0 {
				next := vm.fn.Body[vm.pc]
				lenIsConst := (b & (1 << 1)) != 0
				len = int(vm.intk(next.A, lenIsConst))
				capIsConst := (b & (1 << 2)) != 0
				cap = int(vm.intk(next.B, capIsConst))
			}
			vm.setGeneral(c, reflect.MakeSlice(typ, len, cap).Interface())
			if b > 0 {
				vm.pc++
			}

		// MapIndex
		case OpMapIndex, -OpMapIndex:
			m := vm.general(a)
			switch m := m.(type) {
			case map[int]int:
				v, ok := m[int(vm.intk(b, op < 0))]
				vm.setInt(c, int64(v))
				vm.ok = ok
			case map[int]bool:
				v, ok := m[int(vm.intk(b, op < 0))]
				vm.setBool(c, v)
				vm.ok = ok
			case map[int]string:
				v, ok := m[int(vm.intk(b, op < 0))]
				vm.setString(c, v)
				vm.ok = ok
			case map[string]string:
				v, ok := m[vm.stringk(b, op < 0)]
				vm.setString(c, v)
				vm.ok = ok
			case map[string]bool:
				v, ok := m[vm.stringk(b, op < 0)]
				vm.setBool(c, v)
				vm.ok = ok
			case map[string]int:
				v, ok := m[vm.stringk(b, op < 0)]
				vm.setInt(c, int64(v))
				vm.ok = ok
			case map[string]interface{}:
				v, ok := m[vm.stringk(b, op < 0)]
				vm.setGeneral(c, v)
				vm.ok = ok
			default:
				mv := reflect.ValueOf(m)
				t := mv.Type()
				k := reflect.New(t.Key()).Elem()
				vm.getIntoReflectValue(b, k, op < 0)
				elem := reflect.New(t.Elem()).Elem()
				index := mv.MapIndex(k)
				vm.ok = index.IsValid()
				if vm.ok {
					elem.Set(index)
				}
				vm.setFromReflectValue(c, elem)
			}

		// MethodValue
		case OpMethodValue:
			receiver := vm.general(a)
			if receiver == nil {
				panic(runtimeError("runtime error: invalid memory address or nil pointer dereference"))
			}
			method := vm.stringk(b, true)
			vm.setGeneral(c, &callable{receiver: receiver, method: method})

		// Move
		case OpMove, -OpMove:
			switch Type(a) {
			case TypeFloat:
				vm.setFloat(c, vm.floatk(b, op < 0))
			case TypeGeneral:
				vm.setGeneral(c, vm.generalk(b, op < 0))
			case TypeInt:
				vm.setInt(c, vm.intk(b, op < 0))
			case TypeString:
				vm.setString(c, vm.stringk(b, op < 0))
			}

		// Mul
		case OpMulInt8, -OpMulInt8:
			vm.setInt(c, int64(int8(vm.int(a)*vm.intk(b, op < 0))))
		case OpMulInt16, -OpMulInt16:
			vm.setInt(c, int64(int16(vm.int(a)*vm.intk(b, op < 0))))
		case OpMulInt32, -OpMulInt32:
			vm.setInt(c, int64(int32(vm.int(a)*vm.intk(b, op < 0))))
		case OpMulInt64:
			vm.setInt(c, vm.int(a)*vm.int(b))
		case -OpMulInt64:
			vm.setInt(c, vm.int(a)*int64(b))
		case OpMulFloat32, -OpMulFloat32:
			vm.setFloat(c, float64(float32(vm.float(a))*float32(vm.floatk(b, op < 0))))
		case OpMulFloat64:
			vm.setFloat(c, vm.float(a)*vm.float(b))
		case -OpMulFloat64:
			vm.setFloat(c, vm.float(a)*float64(b))

		// New
		case OpNew:
			t := vm.fn.Types[uint8(b)]
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
		case OpOr, -OpOr:
			vm.setInt(c, vm.int(a)|vm.intk(b, op < 0))

		// Panic
		case OpPanic:
			panic(vm.general(a))

		// Print
		case OpPrint:
			vm.env.doPrint(vm.general(a))

		// Range
		case OpRange:
			var addr Addr
			var breakOut bool
			rangeAddress := vm.pc - 1
			bodyAddress := vm.pc + 1
			s := vm.general(a)
			switch s := s.(type) {
			case []int:
				for i, v := range s {
					if b != 0 {
						vm.setInt(b, int64(i))
					}
					if c != 0 {
						vm.setInt(c, int64(v))
					}
					vm.pc = bodyAddress
					addr, breakOut = vm.run()
					if addr != rangeAddress {
						return addr, breakOut
					}
					if breakOut {
						break
					}
				}
			case []byte:
				for i, v := range s {
					if b != 0 {
						vm.setInt(b, int64(i))
					}
					if c != 0 {
						vm.setInt(c, int64(v))
					}
					vm.pc = bodyAddress
					addr, breakOut = vm.run()
					if addr != rangeAddress {
						return addr, breakOut
					}
					if breakOut {
						break
					}
				}
			case []rune:
				for i, v := range s {
					if b != 0 {
						vm.setInt(b, int64(i))
					}
					if c != 0 {
						vm.setInt(c, int64(v))
					}
					vm.pc = bodyAddress
					addr, breakOut = vm.run()
					if addr != rangeAddress {
						return addr, breakOut
					}
					if breakOut {
						break
					}
				}
			case []float64:
				for i, v := range s {
					if b != 0 {
						vm.setInt(b, int64(i))
					}
					if c != 0 {
						vm.setFloat(c, v)
					}
					vm.pc = bodyAddress
					addr, breakOut = vm.run()
					if addr != rangeAddress {
						return addr, breakOut
					}
					if breakOut {
						break
					}
				}
			case []string:
				for i, v := range s {
					if b != 0 {
						vm.setInt(b, int64(i))
					}
					if c != 0 {
						vm.setString(c, v)
					}
					vm.pc = bodyAddress
					addr, breakOut = vm.run()
					if addr != rangeAddress {
						return addr, breakOut
					}
					if breakOut {
						break
					}
				}
			case []interface{}:
				for i, v := range s {
					if b != 0 {
						vm.setInt(b, int64(i))
					}
					if c != 0 {
						vm.setGeneral(c, v)
					}
					vm.pc = bodyAddress
					addr, breakOut = vm.run()
					if addr != rangeAddress {
						return addr, breakOut
					}
					if breakOut {
						break
					}
				}
			case map[string]int:
				for i, v := range s {
					if b != 0 {
						vm.setString(b, i)
					}
					if c != 0 {
						vm.setInt(c, int64(v))
					}
					vm.pc = bodyAddress
					addr, breakOut = vm.run()
					if addr != rangeAddress {
						return addr, breakOut
					}
					if breakOut {
						break
					}
				}
			case map[string]bool:
				for i, v := range s {
					if b != 0 {
						vm.setString(b, i)
					}
					if c != 0 {
						vm.setBool(c, v)
					}
					vm.pc = bodyAddress
					addr, breakOut = vm.run()
					if addr != rangeAddress {
						return addr, breakOut
					}
					if breakOut {
						break
					}
				}
			case map[string]string:
				for i, v := range s {
					if b != 0 {
						vm.setString(b, i)
					}
					if c != 0 {
						vm.setString(c, v)
					}
					vm.pc = bodyAddress
					addr, breakOut = vm.run()
					if addr != rangeAddress {
						return addr, breakOut
					}
					if breakOut {
						break
					}
				}
			case map[string]interface{}:
				for i, v := range s {
					if b != 0 {
						vm.setString(b, i)
					}
					if c != 0 {
						vm.setGeneral(c, v)
					}
					vm.pc = bodyAddress
					addr, breakOut = vm.run()
					if addr != rangeAddress {
						return addr, breakOut
					}
					if breakOut {
						break
					}
				}
			default:
				rs := reflect.ValueOf(s)
				kind := rs.Kind()
				if kind == reflect.Map {
					iter := rs.MapRange()
					for iter.Next() {
						if b != 0 {
							vm.setFromReflectValue(b, iter.Key())
						}
						if c != 0 {
							vm.setFromReflectValue(c, iter.Value())
						}
						vm.pc = bodyAddress
						addr, breakOut = vm.run()
						if addr != rangeAddress {
							return addr, breakOut
						}
						if breakOut {
							break
						}
					}
				} else {
					if kind == reflect.Ptr {
						rs = rs.Elem()
					}
					length := rs.Len()
					for i := 0; i < length; i++ {
						if b != 0 {
							vm.setInt(b, int64(i))
						}
						if c != 0 {
							vm.setFromReflectValue(c, rs.Index(i))
						}
						vm.pc = bodyAddress
						addr, breakOut = vm.run()
						if addr != rangeAddress {
							return addr, breakOut
						}
						if breakOut {
							break
						}
					}
				}
			}
			if !breakOut {
				vm.pc = rangeAddress + 1
			}

		// RangeString
		case OpRangeString, -OpRangeString:
			var addr Addr
			var breakOut bool
			rangeAddress := vm.pc - 1
			bodyAddress := vm.pc + 1
			s := vm.stringk(a, op < 0)
			for i, e := range s {
				if b != 0 {
					vm.setInt(b, int64(i))
				}
				if c != 0 {
					vm.setInt(c, int64(e))
				}
				vm.pc = bodyAddress
				addr, breakOut = vm.run()
				if addr != rangeAddress {
					return addr, breakOut
				}
				if breakOut {
					break
				}
			}
			if !breakOut {
				vm.pc = rangeAddress + 1
			}

		// OpRealImag
		case OpRealImag, -OpRealImag:
			v := vm.generalk(a, op < 0)
			var re, im float64
			switch v := v.(type) {
			case complex64:
				re, im = float64(real(v)), float64(imag(v))
			case complex128:
				re, im = real(v), imag(v)
			default:
				rv := reflect.ValueOf(v).Complex()
				re, im = real(rv), imag(rv)
			}
			if b > 0 {
				vm.setFloat(b, re)
			}
			if c > 0 {
				vm.setFloat(c, im)
			}

		// Receive
		case OpReceive:
			ch := vm.general(a)
			switch ch := ch.(type) {
			case chan bool:
				var v bool
				if vm.done == nil {
					v, vm.ok = <-ch
				} else {
					select {
					case v, vm.ok = <-ch:
					case <-vm.done:
						panic(OutOfTimeError{vm.env})
					}
				}
				if c != 0 {
					vm.setBool(c, v)
				}
			case chan int:
				var v int
				if vm.done == nil {
					v, vm.ok = <-ch
				} else {
					select {
					case v, vm.ok = <-ch:
					case <-vm.done:
						panic(OutOfTimeError{vm.env})
					}
				}
				if c != 0 {
					vm.setInt(c, int64(v))
				}
			case chan rune:
				var v rune
				if vm.done == nil {
					v, vm.ok = <-ch
				} else {
					select {
					case v, vm.ok = <-ch:
					case <-vm.done:
						panic(OutOfTimeError{vm.env})
					}
				}
				if c != 0 {
					vm.setInt(c, int64(v))
				}
			case chan string:
				var v string
				if vm.done == nil {
					v, vm.ok = <-ch
				} else {
					select {
					case v, vm.ok = <-ch:
					case <-vm.done:
						panic(OutOfTimeError{vm.env})
					}
				}
				if c != 0 {
					vm.setString(c, v)
				}
			case chan struct{}:
				if vm.done == nil {
					_, vm.ok = <-ch
				} else {
					select {
					case _, vm.ok = <-ch:
					case <-vm.done:
						panic(OutOfTimeError{vm.env})
					}
				}
				if c != 0 {
					vm.setGeneral(c, struct{}{})
				}
			default:
				var v reflect.Value
				if vm.done == nil {
					v, vm.ok = reflect.ValueOf(ch).Recv()
				} else {
					var chosen int
					cas := reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
					vm.cases = append(vm.cases, cas, vm.doneCase)
					chosen, v, vm.ok = reflect.Select(vm.cases)
					vm.cases[0].Chan = reflect.Value{}
					vm.cases[1].Chan = reflect.Value{}
					vm.cases = vm.cases[:0]
					if chosen == 1 {
						panic(OutOfTimeError{vm.env})
					}
				}
				if c != 0 {
					vm.setFromReflectValue(c, v)
				}
			}
			if b != 0 {
				vm.setBool(b, vm.ok)
			}

		// Recover
		case OpRecover:
			var msg interface{}
			last := len(vm.calls) - 1
			if a > 0 {
				// Recover down the stack.
				if vm.calls[last].status == panicked {
					last = -1
				} else {
					last--
				}
			}
			for i := last; i >= 0; i-- {
				switch vm.calls[i].status {
				case deferred:
					continue
				case panicked:
					vm.calls[i].status = recovered
					vm.panic.recovered = true
					msg = vm.panic.message
				}
				break
			}
			if c != 0 {
				vm.setGeneral(c, msg)
			}

		// Rem
		case OpRemInt8, -OpRemInt8:
			vm.setInt(c, int64(int8(vm.int(a))%int8(vm.intk(b, op < 0))))
		case OpRemInt16, -OpRemInt16:
			vm.setInt(c, int64(int16(vm.int(a))%int16(vm.intk(b, op < 0))))
		case OpRemInt32, -OpRemInt32:
			vm.setInt(c, int64(int32(vm.int(a))%int32(vm.intk(b, op < 0))))
		case OpRemInt64:
			vm.setInt(c, vm.int(a)%vm.int(b))
		case -OpRemInt64:
			vm.setInt(c, vm.int(a)%int64(b))
		case OpRemUint8, -OpRemUint8:
			vm.setInt(c, int64(uint8(vm.int(a))%uint8(vm.intk(b, op < 0))))
		case OpRemUint16, -OpRemUint16:
			vm.setInt(c, int64(uint16(vm.int(a))%uint16(vm.intk(b, op < 0))))
		case OpRemUint32, -OpRemUint32:
			vm.setInt(c, int64(uint32(vm.int(a))%uint32(vm.intk(b, op < 0))))
		case OpRemUint64, -OpRemUint64:
			vm.setInt(c, int64(uint64(vm.int(a))%uint64(vm.intk(b, op < 0))))

		// Return
		case OpReturn:
			if vm.env.limitMemory {
				in := vm.fn.Body[0]
				if in.Op == -OpAlloc {
					bytes := decodeUint24(in.A, in.B, in.C)
					in = vm.fn.Body[vm.pc-2]
					if bytes > 0 && in.Op != OpTailCall {
						vm.env.mu.Lock()
						if vm.env.freeMemory >= 0 {
							vm.env.freeMemory -= int(bytes)
						}
						vm.env.mu.Unlock()
					}
				}
			}
			i := len(vm.calls) - 1
			if i == -1 {
				// TODO(marco): call finalizer.
				return maxUint32, false
			}
			call := vm.calls[i]
			if call.status == started {
				// TODO(marco): call finalizer.
				vm.calls = vm.calls[:i]
				vm.fp = call.fp
				vm.fn = call.cl.fn
				vm.vars = call.cl.vars
				vm.pc = call.pc
			} else if !vm.nextCall() {
				return maxUint32, false
			}

		// RightShift
		case OpRightShift, -OpRightShift:
			vm.setInt(c, vm.int(a)>>uint(vm.intk(b, op < 0)))
		case OpRightShiftU, -OpRightShiftU:
			vm.setInt(c, int64(uint64(vm.int(a))>>uint(vm.intk(b, op < 0))))

		// Select
		case OpSelect:
			numCase := len(vm.cases)
			if vm.done != nil && !hasDefaultCase {
				vm.cases = append(vm.cases, vm.doneCase)
			}
			chosen, recv, recvOK := reflect.Select(vm.cases)
			step := numCase - chosen
			var pc Addr
			if step > 0 {
				pc = vm.pc - 2*Addr(step)
				if vm.cases[chosen].Dir == reflect.SelectRecv {
					r := vm.fn.Body[pc-1].B
					if r != 0 {
						vm.setFromReflectValue(r, recv)
					}
					vm.ok = recvOK
				}
				hasDefaultCase = false
			}
			for _, c := range vm.cases {
				if c.Dir != reflect.SelectDefault {
					if c.Dir == reflect.SelectSend {
						zero := reflect.Zero(c.Chan.Type().Elem())
						c.Send.Set(zero)
					}
					c.Chan = reflect.Value{}
				}
			}
			vm.cases = vm.cases[:0]
			if step == 0 {
				panic(OutOfTimeError{vm.env})
			}
			vm.pc = pc

		// Send
		case OpSend, -OpSend:
			k := op < 0
			ch := vm.generalk(c, k)
			switch ch := ch.(type) {
			case chan bool:
				if vm.done == nil {
					ch <- vm.boolk(a, k)
				} else {
					select {
					case ch <- vm.boolk(a, k):
					case <-vm.done:
						panic(OutOfTimeError{vm.env})
					}
				}
			case chan int:
				if vm.done == nil {
					ch <- int(vm.intk(a, k))
				} else {
					select {
					case ch <- int(vm.intk(a, k)):
					case <-vm.done:
						panic(OutOfTimeError{vm.env})
					}
				}
			case chan rune:
				if vm.done == nil {
					ch <- rune(vm.intk(a, k))
				} else {
					select {
					case ch <- rune(vm.intk(a, k)):
					case <-vm.done:
						panic(OutOfTimeError{vm.env})
					}
				}
			case chan string:
				if vm.done == nil {
					ch <- vm.stringk(a, k)
				} else {
					select {
					case ch <- vm.stringk(a, k):
					case <-vm.done:
						panic(OutOfTimeError{vm.env})
					}
				}
			case chan struct{}:
				if vm.done == nil {
					ch <- struct{}{}
				} else {
					select {
					case ch <- struct{}{}:
					case <-vm.done:
						panic(OutOfTimeError{vm.env})
					}
				}
			default:
				r := reflect.ValueOf(ch)
				elemType := r.Type().Elem()
				v := reflect.New(elemType).Elem()
				vm.getIntoReflectValue(a, v, k)
				if vm.done == nil {
					r.Send(v)
				} else {
					cas := reflect.SelectCase{Dir: reflect.SelectSend, Chan: reflect.ValueOf(ch), Send: v}
					vm.cases = append(vm.cases, cas, vm.doneCase)
					chosen, _, _ := reflect.Select(vm.cases)
					vm.cases[0].Send.Set(reflect.Zero(elemType))
					vm.cases[0].Chan = reflect.Value{}
					vm.cases[1].Chan = reflect.Value{}
					vm.cases = vm.cases[:0]
					if chosen == 1 {
						panic(OutOfTimeError{vm.env})
					}
				}
			}

		// SetField
		case OpSetField, -OpSetField:
			i := decodeFieldIndex(vm.fn.Constants.Int[uint8(c)])
			s := reflect.ValueOf(vm.general(b))
			vm.getIntoReflectValue(a, s.Elem().FieldByIndex(i), op < 0)

		// SetMap
		case OpSetMap, -OpSetMap:
			m := vm.general(b)
			switch m := m.(type) {
			case map[string]string:
				k := vm.string(c)
				v := vm.stringk(a, op < 0)
				m[k] = v
			case map[string]int:
				k := vm.string(c)
				v := vm.intk(a, op < 0)
				m[k] = int(v)
			case map[string]bool:
				k := vm.string(c)
				v := vm.boolk(a, op < 0)
				m[k] = v
			case map[string]struct{}:
				m[vm.string(c)] = struct{}{}
			case map[string]interface{}:
				k := vm.string(c)
				v := vm.generalk(a, op < 0)
				m[k] = v
			case map[int]int:
				k := vm.int(c)
				v := vm.intk(a, op < 0)
				m[int(k)] = int(v)
			case map[int]bool:
				k := vm.int(c)
				v := vm.boolk(a, op < 0)
				m[int(k)] = v
			case map[int]string:
				k := vm.int(c)
				v := vm.stringk(a, op < 0)
				m[int(k)] = v
			case map[int]struct{}:
				m[int(vm.int(c))] = struct{}{}
			default:
				mv := reflect.ValueOf(m)
				t := mv.Type()
				k := reflect.New(t.Key()).Elem()
				vm.getIntoReflectValue(c, k, false)
				v := reflect.New(t.Elem()).Elem()
				vm.getIntoReflectValue(a, v, op < 0)
				mv.SetMapIndex(k, v)
			}

		// SetSlice
		case OpSetSlice, -OpSetSlice:
			i := vm.int(c)
			s := vm.general(b)
			switch s := s.(type) {
			case []int:
				s[i] = int(vm.intk(a, op < 0))
			case []rune:
				s[i] = rune(vm.intk(a, op < 0))
			case []byte:
				s[i] = byte(vm.intk(a, op < 0))
			case []float64:
				s[i] = vm.floatk(a, op < 0)
			case []string:
				s[i] = vm.stringk(a, op < 0)
			case []interface{}:
				s[i] = vm.generalk(a, op < 0)
			default:
				v := reflect.ValueOf(s).Index(int(i))
				vm.getIntoReflectValue(a, v, op < 0)
			}

		// SetVar
		case OpSetVar, -OpSetVar:
			v := vm.vars[int(b)<<8|int(uint8(c))]
			switch v := v.(type) {
			case *bool:
				*v = vm.boolk(a, op < 0)
			case *int:
				*v = int(vm.intk(a, op < 0))
			case *float64:
				*v = vm.floatk(a, op < 0)
			case *string:
				*v = vm.stringk(a, op < 0)
			default:
				rv := reflect.ValueOf(v).Elem()
				vm.getIntoReflectValue(a, rv, op < 0)
			}

		// Slice
		case OpSlice:
			var i1, i2, i3 int
			s := reflect.ValueOf(vm.general(a))
			next := vm.fn.Body[vm.pc]
			i1 = int(vm.intk(next.A, b&1 != 0))
			if k := b&2 != 0; k && next.B == -1 {
				i2 = s.Len()
			} else {
				i2 = int(vm.intk(next.B, k))
			}
			if k := b&4 != 0; k && next.C == -1 {
				i3 = s.Cap()
			} else {
				i3 = int(vm.intk(next.C, k))
			}
			s = s.Slice3(i1, i2, i3)
			vm.setGeneral(c, s.Interface())
			vm.pc++
		case OpStringSlice:
			var i1, i2 int
			s := vm.string(a)
			next := vm.fn.Body[vm.pc]
			i1 = int(vm.intk(next.A, b&1 != 0))
			if k := b&2 != 0; k && next.B == -1 {
				i2 = len(s)
			} else {
				i2 = int(vm.intk(next.B, k))
			}
			vm.setString(c, s[i1:i2])
			vm.pc++

		// Sub
		case OpSubInt8, -OpSubInt8:
			vm.setInt(c, int64(int8(vm.int(a)-vm.intk(b, op < 0))))
		case OpSubInt16, -OpSubInt16:
			vm.setInt(c, int64(int16(vm.int(a)-vm.intk(b, op < 0))))
		case OpSubInt32, -OpSubInt32:
			vm.setInt(c, int64(int32(vm.int(a)-vm.intk(b, op < 0))))
		case OpSubFloat32, -OpSubFloat32:
			vm.setFloat(c, float64(float32(vm.float(a))-float32(vm.floatk(b, op < 0))))
		case OpSubInt64:
			vm.setInt(c, vm.int(a)-vm.int(b))
		case -OpSubInt64:
			vm.setInt(c, vm.int(a)-int64(b))
		case OpSubFloat64:
			vm.setFloat(c, vm.float(a)-vm.float(b))
		case -OpSubFloat64:
			vm.setFloat(c, vm.float(a)-float64(b))

		// SubInv
		case OpSubInvInt8, -OpSubInvInt8:
			vm.setInt(c, int64(int8(vm.intk(b, op < 0)-vm.int(a))))
		case OpSubInvInt16, -OpSubInvInt16:
			vm.setInt(c, int64(int16(vm.intk(b, op < 0)-vm.int(a))))
		case OpSubInvInt32, -OpSubInvInt32:
			vm.setInt(c, int64(int32(vm.intk(b, op < 0)-vm.int(a))))
		case OpSubInvInt64, -OpSubInvInt64:
			vm.setInt(c, vm.intk(b, op < 0)-vm.int(a))
		case OpSubInvFloat32, -OpSubInvFloat32:
			vm.setFloat(c, float64(float32(vm.floatk(b, op < 0))-float32(vm.float(a))))
		case OpSubInvFloat64:
			vm.setFloat(c, vm.float(b)-vm.float(a))
		case -OpSubInvFloat64:
			vm.setFloat(c, float64(b)-vm.float(a))

		// TailCall
		case OpTailCall:
			if vm.env.limitMemory {
				in := vm.fn.Body[0]
				if in.Op == -OpAlloc {
					bytes := decodeUint24(in.A, in.B, in.C)
					if bytes > 0 {
						vm.env.mu.Lock()
						if vm.env.freeMemory >= 0 {
							vm.env.freeMemory -= int(bytes)
						}
						vm.env.mu.Unlock()
					}
				}
			}
			vm.calls = append(vm.calls, callFrame{cl: callable{fn: vm.fn, vars: vm.vars}, pc: vm.pc, status: tailed})
			if a != CurrentFunction {
				var fn *Function
				if a == 0 {
					closure := vm.general(b).(*callable)
					fn = closure.fn
					vm.vars = closure.vars
				} else {
					fn = vm.fn.Functions[uint8(b)]
					vm.vars = vm.env.globals
				}
				if vm.fp[0]+Addr(fn.NumReg[0]) > vm.st[0] {
					vm.moreIntStack()
				}
				if vm.fp[1]+Addr(fn.NumReg[1]) > vm.st[1] {
					vm.moreFloatStack()
				}
				if vm.fp[2]+Addr(fn.NumReg[2]) > vm.st[2] {
					vm.moreStringStack()
				}
				if vm.fp[3]+Addr(fn.NumReg[3]) > vm.st[3] {
					vm.moreGeneralStack()
				}
				vm.fn = fn
			}
			vm.pc = 0

		// Typify
		case OpTypify, -OpTypify:
			t := vm.fn.Types[uint8(a)]
			v := reflect.New(t).Elem()
			vm.getIntoReflectValue(b, v, op < 0)
			vm.setGeneral(c, v.Interface())

		// Xor
		case OpXor, -OpXor:
			vm.setInt(c, vm.int(a)^vm.intk(b, op < 0))

		}

	}

}
