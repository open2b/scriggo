// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"

	"scriggo/ast"
)

// checkAssignment type checks an assignment node (Var, Const or Assignment)
// and fills the scope, if necessary.
func (tc *typechecker) checkAssignment(node ast.Node) {

	var vars, values []ast.Expression
	var typ *TypeInfo
	var isDecl, isConst, isVar bool

	if tc.lastConstPosition != node.Pos() {
		tc.iota = -1
	}

	switch n := node.(type) {

	case *ast.Var:

		values = n.Rhs
		isDecl = true
		isVar = true
		if n.Type != nil {
			typ = tc.checkType(n.Type, noEllipses)
		}

		if len(values) == 0 {
			for i := range n.Lhs {
				zero := &TypeInfo{Type: typ.Type}
				newVar := tc.assignSingle(node, n.Lhs[i], nil, zero, typ, true, false)
				if newVar == "" && !isBlankIdentifier(n.Lhs[i]) {
					panic(tc.errorf(node, "%s redeclared in this block", n.Lhs[i]))
				}
			}
			// Replaces the type node with a value holding a reflect.Type.
			k := typ.Type.Kind()
			n.Rhs = make([]ast.Expression, len(n.Lhs))
			switch {
			case isNumeric(k):
				for i := range n.Lhs {
					n.Rhs[i] = ast.NewPlaceholder()
					tc.TypeInfo[n.Rhs[i]] = &TypeInfo{Type: typ.Type, Constant: int64Const(0), Properties: PropertyUntyped}
					tc.TypeInfo[n.Rhs[i]].setValue(typ.Type)
				}
			case k == reflect.String:
				for i := range n.Lhs {
					n.Rhs[i] = ast.NewPlaceholder()
					tc.TypeInfo[n.Rhs[i]] = &TypeInfo{Type: typ.Type, Constant: stringConst(""), Properties: PropertyUntyped}
					tc.TypeInfo[n.Rhs[i]].setValue(typ.Type)
				}
			case k == reflect.Bool:
				for i := range n.Lhs {
					n.Rhs[i] = ast.NewPlaceholder()
					tc.TypeInfo[n.Rhs[i]] = &TypeInfo{Type: typ.Type, Constant: boolConst(false), Properties: PropertyUntyped}
					tc.TypeInfo[n.Rhs[i]].setValue(typ.Type)
				}
			case k == reflect.Interface:
				for i := range n.Lhs {
					n.Rhs[i] = ast.NewPlaceholder()
					tc.TypeInfo[n.Rhs[i]] = &TypeInfo{Type: typ.Type}
					tc.TypeInfo[n.Rhs[i]].setValue(typ.Type)
				}
			default:
				for i := range n.Lhs {
					n.Rhs[i] = ast.NewPlaceholder()
					tc.TypeInfo[n.Rhs[i]] = &TypeInfo{Type: typ.Type, value: reflect.Zero(typ.Type).Interface()}
					tc.TypeInfo[n.Rhs[i]].setValue(typ.Type)
				}
			}
			return
		}

		if len(n.Lhs) == 1 && len(values) == 1 {
			newVar := tc.assignSingle(node, n.Lhs[0], values[0], nil, typ, true, false)
			if !isBlankIdentifier(n.Lhs[0]) && newVar == "" {
				panic(tc.errorf(node, "%s redeclared in this block", n.Lhs[0]))
			}
			return
		}

		vars = make([]ast.Expression, len(n.Lhs))
		for i, ident := range n.Lhs {
			vars[i] = ident
		}

	case *ast.Const:

		values = n.Rhs
		isDecl = true
		isConst = true
		if n.Type != nil {
			typ = tc.checkType(n.Type, noEllipses)
		}
		tc.lastConstPosition = node.Pos()

		// TODO(Gianluca): optimization has been disabled. Re-enable or remove?
		// if len(n.Identifiers) == 1 && len(values) == 1 {
		// 	newConst := tc.assignSingle(node, n.Identifiers[0], values[0], nil, typ, true, true)
		// 	if newConst == "" && !isBlankIdentifier(n.Identifiers[0]) {
		// 		panic(tc.errorf(node, "%s redeclared in this block", n.Identifiers[0]))
		// 	}
		// 	return
		// }

		if len(n.Lhs) > len(values) {
			panic(tc.errorf(node, "missing value in const declaration"))
		}
		if len(n.Lhs) < len(values) {
			panic(tc.errorf(node, "extra expression in const declaration"))
		}

		vars = make([]ast.Expression, len(n.Lhs))
		for i, ident := range n.Lhs {
			vars[i] = ident
		}

	case *ast.Assignment:

		switch n.Type {
		case ast.AssignmentIncrement, ast.AssignmentDecrement:
			v := n.Variables[0]
			if isBlankIdentifier(v) {
				panic(tc.errorf(v, "cannot use _ as value"))
			}
			exprTi := tc.checkExpression(v)
			if !isNumeric(exprTi.Type.Kind()) {
				panic(tc.errorf(node, "invalid operation: %v (non-numeric type %s)", node, exprTi))
			}
			if !exprTi.Addressable() {
				panic(tc.errorf(node, "cannot assign to %s", node))
			}
			return
		case ast.AssignmentAddition, ast.AssignmentSubtraction, ast.AssignmentMultiplication,
			ast.AssignmentDivision, ast.AssignmentModulo:
			var opType ast.OperatorType
			switch n.Type {
			case ast.AssignmentAddition:
				opType = ast.OperatorAddition
			case ast.AssignmentSubtraction:
				opType = ast.OperatorSubtraction
			case ast.AssignmentMultiplication:
				opType = ast.OperatorMultiplication
			case ast.AssignmentDivision:
				opType = ast.OperatorDivision
			case ast.AssignmentModulo:
				opType = ast.OperatorModulo
			}
			if isBlankIdentifier(n.Variables[0]) {
				panic(tc.errorf(n.Variables[0], "cannot use _ as value"))
			}
			// TODO (Gianluca): check if operation can be done before
			// calling binaryOp.
			_, err := tc.binaryOp(n.Variables[0], opType, n.Values[0])
			if err != nil {
				panic(tc.errorf(n, "invalid operation: %v (%s)", n, err))
			}
			tc.assignSingle(node, n.Variables[0], n.Values[0], nil, nil, false, false)
			return
		}

		vars = n.Variables
		values = n.Values
		isDecl = n.Type == ast.AssignmentDeclaration

		if len(vars) == 1 && len(values) == 1 {
			newVar := tc.assignSingle(node, vars[0], values[0], nil, typ, isDecl, false)
			if newVar == "" && isDecl {
				panic(tc.errorf(node, "no new variables on left side of :="))
			}
			return
		}

	default:

		panic(fmt.Errorf("bug: unexpected node %T", node))

	}

	if len(vars) >= 2 && len(values) == 1 {
		call, ok := values[0].(*ast.Call)
		if ok {
			tis, isBuiltin, _ := tc.checkCallExpression(call, false)
			if len(vars) != len(tis) {
				if isBuiltin {
					panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(vars), len(values)))
				}
				panic(tc.errorf(node, "assignment mismatch: %d variables but %v returns %d values", len(vars), call, len(values)))
			}
			values = nil
			for _, ti := range tis {
				newCall := ast.NewCall(call.Pos(), call.Func, call.Args, false)
				tc.TypeInfo[newCall] = ti
				values = append(values, newCall)
			}
		}
	}

	if len(vars) == 2 && len(values) == 1 {
		switch v := values[0].(type) {

		case *ast.TypeAssertion:

			v1 := ast.NewTypeAssertion(v.Pos(), v.Expr, v.Type)
			v2 := ast.NewTypeAssertion(v.Pos(), v.Expr, v.Type)
			ti := tc.checkExpression(values[0])
			tc.TypeInfo[v1] = &TypeInfo{Type: ti.Type}
			tc.TypeInfo[v2] = untypedBoolTypeInfo
			values = []ast.Expression{v1, v2}

		case *ast.Index:

			v1 := ast.NewIndex(v.Pos(), v.Expr, v.Index)
			v2 := ast.NewIndex(v.Pos(), v.Expr, v.Index)
			ti := tc.checkExpression(values[0])
			tc.TypeInfo[v1] = &TypeInfo{Type: ti.Type}
			tc.TypeInfo[v2] = untypedBoolTypeInfo
			values = []ast.Expression{v1, v2}

		case *ast.UnaryOperator:

			if v.Op == ast.OperatorReceive {
				v1 := ast.NewUnaryOperator(v.Pos(), ast.OperatorReceive, v.Expr)
				v2 := ast.NewUnaryOperator(v.Pos(), ast.OperatorReceive, v.Expr)
				ti := tc.checkExpression(values[0])
				tc.TypeInfo[v1] = &TypeInfo{Type: ti.Type}
				tc.TypeInfo[v2] = untypedBoolTypeInfo
				values = []ast.Expression{v1, v2}
			}

		}
	}

	if len(vars) != len(values) {
		panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(vars), len(values)))
	}

	newVars := []string{}
	tmpScope := TypeCheckerScope{}
	for i := range vars {
		if isConst {
			tc.iota++
		}
		var newVar string
		if valueTi := tc.TypeInfo[values[i]]; valueTi == nil {
			newVar = tc.assignSingle(node, vars[i], values[i], nil, typ, isDecl, isConst)
		} else {
			newVar = tc.assignSingle(node, vars[i], nil, valueTi, typ, isDecl, isConst)
		}
		if isDecl {
			ti, _ := tc.lookupScopes(newVar, true)
			tmpScope[newVar] = scopeElement{t: ti}
			if len(tc.Scopes) > 0 {
				delete(tc.Scopes[len(tc.Scopes)-1], newVar)
			} else {
				delete(tc.filePackageBlock, newVar)
			}
		}
		if (isVar || isConst) && newVar == "" && !isBlankIdentifier(vars[i]) {
			panic(tc.errorf(node, "%s redeclared in this block", vars[i]))
		}
		for _, v := range newVars {
			if newVar == v {
				panic(tc.errorf(node, "%s repeated on left side of :=", vars[i]))
			}
		}
		if newVar != "" {
			newVars = append(newVars, newVar)
		}
	}
	if len(newVars) == 0 && isDecl && !isVar && !isConst {
		panic(tc.errorf(node, "no new variables on left side of :="))
	}
	for d, ti := range tmpScope {
		tc.assignScope(d, ti.t, nil)
	}
	return

}

// assignSingle assigns value to variable (or valueTi to variable if value is
// nil). typ is the type specified in the declaration, if any. If assignment
// is a declaration and the scope has been updated, returns the identifier of
// the new scope element; otherwise returns an empty string.
func (tc *typechecker) assignSingle(node ast.Node, variable, value ast.Expression, valueTi *TypeInfo, typ *TypeInfo, isDeclaration, isConst bool) string {

	if valueTi == nil {
		valueTi = tc.checkExpression(value)
	}

	if isConst && !valueTi.IsConstant() {
		panic(tc.errorf(node, "const initializer %s is not a constant", value))
	}

	// TODO (Gianluca): not clear.
	if typ != nil {
		if err := isAssignableTo(valueTi, value, typ.Type); err != nil {
			if value == nil {
				panic(tc.errorf(node, "cannot assign %s to %s (type %s) in multiple assignment", valueTi.ShortString(), variable, typ))
			}
			panic(tc.errorf(node, "%s in assignment", err))
		}
		valueTi.setValue(typ.Type)
	} else {
		valueTi.setValue(nil)
	}

	switch v := variable.(type) {

	case *ast.Identifier:

		if v.Name == "_" {
			return ""
		}

		if isDeclaration {
			newValueTi := &TypeInfo{}
			if typ == nil {
				if valueTi.Nil() {
					panic(tc.errorf(node, "use of untyped nil"))
				}
				//  «[if no types are presents], each variable is given the type
				//  of the corresponding initialization value in the
				//  assignment.»
				//
				// «If that value is an untyped constant, it is first
				// implicitly converted to its default type.»
				newValueTi.Type = valueTi.Type
			} else {
				newValueTi.Type = typ.Type
			}
			tc.TypeInfo[v] = newValueTi
			if _, alreadyInCurrentScope := tc.lookupScopes(v.Name, true); alreadyInCurrentScope {
				return ""
			}
			if isConst {
				newValueTi.Constant = valueTi.Constant
				if valueTi.Untyped() {
					newValueTi.Properties = PropertyUntyped
				}
				tc.assignScope(v.Name, newValueTi, nil)
				return v.Name
			}
			newValueTi.Properties |= PropertyAddressable
			tc.assignScope(v.Name, newValueTi, v)
			if !tc.opts.AllowNotUsed {
				tc.unusedVars = append(tc.unusedVars, &scopeVariable{
					ident:      v.Name,
					scopeLevel: len(tc.Scopes) - 1,
					node:       node,
				})
			}
			return v.Name
		}

		variableTi := tc.checkIdentifier(v, false)
		if !variableTi.Addressable() {
			panic(tc.errorf(variable, "cannot assign to %v", variable))
		}
		if err := isAssignableTo(valueTi, value, variableTi.Type); err != nil {
			panic(tc.errorf(value, "%s in assignment", err))
		}
		valueTi.setValue(variableTi.Type)
		tc.TypeInfo[v] = variableTi

	case *ast.Index:

		if isDeclaration {
			panic(tc.errorf(node, "non name %s on left side of :=", variable))
		}
		variableTi := tc.checkExpression(variable)
		switch variableTi.Type.Kind() {
		case reflect.Slice, reflect.Map:
			// Always addressable when used in indexing operation.
		case reflect.Array:
			if !variableTi.Addressable() {
				panic(tc.errorf(node, "cannot assign to %v", variable))
			}
		}
		if err := isAssignableTo(valueTi, value, variableTi.Type); err != nil {
			panic(tc.errorf(node, "%s in assignment", err))
		}
		valueTi.setValue(variableTi.Type)
		return ""

	case *ast.Selector:

		if isDeclaration {
			panic(tc.errorf(node, "non name %s on left side of :=", variable))
		}
		variableTi := tc.checkExpression(variable)
		// TODO(Gianluca): investigate: this always fails.
		// if !variableTi.Addressable() {
		// 	panic(tc.errorf(node, "cannot assign to %v", variable))
		// }
		if err := isAssignableTo(valueTi, value, variableTi.Type); err != nil {
			panic(tc.errorf(node, "%s in assignment", err))
		}
		valueTi.setValue(variableTi.Type)
		return ""

	case *ast.UnaryOperator:

		if isDeclaration {
			panic(tc.errorf(node, "non name %s on left side of :=", variable))
		}
		if v.Operator() == ast.OperatorMultiplication { // pointer indirection.
			variableTi := tc.checkExpression(variable)
			if err := isAssignableTo(valueTi, value, variableTi.Type); err != nil {
				panic(tc.errorf(node, "%s in assignment", err))
			}
			valueTi.setValue(variableTi.Type)
			return ""
		}
		panic(tc.errorf(node, "cannot assign to %v", variable))

	case *ast.Call:

		if isDeclaration {
			panic(tc.errorf(node, "non name %s on left side of :=", variable))
		}
		tis, _, _ := tc.checkCallExpression(v, false)
		switch len(tis) {
		case 0:
			panic(tc.errorf(node, "%s used as value", variable))
		case 1:
			if !tis[0].Addressable() {
				panic(tc.errorf(node, "cannot assign to %v", variable))
			}
		default:
			panic(tc.errorf(node, "multiple-value %s in single-value context", variable))
		}

	default:

		panic("bug/not implemented")
	}

	return ""
}
