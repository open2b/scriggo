// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"strings"

	"scriggo/ast"
)

// checkAssignment type check an assignment node.
func (tc *typechecker) checkAssignment(node *ast.Assignment) {

	switch node.Type {
	case ast.AssignmentDeclaration, ast.AssignmentIncrement, ast.AssignmentDecrement:
		panic("BUG: expected an assignment node")
	}

	// In case of unbalanced assignments a 'fake' rhs must be used for the type
	// checking, but the tree must not be changed.
	nodeRhs := node.Rhs

	if len(node.Lhs) != len(nodeRhs) {
		nodeRhs = tc.rebalanceRightSide(node)
	}

	// +=, -=, *=, ...
	isAssignmentOperation := ast.AssignmentAddition <= node.Type && node.Type <= ast.AssignmentRightShift

	// Type check the left side.
	lhs := make([]*TypeInfo, len(node.Lhs))
	for i, lhExpr := range node.Lhs {
		if isBlankIdentifier(lhExpr) {
			continue
		}
		// Find the Lh type info.
		var lh *TypeInfo
		if ident, ok := lhExpr.(*ast.Identifier); ok && !isAssignmentOperation {
			lh = tc.checkIdentifier(ident, false) // Use checkIdentifier to avoid marking 'ident' as used.
		} else {
			lh = tc.checkExpr(lhExpr)
		}
		switch {
		case lh.Addressable(): // Ok!
		case tc.isMapIndexing(lhExpr): // Ok!
		default:
			if tc.isSelectorOfMapIndexing(lhExpr) {
				panic(tc.errorf(lhExpr, "cannot assign to struct field %v in map", lhExpr))
			}
			panic(tc.errorf(lhExpr, "cannot assign to %v", lhExpr))
		}
		lhs[i] = lh
	}

	// Type check the right side.
	rhs := make([]*TypeInfo, len(nodeRhs))
	for i := range nodeRhs {
		rhs[i] = tc.checkExpr(nodeRhs[i])
	}

	// +=, -=, *= ...
	if isAssignmentOperation {
		op := operatorFromAssignmentType(node.Type)
		_, err := tc.binaryOp(node.Lhs[0], op, nodeRhs[0])
		if err != nil {
			panic(tc.errorf(node, "invalid operation: %v (%s)", node, err))
		}
	}

	// Check for assignability and update the type infos.
	for i, rh := range rhs {
		if isBlankIdentifier(node.Lhs[i]) {
			if rh.Nil() { // _ = nil
				panic(tc.errorf(node.Lhs[i], "use of untyped nil"))
			}
			if rh.IsUntypedConstant() {
				tc.mustBeAssignableTo(nodeRhs[i], rh.Type, false, nil)
			}
			rh.setValue(nil)
			continue
		}
		tc.mustBeAssignableTo(nodeRhs[i], lhs[i].Type, len(node.Lhs) != len(node.Rhs), node.Lhs[i])
		// Update the type info for the emitter.
		if rh.Nil() {
			tc.typeInfos[nodeRhs[i]] = tc.nilOf(lhs[i].Type)
		} else {
			rh.setValue(nil)
		}
	}

	// Transform the tree after the type checking to make things easier for the
	// emitter.
	if op := node.Type; ast.AssignmentAddition <= op && op <= ast.AssignmentRightShift {
		tc.convertNodeForTheEmitter(node)
	}

}

// checkConstantDeclaration type checks a constant declaration.
func (tc *typechecker) checkConstantDeclaration(node *ast.Const) {

	if len(node.Lhs) > len(node.Rhs) {
		panic(tc.errorf(node, "missing value in const declaration"))
	}

	if len(node.Rhs) < len(node.Rhs) {
		panic(tc.errorf(node, "extra expression in const declaration"))
	}

	if tc.lastConstPosition != node.Pos() {
		tc.iota = -1
	}

	tc.lastConstPosition = node.Pos()
	tc.iota++

	// Type check every Rh expression: they must be constant.
	rhs := make([]*TypeInfo, len(node.Rhs))
	for i, rhExpr := range node.Rhs {
		rh := tc.checkExpr(rhExpr)
		if !rh.IsConstant() {
			if rh.Nil() {
				panic(tc.errorf(node, "const initializer cannot be nil"))
			}
			panic(tc.errorf(node, "const initializer %s is not a constant", rhExpr))
		}
		rhs[i] = rh
	}

	var typ *TypeInfo

	if node.Type != nil {
		// Every Rh must be assignable to the type.
		typ = tc.checkType(node.Type)
		for i := range rhs {
			tc.mustBeAssignableTo(node.Rhs[i], typ.Type, false, nil)
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
		if rh.Untyped() && typ == nil {
			constTi.Properties |= PropertyUntyped
		}
		tc.assignScope(node.Lhs[i].Name, constTi, node.Lhs[i])

	}

}

// checkGenericAssignmentNode type checks an *ast.Assignment node. This method
// is necessary because the *ast.Assignment node of the Scriggo AST represents
// short variable declarations, assignments, short assignments and inc-dec
// statements.
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

// checkIncDecStatement checks an IncDec statement.
func (tc *typechecker) checkIncDecStatement(node *ast.Assignment) {

	if node.Type != ast.AssignmentIncrement && node.Type != ast.AssignmentDecrement {
		panic("BUG: expected an IncDec statement")
	}

	lh := tc.checkExpr(node.Lhs[0])
	switch {
	case lh.Addressable():
		// Ok!
	case tc.isMapIndexing(node.Lhs[0]):
		// Ok!
	default:
		if tc.isSelectorOfMapIndexing(node.Lhs[0]) {
			panic(tc.errorf(node.Lhs[0], "cannot assign to struct field %v in map", node.Lhs[0]))
		}
		panic(tc.errorf(node.Lhs[0], "cannot assign to %v", node.Lhs[0]))
	}

	if !isNumeric(lh.Type.Kind()) {
		panic(tc.errorf(node, "invalid operation: %v (non-numeric type %s)", node, lh))
	}

	tc.convertNodeForTheEmitter(node)

}

// checkNonNamesOnLeft panic a type checking error if the short declaration node
// contains a non-name on the left side.
func (tc *typechecker) checkNonNamesOnLeft(node *ast.Assignment) {
	if node.Type != ast.AssignmentDeclaration {
		panic("BUG: expected a short variable declaration")
	}
	for _, lhExpr := range node.Lhs {
		_, isIdent := lhExpr.(*ast.Identifier)
		if !isIdent {
			panic(tc.errorf(node, "non-name %s on left side of :=", lhExpr))
		}
	}
}

// checkRepeatedLhs panics a type checking error the left side of the given node
// contains repeated identifiers.
func (tc *typechecker) checkRepeatedLhs(node *ast.Assignment) {
	if node.Type != ast.AssignmentDeclaration {
		panic("BUG: expecting a := node")
	}
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

// checkShortVariableDeclaration type checks a short variable declaration.
func (tc *typechecker) checkShortVariableDeclaration(node *ast.Assignment) {

	if node.Type != ast.AssignmentDeclaration {
		panic("BUG: expected a short variable declaration")
	}

	// In case of unbalanced short variable declarations a 'fake' rhs must be
	// used for the type checking, but the tree must not be changed.
	nodeRhs := node.Rhs
	if len(node.Lhs) != len(nodeRhs) {
		nodeRhs = tc.rebalanceRightSide(node)
	}

	tc.checkNonNamesOnLeft(node)
	tc.checkRepeatedLhs(node)

	// Type check the right side of :=.
	rhs := make([]*TypeInfo, len(nodeRhs))
	for i, rhExpr := range nodeRhs {
		rhs[i] = tc.checkExpr(rhExpr)
	}

	// Discriminate already declared variables from new variables.
	isAlreadyDeclared := map[ast.Expression]bool{}
	for _, lhExpr := range node.Lhs {
		name := lhExpr.(*ast.Identifier).Name
		if tc.declaredInThisBlock(name) || isBlankIdentifier(lhExpr) {
			isAlreadyDeclared[lhExpr] = true
		}
	}
	if len(node.Lhs) == len(isAlreadyDeclared) {
		if tc.opts.SyntaxType == ScriptSyntax && tc.isScriptFuncDecl {
			panic(tc.errorf(node, "%v already declared in script", node.Lhs[0]))
		}
		panic(tc.errorf(node, "no new variables on left side of :="))
	}

	// Declare or redeclare variables on the left side of :=.
	for i := range node.Lhs {

		rh := rhs[i]

		if isBlankIdentifier(node.Lhs[i]) {
			tc.mustBeAssignableTo(nodeRhs[i], rh.Type, false, nil)
			rh.setValue(nil)
			continue
		}

		if isAlreadyDeclared[node.Lhs[i]] {
			lh := tc.checkIdentifier(node.Lhs[i].(*ast.Identifier), false)
			tc.mustBeAssignableTo(nodeRhs[i], lh.Type, false, nil)
			rh.setValue(lh.Type)
			continue
		}

		// New variable.
		if rh.Nil() {
			panic(tc.errorf(nodeRhs[i], "use of untyped nil"))
		}
		if rh.IsUntypedConstant() {
			tc.mustBeAssignableTo(nodeRhs[i], rh.Type, false, nil)
		}
		tc.declareVariable(node.Lhs[i].(*ast.Identifier), rh.Type)
		rh.setValue(nil)
	}

}

// checkVariableDeclaration type checks a variable declaration.
func (tc *typechecker) checkVariableDeclaration(node *ast.Var) {

	// In case of unbalanced var declarations a 'fake' rhs must be used for the
	// type checking, but the tree must not be changed.
	nodeRhs := node.Rhs
	if len(nodeRhs) > 0 && len(node.Lhs) != len(nodeRhs) {
		nodeRhs = tc.rebalanceRightSide(node)
	}

	// Type check expressions on the right side.
	rhs := make([]*TypeInfo, len(nodeRhs))
	for i := range nodeRhs {
		rhs[i] = tc.checkExpr(nodeRhs[i])
	}

	// 'var' declaration with explicit type: every rh must be assignable to
	// that type.
	var typ *TypeInfo
	if node.Type != nil {
		typ = tc.checkType(node.Type)
		for i := range rhs {
			tc.mustBeAssignableTo(nodeRhs[i], typ.Type, len(node.Lhs) != len(node.Rhs), node.Lhs[i])
		}
	}

	// Type is not specified, so there can't be an untyped nil in the right
	// side expression; more than this, every untyped constant must be
	// representable by it's default type.
	if typ == nil {
		for i, rh := range rhs {
			if rh.Nil() {
				panic(tc.errorf(nodeRhs[i], "use of untyped nil"))
			}
			if rh.IsUntypedConstant() {
				tc.mustBeAssignableTo(nodeRhs[i], rh.Type, false, nil)
			}
		}
	}

	// Variable declaration with no expressions: the zero of the explicit type
	// must be assigned to the left identifiers.
	if len(node.Rhs) == 0 {
		node.Rhs = make([]ast.Expression, len(node.Lhs))
		for i := 0; i < len(node.Lhs); i++ {
			node.Rhs[i] = tc.newPlaceholderFor(typ.Type)
			rhs = append(rhs, tc.checkExpr(node.Rhs[i]))
		}
	}

	// Declare the left hand identifiers and update the type infos.
	for i, rh := range rhs {

		// Determine the type of the new variable.
		var varTyp reflect.Type
		if typ == nil {
			varTyp = rh.Type
		} else {
			varTyp = typ.Type
		}

		// Set the type info of the right operand.
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

// TODO: this implementation is wrong. It has been kept to make the tests pass,
// but it must be changed:
//
//              m[f()] ++
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

// declareVariable declares the variable lh in the current block/scope with the
// given type. Note that a variabile declaration may come from both 'var'
// statements and short variable declaration statements.
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

// mustBeAssignableTo panics a type checking error if the type info associated
// to rhExpr is not assignable to the given type.
func (tc *typechecker) mustBeAssignableTo(rhExpr ast.Expression, typ reflect.Type, isMultipleAssignment bool, multipleAssignmentLh ast.Expression) {
	rh := tc.typeInfos[rhExpr]
	err := tc.isAssignableTo(rh, rhExpr, typ)
	if err != nil {
		if isMultipleAssignment {
			panic(tc.errorf(rhExpr, "cannot assign %v to %v (type %v) in multiple assignment", rh.Type, multipleAssignmentLh, typ))
		}
		if strings.HasPrefix(err.Error(), "constant ") {
			panic(tc.errorf(rhExpr, err.Error()))
		}
		if nilErr, ok := err.(nilConvertionError); ok {
			panic(tc.errorf(rhExpr, "cannot use nil as type %s in assignment", nilErr.typ))
		}
		panic(tc.errorf(rhExpr, "%s in assignment", err))
	}
}

// newPlaceholder returns a new Placeholder node with an associated type info
// that holds the zero for the given type.
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

// rebalanceRightSide rebalances the given node by returning a slice of
// Expressions that must be used as right side in the type checking of the given
// node.
// rebalanceRightSide panics a type checking error if the given node cannot be
// rebalanced.
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

	if len(nodeLhs) == len(nodeRhs) {
		panic("BUG: this method must be called only for multiple assignments")
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
