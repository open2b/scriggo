// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import "github.com/open2b/scriggo/ast"

// TODO: this method is obsolete and must be removed when changing the type
// checking of ForRange nodes.
func (tc *typechecker) obsoleteForRangeAssign(node ast.Node, leftExpr, rightExpr ast.Expression, typ *typeInfo, isVariableDecl, isConstDecl bool) string {

	right := tc.checkExpr(rightExpr)

	if !isVariableDecl && !isConstDecl && right.Nil() {
		left := tc.checkExpr(leftExpr)
		right = tc.nilOf(left.Type)
		tc.compilation.typeInfos[rightExpr] = right
	}

	if isVariableDecl && typ != nil && right.Nil() {
		right = tc.nilOf(typ.Type)
		tc.compilation.typeInfos[rightExpr] = right
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
			newRight := &typeInfo{}
			if typ == nil {
				if right.Nil() {
					panic(tc.errorf(node, "use of untyped nil"))
				}
				newRight.Type = right.Type
			} else {
				newRight.Type = typ.Type
			}
			tc.compilation.typeInfos[leftExpr] = newRight
			if _, ok := tc.scopes.Current(leftExpr.Name); ok {
				return ""
			}
			newRight.Constant = right.Constant
			if right.Untyped() {
				newRight.Properties = propertyUntyped
			}
			tc.assignScope(leftExpr.Name, newRight, nil)
			return leftExpr.Name
		}

		if isVariableDecl {
			newRight := &typeInfo{}
			if typ == nil {
				if right.Nil() {
					panic(tc.errorf(node, "use of untyped nil"))
				}
				newRight.Type = right.Type
			} else {
				newRight.Type = typ.Type
			}
			tc.compilation.typeInfos[leftExpr] = newRight
			if _, ok := tc.scopes.Current(leftExpr.Name); ok {
				return ""
			}
			newRight.Properties |= propertyAddressable
			tc.assignScope(leftExpr.Name, newRight, leftExpr)
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
		tc.compilation.typeInfos[leftExpr] = left
	default:
		panic(tc.errorf(node, "BUG"))
	}

	return ""
}
