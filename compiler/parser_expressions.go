// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"unicode/utf8"

	"github.com/open2b/scriggo/compiler/ast"
)

// parseExpr parses an expression and returns its tree and the last read token
// that does not belong to the expression. It panics on error.
//
// tok is the first token of the expression. canBeSwitchGuard reports whether
// the parsed expression can be a type switch guard, as x.(type). mustBeGuard
// reports whatever the expression can be a type. nextIsBlockBrace report
// whether a left brace block is expected after the expression.
func (p *parsing) parseExpr(tok token, canBeSwitchGuard, mustBeType, nextIsBlockBrace bool) (ast.Expression, token) {

	// canCompositeLiteral reports whether the currently parsed expression can
	// be used as type in composite literals.
	canCompositeLiteral := false

	// path is the tree path that starts from the root operator and ends with
	// the leaf operator.
	var path []ast.Operator

	// mustBeSwitchGuard reports whether the parsed expression must be a type
	// switch guard, that is `expr.(type)`.
	var mustBeSwitchGuard bool

	for {

		var operand ast.Expression
		var operator ast.Operator

		switch tok.typ {
		case tokenLeftParenthesis: // ( e )
			// Call parseExpr recursively to parse the expression in
			// parenthesis and then handle it as a single operand.
			pos := tok.pos
			var expr ast.Expression
			expr, tok = p.parseExpr(p.next(), false, mustBeType, false)
			if expr == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			if tok.typ != tokenRightParenthesis {
				panic(syntaxError(tok.pos, "unexpected %s, expecting )", tok))
			}
			expr.SetParenthesis(expr.Parenthesis() + 1)
			operand = expr
			operand.Pos().Start = pos.Start
			operand.Pos().End = tok.pos.End
			tok = p.next()
		case tokenLeftBraces: // {
			// composite literal with no type.
			if mustBeType {
				panic(syntaxError(tok.pos, "unexpected {, expecting type"))
			}
		case tokenMap: // map
			canCompositeLiteral = true
			mapType := ast.NewMapType(tok.pos, nil, nil)
			tok = p.next()
			if tok.typ != tokenLeftBrackets {
				panic(syntaxError(tok.pos, "unexpected %s, expecting [", tok))
			}
			var typ ast.Expression
			typ, tok = p.parseExpr(p.next(), false, true, false)
			if tok.typ != tokenRightBrackets {
				panic(syntaxError(tok.pos, "unexpected %s, expecting %s", tok, tokenRightBraces))
			}
			mapType.KeyType = typ
			typ, tok = p.parseExpr(p.next(), false, true, false)
			if typ == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting type", tok))
			}
			mapType.Position.End = typ.Pos().End
			mapType.ValueType = typ
			operand = mapType
		case tokenStruct: // struct
			canCompositeLiteral = true
			structType := ast.NewStructType(tok.pos, nil)
			tok = p.next()
			if tok.typ != tokenLeftBraces {
				panic(syntaxError(tok.pos, "unexpected %s, expecting {", tok))
			}
			tok = p.next()
			if tok.typ != tokenRightBraces {
				for {
					fieldDecl := ast.NewField(nil, nil, nil)
					var exprs []ast.Expression
					var typ ast.Expression
					exprs, tok = p.parseExprList(tok, false, true, false)
					if tok.typ == tokenSemicolon || tok.typ == tokenRightBraces {
						// Implicit field declaration.
						if len(exprs) != 1 {
							panic(syntaxError(tok.pos, "unexpected %s, expecting type", tok))
						}
						fieldDecl.Type = exprs[0]
						if tok.typ == tokenSemicolon {
							tok = p.next()
						}
					} else {
						// Explicit field declaration.
						typ, tok = p.parseExpr(tok, false, true, false)
						if tok.typ != tokenSemicolon && tok.typ != tokenRightBraces {
							panic(syntaxError(tok.pos, "unexpected %s, expecting semicolon", tok))
						}
						if tok.typ == tokenSemicolon {
							tok = p.next()
						}
						fieldDecl.Idents = make([]*ast.Identifier, len(exprs))
						for i, e := range exprs {
							ident, ok := e.(*ast.Identifier)
							if !ok {
								panic(syntaxError(tok.pos, "unexpected %s, expecting field name or embedded type ", e))
							}
							fieldDecl.Idents[i] = ident
						}
						fieldDecl.Type = typ
					}
					structType.Fields = append(structType.Fields, fieldDecl)
					if tok.typ == tokenRightBraces {
						break
					}
				}
			}
			structType.Position.End = tok.pos.End
			operand = structType
			tok = p.next()
		case tokenInterface: // interface{}
			pos := tok.pos
			tok = p.next()
			if tok.typ != tokenLeftBraces {
				panic(syntaxError(tok.pos, "unexpected %s, expecting {", tok))
			}
			tok = p.next()
			if tok.typ != tokenRightBraces {
				panic(syntaxError(tok.pos, "unexpected %s, expecting }", tok))
			}
			pos.End = tok.pos.End
			operand = ast.NewInterface(pos)
			tok = p.next()
		case tokenFunc: // func
			var node ast.Node
			if mustBeType {
				node, tok = p.parseFunc(tok, parseFuncType)
			} else {
				node, tok = p.parseFunc(tok, parseFuncType|parseFuncLit)
			}
			operand = node.(ast.Expression)
		case
			tokenArrow, // <-, <-chan
			tokenChan:  // chan, chan<-
			pos := tok.pos
			direction := ast.NoDirection
			if tok.typ == tokenArrow {
				tok = p.next()
				if tok.typ == tokenChan {
					direction = ast.ReceiveDirection
				} else {
					operator = ast.NewUnaryOperator(pos, ast.OperatorReceive, nil)
					if mustBeType {
						panic(syntaxError(tok.pos, "unexpected %s, expecting type", operator))
					}
				}
			}
			if operator == nil {
				tok = p.next()
				if direction == ast.NoDirection && tok.typ == tokenArrow {
					direction = ast.SendDirection
					tok = p.next()
				}
				var elemType ast.Expression
				elemType, tok = p.parseExpr(tok, false, true, false)
				if elemType == nil {
					panic(syntaxError(tok.pos, "missing channel element type"))
				}
				pos.End = elemType.Pos().End
				operand = ast.NewChanType(pos, direction, elemType)
			}
		case
			tokenAddition,       // +e
			tokenSubtraction,    // -e
			tokenNot,            // !e
			tokenRelaxedNot,     // not e
			tokenXor,            // ^e
			tokenMultiplication, // *t, *T
			tokenAmpersand:      // &e
			operator = ast.NewUnaryOperator(tok.pos, operatorFromTokenType(tok.typ, false), nil)
			if mustBeType && tok.typ != tokenMultiplication {
				panic(syntaxError(tok.pos, "unexpected %s, expecting type", tok.txt))
			}
			tok = p.next()
		case
			tokenRune,      // '\x3c'
			tokenInt,       // 18
			tokenFloat,     // 12.895
			tokenImaginary: // 7.2i
			if mustBeType {
				panic(syntaxError(tok.pos, "unexpected literal %s, expecting type", tok.txt))
			}
			operand = ast.NewBasicLiteral(tok.pos, literalType(tok.typ), string(tok.txt))
			tok = p.next()
		case
			tokenInterpretedString, // ""
			tokenRawString:         // ``
			if mustBeType {
				panic(syntaxError(tok.pos, "unexpected literal %s, expecting type", tok.txt))
			}
			operand = ast.NewBasicLiteral(tok.pos, literalType(tok.typ), string(tok.txt))
			tok = p.next()
		case tokenIdentifier: // a
			ident := p.parseIdentifierNode(tok)
			operand = ident
			tok = p.next()
			if mustBeType {
				if tok.typ == tokenPeriod {
					tok = p.next()
					if tok.typ != tokenIdentifier {
						panic(syntaxError(tok.pos, "unexpected %s, expecting name", tok.txt))
					}
					ident := p.parseIdentifierNode(tok)
					operand = ast.NewSelector(tok.pos, operand, ident.Name)
					tok = p.next()
				}
			}
		case tokenLeftBrackets: // [
			canCompositeLiteral = true
			var expr, length ast.Expression
			pos := tok.pos
			isEllipsis := false
			tok = p.next()
			switch tok.typ {
			case tokenEllipsis:
				isEllipsis = true
				tok = p.next()
			case tokenRightBrackets:
			default:
				oldTok := tok
				expr, tok = p.parseExpr(tok, false, false, false)
				if expr == nil {
					panic(syntaxError(tok.pos, "unexpected %s, expecting expression", oldTok))
				}
				length = expr
			}
			if tok.typ != tokenRightBrackets {
				panic(syntaxError(tok.pos, "unexpected %s, expecting ]", tok))
			}
			var typ ast.Expression
			typ, tok = p.parseExpr(p.next(), false, true, false)
			if typ == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			pos.End = typ.Pos().End
			switch {
			case isEllipsis:
				operand = ast.NewArrayType(pos, nil, typ)
			case length == nil:
				operand = ast.NewSliceType(pos, typ)
			default:
				operand = ast.NewArrayType(pos, length, typ)
			}
		default:
			return nil, tok
		}

		for operator == nil {

			dontEatLeftBraces := tok.typ == tokenLeftBraces && nextIsBlockBrace && !canCompositeLiteral
			if dontEatLeftBraces || mustBeType {
				operand = addLastOperand(operand, path)
				return operand, tok
			}

			switch tok.typ {

			case tokenLeftBraces: // ...{
				canCompositeLiteral = false
				if operand != nil && operand.Parenthesis() > 0 {
					panic(syntaxError(tok.pos, "cannot parenthesize type in composite literal"))
				}
				pos := &ast.Position{Line: tok.pos.Line, Column: tok.pos.Column, Start: tok.pos.Start, End: tok.pos.End}
				if operand != nil {
					pos.Start = operand.Pos().Start
				}
				var keyValues []ast.KeyValue
				var expr ast.Expression
				for {
					expr, tok = p.parseExpr(p.next(), false, false, false)
					if expr == nil {
						break
					}
					switch tok.typ {
					case tokenColon:
						var value ast.Expression
						value, tok = p.parseExpr(p.next(), false, false, false)
						if value == nil {
							panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
						}
						keyValues = append(keyValues, ast.KeyValue{Key: expr, Value: value})
					case tokenComma, tokenRightBraces:
						keyValues = append(keyValues, ast.KeyValue{Key: nil, Value: expr})
					default:
						panic(syntaxError(tok.pos, "unexpected %s, expecting comma or }", tok))
					}
					if tok.typ == tokenRightBraces {
						break
					}
				}
				if tok.typ != tokenRightBraces {
					panic(syntaxError(tok.pos, "unexpected %s, expecting expression or }", tok))
				}
				pos.End = tok.pos.End
				operand = ast.NewCompositeLiteral(pos, operand, keyValues)
			case tokenLeftParenthesis: // e(...)
				pos := tok.pos
				pos.Start = operand.Pos().Start
				var args []ast.Expression
				args, tok = p.parseExprListInParenthesis(p.next())
				var isVariadic bool
				if tok.typ == tokenEllipsis {
					if args == nil {
						panic(syntaxError(tok.pos, "unexpected ..., expecting expression"))
					}
					isVariadic = true
					tok = p.next()
				}
				if tok.typ != tokenRightParenthesis {
					panic(syntaxError(tok.pos, "unexpected %s, expecting expression or )", tok))
				}
				pos.End = tok.pos.End
				operand = ast.NewCall(pos, operand, args, isVariadic)
				canCompositeLiteral = false
			case tokenLeftBrackets: // e[...], e[.. : ..], e[.. : .. : ..],
				pos := tok.pos
				pos.Start = operand.Pos().Start
				var index ast.Expression
				index, tok = p.parseExpr(p.next(), false, false, false)
				if tok.typ == tokenColon {
					low := index
					isFull := false
					var high, max ast.Expression
					high, tok = p.parseExpr(p.next(), false, false, false)
					if tok.typ == tokenColon {
						isFull = true
						max, tok = p.parseExpr(p.next(), false, false, false)
					}
					if tok.typ != tokenRightBrackets {
						panic(syntaxError(tok.pos, "unexpected %s, expecting ]", tok))
					}
					pos.End = tok.pos.End
					operand = ast.NewSlicing(pos, operand, low, high, max, isFull)
				} else {
					if tok.typ != tokenRightBrackets {
						panic(syntaxError(tok.pos, "unexpected %s, expecting ]", tok))
					}
					if index == nil {
						panic(syntaxError(tok.pos, "unexpected ], expecting expression"))
					}
					pos.End = tok.pos.End
					operand = ast.NewIndex(pos, operand, index)
				}
			case tokenPeriod: // e.
				pos := tok.pos
				pos.Start = operand.Pos().Start
				tok = p.next()
				switch tok.typ {
				case tokenIdentifier:
					// e.ident
					ident := string(tok.txt)
					pos.End = tok.pos.End
					operand = ast.NewSelector(pos, operand, ident)
				case tokenLeftParenthesis:
					// e.(ident), e.(pkg.ident)
					tok = p.next()
					var typ ast.Expression
					switch tok.typ {
					case tokenType:
						if !canBeSwitchGuard {
							panic(syntaxError(tok.pos, "use of .(type) outside type switch"))
						}
						mustBeSwitchGuard = true
						tok = p.next()
					case tokenIdentifier:
						if len(tok.txt) == 1 && tok.txt[0] == '_' {
							panic(syntaxError(tok.pos, "cannot use _ as value"))
						}
						fallthrough
					default:
						typ, tok = p.parseExpr(tok, true, true, false)
						if typ == nil {
							panic(syntaxError(tok.pos, "unexpected %s, expecting type", tok))
						}
					}
					if tok.typ != tokenRightParenthesis {
						panic(syntaxError(tok.pos, "unexpected %s, expecting )", tok))
					}
					pos.End = tok.pos.End
					operand = ast.NewTypeAssertion(pos, operand, typ)
				default:
					panic(syntaxError(tok.pos, "unexpected %s, expecting name or (", tok))
				}
			case tokenGlobalAssertion: // id::
			    // id::(T)
				pos := operand.Pos()
				if operand.Parenthesis() > 0 {
					panic(syntaxError(pos, "cannot parenthesize identifier in global assertion"))
				}
				ident, ok := operand.(*ast.Identifier)
				if !ok {
					panic(syntaxError(pos, "cannot use %s as identifier in global assertion", ident))
				}
				if ident.Name == "_" {
					panic(syntaxError(pos, "cannot use _ as value"))
				}
				tok = p.next()
				if tok.typ != tokenLeftParenthesis {
					panic(syntaxError(tok.pos, "unexpected %s, expecting (", tok))
				}
				tok = p.next()
				if len(tok.txt) == 1 && tok.txt[0] == '_' {
					panic(syntaxError(tok.pos, "cannot use _ as value"))
				}
				var typ ast.Expression
				typ, tok = p.parseExpr(tok, false, true, false)
				if typ == nil {
					panic(syntaxError(tok.pos, "unexpected %s, expecting type", tok))
				}
				if tok.typ != tokenRightParenthesis {
					panic(syntaxError(tok.pos, "unexpected %s, expecting )", tok))
				}
				operand = ast.NewGlobalAssertion(pos.WithEnd(tok.pos.End), ident, typ)
			case
				tokenEqual,          // e ==
				tokenNotEqual,       // e !=
				tokenLess,           // e <
				tokenLessOrEqual,    // e <=
				tokenGreater,        // e >
				tokenGreaterOrEqual, // e >=
				tokenAnd,            // e &&
				tokenOr,             // e ||
				tokenRelaxedAnd,     // e and
				tokenRelaxedOr,      // e or
				tokenAddition,       // e +
				tokenSubtraction,    // e -
				tokenMultiplication, // e *
				tokenDivision,       // e /
				tokenModulo,         // e %
				tokenAmpersand,      // e &
				tokenVerticalBar,    // e |
				tokenXor,            // e ^
				tokenAndNot,         // e &^
				tokenLeftShift,      // e <<
				tokenRightShift:     // e >>
				operator = ast.NewBinaryOperator(tok.pos, operatorFromTokenType(tok.typ, true), nil, nil)
			default:
				if mustBeSwitchGuard && !isTypeGuard(operand) {
					panic(syntaxError(tok.pos, "use of .(type) outside type switch"))
				}
				operand = addLastOperand(operand, path)
				return operand, tok
			}

			tok = p.next()

		}

		canBeSwitchGuard = false

		// Add the operator to the expression tree.

		switch op := operator.(type) {

		case *ast.UnaryOperator:
			// An unary operator ("!", "+", "-", "()") becomes the new leaf
			// operator as it has an higher precedence than all the other
			// operators.

			if len(path) > 0 {
				// operator becomes a child of the leaf operator.
				switch leaf := path[len(path)-1].(type) {
				case *ast.UnaryOperator:
					leaf.Expr = op
				case *ast.BinaryOperator:
					leaf.Expr2 = op
				}
			}
			// operator becomes the new leaf operator.
			path = append(path, op)

		case *ast.BinaryOperator:
			// For a binary operator ("*", "/", "+", "-", "<", ">", ...),
			// start from the leaf operator (last operator of the path) and
			// go up to the root (first operator of the path) stopping if an
			// operator with lower precedence is found.

			// For all unary operators, set the start at the end of the path.
			start := operand.Pos().Start
			for i := len(path) - 1; i >= 0; i-- {
				if o, ok := path[i].(*ast.UnaryOperator); ok {
					o.Position.Start = start
				} else {
					break
				}
			}

			// p is the position in the path where to add the operator.
			var p = len(path)
			for p > 0 && op.Precedence() <= path[p-1].Precedence() {
				p--
			}
			if p > 0 {
				// operator becomes the child of the operator with lower
				// precedence found going up the path.
				switch o := path[p-1].(type) {
				case *ast.UnaryOperator:
					o.Expr = op
				case *ast.BinaryOperator:
					o.Expr2 = op
				}
			}
			if p < len(path) {
				// operand becomes the child of the leaf operator.
				switch o := path[len(path)-1].(type) {
				case *ast.UnaryOperator:
					o.Expr = operand
				case *ast.BinaryOperator:
					o.Expr2 = operand
				}
				// Set the end for all the operators in the path from p onwards.
				for i := p; i < len(path); i++ {
					switch o := path[i].(type) {
					case *ast.UnaryOperator:
						o.Position.End = operand.Pos().End
					case *ast.BinaryOperator:
						o.Position.End = operand.Pos().End
					}
				}
				// operator becomes the new leaf operator.
				op.Expr1 = path[p]
				op.Position.Start = path[p].Pos().Start
				path[p] = op
				path = path[0 : p+1]
			} else {
				// operator becomes the new leaf operator.
				op.Expr1 = operand
				op.Position.Start = operand.Pos().Start
				path = append(path, op)
			}

		}

	}

}

// addLastOperand adds the last operand to the expression parsing path and
// returns the operand resulting from the parsing of the entire expression.
func addLastOperand(op ast.Expression, path []ast.Operator) ast.Expression {
	if len(path) == 0 {
		return op
	}
	// Add the operand as a child of the leaf operator.
	switch leaf := path[len(path)-1].(type) {
	case *ast.UnaryOperator:
		leaf.Expr = op
	case *ast.BinaryOperator:
		leaf.Expr2 = op
	}
	// Set the end for all the operators in path.
	end := op.Pos().End
	for _, op := range path {
		switch o := op.(type) {
		case *ast.UnaryOperator:
			o.Position.End = end
		case *ast.BinaryOperator:
			o.Position.End = end
		}
	}
	// The operand is the the root of the expression tree.
	return path[0]
}

// parseExprList parses a list of expressions separated by a comma and returns
// the list and the last token read that can not be part of the last
// expression. It panics on error.
//
// tok is the first token of the expression, allowSwitchGuard reports whether
// a parsed expression can contain a type switch guard. allMustBeTypes report
// whatever all the expressions must be types. nextIsBlockBrace report whether
// a left brace block is expected after the expression.
func (p *parsing) parseExprList(tok token, allowSwitchGuard, allMustBeTypes, nextIsBlockBrace bool) ([]ast.Expression, token) {
	var element ast.Expression
	var elements []ast.Expression
	for {
		element, tok = p.parseExpr(tok, allowSwitchGuard, allMustBeTypes, nextIsBlockBrace)
		if element == nil {
			if elements != nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			return elements, tok
		}
		if elements == nil {
			elements = []ast.Expression{element}
		} else {
			elements = append(elements, element)
		}
		if tok.typ != tokenComma {
			return elements, tok
		}
		tok = p.next()
	}
}

// parseExprListInParenthesis parses a list of expressions as parseExprList
// does but allows a trailing comma if it is followed by a right parenthesis.
func (p *parsing) parseExprListInParenthesis(tok token) ([]ast.Expression, token) {
	var element ast.Expression
	var elements []ast.Expression
	for {
		element, tok = p.parseExpr(tok, false, false, false)
		if element == nil {
			if elements != nil && tok.typ != tokenRightParenthesis {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			return elements, tok
		}
		if elements == nil {
			elements = []ast.Expression{element}
		} else {
			elements = append(elements, element)
		}
		if tok.typ != tokenComma {
			return elements, tok
		}
		tok = p.next()
	}
}

// literalType returns a literal type from a token type.
func literalType(typ tokenTyp) ast.LiteralType {
	switch typ {
	case tokenRawString, tokenInterpretedString:
		return ast.StringLiteral
	case tokenRune:
		return ast.RuneLiteral
	case tokenInt:
		return ast.IntLiteral
	case tokenFloat:
		return ast.FloatLiteral
	case tokenImaginary:
		return ast.ImaginaryLiteral
	default:
		panic("invalid token type")
	}
}

// operatorFromTokenType returns a operator type from a token type. binary
// reports if the operator is used in a binary expression.
func operatorFromTokenType(typ tokenTyp, binary bool) ast.OperatorType {
	switch typ {
	case tokenEqual:
		return ast.OperatorEqual
	case tokenNotEqual:
		return ast.OperatorNotEqual
	case tokenNot:
		return ast.OperatorNot
	case tokenAmpersand:
		if binary {
			return ast.OperatorBitAnd
		}
		return ast.OperatorAddress
	case tokenVerticalBar:
		return ast.OperatorBitOr
	case tokenLess:
		return ast.OperatorLess
	case tokenLessOrEqual:
		return ast.OperatorLessEqual
	case tokenGreater:
		return ast.OperatorGreater
	case tokenGreaterOrEqual:
		return ast.OperatorGreaterEqual
	case tokenAnd:
		return ast.OperatorAnd
	case tokenOr:
		return ast.OperatorOr
	case tokenRelaxedAnd:
		return ast.OperatorRelaxedAnd
	case tokenRelaxedOr:
		return ast.OperatorRelaxedOr
	case tokenRelaxedNot:
		return ast.OperatorRelaxedNot
	case tokenAddition:
		return ast.OperatorAddition
	case tokenSubtraction:
		return ast.OperatorSubtraction
	case tokenMultiplication:
		if binary {
			return ast.OperatorMultiplication
		}
		return ast.OperatorPointer
	case tokenDivision:
		return ast.OperatorDivision
	case tokenModulo:
		return ast.OperatorModulo
	case tokenArrow:
		return ast.OperatorReceive
	case tokenXor:
		return ast.OperatorXor
	case tokenAndNot:
		return ast.OperatorAndNot
	case tokenLeftShift:
		return ast.OperatorLeftShift
	case tokenRightShift:
		return ast.OperatorRightShift
	default:
		panic("invalid token type")
	}
}

// parseIdentifierNode returns an Identifier node from a token.
func (p *parsing) parseIdentifierNode(tok token) *ast.Identifier {
	ident := ast.NewIdentifier(tok.pos, string(tok.txt))
	return ident
}

// unquoteString returns the characters in s unquoted as string.
func unquoteString(s []byte) string {
	if len(s) == 2 {
		return ""
	}
	if len(s) == 3 {
		return string(s[1])
	}
	if s[0] == '`' || bytes.IndexByte(s, '\\') == -1 {
		return string(s[1 : len(s)-1])
	}
	var cc = make([]byte, 0, len(s))
	for i := 1; i < len(s)-1; i++ {
		if s[i] == '\\' {
			r, n := parseEscapedRune(s[i:])
			if n == 3 || r < utf8.RuneSelf {
				cc = append(cc, byte(r))
			} else {
				p := [4]byte{}
				j := utf8.EncodeRune(p[:], r)
				cc = append(cc, p[:j]...)
			}
			i += n
		} else {
			cc = append(cc, s[i])
		}
	}
	return string(cc)
}

// parseEscapedRune parses an escaped rune sequence starting with '\\' and
// returns the rune and the length of the parsed sequence.
func parseEscapedRune(s []byte) (rune, int) {
	switch s[1] {
	case 'a':
		return '\a', 1
	case 'b':
		return '\b', 1
	case 'f':
		return '\f', 1
	case 'n':
		return '\n', 1
	case 'r':
		return '\r', 1
	case 't':
		return '\t', 1
	case 'v':
		return '\v', 1
	case '\\':
		return '\\', 1
	case '\'':
		return '\'', 1
	case '"':
		return '"', 1
	case 'x', 'u', 'U':
		var n = 2
		switch s[1] {
		case 'u':
			n = 4
		case 'U':
			n = 8
		}
		var r rune
		for j := 2; j < n+2; j++ {
			r = r * 16
			c := s[j]
			switch {
			case '0' <= c && c <= '9':
				r += rune(c - '0')
			case 'a' <= c && c <= 'f':
				r += rune(c - 'a' + 10)
			case 'A' <= c && c <= 'F':
				r += rune(c - 'A' + 10)
			}
		}
		return r, n + 1
	case '0', '1', '2', '3', '4', '5', '6', '7':
		r := (s[1]-'0')*64 + (s[2]-'0')*8 + (s[3] - '0')
		return rune(r), 3
	}
	panic("unexpected escaped rune")
}
