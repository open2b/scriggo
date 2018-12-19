// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"bytes"
	"fmt"
	"strconv"
	"unicode/utf8"

	"open2b/template/ast"

	"github.com/shopspring/decimal"
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
// token of the expression.
func parseExpr(tok token, lex *lexer) (ast.Expression, token, error) {

	var err error

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

	var ok bool

	if tok.txt == nil {
		tok, ok = <-lex.tokens
		if !ok {
			return nil, token{}, lex.err
		}
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
			expr, tok, err = parseExpr(token{}, lex)
			if err != nil {
				return nil, token{}, err
			}
			if expr == nil {
				return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)}
			}
			if tok.typ != tokenRightParenthesis {
				return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting )", tok)}
			}
			operand = expr
			operand.Pos().Start = pos.Start
			operand.Pos().End = tok.pos.End
		case tokenMap: // map{...}
			pos := tok.pos
			tok, ok = <-lex.tokens
			if !ok {
				return nil, token{}, lex.err
			}
			if tok.typ != tokenLeftBraces {
				return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting {", tok)}
			}
			elements := []ast.KeyValue{}
			for {
				var element ast.KeyValue
				element.Key, tok, err = parseExpr(token{}, lex)
				if err != nil {
					return nil, token{}, err
				}
				if element.Key == nil {
					if tok.typ != tokenRightBraces {
						return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or }", tok)}
					}
				} else {
					if tok.typ != tokenColon {
						return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("missing key in map literal")}
					}
					element.Value, tok, err = parseExpr(token{}, lex)
					if err != nil {
						return nil, token{}, err
					}
					if element.Value == nil {
						return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)}
					}
					elements = append(elements, element)
				}
				if tok.typ == tokenRightBraces {
					pos.End = tok.pos.End
					break
				}
			}
			if len(elements) > 1 {
				duplicates := map[string]struct{}{}
				for _, element := range elements {
					if key, ok := element.Key.(*ast.String); ok {
						if _, ok := duplicates[key.Text]; ok {
							return nil, token{}, &Error{"", *(key.Pos()), fmt.Errorf("duplicate key %q in map literal", key.Text)}
						}
						duplicates[key.Text] = struct{}{}
					}
				}
			}
			operand = ast.NewMap(pos, elements)
		case tokenLeftBraces: // {...}
			pos := tok.pos
			var elements = []ast.Expression{}
			for {
				var element ast.Expression
				element, tok, err = parseExpr(token{}, lex)
				if err != nil {
					return nil, token{}, err
				}
				if element == nil {
					if tok.typ != tokenRightBraces {
						return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or }", tok)}
					}
				} else {
					if tok.typ != tokenComma && tok.typ != tokenRightBraces {
						return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting comma or }", tok)}
					}
					elements = append(elements, element)
				}
				if tok.typ == tokenRightBraces {
					pos.End = tok.pos.End
					break
				}
			}
			operand = ast.NewSlice(pos, elements)
		case
			tokenAddition,    // +e
			tokenSubtraction, // -e
			tokenNot:         // !e
			operator = ast.NewUnaryOperator(tok.pos, operatorType(tok.typ), nil)
		case
			tokenNumber: // 12.895
			// If the number is preceded by the unary operator "-", the sign
			// of the number is changed and the operator is removed from the
			// tree.
			var pos *ast.Position
			if len(path) > 0 {
				var op *ast.UnaryOperator
				if op, ok = path[len(path)-1].(*ast.UnaryOperator); ok && op.Op == ast.OperatorSubtraction {
					pos = op.Pos()
					path = path[:len(path)-1]
				}
			}
			operand, err = parseNumberNode(tok, pos)
			if err != nil {
				return nil, token{}, err
			}
		case
			tokenInterpretedString, // ""
			tokenRawString:         // ``
			operand = parseStringNode(tok)
		case tokenIdentifier: // a
			ident := parseIdentifierNode(tok)
			if ident.Name == "_" {
				return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("cannot use _ as value")}
			}
			operand = ident
		case tokenSemicolon:
			return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected semicolon or newline, expecting expression")}
		default:
			return nil, tok, nil
		}

		for operator == nil {

			tok, ok = <-lex.tokens
			if !ok {
				return nil, token{}, lex.err
			}

			switch tok.typ {
			case tokenLeftParenthesis: // e(...)
				pos := tok.pos
				pos.Start = operand.Pos().Start
				var args = []ast.Expression{}
				for {
					var arg ast.Expression
					arg, tok, err = parseExpr(token{}, lex)
					if err != nil {
						return nil, token{}, err
					}
					if arg == nil {
						if tok.typ != tokenRightParenthesis {
							return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or )", tok)}
						}
					} else {
						if tok.typ != tokenComma && tok.typ != tokenRightParenthesis {
							return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting comma or )", tok)}
						}
						args = append(args, arg)
					}
					if tok.typ == tokenRightParenthesis {
						pos.End = tok.pos.End
						break
					}
				}
				operand = ast.NewCall(pos, operand, args)
			case tokenLeftBrackets: // e[...], e[.. : ..]
				pos := tok.pos
				pos.Start = operand.Pos().Start
				var index ast.Expression
				index, tok, err = parseExpr(token{}, lex)
				if err != nil {
					return nil, token{}, err
				}
				if tok.typ == tokenColon {
					low := index
					var high ast.Expression
					high, tok, err = parseExpr(token{}, lex)
					if err != nil {
						return nil, token{}, err
					}
					if tok.typ != tokenRightBrackets {
						return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ]", tok)}
					}
					pos.End = tok.pos.End
					operand = ast.NewSlicing(pos, operand, low, high)
				} else {
					if tok.typ != tokenRightBrackets {
						return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ]", tok)}
					}
					if index == nil {
						return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected ], expecting expression")}
					}
					pos.End = tok.pos.End
					operand = ast.NewIndex(pos, operand, index)
				}
			case tokenPeriod: // e.
				pos := tok.pos
				pos.Start = operand.Pos().Start
				tok, ok = <-lex.tokens
				if !ok {
					return nil, token{}, lex.err
				}
				switch tok.typ {
				case tokenIdentifier:
					// e.ident
					ident := string(tok.txt)
					pos.End = tok.pos.End
					operand = ast.NewSelector(pos, operand, ident)
				case tokenLeftParenthesis:
					// e.(type)
					tok, ok = <-lex.tokens
					if !ok {
						return nil, token{}, lex.err
					}
					if tok.typ != tokenIdentifier && tok.typ != tokenMap {
						return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting type", tok)}
					}
					typ := ast.NewIdentifier(tok.pos, string(tok.txt))
					tok, ok = <-lex.tokens
					if !ok {
						return nil, token{}, lex.err
					}
					if tok.typ != tokenRightParenthesis {
						return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting )", tok)}
					}
					pos.End = tok.pos.End
					operand = ast.NewTypeAssertion(pos, operand, typ)
				default:
					return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting name or (", tok)}
				}
			case
				tokenEqual,          // e ==
				tokenNotEqual,       // e !=
				tokenLess,           // e <
				tokenLessOrEqual,    // e <=
				tokenGreater,        // e >
				tokenGreaterOrEqual, // e >=
				tokenAnd,            // e &&
				tokenOr,             // e ||
				tokenAddition,       // e +
				tokenSubtraction,    // e -
				tokenMultiplication, // e *
				tokenDivision,       // e /
				tokenModulo:         // e %
				operator = ast.NewBinaryOperator(tok.pos, operatorType(tok.typ), nil, nil)
			case tokenEOF:
				return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected EOF, expecting expression")}
			default:
				if len(path) == 0 {
					// Returns the operand.
					return operand, tok, nil
				}
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
				// Returns the root of the expression tree.
				return path[0], tok, nil
			}

		}

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
				// operand becomes the child of the operator in the path at
				// position p.
				switch o := path[p].(type) {
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

		tok, ok = <-lex.tokens
		if !ok {
			return nil, token{}, lex.err
		}

	}

}

// operatorType returns a operator type from a token type.
func operatorType(typ tokenType) ast.OperatorType {
	switch typ {
	case tokenEqual:
		return ast.OperatorEqual
	case tokenNotEqual:
		return ast.OperatorNotEqual
	case tokenNot:
		return ast.OperatorNot
	case tokenLess:
		return ast.OperatorLess
	case tokenLessOrEqual:
		return ast.OperatorLessOrEqual
	case tokenGreater:
		return ast.OperatorGreater
	case tokenGreaterOrEqual:
		return ast.OperatorGreaterOrEqual
	case tokenAnd:
		return ast.OperatorAnd
	case tokenOr:
		return ast.OperatorOr
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
	default:
		panic("invalid token type")
	}
}

// parseIdentifierNode returns an Identifier node from a token.
func parseIdentifierNode(tok token) *ast.Identifier {
	return ast.NewIdentifier(tok.pos, string(tok.txt))
}

// parseNumberNode returns an Expression node from an integer or decimal token,
// possibly preceded by an unary operator "-" with neg position.
func parseNumberNode(tok token, neg *ast.Position) (ast.Expression, error) {
	p := tok.pos
	s := string(tok.txt)
	if neg != nil {
		p = neg
		p.End = tok.pos.End
		s = "-" + s
	}
	if bytes.IndexByte(tok.txt, '.') == -1 {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, &Error{"", *tok.pos, fmt.Errorf("constant %s overflows int", s)}
		}
		return ast.NewInt(p, n), nil
	}
	d, _ := decimal.NewFromString(s)
	return ast.NewNumber(p, d), nil
}

// parseStringNode returns a String node from a token.
func parseStringNode(tok token) *ast.String {
	return ast.NewString(tok.pos, unquoteString(tok.txt))
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
			switch s[i+1] {
			case 'a':
				cc = append(cc, '\a')
			case 'b':
				cc = append(cc, '\b')
			case 'f':
				cc = append(cc, '\f')
			case 'n':
				cc = append(cc, '\n')
			case 'r':
				cc = append(cc, '\r')
			case 't':
				cc = append(cc, '\t')
			case 'v':
				cc = append(cc, '\v')
			case '\\':
				cc = append(cc, '\\')
			case '\'':
				cc = append(cc, '\'')
			case '"':
				cc = append(cc, '"')
			case 'u', 'U':
				var n = 4
				if s[i+1] == 'U' {
					n = 8
				}
				var r uint32
				for j := 0; j < n; j++ {
					r = r * 16
					var c = s[i+j+2]
					switch {
					case '0' <= c && c <= '9':
						r += uint32(c - '0')
					case 'a' <= c && c <= 'f':
						r += uint32(c - 'a' + 10)
					case 'A' <= c && c <= 'F':
						r += uint32(c - 'A' + 10)
					}
				}
				p := [4]byte{}
				j := utf8.EncodeRune(p[:], rune(r))
				cc = append(cc, p[:j]...)
				i += n
			}
			i++
		} else {
			cc = append(cc, s[i])
		}
	}
	return string(cc)
}
