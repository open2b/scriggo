// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"
	"strconv"

	"scriggo/ast"
	"scriggo/runtime"
)

// emitAdd appends a new "Add" instruction to the function body.
//
//     z = x + y
//
func (builder *functionBuilder) emitAddInt(k bool, x, y, z int8, kind reflect.Kind) {
	op := runtime.OpAdd
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

//
//    c = b + c
//
func (builder *functionBuilder) emitAddX(kb bool, b, c int8, kind reflect.Kind) {
	kind = flattenIntegerKind(kind)
	in := runtime.Instruction{
		Op: runtime.OpAddX,
		A:  int8(kind),
		B:  b,
		C:  c,
	}
	if kb {
		in.Op = -in.Op
	}
	builder.fn.Body = append(builder.fn.Body, in)
}

// emitAddr appends a new "Addr" instruction to the function body.
//
// 	   dest = &expr.Field
// 	   dest = &expr[index]
//
func (builder *functionBuilder) emitAddr(expr, index, dest int8, pos *ast.Position) {
	builder.addPosAndPath(pos)
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpAddr, A: expr, B: index, C: dest})
}

// emitAnd appends a new "And" instruction to the function body.
//
//     z = x & y
//
func (builder *functionBuilder) emitAnd(k bool, x, y, z int8, kind reflect.Kind) {
	op := runtime.OpAnd
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitAndNot appends a new "AndNot" instruction to the function body.
//
//     z = x &^ y
//
func (builder *functionBuilder) emitAndNot(k bool, x, y, z int8, kind reflect.Kind) {
	op := runtime.OpAndNot
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitAppend appends a new "Append" instruction to the function body.
//
func (builder *functionBuilder) emitAppend(start, end, s int8, elementsKind reflect.Kind) {
	builder.addOperandKinds(elementsKind, elementsKind, 0)
	fn := builder.fn
	if builder.allocs != nil {
		fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAlloc})
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAppend, A: start, B: end, C: s})
}

// emitAppendSlice appends a new "AppendSlice" instruction to the function body.
//
//     s = append(s, t)
//
func (builder *functionBuilder) emitAppendSlice(t, s int8, pos *ast.Position) {
	builder.addPosAndPath(pos)
	fn := builder.fn
	if builder.allocs != nil {
		fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAlloc})
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAppendSlice, A: t, C: s})
}

// emitAssert appends a new "assert" instruction to the function body.
//
//     z = e.(t)
//
func (builder *functionBuilder) emitAssert(e int8, typ reflect.Type, z int8) {
	t := builder.addType(typ, true)
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpAssert, A: e, B: int8(t), C: z})
}

// emitBreak appends a new "Break" instruction to the function body.
//
//     break addr
//
func (builder *functionBuilder) emitBreak(lab label) {
	addr := builder.labelAddrs[lab-1]
	if builder.allocs != nil {
		addr += 1
	}
	a, b, c := encodeUint24(uint32(addr))
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpBreak, A: a, B: b, C: c})
}

// emitCall appends a new "Call" instruction to the function body.
//
//     p.f()
//
func (builder *functionBuilder) emitCall(f int8, shift runtime.StackShift, pos *ast.Position) {
	builder.addPosAndPath(pos)
	fn := builder.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpCall, A: f})
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.Operation(shift[0]), A: shift[1], B: shift[2], C: shift[3]})
}

// emitCallPredefined appends a new "CallPredefined" instruction to the function body.
//
//     p.F()
//
func (builder *functionBuilder) emitCallPredefined(f int8, numVariadic int8, shift runtime.StackShift, pos *ast.Position) {
	builder.addPosAndPath(pos)
	fn := builder.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpCallPredefined, A: f, C: numVariadic})
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.Operation(shift[0]), A: shift[1], B: shift[2], C: shift[3]})
}

// emitCallIndirect appends a new "cCallIndirect" instruction to the function body.
//
//     f()
//
func (builder *functionBuilder) emitCallIndirect(f int8, numVariadic int8, shift runtime.StackShift, pos *ast.Position, funcType reflect.Type) {
	builder.addPosAndPath(pos)
	builder.addFunctionType(funcType)
	fn := builder.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpCallIndirect, A: f, C: numVariadic})
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.Operation(shift[0]), A: shift[1], B: shift[2], C: shift[3]})
}

// Assert appends a new "cap" instruction to the function body.
//
//     z = cap(s)
//
func (builder *functionBuilder) emitCap(s, z int8) {
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpCap, A: s, C: z})
}

// emitCase appends a new "Case" instruction to the function body.
//
//     case ch <- value
//     case value = <-ch
//     default
//
func (builder *functionBuilder) emitCase(kvalue bool, dir reflect.SelectDir, value, ch int8, kind reflect.Kind) {
	op := runtime.OpCase
	if kvalue {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: int8(dir), B: value, C: ch})
}

// emitClose appends a new "Close" instruction to the function body.
//
//     close(ch)
//
func (builder *functionBuilder) emitClose(ch int8, pos *ast.Position) {
	builder.addPosAndPath(pos)
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpClose, A: ch})
}

// emitComplex appends a new "Complex" instruction to the function body.
//
//	z = complex(x, y)
//
func (builder *functionBuilder) emitComplex(x, y, z int8, kind reflect.Kind) {
	op := runtime.OpComplex128
	if kind == reflect.Complex64 {
		op = runtime.OpComplex64
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitConcat appends a new "concat" instruction to the function body.
//
//     z = concat(s, t)
//
func (builder *functionBuilder) emitConcat(s, t, z int8) {
	fn := builder.fn
	if builder.allocs != nil {
		fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAlloc})
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpConcat, A: s, B: t, C: z})
}

// emitContinue appends a new "Continue" instruction to the function body.
//
//     continue label
//
func (builder *functionBuilder) emitContinue(lab label) {
	addr := builder.labelAddrs[lab-1]
	if builder.allocs != nil {
		addr += 1
	}
	a, b, c := encodeUint24(uint32(addr))
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpContinue, A: a, B: b, C: c})
}

// emitConvert appends a new "Convert" instruction to the function body.
//
// 	 dst = typ(src)
//
func (builder *functionBuilder) emitConvert(src int8, typ reflect.Type, dst int8, srcKind reflect.Kind) {
	fn := builder.fn
	regType := builder.addType(typ, false)
	var op runtime.Operation
	switch kindToType(srcKind) {
	case generalRegister:
		op = runtime.OpConvert
		if builder.allocs != nil {
			fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAlloc})
		}
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
		if builder.allocs != nil {
			fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAlloc})
		}
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
func (builder *functionBuilder) emitCopy(dst, src, n int8) {
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpCopy, A: src, B: n, C: dst})
}

// emitDefer appends a new "Defer" instruction to the function body.
//
//     defer
//
func (builder *functionBuilder) emitDefer(f int8, numVariadic int8, off, arg runtime.StackShift, funcType reflect.Type) {
	builder.addFunctionType(funcType)
	fn := builder.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpDefer, A: f, C: numVariadic})
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.Operation(off[0]), A: off[1], B: off[2], C: off[3]})
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.Operation(arg[0]), A: arg[1], B: arg[2], C: arg[3]})
}

// emitDelete appends a new "delete" instruction to the function body.
//
//     delete(m, k)
//
func (builder *functionBuilder) emitDelete(m, k int8) {
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpDelete, A: m, B: k})
}

// emitDiv appends a new "div" instruction to the function body.
//
//     z = x / y
//
func (builder *functionBuilder) emitDivInt(ky bool, x, y, z int8, kind reflect.Kind, pos *ast.Position) {

	if kind != reflect.Int {
		panic("BUG")
	}

	builder.addPosAndPath(pos)
	var op runtime.Operation
	switch kind {
	case reflect.Int:
		op = runtime.OpDivInt64
		if strconv.IntSize == 32 {
			op = runtime.OpDivInt32
		}
	case reflect.Int64:
		op = runtime.OpDivInt64
	case reflect.Int32:
		op = runtime.OpDivInt32
	case reflect.Int16:
		op = runtime.OpDivInt16
	case reflect.Int8:
		op = runtime.OpDivInt8
	case reflect.Uint, reflect.Uintptr:
		op = runtime.OpDivUint64
		if strconv.IntSize == 32 {
			op = runtime.OpDivUint32
		}
	case reflect.Uint64:
		op = runtime.OpDivUint64
	case reflect.Uint32:
		op = runtime.OpDivUint32
	case reflect.Uint16:
		op = runtime.OpDivUint16
	case reflect.Uint8:
		op = runtime.OpDivUint8
	case reflect.Float64:
		op = runtime.OpDivFloat64
	case reflect.Float32:
		op = runtime.OpDivFloat32
	default:
		panic(fmt.Errorf("BUG: invalid kind %s", kind))
	}
	if ky {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

//
//    c = b / c
//
func (builder *functionBuilder) emitDivX(kb bool, b, c int8, kind reflect.Kind, pos *ast.Position) {
	kind = flattenIntegerKind(kind)
	in := runtime.Instruction{
		Op: runtime.OpDivX,
		A:  int8(kind),
		B:  b,
		C:  c,
	}
	if kb {
		in.Op = -in.Op
	}
	builder.fn.Body = append(builder.fn.Body, in)
}

// emitRange appends a new "Range" instruction to the function body.
//
//	for i, e := range s
//
func (builder *functionBuilder) emitRange(k bool, s, i, e int8, kind reflect.Kind) {
	fn := builder.fn
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
	if builder.allocs != nil {
		fn.Body = append(fn.Body, runtime.Instruction{Op: -runtime.OpAlloc, C: 100})
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: op, A: s, B: i, C: e})
}

// emitField appends a new "Field" or a "FielRef" instruction to the function
// body. If ref is set then the result of the "field operation" is a reference
// to that field (i.e. is an addressable reflect.Value with the same underlying
// field value); otherwise the field is copied.
//
//  C = A.field
//
func (builder *functionBuilder) emitField(a, field, c int8, dstKind reflect.Kind, ref bool) {
	builder.addOperandKinds(0, 0, dstKind)
	op := runtime.OpField
	if ref {
		op = runtime.OpFieldRef
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: a, B: field, C: c})
}

// emitFunc appends a new "Func" instruction to the function body.
//
//     r = func() { ... }
//
func (builder *functionBuilder) emitFunc(r int8, typ reflect.Type) *runtime.Function {
	fn := builder.fn
	b := len(fn.Functions)
	if b == 256 {
		panic("Functions limit reached")
	}
	scriggoFunc := &runtime.Function{
		Type:   typ,
		Parent: fn,
	}
	fn.Functions = append(fn.Functions, scriggoFunc)
	if builder.allocs != nil {
		builder.allocs = append(builder.allocs, runtime.Addr(len(fn.Body)))
		fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAlloc})
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpLoadFunc, B: int8(b), C: r})
	return scriggoFunc
}

// emitGetVar appends a new "GetVar" instruction to the function body.
//
//     r = v
//
func (builder *functionBuilder) emitGetVar(v int, r int8, varKind reflect.Kind) {
	a, b := encodeInt16(int16(v))
	builder.addOperandKinds(0, 0, varKind)
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpGetVar, A: a, B: b, C: r})
}

// emitGetVarAddr appends a new "GetVarAddr" instruction to the function body.
//
//	   r = &v
//
func (builder *functionBuilder) emitGetVarAddr(v int, r int8) {
	a, b := encodeInt16(int16(v))
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpGetVarAddr, A: a, B: b, C: r})
}

// emitGo appends a new "Go" instruction to the function body.
//
//     go
//
func (builder *functionBuilder) emitGo() {
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpGo})
}

// emitGoto appends a new "goto" instruction to the function body.
//
//     goto label
//
func (builder *functionBuilder) emitGoto(lab label) {
	in := runtime.Instruction{Op: runtime.OpGoto}
	if lab > 0 {
		if lab > label(len(builder.labelAddrs)) {
			panic("BUG") // remove.
		}
		addr := builder.labelAddrs[lab-1]
		if addr == 0 {
			builder.gotos[builder.currentAddr()] = lab
		} else {
			in.A, in.B, in.C = encodeUint24(uint32(addr))
		}
	}
	builder.fn.Body = append(builder.fn.Body, in)
}

// emitIf appends a new "If" instruction to the function body.
//
//     x
//     !x
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
//
func (builder *functionBuilder) emitIf(k bool, x int8, o runtime.Condition, y int8, kind reflect.Kind, pos *ast.Position) {
	builder.addPosAndPath(pos)
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
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: int8(o), C: y})
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
func (builder *functionBuilder) emitIndex(ki bool, expr, i, dst int8, exprType reflect.Type, pos *ast.Position, ref bool) {
	builder.addPosAndPath(pos)
	builder.addOperandKinds(0, 0, exprType.Kind())
	fn := builder.fn
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
		if builder.allocs != nil {
			fn.Body = append(fn.Body, runtime.Instruction{Op: -runtime.OpAlloc, C: 8})
		}
	default:
		panic(fmt.Errorf("BUG: invalid type %s", exprType))
	}
	if ki {
		op = -op
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: op, A: expr, B: i, C: dst})
}

// emitLeftShift appends a new "LeftShift" instruction to the function body.
//
//     z = x << y
//
func (builder *functionBuilder) emitLeftShiftInt(k bool, x, y, z int8, kind reflect.Kind) {
	var op runtime.Operation
	switch kind {
	case reflect.Int, reflect.Uint, reflect.Uintptr:
		op = runtime.OpLeftShift64
		if strconv.IntSize == 32 {
			op = runtime.OpLeftShift32
		}
	case reflect.Int8, reflect.Uint8:
		op = runtime.OpLeftShift8
	case reflect.Int16, reflect.Uint16:
		op = runtime.OpLeftShift16
	case reflect.Int32, reflect.Uint32:
		op = runtime.OpLeftShift32
	case reflect.Int64, reflect.Uint64:
		op = runtime.OpLeftShift64
	}
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

//
//    c = b << c
//
func (builder *functionBuilder) emitLeftShiftX(kb bool, b, c int8, kind reflect.Kind) {
	kind = flattenIntegerKind(kind)
	in := runtime.Instruction{
		Op: runtime.OpLeftShiftX,
		A:  int8(kind),
		B:  b,
		C:  c,
	}
	if kb {
		in.Op = -in.Op
	}
	builder.fn.Body = append(builder.fn.Body, in)
}

// emitLen appends a new "len" instruction to the function body.
//
//     l = len(s)
//
func (builder *functionBuilder) emitLen(s, l int8, t reflect.Type) {
	var a int8
	if t.Kind() != reflect.String {
		a = 1
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpLen, A: a, B: s, C: l})
}

// Load data appends a new "LoadData" instruction to the function body.
func (builder *functionBuilder) emitLoadData(i int16, dst int8) {
	a, b := encodeInt16(i)
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpLoadData, A: a, B: b, C: dst})
}

// emitLoadFunc appends a new "LoadFunc" instruction to the function body.
//
//     z = p.f
//
func (builder *functionBuilder) emitLoadFunc(predefined bool, f int8, z int8) {
	fn := builder.fn
	var a int8
	if predefined {
		a = 1
	}
	if builder.allocs != nil {
		fn.Body = append(fn.Body, runtime.Instruction{Op: -runtime.OpAlloc, C: 32})
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpLoadFunc, A: a, B: f, C: z})
}

// emitLoadNumber appends a new "LoadNumber" instruction to the function body.
//
func (builder *functionBuilder) emitLoadNumber(typ registerType, index, dst int8) {
	var a int8
	switch typ {
	case intRegister:
		a = 0
	case floatRegister:
		a = 1
	default:
		panic("LoadNumber only accepts intRegister or floatRegister as type")
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpLoadNumber, A: a, B: index, C: dst})
}

// emitMakeChan appends a new "MakeChan" instruction to the function body.
//
//     dst = make(typ, capacity)
//
func (builder *functionBuilder) emitMakeChan(typ reflect.Type, kCapacity bool, capacity int8, dst int8, pos *ast.Position) {
	builder.addPosAndPath(pos)
	fn := builder.fn
	t := builder.addType(typ, false)
	op := runtime.OpMakeChan
	if kCapacity {
		op = -op
	}
	if builder.allocs != nil {
		constantAlloc := false
		if kCapacity {
			size := int(typ.Size())
			bytes := size * int(capacity)
			if bytes/size != int(capacity) {
				panic("out of memory")
			}
			bytes += 10 * 8
			if bytes < 0 {
				panic("out of memory")
			}
			if bytes <= maxUint24 {
				a, b, c := encodeUint24(uint32(bytes))
				fn.Body = append(fn.Body, runtime.Instruction{Op: -runtime.OpAlloc, A: a, B: b, C: c})
				constantAlloc = true
			}
		}
		if !constantAlloc {
			fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAlloc})
		}
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: op, A: int8(t), B: capacity, C: dst})
}

// emitMakeMap appends a new "MakeMap" instruction to the function body.
//
//     dst = make(typ, size)
//
func (builder *functionBuilder) emitMakeMap(typ reflect.Type, kSize bool, size int8, dst int8) {
	fn := builder.fn
	t := builder.addType(typ, false)
	op := runtime.OpMakeMap
	if kSize {
		op = -op
	}
	if builder.allocs != nil {
		if kSize {
			bytes := 24 + 50*int(size)
			a, b, c := encodeUint24(uint32(bytes))
			fn.Body = append(fn.Body, runtime.Instruction{Op: -runtime.OpAlloc, A: a, B: b, C: c})
		} else {
			fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAlloc})
		}
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: op, A: int8(t), B: size, C: dst})
}

// emitMakeSlice appends a new "MakeSlice" instruction to the function body.
//
//     make(sliceType, len, cap)
//
func (builder *functionBuilder) emitMakeSlice(kLen, kCap bool, sliceType reflect.Type, len, cap, dst int8, pos *ast.Position) {
	builder.addPosAndPath(pos)
	fn := builder.fn
	t := builder.addType(sliceType, false)
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
	if builder.allocs != nil {
		constantAlloc := false
		if kCap {
			ts := int(sliceType.Elem().Size())
			bytes := ts * int(cap)
			if bytes/ts != int(cap) {
				panic("out of memory")
			}
			bytes += 24
			if bytes < 0 {
				panic("out of memory")
			}
			if bytes <= maxUint24 {
				a, b, c := encodeUint24(uint32(bytes))
				fn.Body = append(fn.Body, runtime.Instruction{Op: -runtime.OpAlloc, A: a, B: b, C: c})
				constantAlloc = true
			}
		}
		if !constantAlloc {
			fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAlloc})
		}
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpMakeSlice, A: int8(t), B: k, C: dst})
	if k > 0 {
		fn.Body = append(fn.Body, runtime.Instruction{A: len, B: cap})
	}
}

// emitMethodValue appends a new "MethodValue" instruction to the function body.
//
//     dst = receiver.name
//
func (builder *functionBuilder) emitMethodValue(name string, receiver int8, dst int8) {
	str := builder.makeStringConstant(name)
	fn := builder.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpMethodValue, A: receiver, B: str, C: dst})
}

// emitMove appends a new "Move" instruction to the function body.
//
//     z = x
//
// TODO: copy should be true only when necessary, otherwise it's just a waste of
// resources. Check every call to emitMove from the emitter and set the 'copy'
// argument correctly.
func (builder *functionBuilder) emitMove(k bool, x, z int8, kind reflect.Kind, copy bool) {
	op := runtime.OpMove
	if k {
		op = -op
	}
	a := int8(kindToType(kind))
	if copy {
		// TODO: enable this check..
		//
		// if kind != reflect.Array && kind != reflect.Struct {
		// 	panic(fmt.Errorf("BUG: emitMove: cannot set copy = true with kind %s, expected kind array or struct", kind.String()))
		// }
		// .. and remove this if:
		if kind == reflect.Array || kind == reflect.Struct {
			a = int8(-generalRegister)
		}
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: a, B: x, C: z})
}

// emitMul appends a new "mul" instruction to the function body.
//
//     z = x * y
//
func (builder *functionBuilder) emitMulInt(ky bool, x, y, z int8, kind reflect.Kind) {

	if kind != reflect.Int {
		panic("BUG")
	}

	var op runtime.Operation
	switch kind {
	case reflect.Int, reflect.Uint, reflect.Uintptr:
		op = runtime.OpMulInt64
		if strconv.IntSize == 32 {
			op = runtime.OpMulInt32
		}
	case reflect.Int64, reflect.Uint64:
		op = runtime.OpMulInt64
	case reflect.Int32, reflect.Uint32:
		op = runtime.OpMulInt32
	case reflect.Int16, reflect.Uint16:
		op = runtime.OpMulInt16
	case reflect.Int8, reflect.Uint8:
		op = runtime.OpMulInt8
	case reflect.Float64:
		op = runtime.OpMulFloat64
	case reflect.Float32:
		op = runtime.OpMulFloat32
	default:
		panic("mul: invalid type")
	}
	if ky {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

//
//    c = b * c
//
func (builder *functionBuilder) emitMulX(kb bool, b, c int8, kind reflect.Kind) {
	kind = flattenIntegerKind(kind)
	in := runtime.Instruction{
		Op: runtime.OpMulX,
		A:  int8(kind),
		B:  b,
		C:  c,
	}
	if kb {
		in.Op = -in.Op
	}
	builder.fn.Body = append(builder.fn.Body, in)
}

// emitNew appends a new "new" instruction to the function body.
//
//     z = new(t)
//
func (builder *functionBuilder) emitNew(typ reflect.Type, z int8) {
	fn := builder.fn
	b := builder.addType(typ, false)
	if builder.allocs != nil {
		bytes := int(typ.Size())
		if bytes <= maxUint24 {
			a, b, c := encodeUint24(uint32(bytes))
			fn.Body = append(fn.Body, runtime.Instruction{Op: -runtime.OpAlloc, A: a, B: b, C: c})
		} else {
			fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAlloc})
		}
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpNew, B: int8(b), C: z})
}

// emitNop appends a new "Nop" instruction to the function body.
//
func (builder *functionBuilder) emitNop() {
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpNone})
}

// emitOr appends a new "Or" instruction to the function body.
//
//     z = x | y
//
func (builder *functionBuilder) emitOr(k bool, x, y, z int8, kind reflect.Kind) {
	op := runtime.OpOr
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitPanic appends a new "Panic" instruction to the function body. If the
// panic follows a type assertion, typ should contain the type of the expression
// of the type assertion, in order to provide additional information to the VM;
// otherwise typ should be nil.
//
//     panic(v)
//
func (builder *functionBuilder) emitPanic(v int8, typ reflect.Type, pos *ast.Position) {
	builder.addPosAndPath(pos)
	fn := builder.fn
	in := runtime.Instruction{Op: runtime.OpPanic, A: v}
	if typ != nil {
		in.C = int8(builder.addType(typ, true))
	}
	fn.Body = append(fn.Body, in)
}

// emitPrint appends a new "Print" instruction to the function body.
//
//     print(arg)
//
func (builder *functionBuilder) emitPrint(arg int8) {
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpPrint, A: arg})
}

// emitRealImag appends a new "RealImag" instruction to the function body.
//
//	y, z = real(x), imag(x)
//
func (builder *functionBuilder) emitRealImag(k bool, x, y, z int8) {
	op := runtime.OpRealImag
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

// emitReceive appends a new "Receive" instruction to the function body.
//
//	dst = <- ch
//
//	dst, ok = <- ch
//
func (builder *functionBuilder) emitReceive(ch, ok, dst int8) {
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpReceive, A: ch, B: ok, C: dst})
}

// emitRecover appends a new "Recover" instruction to the function body.
//
//     recover()
//     defer recover()
//
func (builder *functionBuilder) emitRecover(r int8, down bool) {
	var a int8
	if down {
		// Recover down the stack.
		a = 1
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpRecover, A: a, C: r})
}

// emitRem appends a new "rem" instruction to the function body.
//
//     z = x % y
//
func (builder *functionBuilder) emitRemInt(ky bool, x, y, z int8, kind reflect.Kind, pos *ast.Position) {

	if kind != reflect.Int {
		panic("BUG")
	}

	builder.addPosAndPath(pos)
	var op runtime.Operation
	switch kind {
	case reflect.Int:
		op = runtime.OpRemInt64
		if strconv.IntSize == 32 {
			op = runtime.OpRemInt32
		}
	case reflect.Int64:
		op = runtime.OpRemInt64
	case reflect.Int32:
		op = runtime.OpRemInt32
	case reflect.Int16:
		op = runtime.OpRemInt16
	case reflect.Int8:
		op = runtime.OpRemInt8
	case reflect.Uint, reflect.Uintptr:
		op = runtime.OpRemUint64
		if strconv.IntSize == 32 {
			op = runtime.OpRemUint32
		}
	case reflect.Uint64:
		op = runtime.OpRemUint64
	case reflect.Uint32:
		op = runtime.OpRemUint32
	case reflect.Uint16:
		op = runtime.OpRemUint16
	case reflect.Uint8:
		op = runtime.OpRemUint8
	default:
		panic("rem: invalid type")
	}
	if ky {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

//
//    c = b % c
//
func (builder *functionBuilder) emitRemX(kb bool, b, c int8, kind reflect.Kind, pos *ast.Position) {
	kind = flattenIntegerKind(kind)
	in := runtime.Instruction{
		Op: runtime.OpRemX,
		A:  int8(kind),
		B:  b,
		C:  c,
	}
	if kb {
		in.Op = -in.Op
	}
	builder.fn.Body = append(builder.fn.Body, in)
}

// emitReturn appends a new "return" instruction to the function body.
//
//     return
//
func (builder *functionBuilder) emitReturn() {
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpReturn})
}

// emitRightShift appends a new "RightShift" instruction to the function body.
//
//     z = x >> y
//
func (builder *functionBuilder) emitRightShiftInt(k bool, x, y, z int8, kind reflect.Kind) {

	if kind != reflect.Int {
		panic("BUG")
	}

	op := runtime.OpRightShift
	switch kind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		op = runtime.OpRightShiftU
	}
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

//
//    c = b >> c
//
func (builder *functionBuilder) emitRightShiftX(kb bool, b, c int8, kind reflect.Kind) {
	kind = flattenIntegerKind(kind)
	in := runtime.Instruction{
		Op: runtime.OpRightShiftX,
		A:  int8(kind),
		B:  b,
		C:  c,
	}
	if kb {
		in.Op = -in.Op
	}
	builder.fn.Body = append(builder.fn.Body, in)
}

// emitSelect appends a new "Select" instruction to the function body.
//
//     select
//
func (builder *functionBuilder) emitSelect() {
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpSelect})
}

// emitSend appends a new "Send" instruction to the function body.
//
//	ch <- v
//
func (builder *functionBuilder) emitSend(ch, v int8, pos *ast.Position, chanElemKind reflect.Kind) {
	builder.addPosAndPath(pos)
	builder.addOperandKinds(chanElemKind, 0, 0)
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpSend, A: v, C: ch})
}

// emitSetAlloc sets the alloc property. If true, an Alloc instruction will be
// inserted where necessary.
func (builder *functionBuilder) emitSetAlloc(alloc bool) {
	if alloc && builder.allocs == nil {
		builder.allocs = append(builder.allocs, 0)
		builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: runtime.OpAlloc})
	}
}

// emitSetField appends a new "SetField" instruction to the function body.
//
//     s.field = v
//
func (builder *functionBuilder) emitSetField(k bool, s, field, v int8, fieldKind reflect.Kind) {
	builder.addOperandKinds(fieldKind, 0, 0)
	op := runtime.OpSetField
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: v, B: s, C: field})
}

// emitSetVar appends a new "SetVar" instruction to the function body.
//
//     v = r
//
func (builder *functionBuilder) emitSetVar(k bool, r int8, v int, dstKind reflect.Kind) {
	builder.addOperandKinds(dstKind, 0, 0)
	op := runtime.OpSetVar
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: r, B: int8(v >> 8), C: int8(v)})
}

// emitSetMap appends a new "SetMap" instruction to the function body.
//
//	m[key] = value
//
func (builder *functionBuilder) emitSetMap(k bool, m, value, key int8, mapType reflect.Type, pos *ast.Position) {
	keyType := mapType.Key()
	valueType := mapType.Elem()
	builder.addPosAndPath(pos)
	builder.addOperandKinds(valueType.Kind(), 0, keyType.Kind())
	fn := builder.fn
	op := runtime.OpSetMap
	if k {
		op = -op
	}
	if builder.allocs != nil {
		kSize := int(keyType.Size())
		eSize := int(valueType.Size())
		bytes := kSize + eSize
		if bytes < 0 {
			panic("out of memory")
		}
		if bytes <= maxUint24 {
			a, b, c := encodeUint24(uint32(bytes))
			fn.Body = append(fn.Body, runtime.Instruction{Op: -runtime.OpAlloc, A: a, B: b, C: c})
		} else {
			fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpAlloc})
		}
	}
	fn.Body = append(fn.Body, runtime.Instruction{Op: op, A: value, B: m, C: key})
}

// emitSetSlice appends a new "SetSlice" instruction to the function body.
//
//	slice[index] = value
//
func (builder *functionBuilder) emitSetSlice(k bool, slice, value, index int8, pos *ast.Position, sliceElemKind reflect.Kind) {
	builder.addPosAndPath(pos)
	builder.addOperandKinds(sliceElemKind, 0, 0)
	in := runtime.Instruction{Op: runtime.OpSetSlice, A: value, B: slice, C: index}
	if k {
		in.Op = -in.Op
	}
	builder.fn.Body = append(builder.fn.Body, in)
}

// emitSlice appends a new "Slice" instruction to the function body.
//
//	slice[low:high:max]
//
func (builder *functionBuilder) emitSlice(klow, khigh, kmax bool, src, dst, low, high, max int8, pos *ast.Position) {
	builder.addPosAndPath(pos)
	fn := builder.fn
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
func (builder *functionBuilder) emitStringSlice(klow, khigh bool, src, dst, low, high int8, pos *ast.Position) {
	builder.addPosAndPath(pos)
	fn := builder.fn
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
func (builder *functionBuilder) emitSubInt(k bool, x, y, z int8, kind reflect.Kind) {
	op := runtime.OpSub
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

//
//    c = b - c
//
func (builder *functionBuilder) emitSubX(kb bool, b, c int8, kind reflect.Kind) {
	kind = flattenIntegerKind(kind)
	in := runtime.Instruction{
		Op: runtime.OpSubX,
		A:  int8(kind),
		B:  b,
		C:  c,
	}
	if kb {
		in.Op = -in.Op
	}
	builder.fn.Body = append(builder.fn.Body, in)
}

// emitSubInv appends a new "SubInv" instruction to the function body.
//
//     z = y - x
//
func (builder *functionBuilder) emitSubInvInt(k bool, x, y, z int8, kind reflect.Kind) {
	op := runtime.OpSubInv
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}

func (builder *functionBuilder) emitSubInvX(ka bool, b, c int8, kind reflect.Kind) {
	kind = flattenIntegerKind(kind)
	in := runtime.Instruction{
		Op: runtime.OpSubInvX,
		A:  int8(kind),
		B:  b,
		C:  c,
	}
	if ka {
		in.Op = -in.Op
	}
	builder.fn.Body = append(builder.fn.Body, in)
}

// emitTypify appends a new "Typify" instruction to the function body.
func (builder *functionBuilder) emitTypify(k bool, typ reflect.Type, x, z int8) {
	t := builder.addType(typ, true)
	op := runtime.OpTypify
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: int8(t), B: x, C: z})
}

// emitTailCall appends a new "TailCall" instruction to the function body.
//
//     f()
//
func (builder *functionBuilder) emitTailCall(f int8, pos *ast.Position) {
	builder.addPosAndPath(pos)
	fn := builder.fn
	fn.Body = append(fn.Body, runtime.Instruction{Op: runtime.OpTailCall, A: f})
}

// emitXor appends a new "Xor" instruction to the function body.
//
//     z = x ^ y
//
func (builder *functionBuilder) emitXor(k bool, x, y, z int8, kind reflect.Kind) {
	op := runtime.OpXor
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, runtime.Instruction{Op: op, A: x, B: y, C: z})
}
