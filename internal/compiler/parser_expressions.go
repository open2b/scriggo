// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"fmt"
	"unicode/utf8"

	"scriggo/internal/compiler/ast"
)

// The result of the parsing of an expression is a tree whose intermediate
// nodes are operators and the leaves are operands. For example:
//
//                op1
//              /     \
//            op2     op3
//           /   \     |
//          a     b    c
//
// While the tree is being built, the rightmost leaf is not an operand but an
// operator:
//
//                op1
//              /     \
//            op2     op3
//           /   \
//          a     b
//
// Adding an operator to the tree is called grafting and it takes place along
// the path that starts from the root and ends at the leaf operator. In the
// previous example path is [op1, op3].
//
// A unary operator is grafted as a child of the leaf operator:
//
//                op1
//              /     \
//            op2     op3     path = [ op1, op3, op4 ]
//           /   \     |
//          a     b   op4
//
// To identify where to graft a non-unary operator in the tree, starts from
// the leaf operator and goes up to the root, stopping if an operator with a
// lower precedence is found.
//
// If op5 has an higher precedence than op1 and lower or equal precedence than
// op3 and op4, the tree will be:
//
//                op1
//              /     \
//            op2     op5     path = [ op1, op5 ]
//           /   \   /
//          a     b op3
//                   |
//                  op4
//                   |
//                   c
//
// If op5 has a lower or equal precedence then op1:
//
//                    op5
//                  /
//                op1
//              /     \
//            op2     op3     path = [ op5 ]
//           /   \     |
//          a     b   op4
//                     |
//                     c
//
// If op5 has an higher precedence than op4:
//
//                op1
//              /     \
//            op2     op3     path = [ op1, op3, op4, op5 ]
//           /   \     |
//          a     b   op4
//                     |
//                    op5
//                   /
//                  c
//

// parseExpr parses an expression and returns its tree and the last read token
// that does not belong to the expression. tok, if initialized, is the first
// token of the expression. canBeBlank reports whether the parsed expression
// can be the blank identifier. canBeSwitchGuard reports whether the parsed
// expression can be a type switch guard, as x.(type). It panics on error.
//
// TODO (Gianluca): update doc. including type parsing.
//
// TODO (Gianluca): use allMustBeTypes in switch cases properly
//
// TODO (Gianluca): mustBeType ritorna quando ha finito di parsare un tipo.
// Devono però essere aggiunti i controlli che verifichino che effettivamente
// ciò che è stato parsato sia un tipo.
//
// TODO (Gianluca): nextIsBlockOpen should have a better name.
//
func (p *parsing) parseExpr(tok token, canBeBlank, canBeSwitchGuard, mustBeType, nextIsBlockOpen bool) (ast.Expression, token) {

	// TODO (Gianluca): to doc.
	reuseLastToken := false

	// canCompositeLiteral reports whether the currently parsed expression can
	// be used as type in composite literals. For instance, "interface{}" is a
	// type but cannot be used as type in composite literals, so
	// canCompositeLiteral is false.
	canCompositeLiteral := false

	// Intermediate nodes of an expression tree are unary or binary operators
	// and the leaves are operands.
	//
	// Only during the building of the tree, one of the leaves is an operator
	// and its expression is missing (right expression if the operator is
	// binary).
	//
	// When the tree building is complete, all the leaves are operands.

	// path is the tree path that starts from the root operator and ends with
	// the leaf operator.
	var path []ast.Operator

	// In each iteration of the expression tree building, either an unary
	// operator or a pair operand and binary operator is read. The last to be
	// read is an operator.
	//
	// For example, the expression "-a * ( b + c ) < d || !e" is read as
	// "-", "a *", "b *", "c ()", "<", "d ||", "!", "e".
	//

	// mustBeBlank reports whether the parsed expression must be the blank
	// identifier. It must be if it can be the blank identifier and the
	// first token is "_".
	var mustBeBlank bool

	// TODO (Gianluca): document.
	var mustBeSwitchGuard bool

	if tok.txt == nil {
		tok = next(p.lex)
	}

	for {

		var operand ast.Expression
		var operator ast.Operator

		switch tok.typ {
		case tokenLeftParenthesis: // ( e )
			// Calls parseExpr recursively to parse the expression in
			// parenthesis and then handles it as a single operand.
			// The parenthesis will be omitted from the expression tree.
			pos := tok.pos
			var expr ast.Expression
			expr, tok = p.parseExpr(token{}, false, false, mustBeType, false)
			if expr == nil {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
			}
			if tok.typ != tokenRightParenthesis {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting )", tok)})
			}
			operand = expr
			operand.Pos().Start = pos.Start
			operand.Pos().End = tok.pos.End
		case tokenLeftBraces: // composite literal with no type.
			reuseLastToken = true
		case tokenMap:
			canCompositeLiteral = true
			mapType := ast.NewMapType(tok.pos, nil, nil)
			tok = next(p.lex)
			if tok.typ != tokenLeftBrackets {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting [", tok)})
			}
			var typ ast.Expression
			typ, tok = p.parseExpr(token{}, false, false, true, false)
			if tok.typ != tokenRightBrackets {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %s", tok, tokenRightBraces)})
			}
			mapType.KeyType = typ
			typ, tok = p.parseExpr(token{}, false, false, true, false)
			if typ == nil {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting type", tok)})
			}
			mapType.Position.End = typ.Pos().End
			mapType.ValueType = typ
			operand = mapType
			reuseLastToken = true
		case tokenStruct:
			canCompositeLiteral = true
			structType := ast.NewStructType(tok.pos, nil)
			tok = next(p.lex)
			if tok.typ != tokenLeftBraces {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting {", tok)})
			}
			tok = next(p.lex)
			if tok.typ != tokenRightBraces {
				for {
					fieldDecl := ast.NewFieldDecl(nil, nil, nil)
					var exprs []ast.Expression
					var typ ast.Expression
					exprs, tok = p.parseExprList(tok, true, false, true, false)
					if tok.typ == tokenSemicolon {
						// Implicit field declaration.
						if len(exprs) != 1 {
							panic(&SyntaxError{"", *tok.pos, fmt.Errorf("expecting typooo")})
						}
						fieldDecl.Type = exprs[0]
						tok = next(p.lex)
					} else {
						// Explicit field declaration.
						typ, tok = p.parseExpr(tok, false, false, true, false)
						if tok.typ != tokenSemicolon {
							panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting semicolon", tok)})
						}
						tok = next(p.lex)
						fieldDecl.IdentifierList = make([]*ast.Identifier, len(exprs))
						for i, e := range exprs {
							ident, ok := e.(*ast.Identifier)
							if !ok {
								panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting field name or embedded type ", e)})
							}
							fieldDecl.IdentifierList[i] = ident
						}
						fieldDecl.Type = typ
					}
					structType.FieldDecl = append(structType.FieldDecl, fieldDecl)
					if tok.typ == tokenRightBraces {
						break
					}
				}
			}
			structType.Position.End = tok.pos.End
			operand = structType
		case tokenInterface:
			pos := tok.pos
			tok = next(p.lex)
			if tok.typ != tokenLeftBraces {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting {", tok)})
			}
			tok = next(p.lex)
			if tok.typ != tokenRightBraces {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting }", tok)})
			}
			pos.End = tok.pos.End
			operand = ast.NewIdentifier(pos, "interface{}")
		case tokenFunc:
			var node ast.Node
			if mustBeType {
				node, tok = p.parseFunc(tok, parseFuncType)
				reuseLastToken = true
			} else {
				node, tok = p.parseFunc(tok, parseFuncLit)
			}
			operand = node.(ast.Expression)
		case
			tokenArrow, // <-, <-chan
			tokenChan:  // chan, chan<-
			pos := tok.pos
			direction := ast.NoDirection
			if tok.typ == tokenArrow {
				tok = next(p.lex)
				if tok.typ == tokenChan {
					direction = ast.ReceiveDirection
				} else {
					operator = ast.NewUnaryOperator(pos, ast.OperatorReceive, nil)
					reuseLastToken = true
				}
			}
			if operator == nil {
				tok = next(p.lex)
				if direction == ast.NoDirection && tok.typ == tokenArrow {
					direction = ast.SendDirection
					tok = next(p.lex)
				}
				var elemType ast.Expression
				elemType, tok = p.parseExpr(tok, false, false, true, false)
				if elemType == nil {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("missing channel element type")})
				}
				reuseLastToken = true
				pos.End = elemType.Pos().End
				operand = ast.NewChanType(pos, direction, elemType)
			}
		case
			tokenAddition,       // +e
			tokenSubtraction,    // -e
			tokenNot,            // !e
			tokenXor,            // ^e
			tokenMultiplication, // *t, *T
			tokenAnd:            // &e
			operator = ast.NewUnaryOperator(tok.pos, operatorType(tok.typ), nil)
		case
			tokenRune,      // '\x3c'
			tokenInt,       // 18
			tokenFloat,     // 12.895
			tokenImaginary: // 7.2i
			operand = ast.NewBasicLiteral(tok.pos, literalType(tok.typ), string(tok.txt))
		case
			tokenInterpretedString, // ""
			tokenRawString:         // ``
			if mustBeType {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected literal %q, expecting type", tok.txt)})
			}
			operand = ast.NewBasicLiteral(tok.pos, literalType(tok.typ), string(tok.txt))
		case tokenIdentifier: // a
			ident := p.parseIdentifierNode(tok)
			// TODO (Gianluca): this check must be done during type-checking.
			// if ident.Name == "_" {
			// 	if !canBeBlank {
			// 		panic(&Error{"", *tok.pos, fmt.Errorf("cannot use _ as value")})
			// 	}
			// 	mustBeBlank = true
			// }
			operand = ident
			if mustBeType {
				tok = next(p.lex)
				if tok.typ == tokenPeriod {
					tok = next(p.lex)
					if tok.typ != tokenIdentifier {
						panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting name", tok.txt)})
					}
					ident := p.parseIdentifierNode(tok)
					operand = ast.NewSelector(tok.pos, operand, ident.Name)
				} else {
					reuseLastToken = true
				}
			}
		case tokenLeftBrackets: // [
			canCompositeLiteral = true
			var expr, length ast.Expression
			pos := tok.pos
			isEllipses := false
			tok = next(p.lex)
			switch tok.typ {
			case tokenEllipses:
				isEllipses = true
				tok = next(p.lex)
			case tokenRightBrackets:
			default:
				oldTok := tok
				expr, tok = p.parseExpr(tok, false, false, false, false)
				if expr == nil {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", oldTok)})
				}
				length = expr
			}
			if tok.typ != tokenRightBrackets {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ]", tok)})
			}
			var typ ast.Expression
			typ, tok = p.parseExpr(token{}, false, false, true, false)
			if typ == nil {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
			}
			pos.End = typ.Pos().End
			switch {
			case isEllipses:
				operand = ast.NewArrayType(pos, nil, typ)
			case length == nil:
				operand = ast.NewSliceType(pos, typ)
			default:
				operand = ast.NewArrayType(pos, length, typ)
			}
			reuseLastToken = true
		default:
			return nil, tok
		}

		if mustBeType {
			if operator != nil {
				op, ok := operator.(*ast.UnaryOperator)
				if !ok || op.Operator() != ast.OperatorMultiplication {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting type", operator)})
				}
			}
			if operand != nil {
				switch operand.(type) {
				case *ast.Identifier, *ast.MapType, *ast.ArrayType, *ast.SliceType,
					*ast.ChanType, *ast.Selector, *ast.FuncType, *ast.StructType:
				default:
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting type", operand)})
				}
			}
		}

		canBeBlank = false

		for operator == nil {

			if !reuseLastToken {
				tok = next(p.lex)
			}
			reuseLastToken = false

			skip := mustBeType || (tok.typ == tokenLeftBraces && nextIsBlockOpen && !canCompositeLiteral)
			backupTok := tok
			if skip {
				tok = token{}
			}

			switch tok.typ {

			case tokenLeftBraces: // ...{
				canCompositeLiteral = false
				pos := &ast.Position{Line: tok.pos.Line, Column: tok.pos.Column, Start: tok.pos.Start, End: tok.pos.End}
				if operand != nil {
					pos.Start = operand.Pos().Start
				}
				var keyValues []ast.KeyValue
				var expr ast.Expression
				for {
					expr, tok = p.parseExpr(token{}, false, false, false, false)
					if expr == nil {
						break
					}
					switch tok.typ {
					case tokenColon:
						var value ast.Expression
						value, tok = p.parseExpr(token{}, false, false, false, false)
						if value == nil {
							panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
						}
						keyValues = append(keyValues, ast.KeyValue{Key: expr, Value: value})
					case tokenComma, tokenSemicolon, tokenRightBraces:
						keyValues = append(keyValues, ast.KeyValue{Key: nil, Value: expr})
					default:
						panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expecting comma or }", tok)})
					}
					if tok.typ == tokenRightBraces {
						break
					}
				}
				if tok.typ != tokenRightBraces {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or }", tok)})
				}
				pos.End = tok.pos.End
				operand = ast.NewCompositeLiteral(pos, operand, keyValues)
			case tokenLeftParenthesis: // e(...)
				pos := tok.pos
				pos.Start = operand.Pos().Start
				args, tok := p.parseExprList(token{}, false, false, false, false)
				var isVariadic bool
				if tok.typ == tokenEllipses {
					if len(args) == 0 {
						panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected ..., expecting expression")})
					}
					isVariadic = true
					tok = next(p.lex)
				}
				if tok.typ != tokenRightParenthesis {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or )", tok)})
				}
				pos.End = tok.pos.End
				operand = ast.NewCall(pos, operand, args, isVariadic)
				canCompositeLiteral = false
			case tokenLeftBrackets: // e[...], e[.. : ..], e[.. : .. : ..],
				pos := tok.pos
				pos.Start = operand.Pos().Start
				var index ast.Expression
				index, tok = p.parseExpr(token{}, false, false, false, false)
				if tok.typ == tokenColon {
					low := index
					isFull := false
					var high, max ast.Expression
					high, tok = p.parseExpr(token{}, false, false, false, false)
					if tok.typ == tokenColon {
						isFull = true
						max, tok = p.parseExpr(token{}, false, false, false, false)
					}
					if tok.typ != tokenRightBrackets {
						panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ]", tok)})
					}
					pos.End = tok.pos.End
					operand = ast.NewSlicing(pos, operand, low, high, max, isFull)
				} else {
					if tok.typ != tokenRightBrackets {
						panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ]", tok)})
					}
					if index == nil {
						panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected ], expecting expression")})
					}
					pos.End = tok.pos.End
					operand = ast.NewIndex(pos, operand, index)
				}
			case tokenPeriod: // e.
				pos := tok.pos
				pos.Start = operand.Pos().Start
				tok = next(p.lex)
				switch tok.typ {
				case tokenIdentifier:
					// e.ident
					ident := string(tok.txt)
					if ident == "_" {
						panic(&SyntaxError{"", *tok.pos, fmt.Errorf("cannot refer to blank field")})
					}
					pos.End = tok.pos.End
					operand = ast.NewSelector(pos, operand, ident)
				case tokenLeftParenthesis:
					// e.(ident), e.(pkg.ident)
					tok = next(p.lex)
					if len(tok.txt) == 1 && tok.txt[0] == '_' {
						panic(&SyntaxError{"", *tok.pos, fmt.Errorf("cannot use _ as value")})
					}
					var typ ast.Expression
					switch tok.typ {
					case tokenType:
						if !canBeSwitchGuard {
							panic(&SyntaxError{"", *tok.pos, fmt.Errorf("use of .(type) outside type switch")})
						}
						mustBeSwitchGuard = true
						tok = next(p.lex)
					case tokenIdentifier:
						if len(tok.txt) == 1 && tok.txt[0] == '_' {
							panic(&SyntaxError{"", *tok.pos, fmt.Errorf("cannot use _ as value")})
						}
						fallthrough
					default:
						typ, tok = p.parseExpr(tok, false, true, true, false)
					}
					if tok.typ != tokenRightParenthesis {
						panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting )", tok)})
					}
					pos.End = tok.pos.End
					operand = ast.NewTypeAssertion(pos, operand, typ)
				default:
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting name or (", tok)})
				}
			case
				tokenEqual,          // e ==
				tokenNotEqual,       // e !=
				tokenLess,           // e <
				tokenLessOrEqual,    // e <=
				tokenGreater,        // e >
				tokenGreaterOrEqual, // e >=
				tokenAndAnd,         // e &&
				tokenOrOr,           // e ||
				tokenAddition,       // e +
				tokenSubtraction,    // e -
				tokenMultiplication, // e *
				tokenDivision,       // e /
				tokenModulo,         // e %
				tokenAnd,            // e &
				tokenOr,             // e |
				tokenXor,            // e ^
				tokenAndNot,         // e &^
				tokenLeftShift,      // e <<
				tokenRightShift:     // e >>
				operator = ast.NewBinaryOperator(tok.pos, operatorType(tok.typ), nil, nil)
			default:
				if len(path) > 0 {
					// Adds the operand as a child of the leaf operator.
					switch leaf := path[len(path)-1].(type) {
					case *ast.UnaryOperator:
						leaf.Expr = operand
					case *ast.BinaryOperator:
						leaf.Expr2 = operand
					}
					// Sets the end for all the operators in path.
					end := operand.Pos().End
					for _, op := range path {
						switch o := op.(type) {
						case *ast.UnaryOperator:
							o.Position.End = end
						case *ast.BinaryOperator:
							o.Position.End = end
						}
					}
					// The operand is the the root of the expression tree.
					operand = path[0]
				}
				if mustBeBlank {
					if ident, ok := operand.(*ast.Identifier); !ok || ident.Name != "_" {
						panic(&SyntaxError{"", *ident.Position, fmt.Errorf("cannot use _ as value")})
					}
				}
				if mustBeSwitchGuard && !isTypeGuard(operand) {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("use of .(type) outside type switch")})
				}
				if skip {
					tok = backupTok
				}
				return operand, tok
			}

		}

		canBeSwitchGuard = false

		// Adds the operator to the expression tree.

		switch op := operator.(type) {

		case *ast.UnaryOperator:
			// An unary operator ("!", "+", "-", "()") becomes the new leaf
			// operator as it has an higher precedence than all the other
			// operators.

			if len(path) > 0 {
				// Operator becomes a child of the leaf operator.
				switch leaf := path[len(path)-1].(type) {
				case *ast.UnaryOperator:
					leaf.Expr = op
				case *ast.BinaryOperator:
					leaf.Expr2 = op
				}
			}
			// Operator becomes the new leaf operator.
			path = append(path, op)

		case *ast.BinaryOperator:
			// For a binary operator ("*", "/", "+", "-", "<", ">", ...),
			// starts from the leaf operator (last operator of the path) and
			// goes up to the root (first operator of the path) stopping if an
			// operator with lower precedence is found.

			// For all unary operators, sets the start at the end of the path.
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
				// Sets the end for all the operators in the path from p onwards.
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

		if !reuseLastToken {
			tok = next(p.lex)
		}
		reuseLastToken = false

	}

}

// parseExprList parses a list of expressions separated by a comma and returns
// the list and the last token read that can not be part of the last expression.
// tok, if initialized, is the first token of the expression. canBeBlank
// reports whether a parsed expression can be the blank identifier.
// allowSwitchGuard reports whether a parsed expression can contain a type
// switch guard. It panics on error.
//
// TODO (Gianluca): nextIsBlockOpen should have a better name
//
func (p *parsing) parseExprList(tok token, allowBlank, allowSwitchGuard, allMustBeTypes, nextIsBlockOpen bool) ([]ast.Expression, token) {
	var elements = []ast.Expression{}
	for {
		element, tok2 := p.parseExpr(tok, allowBlank, allowSwitchGuard, allMustBeTypes, nextIsBlockOpen)
		if element == nil {
			return elements, tok2
		}
		elements = append(elements, element)
		if tok2.typ != tokenComma {
			return elements, tok2
		}
		tok = token{}
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

// operatorType returns a operator type from a token type.
func operatorType(typ tokenTyp) ast.OperatorType {
	switch typ {
	case tokenEqual:
		return ast.OperatorEqual
	case tokenNotEqual:
		return ast.OperatorNotEqual
	case tokenNot:
		return ast.OperatorNot
	case tokenAnd:
		return ast.OperatorAnd
	case tokenOr:
		return ast.OperatorOr
	case tokenLess:
		return ast.OperatorLess
	case tokenLessOrEqual:
		return ast.OperatorLessOrEqual
	case tokenGreater:
		return ast.OperatorGreater
	case tokenGreaterOrEqual:
		return ast.OperatorGreaterOrEqual
	case tokenAndAnd:
		return ast.OperatorAndAnd
	case tokenOrOr:
		return ast.OperatorOrOr
	case tokenAddition:
		return ast.OperatorAddition
	case tokenSubtraction:
		return ast.OperatorSubtraction
	case tokenMultiplication:
		return ast.OperatorMultiplication
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
			if r < utf8.RuneSelf {
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
