// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"scriggo/ast"
	"scriggo/runtime"
)

func (em *emitter) emitCompositeLiteral(expr *ast.CompositeLiteral, reg int8, dstType reflect.Type) (int8, bool) {
	typ := em.ti(expr.Type).Type
	switch typ.Kind() {
	case reflect.Slice, reflect.Array:
		if reg == 0 {
			for _, kv := range expr.KeyValues {
				typ := em.ti(kv.Value).Type
				em.emitExprR(kv.Value, typ, 0)
			}
			return reg, false
		}
		length := int8(em.compositeLiteralLen(expr)) // TODO(Gianluca): length is int
		if typ.Kind() == reflect.Slice {
			em.fb.emitMakeSlice(true, true, typ, length, length, reg, expr.Pos())
		} else {
			arrayZero := em.fb.makeGeneralConstant(em.types.New(typ).Elem().Interface())
			em.changeRegister(true, arrayZero, reg, typ, typ)
		}
		var index int8 = -1
		for _, kv := range expr.KeyValues {
			if kv.Key != nil {
				index = int8(em.ti(kv.Key).Constant.int64())
			} else {
				index++
			}
			em.fb.enterStack()
			indexReg := em.fb.newRegister(reflect.Int)
			em.fb.emitMove(true, index, indexReg, reflect.Int, true)
			elem, k := em.emitExprK(kv.Value, typ.Elem())
			if reg != 0 {
				em.fb.emitSetSlice(k, reg, elem, indexReg, expr.Pos(), typ.Elem().Kind())
			}
			em.fb.exitStack()
		}
		em.changeRegister(false, reg, reg, em.ti(expr.Type).Type, dstType)
	case reflect.Struct:
		// Struct should no be created, but its values must be emitted.
		if reg == 0 {
			for _, kv := range expr.KeyValues {
				em.emitExprR(kv.Value, em.ti(kv.Value).Type, 0)
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
			valueType := em.ti(kv.Value).Type
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
				typ := em.ti(kv.Value).Type
				em.emitExprR(kv.Value, typ, 0)
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
	t1 := em.ti(expr.Expr1).Type
	t2 := em.ti(expr.Expr2).Type
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
	case ast.OperatorEqual, ast.OperatorNotEqual, ast.OperatorLess, ast.OperatorLessOrEqual,
		ast.OperatorGreaterOrEqual, ast.OperatorGreater:
		var cond runtime.Condition
		if kind := t1.Kind(); reflect.Uint <= kind && kind <= reflect.Uintptr {
			cond = map[ast.OperatorType]runtime.Condition{
				ast.OperatorEqual:          runtime.ConditionEqual,    // same as signed integers
				ast.OperatorNotEqual:       runtime.ConditionNotEqual, // same as signed integers
				ast.OperatorLess:           runtime.ConditionLessU,
				ast.OperatorLessOrEqual:    runtime.ConditionLessOrEqualU,
				ast.OperatorGreater:        runtime.ConditionGreaterU,
				ast.OperatorGreaterOrEqual: runtime.ConditionGreaterOrEqualU,
			}[expr.Operator()]
		} else {
			cond = map[ast.OperatorType]runtime.Condition{
				ast.OperatorEqual:          runtime.ConditionEqual,
				ast.OperatorNotEqual:       runtime.ConditionNotEqual,
				ast.OperatorLess:           runtime.ConditionLess,
				ast.OperatorLessOrEqual:    runtime.ConditionLessOrEqual,
				ast.OperatorGreater:        runtime.ConditionGreater,
				ast.OperatorGreaterOrEqual: runtime.ConditionGreaterOrEqual,
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

// emitSelector emits selector in register reg.
func (em *emitter) emitSelector(expr *ast.Selector, reg int8, dstType reflect.Type) {

	ti := em.ti(expr)

	// Method value on concrete and interface values.
	if ti.MethodType == MethodValueConcrete || ti.MethodType == MethodValueInterface {
		rcvrExpr := expr.Expr
		rcvrType := em.ti(rcvrExpr).Type
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
			em.changeRegister(false, reg, reg, em.ti(expr).Type, dstType)
			return
		}
	}

	// Struct field.
	exprType := em.ti(expr.Expr).Type
	exprReg := em.emitExpr(expr.Expr, exprType)
	field, _ := exprType.FieldByName(expr.Ident)
	index := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
	fieldType := em.ti(expr).Type
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
