// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"strings"

	"github.com/open2b/scriggo/ast"
)

// typeInfoPair represents a pair of type infos. It is used checking
// assignments, variable and constant declarations. The first element of a
// typeInfoPair value may be not nil only for the type info of a default
// expression.
type typeInfoPair [2]*typeInfo

// TypeInfo returns the type info of the expression that p represents.
func (p typeInfoPair) TypeInfo() *typeInfo {
	if p[0] == nil {
		return p[1]
	}
	return p[0]
}

// checkAssignment type checks an assignment node with the operator '='.
func (tc *typechecker) checkAssignment(node *ast.Assignment) {

	if node.Type != ast.AssignmentSimple {
		panic("BUG: expected an assignment node with an '=' operator")
	}

	// In case of unbalanced assignments a 'fake' rhs must be used for the type
	// checking, but the tree must not be changed.
	nodeRhs := node.Rhs

	if len(node.Lhs) != len(nodeRhs) {
		nodeRhs = tc.rebalancedRightSide(node)
	}

	// Type check the left side.
	lhs := make([]*typeInfo, len(node.Lhs))
	for i, lhExpr := range node.Lhs {
		if isBlankIdentifier(lhExpr) {
			continue
		}
		// Determine the Lh type info.
		var lh *typeInfo
		if ident, ok := lhExpr.(*ast.Identifier); ok {
			lh = tc.checkIdentifier(ident, false) // Use checkIdentifier to avoid marking 'ident' as used.
		} else {
			lh = tc.checkExpr(lhExpr)
		}
		tc.checkAssignTo(lh, lhExpr)
		lhs[i] = lh
	}

	// Type check the right side.
	rhs := make([]typeInfoPair, len(nodeRhs))
	for i, expr := range nodeRhs {
		rhs[i] = tc.checkExpr2(expr, false)
	}

	// Check for assignability and update the type infos.
	for i, rh := range rhs {
		for j, ti := range rh {
			if ti == nil {
				continue
			}
			if isBlankIdentifier(node.Lhs[i]) {
				if ti.Nil() { // _ = nil
					panic(tc.errorf(node.Lhs[i], "use of untyped nil"))
				}
				if ti.Untyped() {
					tc.mustBeAssignableTo(ti, subExpr(nodeRhs[i], j == 0), ti.Type, false, nil)
				}
				continue
			}
			tc.mustBeAssignableTo(ti, subExpr(nodeRhs[i], j == 0), lhs[i].Type, len(node.Lhs) != len(node.Rhs), node.Lhs[i])
		}
		ti := rh.TypeInfo()
		if isBlankIdentifier(node.Lhs[i]) {
			ti.setValue(nil)
			continue
		}
		// Update the type info for the emitter.
		if ti.Nil() {
			tc.compilation.typeInfos[nodeRhs[i]] = tc.nilOf(lhs[i].Type)
		} else {
			ti.setValue(nil)
		}
	}

}

// checkAssignmentOperation type checks an assignment operation x op= y.
// See https://golang.org/ref/spec#assign_op and
// https://golang.org/ref/spec#Assignments.
func (tc *typechecker) checkAssignmentOperation(node *ast.Assignment) {

	if !(ast.AssignmentAddition <= node.Type && node.Type <= ast.AssignmentRightShift) {
		panic("BUG: expected an assignment operation")
	}

	lh := tc.checkExpr(node.Lhs[0])
	rh := tc.checkExpr(node.Rhs[0])

	tc.checkAssignTo(lh, node.Lhs[0])

	op := operatorFromAssignmentType(node.Type)
	_, err := tc.binaryOp(node.Lhs[0], op, node.Rhs[0])
	if err != nil {
		if err == errDivisionByZero {
			panic(tc.errorf(node, "%s", err))
		}
		panic(tc.errorf(node, "invalid operation: %s (%s)", node, err))
	}

	rh.setValue(lh.Type)
}

// checkBalancedConstantDeclaration type checks that a constant declaration is
// balanced, ie that it has the same number of variables and values.
func (tc *typechecker) checkBalancedConstantDeclaration(node *ast.Const) {

	if len(node.Lhs) > len(node.Rhs) {
		pos := node.Lhs[0].Pos()
		if len(node.Rhs) > 0 {
			pos = node.Rhs[len(node.Rhs)-1].Pos()
		}
		panic(tc.errorf(pos, "missing value in const declaration"))
	}

	if len(node.Lhs) < len(node.Rhs) {
		panic(tc.errorf(node.Rhs[len(node.Rhs)-1], "extra expression in const declaration"))
	}

}

// checkConstantDeclaration type checks a constant declaration.
func (tc *typechecker) checkConstantDeclaration(node *ast.Const) {

	tc.checkBalancedConstantDeclaration(node)

	tc.iota = node.Index

	// Type check every Rh expression: they must be constant.
	rhs := make([]typeInfoPair, len(node.Rhs))
	for i, rhExpr := range node.Rhs {
		rh := tc.checkExpr2(rhExpr, false)
		for j, ti := range rh {
			if ti != nil && !ti.IsConstant() {
				if ti.Nil() {
					panic(tc.errorf(node.Lhs[i], "const initializer cannot be nil"))
				}
				panic(tc.errorf(node.Lhs[i], "const initializer %s is not a constant", subExpr(rhExpr, j == 0)))
			}
		}
		rhs[i] = rh
	}

	tc.iota = -1

	var typ *typeInfo

	if node.Type == nil {
		for i, rh := range rhs {
			if rh[0] == nil {
				continue
			}
			if !rh[1].Untyped() {
				tc.mustBeAssignableTo(rh[0], subExpr(node.Rhs[i], true), rh[1].Type, false, nil)
				continue
			}
			if !rh[0].Untyped() {
				panic(tc.errorf(node.Lhs[i], "cannot use typed const %s in untyped const initializer", subExpr(node.Rhs[i], true)))
			}
			if rh[0].Type != rh[1].Type {
				panic(tc.errorf(node.Lhs[i], "mismatched kinds %s and %s in untyped const initializer",
					constantKindName[rh[0].Type.Kind()], constantKindName[rh[1].Type.Kind()]))
			}
		}
	} else {
		typ = tc.checkType(node.Type)
		switch typ.Type.Kind() {
		case reflect.Array, reflect.Chan, reflect.Func,
			reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice,
			reflect.Struct, reflect.UnsafePointer:
			panic(tc.errorf(node.Lhs[0], "invalid constant type %s", typ))
		}
		// Every Rh must be assignable to the type.
		for i, rh := range rhs {
			for j, ti := range rh {
				if ti != nil {
					tc.mustBeAssignableTo(ti, subExpr(node.Rhs[i], j == 0), typ.Type, false, nil)
				}
			}
		}
	}

	for i, rh := range rhs {

		if isBlankIdentifier(node.Lhs[i]) {
			continue
		}

		ti := rh.TypeInfo()

		// Prepare the constant value.
		constValue := ti.Constant

		// Prepare the constant type.
		var constType reflect.Type
		if typ == nil {
			constType = ti.Type
		} else {
			constType = typ.Type
		}

		// Declare the constant in the current block/scope.
		constTi := &typeInfo{Type: constType, Constant: constValue}
		if ti.Untyped() && typ == nil {
			constTi.Properties = propertyUntyped
		}
		tc.assignScope(node.Lhs[i].Name, constTi, node.Lhs[i])

	}

}

// checkGenericAssignmentNode calls the appropriate type checker method based on
// what the given node represents: a short variable declaration, an assignment,
// a short assignment or an inc-dec statement.
func (tc *typechecker) checkGenericAssignmentNode(node *ast.Assignment) {
	switch node.Type {
	case ast.AssignmentDeclaration:
		tc.checkShortVariableDeclaration(node)
	case ast.AssignmentIncrement, ast.AssignmentDecrement:
		tc.checkIncDecStatement(node)
	case ast.AssignmentSimple:
		tc.checkAssignment(node)
	default:
		tc.checkAssignmentOperation(node)
	}
}

// checkIncDecStatement checks an IncDec statement.
func (tc *typechecker) checkIncDecStatement(node *ast.Assignment) {

	if node.Type != ast.AssignmentIncrement && node.Type != ast.AssignmentDecrement {
		panic("BUG: expected an IncDec statement")
	}

	lh := tc.checkExpr(node.Lhs[0])
	tc.checkAssignTo(lh, node.Lhs[0])

	if !isNumeric(lh.Type.Kind()) {
		panic(tc.errorf(node, "invalid operation: %s (non-numeric type %s)", node, lh))
	}

	// Transform the tree for the emitter.
	if node.Type == ast.AssignmentIncrement {
		node.Type = ast.AssignmentAddition
	} else {
		node.Type = ast.AssignmentSubtraction
	}
	rhExpr := ast.NewBasicLiteral(nil, ast.IntLiteral, "1")
	rh := tc.checkExpr(rhExpr)
	rh.setValue(lh.Type)
	node.Rhs = append(node.Rhs, rhExpr)

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
		nodeRhs = tc.rebalancedRightSide(node)
	}

	// Check that every operand on the left is an identifier.
	for _, lhExpr := range node.Lhs {
		_, isIdent := lhExpr.(*ast.Identifier)
		if !isIdent {
			panic(tc.errorf(node, "non-name %s on left side of :=", lhExpr))
		}
	}

	// Check that the left side does not contain repeated names.
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

	// Type check the right side of :=.
	rhs := make([]typeInfoPair, len(nodeRhs))
	for i, rhExpr := range nodeRhs {
		rhs[i] = tc.checkExpr2(rhExpr, false)
	}

	// Discriminate already declared variables from new variables.
	isAlreadyDeclared := map[ast.Expression]bool{}
	for _, lhExpr := range node.Lhs {
		name := lhExpr.(*ast.Identifier).Name
		_, ok := tc.scopes.Current(name)
		if name == "_" || ok {
			isAlreadyDeclared[lhExpr] = true
		}
	}
	if len(node.Lhs) == len(isAlreadyDeclared) {
		panic(tc.errorf(node, "no new variables on left side of :="))
	}

	// Declare or redeclare variables on the left side of :=.
	for i := range node.Lhs {
		for j, ti := range rhs[i] {
			if ti == nil {
				continue
			}
			expr := subExpr(nodeRhs[i], j == 0)
			switch {
			case isBlankIdentifier(node.Lhs[i]):
				if ti.IsConstant() {
					tc.mustBeAssignableTo(ti, expr, ti.Type, false, nil)
					ti.setValue(nil)
				}
			case isAlreadyDeclared[node.Lhs[i]]:
				lh := tc.checkIdentifier(node.Lhs[i].(*ast.Identifier), false)
				tc.mustBeAssignableTo(ti, expr, lh.Type, false, nil)
				ti.setValue(lh.Type)
			case ti.Nil():
				panic(tc.errorf(expr, "use of untyped nil"))
			case j == 0 || ti.Untyped():
				tc.mustBeAssignableTo(ti, expr, rhs[i][1].Type, false, nil)
				fallthrough
			default:
				if j == 1 {
					tc.declareVariable(node.Lhs[i].(*ast.Identifier), ti.Type)
				}
				ti.setValue(nil)
			}
		}

	}

}

// checkVariableDeclaration type checks a variable declaration.
func (tc *typechecker) checkVariableDeclaration(node *ast.Var) {

	// Type check the explicit type, if present.
	var typ *typeInfo
	if node.Type != nil {
		typ = tc.checkType(node.Type)
	}

	// In case of unbalanced var declarations a 'fake' rhs must be used for the
	// type checking, but the tree must not be changed.
	nodeRhs := node.Rhs
	if len(nodeRhs) > 0 && len(node.Lhs) != len(nodeRhs) {
		nodeRhs = tc.rebalancedRightSide(node)
	}

	// Variable declaration with no expressions: the zero of the explicit type
	// must be assigned to the left identifiers.
	if len(nodeRhs) == 0 {
		nodeRhs = make([]ast.Expression, len(node.Lhs))
		for i := 0; i < len(node.Lhs); i++ {
			nodeRhs[i] = tc.newPlaceholderFor(typ.Type)
		}
		node.Rhs = nodeRhs // change the tree for the emitter.
	}

	// Type check expressions on the right side.
	rhs := make([]typeInfoPair, len(nodeRhs))
	for i, expr := range nodeRhs {
		rhs[i] = tc.checkExpr2(expr, false)
	}

	if typ == nil {
		// Type is not specified, so there can't be an untyped nil in the right
		// side expression; more than this, every untyped constant must be
		// representable by it's default type. In case of a default expression,
		// its left expression must be assignable to its right expression.
		for i, rh := range rhs {
			for j, ti := range rh {
				if ti == nil {
					continue
				}
				if ti.Nil() {
					panic(tc.errorf(subExpr(nodeRhs[i], j == 0), "use of untyped nil"))
				}
				if j == 0 || ti.Untyped() {
					tc.mustBeAssignableTo(ti, subExpr(nodeRhs[i], j == 0), rh[1].Type, false, nil)
				}
			}
		}
	} else {
		// 'var' declaration with explicit type: every rh must be assignable to
		// that type.
		typ = tc.checkType(node.Type)
		for i, rh := range rhs {
			for j, ti := range rh {
				if ti != nil {
					tc.mustBeAssignableTo(ti, subExpr(nodeRhs[i], j == 0), typ.Type, len(node.Lhs) != len(node.Rhs), node.Lhs[i])
				}
			}
		}
	}

	// Declare the left hand identifiers and update the type infos.
	for i, rh := range rhs {

		// Determine the type of the new variable.
		var varTyp reflect.Type
		if typ == nil {
			varTyp = rh[1].Type
		} else {
			varTyp = typ.Type
		}

		// Set the type info of the right operand.
		ti := rh.TypeInfo()
		if ti.Nil() {
			tc.compilation.typeInfos[nodeRhs[i]] = tc.nilOf(typ.Type)
		} else if ti.IsConstant() {
			ti.setValue(varTyp)
		}

		if isBlankIdentifier(node.Lhs[i]) {
			continue
		}

		tc.declareVariable(node.Lhs[i], varTyp)

	}

}

// declareVariable declares the variable lh in the current block/scope with the
// given type. Note that a variable declaration may come from both 'var'
// statements and short variable declaration statements.
func (tc *typechecker) declareVariable(lh *ast.Identifier, typ reflect.Type) {
	ti := &typeInfo{
		Type:       typ,
		Properties: propertyAddressable,
	}
	tc.compilation.typeInfos[lh] = ti
	tc.assignScope(lh.Name, ti, lh)
}

// checkAssignTo checks that it is possible to assign to the expression expr.
func (tc *typechecker) checkAssignTo(ti *typeInfo, expr ast.Expression) {
	if ti.Addressable() && !ti.IsMacroDeclaration() || tc.isMapIndexing(expr) {
		return
	}
	format := "cannot assign to %s"
	switch e := expr.(type) {
	case *ast.Selector:
		if tc.isMapIndexing(e.Expr) {
			format = "cannot assign to struct field %s in map"
		}
	case *ast.Index:
		ti := tc.compilation.typeInfos[e.Expr]
		if ti.Type.Kind() == reflect.String {
			format += " (strings are immutable)"
		}
	case *ast.Slicing:
		ti := tc.compilation.typeInfos[e.Expr]
		if ti.Type.Kind() == reflect.String {
			format += " (strings are immutable)"
		}
	case *ast.Identifier:
		if ti.IsConstant() {
			format += " (declared const)"
		}
	}
	panic(tc.errorf(expr, format, expr))
}

// mustBeAssignableTo ensures that the type info of rhExpr is assignable to the
// given type, otherwise panics. unbalanced reports whether the assignment is
// unbalanced and in this case unbalancedLh is its left expression that is
// assigned.
func (tc *typechecker) mustBeAssignableTo(rh *typeInfo, rhExpr ast.Expression, typ reflect.Type, unbalanced bool, unbalancedLh ast.Expression) {
	err := tc.isAssignableTo(rh, rhExpr, typ)
	if err != nil {
		if unbalanced {
			panic(tc.errorf(rhExpr, "cannot assign %s to %s (type %s) in multiple assignment", rh.Type, unbalancedLh, typ))
		}
		if strings.HasPrefix(err.Error(), "constant ") {
			panic(tc.errorf(rhExpr, err.Error()))
		}
		if nilErr, ok := err.(nilConversionError); ok {
			panic(tc.errorf(rhExpr, "cannot use nil as type %s in assignment", nilErr.typ))
		}
		panic(tc.errorf(rhExpr, "%s in assignment", err))
	}
}

// newPlaceholder returns a new Placeholder node with an associated type info
// that holds the zero for the given type.
func (tc *typechecker) newPlaceholderFor(typ reflect.Type) *ast.Placeholder {
	k := typ.Kind()
	var ti *typeInfo
	switch {
	case reflect.Int <= k && k <= reflect.Complex128:
		ti = &typeInfo{Type: typ, Constant: int64Const(0), Properties: propertyUntyped}
		ti.setValue(typ)
	case k == reflect.String:
		ti = &typeInfo{Type: typ, Constant: stringConst(""), Properties: propertyUntyped}
		ti.setValue(typ)
	case k == reflect.Bool:
		ti = &typeInfo{Type: typ, Constant: boolConst(false), Properties: propertyUntyped}
		ti.setValue(typ)
	case k == reflect.Interface, k == reflect.Func:
		ti = tc.nilOf(typ)
	default:
		ti = &typeInfo{Type: typ, value: tc.types.Zero(typ).Interface(), Properties: propertyHasValue}
		ti.setValue(typ)
	}
	ph := ast.NewPlaceholder()
	tc.compilation.typeInfos[ph] = ti
	return ph
}

// rebalancedRightSide returns the right side of the an unbalanced assignment or
// declaration, represented by node, rebalanced so the number of elements are
// the same of the left side.
// rebalancedRightSide type checks the returned right side, panicking if the
// type checking fails.
func (tc *typechecker) rebalancedRightSide(node ast.Node) []ast.Expression {

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

	rhExpr := nodeRhs[0]

	if call, ok := rhExpr.(*ast.Call); ok {
		tis := tc.checkCallExpression(call)
		if len(nodeLhs) != len(tis) {
			ti := tc.compilation.typeInfos[call.Func]
			if ti.IsBuiltinFunction() {
				panic(tc.errorf(node, "assignment mismatch: %d variables but %d values", len(nodeLhs), len(tis)))
			}
			panic(tc.errorf(node, "assignment mismatch: %d variables but %s returns %d values", len(nodeLhs), call, len(tis)))
		}
		rhsExpr := make([]ast.Expression, len(tis))
		for i, ti := range tis {
			rhsExpr[i] = ast.NewCall(call.Pos(), call.Func, call.Args, false)
			tc.compilation.typeInfos[rhsExpr[i]] = ti
		}
		return rhsExpr
	}

	if len(nodeLhs) == 2 && len(nodeRhs) == 1 {
		switch v := rhExpr.(type) {
		case *ast.TypeAssertion:
			v1 := ast.NewTypeAssertion(v.Pos(), v.Expr, v.Type)
			v2 := ast.NewTypeAssertion(v.Pos(), v.Expr, v.Type)
			ti := tc.checkExpr(rhExpr)
			tc.compilation.typeInfos[v1] = &typeInfo{Type: ti.Type}
			tc.compilation.typeInfos[v2] = untypedBoolTypeInfo
			return []ast.Expression{v1, v2}
		case *ast.Index:
			v1 := ast.NewIndex(v.Pos(), v.Expr, v.Index)
			v2 := ast.NewIndex(v.Pos(), v.Expr, v.Index)
			ti := tc.checkExpr(rhExpr)
			tc.compilation.typeInfos[v1] = &typeInfo{Type: ti.Type}
			tc.compilation.typeInfos[v2] = untypedBoolTypeInfo
			return []ast.Expression{v1, v2}
		case *ast.UnaryOperator:
			if v.Op == ast.OperatorReceive {
				v1 := ast.NewUnaryOperator(v.Pos(), ast.OperatorReceive, v.Expr)
				v2 := ast.NewUnaryOperator(v.Pos(), ast.OperatorReceive, v.Expr)
				ti := tc.checkExpr(rhExpr)
				tc.compilation.typeInfos[v1] = &typeInfo{Type: ti.Type}
				tc.compilation.typeInfos[v2] = untypedBoolTypeInfo
				return []ast.Expression{v1, v2}
			}
		}
	}

	panic(tc.errorf(node, "assignment mismatch: %d variables but %d values", len(nodeLhs), len(nodeRhs)))

}
