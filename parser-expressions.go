//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

package template

import (
	"bytes"
	"fmt"
	"strconv"

	"open2b/decimal"
	"open2b/template/ast"
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

	// un albero di una espressione ha come nodi intermedi gli operatori,
	// unari o binari, e come foglie gli operandi.
	//
	// una delle foglie, durante la costruzione dell'albero, invece di un
	// operando sarà un operatore. A questo operatore foglia gli mancherà
	// la sua unica espressione se unario o l'espressione di destra se
	// binario. Conclusa la costruzione, tutte le foglie saranno operandi.

	// path è il percorso nell'albero dall'operatore radice fino
	// all'operatore foglia. path alla fine conterrà solo un operatore
	// che sarà la radice dell'albero dell'espressione.
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
			expr, tok, err := parseExpr(lex)
			if err != nil {
				return nil, token{}, err
			}
			if expr == nil {
				return nil, token{}, fmt.Errorf("unexpected %s, expecting expression at %d", tok, tok.pos)
			}
			if tok.typ != tokenRightParenthesis {
				return nil, token{}, fmt.Errorf("unexpected %s, expecting ) at %d", tok, tok.pos)
			}
			operand = expr
		case
			tokenAddition,    // +e
			tokenSubtraction, // -e
			tokenNot:         // !e
			operator = ast.NewUnaryOperator(tok.pos, operatorType(tok.typ), nil)
		case tokenNumber: // 5.3
			operand = parseNumberNode(tok)
			// cambia il segno del numero e rimuove l'operatore dall'albero
			// se il numero è preceduto dall'operatore unario "-"
			if len(path) > 0 {
				if op, ok := path[len(path)-1].(*ast.UnaryOperator); ok {
					if op.Op == ast.OperatorSubtraction {
						switch n := operand.(type) {
						case *ast.Int:
							operand = ast.NewInt(op.Pos(), -n.Value)
						case *ast.Decimal:
							operand = ast.NewDecimal(op.Pos(), n.Value.Opposite())
						}
						path = path[:len(path)-1]
					}
				}
			}
		case
			tokenInterpretedString, // ""
			tokenRawString:         // ``
			operand = parseStringNode(tok)
		case tokenIdentifier: // a
			operand = parseIdentifierNode(tok)
		case tokenSemicolon:
			return nil, token{}, fmt.Errorf("unexpected semicolon or newline, expecting expression at %d", tok.pos)
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
				var pos = tok.pos
				var args = []ast.Expression{}
				for {
					arg, tok, err := parseExpr(lex)
					if err != nil {
						return nil, token{}, err
					}
					if arg == nil {
						if tok.typ != tokenRightParenthesis {
							return nil, token{}, fmt.Errorf("unexpected %s, expecting expression or ) at %d", tok, tok.pos)
						}
					} else {
						if tok.typ != tokenComma && tok.typ != tokenRightParenthesis {
							return nil, token{}, fmt.Errorf("unexpected %s, expecting comma or ) at %d", tok, tok.pos)
						}
						args = append(args, arg)
					}
					if tok.typ == tokenRightParenthesis {
						break
					}
				}
				operand = ast.NewCall(pos, operand, args)
			case tokenLeftBrackets: // e[...], e[.. : ..]
				pos := tok.pos
				index, tok, err := parseExpr(lex)
				if err != nil {
					return nil, token{}, err
				}
				if tok.typ == tokenColon {
					low := index
					high, tok, err := parseExpr(lex)
					if err != nil {
						return nil, token{}, err
					}
					if tok.typ != tokenRightBrackets {
						return nil, token{}, fmt.Errorf("unexpected %s, expecting ] at %d", tok, tok.pos)
					}
					operand = ast.NewSlice(pos, operand, low, high)
				} else {
					if tok.typ != tokenRightBrackets {
						return nil, token{}, fmt.Errorf("unexpected %s, expecting ] at %d", tok, tok.pos)
					}
					if index == nil {
						return nil, token{}, fmt.Errorf("unexpected ], expecting expression at %d", tok.pos)
					}
					operand = ast.NewIndex(pos, operand, index)
				}
			case tokenPeriod: // e.
				pos := tok.pos
				tok, ok = <-lex.tokens
				if !ok {
					return nil, token{}, lex.err
				}
				if tok.typ != tokenIdentifier {
					return nil, token{}, fmt.Errorf("unexpected %s, expecting name at %d", tok, tok.pos)
				}
				operand = ast.NewSelector(pos, operand, string(tok.txt))
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
				return nil, token{}, fmt.Errorf("unexpected EOF, expecting expression")
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
				// operator diventa il nuovo operatore foglia
				op.Expr1 = path[p]
				path[p] = op
				path = path[0 : p+1]
			} else {
				// operator diventa il nuovo operatore foglia
				op.Expr1 = operand
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

func parseNumberNode(tok token) ast.Expression {
	n, err := strconv.ParseInt(string(tok.txt), 10, strconv.IntSize)
	if err == nil {
		return ast.NewInt(tok.pos, int(n))
	}
	return ast.NewDecimal(tok.pos, decimal.String(string(tok.txt)))
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
				for j := 0; j < n; j++ {
					var c = s[i+j+2]
					switch {
					case '0' <= c && c <= '9':
						cc = append(cc, c-'0')
					case 'a' <= c && c <= 'f':
						cc = append(cc, c-'a'+10)
					case 'A' <= c && c <= 'F':
						cc = append(cc, c-'A'+10)
					}
				}
				i += n
			}
			i += 1
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
		return "", fmt.Errorf("unexpected %s, expecting string at %d", tok, tok.pos)
	}
	return unquoteString(tok.txt), nil
}
