// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"errors"
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
	}
	panic("unknown condition")
}

type NativeFunction struct {
	name   string
	fast   interface{}
	value  reflect.Value
	in     []Kind
	out    []Kind
	args   []reflect.Value
	outOff [4]int8
}

func NewNativeFunction(name string, fn interface{}) *NativeFunction {
	return &NativeFunction{name: name, fast: fn}
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

type Package struct {
	name            string
	packages        []*Package
	scrigoFunctions []*ScrigoFunction
	nativeFunctions []*NativeFunction
	variables       []interface{}
	varNames        []string

	// TODO (Gianluca): are these fields exported to the vm? In such case,
	// these should be kept in compiler, not here!
	functionsNames   map[string]int8
	gofunctionsNames map[string]int8
	isGoPkg          map[string]bool
	packagesNames    map[string]uint8
}

// NewPackage creates a new package.
func NewPackage(name string) *Package {
	p := &Package{
		functionsNames:   make(map[string]int8),
		gofunctionsNames: make(map[string]int8),
		isGoPkg:          make(map[string]bool),
		name:             name,
		packagesNames:    make(map[string]uint8),
	}

	p.packages = []*Package{}
	return p
}

// Import imports a package.
func (p *Package) Import(pkg *Package) uint8 {
	if len(p.packages) == 254 {
		panic("imported packages limit reached")
	}
	p.packages = append(p.packages, pkg)
	return uint8(len(p.packages) - 1)
}

// DefineVariable defines a variable.
func (p *Package) DefineVariable(name string, v interface{}) uint8 {
	index := len(p.variables)
	if index == 256 {
		panic("variables limit reached")
	}
	p.variables = append(p.variables, v)
	if name != "" {
		if p.varNames == nil {
			p.varNames = make([]string, len(p.variables))
		}
		p.varNames[index] = name
	}
	return uint8(index)
}

// Function returns a Scrigo function by name. Returns an error if the
// function does not exists.
func (p *Package) Function(name string) (*ScrigoFunction, error) {
	for _, fn := range p.scrigoFunctions {
		if fn.name == name {
			return fn, nil
		}
	}
	return nil, errors.New("function does not exist")
}

type instruction struct {
	op      operation
	a, b, c int8
}

type Upper struct {
	typ   Type
	index int32
}

// ScrigoFunction represents a Scrigo function.
type ScrigoFunction struct {
	name      string
	file      string
	line      int
	typ       reflect.Type
	pkg       *Package // opCallDirect, opGetFunc, opGetVar, opSetVar
	parent    *ScrigoFunction
	crefs     []int16           // opFunc
	literals  []*ScrigoFunction // opFunc
	types     []reflect.Type    // opAlloc, opAssert, opMakeMap, opMakeSlice, opNew
	regnum    [4]uint8          // opCall, opCallDirect
	constants registers
	body      []instruction // run, opCall, opCallDirect
	lines     []int
}

// AddNativeFunction adds a native function to a package.
func (p *Package) AddNativeFunction(fn *NativeFunction) (uint8, bool) {
	r := len(p.nativeFunctions)
	if r > 255 {
		return 0, false
	}
	p.nativeFunctions = append(p.nativeFunctions, fn)
	return uint8(r), true
}

// NewFunction creates a new Scrigo function and appends it to the package.
func (p *Package) NewFunction(name string, typ reflect.Type) (fun *ScrigoFunction, index int8) {
	if len(p.scrigoFunctions) > 255 {
		panic("scrigoFunctions limit reached")
	}
	fn := &ScrigoFunction{
		name: name,
		pkg:  p,
		typ:  typ,
	}
	p.scrigoFunctions = append(p.scrigoFunctions, fn)
	return fn, int8(len(p.scrigoFunctions) - 1)
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
	fn         *ScrigoFunction
	gotos      map[uint32]uint32
	labels     []uint32
	numRegs    map[reflect.Kind]uint8
	scopes     []map[string]int8
	stackShift StackShift
}

// Builder returns the body of the function.
func (fn *ScrigoFunction) Builder() *FunctionBuilder {
	fn.body = nil
	return &FunctionBuilder{
		fn:      fn,
		gotos:   map[uint32]uint32{},
		numRegs: map[reflect.Kind]uint8{},
		scopes:  []map[string]int8{},
	}
}

// EnterScope enters a new scope.
func (builder *FunctionBuilder) EnterScope() {
	builder.scopes = append(builder.scopes, map[string]int8{})
}

// ExitScope exists the last scope.
func (builder *FunctionBuilder) ExitScope() {
	builder.scopes = builder.scopes[:len(builder.scopes)-1]
}

// NewRegister makes a new register of kind k.
func (builder *FunctionBuilder) NewRegister(k reflect.Kind) int8 {
	reg := int8(builder.numRegs[k]) + 1
	builder.allocRegister(k, reg)
	return reg
}

// BindVarReg binds name with register reg. To create a new
// named-variable, use VariableRegister in conjuction with BindVarReg.
func (builder *FunctionBuilder) BindVarReg(name string, reg int8) {
	builder.scopes[len(builder.scopes)-1][name] = reg
}

// ScopeLookup returns the register in which variable with name n is stored.
func (builder *FunctionBuilder) ScopeLookup(n string) int8 {
	for i := len(builder.scopes) - 1; i >= 0; i-- {
		reg, ok := builder.scopes[i][n]
		if ok {
			return reg
		}
	}
	panic(fmt.Sprintf("bug: %s not found", n))
}

func (builder *FunctionBuilder) MakeStringConstant(c string) int8 {
	r := len(builder.fn.constants.String)
	if r > 255 {
		panic("string refs limit reached")
	}
	builder.fn.constants.String = append(builder.fn.constants.String, c)
	return int8(r)
}

func (builder *FunctionBuilder) MakeFloatConstant(c float64) int8 {
	r := len(builder.fn.constants.Float)
	if r > 255 {
		panic("float refs limit reached")
	}
	builder.fn.constants.Float = append(builder.fn.constants.Float, c)
	return int8(r)
}

func (builder *FunctionBuilder) MakeIntConstant(c int64) int8 {
	r := len(builder.fn.constants.Int)
	if r > 255 {
		panic("int refs limit reached")
	}
	builder.fn.constants.Int = append(builder.fn.constants.Int, c)
	return int8(r)
}

// MakeMap appends a new "MakeMap" instruction to the function body.
//
//     c = make(typ, size, dst)
//
func (builder *FunctionBuilder) MakeMap(typ int8, kSize bool, size int8, dst int8) {
	op := opMakeMap
	if kSize {
		op = -op
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: typ, b: size, c: dst})
}

func (builder *FunctionBuilder) MakeInterfaceConstant(c interface{}) int8 {
	r := -len(builder.fn.constants.General) - 1
	if r == -129 {
		panic("interface refs limit reached")
	}
	builder.fn.constants.General = append(builder.fn.constants.General, c)
	return int8(r)
}

func (builder *FunctionBuilder) CurrentAddr() uint32 {
	return uint32(len(builder.fn.body))
}

func (builder *FunctionBuilder) NewLabel() uint32 {
	builder.labels = append(builder.labels, uint32(0))
	return uint32(len(builder.labels))
}

func (builder *FunctionBuilder) SetLabelAddr(l uint32) {
	builder.labels[l-1] = builder.CurrentAddr()
}

var intType = reflect.TypeOf(0)
var float64Type = reflect.TypeOf(0.0)
var stringType = reflect.TypeOf("")

func encodeAddr(v uint32) (a, b, c int8) {
	a = int8(uint8(v))
	b = int8(uint8(v >> 8))
	c = int8(uint8(v >> 16))
	return
}

// Type returns the index of typ inside the types slice, creating a new entry if
// necessary.
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

func (builder *FunctionBuilder) End() {
	fn := builder.fn
	for addr, label := range builder.gotos {
		i := fn.body[addr]
		i.a, i.b, i.c = encodeAddr(builder.labels[label-1])
		fn.body[addr] = i
	}
	builder.gotos = nil
	for kind, num := range builder.numRegs {
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
	if reg > 0 {
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
func (builder *FunctionBuilder) Call(p int8, f int8, shift StackShift) {
	var fn = builder.fn
	if p != NoPackage {
		builder.allocRegister(reflect.Interface, int8(f))
	}
	fn.body = append(fn.body, instruction{op: opCall, a: p, b: f, c: NoVariadic})
	fn.body = append(fn.body, instruction{op: operation(shift[0]), a: shift[1], b: shift[2], c: shift[3]})
}

// CallFunc appends a new "CallFunc" instruction to the function body.
//
//     p.F()
//
func (builder *FunctionBuilder) CallFunc(p int8, f int8, numVariadic int8, shift StackShift) {
	var fn = builder.fn
	fn.body = append(fn.body, instruction{op: opCallFunc, a: p, b: f, c: numVariadic})
	fn.body = append(fn.body, instruction{op: operation(shift[0]), a: shift[1], b: shift[2], c: shift[3]})
}

// CallIndirect appends a new "CallIndirect" instruction to the function body.
//
//     f()
//
func (builder *FunctionBuilder) CallIndirect(f int8, numVariadic int8, shift StackShift) {
	var fn = builder.fn
	fn.body = append(fn.body, instruction{op: opCallIndirect, b: f, c: numVariadic})
	fn.body = append(fn.body, instruction{op: operation(shift[0]), a: shift[1], b: shift[2], c: shift[3]})
}

// CallMethod appends a new "CallMethod" instruction to the function body.
//
//     p.M()
//
func (builder *FunctionBuilder) CallMethod(typ reflect.Type, m int8, numVariadic int8, shift StackShift) {
	var fn = builder.fn
	a := builder.fn.AddType(typ)
	fn.body = append(fn.body, instruction{op: opCallMethod, a: int8(a), b: m, c: numVariadic})
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

// Copy appends a new "copy" instruction to the function body.
//
//     copy(dst, src)
//
func (builder *FunctionBuilder) Copy(dst, src int8) {
	builder.allocRegister(reflect.Interface, dst)
	builder.allocRegister(reflect.Interface, src)
	builder.fn.body = append(builder.fn.body, instruction{op: opCopy, a: dst, b: src})
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
func (builder *FunctionBuilder) Div(x, y, z int8, kind reflect.Kind) {
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
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: x, b: y, c: z})
}

// ForRange appends a new "ForRange" instruction to the function body.
//
//	TODO
//
func (builder *FunctionBuilder) ForRange(expr int8, kind reflect.Kind) {
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
func (builder *FunctionBuilder) GetFunc(p, f int8, z int8) {
	builder.allocRegister(reflect.Interface, z)
	builder.fn.body = append(builder.fn.body, instruction{op: opGetFunc, a: p, b: f, c: z})
}

// GetVar appends a new "GetVar" instruction to the function body.
//
//     z = p.v
//
func (builder *FunctionBuilder) GetVar(p, v uint8, z int8) {
	builder.allocRegister(reflect.Interface, z)
	builder.fn.body = append(builder.fn.body, instruction{op: opGetVar, a: int8(p), b: int8(v), c: z})
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
			panic("bug!")
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
	switch kind {
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8,
		reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		op = opIfInt
	case reflect.Float64, reflect.Float32:
		op = opIfFloat
	case reflect.String:
		op = opIfString
	default:
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

// JmpOk appends a new "jmpok" instruction to the function body.
//
//     jmpok label
//
func (builder *FunctionBuilder) JmpOk(label uint32) {
	in := instruction{op: opJmpOk}
	if label > 0 {
		if label <= uint32(len(builder.labels)) {
			in.a, in.b, in.c = encodeAddr(builder.labels[label-1])
		} else {
			builder.gotos[builder.CurrentAddr()] = label - 1
		}
	}
	builder.fn.body = append(builder.fn.body, in)
}

// JmpNotOk appends a new "jmpnotok" instruction to the function body.
//
//     jmpnotok label
//
func (builder *FunctionBuilder) JmpNotOk(label uint32) {
	in := instruction{op: opJmpNotOk}
	if label > 0 {
		if label <= uint32(len(builder.labels)) {
			in.a, in.b, in.c = encodeAddr(builder.labels[label-1])
		} else {
			builder.gotos[builder.CurrentAddr()] = label - 1
		}
	}
	builder.fn.body = append(builder.fn.body, in)
}

// Len appends a new "len" instruction to the function body.
//
//     l = len(s)
//
func (builder *FunctionBuilder) Len(s, l int8, t reflect.Type) {
	builder.allocRegister(reflect.Interface, s)
	builder.allocRegister(reflect.Int, l)
	// TODO
	builder.fn.body = append(builder.fn.body, instruction{op: 0, a: s, b: l})
}

// Map appends a new "map" instruction to the function body.
//
//     z = map(t, n)
//
func (builder *FunctionBuilder) Map(typ reflect.Type, n, z int8) {
	builder.allocRegister(reflect.Int, n)
	builder.allocRegister(reflect.Interface, z)
	a := builder.fn.AddType(typ)
	builder.fn.body = append(builder.fn.body, instruction{op: opMakeMap, a: int8(a), b: n, c: z})
}

// Move appends a new "move" instruction to the function body.
//
//     z = x
//
func (builder *FunctionBuilder) Move(k bool, x, z int8, kind reflect.Kind) {
	if !k {
		builder.allocRegister(kind, x)
	}
	builder.allocRegister(kind, z)
	var op operation
	switch kind {
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8,
		reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		op = opMoveInt
	case reflect.Float64, reflect.Float32:
		op = opMoveFloat
	case reflect.String:
		op = opMoveString
	default:
		op = opMove
	}
	if k {
		op = -op
	}
	builder.fn.body = append(builder.fn.body, instruction{op: op, b: x, c: z})
}

// Mul appends a new "mul" instruction to the function body.
//
//     z = x * y
//
func (builder *FunctionBuilder) Mul(x, y, z int8, kind reflect.Kind) {
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

// Rem appends a new "rem" instruction to the function body.
//
//     z = x % y
//
func (builder *FunctionBuilder) Rem(x, y, z int8, kind reflect.Kind) {
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
	builder.fn.body = append(builder.fn.body, instruction{op: op, a: x, b: y, c: z})
}

// Return appends a new "return" instruction to the function body.
//
//     return
//
func (builder *FunctionBuilder) Return() {
	builder.fn.body = append(builder.fn.body, instruction{op: opReturn})
}

// SetVar appends a new "SetVar" instruction to the function body.
//
//     p.v = r
//
func (builder *FunctionBuilder) SetVar(r int8, p, v uint8) {
	builder.fn.body = append(builder.fn.body, instruction{op: opSetVar, a: r, b: int8(p), c: int8(v)})
}

// Slice appends a new "slice" instruction to the function body.
//
//     slice(t, l, c)
//
func (builder *FunctionBuilder) Slice(typ reflect.Type, l, c, dst int8) {
	builder.allocRegister(reflect.Interface, dst)
	tr := builder.Type(typ)
	builder.fn.body = append(builder.fn.body, instruction{op: opMakeSlice, a: tr, b: 0, c: dst})
	// TODO (Gianluca): needed only if length != 0 || cap != 0
	builder.fn.body = append(builder.fn.body, instruction{op: operation(l), a: c})
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

// TailCall appends a new "TailCall" instruction to the function body.
//
//     f()
//
func (builder *FunctionBuilder) TailCall(p int8, f int8) {
	var fn = builder.fn
	if p != NoPackage {
		builder.allocRegister(reflect.Interface, int8(f))
	}
	fn.body = append(fn.body, instruction{op: opTailCall, a: p, b: f})
}
