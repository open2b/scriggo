// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"math"
	"reflect"

	"scriggo/ast"
	"scriggo/runtime"
)

// emitExpr emits expr into a register of a given type. emitExpr tries to not
// create a new register, but to use an existing one. The register used for
// emission is returned.
// TODO(Gianluca): add an option/another method to force the creation of an new
// register? Is necessary?
func (em *emitter) emitExpr(expr ast.Expression, dstType reflect.Type) int8 {
	reg, _ := em._emitExpr(expr, dstType, 0, false, false)
	return reg
}

// emitExprK emits expr into a register of a given type. The boolean return
// parameter reports whether the returned int8 is a constant or not.
func (em *emitter) emitExprK(expr ast.Expression, dstType reflect.Type) (int8, bool) {
	return em._emitExpr(expr, dstType, 0, false, true)
}

// emitExprR emits expr into register reg with the given type.
func (em *emitter) emitExprR(expr ast.Expression, dstType reflect.Type, reg int8) {
	_, _ = em._emitExpr(expr, dstType, reg, true, false)
}

// _emitExpr emits expression expr.
//
// If a register is given and putInReg is true, then such register is used for
// emission; otherwise _emitExpr chooses the output register, returning it to
// the caller.
//
// If allowK is true, then the returned register can be an immediante value and
// the boolean return parameters is true.
//
// _emitExpr is an internal support method, and should be called by emitExpr,
// emitExprK and emitExprR exclusively.
//
func (em *emitter) _emitExpr(expr ast.Expression, dstType reflect.Type, reg int8, useGivenReg bool, allowK bool) (int8, bool) {

	// Take the type info of the expression.
	ti := em.ti(expr)

	// No need to use the given register: check if expr can be emitted without
	// allocating a new one.
	if !useGivenReg {
		// Check if expr can be emitted as immediate.
		if allowK && ti.HasValue() && !ti.IsPredefined() {
			switch v := ti.value.(type) {
			case int64:
				if canEmitDirectly(reflect.Int, dstType.Kind()) {
					if -127 < v && v < 126 {
						return int8(v), true
					}
				}
			case float64:
				if canEmitDirectly(reflect.Float64, dstType.Kind()) {
					if math.Floor(v) == v && -127 < v && v < 126 {
						return int8(v), true
					}
				}
			}
		}
		// Expr cannot be emitted as immediate: check if it's possible to emit
		// it without allocating a new register.
		if expr, ok := expr.(*ast.Identifier); ok && em.fb.isLocalVariable(expr.Name) {
			if canEmitDirectly(ti.Type.Kind(), dstType.Kind()) {
				return em.fb.scopeLookup(expr.Name), false
			}
		}
		// None of the conditions above applied: a new register must be
		// allocated, and the emission must proceed.
		reg = em.fb.newRegister(dstType.Kind())
	}

	// The expression has a value and is not predefined.
	if ti != nil && ti.HasValue() && !ti.IsPredefined() {
		return em.emitValueNotPredefined(ti, reg, dstType)
	}

	// expr is a predefined function.
	if index, ok := em.fnStore.predefFunc(expr, false); ok {
		em.fb.emitLoadFunc(true, index, reg)
		em.changeRegister(false, reg, reg, ti.Type, dstType)
		return reg, false
	}

	switch expr := expr.(type) {

	case *ast.BinaryOperator:

		return em.emitBinaryOp(expr, reg, dstType)

	case *ast.Call:

		// ShowMacro which must be ignored (cannot be resolved).
		if em.ti(expr.Func) == showMacroIgnoredTi {
			return reg, false
		}

		// Predeclared built-in function call.
		if em.ti(expr.Func).IsBuiltinFunction() {
			em.emitBuiltin(expr, reg, dstType)
			return reg, false
		}

		// Conversion.
		if em.ti(expr.Func).IsType() {
			convertType := em.typ(expr.Func)
			// A conversion cannot have side-effects.
			if reg == 0 {
				return reg, false
			}
			typ := em.typ(expr.Args[0])
			arg := em.emitExpr(expr.Args[0], typ)
			if canEmitDirectly(convertType.Kind(), dstType.Kind()) {
				em.changeRegister(false, arg, reg, typ, convertType)
				return reg, false
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(convertType.Kind())
			em.changeRegister(false, arg, tmp, typ, convertType)
			em.changeRegister(false, tmp, reg, convertType, dstType)
			em.fb.exitStack()
			return reg, false
		}

		// Function call.
		em.fb.enterStack()
		regs, types := em.emitCallNode(expr, false, false)
		if reg != 0 {
			em.changeRegister(false, regs[0], reg, types[0], dstType)
		}
		em.fb.exitStack()

	case *ast.CompositeLiteral:

		return em.emitCompositeLiteral(expr, reg, dstType)

	case *ast.TypeAssertion:

		exprType := em.typ(expr.Expr)
		exprReg := em.emitExpr(expr.Expr, exprType)
		assertType := em.typ(expr.Type)
		pos := expr.Pos()
		if canEmitDirectly(assertType.Kind(), dstType.Kind()) {
			em.fb.emitAssert(exprReg, assertType, reg)
			em.fb.emitPanic(0, exprType, pos)
			return reg, false
		}
		em.fb.enterScope()
		tmp := em.fb.newRegister(assertType.Kind())
		em.fb.emitAssert(exprReg, assertType, tmp)
		em.fb.emitPanic(0, exprType, pos)
		em.changeRegister(false, tmp, reg, assertType, dstType)
		em.fb.exitScope()

	case *ast.Selector:

		em.emitSelector(expr, reg, dstType)

	case *ast.UnaryOperator:

		// Emit a generic unary operator.
		em.emitUnaryOperator(expr, reg, dstType)

		return reg, false

	case *ast.Func:

		// Template macro definition.
		if expr.Ident != nil && em.isTemplate {
			macroFn := newFunction("", expr.Ident.Name, expr.Type.Reflect)
			em.fnStore.makeAvailableScriggoFn(em.pkg, expr.Ident.Name, macroFn)
			fb := em.fb
			em.setFunctionVarRefs(macroFn, expr.Upvars)
			em.fb = newBuilder(macroFn, em.fb.getPath())
			em.fb.emitSetAlloc(em.options.MemoryLimit)
			em.fb.enterScope()
			em.prepareFunctionBodyParameters(expr)
			em.emitNodes(expr.Body.Nodes)
			em.fb.end()
			em.fb.exitScope()
			em.fb = fb
			return reg, false
		}

		if reg == 0 {
			return reg, false
		}

		var tmp int8
		if canEmitDirectly(reflect.Func, dstType.Kind()) {
			tmp = reg
		} else {
			tmp = em.fb.newRegister(reflect.Func)
		}

		fn := em.fb.emitFunc(tmp, ti.Type)
		em.setFunctionVarRefs(fn, expr.Upvars)

		funcLitBuilder := newBuilder(fn, em.fb.getPath())
		funcLitBuilder.emitSetAlloc(em.options.MemoryLimit)
		currFB := em.fb
		em.fb = funcLitBuilder

		em.fb.enterScope()
		em.prepareFunctionBodyParameters(expr)
		em.emitNodes(expr.Body.Nodes)
		em.fb.exitScope()
		em.fb.end()
		em.fb = currFB

		em.changeRegister(false, tmp, reg, ti.Type, dstType)

	case *ast.Identifier:

		// An identifier evaluation cannot have side effects.
		if reg == 0 {
			return reg, false
		}

		typ := ti.Type

		if em.fb.isLocalVariable(expr.Name) {
			ident := em.fb.scopeLookup(expr.Name)
			em.changeRegister(false, ident, reg, typ, dstType)
			return reg, false
		}

		// Scriggo variables and closure variables.
		if index, ok := em.varStore.nonLocalVarIndex(expr); ok {
			if canEmitDirectly(typ.Kind(), dstType.Kind()) {
				em.fb.emitGetVar(index, reg, dstType.Kind())
				return reg, false
			}
			tmp := em.fb.newRegister(typ.Kind())
			em.fb.emitGetVar(index, tmp, typ.Kind())
			em.changeRegister(false, tmp, reg, typ, dstType)
			return reg, false
		}

		// Identifier represents a function.
		if fun, ok := em.fnStore.availableScriggoFn(em.pkg, expr.Name); ok {
			em.fb.emitLoadFunc(false, em.fnStore.scriggoFnIndex(fun), reg)
			em.changeRegister(false, reg, reg, ti.Type, dstType)
			return reg, false
		}

		panic(fmt.Errorf("BUG: none of the previous conditions matched identifier %v", expr)) // remove.

	case *ast.Index:

		exprType := em.typ(expr.Expr)
		exprReg := em.emitExpr(expr.Expr, exprType)
		var indexType reflect.Type
		if exprType.Kind() == reflect.Map {
			indexType = exprType.Key()
		} else {
			indexType = intType
		}
		index, kindex := em.emitExprK(expr.Index, indexType)
		var elemType reflect.Type
		if exprType.Kind() == reflect.String {
			elemType = uint8Type
		} else {
			elemType = exprType.Elem()
		}
		pos := expr.Pos()
		if canEmitDirectly(elemType.Kind(), dstType.Kind()) {
			em.fb.emitIndex(kindex, exprReg, index, reg, exprType, pos, true)
			return reg, false
		}
		em.fb.enterStack()
		tmp := em.fb.newRegister(elemType.Kind())
		em.fb.emitIndex(kindex, exprReg, index, tmp, exprType, pos, true)
		em.changeRegister(false, tmp, reg, elemType, dstType)
		em.fb.exitStack()

	case *ast.Slicing:

		exprType := em.typ(expr.Expr)
		src := em.emitExpr(expr.Expr, exprType)
		var low, high int8 = 0, -1
		var kLow, kHigh = true, true
		// emit low
		if expr.Low != nil {
			low, kLow = em.emitExprK(expr.Low, em.typ(expr.Low))
		}
		// emit high
		if expr.High != nil {
			high, kHigh = em.emitExprK(expr.High, em.typ(expr.High))
		}
		pos := expr.Pos()
		if exprType.Kind() == reflect.String {
			em.fb.emitStringSlice(kLow, kHigh, src, reg, low, high, pos)
		} else {
			// emit max
			var max int8 = -1
			var kMax = true
			if expr.Max != nil {
				max, kMax = em.emitExprK(expr.Max, em.typ(expr.Max))
			}
			em.fb.emitSlice(kLow, kHigh, kMax, src, reg, low, high, max, pos)
		}

	default:

		panic(fmt.Sprintf("emitExpr currently does not support %T nodes (expr: %s)", expr, expr))

	}

	return reg, false
}

func (em *emitter) emitBinaryOp(expr *ast.BinaryOperator, reg int8, dstType reflect.Type) (int8, bool) {
	ti := em.ti(expr)
	// Binary operations on complex numbers.
	if exprType := ti.Type; exprType.Kind() == reflect.Complex64 || exprType.Kind() == reflect.Complex128 {
		em.emitComplexOperation(exprType, expr.Expr1, expr.Operator(), expr.Expr2, reg, dstType)
		return reg, false
	}

	// Binary && and ||.
	if op := expr.Operator(); op == ast.OperatorAndAnd || op == ast.OperatorOrOr {
		cmp := int8(0)
		if op == ast.OperatorAndAnd {
			cmp = 1
		}
		if canEmitDirectly(dstType.Kind(), reflect.Bool) {
			em.emitExprR(expr.Expr1, dstType, reg)
			endIf := em.fb.newLabel()
			em.fb.emitIf(true, reg, runtime.ConditionEqual, cmp, reflect.Int, expr.Pos())
			em.fb.emitGoto(endIf)
			em.emitExprR(expr.Expr2, dstType, reg)
			em.fb.setLabelAddr(endIf)
			return reg, false
		}
		em.fb.enterStack()
		tmp := em.fb.newRegister(reflect.Bool)
		em.emitExprR(expr.Expr1, boolType, tmp)
		endIf := em.fb.newLabel()
		em.fb.emitIf(true, tmp, runtime.ConditionEqual, cmp, reflect.Int, expr.Pos())
		em.fb.emitGoto(endIf)
		em.emitExprR(expr.Expr2, boolType, tmp)
		em.fb.setLabelAddr(endIf)
		em.changeRegister(false, tmp, reg, boolType, dstType)
		em.fb.exitStack()
		return reg, false
	}

	// Equality (or not-equality) checking with the predeclared identifier 'nil'.
	if em.ti(expr.Expr1).Nil() || em.ti(expr.Expr2).Nil() {
		em.changeRegister(true, 1, reg, boolType, dstType)
		em.emitCondition(expr)
		em.changeRegister(true, 0, reg, boolType, dstType)
		return reg, false
	}

	// ==, !=, <, <=, >=, >, &&, ||, +, -, *, /, %, ^, &^, <<, >>.
	exprType := ti.Type
	t1 := em.typ(expr.Expr1)
	t2 := em.typ(expr.Expr2)
	v1 := em.emitExpr(expr.Expr1, t1)
	v2, k := em.emitExprK(expr.Expr2, t2)
	if reg == 0 {
		return reg, false
	}
	// String concatenation.
	if expr.Operator() == ast.OperatorAddition && t1.Kind() == reflect.String {
		if k {
			v2 = em.emitExpr(expr.Expr2, t2)
		}
		if canEmitDirectly(exprType.Kind(), dstType.Kind()) {
			em.fb.emitConcat(v1, v2, reg)
			return reg, false
		}
		em.fb.enterStack()
		tmp := em.fb.newRegister(exprType.Kind())
		em.fb.emitConcat(v1, v2, tmp)
		em.changeRegister(false, tmp, reg, exprType, dstType)
		em.fb.exitStack()
		return reg, false
	}
	switch expr.Operator() {
	case ast.OperatorAddition, ast.OperatorSubtraction, ast.OperatorMultiplication, ast.OperatorDivision,
		ast.OperatorModulo, ast.OperatorAnd, ast.OperatorOr, ast.OperatorXor, ast.OperatorAndNot,
		ast.OperatorLeftShift, ast.OperatorRightShift:
		var emitFn func(bool, int8, int8, int8, reflect.Kind)
		switch expr.Operator() {
		case ast.OperatorAddition:
			emitFn = em.fb.emitAdd
		case ast.OperatorSubtraction:
			emitFn = em.fb.emitSub
		case ast.OperatorMultiplication:
			emitFn = em.fb.emitMul
		case ast.OperatorDivision:
			emitFn = func(k bool, x, y, z int8, kind reflect.Kind) { em.fb.emitDiv(k, x, y, z, kind, expr.Pos()) }
		case ast.OperatorModulo:
			emitFn = func(k bool, x, y, z int8, kind reflect.Kind) { em.fb.emitRem(k, x, y, z, kind, expr.Pos()) }
		case ast.OperatorAnd:
			emitFn = em.fb.emitAnd
		case ast.OperatorOr:
			emitFn = em.fb.emitOr
		case ast.OperatorXor:
			emitFn = em.fb.emitXor
		case ast.OperatorAndNot:
			emitFn = em.fb.emitAndNot
		case ast.OperatorLeftShift:
			emitFn = em.fb.emitLeftShift
		case ast.OperatorRightShift:
			emitFn = em.fb.emitRightShift
		}
		if canEmitDirectly(exprType.Kind(), dstType.Kind()) {
			emitFn(k, v1, v2, reg, exprType.Kind())
			return reg, false
		}
		em.fb.enterStack()
		tmp := em.fb.newRegister(exprType.Kind())
		emitFn(k, v1, v2, tmp, exprType.Kind())
		em.changeRegister(false, tmp, reg, exprType, dstType)
		em.fb.exitStack()
		return reg, false
	case ast.OperatorEqual, ast.OperatorNotEqual, ast.OperatorLess, ast.OperatorLessEqual,
		ast.OperatorGreaterEqual, ast.OperatorGreater:
		var cond runtime.Condition
		if kind := t1.Kind(); reflect.Uint <= kind && kind <= reflect.Uintptr {
			cond = map[ast.OperatorType]runtime.Condition{
				ast.OperatorEqual:        runtime.ConditionEqual,    // same as signed integers
				ast.OperatorNotEqual:     runtime.ConditionNotEqual, // same as signed integers
				ast.OperatorLess:         runtime.ConditionLessU,
				ast.OperatorLessEqual:    runtime.ConditionLessEqualU,
				ast.OperatorGreater:      runtime.ConditionGreaterU,
				ast.OperatorGreaterEqual: runtime.ConditionGreaterEqualU,
			}[expr.Operator()]
		} else {
			cond = map[ast.OperatorType]runtime.Condition{
				ast.OperatorEqual:        runtime.ConditionEqual,
				ast.OperatorNotEqual:     runtime.ConditionNotEqual,
				ast.OperatorLess:         runtime.ConditionLess,
				ast.OperatorLessEqual:    runtime.ConditionLessEqual,
				ast.OperatorGreater:      runtime.ConditionGreater,
				ast.OperatorGreaterEqual: runtime.ConditionGreaterEqual,
			}[expr.Operator()]
		}
		pos := expr.Pos()
		if canEmitDirectly(exprType.Kind(), dstType.Kind()) {
			em.fb.emitMove(true, 1, reg, reflect.Bool, false)
			em.fb.emitIf(k, v1, cond, v2, t1.Kind(), pos)
			em.fb.emitMove(true, 0, reg, reflect.Bool, false)
			return reg, false
		}
		em.fb.enterStack()
		tmp := em.fb.newRegister(exprType.Kind())
		em.fb.emitMove(true, 1, tmp, reflect.Bool, false)
		em.fb.emitIf(k, v1, cond, v2, t1.Kind(), pos)
		em.fb.emitMove(true, 0, tmp, reflect.Bool, false)
		em.changeRegister(false, tmp, reg, exprType, dstType)
		em.fb.exitStack()
	}

	return reg, false

}

func (em *emitter) emitCompositeLiteral(expr *ast.CompositeLiteral, reg int8, dstType reflect.Type) (int8, bool) {
	typ := em.typ(expr.Type)
	switch typ.Kind() {
	case reflect.Slice, reflect.Array:
		if reg == 0 {
			for _, kv := range expr.KeyValues {
				typ := em.typ(kv.Value)
				em.emitExprR(kv.Value, typ, 0)
			}
			return reg, false
		}
		length := em.compositeLiteralLen(expr)
		var k bool
		var length8 int8
		if length > 126 {
			length8 = em.fb.newRegister(reflect.Int)
			em.fb.emitLoadNumber(intRegister, em.fb.makeIntConstant(int64(length)), length8)
			k = false
		} else {
			length8 = int8(length)
			k = true
		}
		if typ.Kind() == reflect.Slice {
			em.fb.emitMakeSlice(k, k, typ, length8, length8, reg, expr.Pos())
		} else {
			arrayZero := em.fb.makeGeneralConstant(em.types.New(typ).Elem().Interface())
			em.changeRegister(true, arrayZero, reg, typ, typ)
		}
		var index int64 = -1
		for _, kv := range expr.KeyValues {
			if kv.Key != nil {
				index = em.ti(kv.Key).Constant.int64()
			} else {
				index++
			}
			em.fb.enterStack()
			indexReg := em.fb.newRegister(reflect.Int)
			if index > 126 {
				em.fb.emitLoadNumber(intRegister, em.fb.makeIntConstant(int64(index)), indexReg)
			} else {
				em.fb.emitMove(true, int8(index), indexReg, reflect.Int, true)
			}
			elem, k := em.emitExprK(kv.Value, typ.Elem())
			if reg != 0 {
				em.fb.emitSetSlice(k, reg, elem, indexReg, expr.Pos(), typ.Elem().Kind())
			}
			em.fb.exitStack()
		}
		em.changeRegister(false, reg, reg, em.typ(expr.Type), dstType)
	case reflect.Struct:
		// Struct should no be created, but its values must be emitted.
		if reg == 0 {
			for _, kv := range expr.KeyValues {
				em.emitExprR(kv.Value, em.typ(kv.Value), 0)
			}
			return reg, false
		}
		// TODO: the types instance should be the same of the type checker!
		structZero := em.fb.makeGeneralConstant(em.types.New(typ).Elem().Interface())
		// When there are no values in the composite literal, optimize the
		// creation of the struct.
		if len(expr.KeyValues) == 0 {
			em.changeRegister(true, structZero, reg, typ, dstType)
			return reg, false
		}
		// Assign key-value pairs to the struct fields.
		em.fb.enterStack()
		var structt int8
		if canEmitDirectly(typ.Kind(), dstType.Kind()) {
			structt = em.fb.newRegister(reflect.Struct)
		} else {
			structt = reg
		}
		em.changeRegister(true, structZero, structt, typ, typ)
		for _, kv := range expr.KeyValues {
			name := kv.Key.(*ast.Identifier).Name
			field, _ := typ.FieldByName(name)
			valueType := em.typ(kv.Value)
			if canEmitDirectly(field.Type.Kind(), valueType.Kind()) {
				value, k := em.emitExprK(kv.Value, valueType)
				index := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
				em.fb.emitSetField(k, structt, index, value, field.Type.Kind())
			} else {
				em.fb.enterStack()
				tmp := em.emitExpr(kv.Value, valueType)
				value := em.fb.newRegister(field.Type.Kind())
				em.changeRegister(false, tmp, value, valueType, field.Type)
				em.fb.exitStack()
				index := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
				em.fb.emitSetField(false, structt, index, value, field.Type.Kind())
			}
			// TODO(Gianluca): use field "k" of SetField.
		}
		em.changeRegister(false, structt, reg, typ, dstType)
		em.fb.exitStack()

	case reflect.Map:
		if reg == 0 {
			for _, kv := range expr.KeyValues {
				em.emitExprR(kv.Value, em.typ(kv.Value), 0)
			}
			return reg, false
		}
		tmp := em.fb.newRegister(reflect.Map)
		size := len(expr.KeyValues)
		if 0 <= size && size < 126 {
			em.fb.emitMakeMap(typ, true, int8(size), tmp)
		} else {
			sizeReg := em.fb.makeIntConstant(int64(size))
			em.fb.emitMakeMap(typ, false, sizeReg, tmp)
		}
		for _, kv := range expr.KeyValues {
			key := em.fb.newRegister(typ.Key().Kind())
			em.fb.enterStack()
			em.emitExprR(kv.Key, typ.Key(), key)
			value, k := em.emitExprK(kv.Value, typ.Elem())
			em.fb.exitStack()
			em.fb.emitSetMap(k, tmp, value, key, typ, expr.Pos())
		}
		em.changeRegister(false, tmp, reg, typ, dstType)
	}
	return reg, false
}

// emitSelector emits selector in register reg.
func (em *emitter) emitSelector(expr *ast.Selector, reg int8, dstType reflect.Type) {

	ti := em.ti(expr)

	// Method value on concrete and interface values.
	if ti.MethodType == MethodValueConcrete || ti.MethodType == MethodValueInterface {
		rcvrExpr := expr.Expr
		rcvrType := em.typ(rcvrExpr)
		rcvr := em.emitExpr(rcvrExpr, rcvrType)
		// MethodValue reads receiver from general.
		if kindToType(rcvrType.Kind()) != generalRegister {
			oldRcvr := rcvr
			rcvr = em.fb.newRegister(reflect.Interface)
			em.fb.emitTypify(false, rcvrType, oldRcvr, rcvr)
		}
		if kindToType(dstType.Kind()) == generalRegister {
			em.fb.emitMethodValue(expr.Ident, rcvr, reg)
		} else {
			panic("not implemented")
		}
		return
	}

	// Predefined package variable or imported package variable.
	if index, ok := em.varStore.nonLocalVarIndex(expr); ok {
		if reg == 0 {
			return
		}
		if canEmitDirectly(ti.Type.Kind(), dstType.Kind()) {
			em.fb.emitGetVar(int(index), reg, dstType.Kind())
			return
		}
		tmp := em.fb.newRegister(ti.Type.Kind())
		em.fb.emitGetVar(int(index), tmp, ti.Type.Kind())
		em.changeRegister(false, tmp, reg, ti.Type, dstType)
		return
	}

	// Scriggo-defined package functions.
	if ident, ok := expr.Expr.(*ast.Identifier); ok {
		if sf, ok := em.fnStore.availableScriggoFn(em.pkg, ident.Name+"."+expr.Ident); ok {
			if reg == 0 {
				return
			}
			index := em.fnStore.scriggoFnIndex(sf)
			em.fb.emitLoadFunc(false, index, reg)
			em.changeRegister(false, reg, reg, em.typ(expr), dstType)
			return
		}
	}

	// Struct field.
	exprType := em.typ(expr.Expr)
	exprReg := em.emitExpr(expr.Expr, exprType)
	field, _ := exprType.FieldByName(expr.Ident)
	index := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
	fieldType := em.typ(expr)
	if canEmitDirectly(fieldType.Kind(), dstType.Kind()) {
		em.fb.emitField(exprReg, index, reg, dstType.Kind(), true)
		return
	}
	// TODO: add enter/exit stack method calls.
	tmp := em.fb.newRegister(fieldType.Kind())
	em.fb.emitField(exprReg, index, tmp, fieldType.Kind(), true)
	em.changeRegister(false, tmp, reg, fieldType, dstType)

	return
}

func (em *emitter) emitUnaryOperator(unOp *ast.UnaryOperator, reg int8, dstType reflect.Type) {

	operand := unOp.Expr
	operandType := em.typ(operand)
	unOpType := em.typ(unOp)

	// Receive operation on channel.
	//
	//	v     = <- ch
	//  v, ok = <- ch
	//          <- ch
	if unOp.Operator() == ast.OperatorReceive {
		chanType := em.typ(unOp.Expr)
		valueType := unOpType
		chann := em.emitExpr(unOp.Expr, chanType)
		if reg == 0 {
			em.fb.emitReceive(chann, 0, 0)
			return
		}
		if canEmitDirectly(valueType.Kind(), dstType.Kind()) {
			em.fb.emitReceive(chann, 0, reg)
			return
		}
		tmp := em.fb.newRegister(valueType.Kind())
		em.fb.emitReceive(chann, 0, tmp)
		em.changeRegister(false, tmp, reg, valueType, dstType)
		return
	}

	// Unary operation (negation) on a complex number.
	if exprType := unOpType; exprType.Kind() == reflect.Complex64 || exprType.Kind() == reflect.Complex128 {
		if unOp.Operator() != ast.OperatorSubtraction {
			panic("bug: expected operator subtraction")
		}
		stackShift := em.fb.currentStackShift()
		em.fb.enterScope()
		index := em.fb.complexOperationIndex(ast.OperatorSubtraction, true)
		ret := em.fb.newRegister(reflect.Complex128)
		arg := em.fb.newRegister(reflect.Complex128)
		em.fb.enterScope()
		em.emitExprR(unOp.Expr, exprType, arg)
		em.fb.exitScope()
		em.fb.emitCallPredefined(index, 0, stackShift, unOp.Pos())
		em.changeRegister(false, ret, reg, exprType, dstType)
		em.fb.exitScope()
		return
	}

	switch unOp.Operator() {

	// !operand
	case ast.OperatorNot:
		if reg == 0 {
			em.emitExprR(operand, operandType, 0)
			return
		}
		if canEmitDirectly(unOpType.Kind(), dstType.Kind()) {
			em.emitExprR(operand, operandType, reg)
			em.fb.emitSubInv(true, reg, int8(1), reg, reflect.Int)
			return
		}
		em.fb.enterScope()
		tmp := em.emitExpr(operand, operandType)
		em.fb.emitSubInv(true, tmp, int8(1), tmp, reflect.Int)
		em.changeRegister(false, tmp, reg, operandType, dstType)
		em.fb.exitScope()

	// *operand
	case ast.OperatorMultiplication:
		if reg == 0 {
			em.emitExprR(operand, operandType, 0)
			return
		}
		if canEmitDirectly(unOpType.Kind(), dstType.Kind()) {
			exprReg := em.emitExpr(operand, operandType)
			em.changeRegister(false, -exprReg, reg, operandType.Elem(), dstType)
			return
		}
		exprReg := em.emitExpr(operand, operandType)
		tmp := em.fb.newRegister(operandType.Elem().Kind())
		em.changeRegister(false, -exprReg, tmp, operandType.Elem(), operandType.Elem())
		em.changeRegister(false, tmp, reg, operandType.Elem(), dstType)

	// &operand
	case ast.OperatorAnd:
		switch operand := operand.(type) {

		// &a
		case *ast.Identifier:
			if em.fb.isLocalVariable(operand.Name) {
				varr := em.fb.scopeLookup(operand.Name)
				em.fb.emitNew(em.types.PtrTo(unOpType), reg)
				em.fb.emitMove(false, -varr, reg, dstType.Kind(), false)
				return
			}
			// Address of a non-local variable.
			if index, ok := em.varStore.nonLocalVarIndex(operand); ok {
				em.fb.enterStack()
				tmp := em.fb.newRegister(reflect.Ptr)
				em.fb.emitGetVarAddr(index, tmp)
				em.changeRegister(false, tmp, reg, reflect.PtrTo(operandType), dstType)
				em.fb.exitStack()
				return
			}
			panic("BUG")

		// &*a
		case *ast.UnaryOperator:
			// Deference the pointer to check for "invalid memory address"
			// errors, then discard the result.
			em.fb.enterStack()
			pointedElemType := dstType.Elem()
			pointer := em.emitExpr(operand, pointedElemType)
			tmp := em.fb.newRegister(pointedElemType.Kind())
			em.changeRegister(false, -pointer, tmp, pointedElemType, pointedElemType)
			em.fb.exitStack()
			// The pointer is valid, so &*a is equivalent to a.
			em.emitExprR(operand.Expr, dstType, reg)

		// &v[i]
		// (where v is a slice or an addressable array)
		case *ast.Index:
			expr := em.emitExpr(operand.Expr, em.typ(operand.Expr))
			index := em.emitExpr(operand.Index, intType)
			pos := operand.Expr.Pos()
			if canEmitDirectly(unOpType.Kind(), dstType.Kind()) {
				em.fb.emitAddr(expr, index, reg, pos)
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(unOpType.Kind())
			em.fb.emitAddr(expr, index, tmp, pos)
			em.changeRegister(false, tmp, reg, unOpType, dstType)
			em.fb.exitStack()

		// &s.Field
		case *ast.Selector:
			// Address of a non-local variable.
			if index, ok := em.varStore.nonLocalVarIndex(operand); ok {
				if canEmitDirectly(operandType.Kind(), dstType.Kind()) {
					em.fb.emitGetVarAddr(index, reg)
					return
				}
				tmp := em.fb.newRegister(operandType.Kind())
				em.fb.emitGetVarAddr(index, tmp)
				em.changeRegister(false, tmp, reg, operandType, dstType)
				return
			}
			operandExprType := em.typ(operand.Expr)
			expr := em.emitExpr(operand.Expr, operandExprType)
			field, _ := operandExprType.FieldByName(operand.Ident)
			index := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
			pos := operand.Expr.Pos()
			if canEmitDirectly(reflect.PtrTo(field.Type).Kind(), dstType.Kind()) {
				em.fb.emitAddr(expr, index, reg, pos)
				return
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(reflect.Ptr)
			em.fb.emitAddr(expr, index, tmp, pos)
			em.changeRegister(false, tmp, reg, em.types.PtrTo(field.Type), dstType)
			em.fb.exitStack()

		// &T{..}
		case *ast.CompositeLiteral:
			tmp := em.fb.newRegister(reflect.Ptr)
			em.fb.emitNew(operandType, tmp)
			em.emitExprR(operand, operandType, -tmp)
			em.changeRegister(false, tmp, reg, unOpType, dstType)

		default:
			panic("TODO(Gianluca): not implemented")
		}

	// +operand
	case ast.OperatorAddition:
		// Nothing to do.

	// -operand
	case ast.OperatorSubtraction:
		if reg == 0 {
			em.emitExprR(operand, dstType, 0)
			return
		}
		if canEmitDirectly(operandType.Kind(), dstType.Kind()) {
			em.emitExprR(operand, dstType, reg)
			em.fb.emitSubInv(true, reg, 0, reg, dstType.Kind())
			return
		}
		em.fb.enterStack()
		tmp := em.fb.newRegister(operandType.Kind())
		em.emitExprR(operand, operandType, tmp)
		em.fb.emitSubInv(true, tmp, 0, tmp, operandType.Kind())
		em.changeRegister(false, tmp, reg, operandType, dstType)
		em.fb.exitStack()

	default:
		panic(fmt.Errorf("BUG: not implemented operator %s", unOp.Operator()))
	}

	return

}
