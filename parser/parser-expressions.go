//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

package parser

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"

	"open2b/template/ast"
	"open2b/template/types"
)

// Il risultato del parsing di una espressione è un albero i cui nodi
// intermedi sono operatori e le foglie operandi. Ad esempio:
//
//                op1
//              /     \
//            op2     op3
//           /   \     |
//          a     b    c
//
// Durante la costruzione dell'albero, la foglia più a destra, invece di
// essere un operando, è un operatore:
//
//                op1
//              /     \
//            op2     op3
//           /   \
//          a     b
//
// L'operazione di aggiunta di un operatore all'albero si chiama innesto
// e avviene sempre lungo il percorso, chiamato semplicemente path, che
// parte dalla radice e arriva all'operatore foglia. Nel precedente
// esempio path è [ op1, op3 ].
//
// Un operatore unario è innestato come figlio dell'operatore foglia:
//
//                op1
//              /     \
//            op2     op3     path = [ op1, op3, op4 ]
//           /   \     |
//          a     b   op4
//
// Per determinare dove innestare un operatore non unario, si percorre
// path dall'operatore foglia alla radice fermandosi quando si trova
// o un operatore con precedenza minore o si raggiunge la radice.
// Ad esempio innestando op5 con precedenza maggiore di op1 e minore o
// uguale a op3 e op4 si ottine l'albero:
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
// Se op5 avesse precedenza minore o uguale di op1:
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
// Se op5 avesse presendenza maggiore di op4:
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

// parseExpr esegue il parsing di una espressione e ritorna l'albero
// dell'espressione e l'ultimo token letto che non è parte.
func parseExpr(lex *lexer) (ast.Expression, token, error) {

	var err error

	// un albero di una espressione ha come nodi intermedi gli operatori,
	// unari o binari, e come foglie gli operandi.
	//
	// una delle foglie, durante la costruzione dell'albero, invece di un
	// operando sarà un operatore. A questo operatore foglia gli mancherà
	// la sua unica espressione se unario o l'espressione di destra se
	// binario. Conclusa la costruzione, tutte le foglie saranno operandi.

	// path è il percorso nell'albero dall'operatore radice fino
	// all'operatore foglia.
	var path []ast.Operator

	// ad ogni ciclo della costruzione dell'albero dell'espressione, viene
	// letto o un operatore unario oppure una coppia operando e operatore
	// binario. Per ultimo verrà letto un operando.
	//
	// ad esempio l'espressione "-a * ( b + c ) < d || !e" viene letta
	// come "-", "a *", "b *", "c ()", "<", "d ||", "!", "e".
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
			// richiama ricorsivamente parseExpr per parsare l'espressione
			// tra parentesi per poi trattarla come singolo operando.
			// le parentesi non saranno presenti nell'albero dell'espressione.
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
		case tokenNumber: // 5.3
			// se il numero è preceduto dall'operatore unario "-"
			// cambia il segno del numero e rimuove l'operatore dall'albero
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
				if fc, _ := utf8.DecodeRuneInString(ident); !unicode.Is(unicode.Lu, fc) {
					return nil, token{}, &Error{"", *tok.pos, fmt.Errorf("cannot refer to unexported field or method %s", ident)}
				}
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
					// salta ";" e legge il successivo token
					tok, ok = <-lex.tokens
					if !ok {
						return nil, token{}, lex.err
					}
				}
				if len(path) == 0 {
					// si può ritornare direttamente
					return operand, tok, nil
				}
				// aggiunge l'operando come figlio dell'operatore foglia
				switch leef := path[len(path)-1].(type) {
				case *ast.UnaryOperator:
					leef.Expr = operand
				case *ast.BinaryOperator:
					leef.Expr2 = operand
				}
				// imposta End per tutti gli operatori in path
				end := operand.Pos().End
				for _, op := range path {
					switch o := op.(type) {
					case *ast.UnaryOperator:
						o.Position.End = end
					case *ast.BinaryOperator:
						o.Position.End = end
					}
				}
				// ritorna la radice dell'albero dell'espressione
				return path[0], tok, nil
			}

		}

		// aggiunge operator all'albero dell'espressione

		switch op := operator.(type) {

		case *ast.UnaryOperator:
			// gli operatori unari ("!", "+", "-", "()"), siccome hanno
			// precedenza maggiore di tutti gli altri operatori, diventano
			// il nuovo operatore foglia dell'albero.

			if len(path) > 0 {
				// operator diventa figlio dell'operatore foglia
				switch leef := path[len(path)-1].(type) {
				case *ast.UnaryOperator:
					leef.Expr = op
				case *ast.BinaryOperator:
					leef.Expr2 = op
				}
			}
			// operator diventa il nuovo operatore foglia
			path = append(path, op)

		case *ast.BinaryOperator:
			// per gli operatori binari ("*", "/", "+", "-", "<", ">", ...)
			// si parte dall'operatore foglia (ultimo operatore di path) e
			// si sale verso la radice (primo operatore di path) fermandosi
			// quando o si trova un operatore con precedenza minore o si
			// raggiunge la radice.

			// imposta start per tutti gli operatori unari alla fine di path
			start := operand.Pos().Start
			for i := len(path) - 1; i >= 0; i-- {
				if o, ok := path[i].(*ast.UnaryOperator); ok {
					o.Position.Start = start
				} else {
					break
				}
			}

			// p sarà la posizione in path in cui aggiungere operator
			var p = len(path)
			for p > 0 && op.Precedence() <= path[p-1].Precedence() {
				p--
			}
			if p > 0 {
				// operator diventa figlio dell'operatore con precedenza
				// minore trovato risalendo path
				switch o := path[p-1].(type) {
				case *ast.UnaryOperator:
					o.Expr = op
				case *ast.BinaryOperator:
					o.Expr2 = op
				}
			}
			if p < len(path) {
				// operand diventa figlio dell'operatore
				// in path attualmente in posizione p
				switch o := path[p].(type) {
				case *ast.UnaryOperator:
					o.Expr = operand
				case *ast.BinaryOperator:
					o.Expr2 = operand
				}
				// imposta End per tutti gli operatori in path
				// da p in poi
				for i := p; i < len(path); i++ {
					switch o := path[i].(type) {
					case *ast.UnaryOperator:
						o.Position.End = operand.Pos().End
					case *ast.BinaryOperator:
						o.Position.End = operand.Pos().End
					}
				}
				// operator diventa il nuovo operatore foglia
				op.Expr1 = path[p]
				op.Position.Start = path[p].Pos().Start
				path[p] = op
				path = path[0 : p+1]
			} else {
				// operator diventa il nuovo operatore foglia
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

// parseNumberNode ritorna un nodo Expression da un token numero,
// eventualmente preceduto da un operatore unario "-" con posizione neg.
func parseNumberNode(tok token, neg *ast.Position) ast.Expression {
	p := tok.pos
	s := string(tok.txt)
	if neg != nil {
		p = neg
		p.End = tok.pos.End
		s = "-" + s
	}
	return ast.NewNumber(p, types.NewNumberString(s))
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

// parseString esegue il parsing di una stringa e ne ritorna l'espressione.
func parseString(lex *lexer) (string, error) {
	var tok, ok = <-lex.tokens
	if !ok {
		return "", lex.err
	}
	if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
		return "", &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting string", tok)}
	}
	return unquoteString(tok.txt), nil
}
