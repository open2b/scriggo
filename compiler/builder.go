// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"math"
	"reflect"
	"strconv"

	"scriggo/compiler/ast"
	"scriggo/compiler/types"
	"scriggo/runtime"
)

// Define some constants that define limits of the implementation.
const (
	// Functions.
	maxFunctionsCount           = 255
	maxRegistersCount           = 126
	maxPredefinedFunctionsCount = 255
	maxScriggoFunctionsCount    = 255

	// Types.
	maxTypesCount = 255

	// Constants.
	maxIntConstantsCount     = 255
	maxStringConstantsCount  = maxIntConstantsCount
	maxGeneralConstantsCount = maxIntConstantsCount
	maxFloatConstantsCount   = maxIntConstantsCount

	// Instructions.
	maxNumberOfInstructions = 2<<32 - 1 // X MARCO: what is the correct value?
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

type label runtime.Addr

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
// file and pos are, respectively, the file and the position where the function
// is declared.
func newFunction(pkg, name string, typ reflect.Type, file string, pos *ast.Position) *runtime.Function {
	var runtimePos *runtime.Position
	if pos != nil {
		runtimePos = &runtime.Position{
			Line:   pos.Line,
			Column: pos.Column,
			Start:  pos.Start,
			End:    pos.End,
		}
	}
	return &runtime.Function{Pkg: pkg, Name: name, Type: typ, File: file, Pos: runtimePos}
}

// newPredefinedFunction returns a new predefined function with a given
// package, name and implementation. fn must be a function type.
func newPredefinedFunction(pkg, name string, fn interface{}) *runtime.PredefinedFunction {
	return runtime.NewPredefinedFunction(pkg, name, fn)
}

type functionBuilder struct {
	fn                     *runtime.Function
	labelAddrs             []runtime.Addr // addresses of the labels; the address of the label n is labelAddrs[n-1]
	gotos                  map[runtime.Addr]label
	maxRegs                map[registerType]int8 // max number of registers allocated at the same time.
	numRegs                map[registerType]int8
	scopes                 []map[string]int8
	scopeShifts            []runtime.StackShift
	reserves               []runtime.Addr
	complexBinaryOpIndexes map[ast.OperatorType]int8 // indexes of complex binary op. functions.
	complexUnaryOpIndex    int8                      // index of complex negation function.

	// path of the current file. For example, when emitting an {% include ".." %}
	// statement in a template the file path changes even if the function
	// remains the same.
	path string

	templateRegs struct {
		gA, gB, gC, gD, gE, gF int8 // Reserved general registers.
		iA                     int8 // Reserved int register.
	}
}

// newBuilder returns a new function builder for the function fn in the given
// path.
func newBuilder(fn *runtime.Function, path string) *functionBuilder {
	fn.Body = nil
	builder := &functionBuilder{
		fn:                     fn,
		gotos:                  map[runtime.Addr]label{},
		maxRegs:                map[registerType]int8{},
		numRegs:                map[registerType]int8{},
		scopes:                 []map[string]int8{},
		complexBinaryOpIndexes: map[ast.OperatorType]int8{},
		complexUnaryOpIndex:    -1,
		path:                   path,
	}
	return builder
}

// currentStackShift returns the current stack shift.
func (builder *functionBuilder) currentStackShift() runtime.StackShift {
	return runtime.StackShift{
		builder.numRegs[intRegister],
		builder.numRegs[floatRegister],
		builder.numRegs[stringRegister],
		builder.numRegs[generalRegister],
	}
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
// 		tmp := e.fb.newRegister(..)
// 		// use tmp in some way
// 		// move tmp content to externally-defined reg
// 		e.fb.exitStack()
//	    // tmp location is now available for reusing
//
func (builder *functionBuilder) enterStack() {
	scopeShift := builder.currentStackShift()
	builder.scopeShifts = append(builder.scopeShifts, scopeShift)
}

// exitStack exits current virtual stack, allowing its registers to be reused
// (if necessary).
// Every exitStack call must be paired with a corresponding enterStack call.
// See enterStack documentation for further details and usage.
func (builder *functionBuilder) exitStack() {
	shift := builder.scopeShifts[len(builder.scopeShifts)-1]
	builder.numRegs[intRegister] = shift[intRegister]
	builder.numRegs[floatRegister] = shift[floatRegister]
	builder.numRegs[stringRegister] = shift[stringRegister]
	builder.numRegs[generalRegister] = shift[generalRegister]
	builder.scopeShifts = builder.scopeShifts[:len(builder.scopeShifts)-1]
}

// newRegister makes a new register of a given kind.
func (builder *functionBuilder) newRegister(kind reflect.Kind) int8 {
	t := kindToType(kind)
	num := builder.numRegs[t]
	if num > maxRegistersCount {
		panic(newLimitExceededError(builder.fn.Pos, builder.path, "%s registers count exceeded %d", t, maxRegistersCount))
	}
	builder.allocRegister(t, num+1)
	return num + 1
}

// newIndirectRegister allocates a new indirect register.
func (builder *functionBuilder) newIndirectRegister() int8 {
	return -builder.newRegister(reflect.Interface)
}

// bindVarReg binds name with register reg. To create a new variable, use
// VariableRegister in conjunction with bindVarReg.
func (builder *functionBuilder) bindVarReg(name string, reg int8) {
	builder.scopes[len(builder.scopes)-1][name] = reg
}

// isLocalVariable reports whether v is a local variable.
func (builder *functionBuilder) isLocalVariable(v string) bool {
	for i := len(builder.scopes) - 1; i >= 0; i-- {
		_, ok := builder.scopes[i][v]
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

func (builder *functionBuilder) addPosAndPath(pos *ast.Position) {
	pc := runtime.Addr(len(builder.fn.Body)) + 1
	if builder.fn.DebugInfo == nil {
		builder.fn.DebugInfo = map[runtime.Addr]runtime.DebugInfo{}
	}
	debugInfo := builder.fn.DebugInfo[pc]
	if pos == nil {
		debugInfo.Position.Line = 1
		debugInfo.Position.Column = 1
	} else {
		debugInfo.Position.Line = pos.Line
		debugInfo.Position.Column = pos.Column
		debugInfo.Position.Start = pos.Start
		debugInfo.Position.End = pos.End
	}
	debugInfo.Path = builder.path
	builder.fn.DebugInfo[pc] = debugInfo
}

// addOperandKinds adds the kind of the three operands of the next instruction.
// If an operand has no kind (or if that kind is not meaningful) it is legal to
// pass the zero of reflect.Kind for such operand.
func (builder *functionBuilder) addOperandKinds(a, b, c reflect.Kind) {
	pc := runtime.Addr(len(builder.fn.Body))
	if builder.fn.DebugInfo == nil {
		builder.fn.DebugInfo = map[runtime.Addr]runtime.DebugInfo{}
	}
	debugInfo := builder.fn.DebugInfo[pc]
	debugInfo.OperandKind = [3]reflect.Kind{a, b, c}
	builder.fn.DebugInfo[pc] = debugInfo
}

// addFunctionType adds the type to the next function call instruction as a
// debug information. Note that it's not necessary to call this method for Call
// and CallPredefined instructions because the type of the function is already
// stored into the Functions and Predefined slices.
func (builder *functionBuilder) addFunctionType(typ reflect.Type) {
	pc := runtime.Addr(len(builder.fn.Body))
	if builder.fn.DebugInfo == nil {
		builder.fn.DebugInfo = map[runtime.Addr]runtime.DebugInfo{}
	}
	debugInfo := builder.fn.DebugInfo[pc]
	debugInfo.FuncType = typ
	builder.fn.DebugInfo[pc] = debugInfo
}

// changePath changes the current path. Note that the path is initially set at
// the creation of the function builder; this method should be called only when
// the path changes during the building of the same function, for example when
// emitting an 'include' statement.
func (builder *functionBuilder) changePath(newPath string) {
	builder.path = newPath
}

// getPath returns the current path.
func (builder *functionBuilder) getPath() string {
	return builder.path
}

// addType adds a type to the builder's function, creating it if necessary.
// preserveType controls if typ should be converted to the underlying gc type in
// case of typ is a Scriggo type. So, if preserveType is set to false, this
// method adds typ to the slice of types 'as is', independently from it's
// implementation. Setting 'preserveType' is useful for instructions that need
// to keep the information about the Scriggo type. Note that for every
// instruction of the virtual machine that receives a type 'as is', such type
// must be handled as a special case from the VM; considered that, in the most
// cases you would just simply set 'preserveType' to false.
func (builder *functionBuilder) addType(typ reflect.Type, preserveType bool) int {
	if !preserveType {
		if st, ok := typ.(types.ScriggoType); ok {
			typ = st.Underlying()
		}
	}
	fn := builder.fn
	for i, t := range fn.Types {
		if t == typ {
			return i
		}
	}
	index := len(fn.Types)
	if index > maxTypesCount {
		panic(newLimitExceededError(builder.fn.Pos, builder.path, "types count exceeded %d", maxTypesCount))
	}
	fn.Types = append(fn.Types, typ)
	return index
}

// addPredefinedFunction adds a predefined function to the builder's function.
func (builder *functionBuilder) addPredefinedFunction(f *runtime.PredefinedFunction) uint8 {
	fn := builder.fn
	r := len(fn.Predefined)
	if r > maxPredefinedFunctionsCount {
		panic(newLimitExceededError(builder.fn.Pos, builder.path, "predefined functions count exceeded %d", maxPredefinedFunctionsCount))
	}
	fn.Predefined = append(fn.Predefined, f)
	return uint8(r)
}

// addFunction adds a function to the builder's function.
func (builder *functionBuilder) addFunction(f *runtime.Function) uint8 {
	fn := builder.fn
	r := len(fn.Functions)
	if r > maxScriggoFunctionsCount {
		panic(newLimitExceededError(builder.fn.Pos, builder.path, "Scriggo functions count exceeded %d", maxScriggoFunctionsCount))
	}
	fn.Functions = append(fn.Functions, f)
	return uint8(r)
}

// makeStringConstant makes a new string constant, returning it's index.
func (builder *functionBuilder) makeStringConstant(c string) int8 {
	for i, v := range builder.fn.Constants.String {
		if c == v {
			return int8(i)
		}
	}
	r := len(builder.fn.Constants.String)
	if r > maxStringConstantsCount {
		panic(newLimitExceededError(builder.fn.Pos, builder.path, "string count exceeded %d", maxStringConstantsCount))
	}
	builder.fn.Constants.String = append(builder.fn.Constants.String, c)
	return int8(r)
}

// makeGeneralConstant makes a new general constant, returning it's index.
// c must be nil or must have a comparable type or must be the zero value of
// its type.
//
// If the VM's internal representation of c is different from the external, c
// must always have the external representation. Any conversion, if needed, will
// be internally handled.
func (builder *functionBuilder) makeGeneralConstant(c interface{}) int8 {
	// Check if a constant with the same value has already been added to the
	// general Constants slice.
	if t := reflect.TypeOf(c); c == nil || t.Comparable() {
		for i, v := range builder.fn.Constants.General {
			if c == v {
				return int8(i)
			}
		}
	} else {
		for i, v := range builder.fn.Constants.General {
			if v != nil && t == reflect.TypeOf(v) {
				return int8(i)
			}
		}
	}
	r := len(builder.fn.Constants.General)
	if r > maxGeneralConstantsCount {
		panic(newLimitExceededError(builder.fn.Pos, builder.path, "general constants count exceeded %d", maxGeneralConstantsCount))
	}
	builder.fn.Constants.General = append(builder.fn.Constants.General, c)
	return int8(r)
}

// makeFloatConstant makes a new float constant, returning it's index.
func (builder *functionBuilder) makeFloatConstant(c float64) int8 {
	for i, v := range builder.fn.Constants.Float {
		if c == v {
			return int8(i)
		}
	}
	r := len(builder.fn.Constants.Float)
	if r > maxFloatConstantsCount {
		// REVIEW.
		panic(newLimitExceededError(builder.fn.Pos, builder.path, "floating-point count exceeded %d", maxFloatConstantsCount))
	}
	builder.fn.Constants.Float = append(builder.fn.Constants.Float, c)
	return int8(r)
}

// makeIntConstant makes a new int constant, returning it's index.
func (builder *functionBuilder) makeIntConstant(c int64) int8 {
	for i, v := range builder.fn.Constants.Int {
		if c == v {
			return int8(i)
		}
	}
	r := len(builder.fn.Constants.Int)
	if r > maxIntConstantsCount {
		panic(newLimitExceededError(builder.fn.Pos, builder.path, "integer count exceeded %d", maxIntConstantsCount))
	}
	builder.fn.Constants.Int = append(builder.fn.Constants.Int, c)
	return int8(r)
}

// currentAddr returns builder's current address.
func (builder *functionBuilder) currentAddr() runtime.Addr {
	return runtime.Addr(len(builder.fn.Body))
}

// newLabel creates a new empty label. Use setLabelAddr to associate an
// address to it.
func (builder *functionBuilder) newLabel() label {
	builder.labelAddrs = append(builder.labelAddrs, runtime.Addr(0))
	return label(len(builder.labelAddrs))
}

// setLabelAddr sets label's address as builder's current address.
func (builder *functionBuilder) setLabelAddr(lab label) {
	builder.labelAddrs[lab-1] = builder.currentAddr()
}

func (builder *functionBuilder) end() {
	fn := builder.fn
	builder.emitReturn()
	// https://github.com/open2b/scriggo/issues/537
	// if len(fn.Body) == 0 || fn.Body[len(fn.Body)-1].Op != runtime.OpReturn {
	// 	builder.emitReturn()
	// }
	// Ensure that the number of instructions in the function body is not too
	// big for the runtime.
	if len(fn.Body) > maxNumberOfInstructions {
		panic(newLimitExceededError(fn.Pos, fn.File, "instructions count exceeded %d", maxNumberOfInstructions))
	}
	for addr, label := range builder.gotos {
		i := fn.Body[addr]
		i.A, i.B, i.C = encodeUint24(uint32(builder.labelAddrs[label-1]))
		fn.Body[addr] = i
	}
	builder.gotos = nil
	for typ, num := range builder.maxRegs {
		if num > fn.NumReg[typ] {
			fn.NumReg[typ] = num
		}
	}
	if builder.reserves != nil {
		for _, addr := range builder.reserves {
			var bytes int
			if addr == 0 {
				bytes = runtime.CallFrameSize + 8*int(fn.NumReg[intRegister]+fn.NumReg[floatRegister]) +
					16*int(fn.NumReg[stringRegister]+fn.NumReg[generalRegister])
			} else {
				in := fn.Body[addr+1]
				if in.Op == runtime.OpLoadFunc {
					f := fn.Functions[uint8(in.B)]
					bytes = 32 + len(f.VarRefs)*16
				}
			}
			a, b, c := encodeUint24(uint32(bytes))
			fn.Body[addr] = runtime.Instruction{Op: -runtime.OpReserve, A: a, B: b, C: c}
		}
	}
}

func (builder *functionBuilder) allocRegister(typ registerType, reg int8) {
	if max, ok := builder.maxRegs[typ]; !ok || reg > max {
		builder.maxRegs[typ] = reg
	}
	if num, ok := builder.numRegs[typ]; !ok || reg > num {
		builder.numRegs[typ] = reg
	}
}

// complexOperationIndex returns the index of the function which performs the
// binary or unary operation specified by op.
func (builder *functionBuilder) complexOperationIndex(op ast.OperatorType, unary bool) int8 {
	if unary {
		if builder.complexUnaryOpIndex != -1 {
			return builder.complexUnaryOpIndex
		}
		fn := newPredefinedFunction("scriggo.complex", "neg", negComplex)
		index := int8(builder.addPredefinedFunction(fn))
		builder.complexUnaryOpIndex = index
		return index
	}
	if index, ok := builder.complexBinaryOpIndexes[op]; ok {
		return index
	}
	var f interface{}
	var n string
	switch op {
	case ast.OperatorAddition:
		f = addComplex
		n = "add"
	case ast.OperatorSubtraction:
		f = subComplex
		n = "sub"
	case ast.OperatorMultiplication:
		f = mulComplex
		n = "mul"
	case ast.OperatorDivision:
		f = divComplex
		n = "div"
	}
	_ = n
	fn := newPredefinedFunction("scriggo.complex", n, f)
	index := int8(builder.addPredefinedFunction(fn))
	builder.complexBinaryOpIndexes[op] = index
	return index
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

// TODO: find a better name and description for this function.
func flattenIntegerKind(k reflect.Kind) reflect.Kind {
	switch k {
	case reflect.Bool:
		return reflect.Int64
	case reflect.Int:
		if strconv.IntSize == 32 {
			return reflect.Int32
		} else {
			return reflect.Int64
		}
	case reflect.Uint:
		if strconv.IntSize == 32 {
			return reflect.Uint32
		} else {
			return reflect.Uint64
		}
	case reflect.Uintptr:
		if ^uintptr(0) == math.MaxUint32 {
			return reflect.Uint32
		} else {
			return reflect.Uint64
		}
	default:
		return k
	}
}
