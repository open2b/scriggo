// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"

	"github.com/open2b/scriggo/ast"
)

// Makes a dependency analysis after parsing and before the type checking. See
// https://golang.org/ref/spec#Package_initialization for further information.

// packageDeclsDeps is the result of a dependency analysis performed on a tree.
type packageDeclsDeps map[*ast.Identifier][]*ast.Identifier

type deps struct {
	d packageDeclsDeps
	// itea maps the name of the transformed 'itea' identifier to the
	// identifiers on the left side of a 'var' declarations with an 'using'
	// statement.
	itea                     map[string][]*ast.Identifier
	analyzingVarExprWithItea *ast.Identifier
}

// addDepsToGlobal adds all identifiers that appear in node and in its children
// as dependency of the global identifier ident.
func (d *deps) addDepsToGlobal(ident *ast.Identifier, node ast.Node, scopes depScopes) {
	if scopes == nil {
		scopes = depScopes{map[string]struct{}{}}
	}
	if d.d[ident] == nil {
		d.d[ident] = []*ast.Identifier{}
	}
	for _, dep := range d.nodeDeps(node, scopes) {
		if dep.Name == "_" {
			continue
		}
		alreadyAdded := false
		for _, d := range d.d[ident] {
			if d.Name == dep.Name {
				alreadyAdded = true
				break
			}
		}
		if !alreadyAdded {
			d.d[ident] = append(d.d[ident], dep)
		}
	}
}

// hasIteaInItsExpression reports whether the package-level identifier varLh
// references to the predeclared identifier 'itea' in its corresponding
// expression.
func (d *deps) hasIteaInItsExpression(varLh *ast.Identifier) bool {
	for _, v := range d.itea {
		for _, ident := range v {
			if ident == varLh {
				return true
			}
		}
	}
	return false
}

// analyzeGlobalVar analyzes a global var declaration.
func (d *deps) analyzeGlobalVar(n *ast.Var) {
	if len(n.Lhs) == len(n.Rhs) {
		for i := range n.Lhs {
			d.addDepsToGlobal(n.Lhs[i], n.Type, nil)
			if d.hasIteaInItsExpression(n.Lhs[i]) {
				d.analyzingVarExprWithItea = n.Lhs[i]
				d.addDepsToGlobal(n.Lhs[i], n.Rhs[i], nil)
				d.analyzingVarExprWithItea = nil
			} else {
				d.addDepsToGlobal(n.Lhs[i], n.Rhs[i], nil)
			}
		}
		return
	}
	for _, left := range n.Lhs {
		d.addDepsToGlobal(left, n.Type, nil)
		for _, right := range n.Rhs {
			d.addDepsToGlobal(left, right, nil)
		}
	}
}

// analyzeGlobalConst analyzes a global constant declaration.
func (d *deps) analyzeGlobalConst(n *ast.Const) {
	for i := range n.Lhs {
		d.addDepsToGlobal(n.Lhs[i], n.Type, nil)
		d.addDepsToGlobal(n.Lhs[i], n.Rhs[i], nil)
	}
}

// analyzeGlobalDeclarationAssignment analyzes a global global declaration assignment.
func (d *deps) analyzeGlobalDeclarationAssignment(n *ast.Assignment) {
	if len(n.Lhs) == len(n.Rhs) {
		for i := range n.Lhs {
			if ident, ok := n.Lhs[i].(*ast.Identifier); ok {
				d.addDepsToGlobal(ident, n.Rhs[i], nil)
			}
		}
		return
	}
	for _, left := range n.Lhs {
		for _, right := range n.Rhs {
			d.addDepsToGlobal(left.(*ast.Identifier), right, nil)
		}
	}
}

// analyzeGlobalFunc analyzes a global function declaration.
func (d *deps) analyzeGlobalFunc(n *ast.Func) {
	scopes := depScopes{map[string]struct{}{}}
	for _, f := range n.Type.Parameters {
		if f.Ident != nil {
			scopes = declareLocally(scopes, f.Ident.Name)
		}
	}
	for _, f := range n.Type.Result {
		if f.Ident != nil {
			scopes = declareLocally(scopes, f.Ident.Name)
		}
	}
	d.addDepsToGlobal(n.Ident, n.Type, scopes)
	if n.Body != nil {
		d.addDepsToGlobal(n.Ident, n.Body, scopes)
	}
}

// analyzeGlobalTypeDeclaration analyzes a global type declaration.
func (d *deps) analyzeGlobalTypeDeclaration(td *ast.TypeDeclaration) {
	d.addDepsToGlobal(td.Ident, td.Type, nil)
}

// analyzeTree analyzes tree returning a data structure holding all dependencies
// information.
func analyzeTree(pkg *ast.Package) packageDeclsDeps {
	d := &deps{
		d:    packageDeclsDeps{},
		itea: pkg.IR.IteaNameToVarIdents,
	}
	for _, n := range pkg.Declarations {
		switch n := n.(type) {
		case *ast.Var:
			d.analyzeGlobalVar(n)
		case *ast.Const:
			d.analyzeGlobalConst(n)
		case *ast.Func:
			d.analyzeGlobalFunc(n)
		case *ast.TypeDeclaration:
			d.analyzeGlobalTypeDeclaration(n)
		}
	}
	return packageDeclsDeps(d.d)
}

// depScopes represents a set of scopes used in dependency analysis.
type depScopes []map[string]struct{}

// enterScope enters in a new scope.
func enterScope(scopes depScopes) depScopes {
	return append(scopes, map[string]struct{}{})
}

// exitScope exit from current scope.
func exitScope(scopes depScopes) depScopes {
	return scopes[:len(scopes)-1]
}

// declareLocally declare name as a local name in scopes.
func declareLocally(scopes depScopes, name string) depScopes {
	scopes[len(scopes)-1][name] = struct{}{}
	return scopes
}

// isLocallyDefined reports whether name is locally defined in scopes.
func isLocallyDefined(scopes depScopes, name string) bool {
	for i := len(scopes) - 1; i >= 0; i-- {
		if _, ok := scopes[i][name]; ok {
			return true
		}
	}
	return false
}

// nodeDeps returns all dependencies of node n. scopes contains all active
// scopes.
func (d *deps) nodeDeps(n ast.Node, scopes depScopes) []*ast.Identifier {
	if n == nil {
		return nil
	}
	switch n := n.(type) {
	case *ast.ArrayType:
		return d.nodeDeps(n.ElementType, scopes)
	case *ast.Assignment:
		deps := []*ast.Identifier{}
		for _, right := range n.Rhs {
			deps = append(deps, d.nodeDeps(right, scopes)...)
		}
		if n.Type == ast.AssignmentDeclaration {
			for _, left := range n.Lhs {
				if ident, ok := left.(*ast.Identifier); ok {
					scopes = declareLocally(scopes, ident.Name)
				}
			}
		} else {
			for _, left := range n.Lhs {
				deps = append(deps, d.nodeDeps(left, scopes)...)
			}
		}
		return deps
	case *ast.BasicLiteral:
		return nil
	case *ast.BinaryOperator:
		return append(d.nodeDeps(n.Expr1, scopes), d.nodeDeps(n.Expr2, scopes)...)
	case *ast.Block:
		scopes = enterScope(scopes)
		deps := []*ast.Identifier{}
		for _, n := range n.Nodes {
			deps = append(deps, d.nodeDeps(n, scopes)...)
		}
		scopes = exitScope(scopes)
		return deps
	case *ast.Break:
		return nil
	case *ast.Call:
		deps := d.nodeDeps(n.Func, scopes)
		for _, arg := range n.Args {
			deps = append(deps, d.nodeDeps(arg, scopes)...)
		}
		return deps
	case *ast.Case:
		deps := []*ast.Identifier{}
		for _, expr := range n.Expressions {
			deps = append(deps, d.nodeDeps(expr, scopes)...)
		}
		for _, node := range n.Body {
			deps = append(deps, d.nodeDeps(node, scopes)...)
		}
		return deps
	case *ast.ChanType:
		return d.nodeDeps(n.ElementType, scopes)
	case *ast.CompositeLiteral:
		deps := d.nodeDeps(n.Type, scopes)
		for _, kv := range n.KeyValues {
			deps = append(deps, d.nodeDeps(kv.Key, scopes)...)
			deps = append(deps, d.nodeDeps(kv.Value, scopes)...)
		}
		return deps
	case *ast.Comment:
		return nil
	case *ast.Const:
		deps := []*ast.Identifier{}
		for _, right := range n.Lhs {
			deps = append(deps, d.nodeDeps(right, scopes)...)
		}
		for _, left := range n.Lhs {
			scopes = declareLocally(scopes, left.Name)
		}
		return append(deps, d.nodeDeps(n.Type, scopes)...)
	case *ast.Continue:
		return nil
	case *ast.Defer:
		return d.nodeDeps(n.Call, scopes)
	case *ast.Default:
		return d.nodeDeps(n.Expr2, scopes)
	case *ast.Extends:
		return nil
	case *ast.Fallthrough:
		return nil
	case *ast.For:
		scopes = enterScope(scopes)
		deps := d.nodeDeps(n.Init, scopes)
		deps = append(deps, d.nodeDeps(n.Condition, scopes)...)
		deps = append(deps, d.nodeDeps(n.Post, scopes)...)
		scopes = enterScope(scopes)
		for _, node := range n.Body {
			deps = append(deps, d.nodeDeps(node, scopes)...)
		}
		scopes = exitScope(scopes)
		scopes = exitScope(scopes)
		return deps
	case *ast.ForIn:
		deps := []*ast.Identifier{}
		scopes = enterScope(scopes)
		scopes = declareLocally(scopes, n.Ident.Name)
		deps = append(deps, d.nodeDeps(n.Expr, scopes)...)
		scopes = enterScope(scopes)
		for _, node := range n.Body {
			deps = append(deps, d.nodeDeps(node, scopes)...)
		}
		scopes = exitScope(scopes)
		scopes = exitScope(scopes)
		return deps
	case *ast.ForRange:
		scopes = enterScope(scopes)
		deps := d.nodeDeps(n.Assignment, scopes)
		scopes = enterScope(scopes)
		for _, node := range n.Body {
			deps = append(deps, d.nodeDeps(node, scopes)...)
		}
		scopes = exitScope(scopes)
		scopes = exitScope(scopes)
		return deps
	case *ast.Func:
		for _, f := range n.Type.Parameters {
			if f.Ident != nil {
				scopes = declareLocally(scopes, f.Ident.Name)
			}
		}
		for _, f := range n.Type.Result {
			if f.Ident != nil {
				scopes = declareLocally(scopes, f.Ident.Name)
			}
		}
		deps := d.nodeDeps(n.Type, scopes)
		if n.Body != nil {
			deps = append(deps, d.nodeDeps(n.Body, scopes)...)
		}
		return deps
	case *ast.FuncType:
		deps := []*ast.Identifier{}
		for _, in := range n.Parameters {
			deps = append(deps, d.nodeDeps(in.Type, scopes)...)
		}
		for _, out := range n.Result {
			deps = append(deps, d.nodeDeps(out.Type, scopes)...)
		}
		return deps
	case *ast.DollarIdentifier:
		// A valid dollar identifier can at most depend upon global
		// identifiers, so it is not considered in the dependency analysis.
		return nil
	case *ast.Go:
		return d.nodeDeps(n.Call, scopes)
	case *ast.Goto:
		return nil
	case *ast.Identifier:
		if isLocallyDefined(scopes, n.Name) {
			return nil
		}
		if d.analyzingVarExprWithItea != nil && n.Name == "itea" {
			var thisName string
			for k, v := range d.itea {
				for _, ident := range v {
					if ident == d.analyzingVarExprWithItea {
						thisName = k
						break
					}
				}
			}
			if thisName == "" {
				panic("BUG: unexpected")
			}
			n.Name = thisName
		}
		return []*ast.Identifier{n}
	case *ast.If:
		scopes = enterScope(scopes)
		deps := d.nodeDeps(n.Init, scopes)
		deps = append(deps, d.nodeDeps(n.Condition, scopes)...)
		scopes = enterScope(scopes)
		deps = append(deps, d.nodeDeps(n.Then, scopes)...)
		scopes = enterScope(scopes)
		deps = append(deps, d.nodeDeps(n.Else, scopes)...)
		scopes = exitScope(scopes)
		scopes = exitScope(scopes)
		scopes = exitScope(scopes)
		return deps
	case *ast.Import:
		return nil
	case *ast.Render:
		return nil
	case *ast.Index:
		deps := d.nodeDeps(n.Expr, scopes)
		return append(deps, d.nodeDeps(n.Index, scopes)...)
	case *ast.Interface:
		return nil
	case *ast.Label:
		return nil
	case *ast.MapType:
		deps := d.nodeDeps(n.KeyType, scopes)
		return append(deps, d.nodeDeps(n.ValueType, scopes)...)
	case *ast.Raw:
		return nil
	case *ast.Return:
		deps := []*ast.Identifier{}
		for _, v := range n.Values {
			deps = append(deps, d.nodeDeps(v, scopes)...)
		}
		return deps
	case *ast.Select:
		deps := []*ast.Identifier{}
		for _, cas := range n.Cases {
			deps = append(deps, d.nodeDeps(cas, scopes)...)
		}
		return deps
	case *ast.SelectCase:
		deps := d.nodeDeps(n.Comm, scopes)
		for _, n := range n.Body {
			deps = append(deps, d.nodeDeps(n, scopes)...)
		}
		return deps
	case *ast.Selector:
		return d.nodeDeps(n.Expr, scopes)
	case *ast.Send:
		deps := d.nodeDeps(n.Channel, scopes)
		return append(deps, d.nodeDeps(n.Value, scopes)...)
	case *ast.Show:
		deps := []*ast.Identifier{}
		for _, node := range n.Expressions {
			deps = append(deps, d.nodeDeps(node, scopes)...)
		}
		return deps
	case *ast.SliceType:
		return d.nodeDeps(n.ElementType, scopes)
	case *ast.Slicing:
		deps := d.nodeDeps(n.Expr, scopes)
		deps = append(deps, d.nodeDeps(n.Low, scopes)...)
		deps = append(deps, d.nodeDeps(n.High, scopes)...)
		return append(deps, d.nodeDeps(n.Max, scopes)...)
	case *ast.Statements:
		deps := []*ast.Identifier{}
		for _, node := range n.Nodes {
			deps = append(deps, d.nodeDeps(node, scopes)...)
		}
		return deps
	case *ast.StructType:
		deps := []*ast.Identifier{}
		for _, fd := range n.Fields {
			deps = append(deps, d.nodeDeps(fd.Type, scopes)...)
		}
		return deps
	case *ast.Switch:
		scopes = enterScope(scopes)
		deps := d.nodeDeps(n.Init, scopes)
		deps = append(deps, d.nodeDeps(n.Expr, scopes)...)
		for _, cas := range n.Cases {
			deps = append(deps, d.nodeDeps(cas, scopes)...)
		}
		scopes = exitScope(scopes)
		return deps
	case *ast.Text:
		return nil
	case *ast.TypeAssertion:
		deps := d.nodeDeps(n.Expr, scopes)
		deps = append(deps, d.nodeDeps(n.Type, scopes)...)
		return deps
	case *ast.TypeDeclaration:
		return d.nodeDeps(n.Type, scopes)
	case *ast.TypeSwitch:
		scopes = enterScope(scopes)
		deps := d.nodeDeps(n.Init, scopes)
		deps = append(deps, d.nodeDeps(n.Assignment, scopes)...)
		for _, cas := range n.Cases {
			deps = append(deps, d.nodeDeps(cas, scopes)...)
		}
		scopes = exitScope(scopes)
		return deps
	case *ast.UnaryOperator:
		return d.nodeDeps(n.Expr, scopes)
	case *ast.URL:
		deps := []*ast.Identifier{}
		for _, node := range n.Value {
			deps = append(deps, d.nodeDeps(node, scopes)...)
		}
		return deps
	case *ast.Using:
		deps := []*ast.Identifier{}
		deps = append(deps, d.nodeDeps(n.Statement, scopes)...)
		for _, node := range n.Body.Nodes {
			deps = append(deps, d.nodeDeps(node, scopes)...)
		}
		return deps
	case *ast.Var:
		deps := []*ast.Identifier{}
		deps = append(deps, d.nodeDeps(n.Type, scopes)...)
		for _, right := range n.Rhs {
			deps = append(deps, d.nodeDeps(right, scopes)...)
		}
		for _, left := range n.Lhs {
			scopes = declareLocally(scopes, left.Name)
		}
		return deps
	default:
		panic(fmt.Errorf("missing case for type %T", n))
	}
}
