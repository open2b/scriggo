// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"fmt"
	"reflect"
)

const maxInt8 = 128

type Type int8

const (
	TypeInt Type = iota
	TypeFloat
	TypeString
	TypeIface
)

type Kind uint8

const (
	Bool      = Kind(reflect.Bool)
	Int       = Kind(reflect.Int)
	Int8      = Kind(reflect.Int8)
	Int16     = Kind(reflect.Int16)
	Int32     = Kind(reflect.Int32)
	Int64     = Kind(reflect.Int64)
	Uint      = Kind(reflect.Uint)
	Uint8     = Kind(reflect.Uint8)
	Uint16    = Kind(reflect.Uint16)
	Uint32    = Kind(reflect.Uint32)
	Uint64    = Kind(reflect.Uint64)
	Float32   = Kind(reflect.Float32)
	Float64   = Kind(reflect.Float64)
	String    = Kind(reflect.String)
	Func      = Kind(reflect.Func)
	Interface = Kind(reflect.Interface)
)

func (t Type) String() string {
	switch t {
	case TypeInt:
		return "int"
	case TypeFloat:
		return "float"
	case TypeString:
		return "string"
	case TypeIface:
		return "iface"
	}
	panic("unknown type")
}

type Condition int8

const (
	ConditionEqual             Condition = iota // x == y
	ConditionNotEqual                           // x != y
	ConditionLess                               // x <  y
	ConditionLessOrEqual                        // x <= y
	ConditionGreater                            // x >  y
	ConditionGreaterOrEqual                     // x >= y
	ConditionEqualLen                           // len(x) == y
	ConditionNotEqualLen                        // len(x) != y
	ConditionLessLen                            // len(x) <  y
	ConditionLessOrEqualLen                     // len(x) <= y
	ConditionGreaterLen                         // len(x) >  y
	ConditionGreaterOrEqualLen                  // len(x) >= y
	ConditionNil                                // x == nil
	ConditionNotNil                             // x != nil
	ConditionOK                                 // [vm.ok]
	ConditionNotOK                              // ![vm.ok]
)

func (c Condition) String() string {
	switch c {
	case ConditionEqual:
		return "Equal"
	case ConditionNotEqual:
		return "NotEqual"
	case ConditionLess:
		return "Less"
	case ConditionLessOrEqual:
		return "LessOrEqual"
	case ConditionGreater:
		return "Greater"
	case ConditionGreaterOrEqual:
		return "GreaterOrEqual"
	case ConditionEqualLen:
		return "EqualLen"
	case ConditionNotEqualLen:
		return "NotEqualLen"
	case ConditionLessLen:
		return "LessLen"
	case ConditionLessOrEqualLen:
		return "LessOrEqualLen"
	case ConditionGreaterLen:
		return "GreaterOrEqualLen"
	case ConditionGreaterOrEqualLen:
		return "GreaterOrEqualLen"
	case ConditionNil:
		return "Nil"
	case ConditionNotNil:
		return "NotNil"
	case ConditionOK:
		return "OK"
	case ConditionNotOK:
		return "NotOK"
	}
	panic("unknown condition")
}

type NativeFunction struct {
	pkg    string
	name   string
	fast   interface{}
	value  reflect.Value
	in     []Kind
	out    []Kind
	args   []reflect.Value
	outOff [4]int8
}

func NewNativeFunction(pkg, name string, fn interface{}) *NativeFunction {
	return &NativeFunction{pkg: pkg, name: name, fast: fn}
}

func (fn *NativeFunction) slow() {
	if !fn.value.IsValid() {
		fn.value = reflect.ValueOf(fn.fast)
	}
	typ := fn.value.Type()
	nIn := typ.NumIn()
	fn.in = make([]Kind, nIn)
	fn.args = make([]reflect.Value, nIn)
	for i := 0; i < nIn; i++ {
		t := typ.In(i)
		k := t.Kind()
		switch {
		case k == reflect.Bool:
			fn.in[i] = Bool
		case reflect.Int <= k && k <= reflect.Int64:
			fn.in[i] = Int
		case reflect.Uint <= k && k <= reflect.Uint64:
			fn.in[i] = Uint
		case k == reflect.Float64 || k == reflect.Float32:
			fn.in[i] = Float64
		case k == reflect.String:
			fn.in[i] = String
		case k == reflect.Func:
			fn.in[i] = Func
		default:
			fn.in[i] = Interface
		}
		fn.args[i] = reflect.New(t).Elem()
	}
	nOut := typ.NumOut()
	fn.out = make([]Kind, nOut)
	for i := 0; i < nOut; i++ {
		k := typ.Out(i).Kind()
		switch {
		case k == reflect.Bool:
			fn.out[i] = Bool
			fn.outOff[0]++
		case reflect.Int <= k && k <= reflect.Int64:
			fn.out[i] = Int
			fn.outOff[0]++
		case reflect.Uint <= k && k <= reflect.Uint64:
			fn.out[i] = Uint
			fn.outOff[0]++
		case k == reflect.Float64 || k == reflect.Float32:
			fn.out[i] = Float64
			fn.outOff[1]++
		case k == reflect.String:
			fn.out[i] = String
			fn.outOff[2]++
		case k == reflect.Func:
			fn.out[i] = Func
			fn.outOff[3]++
		default:
			fn.out[i] = Interface
			fn.outOff[3]++
		}
	}
	fn.fast = nil
}

type variable struct {
	pkg   string
	name  string
	value interface{}
}

func NewVariable(pkg, name string, value interface{}) variable {
	return variable{pkg, name, value}
}

// AddVariable adds a variable to a Scrigo function.
func (fn *ScrigoFunction) AddVariable(v variable) uint8 {
	r := len(fn.variables)
	if r > 255 {
		panic("variables limit reached")
	}
	fn.variables = append(fn.variables, v)
	return uint8(r)
}

type instruction struct {
	op      operation
	a, b, c int8
}

// TODO(Gianluca): this is unused, will be removed.
// type Upper struct {
// 	typ   Type
// 	index int32
// }

// ScrigoFunction represents a Scrigo function.
type ScrigoFunction struct {
	pkg             string
	name            string
	file            string
	line            int
	typ             reflect.Type
	parent          *ScrigoFunction
	crefs           []int16           // opFunc
	literals        []*ScrigoFunction // opFunc
	types           []reflect.Type    // opAlloc, opAssert, opMakeMap, opMakeSlice, opNew
	regnum          [4]uint8          // opCall, opCallDirect
	constants       registers
	variables       []variable
	scrigoFunctions []*ScrigoFunction
	nativeFunctions []*NativeFunction
	body            []instruction // run, opCall, opCallDirect
	lines           map[uint32]int
}

func NewScrigoFunction(pkg, name string, typ reflect.Type) *ScrigoFunction {
	return &ScrigoFunction{pkg: pkg, name: name, typ: typ}
}

func (fn *ScrigoFunction) AddLine(pc uint32, line int) {
	if fn.lines == nil {
		fn.lines = map[uint32]int{pc: line}
	} else {
		fn.lines[pc] = line
	}
}

// AddNativeFunction adds a native function to a function.
func (fn *ScrigoFunction) AddNativeFunction(f *NativeFunction) uint8 {
	r := len(fn.nativeFunctions)
	if r > 255 {
		panic("native functions limit reached")
	}
	fn.nativeFunctions = append(fn.nativeFunctions, f)
	return uint8(r)
}

// AddScrigoFunction add a Scrigo function to a function.
// TODO(Gianluca): is this method obsolete?
func (fn *ScrigoFunction) AddScrigoFunction(f *ScrigoFunction) uint8 {
	r := len(fn.scrigoFunctions)
	if r > 255 {
		panic("Scrigo functions limit reached")
	}
	fn.scrigoFunctions = append(fn.scrigoFunctions, f)
	return uint8(r)
}

func (fn *ScrigoFunction) SetClosureRefs(refs []int16) {
	fn.crefs = refs
}

func (fn *ScrigoFunction) SetFileLine(file string, line int) {
	fn.file = file
	fn.line = line
}

func (fn *ScrigoFunction) AddType(typ reflect.Type) uint8 {
	index := len(fn.types)
	if index > 255 {
		panic("types limit reached")
	}
	for i, t := range fn.types {
		if t == typ {
			return uint8(i)
		}
	}
	fn.types = append(fn.types, typ)
	return uint8(index)
}

type FunctionBuilder struct {
	fn          *ScrigoFunction
	labels      []uint32
	gotos       map[uint32]uint32
	maxRegs     map[reflect.Kind]uint8 // max number of registers allocated at the same time.
	numRegs     map[reflect.Kind]uint8
	scopes      []map[string]int8
	scopeShifts []StackShift
}

// Builder returns the body of the function.
func (fn *ScrigoFunction) Builder() *FunctionBuilder {
	fn.body = nil
	return &FunctionBuilder{
		fn:      fn,
		gotos:   map[uint32]uint32{},
		maxRegs: map[reflect.Kind]uint8{},
		numRegs: map[reflect.Kind]uint8{},
		scopes:  []map[string]int8{},
	}
}

// EnterScope enters a new scope.
// Every EnterScope call must be paired with a corresponding ExitScope call.
func (builder *FunctionBuilder) EnterScope() {
	builder.scopes = append(builder.scopes, map[string]int8{})
	builder.EnterStack()
}

// ExitScope exits last scope.
// Every ExitScope call must be paired with a corresponding EnterScope call.
func (builder *FunctionBuilder) ExitScope() {
	builder.scopes = builder.scopes[:len(builder.scopes)-1]
	builder.ExitStack()
}

// EnterStack enters a new virtual stack, whose registers will be reused (if
// necessary) after calling ExitScope.
// Every EnterStack call must be paired with a corresponding ExitStack call.
func (builder *FunctionBuilder) EnterStack() {
	scopeShift := StackShift{
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
func (builder *FunctionBuilder) ExitStack() {
	shift := builder.scopeShifts[len(builder.scopeShifts)-1]
	builder.numRegs[reflect.Int] = uint8(shift[0])
	builder.numRegs[reflect.Float64] = uint8(shift[1])
	builder.numRegs[reflect.String] = uint8(shift[2])
	builder.numRegs[reflect.Interface] = uint8(shift[3])
	builder.scopeShifts = builder.scopeShifts[:len(builder.scopeShifts)-1]
}

// NewRegister makes a new register of a given kind.
func (builder *FunctionBuilder) NewRegister(kind reflect.Kind) int8 {
	switch kind {
	// TODO (Gianluca): to review (same as allocRegister)
	case reflect.Bool:
		kind = reflect.Int
	case reflect.Func:
		kind = reflect.Interface
	}
	reg := int8(builder.numRegs[kind]) + 1
	builder.allocRegister(kind, reg)
	return reg
}

// BindVarReg binds name with register reg. To create a new variable, use
// VariableRegister in conjunction with BindVarReg.
func (builder *FunctionBuilder) BindVarReg(name string, reg int8) {
	builder.scopes[len(builder.scopes)-1][name] = reg
}

// IsVariable indicates if n is a variable (i.e. is a name defined in some of
// the current scopes).
func (builder *FunctionBuilder) IsVariable(n string) bool {
	for i := len(builder.scopes) - 1; i >= 0; i-- {
		_, ok := builder.scopes[i][n]
		if ok {
			return true
		}
	}
	return false
}

// ScopeLookup returns n's register.
func (builder *FunctionBuilder) ScopeLookup(n string) int8 {
	for i := len(builder.scopes) - 1; i >= 0; i-- {
		reg, ok := builder.scopes[i][n]
		if ok {
			return reg
		}
	}
	panic(fmt.Sprintf("bug: %s not found", n))
}

// MakeStringConstant makes a new string constant, returning it's index.
func (builder *FunctionBuilder) MakeStringConstant(c string) int8 {
	r := len(builder.fn.constants.String)
	if r > 255 {
		panic("string refs limit reached")
	}
	builder.fn.constants.String = append(builder.fn.constants.String, c)
	return int8(r)
}

// MakeGeneralConstant makes a new general constant, returning it's index.
func (builder *FunctionBuilder) MakeGeneralConstant(v interface{}) int8 {
	r := len(builder.fn.constants.General)
	if r > 255 {
		panic("general refs limit reached")
	}
	builder.fn.constants.General = append(builder.fn.constants.General, v)
	return int8(r)
}

// MakeFloatConstant makes a new float constant, returning it's index.
func (builder *FunctionBuilder) MakeFloatConstant(c float64) int8 {
	r := len(builder.fn.constants.Float)
	if r > 255 {
		panic("float refs limit reached")
	}
	builder.fn.constants.Float = append(builder.fn.constants.Float, c)
	return int8(r)
}

// MakeIntConstant makes a new int constant, returning it's index.
func (builder *FunctionBuilder) MakeIntConstant(c int64) int8 {
	r := len(builder.fn.constants.Int)
	if r > 255 {
		panic("int refs limit reached")
	}
	builder.fn.constants.Int = append(builder.fn.constants.Int, c)
	return int8(r)
}

func (builder *FunctionBuilder) MakeInterfaceConstant(c interface{}) int8 {
	r := -len(builder.fn.constants.General) - 1
	if r == -129 {
		panic("interface refs limit reached")
	}
	builder.fn.constants.General = append(builder.fn.constants.General, c)
	return int8(r)
}

// CurrentAddr returns builder's current address.
func (builder *FunctionBuilder) CurrentAddr() uint32 {
	return uint32(len(builder.fn.body))
}

// NewLabel creates a new empty label. Use SetLabelAddr to associate an address
// to it.
func (builder *FunctionBuilder) NewLabel() uint32 {
	builder.labels = append(builder.labels, uint32(0))
	return uint32(len(builder.labels))
}

// SetLabelAddr sets label's address as builder's current address.
func (builder *FunctionBuilder) SetLabelAddr(label uint32) {
	builder.labels[label-1] = builder.CurrentAddr()
}

var intType = reflect.TypeOf(0)
var float64Type = reflect.TypeOf(0.0)
var stringType = reflect.TypeOf("")
var emptyInterfaceType = reflect.TypeOf(&[]interface{}{interface{}(nil)}[0]).Elem()

func encodeAddr(v uint32) (a, b, c int8) {
	a = int8(uint8(v))
	b = int8(uint8(v >> 8))
	c = int8(uint8(v >> 16))
	return
}

// Type returns typ's index, creating it if necessary.
func (builder *FunctionBuilder) Type(typ reflect.Type) int8 {
	var tr int8
	var found bool
	types := builder.fn.types
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
		builder.fn.types = append(types, typ)
	}
	return tr
}

// TODO (Gianluca): what's the point of this method? Can it be embedded in
// another method?
func (builder *FunctionBuilder) End() {
	fn := builder.fn
	for addr, label := range builder.gotos {
		i := fn.body[addr]
		i.a, i.b, i.c = encodeAddr(builder.labels[label-1])
		fn.body[addr] = i
	}
	builder.gotos = nil
	for kind, num := range builder.maxRegs {
		switch {
		case reflect.Int <= kind && kind <= reflect.Uint64:
			if num > fn.regnum[0] {
				fn.regnum[0] = num
			}
		case kind == reflect.Float64 || kind == reflect.Float32:
			if num > fn.regnum[1] {
				fn.regnum[1] = num
			}
		case kind == reflect.String:
			if num > fn.regnum[2] {
				fn.regnum[2] = num
			}
		default:
			if num > fn.regnum[3] {
				fn.regnum[3] = num
			}
		}
	}

}

func (builder *FunctionBuilder) allocRegister(kind reflect.Kind, reg int8) {
	switch kind {
	// TODO (Gianluca): to review (same as NewRegister)
	case reflect.Bool:
		kind = reflect.Int
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
func (builder *FunctionBuilder) Add(k bool, x, y, z int8, kind reflect.Kind) {
	var op operation
	builder.allocRegister(kind, x)
	if !k {
		builder.allocRegister(kind, y)
	}
	builder.allocRegister(kind, z)
	switch kind {
	case reflect.Int, reflect.Int64, reflect.Uint64:
		op = opAddInt
	case reflect.Int32, reflect.Uint32:
		op = opAddInt32
	case reflect.Int16, reflect.Uint16:
		op = opAddInt16
	case reflect.Int8, reflect.Uint8:
		op = opAddInt8
	case reflect.Float64:
		op = opAddFloat64
	case reflect.Float32:
		op = opAddFloat32
	default:
		panic("add: invalid type")
	}
	if k {
		op = -op
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: x, b: y, c: z})
}

// Assert appends a new "assert" instruction to the function body.
//
//     z = e.(t)
//
func (builder *FunctionBuilder) Assert(e int8, typ reflect.Type, z int8) {
	var op operation
	var tr int8
	builder.allocRegister(reflect.Interface, e)
	switch typ {
	case intType:
		builder.allocRegister(reflect.Int, z)
		op = opAssertInt
	case float64Type:
		builder.allocRegister(reflect.Float64, z)
		op = opAssertFloat64
	case stringType:
		builder.allocRegister(reflect.String, z)
		op = opAssertString
	default:
		builder.allocRegister(reflect.Interface, z)
		op = opAssert
		var found bool
		for i, t := range builder.fn.types {
			if t == typ {
				tr = int8(i)
				found = true
			}
		}
		if !found {
			tr = int8(len(builder.fn.types))
			builder.fn.types = append(builder.fn.types, typ)
		}
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: e, b: tr, c: z})
}

// Bind appends a new "Bind" instruction to the function body.
//
//     r = cv
//
func (builder *FunctionBuilder) Bind(cv uint8, r int8) {
	builder.allocRegister(reflect.Interface, r)
	builder.fn.body = append(builder.fn.body, instruction{op: opBind, b: int8(cv), c: r})
}

type StackShift [4]int8

// Call appends a new "Call" instruction to the function body.
//
//     p.f()
//
func (builder *FunctionBuilder) Call(f int8, shift StackShift, line int) {
	var fn = builder.fn
	fn.body = append(fn.body, instruction{op: opCall, a: f})
	fn.body = append(fn.body, instruction{op: operation(shift[0]), a: shift[1], b: shift[2], c: shift[3]})
	fn.AddLine(uint32(len(fn.body)-2), line)
}

// CallNative appends a new "CallNative" instruction to the function body.
//
//     p.F()
//
func (builder *FunctionBuilder) CallNative(f int8, numVariadic int8, shift StackShift) {
	var fn = builder.fn
	fn.body = append(fn.body, instruction{op: opCallNative, a: f, c: numVariadic})
	fn.body = append(fn.body, instruction{op: operation(shift[0]), a: shift[1], b: shift[2], c: shift[3]})
}

// CallIndirect appends a new "CallIndirect" instruction to the function body.
//
//     f()
//
func (builder *FunctionBuilder) CallIndirect(f int8, numVariadic int8, shift StackShift) {
	var fn = builder.fn
	fn.body = append(fn.body, instruction{op: opCallIndirect, a: f, c: numVariadic})
	fn.body = append(fn.body, instruction{op: operation(shift[0]), a: shift[1], b: shift[2], c: shift[3]})
}

// Assert appends a new "cap" instruction to the function body.
//
//     z = cap(s)
//
func (builder *FunctionBuilder) Cap(s, z int8) {
	builder.allocRegister(reflect.Interface, s)
	builder.allocRegister(reflect.Int, z)
	builder.fn.body = append(builder.fn.body, instruction{op: opCap, a: s, c: z})
}

// Concat appends a new "concat" instruction to the function body.
//
//     z = concat(s, t)
//
func (builder *FunctionBuilder) Concat(s, t, z int8) {
	builder.allocRegister(reflect.Interface, s)
	builder.allocRegister(reflect.Interface, t)
	builder.allocRegister(reflect.Interface, z)
	builder.fn.body = append(builder.fn.body, instruction{op: opConcat, a: s, b: t, c: z})
}

// Convert appends a new "Convert" instruction to the function body.
//
// 	 dst = typ(expr)
//
func (builder *FunctionBuilder) Convert(expr int8, dstType reflect.Type, dst int8) {
	// TODO (Gianluca): add support for every kind of convert operator.
	regType := builder.Type(dstType)
	builder.allocRegister(reflect.Interface, dst)
	builder.fn.body = append(builder.fn.body, instruction{op: opConvertInt, a: expr, b: regType, c: dst})
}

// Copy appends a new "Copy" instruction to the function body.
//
//     n == 0:   copy(dst, src)
// 	 n != 0:   n := copy(dst, src)
//
func (builder *FunctionBuilder) Copy(dst, src, n int8) {
	builder.allocRegister(reflect.Interface, dst)
	builder.allocRegister(reflect.Interface, src)
	builder.fn.body = append(builder.fn.body, instruction{op: opCopy, a: src, b: n, c: dst})
}

// Defer appends a new "Defer" instruction to the function body.
//
//     defer
//
func (builder *FunctionBuilder) Defer(f int8, numVariadic int8, off, arg StackShift) {
	var fn = builder.fn
	builder.allocRegister(reflect.Interface, f)
	fn.body = append(fn.body, instruction{op: opDefer, a: f, c: numVariadic})
	fn.body = append(fn.body, instruction{op: operation(off[0]), a: off[1], b: off[2], c: off[3]})
	fn.body = append(fn.body, instruction{op: operation(arg[0]), a: arg[1], b: arg[2], c: arg[3]})
}

// Delete appends a new "delete" instruction to the function body.
//
//     delete(m, k)
//
func (builder *FunctionBuilder) Delete(m, k int8) {
	builder.allocRegister(reflect.Interface, m)
	builder.allocRegister(reflect.Interface, k)
	builder.fn.body = append(builder.fn.body, instruction{op: opDelete, a: m, b: k})
}

// Div appends a new "div" instruction to the function body.
//
//     z = x / y
//
func (builder *FunctionBuilder) Div(ky bool, x, y, z int8, kind reflect.Kind) {
	builder.allocRegister(kind, x)
	builder.allocRegister(kind, y)
	builder.allocRegister(kind, z)
	var op operation
	switch kind {
	case reflect.Int, reflect.Int64:
		op = opDivInt
	case reflect.Int32:
		op = opDivInt32
	case reflect.Int16:
		op = opDivInt16
	case reflect.Int8:
		op = opDivInt8
	case reflect.Uint, reflect.Uint64:
		op = opDivUint64
	case reflect.Uint32:
		op = opDivUint32
	case reflect.Uint16:
		op = opDivUint16
	case reflect.Uint8:
		op = opDivUint8
	case reflect.Float64:
		op = opDivFloat64
	case reflect.Float32:
		op = opDivFloat32
	default:
		panic("div: invalid type")
	}
	if ky {
		op = -op
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: x, b: y, c: z})
}

// Range appends a new "Range" instruction to the function body.
//
//	TODO
//
func (builder *FunctionBuilder) Range(expr int8, kind reflect.Kind) {
	switch kind {
	case reflect.String:
		builder.fn.body = append(builder.fn.body, instruction{op: opRangeString, c: expr})
	default:
		panic("TODO: not implemented")
	}
}

// Func appends a new "Func" instruction to the function body.
//
//     r = func() { ... }
//
func (builder *FunctionBuilder) Func(r int8, typ reflect.Type) *ScrigoFunction {
	b := len(builder.fn.literals)
	if b == 256 {
		panic("scrigoFunctions limit reached")
	}
	builder.allocRegister(reflect.Interface, r)
	fn := &ScrigoFunction{
		typ:    typ,
		parent: builder.fn,
	}
	builder.fn.literals = append(builder.fn.literals, fn)
	builder.fn.body = append(builder.fn.body, instruction{op: opFunc, b: int8(b), c: r})
	return fn
}

// GetFunc appends a new "GetFunc" instruction to the function body.
//
//     z = p.f
//
func (builder *FunctionBuilder) GetFunc(native bool, f int8, z int8) {
	builder.allocRegister(reflect.Interface, z)
	var a int8
	if native {
		a = 1
	}
	builder.fn.body = append(builder.fn.body, instruction{op: opGetFunc, a: a, b: f, c: z})
}

// GetVar appends a new "GetVar" instruction to the function body.
//
//     z = p.v
//
func (builder *FunctionBuilder) GetVar(v uint8, z int8) {
	builder.allocRegister(reflect.Interface, z)
	builder.fn.body = append(builder.fn.body, instruction{op: opGetVar, a: int8(v), c: z})
}

// Go appends a new "Go" instruction to the function body.
//
//     go
//
func (builder *FunctionBuilder) Go() {
	builder.fn.body = append(builder.fn.body, instruction{op: opGo})
}

// Goto appends a new "goto" instruction to the function body.
//
//     goto label
//
func (builder *FunctionBuilder) Goto(label uint32) {
	in := instruction{op: opGoto}
	if label > 0 {
		if label > uint32(len(builder.labels)) {
			panic("bug!") // TODO(Gianluca): remove.
		}
		addr := builder.labels[label-1]
		if addr == 0 {
			builder.gotos[builder.CurrentAddr()] = label
		} else {
			in.a, in.b, in.c = encodeAddr(addr)
		}
	}
	builder.fn.body = append(builder.fn.body, in)
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
func (builder *FunctionBuilder) If(k bool, x int8, o Condition, y int8, kind reflect.Kind) {
	builder.allocRegister(kind, x)
	if !k {
		builder.allocRegister(kind, y)
	}
	var op operation
	switch kindToType(kind) {
	case TypeInt:
		op = opIfInt
	case TypeFloat:
		op = opIfFloat
	case TypeString:
		op = opIfString
	case TypeIface:
		panic("If: invalid type")
	}
	if k {
		op = -op
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: x, b: int8(o), c: y})
}

// Ifc appends a new "Ifc" instruction to the function body.
//
//     x == c
//     x != c
//     x <  c
//     x <= c
//     x >  c
//     x >= c
//     len(x) == c
//     len(x) != c
//     len(x) <  c
//     len(x) <= c
//     len(x) >  c
//     len(x) >= c
//
func (builder *FunctionBuilder) Ifc(x int8, o Condition, c int8, kind reflect.Kind) {
	builder.allocRegister(kind, x)
	var op operation
	switch kind {
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8,
		reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		op = opIfInt
	case reflect.Float64, reflect.Float32:
		op = opIfFloat
	case reflect.String:
		op = opIfString
	default:
		panic("Ifc: invalid type")
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: x, b: int8(o), c: c})
}

// Index appends a new "index" instruction to the function body
//
//	dst = expr[i]
//
func (builder *FunctionBuilder) Index(ki bool, expr, i, dst int8, exprType reflect.Type) {
	kind := exprType.Kind()
	var op operation
	switch kind {
	default:
		op = opIndex
	case reflect.Slice:
		op = opSliceIndex
	case reflect.String:
		op = opStringIndex
	case reflect.Map:
		op = opMapIndex
		switch exprType {
		case reflect.TypeOf(map[string]int{}):
			op = opMapIndexStringInt
		case reflect.TypeOf(map[string]bool{}):
			op = opMapIndexStringBool
		case reflect.TypeOf(map[string]string{}):
			op = opMapIndexStringString
		case reflect.TypeOf(map[string]interface{}{}):
			op = opMapIndexStringInterface
		}
	}
	if ki {
		op = -op
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: expr, b: i, c: dst})
}

// Len appends a new "len" instruction to the function body.
//
//     l = len(s)
//
func (builder *FunctionBuilder) Len(s, l int8, t reflect.Type) {
	builder.allocRegister(reflect.Interface, s)
	builder.allocRegister(reflect.Int, l)
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
	builder.fn.body = append(builder.fn.body, instruction{op: opLen, a: a, b: s, c: l})
}

// MakeChan appends a new "MakeChan" instruction to the function body.
//
//     dst = make(typ, capacity)
//
func (builder *FunctionBuilder) MakeChan(typ int8, kCapacity bool, capacity int8, dst int8) {
	op := opMakeChan
	if kCapacity {
		op = -op
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: typ, b: capacity, c: dst})
}

// MakeMap appends a new "MakeMap" instruction to the function body.
//
//     dst = make(typ, size)
//
func (builder *FunctionBuilder) MakeMap(typ int8, kSize bool, size int8, dst int8) {
	op := opMakeMap
	if kSize {
		op = -op
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: typ, b: size, c: dst})
}

// MakeSlice appends a new "MakeSlice" instruction to the function body.
//
//     make(sliceType, len, cap)
//
func (builder *FunctionBuilder) MakeSlice(kLen, kCap bool, sliceType reflect.Type, len, cap, dst int8) {
	builder.allocRegister(reflect.Interface, dst)
	t := builder.Type(sliceType)
	var k int8
	if len == 0 && cap == 0 {
		k = 1
	} else {
		if kLen {
			k |= 1 << 1
		}
		if kCap {
			k |= 1 << 2
		}
	}
	builder.fn.body = append(builder.fn.body, instruction{op: opMakeSlice, a: t, b: k, c: dst})
	if k > 1 {
		builder.fn.body = append(builder.fn.body, instruction{a: len, b: cap})
	}
}

// Move appends a new "Move" instruction to the function body.
//
//     z = x
//
func (builder *FunctionBuilder) Move(k bool, x, z int8, srcKind, dstKind reflect.Kind) {
	if !k {
		builder.allocRegister(srcKind, x)
	}
	builder.allocRegister(srcKind, z)
	op := opMove
	if k {
		op = -op
	}
	var moveType MoveType
	switch kindToType(dstKind) {
	case TypeInt:
		moveType = IntInt
	case TypeFloat:
		moveType = FloatFloat
	case TypeString:
		moveType = StringString
	case TypeIface:
		switch kindToType(srcKind) {
		case TypeInt:
			moveType = IntGeneral
		case TypeFloat:
			moveType = FloatGeneral
		case TypeString:
			moveType = StringGeneral
		case TypeIface:
			moveType = GeneralGeneral
		}
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: int8(moveType), b: x, c: z})
}

// Mul appends a new "mul" instruction to the function body.
//
//     z = x * y
//
func (builder *FunctionBuilder) Mul(ky bool, x, y, z int8, kind reflect.Kind) {
	builder.allocRegister(kind, x)
	builder.allocRegister(kind, y)
	builder.allocRegister(kind, z)
	var op operation
	switch kind {
	case reflect.Int, reflect.Int64, reflect.Uint64:
		op = opMulInt
	case reflect.Int32, reflect.Uint32:
		op = opMulInt32
	case reflect.Int16, reflect.Uint16:
		op = opMulInt16
	case reflect.Int8, reflect.Uint8:
		op = opMulInt8
	case reflect.Float64:
		op = opMulFloat64
	case reflect.Float32:
		op = opMulFloat32
	default:
		panic("mul: invalid type")
	}
	if ky {
		op = -op
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: x, b: y, c: z})
}

// New appends a new "new" instruction to the function body.
//
//     z = new(t)
//
func (builder *FunctionBuilder) New(typ reflect.Type, z int8) {
	builder.allocRegister(reflect.Interface, z)
	a := builder.fn.AddType(typ)
	builder.fn.body = append(builder.fn.body, instruction{op: opNew, a: int8(a), c: z})
}

// Nop appends a new "Nop" instruction to the function body.
//
func (builder *FunctionBuilder) Nop() {
	builder.fn.body = append(builder.fn.body, instruction{op: opNone})
}

// Panic appends a new "Panic" instruction to the function body.
//
//     panic(v)
//
func (builder *FunctionBuilder) Panic(v int8, line int) {
	fn := builder.fn
	builder.allocRegister(reflect.Interface, v)
	fn.body = append(fn.body, instruction{op: opPanic, a: v})
	fn.AddLine(uint32(len(fn.body)-1), line)
}

// Print appends a new "Print" instruction to the function body.
//
//     print(arg)
//
func (builder *FunctionBuilder) Print(arg int8) {
	builder.fn.body = append(builder.fn.body, instruction{op: opPrint, a: arg})
}

// Receive appends a new "Receive" instruction to the function body.
//
//	dst = <- ch
//
//	dst, ok = <- ch
//
func (builder *FunctionBuilder) Receive(ch, ok, dst int8) {
	builder.fn.body = append(builder.fn.body, instruction{op: opReceive, a: ch, b: ok, c: dst})
}

// Recover appends a new "Recover" instruction to the function body.
//
//     recover()
//
func (builder *FunctionBuilder) Recover(r int8) {
	builder.allocRegister(reflect.Interface, r)
	builder.fn.body = append(builder.fn.body, instruction{op: opRecover, c: r})
}

// Rem appends a new "rem" instruction to the function body.
//
//     z = x % y
//
func (builder *FunctionBuilder) Rem(ky bool, x, y, z int8, kind reflect.Kind) {
	builder.allocRegister(kind, x)
	builder.allocRegister(kind, y)
	builder.allocRegister(kind, z)
	var op operation
	switch kind {
	case reflect.Int, reflect.Int64:
		op = opRemInt
	case reflect.Int32:
		op = opRemInt32
	case reflect.Int16:
		op = opRemInt16
	case reflect.Int8:
		op = opRemInt8
	case reflect.Uint, reflect.Uint64:
		op = opRemUint64
	case reflect.Uint32:
		op = opRemUint32
	case reflect.Uint16:
		op = opRemUint16
	case reflect.Uint8:
		op = opRemUint8
	default:
		panic("rem: invalid type")
	}
	if ky {
		op = -op
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: x, b: y, c: z})
}

// Return appends a new "return" instruction to the function body.
//
//     return
//
func (builder *FunctionBuilder) Return() {
	builder.fn.body = append(builder.fn.body, instruction{op: opReturn})
}

// Selector appends a new "Selector" instruction to the function body.
//
// 	c = a.field
//
func (builder *FunctionBuilder) Selector(a, field, c int8) {
	builder.fn.body = append(builder.fn.body, instruction{op: opSelector, a: a, b: field, c: c})
}

// Send appends a new "Send" instruction to the function body.
//
//	ch <- v
//
func (builder *FunctionBuilder) Send(ch, v int8) {
	// TODO(Gianluca): how can send know kind/type?
	builder.fn.body = append(builder.fn.body, instruction{op: opSend, a: v, c: ch})
}

// SetVar appends a new "SetVar" instruction to the function body.
//
//     p.v = r
//
func (builder *FunctionBuilder) SetVar(r int8, v uint8) {
	builder.fn.body = append(builder.fn.body, instruction{op: opSetVar, b: r, c: int8(v)})
}

// SetMap appends a new "SetMap" instruction to the function body.
//
//	m[key] = value
//
func (builder *FunctionBuilder) SetMap(k bool, m, value, key int8) {
	op := opSetMap
	if k {
		op = -op
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: m, b: value, c: key})
}

// SetSlice appends a new "SetSlice" instruction to the function body.
//
//	slice[index] = value
//
func (builder *FunctionBuilder) SetSlice(k bool, slice, value, index int8, elemKind reflect.Kind) {
	_ = elemKind // TODO(Gianluca): remove.
	in := instruction{op: opSetSlice, a: slice, b: value, c: index}
	if k {
		in.op = -in.op
	}
	builder.fn.body = append(builder.fn.body, in)
}

// Sub appends a new "Sub" instruction to the function body.
//
//     z = x - y
//
func (builder *FunctionBuilder) Sub(k bool, x, y, z int8, kind reflect.Kind) {
	builder.allocRegister(reflect.Int, x)
	if !k {
		builder.allocRegister(reflect.Int, y)
	}
	builder.allocRegister(reflect.Int, z)
	var op operation
	switch kind {
	case reflect.Int, reflect.Int64, reflect.Uint64:
		op = opSubInt
	case reflect.Int32, reflect.Uint32:
		op = opSubInt32
	case reflect.Int16, reflect.Uint16:
		op = opSubInt16
	case reflect.Int8, reflect.Uint8:
		op = opSubInt8
	case reflect.Float64:
		op = opSubFloat64
	case reflect.Float32:
		op = opSubFloat32
	default:
		panic("sub: invalid type")
	}
	if k {
		op = -op
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: x, b: y, c: z})
}

// SubInv appends a new "SubInv" instruction to the function body.
//
//     z = y - x
//
func (builder *FunctionBuilder) SubInv(k bool, x, y, z int8, kind reflect.Kind) {
	builder.allocRegister(reflect.Int, x)
	if !k {
		builder.allocRegister(reflect.Int, y)
	}
	builder.allocRegister(reflect.Int, z)
	var op operation
	switch kind {
	case reflect.Int, reflect.Int64, reflect.Uint64:
		op = opSubInvInt
	case reflect.Int32, reflect.Uint32:
		op = opSubInvInt32
	case reflect.Int16, reflect.Uint16:
		op = opSubInvInt16
	case reflect.Int8, reflect.Uint8:
		op = opSubInvInt8
	case reflect.Float64:
		op = opSubInvFloat64
	case reflect.Float32:
		op = opSubInvFloat32
	default:
		panic("subInv: invalid type")
	}
	if k {
		op = -op
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: x, b: y, c: z})
}

// TailCall appends a new "TailCall" instruction to the function body.
//
//     f()
//
func (builder *FunctionBuilder) TailCall(f int8, line int) {
	var fn = builder.fn
	fn.body = append(fn.body, instruction{op: opTailCall, a: f})
	fn.AddLine(uint32(len(fn.body)-1), line)
}
