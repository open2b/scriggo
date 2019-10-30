// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"

	"scriggo/ast"
)

// Makes a dependency analysis after parsing and before the type checking. See
// https://golang.org/ref/spec#Package_initialization for further information.

// PackageDeclsDeps is the result of a dependency analysis performed on a tree.
type PackageDeclsDeps map[*ast.Identifier][]*ast.Identifier

type deps PackageDeclsDeps

// addDepsToGlobal adds all identifiers that appear in node and in its children
// as dependency of the global identifier ident.
func (d deps) addDepsToGlobal(ident *ast.Identifier, node ast.Node, scopes depScopes) {
	if scopes == nil {
		scopes = depScopes{map[string]struct{}{}}
	}
	if d[ident] == nil {
		d[ident] = []*ast.Identifier{}
	}
	for _, dep := range nodeDeps(node, scopes) {
		if dep.Name == "_" {
			continue
		}
		alreadyAdded := false
		for _, d := range d[ident] {
			if d.Name == dep.Name {
				alreadyAdded = true
				break
			}
		}
		if !alreadyAdded {
			d[ident] = append(d[ident], dep)
		}
	}
}

// analyzeGlobalVar analyzes a global var declaration.
func (d deps) analyzeGlobalVar(n *ast.Var) {
	if len(n.Lhs) == len(n.Rhs) {
		for i := range n.Lhs {
			d.addDepsToGlobal(n.Lhs[i], n.Type, nil)
			d.addDepsToGlobal(n.Lhs[i], n.Rhs[i], nil)
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
func (d deps) analyzeGlobalConst(n *ast.Const) {
	for i := range n.Lhs {
		d.addDepsToGlobal(n.Lhs[i], n.Type, nil)
		d.addDepsToGlobal(n.Lhs[i], n.Rhs[i], nil)
	}
}

// analyzeGlobalDeclarationAssignment analyzes a global global declaration assignment.
func (d deps) analyzeGlobalDeclarationAssignment(n *ast.Assignment) {
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
func (d deps) analyzeGlobalFunc(n *ast.Func) {
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
	d.addDepsToGlobal(n.Ident, n.Body, scopes)
}

// analyzeGlobalMacro analyzes a global macro declaration.
func (d deps) analyzeGlobalMacro(n *ast.Macro) {
	scopes := depScopes{map[string]struct{}{}}
	for _, f := range n.Type.Parameters {
		if f.Ident != nil {
			scopes = declareLocally(scopes, f.Ident.Name)
		}
	}
	d.addDepsToGlobal(n.Ident, n.Type, scopes)
	d.addDepsToGlobal(n.Ident, n.Type, nil)
	for _, node := range n.Body {
		d.addDepsToGlobal(n.Ident, node, nil)
	}
}

// analyzeGlobalTypeDeclaration analyzes a global type declaration.
func (d deps) analyzeGlobalTypeDeclaration(td *ast.TypeDeclaration) {
	d.addDepsToGlobal(td.Identifier, td.Type, nil)
}

// AnalyzeTree analyzes tree returning a data structure holding all dependencies
// information.
func AnalyzeTree(pkg *ast.Package) PackageDeclsDeps {
	d := deps{}
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
	return PackageDeclsDeps(d)
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
func nodeDeps(n ast.Node, scopes depScopes) []*ast.Identifier {
	if n == nil {
		return nil
	}
	switch n := n.(type) {
	case *ast.ArrayType:
		return nodeDeps(n.ElementType, scopes)
	case *ast.Assignment:
		if n == nil { // TODO: why is this necessary?
			return nil
		}
		deps := []*ast.Identifier{}
		for _, right := range n.Rhs {
			deps = append(deps, nodeDeps(right, scopes)...)
		}
		if n.Type == ast.AssignmentDeclaration {
			for _, left := range n.Lhs {
				if ident, ok := left.(*ast.Identifier); ok {
					scopes = declareLocally(scopes, ident.Name)
				}
			}
		} else {
			for _, left := range n.Lhs {
				deps = append(deps, nodeDeps(left, scopes)...)
			}
		}
		return deps
	case *ast.BasicLiteral:
		return nil
	case *ast.BinaryOperator:
		return append(nodeDeps(n.Expr1, scopes), nodeDeps(n.Expr2, scopes)...)
	case *ast.Block:
		scopes = enterScope(scopes)
		deps := []*ast.Identifier{}
		for _, n := range n.Nodes {
			deps = append(deps, nodeDeps(n, scopes)...)
		}
		scopes = exitScope(scopes)
		return deps
	case *ast.Break:
		return nil
	case *ast.Call:
		deps := nodeDeps(n.Func, scopes)
		for _, arg := range n.Args {
			deps = append(deps, nodeDeps(arg, scopes)...)
		}
		return deps
	case *ast.Case:
		deps := []*ast.Identifier{}
		for _, expr := range n.Expressions {
			deps = append(deps, nodeDeps(expr, scopes)...)
		}
		for _, node := range n.Body {
			deps = append(deps, nodeDeps(node, scopes)...)
		}
		return deps
	case *ast.ChanType:
		return nodeDeps(n.ElementType, scopes)
	case *ast.CompositeLiteral:
		deps := nodeDeps(n.Type, scopes)
		for _, kv := range n.KeyValues {
			deps = append(deps, nodeDeps(kv.Key, scopes)...)
			deps = append(deps, nodeDeps(kv.Value, scopes)...)
		}
		return deps
	case *ast.Comment:
		return nil
	case *ast.Const:
		deps := []*ast.Identifier{}
		for _, right := range n.Lhs {
			deps = append(deps, nodeDeps(right, scopes)...)
		}
		for _, left := range n.Lhs {
			scopes = declareLocally(scopes, left.Name)
		}
		return append(deps, nodeDeps(n.Type, scopes)...)
	case *ast.Continue:
		return nil
	case *ast.Defer:
		return nodeDeps(n.Call, scopes)
	case *ast.Extends:
		return nil
	case *ast.Fallthrough:
		return nil
	case *ast.For:
		scopes = enterScope(scopes)
		deps := nodeDeps(n.Init, scopes)
		deps = append(deps, nodeDeps(n.Condition, scopes)...)
		deps = append(deps, nodeDeps(n.Post, scopes)...)
		scopes = enterScope(scopes)
		for _, node := range n.Body {
			deps = append(deps, nodeDeps(node, scopes)...)
		}
		scopes = exitScope(scopes)
		scopes = exitScope(scopes)
		return deps
	case *ast.ForRange:
		scopes = enterScope(scopes)
		deps := nodeDeps(n.Assignment, scopes)
		scopes = enterScope(scopes)
		for _, node := range n.Body {
			deps = append(deps, nodeDeps(node, scopes)...)
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
		deps := nodeDeps(n.Type, scopes)
		return append(deps, nodeDeps(n.Body, scopes)...)
	case *ast.FuncType:
		deps := []*ast.Identifier{}
		for _, in := range n.Parameters {
			deps = append(deps, nodeDeps(in.Type, scopes)...)
		}
		for _, out := range n.Result {
			deps = append(deps, nodeDeps(out.Type, scopes)...)
		}
		return deps
	case *ast.Go:
		return nodeDeps(n.Call, scopes)
	case *ast.Goto:
		return nil
	case *ast.Identifier:
		if isLocallyDefined(scopes, n.Name) {
			return nil
		}
		return []*ast.Identifier{n}
	case *ast.If:
		scopes = enterScope(scopes)
		deps := nodeDeps(n.Assignment, scopes)
		deps = append(deps, nodeDeps(n.Condition, scopes)...)
		scopes = enterScope(scopes)
		deps = append(deps, nodeDeps(n.Then, scopes)...)
		scopes = enterScope(scopes)
		deps = append(deps, nodeDeps(n.Else, scopes)...)
		scopes = exitScope(scopes)
		scopes = exitScope(scopes)
		scopes = exitScope(scopes)
		return deps
	case *ast.Import:
		return nil
	case *ast.Include:
		return nil
	case *ast.Index:
		deps := nodeDeps(n.Expr, scopes)
		return append(deps, nodeDeps(n.Index, scopes)...)
	case *ast.Interface:
		return nil
	case *ast.Label:
		return nil
	case *ast.MapType:
		deps := nodeDeps(n.KeyType, scopes)
		return append(deps, nodeDeps(n.ValueType, scopes)...)
	case *ast.Return:
		deps := []*ast.Identifier{}
		for _, v := range n.Values {
			deps = append(deps, nodeDeps(v, scopes)...)
		}
		return deps
	case *ast.Select:
		deps := []*ast.Identifier{}
		for _, cas := range n.Cases {
			deps = append(deps, nodeDeps(cas, scopes)...)
		}
		return deps
	case *ast.SelectCase:
		deps := nodeDeps(n.Comm, scopes)
		for _, n := range n.Body {
			deps = append(deps, nodeDeps(n, scopes)...)
		}
		return deps
	case *ast.Selector:
		return nodeDeps(n.Expr, scopes)
	case *ast.Send:
		deps := nodeDeps(n.Channel, scopes)
		return append(deps, nodeDeps(n.Value, scopes)...)
	case *ast.Show:
		return nodeDeps(n.Expr, scopes)
	case *ast.ShowMacro:
		deps := nodeDeps(n.Macro, scopes)
		for _, arg := range n.Args {
			deps = append(deps, nodeDeps(arg, scopes)...)
		}
		return deps
	case *ast.SliceType:
		return nodeDeps(n.ElementType, scopes)
	case *ast.Slicing:
		deps := nodeDeps(n.Expr, scopes)
		deps = append(deps, nodeDeps(n.Low, scopes)...)
		deps = append(deps, nodeDeps(n.High, scopes)...)
		return append(deps, nodeDeps(n.Max, scopes)...)
	case *ast.StructType:
		deps := []*ast.Identifier{}
		for _, fd := range n.Fields {
			deps = append(deps, nodeDeps(fd.Type, scopes)...)
		}
		return deps
	case *ast.Switch:
		scopes = enterScope(scopes)
		deps := nodeDeps(n.Init, scopes)
		deps = append(deps, nodeDeps(n.Expr, scopes)...)
		for _, cas := range n.Cases {
			deps = append(deps, nodeDeps(cas, scopes)...)
		}
		scopes = exitScope(scopes)
		return deps
	case *ast.Text:
		return nil
	case *ast.TypeAssertion:
		deps := nodeDeps(n.Expr, scopes)
		deps = append(deps, nodeDeps(n.Type, scopes)...)
		return deps
	case *ast.TypeDeclaration:
		return nodeDeps(n.Type, scopes)
	case *ast.TypeSwitch:
		scopes = enterScope(scopes)
		deps := nodeDeps(n.Init, scopes)
		deps = append(deps, nodeDeps(n.Assignment, scopes)...)
		for _, cas := range n.Cases {
			deps = append(deps, nodeDeps(cas, scopes)...)
		}
		scopes = exitScope(scopes)
		return deps
	case *ast.UnaryOperator:
		return nodeDeps(n.Expr, scopes)
	case *ast.URL:
		deps := []*ast.Identifier{}
		for _, node := range n.Value {
			deps = append(deps, nodeDeps(node, scopes)...)
		}
		return deps
	case *ast.Var:
		deps := []*ast.Identifier{}
		deps = append(deps, nodeDeps(n.Type, scopes)...)
		for _, right := range n.Rhs {
			deps = append(deps, nodeDeps(right, scopes)...)
		}
		for _, left := range n.Lhs {
			scopes = declareLocally(scopes, left.Name)
		}
		return deps
	default:
		panic(fmt.Errorf("missing case for type %T", n))
	}
}
