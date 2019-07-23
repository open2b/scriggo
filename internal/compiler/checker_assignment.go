// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"

	"scriggo/ast"
)

// checkAssignment type checks an assignment node (Var, Const or Assignment)
// and fills the scope, if necessary.
func (tc *typechecker) checkAssignment(node ast.Node) {

	lhs := []ast.Expression{}
	rhs := []ast.Expression{}
	declType := (*TypeInfo)(nil)
	isVarDecl := false
	isConstDecl := false
	isAssignmentNode := false

	if tc.lastConstPosition != node.Pos() {
		tc.iota = -1
	}

	switch n := node.(type) {

	case *ast.Var:

		rhs = n.Rhs
		isVarDecl = true
		if n.Type != nil {
			declType = tc.checkType(n.Type)
		}

		if len(rhs) == 0 {
			for i := range n.Lhs {
				zero := &TypeInfo{Type: declType.Type}
				newVar := tc.assign(node, n.Lhs[i], nil, zero, declType, true, false)
				if newVar == "" && !isBlankIdentifier(n.Lhs[i]) {
					panic(tc.errorf(node, "%s redeclared in this block", n.Lhs[i]))
				}
			}
			// Replaces the type node with a value holding a reflect.Type.
			k := declType.Type.Kind()
			n.Rhs = make([]ast.Expression, len(n.Lhs))
			switch {
			case isNumeric(k):
				for i := range n.Lhs {
					n.Rhs[i] = ast.NewPlaceholder()
					tc.typeInfos[n.Rhs[i]] = &TypeInfo{Type: declType.Type, Constant: int64Const(0), Properties: PropertyUntyped}
					tc.typeInfos[n.Rhs[i]].setValue(declType.Type)
				}
			case k == reflect.String:
				for i := range n.Lhs {
					n.Rhs[i] = ast.NewPlaceholder()
					tc.typeInfos[n.Rhs[i]] = &TypeInfo{Type: declType.Type, Constant: stringConst(""), Properties: PropertyUntyped}
					tc.typeInfos[n.Rhs[i]].setValue(declType.Type)
				}
			case k == reflect.Bool:
				for i := range n.Lhs {
					n.Rhs[i] = ast.NewPlaceholder()
					tc.typeInfos[n.Rhs[i]] = &TypeInfo{Type: declType.Type, Constant: boolConst(false), Properties: PropertyUntyped}
					tc.typeInfos[n.Rhs[i]].setValue(declType.Type)
				}
			case k == reflect.Interface:
				for i := range n.Lhs {
					n.Rhs[i] = ast.NewPlaceholder()
					tc.typeInfos[n.Rhs[i]] = &TypeInfo{Type: declType.Type}
					tc.typeInfos[n.Rhs[i]].setValue(declType.Type)
				}
			default:
				for i := range n.Lhs {
					n.Rhs[i] = ast.NewPlaceholder()
					tc.typeInfos[n.Rhs[i]] = &TypeInfo{Type: declType.Type, value: reflect.Zero(declType.Type).Interface()}
					tc.typeInfos[n.Rhs[i]].setValue(declType.Type)
				}
			}
			return
		}

		lhs = make([]ast.Expression, len(n.Lhs))
		for i, ident := range n.Lhs {
			lhs[i] = ident
		}

	case *ast.Const:

		rhs = n.Rhs
		isConstDecl = true
		if n.Type != nil {
			declType = tc.checkType(n.Type)
		}
		tc.lastConstPosition = node.Pos()

		if len(n.Lhs) > len(rhs) {
			panic(tc.errorf(node, "missing value in const declaration"))
		}
		if len(n.Lhs) < len(rhs) {
			panic(tc.errorf(node, "extra expression in const declaration"))
		}

		lhs = make([]ast.Expression, len(n.Lhs))
		for i, ident := range n.Lhs {
			lhs[i] = ident
		}

	case *ast.Assignment:

		switch n.Type {
		case ast.AssignmentIncrement, ast.AssignmentDecrement:
			v := n.Lhs[0]
			tc.cantBeBlank(v)
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
			tc.cantBeBlank(n.Lhs[0])
			_, err := tc.binaryOp(n.Lhs[0], opType, n.Rhs[0])
			if err != nil {
				panic(tc.errorf(n, "invalid operation: %v (%s)", n, err))
			}
			tc.assign(node, n.Lhs[0], n.Rhs[0], nil, nil, false, false)
			return
		}

		lhs = n.Lhs
		rhs = n.Rhs
		isVarDecl = n.Type == ast.AssignmentDeclaration
		isAssignmentNode = true

	}

	if len(lhs) >= 2 && len(rhs) == 1 {
		call, ok := rhs[0].(*ast.Call)
		if ok {
			tis, isBuiltin, _ := tc.checkCallExpression(call, false)
			if len(lhs) != len(tis) {
				if isBuiltin {
					panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(lhs), len(rhs)))
				}
				panic(tc.errorf(node, "assignment mismatch: %d variables but %v returns %d values", len(lhs), call, len(rhs)))
			}
			rhs = nil
			for _, ti := range tis {
				newCall := ast.NewCall(call.Pos(), call.Func, call.Args, false)
				tc.typeInfos[newCall] = ti
				rhs = append(rhs, newCall)
			}
		}
	}

	if len(lhs) == 2 && len(rhs) == 1 {
		switch v := rhs[0].(type) {

		case *ast.TypeAssertion:

			v1 := ast.NewTypeAssertion(v.Pos(), v.Expr, v.Type)
			v2 := ast.NewTypeAssertion(v.Pos(), v.Expr, v.Type)
			ti := tc.checkExpression(rhs[0])
			tc.typeInfos[v1] = &TypeInfo{Type: ti.Type}
			tc.typeInfos[v2] = untypedBoolTypeInfo
			rhs = []ast.Expression{v1, v2}

		case *ast.Index:

			v1 := ast.NewIndex(v.Pos(), v.Expr, v.Index)
			v2 := ast.NewIndex(v.Pos(), v.Expr, v.Index)
			ti := tc.checkExpression(rhs[0])
			tc.typeInfos[v1] = &TypeInfo{Type: ti.Type}
			tc.typeInfos[v2] = untypedBoolTypeInfo
			rhs = []ast.Expression{v1, v2}

		case *ast.UnaryOperator:

			if v.Op == ast.OperatorReceive {
				v1 := ast.NewUnaryOperator(v.Pos(), ast.OperatorReceive, v.Expr)
				v2 := ast.NewUnaryOperator(v.Pos(), ast.OperatorReceive, v.Expr)
				ti := tc.checkExpression(rhs[0])
				tc.typeInfos[v1] = &TypeInfo{Type: ti.Type}
				tc.typeInfos[v2] = untypedBoolTypeInfo
				rhs = []ast.Expression{v1, v2}
			}

		}
	}

	if len(lhs) != len(rhs) {
		panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(lhs), len(rhs)))
	}

	newVars := []string{}
	tmpScope := typeCheckerScope{}
	for i := range lhs {
		if isConstDecl {
			tc.iota++
		}
		var newVar string
		if valueTi := tc.typeInfos[rhs[i]]; valueTi == nil {
			newVar = tc.assign(node, lhs[i], rhs[i], nil, declType, isVarDecl, isConstDecl)
		} else {
			newVar = tc.assign(node, lhs[i], nil, valueTi, declType, isVarDecl, isConstDecl)
		}
		if isVarDecl || isConstDecl {
			ti, _ := tc.lookupScopes(newVar, true)
			tmpScope[newVar] = scopeElement{t: ti, decl: lhs[i].(*ast.Identifier)}
			if len(tc.scopes) > 0 {
				delete(tc.scopes[len(tc.scopes)-1], newVar)
			} else {
				delete(tc.filePackageBlock, newVar)
			}
		}
		if (isVarDecl || isConstDecl) && !isAssignmentNode && newVar == "" && !isBlankIdentifier(lhs[i]) {
			panic(tc.errorf(node, "%s redeclared in this block", lhs[i]))
		}
		for _, v := range newVars {
			if newVar == v {
				panic(tc.errorf(node, "%s repeated on left side of :=", lhs[i]))
			}
		}
		if newVar != "" {
			newVars = append(newVars, newVar)
		}
	}
	if len(newVars) == 0 && isVarDecl && isAssignmentNode {
		panic(tc.errorf(node, "no new variables on left side of :="))
	}
	for d, ti := range tmpScope {
		tc.assignScope(d, ti.t, ti.decl)
	}
	return

}

// assign assigns rightExpr to leftExpr. If right is not nil, then is used
// instead of rightExpr. typ is the type specified in the declaration, if any.
// If assignment is a declaration and the scope has been updated, returns the
// identifier of the new scope element; otherwise returns an empty string.
func (tc *typechecker) assign(node ast.Node, leftExpr, rightExpr ast.Expression, right *TypeInfo, typ *TypeInfo, isVarDecl, isConstDecl bool) string {

	if right == nil {
		right = tc.checkExpression(rightExpr)
	}

	if isConstDecl && !right.IsConstant() {
		panic(tc.errorf(node, "const initializer %s is not a constant", rightExpr))
	}

	if typ == nil {
		// Type is not explicit, so is deducted by value.
		right.setValue(nil)
	} else {
		// Type is explicit, so must check assignability.
		if err := isAssignableTo(right, rightExpr, typ.Type); err != nil {
			if rightExpr == nil {
				panic(tc.errorf(node, "cannot assign %s to %s (type %s) in multiple assignment", right.ShortString(), leftExpr, typ))
			}
			panic(tc.errorf(node, "%s in assignment", err))
		}
		right.setValue(typ.Type)
	}

	// When declaring a variable, left side must be a name.
	if _, isIdent := leftExpr.(*ast.Identifier); isVarDecl && !isIdent {
		panic(tc.errorf(node, "non-name %s on left side of :=", leftExpr))
	}

	switch leftExpr := leftExpr.(type) {

	case *ast.Identifier:

		if leftExpr.Name == "_" {
			return ""
		}

		if isConstDecl {
			newRight := &TypeInfo{}
			if typ == nil {
				if right.Nil() {
					panic(tc.errorf(node, "use of untyped nil"))
				}
				newRight.Type = right.Type
			} else {
				newRight.Type = typ.Type
			}
			tc.typeInfos[leftExpr] = newRight
			if _, alreadyInCurrentScope := tc.lookupScopes(leftExpr.Name, true); alreadyInCurrentScope {
				return ""
			}
			newRight.Constant = right.Constant
			if right.Untyped() {
				newRight.Properties = PropertyUntyped
			}
			tc.assignScope(leftExpr.Name, newRight, nil)
			return leftExpr.Name
		}

		if isVarDecl {
			newRight := &TypeInfo{}
			if typ == nil {
				if right.Nil() {
					panic(tc.errorf(node, "use of untyped nil"))
				}
				newRight.Type = right.Type
			} else {
				newRight.Type = typ.Type
			}
			tc.typeInfos[leftExpr] = newRight
			if _, alreadyInCurrentScope := tc.lookupScopes(leftExpr.Name, true); alreadyInCurrentScope {
				return ""
			}
			newRight.Properties |= PropertyAddressable
			tc.assignScope(leftExpr.Name, newRight, leftExpr)
			if !tc.opts.AllowNotUsed {
				tc.unusedVars = append(tc.unusedVars, &scopeVariable{
					ident:      leftExpr.Name,
					scopeLevel: len(tc.scopes) - 1,
					node:       node,
				})
			}
			return leftExpr.Name
		}

		left := tc.checkIdentifier(leftExpr, false)
		if !left.Addressable() {
			panic(tc.errorf(leftExpr, "cannot assign to %v", leftExpr))
		}
		if err := isAssignableTo(right, rightExpr, left.Type); err != nil {
			panic(tc.errorf(rightExpr, "%s in assignment", err))
		}
		right.setValue(left.Type)
		tc.typeInfos[leftExpr] = left

	case *ast.Index:

		left := tc.checkExpression(leftExpr)
		switch left.Type.Kind() {
		case reflect.Slice, reflect.Map:
			// Always addressable when used in indexing operation.
		case reflect.Array:
			if !left.Addressable() {
				panic(tc.errorf(node, "cannot assign to %v", leftExpr))
			}
		}
		if err := isAssignableTo(right, rightExpr, left.Type); err != nil {
			panic(tc.errorf(node, "%s in assignment", err))
		}
		right.setValue(left.Type)
		return ""

	case *ast.Selector:

		left := tc.checkExpression(leftExpr)
		if !left.Addressable() {
			panic(tc.errorf(node, "cannot assign to %v", left))
		}
		if err := isAssignableTo(right, rightExpr, left.Type); err != nil {
			panic(tc.errorf(node, "%s in assignment", err))
		}
		right.setValue(left.Type)
		return ""

	case *ast.UnaryOperator:

		if leftExpr.Operator() == ast.OperatorMultiplication { // pointer indirection.
			left := tc.checkExpression(leftExpr)
			if err := isAssignableTo(right, rightExpr, left.Type); err != nil {
				panic(tc.errorf(node, "%s in assignment", err))
			}
			right.setValue(left.Type)
			return ""
		}
		panic(tc.errorf(node, "cannot assign to %v", leftExpr))

	case *ast.Call: // call on left side of assignment: f() = 10 .

		retValues, _, _ := tc.checkCallExpression(leftExpr, false)
		switch len(retValues) {
		case 0:
			panic(tc.errorf(node, "%s used as value", leftExpr))
		case 1:
			panic(tc.errorf(node, "cannot assign to %v", leftExpr))
		default:
			panic(tc.errorf(node, "multiple-value %s in single-value context", leftExpr))
		}

	default:

		panic(tc.errorf(node, "cannot assign to %v", leftExpr))
	}

	return ""
}

// cantBeBlank panics if expr is the blank identifier.
func (tc *typechecker) cantBeBlank(expr ast.Expression) {
	if isBlankIdentifier(expr) {
		panic(tc.errorf(expr, "cannot use _ as value"))
	}
}
