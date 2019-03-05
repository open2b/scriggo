// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"
	"scrigo/ast"
)

// checkAssignment checks the assignment node.
func (tc *typechecker) checkAssignment(node ast.Node) {

	var variables, values []ast.Expression
	var typ *ast.TypeInfo
	var isDeclaration, isConst bool

	switch n := node.(type) {

	case *ast.Var:

		variables = make([]ast.Expression, len(n.Identifiers))
		for i, ident := range n.Identifiers {
			variables[i] = ident
		}
		values = n.Values
		isDeclaration = true
		typ = tc.checkType(n.Type, noEllipses)

	case *ast.Const:

		variables = make([]ast.Expression, len(n.Identifiers))
		for i, ident := range n.Identifiers {
			variables[i] = ident
		}
		values = n.Values
		isConst = true
		isDeclaration = true
		typ = tc.checkType(n.Type, noEllipses)

	case *ast.Assignment:

		variables = n.Variables
		values = n.Values
		isDeclaration = n.Type == ast.AssignmentDeclaration
		// TODO (Gianluca): ast.Assignment does not have a type field (Type
		// indicates the type of the assignment itself, not the type of the
		// values). typ = tc.checkType(...?, noEllipses)

	default:

		panic(fmt.Errorf("bug: unexpected node %T", node))

	}

	if len(variables) == 1 && len(values) == 1 {
		tc.assignValueToVariable(node, variables[0], values[0], typ, isDeclaration, isConst)
		return
	}

	// { "var" IdentifierList Type . }
	if len(values) == 0 && typ != nil {
		for i := range variables {
			zero := (ast.Expression)(nil) // TODO (Gianluca): zero must contain the zero of type "typ".
			if isConst || isDeclaration {
				panic("bug?") // TODO (Gianluca): review!
			}
			tc.assignValueToVariable(node, variables[i], zero, typ, false, false)
		}
		return
	}

	if len(variables) == len(values) {
		for i := range variables {
			// TODO (Gianluca): if all variables have already been declared
			// previously, a declaration must end with an error.
			// _, alreadyDefined := tc.LookupScope("variable name", true) // TODO (Gianluca): to review.
			alreadyDefined := false
			isDecl := isDeclaration && !alreadyDefined
			tc.assignValueToVariable(node, variables[i], values[i], typ, isDecl, isConst)
		}
		return
	}

	if len(variables) >= 3 && len(values) == 1 {
		call, ok := values[0].(*ast.Call)
		if ok {
			tc.checkAssignmentWithCall(node, variables, call, typ, isDeclaration, isConst)
			return
		}
	}

	if len(variables) == 2 && len(values) == 1 {
		switch value := values[0].(type) {

		case *ast.Call:

			tc.checkAssignmentWithCall(node, variables, value, typ, isDeclaration, isConst)
			return

		case *ast.TypeAssertion:

			// TODO (Gianluca):
			// tc.checkTypeAssertion(value)
			// tc.assignValueToVariable(node, variable[0], typeAssertionType, nil, isDeclaration, isConst)
			// tc.assignValueToVariable(node, variable[1], boolTi, nil, isDeclaration, isConst)
			return

		case *ast.Index:

			// TODO (Gianluca):
			// tc.checkMapIndexint(value)
			// tc.assignValueToVariable(node, variable[0], mapType, nil, isDeclaration, isConst)
			// tc.assignValueToVariable(node, variable[1], boolTi, nil, isDeclaration, isConst)
			return

		}
	}

	panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(variables), len(values)))
}

// assignValueToVariable generically assigns value to variable. node must
// contain the assignment node (or the var/const declaration node) and it's used
// for error messages only. If the declaration specified a type, that must be
// passed as "typ" argument. isDeclaration and isConst indicates, respectively,
// if the assignment is a declaration and if it's a constant.
//
// TODO (Gianluca): handle "isConst"
//
// TODO (Gianluca): typ doesn't get the type's zero, just checks if type is
// correct when a value is provided. Implement "var a int"
//
// TODO (Gianluca):when assigning a costant to a value in scope, constant isn't
// constant anymore.
//
func (tc *typechecker) assignValueToVariable(node ast.Node, variable, value ast.Expression, typ *ast.TypeInfo, isDeclaration, isConst bool) {

	if isConst && !isDeclaration {
		panic("bug: cannot have a constant assignment outside a declaration")
	}

	valueTi := tc.checkExpression(value)

	// If it's a constant declaration, a constant value must be provided.
	if isConst && (valueTi.Constant == nil) {
		panic(tc.errorf(node, "const initializer %s is not a constant", value))
	}

	// If a type is provided, value must be assignable to type.
	if typ != nil && !tc.isAssignableTo(valueTi, typ.Type) {
		panic(tc.errorf(node, "canont use %v (type %v) as type %v in assignment", value, valueTi, typ))
	}

	switch v := variable.(type) {

	case *ast.Identifier:

		_, alreadyInCurrentScope := tc.LookupScope(v.Name, true)

		// Cannot declarate a variable if already exists in current scope.
		if isDeclaration && alreadyInCurrentScope {
			panic(tc.errorf(node, "no new variables on left side of :="))
		}

		// If it's not a declaration, variable must already exists in some
		// scope. Its type must be retrieved, and value must be assignable
		// to that.
		variableTi := tc.checkExpression(variable)
		if !isDeclaration {
			if !tc.isAssignableTo(valueTi, variableTi.Type) {
				panic(tc.errorf(node, "cannot use %v (type %v) as type %v in assignment", value, valueTi.Type, variableTi.Type))
			}
		}

		newValueTi := &ast.TypeInfo{}
		if isDeclaration {
			defaultType := tc.concreteType(valueTi)
			newValueTi.Type = defaultType
		}
		tc.AssignScope(v.Name, newValueTi)

	default:
		panic("bug/not implemented") // TODO (Gianluca): can we have a declaration without an identifier?
	}
	return
}

// checkAssignmentWithCall checks an assignment where left value is a function
// call or a convertion.
//
// TODO (Gianluca): handle builtin functions.
//
func (tc *typechecker) checkAssignmentWithCall(node ast.Node, variables []ast.Expression, call *ast.Call, typ *ast.TypeInfo, isDeclaration, isConst bool) {
	values := tc.checkCallExpression(call, false) // TODO (Gianluca): is "false" correct?
	if len(variables) != len(values) {
		panic(tc.errorf(node, "assignment mismatch: %d variables but %v returns %d values", len(variables), call, len(values)))
	}
	for i := range variables {
		// TODO (Gianluca): replace the second "variables[i]" with "values[i]"
		tc.assignValueToVariable(node, variables[i], variables[i], typ, isDeclaration, isConst)
	}
}
