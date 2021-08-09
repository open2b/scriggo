// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package astutil

import (
	"fmt"

	"github.com/open2b/scriggo/ast"
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
		return
	}

	v = v.Visit(node)

	if v == nil {
		return
	}

	switch n := node.(type) {

	// If the child is a concrete type, it leaves it under management at
	// the Visit, otherwise he manages it here on the Walk.

	case *ast.ArrayType:
		Walk(v, n.Len)
		Walk(v, n.ElementType)

	case *ast.Assignment:
		for _, child := range n.Lhs {
			Walk(v, child)
		}
		for _, child := range n.Rhs {
			Walk(v, child)
		}

	case *ast.BinaryOperator:
		Walk(v, n.Expr1)
		Walk(v, n.Expr2)

	case *ast.Block:
		for _, child := range n.Nodes {
			Walk(v, child)
		}

	case *ast.Break:
		Walk(v, n.Label)

	case *ast.Call:
		for _, arg := range n.Args {
			Walk(v, arg)
		}

	case *ast.Case:
		for _, e := range n.Expressions {
			Walk(v, e)
		}
		for _, child := range n.Body {
			Walk(v, child)
		}

	case *ast.ChanType:
		Walk(v, n.ElementType)

	case *ast.CompositeLiteral:
		Walk(v, n.Type)
		for _, vv := range n.KeyValues {
			if vv.Key != nil {
				Walk(v, vv.Key)
			}
			Walk(v, vv.Value)
		}

	case *ast.Const:
		for _, ident := range n.Lhs {
			Walk(v, ident)
		}
		Walk(v, n.Type)
		for _, value := range n.Rhs {
			Walk(v, value)
		}

	case *ast.Continue:
		Walk(v, n.Label)

	case *ast.Defer:
		Walk(v, n.Call)

	case *ast.Default:
		Walk(v, n.Expr1)
		Walk(v, n.Expr2)

	case *ast.DollarIdentifier:
		Walk(v, n.Ident)

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

	case *ast.ForIn:
		Walk(v, n.Ident)
		Walk(v, n.Expr)
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

	case *ast.Func:
		for _, child := range n.Body.Nodes {
			Walk(v, child)
		}

	case *ast.FuncType:
		for _, param := range n.Parameters {
			Walk(v, param.Type)
		}
		for _, res := range n.Result {
			Walk(v, res.Type)
		}

	case *ast.Go:
		Walk(v, n.Call)

	case *ast.Goto:
		Walk(v, n.Label)

	case *ast.If:
		if n.Init != nil {
			Walk(v, n.Init)
		}
		Walk(v, n.Condition)
		if n.Then != nil {
			Walk(v, n.Then)
		}
		if n.Else != nil {
			Walk(v, n.Else)
		}

	case *ast.Index:
		Walk(v, n.Expr)
		Walk(v, n.Index)

	case *ast.Label:
		Walk(v, n.Ident)
		Walk(v, n.Statement)

	case *ast.MapType:
		Walk(v, n.KeyType)
		Walk(v, n.ValueType)

	case *ast.Package:
		for _, declaration := range n.Declarations {
			Walk(v, declaration)
		}

	case *ast.Return:
		for _, value := range n.Values {
			Walk(v, value)
		}

	case *ast.Select:
		for _, c := range n.Cases {
			Walk(v, c)
		}

	case *ast.SelectCase:
		Walk(v, n.Comm)
		for _, child := range n.Body {
			Walk(v, child)
		}

	case *ast.Selector:
		Walk(v, n.Expr)

	case *ast.Send:
		Walk(v, n.Channel)
		Walk(v, n.Value)

	case *ast.Show:
		for _, expr := range n.Expressions {
			Walk(v, expr)
		}

	case *ast.SliceType:
		Walk(v, n.ElementType)

	case *ast.Slicing:
		Walk(v, n.Expr)
		if n.Low != nil {
			Walk(v, n.Low)
		}
		if n.High != nil {
			Walk(v, n.High)
		}
		if n.Max != nil {
			Walk(v, n.Max)
		}

	case *ast.Statements:
		for _, child := range n.Nodes {
			Walk(v, child)
		}

	case *ast.Switch:
		Walk(v, n.Init)
		Walk(v, n.Expr)
		for _, c := range n.Cases {
			Walk(v, c)
		}

	case *ast.Tree:
		for _, child := range n.Nodes {
			Walk(v, child)
		}

	case *ast.TypeAssertion:
		Walk(v, n.Expr)

	case *ast.TypeSwitch:
		Walk(v, n.Init)
		Walk(v, n.Assignment)
		for _, c := range n.Cases {
			Walk(v, c)
		}

	case *ast.URL:
		for _, child := range n.Value {
			Walk(v, child)
		}

	case *ast.UnaryOperator:
		Walk(v, n.Expr)

	case *ast.Var:
		for _, ident := range n.Lhs {
			Walk(v, ident)
		}
		Walk(v, n.Type)
		for _, value := range n.Rhs {
			Walk(v, value)
		}

	case *ast.Extends:
	case *ast.Import:
	case *ast.Render:
	// Nothing to do, visiting the expanded tree is done
	// by the Visit function if necessary.

	case *ast.BasicLiteral,
		*ast.Identifier,
		*ast.Comment,
		*ast.Text,
		*ast.Raw,
		*ast.Placeholder,
		*ast.Interface,
		*ast.Fallthrough:
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
