// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"

	"scriggo/internal/compiler/ast"
	"scriggo/vm"
)

// addressType is the type of an element on the left side of and assignment.
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
	addressUpVar
)

// address represents an element on the left side of an assignment.
type address struct {
	c          *emitter
	addrType   addressType
	staticType reflect.Type // Type of the addressed element.
	reg1       int8         // Register containing the main expression.
	reg2       int8         // Auxiliary register used in slice, map, array and selector assignments.
}

// newAddress returns a new address. The meaning of reg1 and reg2 depends on the address type.
func (e *emitter) newAddress(addrType addressType, staticType reflect.Type, reg1, reg2 int8) address {
	return address{c: e, addrType: addrType, staticType: staticType, reg1: reg1, reg2: reg2}
}

// assign assigns value, with type valueType, to the address. If k is true
// value is a constant otherwise is a register.
func (a address) assign(k bool, value int8, valueType reflect.Type) {
	switch a.addrType {
	case addressUpVar:
		a.c.fb.SetVar(k, value, int(a.reg1))
	case addressBlank:
		// Nothing to do.
	case addressRegister:
		a.c.changeRegister(k, value, a.reg1, valueType, a.staticType)
	case addressIndirectDeclaration:
		a.c.fb.New(a.staticType, -a.reg1)
		a.c.changeRegister(k, value, a.reg1, valueType, a.staticType)
	case addressPointerIndirection:
		a.c.changeRegister(k, value, -a.reg1, valueType, a.staticType)
	case addressSliceIndex:
		a.c.fb.SetSlice(k, a.reg1, value, a.reg2, a.staticType.Elem().Kind())
	case addressMapIndex:
		a.c.fb.SetMap(k, a.reg1, value, a.reg2, a.staticType)
	case addressStructSelector:
		panic("TODO(Gianluca): not implemented")
	case addressPackageVariable:
		if k {
			tmpReg := a.c.fb.NewRegister(valueType.Kind())
			a.c.fb.Move(true, value, tmpReg, valueType.Kind())
			a.c.fb.SetVar(false, tmpReg, int(a.reg1))
		} else {
			a.c.fb.SetVar(false, value, int(a.reg1))
		}
	}
}

// assign assigns values to addresses.
func (e *emitter) assign(addresses []address, values []ast.Expression) {
	// TODO(Gianluca): use mayHaveDependencies.
	if len(addresses) == 1 && len(values) == 1 {
		addr := addresses[0]
		valueType := e.typeInfos[values[0]].Type
		value, k, ok := e.quickEmitExpr(values[0], valueType)
		if !ok {
			value = e.fb.NewRegister(valueType.Kind())
			e.emitExpr(values[0], value, valueType)
		}
		addr.assign(k, value, valueType)
	} else if len(addresses) == len(values) {
		valueRegs := make([]int8, len(values))
		valueTypes := make([]reflect.Type, len(values))
		valueIsK := make([]bool, len(values))
		for i := range values {
			valueTypes[i] = e.typeInfos[values[i]].Type
			valueRegs[i], valueIsK[i], _ = e.quickEmitExpr(values[i], valueTypes[i])
			if !valueIsK[i] {
				valueRegs[i] = e.fb.NewRegister(valueTypes[i].Kind())
				e.emitExpr(values[i], valueRegs[i], valueTypes[i])
			}
		}
		for i, addr := range addresses {
			addr.assign(valueIsK[i], valueRegs[i], valueTypes[i])
		}
	} else {
		switch value := values[0].(type) {
		case *ast.Call:
			retRegs, retTypes := e.emitCall(value)
			for i, addr := range addresses {
				addr.assign(false, retRegs[i], retTypes[i])
			}
		case *ast.Index: // map index.
			mapExpr := value.Expr
			mapType := e.typeInfos[mapExpr].Type
			mapReg := e.fb.NewRegister(mapType.Kind())
			e.emitExpr(mapExpr, mapReg, mapType)
			keyExpr := value.Index
			keyType := e.typeInfos[keyExpr].Type
			keyReg, kKeyReg, ok := e.quickEmitExpr(keyExpr, keyType)
			if !ok {
				keyReg = e.fb.NewRegister(keyType.Kind())
				e.emitExpr(keyExpr, keyReg, keyType)
			}
			valueType := mapType.Elem()
			valueReg := e.fb.NewRegister(valueType.Kind())
			okType := addresses[1].staticType
			okReg := e.fb.NewRegister(reflect.Bool)
			e.fb.Index(kKeyReg, mapReg, keyReg, valueReg, mapType)
			e.fb.Move(true, 1, okReg, reflect.Bool)
			e.fb.If(false, 0, vm.ConditionOK, 0, reflect.Interface)
			e.fb.Move(true, 0, okReg, reflect.Bool)
			addresses[0].assign(false, valueReg, valueType)
			addresses[1].assign(false, okReg, okType)
		}
	}
}

// emitAssignmentNode emits instructions for an assignment node.
func (e *emitter) emitAssignmentNode(node *ast.Assignment) {
	switch node.Type {
	case ast.AssignmentDeclaration:
		addresses := make([]address, len(node.Variables))
		for i, v := range node.Variables {
			if isBlankIdentifier(v) {
				addresses[i] = e.newAddress(addressBlank, reflect.Type(nil), 0, 0)
			} else {
				v := v.(*ast.Identifier)
				staticType := e.typeInfos[v].Type
				if e.indirectVars[v] {
					varReg := -e.fb.NewRegister(reflect.Interface)
					e.fb.BindVarReg(v.Name, varReg)
					addresses[i] = e.newAddress(addressIndirectDeclaration, staticType, varReg, 0)
				} else {
					varReg := e.fb.NewRegister(staticType.Kind())
					e.fb.BindVarReg(v.Name, varReg)
					addresses[i] = e.newAddress(addressRegister, staticType, varReg, 0)
				}
			}
		}
		e.assign(addresses, node.Values)
	case ast.AssignmentSimple:
		addresses := make([]address, len(node.Variables))
		for i, v := range node.Variables {
			switch v := v.(type) {
			case *ast.Identifier:
				if !isBlankIdentifier(v) {
					staticType := e.typeInfos[v].Type
					if reg, ok := e.upvarsNames[e.fb.fn][v.Name]; ok {
						// TODO(Gianluca): reg is converted into an
						// int8; should we change address to store
						// int32/64?
						addresses[i] = e.newAddress(addressUpVar, staticType, int8(reg), 0)
					} else if index, ok := e.pkgVariables[e.pkg][v.Name]; ok {
						// TODO(Gianluca): split index in 2 bytes, assigning first to reg1 and second to reg2.
						addresses[i] = e.newAddress(addressPackageVariable, staticType, int8(index), 0)
					} else if ti := e.typeInfos[v]; ti.IsPredefined() {
						index := e.predefVarIndex(ti.Value.(reflect.Value), ti.PredefPackageName, v.Name)
						addresses[i] = e.newAddress(addressPackageVariable, staticType, int8(index), 0)
					} else {
						reg := e.fb.ScopeLookup(v.Name)
						addresses[i] = e.newAddress(addressRegister, staticType, reg, 0)
					}
				} else {
					addresses[i] = e.newAddress(addressBlank, reflect.Type(nil), 0, 0)
				}
			case *ast.Index:
				exprType := e.typeInfos[v.Expr].Type
				expr, k, ok := e.quickEmitExpr(v.Expr, exprType)
				if !ok || k {
					expr = e.fb.NewRegister(exprType.Kind())
					e.emitExpr(v.Expr, expr, exprType)
				}
				indexType := e.typeInfos[v.Index].Type
				index, k, ok := e.quickEmitExpr(v.Index, indexType)
				if !ok || k {
					index = e.fb.NewRegister(indexType.Kind())
					e.emitExpr(v.Index, index, indexType)
				}
				addrType := addressSliceIndex
				if exprType.Kind() == reflect.Map {
					addrType = addressMapIndex
				}
				addresses[i] = e.newAddress(addrType, exprType, expr, index)
			case *ast.Selector:
				if varIndex, ok := e.pkgVariables[e.pkg][v.Expr.(*ast.Identifier).Name+"."+v.Ident]; ok {
					addresses[i] = e.newAddress(addressPackageVariable, e.typeInfos[v].Type, int8(varIndex), 0)
				}
				if ti := e.typeInfos[v]; ti.IsPredefined() {
					varRv := ti.Value.(reflect.Value)
					index := e.predefVarIndex(varRv, ti.PredefPackageName, v.Ident)
					addresses[i] = e.newAddress(addressPackageVariable, e.typeInfos[v].Type, int8(index), 0)
				}
			case *ast.UnaryOperator:
				if v.Operator() != ast.OperatorMultiplication {
					panic("bug: v.Operator() != ast.OperatorMultiplication") // TODO(Gianluca): remove.
				}
				switch expr := v.Expr.(type) {
				case *ast.Identifier:
					if e.fb.IsVariable(expr.Name) {
						varReg := e.fb.ScopeLookup(expr.Name)
						exprType := e.typeInfos[expr].Type
						addresses[i] = e.newAddress(addressPointerIndirection, exprType, varReg, 0)
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
		e.assign(addresses, node.Values)
	default:
		var addr address
		var valueReg int8
		var valueType reflect.Type
		switch v := node.Variables[0].(type) {
		case *ast.Identifier:
			staticType := e.typeInfos[v].Type
			// TODO(Gianluca): support predefined variables in other cases.
			if ti := e.typeInfos[v]; ti.IsPredefined() {
				varRv := ti.Value.(reflect.Value)
				index := e.predefVarIndex(varRv, ti.PredefPackageName, v.Name)
				addr = e.newAddress(addressPackageVariable, e.typeInfos[v].Type, int8(index), 0)
				valueType = ti.Type
				valueReg = e.fb.NewRegister(staticType.Kind())
				e.emitExpr(v, valueReg, valueType)
			} else {
				reg := e.fb.ScopeLookup(v.Name)
				addr = e.newAddress(addressRegister, staticType, reg, 0)
				valueReg = reg
				valueType = staticType
			}
		case *ast.Index:
			exprType := e.typeInfos[v.Expr].Type
			expr, k, ok := e.quickEmitExpr(v.Expr, exprType)
			if !ok || k {
				expr = e.fb.NewRegister(exprType.Kind())
				e.emitExpr(v.Expr, expr, exprType)
			}
			indexType := e.typeInfos[v.Index].Type
			index, k, ok := e.quickEmitExpr(v.Index, indexType)
			if !ok || k {
				index = e.fb.NewRegister(indexType.Kind())
				e.emitExpr(v.Index, index, indexType)
			}
			addrType := addressSliceIndex
			if exprType.Kind() == reflect.Map {
				addrType = addressMapIndex
			}
			addr = e.newAddress(addrType, exprType, expr, index)
			valueType = exprType.Elem()
			valueReg = e.fb.NewRegister(valueType.Kind())
			e.fb.Index(false, expr, index, valueReg, exprType)
		default:
			panic("TODO(Gianluca): not implemented")
		}
		switch node.Type {
		case ast.AssignmentIncrement:
			e.fb.Add(true, valueReg, 1, valueReg, valueType.Kind())
		case ast.AssignmentDecrement:
			e.fb.Sub(true, valueReg, 1, valueReg, valueType.Kind())
		default:
			rightOpType := e.typeInfos[node.Values[0]].Type
			rightOp := e.fb.NewRegister(rightOpType.Kind())
			e.emitExpr(node.Values[0], rightOp, rightOpType)
			switch node.Type {
			case ast.AssignmentAddition:
				if valueType.Kind() == reflect.String {
					e.fb.Concat(valueReg, rightOp, valueReg)
				} else {
					e.fb.Add(false, valueReg, rightOp, valueReg, valueType.Kind())
				}
			case ast.AssignmentSubtraction:
				e.fb.Sub(false, valueReg, rightOp, valueReg, valueType.Kind())
			case ast.AssignmentMultiplication:
				e.fb.Mul(false, valueReg, rightOp, valueReg, valueType.Kind())
			case ast.AssignmentDivision:
				e.fb.Div(false, valueReg, rightOp, valueReg, valueType.Kind())
			case ast.AssignmentModulo:
				e.fb.Rem(false, valueReg, rightOp, valueReg, valueType.Kind())
			case ast.AssignmentLeftShift:
				panic("TODO(Gianluca): not implemented")
			case ast.AssignmentRightShift:
				panic("TODO(Gianluca): not implemented")
			case ast.AssignmentAnd:
				panic("TODO(Gianluca): not implemented")
			case ast.AssignmentOr:
				panic("TODO(Gianluca): not implemented")
			case ast.AssignmentXor:
				panic("TODO(Gianluca): not implemented")
			case ast.AssignmentAndNot:
				panic("TODO(Gianluca): not implemented")
			}
		}
		addr.assign(false, valueReg, valueType)
	}
}
