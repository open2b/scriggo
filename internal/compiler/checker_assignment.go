// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"

	"scriggo/ast"
)

// declaredInThisBlock reports whether name is declared in this block. If so
// then true and the ast.Node where name is declared are returned. Otherwise, a
// nil ast.Node and false are returned.
func (tc *typechecker) declaredInThisBlock(name string) (ast.Node, bool) {
	scopeElem, ok := tc.lookupScopesElem(name, true)
	if ok {
		return scopeElem.decl, true
	}
	return nil, false
}

func (tc *typechecker) declareConstant(lhNode *ast.Identifier, typ reflect.Type, value constant, untyped bool) {
	ti := &TypeInfo{
		Type:     typ,
		Constant: value,
	}
	if untyped {
		ti.Properties |= PropertyUntyped
	}
	tc.assignScope(lhNode.Name, ti, lhNode)
}

func (tc *typechecker) declareVariable(lh *ast.Identifier, typ reflect.Type) {
	if _, ok := tc.declaredInThisBlock(lh.Name); ok {
		panic(tc.errorf(lh, "declared in this block..")) // TODO
	}
	ti := &TypeInfo{
		Type:       typ,
		Properties: PropertyAddressable,
	}
	tc.typeInfos[lh] = ti
	tc.assignScope(lh.Name, ti, lh)
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

// // checkLhsRhs takes a simple assignment node, a short declaration node or a
// // variable declaration node and returns the lists of the type infos for the
// // left and the right sides of the node. This methods also handles "unbalanced"
// // nodes where there is just one value on the right and more than one value on
// // the left. If the number of elements on the right side does not match with the
// // number of elements on the left, checkLhsRhs panics with an "assignment
// // mismatch" error.
// //
// // TODO: if checkLhsRhs does not modify the source node, then it's illegal to
// // access from the outside to node.Rhs[i] because the node could be unbalanced.
// //
// func (tc *typechecker) checkLhsRhs(node ast.Node) ([]*TypeInfo, []*TypeInfo) {

// 	var lhsExpr, rhsExpr []ast.Expression

// 	// TODO: check that type is correct.
// 	switch node := node.(type) {
// 	case *ast.Assignment:
// 		switch node.Type {
// 		case ast.AssignmentSimple, ast.AssignmentDeclaration:
// 			lhsExpr = node.Lhs
// 			rhsExpr = node.Rhs
// 		default:
// 			panic("BUG: expecting a simple assignment, a short declaration or a variable declaration")
// 		}
// 	case *ast.Var:
// 		for _, lhExpr := range node.Lhs {
// 			lhsExpr = append(lhsExpr, lhExpr)
// 		}
// 		rhsExpr = node.Rhs
// 	default:
// 		panic("BUG: expecting a simple assignment, a short declaration or a variable declaration")
// 	}

// 	if len(lhsExpr) != len(rhsExpr) {
// 		panic("not implemented")
// 	}

// 	var lhs, rhs []*TypeInfo

// 	for _, lhExpr := range lhsExpr {
// 		lh := tc.checkExpr(lhExpr)
// 		lhs = append(lhs, lh)
// 	}

// 	for _, rhExpr := range rhsExpr {
// 		rh := tc.checkExpr(rhExpr)
// 		rhs = append(rhs, rh)
// 	}

// 	return lhs, rhs

// }

// checkConstantDeclaration type checks a constant declaration.
// See https://golang.org/ref/spec#Constant_declarations.
func (tc *typechecker) checkConstantDeclaration(node *ast.Const) {

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
			panic(tc.errorf(node, "const initializer %s is not a constant", rh))
		}
		rhs = append(rhs, rh)
	}

	// Every Lh identifier must not be defined in the current block.
	for _, lhIdent := range node.Lhs {
		if decl, ok := tc.declaredInThisBlock(lhIdent.Name); ok {
			_ = decl                              // TODO
			panic("..redeclared in this block..") // TODO
		}
	}

	var typ *TypeInfo

	if node.Type != nil {
		// Every Rh must be assignable to the type.
		typ = tc.checkType(node.Type)
		for i := range rhs {
			err := tc.isAssignableTo(rhs[i], node.Rhs[i], typ.Type)
			if err != nil {
				panic("not assignable") // TODO
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

		tc.declareConstant(node.Lhs[i], constType, constValue, rh.Untyped())

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

// checkVariableDeclaration type checks a variable declaration.
// See https://golang.org/ref/spec#Variable_declarations.
func (tc *typechecker) checkVariableDeclaration(node *ast.Var) {

	var rhs []*TypeInfo

	if len(node.Rhs) > 0 && len(node.Lhs) != len(node.Rhs) {
		panic("not implemented")
	}

	for _, rhExpr := range node.Rhs {
		rh := tc.checkExpr(rhExpr)
		rhs = append(rhs, rh)
	}

	var typ *TypeInfo

	if node.Type != nil {
		// Every Rh must be assignable to the type.
		typ = tc.checkType(node.Type)
		for i := range rhs {
			err := tc.isAssignableTo(rhs[i], node.Rhs[i], typ.Type)
			if err != nil {
				panic("not assignable") // TODO
			}
		}
	}

	// If typ is not specified, none of the Rh values must be the predeclared
	// nil.
	if typ == nil {
		for i, rh := range rhs {
			if rh.Nil() {
				panic(tc.errorf(node.Rhs[i], "use of untyped nil"))
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
		var varTyp reflect.Type
		if typ == nil {
			varTyp = rh.Type
		} else {
			varTyp = typ.Type
		}
		rh.setValue(varTyp)
		tc.declareVariable(node.Lhs[i], varTyp)
	}

}

// checkShortVariableDeclaration type checks a short variable declaration. See
// https://golang.org/ref/spec#Short_variable_declarations.
func (tc *typechecker) checkShortVariableDeclaration(node *ast.Assignment) {

	// Check that node is a short variable declaration.
	if node.Type != ast.AssignmentDeclaration {
		panic("BUG: expected a short variable declaration")
	}

	for _, lhExpr := range node.Lhs {
		_, isIdent := lhExpr.(*ast.Identifier)
		if !isIdent {
			panic("non-name .. on left side of :=") // TODO
		}
	}

	var rhs []*TypeInfo
	for _, rhExpr := range node.Rhs {
		rh := tc.checkExpr(rhExpr)
		rhs = append(rhs, rh)
	}

	var lhsToDeclare, lhsToRedeclare []ast.Expression

	for _, lhExpr := range node.Lhs {
		name := lhExpr.(*ast.Identifier).Name
		if _, ok := tc.declaredInThisBlock(name); ok {
			lhsToRedeclare = append(lhsToRedeclare, lhExpr)
		} else {
			lhsToDeclare = append(lhsToDeclare, lhExpr)
		}
	}

	if len(lhsToDeclare) == 0 {
		panic(tc.errorf(node, "no new variables on left side of :="))
	}

	for i, rh := range rhs {
		var varTyp reflect.Type
		varTyp = rh.Type
		rh.setValue(varTyp)
		tc.declareVariable(node.Lhs[i].(*ast.Identifier), varTyp)
	}

	// for i, lh := range lhs {
	// 	switch {
	// 	case lh.Addressable():
	// 		// Ok!
	// 	case isBlankIdentifier(node.Lhs[i]):
	// 		// Ok!
	// 	default:
	// 		panic("cannot assign to ..") // TODO
	// 	}
	// 	err := tc.isAssignableTo(rhs[i], node.Rhs[i], lh.Type)
	// 	if err != nil {
	// 		panic("not assignable") // TODO
	// 	}
	// }

	// TODO

}

// See https://golang.org/ref/spec#Assignments.
// checkAssignments type check an assignment node.
func (tc *typechecker) checkAssignment(node *ast.Assignment) {

	// Check that node is an assignment node.
	switch node.Type {
	case ast.AssignmentDeclaration, ast.AssignmentIncrement, ast.AssignmentDecrement:
		panic("BUG: expected an assignment node")
	}

	var lhs, rhs []*TypeInfo
	for _, lhExpr := range node.Lhs {
		lh := tc.checkExpr(lhExpr)
		lhs = append(lhs, lh)
	}
	for _, rhExpr := range node.Rhs {
		rh := tc.checkExpr(rhExpr)
		rhs = append(rhs, rh)
	}

	if op := node.Type; ast.AssignmentAddition <= op && op <= ast.AssignmentRightShift {
		if len(lhs) != 1 {
			panic("...") // TODO
		}
		if len(rhs) != 1 {
			panic("...") // TODO
		}
		if isBlankIdentifier(node.Lhs[0]) {
			panic("...") // TODO
		}
	}

	for i, lh := range lhs {
		switch {
		case lh.Addressable():
			// Ok!
		case tc.isMapIndexExpression(node.Lhs[i]):
			// Ok!
		case isBlankIdentifier(node.Lhs[i]):
			// Ok!
		default:
			panic("not assignable") // TODO
		}
	}

	for i, rh := range rhs {
		err := tc.isAssignableTo(rh, node.Rhs[i], lhs[i].Type)
		if err != nil {
			panic("not assignable") // TODO
		}
	}

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
	case isBlankIdentifier(node.Lhs[0]):
		// Ok!
	default:
		panic("cannot assign to..") // TODO
	}

	// TODO: the untyped constant '1' must be assignable to the type of lh.

}
