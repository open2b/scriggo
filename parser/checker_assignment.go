// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"
	"reflect"

	"scrigo/ast"
)

// checkAssignment checks the assignment node.
//
// TODO (Gianluca): check error checking order.
//
// TODO (Gianluca): "a, b, c := 1, 2, a": how does Go behave? Does it find "a"
// or evaluation order doesn't matter, so "a" is not declared? -> "undefined: a"
// (stessa cosa con le costanti)
//
// TODO (Gianluca): Unlike regular variable declarations, a short variable
// declaration may redeclare variables provided they were originally declared
// earlier in the same block (or the parameter lists if the block is the
// function body) with the same type, and at least one of the non-blank
// variables is new. As a consequence, redeclaration can only appear in a
// multi-variable short declaration. Redeclaration does not introduce a new
// variable; it just assigns a new value to the original.
// [https://golang.org/ref/spec#Short_variable_declarations]
//
func (tc *typechecker) checkAssignment(node ast.Node) {

	var vars, values []ast.Expression
	var typ *ast.TypeInfo
	var isDecl, isConst bool

	switch n := node.(type) {

	case *ast.Var:

		values = n.Values
		isDecl = true
		if n.Type != nil {
			typ = tc.checkType(n.Type, noEllipses)
		}

		// [...] Otherwise [no list of expressions is given], each variable is
		// initialized to its zero value.
		// [https://golang.org/ref/spec#Variable_declarations]
		if len(values) == 0 {
			for i := range n.Identifiers {
				zero := &ast.TypeInfo{Type: typ.Type}
				tc.assignSingle(node, n.Identifiers[i], nil, zero, typ, true, false)
			}
			return
		}

		if len(vars) == 1 && len(values) == 1 {
			tc.assignSingle(node, n.Identifiers[0], values[0], nil, typ, true, false)
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
		if n.Type != nil {
			typ = tc.checkType(n.Type, noEllipses)
		}

		if len(vars) == 1 && len(values) == 1 {
			tc.assignSingle(node, n.Identifiers[0], values[0], nil, typ, true, true)
			return
		}

		// The number of identifiers must be equal to the number of expressions.
		// [https://golang.org/ref/spec#Constant_declarations]
		if len(n.Identifiers) != len(values) {
			panic("len(variables) != len(values)") // TODO (Gianluca): to review.
		}

		vars = make([]ast.Expression, len(n.Identifiers))
		for i, ident := range n.Identifiers {
			vars[i] = ident
		}

	case *ast.Assignment:

		switch n.Type {
		case ast.AssignmentIncrement, ast.AssignmentDecrement:
			v := n.Variables[0]
			exprTi := tc.checkExpression(v)
			if !numericKind[exprTi.Type.Kind()] {
				panic(tc.errorf(node, "invalid operation: %v (non-numeric type %s)", node, exprTi))
			}
			// TODO (Gianluca): also check for assignability
			return
		case ast.AssignmentAddition, ast.AssignmentSubtraction, ast.AssignmentMultiplication,
			ast.AssignmentDivision, ast.AssignmentModulo:
			variable := n.Variables[0]
			tc.assignSingle(node, variable, n.Values[0], nil, nil, false, false)
			return
		}

		vars = n.Variables
		values = n.Values
		isDecl = n.Type == ast.AssignmentDeclaration

		// TODO (Gianluca): ast.Assignment does not have a type field (Type
		// indicates the type of the assignment itself, not the type of the
		// values). typ = tc.checkType(...?, noEllipses)

		if len(vars) == 1 && len(values) == 1 {
			tc.assignSingle(node, vars[0], values[0], nil, typ, isDecl, false)
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
			tc.assignSingle(node, vars[i], values[i], nil, typ, isDecl && !alreadyDefined, isConst)
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
				tc.assignSingle(node, vars[i], nil, values[i], typ, isDecl, isConst)
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
// TODO (Gianluca): assegnamento con funzione con tipo errato: https://play.golang.org/p/0J7GSWft4aM
//
func (tc *typechecker) assignSingle(node ast.Node, variable, value ast.Expression, valueTi *ast.TypeInfo, typ *ast.TypeInfo, isDeclaration, isConst bool) {

	if valueTi == nil {
		valueTi = tc.checkExpression(value)
	}

	// If it's a constant declaration, a constant value must be provided.
	if isConst && (valueTi.Value == nil) {
		panic(tc.errorf(node, "const initializer %s is not a constant", value))
	}

	// If a type is provided, value must be assignable to type.
	if typ != nil && !tc.isAssignableTo(valueTi, typ.Type) {
		if value == nil {
			panic(tc.errorf(node, "cannot assign %s to %s (type %s) in multiple assignment", valueTi.Type, variable, typ))
		}
		panic(tc.errorf(node, "cannot use %v (type %v) as type %v in assignment", value, valueTi.ShortString(), typ))
	}

	switch v := variable.(type) {

	case *ast.Identifier:

		if v.Name == "_" {
			// TODO (Gianluca): check if blank identifier is used correctly (has
			// no type, etc..).. or delegate this to parser?
			return
		}

		_, alreadyInCurrentScope := tc.LookupScope(v.Name, true)

		// Cannot declarate a variable if already exists in current scope.
		if isDeclaration && alreadyInCurrentScope {
			panic(tc.errorf(node, "no new variables on left side of :="))
		}

		// If it's not a declaration, variable must already exists in some
		// scope. Its type must be retrieved, and value must be assignable
		// to that.
		if !isDeclaration {
			variableTi := tc.checkExpression(variable)
			if !tc.isAssignableTo(valueTi, variableTi.Type) {
				panic(tc.errorf(node, "cannot use %v (type %v) as type %v in assignment", value, valueTi.Type, variableTi.Type))
			}
		}

		newValueTi := &ast.TypeInfo{}

		if isDeclaration {

			if typ != nil {

				// «If a type is present, each variable is given that type.»
				newValueTi.Type = typ.Type

			} else {

				// «The predeclared value nil cannot be used to initialize a
				// variable with no explicit type.»
				if valueTi.Nil() {
					panic(tc.errorf(node, "use of untyped nil"))
				}

				//  «[if no types are presents], each variable is given the type
				//  of the corresponding initialization value in the
				//  assignment.»
				if valueTi.Type != nil {
					newValueTi.Type = valueTi.Type
				} else {
					// «If that value is an untyped constant, it is first
					// implicitly converted to its default type.»
					newValueTi = assignableDefaultType[valueTi.Value.(*ast.UntypedValue).DefaultType]

				}
			}
		}

		tc.AssignScope(v.Name, newValueTi)

	case *ast.Index, *ast.Selector:

		if isDeclaration {
			panic(tc.errorf(node, "non name %s on left side of :=", variable))
		}
		variableTi := tc.checkExpression(variable)
		if !tc.isAssignableTo(valueTi, variableTi.Type) {
			panic(tc.errorf(node, "cannot use %v (type %v) as type %v in assignment", value, valueTi.Type, variableTi.Type))
		}
		return

	case *ast.UnaryOperator:

		if isDeclaration {
			panic(tc.errorf(node, "non name %s on left side of :=", variable))
		}
		if v.Operator() == ast.OperatorMultiplication {
			variableTi := tc.checkExpression(variable)
			if !tc.isAssignableTo(valueTi, variableTi.Type) {
				panic(tc.errorf(node, "cannot use %v (type %v) as type %v in assignment", value, valueTi.Type, variableTi.Type))
			}
			return
		}

	default:

		panic("bug/not implemented")
	}

	return
}

var assignableDefaultType = [...]*ast.TypeInfo{
	ast.DefaultTypeInt:     &ast.TypeInfo{Type: universe["int"].Type, Properties: ast.PropertyAddressable},
	ast.DefaultTypeRune:    &ast.TypeInfo{Type: universe["rune"].Type, Properties: ast.PropertyAddressable},
	ast.DefaultTypeFloat64: &ast.TypeInfo{Type: universe["float64"].Type, Properties: ast.PropertyAddressable},
	ast.DefaultTypeComplex: &ast.TypeInfo{Type: universe["complex"].Type, Properties: ast.PropertyAddressable},
	ast.DefaultTypeString:  &ast.TypeInfo{Type: universe["string"].Type, Properties: ast.PropertyAddressable},
	ast.DefaultTypeBool:    &ast.TypeInfo{Type: universe["bool"].Type, Properties: ast.PropertyAddressable},
}

// isAssignableTo reports whether x is assignable to type T.
// See https://golang.org/ref/spec#Assignability for details.
//
// TODO (Gianluca): perhaps this method can be optimized, but this
// implementation reflects Golang specs, trying to consider any special case.
// Type 'reflect.Type' has a 'AssignableTo' method, but it covers only some of
// the cases below.
func (tc *typechecker) isAssignableTo(x *ast.TypeInfo, T reflect.Type) bool {

	// «x's type is identical to T.»
	if x.Type == T {
		return true
	}

	// «x's type V and T have identical underlying types and at least one of V
	// or T is not a defined type.»
	if x.Type != nil && x.Type.Kind() == T.Kind() {
		xIsNotDefType := x.Type.Name() == ""
		TIsNotDefType := T.Name() == ""
		if xIsNotDefType || TIsNotDefType {
			return true
		}
	}

	// «T is an interface type and x implements T.»
	if T.Kind() == reflect.Interface {
		if x.Type != nil {
			if x.Type.Implements(T) {
				return true
			}
		}
	}

	// «x is the predeclared identifier nil and T is a pointer, function, slice,
	// map, channel, or interface type.»
	if x.Nil() {
		switch T.Kind() {
		case reflect.Ptr, reflect.Func, reflect.Slice, reflect.Map, reflect.Chan, reflect.Interface:
			return true
		}
		return false
	}

	// «x is an untyped constant representable by a value of type T.»
	if x.Type == nil {
		_, err := tc.convert(x, T, false)
		return err == nil
	}

	return false
}
