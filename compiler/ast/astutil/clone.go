// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package astutil

import (
	"fmt"

	"scriggo/compiler/ast"
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
		return ast.NewTree(n.Path, nn, n.Language)
	case *ast.Package:
		var nn = make([]ast.Node, 0, len(n.Declarations))
		for _, n := range n.Declarations {
			nn = append(nn, CloneNode(n))
		}
		return ast.NewPackage(ClonePosition(n.Position), n.Name, nn)
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
		return ast.NewURL(ClonePosition(n.Position), n.Tag, n.Attribute, value, n.Context)
	case *ast.Show:
		return *ast.NewShow(ClonePosition(n.Position), CloneExpression(n.Expr), n.Context)
	case *ast.Block:
		var b *ast.Block
		if n.Nodes != nil {
			b.Nodes = make([]ast.Node, len(n.Nodes))
			for i := range n.Nodes {
				b.Nodes[i] = CloneNode(n.Nodes[i])
			}
		}
		return b
	case *ast.If:
		var init ast.Node
		if n.Init != nil {
			init = CloneNode(n.Init)
		}
		var then *ast.Block
		if n.Then != nil {
			then = CloneNode(n.Then).(*ast.Block)
		}
		var els ast.Node
		if n.Else != nil {
			els = CloneNode(n.Else)
		}
		return ast.NewIf(ClonePosition(n.Position), init, CloneExpression(n.Condition), then, els)
	case *ast.For:
		var body = make([]ast.Node, len(n.Body))
		for i, n2 := range n.Body {
			body[i] = CloneNode(n2)
		}
		var init, post ast.Node
		if n.Init != nil {
			init = CloneNode(n.Init)
		}
		if n.Post != nil {
			post = CloneNode(n.Post)
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
		label := CloneExpression(n.Label).(*ast.Identifier)
		return ast.NewBreak(ClonePosition(n.Position), label)
	case *ast.Continue:
		label := CloneExpression(n.Label).(*ast.Identifier)
		return ast.NewContinue(ClonePosition(n.Position), label)
	case *ast.Switch:
		var init ast.Node
		if n.Init != nil {
			init = CloneNode(n.Init)
		}
		var text *ast.Text
		if n.LeadingText != nil {
			text = CloneNode(n.LeadingText).(*ast.Text)
		}
		var cases []*ast.Case
		if n.Cases != nil {
			cases = make([]*ast.Case, len(n.Cases))
			for i, c := range n.Cases {
				cases[i] = CloneNode(c).(*ast.Case)
			}
		}
		return ast.NewSwitch(ClonePosition(n.Position), init, CloneExpression(n.Expr), text, cases)
	case *ast.TypeSwitch:
		var init ast.Node
		if n.Init != nil {
			init = CloneNode(n.Init)
		}
		var assignment *ast.Assignment
		if n.Assignment != nil {
			assignment = CloneNode(n.Assignment).(*ast.Assignment)
		}
		var text *ast.Text
		if n.LeadingText != nil {
			text = CloneNode(n.LeadingText).(*ast.Text)
		}
		var cases []*ast.Case
		if n.Cases != nil {
			cases = make([]*ast.Case, len(n.Cases))
			for i, c := range n.Cases {
				cases[i] = CloneNode(c).(*ast.Case)
			}
		}
		return ast.NewTypeSwitch(ClonePosition(n.Position), init, assignment, text, cases)
	case *ast.Case:
		var expressions []ast.Expression
		if n.Expressions != nil {
			expressions = make([]ast.Expression, len(n.Expressions))
			for i, e := range n.Expressions {
				expressions[i] = CloneExpression(e)
			}
		}
		var body []ast.Node
		if n.Body != nil {
			body = make([]ast.Node, len(n.Body))
			for i, node := range n.Body {
				body[i] = CloneNode(node)
			}
		}
		return ast.NewCase(ClonePosition(n.Position), expressions, body)
	case *ast.Fallthrough:
		return ast.NewFallthrough(ClonePosition(n.Position))
	case *ast.Select:
		var text *ast.Text
		if n.LeadingText != nil {
			text = CloneNode(n.LeadingText).(*ast.Text)
		}
		var cases []*ast.SelectCase
		if n.Cases != nil {
			cases = make([]*ast.SelectCase, len(n.Cases))
			for i, c := range n.Cases {
				cases[i] = CloneNode(c).(*ast.SelectCase)
			}
		}
		return ast.NewSelect(ClonePosition(n.Position), text, cases)
	case *ast.SelectCase:
		var comm ast.Node
		if n.Comm != nil {
			comm = CloneNode(n.Comm)
		}
		var body []ast.Node
		if n.Body != nil {
			body = make([]ast.Node, len(n.Body))
			for i, node := range n.Body {
				body[i] = CloneNode(node)
			}
		}
		return ast.NewSelectCase(ClonePosition(n.Position), comm, body)
	case *ast.Extends:
		extends := ast.NewExtends(ClonePosition(n.Position), n.Path, n.Context)
		if n.Tree != nil {
			extends.Tree = CloneTree(n.Tree)
		}
		return extends
	case *ast.Macro:
		var ident *ast.Identifier
		if n.Ident != nil {
			// Ident must be nil for a function literal, but clone it anyway.
			ident = ast.NewIdentifier(ClonePosition(n.Ident.Position), n.Ident.Name)
		}
		typ := CloneExpression(n.Type).(*ast.FuncType)
		var body = make([]ast.Node, len(n.Body))
		for i, n2 := range n.Body {
			body[i] = CloneNode(n2)
		}
		return ast.NewMacro(ClonePosition(n.Position), ident, typ, body, n.Context)
	case *ast.ShowMacro:
		var macro = CloneExpression(n.Macro)
		var arguments []ast.Expression
		if n.Args != nil {
			arguments = make([]ast.Expression, len(n.Args))
			for i, a := range n.Args {
				arguments[i] = CloneExpression(a)
			}
		}
		return ast.NewShowMacro(ClonePosition(n.Position), macro, arguments, n.IsVariadic, n.Or, n.Context)
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
	case *ast.ShowFile:
		sp := ast.NewShowFile(ClonePosition(n.Position), n.Path, n.Context)
		if n.Tree != nil {
			sp.Tree = CloneTree(n.Tree)
		}
		return sp
	case *ast.Var:
		idents := make([]*ast.Identifier, len(n.Lhs))
		for i, v := range n.Lhs {
			idents[i] = CloneExpression(v).(*ast.Identifier)
		}
		typ := CloneExpression(n.Type)
		values := make([]ast.Expression, len(n.Rhs))
		for i, v := range n.Rhs {
			values[i] = CloneExpression(v)
		}
		return ast.NewVar(ClonePosition(n.Position), idents, typ, values)
	case *ast.Const:
		idents := make([]*ast.Identifier, len(n.Lhs))
		for i, v := range n.Lhs {
			idents[i] = CloneExpression(v).(*ast.Identifier)
		}
		typ := CloneExpression(n.Type)
		values := make([]ast.Expression, len(n.Rhs))
		for i, v := range n.Rhs {
			values[i] = CloneExpression(v)
		}
		return ast.NewConst(ClonePosition(n.Position), idents, typ, values, n.Index)
	case *ast.Assignment:
		variables := make([]ast.Expression, len(n.Lhs))
		for i, v := range n.Lhs {
			variables[i] = CloneExpression(v)
		}
		values := make([]ast.Expression, len(n.Rhs))
		for i, v := range n.Rhs {
			variables[i] = CloneExpression(v)
		}
		return ast.NewAssignment(ClonePosition(n.Position), variables, n.Type, values)
	case *ast.Comment:
		return ast.NewComment(ClonePosition(n.Position), n.Text)
	case *ast.Func:
		var ident *ast.Identifier
		if n.Ident != nil {
			ident = ast.NewIdentifier(ClonePosition(n.Ident.Position), n.Ident.Name)
		}
		typ := CloneExpression(n.Type).(*ast.FuncType)
		return ast.NewFunc(ClonePosition(n.Position), ident, typ, CloneNode(n.Body).(*ast.Block))
	case *ast.Defer:
		return ast.NewDefer(ClonePosition(n.Position), CloneExpression(n.Call))
	case *ast.Go:
		return ast.NewGo(ClonePosition(n.Position), CloneExpression(n.Call))
	case *ast.Goto:
		return ast.NewGoto(ClonePosition(n.Position), CloneExpression(n.Label).(*ast.Identifier))
	case *ast.Label:
		return ast.NewLabel(ClonePosition(n.Position), CloneExpression(n.Ident).(*ast.Identifier), CloneNode(n.Statement))
	case *ast.Send:
		return ast.NewSend(ClonePosition(n.Position), CloneExpression(n.Channel), CloneExpression(n.Value))
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
	var expr2 ast.Expression
	switch e := expr.(type) {
	case *ast.BasicLiteral:
		expr2 = ast.NewBasicLiteral(ClonePosition(e.Position), e.Type, e.Value)
	case *ast.Identifier:
		expr2 = ast.NewIdentifier(ClonePosition(e.Position), e.Name)
	case *ast.UnaryOperator:
		expr2 = ast.NewUnaryOperator(ClonePosition(e.Position), e.Op, CloneExpression(e.Expr))
	case *ast.BinaryOperator:
		expr2 = ast.NewBinaryOperator(ClonePosition(e.Position), e.Op, CloneExpression(e.Expr1), CloneExpression(e.Expr2))
	case *ast.MapType:
		expr2 = ast.NewMapType(ClonePosition(e.Pos()), CloneExpression(e.KeyType), CloneExpression(e.ValueType))
	case *ast.SliceType:
		expr2 = ast.NewSliceType(ClonePosition(e.Pos()), CloneExpression(e.ElementType))
	case *ast.ArrayType:
		expr2 = ast.NewArrayType(ClonePosition(e.Pos()), CloneExpression(e.Len), CloneExpression(e.ElementType))
	case *ast.ChanType:
		expr2 = ast.NewChanType(ClonePosition(e.Pos()), e.Direction, CloneExpression(e.ElementType))
	case *ast.CompositeLiteral:
		keyValues := make([]ast.KeyValue, len(e.KeyValues))
		for i, kv := range e.KeyValues {
			keyValues[i].Key = CloneExpression(kv.Key)
			keyValues[i].Value = CloneExpression(kv.Value)
		}
		return ast.NewCompositeLiteral(ClonePosition(e.Pos()), CloneExpression(e.Type), keyValues)
	case *ast.Interface:
		expr2 = ast.NewInterface(ClonePosition(e.Pos()))
	case *ast.Call:
		var args = make([]ast.Expression, len(e.Args))
		for i, arg := range e.Args {
			args[i] = CloneExpression(arg)
		}
		expr2 = ast.NewCall(ClonePosition(e.Position), CloneExpression(e.Func), args, e.IsVariadic)
	case *ast.Index:
		expr2 = ast.NewIndex(ClonePosition(e.Position), CloneExpression(e.Expr), CloneExpression(e.Index))
	case *ast.Slicing:
		expr2 = ast.NewSlicing(ClonePosition(e.Position), CloneExpression(e.Expr), CloneExpression(e.Low),
			CloneExpression(e.High), CloneExpression(e.Max), e.IsFull)
	case *ast.Selector:
		expr2 = ast.NewSelector(ClonePosition(e.Position), CloneExpression(e.Expr), e.Ident)
	case *ast.TypeAssertion:
		expr2 = ast.NewTypeAssertion(ClonePosition(e.Position), CloneExpression(e.Expr), CloneExpression(e.Type))
	case *ast.FuncType:
		var parameters []*ast.Parameter
		if e.Parameters != nil {
			parameters = make([]*ast.Parameter, len(e.Parameters))
			for i, param := range e.Parameters {
				var ident *ast.Identifier
				if param.Ident != nil {
					ident = ast.NewIdentifier(ClonePosition(param.Ident.Position), param.Ident.Name)
				}
				parameters[i] = &ast.Parameter{Ident: ident, Type: CloneExpression(param.Type)}
			}
		}
		var result []*ast.Parameter
		if e.Result != nil {
			result = make([]*ast.Parameter, len(e.Result))
			for i, res := range e.Result {
				var ident *ast.Identifier
				if res.Ident != nil {
					ident = ast.NewIdentifier(ClonePosition(res.Ident.Position), res.Ident.Name)
				}
				result[i] = &ast.Parameter{Ident: ident, Type: CloneExpression(res.Type)}
			}
		}
		expr2 = ast.NewFuncType(ClonePosition(e.Position), parameters, result, e.IsVariadic)
	case *ast.Func:
		var ident *ast.Identifier
		if e.Ident != nil {
			// Ident must be nil for a function literal, but clone it anyway.
			ident = ast.NewIdentifier(ClonePosition(e.Ident.Position), e.Ident.Name)
		}
		typ := CloneExpression(e.Type).(*ast.FuncType)
		expr2 = ast.NewFunc(ClonePosition(e.Position), ident, typ, CloneNode(e.Body).(*ast.Block))
	default:
		panic(fmt.Sprintf("unexpected node type %#v", expr))
	}
	expr2.SetParenthesis(expr.Parenthesis())
	return expr2
}

// ClonePosition returns a copy of position pos.
func ClonePosition(pos *ast.Position) *ast.Position {
	return &ast.Position{Line: pos.Line, Column: pos.Column, Start: pos.Start, End: pos.End}
}
