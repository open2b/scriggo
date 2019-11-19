// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"scriggo/ast"
)

func (tc *typechecker) obsoleteForRangeAssign(node ast.Node, leftExpr, rightExpr ast.Expression, typ *TypeInfo, isVariableDecl, isConstDecl bool) string {

	right := tc.checkExpr(rightExpr)

	if !isVariableDecl && !isConstDecl && right.Nil() {
		left := tc.checkExpr(leftExpr)
		right = tc.nilOf(left.Type)
		tc.typeInfos[rightExpr] = right
	}

	if isVariableDecl && typ != nil && right.Nil() {
		right = tc.nilOf(typ.Type)
		tc.typeInfos[rightExpr] = right
	}

	if typ == nil {
		// Type is not explicit, so is deducted by value.
		right.setValue(nil)
	} else {
		// Type is explicit, so must check assignability.
		if err := tc.isAssignableTo(right, rightExpr, typ.Type); err != nil {
			if _, ok := rightExpr.(*ast.Placeholder); ok {
				panic(tc.errorf(node, "cannot assign %s to %s (type %s) in multiple assignment", right, leftExpr, typ))
			}
			panic(tc.errorf(node, "%s in assignment", err))
		}
		if right.Nil() {
			// Note that this doesn't change the type info associated to node
			// 'right'; it just uses a new type info inside this function.
			right = tc.nilOf(typ.Type)
		} else {
			right.setValue(typ.Type)
		}
	}

	// When declaring a variable, left side must be a name.
	// Note that the error message takes for granted that isVariableDecl refers
	// to a declaration assignment. This is always true because 'var' nodes
	// require an *ast.Identifier as lhs, so !isIdent is always false.
	if isVariableDecl {
		if _, isIdent := leftExpr.(*ast.Identifier); !isIdent {
			panic(tc.errorf(node, "non-name %s on left side of :=", leftExpr))
		}
	}

	switch leftExpr := leftExpr.(type) {

	case *ast.Identifier:

		if leftExpr.Name == "_" {
			return ""
		}

		if isConstDecl {
			newRight := &TypeInfo{}
			if typ == nil {
				if right.Nil() {
					panic(tc.errorf(node, "use of untyped nil"))
				}
				newRight.Type = right.Type
			} else {
				newRight.Type = typ.Type
			}
			tc.typeInfos[leftExpr] = newRight
			if _, alreadyInCurrentScope := tc.lookupScopes(leftExpr.Name, true); alreadyInCurrentScope {
				return ""
			}
			newRight.Constant = right.Constant
			if right.Untyped() {
				newRight.Properties = PropertyUntyped
			}
			tc.assignScope(leftExpr.Name, newRight, nil)
			return leftExpr.Name
		}

		if isVariableDecl {
			newRight := &TypeInfo{}
			if typ == nil {
				if right.Nil() {
					panic(tc.errorf(node, "use of untyped nil"))
				}
				newRight.Type = right.Type
			} else {
				newRight.Type = typ.Type
			}
			tc.typeInfos[leftExpr] = newRight
			if _, alreadyInCurrentScope := tc.lookupScopes(leftExpr.Name, true); alreadyInCurrentScope {
				return ""
			}
			newRight.Properties |= PropertyAddressable
			tc.assignScope(leftExpr.Name, newRight, leftExpr)
			if !tc.opts.AllowNotUsed {
				tc.unusedVars = append(tc.unusedVars, &scopeVariable{
					ident:      leftExpr.Name,
					scopeLevel: len(tc.scopes) - 1,
					node:       node,
				})
			}
			return leftExpr.Name
		}

		// Simple assignment.
		left := tc.checkIdentifier(leftExpr, false)
		if !left.Addressable() {
			panic(tc.errorf(leftExpr, "cannot assign to %v", leftExpr))
		}
		if err := tc.isAssignableTo(right, rightExpr, left.Type); err != nil {
			if _, ok := rightExpr.(*ast.Placeholder); ok {
				panic(tc.errorf(node, "cannot assign %s to %s (type %s) in multiple assignment", right, leftExpr, left))
			}
			panic(tc.errorf(node, "%s in assignment", err))
		}
		right.setValue(left.Type)
		tc.typeInfos[leftExpr] = left
	default:
		panic(tc.errorf(node, "BUG"))
	}

	return ""
}

// cantBeBlank panics if expr is the blank identifier.
func (tc *typechecker) cantBeBlank(expr ast.Expression) {
	if isBlankIdentifier(expr) {
		panic(tc.errorf(expr, "cannot use _ as value"))
	}
}

// operatorFromAssignmentType returns an operator type from an assignment
// type.
func operatorFromAssignmentType(assignmentType ast.AssignmentType) ast.OperatorType {
	switch assignmentType {
	case ast.AssignmentAddition, ast.AssignmentIncrement:
		return ast.OperatorAddition
	case ast.AssignmentSubtraction, ast.AssignmentDecrement:
		return ast.OperatorSubtraction
	case ast.AssignmentMultiplication:
		return ast.OperatorMultiplication
	case ast.AssignmentDivision:
		return ast.OperatorDivision
	case ast.AssignmentModulo:
		return ast.OperatorModulo
	case ast.AssignmentAnd:
		return ast.OperatorAnd
	case ast.AssignmentOr:
		return ast.OperatorOr
	case ast.AssignmentXor:
		return ast.OperatorXor
	case ast.AssignmentAndNot:
		return ast.OperatorAndNot
	case ast.AssignmentLeftShift:
		return ast.OperatorLeftShift
	case ast.AssignmentRightShift:
		return ast.OperatorRightShift
	}
	panic("unexpected assignment type")
}
