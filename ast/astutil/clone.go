// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package astutil

import (
	"fmt"

	"open2b/template/ast"
)

// CloneTree returns a complete copy of tree.
func CloneTree(tree *ast.Tree) *ast.Tree {
	return CloneNode(tree).(*ast.Tree)
}

// CloneNode returns a deep copy of node.
func CloneNode(node ast.Node) ast.Node {
	switch n := node.(type) {
	case *ast.Tree:
		var nn = make([]ast.Node, 0, len(n.Nodes))
		for _, n := range n.Nodes {
			nn = append(nn, CloneNode(n))
		}
		return ast.NewTree(n.Path, nn, n.Context)
	case *ast.Text:
		var text []byte
		if n.Text != nil {
			text = make([]byte, len(n.Text))
			copy(text, n.Text)
		}
		return ast.NewText(ClonePosition(n.Position), text, n.Cut)
	case *ast.URL:
		var value = make([]ast.Node, len(n.Value))
		for i, n2 := range n.Value {
			value[i] = CloneNode(n2)
		}
		return ast.NewURL(ClonePosition(n.Position), n.Tag, n.Attribute, value)
	case *ast.Value:
		return *ast.NewValue(ClonePosition(n.Position), CloneExpression(n.Expr), n.Context)
	case *ast.If:
		var assignment *ast.Assignment
		if n.Assignment != nil {
			assignment = CloneNode(n.Assignment).(*ast.Assignment)
		}
		var then = make([]ast.Node, len(n.Then))
		for i, n2 := range n.Then {
			then[i] = CloneNode(n2)
		}
		var els []ast.Node
		if n.Else != nil {
			els = make([]ast.Node, len(n.Else))
			for i, n2 := range n.Else {
				els[i] = CloneNode(n2)
			}
		}
		return ast.NewIf(ClonePosition(n.Position), assignment, CloneExpression(n.Condition), then, els)
	case *ast.For:
		var body = make([]ast.Node, len(n.Body))
		for i, n2 := range n.Body {
			body[i] = CloneNode(n2)
		}
		var init, post *ast.Assignment
		if n.Init != nil {
			init = CloneNode(n.Init).(*ast.Assignment)
		}
		if n.Post != nil {
			post = CloneNode(n.Post).(*ast.Assignment)
		}
		return ast.NewFor(ClonePosition(n.Position), init, CloneExpression(n.Condition), post, body)
	case *ast.ForRange:
		var body = make([]ast.Node, len(n.Body))
		for i, n2 := range n.Body {
			body[i] = CloneNode(n2)
		}
		assignment := CloneNode(n.Assignment).(*ast.Assignment)
		return ast.NewForRange(ClonePosition(n.Position), assignment, body)
	case *ast.Break:
		return ast.NewBreak(ClonePosition(n.Position))
	case *ast.Continue:
		return ast.NewContinue(ClonePosition(n.Position))
	case *ast.Extends:
		extends := ast.NewExtends(ClonePosition(n.Position), n.Path, n.Context)
		if n.Tree != nil {
			extends.Tree = CloneTree(n.Tree)
		}
		return extends
	case *ast.Macro:
		var ident = ast.NewIdentifier(ClonePosition(n.Ident.Position), n.Ident.Name)
		var parameters []*ast.Identifier
		if n.Parameters != nil {
			parameters = make([]*ast.Identifier, len(n.Parameters))
			for i, p := range n.Parameters {
				parameters[i] = ast.NewIdentifier(ClonePosition(p.Position), p.Name)
			}
		}
		var body = make([]ast.Node, len(n.Body))
		for i, n2 := range n.Body {
			body[i] = CloneNode(n2)
		}
		return ast.NewMacro(ClonePosition(n.Position), ident, parameters, body, n.IsVariadic, n.Context)
	case *ast.ShowMacro:
		var impor *ast.Identifier
		if n.Import != nil {
			impor = ast.NewIdentifier(ClonePosition(n.Import.Position), n.Import.Name)
		}
		var macro = ast.NewIdentifier(ClonePosition(n.Macro.Position), n.Macro.Name)
		var arguments []ast.Expression
		if n.Arguments != nil {
			arguments = make([]ast.Expression, len(n.Arguments))
			for i, a := range n.Arguments {
				arguments[i] = CloneExpression(a)
			}
		}
		return ast.NewShowMacro(ClonePosition(n.Position), impor, macro, arguments, n.Context)
	case *ast.Import:
		var ident *ast.Identifier
		if n.Ident != nil {
			ident = ast.NewIdentifier(ClonePosition(n.Ident.Position), n.Ident.Name)
		}
		imp := ast.NewImport(ClonePosition(n.Position), ident, n.Path, n.Context)
		if n.Tree != nil {
			imp.Tree = CloneTree(n.Tree)
		}
		return imp
	case *ast.Include:
		sp := ast.NewInclude(ClonePosition(n.Position), n.Path, n.Context)
		if n.Tree != nil {
			sp.Tree = CloneTree(n.Tree)
		}
		return sp
	case *ast.Assignment:
		variables := make([]ast.Expression, len(n.Variables))
		for i, v := range n.Variables {
			variables[i] = CloneExpression(v)
		}
		return ast.NewAssignment(ClonePosition(n.Position), variables, n.Type, CloneExpression(n.Expr))
	case *ast.Comment:
		return ast.NewComment(ClonePosition(n.Position), n.Text)
	case ast.Expression:
		return CloneExpression(n)
	default:
		panic(fmt.Sprintf("unexpected node type %#v", node))
	}
}

// CloneExpression returns a complete copy of expression expr.
func CloneExpression(expr ast.Expression) ast.Expression {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *ast.Parentesis:
		return ast.NewParentesis(ClonePosition(e.Position), CloneExpression(e.Expr))
	case *ast.Int:
		return ast.NewInt(ClonePosition(e.Position), e.Value)
	case *ast.Number:
		return ast.NewNumber(ClonePosition(e.Position), e.Value)
	case *ast.String:
		return ast.NewString(ClonePosition(e.Position), e.Text)
	case *ast.Identifier:
		return ast.NewIdentifier(ClonePosition(e.Position), e.Name)
	case *ast.UnaryOperator:
		return ast.NewUnaryOperator(ClonePosition(e.Position), e.Op, CloneExpression(e.Expr))
	case *ast.BinaryOperator:
		return ast.NewBinaryOperator(ClonePosition(e.Position), e.Op, CloneExpression(e.Expr1), CloneExpression(e.Expr2))
	case *ast.Map:
		var elements = make([]ast.KeyValue, len(e.Elements))
		for i, element := range e.Elements {
			elements[i].Key = CloneExpression(element.Key)
			elements[i].Value = CloneExpression(element.Value)
		}
		return ast.NewMap(ClonePosition(e.Position), elements)
	case *ast.Slice:
		var elements = make([]ast.Expression, len(e.Elements))
		for i, element := range e.Elements {
			elements[i] = CloneExpression(element)
		}
		return ast.NewSlice(ClonePosition(e.Position), elements)
	case *ast.Bytes:
		var elements = make([]ast.Expression, len(e.Elements))
		for i, element := range e.Elements {
			elements[i] = CloneExpression(element)
		}
		return ast.NewBytes(ClonePosition(e.Position), elements)
	case *ast.Call:
		var args = make([]ast.Expression, len(e.Args))
		for i, arg := range e.Args {
			args[i] = CloneExpression(arg)
		}
		return ast.NewCall(ClonePosition(e.Position), CloneExpression(e.Func), args)
	case *ast.Index:
		return ast.NewIndex(ClonePosition(e.Position), CloneExpression(e.Expr), CloneExpression(e.Index))
	case *ast.Slicing:
		return ast.NewSlicing(ClonePosition(e.Position), CloneExpression(e.Expr), CloneExpression(e.Low), CloneExpression(e.High))
	case *ast.Selector:
		return ast.NewSelector(ClonePosition(e.Position), CloneExpression(e.Expr), e.Ident)
	case *ast.TypeAssertion:
		typ := ast.NewIdentifier(ClonePosition(e.Type.Position), e.Type.Name)
		return ast.NewTypeAssertion(ClonePosition(e.Position), CloneExpression(e.Expr), typ)
	default:
		panic(fmt.Sprintf("unexpected node type %#v", expr))
	}
}

// ClonePosition returns a copy of position pos.
func ClonePosition(pos *ast.Position) *ast.Position {
	return &ast.Position{Line: pos.Line, Column: pos.Column, Start: pos.Start, End: pos.End}
}
