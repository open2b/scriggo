//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package parser

import (
	"bytes"
	"fmt"
	"strconv"
	"unicode/utf8"

	"open2b/template/ast"

	"github.com/shopspring/decimal"
)

// The result of parsing an expression is a tree whose intermediate
// nodes are operators and leaves are operands. For example:
//
//                op1
//              /     \
//            op2     op3
//           /   \     |
//          a     b    c
//
// During the construction of the tree, the rightmost leaf, instead of
// being an operand, is an operator:
//
//                op1
//              /     \
//            op2     op3
//           /   \
//          a     b
//
// The operation of adding an operator to the tree is called grafting and
// always takes place along the path, called simply path, which starts from
// the root and arrives at the leaf operator. In the previous example path
// is [op1, op3].
//
// A unary operator is grafted as a child of the leaf operator:
//
//                op1
//              /     \
//            op2     op3     path = [ op1, op3, op4 ]
//           /   \     |
//          a     b   op4
//
// To determine where to grab a non-unary operator, you run path from the
// leaf operator to the root, stopping when it is either an operator with
// lower precedence or reaching the root. For example, grafting op5 with
// precedence greater than op1 and less than or equal to op3 and op4 obtain
// the tree:
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
// If op5 had precedence less than or equal to op1:
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
// If op5 had a precedence greater than op4:
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

// parseExpr parses an expression and returns the expression tree and
// the last read token that is not a part.
func parseExpr(lex *lexer) (ast.Expression, token, error) {

	var err error

	// A tree of an expression has, as intermediate nodes, the operators,
	// unary or binary, and as leaves the operands.
	//
	// One of the leaves, during the building of the tree, instead of
	// an operand will be an operator. This leaf operator will miss his
	// unique expression if unary or the right-hand expression if binary.
	// Once the building is complete, all the leaves will be operands.

	// path is the path in the tree from the root operator to the leaf
	// operator.
	var path []ast.Operator

	// At each cycle of the expression tree building, either an unary
	// operator or a pair operand and binary operator is read. Finally an
	// operand will be read.
	//
	// For example the expression "-a * ( b + c ) < d || !e" is read as
	// "-", "a *", "b *", "c ()", "<", "d ||", "!", "e".
	//

	for {

		tok, ok := <-lex.tokens
		if !ok {
			return nil, token{}, lex.err
		}

		var operand ast.Expression
		var operator ast.Operator

		switch tok.typ {
		case tokenLeftParenthesis: // ( e )
			// Calls parseExpr recursively to parse the expression in
			// parenthesis and then treats it as a single operand.
			// The parenthesis will be omitted from the expression tree.
			pos := tok.pos
			var expr ast.Expression
			expr, tok, err = parseExpr(lex)
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
		case
			tokenAddition,    // +e
			tokenSubtraction, // -e
			tokenNot:         // !e
			operator = ast.NewUnaryOperator(tok.pos, operatorType(tok.typ), nil)
		case
			tokenNumber: // 12.895
			// If the number is preceded by the unary operator "-" it changes
			// the sign of the number and removes the operator from the tree.
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
			operand = parseIdentifierNode(tok)
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
					arg, tok, err = parseExpr(lex)
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
				index, tok, err = parseExpr(lex)
				if err != nil {
					return nil, token{}, err
				}
				if tok.typ == tokenColon {
					low := index
					var high ast.Expression
					high, tok, err = parseExpr(lex)
					if err != nil {
						return nil, token{}, err
					}
					if tok.typ != tokenRightBrackets {
						return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ]", tok)}
					}
					pos.End = tok.pos.End
					operand = ast.NewSlice(pos, operand, low, high)
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
				if tok.typ != tokenIdentifier {
					return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting name", tok)}
				}
				ident := string(tok.txt)
				pos.End = tok.pos.End
				operand = ast.NewSelector(pos, operand, ident)
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
				if tok.typ == tokenSemicolon {
					// Skips ";" and read the next token.
					tok, ok = <-lex.tokens
					if !ok {
						return nil, token{}, lex.err
					}
				}
				if len(path) == 0 {
					// It is possible to return directly.
					return operand, tok, nil
				}
				// Adds the operand as a child of the leaf operator.
				switch leef := path[len(path)-1].(type) {
				case *ast.UnaryOperator:
					leef.Expr = operand
				case *ast.BinaryOperator:
					leef.Expr2 = operand
				}
				// Sets End for all the operators in path.
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

		// Adds operator to the expression tree.

		switch op := operator.(type) {

		case *ast.UnaryOperator:
			// Unary operators ("!", "+", "-", "()"), as they have higher
			// precedence than all other operators, become the new leaf
			// operator of the tree.

			if len(path) > 0 {
				// Operator becomes a child of the leaf operator.
				switch leef := path[len(path)-1].(type) {
				case *ast.UnaryOperator:
					leef.Expr = op
				case *ast.BinaryOperator:
					leef.Expr2 = op
				}
			}
			// Operator becomes the new leaf operator.
			path = append(path, op)

		case *ast.BinaryOperator:
			// For binary operators ("*", "/", "+", "-", "<", ">", ...) it starts
			// from the leaf operator (last operator of path) and climb towards the
			// root (first path operator) stopping when either an operator with
			// lower precedence is found or the root is reached.

			// Sets start for all unary operators at the end of path.
			start := operand.Pos().Start
			for i := len(path) - 1; i >= 0; i-- {
				if o, ok := path[i].(*ast.UnaryOperator); ok {
					o.Position.Start = start
				} else {
					break
				}
			}

			// p will be the position in path in which to add operator.
			var p = len(path)
			for p > 0 && op.Precedence() <= path[p-1].Precedence() {
				p--
			}
			if p > 0 {
				// operator becomes the child of the operator with minor
				// precedence found going up path.
				switch o := path[p-1].(type) {
				case *ast.UnaryOperator:
					o.Expr = op
				case *ast.BinaryOperator:
					o.Expr2 = op
				}
			}
			if p < len(path) {
				// operand becomes the child of the operator in path
				// currently in position p.
				switch o := path[p].(type) {
				case *ast.UnaryOperator:
					o.Expr = operand
				case *ast.BinaryOperator:
					o.Expr2 = operand
				}
				// Sets End for all operators in the path from p onwards.
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

func parseIdentifierNode(tok token) *ast.Identifier {
	return ast.NewIdentifier(tok.pos, string(tok.txt))
}

// parseNumberNode returns an Expression node from an integer or decimal token,
// possibly preceded by a unary operator "-" with position neg.
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

func parseStringNode(tok token) *ast.String {
	return ast.NewString(tok.pos, unquoteString(tok.txt))
}

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
