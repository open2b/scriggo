// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"scriggo/vm"
	"strconv"
)

// Add appends a new "add" instruction to the function body.
//
//     z = x + y
//
func (builder *functionBuilder) Add(k bool, x, y, z int8, kind reflect.Kind) {
	var op vm.Operation
	switch kind {
	case reflect.Int, reflect.Uint, reflect.Uintptr:
		op = vm.OpAddInt64
		if strconv.IntSize == 32 {
			op = vm.OpAddInt32
		}
	case reflect.Int64, reflect.Uint64:
		op = vm.OpAddInt64
	case reflect.Int32, reflect.Uint32:
		op = vm.OpAddInt32
	case reflect.Int16, reflect.Uint16:
		op = vm.OpAddInt16
	case reflect.Int8, reflect.Uint8:
		op = vm.OpAddInt8
	case reflect.Float64:
		op = vm.OpAddFloat64
	case reflect.Float32:
		op = vm.OpAddFloat32
	default:
		panic("add: invalid type")
	}
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: x, B: y, C: z})
}

// And appends a new "And" instruction to the function body.
//
//     z = x & y
//
func (builder *functionBuilder) And(k bool, x, y, z int8, kind reflect.Kind) {
	op := vm.OpAnd
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: x, B: y, C: z})
}

// AndNot appends a new "AndNot" instruction to the function body.
//
//     z = x &^ y
//
func (builder *functionBuilder) AndNot(k bool, x, y, z int8, kind reflect.Kind) {
	op := vm.OpAndNot
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: x, B: y, C: z})
}

// Append appends a new "Append" instruction to the function body.
//
//     s = append(s, regs[first:first+length]...)
//
func (builder *functionBuilder) Append(first, length, s int8) {
	fn := builder.fn
	if builder.allocs != nil {
		fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpAlloc})
	}
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpAppend, A: first, B: length, C: s})
}

// AppendSlice appends a new "AppendSlice" instruction to the function body.
//
//     s = append(s, t)
//
func (builder *functionBuilder) AppendSlice(t, s int8) {
	fn := builder.fn
	if builder.allocs != nil {
		fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpAlloc})
	}
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpAppendSlice, A: t, C: s})
}

// Assert appends a new "assert" instruction to the function body.
//
//     z = e.(t)
//
func (builder *functionBuilder) Assert(e int8, typ reflect.Type, z int8) {
	index := -1
	for i, t := range builder.fn.Types {
		if t == typ {
			index = i
			break
		}
	}
	if index == -1 {
		index = len(builder.fn.Types)
		if index > 255 {
			panic("types limit reached")
		}
		builder.fn.Types = append(builder.fn.Types, typ)
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpAssert, A: e, B: int8(index), C: z})
}

// Bind appends a new "Bind" instruction to the function body.
//
//     r = v
//
func (builder *functionBuilder) Bind(v int, r int8) {
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpBind, A: int8(v >> 8), B: int8(v), C: r})
}

// Break appends a new "Break" instruction to the function body.
//
//     break addr
//
func (builder *functionBuilder) Break(label uint32) {
	addr := builder.labels[label-1]
	if builder.allocs != nil {
		addr += 1
	}
	a, b, c := encodeUint24(addr)
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpBreak, A: a, B: b, C: c})
}

// Call appends a new "Call" instruction to the function body.
//
//     p.f()
//
func (builder *functionBuilder) Call(f int8, shift vm.StackShift, line int) {
	fn := builder.fn
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpCall, A: f})
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.Operation(shift[0]), A: shift[1], B: shift[2], C: shift[3]})
	builder.addLine(uint32(len(fn.Body)-2), line)
}

// CallPredefined appends a new "CallPredefined" instruction to the function body.
//
//     p.F()
//
func (builder *functionBuilder) CallPredefined(f int8, numVariadic int8, shift vm.StackShift) {
	fn := builder.fn
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpCallPredefined, A: f, C: numVariadic})
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.Operation(shift[0]), A: shift[1], B: shift[2], C: shift[3]})
}

// CallIndirect appends a new "CallIndirect" instruction to the function body.
//
//     f()
//
func (builder *functionBuilder) CallIndirect(f int8, numVariadic int8, shift vm.StackShift) {
	fn := builder.fn
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpCallIndirect, A: f, C: numVariadic})
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.Operation(shift[0]), A: shift[1], B: shift[2], C: shift[3]})
}

// Assert appends a new "cap" instruction to the function body.
//
//     z = cap(s)
//
func (builder *functionBuilder) Cap(s, z int8) {
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpCap, A: s, C: z})
}

// Case appends a new "Case" instruction to the function body.
//
//     case ch <- value
//     case value = <-ch
//     default
//
func (builder *functionBuilder) Case(kvalue bool, dir reflect.SelectDir, value, ch int8, kind reflect.Kind) {
	op := vm.OpCase
	if kvalue {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: int8(dir), B: value, C: ch})
}

// Close appends a new "Close" instruction to the function body.
//
//     close(ch)
//
func (builder *functionBuilder) Close(ch int8) {
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpClose, A: ch})
}

// Concat appends a new "concat" instruction to the function body.
//
//     z = concat(s, t)
//
func (builder *functionBuilder) Concat(s, t, z int8) {
	fn := builder.fn
	if builder.allocs != nil {
		fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpAlloc})
	}
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpConcat, A: s, B: t, C: z})
}

// Continue appends a new "Continue" instruction to the function body.
//
//     continue addr
//
func (builder *functionBuilder) Continue(label uint32) {
	addr := builder.labels[label-1]
	if builder.allocs != nil {
		addr += 1
	}
	a, b, c := encodeUint24(addr)
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpContinue, A: a, B: b, C: c})
}

// Convert appends a new "Convert" instruction to the function body.
//
// 	 dst = typ(src)
//
func (builder *functionBuilder) Convert(src int8, typ reflect.Type, dst int8, srcKind reflect.Kind) {
	fn := builder.fn
	regType := builder.addType(typ)
	var op vm.Operation
	switch kindToType(srcKind) {
	case vm.TypeGeneral:
		op = vm.OpConvertGeneral
		if builder.allocs != nil {
			fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpAlloc})
		}
	case vm.TypeInt:
		switch srcKind {
		case reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64,
			reflect.Uintptr:
			op = vm.OpConvertUint
		default:
			op = vm.OpConvertInt
		}
	case vm.TypeString:
		op = vm.OpConvertString
		if builder.allocs != nil {
			fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpAlloc})
		}
	case vm.TypeFloat:
		op = vm.OpConvertFloat
	}
	fn.Body = append(fn.Body, vm.Instruction{Op: op, A: src, B: regType, C: dst})
}

// Copy appends a new "Copy" instruction to the function body.
//
//     n == 0:   copy(dst, src)
// 	 n != 0:   n := copy(dst, src)
//
func (builder *functionBuilder) Copy(dst, src, n int8) {
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpCopy, A: src, B: n, C: dst})
}

// Defer appends a new "Defer" instruction to the function body.
//
//     defer
//
func (builder *functionBuilder) Defer(f int8, numVariadic int8, off, arg vm.StackShift) {
	fn := builder.fn
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpDefer, A: f, C: numVariadic})
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.Operation(off[0]), A: off[1], B: off[2], C: off[3]})
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.Operation(arg[0]), A: arg[1], B: arg[2], C: arg[3]})
}

// Delete appends a new "delete" instruction to the function body.
//
//     delete(m, k)
//
func (builder *functionBuilder) Delete(m, k int8) {
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpDelete, A: m, B: k})
}

// Div appends a new "div" instruction to the function body.
//
//     z = x / y
//
func (builder *functionBuilder) Div(ky bool, x, y, z int8, kind reflect.Kind) {
	var op vm.Operation
	switch kind {
	case reflect.Int:
		op = vm.OpDivInt64
		if strconv.IntSize == 32 {
			op = vm.OpDivInt32
		}
	case reflect.Int64:
		op = vm.OpDivInt64
	case reflect.Int32:
		op = vm.OpDivInt32
	case reflect.Int16:
		op = vm.OpDivInt16
	case reflect.Int8:
		op = vm.OpDivInt8
	case reflect.Uint, reflect.Uintptr:
		op = vm.OpDivUint64
		if strconv.IntSize == 32 {
			op = vm.OpDivUint32
		}
	case reflect.Uint64:
		op = vm.OpDivUint64
	case reflect.Uint32:
		op = vm.OpDivUint32
	case reflect.Uint16:
		op = vm.OpDivUint16
	case reflect.Uint8:
		op = vm.OpDivUint8
	case reflect.Float64:
		op = vm.OpDivFloat64
	case reflect.Float32:
		op = vm.OpDivFloat32
	default:
		panic("div: invalid type")
	}
	if ky {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: x, B: y, C: z})
}

// Range appends a new "Range" instruction to the function body.
//
//	for i, e := range s
//
func (builder *functionBuilder) Range(k bool, s, i, e int8, kind reflect.Kind) {
	fn := builder.fn
	var op vm.Operation
	switch kind {
	case reflect.String:
		op = vm.OpRangeString
		if k {
			op = -op
		}
	default:
		if k {
			// TODO(Gianluca):  if Range is called with k == true but kind
			// is not reflect.String, s (which is a constant) is threated
			// as non-constant by builder and VM.
			panic("bug!")
		}
		op = vm.OpRange
	}
	if builder.allocs != nil {
		fn.Body = append(fn.Body, vm.Instruction{Op: -vm.OpAlloc, C: 100})
	}
	fn.Body = append(fn.Body, vm.Instruction{Op: op, A: s, B: i, C: e})
}

// Field appends a new "Field" instruction to the function body.
//
// 	C = A.field
//
func (builder *functionBuilder) Field(a, field, c int8) {
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpField, A: a, B: field, C: c})
}

// Func appends a new "Func" instruction to the function body.
//
//     r = func() { ... }
//
func (builder *functionBuilder) Func(r int8, typ reflect.Type) *vm.Function {
	fn := builder.fn
	b := len(fn.Literals)
	if b == 256 {
		panic("Functions limit reached")
	}
	scriggoFunc := &vm.Function{
		Type:   typ,
		Parent: fn,
	}
	fn.Literals = append(fn.Literals, scriggoFunc)
	if builder.allocs != nil {
		builder.allocs = append(builder.allocs, uint32(len(fn.Body)))
		fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpAlloc})
	}
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpFunc, B: int8(b), C: r})
	return scriggoFunc
}

// GetFunc appends a new "GetFunc" instruction to the function body.
//
//     z = p.f
//
func (builder *functionBuilder) GetFunc(predefined bool, f int8, z int8) {
	fn := builder.fn
	var a int8
	if predefined {
		a = 1
	}
	if builder.allocs != nil {
		fn.Body = append(fn.Body, vm.Instruction{Op: -vm.OpAlloc, C: 32})
	}
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpGetFunc, A: a, B: f, C: z})
}

// GetVar appends a new "GetVar" instruction to the function body.
//
//     r = v
//
func (builder *functionBuilder) GetVar(v int, r int8) {
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpGetVar, A: int8(v >> 8), B: int8(v), C: r})
}

// Go appends a new "Go" instruction to the function body.
//
//     go
//
func (builder *functionBuilder) Go() {
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpGo})
}

// Goto appends a new "goto" instruction to the function body.
//
//     goto label
//
func (builder *functionBuilder) Goto(label uint32) {
	in := vm.Instruction{Op: vm.OpGoto}
	if label > 0 {
		if label > uint32(len(builder.labels)) {
			panic("bug!") // TODO(Gianluca): remove.
		}
		addr := builder.labels[label-1]
		if addr == 0 {
			builder.gotos[builder.currentAddr()] = label
		} else {
			in.A, in.B, in.C = encodeUint24(addr)
		}
	}
	builder.fn.Body = append(builder.fn.Body, in)
}

// If appends a new "If" instruction to the function body.
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
func (builder *functionBuilder) If(k bool, x int8, o vm.Condition, y int8, kind reflect.Kind) {
	var op vm.Operation
	switch kindToType(kind) {
	case vm.TypeInt:
		op = vm.OpIfInt
	case vm.TypeFloat:
		op = vm.OpIfFloat
	case vm.TypeString:
		op = vm.OpIfString
	case vm.TypeGeneral:
		op = vm.OpIf
	}
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: x, B: int8(o), C: y})
}

// Index appends a new "index" instruction to the function body
//
//	dst = expr[i]
//
func (builder *functionBuilder) Index(ki bool, expr, i, dst int8, exprType reflect.Type) {
	fn := builder.fn
	kind := exprType.Kind()
	var op vm.Operation
	switch kind {
	default:
		op = vm.OpIndex
	case reflect.Slice:
		op = vm.OpSliceIndex
	case reflect.String:
		op = vm.OpStringIndex
		if builder.allocs != nil {
			fn.Body = append(fn.Body, vm.Instruction{Op: -vm.OpAlloc, C: 8})
		}
	case reflect.Map:
		op = vm.OpMapIndex
	}
	if ki {
		op = -op
	}
	fn.Body = append(fn.Body, vm.Instruction{Op: op, A: expr, B: i, C: dst})
}

// LeftShift appends a new "LeftShift" instruction to the function body.
//
//     z = x << y
//
func (builder *functionBuilder) LeftShift(k bool, x, y, z int8, kind reflect.Kind) {
	var op vm.Operation
	switch kind {
	case reflect.Int, reflect.Uint, reflect.Uintptr:
		op = vm.OpLeftShift64
		if strconv.IntSize == 32 {
			op = vm.OpLeftShift32
		}
	case reflect.Int8, reflect.Uint8:
		op = vm.OpLeftShift8
	case reflect.Int16, reflect.Uint16:
		op = vm.OpLeftShift16
	case reflect.Int32, reflect.Uint32:
		op = vm.OpLeftShift32
	case reflect.Int64, reflect.Uint64:
		op = vm.OpLeftShift64
	}
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: x, B: y, C: z})
}

// Len appends a new "len" instruction to the function body.
//
//     l = len(s)
//
func (builder *functionBuilder) Len(s, l int8, t reflect.Type) {
	var a int8
	switch t {
	case reflect.TypeOf(""):
		// TODO(Gianluca): this case catches string types only, not defined
		// types with underlying type string.
		a = 0
	default:
		a = 1
	case reflect.TypeOf([]byte{}):
		a = 2
	case reflect.TypeOf([]string{}):
		a = 4
	case reflect.TypeOf([]interface{}{}):
		a = 5
	case reflect.TypeOf(map[string]string{}):
		a = 6
	case reflect.TypeOf(map[string]int{}):
		a = 7
	case reflect.TypeOf(map[string]interface{}{}):
		a = 8
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpLen, A: a, B: s, C: l})
}

// Load data appends a new "LoadData" instruction to the function body.
func (builder *functionBuilder) LoadData(i int16, dst int8) {
	a, b := encodeInt16(i)
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpLoadData, A: a, B: b, C: dst})
}

// LoadNumber appends a new "LoadNumber" instruction to the function body.
//
func (builder *functionBuilder) LoadNumber(typ vm.Type, index, dst int8) {
	var a int8
	switch typ {
	case vm.TypeInt:
		a = 0
	case vm.TypeFloat:
		a = 1
	default:
		panic("LoadNumber only accepts vm.TypeInt or vm.TypeFloat as type")
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpLoadNumber, A: a, B: index, C: dst})
}

// MakeChan appends a new "MakeChan" instruction to the function body.
//
//     dst = make(typ, capacity)
//
func (builder *functionBuilder) MakeChan(typ reflect.Type, kCapacity bool, capacity int8, dst int8) {
	fn := builder.fn
	t := builder.addType(typ)
	op := vm.OpMakeChan
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
				fn.Body = append(fn.Body, vm.Instruction{Op: -vm.OpAlloc, A: a, B: b, C: c})
				constantAlloc = true
			}
		}
		if !constantAlloc {
			fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpAlloc})
		}
	}
	fn.Body = append(fn.Body, vm.Instruction{Op: op, A: t, B: capacity, C: dst})
}

// MakeMap appends a new "MakeMap" instruction to the function body.
//
//     dst = make(typ, size)
//
func (builder *functionBuilder) MakeMap(typ reflect.Type, kSize bool, size int8, dst int8) {
	fn := builder.fn
	t := builder.addType(typ)
	op := vm.OpMakeMap
	if kSize {
		op = -op
	}
	if builder.allocs != nil {
		if kSize {
			bytes := 24 + 50*int(size)
			a, b, c := encodeUint24(uint32(bytes))
			fn.Body = append(fn.Body, vm.Instruction{Op: -vm.OpAlloc, A: a, B: b, C: c})
		} else {
			fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpAlloc})
		}
	}
	fn.Body = append(fn.Body, vm.Instruction{Op: op, A: t, B: size, C: dst})
}

// MakeSlice appends a new "MakeSlice" instruction to the function body.
//
//     make(sliceType, len, cap)
//
func (builder *functionBuilder) MakeSlice(kLen, kCap bool, sliceType reflect.Type, len, cap, dst int8) {
	fn := builder.fn
	t := builder.addType(sliceType)
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
				fn.Body = append(fn.Body, vm.Instruction{Op: -vm.OpAlloc, A: a, B: b, C: c})
				constantAlloc = true
			}
		}
		if !constantAlloc {
			fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpAlloc})
		}
	}
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpMakeSlice, A: t, B: k, C: dst})
	if k > 0 {
		fn.Body = append(fn.Body, vm.Instruction{A: len, B: cap})
	}
}

// MethodValue appends a new "MethodValue" instruction to the function body.
//
//     dst = receiver.name
//
func (builder *functionBuilder) MethodValue(name string, receiver int8, dst int8) {
	str := builder.makeStringConstant(name)
	fn := builder.fn
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpMethodValue, A: receiver, B: str, C: dst})
}

// Move appends a new "Move" instruction to the function body.
//
//     z = x
//
func (builder *functionBuilder) Move(k bool, x, z int8, kind reflect.Kind) {
	op := vm.OpMove
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: int8(kindToType(kind)), B: x, C: z})
}

// Mul appends a new "mul" instruction to the function body.
//
//     z = x * y
//
func (builder *functionBuilder) Mul(ky bool, x, y, z int8, kind reflect.Kind) {
	var op vm.Operation
	switch kind {
	case reflect.Int, reflect.Uint, reflect.Uintptr:
		op = vm.OpMulInt64
		if strconv.IntSize == 32 {
			op = vm.OpMulInt32
		}
	case reflect.Int64, reflect.Uint64:
		op = vm.OpMulInt64
	case reflect.Int32, reflect.Uint32:
		op = vm.OpMulInt32
	case reflect.Int16, reflect.Uint16:
		op = vm.OpMulInt16
	case reflect.Int8, reflect.Uint8:
		op = vm.OpMulInt8
	case reflect.Float64:
		op = vm.OpMulFloat64
	case reflect.Float32:
		op = vm.OpMulFloat32
	default:
		panic("mul: invalid type")
	}
	if ky {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: x, B: y, C: z})
}

// New appends a new "new" instruction to the function body.
//
//     z = new(t)
//
func (builder *functionBuilder) New(typ reflect.Type, z int8) {
	fn := builder.fn
	b := builder.addType(typ)
	if builder.allocs != nil {
		bytes := int(typ.Size())
		if bytes <= maxUint24 {
			a, b, c := encodeUint24(uint32(bytes))
			fn.Body = append(fn.Body, vm.Instruction{Op: -vm.OpAlloc, A: a, B: b, C: c})
		} else {
			fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpAlloc})
		}
	}
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpNew, B: int8(b), C: z})
}

// Nop appends a new "Nop" instruction to the function body.
//
func (builder *functionBuilder) Nop() {
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpNone})
}

// Or appends a new "Or" instruction to the function body.
//
//     z = x | y
//
func (builder *functionBuilder) Or(k bool, x, y, z int8, kind reflect.Kind) {
	op := vm.OpOr
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: x, B: y, C: z})
}

// Panic appends a new "Panic" instruction to the function body.
//
//     panic(v)
//
func (builder *functionBuilder) Panic(v int8, line int) {
	fn := builder.fn
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpPanic, A: v})
	builder.addLine(uint32(len(fn.Body)-1), line)
}

// Print appends a new "Print" instruction to the function body.
//
//     print(arg)
//
func (builder *functionBuilder) Print(arg int8) {
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpPrint, A: arg})
}

// Receive appends a new "Receive" instruction to the function body.
//
//	dst = <- ch
//
//	dst, ok = <- ch
//
func (builder *functionBuilder) Receive(ch, ok, dst int8) {
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpReceive, A: ch, B: ok, C: dst})
}

// Recover appends a new "Recover" instruction to the function body.
//
//     recover()
//
func (builder *functionBuilder) Recover(r int8) {
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpRecover, C: r})
}

// Rem appends a new "rem" instruction to the function body.
//
//     z = x % y
//
func (builder *functionBuilder) Rem(ky bool, x, y, z int8, kind reflect.Kind) {
	var op vm.Operation
	switch kind {
	case reflect.Int:
		op = vm.OpRemInt64
		if strconv.IntSize == 32 {
			op = vm.OpRemInt32
		}
	case reflect.Int64:
		op = vm.OpRemInt64
	case reflect.Int32:
		op = vm.OpRemInt32
	case reflect.Int16:
		op = vm.OpRemInt16
	case reflect.Int8:
		op = vm.OpRemInt8
	case reflect.Uint, reflect.Uintptr:
		op = vm.OpRemUint64
		if strconv.IntSize == 32 {
			op = vm.OpRemUint32
		}
	case reflect.Uint64:
		op = vm.OpRemUint64
	case reflect.Uint32:
		op = vm.OpRemUint32
	case reflect.Uint16:
		op = vm.OpRemUint16
	case reflect.Uint8:
		op = vm.OpRemUint8
	default:
		panic("rem: invalid type")
	}
	if ky {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: x, B: y, C: z})
}

// Return appends a new "return" instruction to the function body.
//
//     return
//
func (builder *functionBuilder) Return() {
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpReturn})
}

// RightShift appends a new "RightShift" instruction to the function body.
//
//     z = x >> y
//
func (builder *functionBuilder) RightShift(k bool, x, y, z int8, kind reflect.Kind) {
	op := vm.OpRightShift
	switch kind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		op = vm.OpRightShiftU
	}
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: x, B: y, C: z})
}

// Select appends a new "Select" instruction to the function body.
//
//     select
//
func (builder *functionBuilder) Select() {
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpSelect})
}

// Send appends a new "Send" instruction to the function body.
//
//	ch <- v
//
func (builder *functionBuilder) Send(ch, v int8) {
	// TODO(Gianluca): how can send know kind/type?
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpSend, A: v, C: ch})
}

// SetAlloc sets the alloc property. If true, an Alloc instruction will be
// inserted where necessary.
func (builder *functionBuilder) SetAlloc(alloc bool) {
	if alloc && builder.allocs == nil {
		builder.allocs = append(builder.allocs, 0)
		builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpAlloc})
	}
}

// SetField appends a new "SetField" instruction to the function body.
//
//     s.field = v
//
func (builder *functionBuilder) SetField(k bool, s, field, v int8) {
	op := vm.OpSetField
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpSetField, A: v, B: field, C: s})
}

// SetVar appends a new "SetVar" instruction to the function body.
//
//     v = r
//
func (builder *functionBuilder) SetVar(k bool, r int8, v int) {
	op := vm.OpSetVar
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: r, B: int8(v >> 8), C: int8(v)})
}

// SetMap appends a new "SetMap" instruction to the function body.
//
//	m[key] = value
//
func (builder *functionBuilder) SetMap(k bool, m, value, key int8, typ reflect.Type) {
	fn := builder.fn
	op := vm.OpSetMap
	if k {
		op = -op
	}
	if builder.allocs != nil {
		kSize := int(typ.Key().Size())
		eSize := int(typ.Elem().Size())
		bytes := kSize + eSize
		if bytes < 0 {
			panic("out of memory")
		}
		if bytes <= maxUint24 {
			a, b, c := encodeUint24(uint32(bytes))
			fn.Body = append(fn.Body, vm.Instruction{Op: -vm.OpAlloc, A: a, B: b, C: c})
		} else {
			fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpAlloc})
		}
	}
	fn.Body = append(fn.Body, vm.Instruction{Op: op, A: m, B: value, C: key})
}

// SetSlice appends a new "SetSlice" instruction to the function body.
//
//	slice[index] = value
//
func (builder *functionBuilder) SetSlice(k bool, slice, value, index int8, elemKind reflect.Kind) {
	_ = elemKind // TODO(Gianluca): remove.
	in := vm.Instruction{Op: vm.OpSetSlice, A: slice, B: value, C: index}
	if k {
		in.Op = -in.Op
	}
	builder.fn.Body = append(builder.fn.Body, in)
}

// Slice appends a new "Slice" instruction to the function body.
//
//	slice[low:high:max]
//
func (builder *functionBuilder) Slice(klow, khigh, kmax bool, src, dst, low, high, max int8) {
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
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpSlice, A: src, B: b, C: dst})
	fn.Body = append(fn.Body, vm.Instruction{A: low, B: high, C: max})
}

// Sub appends a new "Sub" instruction to the function body.
//
//     z = x - y
//
func (builder *functionBuilder) Sub(k bool, x, y, z int8, kind reflect.Kind) {
	var op vm.Operation
	switch kind {
	case reflect.Int, reflect.Uint, reflect.Uintptr:
		op = vm.OpSubInt64
		if strconv.IntSize == 32 {
			op = vm.OpSubInt32
		}
	case reflect.Int64, reflect.Uint64:
		op = vm.OpSubInt64
	case reflect.Int32, reflect.Uint32:
		op = vm.OpSubInt32
	case reflect.Int16, reflect.Uint16:
		op = vm.OpSubInt16
	case reflect.Int8, reflect.Uint8:
		op = vm.OpSubInt8
	case reflect.Float64:
		op = vm.OpSubFloat64
	case reflect.Float32:
		op = vm.OpSubFloat32
	default:
		panic("sub: invalid type")
	}
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: x, B: y, C: z})
}

// SubInv appends a new "SubInv" instruction to the function body.
//
//     z = y - x
//
func (builder *functionBuilder) SubInv(k bool, x, y, z int8, kind reflect.Kind) {
	var op vm.Operation
	switch kind {
	case reflect.Int, reflect.Uint, reflect.Uintptr:
		op = vm.OpSubInvInt64
		if strconv.IntSize == 32 {
			op = vm.OpSubInvInt32
		}
	case reflect.Int64, reflect.Uint64:
		op = vm.OpSubInvInt64
	case reflect.Int32, reflect.Uint32:
		op = vm.OpSubInvInt32
	case reflect.Int16, reflect.Uint16:
		op = vm.OpSubInvInt16
	case reflect.Int8, reflect.Uint8:
		op = vm.OpSubInvInt8
	case reflect.Float64:
		op = vm.OpSubInvFloat64
	case reflect.Float32:
		op = vm.OpSubInvFloat32
	default:
		panic("subInv: invalid type")
	}
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: x, B: y, C: z})
}

// Typify appends a new "Typify" instruction to the function body.
func (builder *functionBuilder) Typify(k bool, typ reflect.Type, x, z int8) {
	t := builder.addType(typ)
	op := vm.OpTypify
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: t, B: x, C: z})
}

// TailCall appends a new "TailCall" instruction to the function body.
//
//     f()
//
func (builder *functionBuilder) TailCall(f int8, line int) {
	fn := builder.fn
	fn.Body = append(fn.Body, vm.Instruction{Op: vm.OpTailCall, A: f})
	builder.addLine(uint32(len(fn.Body)-1), line)
}

// Xor appends a new "Xor" instruction to the function body.
//
//     z = x ^ y
//
func (builder *functionBuilder) Xor(k bool, x, y, z int8, kind reflect.Kind) {
	op := vm.OpXor
	if k {
		op = -op
	}
	builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: op, A: x, B: y, C: z})
}
