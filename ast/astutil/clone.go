// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package astutil

import (
	"fmt"

	"github.com/open2b/scriggo/ast"
)

// CloneTree returns a complete copy of tree.
func CloneTree(tree *ast.Tree) *ast.Tree {
	return CloneNode(tree).(*ast.Tree)
}

// CloneNode returns a deep copy of node.
func CloneNode(node ast.Node) ast.Node {

	switch n := node.(type) {

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

	case *ast.Block:
		var nodes []ast.Node
		if n.Nodes != nil {
			nodes = make([]ast.Node, len(n.Nodes))
			for i := range n.Nodes {
				nodes[i] = CloneNode(n.Nodes[i])
			}
		}
		return ast.NewBlock(ClonePosition(n.Position), nodes)

	case *ast.Break:
		label := CloneExpression(n.Label).(*ast.Identifier)
		return ast.NewBreak(ClonePosition(n.Position), label)

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

	case *ast.Comment:
		return ast.NewComment(ClonePosition(n.Position), n.Text)

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

	case *ast.Continue:
		label := CloneExpression(n.Label).(*ast.Identifier)
		return ast.NewContinue(ClonePosition(n.Position), label)

	case *ast.Defer:
		return ast.NewDefer(ClonePosition(n.Position), CloneExpression(n.Call))

	case ast.Expression:
		return CloneExpression(n)

	case *ast.Extends:
		extends := ast.NewExtends(ClonePosition(n.Position), n.Path, n.Format)
		if n.Tree != nil {
			extends.Tree = CloneTree(n.Tree)
		}
		return extends

	case *ast.Fallthrough:
		return ast.NewFallthrough(ClonePosition(n.Position))

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

	case *ast.ForIn:
		ident := CloneNode(n.Ident).(*ast.Identifier)
		expr := CloneExpression(n.Expr)
		var body = make([]ast.Node, len(n.Body))
		for i, n2 := range n.Body {
			body[i] = CloneNode(n2)
		}
		return ast.NewForIn(ClonePosition(n.Position), ident, expr, body)

	case *ast.ForRange:
		var body = make([]ast.Node, len(n.Body))
		for i, n2 := range n.Body {
			body[i] = CloneNode(n2)
		}
		assignment := CloneNode(n.Assignment).(*ast.Assignment)
		return ast.NewForRange(ClonePosition(n.Position), assignment, body)

	case *ast.Func:
		var ident *ast.Identifier
		if n.Ident != nil {
			ident = ast.NewIdentifier(ClonePosition(n.Ident.Position), n.Ident.Name)
		}
		typ := CloneExpression(n.Type).(*ast.FuncType)
		return ast.NewFunc(ClonePosition(n.Position), ident, typ, CloneNode(n.Body).(*ast.Block), n.Endless, n.Format)

	case *ast.Go:
		return ast.NewGo(ClonePosition(n.Position), CloneExpression(n.Call))

	case *ast.Goto:
		return ast.NewGoto(ClonePosition(n.Position), CloneExpression(n.Label).(*ast.Identifier))

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

	case *ast.Import:
		var ident *ast.Identifier
		if n.Ident != nil {
			ident = ast.NewIdentifier(ClonePosition(n.Ident.Position), n.Ident.Name)
		}
		var forIdents []*ast.Identifier
		if n.For != nil {
			forIdents = make([]*ast.Identifier, len(n.For))
			for i, ident := range n.For {
				forIdents[i] = CloneExpression(ident).(*ast.Identifier)
			}
		}
		imp := ast.NewImport(ClonePosition(n.Position), ident, n.Path, forIdents)
		if n.Tree != nil {
			imp.Tree = CloneTree(n.Tree)
		}
		return imp

	case *ast.Label:
		return ast.NewLabel(ClonePosition(n.Position), CloneExpression(n.Ident).(*ast.Identifier), CloneNode(n.Statement))

	case *ast.Package:
		var nn = make([]ast.Node, 0, len(n.Declarations))
		for _, n := range n.Declarations {
			nn = append(nn, CloneNode(n))
		}
		return ast.NewPackage(ClonePosition(n.Position), n.Name, nn)

	case *ast.Raw:
		return ast.NewRaw(ClonePosition(n.Position), n.Marker, n.Tag, CloneNode(n.Text).(*ast.Text))

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

	case *ast.Send:
		return ast.NewSend(ClonePosition(n.Position), CloneExpression(n.Channel), CloneExpression(n.Value))

	case *ast.Show:
		expressions := make([]ast.Expression, len(n.Expressions))
		for i, expr := range n.Expressions {
			expressions[i] = CloneExpression(expr)
		}
		return ast.NewShow(ClonePosition(n.Position), expressions, n.Context)

	case *ast.Statements:
		var nodes []ast.Node
		if n.Nodes != nil {
			nodes = make([]ast.Node, len(n.Nodes))
			for i := range n.Nodes {
				nodes[i] = CloneNode(n.Nodes[i])
			}
		}
		return ast.NewStatements(ClonePosition(n.Position), nodes)

	case *ast.StructType:
		var fields []*ast.Field
		if n.Fields != nil {
			fields = make([]*ast.Field, len(n.Fields))
			for i, field := range n.Fields {
				var idents []*ast.Identifier
				if field.Idents != nil {
					idents = make([]*ast.Identifier, len(field.Idents))
					for j, ident := range field.Idents {
						idents[j] = CloneExpression(ident).(*ast.Identifier)
					}
				}
				var typ ast.Expression
				if field.Type != nil {
					typ = CloneExpression(field.Type)
				}
				fields[i] = ast.NewField(idents, typ, field.Tag)
			}
		}
		return ast.NewStructType(ClonePosition(n.Position), fields)

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

	case *ast.Text:
		var text []byte
		if n.Text != nil {
			text = make([]byte, len(n.Text))
			copy(text, n.Text)
		}
		return ast.NewText(ClonePosition(n.Position), text, n.Cut)

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

	case *ast.Tree:
		var nn = make([]ast.Node, 0, len(n.Nodes))
		for _, n := range n.Nodes {
			nn = append(nn, CloneNode(n))
		}
		return ast.NewTree(n.Path, nn, n.Format)

	case *ast.URL:
		var value = make([]ast.Node, len(n.Value))
		for i, n2 := range n.Value {
			value[i] = CloneNode(n2)
		}
		return ast.NewURL(ClonePosition(n.Position), n.Tag, n.Attribute, value)

	case *ast.Using:
		var body *ast.Block
		if n.Body != nil {
			body = CloneNode(n.Body).(*ast.Block)
		}
		return ast.NewUsing(ClonePosition(n.Position), CloneNode(n.Statement), CloneExpression(n.Type), body, n.Format)

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

	case *ast.ArrayType:
		expr2 = ast.NewArrayType(ClonePosition(e.Pos()), CloneExpression(e.Len), CloneExpression(e.ElementType))

	case *ast.BasicLiteral:
		expr2 = ast.NewBasicLiteral(ClonePosition(e.Position), e.Type, e.Value)

	case *ast.BinaryOperator:
		expr2 = ast.NewBinaryOperator(ClonePosition(e.Position), e.Op, CloneExpression(e.Expr1), CloneExpression(e.Expr2))

	case *ast.Call:
		var args = make([]ast.Expression, len(e.Args))
		for i, arg := range e.Args {
			args[i] = CloneExpression(arg)
		}
		expr2 = ast.NewCall(ClonePosition(e.Position), CloneExpression(e.Func), args, e.IsVariadic)

	case *ast.ChanType:
		expr2 = ast.NewChanType(ClonePosition(e.Pos()), e.Direction, CloneExpression(e.ElementType))

	case *ast.CompositeLiteral:
		keyValues := make([]ast.KeyValue, len(e.KeyValues))
		for i, kv := range e.KeyValues {
			keyValues[i].Key = CloneExpression(kv.Key)
			keyValues[i].Value = CloneExpression(kv.Value)
		}
		return ast.NewCompositeLiteral(ClonePosition(e.Pos()), CloneExpression(e.Type), keyValues)

	case *ast.Default:
		expr2 = ast.NewDefault(ClonePosition(e.Position), CloneExpression(e.Expr1), CloneExpression(e.Expr2))

	case *ast.DollarIdentifier:
		expr2 = ast.NewDollarIdentifier(ClonePosition(e.Position), CloneExpression(e.Ident).(*ast.Identifier))

	case *ast.Func:
		var ident *ast.Identifier
		if e.Ident != nil {
			// Ident must be nil for a function literal, but clone it anyway.
			ident = ast.NewIdentifier(ClonePosition(e.Ident.Position), e.Ident.Name)
		}
		typ := CloneExpression(e.Type).(*ast.FuncType)
		expr2 = ast.NewFunc(ClonePosition(e.Position), ident, typ, CloneNode(e.Body).(*ast.Block), false, e.Format)

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
		expr2 = ast.NewFuncType(ClonePosition(e.Position), e.Macro, parameters, result, e.IsVariadic)

	case *ast.Identifier:
		expr2 = ast.NewIdentifier(ClonePosition(e.Position), e.Name)

	case *ast.Index:
		expr2 = ast.NewIndex(ClonePosition(e.Position), CloneExpression(e.Expr), CloneExpression(e.Index))

	case *ast.Interface:
		expr2 = ast.NewInterface(ClonePosition(e.Pos()))

	case *ast.MapType:
		expr2 = ast.NewMapType(ClonePosition(e.Pos()), CloneExpression(e.KeyType), CloneExpression(e.ValueType))

	case *ast.Render:
		n := ast.NewRender(ClonePosition(e.Position), e.Path)
		if e.Tree != nil {
			n.Tree = CloneTree(e.Tree)
		}
		expr2 = n

	case *ast.Selector:
		expr2 = ast.NewSelector(ClonePosition(e.Position), CloneExpression(e.Expr), e.Ident)

	case *ast.SliceType:
		expr2 = ast.NewSliceType(ClonePosition(e.Pos()), CloneExpression(e.ElementType))

	case *ast.Slicing:
		expr2 = ast.NewSlicing(ClonePosition(e.Position), CloneExpression(e.Expr), CloneExpression(e.Low),
			CloneExpression(e.High), CloneExpression(e.Max), e.IsFull)

	case *ast.TypeAssertion:
		expr2 = ast.NewTypeAssertion(ClonePosition(e.Position), CloneExpression(e.Expr), CloneExpression(e.Type))

	case *ast.UnaryOperator:
		expr2 = ast.NewUnaryOperator(ClonePosition(e.Position), e.Op, CloneExpression(e.Expr))

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
