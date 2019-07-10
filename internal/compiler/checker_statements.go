// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"
	"strings"

	"scriggo/ast"
)

// templateToPackage extract first-level declarations in tree and appends them
// to a package, which will be the only node of tree. If tree is already a
// package, templateToPackage does nothing.
func (tc *typechecker) templateToPackage(tree *ast.Tree, path string) error {
	// tree is already a package: do nothing and return.
	if len(tree.Nodes) == 1 {
		if _, ok := tree.Nodes[0].(*ast.Package); ok {
			return nil
		}
	}
	currentPath := tc.path
	tc.path = path
	nodes := []ast.Node{}
	for _, n := range tree.Nodes {
		switch n := n.(type) {
		case *ast.Comment:
		case *ast.Macro, *ast.Var, *ast.TypeDeclaration, *ast.Const, *ast.Import, *ast.Extends:
			nodes = append(nodes, n)
		default:
			// TODO(Gianluca): review error.
			if txt, ok := n.(*ast.Text); ok && len(strings.TrimSpace(string(txt.Text))) == 0 {
				// Nothing to do
			} else {
				return tc.errorf(n, "unexpected %T node as top-level declaration in template", n)
			}
		}
	}
	tree.Nodes = []ast.Node{
		ast.NewPackage(tree.Pos(), "", nodes),
	}
	tc.path = currentPath
	return nil
}

// checkNodesInNewScopeError calls checkNodesInNewScope returning checking errors.
func (tc *typechecker) checkNodesInNewScopeError(nodes []ast.Node) error {
	tc.addScope()
	err := tc.checkNodesError(nodes)
	if err != nil {
		return err
	}
	tc.removeCurrentScope()
	return nil
}

// checkNodesInNewScope type checks nodes in a new scope. Panics on error.
func (tc *typechecker) checkNodesInNewScope(nodes []ast.Node) {
	tc.addScope()
	tc.checkNodes(nodes)
	tc.removeCurrentScope()
}

// checkNodesError calls checkNodes catching panics and returning their errors
// as return parameter.
func (tc *typechecker) checkNodesError(nodes []ast.Node) (err error) {
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
		tc.checkNodes(nodes)
	}()
	return err
}

// checkNodes type checks one or more statements. Panics on error.
func (tc *typechecker) checkNodes(nodes []ast.Node) {

	tc.terminating = false

	for i, node := range nodes {

		switch node := node.(type) {

		case *ast.Import:
			if tc.opts.DisallowImports {
				panic(tc.errorf(node, "\"import\" statement not available"))
			}
			if node.Tree == nil {
				// Import statement in script.
				pkg, err := tc.predefinedPkgs.Load(node.Path)
				if err != nil {
					panic(tc.errorf(node, "%s", err))
				}
				predefinedPkg := pkg.(predefinedPackage)
				declarations := predefinedPkg.DeclarationNames()
				importedPkg := &PackageInfo{}
				importedPkg.Declarations = make(map[string]*TypeInfo, len(declarations))
				for n, d := range ToTypeCheckerScope(predefinedPkg) {
					importedPkg.Declarations[n] = d.t
				}
				importedPkg.Name = predefinedPkg.Name()
				if node.Ident == nil {
					tc.filePackageBlock[importedPkg.Name] = scopeElement{t: &TypeInfo{value: importedPkg, Properties: PropertyIsPackage}}
					tc.unusedImports[importedPkg.Name] = nil
				} else {
					switch node.Ident.Name {
					case "_":
					case ".":
						tc.unusedImports[importedPkg.Name] = nil
						for ident, ti := range importedPkg.Declarations {
							tc.unusedImports[importedPkg.Name] = append(tc.unusedImports[importedPkg.Name], ident)
							tc.filePackageBlock[ident] = scopeElement{t: ti}
						}
					default:
						tc.filePackageBlock[node.Ident.Name] = scopeElement{t: &TypeInfo{value: importedPkg, Properties: PropertyIsPackage}}
						tc.unusedImports[node.Ident.Name] = nil
					}
				}
			} else {
				// Imports a template page in templates.
				if node.Ident != nil && node.Ident.Name == "_" {
					// Nothing to do.
				} else {
					err := tc.templateToPackage(node.Tree, node.Path)
					if err != nil {
						panic(err)
					}
					pkgInfos := map[string]*PackageInfo{}
					err = checkPackage(node.Tree.Nodes[0].(*ast.Package), node.Path, nil, nil, pkgInfos, tc.opts)
					if err != nil {
						panic(err)
					}
					// TypeInfos of imported packages in templates are
					// "manually" added to the map of typeinfos of typechecker.
					for k, v := range pkgInfos[node.Path].TypeInfo {
						tc.TypeInfo[k] = v
					}
					importedPkg, ok := pkgInfos[node.Path]
					if !ok {
						panic(fmt.Errorf("cannot find path %q inside pkgInfos (%v)", node.Path, pkgInfos)) // TODO(Gianluca): remove.
					}
					if node.Ident == nil {
						tc.unusedImports[importedPkg.Name] = nil
						for ident, ti := range importedPkg.Declarations {
							tc.unusedImports[importedPkg.Name] = append(tc.unusedImports[importedPkg.Name], ident)
							tc.filePackageBlock[ident] = scopeElement{t: ti}
						}
					} else {
						switch node.Ident.Name {
						case "_":
						case ".":
							tc.unusedImports[importedPkg.Name] = nil
							for ident, ti := range importedPkg.Declarations {
								tc.unusedImports[importedPkg.Name] = append(tc.unusedImports[importedPkg.Name], ident)
								tc.filePackageBlock[ident] = scopeElement{t: ti}
							}
						default:
							tc.filePackageBlock[node.Ident.Name] = scopeElement{
								t: &TypeInfo{
									value:      importedPkg,
									Properties: PropertyIsPackage,
								},
							}
							tc.unusedImports[node.Ident.Name] = nil
						}
					}
				}
			}

		case *ast.Text:

		case *ast.Include:
			// TODO(Gianluca): can node.Tree.Nodes be nil?
			tc.checkNodes(node.Tree.Nodes)

		case *ast.Block:
			tc.checkNodesInNewScope(node.Nodes)

		case *ast.If:
			tc.addScope()
			if node.Assignment != nil {
				tc.checkAssignment(node.Assignment)
			}
			ti := tc.checkExpression(node.Condition)
			if ti.Type.Kind() != reflect.Bool {
				panic(tc.errorf(node.Condition, "non-bool %s (type %v) used as if condition", node.Condition, ti.ShortString()))
			}
			ti.setValue(nil)
			tc.checkNodesInNewScope(node.Then.Nodes)
			terminating := tc.terminating
			if node.Else == nil {
				terminating = false
			} else {
				switch els := node.Else.(type) {
				case *ast.Block:
					tc.checkNodesInNewScope(els.Nodes)
				case *ast.If:
					tc.checkNodes([]ast.Node{els})
				}
				terminating = terminating && tc.terminating
			}
			tc.removeCurrentScope()
			tc.terminating = terminating

		case *ast.For:
			tc.addScope()
			tc.addToAncestors(node)
			if node.Init != nil {
				tc.checkAssignment(node.Init)
			}
			if node.Condition != nil {
				ti := tc.checkExpression(node.Condition)
				if ti.Type.Kind() != reflect.Bool {
					panic(tc.errorf(node.Condition, "non-bool %s (type %v) used as for condition", node.Condition, ti.ShortString()))
				}
				ti.setValue(nil)
			}
			if node.Post != nil {
				tc.checkAssignment(node.Post)
			}
			tc.checkNodesInNewScope(node.Body)
			tc.removeLastAncestor()
			tc.removeCurrentScope()
			tc.terminating = node.Condition == nil && !tc.hasBreak[node]

		case *ast.ForRange:
			tc.addScope()
			tc.addToAncestors(node)
			// Check range expression.
			expr := node.Assignment.Values[0]
			ti := tc.checkExpression(expr)
			if ti.Nil() {
				panic(tc.errorf(node, "cannot range over nil"))
			}
			ti.setValue(nil)
			maxVars := 2
			vars := node.Assignment.Variables
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
					panic(tc.errorf(node.Assignment.Values[0], "invalid operation: range %s (receive from send-only type %s)", expr, ti.String()))
				}
				typ1 = typ.Elem()
				maxVars = 1
			default:
				panic(tc.errorf(node.Assignment.Values[0], "cannot range over %s (type %s)", expr, ti))
			}
			// Check variables.
			if vars != nil {
				if len(vars) > maxVars {
					panic(tc.errorf(node, "too many variables in range"))
				}
				ti1 := &TypeInfo{Type: typ1, Properties: PropertyAddressable}
				declaration := node.Assignment.Type == ast.AssignmentDeclaration
				tc.assignSingle(node.Assignment, vars[0], nil, ti1, nil, declaration, false)
				if len(vars) == 2 {
					tc.assignSingle(node.Assignment, vars[1], nil, &TypeInfo{Type: typ2}, nil, declaration, false)
				}
			}
			tc.checkNodesInNewScope(node.Body)
			tc.removeLastAncestor()
			tc.removeCurrentScope()
			tc.terminating = !tc.hasBreak[node]

		case *ast.Assignment:
			tc.checkAssignment(node)
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

		case *ast.Return:
			tc.checkReturn(node)
			tc.terminating = true

		case *ast.Switch:
			tc.addScope()
			tc.addToAncestors(node)
			// Checks the init.
			if node.Init != nil {
				tc.checkAssignment(node.Init)
			}
			// Checks the expression.
			typ := boolType
			var ti *TypeInfo
			if node.Expr != nil {
				ti = tc.checkExpression(node.Expr)
				if ti.Nil() {
					panic(tc.errorf(node, "use of untyped nil"))
				}
				ti.setValue(nil)
				typ = ti.Type
			}
			// Checks the cases.
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
					t := tc.checkExpression(ex)
					if err := isAssignableTo(t, ex, typ); err != nil {
						if _, ok := err.(invalidTypeInAssignment); ok {
							var ne string
							if node.Expr != nil {
								ne = " on " + node.Expr.String()
							}
							panic(tc.errorf(cas, "invalid case %s in switch%s (mismatched types %s and %s)", ex, ne, t.ShortString(), typ))
						}
						panic(tc.errorf(cas, "%s", err))
					}
					if t.IsConstant() {
						if typ.Kind() != reflect.Bool {
							// Check duplicate.
							value := typedValue(t, typ)
							if pos, ok := positionOf[value]; ok {
								panic(tc.errorf(cas, "duplicate case %v in switch\n\tprevious case at %s", ex, pos))
							}
							positionOf[value] = ex.Pos()
						}
					}
					t.setValue(typ)
				}
				tc.checkNodesInNewScope(cas.Body)
				hasFallthrough = hasFallthrough || cas.Fallthrough
				terminating = terminating && (tc.terminating || hasFallthrough)
			}
			tc.removeLastAncestor()
			tc.removeCurrentScope()
			tc.terminating = terminating && !tc.hasBreak[node] && positionOfDefault != nil

		case *ast.TypeSwitch:
			terminating := true
			tc.addScope()
			tc.addToAncestors(node)
			if node.Init != nil {
				tc.checkAssignment(node.Init)
			}
			ta := node.Assignment.Values[0].(*ast.TypeAssertion)
			t := tc.checkExpression(ta.Expr)
			if t.Type.Kind() != reflect.Interface {
				panic(tc.errorf(node, "cannot type switch on non-interface value %v (type %s)", ta.Expr, t.ShortString()))
			}
			if len(node.Assignment.Variables) == 1 {
				n := ast.NewAssignment(
					node.Assignment.Pos(),
					[]ast.Expression{node.Assignment.Variables[0]},
					node.Assignment.Type,
					[]ast.Expression{ta.Expr},
				)
				tc.checkAssignment(n)
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
					t := tc.typeof(expr, noEllipses)
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
					tc.TypeInfo[expr] = t
					// Check duplicate.
					if pos, ok := positionOf[t.Type]; ok {
						panic(tc.errorf(cas, "duplicate case %v in type switch\n\tprevious case at %s", ex, pos))
					}
					positionOf[t.Type] = ex.Pos()
				}
				tc.checkNodesInNewScope(cas.Body)
				terminating = terminating && tc.terminating
			}
			tc.removeLastAncestor()
			tc.removeCurrentScope()
			tc.terminating = terminating && !tc.hasBreak[node] && positionOfDefault != nil

		case *ast.Select:
			tc.addScope()
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
					_ = tc.checkExpression(comm)
					if recv, ok := comm.(*ast.UnaryOperator); !ok || recv.Op != ast.OperatorReceive {
						panic(tc.errorf(node, "select case must be receive, send or assign recv"))
					}
				case *ast.Assignment:
					tc.checkAssignment(comm)
					if comm.Type != ast.AssignmentSimple && comm.Type != ast.AssignmentDeclaration {
						panic(tc.errorf(node, "select case must be receive, send or assign recv"))
					}
					if recv, ok := comm.Values[0].(*ast.UnaryOperator); !ok || recv.Op != ast.OperatorReceive {
						panic(tc.errorf(node, "select case must be receive, send or assign recv"))
					}
				case *ast.Send:
					tc.checkNodes([]ast.Node{comm})
				}
				tc.checkNodesInNewScope(cas.Body)
				terminating = terminating && tc.terminating
			}
			tc.removeLastAncestor()
			tc.removeCurrentScope()
			tc.terminating = terminating && !tc.hasBreak[node]

		case *ast.Const:
			tc.checkAssignment(node)
			tc.terminating = false

		case *ast.Var:
			tc.checkAssignment(node)
			tc.nextValidGoto = len(tc.gotos)
			tc.terminating = false

		case *ast.TypeDeclaration:
			// TODO (Gianluca): it currently evaluates every type
			// declaration as alias declaration, cause defining new types
			// is currently not supported.
			if isBlankIdentifier(node.Identifier) {
				continue
			}
			name := node.Identifier.Name
			typ := tc.checkType(node.Type, noEllipses)
			tc.assignScope(name, typ, node.Identifier)

		case *ast.Show:
			ti := tc.checkExpression(node.Expr)
			ti.setValue(nil)
			tc.terminating = false

		case *ast.ShowMacro:
			tc.showMacros = append(tc.showMacros, node)
			var fun ast.Expression
			if node.Import != nil {
				fun = ast.NewSelector(node.Import.Pos(), node.Import, node.Macro.Name)
			} else {
				fun = node.Macro
			}
			nodes[i] = ast.NewCall(node.Pos(), fun, node.Args, node.IsVariadic)
			tc.checkNodes(nodes[i : i+1])

		case *ast.Macro:
			nodes[i] = macroToFunc(node)
			tc.checkNodes(nodes[i : i+1])

		case *ast.Call:
			tis, isBuiltin, _ := tc.checkCallExpression(node, true)
			if isBuiltin {
				switch node.Func.(*ast.Identifier).Name {
				case "copy", "recover":
				case "panic":
					tc.terminating = true
				default:
					if len(tis) > 0 {
						panic(tc.errorf(node, "%s evaluated but not used", node))
					}
				}
			}

		case *ast.Defer:
			_, isBuiltin, isConversion := tc.checkCallExpression(node.Call, true)
			if isBuiltin {
				name := node.Call.Func.(*ast.Identifier).Name
				switch name {
				case "append", "cap", "len", "make", "new":
					panic(tc.errorf(node, "defer discards result of %s", node.Call))
				}
			}
			if isConversion {
				panic(tc.errorf(node, "defer requires function call, not conversion"))
			}
			tc.terminating = false

		case *ast.Go:
			_, isBuiltin, isConversion := tc.checkCallExpression(node.Call, true)
			if isBuiltin {
				name := node.Call.Func.(*ast.Identifier).Name
				switch name {
				case "append", "cap", "len", "make", "new":
					panic(tc.errorf(node, "go discards result of %s", node.Call))
				}
			}
			if isConversion {
				panic(tc.errorf(node, "go requires function call, not conversion"))
			}
			if tc.opts.DisallowGoStmt {
				panic(tc.errorf(node, "\"go\" statement not available"))
			}
			tc.terminating = false

		case *ast.Send:
			tic := tc.checkExpression(node.Channel)
			if tic.Type.Kind() != reflect.Chan {
				panic(tc.errorf(node, "invalid operation: %s (send to non-chan type %s)", node, tic.ShortString()))
			}
			if tic.Type.ChanDir() == reflect.RecvDir {
				panic(tc.errorf(node, "invalid operation: %s (send to receive-only type %s)", node, tic.ShortString()))
			}
			elemType := tic.Type.Elem()
			tiv := tc.checkExpression(node.Value)
			if err := isAssignableTo(tiv, node.Value, elemType); err != nil {
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
			tiv.setValue(elemType)

		case *ast.UnaryOperator:
			ti := tc.checkExpression(node)
			if node.Op != ast.OperatorReceive {
				isLastScriptStatement := len(tc.Scopes) == 2 && i == len(nodes)-1
				if tc.opts.SyntaxType == ProgramSyntax || !isLastScriptStatement {
					panic(tc.errorf(node, "%s evaluated but not used", node))
				}
			}
			ti.setValue(nil)

		case *ast.Goto:
			tc.gotos = append(tc.gotos, node.Label.Name)

		case *ast.Label:
			tc.labels[len(tc.labels)-1] = append(tc.labels[len(tc.labels)-1], node.Name.Name)
			for i, g := range tc.gotos {
				if g == node.Name.Name {
					if i < tc.nextValidGoto {
						panic(tc.errorf(node, "goto %s jumps over declaration of ? at ?", node.Name.Name)) // TODO(Gianluca).
					}
					break
				}
			}
			if node.Statement != nil {
				tc.checkNodes([]ast.Node{node.Statement})
			}

		case *ast.Comment:

		case ast.Expression:
			ti := tc.checkExpression(node)
			if tc.opts.SyntaxType == ScriptSyntax {
				isLastScriptStatement := len(tc.Scopes) == 2 && i == len(nodes)-1
				switch node := node.(type) {
				case *ast.Func:
					if node.Ident == nil {
						if !isLastScriptStatement {
							panic(tc.errorf(node, "%s evaluated but not used", node))
						}
					} else {
						tc.assignScope(node.Ident.Name, ti, node.Ident)
					}
				default:
					if !isLastScriptStatement {
						panic(tc.errorf(node, "%s evaluated but not used", node))
					}
				}
			} else if tc.opts.SyntaxType == TemplateSyntax {
				// TODO(Gianluca): handle expression statements in templates.
				switch node := node.(type) {
				case *ast.Func:
					tc.assignScope(node.Ident.Name, ti, node.Ident)
				default:
					panic(tc.errorf(node, "%s evaluated but not used", node))
				}
			} else {
				panic(tc.errorf(node, "%s evaluated but not used", node))
			}

		default:
			panic(fmt.Errorf("checkNodes not implemented for type: %T", node))

		}

	}

}

// checkReturn type checks a return statement.
func (tc *typechecker) checkReturn(node *ast.Return) {

	fn, funcBound := tc.currentFunction()
	if fn == nil {
		panic(tc.errorf(node, "non-declaration statement outside function body"))
	}

	fillParametersTypes(fn.Type.Result)
	expected := fn.Type.Result
	got := node.Values

	if len(expected) == 0 && len(got) == 0 {
		return
	}

	// Named return arguments with empty return: check if any value has been
	// shadowed.
	if len(expected) > 0 && expected[0].Ident != nil && len(got) == 0 {
		// If "return" belongs to an inner scope (not the function scope).
		if len(tc.Scopes) > funcBound {
			for _, e := range expected {
				name := e.Ident.Name
				_, ok := tc.lookupScopes(name, true)
				if ok {
					panic(tc.errorf(node, "%s is shadowed during return", name))
				}
			}
		}
		return
	}

	var expectedTypes []reflect.Type
	for _, exp := range expected {
		ti := tc.checkType(exp.Type, noEllipses)
		expectedTypes = append(expectedTypes, ti.Type)
	}

	needsCheck := true
	if len(expected) > 1 && len(got) == 1 {
		if c, ok := got[0].(*ast.Call); ok {
			tis, _, _ := tc.checkCallExpression(c, false)
			got = nil
			for _, ti := range tis {
				v := ast.NewCall(c.Pos(), c.Func, c.Args, false)
				tc.TypeInfo[v] = ti
				got = append(got, v)
				needsCheck = false
			}
		}
	}

	if needsCheck {
		for _, g := range got {
			_ = tc.checkExpression(g)
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
			msg += tc.TypeInfo[x].StringWithNumber(false)
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
		ti := tc.TypeInfo[x]
		if err := isAssignableTo(ti, x, typ); err != nil {
			if _, ok := err.(invalidTypeInAssignment); ok {
				panic(tc.errorf(node, "%s in return argument", err))
			}
			panic(tc.errorf(node, "%s", err))
		}
		ti.setValue(typ)
	}

	return
}
