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

func (vm *VM) Run(fn *ScrigoFunction) (int, error) {
	var isPanicked bool
	vm.fn = fn
	for {
		isPanicked = vm.runRecoverable()
		if isPanicked && len(vm.calls) > 0 {
			var call = callFrame{fn: callable{scrigo: vm.fn}, fp: vm.fp, status: panicked}
			vm.calls = append(vm.calls, call)
			vm.fn = nil
			if vm.cases != nil {
				vm.cases = vm.cases[:0]
			}
			continue
		}
		break
	}
	if len(vm.panics) > 0 {
		var msg string
		for i, p := range vm.panics {
			if i > 0 {
				msg += "\t"
			}
			msg += fmt.Sprintf("panic %d: %#v", i+1, p.Msg)
			if p.Recovered {
				msg += " [recovered]"
			}
			msg += "\n"
		}
		panic(msg)
	}
	return 0, nil
}

func (vm *VM) runRecoverable() (panicked bool) {
	panicked = true
	defer func() {
		if panicked {
			msg := recover()
			vm.panics = append(vm.panics, Panic{Msg: msg})
		}
	}()
	if vm.fn != nil || vm.nextCall() {
		vm.run()
	}
	return false
}

func (vm *VM) run() int {

	var startNativeGoroutine bool

	var op Operation
	var a, b, c int8

	for {

		in := vm.fn.Body[vm.pc]

		if DebugTraceExecution {
			funcName := vm.fn.Name
			if funcName != "" {
				funcName += ":"
			}
			_, _ = fmt.Fprintf(os.Stderr, "i%v f%v\t%s\t",
				vm.regs.int[vm.fp[0]+1:vm.fp[0]+uint32(vm.fn.RegNum[0])+1],
				vm.regs.float[vm.fp[1]+1:vm.fp[1]+uint32(vm.fn.RegNum[1])+1],
				funcName)
			_, _ = DisassembleInstruction(os.Stderr, vm.fn, vm.pc)
			println()
		}

		vm.pc++
		op, a, b, c = in.Op, in.A, in.B, in.C

		switch op {

		// Add
		case OpAddInt:
			vm.setInt(c, vm.int(a)+vm.int(b))
		case -OpAddInt:
			vm.setInt(c, vm.int(a)+int64(b))
		case OpAddInt8, -OpAddInt8:
			vm.setInt(c, int64(int8(vm.int(a)+vm.intk(b, op < 0))))
		case OpAddInt16, -OpAddInt16:
			vm.setInt(c, int64(int16(vm.int(a)+vm.intk(b, op < 0))))
		case OpAddInt32, -OpAddInt32:
			vm.setInt(c, int64(int32(vm.int(a)+vm.intk(b, op < 0))))
		case OpAddFloat32, -OpAddFloat32:
			vm.setFloat(c, float64(float32(vm.float(a)+vm.floatk(b, op < 0))))
		case OpAddFloat64:
			vm.setFloat(c, vm.float(a)+vm.float(b))
		case -OpAddFloat64:
			vm.setFloat(c, vm.float(a)+float64(b))

		// And
		case OpAnd:
			vm.setInt(c, vm.int(a)&vm.intk(b, op < 0))

		// AndNot
		case OpAndNot:
			vm.setInt(c, vm.int(a)&^vm.intk(b, op < 0))

		// Append
		case OpAppend:
			vm.setGeneral(c, vm.appendSlice(a, int(b), vm.general(c)))

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
			if t.Kind() == reflect.Interface {
				ok = v.Type().Implements(t)
			} else {
				ok = v.Type() == t
			}
			vm.ok = ok
			if ok {
				vm.pc++
			}
			if c != 0 {
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
			}
		case OpAssertInt:
			i, ok := vm.general(a).(int)
			if c != 0 {
				vm.setInt(c, int64(i))
			}
			vm.ok = ok
		case OpAssertFloat64:
			f, ok := vm.general(a).(float64)
			if c != 0 {
				vm.setFloat(c, f)
			}
			vm.ok = ok
		case OpAssertString:
			s, ok := vm.general(a).(string)
			if c != 0 {
				vm.setString(c, s)
			}
			vm.ok = ok

		// Bind
		case OpBind:
			vm.setGeneral(c, vm.cvars[uint8(b)])

		// Call
		case OpCall:
			fn := vm.fn.ScrigoFunctions[uint8(a)]
			off := vm.fn.Body[vm.pc]
			call := callFrame{fn: callable{scrigo: vm.fn, vars: vm.cvars}, fp: vm.fp, pc: vm.pc + 1}
			vm.fp[0] += uint32(off.Op)
			if vm.fp[0]+uint32(fn.RegNum[0]) > vm.st[0] {
				vm.moreIntStack()
			}
			vm.fp[1] += uint32(off.A)
			if vm.fp[1]+uint32(fn.RegNum[1]) > vm.st[1] {
				vm.moreFloatStack()
			}
			vm.fp[2] += uint32(off.B)
			if vm.fp[2]+uint32(fn.RegNum[2]) > vm.st[2] {
				vm.moreStringStack()
			}
			vm.fp[3] += uint32(off.C)
			if vm.fp[3]+uint32(fn.RegNum[3]) > vm.st[3] {
				vm.moreGeneralStack()
			}
			vm.fn = fn
			vm.cvars = nil
			vm.calls = append(vm.calls, call)
			vm.pc = 0

		// CallIndirect
		case OpCallIndirect:
			f := vm.general(a).(*callable)
			if f.scrigo == nil {
				off := vm.fn.Body[vm.pc]
				vm.callNative(f.native, c, StackShift{int8(off.Op), off.A, off.B, off.C}, startNativeGoroutine)
				startNativeGoroutine = false
			} else {
				fn := f.scrigo
				vm.cvars = f.vars
				off := vm.fn.Body[vm.pc]
				call := callFrame{fn: callable{scrigo: vm.fn, vars: vm.cvars}, fp: vm.fp, pc: vm.pc + 1}
				vm.fp[0] += uint32(off.Op)
				if vm.fp[0]+uint32(fn.RegNum[0]) > vm.st[0] {
					vm.moreIntStack()
				}
				vm.fp[1] += uint32(off.A)
				if vm.fp[1]+uint32(fn.RegNum[1]) > vm.st[1] {
					vm.moreFloatStack()
				}
				vm.fp[2] += uint32(off.B)
				if vm.fp[2]+uint32(fn.RegNum[2]) > vm.st[2] {
					vm.moreStringStack()
				}
				vm.fp[3] += uint32(off.C)
				if vm.fp[3]+uint32(fn.RegNum[3]) > vm.st[3] {
					vm.moreGeneralStack()
				}
				vm.fn = fn
				vm.calls = append(vm.calls, call)
				vm.pc = 0
			}

		// CallNative
		case OpCallNative:
			fn := vm.fn.NativeFunctions[uint8(a)]
			off := vm.fn.Body[vm.pc]
			vm.callNative(fn, c, StackShift{int8(off.Op), off.A, off.B, off.C}, startNativeGoroutine)
			startNativeGoroutine = false

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
			if dir != reflect.SelectDefault {
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

		// Continue
		case OpContinue:
			return int(a)

		// Convert
		case OpConvert:
			t := vm.fn.Types[uint8(b)]
			switch t.Kind() {
			case reflect.Array:
				slice := reflect.ValueOf(vm.general(a))
				array := reflect.New(t)
				for i := 0; i < slice.Len(); i++ {
					array.Elem().Index(i).Set(slice.Index(i))
				}
				vm.setGeneral(c, array.Elem().Interface())
			case reflect.Func:
				call := vm.general(a).(*callable)
				vm.setGeneral(c, call.reflectValue())
			default:
				vm.setGeneral(c, reflect.ValueOf(vm.general(a)).Convert(t).Interface())
			}
		case OpConvertInt:
			t := vm.fn.Types[uint8(b)]
			if t.Kind() == reflect.Bool {
				vm.setGeneral(c, reflect.ValueOf(vm.bool(a)).Convert(t).Interface())
			} else {
				vm.setGeneral(c, reflect.ValueOf(vm.int(a)).Convert(t).Interface())
			}
		case OpConvertUint:
			t := vm.fn.Types[uint8(b)]
			vm.setGeneral(c, reflect.ValueOf(uint64(vm.int(a))).Convert(t).Interface())
		case OpConvertFloat:
			t := vm.fn.Types[uint8(b)]
			vm.setGeneral(c, reflect.ValueOf(vm.float(a)).Convert(t).Interface())
		case OpConvertString:
			t := vm.fn.Types[uint8(b)]
			vm.setGeneral(c, reflect.ValueOf(vm.string(a)).Convert(t).Interface())

		// Copy
		case OpCopy:
			src := reflect.ValueOf(vm.general(a))
			dst := reflect.ValueOf(vm.general(c))
			n := reflect.Copy(dst, src)
			if b != 0 {
				vm.setInt(b, int64(n))
			}

		// Concat
		case OpConcat:
			vm.setString(c, vm.string(a)+vm.string(b))

		// Defer
		case OpDefer:
			off := vm.fn.Body[vm.pc]
			arg := vm.fn.Body[vm.pc+1]
			vm.deferCall(vm.general(a).(*callable), c,
				StackShift{int8(off.Op), off.A, off.B, off.C},
				StackShift{int8(arg.Op), arg.A, arg.B, arg.C})
			vm.pc += 2

		// Delete
		case OpDelete:
			m := reflect.ValueOf(vm.general(a))
			k := reflect.ValueOf(vm.general(b))
			m.SetMapIndex(k, reflect.Value{})

		// Div
		case OpDivInt:
			vm.setInt(c, vm.int(a)/vm.int(b))
		case -OpDivInt:
			vm.setInt(c, vm.int(a)/int64(b))
		case OpDivInt8, -OpDivInt8:
			vm.setInt(c, int64(int8(vm.int(a))/int8(vm.intk(b, op < 0))))
		case OpDivInt16, -OpDivInt16:
			vm.setInt(c, int64(int16(vm.int(a))/int16(vm.intk(b, op < 0))))
		case OpDivInt32, -OpDivInt32:
			vm.setInt(c, int64(int32(vm.int(a))/int32(vm.intk(b, op < 0))))
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

		// Func
		case OpFunc:
			fn := vm.fn.Literals[uint8(b)]
			var vars []interface{}
			if fn.CRefs != nil {
				vars = make([]interface{}, len(fn.CRefs))
				for i, ref := range fn.CRefs {
					if ref < 0 {
						vars[i] = vm.general(int8(-ref))
					} else {
						vars[i] = vm.cvars[ref]
					}
				}
			}
			vm.setGeneral(c, &callable{scrigo: fn, vars: vars})

		// GetFunc
		case OpGetFunc:
			fn := callable{}
			if a == 0 {
				fn.scrigo = vm.fn.ScrigoFunctions[uint8(b)]
			} else {
				fn.native = vm.fn.NativeFunctions[uint8(b)]
			}
			vm.setGeneral(c, &fn)

		// GetVar
		case OpGetVar:
			v := vm.fn.Variables[uint8(a)].Value
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

		// Go
		case OpGo:
			wasNative := vm.startScrigoGoroutine()
			if wasNative {
				startNativeGoroutine = true
			}

		// Goto
		case OpGoto:
			vm.pc = DecodeAddr(a, b, c)

		// If
		case OpIf:
			var cond bool
			switch Condition(b) {
			case ConditionOK, ConditionNotOK:
				cond = vm.ok
			case ConditionNil, ConditionNotNil:
				cond = vm.general(a) == nil
			case ConditionEqual, ConditionNotEqual:
				cond = vm.general(a) == vm.generalk(c, op < 0)
			}
			switch Condition(b) {
			case ConditionNotOK, ConditionNotNil, ConditionNotEqual:
				cond = !cond
			}
			if cond {
				vm.pc++
			}
		case OpIfInt, -OpIfInt:
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
		case OpIfUint, -OpIfUint:
			var cond bool
			v1 := uint64(vm.int(a))
			v2 := uint64(vm.intk(c, op < 0))
			switch Condition(b) {
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
			i := int(vm.intk(b, op < 0))
			v := reflect.ValueOf(vm.general(a)).Index(i)
			vm.setFromReflectValue(c, v)

		// LeftShift
		case OpLeftShift, -OpLeftShift:
			vm.setInt(c, vm.int(a)<<uint(vm.intk(b, op < 0)))
		case OpLeftShift8, -OpLeftShift8:
			vm.setInt(c, int64(int8(vm.int(a))<<uint(vm.intk(b, op < 0))))
		case OpLeftShift16, -OpLeftShift16:
			vm.setInt(c, int64(int16(vm.int(a))<<uint(vm.intk(b, op < 0))))
		case OpLeftShift32, -OpLeftShift32:
			vm.setInt(c, int64(int32(vm.int(a))<<uint(vm.intk(b, op < 0))))

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

		// MakeChan
		case OpMakeChan, -OpMakeChan:
			typ := vm.fn.Types[uint8(a)]
			buffer := int(vm.intk(b, op < 0))
			vm.setGeneral(c, reflect.MakeChan(typ, buffer).Interface())

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
				elem := mv.MapIndex(k)
				vm.ok = elem.IsValid()
				if !vm.ok {
					elem = reflect.Zero(t.Elem())
				}
				vm.setFromReflectValue(c, elem)
			}

		// MakeMap
		case OpMakeMap, -OpMakeMap:
			typ := vm.fn.Types[uint8(a)]
			n := int(vm.intk(b, op < 0))
			vm.setGeneral(c, reflect.MakeMapWithSize(typ, n).Interface())

		// MakeSlice
		case OpMakeSlice:
			typ := vm.fn.Types[uint8(a)]
			var len, cap int
			if b > 1 {
				next := vm.fn.Body[vm.pc]
				vm.pc++
				lenIsConst := (b & (1 << 1)) != 0
				len = int(vm.intk(next.A, lenIsConst))
				capIsConst := (b & (1 << 2)) != 0
				cap = int(vm.intk(next.B, capIsConst))
			}
			vm.setGeneral(c, reflect.MakeSlice(typ, len, cap).Interface())

		// Move
		case OpMove, -OpMove:
			switch MoveType(a) {
			case FloatFloat:
				vm.setFloat(c, vm.floatk(b, op < 0))
			case FloatGeneral:
				vm.setGeneral(c, vm.floatk(b, op < 0))
			case GeneralGeneral:
				vm.setGeneral(c, vm.generalk(b, op < 0))
			case IntGeneral:
				vm.setGeneral(c, vm.intk(b, op < 0))
			case IntInt:
				vm.setInt(c, vm.intk(b, op < 0))
			case StringGeneral:
				vm.setGeneral(c, vm.stringk(b, op < 0))
			case StringString:
				vm.setString(c, vm.stringk(b, op < 0))
			}

		// LoadNumber.
		case OpLoadNumber:
			switch a {
			case 0:
				vm.setInt(c, vm.fn.Constants.Int[uint8(b)])
			case 1:
				vm.setFloat(c, vm.fn.Constants.Float[uint8(b)])
			}

		// Mul
		case OpMulInt:
			vm.setInt(c, vm.int(a)*vm.int(b))
		case -OpMulInt:
			vm.setInt(c, vm.int(a)*int64(b))
		case OpMulInt8, -OpMulInt8:
			vm.setInt(c, int64(int8(vm.int(a)*vm.intk(b, op < 0))))
		case OpMulInt16, -OpMulInt16:
			vm.setInt(c, int64(int16(vm.int(a)*vm.intk(b, op < 0))))
		case OpMulInt32, -OpMulInt32:
			vm.setInt(c, int64(int32(vm.int(a)*vm.intk(b, op < 0))))
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
			// TODO(Gianluca): if argument is 0, check if previous
			// instruction is Assert; in such case, raise a type-assertion
			// panic, retrieving informations from its operands.
			panic(vm.general(a))

		// Print
		case OpPrint:
			print(string(sprint(vm.general(a))))

		// Range
		case OpRange:
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
		case OpRangeString, -OpRangeString:
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
		case OpReceive:
			ch := vm.general(a)
			switch ch := ch.(type) {
			case chan bool:
				var v bool
				v, vm.ok = <-ch
				if c != 0 {
					vm.setBool(c, v)
				}
			case chan int:
				var v int
				v, vm.ok = <-ch
				if c != 0 {
					vm.setInt(c, int64(v))
				}
			case chan rune:
				var v rune
				v, vm.ok = <-ch
				if c != 0 {
					vm.setInt(c, int64(v))
				}
			case chan string:
				var v string
				v, vm.ok = <-ch
				if c != 0 {
					vm.setString(c, v)
				}
			case chan struct{}:
				_, vm.ok = <-ch
				if c != 0 {
					vm.setGeneral(c, struct{}{})
				}
			default:
				var v reflect.Value
				v, vm.ok = reflect.ValueOf(ch).Recv()
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
			for i := len(vm.calls) - 1; i >= 0; i-- {
				switch vm.calls[i].status {
				case deferred:
					continue
				case panicked:
					vm.calls[i].status = recovered
					last := len(vm.panics) - 1
					vm.panics[last].Recovered = true
					msg = vm.panics[last].Msg
				}
				break
			}
			if c != 0 {
				vm.setGeneral(c, msg)
			}

		// Rem
		case OpRemInt:
			vm.setInt(c, vm.int(a)%vm.int(b))
		case -OpRemInt:
			vm.setInt(c, vm.int(a)%int64(b))
		case OpRemInt8, -OpRemInt8:
			vm.setInt(c, int64(int8(vm.int(a))%int8(vm.intk(b, op < 0))))
		case OpRemInt16, -OpRemInt16:
			vm.setInt(c, int64(int16(vm.int(a))%int16(vm.intk(b, op < 0))))
		case OpRemInt32, -OpRemInt32:
			vm.setInt(c, int64(int32(vm.int(a))%int32(vm.intk(b, op < 0))))
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
			i := len(vm.calls) - 1
			if i == -1 {
				// TODO(marco): call finalizer.
				return maxInt8
			}
			call := vm.calls[i]
			if call.status == started {
				// TODO(marco): call finalizer.
				vm.calls = vm.calls[:i]
				vm.fp = call.fp
				vm.pc = call.pc
				vm.fn = call.fn.scrigo
				vm.cvars = call.fn.vars
			} else if !vm.nextCall() {
				return maxInt8
			}

		// RightShift
		case OpRightShift, -OpRightShift:
			vm.setInt(c, vm.int(a)>>uint(vm.intk(b, op < 0)))
		case OpRightShiftU, -OpRightShiftU:
			vm.setInt(c, int64(uint64(vm.int(a))>>uint(vm.intk(b, op < 0))))

		// Select
		case OpSelect:
			chosen, recv, recvOK := reflect.Select(vm.cases)
			vm.pc -= 2 * uint32(len(vm.cases)-chosen)
			if vm.cases[chosen].Dir == reflect.SelectRecv {
				r := vm.fn.Body[vm.pc-1].B
				if r != 0 {
					vm.setFromReflectValue(r, recv)
				}
				vm.ok = recvOK
			}
			vm.cases = vm.cases[:0]

		// Selector
		case OpSelector:
			v := reflect.ValueOf(vm.general(a)).Field(int(uint8(b)))
			vm.setFromReflectValue(c, v)

		// Send
		case OpSend, -OpSend:
			k := op < 0
			ch := vm.generalk(c, k)
			switch ch := ch.(type) {
			case chan bool:
				ch <- vm.boolk(a, k)
			case chan int:
				ch <- int(vm.intk(a, k))
			case chan rune:
				ch <- rune(vm.intk(a, k))
			case chan string:
				ch <- vm.stringk(a, k)
			case chan struct{}:
				ch <- struct{}{}
			default:
				r := reflect.ValueOf(ch)
				elemType := r.Type().Elem()
				v := reflect.New(elemType).Elem()
				vm.getIntoReflectValue(a, v, k)
				r.Send(v)
			}

		// SetMap
		case OpSetMap, -OpSetMap:
			m := vm.general(a)
			switch m := m.(type) {
			case map[string]string:
				k := vm.string(c)
				v := vm.stringk(b, op < 0)
				m[k] = v
			case map[string]int:
				k := vm.string(c)
				v := vm.intk(b, op < 0)
				m[k] = int(v)
			case map[string]bool:
				k := vm.string(c)
				v := vm.boolk(b, op < 0)
				m[k] = v
			case map[string]struct{}:
				m[vm.string(c)] = struct{}{}
			case map[string]interface{}:
				k := vm.string(c)
				v := vm.generalk(b, op < 0)
				m[k] = v
			case map[int]int:
				k := vm.int(c)
				v := vm.intk(b, op < 0)
				m[int(k)] = int(v)
			case map[int]bool:
				k := vm.int(c)
				v := vm.boolk(b, op < 0)
				m[int(k)] = v
			case map[int]string:
				k := vm.int(c)
				v := vm.stringk(b, op < 0)
				m[int(k)] = v
			case map[int]struct{}:
				m[int(vm.int(c))] = struct{}{}
			default:
				mv := reflect.ValueOf(m)
				t := mv.Type()
				k := reflect.New(t.Key()).Elem()
				vm.getIntoReflectValue(c, k, false)
				v := reflect.New(t.Elem()).Elem()
				vm.getIntoReflectValue(b, v, op < 0)
				mv.SetMapIndex(k, v)
			}

		// SetSlice
		case OpSetSlice, -OpSetSlice:
			i := vm.int(c)
			s := vm.general(a)
			switch s := s.(type) {
			case []int:
				s[i] = int(vm.intk(b, op < 0))
			case []rune:
				s[i] = rune(vm.intk(b, op < 0))
			case []byte:
				s[i] = byte(vm.intk(b, op < 0))
			case []float64:
				s[i] = vm.floatk(b, op < 0)
			case []string:
				s[i] = vm.stringk(b, op < 0)
			case []interface{}:
				s[i] = vm.generalk(b, op < 0)
			default:
				v := reflect.ValueOf(s).Index(int(i))
				vm.getIntoReflectValue(b, v, op < 0)
			}

		// SetVar
		case OpSetVar, -OpSetVar:
			v := vm.fn.Variables[uint8(c)].Value
			switch v := v.(type) {
			case *bool:
				*v = vm.boolk(b, op < 0)
			case *int:
				*v = int(vm.intk(b, op < 0))
			case *float64:
				*v = vm.floatk(b, op < 0)
			case *string:
				*v = vm.stringk(b, op < 0)
			default:
				rv := reflect.ValueOf(v).Elem()
				vm.getIntoReflectValue(b, rv, op < 0)
			}

		// SliceIndex
		case OpSliceIndex, -OpSliceIndex:
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
				vm.setFloat(c, v[i])
			case []string:
				vm.setString(c, v[i])
			case []interface{}:
				vm.setGeneral(c, v[i])
			default:
				vm.setGeneral(c, reflect.ValueOf(v).Index(i).Interface())
			}

		// StringIndex
		case OpStringIndex, -OpStringIndex:
			vm.setInt(c, int64(vm.string(a)[int(vm.intk(b, op < 0))]))

		// Sub
		case OpSubInt:
			vm.setInt(c, vm.int(a)-vm.int(b))
		case -OpSubInt:
			vm.setInt(c, vm.int(a)-int64(b))
		case OpSubInt8, -OpSubInt8:
			vm.setInt(c, int64(int8(vm.int(a)-vm.intk(b, op < 0))))
		case OpSubInt16, -OpSubInt16:
			vm.setInt(c, int64(int16(vm.int(a)-vm.intk(b, op < 0))))
		case OpSubInt32, -OpSubInt32:
			vm.setInt(c, int64(int32(vm.int(a)-vm.intk(b, op < 0))))
		case OpSubFloat32, -OpSubFloat32:
			vm.setFloat(c, float64(float32(float32(vm.float(a))-float32(vm.floatk(b, op < 0)))))
		case OpSubFloat64:
			vm.setFloat(c, vm.float(a)-vm.float(b))
		case -OpSubFloat64:
			vm.setFloat(c, vm.float(a)-float64(b))

		// SubInv
		case OpSubInvInt, -OpSubInvInt:
			vm.setInt(c, vm.intk(b, op < 0)-vm.int(a))
		case OpSubInvInt8, -OpSubInvInt8:
			vm.setInt(c, int64(int8(vm.intk(b, op < 0)-vm.int(a))))
		case OpSubInvInt16, -OpSubInvInt16:
			vm.setInt(c, int64(int16(vm.intk(b, op < 0)-vm.int(a))))
		case OpSubInvInt32, -OpSubInvInt32:
			vm.setInt(c, int64(int32(vm.intk(b, op < 0)-vm.int(a))))
		case OpSubInvFloat32, -OpSubInvFloat32:
			vm.setFloat(c, float64(float32(float32(vm.floatk(b, op < 0))-float32(vm.float(a)))))
		case OpSubInvFloat64:
			vm.setFloat(c, vm.float(b)-vm.float(a))
		case -OpSubInvFloat64:
			vm.setFloat(c, float64(b)-vm.float(a))

		// TailCall
		case OpTailCall:
			vm.calls = append(vm.calls, callFrame{fn: callable{scrigo: vm.fn, vars: vm.cvars}, pc: vm.pc, status: tailed})
			vm.pc = 0
			if a != CurrentFunction {
				var fn *ScrigoFunction
				if a == 0 {
					closure := vm.general(b).(*callable)
					fn = closure.scrigo
					vm.cvars = closure.vars
				} else {
					fn = vm.fn.ScrigoFunctions[uint8(b)]
					vm.cvars = nil
				}
				if vm.fp[0]+uint32(fn.RegNum[0]) > vm.st[0] {
					vm.moreIntStack()
				}
				if vm.fp[1]+uint32(fn.RegNum[1]) > vm.st[1] {
					vm.moreFloatStack()
				}
				if vm.fp[2]+uint32(fn.RegNum[2]) > vm.st[2] {
					vm.moreStringStack()
				}
				if vm.fp[3]+uint32(fn.RegNum[3]) > vm.st[3] {
					vm.moreGeneralStack()
				}
				vm.fn = fn
			}

		// Xor
		case OpXor, -OpXor:
			vm.setInt(c, vm.int(a)^vm.intk(b, op < 0))

		}

	}

}
