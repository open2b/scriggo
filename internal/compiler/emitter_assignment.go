// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/runtime"
)

// address represents an element on the left side of an assignment.
// The meaning of the various fields is explained in the constructor methods for
// this type.
type address struct {
	em            *emitter           // a reference to the current emitter.
	target        assignmentTarget   // target of the assignment.
	addressedType reflect.Type       // type of the addressed type (see the methods below).
	op1, op2      int8               // two bytes for store addressing information (see the methods below).
	pos           *ast.Position      // position of the addressed element in the source code.
	operator      ast.AssignmentType // type of the assignment that involves this address.
	nonLocal      int                // index of non-local vars. Not relevant if the assignment happens locally.
}

// assignmentTarget is the target of an assignment.
type assignmentTarget int16

const (
	assignBlank          assignmentTarget = iota // Assign to a blank identifier.
	assignNewIndirectVar                         // Assign to a new indirect variable.
	assignPtrIndirection                         // Assign to a pointer indirection.

	// Assign to a variable.
	assignLocalVar    // local.
	assignNonLocalVar // non-local.

	// Assign to a map index.
	assignLocalMapIndex    // local.
	assignNonLocalMapIndex // non-local.

	// Assign to a slice index.
	assignLocalSliceIndex    // local.
	assignNonLocalSliceIndex // non-local.

	// Assign to a struct selector.
	assignLocalStructSelector    // local.
	assignNonLocalStructSelector // non-local.
)

// addressBlankIdent returns a new address that addresses a blank identifier.
// op is the type of the assignment that involves this address, and pos is the
// position of the assignment in the source code.
func (em *emitter) addressBlankIdent(pos *ast.Position) address {
	return address{
		em:     em,
		pos:    pos,
		target: assignBlank,
	}
}

// addressLocalVar returns a new address that addresses the local variable with
// the given type that is stored in reg.
// op is the type of the assignment that involves this address, and pos is the
// position of the assignment in the source code.
func (em *emitter) addressLocalVar(reg int8, typ reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{
		addressedType: typ,
		em:            em,
		op1:           reg,
		operator:      op,
		pos:           pos,
		target:        assignLocalVar,
	}
}

// addressNonLocalVar returns a new address that addresses the non-local
// variable with the given type that is indexed by index. op is the type of the
// assignment that involves this address, and pos is the position of the
// assignment in the source code.
func (em *emitter) addressNonLocalVar(index int, typ reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{
		addressedType: typ,
		em:            em,
		operator:      op,
		pos:           pos,
		target:        assignNonLocalVar,
		nonLocal:      index,
	}
}

// addressLocalMapIndex returns a new address that addresses a local map index
// expression, with the map and key stored into the given registers. op is the
// type of the assignment that involves this address, and pos is the position
// of the assignment in the source code.
func (em *emitter) addressLocalMapIndex(mapReg int8, keyReg int8, mapType reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{
		addressedType: mapType,
		em:            em,
		op1:           mapReg,
		op2:           keyReg,
		operator:      op,
		pos:           pos,
		target:        assignLocalMapIndex,
	}
}

// addressNonLocalMapIndex returns a new address that addresses a non-local map
// index expression, with the evaluated map and key stored into the given
// registers. nonLocalMap refers to the index of the non-local map. op is the
// type of the assignment that involves this address, and pos is the position
// of the assignment in the source code.
func (em *emitter) addressNonLocalMapIndex(nonLocalMap int, mapReg int8, keyReg int8, mapType reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{
		addressedType: mapType,
		em:            em,
		op1:           mapReg,
		op2:           keyReg,
		operator:      op,
		pos:           pos,
		target:        assignNonLocalMapIndex,
		nonLocal:      nonLocalMap,
	}
}

// addressNewIndirectVar returns a new address that addresses a new variable
// declared as 'indirect' that is going to be stored at the given register.
// op is the type of the assignment that involves this address, and pos is the
// position of the assignment in the source code.
func (em *emitter) addressNewIndirectVar(reg int8, typ reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{
		addressedType: typ,
		em:            em,
		op1:           reg,
		operator:      op,
		pos:           pos,
		target:        assignNewIndirectVar,
	}
}

// addressPtrIndirect returns a new address that addresses a pointer
// indirection. reg contains the pointed value, and pointedType is its type.
// op is the type of the assignment that involves this address, and pos is the
// position of the assignment in the source code.
func (em *emitter) addressPtrIndirect(reg int8, pointedType reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{
		addressedType: pointedType,
		em:            em,
		op1:           reg,
		operator:      op,
		pos:           pos,
		target:        assignPtrIndirection,
	}
}

// addressSliceIndex returns a new address that addresses an index expression
// of a slice stored in a local register. sliceReg is the register that holds
// the slice and indexReg is the register that holds the index of the slice. op
// is the type of the assignment that involves this address, and pos is the
// position of the assignment in the source code.
func (em *emitter) addressSliceIndex(sliceReg int8, indexReg int8, sliceType reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{
		addressedType: sliceType,
		em:            em,
		op1:           sliceReg,
		op2:           indexReg,
		operator:      op,
		pos:           pos,
		target:        assignLocalSliceIndex,
	}
}

// addressGlobalSliceIndex returns a new address that addresses an index
// expression of a non-local slice. sliceReg is the register that holds the
// evaluated slice and indexReg is the register that holds the index of the
// slice. sliceIndex is the index of the non-local slice. op is the type of the
// assignment that involves this address, and pos is the position of the
// assignment in the source code.
func (em *emitter) addressGlobalSliceIndex(sliceIndex int, sliceReg int8, indexReg int8, sliceType reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{
		addressedType: sliceType,
		em:            em,
		op1:           sliceReg,
		op2:           indexReg,
		operator:      op,
		pos:           pos,
		target:        assignNonLocalSliceIndex,
		nonLocal:      sliceIndex,
	}
}

// addressLocalStructSelector returns a new address that addresses a local
// struct field expression. structReg is the register that holds the struct
// value and kFieldIndex is the index of the integer constant that contains the
// encoded slice of the field index. op is the type of the assignment that
// involves this address, and pos is the position of the assignment in the
// source code.
func (em *emitter) addressLocalStructSelector(structReg int8, kFieldIndex int8, structType reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{
		addressedType: structType,
		em:            em,
		op1:           structReg,
		op2:           kFieldIndex,
		operator:      op,
		pos:           pos,
		target:        assignLocalStructSelector,
	}
}

// addressNonLocalStructSelector returns a new address that addresses a
// non-local struct (located at structIndex) field expression. localStructReg
// is the register that holds the evaluated struct value and kFieldIndex is the
// index of the integer constant that contains the encoded slice of the field
// index. op is the type of the assignment that involves this address, and pos
// is the position of the assignment in the source code.
func (em *emitter) addressNonLocalStructSelector(structIndex int, localStructReg int8, kFieldIndex int8, structType reflect.Type, pos *ast.Position, op ast.AssignmentType) address {
	return address{
		addressedType: structType,
		em:            em,
		op1:           localStructReg,
		op2:           kFieldIndex,
		operator:      op,
		pos:           pos,
		target:        assignNonLocalStructSelector,
		nonLocal:      structIndex,
	}
}

// assign assigns value, with type valueType, to the address. If k is true
// value is a constant otherwise is a register.
func (a address) assign(k bool, value int8, valueType reflect.Type) {
	switch a.target {
	case assignNonLocalVar:
		a.em.fb.emitSetVar(k, value, a.nonLocal, a.addressedType.Kind())
	case assignBlank:
		// Nothing to do.
	case assignLocalVar:
		a.em.changeRegister(k, value, a.op1, a.targetType(), a.addressedType)
	case assignNewIndirectVar:
		a.em.fb.emitNew(a.addressedType, -a.op1)
		a.em.changeRegister(k, value, a.op1, a.targetType(), a.addressedType)
	case assignPtrIndirection:
		a.em.changeRegister(k, value, -a.op1, a.targetType(), a.addressedType)
	case assignLocalSliceIndex:
		a.em.fb.emitSetSlice(k, a.op1, value, a.op2, a.pos, valueType.Kind())
	case assignNonLocalSliceIndex:
		a.em.fb.emitSetSlice(k, a.op1, value, a.op2, a.pos, valueType.Kind())
		a.em.fb.emitSetVar(false, a.op1, a.nonLocal, a.addressedType.Kind())
	case assignLocalMapIndex:
		a.em.fb.emitSetMap(k, a.op1, value, a.op2, a.addressedType, a.pos)
	case assignNonLocalMapIndex:
		a.em.fb.emitSetMap(k, a.op1, value, a.op2, a.addressedType, a.pos)
		a.em.fb.emitSetVar(false, a.op1, a.nonLocal, a.addressedType.Kind())
	case assignLocalStructSelector:
		a.em.fb.emitSetField(k, a.op1, a.op2, value, valueType.Kind())
	case assignNonLocalStructSelector:
		a.em.fb.emitSetField(k, a.op1, a.op2, value, valueType.Kind())
		a.em.fb.emitSetVar(false, a.op1, a.nonLocal, a.addressedType.Kind())
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
	case assignNonLocalVar:
		return a.addressedType
	case assignNewIndirectVar:
		return a.addressedType
	case assignLocalVar:
		return a.addressedType
	case assignLocalMapIndex,
		assignNonLocalMapIndex:
		return a.addressedType.Elem()
	case assignPtrIndirection:
		return a.addressedType.Elem()
	case assignLocalSliceIndex,
		assignNonLocalSliceIndex:
		return a.addressedType.Elem()
	case assignLocalStructSelector,
		assignNonLocalStructSelector:
		index := a.em.fb.fn.FieldIndexes[a.op2]
		typ := a.addressedType
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		return typ.FieldByIndex(index).Type
	}
	return nil
}

// emitAssignmentOperation emits an assignment operation
//
//	x op= rh
//
// addr represents the address of x and rh is the right hand side of the
// assignment operation.
func (em *emitter) emitAssignmentOperation(addr address, rh ast.Expression) {

	addrTyp := addr.addressedType // type of the addressed element (eg. type of the slice).
	typ := addr.targetType()      // type of the "target" (eg. type of the slice element).

	// Emit the code that evaluates the left side of the assignment.
	c := em.fb.newRegister(typ.Kind())
	switch addr.target {
	case assignBlank, assignNewIndirectVar:
		panic(internalError("unexpected, this is a type checking bug"))
	case assignNonLocalVar:
		em.fb.emitGetVar(addr.nonLocal, c, addrTyp.Kind())
	case assignLocalVar:
		em.changeRegister(false, addr.op1, c, addrTyp, typ)
	case assignLocalMapIndex,
		assignLocalSliceIndex,
		assignNonLocalMapIndex,
		assignNonLocalSliceIndex:
		em.fb.emitIndex(false, addr.op1, addr.op2, c, addrTyp, addr.pos, false)
	case assignPtrIndirection:
		em.changeRegister(false, addr.op1, c, addrTyp, addrTyp)
	case assignLocalStructSelector,
		assignNonLocalStructSelector:
		em.fb.emitField(addr.op1, addr.op2, c, typ.Kind())
	}

	// Emit the code that evaluates the right side of the assignment.
	// TODO: use k?
	b := em.fb.newRegister(typ.Kind())
	em.emitExprR(rh, typ, b)

	// Emit the code that computes the result of the operation; such result will
	// be put back into the left side.
	if k := typ.Kind(); k == reflect.Complex64 || k == reflect.Complex128 {
		// Operation on complex numbers.
		stackShift := em.fb.currentStackShift()
		em.fb.enterScope()
		ret := em.fb.newRegister(reflect.Complex128)
		c1 := em.fb.newRegister(reflect.Complex128)
		c2 := em.fb.newRegister(reflect.Complex128)
		em.changeRegister(false, b, c1, typ, typ)
		em.changeRegister(false, c, c2, typ, typ)
		index := em.fb.complexOperationIndex(operatorFromAssignmentType(addr.operator), false)
		em.fb.emitCallNative(index, 0, stackShift, addr.pos)
		em.changeRegister(false, ret, c, typ, typ)
		em.fb.exitScope()
		addr.assign(false, c, typ)
	} else {
		switch addr.operator {
		case ast.AssignmentAddition:
			if typ.Kind() == reflect.String {
				em.fb.emitConcat(c, b, c)
			} else {
				em.fb.emitAdd(false, c, b, c, typ.Kind())
			}
		case ast.AssignmentSubtraction:
			em.fb.emitSub(false, c, b, c, typ.Kind())
		case ast.AssignmentMultiplication:
			em.fb.emitMul(false, c, b, c, typ.Kind())
		case ast.AssignmentDivision:
			em.fb.emitDiv(false, c, b, c, typ.Kind(), addr.pos)
		case ast.AssignmentModulo:
			em.fb.emitRem(false, c, b, c, typ.Kind(), addr.pos)
		case ast.AssignmentAnd:
			em.fb.emitAnd(false, c, b, c, typ.Kind())
		case ast.AssignmentOr:
			em.fb.emitOr(false, c, b, c, typ.Kind())
		case ast.AssignmentXor:
			em.fb.emitXor(false, c, b, c, typ.Kind())
		case ast.AssignmentAndNot:
			em.fb.emitAndNot(false, c, b, c, typ.Kind())
		case ast.AssignmentLeftShift:
			em.fb.emitShl(false, c, b, c, typ.Kind())
		case ast.AssignmentRightShift:
			em.fb.emitShr(false, c, b, c, typ.Kind())
		}
	}

	// Put back the result into the left side of the assignment.
	addr.assign(false, c, typ)

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
			t = em.typ(values[0])
		}
		em.fb.enterStack()
		v, k := em.emitExprK(values[0], t)
		addresses[0].assign(k, v, t)
		em.fb.exitStack()
		return
	}

	if len(addresses) == len(values) {
		em.fb.enterStack()
		regs := make([]int8, len(values))
		types := make([]reflect.Type, len(values))
		ks := make([]bool, len(values))
		for i := range values {
			types[i] = em.typ(values[i])
			regs[i] = em.fb.newRegister(types[i].Kind())
			em.emitExprR(values[i], types[i], regs[i])
		}
		for i, addr := range addresses {
			addr.assign(ks[i], regs[i], types[i])
		}
		em.fb.exitStack()
		return
	}

	switch valueExpr := values[0].(type) {

	case *ast.Call:
		regs, retTypes := em.emitCallNode(valueExpr, false, false, runtime.ReturnString)
		for i, addr := range addresses {
			addr.assign(false, regs[i], retTypes[i])
		}

	case *ast.Index: // map index.
		mapType := em.typ(valueExpr.Expr)
		mapp := em.emitExpr(valueExpr.Expr, mapType)
		keyType := em.typ(valueExpr.Index)
		key, kKey := em.emitExprK(valueExpr.Index, keyType)
		valueType := mapType.Elem()
		value := em.fb.newRegister(valueType.Kind())
		okType := addresses[1].addressedType
		okReg := em.fb.newRegister(reflect.Bool)
		pos := valueExpr.Pos()
		em.fb.emitIndex(kKey, mapp, key, value, mapType, pos, false)
		em.fb.emitMove(true, 1, okReg, reflect.Bool)
		em.fb.emitIf(false, 0, runtime.ConditionOK, 0, reflect.Interface, pos)
		em.fb.emitMove(true, 0, okReg, reflect.Bool)
		addresses[0].assign(false, value, valueType)
		addresses[1].assign(false, okReg, okType)

	case *ast.Selector: // key selector.
		exprType := em.typ(valueExpr.Expr)
		expr := em.emitExpr(valueExpr.Expr, exprType)
		key := em.fb.makeStringValue(valueExpr.Ident)
		value := em.fb.newRegister(reflect.Interface)
		okType := addresses[1].addressedType
		okReg := em.fb.newRegister(reflect.Bool)
		pos := valueExpr.Pos()
		em.fb.emitIndex(true, expr, key, value, exprType, pos, false)
		em.fb.emitMove(true, 1, okReg, reflect.Bool)
		em.fb.emitIf(false, 0, runtime.ConditionOK, 0, reflect.Interface, pos)
		em.fb.emitMove(true, 0, okReg, reflect.Bool)
		addresses[0].assign(false, value, emptyInterfaceType)
		addresses[1].assign(false, okReg, okType)

	case *ast.TypeAssertion:
		typ := em.typ(valueExpr.Type)
		expr := em.emitExpr(valueExpr.Expr, emptyInterfaceType)
		okType := addresses[1].addressedType
		ok := em.fb.newRegister(reflect.Bool)
		em.fb.emitMove(true, 1, ok, reflect.Bool)
		result := em.fb.newRegister(typ.Kind())
		em.fb.emitAssert(expr, typ, result)
		em.fb.emitMove(true, 0, ok, reflect.Bool)
		addresses[0].assign(false, result, typ)
		addresses[1].assign(false, ok, okType)

	case *ast.UnaryOperator: // receive from channel.
		chanType := em.typ(valueExpr.Expr)
		valueType := em.typ(valueExpr)
		okType := addresses[1].addressedType
		chann := em.emitExpr(valueExpr.Expr, chanType)
		ok := em.fb.newRegister(reflect.Bool)
		value := em.fb.newRegister(valueType.Kind())
		em.fb.emitReceive(chann, ok, value)
		addresses[0].assign(false, value, valueType)
		addresses[1].assign(false, ok, okType)

	}

}
