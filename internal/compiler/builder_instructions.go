// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/runtime"
)

// emitNop appends a new "Nop" instruction to the function body.
//
func (fb *functionBuilder) emitNop() {
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpNone})
}

// emitAdd appends a new "Add" instruction to the function body.
//
//     z = x + y
//
func (fb *functionBuilder) emitAdd(k bool, x, y, z int8, kind reflect.Kind) {
	var op runtime.Operation
	switch kind {
	case reflect.Int:
		op = runtime.OpAddInt
	case reflect.Float64:
		op = runtime.OpAddFloat64
	default:
		if z != x {
			panic(fmt.Errorf("z must be == x for kind %s", kind))
		}
		x = int8(flattenIntegerKind(kind))
		op = runtime.OpAdd
	}
	if k {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitAddr appends a new "Addr" instruction to the function body.
//
// 	   dest = &expr.Field
// 	   dest = &expr[index]
//
func (fb *functionBuilder) emitAddr(expr, index, dest int8, pos *ast.Position) {
	fb.addPosAndPath(pos)
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpAddr, A: expr, B: index, C: dest})
}

// emitAnd appends a new "And" instruction to the function body.
//
//     z = x & y
//
func (fb *functionBuilder) emitAnd(k bool, x, y, z int8, kind reflect.Kind) {
	op := runtime.OpAnd
	if k {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitAndNot appends a new "AndNot" instruction to the function body.
//
//     z = x &^ y
//
func (fb *functionBuilder) emitAndNot(k bool, x, y, z int8, kind reflect.Kind) {
	op := runtime.OpAndNot
	if k {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitAppend appends a new "Append" instruction to the function body.
//
func (fb *functionBuilder) emitAppend(start, end, s int8, elementsKind reflect.Kind) {
	fb.addOperandKinds(elementsKind, elementsKind, 0)
	fn := fb.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAppend, A: start, B: end, C: s})
}

// emitAppendSlice appends a new "AppendSlice" instruction to the function body.
//
//     s = append(s, t)
//
func (fb *functionBuilder) emitAppendSlice(t, s int8, pos *ast.Position) {
	fb.addPosAndPath(pos)
	fn := fb.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAppendSlice, A: t, C: s})
}

// emitAssert appends a new "assert" instruction to the function body.
//
//     z = e.(t)
//
func (fb *functionBuilder) emitAssert(e int8, typ reflect.Type, z int8) {
	t := fb.addType(typ, true)
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpAssert, A: e, B: int8(t), C: z})
}

// emitBreak appends a new "Break" instruction to the function body.
//
//     break addr
//
func (fb *functionBuilder) emitBreak(lab label) {
	addr := fb.labelAddrs[lab-1]
	a, b, c := encodeUint24(uint32(addr))
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpBreak, A: a, B: b, C: c})
}

// emitCallFunc appends a new "CallFunc" instruction to the function body.
//
//     p.f()
//
func (fb *functionBuilder) emitCallFunc(f int8, shift runtime.StackShift, pos *ast.Position) {
	fb.addPosAndPath(pos)
	fn := fb.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpCallFunc, A: f})
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.Operation(shift[0]), A: shift[1], B: shift[2], C: shift[3]})
}

// emitCallMacro appends a new "CallMacro" instruction to the function body.
//
//     p.m()
//
func (fb *functionBuilder) emitCallMacro(f int8, shift runtime.StackShift, pos *ast.Position, toFormat ast.Format) {
	fb.addPosAndPath(pos)
	fn := fb.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpCallMacro, A: f, B: int8(toFormat)})
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.Operation(shift[0]), A: shift[1], B: shift[2], C: shift[3]})
}

// emitCallIndirect appends a new "CallIndirect" instruction to the function body.
//
//     f()
//
func (fb *functionBuilder) emitCallIndirect(f int8, numVariadic int8, shift runtime.StackShift, pos *ast.Position, funcType reflect.Type, toFormat ast.Format) {
	fb.addPosAndPath(pos)
	fb.addFunctionType(funcType)
	fn := fb.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpCallIndirect, A: f, B: int8(toFormat), C: numVariadic})
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.Operation(shift[0]), A: shift[1], B: shift[2], C: shift[3]})
}

// emitCallNative appends a new "CallNative" instruction to the function body.
//
//     p.F()
//
func (fb *functionBuilder) emitCallNative(f int8, numVariadic int8, shift runtime.StackShift, pos *ast.Position) {
	fb.addPosAndPath(pos)
	fn := fb.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpCallNative, A: f, C: numVariadic})
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.Operation(shift[0]), A: shift[1], B: shift[2], C: shift[3]})
}

// emitCap appends a new "cap" instruction to the function body.
//
//     z = cap(s)
//
func (fb *functionBuilder) emitCap(s, z int8) {
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpCap, A: s, C: z})
}

// emitCase appends a new "Case" instruction to the function body.
//
//     case ch <- value
//     case value = <-ch
//     default
//
func (fb *functionBuilder) emitCase(kvalue bool, dir reflect.SelectDir, value, ch int8) {
	in := runtime.Instruction{Op: runtime.OpCase, A: int8(dir)}
	if kvalue {
		in.Op = -in.Op
	}
	switch dir {
	case reflect.SelectSend:
		in.B = value
		in.C = ch
	case reflect.SelectRecv:
		in.B = ch
		in.C = value
	}
	fb.fn.Body = append(fb.fn.Body, in)
}

// emitClose appends a new "Close" instruction to the function body.
//
//     close(ch)
//
func (fb *functionBuilder) emitClose(ch int8, pos *ast.Position) {
	fb.addPosAndPath(pos)
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpClose, A: ch})
}

// emitComplex appends a new "Complex" instruction to the function body.
//
//	z = complex(x, y)
//
func (fb *functionBuilder) emitComplex(x, y, z int8, kind reflect.Kind) {
	op := runtime.OpComplex128
	if kind == reflect.Complex64 {
		op = runtime.OpComplex64
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitConcat appends a new "concat" instruction to the function body.
//
//     z = concat(s, t)
//
func (fb *functionBuilder) emitConcat(s, t, z int8) {
	fn := fb.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpConcat, A: s, B: t, C: z})
}

// emitContinue appends a new "Continue" instruction to the function body.
//
//     continue label
//
func (fb *functionBuilder) emitContinue(lab label) {
	addr := fb.labelAddrs[lab-1]
	a, b, c := encodeUint24(uint32(addr))
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpContinue, A: a, B: b, C: c})
}

// emitConvert appends a new "Convert" instruction to the function body.
//
// 	 dst = typ(src)
//
func (fb *functionBuilder) emitConvert(src int8, typ reflect.Type, dst int8, srcKind reflect.Kind) {
	fn := fb.fn
	regType := fb.addType(typ, false)
	var op runtime.Operation
	switch kindToType(srcKind) {
	case generalRegister:
		op = runtime.OpConvert
	case intRegister:
		switch srcKind {
		case reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64,
			reflect.Uintptr:
			op = runtime.OpConvertUint
		default:
			op = runtime.OpConvertInt
		}
	case stringRegister:
		op = runtime.OpConvertString
	case floatRegister:
		op = runtime.OpConvertFloat
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: op, A: src, B: int8(regType), C: dst})
}

// emitCopy appends a new "Copy" instruction to the function body.
//
//     n == 0:   copy(dst, src)
// 	 n != 0:   n := copy(dst, src)
//
func (fb *functionBuilder) emitCopy(dst, src, n int8) {
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpCopy, A: src, B: n, C: dst})
}

// emitDefer appends a new "Defer" instruction to the function body.
//
//     defer
//
func (fb *functionBuilder) emitDefer(f int8, numVariadic int8, off, arg runtime.StackShift, funcType reflect.Type) {
	fb.addFunctionType(funcType)
	fn := fb.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpDefer, A: f, C: numVariadic})
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.Operation(off[0]), A: off[1], B: off[2], C: off[3]})
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.Operation(arg[0]), A: arg[1], B: arg[2], C: arg[3]})
}

// emitDelete appends a new "delete" instruction to the function body.
//
//     delete(m, k)
//
func (fb *functionBuilder) emitDelete(m, k int8) {
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpDelete, A: m, B: k})
}

// emitDiv appends a new "div" instruction to the function body.
//
//     z = x / y
//
func (fb *functionBuilder) emitDiv(ky bool, x, y, z int8, kind reflect.Kind, pos *ast.Position) {
	var op runtime.Operation
	switch kind {
	case reflect.Int:
		op = runtime.OpDivInt
	case reflect.Float64:
		op = runtime.OpDivFloat64
	default:
		if z != x {
			panic(fmt.Errorf("z must be == x for kind %s", kind))
		}
		x = int8(flattenIntegerKind(kind))
		op = runtime.OpDiv
	}
	fb.addPosAndPath(pos)
	if ky {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitField appends a new "Field" instruction to the function body.
//
//  c = a.field
//
func (fb *functionBuilder) emitField(a, field, c int8, dstKind reflect.Kind) {
	fb.addOperandKinds(0, 0, dstKind)
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpField, A: a, B: field, C: c})
}

// emitGetVar appends a new "GetVar" instruction to the function body.
//
//     r = v
//
func (fb *functionBuilder) emitGetVar(v int, r int8, varKind reflect.Kind) {
	a, b := encodeInt16(int16(v))
	fb.addOperandKinds(0, 0, varKind)
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpGetVar, A: a, B: b, C: r})
}

// emitGetVarAddr appends a new "GetVarAddr" instruction to the function body.
//
//	   r = &v
//
func (fb *functionBuilder) emitGetVarAddr(v int, r int8) {
	a, b := encodeInt16(int16(v))
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpGetVarAddr, A: a, B: b, C: r})
}

// emitGo appends a new "Go" instruction to the function body.
//
//     go
//
func (fb *functionBuilder) emitGo() {
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpGo})
}

// emitGoto appends a new "goto" instruction to the function body.
//
//     goto label
//
func (fb *functionBuilder) emitGoto(lab label) {
	in := runtime.Instruction{Op: runtime.OpGoto}
	if lab > 0 {
		if lab > label(len(fb.labelAddrs)) {
			panic("BUG") // remove.
		}
		addr := fb.labelAddrs[lab-1]
		if addr == 0 {
			fb.gotos[fb.currentAddr()] = lab
		} else {
			in.A, in.B, in.C = encodeUint24(uint32(addr))
		}
	}
	fb.fn.Body = append(fb.fn.Body, in)
}

// emitIf appends a new "If" instruction to the function body.
//
//     x == 0
//     x != 0
//     x == y
//     x != y
//     x <  y
//     x <= y
//     x >  y
//     x >= y
//     x == nil
//     x != nil
//     len(x) == y
//     len(x) != y
//     len(x) <  y
//     len(x) <= y
//     len(x) >  y
//     len(x) >= y
//     x contains y
//
func (fb *functionBuilder) emitIf(ky bool, x int8, o runtime.Condition, y int8, kind reflect.Kind, pos *ast.Position) {
	fb.addPosAndPath(pos)
	var op runtime.Operation
	switch kindToType(kind) {
	case intRegister:
		op = runtime.OpIfInt
	case floatRegister:
		op = runtime.OpIfFloat
	case stringRegister:
		op = runtime.OpIfString
	case generalRegister:
		op = runtime.OpIf
	}
	if ky {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: int8(o), C: y})
}

// emitIndex appends a new "Index", "IndexRef" or "MapIndex" instruction to the
// function body. If ref is set then the result of the indexing operation is a
// reference to the index (i.e. is an addressable reflect.Value with the same
// underlying index value); otherwise the index is copied.
//
//  dst = expr[i]
//
// TODO: consider splitting emitIndex in two methods removing the 'ref bool'
// argument.
func (fb *functionBuilder) emitIndex(ki bool, expr, i, dst int8, exprType reflect.Type, pos *ast.Position, ref bool) {
	fb.addPosAndPath(pos)
	fb.addOperandKinds(0, 0, exprType.Kind())
	fn := fb.fn
	kind := exprType.Kind()
	// TODO: re-enable this check?
	// if ref && (kind != reflect.Array && kind != reflect.Slice) {
	// 	panic(fmt.Errorf("BUG: cannot set the ref argument if the expression has kind %s", kind))
	// }
	var op runtime.Operation
	switch kind {
	case reflect.Array, reflect.Slice:
		if ref {
			op = runtime.OpIndexRef
		} else {
			op = runtime.OpIndex
		}
	case reflect.Map:
		op = runtime.OpMapIndex
	case reflect.String:
		op = runtime.OpIndexString
	default:
		panic(fmt.Errorf("BUG: invalid type %s", exprType))
	}
	if ki {
		op = -op
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: op, A: expr, B: i, C: dst})
}

// emitLen appends a new "len" instruction to the function body.
//
//     l = len(s)
//
func (fb *functionBuilder) emitLen(s, l int8, t reflect.Type) {
	a := stringRegister
	if t.Kind() != reflect.String {
		a = generalRegister
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpLen, A: int8(a), B: s, C: l})
}

// emitLoadFunc appends a new "LoadFunc" instruction to the function body.
//
//     z = p.f
//
func (fb *functionBuilder) emitLoadFunc(native bool, f int8, z int8) {
	fn := fb.fn
	var a int8
	if native {
		a = 1
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpLoadFunc, A: a, B: f, C: z})
}

// emitLoad appends a new "Load" instruction to the function body.
//
func (fb *functionBuilder) emitLoad(index int, dst int8, kind reflect.Kind) {
	a, b := encodeValueIndex(kindToType(kind), index)
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpLoad, A: a, B: b, C: dst})
}

// emitMakeArray appends a new "MakeArray" instruction to the function body.
func (fb *functionBuilder) emitMakeArray(typ reflect.Type, dst int8) {
	if typ.Kind() != reflect.Array {
		panic("BUG: not an array type")
	}
	// NOTE: the code of emitMakeArray, emitMakeStruct and emitNew is very
	// similar. If you change this code remember to review/change the code of
	// the other methods.
	fn := fb.fn
	b := fb.addType(typ, false)
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpMakeArray, B: int8(b), C: dst})
}

// emitMakeChan appends a new "MakeChan" instruction to the function body.
//
//     dst = make(typ, capacity)
//
func (fb *functionBuilder) emitMakeChan(typ reflect.Type, kCapacity bool, capacity int8, dst int8, pos *ast.Position) {
	fb.addPosAndPath(pos)
	fn := fb.fn
	t := fb.addType(typ, false)
	op := runtime.OpMakeChan
	if kCapacity {
		op = -op
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: op, A: int8(t), B: capacity, C: dst})
}

// emitMakeMap appends a new "MakeMap" instruction to the function body.
//
//     dst = make(typ, size)
//
func (fb *functionBuilder) emitMakeMap(typ reflect.Type, kSize bool, size int8, dst int8) {
	fn := fb.fn
	t := fb.addType(typ, false)
	op := runtime.OpMakeMap
	if kSize {
		op = -op
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: op, A: int8(t), B: size, C: dst})
}

// emitMakeSlice appends a new "MakeSlice" instruction to the function body.
//
//     make(sliceType, len, cap)
//
func (fb *functionBuilder) emitMakeSlice(kLen, kCap bool, sliceType reflect.Type, len, cap, dst int8, pos *ast.Position) {
	fb.addPosAndPath(pos)
	fn := fb.fn
	t := fb.addType(sliceType, false)
	var k int8
	if len == 0 && cap == 0 {
		k = 0
	} else {
		k = 1
		if kLen {
			k |= 1 << 1
		}
		if kCap {
			k |= 1 << 2
		}
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpMakeSlice, A: int8(t), B: k, C: dst})
	if k > 0 {
		fn.Body = append(fn.Body, runtime.Instruction{A: len, B: cap})
	}
}

// emitMakeStruct appends a new "MakeStruct" instruction to the function body.
func (fb *functionBuilder) emitMakeStruct(typ reflect.Type, dst int8) {
	if typ.Kind() != reflect.Struct {
		panic("BUG: not a struct type")
	}
	// NOTE: the code of emitMakeArray, emitMakeStruct and emitNew is very
	// similar. If you change this code remember to review/change the code of
	// the other methods.
	fn := fb.fn
	b := fb.addType(typ, false)
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpMakeStruct, B: int8(b), C: dst})
}

// emitMethodValue appends a new "MethodValue" instruction to the function body.
//
//     dst = receiver.name
//
func (fb *functionBuilder) emitMethodValue(name int8, receiver int8, dst int8, pos *ast.Position) {
	fb.addPosAndPath(pos)
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpMethodValue, A: receiver, B: name, C: dst})
}

// emitMove appends a new "Move" instruction to the function body.
//
//     z = x
//
func (fb *functionBuilder) emitMove(k bool, x, z int8, kind reflect.Kind) {
	op := runtime.OpMove
	if k {
		op = -op
	}
	a := int8(kindToType(kind))
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: a, B: x, C: z})
}

// emitMul appends a new "mul" instruction to the function body.
//
//     z = x * y
//
func (fb *functionBuilder) emitMul(ky bool, x, y, z int8, kind reflect.Kind) {
	var op runtime.Operation
	switch kind {
	case reflect.Int:
		op = runtime.OpMulInt
	case reflect.Float64:
		op = runtime.OpMulFloat64
	default:
		if z != x {
			panic(fmt.Errorf("z must be == x for kind %s", kind))
		}
		x = int8(flattenIntegerKind(kind))
		op = runtime.OpMul
	}
	if ky {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitNeg appends a new "neg" instruction to the function body.
//
//     z = -y
//
func (fb *functionBuilder) emitNeg(y, z int8, kind reflect.Kind) {
	x := int8(flattenIntegerKind(kind))
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpNeg, A: x, B: y, C: z})
}

// emitNew appends a new "new" instruction to the function body.
//
//     z = new(t)
//
func (fb *functionBuilder) emitNew(typ reflect.Type, z int8) {
	// NOTE: the code of emitMakeArray, emitMakeStruct and emitNew is very
	// similar. If you change this code remember to review/change the code of
	// the other methods.
	fn := fb.fn
	b := fb.addType(typ, false)
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpNew, B: int8(b), C: z})
}

// emitNotZero appends a new "NotZero" instruction to the function body.
func (fb *functionBuilder) emitNotZero(kind reflect.Kind, dst, src int8) {
	regType := int8(kindToType(kind))
	regType += 10 // to distinguish "NotZero" from "Zero".
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpZero, A: regType, B: src, C: dst})
}

// emitOr appends a new "Or" instruction to the function body.
//
//     z = x | y
//
func (fb *functionBuilder) emitOr(k bool, x, y, z int8, kind reflect.Kind) {
	op := runtime.OpOr
	if k {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitPanic appends a new "Panic" instruction to the function body. If the
// panic follows a type assertion, typ should contain the type of the expression
// of the type assertion, in order to provide additional information to the VM;
// otherwise typ should be nil.
//
//     panic(v)
//
func (fb *functionBuilder) emitPanic(v int8, typ reflect.Type, pos *ast.Position) {
	fb.addPosAndPath(pos)
	fn := fb.fn
	in := runtime.Instruction{Op: runtime.OpPanic, A: v}
	if typ != nil {
		in.C = int8(fb.addType(typ, true))
	}
	fn.Body = append(fn.Body, in)
}

// emitPrint appends a new "Print" instruction to the function body.
//
//     print(arg)
//
func (fb *functionBuilder) emitPrint(arg int8) {
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpPrint, A: arg})
}

// emitRange appends a new "Range" instruction to the function body.
//
//	for i, e := range s
//
func (fb *functionBuilder) emitRange(k bool, s, i, e int8, kind reflect.Kind) {
	fn := fb.fn
	var op runtime.Operation
	switch kind {
	case reflect.String:
		op = runtime.OpRangeString
		if k {
			op = -op
		}
	default:
		if k {
			panic("bug on emitter: emitRange with k = true is compatible only with kind == reflect.String")
		}
		op = runtime.OpRange
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: op, A: s, B: i, C: e})
}

// emitRealImag appends a new "RealImag" instruction to the function body.
//
//	y, z = real(x), imag(x)
//
func (fb *functionBuilder) emitRealImag(k bool, x, y, z int8) {
	op := runtime.OpRealImag
	if k {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitReceive appends a new "Receive" instruction to the function body.
//
//	dst = <- ch
//
//	dst, ok = <- ch
//
func (fb *functionBuilder) emitReceive(ch, ok, dst int8) {
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpReceive, A: ch, B: ok, C: dst})
}

// emitRecover appends a new "Recover" instruction to the function body.
//
//     recover()
//     defer recover()
//
func (fb *functionBuilder) emitRecover(r int8, down bool) {
	var a int8
	if down {
		// Recover down the stack.
		a = 1
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpRecover, A: a, C: r})
}

// emitRem appends a new "Rem" instruction to the function body.
//
//     z = x % y
//
func (fb *functionBuilder) emitRem(ky bool, x, y, z int8, kind reflect.Kind, pos *ast.Position) {
	fb.addPosAndPath(pos)
	var op runtime.Operation
	switch kind {
	case reflect.Int:
		op = runtime.OpRemInt
	default:
		if z != x {
			panic(fmt.Errorf("z must be == x for kind %s", kind))
		}
		x = int8(flattenIntegerKind(kind))
		op = runtime.OpRem
	}
	if ky {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitReturn appends a new "return" instruction to the function body.
//
//     return
//
func (fb *functionBuilder) emitReturn() {
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpReturn})
}

// emitSelect appends a new "Select" instruction to the function body.
//
//     select
//
func (fb *functionBuilder) emitSelect() {
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpSelect})
}

// emitSend appends a new "Send" instruction to the function body.
//
//	ch <- v
//
func (fb *functionBuilder) emitSend(ch, v int8, pos *ast.Position, chanElemKind reflect.Kind) {
	fb.addPosAndPath(pos)
	fb.addOperandKinds(chanElemKind, 0, 0)
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpSend, A: v, C: ch})
}

// emitSetField appends a new "SetField" instruction to the function body.
//
//     s.field = v
//
func (fb *functionBuilder) emitSetField(k bool, s, field, v int8, fieldKind reflect.Kind) {
	fb.addOperandKinds(fieldKind, 0, 0)
	op := runtime.OpSetField
	if k {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: v, B: s, C: field})
}

// emitSetMap appends a new "SetMap" instruction to the function body.
//
//	m[key] = value
//
func (fb *functionBuilder) emitSetMap(k bool, m, value, key int8, mapType reflect.Type, pos *ast.Position) {
	keyType := mapType.Key()
	valueType := mapType.Elem()
	fb.addPosAndPath(pos)
	fb.addOperandKinds(valueType.Kind(), 0, keyType.Kind())
	fn := fb.fn
	op := runtime.OpSetMap
	if k {
		op = -op
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: op, A: value, B: m, C: key})
}

// emitSetSlice appends a new "SetSlice" instruction to the function body.
//
//	slice[index] = value
//
func (fb *functionBuilder) emitSetSlice(k bool, slice, value, index int8, pos *ast.Position, sliceElemKind reflect.Kind) {
	fb.addPosAndPath(pos)
	fb.addOperandKinds(sliceElemKind, 0, 0)
	in := runtime.Instruction{Op: runtime.OpSetSlice, A: value, B: slice, C: index}
	if k {
		in.Op = -in.Op
	}
	fb.fn.Body = append(fb.fn.Body, in)
}

// emitSetVar appends a new "SetVar" instruction to the function body.
//
//     v = r
//
func (fb *functionBuilder) emitSetVar(k bool, r int8, v int, dstKind reflect.Kind) {
	fb.addOperandKinds(dstKind, 0, 0)
	op := runtime.OpSetVar
	if k {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: r, B: int8(v >> 8), C: int8(v)})
}

// emitShl appends a new "Shl" instruction to the function body.
//
//     z = x << y
//
func (fb *functionBuilder) emitShl(k bool, x, y, z int8, kind reflect.Kind) {
	var op runtime.Operation
	switch kind {
	case reflect.Int:
		op = runtime.OpShlInt
	default:
		if z != x {
			panic(fmt.Errorf("z must be == x for kind %s", kind))
		}
		x = int8(flattenIntegerKind(kind))
		op = runtime.OpShl
	}
	if k {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitShow appends a new "Show" instruction to the function body.
//
//     show(type, value, ctx)
//
func (fb *functionBuilder) emitShow(typ reflect.Type, v int8, ctx ast.Context, inURL, isURLSet bool) {
	t := fb.addType(typ, true)
	c := encodeRenderContext(ctx, inURL, isURLSet)
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpShow, A: int8(t), B: v, C: int8(c)})
}

// emitShr appends a new "Shr" instruction to the function body.
//
//     z = x >> y
//
func (fb *functionBuilder) emitShr(k bool, x, y, z int8, kind reflect.Kind) {
	var op runtime.Operation
	switch kind {
	case reflect.Int:
		op = runtime.OpShrInt
	default:
		if z != x {
			panic(fmt.Errorf("z must be == x for kind %s", kind))
		}
		x = int8(flattenIntegerKind(kind))
		op = runtime.OpShr
	}
	if k {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitSlice appends a new "Slice" instruction to the function body.
//
//	slice[low:high:max]
//
func (fb *functionBuilder) emitSlice(klow, khigh, kmax bool, src, dst, low, high, max int8, pos *ast.Position) {
	fb.addPosAndPath(pos)
	fn := fb.fn
	var b int8
	if klow {
		b = 1
	}
	if khigh {
		b |= 2
	}
	if kmax {
		b |= 4
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpSlice, A: src, B: b, C: dst})
	fn.Body = append(fn.Body, runtime.Instruction{A: low, B: high, C: max})
}

// emitStringSlice appends a new "StringSlice" instruction to the function body.
//
//	string[low:high]
//
func (fb *functionBuilder) emitStringSlice(klow, khigh bool, src, dst, low, high int8, pos *ast.Position) {
	fb.addPosAndPath(pos)
	fn := fb.fn
	var b int8
	if klow {
		b = 1
	}
	if khigh {
		b |= 2
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpStringSlice, A: src, B: b, C: dst})
	fn.Body = append(fn.Body, runtime.Instruction{A: low, B: high})
}

// emitSub appends a new "Sub" instruction to the function body.
//
//     z = x - y
//
func (fb *functionBuilder) emitSub(k bool, x, y, z int8, kind reflect.Kind) {
	var op runtime.Operation
	switch kind {
	case reflect.Int:
		op = runtime.OpSubInt
	case reflect.Float64:
		op = runtime.OpSubFloat64
	default:
		if z != x {
			panic(fmt.Errorf("z must be == x for kind %s", kind))
		}
		x = int8(flattenIntegerKind(kind))
		op = runtime.OpSub
	}
	if k {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitSubInv appends a new "SubInv" instruction to the function body.
//
//     z = y - x
//
func (fb *functionBuilder) emitSubInv(k bool, x, y, z int8, kind reflect.Kind) {
	var op runtime.Operation
	switch kind {
	case reflect.Int:
		op = runtime.OpSubInvInt
	case reflect.Float64:
		op = runtime.OpSubInvFloat64
	default:
		if z != x {
			panic(fmt.Errorf("z must be == x for kind %s", kind))
		}
		x = int8(flattenIntegerKind(kind))
		op = runtime.OpSubInv
	}
	if k {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitText appends a new "Text" instruction to the function body.
//
//     text(txt, ctx)
//
func (fb *functionBuilder) emitText(txt []byte, inURL, isURLSet bool) {
	if len(fb.text.txt) > 0 {
		addr := fb.currentAddr()
		if addr == fb.text.addr+1 && inURL == fb.text.inURL {
			var hasLabel bool
			for _, la := range fb.labelAddrs {
				if addr == la {
					hasLabel = true
					break
				}
			}
			if !hasLabel {
				fb.text.txt = append(fb.text.txt, txt)
				return
			}
		}
		fb.flushText()
	}
	fb.text.addr = fb.currentAddr()
	fb.text.txt = append(fb.text.txt, txt)
	fb.text.inURL = inURL
	a, b := encodeUint16(uint16(len(fb.fn.Text)))
	var c int8
	if inURL {
		c = 1
		if isURLSet {
			c = 2
		}
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpText, A: a, B: b, C: c})
}

// emitTailCall appends a new "TailCall" instruction to the function body.
//
//     f()
//
func (fb *functionBuilder) emitTailCall(f int8, pos *ast.Position) {
	fb.addPosAndPath(pos)
	fn := fb.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpTailCall, A: f})
}

// emitTypify appends a new "Typify" instruction to the function body.
func (fb *functionBuilder) emitTypify(k bool, typ reflect.Type, x, z int8) {
	t := fb.addType(typ, true)
	op := runtime.OpTypify
	if k {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: int8(t), B: x, C: z})
}

// emitXor appends a new "Xor" instruction to the function body.
//
//     z = x ^ y
//
func (fb *functionBuilder) emitXor(k bool, x, y, z int8, kind reflect.Kind) {
	op := runtime.OpXor
	if k {
		op = -op
	}
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitZero appends a new "Zero" instruction to the function body.
func (fb *functionBuilder) emitZero(kind reflect.Kind, dst, src int8) {
	regType := int8(kindToType(kind))
	fb.fn.Body = append(fb.fn.Body, runtime.Instruction{Op: runtime.OpZero, A: regType, B: src, C: dst})
}
