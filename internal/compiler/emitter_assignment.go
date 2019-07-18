// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"

	"scriggo/ast"
	"scriggo/vm"
)

// addressType is the type of the element on the left side of an assignment.
type addressType int8

const (
	addressRegister            addressType = iota // Variable assignments.
	addressIndirectDeclaration                    // Indirect variable declaration.
	addressBlank                                  // Blank identifier assignments.
	addressPackageVariable                        // Package variable assignments.
	addressPointerIndirection                     // Pointer indirection assignments.
	addressSliceIndex                             // Slice and array index assignments.
	addressMapIndex                               // Map index assignments.
	addressStructSelector                         // Struct selector assignments.
	addressClosureVariable                        // Closure variable assignments.
)

// address represents an element on the left side of an assignment.
type address struct {
	em         *emitter
	addrType   addressType
	staticType reflect.Type // Type of the addressed element.
	reg1       int8         // Register containing the main expression.
	reg2       int8         // Auxiliary register used in slice, map, array and selector assignments.
}

// newAddress returns a new address. The meaning of reg1 and reg2 depends on
// the address type.
func (em *emitter) newAddress(addrType addressType, staticType reflect.Type, reg1, reg2 int8) address {
	return address{em: em, addrType: addrType, staticType: staticType, reg1: reg1, reg2: reg2}
}

// assign assigns value, with type valueType, to the address. If k is true
// value is a constant otherwise is a register.
func (a address) assign(k bool, value int8, valueType reflect.Type) {
	switch a.addrType {
	case addressClosureVariable:
		a.em.fb.emitSetVar(k, value, int(a.reg1))
	case addressBlank:
		// Nothing to do.
	case addressRegister:
		a.em.changeRegister(k, value, a.reg1, valueType, a.staticType)
	case addressIndirectDeclaration:
		a.em.fb.emitNew(a.staticType, -a.reg1)
		a.em.changeRegister(k, value, a.reg1, valueType, a.staticType)
	case addressPointerIndirection:
		a.em.changeRegister(k, value, -a.reg1, valueType, a.staticType)
	case addressSliceIndex:
		a.em.fb.emitSetSlice(k, a.reg1, value, a.reg2, a.staticType.Elem().Kind())
	case addressMapIndex:
		a.em.fb.emitSetMap(k, a.reg1, value, a.reg2, a.staticType)
	case addressStructSelector:
		a.em.fb.emitSetField(k, a.reg1, a.reg2, value)
	case addressPackageVariable:
		if k {
			tmp := a.em.fb.newRegister(valueType.Kind())
			a.em.fb.emitMove(true, value, tmp, valueType.Kind())
			a.em.fb.emitSetVar(false, tmp, int(a.reg1))
		} else {
			a.em.fb.emitSetVar(false, value, int(a.reg1))
		}
	}
}

// assign assigns values to addresses.
func (em *emitter) assign(addresses []address, values []ast.Expression) {
	// TODO(Gianluca): use mayHaveDependencies.
	if len(addresses) == 1 && len(values) == 1 {
		t := em.ti(values[0]).Type
		v, k := em.emitExprK(values[0], t)
		addresses[0].assign(k, v, t)
	} else if len(addresses) == len(values) {
		regs := make([]int8, len(values))
		types := make([]reflect.Type, len(values))
		ks := make([]bool, len(values))
		for i := range values {
			types[i] = em.ti(values[i]).Type
			regs[i], ks[i], _ = em.quickEmitExpr_old(values[i], types[i])
			if !ks[i] {
				regs[i] = em.fb.newRegister(types[i].Kind())
				em.emitExprR(values[i], types[i], regs[i])
			}
		}
		for i, addr := range addresses {
			addr.assign(ks[i], regs[i], types[i])
		}
	} else {
		switch valueExpr := values[0].(type) {
		case *ast.Call:
			regs, retTypes := em.emitCall(valueExpr)
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
			okType := addresses[1].staticType
			okReg := em.fb.newRegister(reflect.Bool)
			em.fb.emitIndex(kKey, mapp, key, value, mapType)
			em.fb.emitMove(true, 1, okReg, reflect.Bool)
			em.fb.emitIf(false, 0, vm.ConditionOK, 0, reflect.Interface)
			em.fb.emitMove(true, 0, okReg, reflect.Bool)
			addresses[0].assign(false, value, valueType)
			addresses[1].assign(false, okReg, okType)
		case *ast.TypeAssertion:
			typ := em.ti(valueExpr.Type).Type
			expr := em.emitExpr(valueExpr.Expr, emptyInterfaceType)
			okType := addresses[1].staticType
			ok := em.fb.newRegister(reflect.Bool)
			em.fb.emitMove(true, 1, ok, reflect.Bool)
			result := em.fb.newRegister(typ.Kind())
			em.fb.emitAssert(expr, typ, result)
			em.fb.emitMove(true, 0, ok, reflect.Bool)
			addresses[0].assign(false, result, typ)
			addresses[1].assign(false, ok, okType)
		}
	}
}

// emitAssignmentNode emits the instructions for an assignment node.
func (em *emitter) emitAssignmentNode(node *ast.Assignment) {
	// TODO(Gianluca): add support for recursive assignment expressions (eg. a[4][t].field[0]).
	switch node.Type {
	case ast.AssignmentDeclaration:
		addresses := make([]address, len(node.Lhs))
		for i, v := range node.Lhs {
			if isBlankIdentifier(v) {
				addresses[i] = em.newAddress(addressBlank, reflect.Type(nil), 0, 0)
			} else {
				v := v.(*ast.Identifier)
				staticType := em.ti(v).Type
				if em.indirectVars[v] {
					varReg := -em.fb.newRegister(reflect.Interface)
					em.fb.bindVarReg(v.Name, varReg)
					addresses[i] = em.newAddress(addressIndirectDeclaration, staticType, varReg, 0)
				} else {
					varReg := em.fb.newRegister(staticType.Kind())
					em.fb.bindVarReg(v.Name, varReg)
					addresses[i] = em.newAddress(addressRegister, staticType, varReg, 0)
				}
			}
		}
		em.assign(addresses, node.Rhs)
	case ast.AssignmentSimple:
		addresses := make([]address, len(node.Lhs))
		for i, v := range node.Lhs {
			switch v := v.(type) {
			case *ast.Identifier:
				if !isBlankIdentifier(v) {
					staticType := em.ti(v).Type
					if reg, ok := em.closureVarRefs[em.fb.fn][v.Name]; ok {
						// TODO(Gianluca): reg is converted into an
						// int8; should we change address to store
						// int32/64?
						addresses[i] = em.newAddress(addressClosureVariable, staticType, int8(reg), 0)
					} else if index, ok := em.varIndexes[em.pkg][v.Name]; ok {
						// TODO(Gianluca): split index in 2 bytes, assigning first to reg1 and second to reg2.
						addresses[i] = em.newAddress(addressPackageVariable, staticType, int8(index), 0)
					} else if ti := em.ti(v); ti.IsPredefined() {
						index := em.predVarIndex(ti.value.(reflect.Value), ti.PredefPackageName, v.Name)
						addresses[i] = em.newAddress(addressPackageVariable, staticType, int8(index), 0)
					} else {
						reg := em.fb.scopeLookup(v.Name)
						addresses[i] = em.newAddress(addressRegister, staticType, reg, 0)
					}
				} else {
					addresses[i] = em.newAddress(addressBlank, reflect.Type(nil), 0, 0)
				}
			case *ast.Index:
				exprType := em.ti(v.Expr).Type
				expr := em.emitExpr(v.Expr, exprType)
				indexType := em.ti(v.Index).Type
				index := em.emitExpr(v.Index, indexType)
				addrType := addressSliceIndex
				if exprType.Kind() == reflect.Map {
					addrType = addressMapIndex
				}
				addresses[i] = em.newAddress(addrType, exprType, expr, index)
			case *ast.Selector:
				if varIndex, ok := em.varIndexes[em.pkg][v.Expr.(*ast.Identifier).Name+"."+v.Ident]; ok {
					addresses[i] = em.newAddress(addressPackageVariable, em.ti(v).Type, int8(varIndex), 0)
				} else if ti := em.ti(v); ti.IsPredefined() {
					rv := ti.value.(reflect.Value)
					index := em.predVarIndex(rv, ti.PredefPackageName, v.Ident)
					addresses[i] = em.newAddress(addressPackageVariable, em.ti(v).Type, int8(index), 0)
				} else {
					typ := em.ti(v.Expr).Type
					reg := em.emitExpr(v.Expr, typ)
					field, _ := typ.FieldByName(v.Ident)
					index := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
					addresses[i] = em.newAddress(addressStructSelector, field.Type, reg, index)
				}
			case *ast.UnaryOperator:
				if v.Operator() != ast.OperatorMultiplication {
					panic("bug: v.Operator() != ast.OperatorMultiplication") // TODO(Gianluca): remove.
				}
				switch expr := v.Expr.(type) {
				case *ast.Identifier:
					if em.fb.isVariable(expr.Name) {
						reg := em.fb.scopeLookup(expr.Name)
						typ := em.ti(expr).Type
						addresses[i] = em.newAddress(addressPointerIndirection, typ, reg, 0)
					} else {
						panic("TODO(Gianluca): not implemented")
					}
				default:
					panic("TODO(Gianluca): not implemented")
				}
			default:
				panic("TODO(Gianluca): not implemented")
			}
		}
		em.assign(addresses, node.Rhs)
	default:
		var addr address
		var value int8
		var valueType reflect.Type
		switch v := node.Lhs[0].(type) {
		case *ast.Identifier:
			// TODO(Gianluca): support predefined variables in other cases.
			if ti := em.ti(v); ti.IsPredefined() {
				rv := ti.value.(reflect.Value)
				index := em.predVarIndex(rv, ti.PredefPackageName, v.Name)
				addr = em.newAddress(addressPackageVariable, em.ti(v).Type, int8(index), 0)
				staticType := em.ti(v).Type
				valueType = ti.Type
				value = em.emitExpr(v, staticType)
			} else {
				valueType = em.ti(v).Type
				value = em.fb.scopeLookup(v.Name)
				addr = em.newAddress(addressRegister, valueType, value, 0)
			}
		case *ast.Index:
			exprType := em.ti(v.Expr).Type
			expr := em.emitExpr(v.Expr, exprType)
			index := em.emitExpr(v.Index, em.ti(v.Index).Type)
			addrType := addressSliceIndex
			if exprType.Kind() == reflect.Map {
				addrType = addressMapIndex
			}
			addr = em.newAddress(addrType, exprType, expr, index)
			valueType = exprType.Elem()
			value = em.fb.newRegister(valueType.Kind())
			em.fb.emitIndex(false, expr, index, value, exprType)
		default:
			panic("TODO(Gianluca): not implemented")
		}
		switch node.Type {
		case ast.AssignmentIncrement:
			em.fb.emitAdd(true, value, 1, value, valueType.Kind())
		case ast.AssignmentDecrement:
			em.fb.emitSub(true, value, 1, value, valueType.Kind())
		default:
			rightOp := em.emitExpr(node.Rhs[0], em.ti(node.Rhs[0]).Type)
			switch node.Type {
			case ast.AssignmentAddition:
				if valueType.Kind() == reflect.String {
					em.fb.emitConcat(value, rightOp, value)
				} else {
					em.fb.emitAdd(false, value, rightOp, value, valueType.Kind())
				}
			case ast.AssignmentSubtraction:
				em.fb.emitSub(false, value, rightOp, value, valueType.Kind())
			case ast.AssignmentMultiplication:
				em.fb.emitMul(false, value, rightOp, value, valueType.Kind())
			case ast.AssignmentDivision:
				em.fb.emitDiv(false, value, rightOp, value, valueType.Kind())
			case ast.AssignmentModulo:
				em.fb.emitRem(false, value, rightOp, value, valueType.Kind())
			case ast.AssignmentAnd:
				em.fb.emitAnd(false, value, rightOp, value, valueType.Kind())
			case ast.AssignmentOr:
				em.fb.emitOr(false, value, rightOp, value, valueType.Kind())
			case ast.AssignmentXor:
				em.fb.emitXor(false, value, rightOp, value, valueType.Kind())
			case ast.AssignmentAndNot:
				em.fb.emitAndNot(false, value, rightOp, value, valueType.Kind())
			case ast.AssignmentLeftShift:
				em.fb.emitLeftShift(false, value, rightOp, value, valueType.Kind())
			case ast.AssignmentRightShift:
				em.fb.emitRightShift(false, value, rightOp, value, valueType.Kind())
			}
		}
		addr.assign(false, value, valueType)
	}
}
