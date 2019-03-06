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
//
// TODO (Gianluca): check error checking order.
//
// TODO (Gianluca): "a, b, c := 1, 2, a": how does Go behave? Does it find "a"
// or evaluation order doesn't matter, so "a" is not declared?
//
func (tc *typechecker) checkAssignment(node ast.Node) {

	var vars, values []ast.Expression
	var typ *ast.TypeInfo
	var isDecl, isConst bool

	switch n := node.(type) {

	case *ast.Var:

		values = n.Values
		isDecl = true
		typ = tc.checkType(n.Type, noEllipses)

		// { "var" IdentifierList Type . }
		if len(values) == 0 && typ != nil {
			for i := range vars {
				zero := (ast.Expression)(nil) // TODO (Gianluca): zero must contain the zero of type "typ".
				if isConst || isDecl {
					panic("bug?") // TODO (Gianluca): review!
				}
				tc.assignSingle(node, vars[i], zero, typ, false, false)
			}
			return
		}

		if len(vars) == 1 && len(values) == 1 {
			tc.assignSingle(node, vars[0], values[0], typ, true, false)
			return
		}

		vars = make([]ast.Expression, len(n.Identifiers))
		for i, ident := range n.Identifiers {
			vars[i] = ident
		}

	case *ast.Const:

		values = n.Values
		isConst = true
		isDecl = true
		typ = tc.checkType(n.Type, noEllipses)

		if len(vars) == 1 && len(values) == 1 {
			tc.assignSingle(node, vars[0], values[0], typ, true, true)
			return
		}

		// The number of identifiers must be equal to the number of expressions.
		// [https://golang.org/ref/spec#Constant_declarations]
		if len(vars) != len(values) {
			panic("len(variables) != len(values)") // TODO (Gianluca): to review.
		}

		vars = make([]ast.Expression, len(n.Identifiers))
		for i, ident := range n.Identifiers {
			vars[i] = ident
		}

	case *ast.Assignment:

		vars = n.Variables
		values = n.Values
		isDecl = n.Type == ast.AssignmentDeclaration

		// TODO (Gianluca): ast.Assignment does not have a type field (Type
		// indicates the type of the assignment itself, not the type of the
		// values). typ = tc.checkType(...?, noEllipses)

		if len(vars) == 1 && len(values) == 1 {
			tc.assignSingle(node, vars[0], values[0], typ, isDecl, false)
			return
		}

	default:

		panic(fmt.Errorf("bug: unexpected node %T", node))

	}

	if len(vars) == len(values) {
		for i := range vars {
			// TODO (Gianluca): if all variables have already been declared
			// previously, a declaration must end with an error.
			// _, alreadyDefined := tc.LookupScope("variable name", true) // TODO (Gianluca): to review.
			alreadyDefined := false
			tc.assignSingle(node, vars[i], values[i], typ, isDecl && !alreadyDefined, isConst)
		}
		return
	}

	if len(vars) >= 2 && len(values) == 1 {
		call, ok := values[0].(*ast.Call)
		if ok {
			values := tc.checkCallExpression(call, false) // TODO (Gianluca): is "false" correct?
			if len(vars) != len(values) {
				panic(tc.errorf(node, "assignment mismatch: %d variables but %v returns %d values", len(vars), call, len(values)))
			}
			for i := range vars {
				// TODO (Gianluca): replace the second "variables[i]" with "values[i]"
				tc.assignSingle(node, vars[i], vars[i], typ, isDecl, isConst)
			}
			return
		}
	}

	if len(vars) == 2 && len(values) == 1 {
		switch values[0].(type) {

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

	panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(vars), len(values)))
}

// assignSingle generically assigns value to variable. node must
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
func (tc *typechecker) assignSingle(node ast.Node, variable, value ast.Expression, typ *ast.TypeInfo, isDeclaration, isConst bool) {

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

	// The predeclared value nil cannot be used to initialize a variable with no
	// explicit type. [https://golang.org/ref/spec#Variable_declarations]
	if valueTi.Nil() && typ == nil {
		panic("typ == nil && valueTi.Nil()") // TODO (Gianluca): to review.
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
			if typ != nil {
				// If a type is present, each variable is given that type...
				// [https://golang.org/ref/spec#Variable_declarations]
				newValueTi.Type = typ.Type
			} else {
				// ...otherwise, each variable is given the type of the
				// corresponding initialization value in the assignment.
				// [https://golang.org/ref/spec#Variable_declarations]
				defaultType := tc.concreteType(valueTi)
				newValueTi.Type = defaultType
			}
		}
		tc.AssignScope(v.Name, newValueTi)

	default:
		panic("bug/not implemented") // TODO (Gianluca): can we have a declaration without an identifier?
	}
	return
}
