// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"
	"strconv"

	"scriggo/vm"
)

const maxUint24 = 16777215

var intType = reflect.TypeOf(0)
var int64Type = reflect.TypeOf(int64(0))
var float64Type = reflect.TypeOf(0.0)
var float32Type = reflect.TypeOf(float32(0.0))
var complex128Type = reflect.TypeOf(0i)
var complex64Type = reflect.TypeOf(complex64(0))
var stringType = reflect.TypeOf("")
var emptyInterfaceType = reflect.TypeOf(&[]interface{}{interface{}(nil)}[0]).Elem()

func encodeUint24(v uint32) (a, b, c int8) {
	a = int8(uint8(v >> 16))
	b = int8(uint8(v >> 8))
	c = int8(uint8(v))
	return
}

func encodeInt16(v int16) (a, b int8) {
	a = int8(v >> 8)
	b = int8(v)
	return
}

// encodeFieldIndex encodes a field index slice used by reflect into an int64.
func encodeFieldIndex(s []int) int64 {
	if len(s) == 1 {
		if s[0] > 255 {
			panic("struct field index #0 > 255")
		}
		return int64(s[0])
	}
	ss := make([]int, len(s))
	copy(ss, s)
	for i := range ss[1:] {
		if ss[i] > 254 {
			panic("struct field index > 254")
		}
		ss[i]++
	}
	fill := 8 - len(ss)
	for i := 0; i < fill; i++ {
		ss = append([]int{0}, ss...)
	}
	i := int64(0)
	i += int64(ss[0]) << 0
	i += int64(ss[1]) << 8
	i += int64(ss[2]) << 16
	i += int64(ss[3]) << 24
	i += int64(ss[4]) << 32
	i += int64(ss[5]) << 40
	i += int64(ss[6]) << 48
	i += int64(ss[7]) << 56
	return i
}

// decodeFieldIndex decodes i as a field index slice used by package reflect.
// Sync with vm.decodeFieldIndex.
func decodeFieldIndex(i int64) []int {
	if i <= 255 {
		return []int{int(i)}
	}
	s := []int{
		int(uint8(i >> 0)),
		int(uint8(i >> 8)),
		int(uint8(i >> 16)),
		int(uint8(i >> 24)),
		int(uint8(i >> 32)),
		int(uint8(i >> 40)),
		int(uint8(i >> 48)),
		int(uint8(i >> 56)),
	}
	ns := []int{}
	for i := 0; i < len(s); i++ {
		if i == len(s)-1 {
			ns = append(ns, s[i])
		} else {
			if s[i] > 0 {
				ns = append(ns, s[i]-1)
			}
		}
	}
	return ns
}

func decodeInt16(a, b int8) int16 {
	return int16(int(a)<<8 | int(uint8(b)))
}

func decodeUint24(a, b, c int8) uint32 {
	return uint32(uint8(a))<<16 | uint32(uint8(b))<<8 | uint32(uint8(c))
}

func negComplex(c interface{}) interface{} {
	switch c := c.(type) {
	case complex64:
		return -c
	case complex128:
		return -c
	}
	v := reflect.ValueOf(c)
	v2 := reflect.New(v.Type()).Elem()
	v2.SetComplex(-v.Complex())
	return v2.Interface()
}

func addComplex(c1, c2 interface{}) interface{} {
	switch c1 := c1.(type) {
	case complex64:
		return c1 + c2.(complex64)
	case complex128:
		return c1 + c2.(complex128)
	}
	v1 := reflect.ValueOf(c1)
	v2 := reflect.ValueOf(c2)
	v3 := reflect.New(v1.Type()).Elem()
	v3.SetComplex(v1.Complex() + v2.Complex())
	return v3.Interface()
}

func subComplex(c1, c2 interface{}) interface{} {
	switch c1 := c1.(type) {
	case complex64:
		return c1 - c2.(complex64)
	case complex128:
		return c1 - c2.(complex128)
	}
	v1 := reflect.ValueOf(c1)
	v2 := reflect.ValueOf(c2)
	v3 := reflect.New(v1.Type()).Elem()
	v3.SetComplex(v1.Complex() - v2.Complex())
	return v3.Interface()
}

func mulComplex(c1, c2 interface{}) interface{} {
	switch c1 := c1.(type) {
	case complex64:
		return c1 * c2.(complex64)
	case complex128:
		return c1 * c2.(complex128)
	}
	v1 := reflect.ValueOf(c1)
	v2 := reflect.ValueOf(c2)
	v3 := reflect.New(v1.Type()).Elem()
	v3.SetComplex(v1.Complex() * v2.Complex())
	return v3.Interface()
}

func divComplex(c1, c2 interface{}) interface{} {
	switch c1 := c1.(type) {
	case complex64:
		return c1 / c2.(complex64)
	case complex128:
		return c1 / c2.(complex128)
	}
	v1 := reflect.ValueOf(c1)
	v2 := reflect.ValueOf(c2)
	v3 := reflect.New(v1.Type()).Elem()
	v3.SetComplex(v1.Complex() / v2.Complex())
	return v3.Interface()
}

type functionBuilder struct {
	fn          *vm.Function
	labels      []uint32
	gotos       map[uint32]uint32
	maxRegs     map[reflect.Kind]uint8 // max number of registers allocated at the same time.
	numRegs     map[reflect.Kind]uint8
	scopes      []map[string]int8
	scopeShifts []vm.StackShift
	allocs      []uint32
}

// newBuilder returns a new function builder for the function fn.
func newBuilder(fn *vm.Function) *functionBuilder {
	fn.Body = nil
	builder := &functionBuilder{
		fn:      fn,
		gotos:   map[uint32]uint32{},
		maxRegs: map[reflect.Kind]uint8{},
		numRegs: map[reflect.Kind]uint8{},
		scopes:  []map[string]int8{},
	}
	return builder
}

// EnterScope enters a new scope.
// Every EnterScope call must be paired with a corresponding ExitScope call.
func (builder *functionBuilder) EnterScope() {
	builder.scopes = append(builder.scopes, map[string]int8{})
	builder.EnterStack()
}

// ExitScope exits last scope.
// Every ExitScope call must be paired with a corresponding EnterScope call.
func (builder *functionBuilder) ExitScope() {
	builder.scopes = builder.scopes[:len(builder.scopes)-1]
	builder.ExitStack()
}

// EnterStack enters a new virtual stack, whose registers will be reused (if
// necessary) after calling ExitScope.
// Every EnterStack call must be paired with a corresponding ExitStack call.
// EnterStack/ExitStack should be called before every temporary register
// allocation, which will be reused when ExitStack is called.
//
// Usage:
//
// 		e.fb.EnterStack()
// 		tmpReg := e.fb.NewRegister(..)
// 		// use tmpReg in some way
// 		// move tmpReg content to externally-defined reg
// 		e.fb.ExitStack()
//	    // tmpReg location is now available for reusing
//
func (builder *functionBuilder) EnterStack() {
	scopeShift := vm.StackShift{
		int8(builder.numRegs[reflect.Int]),
		int8(builder.numRegs[reflect.Float64]),
		int8(builder.numRegs[reflect.String]),
		int8(builder.numRegs[reflect.Interface]),
	}
	builder.scopeShifts = append(builder.scopeShifts, scopeShift)
}

// ExitStack exits current virtual stack, allowing its registers to be reused
// (if necessary).
// Every ExitStack call must be paired with a corresponding EnterStack call.
// See EnterStack documentation for further details and usage.
func (builder *functionBuilder) ExitStack() {
	shift := builder.scopeShifts[len(builder.scopeShifts)-1]
	builder.numRegs[reflect.Int] = uint8(shift[0])
	builder.numRegs[reflect.Float64] = uint8(shift[1])
	builder.numRegs[reflect.String] = uint8(shift[2])
	builder.numRegs[reflect.Interface] = uint8(shift[3])
	builder.scopeShifts = builder.scopeShifts[:len(builder.scopeShifts)-1]
}

// NewRegister makes a new register of a given kind.
func (builder *functionBuilder) NewRegister(kind reflect.Kind) int8 {
	switch kindToType(kind) {
	case vm.TypeInt:
		kind = reflect.Int
	case vm.TypeFloat:
		kind = reflect.Float64
	case vm.TypeString:
		kind = reflect.String
	case vm.TypeGeneral:
		kind = reflect.Interface
	}
	reg := int8(builder.numRegs[kind]) + 1
	builder.allocRegister(kind, reg)
	return reg
}

// BindVarReg binds name with register reg. To create a new variable, use
// VariableRegister in conjunction with BindVarReg.
func (builder *functionBuilder) BindVarReg(name string, reg int8) {
	builder.scopes[len(builder.scopes)-1][name] = reg
}

// IsVariable reports whether n is a variable (i.e. is a name defined in some
// of the current scopes).
func (builder *functionBuilder) IsVariable(n string) bool {
	for i := len(builder.scopes) - 1; i >= 0; i-- {
		_, ok := builder.scopes[i][n]
		if ok {
			return true
		}
	}
	return false
}

// ScopeLookup returns n's register.
func (builder *functionBuilder) ScopeLookup(n string) int8 {
	for i := len(builder.scopes) - 1; i >= 0; i-- {
		reg, ok := builder.scopes[i][n]
		if ok {
			return reg
		}
	}
	panic(fmt.Sprintf("bug: %s not found", n))
}

func (builder *functionBuilder) AddLine(pc uint32, line int) {
	if builder.fn.Lines == nil {
		builder.fn.Lines = map[uint32]int{pc: line}
	} else {
		builder.fn.Lines[pc] = line
	}
}

// SetFileLine sets the file name and line number of the Scriggo function.
func (builder *functionBuilder) SetFileLine(file string, line int) {
	builder.fn.File = file
	builder.fn.Line = line
}

// SetAlloc sets the alloc property. If true, an Alloc instruction will be
// inserted where necessary.
func (builder *functionBuilder) SetAlloc(alloc bool) {
	if alloc && builder.allocs == nil {
		builder.allocs = append(builder.allocs, 0)
		builder.fn.Body = append(builder.fn.Body, vm.Instruction{Op: vm.OpAlloc})
	}
}

// newFunction returns a new function with a given package, name and type.
func newFunction(pkg, name string, typ reflect.Type) *vm.Function {
	return &vm.Function{Pkg: pkg, Name: name, Type: typ}
}

// newPredefinedFunction returns a new predefined function with a given
// package, name and implementation. fn must be a function type.
func newPredefinedFunction(pkg, name string, fn interface{}) *vm.PredefinedFunction {
	return &vm.PredefinedFunction{Pkg: pkg, Name: name, Func: fn}
}

// AddType adds a type to the builder's function.
func (builder *functionBuilder) AddType(typ reflect.Type) uint8 {
	fn := builder.fn
	index := len(fn.Types)
	if index > 255 {
		panic("types limit reached")
	}
	for i, t := range fn.Types {
		if t == typ {
			return uint8(i)
		}
	}
	fn.Types = append(fn.Types, typ)
	return uint8(index)
}

// AddPredefinedFunction adds a predefined function to the builder's function.
func (builder *functionBuilder) AddPredefinedFunction(f *vm.PredefinedFunction) uint8 {
	fn := builder.fn
	r := len(fn.Predefined)
	if r > 255 {
		panic("predefined functions limit reached")
	}
	fn.Predefined = append(fn.Predefined, f)
	return uint8(r)
}

// AddFunction adds a function to the builder's function.
func (builder *functionBuilder) AddFunction(f *vm.Function) uint8 {
	fn := builder.fn
	r := len(fn.Functions)
	if r > 255 {
		panic("Scriggo functions limit reached")
	}
	fn.Functions = append(fn.Functions, f)
	return uint8(r)
}

// MakeStringConstant makes a new string constant, returning it's index.
func (builder *functionBuilder) MakeStringConstant(c string) int8 {
	r := len(builder.fn.Constants.String)
	if r > 255 {
		panic("string refs limit reached")
	}
	builder.fn.Constants.String = append(builder.fn.Constants.String, c)
	return int8(r)
}

// MakeGeneralConstant makes a new general constant, returning it's index.
func (builder *functionBuilder) MakeGeneralConstant(v interface{}) int8 {
	r := len(builder.fn.Constants.General)
	if r > 255 {
		panic("general refs limit reached")
	}
	builder.fn.Constants.General = append(builder.fn.Constants.General, v)
	return int8(r)
}

// MakeFloatConstant makes a new float constant, returning it's index.
func (builder *functionBuilder) MakeFloatConstant(c float64) int8 {
	r := len(builder.fn.Constants.Float)
	if r > 255 {
		panic("float refs limit reached")
	}
	builder.fn.Constants.Float = append(builder.fn.Constants.Float, c)
	return int8(r)
}

// MakeIntConstant makes a new int constant, returning it's index.
func (builder *functionBuilder) MakeIntConstant(c int64) int8 {
	r := len(builder.fn.Constants.Int)
	if r > 255 {
		panic("int refs limit reached")
	}
	builder.fn.Constants.Int = append(builder.fn.Constants.Int, c)
	return int8(r)
}

func (builder *functionBuilder) MakeInterfaceConstant(c interface{}) int8 {
	r := -len(builder.fn.Constants.General) - 1
	if r == -129 {
		panic("interface refs limit reached")
	}
	builder.fn.Constants.General = append(builder.fn.Constants.General, c)
	return int8(r)
}

// CurrentAddr returns builder's current address.
func (builder *functionBuilder) CurrentAddr() uint32 {
	return uint32(len(builder.fn.Body))
}

// NewLabel creates a new empty label. Use SetLabelAddr to associate an
// address to it.
func (builder *functionBuilder) NewLabel() uint32 {
	builder.labels = append(builder.labels, uint32(0))
	return uint32(len(builder.labels))
}

// SetLabelAddr sets label's address as builder's current address.
func (builder *functionBuilder) SetLabelAddr(label uint32) {
	builder.labels[label-1] = builder.CurrentAddr()
}

// Type returns typ's index, creating it if necessary.
func (builder *functionBuilder) Type(typ reflect.Type) int8 {
	var tr int8
	var found bool
	types := builder.fn.Types
	for i, t := range types {
		if t == typ {
			tr = int8(i)
			found = true
		}
	}
	if !found {
		if len(types) == 256 {
			panic("types limit reached")
		}
		tr = int8(len(types))
		builder.fn.Types = append(types, typ)
	}
	return tr
}

func (builder *functionBuilder) End() {
	fn := builder.fn
	if len(fn.Body) == 0 || fn.Body[len(fn.Body)-1].Op != vm.OpReturn {
		builder.Return()
	}
	for addr, label := range builder.gotos {
		i := fn.Body[addr]
		i.A, i.B, i.C = encodeUint24(builder.labels[label-1])
		fn.Body[addr] = i
	}
	builder.gotos = nil
	for kind, num := range builder.maxRegs {
		switch {
		case reflect.Int <= kind && kind <= reflect.Uintptr:
			if num > fn.RegNum[0] {
				fn.RegNum[0] = num
			}
		case kind == reflect.Float64 || kind == reflect.Float32:
			if num > fn.RegNum[1] {
				fn.RegNum[1] = num
			}
		case kind == reflect.String:
			if num > fn.RegNum[2] {
				fn.RegNum[2] = num
			}
		default:
			if num > fn.RegNum[3] {
				fn.RegNum[3] = num
			}
		}
	}
	if builder.allocs != nil {
		for _, addr := range builder.allocs {
			var bytes int
			if addr == 0 {
				bytes = vm.CallFrameSize + 8*int(fn.RegNum[0]+fn.RegNum[1]) + 16*int(fn.RegNum[2]+fn.RegNum[3])
			} else {
				in := fn.Body[addr+1]
				if in.Op == vm.OpFunc {
					f := fn.Literals[uint8(in.B)]
					bytes = 32 + len(f.VarRefs)*16
				}
			}
			a, b, c := encodeUint24(uint32(bytes))
			fn.Body[addr] = vm.Instruction{Op: -vm.OpAlloc, A: a, B: b, C: c}
		}
	}
}

func (builder *functionBuilder) allocRegister(kind reflect.Kind, reg int8) {
	switch kindToType(kind) {
	case vm.TypeInt:
		kind = reflect.Int
	case vm.TypeFloat:
		kind = reflect.Float64
	case vm.TypeString:
		kind = reflect.String
	case vm.TypeGeneral:
		kind = reflect.Interface
	}
	if reg > 0 {
		if num, ok := builder.maxRegs[kind]; !ok || uint8(reg) > num {
			builder.maxRegs[kind] = uint8(reg)
		}
		if num, ok := builder.numRegs[kind]; !ok || uint8(reg) > num {
			builder.numRegs[kind] = uint8(reg)
		}
	}
}

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
	builder.AddLine(uint32(len(fn.Body)-2), line)
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
	regType := builder.Type(typ)
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
			builder.gotos[builder.CurrentAddr()] = label
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
	t := builder.Type(typ)
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
	t := builder.Type(typ)
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
	t := builder.Type(sliceType)
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
	str := builder.MakeStringConstant(name)
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
	b := builder.AddType(typ)
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
	builder.AddLine(uint32(len(fn.Body)-1), line)
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
	t := builder.Type(typ)
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
	builder.AddLine(uint32(len(fn.Body)-1), line)
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
