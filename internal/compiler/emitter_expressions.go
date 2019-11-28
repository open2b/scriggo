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
