// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/compiler/types"
	"github.com/open2b/scriggo/native"
)

var (
	stringerType    = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	envStringerType = reflect.TypeOf((*native.EnvStringer)(nil)).Elem()

	htmlStringerType    = reflect.TypeOf((*native.HTMLStringer)(nil)).Elem()
	htmlEnvStringerType = reflect.TypeOf((*native.HTMLEnvStringer)(nil)).Elem()

	cssStringerType    = reflect.TypeOf((*native.CSSStringer)(nil)).Elem()
	cssEnvStringerType = reflect.TypeOf((*native.CSSEnvStringer)(nil)).Elem()

	jsStringerType    = reflect.TypeOf((*native.JSStringer)(nil)).Elem()
	jsEnvStringerType = reflect.TypeOf((*native.JSEnvStringer)(nil)).Elem()

	jsonStringerType    = reflect.TypeOf((*native.JSONStringer)(nil)).Elem()
	jsonEnvStringerType = reflect.TypeOf((*native.JSONEnvStringer)(nil)).Elem()

	mdStringerType    = reflect.TypeOf((*native.MarkdownStringer)(nil)).Elem()
	mdEnvStringerType = reflect.TypeOf((*native.MarkdownEnvStringer)(nil)).Elem()
)

// templateFileToPackage transforms a tree of a declarations file to a package
// tree, moving the top level nodes, excluding text and comment nodes,
// (partially) transforming "using" nodes, in a package node that becomes the
// only node of tree.
// If tree already has a package, templateFileToPackage does nothing.
func (tc *typechecker) templateFileToPackage(tree *ast.Tree) {
	if len(tree.Nodes) == 1 {
		if _, ok := tree.Nodes[0].(*ast.Package); ok {
			// tree has already a package, do nothing.
			return
		}
	}

	var iteaToDeclarations map[string][]*ast.Identifier
	nodes := make([]ast.Node, 0, len(tree.Nodes)/2)
	for _, n := range tree.Nodes {
		switch n := n.(type) {
		case *ast.Text, *ast.Comment:
		case *ast.Extends, *ast.Import, *ast.Func, *ast.Var, *ast.Const, *ast.TypeDeclaration:
			nodes = append(nodes, n)
		case *ast.Statements:
			nodes = append(nodes, n.Nodes...)
		case *ast.Using:
			iteaName := tc.compilation.generateIteaName()
			iteaDeclaration, statement := tc.explodeUsingStatement(n, iteaName)
			nodes = append(nodes, iteaDeclaration, statement)
			if iteaToDeclarations == nil {
				iteaToDeclarations = map[string][]*ast.Identifier{}
			}
			if iteaHasBeenShadowed(tree.Nodes) {
				iteaToDeclarations[iteaName] = nil
			} else {
				iteaToDeclarations[iteaName] = n.Statement.(*ast.Var).Lhs
			}
		default:
			panic(fmt.Sprintf("BUG: unexpected node %s", n))
		}
	}

	pkg := ast.NewPackage(tree.Pos(), "", nodes)
	pkg.IR.IteaNameToVarIdents = iteaToDeclarations
	tree.Nodes = []ast.Node{pkg}
}

// iteaHasBeenShadowed reports whether the predeclared identifier 'itea' has
// been shadowed by a package-level declaration within nodes.
func iteaHasBeenShadowed(nodes []ast.Node) bool {
	for _, n := range nodes {
		switch n := n.(type) {
		case *ast.Var:
			for _, lh := range n.Lhs {
				if lh.Name == "itea" {
					return true
				}
			}
		case *ast.Func:
			if n.Ident.Name == "itea" {
				return true
			}
		case *ast.TypeDeclaration:
			if n.Ident.Name == "itea" {
				return true
			}
		case *ast.Const:
			for _, lh := range n.Lhs {
				if lh.Name == "itea" {
					return true
				}
			}
		case *ast.Import:
			if n.Ident != nil && n.Ident.Name == "itea" {
				return true
			}
		case *ast.Statements:
			if iteaHasBeenShadowed(n.Nodes) {
				return true
			}
		}
	}
	return false
}

// checkNodesInNewScopeError calls checkNodesInNewScope returning checking errors.
func (tc *typechecker) checkNodesInNewScopeError(block ast.Node, nodes []ast.Node) (newNodes []ast.Node, err error) {
	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(*CheckingError); ok {
				err = rerr
			} else {
				panic(r)
			}
		}
	}()
	tc.scopes.Enter(block)
	newNodes = tc.checkNodes(nodes)
	tc.scopes.Exit()
	return
}

// checkNodesInNewScope type checks nodes in a new scope. Panics on error.
func (tc *typechecker) checkNodesInNewScope(block ast.Node, nodes []ast.Node) []ast.Node {
	tc.scopes.Enter(block)
	nodes = tc.checkNodes(nodes)
	tc.scopes.Exit()
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
			node.Nodes = tc.checkNodesInNewScope(node, node.Nodes)

		case *ast.Statements:
			node.Nodes = tc.checkNodes(node.Nodes)

		case *ast.If:
			tc.scopes.Enter(node)
			if node.Init != nil {
				tc.checkNodes([]ast.Node{node.Init})
			}
			ti := tc.checkExpr(node.Condition)
			if ti.Nil() {
				panic(tc.errorf(node.Condition, "use of untyped nil"))
			}
			if ti.Type.Kind() != reflect.Bool {
				if tc.opts.mod != programMod {
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
			node.Then.Nodes = tc.checkNodesInNewScope(node.Then, node.Then.Nodes)
			terminating := tc.terminating
			if node.Else == nil {
				terminating = false
			} else {
				switch els := node.Else.(type) {
				case *ast.Block:
					els.Nodes = tc.checkNodesInNewScope(els, els.Nodes)
				case *ast.If:
					_ = tc.checkNodes([]ast.Node{els})
				}
				terminating = terminating && tc.terminating
			}
			tc.scopes.Exit()
			tc.terminating = terminating

		case *ast.For:
			tc.scopes.Enter(node)
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
			node.Body = tc.checkNodesInNewScope(node, node.Body)
			tc.removeLastAncestor()
			tc.scopes.Exit()
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
			tc.scopes.Enter(node)
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
			node.Body = tc.checkNodesInNewScope(node, node.Body)
			tc.removeLastAncestor()
			tc.scopes.Exit()
			tc.terminating = !tc.hasBreak[node]

		case *ast.Assignment:
			tc.checkGenericAssignmentNode(node)
			tc.terminating = false

		case *ast.Break:
			found := false
			label := tc.scopes.UseLabel("break", node.Label)
			for i := len(tc.ancestors) - 1; i >= 0 && !found; i-- {
				switch n := tc.ancestors[i].(type) {
				case *ast.For, *ast.ForRange, *ast.Switch, *ast.TypeSwitch, *ast.Select:
					if label == nil || label.Statement == n {
						tc.hasBreak[n] = true
						found = true
					}
				case *ast.Func:
					i = -1
				}
			}
			if !found {
				if label != nil {
					panic(tc.errorf(node.Label, "invalid break label %s", node.Label.Name))
				}
				panic(tc.errorf(node, "break is not in a loop, switch, or select"))
			}
			tc.terminating = false

		case *ast.Continue:
			found := false
			label := tc.scopes.UseLabel("continue", node.Label)
			for i := len(tc.ancestors) - 1; i >= 0 && !found; i-- {
				switch n := tc.ancestors[i].(type) {
				case *ast.For, *ast.ForRange:
					if label == nil || label.Statement == n {
						found = true
					}
				case *ast.Func:
					i = -1
				}
			}
			if !found {
				if label != nil {
					panic(tc.errorf(node.Label, "invalid continue label %s", node.Label.Name))
				}
				panic(tc.errorf(node, "continue is not in a loop"))
			}
			tc.terminating = false

		case *ast.Fallthrough:
			outOfPlace := true
			if len(tc.ancestors) > 0 {
				parent := tc.ancestors[len(tc.ancestors)-1]
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
						parent = tc.ancestors[len(tc.ancestors)-2]
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
			tc.scopes.Enter(node)
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
				tc.scopes.Enter(cas)
				tc.addToAncestors(cas)
				cas.Body = tc.checkNodes(cas.Body)
				tc.removeLastAncestor()
				tc.scopes.Exit()
				if !hasFallthrough && len(cas.Body) > 0 {
					_, hasFallthrough = cas.Body[len(cas.Body)-1].(*ast.Fallthrough)
				}
				terminating = terminating && (tc.terminating || hasFallthrough)
			}
			tc.removeLastAncestor()
			tc.scopes.Exit()
			tc.terminating = terminating && !tc.hasBreak[node] && positionOfDefault != nil

		case *ast.TypeSwitch:
			terminating := true
			tc.scopes.Enter(node)
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
			var name string
			var ti *typeInfo
			if a := node.Assignment; a.Type == ast.AssignmentDeclaration {
				ident := a.Lhs[0].(*ast.Identifier)
				name = ident.Name
				ti = &typeInfo{Type: t.Type, Properties: propertyAddressable}
				tc.scopes.Declare(name, ti, ident)
			}
			var positionOfDefault *ast.Position
			var positionOfNil *ast.Position
			positionOf := map[reflect.Type]*ast.Position{}
			for _, cas := range node.Cases {
				tc.scopes.Enter(cas)
				tc.addToAncestors(cas)
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
					if name != "" && len(cas.Expressions) == 1 {
						ti := &typeInfo{Type: t.Type, Properties: propertyAddressable}
						ident := ast.NewIdentifier(cas.Expressions[0].Pos(), name)
						tc.scopes.Declare(name, ti, ident)
					}
					// Check duplicate.
					if pos, ok := positionOf[t.Type]; ok {
						panic(tc.errorf(cas, "duplicate case %v in type switch\n\tprevious case at %s", ex, pos))
					}
					positionOf[t.Type] = ex.Pos()
				}
				if name != "" && len(cas.Expressions) != 1 {
					tc.scopes.Declare(name, ti, ast.NewIdentifier(cas.Position, name))
				}
				cas.Body = tc.checkNodes(cas.Body)
				used := name != "" && tc.scopes.Use(name)
				tc.removeLastAncestor()
				tc.scopes.Exit()
				if used {
					tc.scopes.Use(name)
				}
				terminating = terminating && tc.terminating
			}
			tc.removeLastAncestor()
			tc.scopes.Exit()
			tc.terminating = terminating && !tc.hasBreak[node] && positionOfDefault != nil

		case *ast.Select:
			tc.scopes.Enter(node)
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
				cas.Body = tc.checkNodesInNewScope(node, cas.Body)
				terminating = terminating && tc.terminating
			}
			tc.removeLastAncestor()
			tc.scopes.Exit()
			tc.terminating = terminating && !tc.hasBreak[node]

		case *ast.Const:
			tc.checkConstantDeclaration(node)
			tc.terminating = false

		case *ast.Var:
			tc.checkVariableDeclaration(node)
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
					if len(tis) == 2 && types.Implements(tis[1].Type, errorType) {
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
					err := checkShow(ti.Type, node.Context)
					if err != nil {
						panic(tc.errorf(node, "cannot show %s (%s)", expr, err))
					}
				}
				ti := tis.TypeInfo()
				ti.setValue(nil)
			}

			tc.terminating = false

		case *ast.Using:

			iteaName := tc.compilation.generateIteaName()

			iteaDeclaration, statement := tc.explodeUsingStatement(node, iteaName)

			// Type check the dummy assignment of the 'using' statement, along
			// with its content, and transform the tree.
			nn := tc.checkNodes([]ast.Node{iteaDeclaration})
			nodes = append(nodes[:i], append(nn, nodes[i:]...)...)
			i += len(nn)

			// Type check the affected statement of the 'using' and transform
			// the tree.
			withinStmt := tc.withinUsingAffectedStmt
			tc.withinUsingAffectedStmt = true
			backupIteaName := tc.compilation.iteaName
			tc.compilation.iteaName = iteaName
			nn = tc.checkNodes([]ast.Node{statement})
			nodes = append(nodes[:i], append(nn, nodes[i+1:]...)...)
			i += len(nn)
			tc.withinUsingAffectedStmt = withinStmt
			tc.compilation.iteaName = backupIteaName

			continue nodesLoop

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
				case "append", "cap", "complex", "imag", "len", "make", "new", "real":
					panic(tc.errorf(node, "defer discards result of %s", call))
				case "recover":
					// The statement "defer recover()" is a special case
					// implemented by the emitter.
				case "close", "copy", "delete", "panic", "print", "println":
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
				case "append", "cap", "complex", "imag", "len", "make", "new", "real":
					panic(tc.errorf(node, "go discards result of %s", call))
				case "close", "copy", "delete", "panic", "print", "println", "recover":
					tc.compilation.typeInfos[call.Func] = deferGoBuiltin(name)
				}
			}
			if ti.IsType() {
				panic(tc.errorf(node, "go requires function call, not conversion"))
			}
			if !tc.opts.allowGoStmt {
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
			if node.Op != ast.OperatorReceive {
				panic(tc.errorf(node, "%s evaluated but not used", node))
			}
			ti.setValue(nil)

		case *ast.Goto:
			tc.scopes.UseLabel("goto", node.Label)

		case *ast.Label:
			tc.scopes.DeclareLabel(node)
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
			if fun, ok := node.(*ast.Func); ok && fun.Ident != nil && tc.opts.mod != programMod {
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
				newNodes := []ast.Node{varDecl, nodeAssign}

				_ = tc.checkNodes(newNodes)
				// Append the new nodes removing the function literal.
				nodes = append(nodes[:i], append(newNodes, nodes[i+1:]...)...)
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
			if tc.opts.mod == templateMod {
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
	if tc.opts.mod == scriptMod && impor.Tree != nil {
		panic("BUG: native packages only can be imported in script")
	}

	// Import a native package.
	if impor.Tree == nil {

		// Import the native package.
		if tc.importer == nil {
			return tc.errorf(impor, "cannot find package %q", impor.Path)
		}
		pkg, err := tc.importer.Import(impor.Path)
		if err != nil {
			return tc.errorf(impor, "%s", err)
		}
		if pkg == nil {
			return tc.errorf(impor, "cannot find package %q", impor.Path)
		}

		// 'import _ "pkg"': nothing to do.
		if isBlankImport(impor) {
			return nil
		}

		// Read the declarations from the native package.
		imported := &packageInfo{}
		imported.Declarations = map[string]*typeInfo{}
		for n, d := range toTypeCheckerScope(pkg, tc.opts.mod, false, 0) {
			imported.Declarations[n] = d.ti
		}
		imported.Name = pkg.PackageName()

		// {% import "path" for N1, N2 %}
		//
		// where "path" is a native package path.
		for _, ident := range impor.For {
			ti, ok := imported.Declarations[ident.Name]
			if !ok {
				return tc.errorf(impor, "undefined: %s", ident)
			}
			tc.scopes.Declare(ident.Name, ti, impor)
			return nil
		}

		// 'import . "pkg"': add every declaration to the file package block.
		if isPeriodImport(impor) {
			for ident, ti := range imported.Declarations {
				tc.scopes.Declare(ident, ti, impor)
			}
			return nil
		}

		// Determine the package name.
		var pkgName string
		if impor.Ident == nil {
			// import "pkg"
			pkgName = imported.Name
			if pkgName == "main" {
				return tc.errorf(impor, `import "%s" is a program, not an importable package`, impor.Path)
			}
			// Don't allow '$' as first character because used for Scriggo special names.
			if len(pkgName) > 0 && pkgName[0] == '$' {
				pkgName = ""
			}
		} else {
			pkgName = impor.Ident.Name // import pkgName "pkg"
		}
		if pkgName == "init" {
			return tc.errorf(impor, "cannot import package as init - init must be a func")
		}

		// Add the package to the file/package block.
		tc.assignScope(pkgName, &typeInfo{value: imported, Properties: propertyIsPackage | propertyHasValue}, impor)

		return nil
	}

	// Non-native package (i.e. a package declared in Scriggo).

	if tc.opts.mod == templateMod {
		tc.templateFileToPackage(impor.Tree)
	}
	if impor.Tree.Nodes[0].(*ast.Package).Name == "main" {
		return tc.programImportError(impor)
	}

	// Check the package and retrieve the package infos.
	err := checkPackage(tc.compilation, impor.Tree.Nodes[0].(*ast.Package), impor.Tree.Path, tc.importer, tc.opts, tc.compilation.extendingTrees[impor.Tree])
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

	// {% import "path" for N1, N2 %}
	case impor.For != nil:
		for _, ident := range impor.For {
			ti, ok := imported.Declarations[ident.Name]
			if !ok {
				return tc.errorf(impor, "undefined: %s", ident)
			}
			tc.scopes.Declare(ident.Name, ti, impor)
		}

	// import "path"
	// {% import "path" %}
	case impor.Ident == nil:

		// {% import "path" %}
		if tc.opts.mod == templateMod {
			return tc.errorf(impor, `template file import requires name or "import for" form`)
		}

		// import "path"
		tc.scopes.Declare(imported.Name, &typeInfo{value: imported, Properties: propertyIsPackage | propertyHasValue}, impor)
		return nil

	// import . "path"
	// {% import . "path" %}
	case isPeriodImport(impor):
		for ident, ti := range imported.Declarations {
			tc.scopes.Declare(ident, ti, impor)
		}
		return nil

	// import name "path"
	// {% import name "path" %}
	default:
		ti := &typeInfo{
			value:      imported,
			Properties: propertyIsPackage | propertyHasValue,
		}
		tc.scopes.Declare(impor.Ident.Name, ti, impor)
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
	ti, ok := tc.scopes.Universe(name)
	if !ok {
		panic("no type defined for format " + macro.Format.String())
	}
	ident := ast.NewIdentifier(macro.Pos(), name)
	tc.compilation.typeInfos[ident] = ti
	macro.Type.Result = []*ast.Parameter{ast.NewParameter(
		// Using (_ string) as return parameter makes the macro return an empty string.
		ast.NewIdentifier(macro.Pos(), "_"), ident,
	)}
}

// checkFunc checks a function.
func (tc *typechecker) checkFunc(node *ast.Func) {

	tc.scopes.Enter(node)
	tc.addToAncestors(node)

	// Adds parameters to the function body scope.
	t := node.Type.Reflect
	for i := 0; i < t.NumIn(); i++ {
		param := node.Type.Parameters[i]
		if param.Ident != nil && !isBlankIdentifier(param.Ident) {
			tc.scopes.Declare(param.Ident.Name, &typeInfo{Type: t.In(i), Properties: propertyAddressable}, param.Ident)
			tc.scopes.Use(param.Ident.Name)
		}
	}

	// Adds named return values to the function body scope and add some dummy
	// assignment nodes to initialize them, if necessary.
	//
	// For example:
	//
	//     func f() (x int, slice []int) {
	//         return
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
	// Any blank identifier is transformed into something like "$blank0" so
	// that it can be assigned as any other parameter.
	var initRetParams []ast.Node
	for i := 0; i < t.NumOut(); i++ {
		ret := node.Type.Result[i]
		if ret.Ident != nil {
			if isBlankIdentifier(ret.Ident) {
				ret.Ident.Name = "$blank" + strconv.Itoa(i)
			}
			tc.scopes.Declare(ret.Ident.Name, &typeInfo{Type: t.Out(i), Properties: propertyAddressable}, ret.Ident)
			tc.scopes.Use(ret.Ident.Name)
			assignment := ast.NewAssignment(
				ret.Ident.Position,
				[]ast.Expression{ret.Ident},
				ast.AssignmentSimple,
				[]ast.Expression{tc.newPlaceholderFor(t.Out(i))},
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
	tc.scopes.Exit()
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

	fn := tc.scopes.CurrentFunction()
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
		for _, e := range expected {
			if name := e.Ident.Name; name != "_" {
				if _, ident, _ := tc.scopes.LookupInFunc(name); ident != e.Ident {
					panic(tc.errorf(e.Ident, "%s is shadowed during return", name))
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
		return name, &typeInfo{Type: typ.Type, Alias: node.Ident.Name, Properties: typ.Properties}
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

// explodeUsingStatement explodes an 'using' statement.
func (tc *typechecker) explodeUsingStatement(using *ast.Using, iteaIdent string) (*ast.Var, ast.Node) {

	// Make the type explicit, if necessary.
	if using.Type == nil {
		name := formatTypeName[using.Format]
		ti, ok := tc.scopes.Universe(name)
		if !ok {
			panic("no type defined for format " + using.Format.String())
		}
		ident := ast.NewIdentifier(using.Pos(), name)
		tc.compilation.typeInfos[ident] = ti
		using.Type = ident
	}

	var itea ast.Expression
	switch typ := using.Type.(type) {
	case *ast.Identifier:
		itea = ast.NewCall(nil,
			ast.NewFunc(nil, nil,
				ast.NewFuncType(nil, true, nil, []*ast.Parameter{ast.NewParameter(nil, typ)}, false),
				using.Body, false, using.Format),
			nil, false)
	case *ast.FuncType:
		itea = ast.NewFunc(nil, nil, typ, using.Body, false, using.Format)
	default:
		panic("BUG: the parser should not allow this")
	}

	iteaDeclaration := ast.NewVar(
		nil,
		[]*ast.Identifier{ast.NewIdentifier(nil, iteaIdent)},
		nil,
		[]ast.Expression{itea},
	)
	uc := usingCheck{
		itea: iteaDeclaration,
		pos:  using.Position,
		typ:  using.Type,
	}
	if tc.compilation.iteaToUsingCheck == nil {
		tc.compilation.iteaToUsingCheck = map[string]usingCheck{iteaIdent: uc}
	} else {
		tc.compilation.iteaToUsingCheck[iteaIdent] = uc
	}

	return iteaDeclaration, using.Statement
}

var byteSliceType = reflect.TypeOf([]byte(nil))
var timeType = reflect.TypeOf(time.Time{})

// checkShow type checks the show of a value of type t in context ctx.
func checkShow(t reflect.Type, ctx ast.Context) error {
	if t == emptyInterfaceType {
		return nil
	}
	kind := t.Kind()
	switch ctx {
	case ast.ContextText, ast.ContextTag, ast.ContextQuotedAttr, ast.ContextUnquotedAttr,
		ast.ContextCSSString, ast.ContextJSString, ast.ContextJSONString,
		ast.ContextTabCodeBlock, ast.ContextSpacesCodeBlock:
		switch {
		case kind == reflect.String:
		case reflect.Bool <= kind && kind <= reflect.Complex128:
		case ctx == ast.ContextCSSString && t == byteSliceType:
		case t.Implements(stringerType):
		case t.Implements(envStringerType):
		case t.Implements(errorType):
		default:
			return fmt.Errorf("cannot show type %s as %s", t, ctx)
		}
	case ast.ContextHTML:
		switch {
		case kind == reflect.String:
		case reflect.Bool <= kind && kind <= reflect.Complex128:
		case t == byteSliceType:
		case t.Implements(stringerType):
		case t.Implements(envStringerType):
		case t.Implements(htmlStringerType):
		case t.Implements(htmlEnvStringerType):
		case t.Implements(errorType):
		default:
			return fmt.Errorf("cannot show type %s as HTML", t)
		}
	case ast.ContextCSS:
		switch {
		case kind == reflect.String:
		case reflect.Int <= kind && kind <= reflect.Float64:
		case t == byteSliceType:
		case t.Implements(stringerType):
		case t.Implements(envStringerType):
		case t.Implements(cssStringerType):
		case t.Implements(cssEnvStringerType):
		case t.Implements(errorType):
		default:
			return fmt.Errorf("cannot show type %s as CSS", t)
		}
	case ast.ContextJS:
		err := checkShowJS(t, nil)
		if err != nil {
			return err
		}
	case ast.ContextJSON:
		err := checkShowJSON(t, nil)
		if err != nil {
			return err
		}
	case ast.ContextMarkdown:
		switch {
		case kind == reflect.String:
		case reflect.Bool <= kind && kind <= reflect.Complex128:
		case t.Implements(stringerType):
		case t.Implements(envStringerType):
		case t.Implements(mdStringerType):
		case t.Implements(mdEnvStringerType):
		case t.Implements(htmlStringerType):
		case t.Implements(htmlEnvStringerType):
		case t.Implements(errorType):
		default:
			return fmt.Errorf("cannot show type %s as Markdown", t)
		}
	default:
		panic("unexpected context")
	}
	return nil
}

// checkShowJS reports whether a type can be shown as JavaScript. It returns
// an error if the type cannot be shown.
func checkShowJS(t reflect.Type, types []reflect.Type) error {
	for _, typ := range types {
		if t == typ {
			return nil
		}
	}
	kind := t.Kind()
	if reflect.Bool <= kind && kind <= reflect.Float64 || kind == reflect.String ||
		t == timeType ||
		t.Implements(jsStringerType) ||
		t.Implements(jsEnvStringerType) ||
		t.Implements(errorType) {
		return nil
	}
	switch kind {
	case reflect.Array:
		if err := checkShowJS(t.Elem(), append(types, t)); err != nil {
			return fmt.Errorf("cannot show array of %s as JavaScript", t.Elem())
		}
	case reflect.Interface:
	case reflect.Map:
		key := t.Key().Kind()
		switch {
		case key == reflect.String:
		case reflect.Bool <= key && key <= reflect.Complex128:
		case t.Implements(stringerType):
		case t.Implements(envStringerType):
		default:
			return fmt.Errorf("cannot show map with %s key as JavaScript", t.Key())
		}
		te := t.Elem()
		err := checkShowJS(te, append(types, t))
		if err != nil {
			return fmt.Errorf("cannot show map with %s element as JavaScript", t.Elem())
		}
	case reflect.Ptr, reflect.UnsafePointer:
		return checkShowJS(t.Elem(), append(types, t))
	case reflect.Slice:
		if err := checkShowJS(t.Elem(), append(types, t)); err != nil {
			return fmt.Errorf("cannot show slice of %s as JavaScript", t.Elem())
		}
	case reflect.Struct:
		n := t.NumField()
		for i := 0; i < n; i++ {
			field := t.Field(i)
			if field.PkgPath == "" {
				if err := checkShowJS(field.Type, append(types, t)); err != nil {
					return fmt.Errorf("cannot show struct containing %s as JavaScript", field.Type)
				}
			}
		}
	default:
		return fmt.Errorf("cannot show type %s as JavaScript", t)
	}
	return nil
}

// checkShowJSON reports whether a type can be shown as JSON. It returns an
// error if the type cannot be shown.
func checkShowJSON(t reflect.Type, types []reflect.Type) error {
	for _, typ := range types {
		if t == typ {
			return nil
		}
	}
	kind := t.Kind()
	if reflect.Bool <= kind && kind <= reflect.Float64 || kind == reflect.String ||
		t == timeType ||
		t.Implements(jsonStringerType) ||
		t.Implements(jsonEnvStringerType) ||
		t.Implements(errorType) {
		return nil
	}
	switch kind {
	case reflect.Array:
		if err := checkShowJSON(t.Elem(), append(types, t)); err != nil {
			return fmt.Errorf("cannot show array of %s as JSON", t.Elem())
		}
	case reflect.Interface:
	case reflect.Map:
		key := t.Key().Kind()
		switch {
		case key == reflect.String:
		case reflect.Bool <= key && key <= reflect.Complex128:
		case t.Implements(stringerType):
		case t.Implements(envStringerType):
		default:
			return fmt.Errorf("cannot show map with %s key as JSON", t.Key())
		}
		err := checkShowJSON(t.Elem(), append(types, t))
		if err != nil {
			return fmt.Errorf("cannot show map with %s element as JSON", t.Elem())
		}
	case reflect.Ptr, reflect.UnsafePointer:
		return checkShowJSON(t.Elem(), append(types, t))
	case reflect.Slice:
		if err := checkShowJSON(t.Elem(), append(types, t)); err != nil {
			return fmt.Errorf("cannot show slice of %s as JSON", t.Elem())
		}
	case reflect.Struct:
		n := t.NumField()
		for i := 0; i < n; i++ {
			field := t.Field(i)
			if field.PkgPath == "" {
				if err := checkShowJSON(field.Type, append(types, t)); err != nil {
					return fmt.Errorf("cannot show struct containing %s as JSON", field.Type)
				}
			}
		}
	default:
		return fmt.Errorf("cannot show type %s as JSON", t)
	}
	return nil
}
