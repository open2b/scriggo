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

	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/compiler/types"
	"github.com/open2b/scriggo/runtime"
)

// Define some constants that define limits of the implementation.
const (
	// Functions.
	maxFunctionsCount           = 256
	maxRegistersCount           = 127
	maxPredefinedFunctionsCount = 256
	maxScriggoFunctionsCount    = 256

	// Types.
	maxTypesCount = 256

	// Constants.
	maxIntConstantsCount     = 256
	maxStringConstantsCount  = maxIntConstantsCount
	maxGeneralConstantsCount = maxIntConstantsCount
	maxFloatConstantsCount   = maxIntConstantsCount
)

var intType = reflect.TypeOf(0)
var float64Type = reflect.TypeOf(0.0)
var float32Type = reflect.TypeOf(float32(0.0))
var complex128Type = reflect.TypeOf(0i)
var complex64Type = reflect.TypeOf(complex64(0))
var stringType = reflect.TypeOf("")
var emptyInterfaceType = reflect.TypeOf(&[]interface{}{interface{}(nil)}[0]).Elem()

type label runtime.Addr

// encodeRenderContext encodes a runtime.Renderer context.
func encodeRenderContext(ctx ast.Context, inURL, isURLSet bool) uint8 {
	c := uint8(ctx)
	if inURL {
		if isURLSet {
			c |= 0b11000000
		} else {
			c |= 0b10000000
		}
	}
	return c
}

// decodeRenderContext decodes a runtime.Renderer context.
func decodeRenderContext(c uint8) (ast.Context, bool, bool) {
	ctx := ast.Context(c & 0b00001111)
	inURL := c&0b10000000 != 0
	isURLSet := false
	if inURL {
		isURLSet = c&0b01000000 != 0
	}
	return ctx, inURL, isURLSet
}

func encodeInt16(v int16) (a, b int8) {
	a = int8(v >> 8)
	b = int8(v)
	return
}

func decodeInt16(a, b int8) int16 {
	return int16(int(a)<<8 | int(uint8(b)))
}

func encodeUint16(v uint16) (a, b int8) {
	a = int8(uint8(v >> 8))
	b = int8(uint8(v))
	return
}

func decodeUint16(a, b int8) uint16 {
	return uint16(uint8(a))<<8 | uint16(uint8(b))
}

func encodeUint24(v uint32) (a, b, c int8) {
	a = int8(uint8(v >> 16))
	b = int8(uint8(v >> 8))
	c = int8(uint8(v))
	return
}

func decodeUint24(a, b, c int8) uint32 {
	return uint32(uint8(a))<<16 | uint32(uint8(b))<<8 | uint32(uint8(c))
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

// newFunction returns a new function with a given package, name and type.
// file and pos are, respectively, the file and the position where the
// function is declared.
func newFunction(pkg, name string, typ reflect.Type, file string, pos *ast.Position) *runtime.Function {
	fn := runtime.Function{
		Pkg:  pkg,
		Name: name,
		Type: typ,
		File: file,
	}
	if pos != nil {
		fn.Pos = &runtime.Position{
			Line:   pos.Line,
			Column: pos.Column,
			Start:  pos.Start,
			End:    pos.End,
		}
	}
	return &fn
}

// newMacro returns a new macro with a given package, name and type. format,
// file and pos are, respectively, the format, the file and the position where
// the macro is declared.
func newMacro(pkg, name string, typ reflect.Type, format ast.Format, file string, pos *ast.Position) *runtime.Function {
	fn := runtime.Function{
		Pkg:    pkg,
		Name:   name,
		Macro:  true,
		Format: uint8(format),
		Type:   typ,
		File:   file,
	}
	if pos != nil {
		fn.Pos = &runtime.Position{
			Line:   pos.Line,
			Column: pos.Column,
			Start:  pos.Start,
			End:    pos.End,
		}
	}
	return &fn
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
	complexBinaryOpIndexes map[ast.OperatorType]int8 // indexes of complex binary op. functions.
	complexUnaryOpIndex    int8                      // index of complex negation function.

	// text refers to the latest emitted Text instruction with its text to be flushed into the function.
	text struct {
		addr  runtime.Addr
		txt   [][]byte
		inURL bool
	}

	// path of the current file. For example, when emitting a "render <path>"
	// expression in a template the file path changes even if the function
	// remains the same.
	path string
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
func (fb *functionBuilder) currentStackShift() runtime.StackShift {
	return runtime.StackShift{
		fb.numRegs[intRegister],
		fb.numRegs[floatRegister],
		fb.numRegs[stringRegister],
		fb.numRegs[generalRegister],
	}
}

// enterScope enters a new scope.
// Every enterScope call must be paired with a corresponding exitScope call.
func (fb *functionBuilder) enterScope() {
	fb.scopes = append(fb.scopes, map[string]int8{})
	fb.enterStack()
}

// exitScope exits last scope.
// Every exitScope call must be paired with a corresponding enterScope call.
func (fb *functionBuilder) exitScope() {
	fb.scopes = fb.scopes[:len(fb.scopes)-1]
	fb.exitStack()
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
func (fb *functionBuilder) enterStack() {
	scopeShift := fb.currentStackShift()
	fb.scopeShifts = append(fb.scopeShifts, scopeShift)
}

// exitStack exits current virtual stack, allowing its registers to be reused
// (if necessary).
// Every exitStack call must be paired with a corresponding enterStack call.
// See enterStack documentation for further details and usage.
func (fb *functionBuilder) exitStack() {
	shift := fb.scopeShifts[len(fb.scopeShifts)-1]
	fb.numRegs[intRegister] = shift[intRegister]
	fb.numRegs[floatRegister] = shift[floatRegister]
	fb.numRegs[stringRegister] = shift[stringRegister]
	fb.numRegs[generalRegister] = shift[generalRegister]
	fb.scopeShifts = fb.scopeShifts[:len(fb.scopeShifts)-1]
}

// newRegister makes a new register of a given kind.
func (fb *functionBuilder) newRegister(kind reflect.Kind) int8 {
	t := kindToType(kind)
	num := fb.numRegs[t]
	if num == maxRegistersCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "%s registers count exceeded %d", t, maxRegistersCount))
	}
	fb.allocRegister(t, num+1)
	return num + 1
}

// newIndirectRegister allocates a new indirect register.
func (fb *functionBuilder) newIndirectRegister() int8 {
	return -fb.newRegister(reflect.Interface)
}

// bindVarReg binds name with register reg. To create a new variable, use
// VariableRegister in conjunction with bindVarReg.
func (fb *functionBuilder) bindVarReg(name string, reg int8) {
	fb.scopes[len(fb.scopes)-1][name] = reg
}

// isLocalVariable reports whether v is a local variable.
func (fb *functionBuilder) isLocalVariable(v string) bool {
	for i := len(fb.scopes) - 1; i >= 0; i-- {
		_, ok := fb.scopes[i][v]
		if ok {
			return true
		}
	}
	return false
}

// scopeLookup returns n's register.
func (fb *functionBuilder) scopeLookup(n string) int8 {
	for i := len(fb.scopes) - 1; i >= 0; i-- {
		reg, ok := fb.scopes[i][n]
		if ok {
			return reg
		}
	}
	panic(fmt.Sprintf("bug: %s not found", n))
}

func (fb *functionBuilder) addPosAndPath(pos *ast.Position) {
	pc := runtime.Addr(len(fb.fn.Body))
	if fb.fn.DebugInfo == nil {
		fb.fn.DebugInfo = map[runtime.Addr]runtime.DebugInfo{}
	}
	debugInfo := fb.fn.DebugInfo[pc]
	if pos == nil {
		debugInfo.Position.Line = 1
		debugInfo.Position.Column = 1
	} else {
		debugInfo.Position.Line = pos.Line
		debugInfo.Position.Column = pos.Column
		debugInfo.Position.Start = pos.Start
		debugInfo.Position.End = pos.End
	}
	debugInfo.Path = fb.path
	fb.fn.DebugInfo[pc] = debugInfo
}

// addOperandKinds adds the kind of the three operands of the next instruction.
// If an operand has no kind (or if that kind is not meaningful) it is legal to
// pass the zero of reflect.Kind for such operand.
func (fb *functionBuilder) addOperandKinds(a, b, c reflect.Kind) {
	pc := runtime.Addr(len(fb.fn.Body))
	if fb.fn.DebugInfo == nil {
		fb.fn.DebugInfo = map[runtime.Addr]runtime.DebugInfo{}
	}
	debugInfo := fb.fn.DebugInfo[pc]
	debugInfo.OperandKind = [3]reflect.Kind{a, b, c}
	fb.fn.DebugInfo[pc] = debugInfo
}

// addFunctionType adds the type to the next function call instruction as a
// debug information. Note that it's not necessary to call this method for Call
// and CallPredefined instructions because the type of the function is already
// stored into the Functions and Predefined slices.
func (fb *functionBuilder) addFunctionType(typ reflect.Type) {
	pc := runtime.Addr(len(fb.fn.Body))
	if fb.fn.DebugInfo == nil {
		fb.fn.DebugInfo = map[runtime.Addr]runtime.DebugInfo{}
	}
	debugInfo := fb.fn.DebugInfo[pc]
	debugInfo.FuncType = typ
	fb.fn.DebugInfo[pc] = debugInfo
}

// changePath changes the current path. Note that the path is initially set at
// the creation of the function builder; this method should be called only when
// the path changes during the building of the same function, for example when
// emitting a render expression.
func (fb *functionBuilder) changePath(newPath string) {
	fb.path = newPath
}

// getPath returns the current path.
func (fb *functionBuilder) getPath() string {
	return fb.path
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
func (fb *functionBuilder) addType(typ reflect.Type, preserveType bool) int {
	if !preserveType {
		if st, ok := typ.(types.ScriggoType); ok {
			typ = st.Underlying()
		}
	}
	fn := fb.fn
	for i, t := range fn.Types {
		if t == typ {
			return i
		}
	}
	index := len(fn.Types)
	if index == maxTypesCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "types count exceeded %d", maxTypesCount))
	}
	fn.Types = append(fn.Types, typ)
	return index
}

// addPredefinedFunction adds a predefined function to the builder's function.
func (fb *functionBuilder) addPredefinedFunction(f *runtime.PredefinedFunction) uint8 {
	fn := fb.fn
	r := len(fn.Predefined)
	if r == maxPredefinedFunctionsCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "predefined functions count exceeded %d", maxPredefinedFunctionsCount))
	}
	fn.Predefined = append(fn.Predefined, f)
	return uint8(r)
}

// addFunction adds a function to the builder's function.
func (fb *functionBuilder) addFunction(f *runtime.Function) uint8 {
	fn := fb.fn
	r := len(fn.Functions)
	if r == maxScriggoFunctionsCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "Scriggo functions count exceeded %d", maxScriggoFunctionsCount))
	}
	fn.Functions = append(fn.Functions, f)
	return uint8(r)
}

// makeStringConstant makes a new string constant, returning it's index.
func (fb *functionBuilder) makeStringConstant(c string) int8 {
	for i, v := range fb.fn.Constants.String {
		if c == v {
			return int8(i)
		}
	}
	r := len(fb.fn.Constants.String)
	if r == maxStringConstantsCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "string count exceeded %d", maxStringConstantsCount))
	}
	fb.fn.Constants.String = append(fb.fn.Constants.String, c)
	return int8(r)
}

// makeGeneralConstant makes a new general constant, returning it's index.
// c must be nil (the zero reflect.Value) or must have a comparable type or
// must be the zero value of its type.
//
// If the VM's internal representation of c is different from the external, c
// must always have the external representation. Any conversion, if needed, will
// be internally handled.
func (fb *functionBuilder) makeGeneralConstant(c reflect.Value) int8 {
	// Check if a constant with the same value has already been added to the
	// general Constants slice.
	if c.IsValid() {
		t := c.Type()
		if t.Comparable() {
			ci := c.Interface()
			for i, v := range fb.fn.Constants.General {
				if v.IsValid() && ci == v.Interface() {
					return int8(i)
				}
			}
		} else {
			for i, v := range fb.fn.Constants.General {
				if v.IsValid() && t == v.Type() {
					return int8(i)
				}
			}
		}
	} else {
		for i, v := range fb.fn.Constants.General {
			if !v.IsValid() {
				return int8(i)
			}
		}
	}
	r := len(fb.fn.Constants.General)
	if r == maxGeneralConstantsCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "general constants count exceeded %d", maxGeneralConstantsCount))
	}
	fb.fn.Constants.General = append(fb.fn.Constants.General, c)
	return int8(r)
}

// makeFloatConstant makes a new float constant, returning it's index.
func (fb *functionBuilder) makeFloatConstant(c float64) int8 {
	for i, v := range fb.fn.Constants.Float {
		if c == v {
			return int8(i)
		}
	}
	r := len(fb.fn.Constants.Float)
	if r == maxFloatConstantsCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "floating-point count exceeded %d", maxFloatConstantsCount))
	}
	fb.fn.Constants.Float = append(fb.fn.Constants.Float, c)
	return int8(r)
}

// makeIntConstant makes a new int constant, returning it's index.
func (fb *functionBuilder) makeIntConstant(c int64) int8 {
	for i, v := range fb.fn.Constants.Int {
		if c == v {
			return int8(i)
		}
	}
	r := len(fb.fn.Constants.Int)
	if r == maxIntConstantsCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "integer count exceeded %d", maxIntConstantsCount))
	}
	fb.fn.Constants.Int = append(fb.fn.Constants.Int, c)
	return int8(r)
}

// currentAddr returns builder's current address.
func (fb *functionBuilder) currentAddr() runtime.Addr {
	return runtime.Addr(len(fb.fn.Body))
}

// newLabel creates a new empty label. Use setLabelAddr to associate an
// address to it.
func (fb *functionBuilder) newLabel() label {
	fb.labelAddrs = append(fb.labelAddrs, runtime.Addr(0))
	return label(len(fb.labelAddrs))
}

// setLabelAddr sets label's address as builder's current address.
func (fb *functionBuilder) setLabelAddr(lab label) {
	fb.labelAddrs[lab-1] = fb.currentAddr()
}

// flushText flushes the buffered text of the emitted Text instructions.
func (fb *functionBuilder) flushText() {
	if len(fb.text.txt) == 0 {
		return
	}
	var size int
	for _, b := range fb.text.txt {
		size += len(b)
	}
	text := make([]byte, 0, size)
	for _, b := range fb.text.txt {
		text = append(text, b...)
	}
	fb.text.txt = fb.text.txt[0:0]
	fb.fn.Text = append(fb.fn.Text, text)
}

func (fb *functionBuilder) end() {
	fn := fb.fn
	fb.flushText()
	fb.emitReturn()
	// https://github.com/open2b/scriggo/issues/537
	// if len(fn.Body) == 0 || fn.Body[len(fn.Body)-1].Op != runtime.OpReturn {
	// 	fb.emitReturn()
	// }
	if len(fn.Body) > math.MaxUint32 {
		panic(newLimitExceededError(fn.Pos, fn.File, "instructions count exceeded %d", math.MaxUint32))
	}
	for addr, label := range fb.gotos {
		i := fn.Body[addr]
		i.A, i.B, i.C = encodeUint24(uint32(fb.labelAddrs[label-1]))
		fn.Body[addr] = i
	}
	fb.gotos = nil
	for typ, num := range fb.maxRegs {
		if num > fn.NumReg[typ] {
			fn.NumReg[typ] = num
		}
	}
}

func (fb *functionBuilder) allocRegister(typ registerType, reg int8) {
	if max, ok := fb.maxRegs[typ]; !ok || reg > max {
		fb.maxRegs[typ] = reg
	}
	if num, ok := fb.numRegs[typ]; !ok || reg > num {
		fb.numRegs[typ] = reg
	}
}

// complexOperationIndex returns the index of the function which performs the
// binary or unary operation specified by op.
func (fb *functionBuilder) complexOperationIndex(op ast.OperatorType, unary bool) int8 {
	if unary {
		if fb.complexUnaryOpIndex != -1 {
			return fb.complexUnaryOpIndex
		}
		fn := newPredefinedFunction("scriggo.complex", "neg", negComplex)
		index := int8(fb.addPredefinedFunction(fn))
		fb.complexUnaryOpIndex = index
		return index
	}
	if index, ok := fb.complexBinaryOpIndexes[op]; ok {
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
	index := int8(fb.addPredefinedFunction(fn))
	fb.complexBinaryOpIndexes[op] = index
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
