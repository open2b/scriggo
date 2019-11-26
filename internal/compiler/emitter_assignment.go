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

// assignmentTarget is the target of an assignment.
type assignmentTarget int8

const (
	assignBlank          assignmentTarget = iota // Assign to a blank identifier.
	assignClosureVar                             // Assign to a closure variable.
	assignLocalVar                               // Assign to a local variable.
	assignMapIndex                               // Assign to a map index.
	assignNewIndirectVar                         // Assign to a new indirect variable.
	assignPtrIndirection                         // Assign to a pointer indirection.
	assignSliceIndex                             // Assign to a slice index.
	assignStructSelector                         // Assign to a struct selector.
)

// address represents an element on the left side of an assignment.
// See em.newAddress for a detailed explanation of the fields.
type address struct {
	em            *emitter
	target        assignmentTarget
	addressedType reflect.Type
	op1, op2      int8
	pos           *ast.Position
	operator      ast.AssignmentType
}

// newAddress returns a new address that represent one element on the left side
// of an assignment.
//
// pos is the position of the assignment in the source code.
//
// To get an explanation of the different assignment targets, see the
// declaration of the assignmentTarget constants. The meaning of the argument
// op1, op2 and addressedType is explained in the table below:
//
//  Assignment target          op1                  op2                           Addressed Type
//
//  assignBlank                (unused)             (unused)                      (unused)
//  assignClosureVariable      msb of the var index lsb of the var index          type of the variable
//  assignNewIndirectVar       register             (unused)                      type of the variable
//  assignLocalVariable        register             (unused)                      type of the variable
//  assignMapIndex             map register         key register                  type of the map
//  assignPtrIndirection       register             (unused)                      type of the *v expression
//  assignSliceIndex           slice register       index register                type of the slice
//  assignStructSelector       struct register      index of the field (const)    type of the struct
//
// TODO: this method as a lot of arguments, consider splitting this method in
// submethods.
func (em *emitter) newAddress(target assignmentTarget, addressedType reflect.Type, operator ast.AssignmentType, op1, op2 int8, pos *ast.Position) address {
	return address{em: em, target: target, addressedType: addressedType, operator: operator, op1: op1, op2: op2, pos: pos}
}

func (em *emitter) newAddressClosureVariable(index int16, varType reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	msb, lsb := encodeInt16(index)
	return address{
		addressedType: varType,
		em:            em,
		op1:           msb,
		op2:           lsb,
		operator:      op,
		pos:           pos,
		target:        assignClosureVar,
	}
}

func (em *emitter) newAddressBlankIdent(pos *ast.Position) address {
	return address{em: em, target: assignBlank, pos: pos}
}

func (em *emitter) newAddressLocalVar(reg int8, varType reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{em: em, target: assignLocalVar, addressedType: varType, op1: reg, pos: pos, operator: op}
}

func (em *emitter) newAddressPtrIndirect(reg int8, pointedType reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{
		addressedType: pointedType,
		em:            em,
		op1:           reg,
		operator:      op,
		pos:           pos,
		target:        assignPtrIndirection,
	}
}

func (em *emitter) newAddressStructSelector(structReg int8, kFieldIndex int8, structType reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{em: em, target: assignStructSelector, addressedType: structType, op1: structReg, op2: kFieldIndex, pos: pos, operator: op}
}

func (em *emitter) newAddressNewIndirectVar(reg int8, varType reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{em: em, target: assignNewIndirectVar, addressedType: varType, op1: reg, pos: pos, operator: op}
}

func (em *emitter) newAddressMapIndex(mapReg int8, keyReg int8, mapType reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{em: em, target: assignMapIndex, addressedType: mapType, op1: mapReg, op2: keyReg, pos: pos, operator: op}
}

func (em *emitter) newAddressSliceIndex(sliceReg int8, indexReg int8, sliceType reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{em: em, target: assignSliceIndex, addressedType: sliceType, op1: sliceReg, op2: indexReg, pos: pos, operator: op}
}

// assign assigns value, with type valueType, to the address. If k is true
// value is a constant otherwise is a register.
func (a address) assign(k bool, value int8, valueType reflect.Type) {
	switch a.target {
	case assignClosureVar:
		a.em.fb.emitSetVar(k, value, int(decodeInt16(a.op1, a.op2)), a.addressedType.Kind())
	case assignBlank:
		// Nothing to do.
	case assignLocalVar:
		a.em.changeRegister(k, value, a.op1, a.targetType(), a.addressedType)
	case assignNewIndirectVar:
		a.em.fb.emitNew(a.addressedType, -a.op1)
		a.em.changeRegister(k, value, a.op1, a.targetType(), a.addressedType)
	case assignPtrIndirection:
		a.em.changeRegister(k, value, -a.op1, a.targetType(), a.addressedType)
	case assignSliceIndex:
		a.em.fb.emitSetSlice(k, a.op1, value, a.op2, a.pos, valueType.Kind())
	case assignMapIndex:
		a.em.fb.emitSetMap(k, a.op1, value, a.op2, a.addressedType, a.pos)
	case assignStructSelector:
		a.em.fb.emitSetField(k, a.op1, a.op2, value, valueType.Kind())
	}
}

// targetType returns the type of the target of the assignment. The target type
// can be different from the addressed type; for example in a slice assignment
// the addressed type is the type of the slice (eg. '[]int'), while the target
// type is 'int'.
func (a address) targetType() reflect.Type {
	switch a.target {
	case assignBlank:
		return nil
	case assignClosureVar:
		return a.addressedType
	case assignNewIndirectVar:
		return a.addressedType
	case assignLocalVar:
		return a.addressedType
	case assignMapIndex:
		return a.addressedType.Elem()
	case assignPtrIndirection:
		return a.addressedType.Elem()
	case assignSliceIndex:
		return a.addressedType.Elem()
	case assignStructSelector:
		encodedField := a.em.fb.fn.Constants.Int[a.op2]
		index := decodeFieldIndex(encodedField)
		return a.addressedType.FieldByIndex(index).Type
	}
	return nil
}

// emitAssignmentOperation emits an assignment operation
//
//      x op= rh
//
// addr represents the address of x and rh is the right hand side of the
// assignment operation.
func (em *emitter) emitAssignmentOperation(addr address, rh ast.Expression) {

	addrTyp := addr.addressedType // type of the addressed element (eg. type of the slice).
	typ := addr.targetType()      // type of the "target" (eg. type of the slice element).

	// Emit the code that evaluates the left side of the assignment.
	lhReg := em.fb.newRegister(typ.Kind())
	switch addr.target {
	case assignBlank, assignNewIndirectVar:
		panic("Type checking BUG")
	case assignClosureVar:
		em.fb.emitGetVar(int(decodeInt16(addr.op1, addr.op2)), lhReg, addrTyp.Kind())
	case assignLocalVar:
		em.changeRegister(false, addr.op1, lhReg, addrTyp, typ)
	case assignMapIndex,
		assignSliceIndex:
		em.fb.emitIndex(false, addr.op1, addr.op2, lhReg, addrTyp, addr.pos, false)
	case assignPtrIndirection:
		em.changeRegister(false, -addr.op1, lhReg, addrTyp, addrTyp)
	case assignStructSelector:
		em.fb.emitField(addr.op1, addr.op2, lhReg, typ.Kind(), false)
	}

	// Emit the code that evaluataes the right side of the assignment.
	rhReg := em.emitExpr(rh, typ)

	// Emit the code that computes the result of the operation; such result will
	// be put back into the left side.
	result := em.fb.newRegister(typ.Kind())
	if k := typ.Kind(); k == reflect.Complex64 || k == reflect.Complex128 {
		// Operation on complex numbers.
		stackShift := em.fb.currentStackShift()
		em.fb.enterScope()
		ret := em.fb.newRegister(reflect.Complex128)
		c1 := em.fb.newRegister(reflect.Complex128)
		c2 := em.fb.newRegister(reflect.Complex128)
		em.changeRegister(false, lhReg, c1, typ, typ)
		em.changeRegister(false, rhReg, c2, typ, typ)
		index := em.fb.complexOperationIndex(operatorFromAssignmentType(addr.operator), false)
		em.fb.emitCallPredefined(index, 0, stackShift, addr.pos)
		em.changeRegister(false, ret, result, typ, typ)
		em.fb.exitScope()
		addr.assign(false, result, typ)
	} else {
		switch addr.operator {
		case ast.AssignmentAddition:
			if typ.Kind() == reflect.String {
				em.fb.emitConcat(lhReg, rhReg, result)
			} else {
				em.fb.emitAdd(false, lhReg, rhReg, result, typ.Kind())
			}
		case ast.AssignmentSubtraction:
			em.fb.emitSub(false, lhReg, rhReg, result, typ.Kind())
		case ast.AssignmentMultiplication:
			em.fb.emitMul(false, lhReg, rhReg, result, typ.Kind())
		case ast.AssignmentDivision:
			em.fb.emitDiv(false, lhReg, rhReg, result, typ.Kind(), addr.pos)
		case ast.AssignmentModulo:
			em.fb.emitRem(false, lhReg, rhReg, result, typ.Kind(), addr.pos)
		case ast.AssignmentAnd:
			em.fb.emitAnd(false, lhReg, rhReg, result, typ.Kind())
		case ast.AssignmentOr:
			em.fb.emitOr(false, lhReg, rhReg, result, typ.Kind())
		case ast.AssignmentXor:
			em.fb.emitXor(false, lhReg, rhReg, result, typ.Kind())
		case ast.AssignmentAndNot:
			em.fb.emitAndNot(false, lhReg, rhReg, result, typ.Kind())
		case ast.AssignmentLeftShift:
			em.fb.emitLeftShift(false, lhReg, rhReg, result, typ.Kind())
		case ast.AssignmentRightShift:
			em.fb.emitRightShift(false, lhReg, rhReg, result, typ.Kind())
		}
	}

	// Put back the result into the left side of the assignment.
	addr.assign(false, result, typ)

}

// assignValuesToAddresses assigns values to addresses.
func (em *emitter) assignValuesToAddresses(addresses []address, values []ast.Expression) {

	if len(addresses) == 1 && len(values) == 1 {
		// Assignment operation.
		if op := addresses[0].operator; ast.AssignmentAddition <= op && op <= ast.AssignmentRightShift {
			em.emitAssignmentOperation(addresses[0], values[0])
			return
		}
		// Optimize the case when there's just one element on the left and one
		// element on the right side.
		t := addresses[0].targetType()
		if t == nil {
			t = em.ti(values[0]).Type
		}
		v, k := em.emitExprK(values[0], t)
		addresses[0].assign(k, v, t)
		return
	}

	if len(addresses) == len(values) {
		regs := make([]int8, len(values))
		types := make([]reflect.Type, len(values))
		ks := make([]bool, len(values))
		for i := range values {
			types[i] = em.ti(values[i]).Type
			regs[i], ks[i] = em.emitExprK(values[i], types[i])
			if !ks[i] {
				regs[i] = em.fb.newRegister(types[i].Kind())
				em.emitExprR(values[i], types[i], regs[i])
			}
		}
		for i, addr := range addresses {
			addr.assign(ks[i], regs[i], types[i])
		}
		return
	}

	switch valueExpr := values[0].(type) {

	case *ast.Call:
		regs, retTypes := em.emitCallNode(valueExpr, false, false)
		for i, addr := range addresses {
			addr.assign(false, regs[i], retTypes[i])
		}

	case *ast.Index: // map index.
		mapType := em.ti(valueExpr.Expr).Type
		mapp := em.emitExpr(valueExpr.Expr, mapType)
		keyType := em.ti(valueExpr.Index).Type
		key, kKey := em.emitExprK(valueExpr.Index, keyType)
		valueType := mapType.Elem()
		value := em.fb.newRegister(valueType.Kind())
		okType := addresses[1].addressedType
		okReg := em.fb.newRegister(reflect.Bool)
		pos := valueExpr.Pos()
		em.fb.emitIndex(kKey, mapp, key, value, mapType, pos, false)
		em.fb.emitMove(true, 1, okReg, reflect.Bool, false)
		em.fb.emitIf(false, 0, runtime.ConditionOK, 0, reflect.Interface, pos)
		em.fb.emitMove(true, 0, okReg, reflect.Bool, false)
		addresses[0].assign(false, value, valueType)
		addresses[1].assign(false, okReg, okType)

	case *ast.TypeAssertion:
		typ := em.ti(valueExpr.Type).Type
		expr := em.emitExpr(valueExpr.Expr, emptyInterfaceType)
		okType := addresses[1].addressedType
		ok := em.fb.newRegister(reflect.Bool)
		em.fb.emitMove(true, 1, ok, reflect.Bool, false)
		result := em.fb.newRegister(typ.Kind())
		em.fb.emitAssert(expr, typ, result)
		em.fb.emitMove(true, 0, ok, reflect.Bool, false)
		addresses[0].assign(false, result, typ)
		addresses[1].assign(false, ok, okType)

	case *ast.UnaryOperator: // receive from channel.
		chanType := em.ti(valueExpr.Expr).Type
		valueType := em.ti(valueExpr).Type
		okType := addresses[1].addressedType
		chann := em.emitExpr(valueExpr.Expr, chanType)
		ok := em.fb.newRegister(reflect.Bool)
		value := em.fb.newRegister(valueType.Kind())
		em.fb.emitReceive(chann, ok, value)
		addresses[0].assign(false, value, valueType)
		addresses[1].assign(false, ok, okType)

	}

}
