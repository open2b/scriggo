// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"
	"scrigo/ast"
)

func (tc *typechecker) checkInNewScope(nodes []ast.Node) {
	tc.AddScope()
	tc.checkNodes(nodes)
	tc.RemoveCurrentScope()
}

func (tc *typechecker) checkNodes(nodes []ast.Node) {

	for _, node := range nodes {

		switch node := node.(type) {

		case *ast.Block:

			tc.checkInNewScope(node.Nodes)

		case *ast.If:

			tc.AddScope()
			tc.checkAssignment(node.Assignment)
			expr := tc.checkExpression(node.Condition)
			// TODO(marco): types with underlying type bool and the untyped bool are both allowed as condition.
			if expr.Type != boolType {
				panic(tc.errorf(node.Condition, "non-bool %v (type %s) used as if condition", node.Condition, expr.Type))
			}
			if node.Then != nil {
				tc.checkInNewScope(node.Then.Nodes)
			}
			if node.Else != nil {
				switch els := node.Else.(type) {
				case *ast.Block:
					tc.checkInNewScope(els.Nodes)
				case *ast.If:
					// TODO (Gianluca): same problem we had in renderer:
					tc.checkNodes([]ast.Node{els})
				}
			}
			tc.RemoveCurrentScope()

		case *ast.For:

			tc.AddScope()
			tc.checkAssignment(node.Init)
			expr := tc.checkExpression(node.Condition)
			if expr.Type != boolType {
				panic(tc.errorf(node.Condition, "non-bool %v (type %s) used as for condition", node.Condition, expr.Type))
			}
			tc.checkAssignment(node.Post)
			tc.checkInNewScope(node.Body)
			tc.RemoveCurrentScope()

		case *ast.ForRange:

			tc.AddScope()
			tc.checkAssignment(node.Assignment)
			tc.checkInNewScope(node.Body)
			tc.RemoveCurrentScope()

		case *ast.Assignment:

			switch node.Type {
			case ast.AssignmentIncrement, ast.AssignmentDecrement:
				v := node.Variables[0]
				exprType := tc.checkExpression(v)
				// TODO (Gianluca): to review.
				_ = exprType
				// if !exprType.Numeric() {
				// 	panic(tc.errorf(node, "invalid operation: %v (non-numeric type %s)", v, exprType))
				// }
				return
			case ast.AssignmentAddition, ast.AssignmentSubtraction, ast.AssignmentMultiplication,
				ast.AssignmentDivision, ast.AssignmentModulo:
				lv := node.Variables[0]
				lt := tc.checkExpression(lv)
				// TODO (Gianluca): to review.
				_ = lt
				// if !lt.Numeric() {
				// 	panic(tc.errorf(node, "invalid operation: %v (non-numeric type %s)", lv, lt))
				// }
				rv := node.Values[0]
				tc.assignValueToVariable(node, lv, rv, nil, false, false)
			case ast.AssignmentSimple, ast.AssignmentDeclaration:
				tc.checkAssignment(node)
			}

		case *ast.Identifier:

			t := tc.evalIdentifier(node)
			if t.IsPackage() {
				panic(tc.errorf(node, "use of package %s without selector", t))
			}
			panic(tc.errorf(node, "%s evaluated but not used", node.Name))

		case *ast.Break, *ast.Continue:

		case *ast.Return:

			// TODO (Gianluca): should check if return is expected?
			// TODO (Gianluca): check if return's value is the same of the function where return is in.
			panic("not implemented")

		case *ast.Switch:

			tc.AddScope()
			tc.checkAssignment(node.Init)
			for _, cas := range node.Cases {
				err := tc.checkCase(cas, false, node.Expr)
				if err != nil {
					panic(err)
				}
			}
			tc.RemoveCurrentScope()

		case *ast.TypeSwitch:

			tc.AddScope()
			tc.checkAssignment(node.Init)
			tc.checkAssignment(node.Assignment)
			for _, cas := range node.Cases {
				err := tc.checkCase(cas, true, nil)
				if err != nil {
					panic(err)
				}
			}
			tc.RemoveCurrentScope()

		case *ast.Const, *ast.Var:

			tc.checkAssignment(node)

		default:

			panic(fmt.Errorf("checkNodes not implemented for type: %T", node))

		}

	}

}

func (tc *typechecker) checkCase(node *ast.Case, isTypeSwitch bool, switchExpr ast.Expression) error {
	tc.AddScope()
	switchExprTyp := tc.typeof(switchExpr, noEllipses)
	for _, expr := range node.Expressions {
		cas := tc.typeof(expr, noEllipses)
		if isTypeSwitch && !switchExprTyp.IsType() {
			return tc.errorf(expr, "%v (type %v) is not a type", expr, cas.Type)
		}
		if !isTypeSwitch && switchExprTyp.IsType() {
			return tc.errorf(expr, "type %v is not an expression", cas.Type)
		}
		if cas.Type != switchExprTyp.Type {
			return tc.errorf(expr, "invalid case %v in switch on %v (mismatched types %v and %v)", expr, switchExpr, cas.Type, switchExprTyp.Type)
		}
	}
	tc.checkNodes(node.Body)
	tc.RemoveCurrentScope()
	return nil
}

// TODO (Gianluca): handle "isConst"
// TODO (Gianluca): typ doesn't get the type zero, just checks if type is
// correct when a value is provided. Implement "var a int"
func (tc *typechecker) assignValueToVariable(node ast.Node, variable, value ast.Expression, typ *ast.TypeInfo, isDeclaration, isConst bool) {
	variableTi := tc.checkExpression(variable)
	valueTi := tc.checkExpression(value)
	if isConst && (valueTi.Constant == nil) {
		panic(tc.errorf(node, "const initializer %s is not a constant", value))
	}
	if !tc.isAssignableTo(valueTi, typ.Type) {
		panic(tc.errorf(node, "canont use %v (type %v) as type %v in assignment", value, valueTi, typ))
	}
	if !tc.isAssignableTo(variableTi, valueTi.Type) {
		panic(tc.errorf(node, "cannot use %v (type %v) as type %v in assignment", variable, variableTi.Type, valueTi.Type))
	}
	if isDeclaration {
		if ident, ok := variable.(*ast.Identifier); ok {
			_, alreadyDefined := tc.LookupScope(ident.Name, true)
			if alreadyDefined {
				panic(tc.errorf(node, "no new variables on left side of :="))
			}
			tc.AssignScope(ident.Name, valueTi)
			return
		}
		panic("bug/not implemented") // TODO (Gianluca): can we have a declaration without an identifier?
	}
	return
}

// TODO (Gianluca): manage builtin functions.
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

// TODO (Gianluca): handle
//		 var a, b int = f() // func f() (int, string)
// (should be automatically handled, to verify)
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
	case *ast.Const:
		variables = make([]ast.Expression, len(n.Identifiers))
		for i, ident := range n.Identifiers {
			variables[i] = ident
		}
		values = n.Values
		isConst = true
		isDeclaration = true
	case *ast.Assignment:
		variables = n.Variables
		values = n.Values
		isDeclaration = n.Type == ast.AssignmentDeclaration
	}
	if len(variables) == 1 && len(values) == 1 {
		tc.assignValueToVariable(node, variables[0], values[0], typ, isDeclaration, isConst)
		return
	}
	if len(values) == 0 && typ != nil {
		for i := range variables {
			// TODO (Gianluca): zero must contain the zero of type "typ".
			zero := (ast.Expression)(nil)
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
	if len(variables) == 2 && len(values) == 1 {
		switch value := values[0].(type) {
		case *ast.Call:
			tc.checkAssignmentWithCall(node, variables, value, typ, isDeclaration, isConst)
		case *ast.TypeAssertion:
			// TODO (Gianluca):
			// Se la type assertion è valida,
			// if type assertion is valid:
			// 		first value is type assertion type
			//		second value is always a boolean
		case *ast.Index:
			// TODO (Gianluca):
			// if indexing a map && key is valid ... :
			//		first value is maptype.Elem()
			// 		second value is always a boolean
		}
		return
	}
	if len(variables) > 2 && len(values) == 1 {
		call, ok := values[0].(*ast.Call)
		if ok {
			tc.checkAssignmentWithCall(node, variables, call, typ, isDeclaration, isConst)
		}
		return
	}
	panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(variables), len(values)))
}
