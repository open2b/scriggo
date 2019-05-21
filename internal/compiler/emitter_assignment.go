// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"

	"scrigo/internal/compiler/ast"
	"scrigo/vm"
)

// AddressType is the type of an element on the left side of and assignment.
type AddressType int8

const (
	AddressRegister            AddressType = iota // Variable assignments.
	AddressIndirectDeclaration                    // Indirect variable declaration.
	AddressesBlank                                // Blank identifier assignments.
	AddressPackageVariable                        // Package variable assignments.
	AddressPointerIndirection                     // Pointer indirection assignments.
	AddressSliceIndex                             // Slice and array index assignments.
	AddressMapIndex                               // Map index assignments.
	AddressStructSelector                         // Struct selector assignments.
	AddressUpVar
)

// Address represents an element on the left side of assignments.
type Address struct {
	c           *Emitter
	Type        AddressType
	ReflectType reflect.Type // Type of the addressed element.
	Reg1        int8         // Register containing the main expression.
	Reg2        int8         // Auxiliary register used in slice, map, array and selector assignments.
}

// NewAddress returns a new address. Meaning of reg1 and reg2 depends on address type.
func (e *Emitter) NewAddress(addrType AddressType, reflectType reflect.Type, reg1, reg2 int8) Address {
	return Address{c: e, Type: addrType, ReflectType: reflectType, Reg1: reg1, Reg2: reg2}
}

// Assign assigns value to a.
func (a Address) Assign(k bool, value int8, valueType reflect.Type) {
	switch a.Type {
	case AddressUpVar:
		a.c.FB.SetVar(k, value, int(a.Reg1))
	case AddressesBlank:
		// Nothing to do.
	case AddressRegister:
		a.c.changeRegister(k, value, a.Reg1, valueType, a.ReflectType)
	case AddressIndirectDeclaration:
		a.c.FB.New(a.ReflectType, -a.Reg1)
		a.c.changeRegister(k, value, a.Reg1, valueType, a.ReflectType)
	case AddressPointerIndirection:
		a.c.changeRegister(k, value, -a.Reg1, valueType, a.ReflectType)
	case AddressSliceIndex:
		a.c.FB.SetSlice(k, a.Reg1, value, a.Reg2, a.ReflectType.Elem().Kind())
	case AddressMapIndex:
		a.c.FB.SetMap(k, a.Reg1, value, a.Reg2)
	case AddressStructSelector:
		panic("TODO(Gianluca): not implemented")
	case AddressPackageVariable:
		if k {
			tmpReg := a.c.FB.NewRegister(valueType.Kind())
			a.c.FB.Move(true, value, tmpReg, valueType.Kind())
			a.c.FB.SetVar(false, tmpReg, int(a.Reg1))
		} else {
			a.c.FB.SetVar(false, value, int(a.Reg1))
		}
	}
}

// assign assigns values to addresses.
func (e *Emitter) assign(addresses []Address, values []ast.Expression) {
	// TODO(Gianluca): use mayHaveDependencies.
	if len(addresses) == len(values) {
		valueRegs := make([]int8, len(values))
		valueTypes := make([]reflect.Type, len(values))
		valueIsK := make([]bool, len(values))
		for i := range values {
			valueTypes[i] = e.TypeInfo[values[i]].Type
			valueRegs[i], valueIsK[i], _ = e.quickEmitExpr(values[i], valueTypes[i])
			if !valueIsK[i] {
				valueRegs[i] = e.FB.NewRegister(valueTypes[i].Kind())
				e.emitExpr(values[i], valueRegs[i], valueTypes[i])
			}
		}
		for i, addr := range addresses {
			addr.Assign(valueIsK[i], valueRegs[i], valueTypes[i])
		}
	} else {
		switch value := values[0].(type) {
		case *ast.Call:
			retRegs, retTypes := e.emitCall(value)
			for i, addr := range addresses {
				addr.Assign(false, retRegs[i], retTypes[i])
			}
		case *ast.Index: // map index.
			mapExpr := value.Expr
			mapType := e.TypeInfo[mapExpr].Type
			mapReg := e.FB.NewRegister(mapType.Kind())
			e.emitExpr(mapExpr, mapReg, mapType)
			keyExpr := value.Index
			keyType := e.TypeInfo[keyExpr].Type
			keyReg, kKeyReg, isRegister := e.quickEmitExpr(keyExpr, keyType)
			if !kKeyReg && !isRegister {
				keyReg = e.FB.NewRegister(keyType.Kind())
				e.emitExpr(keyExpr, keyReg, keyType)
			}
			valueType := mapType.Elem()
			valueReg := e.FB.NewRegister(valueType.Kind())
			okType := addresses[1].ReflectType
			okReg := e.FB.NewRegister(reflect.Bool)
			e.FB.Index(kKeyReg, mapReg, keyReg, valueReg, mapType)
			e.FB.Move(true, 1, okReg, reflect.Bool)
			e.FB.If(false, 0, vm.ConditionOK, 0, reflect.Interface)
			e.FB.Move(true, 0, okReg, reflect.Bool)
			addresses[0].Assign(false, valueReg, valueType)
			addresses[1].Assign(false, okReg, okType)
		}
	}
}

// emitAssignmentNode emits instructions for an assignment node.
func (e *Emitter) emitAssignmentNode(node *ast.Assignment) {
	switch node.Type {
	case ast.AssignmentDeclaration:
		addresses := make([]Address, len(node.Variables))
		for i, v := range node.Variables {
			if isBlankIdentifier(v) {
				addresses[i] = e.NewAddress(AddressesBlank, reflect.Type(nil), 0, 0)
			} else {
				v := v.(*ast.Identifier)
				varType := e.TypeInfo[v].Type
				if e.IndirectVars[v] {
					varReg := -e.FB.NewRegister(reflect.Interface)
					e.FB.BindVarReg(v.Name, varReg)
					addresses[i] = e.NewAddress(AddressIndirectDeclaration, varType, varReg, 0)
				} else {
					varReg := e.FB.NewRegister(varType.Kind())
					e.FB.BindVarReg(v.Name, varReg)
					addresses[i] = e.NewAddress(AddressRegister, varType, varReg, 0)
				}
			}
		}
		e.assign(addresses, node.Values)
	case ast.AssignmentSimple:
		addresses := make([]Address, len(node.Variables))
		for i, v := range node.Variables {
			switch v := v.(type) {
			case *ast.Identifier:
				if !isBlankIdentifier(v) {
					varType := e.TypeInfo[v].Type
					if reg, ok := e.upvarsNames[e.CurrentFunction][v.Name]; ok {
						// TODO(Gianluca): reg is converted into an
						// int8; should we change address to store
						// int32/64?
						addresses[i] = e.NewAddress(AddressUpVar, varType, int8(reg), 0)
					} else if index, ok := e.globalNameIndex[e.currentPackage][v.Name]; ok {
						// TODO(Gianluca): split index in 2 bytes, assigning first to reg1 and second to reg2.
						addresses[i] = e.NewAddress(AddressPackageVariable, varType, int8(index), 0)
					} else {
						reg := e.FB.ScopeLookup(v.Name)
						addresses[i] = e.NewAddress(AddressRegister, varType, reg, 0)
					}
				} else {
					addresses[i] = e.NewAddress(AddressesBlank, reflect.Type(nil), 0, 0)
				}
			case *ast.Index:
				exprType := e.TypeInfo[v.Expr].Type
				expr, _, isRegister := e.quickEmitExpr(v.Expr, exprType)
				if !isRegister {
					expr = e.FB.NewRegister(exprType.Kind())
					e.emitExpr(v.Expr, expr, exprType)
				}
				indexType := e.TypeInfo[v.Index].Type
				index, _, isRegister := e.quickEmitExpr(v.Index, indexType)
				if !isRegister {
					index = e.FB.NewRegister(indexType.Kind())
					e.emitExpr(v.Index, index, indexType)
				}
				addrType := AddressSliceIndex
				if exprType.Kind() == reflect.Map {
					addrType = AddressMapIndex
				}
				addresses[i] = e.NewAddress(addrType, exprType, expr, index)
			case *ast.Selector:
				if varIndex, ok := e.globalNameIndex[e.currentPackage][v.Expr.(*ast.Identifier).Name+"."+v.Ident]; ok {
					addresses[i] = e.NewAddress(AddressPackageVariable, e.TypeInfo[v].Type, int8(varIndex), 0)
				} else {
					panic("TODO(Gianluca): not implemented")
				}
			case *ast.UnaryOperator:
				if v.Operator() != ast.OperatorMultiplication {
					panic("bug: v.Operator() != ast.OperatorMultiplication") // TODO(Gianluca): remove.
				}
				switch expr := v.Expr.(type) {
				case *ast.Identifier:
					if e.FB.IsVariable(expr.Name) {
						varReg := e.FB.ScopeLookup(expr.Name)
						exprType := e.TypeInfo[expr].Type
						addresses[i] = e.NewAddress(AddressPointerIndirection, exprType, varReg, 0)
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
		var address Address
		var valueReg int8
		var valueType reflect.Type
		switch v := node.Variables[0].(type) {
		case *ast.Identifier:
			varType := e.TypeInfo[v].Type
			reg := e.FB.ScopeLookup(v.Name)
			address = e.NewAddress(AddressRegister, varType, reg, 0)
			valueReg = reg
			valueType = varType
		case *ast.Index:
			exprType := e.TypeInfo[v.Expr].Type
			expr, _, isRegister := e.quickEmitExpr(v.Expr, exprType)
			if !isRegister {
				expr = e.FB.NewRegister(exprType.Kind())
				e.emitExpr(v.Expr, expr, exprType)
			}
			indexType := e.TypeInfo[v.Index].Type
			index, _, isRegister := e.quickEmitExpr(v.Index, indexType)
			if !isRegister {
				index = e.FB.NewRegister(indexType.Kind())
				e.emitExpr(v.Index, index, indexType)
			}
			addrType := AddressSliceIndex
			if exprType.Kind() == reflect.Map {
				addrType = AddressMapIndex
			}
			address = e.NewAddress(addrType, exprType, expr, index)
			valueType = exprType.Elem()
			valueReg = e.FB.NewRegister(valueType.Kind())
			e.FB.Index(false, expr, index, valueReg, exprType)
		default:
			panic("TODO(Gianluca): not implemented")
		}
		switch node.Type {
		case ast.AssignmentIncrement:
			e.FB.Add(true, valueReg, 1, valueReg, valueType.Kind())
		case ast.AssignmentDecrement:
			e.FB.Sub(true, valueReg, 1, valueReg, valueType.Kind())
		default:
			rightOpType := e.TypeInfo[node.Values[0]].Type
			rightOp := e.FB.NewRegister(rightOpType.Kind())
			e.emitExpr(node.Values[0], rightOp, rightOpType)
			switch node.Type {
			case ast.AssignmentAddition:
				e.FB.Add(false, valueReg, rightOp, valueReg, valueType.Kind())
			case ast.AssignmentSubtraction:
				e.FB.Sub(false, valueReg, rightOp, valueReg, valueType.Kind())
			case ast.AssignmentMultiplication:
				e.FB.Mul(false, valueReg, rightOp, valueReg, valueType.Kind())
			case ast.AssignmentDivision:
				e.FB.Div(false, valueReg, rightOp, valueReg, valueType.Kind())
			case ast.AssignmentModulo:
				e.FB.Rem(false, valueReg, rightOp, valueReg, valueType.Kind())
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
		address.Assign(false, valueReg, valueType)
	}
}
