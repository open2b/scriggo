// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"math"
	"reflect"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/runtime"
)

// emitExpr emits expr into a register of a given type. emitExpr tries to not
// create a new register, but to use an existing one. The register used for
// emission is returned.
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
		if allowK && ti.HasValue() && !ti.IsNative() {
			switch v := ti.value.(type) {
			case int64:
				if canEmitDirectly(reflect.Int, dstType.Kind()) {
					if -128 <= v && v <= 127 {
						return int8(v), true
					}
				}
			case float64:
				if canEmitDirectly(reflect.Float64, dstType.Kind()) {
					if math.Floor(v) == v && -128 <= v && v <= 127 {
						return int8(v), true
					}
				}
			}
		}
		// Expr cannot be emitted as immediate: check if it's possible to emit
		// it without allocating a new register.
		if expr, ok := expr.(*ast.Identifier); ok && em.fb.declaredInFunc(expr.Name) {
			if canEmitDirectly(ti.Type.Kind(), dstType.Kind()) {
				return em.fb.scopeLookup(expr.Name), false
			}
		}
		// None of the conditions above applied: a new register must be
		// allocated, and the emission must proceed.
		reg = em.fb.newRegister(dstType.Kind())
	}

	// The expression has a value and is not predefined.
	if ti != nil && ti.HasValue() && !ti.IsNative() {
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

		em.emitBinaryOp(expr, reg, dstType)
		return reg, false

	case *ast.Call:

		// Predeclared built-in function call.
		if em.ti(expr.Func).IsBuiltinFunction() {
			em.emitBuiltin(expr, reg, dstType)
			return reg, false
		}

		// Conversion.
		if ti := em.ti(expr.Func); ti.IsType() {
			convertType := em.typ(expr.Func)
			// A conversion cannot have side-effects.
			if reg == 0 {
				return reg, false
			}
			typ := em.typ(expr.Args[0])
			arg := em.emitExpr(expr.Args[0], typ)
			markdownToHTML := ti.IsFormatType() && ti.Type != stringType
			if canEmitDirectly(convertType.Kind(), dstType.Kind()) {
				if markdownToHTML {
					em.changeRegisterConvertFormat(false, arg, reg, typ, convertType)
				} else {
					em.changeRegister(false, arg, reg, typ, convertType)
				}
				return reg, false
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(convertType.Kind())
			if markdownToHTML {
				em.changeRegisterConvertFormat(false, arg, tmp, typ, convertType)
			} else {
				em.changeRegister(false, arg, tmp, typ, convertType)
			}
			em.changeRegister(false, tmp, reg, convertType, dstType)
			em.fb.exitStack()
			return reg, false
		}

		// Function call.
		em.fb.enterStack()
		regs, types := em.emitCallNode(expr, false, false, runtime.ReturnString)
		if reg != 0 {
			em.changeRegister(false, regs[0], reg, types[0], dstType)
		}
		em.fb.exitStack()

	case *ast.CompositeLiteral:

		return em.emitCompositeLiteral(expr, reg, dstType)

	case *ast.DollarIdentifier:

		// The field expr.IR.Ident has been set by the type checker; it may be
		// interface{}(x) if x is an existing global identifier, or
		// interface{}(nil) otherwise.
		//
		// So, the emission of a dollar identifier can be done simply by
		// emitting expr.IR.Ident.
		return em._emitExpr(expr.IR.Ident, dstType, reg, useGivenReg, allowK)

	case *ast.Default:
		ex := expr.Expr1
		if ti := em.ti(expr.Expr1); ti == nil {
			ex = expr.Expr2
		}
		return em._emitExpr(ex, dstType, reg, useGivenReg, allowK)

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
		em.emitUnaryOp(expr, reg, dstType)

		return reg, false

	case *ast.Func:

		if reg == 0 {
			return reg, false
		}

		var tmp int8
		if canEmitDirectly(reflect.Func, dstType.Kind()) {
			tmp = reg
		} else {
			tmp = em.fb.newRegister(reflect.Func)
		}
		fn := &runtime.Function{
			Pkg:    em.fb.fn.Pkg,
			File:   em.fb.fn.File,
			Macro:  expr.Type.Macro,
			Format: expr.Format,
			Pos:    convertPosition(expr.Pos()),
			Type:   ti.Type,
			Parent: em.fb.fn,
		}
		em.fb.emitLoadFunc(false, em.fb.addFunction(fn), tmp)
		em.setFunctionVarRefs(fn, expr.Upvars)

		funcLitBuilder := newBuilder(fn, em.fb.getPath())
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

		if em.fb.declaredInFunc(expr.Name) {
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
			em.fb.enterStack()
			tmp := em.fb.newRegister(typ.Kind())
			em.fb.emitGetVar(index, tmp, typ.Kind())
			em.changeRegister(false, tmp, reg, typ, dstType)
			em.fb.exitStack()
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

	case *ast.Render:

		// Emit the code that imports the dummy file, then emit the call to the
		// dummy macro declared on it.
		em.emitNodes([]ast.Node{expr.IR.Import})
		return em._emitExpr(expr.IR.Call, dstType, reg, useGivenReg, allowK)

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
			if canEmitDirectly(reflect.String, dstType.Kind()) {
				em.fb.emitStringSlice(kLow, kHigh, src, reg, low, high, pos)
			} else {
				em.fb.enterStack()
				tmp := em.fb.newRegister(reflect.String)
				em.fb.emitStringSlice(kLow, kHigh, src, tmp, low, high, pos)
				em.changeRegister(false, tmp, reg, exprType, dstType)
				em.fb.exitStack()
			}
		} else {
			// If necessary, emit max.
			var max int8 = -1
			var kMax = true
			if expr.Max != nil {
				max, kMax = em.emitExprK(expr.Max, em.typ(expr.Max))
			}
			if canEmitDirectly(exprType.Kind(), dstType.Kind()) {
				em.fb.emitSlice(kLow, kHigh, kMax, src, reg, low, high, max, pos)
			} else {
				em.fb.enterStack()
				tmpType := exprType
				// Slicing preserves the type, except for arrays where the
				// slicing result is a slice.
				if tmpType.Kind() == reflect.Array {
					tmpType = reflect.SliceOf(tmpType.Elem())
				}
				tmp := em.fb.newRegister(tmpType.Kind())
				em.fb.emitSlice(kLow, kHigh, kMax, src, tmp, low, high, max, pos)
				em.changeRegister(false, tmp, reg, tmpType, dstType)
				em.fb.exitStack()
			}
		}

	default:

		panic(fmt.Sprintf("scriggo/emitter: unexpected node type %T (expr: %s)", expr, expr))

	}

	return reg, false
}

// emitBinaryOp emits the code for the binary expression expr and stores the
// result in the register reg of type regType.
func (em *emitter) emitBinaryOp(expr *ast.BinaryOperator, reg int8, regType reflect.Type) {

	var (
		ti   = em.ti(expr)
		typ  = ti.Type
		kind = typ.Kind()
		op   = expr.Operator()
		pos  = expr.Pos()
	)

	// Emit code for complex numbers.
	if kind == reflect.Complex64 || kind == reflect.Complex128 {
		em.emitComplexOperation(typ, expr.Expr1, op, expr.Expr2, reg, regType)
		return
	}

	// Emit code for the operators && and ||.
	if op == ast.OperatorAnd || op == ast.OperatorOr {
		x := reg
		y := int8(0)
		direct := canEmitDirectly(regType.Kind(), reflect.Bool)
		if !direct {
			em.fb.enterStack()
			x = em.fb.newRegister(reflect.Bool)
		}
		if op == ast.OperatorAnd {
			y = 1
		}
		em.emitExprR(expr.Expr1, boolType, x)
		endIf := em.fb.newLabel()
		em.fb.emitIf(true, x, runtime.ConditionEqual, y, reflect.Int, pos)
		em.fb.emitGoto(endIf)
		em.emitExprR(expr.Expr2, boolType, x)
		em.fb.setLabelAddr(endIf)
		if !direct {
			em.changeRegister(false, x, reg, boolType, regType)
			em.fb.exitStack()
		}
		return
	}

	// Emit code where an operand is the predeclared nil.
	if em.ti(expr.Expr1).Nil() || em.ti(expr.Expr2).Nil() {
		em.changeRegister(true, 1, reg, boolType, regType)
		em.emitCondition(expr)
		em.changeRegister(true, 0, reg, boolType, regType)
		return
	}

	// Emit code for the two operands.
	t1 := em.typ(expr.Expr1)
	t2 := em.typ(expr.Expr2)
	x := em.emitExpr(expr.Expr1, t1)
	y, ky := em.emitExprK(expr.Expr2, t2)
	if reg == 0 {
		return
	}

	// Emit code for concatenate the two operands.
	if op == ast.OperatorAddition && t1.Kind() == reflect.String {
		if ky {
			y = em.emitExpr(expr.Expr2, t2)
		}
		if canEmitDirectly(kind, regType.Kind()) {
			em.fb.emitConcat(x, y, reg)
			return
		}
		em.fb.enterStack()
		tmp := em.fb.newRegister(kind)
		em.fb.emitConcat(x, y, tmp)
		em.changeRegister(false, tmp, reg, typ, regType)
		em.fb.exitStack()
		return
	}

	// Emit code for bit operations on the two operands.
	switch op {
	case ast.OperatorBitAnd, ast.OperatorBitOr, ast.OperatorXor, ast.OperatorAndNot:
		z := reg
		direct := canEmitDirectly(kind, regType.Kind())
		if !direct {
			em.fb.enterStack()
			z = em.fb.newRegister(kind)
		}
		switch op {
		case ast.OperatorBitAnd:
			em.fb.emitAnd(ky, x, y, z, kind)
		case ast.OperatorBitOr:
			em.fb.emitOr(ky, x, y, z, kind)
		case ast.OperatorXor:
			em.fb.emitXor(ky, x, y, z, kind)
		case ast.OperatorAndNot:
			em.fb.emitAndNot(ky, x, y, z, kind)
		}
		if !direct {
			em.changeRegister(false, z, reg, typ, regType)
			em.fb.exitStack()
		}
		return
	}

	// Emit code for arithmetic operations.
	if ast.OperatorAddition <= op && op <= ast.OperatorModulo ||
		op == ast.OperatorLeftShift || op == ast.OperatorRightShift {

		if kind == reflect.Int {
			// TODO(gianluca): also add reflect.Float64.
			z := reg
			direct := canEmitDirectly(kind, regType.Kind())
			if !direct {
				em.fb.enterStack()
				z = em.fb.newRegister(kind)
			}
			switch op {
			case ast.OperatorAddition:
				em.fb.emitAdd(ky, x, y, z, kind)
			case ast.OperatorSubtraction:
				em.fb.emitSub(ky, x, y, z, kind)
			case ast.OperatorMultiplication:
				em.fb.emitMul(ky, x, y, z, kind)
			case ast.OperatorDivision:
				em.fb.emitDiv(ky, x, y, z, kind, pos)
			case ast.OperatorModulo:
				em.fb.emitRem(ky, x, y, z, kind, pos)
			case ast.OperatorLeftShift:
				em.fb.emitShl(ky, x, y, z, kind)
			case ast.OperatorRightShift:
				em.fb.emitShr(ky, x, y, z, kind)
			}
			if !direct {
				em.changeRegister(false, z, reg, typ, regType)
				em.fb.exitStack()
			}
			return
		}

		em.fb.enterStack()
		// TODO(gianluca): consider the removal of the allocation of register z using reg directly.
		z := em.fb.newRegister(kind)
		em.changeRegister(false, x, z, typ, typ)
		switch op {
		case ast.OperatorAddition:
			em.fb.emitAdd(ky, z, y, z, kind)
		case ast.OperatorSubtraction:
			em.fb.emitSub(ky, z, y, z, kind)
		case ast.OperatorMultiplication:
			em.fb.emitMul(ky, z, y, z, kind)
		case ast.OperatorDivision:
			em.fb.emitDiv(ky, z, y, z, kind, pos)
		case ast.OperatorModulo:
			em.fb.emitRem(ky, z, y, z, kind, pos)
		case ast.OperatorLeftShift:
			em.fb.emitShl(ky, z, y, z, kind)
		case ast.OperatorRightShift:
			em.fb.emitShr(ky, z, y, z, kind)
		}
		em.changeRegister(false, z, reg, typ, regType)
		em.fb.exitStack()
		return

	}

	// Emit code for a contains operator.
	if op == ast.OperatorContains || op == ast.OperatorNotContains {
		not := op == ast.OperatorNotContains
		z := reg
		directly := canEmitDirectly(reflect.Bool, regType.Kind())
		if !directly {
			em.fb.enterStack()
			z = em.fb.newRegister(reflect.Bool)
		}
		em.fb.emitMove(true, 1, z, reflect.Bool)
		em.emitContains(not, ky, x, y, t1, t2, pos)
		em.fb.emitMove(true, 0, z, reflect.Bool)
		if !directly {
			em.changeRegister(false, z, reg, typ, regType)
			em.fb.exitStack()
		}
		return
	}

	// Emit code for comparison operators.
	z := reg
	directly := canEmitDirectly(kind, regType.Kind())
	if !directly {
		em.fb.enterStack()
		z = em.fb.newRegister(kind)
	}
	em.fb.emitMove(true, 1, z, reflect.Bool)
	em.emitComparison(op, ky, x, y, t1, t2, pos)
	em.fb.emitMove(true, 0, z, reflect.Bool)
	if !directly {
		em.changeRegister(false, z, reg, typ, regType)
		em.fb.exitStack()
	}

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
		em.fb.enterScope()
		workingReg := reg
		if !canEmitDirectly(typ.Kind(), dstType.Kind()) {
			workingReg = em.fb.newRegister(typ.Kind())
		}
		if typ.Kind() == reflect.Slice {
			length := em.compositeLiteralLen(expr)
			k := length <= 127
			if !k {
				r := em.fb.newRegister(reflect.Int)
				em.fb.emitLoad(em.fb.makeIntValue(int64(length)), r, reflect.Int)
				length = int(r)
			}
			em.fb.emitMakeSlice(k, k, typ, int8(length), int8(length), workingReg, expr.Pos())
		} else {
			em.fb.emitMakeArray(typ, workingReg)
		}
		elemKind := typ.Elem().Kind()
		var index int64 = -1
		for _, kv := range expr.KeyValues {
			if kv.Key == nil {
				index++
			} else {
				index = em.ti(kv.Key).Constant.int64()
			}
			// Don't emit code for the zero value.
			if ti := em.ti(kv.Value); ti.HasValue() {
				var isZero bool
				switch elemKind {
				case reflect.Interface, reflect.Func:
					isZero = ti.value == nil
				case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan:
					isZero = reflect.ValueOf(ti.value).IsNil()
				default:
					isZero = ti.IsConstant() && ti.Constant.zero()
				}
				if isZero {
					continue
				}
			}
			em.fb.enterStack()
			indexReg := em.fb.newRegister(reflect.Int)
			if index > 127 {
				em.fb.emitLoad(em.fb.makeIntValue(index), indexReg, reflect.Int)
			} else {
				em.fb.emitMove(true, int8(index), indexReg, reflect.Int)
			}
			elem, k := em.emitExprK(kv.Value, typ.Elem())
			if workingReg != 0 {
				em.fb.emitSetSlice(k, workingReg, elem, indexReg, expr.Pos(), elemKind)
			}
			em.fb.exitStack()
		}
		if !canEmitDirectly(typ.Kind(), dstType.Kind()) {
			em.changeRegister(false, workingReg, reg, typ, dstType)
		}
		em.fb.exitScope()
	case reflect.Struct:
		// Struct should no be created, but its values must be emitted.
		if reg == 0 {
			for _, kv := range expr.KeyValues {
				em.emitExprR(kv.Value, em.typ(kv.Value), 0)
			}
			return reg, false
		}
		// When there are no values in the composite literal, optimize the
		// creation of the struct.
		if len(expr.KeyValues) == 0 {
			if canEmitDirectly(typ.Kind(), dstType.Kind()) {
				em.fb.emitMakeStruct(typ, reg)
				return reg, false
			}
			em.fb.enterStack()
			tmp := em.fb.newRegister(typ.Kind())
			em.fb.emitMakeStruct(typ, tmp)
			em.changeRegister(false, tmp, reg, typ, dstType)
			em.fb.exitStack()
		}
		// Assign key-value pairs to the struct fields.
		em.fb.enterStack()
		var structt int8
		if canEmitDirectly(typ.Kind(), dstType.Kind()) {
			structt = em.fb.newRegister(reflect.Struct)
		} else {
			structt = reg
		}
		em.fb.emitMakeStruct(typ, structt)
		for _, kv := range expr.KeyValues {
			name := kv.Key.(*ast.Identifier).Name
			field, _ := typ.FieldByName(name)
			valueType := em.typ(kv.Value)
			if canEmitDirectly(valueType.Kind(), field.Type.Kind()) {
				value, k := em.emitExprK(kv.Value, valueType)
				index := em.fb.makeFieldIndex(field.Index)
				em.fb.emitSetField(k, structt, index, value, field.Type.Kind())
			} else {
				em.fb.enterStack()
				tmp := em.emitExpr(kv.Value, valueType)
				value := em.fb.newRegister(field.Type.Kind())
				em.changeRegister(false, tmp, value, valueType, field.Type)
				em.fb.exitStack()
				index := em.fb.makeFieldIndex(field.Index)
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
		if size <= 127 {
			em.fb.emitMakeMap(typ, true, int8(size), tmp)
		} else {
			index := em.fb.makeIntValue(int64(size))
			sizeReg := em.fb.newRegister(reflect.Int)
			em.fb.emitLoad(index, sizeReg, reflect.Int)
			em.fb.emitMakeMap(typ, false, sizeReg, tmp)
		}
		for _, kv := range expr.KeyValues {
			em.fb.enterStack()
			key := em.fb.newRegister(typ.Key().Kind())
			em.emitExprR(kv.Key, typ.Key(), key)
			value, k := em.emitExprK(kv.Value, typ.Elem())
			em.fb.emitSetMap(k, tmp, value, key, typ, expr.Pos())
			em.fb.exitStack()
		}
		em.changeRegister(false, tmp, reg, typ, dstType)
	}
	return reg, false
}

// emitSelector emits selector in register reg.
func (em *emitter) emitSelector(v *ast.Selector, reg int8, dstType reflect.Type) {

	ti := em.ti(v)

	// Method value on concrete and interface values.
	if ti.MethodType == methodValueConcrete || ti.MethodType == methodValueInterface {
		expr := v.Expr
		typ := em.typ(expr)
		rcvr := em.emitExpr(expr, typ)
		// MethodValue reads receiver from general.
		if kindToType(typ.Kind()) != generalRegister {
			oldRcvr := rcvr
			rcvr = em.fb.newRegister(reflect.Interface)
			em.fb.emitTypify(false, typ, oldRcvr, rcvr)
		}
		if kindToType(dstType.Kind()) == generalRegister {
			s := em.fb.makeStringValue(v.Ident)
			em.fb.emitMethodValue(s, rcvr, reg, v.Pos())
		} else {
			panic("not implemented")
		}
		return
	}

	// Predefined package variable or imported package variable.
	if index, ok := em.varStore.nonLocalVarIndex(v); ok {
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
	if ident, ok := v.Expr.(*ast.Identifier); ok {
		if sf, ok := em.fnStore.availableScriggoFn(em.pkg, ident.Name+"."+v.Ident); ok {
			if reg == 0 {
				return
			}
			index := em.fnStore.scriggoFnIndex(sf)
			em.fb.emitLoadFunc(false, index, reg)
			em.changeRegister(false, reg, reg, em.typ(v), dstType)
			return
		}
	}

	// Struct field.
	expr := v.Expr
	if op, ok := expr.(*ast.UnaryOperator); ok && op.Op == ast.OperatorPointer {
		expr = op.Expr
	}
	typ := em.typ(expr)
	exprReg := em.emitExpr(expr, typ)
	var field reflect.StructField
	if typ.Kind() == reflect.Ptr {
		field, _ = typ.Elem().FieldByName(v.Ident)
	} else {
		field, _ = typ.FieldByName(v.Ident)
	}
	index := em.fb.makeFieldIndex(field.Index)
	if canEmitDirectly(field.Type.Kind(), dstType.Kind()) {
		em.fb.emitField(exprReg, index, reg, dstType.Kind())
		return
	}
	// TODO: add enter/exit stack method calls.
	tmp := em.fb.newRegister(field.Type.Kind())
	em.fb.emitField(exprReg, index, tmp, field.Type.Kind())
	em.changeRegister(false, tmp, reg, field.Type, dstType)

}

// emitUnaryOp emits the code for the unary expression expr and stores the
// result in the register reg of type regType.
func (em *emitter) emitUnaryOp(expr *ast.UnaryOperator, reg int8, regType reflect.Type) {

	var (
		exprType    = em.typ(expr)
		exprKind    = exprType.Kind()
		operand     = expr.Expr
		operandType = em.typ(operand)
		operandKind = operandType.Kind()
		op          = expr.Operator()
	)

	// Emit the internal operators Zero and NotZero.
	if op == internalOperatorZero || op == internalOperatorNotZero {
		directly := canEmitDirectly(operandKind, regType.Kind())
		em.fb.enterStack()
		src := em.emitExpr(operand, operandType)
		dst := reg
		if !directly {
			dst = em.fb.newRegister(reflect.Bool)
		}
		if op == internalOperatorNotZero {
			em.fb.emitNotZero(operandKind, dst, src)
		} else {
			em.fb.emitZero(operandKind, dst, src)
		}
		if !directly {
			em.changeRegister(false, dst, reg, boolType, regType)
		}
		em.fb.exitStack()
		return
	}

	// Emit code for a receive operation.
	//
	//          <- ch
	//  v     = <- ch
	//  v, ok = <- ch
	//
	if op == ast.OperatorReceive {
		ch := em.emitExpr(operand, operandType)
		if reg == 0 {
			em.fb.emitReceive(ch, 0, 0)
			return
		}
		if canEmitDirectly(exprKind, regType.Kind()) {
			em.fb.emitReceive(ch, 0, reg)
			return
		}
		dst := em.fb.newRegister(exprKind)
		em.fb.emitReceive(ch, 0, dst)
		em.changeRegister(false, dst, reg, exprType, regType)
		return
	}

	// Emit code for the negation of a complex number.
	if exprKind == reflect.Complex64 || exprKind == reflect.Complex128 {
		if op != ast.OperatorSubtraction {
			panic("bug: expected operator subtraction")
		}
		stackShift := em.fb.currentStackShift()
		em.fb.enterScope()
		index := em.fb.complexOperationIndex(ast.OperatorSubtraction, true)
		src := em.fb.newRegister(reflect.Complex128)
		arg := em.fb.newRegister(reflect.Complex128)
		em.fb.enterScope()
		em.emitExprR(operand, exprType, arg)
		em.fb.exitScope()
		em.fb.emitCallNative(index, 0, stackShift, expr.Pos())
		em.changeRegister(false, src, reg, exprType, regType)
		em.fb.exitScope()
		return
	}

	// Emit code for the other operand types.
	switch op {

	// !operand
	case ast.OperatorNot:
		if reg == 0 {
			em.emitExprR(operand, operandType, 0)
			return
		}
		em.fb.enterScope()
		// TODO(gianluca): improve this code.
		y := em.emitExpr(operand, operandType)
		x := em.fb.newRegister(reflect.Int)
		em.changeRegister(true, 1, x, intType, intType)
		em.fb.emitSub(false, x, y, x, operandKind)
		em.changeRegister(false, x, reg, operandType, regType)
		em.fb.exitScope()

	// *operand
	case ast.OperatorPointer:
		exprReg := em.emitExpr(operand, operandType)
		if canEmitDirectly(exprType.Kind(), regType.Kind()) {
			em.changeRegister(false, -exprReg, reg, operandType.Elem(), regType)
			return
		}
		r := em.fb.newRegister(operandType.Elem().Kind())
		em.changeRegister(false, -exprReg, r, operandType.Elem(), operandType.Elem())
		em.changeRegister(false, r, reg, operandType.Elem(), regType)

	// &operand
	case ast.OperatorAddress:
		switch operand := operand.(type) {

		// &a
		case *ast.Identifier:
			if em.fb.declaredInFunc(operand.Name) {
				r := em.fb.scopeLookup(operand.Name)
				em.fb.emitNew(em.types.PtrTo(exprType), reg)
				em.fb.emitMove(false, -r, reg, regType.Kind())
				return
			}
			// Address of a non-local variable.
			index, ok := em.varStore.nonLocalVarIndex(operand)
			if !ok {
				panic("BUG")
			}
			em.fb.enterStack()
			r := em.fb.newRegister(reflect.Ptr)
			em.fb.emitGetVarAddr(index, r)
			em.changeRegister(false, r, reg, em.types.PtrTo(operandType), regType)
			em.fb.exitStack()

		// &*a
		case *ast.UnaryOperator:
			// Deference the pointer to check for "invalid memory address"
			// errors, then discard the result.
			em.fb.enterStack()
			pointedElemType := regType.Elem()
			pointer := em.emitExpr(operand, pointedElemType)
			dst := em.fb.newRegister(pointedElemType.Kind())
			em.changeRegister(false, -pointer, dst, pointedElemType, pointedElemType)
			em.fb.exitStack()
			// The pointer is valid, so &*a is equivalent to a.
			em.emitExprR(operand.Expr, regType, reg)

		// &v[i]
		// (where v is a slice or an addressable array)
		case *ast.Index:
			expr := em.emitExpr(operand.Expr, em.typ(operand.Expr))
			index := em.emitExpr(operand.Index, intType)
			pos := operand.Expr.Pos()
			if canEmitDirectly(exprType.Kind(), regType.Kind()) {
				em.fb.emitAddr(expr, index, reg, pos)
			}
			em.fb.enterStack()
			dest := em.fb.newRegister(exprType.Kind())
			em.fb.emitAddr(expr, index, dest, pos)
			em.changeRegister(false, dest, reg, exprType, regType)
			em.fb.exitStack()

		// &s.Field
		case *ast.Selector:
			// Address of a non-local variable.
			if index, ok := em.varStore.nonLocalVarIndex(operand); ok {
				if canEmitDirectly(operandKind, regType.Kind()) {
					em.fb.emitGetVarAddr(index, reg)
					return
				}
				r := em.fb.newRegister(operandKind)
				em.fb.emitGetVarAddr(index, r)
				em.changeRegister(false, r, reg, operandType, regType)
				return
			}
			expr := operand.Expr
			if op, ok := expr.(*ast.UnaryOperator); ok && op.Op == ast.OperatorPointer {
				expr = op.Expr
			}
			operandExprType := em.typ(expr)
			exprReg := em.emitExpr(expr, operandExprType)
			var field reflect.StructField
			if operandExprType.Kind() == reflect.Ptr {
				field, _ = operandExprType.Elem().FieldByName(operand.Ident)
			} else {
				field, _ = operandExprType.FieldByName(operand.Ident)
			}
			index := em.fb.makeFieldIndex(field.Index)
			pos := operand.Expr.Pos()
			if canEmitDirectly(em.types.PtrTo(field.Type).Kind(), regType.Kind()) {
				em.fb.emitAddr(exprReg, index, reg, pos)
				return
			}
			em.fb.enterStack()
			dest := em.fb.newRegister(reflect.Ptr)
			em.fb.emitAddr(exprReg, index, dest, pos)
			em.changeRegister(false, dest, reg, em.types.PtrTo(field.Type), regType)
			em.fb.exitStack()

		// &T{..}
		case *ast.CompositeLiteral:
			z := em.fb.newRegister(reflect.Ptr)
			em.fb.emitNew(operandType, z)
			em.emitExprR(operand, operandType, -z)
			em.changeRegister(false, z, reg, exprType, regType)

		default:
			panic("unexpected operand")
		}

	// +operand
	case ast.OperatorAddition:
		// Nothing to do.

	// -operand
	case ast.OperatorSubtraction:
		if reg == 0 {
			em.emitExprR(operand, regType, 0)
			return
		}
		em.fb.enterScope()
		y := em.emitExpr(operand, operandType)
		if canEmitDirectly(operandKind, regType.Kind()) {
			em.fb.emitNeg(y, reg, regType.Kind())
		} else {
			z := em.fb.newRegister(operandKind)
			em.fb.emitNeg(y, z, operandKind)
			em.changeRegister(false, z, reg, operandType, regType)
		}
		em.fb.exitScope()

	// ^operand
	case ast.OperatorXor:
		if reg == 0 {
			em.emitExprR(operand, regType, 0)
			return
		}
		em.fb.enterStack()
		// TODO(gianluca): improve this code.
		y := em.fb.newRegister(operandKind)
		em.emitExprR(operand, operandType, y)
		x := em.fb.newRegister(operandKind)
		if isSigned(operandKind) {
			em.changeRegister(true, -1, x, operandType, operandType)
		} else {
			m := maxUnsigned(operandKind)
			em.fb.emitLoad(em.fb.makeIntValue(int64(m)), x, reflect.Int)
		}
		em.fb.emitXor(false, x, y, x, operandKind)
		em.changeRegister(false, x, reg, operandType, regType)
		em.fb.exitStack()

	default:
		panic("unexpected operator")

	}

}
