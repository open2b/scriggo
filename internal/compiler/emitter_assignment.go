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
//  assignIndirectDeclaration  register             (unused)                      type of the variable
//  assignLocalVariable        register             (unused)                      type of the variable
//  assignMapIndex             map register         key register                  type of the map
//  assignPointerIndirection   register             (unused)                      type of the *v expression
//  assignSliceIndex           slice register       index register                type of the slice
//  assignStructSelector       struct register      index of the field (const)    type of the struct
//
func (em *emitter) newAddress(target assignmentTarget, addressedType reflect.Type, op1, op2 int8, pos *ast.Position) address {
	return address{em: em, target: target, addressedType: addressedType, op1: op1, op2: op2, pos: pos}
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

// assignValuesToAddresses assigns values to addresses.
func (em *emitter) assignValuesToAddresses(addresses []address, values []ast.Expression) {

	if len(addresses) == 1 && len(values) == 1 {
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
