// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"

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

// newFunction returns a new function with a given package, name and type.
func newFunction(pkg, name string, typ reflect.Type) *vm.Function {
	return &vm.Function{Pkg: pkg, Name: name, Type: typ}
}

// newPredefinedFunction returns a new predefined function with a given
// package, name and implementation. fn must be a function type.
func newPredefinedFunction(pkg, name string, fn interface{}) *vm.PredefinedFunction {
	return &vm.PredefinedFunction{Pkg: pkg, Name: name, Func: fn}
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

// enterScope enters a new scope.
// Every enterScope call must be paired with a corresponding exitScope call.
func (builder *functionBuilder) enterScope() {
	builder.scopes = append(builder.scopes, map[string]int8{})
	builder.enterStack()
}

// exitScope exits last scope.
// Every exitScope call must be paired with a corresponding enterScope call.
func (builder *functionBuilder) exitScope() {
	builder.scopes = builder.scopes[:len(builder.scopes)-1]
	builder.exitStack()
}

// enterStack enters a new virtual stack, whose registers will be reused (if
// necessary) after calling exitScope.
// Every enterStack call must be paired with a corresponding exitStack call.
// enterStack/exitStack should be called before every temporary register
// allocation, which will be reused when exitStack is called.
//
// Usage:
//
// 		e.fb.enterStack()
// 		tmpReg := e.fb.newRegister(..)
// 		// use tmpReg in some way
// 		// move tmpReg content to externally-defined reg
// 		e.fb.exitStack()
//	    // tmpReg location is now available for reusing
//
func (builder *functionBuilder) enterStack() {
	scopeShift := vm.StackShift{
		int8(builder.numRegs[reflect.Int]),
		int8(builder.numRegs[reflect.Float64]),
		int8(builder.numRegs[reflect.String]),
		int8(builder.numRegs[reflect.Interface]),
	}
	builder.scopeShifts = append(builder.scopeShifts, scopeShift)
}

// exitStack exits current virtual stack, allowing its registers to be reused
// (if necessary).
// Every exitStack call must be paired with a corresponding enterStack call.
// See enterStack documentation for further details and usage.
func (builder *functionBuilder) exitStack() {
	shift := builder.scopeShifts[len(builder.scopeShifts)-1]
	builder.numRegs[reflect.Int] = uint8(shift[0])
	builder.numRegs[reflect.Float64] = uint8(shift[1])
	builder.numRegs[reflect.String] = uint8(shift[2])
	builder.numRegs[reflect.Interface] = uint8(shift[3])
	builder.scopeShifts = builder.scopeShifts[:len(builder.scopeShifts)-1]
}

// newRegister makes a new register of a given kind.
func (builder *functionBuilder) newRegister(kind reflect.Kind) int8 {
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

// bindVarReg binds name with register reg. To create a new variable, use
// VariableRegister in conjunction with bindVarReg.
func (builder *functionBuilder) bindVarReg(name string, reg int8) {
	builder.scopes[len(builder.scopes)-1][name] = reg
}

// isVariable reports whether n is a variable (i.e. is a name defined in some
// of the current scopes).
func (builder *functionBuilder) isVariable(n string) bool {
	for i := len(builder.scopes) - 1; i >= 0; i-- {
		_, ok := builder.scopes[i][n]
		if ok {
			return true
		}
	}
	return false
}

// scopeLookup returns n's register.
func (builder *functionBuilder) scopeLookup(n string) int8 {
	for i := len(builder.scopes) - 1; i >= 0; i-- {
		reg, ok := builder.scopes[i][n]
		if ok {
			return reg
		}
	}
	panic(fmt.Sprintf("bug: %s not found", n))
}

func (builder *functionBuilder) addLine(pc uint32, line int) {
	if builder.fn.Lines == nil {
		builder.fn.Lines = map[uint32]int{pc: line}
	} else {
		builder.fn.Lines[pc] = line
	}
}

// setFileLine sets the file name and line number of the Scriggo function.
func (builder *functionBuilder) setFileLine(file string, line int) {
	builder.fn.File = file
	builder.fn.Line = line
}

// addType adds a type to the builder's function, creating it if necessary.
func (builder *functionBuilder) addType(typ reflect.Type) int {
	fn := builder.fn
	for i, t := range fn.Types {
		if t == typ {
			return i
		}
	}
	index := len(fn.Types)
	if index > 255 {
		panic("types limit reached")
	}
	fn.Types = append(fn.Types, typ)
	return index
}

// addPredefinedFunction adds a predefined function to the builder's function.
func (builder *functionBuilder) addPredefinedFunction(f *vm.PredefinedFunction) uint8 {
	fn := builder.fn
	r := len(fn.Predefined)
	if r > 255 {
		panic("predefined functions limit reached")
	}
	fn.Predefined = append(fn.Predefined, f)
	return uint8(r)
}

// addFunction adds a function to the builder's function.
func (builder *functionBuilder) addFunction(f *vm.Function) uint8 {
	fn := builder.fn
	r := len(fn.Functions)
	if r > 255 {
		panic("Scriggo functions limit reached")
	}
	fn.Functions = append(fn.Functions, f)
	return uint8(r)
}

// makeStringConstant makes a new string constant, returning it's index.
func (builder *functionBuilder) makeStringConstant(c string) int8 {
	r := len(builder.fn.Constants.String)
	if r > 255 {
		panic("string refs limit reached")
	}
	builder.fn.Constants.String = append(builder.fn.Constants.String, c)
	return int8(r)
}

// makeGeneralConstant makes a new general constant, returning it's index.
func (builder *functionBuilder) makeGeneralConstant(v interface{}) int8 {
	r := len(builder.fn.Constants.General)
	if r > 255 {
		panic("general refs limit reached")
	}
	builder.fn.Constants.General = append(builder.fn.Constants.General, v)
	return int8(r)
}

// makeFloatConstant makes a new float constant, returning it's index.
func (builder *functionBuilder) makeFloatConstant(c float64) int8 {
	r := len(builder.fn.Constants.Float)
	if r > 255 {
		panic("float refs limit reached")
	}
	builder.fn.Constants.Float = append(builder.fn.Constants.Float, c)
	return int8(r)
}

// makeIntConstant makes a new int constant, returning it's index.
func (builder *functionBuilder) makeIntConstant(c int64) int8 {
	r := len(builder.fn.Constants.Int)
	if r > 255 {
		panic("int refs limit reached")
	}
	builder.fn.Constants.Int = append(builder.fn.Constants.Int, c)
	return int8(r)
}

func (builder *functionBuilder) makeInterfaceConstant(c interface{}) int8 {
	r := -len(builder.fn.Constants.General) - 1
	if r == -129 {
		panic("interface refs limit reached")
	}
	builder.fn.Constants.General = append(builder.fn.Constants.General, c)
	return int8(r)
}

// currentAddr returns builder's current address.
func (builder *functionBuilder) currentAddr() uint32 {
	return uint32(len(builder.fn.Body))
}

// newLabel creates a new empty label. Use setLabelAddr to associate an
// address to it.
func (builder *functionBuilder) newLabel() uint32 {
	builder.labels = append(builder.labels, uint32(0))
	return uint32(len(builder.labels))
}

// setLabelAddr sets label's address as builder's current address.
func (builder *functionBuilder) setLabelAddr(label uint32) {
	builder.labels[label-1] = builder.currentAddr()
}

func (builder *functionBuilder) end() {
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
