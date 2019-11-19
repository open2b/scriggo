// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"

	"scriggo/ast"
)

func (tc *typechecker) declareVariable(lh *ast.Identifier, typ reflect.Type) {
	ti := &TypeInfo{
		Type:       typ,
		Properties: PropertyAddressable,
	}
	tc.typeInfos[lh] = ti
	tc.assignScope(lh.Name, ti, lh)
	if !tc.opts.AllowNotUsed {
		tc.unusedVars = append(tc.unusedVars, &scopeVariable{
			ident:      lh.Name,
			scopeLevel: len(tc.scopes) - 1,
			node:       lh,
		})
	}
}

// isMapIndexExpression reports whether node is a map index expression.
func (tc *typechecker) isMapIndexExpression(node ast.Node) bool {
	index, ok := node.(*ast.Index)
	if !ok {
		return false
	}
	expr := index.Expr
	exprKind := tc.checkExpr(expr).Type.Kind()
	return exprKind == reflect.Map
}

// checkConstantDeclaration type checks a constant declaration.
// See https://golang.org/ref/spec#Constant_declarations.
func (tc *typechecker) checkConstantDeclaration(node *ast.Const) {

	if tc.lastConstPosition != node.Pos() {
		tc.iota = -1
	}

	tc.lastConstPosition = node.Pos()
	tc.iota++

	if len(node.Lhs) > len(node.Rhs) {
		panic(tc.errorf(node, "missing value in const declaration"))
	}

	if len(node.Rhs) < len(node.Rhs) {
		panic(tc.errorf(node, "extra expression in const declaration"))
	}

	rhs := []*TypeInfo{}

	// Type check every Rh expression: they must be constant.
	for _, rhExpr := range node.Rhs {
		rh := tc.checkExpr(rhExpr)
		if !rh.IsConstant() {
			if rh.Nil() {
				panic(tc.errorf(node, "const initializer cannot be nil"))
			}
			panic(tc.errorf(node, "const initializer %s is not a constant", rhExpr))
		}
		rhs = append(rhs, rh)
	}

	var typ *TypeInfo

	if node.Type != nil {
		// Every Rh must be assignable to the type.
		typ = tc.checkType(node.Type)
		for i := range rhs {
			err := tc.isAssignableTo(rhs[i], node.Rhs[i], typ.Type)
			if err != nil {
				panic(tc.errorf(node.Rhs[i], "%s in assignment", err))
			}
		}
	}

	for i, rh := range rhs {

		// Prepare the constant value.
		constValue := rh.Constant

		// Prepare the constant type.
		var constType reflect.Type
		if typ == nil {
			constType = rh.Type
		} else {
			constType = typ.Type
		}

		if isBlankIdentifier(node.Lhs[i]) {
			continue
		}

		// Declare the constant in the current block/scope.
		constTi := &TypeInfo{Type: constType, Constant: constValue}
		if rh.Untyped() {
			constTi.Properties |= PropertyUntyped
		}
		tc.assignScope(node.Lhs[i].Name, constTi, node.Lhs[i])

	}

}

func (tc *typechecker) newPlaceholderFor(typ reflect.Type) *ast.Placeholder {
	k := typ.Kind()
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
		ti = &TypeInfo{Type: typ, value: tc.types.Zero(typ).Interface(), Properties: PropertyHasValue}
		ti.setValue(typ)
	}
	ph := ast.NewPlaceholder()
	tc.typeInfos[ph] = ti
	return ph
}

func (tc *typechecker) rebalanceRightSide(node ast.Node) []ast.Expression {

	var nodeLhs, nodeRhs []ast.Expression

	switch node := node.(type) {
	case *ast.Var:
		for _, lh := range node.Lhs {
			nodeLhs = append(nodeLhs, lh)
		}
		nodeRhs = node.Rhs
	case *ast.Assignment:
		nodeLhs = node.Lhs
		nodeRhs = node.Rhs
	}

	if call, ok := nodeRhs[0].(*ast.Call); ok {
		tis, isBuiltin, _ := tc.checkCallExpression(call, false)
		if len(nodeLhs) != len(tis) {
			if isBuiltin {
				panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(nodeLhs), len(tis)))
			}
			panic(tc.errorf(node, "assignment mismatch: %d variables but %v returns %d values", len(nodeLhs), call, len(tis)))
		}
		rhsExpr := make([]ast.Expression, len(tis))
		for i, ti := range tis {
			rhsExpr[i] = ast.NewCall(call.Pos(), call.Func, call.Args, false)
			tc.typeInfos[rhsExpr[i]] = ti
		}
		return rhsExpr
	}

	if len(nodeLhs) == 2 && len(nodeRhs) == 1 {
		switch v := nodeRhs[0].(type) {
		case *ast.TypeAssertion:
			v1 := ast.NewTypeAssertion(v.Pos(), v.Expr, v.Type)
			v2 := ast.NewTypeAssertion(v.Pos(), v.Expr, v.Type)
			ti := tc.checkExpr(nodeRhs[0])
			tc.typeInfos[v1] = &TypeInfo{Type: ti.Type}
			tc.typeInfos[v2] = untypedBoolTypeInfo
			return []ast.Expression{v1, v2}
		case *ast.Index:
			v1 := ast.NewIndex(v.Pos(), v.Expr, v.Index)
			v2 := ast.NewIndex(v.Pos(), v.Expr, v.Index)
			ti := tc.checkExpr(nodeRhs[0])
			tc.typeInfos[v1] = &TypeInfo{Type: ti.Type}
			tc.typeInfos[v2] = untypedBoolTypeInfo
			return []ast.Expression{v1, v2}
		case *ast.UnaryOperator:
			if v.Op == ast.OperatorReceive {
				v1 := ast.NewUnaryOperator(v.Pos(), ast.OperatorReceive, v.Expr)
				v2 := ast.NewUnaryOperator(v.Pos(), ast.OperatorReceive, v.Expr)
				ti := tc.checkExpr(nodeRhs[0])
				tc.typeInfos[v1] = &TypeInfo{Type: ti.Type}
				tc.typeInfos[v2] = untypedBoolTypeInfo
				return []ast.Expression{v1, v2}
			}
		}
	}

	panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(nodeLhs), len(nodeRhs)))

}

// checkVariableDeclaration type checks a variable declaration.
// See https://golang.org/ref/spec#Variable_declarations.
func (tc *typechecker) checkVariableDeclaration(node *ast.Var) {

	var rhs []*TypeInfo

	nodeRhs := node.Rhs

	isMultipleAssignment := false // just for error messages

	if len(node.Rhs) > 0 && len(node.Lhs) != len(node.Rhs) {
		nodeRhs = tc.rebalanceRightSide(node)
		isMultipleAssignment = true
	}

	for _, rhExpr := range nodeRhs {
		rh := tc.checkExpr(rhExpr)
		rhs = append(rhs, rh)
	}

	var typ *TypeInfo

	if node.Type != nil {
		// Every Rh must be assignable to the type.
		typ = tc.checkType(node.Type)
		for i := range rhs {
			err := tc.isAssignableTo(rhs[i], nodeRhs[i], typ.Type)
			if err != nil {
				if isMultipleAssignment {
					panic(tc.errorf(node.Rhs[0], "cannot assign %v to %v (type %v) in multiple assignment", rhs[i].Type, node.Lhs[i], typ.Type))
				}
				panic(tc.errorf(nodeRhs[i], "%s in assignment", err))
			}
		}
	}

	// If typ is not specified, none of the Rh values must be the predeclared
	// nil.
	if typ == nil {
		for i, rh := range rhs {
			if rh.Nil() {
				panic(tc.errorf(nodeRhs[i], "use of untyped nil"))
			}
		}
	}

	if len(node.Rhs) == 0 {
		node.Rhs = make([]ast.Expression, len(node.Lhs))
		for i := 0; i < len(node.Lhs); i++ {
			node.Rhs[i] = tc.newPlaceholderFor(typ.Type)
			rhs = append(rhs, tc.checkExpr(node.Rhs[i]))
		}
	}

	// TODO: check for repetitions on the left side of the =

	for i, rh := range rhs {
		varTyp := rh.Type
		if typ != nil {
			varTyp = typ.Type
		}
		if rh.Nil() {
			tc.typeInfos[nodeRhs[i]] = tc.nilOf(typ.Type)
		} else {
			rh.setValue(varTyp)
		}
		if isBlankIdentifier(node.Lhs[i]) {
			continue
		}
		tc.declareVariable(node.Lhs[i], varTyp)
	}

}

func (tc *typechecker) checkRepeatedLhs(node *ast.Assignment) {
	names := map[string]bool{}
	for _, lh := range node.Lhs {
		if isBlankIdentifier(lh) {
			continue
		}
		name := lh.(*ast.Identifier).Name
		if names[name] {
			panic(tc.errorf(node, "%s repeated on left side of :=", name))
		}
		names[name] = true
	}
}

// checkShortVariableDeclaration type checks a short variable declaration. See
// https://golang.org/ref/spec#Short_variable_declarations.
func (tc *typechecker) checkShortVariableDeclaration(node *ast.Assignment) {

	// Check that node is a short variable declaration.
	if node.Type != ast.AssignmentDeclaration {
		panic("BUG: expected a short variable declaration")
	}

	nodeRhs := node.Rhs

	if len(node.Lhs) != len(nodeRhs) {
		nodeRhs = tc.rebalanceRightSide(node)
	}

	for _, lhExpr := range node.Lhs {
		_, isIdent := lhExpr.(*ast.Identifier)
		if !isIdent {
			panic(tc.errorf(node, "non-name %s on left side of :=", lhExpr))
		}
	}

	tc.checkRepeatedLhs(node)

	var rhs []*TypeInfo
	for _, rhExpr := range nodeRhs {
		rh := tc.checkExpr(rhExpr)
		rhs = append(rhs, rh)
	}

	alreadyDeclared := map[ast.Expression]bool{}

	for _, lhExpr := range node.Lhs {
		name := lhExpr.(*ast.Identifier).Name
		if tc.declaredInThisBlock(name) || isBlankIdentifier(lhExpr) {
			alreadyDeclared[lhExpr] = true
		}
	}

	// no new variables on left side of :=
	if len(node.Lhs) == len(alreadyDeclared) {
		if tc.opts.SyntaxType == ScriptSyntax && tc.isScriptFuncDecl {
			panic(tc.errorf(node, "%v already declared in script", node.Lhs[0]))
		}
		panic(tc.errorf(node, "no new variables on left side of :="))
	}

	for i, rh := range rhs {
		rh.setValue(nil)
		if isBlankIdentifier(node.Lhs[i]) {
			continue
		}
		if alreadyDeclared[node.Lhs[i]] {
			tc.checkIdentifier(node.Lhs[i].(*ast.Identifier), false)
		} else {
			tc.declareVariable(node.Lhs[i].(*ast.Identifier), rh.Type)
		}
	}

	// TODO

}

// checkGenericAssignmentNode should be used in every context where an
// *ast.Assignment node must be checked. This is necessary because the
// *ast.Assignment node of the Scriggo AST represents short variable
// declarations, assignments, short assignments and inc-dec statements.
func (tc *typechecker) checkGenericAssignmentNode(node *ast.Assignment) {
	switch node.Type {
	case ast.AssignmentDeclaration:
		tc.checkShortVariableDeclaration(node)
	case ast.AssignmentIncrement, ast.AssignmentDecrement:
		tc.checkIncDecStatement(node)
	default:
		tc.checkAssignment(node)
	}
}

// See https://golang.org/ref/spec#Assignments.
// checkAssignments type check an assignment node.
func (tc *typechecker) checkAssignment(node *ast.Assignment) {

	// Check that node is an assignment node.
	switch node.Type {
	case ast.AssignmentDeclaration, ast.AssignmentIncrement, ast.AssignmentDecrement:
		panic("BUG: expected an assignment node")
	}

	nodeRhs := node.Rhs

	isMultipleAssignment := false // for error messages

	isShortAssignment := ast.AssignmentAddition <= node.Type && node.Type <= ast.AssignmentRightShift

	if len(node.Lhs) != len(nodeRhs) {
		nodeRhs = tc.rebalanceRightSide(node)
		isMultipleAssignment = true
	}

	var lhs, rhs []*TypeInfo
	for _, lhExpr := range node.Lhs {
		if isBlankIdentifier(lhExpr) {
			lhs = append(lhs, nil)
		} else {
			var lh *TypeInfo
			if ident, ok := lhExpr.(*ast.Identifier); ok && !isShortAssignment {
				lh = tc.checkIdentifier(ident, false)
			} else {
				lh = tc.checkExpr(lhExpr)
			}
			lhs = append(lhs, lh)
		}
	}
	for _, rhExpr := range nodeRhs {
		rh := tc.checkExpr(rhExpr)
		rhs = append(rhs, rh)
	}

	if isShortAssignment {
		op := operatorFromAssignmentType(node.Type)
		_, err := tc.binaryOp(node.Lhs[0], op, node.Rhs[0])
		if err != nil {
			panic(tc.errorf(node, "invalid operation: %v (%s)", node, err))
		}
	}

	for i, lh := range lhs {
		switch {
		case isBlankIdentifier(node.Lhs[i]):
			// Ok!
		case lh.Addressable():
			// Ok!
		case tc.isMapIndexExpression(node.Lhs[i]):
			// Ok!
		default:
			if tc.isSelectorOfMapField(node.Lhs[i]) {
				panic(tc.errorf(node.Lhs[i], "cannot assign to struct field %v in map", node.Lhs[i]))
			}
			panic(tc.errorf(node.Lhs[i], "cannot assign to %v", node.Lhs[i]))
		}
	}

	for i, rh := range rhs {
		if rh.Nil() {
			rh = tc.nilOf(lhs[i].Type)    // needed by the type checker
			tc.typeInfos[nodeRhs[i]] = rh // needed by the emitter
		} else {
			rh.setValue(nil)
		}
		if isBlankIdentifier(node.Lhs[i]) {
			continue
		}
		err := tc.isAssignableTo(rh, nodeRhs[i], lhs[i].Type)
		if err != nil {
			if isMultipleAssignment {
				panic(tc.errorf(node.Rhs[0], "cannot assign %v to %v (type %v) in multiple assignment", rh.Type, node.Lhs[i], lhs[i].Type))
			}
			panic(tc.errorf(nodeRhs[i], "%s in assignment", err))
		}
	}

	if op := node.Type; ast.AssignmentAddition <= op && op <= ast.AssignmentRightShift {
		tc.convertNodeForTheEmitter(node)
	}

}

func (tc *typechecker) isSelectorOfMapField(expr ast.Expression) bool {
	field, ok := expr.(*ast.Selector)
	if !ok {
		return false
	}
	index, ok := field.Expr.(*ast.Index)
	if !ok {
		return false
	}
	kind := tc.checkExpr(index.Expr).Type.Kind()
	return kind == reflect.Map
}

// checkIncDecStatement checks an IncDec statement. See
// https://golang.org/ref/spec#IncDec_statements.
func (tc *typechecker) checkIncDecStatement(node *ast.Assignment) {

	// Check that node is an IncDec statement.
	if node.Type != ast.AssignmentIncrement && node.Type != ast.AssignmentDecrement {
		panic("BUG: expected an IncDec statement")
	}

	lh := tc.checkExpr(node.Lhs[0])
	switch {
	case lh.Addressable():
		// Ok!
	case tc.isMapIndexExpression(node.Lhs[0]):
		// Ok!
	default:
		if tc.isSelectorOfMapField(node.Lhs[0]) {
			panic(tc.errorf(node.Lhs[0], "cannot assign to struct field %v in map", node.Lhs[0]))
		}
		panic(tc.errorf(node.Lhs[0], "cannot assign to %v", node.Lhs[0]))
	}

	if !isNumeric(lh.Type.Kind()) {
		panic(tc.errorf(node, "invalid operation: %v (non-numeric type %s)", node, lh))
	}

	tc.convertNodeForTheEmitter(node)

	// TODO: the untyped constant '1' must be assignable to the type of lh.

}

// TODO: this implementation is wrong. It has been kept to make the test pass,
// but it must be changed:
//
//		m[f()] ++
//
// currenlty calls twice f(), which is wrong.
//
// See https://github.com/open2b/scriggo/issues/222.
func (tc *typechecker) convertNodeForTheEmitter(node *ast.Assignment) {
	pos := node.Lhs[0].Pos()
	op := operatorFromAssignmentType(node.Type)
	var right ast.Expression
	if node.Type == ast.AssignmentIncrement || node.Type == ast.AssignmentDecrement {
		right = ast.NewBinaryOperator(pos, op, node.Lhs[0], ast.NewBasicLiteral(pos, ast.IntLiteral, "1"))
	} else {
		right = ast.NewBinaryOperator(pos, op, node.Lhs[0], node.Rhs[0])
	}
	tc.checkExpr(right)
	node.Rhs = []ast.Expression{right}
	node.Type = ast.AssignmentSimple
}
