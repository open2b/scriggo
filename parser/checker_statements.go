// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"

	"scrigo/ast"
)

// checkNodesInNewScope checks nodes in a dedicated scope, which will be
// destroyed after use.
func (tc *typechecker) checkNodesInNewScope(nodes []ast.Node) {
	tc.AddScope()
	tc.checkNodes(nodes)
	tc.RemoveCurrentScope()
}

// checkNodes checks nodes an orderer list of statements.
func (tc *typechecker) checkNodes(nodes []ast.Node) {

	for _, node := range nodes {

		switch node := node.(type) {

		case *ast.Block:

			tc.checkNodesInNewScope(node.Nodes)

		case *ast.If:

			tc.AddScope()
			if node.Assignment != nil {
				tc.checkAssignment(node.Assignment)
			}
			expr := tc.checkExpression(node.Condition)
			// TODO(marco): types with underlying type bool and the untyped bool are both allowed as condition.
			// TODO (Gianluca): currently using isAssignableTo (not sure if it's right)
			// if expr.Type != boolType {
			// 	panic(tc.errorf(node.Condition, "non-bool %v (type %s) used as if condition", node.Condition, expr.Type))
			// }
			if !tc.isAssignableTo(expr, boolType) {
				// TODO (Gianluca): error message must include default type.
				panic(tc.errorf(node.Condition, "non-bool %s (type %v) used as if condition", node.Condition, tc.concreteType(expr)))
			}
			if node.Then != nil {
				tc.checkNodesInNewScope(node.Then.Nodes)
			}
			if node.Else != nil {
				switch els := node.Else.(type) {
				case *ast.Block:
					tc.checkNodesInNewScope(els.Nodes)
				case *ast.If:
					// TODO (Gianluca): same problem we had in renderer:
					tc.checkNodes([]ast.Node{els})
				}
			}
			tc.RemoveCurrentScope()

		case *ast.For:

			tc.AddScope()
			if node.Init != nil {
				tc.checkAssignment(node.Init)
			}
			expr := tc.checkExpression(node.Condition)
			// TODO (Gianluca): same as for if
			if !tc.isAssignableTo(expr, boolType) {
				// TODO (Gianluca): error message must include default type.
				panic(tc.errorf(node.Condition, "non-bool %s (type %v) used as for condition", node.Condition, tc.concreteType(expr)))
			}
			if node.Post != nil {
				tc.checkAssignment(node.Post)
			}
			// TODO (Gianluca): can node.Body be nil?
			tc.checkNodesInNewScope(node.Body)
			tc.RemoveCurrentScope()

		case *ast.ForRange:

			tc.AddScope()
			tc.checkAssignment(node.Assignment)
			tc.checkNodesInNewScope(node.Body)
			tc.RemoveCurrentScope()

		case *ast.Assignment:

			switch node.Type {
			case ast.AssignmentIncrement, ast.AssignmentDecrement:
				v := node.Variables[0]
				exprTi := tc.checkExpression(v)
				if !numericKind[exprTi.Type.Kind()] {
					panic(tc.errorf(node, "invalid operation: %v (non-numeric type %s)", v, exprTi))
				}
				return
			case ast.AssignmentAddition, ast.AssignmentSubtraction, ast.AssignmentMultiplication,
				ast.AssignmentDivision, ast.AssignmentModulo:
				variable := node.Variables[0]
				variableTi := tc.checkExpression(variable)
				if !numericKind[variableTi.Type.Kind()] {
					panic(tc.errorf(node, "invalid operation: %v (non-numeric type %s)", node, variableTi))
				}
				tc.assignSingle(node, variable, node.Values[0], nil, false, false)
			case ast.AssignmentSimple, ast.AssignmentDeclaration:
				tc.checkAssignment(node)
			}

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

		case *ast.Value:

			tc.checkExpression(node.Expr)

		case *ast.Identifier:

			// TODO (Gianluca): remove this case and use ast.Expression directly?

			t := tc.checkIdentifier(node)
			if t.IsPackage() {
				panic(tc.errorf(node, "use of package %s without selector", t))
			}
			panic(tc.errorf(node, "%s evaluated but not used", node.Name))

		case ast.Expression:

			tc.checkExpression(node)
			panic(tc.errorf(node, "%s evaluated but not used", node))

		default:

			panic(fmt.Errorf("checkNodes not implemented for type: %T", node))

		}

	}

}

// checkCase checks a switch or type-switch case. isTypeSwitch indicates if the
// parent switch is a type-switch. switchExpr contains the expression of the
// parent switch.
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
		// TODO (Gianluca): this is wrong:
		if cas.Type != switchExprTyp.Type {
			return tc.errorf(expr, "invalid case %v in switch on %v (mismatched types %v and %v)", expr, switchExpr, cas.Type, switchExprTyp.Type)
		}
	}
	tc.checkNodes(node.Body)
	tc.RemoveCurrentScope()
	return nil
}
