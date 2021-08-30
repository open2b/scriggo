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

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/runtime"
)

// Define some constants that define limits of the implementation.
const (
	// Functions.
	maxRegistersCount        = 127
	maxNativeFunctionsCount  = 256
	maxScriggoFunctionsCount = 256
	maxFieldIndexesCount     = 256
	maxSelectCasesCount      = 65536

	// Types.
	maxTypesCount = 256

	// Values.
	maxIntValuesCount     = 1 << 14 // 16384
	maxFloatValuesCount   = 1 << 14 // 16384
	maxStringValuesCount  = 256
	maxGeneralValuesCount = 256
)

var intType = reflect.TypeOf(0)
var float64Type = reflect.TypeOf(0.0)
var float32Type = reflect.TypeOf(float32(0.0))
var complex128Type = reflect.TypeOf(0i)
var complex64Type = reflect.TypeOf(complex64(0))
var stringType = reflect.TypeOf("")
var emptyInterfaceType = reflect.TypeOf(&[]interface{}{interface{}(nil)}[0]).Elem()

type label runtime.Addr

// encodeRenderContext encodes a runtime.Context.
func encodeRenderContext(ctx ast.Context, inURL, isURLSet bool) runtime.Context {
	c := runtime.Context(ctx)
	if inURL {
		if isURLSet {
			c |= 0b11000000
		} else {
			c |= 0b10000000
		}
	}
	return c
}

// decodeRenderContext decodes a runtime.Context.
func decodeRenderContext(c runtime.Context) (ast.Context, bool, bool) {
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

// encodeValueIndex encodes a value index in the Function.Values slices of the
// runtime.
func encodeValueIndex(t registerType, i int) (a, b int8) {
	a, b = encodeInt16(int16(i))
	a |= int8(t << 6)
	return a, b
}

// decodeValueIndex decodes a value index in the Function.Values slices of the
// runtime.
func decodeValueIndex(a, b int8) (t registerType, i int) {
	return registerType(uint8(a) >> 6), int(decodeUint16(a, b) &^ (3 << 14))
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
		Format: format,
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

// newNativeFunction returns a new native function with a given package, name
// and implementation. fn must be a function type.
func newNativeFunction(pkg, name string, fn interface{}) *runtime.NativeFunction {
	return runtime.NewNativeFunction(pkg, name, fn)
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

// declaredInCurrentScope returns the register where v is stored and true in
// case of v is a variable declared in the current scope, else returns 0 and
// false.
func (fb *functionBuilder) declaredInCurrentScope(v string) (int8, bool) {
	reg, ok := fb.scopes[len(fb.scopes)-1][v]
	return reg, ok
}

// declaredInFunc reports whether v is a variable declared within a function.
func (fb *functionBuilder) declaredInFunc(v string) bool {
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
// and CallNative instructions because the type of the function is already
// stored into the Functions and NativeFunctions slices.
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
		if st, ok := typ.(runtime.ScriggoType); ok {
			typ = st.GoType()
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

// addNativeFunction adds a native function to the builder's function.
func (fb *functionBuilder) addNativeFunction(f *runtime.NativeFunction) int8 {
	fn := fb.fn
	r := len(fn.NativeFunctions)
	if r == maxNativeFunctionsCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "native functions count exceeded %d", maxNativeFunctionsCount))
	}
	fn.NativeFunctions = append(fn.NativeFunctions, f)
	return int8(r)
}

// addFunction adds a function to the builder's function.
func (fb *functionBuilder) addFunction(f *runtime.Function) int8 {
	fn := fb.fn
	r := len(fn.Functions)
	if r == maxScriggoFunctionsCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "Scriggo functions count exceeded %d", maxScriggoFunctionsCount))
	}
	fn.Functions = append(fn.Functions, f)
	return int8(r)
}

// makeStringValue makes a new string value, returning it's index.
func (fb *functionBuilder) makeStringValue(v string) int8 {
	for i, vv := range fb.fn.Values.String {
		if v == vv {
			return int8(i)
		}
	}
	r := len(fb.fn.Values.String)
	if r == maxStringValuesCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "string values count exceeded %d", maxStringValuesCount))
	}
	fb.fn.Values.String = append(fb.fn.Values.String, v)
	return int8(r)
}

// makeGeneralValue makes a new general value, returning it's index.
// v must be nil (the zero reflect.Value) or must have a comparable type or
// must be the zero value of its type.
//
// If the VM's internal representation of v is different from the external, v
// must always have the external representation. Any conversion, if needed,
// will be internally handled.
func (fb *functionBuilder) makeGeneralValue(v reflect.Value) int8 {
	// Check if v has already been added to the general Values slice.
	if v.IsValid() {
		t := v.Type()
		if t.Comparable() {
			vi := v.Interface()
			for i, vv := range fb.fn.Values.General {
				if vv.IsValid() && vi == vv.Interface() {
					return int8(i)
				}
			}
		} else {
			for i, vv := range fb.fn.Values.General {
				if vv.IsValid() && t == vv.Type() {
					return int8(i)
				}
			}
		}
	} else {
		for i, vv := range fb.fn.Values.General {
			if !vv.IsValid() {
				return int8(i)
			}
		}
	}
	r := len(fb.fn.Values.General)
	if r == maxGeneralValuesCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "general values count exceeded %d", maxGeneralValuesCount))
	}
	fb.fn.Values.General = append(fb.fn.Values.General, v)
	return int8(r)
}

// makeFloatValue makes a new float value, returning it's index.
func (fb *functionBuilder) makeFloatValue(v float64) int {
	for i, vv := range fb.fn.Values.Float {
		if v == vv {
			return i
		}
	}
	r := len(fb.fn.Values.Float)
	if r == maxFloatValuesCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "floating-point values count exceeded %d", maxFloatValuesCount))
	}
	fb.fn.Values.Float = append(fb.fn.Values.Float, v)
	return r
}

// makeIntValue makes a new int value, returning it's index.
func (fb *functionBuilder) makeIntValue(v int64) int {
	for i, vv := range fb.fn.Values.Int {
		if v == vv {
			return i
		}
	}
	r := len(fb.fn.Values.Int)
	if r == maxIntValuesCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "integer values count exceeded %d", maxIntValuesCount))
	}
	fb.fn.Values.Int = append(fb.fn.Values.Int, v)
	return r
}

// sameFieldIndex reports whether i1 and i2 are the same field index.
func sameFieldIndex(i1, i2 []int) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k, i := range i1 {
		if i != i2[k] {
			return false
		}
	}
	return true
}

// makeFieldIndex makes a new field index, returning it's index.
func (fb *functionBuilder) makeFieldIndex(index []int) int8 {
	for i, index2 := range fb.fn.FieldIndexes {
		if sameFieldIndex(index, index2) {
			return int8(i)
		}
	}
	r := len(fb.fn.FieldIndexes)
	if r == maxFieldIndexesCount {
		panic(newLimitExceededError(fb.fn.Pos, fb.path, "field indexes count exceeded %d", maxFieldIndexesCount))
	}
	fb.fn.FieldIndexes = append(fb.fn.FieldIndexes, index)
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
		fn := newNativeFunction("scriggo.complex", "neg", negComplex)
		index := fb.addNativeFunction(fn)
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
	fn := newNativeFunction("scriggo.complex", n, f)
	index := fb.addNativeFunction(fn)
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
