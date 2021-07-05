// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"

	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/runtime"
)

// templateFileToPackage transforms a tree of a declarations file to a package
// tree, moving the top level nodes, excluding text and comment nodes, in a
// package node that becomes the only node of tree. If tree already has a
// package, templateFileToPackage does nothing.
func (tc *typechecker) templateFileToPackage(tree *ast.Tree) {
	if len(tree.Nodes) == 1 {
		if _, ok := tree.Nodes[0].(*ast.Package); ok {
			// tree has already a package, do nothing.
			return
		}
	}
	nodes := make([]ast.Node, 0, len(tree.Nodes)/2)
	for _, n := range tree.Nodes {
		switch n := n.(type) {
		case *ast.Text, *ast.Comment:
		case *ast.Extends, *ast.Import, *ast.Func, *ast.Var, *ast.Const, *ast.TypeDeclaration:
			nodes = append(nodes, n)
		case *ast.Statements:
			nodes = append(nodes, n.Nodes...)
		default:
			panic(fmt.Sprintf("BUG: unexpected node %s", n))
		}
	}
	tree.Nodes = []ast.Node{
		ast.NewPackage(tree.Pos(), "", nodes),
	}
}

// checkNodesInNewScopeError calls checkNodesInNewScope returning checking errors.
func (tc *typechecker) checkNodesInNewScopeError(nodes []ast.Node) ([]ast.Node, error) {
	tc.enterScope()
	nodes, err := tc.checkNodesError(nodes)
	if err != nil {
		return nil, err
	}
	tc.exitScope()
	return nodes, nil
}

// checkNodesInNewScope type checks nodes in a new scope. Panics on error.
func (tc *typechecker) checkNodesInNewScope(nodes []ast.Node) []ast.Node {
	tc.enterScope()
	nodes = tc.checkNodes(nodes)
	tc.exitScope()
	return nodes
}

// checkNodesError calls checkNodes catching panics and returning their errors
// as return parameter.
func (tc *typechecker) checkNodesError(nodes []ast.Node) (newNodes []ast.Node, err error) {
	func() {
		defer func() {
			if r := recover(); r != nil {
				if rerr, ok := r.(*CheckingError); ok {
					err = rerr
				} else {
					panic(r)
				}
			}
		}()
		newNodes = tc.checkNodes(nodes)
	}()
	return newNodes, err
}

// checkNodes type checks one or more statements, returning the new tree branch
// with transformations, if any. Panics on error.
func (tc *typechecker) checkNodes(nodes []ast.Node) []ast.Node {

	tc.terminating = false

	i := 0

nodesLoop:
	for {

		if i >= len(nodes) {
			break nodesLoop
		}
		node := nodes[i]

		switch node := node.(type) {

		case *ast.Import:
			err := tc.checkImport(node)
			if err != nil {
				panic(err)
			}

		case *ast.Text:

		case *ast.Block:
			node.Nodes = tc.checkNodesInNewScope(node.Nodes)

		case *ast.Statements:
			node.Nodes = tc.checkNodes(node.Nodes)

		case *ast.If:
			tc.enterScope()
			if node.Init != nil {
				tc.checkNodes([]ast.Node{node.Init})
			}
			ti := tc.checkExpr(node.Condition)
			if ti.Nil() {
				panic(tc.errorf(node.Condition, "use of untyped nil"))
			}
			if ti.Type.Kind() != reflect.Bool {
				if tc.opts.modality != programMod {
					if ti.IsConstant() {
						c := &typeInfo{
							Constant:   boolConst(!ti.Constant.zero()),
							Properties: propertyUntyped,
							Type:       boolType,
						}
						tc.compilation.typeInfos[node.Condition] = c
						ti = c
					} else {
						node.Condition = ast.NewUnaryOperator(node.Condition.Pos(), internalOperatorNotZero, node.Condition)
						ti = tc.checkExpr(node.Condition)
						tc.compilation.typeInfos[node.Condition] = ti
					}
				} else {
					panic(tc.errorf(node.Condition, "non-bool %s (type %v) used as if condition", node.Condition, ti.ShortString()))
				}
			}
			ti.setValue(nil)
			node.Then.Nodes = tc.checkNodesInNewScope(node.Then.Nodes)
			terminating := tc.terminating
			if node.Else == nil {
				terminating = false
			} else {
				switch els := node.Else.(type) {
				case *ast.Block:
					els.Nodes = tc.checkNodesInNewScope(els.Nodes)
				case *ast.If:
					_ = tc.checkNodes([]ast.Node{els})
				}
				terminating = terminating && tc.terminating
			}
			tc.exitScope()
			tc.terminating = terminating

		case *ast.For:
			tc.enterScope()
			tc.addToAncestors(node)
			if node.Init != nil {
				tc.checkNodes([]ast.Node{node.Init})
			}
			if node.Condition != nil {
				ti := tc.checkExpr(node.Condition)
				if ti.Type.Kind() != reflect.Bool {
					panic(tc.errorf(node.Condition, "non-bool %s (type %v) used as for condition", node.Condition, ti.ShortString()))
				}
				ti.setValue(nil)
			}
			if node.Post != nil {
				tc.checkNodes([]ast.Node{node.Post})
			}
			node.Body = tc.checkNodesInNewScope(node.Body)
			tc.removeLastAncestor()
			tc.exitScope()
			tc.terminating = node.Condition == nil && !tc.hasBreak[node]

		case *ast.ForIn:
			// Check range expression.
			expr := node.Expr
			ti := tc.checkExpr(expr)
			if ti.Nil() {
				panic(tc.errorf(node, "cannot range over nil"))
			}
			ti.setValue(nil)
			// Replace the node with a ForRange node.
			ipos := node.Ident.Pos()
			blank := ast.NewIdentifier(ipos.WithEnd(ipos.Start), "_")
			aPos := ipos.WithEnd(node.Expr.Pos().End)
			var lhs []ast.Expression
			switch ti.Type.Kind() {
			default:
				lhs = []ast.Expression{blank, node.Ident}
			case reflect.Map:
				lhs = []ast.Expression{node.Ident, blank}
			case reflect.Chan:
				lhs = []ast.Expression{node.Ident}
			}
			assignment := ast.NewAssignment(aPos, lhs, ast.AssignmentDeclaration, []ast.Expression{expr})
			assignment.End = node.Expr.Pos().End
			nodes[i] = ast.NewForRange(node.Pos(), assignment, node.Body)
			continue

		case *ast.ForRange:
			tc.enterScope()
			tc.addToAncestors(node)
			// Check range expression.
			expr := node.Assignment.Rhs[0]
			ti := tc.checkExpr(expr)
			if ti.Nil() {
				panic(tc.errorf(node, "cannot range over nil"))
			}
			ti.setValue(nil)
			maxLhs := 2
			lhs := node.Assignment.Lhs
			var typ1, typ2 reflect.Type
			switch typ := ti.Type; typ.Kind() {
			case reflect.Array, reflect.Slice:
				typ1 = intType
				typ2 = typ.Elem()
			case reflect.Map:
				typ1 = typ.Key()
				typ2 = typ.Elem()
			case reflect.String:
				typ1 = intType
				typ2 = runeType
			case reflect.Ptr:
				if typ.Elem().Kind() != reflect.Array {
					panic(tc.errorf(expr, "cannot range over %s (type %s)", expr, ti))
				}
				typ1 = intType
				typ2 = typ.Elem().Elem()
			case reflect.Chan:
				if dir := typ.ChanDir(); dir == reflect.SendDir {
					panic(tc.errorf(node.Assignment.Rhs[0], "invalid operation: range %s (receive from send-only type %s)", expr, ti.String()))
				}
				typ1 = typ.Elem()
				maxLhs = 1
			default:
				panic(tc.errorf(node.Assignment.Rhs[0], "cannot range over %s (type %s)", expr, ti.StringWithNumber(true)))
			}
			// Check variables.
			if lhs != nil {
				if len(lhs) > maxLhs {
					panic(tc.errorf(node, "too many variables in range"))
				}
				ti1 := &typeInfo{Type: typ1, Properties: propertyAddressable}
				declaration := node.Assignment.Type == ast.AssignmentDeclaration
				indexPh := ast.NewPlaceholder()
				tc.compilation.typeInfos[indexPh] = ti1
				tc.obsoleteForRangeAssign(node.Assignment, lhs[0], indexPh, nil, declaration, false)
				if len(lhs) == 2 {
					valuePh := ast.NewPlaceholder()
					tc.compilation.typeInfos[valuePh] = &typeInfo{Type: typ2}
					tc.obsoleteForRangeAssign(node.Assignment, lhs[1], valuePh, nil, declaration, false)
				}
			}
			node.Body = tc.checkNodesInNewScope(node.Body)
			tc.removeLastAncestor()
			tc.exitScope()
			tc.terminating = !tc.hasBreak[node]

		case *ast.Assignment:
			tc.checkGenericAssignmentNode(node)
			if node.Type == ast.AssignmentDeclaration {
				tc.nextValidGoto = len(tc.gotos)
			}
			tc.terminating = false

		case *ast.Break:
			found := false
			for i := len(tc.ancestors) - 1; i >= 0; i-- {
				switch n := tc.ancestors[i].node.(type) {
				case *ast.For, *ast.ForRange, *ast.Switch, *ast.TypeSwitch, *ast.Select:
					tc.hasBreak[n] = true
					found = true
					break
				}
			}
			if !found {
				panic(tc.errorf(node, "break is not in a loop, switch, or select"))
			}
			tc.terminating = false

		case *ast.Continue:
			found := false
			for i := len(tc.ancestors) - 1; i >= 0; i-- {
				switch tc.ancestors[i].node.(type) {
				case *ast.For, *ast.ForRange:
					found = true
					break
				}
			}
			if !found {
				panic(tc.errorf(node, "continue is not in a loop"))
			}
			tc.terminating = false

		case *ast.Fallthrough:
			outOfPlace := true
			if len(tc.ancestors) > 0 {
				parent := tc.ancestors[len(tc.ancestors)-1].node
				if cas, ok := parent.(*ast.Case); ok {
					nn := len(nodes)
				CASE:
					switch i {
					default:
						for j := nn - 1; j >= 0; j-- {
							if nodes[j] == node {
								break
							}
							switch n := nodes[j].(type) {
							case *ast.Comment:
							case *ast.Text:
								if !containsOnlySpaces(n.Text) {
									break CASE
								}
							default:
								break CASE
							}
						}
						fallthrough
					case nn - 1:
						parent = tc.ancestors[len(tc.ancestors)-2].node
						switch sw := parent.(type) {
						case *ast.Switch:
							if cas == sw.Cases[len(sw.Cases)-1] {
								panic(tc.errorf(node, "cannot fallthrough final case in switch"))
							}
							outOfPlace = false
						case *ast.TypeSwitch:
							panic(tc.errorf(node, "cannot fallthrough in type switch"))
						}
					}
				}
			}
			if outOfPlace {
				panic(tc.errorf(node, "fallthrough statement out of place"))
			}

		case *ast.Return:
			assign := tc.checkReturn(node)
			if assign != nil {
				// Create a block statement that contains the assignment and the
				// return statement without its return values.
				//
				// The creation of the block is necessary because we are
				// iterating over nodes, so only one node can be changed without
				// breaking the iteration.
				node.Values = nil
				nodes[i] = ast.NewBlock(node.Pos(), []ast.Node{assign, node})
			}
			tc.terminating = true

		case *ast.Switch:
			tc.enterScope()
			tc.addToAncestors(node)
			// Check the init.
			if node.Init != nil {
				// TODO: this type assertion should be removed/handled differently.
				tc.checkGenericAssignmentNode(node.Init.(*ast.Assignment))
			}
			// Check the expression.
			var texpr *typeInfo
			if node.Expr == nil {
				texpr = &typeInfo{Type: boolType, Constant: boolConst(true)}
				texpr.setValue(nil)
			} else {
				texpr = tc.checkExpr(node.Expr)
				if texpr.Nil() {
					panic(tc.errorf(node, "use of untyped nil"))
				}
				if k := texpr.Type.Kind(); (k == reflect.Struct || k == reflect.Array) && !texpr.Type.Comparable() {
					panic(tc.errorf(node.Expr, "cannot switch on %s (%s is not comparable)", node.Expr, texpr.ShortString()))
				}
				texpr.setValue(nil)
				if texpr.Untyped() {
					c, err := tc.convert(texpr, node.Expr, texpr.Type)
					if err != nil {
						panic(tc.errorf(node.Expr, "%s", err))
					}
					texpr = &typeInfo{Type: texpr.Type, Constant: c}
				}
			}
			// Check the cases.
			terminating := true
			hasFallthrough := false
			positionOf := map[interface{}]*ast.Position{}
			var positionOfDefault *ast.Position
			for _, cas := range node.Cases {
				if cas.Expressions == nil {
					if positionOfDefault != nil {
						panic(tc.errorf(cas, "multiple defaults in switch (first at %s)", positionOfDefault))
					}
					positionOfDefault = cas.Pos()
				}
				for _, ex := range cas.Expressions {
					var ne string
					if node.Expr != nil {
						ne = " on " + node.Expr.String()
					}
					tcase := tc.checkExpr(ex)
					if tcase.Untyped() {
						c, err := tc.convert(tcase, ex, texpr.Type)
						if err != nil {
							if err == errNotRepresentable || err == errTypeConversion {
								panic(tc.errorf(cas, "invalid case %s in switch%s (mismatched types %s and %s)", ex, ne, tcase.ShortString(), texpr.ShortString()))
							}
							panic(tc.errorf(cas, "%s", err))
						}
						if !tcase.Nil() {
							tcase.setValue(texpr.Type)
						}
						tcase = &typeInfo{Type: texpr.Type, Constant: c}
					} else {
						if tc.isAssignableTo(tcase, ex, texpr.Type) != nil && tc.isAssignableTo(texpr, ex, tcase.Type) != nil {
							panic(tc.errorf(cas, "invalid case %s in switch%s (mismatched types %s and %s)", ex, ne, tcase.ShortString(), texpr.ShortString()))
						}
						if !texpr.Type.Comparable() {
							panic(tc.errorf(cas, "invalid case %s in switch (can only compare %s %s to nil)", ex, texpr.Type.Kind(), node.Expr))
						}
						if !tcase.Type.Comparable() {
							panic(tc.errorf(cas, "invalid case %s (type %s) in switch (incomparable type)", ex, tcase.ShortString()))
						}
						tcase.setValue(nil)
					}
					if tcase.IsConstant() && texpr.Type.Kind() != reflect.Bool {
						// Check for duplicates.
						value := tc.typedValue(tcase, tcase.Type)
						if pos, ok := positionOf[value]; ok {
							panic(tc.errorf(cas, "duplicate case %v in switch\n\tprevious case at %s", ex, pos))
						}
						positionOf[value] = ex.Pos()
					}
					tcase.setValue(texpr.Type)
				}
				tc.enterScope()
				tc.addToAncestors(cas)
				cas.Body = tc.checkNodes(cas.Body)
				tc.removeLastAncestor()
				tc.exitScope()
				if !hasFallthrough && len(cas.Body) > 0 {
					_, hasFallthrough = cas.Body[len(cas.Body)-1].(*ast.Fallthrough)
				}
				terminating = terminating && (tc.terminating || hasFallthrough)
			}
			tc.removeLastAncestor()
			tc.exitScope()
			tc.terminating = terminating && !tc.hasBreak[node] && positionOfDefault != nil

		case *ast.TypeSwitch:
			terminating := true
			tc.enterScope()
			tc.addToAncestors(node)
			if node.Init != nil {
				// TODO: this type assertion should be removed/handled differently.
				tc.checkGenericAssignmentNode(node.Init.(*ast.Assignment))
			}
			ta := node.Assignment.Rhs[0].(*ast.TypeAssertion)
			t := tc.checkExpr(ta.Expr)
			if t.Nil() {
				panic(tc.errorf(node, "cannot type switch on non-interface value nil"))
			}
			if t.Type.Kind() != reflect.Interface {
				panic(tc.errorf(node, "cannot type switch on non-interface value %v (type %s)", ta.Expr,
					t.StringWithNumber(true)))
			}
			var positionOfDefault *ast.Position
			var positionOfNil *ast.Position
			positionOf := map[reflect.Type]*ast.Position{}
			for _, cas := range node.Cases {
				if cas.Expressions == nil {
					if positionOfDefault != nil {
						panic(tc.errorf(cas, "multiple defaults in switch (first at %s)", positionOfDefault))
					}
					positionOfDefault = cas.Pos()
				}
				for i, ex := range cas.Expressions {
					expr := cas.Expressions[i]
					t := tc.checkExprOrType(expr)
					if t.Nil() {
						if positionOfNil != nil {
							panic(tc.errorf(cas, "multiple nil cases in type switch (first at %s)", positionOfNil))
						}
						positionOfNil = ex.Pos()
						continue
					}
					if !t.IsType() {
						panic(tc.errorf(cas, "%v (type %s) is not a type", expr, t.StringWithNumber(true)))
					}
					// Check duplicate.
					if pos, ok := positionOf[t.Type]; ok {
						panic(tc.errorf(cas, "duplicate case %v in type switch\n\tprevious case at %s", ex, pos))
					}
					positionOf[t.Type] = ex.Pos()
				}
				tc.enterScope()
				// Case has only one expression (one type), so in its body the
				// type switch variable has the same type of the case type.
				if len(cas.Expressions) == 1 && !tc.compilation.typeInfos[cas.Expressions[0]].Nil() {
					if len(node.Assignment.Lhs) == 1 {
						lh := node.Assignment.Lhs[0]
						n := ast.NewAssignment(
							node.Assignment.Pos(),
							[]ast.Expression{lh},
							node.Assignment.Type,
							[]ast.Expression{
								ast.NewTypeAssertion(ta.Pos(), ta.Expr, cas.Expressions[0]),
							},
						)
						tc.checkGenericAssignmentNode(n)
						// Mark lh as used.
						if !isBlankIdentifier(lh) {
							_ = tc.checkIdentifier(lh.(*ast.Identifier), true)
						}
					}
				} else {
					if len(node.Assignment.Lhs) == 1 {
						lh := node.Assignment.Lhs[0]
						n := ast.NewAssignment(
							node.Assignment.Pos(),
							[]ast.Expression{lh},
							node.Assignment.Type,
							[]ast.Expression{
								ta.Expr,
							},
						)
						tc.checkGenericAssignmentNode(n)
						// Mark lh as used.
						if !isBlankIdentifier(lh) {
							_ = tc.checkIdentifier(lh.(*ast.Identifier), true)
						}
					}
				}
				tc.enterScope()
				tc.addToAncestors(cas)
				cas.Body = tc.checkNodes(cas.Body)
				tc.removeLastAncestor()
				tc.exitScope()
				terminating = terminating && tc.terminating
			}
			tc.removeLastAncestor()
			tc.exitScope()
			tc.terminating = terminating && !tc.hasBreak[node] && positionOfDefault != nil

		case *ast.Select:
			tc.enterScope()
			tc.addToAncestors(node)
			// Check the cases.
			terminating := true
			var positionOfDefault *ast.Position
			for _, cas := range node.Cases {
				switch comm := cas.Comm.(type) {
				case nil:
					if positionOfDefault != nil {
						panic(tc.errorf(cas, "multiple defaults in select (first at %s)", positionOfDefault))
					}
					positionOfDefault = cas.Pos()
				case ast.Expression:
					_ = tc.checkExpr(comm)
					if recv, ok := comm.(*ast.UnaryOperator); !ok || recv.Op != ast.OperatorReceive {
						panic(tc.errorf(node, "select case must be receive, send or assign recv"))
					}
				case *ast.Assignment:
					tc.checkGenericAssignmentNode(comm)
					if comm.Type != ast.AssignmentSimple && comm.Type != ast.AssignmentDeclaration {
						panic(tc.errorf(node, "select case must be receive, send or assign recv"))
					}
					if recv, ok := comm.Rhs[0].(*ast.UnaryOperator); !ok || recv.Op != ast.OperatorReceive {
						panic(tc.errorf(node, "select case must be receive, send or assign recv"))
					}
				case *ast.Send:
					_ = tc.checkNodes([]ast.Node{comm})
				}
				cas.Body = tc.checkNodesInNewScope(cas.Body)
				terminating = terminating && tc.terminating
			}
			tc.removeLastAncestor()
			tc.exitScope()
			tc.terminating = terminating && !tc.hasBreak[node]

		case *ast.Const:
			tc.checkConstantDeclaration(node)
			tc.terminating = false

		case *ast.Var:
			tc.checkVariableDeclaration(node)
			tc.nextValidGoto = len(tc.gotos)
			tc.terminating = false

		case *ast.TypeDeclaration:
			name, ti := tc.checkTypeDeclaration(node)
			if ti != nil {
				tc.assignScope(name, ti, node.Ident)
			}

		case *ast.Show:

			// Handle {{ f() }} where f returns two values and the second value
			// implements 'error'.
			if len(node.Expressions) == 1 {
				if call, ok := node.Expressions[0].(*ast.Call); ok {
					tis := tc.checkCallExpression(call)
					if len(tis) == 2 && tis[1].Type.Implements(errorType) {
						// Change the tree:
						//
						//     from: {{ f(..) }}
						//
						//     to:   {% if v, err := f(arg1, arg2); err == nil %}{{ v }}{% end if %}
						//
						pos := call.Pos()
						init := ast.NewAssignment( // v, err := f(..)
							pos,
							[]ast.Expression{
								ast.NewIdentifier(pos, "v"),
								ast.NewIdentifier(pos, "err"),
							},
							ast.AssignmentDeclaration,
							[]ast.Expression{call}, // f(..)
						)
						cond := ast.NewBinaryOperator(
							pos,
							ast.OperatorEqual,
							ast.NewIdentifier(pos, "err"),
							ast.NewIdentifier(pos, "nil"),
						)
						then := ast.NewBlock( // {{ v }}
							pos,
							[]ast.Node{ast.NewShow(pos, []ast.Expression{ast.NewIdentifier(pos, "v")}, node.Context)},
						)
						nodes[i] = ast.NewIf(pos, init, cond, then, nil)
						continue nodesLoop // check nodes[i]
					}
				}
			}

			for _, expr := range node.Expressions {
				tis := tc.checkExpr2(expr, true)
				for _, ti := range tis {
					if ti == nil {
						continue
					}
					if ti.Nil() {
						panic(tc.errorf(node, "use of untyped nil"))
					}
					if tc.opts.renderer != nil && ti.Type != emptyInterfaceType {
						zero := tc.types.Zero(ti.Type)
						if w, ok := ti.Type.(runtime.Wrapper); ok {
							zero = w.Wrap(zero)
						}
						tc.opts.renderer.Show(tc.env, zero.Interface(), encodeRenderContext(node.Context, false, false))
						if err := tc.env.err; err != nil {
							panic(tc.errorf(node, "cannot show %s (%s)", expr, err))
						}
					}
				}
				ti := tis.TypeInfo()
				ti.setValue(nil)
			}

			tc.terminating = false

		case *ast.Defer:
			if node.Call.Parenthesis() > 0 {
				panic(tc.errorf(node.Call, "expression in defer must not be parenthesized"))
			}
			call, ok := node.Call.(*ast.Call)
			if !ok {
				panic(tc.errorf(node.Call, "expression in defer must be function call"))
			}
			tc.checkCallExpression(call)
			ti := tc.compilation.typeInfos[call.Func]
			if ti.IsBuiltinFunction() {
				name := call.Func.(*ast.Identifier).Name
				switch name {
				case "append", "cap", "len", "make", "new":
					panic(tc.errorf(node, "defer discards result of %s", call))
				case "recover":
					// The statement "defer recover()" is a special case
					// implemented by the emitter.
				case "close", "delete", "panic", "print", "println":
					tc.compilation.typeInfos[call.Func] = deferGoBuiltin(name)
				}
			}
			if ti.IsType() {
				panic(tc.errorf(node, "defer requires function call, not conversion"))
			}
			tc.terminating = false

		case *ast.Go:
			if node.Call.Parenthesis() > 0 {
				panic(tc.errorf(node.Call, "expression in go must not be parenthesized"))
			}
			call, ok := node.Call.(*ast.Call)
			if !ok {
				panic(tc.errorf(node.Call, "expression in go must be function call"))
			}
			tc.checkCallExpression(call)
			ti := tc.compilation.typeInfos[call.Func]
			if ti.IsBuiltinFunction() {
				name := call.Func.(*ast.Identifier).Name
				switch name {
				case "append", "cap", "len", "make", "new":
					panic(tc.errorf(node, "go discards result of %s", call))
				case "close", "delete", "panic", "print", "println", "recover":
					tc.compilation.typeInfos[call.Func] = deferGoBuiltin(name)
				}
			}
			if ti.IsType() {
				panic(tc.errorf(node, "go requires function call, not conversion"))
			}
			if tc.opts.disallowGoStmt {
				panic(tc.errorf(node, "\"go\" statement not available"))
			}
			tc.terminating = false

		case *ast.Send:
			tic := tc.checkExpr(node.Channel)
			if tic.Type.Kind() != reflect.Chan {
				panic(tc.errorf(node, "invalid operation: %s (send to non-chan type %s)", node, tic.ShortString()))
			}
			if tic.Type.ChanDir() == reflect.RecvDir {
				panic(tc.errorf(node, "invalid operation: %s (send to receive-only type %s)", node, tic.ShortString()))
			}
			elemType := tic.Type.Elem()
			tiv := tc.checkExpr(node.Value)
			if err := tc.isAssignableTo(tiv, node.Value, elemType); err != nil {
				if _, ok := err.(invalidTypeInAssignment); ok {
					if tiv.Nil() {
						panic(tc.errorf(node, "cannot convert nil to type %s", elemType))
					}
					if tiv.Type == stringType {
						panic(tc.errorf(node, "cannot convert %s (type %s) to type %s", node.Value, tiv, elemType))
					}
					panic(tc.errorf(node, "%s in send", err))
				}
				panic(tc.errorf(node, "%s", err))
			}
			if tiv.Nil() {
				tiv = tc.nilOf(elemType)
				tc.compilation.typeInfos[node.Value] = tiv
			} else {
				tiv.setValue(elemType)
			}

		case *ast.URL:
			node.Value = tc.checkNodes(node.Value)

		case *ast.UnaryOperator:
			ti := tc.checkExpr(node)
			ti.setValue(nil)

		case *ast.Goto:
			tc.gotos = append(tc.gotos, node.Label.Name)

		case *ast.Label:
			tc.labels[len(tc.labels)-1] = append(tc.labels[len(tc.labels)-1], node.Ident.Name)
			for i, g := range tc.gotos {
				if g == node.Ident.Name {
					if i < tc.nextValidGoto {
						panic(tc.errorf(node, "goto %s jumps over declaration of ? at ?", node.Ident.Name)) // TODO(Gianluca).
					}
					break
				}
			}
			if node.Statement != nil {
				_ = tc.checkNodes([]ast.Node{node.Statement})
			}

		case *ast.Comment, *ast.Raw:

		case *ast.Call:
			tis := tc.checkCallExpression(node)
			ti := tc.compilation.typeInfos[node.Func]
			if ti.IsBuiltinFunction() {
				switch node.Func.(*ast.Identifier).Name {
				case "copy", "recover":
				case "panic":
					tc.terminating = true
				default:
					if len(tis) > 0 {
						panic(tc.errorf(node, "%s evaluated but not used", node))
					}
				}
			} else if ti.IsType() {
				panic(tc.errorf(node, "%s evaluated but not used", node))
			}

		case ast.Expression:

			// Handle function and macro declarations in scripts and templates.
			if fun, ok := node.(*ast.Func); ok && fun.Ident != nil && tc.opts.modality != programMod {
				if fun.Type.Macro && len(fun.Type.Result) == 0 {
					tc.makeMacroResultExplicit(fun)
				}
				// Remove the identifier from the function expression and
				// use it during the assignment.
				ident := fun.Ident
				fun.Ident = nil
				varDecl := ast.NewVar(
					fun.Pos(),
					[]*ast.Identifier{ident},
					fun.Type,
					nil,
				)
				nodeAssign := ast.NewAssignment(
					fun.Pos(),
					[]ast.Expression{ident},
					ast.AssignmentSimple,
					[]ast.Expression{fun},
				)
				// Check the new node, informing the type checker that the
				// current assignment is a function declaration in a script
				// or a macro declaration in a template.
				backup := tc.scriptFuncOrMacroDecl
				tc.scriptFuncOrMacroDecl = true
				newNodes := []ast.Node{varDecl, nodeAssign}

				_ = tc.checkNodes(newNodes)
				// Append the new nodes removing the function literal.
				nodes = append(nodes[:i], append(newNodes, nodes[i+1:]...)...)
				tc.scriptFuncOrMacroDecl = backup
				// Avoid error 'declared but not used' by "using" the
				// identifier.
				identTi := tc.checkIdentifier(ident, true)
				if fun.Type.Macro {
					identTi.Properties |= propertyIsMacroDeclaration
				}
				i += 2

				continue nodesLoop
			}

			ti := tc.checkExpr(node)
			if tc.opts.modality == templateMod {
				if node, ok := node.(*ast.Func); ok && node.Ident != nil {
					tc.assignScope(node.Ident.Name, ti, node.Ident)
					i++
					continue nodesLoop
				}
			}
			panic(tc.errorf(node, "%s evaluated but not used", node))

		default:
			panic(fmt.Errorf("BUG: checkNodes not implemented for type: %T", node)) // remove.

		}

		i++

	}

	return nodes

}

// checkImport type checks the import statement.
func (tc *typechecker) checkImport(impor *ast.Import) error {
	if tc.opts.modality == scriptMod && impor.Tree != nil {
		panic("BUG: only precompiled packages can be imported in script")
	}

	// Import a precompiled package.
	if isPrecompiled := impor.Tree == nil; isPrecompiled {

		// Load the precompiled package.
		pkg, err := tc.precompiledPkgs.Load(impor.Path)
		if err != nil {
			return tc.errorf(impor, "%s", err)
		}
		precompiledPkg := pkg.(predefinedPackage)
		if precompiledPkg.Name() == "main" {
			return tc.programImportError(impor)
		}

		// Read the declarations from the precompiled package.
		imported := &packageInfo{}
		imported.Declarations = make(map[string]*typeInfo, len(precompiledPkg.DeclarationNames()))
		for n, d := range toTypeCheckerScope(precompiledPkg, tc.opts.modality, false, 0) {
			imported.Declarations[n] = d.t
		}
		imported.Name = precompiledPkg.Name()

		// 'import _ "pkg"': nothing to do.
		if isBlankImport(impor) {
			return nil
		}

		// 'import . "pkg"': add every declaration to the file package block.
		if isPeriodImport(impor) {
			for ident, ti := range imported.Declarations {
				tc.filePackageBlock[ident] = scopeElement{t: ti}
				tc.setImportedButNotUsed(imported.Name, ident)
			}
			return nil
		}

		// Determine the package name.
		var pkgName string
		if impor.Ident == nil {
			pkgName = imported.Name // import "pkg"
		} else {
			pkgName = impor.Ident.Name // import pkgName "pkg"
		}

		// Add the package to the file/package block.
		tc.filePackageBlock[pkgName] = scopeElement{
			t: &typeInfo{value: imported, Properties: propertyIsPackage | propertyHasValue},
		}

		// Set the package as imported but not used.
		for declName := range imported.Declarations {
			tc.setImportedButNotUsed(pkgName, declName)
		}

		return nil
	}

	// Not precompiled package (i.e. a package declared in Scriggo).

	if tc.opts.modality == templateMod {
		tc.templateFileToPackage(impor.Tree)
	}
	if impor.Tree.Nodes[0].(*ast.Package).Name == "main" {
		return tc.programImportError(impor)
	}

	// Check the package and retrieve the package infos.
	err := checkPackage(tc.compilation, impor.Tree.Nodes[0].(*ast.Package), impor.Tree.Path, tc.precompiledPkgs, tc.opts, tc.globalScope)
	if err != nil {
		return err
	}
	imported := tc.compilation.pkgInfos[impor.Tree.Path]

	// import _ "path"
	// {% import _ "path" %}
	if isBlankImport(impor) {
		// Nothing to do.
		return nil
	}

	switch {

	// import "path"
	// {% import "path" %}
	case impor.Ident == nil:

		// {% import "path" %}
		if tc.opts.modality == templateMod {
			for ident, ti := range imported.Declarations {
				tc.filePackageBlock[ident] = scopeElement{t: ti}
				tc.setImportedButNotUsed(imported.Name, ident)
			}
			return nil
		}

		// import "path"
		tc.filePackageBlock[imported.Name] = scopeElement{
			t: &typeInfo{value: imported, Properties: propertyIsPackage | propertyHasValue},
		}
		return nil

	// import . "path"
	// {% import . "path" %}
	case isPeriodImport(impor):
		for ident, ti := range imported.Declarations {
			tc.filePackageBlock[ident] = scopeElement{t: ti}
			tc.setImportedButNotUsed(imported.Name, ident)
		}
		return nil

	// import name "path"
	// {% import name "path" %}
	default:
		tc.filePackageBlock[impor.Ident.Name] = scopeElement{
			t: &typeInfo{
				value:      imported,
				Properties: propertyIsPackage | propertyHasValue,
			},
		}
		// TODO: is the error "imported but not used" correctly reported for
		// this case?
	}

	return nil
}

// makeMacroResultExplicit makes the result argument of a macro explicit.
// It should be called before checking the type of a macro declaration without
// an explicit result type.
func (tc *typechecker) makeMacroResultExplicit(macro *ast.Func) {
	name := formatTypeName[macro.Format]
	scope, ok := tc.universe[name]
	if !ok {
		panic("no type defined for format " + macro.Format.String())
	}
	ident := ast.NewIdentifier(macro.Pos(), name)
	tc.compilation.typeInfos[ident] = scope.t
	macro.Type.Result = []*ast.Parameter{ast.NewParameter(
		// Using (_ string) as return parameter makes the macro return an empty string.
		ast.NewIdentifier(macro.Pos(), "_"), ident,
	)}
}

// checkFunc checks a function.
func (tc *typechecker) checkFunc(node *ast.Func) {

	tc.enterScope()
	tc.addToAncestors(node)
	// Adds parameters to the function body scope.
	tc.addMissingTypes(node.Type)
	isVariadic := node.Type.IsVariadic
	for i, param := range node.Type.Parameters {
		t := tc.checkType(param.Type)
		if param.Ident != nil && !isBlankIdentifier(param.Ident) {
			if isVariadic && i == len(node.Type.Parameters)-1 {
				tc.assignScope(param.Ident.Name, &typeInfo{Type: tc.types.SliceOf(t.Type), Properties: propertyAddressable}, param.Ident)
			} else {
				tc.assignScope(param.Ident.Name, &typeInfo{Type: t.Type, Properties: propertyAddressable}, param.Ident)
			}
		}
	}

	// Adds named return values to the function body scope and add some dummy
	// assignment nodes to initialize them, if necessary.
	//
	// For example:
	//
	//     func f() (x int, slice []int) {
	// 	       return
	//     }
	//
	// is transformed to:
	//
	//    func f() (x int, slice []int) {
	//        x = 0
	//        slice = []int(nil)
	//        return
	//    }
	//
	var initRetParams []ast.Node
	for _, ret := range node.Type.Result {
		t := tc.checkType(ret.Type)
		if ret.Ident != nil && !isBlankIdentifier(ret.Ident) {
			tc.assignScope(ret.Ident.Name, &typeInfo{Type: t.Type, Properties: propertyAddressable}, ret.Ident)
			assignment := ast.NewAssignment(
				ret.Ident.Position,
				[]ast.Expression{ret.Ident},
				ast.AssignmentSimple,
				[]ast.Expression{tc.newPlaceholderFor(t.Type)},
			)
			initRetParams = append(initRetParams, assignment)
		}
	}
	if len(initRetParams) > 0 {
		node.Body.Nodes = append(initRetParams, node.Body.Nodes...)
	}

	node.Body.Nodes = tc.checkNodes(node.Body.Nodes)
	// «If the function's signature declares result parameters, the function
	// body's statement list must end in a terminating statement.»
	//
	// As a special case, if the function is a macro then it is legal to be
	// non-terminating.
	if !tc.terminating && !node.Type.Macro && len(node.Type.Result) > 0 {
		panic(tc.errorf(node, "missing return at end of function"))
	}
	tc.ancestors = tc.ancestors[:len(tc.ancestors)-1]
	tc.exitScope()
}

// checkReturn type checks a return statement.
//
// If the return statement has an expression list and the returning function has
// named return parameters, then an assignment node is returned, that has on the
// left side the list of the function named parameters and on the right side the
// expression list of the return statement; in this way, when checkReturn is
// called, the tree can be changed from
//
//      func F() (a, b int) {
//          return b, a
//      }
//
// to
//
//      func F() (a, b int) {
//          a, b = b, a
//          return
//      }
//
// This simplifies the value swapping by handling it as a generic assignment.
func (tc *typechecker) checkReturn(node *ast.Return) ast.Node {

	fn, _ := tc.currentFunction()
	if fn.Type.Macro {
		return nil
	}

	expected := fn.Type.Result
	got := node.Values

	if len(expected) == 0 && len(got) == 0 {
		return nil
	}

	// Named return arguments with empty return: check if any value has been
	// shadowed.
	if len(expected) > 0 && expected[0].Ident != nil && len(got) == 0 {
		// If "return" belongs to an inner scope (not the function scope).
		_, funcBound := tc.currentFunction()
		if len(tc.scopes) > funcBound {
			for _, e := range expected {
				name := e.Ident.Name
				_, ok := tc.lookupScopes(name, true)
				if ok {
					panic(tc.errorf(node, "%s is shadowed during return", name))
				}
			}
		}
		return nil
	}

	var expectedTypes []reflect.Type
	for _, exp := range expected {
		ti := tc.checkType(exp.Type)
		expectedTypes = append(expectedTypes, ti.Type)
	}

	needsCheck := true
	if len(expected) > 1 && len(got) == 1 {
		if c, ok := got[0].(*ast.Call); ok {
			tis := tc.checkCallExpression(c)
			got = nil
			for _, ti := range tis {
				v := ast.NewCall(c.Pos(), c.Func, c.Args, false)
				tc.compilation.typeInfos[v] = ti
				got = append(got, v)
				needsCheck = false
			}
		}
	}

	if needsCheck {
		for _, g := range got {
			_ = tc.checkExpr(g)
		}
	}

	if len(expected) != len(got) {
		msg := ""
		if len(expected) > len(got) {
			msg = "not enough arguments to return"
		}
		if len(expected) < len(got) {
			msg = "too many arguments to return"
		}
		msg += "\n\thave ("
		for i, x := range got {
			msg += tc.compilation.typeInfos[x].StringWithNumber(false)
			if i != len(got)-1 {
				msg += ", "
			}
		}
		msg += ")\n\twant ("
		for i, typ := range expectedTypes {
			msg += typ.String()
			if i != len(expectedTypes)-1 {
				msg += ", "
			}
		}
		msg += ")"
		panic(tc.errorf(node, msg))
	}

	for i, typ := range expectedTypes {
		x := got[i]
		ti := tc.compilation.typeInfos[x]
		if err := tc.isAssignableTo(ti, x, typ); err != nil {
			if _, ok := err.(invalidTypeInAssignment); ok {
				panic(tc.errorf(node, "%s in return argument", err))
			}
			panic(tc.errorf(node, "%s", err))
		}
		if ti.Nil() {
			ti = tc.nilOf(typ)
			tc.compilation.typeInfos[x] = ti
		} else {
			ti.setValue(typ)
		}
	}

	if len(expected) > 0 && expected[0].Ident != nil && len(got) > 0 {
		lhs := make([]ast.Expression, len(expected))
		for i := range expected {
			lhs[i] = expected[i].Ident
		}
		assign := ast.NewAssignment(nil, lhs, ast.AssignmentSimple, node.Values)
		tc.checkAssignment(assign)
		return assign
	}

	return nil
}

// checkTypeDeclaration checks a type declaration node that can be both a type
// definition or an alias declaration. Returns the type name and a type info
// representing the declared type. If the type declaration has a blank
// identifier as name, an empty string and a nil type info are returned.
//
//  type Int int
//  type Int = int
//
func (tc *typechecker) checkTypeDeclaration(node *ast.TypeDeclaration) (string, *typeInfo) {
	typ := tc.checkType(node.Type)
	if isBlankIdentifier(node.Ident) {
		return "", nil
	}
	name := node.Ident.Name
	if node.IsAliasDeclaration {
		// Return the base type.
		return name, typ
	}
	// Create a new Scriggo type.
	defType := tc.types.DefinedOf(name, typ.Type)
	// Associate to
	//
	//    type T struct { .. }
	//
	// the package in which the type T is declared.
	if defType.Kind() == reflect.Struct {
		tc.structDeclPkg[defType] = tc.path
	}
	return name, &typeInfo{
		Type:       defType,
		Properties: propertyIsType,
	}
}
