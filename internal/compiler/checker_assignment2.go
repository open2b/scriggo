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
	// TODO: this method can be renamed/removed or implemented using an existing
	// type checking function.
	panic("not implemented")
}

func (tc *typechecker) declareConstant(name string, typ reflect.Type, value constant) {
	// TODO: this method can be renamed/removed or implemented using an existing
	// type checking function.
	panic("not implemented")
}

// checkLhsRhs takes a simple assignment node, a short declaration node or a
// variable declaration node and returns the lists of the type infos for the
// left and the right sides of the node. This methods also handles "unbalanced"
// nodes where there is just one value on the right and more than one value on
// the left. If the number of elements on the right side does not match with the
// number of elements on the left, checkLhsRhs panics with an "assignment
// mismatch" error.
func (tc *typechecker) checkLhsRhs(node ast.Node) ([]*TypeInfo, []*TypeInfo) {

	// TODO: check that type is correct.
	switch node := node.(type) {
	case *ast.Assignment:
		switch node.Type {
		case ast.AssignmentSimple:
		case ast.AssignmentDeclaration:
		default:
			panic("BUG: expecting a simple assignment, a short declaration or a variable declaration")
		}
	case *ast.Var:
	default:
		panic("BUG: expecting a simple assignment, a short declaration or a variable declaration")
	}

	// TODO: this method can be renamed/removed or implemented using an existing
	// type checking function.
	panic("not implemented")
}

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

		tc.declareConstant(node.Lhs[i].Name, constType, constValue)

	}

}

// checkVariableDeclaration type checks a variable declaration.
// See https://golang.org/ref/spec#Variable_declarations.
func (tc *typechecker) checkVariableDeclaration(node *ast.Var) {

	lhs, rhs := tc.checkLhsRhs(node)

	// Every Lh identifier must not be defined in the current block.
	for _, lhIdent := range node.Lhs {
		if decl, ok := tc.declaredInThisBlock(lhIdent.Name); ok {
			_ = decl                              // TODO
			panic("..redeclared in this block..") // TODO
		}
	}

}

// checkShortVariableDeclarations type checks a short variable declaration.
// See https://golang.org/ref/spec#Short_variable_declarations.
func (tc *typechecker) checkShortVariableDeclarations(node *ast.Assignment) {

	// Check that node is a short variable declaration.
	if node.Type != ast.AssignmentDeclaration {
		panic("BUG: expected a short variable declaration")
	}

}

// See https://golang.org/ref/spec#Assignments.
// checkAssignments type check an assignment node.
func (tc *typechecker) checkAssignments(node *ast.Assignment) {

	// Check that node is an assignment node.
	switch node.Type {
	case ast.AssignmentDeclaration, ast.AssignmentIncrement, ast.AssignmentDecrement:
		panic("BUG: expected an assignment node")
	}

}

// checkIncDecStatements checks an IncDec statement.
// See https://golang.org/ref/spec#IncDec_statements.
func (tc *typechecker) checkIncDecStatements(node *ast.Assignment) {

	// Check that node is an IncDec statement.
	if node.Type != ast.AssignmentIncrement && node.Type != ast.AssignmentDecrement {
		panic("BUG: expected an IncDec statement")
	}

}
