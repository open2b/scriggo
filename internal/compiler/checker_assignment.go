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

	var lhs []ast.Expression
	var rhs []ast.Expression
	var declTi *TypeInfo

	// isVariableDecl indicates if node is:
	//   - an *ast.Var declaration
	//   - an *ast.Assignment with ':='
	var isVariableDecl bool
	var isConstDecl bool
	var isAssignmentNode bool

	if tc.lastConstPosition != node.Pos() {
		tc.iota = -1
	}

	switch n := node.(type) {

	case *ast.Var:

		if n.Type != nil {
			declTi = tc.checkType(n.Type)
			if len(n.Rhs) == 0 {
				typ := declTi.Type
				for i := range n.Lhs {
					ph := ast.NewPlaceholder()
					tc.typeInfos[ph] = &TypeInfo{Type: typ}
					newVar := tc.assign(node, n.Lhs[i], ph, declTi, true, false)
					if newVar == "" && !isBlankIdentifier(n.Lhs[i]) {
						panic(tc.errorf(node, "%s redeclared in this block", n.Lhs[i]))
					}
				}
				k := typ.Kind()
				n.Rhs = make([]ast.Expression, len(n.Lhs))
				for i := range n.Lhs {
					var ti *TypeInfo
					switch {
					case reflect.Int <= k && k <= reflect.Complex128:
						ti = &TypeInfo{Type: typ, Constant: int64Const(0), Properties: PropertyUntyped}
						ti.setValue(typ)
					case k == reflect.String:
						ti = &TypeInfo{Type: typ, Constant: stringConst(""), Properties: PropertyUntyped}
						ti.setValue(typ)
					case k == reflect.Bool:
						ti = &TypeInfo{Type: typ, Constant: boolConst(false), Properties: PropertyUntyped}
						ti.setValue(typ)
					case k == reflect.Interface, k == reflect.Func:
						ti = tc.nilOf(typ)
					default:
						ti = &TypeInfo{Type: typ, value: reflect.Zero(typ).Interface(), Properties: PropertyHasValue}
						ti.setValue(typ)
					}
					n.Rhs[i] = ast.NewPlaceholder()
					tc.typeInfos[n.Rhs[i]] = ti
				}
				return
			}
		}

		isVariableDecl = true
		lhs = make([]ast.Expression, len(n.Lhs))
		for i, ident := range n.Lhs {
			lhs[i] = ident
		}
		rhs = n.Rhs

	case *ast.Const:

		isConstDecl = true
		if n.Type != nil {
			declTi = tc.checkType(n.Type)
		}
		tc.lastConstPosition = node.Pos()
		if len(n.Lhs) != len(n.Rhs) {
			if len(n.Lhs) < len(n.Rhs) {
				panic(tc.errorf(node, "extra expression in const declaration"))
			}
			panic(tc.errorf(node, "missing value in const declaration"))
		}
		lhs = make([]ast.Expression, len(n.Lhs))
		for i, ident := range n.Lhs {
			lhs[i] = ident
		}
		rhs = n.Rhs

	case *ast.Assignment:

		switch n.Type {
		case ast.AssignmentIncrement, ast.AssignmentDecrement:
			lh := n.Lhs[0]
			ti := tc.checkExpr(lh)
			if ti.Nil() {
				panic(tc.errorf(n, "cannot assign to nil"))
			}
			if ti.IsConstant() {
				panic(tc.errorf(n, "cannot assign to %v", lh))
			}
			var addressable bool
			switch lh := lh.(type) {
			case *ast.Identifier:
				addressable = true
			case *ast.Index:
				kind := tc.typeInfos[lh.Expr].Type.Kind()
				addressable = kind == reflect.Map || ti.Addressable()
			case *ast.Selector:
				addressable = ti.Addressable()
			case *ast.UnaryOperator:
				addressable = lh.Operator() == ast.OperatorMultiplication
			}
			if !addressable {
				panic(tc.errorf(n, "cannot assign to %v", lh))
			}
			if !isNumeric(ti.Type.Kind()) {
				panic(tc.errorf(n, "invalid operation: %v (non-numeric type %s)", n, ti))
			}
			// Convert the assignment node from ++ and -- to a simple
			// assignment. This change has no effects on type checking but
			// simplifies the emitting of assignment nodes.
			// a++ is semantically equivalent to a += 1, which is semantically
			// equivalent to a = a + 1.
			op := operatorFromAssignmentType(n.Type)
			n.Type = ast.AssignmentSimple
			rh := ast.NewBinaryOperator(lh.Pos(), op, lh, ast.NewBasicLiteral(lh.Pos(), ast.IntLiteral, "1"))
			tc.checkExpr(rh)
			n.Rhs = []ast.Expression{rh}
			return
		case ast.AssignmentAddition, ast.AssignmentSubtraction, ast.AssignmentMultiplication,
			ast.AssignmentDivision, ast.AssignmentModulo, ast.AssignmentAnd, ast.AssignmentOr,
			ast.AssignmentXor, ast.AssignmentAndNot, ast.AssignmentLeftShift, ast.AssignmentRightShift:
			tc.cantBeBlank(n.Lhs[0])
			var op = operatorFromAssignmentType(n.Type)
			_, err := tc.binaryOp(n.Lhs[0], op, n.Rhs[0])
			if err != nil {
				panic(tc.errorf(n, "invalid operation: %v (%s)", n, err))
			}
			tc.assign(node, n.Lhs[0], n.Rhs[0], nil, false, false)
			// Convert the assignment node from l op= r to a simple assignment.
			// This change has no effects on type checking but simplifies the
			// emitting of assignment nodes. a += 1 is semantically equivalent
			// to a = a + 1.
			pos := n.Lhs[0].Pos()
			right := ast.NewBinaryOperator(pos, op, n.Lhs[0], n.Rhs[0])
			tc.checkExpr(right)
			n.Rhs = []ast.Expression{right}
			n.Type = ast.AssignmentSimple
			return
		}

		lhs = n.Lhs
		rhs = n.Rhs
		isVariableDecl = n.Type == ast.AssignmentDeclaration
		isAssignmentNode = true

	}

	if len(lhs) >= 2 && len(rhs) == 1 {
		if call, ok := rhs[0].(*ast.Call); ok {
			tis, isBuiltin, _ := tc.checkCallExpression(call, false)
			if len(lhs) != len(tis) {
				if isBuiltin {
					panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(lhs), len(rhs)))
				}
				panic(tc.errorf(node, "assignment mismatch: %d variables but %v returns %d values", len(lhs), call, len(rhs)))
			}
			rhs = make([]ast.Expression, len(tis))
			for i, ti := range tis {
				rhs[i] = ast.NewCall(call.Pos(), call.Func, call.Args, false)
				tc.typeInfos[rhs[i]] = ti
			}
		}
	}

	if len(lhs) == 2 && len(rhs) == 1 {
		switch v := rhs[0].(type) {
		case *ast.TypeAssertion:
			v1 := ast.NewTypeAssertion(v.Pos(), v.Expr, v.Type)
			v2 := ast.NewTypeAssertion(v.Pos(), v.Expr, v.Type)
			ti := tc.checkExpr(rhs[0])
			tc.typeInfos[v1] = &TypeInfo{Type: ti.Type}
			tc.typeInfos[v2] = untypedBoolTypeInfo
			rhs = []ast.Expression{v1, v2}
		case *ast.Index:
			v1 := ast.NewIndex(v.Pos(), v.Expr, v.Index)
			v2 := ast.NewIndex(v.Pos(), v.Expr, v.Index)
			ti := tc.checkExpr(rhs[0])
			tc.typeInfos[v1] = &TypeInfo{Type: ti.Type}
			tc.typeInfos[v2] = untypedBoolTypeInfo
			rhs = []ast.Expression{v1, v2}
		case *ast.UnaryOperator:
			if v.Op == ast.OperatorReceive {
				v1 := ast.NewUnaryOperator(v.Pos(), ast.OperatorReceive, v.Expr)
				v2 := ast.NewUnaryOperator(v.Pos(), ast.OperatorReceive, v.Expr)
				ti := tc.checkExpr(rhs[0])
				tc.typeInfos[v1] = &TypeInfo{Type: ti.Type}
				tc.typeInfos[v2] = untypedBoolTypeInfo
				rhs = []ast.Expression{v1, v2}
			}
		}
	}

	if len(lhs) != len(rhs) {
		panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(lhs), len(rhs)))
	}

	var newVars []string
	tmpScope := typeCheckerScope{}
	for i := range lhs {
		if isConstDecl {
			tc.iota++
		}
		var newVar string
		if valueTi := tc.typeInfos[rhs[i]]; valueTi == nil {
			newVar = tc.assign(node, lhs[i], rhs[i], declTi, isVariableDecl, isConstDecl)
		} else {
			ph := ast.NewPlaceholder()
			tc.typeInfos[ph] = valueTi
			newVar = tc.assign(node, lhs[i], ph, declTi, isVariableDecl, isConstDecl)
		}
		if isVariableDecl || isConstDecl {
			if !isAssignmentNode && newVar == "" && !isBlankIdentifier(lhs[i]) {
				panic(tc.errorf(node, "%s redeclared in this block", lhs[i]))
			}
			ti, _ := tc.lookupScopes(newVar, true)
			tmpScope[newVar] = scopeElement{t: ti, decl: lhs[i].(*ast.Identifier)}
			if len(tc.scopes) > 0 {
				delete(tc.scopes[len(tc.scopes)-1], newVar)
			} else {
				delete(tc.filePackageBlock, newVar)
			}
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
	if len(newVars) == 0 && isVariableDecl && isAssignmentNode {
		if tc.opts.SyntaxType == ScriptSyntax && tc.isScriptFuncDecl {
			panic(tc.errorf(node, "%v already declared in script", lhs[0]))
		}
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
func (tc *typechecker) assign(node ast.Node, leftExpr, rightExpr ast.Expression, typ *TypeInfo, isVariableDecl, isConstDecl bool) string {

	right := tc.checkExpr(rightExpr)

	// Assignment using '=' with 'nil' as right value.
	//
	//	s = nil // where s has type []int
	//
	if !isVariableDecl && !isConstDecl && right.Nil() {
		left := tc.checkExpr(leftExpr)
		right = tc.nilOf(left.Type)
		tc.typeInfos[rightExpr] = right
	}

	// Variable declaration using 'var' with an explicit type and 'nil' as right value.
	//
	//	var a []int = nil
	//
	if isVariableDecl && typ != nil && right.Nil() {
		right = tc.nilOf(typ.Type)
		tc.typeInfos[rightExpr] = right
	}

	if isConstDecl {
		if right.Nil() {
			panic(tc.errorf(node, "const initializer cannot be nil"))
		}
		if !right.IsConstant() {
			panic(tc.errorf(node, "const initializer %s is not a constant", rightExpr))
		}
	}

	if typ == nil {
		// Type is not explicit, so is deducted by value.
		right.setValue(nil)
	} else {
		// Type is explicit, so must check assignability.
		if err := tc.isAssignableTo(right, rightExpr, typ.Type); err != nil {
			if _, ok := rightExpr.(*ast.Placeholder); ok {
				panic(tc.errorf(node, "cannot assign %s to %s (type %s) in multiple assignment", right, leftExpr, typ))
			}
			panic(tc.errorf(node, "%s in assignment", err))
		}
		if right.Nil() {
			// Note that this doesn't change the type info associated to node
			// 'right'; it just uses a new type info inside this function.
			right = tc.nilOf(typ.Type)
		} else {
			right.setValue(typ.Type)
		}
	}

	// When declaring a variable, left side must be a name.
	// Note that the error message takes for granted that isVariableDecl refers
	// to a declaration assignment. This is always true because 'var' nodes
	// require an *ast.Identifier as lhs, so !isIdent is always false.
	if isVariableDecl {
		if _, isIdent := leftExpr.(*ast.Identifier); !isIdent {
			panic(tc.errorf(node, "non-name %s on left side of :=", leftExpr))
		}
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

		if isVariableDecl {
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

		// Simple assignment.
		left := tc.checkIdentifier(leftExpr, false)
		if !left.Addressable() {
			panic(tc.errorf(leftExpr, "cannot assign to %v", leftExpr))
		}
		if err := tc.isAssignableTo(right, rightExpr, left.Type); err != nil {
			if _, ok := rightExpr.(*ast.Placeholder); ok {
				panic(tc.errorf(node, "cannot assign %s to %s (type %s) in multiple assignment", right, leftExpr, left))
			}
			panic(tc.errorf(node, "%s in assignment", err))
		}
		right.setValue(left.Type)
		tc.typeInfos[leftExpr] = left

	case *ast.Index:

		left := tc.checkExpr(leftExpr)
		ti := tc.typeInfos[leftExpr.Expr]
		switch ti.Type.Kind() {
		case reflect.Array:
			if !left.Addressable() {
				panic(tc.errorf(leftExpr, "cannot assign to %v", leftExpr))
			}
		case reflect.String:
			panic(tc.errorf(node, "cannot assign to %v", leftExpr))
		default:
			// Slices, maps and pointers to array are always addressable when
			// used in indexing operation.
		}
		if err := tc.isAssignableTo(right, rightExpr, left.Type); err != nil {
			if _, ok := rightExpr.(*ast.Placeholder); ok {
				panic(tc.errorf(node, "cannot assign %s to %s (type %s) in multiple assignment", right, leftExpr, left))
			}
			panic(tc.errorf(node, "%s in assignment", err))
		}
		right.setValue(left.Type)
		return ""

	case *ast.Selector:

		left := tc.checkExpr(leftExpr)
		if !left.Addressable() {
			if e, ok := leftExpr.Expr.(*ast.Index); ok && tc.typeInfos[e.Expr].Type.Kind() == reflect.Map {
				panic(tc.errorf(leftExpr, "cannot assign to struct field %v in map", leftExpr))
			}
			panic(tc.errorf(leftExpr, "cannot assign to %v", leftExpr))
		}
		if err := tc.isAssignableTo(right, rightExpr, left.Type); err != nil {
			if _, ok := rightExpr.(*ast.Placeholder); ok {
				panic(tc.errorf(node, "cannot assign %s to %s (type %s) in multiple assignment", right, leftExpr, left))
			}
			panic(tc.errorf(node, "%s in assignment", err))
		}
		right.setValue(left.Type)
		return ""

	case *ast.UnaryOperator:

		if leftExpr.Operator() == ast.OperatorMultiplication { // pointer indirection.
			left := tc.checkExpr(leftExpr)
			if err := tc.isAssignableTo(right, rightExpr, left.Type); err != nil {
				if _, ok := rightExpr.(*ast.Placeholder); ok {
					panic(tc.errorf(node, "cannot assign %s to %s (type %s) in multiple assignment", right, leftExpr, left))
				}
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

// operatorFromAssignmentType returns an operator type from an assignment
// type.
func operatorFromAssignmentType(assignmentType ast.AssignmentType) ast.OperatorType {
	switch assignmentType {
	case ast.AssignmentAddition, ast.AssignmentIncrement:
		return ast.OperatorAddition
	case ast.AssignmentSubtraction, ast.AssignmentDecrement:
		return ast.OperatorSubtraction
	case ast.AssignmentMultiplication:
		return ast.OperatorMultiplication
	case ast.AssignmentDivision:
		return ast.OperatorDivision
	case ast.AssignmentModulo:
		return ast.OperatorModulo
	case ast.AssignmentAnd:
		return ast.OperatorAnd
	case ast.AssignmentOr:
		return ast.OperatorOr
	case ast.AssignmentXor:
		return ast.OperatorXor
	case ast.AssignmentAndNot:
		return ast.OperatorAndNot
	case ast.AssignmentLeftShift:
		return ast.OperatorLeftShift
	case ast.AssignmentRightShift:
		return ast.OperatorRightShift
	}
	panic("unexpected assignment type")
}
