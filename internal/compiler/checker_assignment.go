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

	var leftExprs, rightExprs []ast.Expression
	var typ *TypeInfo
	var isDecl, isConst, isVar bool

	if tc.lastConstPosition != node.Pos() {
		tc.iota = -1
	}

	switch n := node.(type) {

	case *ast.Var:

		rightExprs = n.Rhs
		isDecl = true
		isVar = true
		if n.Type != nil {
			typ = tc.checkType(n.Type, noEllipses)
		}

		if len(rightExprs) == 0 {
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

		if len(n.Lhs) == 1 && len(rightExprs) == 1 {
			newVar := tc.assignSingle(node, n.Lhs[0], rightExprs[0], nil, typ, true, false)
			if !isBlankIdentifier(n.Lhs[0]) && newVar == "" {
				panic(tc.errorf(node, "%s redeclared in this block", n.Lhs[0]))
			}
			return
		}

		leftExprs = make([]ast.Expression, len(n.Lhs))
		for i, ident := range n.Lhs {
			leftExprs[i] = ident
		}

	case *ast.Const:

		rightExprs = n.Rhs
		isDecl = true
		isConst = true
		if n.Type != nil {
			typ = tc.checkType(n.Type, noEllipses)
		}
		tc.lastConstPosition = node.Pos()

		if len(n.Lhs) > len(rightExprs) {
			panic(tc.errorf(node, "missing value in const declaration"))
		}
		if len(n.Lhs) < len(rightExprs) {
			panic(tc.errorf(node, "extra expression in const declaration"))
		}

		leftExprs = make([]ast.Expression, len(n.Lhs))
		for i, ident := range n.Lhs {
			leftExprs[i] = ident
		}

	case *ast.Assignment:

		switch n.Type {
		case ast.AssignmentIncrement, ast.AssignmentDecrement:
			v := n.Lhs[0]
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
			if isBlankIdentifier(n.Lhs[0]) {
				panic(tc.errorf(n.Lhs[0], "cannot use _ as value"))
			}
			_, err := tc.binaryOp(n.Lhs[0], opType, n.Rhs[0])
			if err != nil {
				panic(tc.errorf(n, "invalid operation: %v (%s)", n, err))
			}
			tc.assignSingle(node, n.Lhs[0], n.Rhs[0], nil, nil, false, false)
			return
		}

		leftExprs = n.Lhs
		rightExprs = n.Rhs
		isDecl = n.Type == ast.AssignmentDeclaration

		if len(leftExprs) == 1 && len(rightExprs) == 1 {
			newVar := tc.assignSingle(node, leftExprs[0], rightExprs[0], nil, typ, isDecl, false)
			if newVar == "" && isDecl {
				panic(tc.errorf(node, "no new variables on left side of :="))
			}
			return
		}

	default:

		panic(fmt.Errorf("bug: unexpected node %T", node))

	}

	if len(leftExprs) >= 2 && len(rightExprs) == 1 {
		call, ok := rightExprs[0].(*ast.Call)
		if ok {
			tis, isBuiltin, _ := tc.checkCallExpression(call, false)
			if len(leftExprs) != len(tis) {
				if isBuiltin {
					panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(leftExprs), len(rightExprs)))
				}
				panic(tc.errorf(node, "assignment mismatch: %d variables but %v returns %d values", len(leftExprs), call, len(rightExprs)))
			}
			rightExprs = nil
			for _, ti := range tis {
				newCall := ast.NewCall(call.Pos(), call.Func, call.Args, false)
				tc.TypeInfo[newCall] = ti
				rightExprs = append(rightExprs, newCall)
			}
		}
	}

	if len(leftExprs) == 2 && len(rightExprs) == 1 {
		switch v := rightExprs[0].(type) {

		case *ast.TypeAssertion:

			v1 := ast.NewTypeAssertion(v.Pos(), v.Expr, v.Type)
			v2 := ast.NewTypeAssertion(v.Pos(), v.Expr, v.Type)
			ti := tc.checkExpression(rightExprs[0])
			tc.TypeInfo[v1] = &TypeInfo{Type: ti.Type}
			tc.TypeInfo[v2] = untypedBoolTypeInfo
			rightExprs = []ast.Expression{v1, v2}

		case *ast.Index:

			v1 := ast.NewIndex(v.Pos(), v.Expr, v.Index)
			v2 := ast.NewIndex(v.Pos(), v.Expr, v.Index)
			ti := tc.checkExpression(rightExprs[0])
			tc.TypeInfo[v1] = &TypeInfo{Type: ti.Type}
			tc.TypeInfo[v2] = untypedBoolTypeInfo
			rightExprs = []ast.Expression{v1, v2}

		case *ast.UnaryOperator:

			if v.Op == ast.OperatorReceive {
				v1 := ast.NewUnaryOperator(v.Pos(), ast.OperatorReceive, v.Expr)
				v2 := ast.NewUnaryOperator(v.Pos(), ast.OperatorReceive, v.Expr)
				ti := tc.checkExpression(rightExprs[0])
				tc.TypeInfo[v1] = &TypeInfo{Type: ti.Type}
				tc.TypeInfo[v2] = untypedBoolTypeInfo
				rightExprs = []ast.Expression{v1, v2}
			}

		}
	}

	if len(leftExprs) != len(rightExprs) {
		panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(leftExprs), len(rightExprs)))
	}

	newVars := []string{}
	tmpScope := TypeCheckerScope{}
	for i := range leftExprs {
		if isConst {
			tc.iota++
		}
		var newVar string
		if valueTi := tc.TypeInfo[rightExprs[i]]; valueTi == nil {
			newVar = tc.assignSingle(node, leftExprs[i], rightExprs[i], nil, typ, isDecl, isConst)
		} else {
			newVar = tc.assignSingle(node, leftExprs[i], nil, valueTi, typ, isDecl, isConst)
		}
		if isDecl {
			ti, _ := tc.lookupScopes(newVar, true)
			tmpScope[newVar] = scopeElement{t: ti, decl: leftExprs[i].(*ast.Identifier)}
			if len(tc.scopes) > 0 {
				delete(tc.scopes[len(tc.scopes)-1], newVar)
			} else {
				delete(tc.filePackageBlock, newVar)
			}
		}
		if (isVar || isConst) && newVar == "" && !isBlankIdentifier(leftExprs[i]) {
			panic(tc.errorf(node, "%s redeclared in this block", leftExprs[i]))
		}
		for _, v := range newVars {
			if newVar == v {
				panic(tc.errorf(node, "%s repeated on left side of :=", leftExprs[i]))
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
		tc.assignScope(d, ti.t, ti.decl)
	}
	return

}

// assignSingle assigns value to variable (or valueTi to variable if value is
// nil). typ is the type specified in the declaration, if any. If assignment
// is a declaration and the scope has been updated, returns the identifier of
// the new scope element; otherwise returns an empty string.
func (tc *typechecker) assignSingle(node ast.Node, leftExpr, rightExpr ast.Expression, right *TypeInfo, typ *TypeInfo, isDeclaration, isConst bool) string {

	if right == nil {
		right = tc.checkExpression(rightExpr)
	}

	if isConst && !right.IsConstant() {
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

	switch leftExpr := leftExpr.(type) {

	case *ast.Identifier:

		if leftExpr.Name == "_" {
			return ""
		}

		if isDeclaration {
			newRight := &TypeInfo{}
			if typ == nil {
				if right.Nil() {
					panic(tc.errorf(node, "use of untyped nil"))
				}
				//  «[if no types are presents], each variable is given the type
				//  of the corresponding initialization value in the
				//  assignment.»
				//
				// «If that value is an untyped constant, it is first
				// implicitly converted to its default type.»
				newRight.Type = right.Type
			} else {
				newRight.Type = typ.Type
			}
			tc.TypeInfo[leftExpr] = newRight
			if _, alreadyInCurrentScope := tc.lookupScopes(leftExpr.Name, true); alreadyInCurrentScope {
				return ""
			}
			if isConst {
				newRight.Constant = right.Constant
				if right.Untyped() {
					newRight.Properties = PropertyUntyped
				}
				tc.assignScope(leftExpr.Name, newRight, nil)
				return leftExpr.Name
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
		tc.TypeInfo[leftExpr] = left

	case *ast.Index:

		if isDeclaration {
			panic(tc.errorf(node, "non name %s on left side of :=", leftExpr))
		}
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

		if isDeclaration {
			panic(tc.errorf(node, "non name %s on left side of :=", leftExpr))
		}
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

		if isDeclaration {
			panic(tc.errorf(node, "non name %s on left side of :=", leftExpr))
		}
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

		if isDeclaration {
			panic(tc.errorf(node, "non name %s on left side of :=", leftExpr))
		}
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
