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

func isBlankIdentifier(expr ast.Expression) bool {
	ident, ok := expr.(*ast.Identifier)
	if !ok {
		return false
	}
	return ident.Name == "_"
}

// checkAssignment checks the assignment node.
//
// TODO (Gianluca): check error checking order.
func (tc *typechecker) checkAssignment(node ast.Node) {

	var vars, values []ast.Expression
	var typ *ast.TypeInfo
	var isDecl, isConst bool

	isVarOrConst := false

	switch n := node.(type) {

	case *ast.Var:

		isVarOrConst = true
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
				newVar := tc.assignSingle(node, n.Identifiers[i], nil, zero, typ, true, false)
				if newVar == "" && !isBlankIdentifier(n.Identifiers[i]) {
					panic(tc.errorf(node, "%s redeclared in this block", n.Identifiers[i]))
				}
			}
			return
		}

		if len(n.Identifiers) == 1 && len(values) == 1 {
			newVar := tc.assignSingle(node, n.Identifiers[0], values[0], nil, typ, true, false)
			if newVar == "" && !isBlankIdentifier(n.Identifiers[0]) {
				panic(tc.errorf(node, "%s redeclared in this block", n.Identifiers[0]))
			}
			return
		}

		vars = make([]ast.Expression, len(n.Identifiers))
		for i, ident := range n.Identifiers {
			vars[i] = ident
		}

	case *ast.Const:

		isVarOrConst = true
		values = n.Values
		isConst = true
		isDecl = true
		if n.Type != nil {
			typ = tc.checkType(n.Type, noEllipses)
		}

		if len(n.Identifiers) == 1 && len(values) == 1 {
			newConst := tc.assignSingle(node, n.Identifiers[0], values[0], nil, typ, true, true)
			if newConst == "" && !isBlankIdentifier(n.Identifiers[0]) {
				panic(tc.errorf(node, "%s redeclared in this block", n.Identifiers[0]))
			}
			return
		}

		// The number of identifiers must be equal to the number of expressions.
		// [https://golang.org/ref/spec#Constant_declarations]
		if len(n.Identifiers) > len(values) {
			panic(tc.errorf(node, "missing value in const declaration"))
		}
		if len(n.Identifiers) < len(values) {
			panic(tc.errorf(node, "extra expression in const declaration"))
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
			_, err := tc.binaryOp(ast.NewBinaryOperator(n.Pos(), opType, n.Variables[0], n.Values[0]))
			if err != nil {
				panic(err)
			}
			variable := n.Variables[0]
			tc.assignSingle(node, variable, n.Values[0], nil, nil, false, false)
			return
		}

		vars = n.Variables
		values = n.Values
		isDecl = n.Type == ast.AssignmentDeclaration

		if len(vars) == 1 && len(values) == 1 {
			newVar := tc.assignSingle(node, vars[0], values[0], nil, typ, isDecl, false)
			if newVar == "" && isDecl {
				panic(tc.errorf(node, "no new variables on left side of :="))
			}
			return
		}

	default:

		panic(fmt.Errorf("bug: unexpected node %T", node))

	}

	if len(vars) >= 2 && len(values) == 1 {
		call, ok := values[0].(*ast.Call)
		if ok {
			tis := tc.checkCallExpression(call, false)
			if len(vars) != len(tis) {
				panic(tc.errorf(node, "assignment mismatch: %d variables but %v returns %d values", len(vars), call, len(values)))
			}
			values = nil
			for _, ti := range tis {
				newCall := ast.NewCall(call.Pos(), call.Func, call.Args, false)
				newCall.SetTypeInfo(ti)
				values = append(values, newCall)
			}
		}
	}

	if len(vars) == 2 && len(values) == 1 {
		switch v := values[0].(type) {

		case *ast.TypeAssertion:

			value1 := ast.NewTypeAssertion(v.Pos(), v.Expr, v.Type)
			value2 := ast.NewTypeAssertion(v.Pos(), v.Expr, v.Type)
			ti := tc.checkType(values[0], noEllipses)
			value1.SetTypeInfo(&ast.TypeInfo{Type: ti.Type})
			value2.SetTypeInfo(untypedBoolTypeInfo)
			values = []ast.Expression{value1, value2}

		case *ast.Index:

			value1 := ast.NewIndex(v.Pos(), v.Expr, v.Index)
			value2 := ast.NewIndex(v.Pos(), v.Expr, v.Index)
			ti := tc.checkExpression(values[0])
			value1.SetTypeInfo(&ast.TypeInfo{Type: ti.Type})
			value2.SetTypeInfo(untypedBoolTypeInfo)
			values = []ast.Expression{value1, value2}

		}
	}

	if len(vars) != len(values) {
		panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(vars), len(values)))
	}

	newVars := ""
	tmpScope := typeCheckerScope{}
	for i := range vars {
		var newVar string
		if valueTi := values[i].TypeInfo(); valueTi == nil {
			newVar = tc.assignSingle(node, vars[i], values[i], nil, typ, isDecl, isConst)
		} else {
			newVar = tc.assignSingle(node, vars[i], nil, valueTi, typ, isDecl, isConst)
		}
		if isDecl {
			tmpScope[newVar], _ = tc.LookupScopes(newVar, true)
			delete(tc.scopes[len(tc.scopes)-1], newVar)
		}
		if isVarOrConst && newVar == "" && !isBlankIdentifier(vars[i]) {
			panic(tc.errorf(node, "%s redeclared in this block", vars[i]))
		}
		newVars = newVars + newVar
	}
	if newVars == "" && isDecl {
		panic(tc.errorf(node, "no new variables on left side of :="))
	}
	for d, ti := range tmpScope {
		tc.AssignScope(d, ti)
	}
	return

}

// assignSingle generically assigns value to variable. node must
// contain the assignment node (or the var/const declaration node) and it's used
// for error messages only. If the declaration specified a type, that must be
// passed as "typ" argument. isDeclaration and isConst indicates, respectively,
// if the assignment is a declaration and if it's a constant.
//
// TODO (Gianluca): typ doesn't get the type's zero, just checks if type is
// correct when a value is provided. Implement "var a int"
//
// TODO (Gianluca): when assigning a costant to a value in scope, constant isn't
// constant anymore.
//
// TODO (Gianluca): assegnamento con funzione con tipo errato:
// https://play.golang.org/p/0J7GSWft4aM
//
// Returns the identifier of the new declared variable, otherwise empty string.
//
// TODO (Gianluca): value and valueTi can be the same argument: use TypeInfo and
// SetTypeInfo.
//
func (tc *typechecker) assignSingle(node ast.Node, variable, value ast.Expression, valueTi *ast.TypeInfo, typ *ast.TypeInfo, isDeclaration, isConst bool) string {

	if valueTi == nil {
		valueTi = tc.checkExpression(value)
	}

	// If it's a constant declaration, a constant value must be provided.
	if isConst && !valueTi.IsConstant() {
		panic(tc.errorf(node, "const initializer %s is not a constant", value))
	}

	// If a type is provided, value must be assignable to type.
	if typ != nil && !tc.isAssignableTo(valueTi, typ.Type) {
		if value == nil {
			panic(tc.errorf(node, "cannot assign %s to %s (type %s) in multiple assignment", valueTi.ShortString(), variable, typ))
		}
		panic(tc.errorf(node, "cannot use %v (type %v) as type %v in assignment", value, valueTi.ShortString(), typ))
	}

	switch v := variable.(type) {

	case *ast.Identifier:

		if v.Name == "_" {
			// TODO (Gianluca): check if blank identifier is used correctly (has
			// no type, etc..).. or delegate this to parser?
			return ""
		}

		// If it's not a declaration, variable must already exists in some
		// scope. Its type must be retrieved, and value must be assignable
		// to that.
		if isDeclaration {
			newValueTi := &ast.TypeInfo{}
			// Cannot declarate a variable if already exists in current scope.
			if _, alreadyInCurrentScope := tc.LookupScopes(v.Name, true); alreadyInCurrentScope {
				return ""
			}
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
				//
				// «If that value is an untyped constant, it is first
				// implicitly converted to its default type.»
				newValueTi.Type = valueTi.Type
			}
			v.SetTypeInfo(newValueTi)
			if isConst {
				newValueTi.Value = valueTi.Value
				tc.AssignScope(v.Name, newValueTi)
				return v.Name
			}
			newValueTi.Properties |= ast.PropertyAddressable
			tc.AssignScope(v.Name, newValueTi)
			tc.unusedVars = append(tc.unusedVars, &scopeVariable{
				ident:      v.Name,
				scopeLevel: len(tc.scopes) - 1,
				node:       node,
			})
			return v.Name
		}
		variableTi := tc.checkIdentifier(v, false)
		if !variableTi.Addressable() {
			panic(tc.errorf(node, "cannot assign to %v", variable))
		}
		if !tc.isAssignableTo(valueTi, variableTi.Type) {
			panic(tc.errorf(node, "cannot use %v (type %v) as type %v in assignment", value, valueTi.ShortString(), variableTi.Type))
		}

	case *ast.Index:

		if isDeclaration {
			panic(tc.errorf(node, "non name %s on left side of :=", variable))
		}
		variableTi := tc.checkExpression(variable)
		switch variableTi.Type.Kind() {
		case reflect.Slice, reflect.Map:
			// Always addressable when used in indexing operation.
		case reflect.Array:
			if !variableTi.Addressable() {
				panic(tc.errorf(node, "cannot assign to %v", variable))
			}
		}
		if !tc.isAssignableTo(valueTi, variableTi.Type) {
			panic(tc.errorf(node, "cannot use %v (type %v) as type %v in assignment", value, valueTi.ShortString(), variableTi.Type))
		}
		return ""

	case *ast.Selector:

		if isDeclaration {
			panic(tc.errorf(node, "non name %s on left side of :=", variable))
		}
		variableTi := tc.checkExpression(variable)
		if !variableTi.Addressable() {
			panic(tc.errorf(node, "cannot assign to %v", variable))
		}
		if !tc.isAssignableTo(valueTi, variableTi.Type) {
			panic(tc.errorf(node, "cannot use %v (type %v) as type %v in assignment", value, valueTi.ShortString(), variableTi.Type))
		}
		return ""

	case *ast.UnaryOperator:

		if isDeclaration {
			panic(tc.errorf(node, "non name %s on left side of :=", variable))
		}
		if v.Operator() == ast.OperatorMultiplication { // pointer indirection.
			variableTi := tc.checkExpression(variable)
			if !tc.isAssignableTo(valueTi, variableTi.Type) {
				panic(tc.errorf(node, "cannot use %v (type %v) as type %v in assignment", value, valueTi.ShortString(), variableTi.Type))
			}
			return ""
		}

	default:

		panic("bug/not implemented")
	}

	return ""
}

// isAssignableTo reports whether x is assignable to type t.
// See https://golang.org/ref/spec#Assignability for details.
func (tc *typechecker) isAssignableTo(x *ast.TypeInfo, t reflect.Type) bool {
	if x.Type == t {
		return true
	}
	if x.Nil() {
		switch t.Kind() {
		case reflect.Ptr, reflect.Func, reflect.Slice, reflect.Map, reflect.Chan, reflect.Interface:
			return true
		}
		return false
	}
	if x.Untyped() {
		_, err := tc.representedBy(x, t)
		return err == nil
	}
	if t.Kind() == reflect.Interface && x.Type.Implements(t) {
		return true
	}
	// Checks if the type of x and t have identical underlying types and at
	// least one is not a defined type.
	return x.Type.AssignableTo(t)
}
