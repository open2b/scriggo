// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"bytes"
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/cockroachdb/apd"

	"open2b/template/ast"
)

var maxInt = apd.New(int64(^uint(0)>>1), 0)
var minInt = apd.New(-int64(^uint(0)>>1)-1, 0)

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
// token of the expression. canBeBlank indicates if the parsed expression can be
// the blank identifier. canBeSwitchGuard indicates if the parsed expression can
// be a type switch guard, as x.(type). It panics on error.
func parseExpr(tok token, lex *lexer, canBeBlank, canBeSwitchGuard bool) (ast.Expression, token) {

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

	// mustBeBlank indicates if the parsed expression must be the blank
	// identifier. It must be if it can be the blank identifier and the
	// first token is "_".
	var mustBeBlank bool

	// TODO (Gianluca): document.
	var mustBeSwitchGuard bool

	var ok bool

	if tok.txt == nil {
		tok = next(lex)
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
			expr, tok = parseExpr(token{}, lex, false, false)
			if expr == nil {
				panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
			}
			if tok.typ != tokenRightParenthesis {
				panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting )", tok)})
			}
			operand = expr
			operand.Pos().Start = pos.Start
			operand.Pos().End = tok.pos.End
		case tokenMap: // map{...}, map(...)
			typeTok := tok
			tok = next(lex)
			switch tok.typ {
			case tokenLeftBraces:
				// Map definition.
				elements := []ast.KeyValue{}
				for {
					var element ast.KeyValue
					element.Key, tok = parseExpr(token{}, lex, false, false)
					if element.Key == nil {
						if tok.typ != tokenRightBraces {
							panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or }", tok)})
						}
					} else {
						if tok.typ != tokenColon {
							panic(&Error{"", *tok.pos, fmt.Errorf("missing key in map literal")})
						}
						element.Value, tok = parseExpr(token{}, lex, false, false)
						if element.Value == nil {
							panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
						}
						elements = append(elements, element)
					}
					if tok.typ == tokenRightBraces {
						typeTok.pos.End = tok.pos.End
						break
					}
				}
				if len(elements) > 1 {
					duplicates := map[interface{}]struct{}{}
					for _, element := range elements {
						switch key := element.Key.(type) {
						case nil:
							if _, ok := duplicates[nil]; ok {
								panic(&Error{"", *(key.Pos()), fmt.Errorf("duplicate key nil in map literal")})
							}
							duplicates[nil] = struct{}{}
						case *ast.String:
							if _, ok := duplicates[key.Text]; ok {
								panic(&Error{"", *(key.Pos()), fmt.Errorf("duplicate key %q in map literal", key.Text)})
							}
							duplicates[key.Text] = struct{}{}
						case *ast.Number:
							if _, ok := duplicates[key.Value]; ok {
								panic(&Error{"", *(key.Pos()), fmt.Errorf("duplicate key %s in map literal", key.Value.String())})
							}
							duplicates[key.Value] = struct{}{}
						case *ast.Int:
							if _, ok := duplicates[key.Value]; ok {
								panic(&Error{"", *(key.Pos()), fmt.Errorf("duplicate key %d in map literal", key.Value)})
							}
							duplicates[key.Value] = struct{}{}
						}
					}
				}
				operand = ast.NewMap(typeTok.pos, elements)
			case tokenLeftParenthesis:
				// Map conversion.
				operand = parseConversionExpr(typeTok, lex)
			default:
				panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting { or (", tok)})
			}
		case tokenSlice: // slice{...}, slice(...)
			typeTok := tok
			tok = next(lex)
			switch tok.typ {
			case tokenLeftBraces:
				// Slice definition.
				elements, tok := parseExprList(token{}, lex, false, false)
				if tok.typ != tokenRightBraces {
					panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or }", tok)})
				}
				typeTok.pos.End = tok.pos.End
				operand = ast.NewSlice(typeTok.pos, elements)
			case tokenLeftParenthesis:
				// Slice conversion.
				operand = parseConversionExpr(typeTok, lex)
			default:
				panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting { or (", tok)})
			}
		case tokenBytes: // bytes{...}, bytes(...)
			typeTok := tok
			tok = next(lex)
			switch tok.typ {
			case tokenLeftBraces:
				// Bytes definition.
				elements, tok := parseExprList(token{}, lex, false, false)
				if tok.typ != tokenRightBraces {
					panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or }", tok)})
				}
				typeTok.pos.End = tok.pos.End
				operand = ast.NewBytes(typeTok.pos, elements)
			case tokenLeftParenthesis:
				// Bytes conversion.
				operand = parseConversionExpr(typeTok, lex)
			default:
				panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting { or (", tok)})
			}
		case
			tokenAddition,    // +e
			tokenSubtraction, // -e
			tokenNot:         // !e
			operator = ast.NewUnaryOperator(tok.pos, operatorType(tok.typ), nil)
		case
			tokenNumber,      // 12.895
			tokenRuneLiteral: // '\x3c'
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
			operand = parseNumberNode(tok, pos)
		case
			tokenInterpretedString, // ""
			tokenRawString:         // ``
			operand = parseStringNode(tok)
		case tokenIdentifier: // a
			ident := parseIdentifierNode(tok)
			if ident.Name == "_" {
				if !canBeBlank {
					panic(&Error{"", *tok.pos, fmt.Errorf("cannot use _ as value")})
				}
				mustBeBlank = true
			}
			operand = ident
		default:
			return nil, tok
		}

		canBeBlank = false

		for operator == nil {

			tok = next(lex)

			switch tok.typ {
			case tokenLeftParenthesis: // e(...)
				pos := tok.pos
				pos.Start = operand.Pos().Start
				args, tok := parseExprList(token{}, lex, false, false)
				if tok.typ != tokenRightParenthesis {
					panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or )", tok)})
				}
				pos.End = tok.pos.End
				operand = ast.NewCall(pos, operand, args)
			case tokenLeftBrackets: // e[...], e[.. : ..]
				pos := tok.pos
				pos.Start = operand.Pos().Start
				var index ast.Expression
				index, tok = parseExpr(token{}, lex, false, false)
				if tok.typ == tokenColon {
					low := index
					var high ast.Expression
					high, tok = parseExpr(token{}, lex, false, false)
					if tok.typ != tokenRightBrackets {
						panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ]", tok)})
					}
					pos.End = tok.pos.End
					operand = ast.NewSlicing(pos, operand, low, high)
				} else {
					if tok.typ != tokenRightBrackets {
						panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ]", tok)})
					}
					if index == nil {
						panic(&Error{"", *tok.pos, fmt.Errorf("unexpected ], expecting expression")})
					}
					pos.End = tok.pos.End
					operand = ast.NewIndex(pos, operand, index)
				}
			case tokenPeriod: // e.
				pos := tok.pos
				pos.Start = operand.Pos().Start
				tok = next(lex)
				switch tok.typ {
				case tokenIdentifier:
					// e.ident
					ident := string(tok.txt)
					if ident == "_" {
						panic(&Error{"", *tok.pos, fmt.Errorf("cannot refer to blank field")})
					}
					pos.End = tok.pos.End
					operand = ast.NewSelector(pos, operand, ident)
				case tokenLeftParenthesis:
					// e.(type)
					tok = next(lex)
					if tok.typ != tokenIdentifier && tok.typ != tokenMap && tok.typ != tokenSlice && tok.typ != tokenBytes && tok.typ != tokenSwitchType {
						panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting type", tok)})
					}
					if len(tok.txt) == 1 && tok.txt[0] == '_' {
						panic(&Error{"", *tok.pos, fmt.Errorf("cannot use _ as value")})
					}
					var typ *ast.Identifier
					switch tok.typ {
					case tokenSwitchType:
						if !canBeSwitchGuard {
							panic(&Error{"", *tok.pos, fmt.Errorf("use of .(type) outside type switch")})
						}
						mustBeSwitchGuard = true
					case tokenIdentifier:
						if len(tok.txt) == 1 && tok.txt[0] == '_' {
							panic(&Error{"", *tok.pos, fmt.Errorf("cannot use _ as value")})
						}
						fallthrough
					default:
						typ = ast.NewIdentifier(tok.pos, string(tok.txt))
					}
					tok = next(lex)
					if tok.typ != tokenRightParenthesis {
						panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting )", tok)})
					}
					pos.End = tok.pos.End
					operand = ast.NewTypeAssertion(pos, operand, typ)
				default:
					panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting name or (", tok)})
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
				panic(&Error{"", *tok.pos, fmt.Errorf("unexpected EOF, expecting expression")})
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
						panic(&Error{"", *ident.Position, fmt.Errorf("cannot use _ as value")})
					}
				}
				if mustBeSwitchGuard && !isTypeGuard(operand) {
					panic(&Error{"", *tok.pos, fmt.Errorf("use of .(type) outside type switch")})
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

		tok = next(lex)

	}

}

// parseExprList parses a list of expressions separated by a comma and returns
// the list and the last token read that can not be part of the last expression.
// tok, if initialized, is the first token of the expression. canBeBlank
// indicates if a parsed expression can be the blank identifier.
// allowSwitchGuard indicates if a parsed expression can containt a type switch
// guard. It panics on error.
func parseExprList(tok token, lex *lexer, allowBlank, allowSwitchGuard bool) ([]ast.Expression, token) {
	var elements = []ast.Expression{}
	for {
		element, tok2 := parseExpr(tok, lex, allowBlank, allowSwitchGuard)
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

// exprListString returns elements as its string representation.
func exprListString(elements []ast.Expression) string {
	s := ""
	for i, element := range elements {
		if i > 0 {
			s += ", "
		}
		s += element.String()
	}
	return s
}

// parseConversionExpr parses a conversion expression where tok is the token
// "map" or "slice". It panics on error.
func parseConversionExpr(tok token, lex *lexer) ast.Expression {
	pos := tok.pos
	ident := ast.NewIdentifier(pos, string(tok.txt))
	elements, tok := parseExprList(token{}, lex, false, false)
	if tok.typ != tokenRightParenthesis {
		panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or )", tok)})
	}
	if len(elements) == 0 {
		panic(&Error{"", *tok.pos, fmt.Errorf("missing argument to conversion to %s: %s()", ident.Name, ident.Name)})
	}
	if len(elements) > 1 {
		panic(&Error{"", *tok.pos, fmt.Errorf("too many arguments to conversion to %s: %s(%s)", ident.Name, ident.Name, exprListString(elements))})
	}
	pos2 := *pos
	pos2.End = tok.pos.End
	return ast.NewCall(&pos2, ident, elements)
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

// parseNumberNode returns an Expression node from an integer, decimal or rune
// literal token possibly preceded by an unary operator "-" with neg position.
// It panics on error.
func parseNumberNode(tok token, neg *ast.Position) ast.Expression {
	p := tok.pos
	if neg != nil {
		p = neg
		p.End = tok.pos.End
	}
	if tok.typ == tokenRuneLiteral {
		var r rune
		if len(tok.txt) == 3 {
			r = rune(tok.txt[1])
		} else {
			r, _ = parseEscapedRune(tok.txt[1:])
		}
		if neg != nil {
			r = -r
		}
		return ast.NewInt(p, int(r))
	}
	s := string(tok.txt)
	if neg != nil {
		s = "-" + s
	}
	if bytes.IndexByte(tok.txt, '.') == -1 {
		n, err := strconv.Atoi(s)
		if err == nil {
			return ast.NewInt(p, n)
		}
	}
	d, _, _ := apd.NewFromString(s)
	if d.Cmp(minInt) >= 0 && d.Cmp(maxInt) <= 0 {
		i, err := d.Int64()
		if err == nil {
			return ast.NewInt(p, int(i))
		}
	}
	return ast.NewNumber(p, d)
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
