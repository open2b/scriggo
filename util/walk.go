//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package util

import (
	"fmt"

	"open2b/template/ast"
)

// Visitor's visit method is invoked for every node encountered by Walk.
type Visitor interface {
	Visit(node ast.Node) (w Visitor)
}

// Walk visits a tree in depth. Initially it calls v.Visit (node),
// where node must not be nil. If the value w returned by v.Visit (node)
// is different from nil, Walk is called recursively using w as the Visitor
// on all children other than nil of the tree. Finally, call w.Visit (nil).
func Walk(v Visitor, node ast.Node) {

	if v == nil {
		panic("v can't be nil")
	}

	if node == nil {
		panic("node can't be nil")
	}

	v = v.Visit(node)

	if v == nil {
		return
	}

	switch n := node.(type) {

	// If the child is a concrete type, it leaves it under management at
	// the Visit, otherwise he manages it here on the Walk.

	case *ast.Tree:
		for _, child := range n.Nodes {
			Walk(v, child)
		}

	case *ast.Var:
		Walk(v, n.Expr)

	case *ast.Assignment:
		Walk(v, n.Expr)

	case *ast.For:
		Walk(v, n.Expr1)

		if n.Expr2 != nil {
			Walk(v, n.Expr2)
		}

		for _, n := range n.Nodes {
			Walk(v, n)
		}

	case *ast.If:
		Walk(v, n.Expr)
		for _, child := range n.Then {
			Walk(v, child)
		}

		for _, child := range n.Else {
			Walk(v, child)
		}

	case *ast.Region:
		for _, child := range n.Body {
			Walk(v, child)
		}

	case *ast.Value:
		Walk(v, n.Expr)

	case *ast.Parentesis:
		Walk(v, n.Expr)

	case *ast.UnaryOperator:
		Walk(v, n.Expr)

	case *ast.BinaryOperator:
		Walk(v, n.Expr1)
		Walk(v, n.Expr2)

	case *ast.Call:
		for _, arg := range n.Args {
			Walk(v, arg)
		}

	case *ast.Index:
		Walk(v, n.Expr)
		Walk(v, n.Index)

	case *ast.Slice:
		Walk(v, n.Expr)
		if n.Low != nil {
			Walk(v, n.Low)
		}
		if n.High != nil {
			Walk(v, n.High)
		}

	case *ast.Selector:
		Walk(v, n.Expr)

	case *ast.Extend:
	case *ast.Import:
	case *ast.ShowPath:
		// Nothing to do, visiting the expanded tree is done
		// by the Visit function if necessary.

	case *ast.Int:
	case *ast.Number:
	case *ast.Identifier:
	case *ast.String:
	case *ast.ShowRegion:
	case *ast.Comment:
	case *ast.Break:
	case *ast.Continue:
	case *ast.Text:
		// Niente da fare

	default:
		panic(fmt.Sprintf("No cases were defined for type %T on function Walk", n))
	}

	v.Visit(nil)

}

// Visit implements the Visitor interface for the f function.
func (f inspector) Visit(node ast.Node) Visitor {
	if f(node) {
		return f
	}
	return nil
}

type inspector func(ast.Node) bool

// Inspect visits the tree by calling the function f on every node.
// For more information, see the documentation of the Walk function.
func Inspect(node ast.Node, f func(ast.Node) bool) {
	Walk(inspector(f), node)
}
