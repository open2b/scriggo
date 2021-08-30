// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"bytes"
	"reflect"
	"strings"
	"unicode"

	"github.com/open2b/scriggo/ast"
)

func (vm *VM) runFunc(fn *Function, vars []reflect.Value) error {
	vm.fn = fn
	vm.vars = vars
	for {
		err := vm.runRecoverable()
		if err == nil {
			break
		}
		p, ok := err.(*PanicError)
		if !ok {
			return err
		}
		p.next = vm.panic
		vm.panic = p
		if len(vm.calls) == 0 {
			break
		}
		vm.calls = append(vm.calls, callFrame{cl: callable{fn: vm.fn}, renderer: vm.renderer, fp: vm.fp, status: panicked})
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

	var startNativeGoroutine bool

	var op Operation
	var a, b, c int8

	for {

		in := vm.fn.Body[vm.pc]

		vm.pc++
		op, a, b, c = in.Op, in.A, in.B, in.C

		// If an instruction needs to change the program counter,
		// it must be changed, if possible, at the end of the instruction execution.

		switch op {

		// Add
		case OpAdd, -OpAdd:
			switch a := reflect.Kind(a); a {
			case reflect.Float64:
				vm.setFloat(c, vm.floatk(b, op < 0)+vm.float(c))
			case reflect.Float32:
				vm.setFloat(c, float64(float32(vm.floatk(b, op < 0)+vm.float(c))))
			default:
				v := vm.intk(b, op < 0) + vm.int(c)
				switch a {
				case reflect.Int8:
					v = int64(int8(v))
				case reflect.Int16:
					v = int64(int16(v))
				case reflect.Int32:
					v = int64(int32(v))
				case reflect.Uint8:
					v = int64(uint8(v))
				case reflect.Uint16:
					v = int64(uint16(v))
				case reflect.Uint32:
					v = int64(uint32(v))
				case reflect.Uint64:
					v = int64(uint64(v))
				}
				vm.setInt(c, v)
			}
		case OpAddInt, -OpAddInt:
			vm.setInt(c, vm.int(a)+vm.intk(b, op < 0))
		case OpAddFloat64, -OpAddFloat64:
			vm.setFloat(c, vm.float(a)+vm.floatk(b, op < 0))

		// Addr
		case OpAddr:
			v := vm.general(a)
			switch v.Kind() {
			case reflect.Slice, reflect.Array:
				i := int(vm.int(b))
				vm.setGeneral(c, v.Index(i).Addr())
			case reflect.Struct, reflect.Ptr:
				vm.setGeneral(c, vm.fieldByIndex(v, uint8(b)).Addr())
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
			vm.setGeneral(c, reflect.AppendSlice(vm.general(c), vm.general(a)))

		// Assert
		case OpAssert:
			v := vm.general(a)
			t := vm.fn.Types[uint8(b)]
			var ok bool
			if v.IsValid() {
				if w, isScriggoType := t.(ScriggoType); isScriggoType {
					v, ok = w.Unwrap(v)
				} else {
					if t.Kind() == reflect.Interface {
						ok = v.Type().Implements(t)
					} else {
						ok = v.Type() == t
					}
				}
			}
			vm.ok = ok
			if !ok {
				in := vm.fn.Body[vm.pc]
				if in.Op == OpPanic {
					var concrete reflect.Type
					var method string
					if v.IsValid() {
						concrete = v.Type()
						if t.Kind() == reflect.Interface {
							method = missingMethod(concrete, t)
						}
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
					if w, ok := t.(ScriggoType); ok {
						t = w.GoType()
					}
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
		case OpCallFunc:
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
		case OpCallIndirect:
			f := vm.general(a).Interface().(*callable)
			if f.fn == nil {
				off := vm.fn.Body[vm.pc]
				vm.callNative(f.Native(), c, StackShift{int8(off.Op), off.A, off.B, off.C}, startNativeGoroutine)
				startNativeGoroutine = false
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
				if fn.Macro {
					call.renderer = vm.renderer
					if b == ReturnString {
						vm.renderer = vm.renderer.WithOut(&macroOutBuffer{})
					} else if ast.Format(b) != fn.Format {
						vm.renderer = vm.renderer.WithConversion(fn.Format, ast.Format(b))
					}
				}
				vm.fn = fn
				vm.vars = f.vars
				vm.calls = append(vm.calls, call)
				vm.pc = 0
			}
		case OpCallMacro:
			call := callFrame{cl: callable{fn: vm.fn, vars: vm.vars}, renderer: vm.renderer, fp: vm.fp, pc: vm.pc + 1}
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
			if b == ReturnString {
				vm.renderer = vm.renderer.WithOut(&macroOutBuffer{})
			} else if ast.Format(b) != fn.Format {
				vm.renderer = vm.renderer.WithConversion(fn.Format, ast.Format(b))
			}
			vm.fn = fn
			vm.vars = vm.env.globals
			vm.calls = append(vm.calls, call)
			vm.pc = 0
		case OpCallNative:
			fn := vm.fn.NativeFunctions[uint8(a)]
			off := vm.fn.Body[vm.pc]
			vm.callNative(fn, c, StackShift{int8(off.Op), off.A, off.B, off.C}, startNativeGoroutine)
			startNativeGoroutine = false
			vm.pc++

		// Cap
		case OpCap:
			vm.setInt(c, int64(vm.general(a).Cap()))

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
			switch dir {
			case reflect.SelectSend:
				vm.cases[i].Chan = vm.general(c)
				t := vm.cases[i].Chan.Type().Elem()
				if !vm.cases[i].Send.IsValid() || t != vm.cases[i].Send.Type() {
					vm.cases[i].Send = reflect.New(t).Elem()
				}
				vm.getIntoReflectValue(b, vm.cases[i].Send, op < 0)
			case reflect.SelectRecv:
				vm.cases[i].Chan = vm.general(b)
			}
			vm.pc++

		// Close
		case OpClose:
			vm.general(a).Close()

		// Complex
		case OpComplex64:
			var re, im float32
			if a > 0 {
				re = float32(vm.float(a))
			}
			if b > 0 {
				im = float32(vm.float(b))
			}
			vm.setGeneral(c, reflect.ValueOf(complex(re, im)))
		case OpComplex128:
			var re, im float64
			if a > 0 {
				re = vm.float(a)
			}
			if b > 0 {
				im = vm.float(b)
			}
			vm.setGeneral(c, reflect.ValueOf(complex(re, im)))

		// Continue
		case OpContinue:
			return Addr(decodeUint24(a, b, c)), false

		// Convert
		case OpConvert:
			t := vm.fn.Types[uint8(b)]
			switch t.Kind() {
			case reflect.String:
				vm.setString(c, vm.general(a).Convert(t).String())
			default:
				vm.setGeneral(c, vm.general(a).Convert(t))
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
				s := string(unicode.ReplacementChar)
				if 0 <= v && v <= unicode.MaxRune {
					s = string(rune(v))
				}
				vm.setString(c, s)
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
				s := string(unicode.ReplacementChar)
				if v <= unicode.MaxRune {
					s = string(rune(v))
				}
				vm.setString(c, s)
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
			v := reflect.ValueOf(vm.string(a))
			if t.Kind() == reflect.Slice {
				vm.setGeneral(c, v.Convert(t))
			} else {
				var b bytes.Buffer
				r1 := vm.renderer.WithOut(&b)
				r2 := r1.WithConversion(ast.FormatMarkdown, ast.FormatHTML)
				_, _ = r2.Out().Write([]byte(v.String()))
				_ = r2.Close()
				_ = r1.Close()
				vm.setString(c, b.String())
			}

		// Concat
		case OpConcat:
			vm.setString(c, vm.string(a)+vm.string(b))

		// Copy
		case OpCopy:
			n := reflect.Copy(vm.general(c), vm.general(a))
			if b != 0 {
				vm.setInt(b, int64(n))
			}

		// Defer
		case OpDefer:
			cl := vm.general(a).Interface().(*callable)
			off := vm.fn.Body[vm.pc]
			arg := vm.fn.Body[vm.pc+1]
			fp := [4]Addr{
				vm.fp[0] + Addr(off.Op),
				vm.fp[1] + Addr(off.A),
				vm.fp[2] + Addr(off.B),
				vm.fp[3] + Addr(off.C),
			}
			vm.swapStack(&vm.fp, &fp, StackShift{int8(arg.Op), arg.A, arg.B, arg.C})
			vm.calls = append(vm.calls, callFrame{cl: *cl, renderer: vm.renderer, fp: fp, pc: 0, status: deferred, numVariadic: c})
			vm.pc += 2

		// Delete
		case OpDelete:
			vm.general(a).SetMapIndex(vm.general(b), reflect.Value{})

		// Div
		case OpDiv, -OpDiv:
			switch a := reflect.Kind(a); a {
			case reflect.Float32:
				vm.setFloat(c, float64(float32(vm.float(c))/float32(vm.floatk(b, op < 0))))
			case reflect.Float64:
				vm.setFloat(c, vm.float(c)/vm.floatk(b, op < 0))
			default:
				var bv = vm.intk(b, op < 0)
				var cv = vm.int(c)
				var v int64
				switch a {
				case reflect.Int8:
					v = int64(int8(cv) / int8(bv))
				case reflect.Int16:
					v = int64(int16(cv) / int16(bv))
				case reflect.Int32:
					v = int64(int32(cv) / int32(bv))
				case reflect.Int64:
					v = cv / bv
				case reflect.Uint8:
					v = int64(uint8(cv) / uint8(bv))
				case reflect.Uint16:
					v = int64(uint16(cv) / uint16(bv))
				case reflect.Uint32:
					v = int64(uint32(cv) / uint32(bv))
				case reflect.Uint64:
					v = int64(uint64(cv) / uint64(bv))
				}
				vm.setInt(c, v)
			}
		case OpDivInt, -OpDivInt:
			vm.setInt(c, vm.int(a)/vm.intk(b, op < 0))
		case OpDivFloat64, -OpDivFloat64:
			vm.setFloat(c, vm.float(a)/vm.floatk(b, op < 0))

		// Field
		case OpField:
			v := vm.general(a)
			vm.setFromReflectValue(c, vm.fieldByIndex(v, uint8(b)))

		// GetVar
		case OpGetVar:
			v := vm.vars[decodeInt16(a, b)]
			vm.setFromReflectValue(c, v)

		// GetVarAddr
		case OpGetVarAddr:
			ptr := vm.vars[decodeInt16(a, b)].Addr()
			vm.setFromReflectValue(c, ptr)

		// Go
		case OpGo:
			wasNative := vm.startGoroutine()
			if wasNative {
				startNativeGoroutine = true
			}

		// Goto
		case OpGoto:
			vm.pc = Addr(decodeUint24(a, b, c))

		// If
		case OpIf, -OpIf:
			var cond bool
			switch Condition(b) {
			case ConditionOK, ConditionNotOK:
				cond = vm.ok
			case ConditionInterfaceNil, ConditionInterfaceNotNil:
				// An invalid reflect.Value represents a nil interface value.
				cond = !vm.general(a).IsValid()
			case ConditionNil, ConditionNotNil:
				cond = vm.general(a).IsNil()
			case ConditionEqual, ConditionNotEqual:
				x := vm.general(a)
				y := vm.generalk(c, op < 0)
				cond = x.Interface() == y.Interface()
			case ConditionInterfaceEqual, ConditionInterfaceNotEqual:
				x := vm.general(a)
				y := vm.generalk(c, op < 0)
				cond = vm.equals(x, y)
			case ConditionContainsElement, ConditionNotContainsElement:
				x := vm.general(a)
				y := vm.generalk(c, op < 0)
				n := x.Len()
				k := x.Type().Elem().Kind()
				if k == reflect.Interface {
					for i := 0; i < n; i++ {
						if vm.equals(x.Index(i).Elem(), y) {
							cond = true
							break
						}
					}
				} else {
					e := y.Interface()
					for i := 0; i < n; i++ {
						if x.Index(i).Interface() == e {
							cond = true
							break
						}
					}
				}
			case ConditionContainsKey, ConditionNotContainsKey:
				x := vm.general(a)
				y := vm.generalk(c, op < 0)
				cond = x.MapIndex(y).IsValid()
			case ConditionContainsNil, ConditionNotContainsNil:
				x := vm.general(a)
				if x.Kind() == reflect.Map {
					zero := reflect.Zero(x.Type().Key())
					cond = x.MapIndex(zero).IsValid()
				} else {
					n := x.Len()
					for i := 0; i < n; i++ {
						if x.Index(i).IsZero() {
							cond = true
							break
						}
					}
				}
			}
			switch Condition(b) {
			case ConditionNotOK, ConditionInterfaceNotNil, ConditionNotNil,
				ConditionNotEqual, ConditionInterfaceNotEqual, ConditionNotContainsSubstring,
				ConditionNotContainsRune, ConditionNotContainsKey, ConditionNotContainsElement,
				ConditionNotContainsNil:
				cond = !cond
			}
			if cond {
				vm.pc++
			}
		case OpIfInt, -OpIfInt:
			var cond bool
			switch Condition(b) {
			case ConditionZero:
				cond = vm.int(a) == 0
			case ConditionNotZero:
				cond = vm.int(a) != 0
			case ConditionLessU, ConditionLessEqualU, ConditionGreaterU, ConditionGreaterEqualU:
				v1 := uint64(vm.int(a))
				v2 := uint64(vm.intk(c, op < 0))
				switch Condition(b) {
				case ConditionLessU:
					cond = v1 < v2
				case ConditionLessEqualU:
					cond = v1 <= v2
				case ConditionGreaterU:
					cond = v1 > v2
				case ConditionGreaterEqualU:
					cond = v1 >= v2
				}
			case ConditionContainsElement, ConditionNotContainsElement:
				x := vm.general(a)
				if s, ok := x.Interface().([]int); ok {
					y := int(vm.intk(c, op < 0))
					for _, e := range s {
						if e == y {
							cond = true
							break
						}
					}
				} else {
					n := x.Len()
					k := x.Type().Elem().Kind()
					switch {
					case k == reflect.Bool:
						y := vm.boolk(c, op < 0)
						for i := 0; i < n; i++ {
							if cond = x.Index(i).Bool() == y; cond {
								break
							}
						}
					case reflect.Int <= k && k <= reflect.Int64:
						y := vm.intk(c, op < 0)
						for i := 0; i < n; i++ {
							if cond = x.Index(i).Int() == y; cond {
								break
							}
						}
					default:
						y := uint64(vm.intk(c, op < 0))
						for i := 0; i < n; i++ {
							if cond = x.Index(i).Uint() == y; cond {
								break
							}
						}
					}
				}
				if Condition(b) == ConditionNotContainsElement {
					cond = !cond
				}
			case ConditionContainsKey, ConditionNotContainsKey:
				x := vm.general(a)
				if x.Len() > 0 {
					v := reflect.New(x.Type().Key()).Elem()
					_ = vm.getIntoReflectValue(c, v, op < 0)
					cond = x.MapIndex(v).IsValid()
				}
				if Condition(b) == ConditionNotContainsKey {
					cond = !cond
				}
			case ConditionContainsRune, ConditionNotContainsRune:
				x := vm.string(a)
				if x != "" {
					y := vm.intk(c, op < 0)
					cond = strings.ContainsRune(x, rune(y))
				}
				if Condition(b) == ConditionNotContainsRune {
					cond = !cond
				}
			default:
				v1 := vm.int(a)
				v2 := vm.intk(c, op < 0)
				switch Condition(b) {
				case ConditionEqual:
					cond = v1 == v2
				case ConditionNotEqual:
					cond = v1 != v2
				case ConditionLess:
					cond = v1 < v2
				case ConditionLessEqual:
					cond = v1 <= v2
				case ConditionGreater:
					cond = v1 > v2
				case ConditionGreaterEqual:
					cond = v1 >= v2
				}
			}
			if cond {
				vm.pc++
			}
		case OpIfFloat, -OpIfFloat:
			var cond bool
			switch bb := Condition(b); bb {
			case ConditionContainsElement, ConditionNotContainsElement:
				x := vm.general(a)
				y := vm.floatk(c, op < 0)
				if s, ok := x.Interface().([]float64); ok {
					for _, e := range s {
						if e == y {
							cond = true
							break
						}
					}
				} else {
					n := x.Len()
					for i := 0; i < n; i++ {
						if x.Index(i).Float() == y {
							cond = true
							break
						}
					}
				}
				if bb == ConditionNotContainsElement {
					cond = !cond
				}
			case ConditionContainsKey, ConditionNotContainsKey:
				x := vm.general(a)
				if x.Len() > 0 {
					v := reflect.New(x.Type().Key()).Elem()
					_ = vm.getIntoReflectValue(c, v, op < 0)
					cond = x.MapIndex(v).IsValid()
				}
				if bb == ConditionNotContainsKey {
					cond = !cond
				}
			default:
				x := vm.float(a)
				y := vm.floatk(c, op < 0)
				switch bb {
				case ConditionEqual:
					cond = x == y
				case ConditionNotEqual:
					cond = x != y
				case ConditionLess:
					cond = x < y
				case ConditionLessEqual:
					cond = x <= y
				case ConditionGreater:
					cond = x > y
				case ConditionGreaterEqual:
					cond = x >= y
				}
			}
			if cond {
				vm.pc++
			}
		case OpIfString, -OpIfString:
			var cond bool
			bb := Condition(b)
			if bb <= ConditionGreaterEqual {
				v1 := vm.string(a)
				v2 := vm.stringk(c, op < 0)
				switch bb {
				case ConditionEqual:
					cond = v1 == v2
				case ConditionNotEqual:
					cond = v1 != v2
				case ConditionLess:
					cond = v1 < v2
				case ConditionLessEqual:
					cond = v1 <= v2
				case ConditionGreater:
					cond = v1 > v2
				case ConditionGreaterEqual:
					cond = v1 >= v2
				}
			} else if bb <= ConditionLenGreaterEqual {
				v1 := vm.string(a)
				v2 := int(vm.intk(c, op < 0))
				switch bb {
				case ConditionLenEqual:
					cond = len(v1) == v2
				case ConditionLenNotEqual:
					cond = len(v1) != v2
				case ConditionLenLess:
					cond = len(v1) < v2
				case ConditionLenLessEqual:
					cond = len(v1) <= v2
				case ConditionLenGreater:
					cond = len(v1) > v2
				case ConditionLenGreaterEqual:
					cond = len(v1) >= v2
				}
			} else if bb == ConditionContainsSubstring || bb == ConditionNotContainsSubstring {
				x := vm.string(a)
				y := vm.stringk(c, op < 0)
				cond = strings.Contains(x, y)
				if bb == ConditionNotContainsSubstring {
					cond = !cond
				}
			} else if bb == ConditionContainsElement || bb == ConditionNotContainsElement {
				x := vm.general(a)
				n := x.Len()
				if n > 0 {
					y := vm.stringk(c, op < 0)
					for i := 0; i < n; i++ {
						if x.Index(i).String() == y {
							cond = true
							break
						}
					}
				}
				if bb == ConditionNotContainsElement {
					cond = !cond
				}
			} else {
				// ConditionContainsKey, ConditionNotContainsKey
				x := vm.general(a)
				if x.Len() > 0 {
					v := reflect.New(x.Type().Key()).Elem()
					_ = vm.getIntoReflectValue(c, v, op < 0)
					cond = x.MapIndex(v).IsValid()
				}
				if bb == ConditionNotContainsKey {
					cond = !cond
				}
			}
			if cond {
				vm.pc++
			}

		// Index
		case OpIndex, -OpIndex:
			// TODO: OpIndex and -OpIndex currently return the reference to the
			// element returned by the indexing operation, so the implementation
			// is the same as OpIndexRef and -OpIndexRef. This is going to
			// change in a future commit.
			v := vm.general(a)
			i := int(vm.intk(b, op < 0))
			vm.setFromReflectValue(c, v.Index(i))
		case OpIndexString, -OpIndexString:
			vm.setInt(c, int64(vm.string(a)[int(vm.intk(b, op < 0))]))
		case OpIndexRef, -OpIndexRef:
			v := vm.general(a)
			i := int(vm.intk(b, op < 0))
			vm.setFromReflectValue(c, v.Index(i))

		// Len
		case OpLen:
			var length int
			if registerType(a) == stringRegister {
				length = len(vm.string(b))
			} else {
				length = vm.general(b).Len()
			}
			vm.setInt(c, int64(length))

		// Load
		case OpLoad:
			t, i := decodeValueIndex(a, b)
			switch t {
			case intRegister:
				vm.setInt(c, vm.fn.Values.Int[i])
			case floatRegister:
				vm.setFloat(c, vm.fn.Values.Float[i])
			case stringRegister:
				vm.setString(c, vm.fn.Values.String[i])
			case generalRegister:
				vm.setGeneral(c, vm.fn.Values.General[i])
			}

		// LoadFunc
		case OpLoadFunc:
			if a == 1 {
				fn := callable{}
				fn.native = vm.fn.NativeFunctions[uint8(b)]
				vm.setGeneral(c, reflect.ValueOf(&fn))
			} else {
				fn := vm.fn.Functions[uint8(b)]
				var vars []reflect.Value
				if fn.VarRefs != nil {
					vars = make([]reflect.Value, len(fn.VarRefs))
					for i, ref := range fn.VarRefs {
						if ref < 0 {
							// Calling Elem() is necessary because the general
							// register contains an indirect value, that is
							// stored as a pointer to a value.
							vars[i] = vm.general(int8(-ref)).Elem()
						} else {
							vars[i] = vm.vars[ref]
						}
					}
				} else {
					vars = vm.env.globals
				}
				vm.setGeneral(c, reflect.ValueOf(&callable{fn: fn, vars: vars}))
			}

		// MakeArray
		case OpMakeArray:
			t := vm.fn.Types[uint8(b)]
			vm.setGeneral(c, reflect.New(t).Elem())

		// MakeChan
		case OpMakeChan, -OpMakeChan:
			typ := vm.fn.Types[uint8(a)]
			buffer := int(vm.intk(b, op < 0))
			var ch reflect.Value
			if typ.ChanDir() == reflect.BothDir {
				ch = reflect.MakeChan(typ, buffer)
			} else {
				t := reflect.ChanOf(reflect.BothDir, typ.Elem())
				ch = reflect.MakeChan(t, buffer).Convert(typ)
			}
			vm.setGeneral(c, ch)

		// MakeMap
		case OpMakeMap, -OpMakeMap:
			typ := vm.fn.Types[uint8(a)]
			n := int(vm.intk(b, op < 0))
			if n > 0 {
				vm.setGeneral(c, reflect.MakeMapWithSize(typ, n))
			} else {
				vm.setGeneral(c, reflect.MakeMap(typ))
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
			vm.setGeneral(c, reflect.MakeSlice(typ, len, cap))
			if b > 0 {
				vm.pc++
			}

		// MakeStruct
		case OpMakeStruct:
			t := vm.fn.Types[uint8(b)]
			vm.setGeneral(c, reflect.New(t).Elem())

		// MapIndex
		case OpMapIndex, -OpMapIndex:
			m := vm.general(a)
			t := m.Type()
			k := reflect.New(t.Key()).Elem()
			vm.getIntoReflectValue(b, k, op < 0)
			elem := reflect.New(t.Elem()).Elem()
			index := m.MapIndex(k)
			vm.ok = index.IsValid()
			if vm.ok {
				elem.Set(index)
			}
			vm.setFromReflectValue(c, elem)

		// MethodValue
		case OpMethodValue:
			receiver := vm.general(a)
			if !receiver.IsValid() {
				panic(errNilPointer)
			}
			method := vm.stringk(b, true)
			vm.setGeneral(c, reflect.ValueOf(&callable{value: receiver.MethodByName(method)}))

		// Move
		case OpMove, -OpMove:
			switch registerType(a) {
			case intRegister:
				vm.setInt(c, vm.intk(b, op < 0))
			case floatRegister:
				vm.setFloat(c, vm.floatk(b, op < 0))
			case stringRegister:
				vm.setString(c, vm.stringk(b, op < 0))
			case generalRegister:
				rv := vm.generalk(b, op < 0)
				if k := rv.Kind(); k == reflect.Array || k == reflect.Struct {
					newRv := reflect.New(rv.Type()).Elem()
					newRv.Set(reflect.ValueOf(rv.Interface()))
					rv = newRv
				}
				vm.setGeneral(c, rv)
			}

		// Mul
		case OpMul, -OpMul:
			switch a := reflect.Kind(a); a {
			case reflect.Float32:
				vm.setFloat(c, float64(float32(vm.float(c)*vm.floatk(b, op < 0))))
			case reflect.Float64:
				vm.setFloat(c, vm.float(c)*vm.floatk(b, op < 0))
			default:
				v := vm.int(c) * vm.intk(b, op < 0)
				switch a {
				case reflect.Int8:
					v = int64(int8(v))
				case reflect.Int16:
					v = int64(int16(v))
				case reflect.Int32:
					v = int64(int32(v))
				case reflect.Uint8:
					v = int64(uint8(v))
				case reflect.Uint16:
					v = int64(uint16(v))
				case reflect.Uint32:
					v = int64(uint32(v))
				case reflect.Uint64:
					v = int64(uint64(v))
				}
				vm.setInt(c, v)
			}
		case OpMulInt, -OpMulInt:
			vm.setInt(c, vm.int(a)*vm.intk(b, op < 0))
		case OpMulFloat64, -OpMulFloat64:
			vm.setFloat(c, vm.float(a)*vm.floatk(b, op < 0))

		// Neg
		case OpNeg:
			switch a := reflect.Kind(a); a {
			case reflect.Float32:
				vm.setFloat(c, float64(-float32(vm.float(b))))
			case reflect.Float64:
				vm.setFloat(c, -vm.float(b))
			default:
				v := vm.int(b)
				switch a {
				case reflect.Int8:
					v = int64(-int8(v))
				case reflect.Int16:
					v = int64(-int16(v))
				case reflect.Int32:
					v = int64(-int32(v))
				case reflect.Int64:
					v = -v
				case reflect.Uint8:
					v = int64(-uint8(v))
				case reflect.Uint16:
					v = int64(-uint16(v))
				case reflect.Uint32:
					v = int64(-uint32(v))
				case reflect.Uint64:
					v = int64(-uint64(v))
				}
				vm.setInt(c, v)
			}

		// New
		case OpNew:
			t := vm.fn.Types[uint8(b)]
			vm.setGeneral(c, reflect.New(t))

		// Or
		case OpOr, -OpOr:
			vm.setInt(c, vm.int(a)|vm.intk(b, op < 0))

		// Panic
		case OpPanic:
			rv := vm.general(a)
			if rv.IsValid() {
				panic(rv.Interface())
			} else {
				panic(nil)
			}

		// Print
		case OpPrint:
			rv := vm.general(a)
			if rv.IsValid() {
				vm.env.doPrint(rv.Interface())
			} else {
				vm.env.doPrint(nil)
			}

		// Range
		case OpRange:
			var addr Addr
			var breakOut bool
			rangeAddress := vm.pc - 1
			bodyAddress := vm.pc + 1
			v := vm.general(a)
			switch s := v.Interface().(type) {
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
						vm.setGeneral(c, reflect.ValueOf(v))
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
						vm.setGeneral(c, reflect.ValueOf(v))
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
				kind := v.Kind()
				if kind == reflect.Map {
					iter := v.MapRange()
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
						v = v.Elem()
					}
					length := v.Len()
					for i := 0; i < length; i++ {
						if b != 0 {
							vm.setInt(b, int64(i))
						}
						if c != 0 {
							vm.setFromReflectValue(c, v.Index(i))
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

		// RealImag
		case OpRealImag, -OpRealImag:
			v := vm.generalk(a, op < 0)
			cmpx := v.Complex()
			re, im := real(cmpx), imag(cmpx)
			if b > 0 {
				vm.setFloat(b, re)
			}
			if c > 0 {
				vm.setFloat(c, im)
			}

		// Receive
		case OpReceive:
			channel := vm.general(a)
			switch ch := channel.Interface().(type) {
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
					vm.setGeneral(c, reflect.ValueOf(struct{}{}))
				}
			default:
				var v reflect.Value
				v, vm.ok = channel.Recv()
				if c != 0 {
					vm.setFromReflectValue(c, v)
				}
			}
			if b != 0 {
				vm.setBool(b, vm.ok)
			}

		// Recover
		case OpRecover:
			var msg reflect.Value
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
					msg = reflect.ValueOf(vm.panic.message)
				}
				break
			}
			if c != 0 {
				vm.setGeneral(c, msg)
			}

		// Rem
		case OpRem, -OpRem:
			bv := vm.intk(b, op < 0)
			cv := vm.int(c)
			var v int64
			switch reflect.Kind(a) {
			case reflect.Int8:
				v = int64(int8(cv) % int8(bv))
			case reflect.Int16:
				v = int64(int16(cv) % int16(bv))
			case reflect.Int32:
				v = int64(int32(cv) % int32(bv))
			case reflect.Int64:
				v = cv % bv
			case reflect.Uint8:
				v = int64(uint8(cv) % uint8(bv))
			case reflect.Uint16:
				v = int64(uint16(cv) % uint16(bv))
			case reflect.Uint32:
				v = int64(uint32(cv) % uint32(bv))
			case reflect.Uint64:
				v = int64(uint64(cv) % uint64(bv))
			}
			vm.setInt(c, v)
		case OpRemInt, -OpRemInt:
			vm.setInt(c, vm.int(a)%vm.intk(b, op < 0))

		// Return
		case OpReturn:
			i := len(vm.calls) - 1
			if i == -1 {
				return maxUint32, false
			}
			call := vm.calls[i]
			if call.status == started {
				if vm.fn.Macro {
					if call.renderer != vm.renderer {
						out := vm.renderer.Out()
						if b, ok := out.(*macroOutBuffer); ok {
							vm.setString(1, b.String())
						}
						err := vm.renderer.Close()
						if err != nil {
							panic(&fatalError{env: vm.env, msg: err})
						}
					}
					vm.renderer = call.renderer
				} else if regs := vm.fn.FinalRegs; regs != nil {
					vm.finalize(vm.fn.FinalRegs)
				}
				vm.calls = vm.calls[:i]
				vm.fp = call.fp
				vm.fn = call.cl.fn
				vm.vars = call.cl.vars
				vm.pc = call.pc
			} else if !vm.nextCall() {
				return maxUint32, false
			}

		// Select
		case OpSelect:
			numCase := len(vm.cases)
			chosen, recv, recvOK := reflect.Select(vm.cases)
			step := numCase - chosen
			var pc Addr
			if step > 0 {
				pc = vm.pc - 2*Addr(step)
				if vm.cases[chosen].Dir == reflect.SelectRecv {
					r := vm.fn.Body[pc-1].C
					if r != 0 {
						vm.setFromReflectValue(r, recv)
					}
					vm.ok = recvOK
				}
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
			vm.pc = pc

		// Send
		case OpSend, -OpSend:
			k := op < 0
			channel := vm.generalk(c, k)
			switch ch := channel.Interface().(type) {
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
				elemType := channel.Type().Elem()
				v := reflect.New(elemType).Elem()
				vm.getIntoReflectValue(a, v, k)
				channel.Send(v)
			}

		// SetField
		case OpSetField, -OpSetField:
			v := vm.general(b)
			vm.getIntoReflectValue(a, vm.fieldByIndex(v, uint8(c)), op < 0)

		// SetMap
		case OpSetMap, -OpSetMap:
			mv := vm.general(b)
			switch m := mv.Interface().(type) {
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
				rv := vm.generalk(a, op < 0)
				if rv.IsValid() {
					m[k] = rv.Interface()
				} else {
					m[k] = nil
				}
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
			sv := vm.general(b)
			switch s := sv.Interface().(type) {
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
				rv := vm.generalk(a, op < 0)
				if rv.IsValid() {
					s[i] = rv.Interface()
				} else {
					s[i] = nil
				}
			default:
				v := sv.Index(int(i))
				vm.getIntoReflectValue(a, v, op < 0)
			}

		// SetVar
		case OpSetVar, -OpSetVar:
			v := vm.vars[decodeInt16(b, c)]
			vm.getIntoReflectValue(a, v, op < 0)

		// Shl
		case OpShl, -OpShl:
			v := vm.int(c) << uint(vm.intk(b, op < 0))
			switch reflect.Kind(a) {
			case reflect.Int8:
				v = int64(int8(v))
			case reflect.Int16:
				v = int64(int16(v))
			case reflect.Int32:
				v = int64(int32(v))
			case reflect.Uint8:
				v = int64(uint8(v))
			case reflect.Uint16:
				v = int64(uint16(v))
			case reflect.Uint32:
				v = int64(uint32(v))
			case reflect.Uint64:
				v = int64(uint64(v))
			}
			vm.setInt(c, v)
		case OpShlInt, -OpShlInt:
			vm.setInt(c, vm.int(a)<<uint(vm.intk(b, op < 0)))

		// Shr
		case OpShr, -OpShr:
			bv := uint(vm.intk(b, op < 0))
			cv := vm.int(c)
			var v int64
			if reflect.Kind(a) < reflect.Uint8 {
				v = cv >> bv
			} else {
				v = int64(uint64(cv) >> bv)
			}
			vm.setInt(c, v)
		case OpShrInt, -OpShrInt:
			vm.setInt(c, vm.int(a)>>uint(vm.intk(b, op < 0)))

		// Show
		case OpShow:
			t := vm.fn.Types[uint8(a)]
			st, ok := t.(ScriggoType)
			if ok {
				t = st.GoType()
			}
			rv := reflect.New(t).Elem()
			vm.getIntoReflectValue(b, rv, op < 0)
			if st != nil {
				rv = st.Wrap(rv)
			}
			var v interface{}
			if rv.IsValid() {
				v = rv.Interface()
			}
			err := vm.renderer.Show(v, Context(c))
			if err != nil {
				panic(outError{err})
			}

		// Slice
		case OpSlice:
			var i1, i2, i3 int
			s := vm.general(a)
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
			vm.setGeneral(c, s)
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
		case OpSub, -OpSub:
			switch a := reflect.Kind(a); a {
			case reflect.Float32:
				vm.setFloat(c, float64(float32(vm.float(c)-vm.floatk(b, op < 0))))
			case reflect.Float64:
				vm.setFloat(c, vm.float(c)-vm.floatk(b, op < 0))
			default:
				v := vm.int(c) - vm.intk(b, op < 0)
				switch a {
				case reflect.Int8:
					v = int64(int8(v))
				case reflect.Int16:
					v = int64(int16(v))
				case reflect.Int32:
					v = int64(int32(v))
				case reflect.Uint8:
					v = int64(uint8(v))
				case reflect.Uint16:
					v = int64(uint16(v))
				case reflect.Uint32:
					v = int64(uint32(v))
				case reflect.Uint64:
					v = int64(uint64(v))
				}
				vm.setInt(c, v)
			}
		case OpSubInt, -OpSubInt:
			vm.setInt(c, vm.int(a)-vm.intk(b, op < 0))
		case OpSubFloat64, -OpSubFloat64:
			vm.setFloat(c, vm.float(a)-vm.floatk(b, op < 0))

		// SubInv
		case OpSubInv, -OpSubInv:
			switch a := reflect.Kind(a); a {
			case reflect.Float32:
				vm.setFloat(c, float64(float32(vm.floatk(b, op < 0)-vm.float(c))))
			case reflect.Float64:
				vm.setFloat(c, vm.floatk(b, op < 0)-vm.float(c))
			default:
				v := vm.intk(b, op < 0) - vm.int(c)
				switch a {
				case reflect.Int8:
					v = int64(int8(v))
				case reflect.Int16:
					v = int64(int16(v))
				case reflect.Int32:
					v = int64(int32(v))
				case reflect.Uint8:
					v = int64(uint8(v))
				case reflect.Uint16:
					v = int64(uint16(v))
				case reflect.Uint32:
					v = int64(uint32(v))
				case reflect.Uint64:
					v = int64(uint64(v))
				}
				vm.setInt(c, v)
			}
		case OpSubInvInt, -OpSubInvInt:
			vm.setInt(c, vm.intk(b, op < 0)-vm.int(a))
		case OpSubInvFloat64, -OpSubInvFloat64:
			vm.setFloat(c, vm.float(b)-vm.floatk(a, op < 0))

		// TailCall
		case OpTailCall:
			vm.calls = append(vm.calls, callFrame{cl: callable{fn: vm.fn, vars: vm.vars}, pc: vm.pc, status: tailed})
			if a != CurrentFunction {
				var fn *Function
				if a == 0 {
					closure := vm.general(b).Interface().(*callable)
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

		// Text
		case OpText:
			txt := vm.fn.Text[decodeUint16(a, b)]
			inURL, isSet := c > 0, c == 2
			err := vm.renderer.Text(txt, inURL, isSet)
			if err != nil {
				panic(outError{err})
			}

		// Typify
		case OpTypify, -OpTypify:
			t := vm.fn.Types[uint8(a)]
			st, ok := t.(ScriggoType)
			if ok {
				t = st.GoType()
			}
			v := reflect.New(t).Elem()
			vm.getIntoReflectValue(b, v, op < 0)
			if st != nil {
				v = st.Wrap(v)
			}
			vm.setGeneral(c, v)

		// Xor
		case OpXor, -OpXor:
			vm.setInt(c, vm.int(a)^vm.intk(b, op < 0))

		// Zero
		case OpZero:
			not := false
			if a >= 10 {
				a -= 10
				not = true
			}
			zero := true
			switch registerType(a) {
			case intRegister:
				zero = vm.int(b) == 0
			case floatRegister:
				zero = vm.float(b) == 0
			case stringRegister:
				zero = vm.string(b) == ""
			case generalRegister:
				rv := vm.general(b)
				// rv is not valid when the register addressed by b stores the
				// nil interface.
				if rv.IsValid() {
					// First of all check if the type of the value has a method
					// 'IsZero() bool' defined on it; if so, call such method
					// and use its return value to determine if rv is zero or
					// not.
					if rv.Type().Implements(isZeroType) {
						isZero := rv.MethodByName("IsZero")
						zero = isZero.Call(nil)[0].Bool()
					} else {
						// Note that rv.IsZero() also handles values that have
						// static type interface because the interface is lost when
						// the value is inserted into the register, so IsZero checks
						// for the zero of the dynamic type.
						zero = rv.IsZero()
						// Not-nil channels and slices are zero if they contain
						// zero elements.
						if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Chan {
							// Note that rv.Len() == 0 also handles values that have
							// static type interface because the interface is lost
							// when the value is inserted into the register, so
							// rv.Len() returns the number of elements of the
							// dynamic value.
							zero = zero || rv.Len() == 0
						}
					}
				}
			}
			if not {
				zero = !zero
			}
			vm.setBool(c, zero)
		}

	}

}
