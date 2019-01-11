// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package astutil

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

	case *ast.URL:
		for _, child := range n.Value {
			Walk(v, child)
		}

	case *ast.Assignment:
		for _, child := range n.Values {
			Walk(v, child)
		}

	case *ast.For:
		if n.Init != nil {
			Walk(v, n.Init)
		}

		if n.Condition != nil {
			Walk(v, n.Condition)
		}

		if n.Post != nil {
			Walk(v, n.Post)
		}

		for _, n := range n.Body {
			Walk(v, n)
		}

	case *ast.ForRange:
		if n.Assignment != nil {
			Walk(v, n.Assignment)
		}

		for _, n := range n.Body {
			Walk(v, n)
		}

	case *ast.If:
		Walk(v, n.Condition)
		for _, child := range n.Then {
			Walk(v, child)
		}

		for _, child := range n.Else {
			Walk(v, child)
		}

	case *ast.Macro:
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

	case *ast.Map:
		for _, element := range n.Elements {
			Walk(v, element.Key)
			Walk(v, element.Value)
		}

	case *ast.Slice:
		for _, element := range n.Elements {
			Walk(v, element)
		}

	case *ast.Bytes:
		for _, element := range n.Elements {
			Walk(v, element)
		}

	case *ast.Call:
		for _, arg := range n.Args {
			Walk(v, arg)
		}

	case *ast.Index:
		Walk(v, n.Expr)
		Walk(v, n.Index)

	case *ast.Slicing:
		Walk(v, n.Expr)
		if n.Low != nil {
			Walk(v, n.Low)
		}
		if n.High != nil {
			Walk(v, n.High)
		}

	case *ast.Selector:
		Walk(v, n.Expr)

	case *ast.TypeAssertion:
		Walk(v, n.Expr)

	case *ast.Extends:
	case *ast.Import:
	case *ast.Include:
		// Nothing to do, visiting the expanded tree is done
		// by the Visit function if necessary.

	case *ast.Int:
	case *ast.Number:
	case *ast.Identifier:
	case *ast.String:
	case *ast.ShowMacro:
	case *ast.Comment:
	case *ast.Break:
	case *ast.Continue:
	case *ast.Text:
		// Nothing to do

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
