//
// Copyright (c) 2016 Open2b Software Snc. All Rights Reserved.
//

package template

import (
	"bytes"
	"fmt"
	"strings"

	"open2b/decimal"
	"open2b/template/ast"
)

// expressionTree è utilizzato per mantenere il tree di una espressione
// durante il parsing.
// I nodi intermedi dell'albero sono gli operatori e le foglie gli operandi.
// path contiene gli operatori dalla root discendendo fino all'ultimo
// operatore parsato.
// Il seguente tree:
//
//                op1
//              /     \
//            op2     op3
//           /   \   /  \
//          a     b c
//
// può essere rappresentato:
//
//           op1----op3--
//            |      |
//           op2--b  c
//            |
//            a
//
// dove path sarà [ op1, op3 ]
//

// add aggiunge un operatore unario ( ad esempio "!" ) oppure un operando
// e un operatore binario ( ad esempio "a+" ) al tree.
// L'operatore viene aggiunto nella posizione in path in base alla sua
// precedenza. Partendo dall'ultimo operatore in path si risale il tree
// fino a trovare il primo operatore con precedenza minore e lo si
// aggiunge dopo di questo.
// Nel precedente tree di esempio se si aggiunge la coppia (d, op4)
// tra op1 e op3, il tree diventa:
//
//           op1----op4--
//            |      |
//           op2--b op3--d
//            |      |
//            a      c
//
// e path diventa [ op1, op4 ].
// Si fa notare che gli operatori unari hanno precedenza massima e
// pertanto vengono sempre aggiunti in testa a path. Ad esempio
// aggiungendo l'operatore unario op5:
//
//           op1----op4----op5--
//            |      |
//           op2--b op3--d
//            |      |
//            a      c
//
// e path diventa [ op1, op4, op5 ].
// Se l'operatore da aggiungere ha precendenza mimore di tutti gli
// altri operatori in path allora diventerà la nuova root e anche
// l'unico operatore in path. Ad esempio aggiungendo (e,op6):
//
//           op6--
//            |
//           op1----op4----op5--e
//            |      |
//           op2--b op3--d
//            |      |
//            a      c
//
// e path diventa [ op6 ].
//
// Una parentesi aperta si comporta come operatore a maggior precedenza
// quando viene aggiunto e a minore precedenza quando viene confrontato
// con i successivi operatori aggiunti.
//

// Legge prima un token, se è un operatore allora lo aggiunge al tree,
// se è un operando allora legge l'operatore successivo e gli aggiunge
// assieme al tree.
// Ad esempio l'espressione -a * ( b + c ) < d || !e
// viene letta come "-", "a*", "b*", "c)", "<", "d||", "!", "e " e aggiunta
// in questo ordine al tree.
// Al termine del parsing il primo operatore di tree sarà la root
// dell'espressione parsata.

// parseExpr esegue il parsing di una espressione e ne ritorna il tree e
// l'ultimo token letto che non è parte dell'espressione ritornata.
func parseExpr(lex *lexer) (ast.Expression, *token, error) {

	var path []ast.Operator

	for {

		tok, ok := <-lex.tokens
		if !ok {
			return nil, nil, lex.err
		}

		var operand ast.Expression
		var operator ast.Operator

		switch tok.typ {
		case tokenLeftParenthesis: // ( e )
			var pos = tok.pos
			var err error
			expr, tok, err := parseExpr(lex)
			if err != nil {
				return nil, nil, err
			}
			if expr == nil {
				return nil, nil, fmt.Errorf("unexpected %s, expecting expression at %d", tok, tok.pos)
			}
			if tok.typ != tokenRightParenthesis {
				return nil, nil, fmt.Errorf("unexpected %s, expecting ) at %d", tok, tok.pos)
			}
			operand = ast.NewParentesis(pos, expr)
		case
			tokenAddition,    // +e
			tokenSubtraction, // -e
			tokenNot:         // !e
			operator = ast.NewUnaryOperator(tok.pos, operatorType(tok.typ), nil)
		case tokenInt32: // 5
			operand = parseInt32Node(tok)
		case tokenInt64: // 5
			operand = parseInt64Node(tok)
		case tokenDecimal: // 5.3
			operand = parseDecimalNode(tok)
		case
			tokenInterpretedString, // ""
			tokenRawString:         // ``
			operand = parseStringNode(tok)
		case tokenIdentifier: // a
			operand = parseIdentifierNode(tok)
		case tokenSemicolon:
			return nil, nil, fmt.Errorf("unexpected semicolon or newline, expecting expression at %d", tok.pos)
		default:
			return nil, &tok, nil
		}

		for operator == nil {

			tok, ok = <-lex.tokens
			if !ok {
				return nil, nil, lex.err
			}

			switch tok.typ {
			case tokenLeftParenthesis:
				// e(...)
				var pos = tok.pos
				var args = []ast.Expression{}
				for {
					arg, tok, err := parseExpr(lex)
					if err != nil {
						return nil, nil, err
					}
					if arg == nil {
						if tok.typ != tokenRightParenthesis {
							return nil, nil, fmt.Errorf("unexpected %s, expecting expression or ) at %d", tok, tok.pos)
						}
					} else {
						if tok.typ != tokenComma && tok.typ != tokenRightParenthesis {
							return nil, nil, fmt.Errorf("unexpected %s, expecting comma or ) at %d", tok, tok.pos)
						}
						args = append(args, arg)
					}
					if tok.typ == tokenRightParenthesis {
						break
					}
				}
				operand = ast.NewCall(pos, operand, args)
			case tokenLeftBrackets: // e[...]
				pos := tok.pos
				index, tok, err := parseExpr(lex)
				if err != nil {
					return nil, nil, err
				}
				if tok.typ != tokenRightBrackets {
					return nil, nil, fmt.Errorf("unexpected %s, expecting ] at %d", tok, tok.pos)
				}
				if index == nil {
					return nil, nil, fmt.Errorf("unexpected ], expecting expression at %d", tok.pos)
				}
				operand = ast.NewIndex(pos, operand, index)
			case tokenPeriod: // e.
				pos := tok.pos
				tok, ok = <-lex.tokens
				if !ok {
					return nil, nil, lex.err
				}
				if tok.typ != tokenIdentifier {
					return nil, nil, fmt.Errorf("unexpected %s, expecting name at %d", tok, tok.pos)
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
				return nil, nil, fmt.Errorf("unexpected EOF, expecting expression")
			default:
				if tok.typ == tokenSemicolon {
					// salta ";" e legge il successivo token
					tok, ok = <-lex.tokens
					if !ok {
						return nil, nil, lex.err
					}
				}
				if path == nil {
					// si può ritornare direttamente
					return operand, &tok, nil
				}
				// aggiunge l'operando come figlio dell'ultimo operatore in path
				switch last := path[len(path)-1].(type) {
				case *ast.UnaryOperator:
					last.Expr = operand
				case *ast.BinaryOperator:
					last.Expr2 = operand
				}
				return path[0], &tok, nil
			}

		}

		// aggiunge operator a path. Per determinare la posizione
		// in cui aggiungerlo, parte dal fondo e risale fino a che
		// non incontra un operatore con precedenza minore.

		var p int
		for p = len(path); p > 0; p-- {
			if path[p-1].Precedence() < operator.Precedence() {
				break
			}
		}

		// se sopra c'è un altro operatore...
		if p > 0 {
			// ...diventa il genitore di operator
			switch parent := path[p-1].(type) {
			case *ast.UnaryOperator:
				parent.Expr = operator
			case *ast.BinaryOperator:
				parent.Expr2 = operator
			}
		}
		// se sotto c'è un altro operatore...
		if p < len(path) {
			// ...diventa il figlio di operator
			if op, ok := operator.(*ast.BinaryOperator); ok {
				op.Expr1 = path[p]
			}
			switch child := path[p].(type) {
			case *ast.UnaryOperator:
				child.Expr = operand
			case *ast.BinaryOperator:
				child.Expr2 = operand
			}
			path[p] = operator
			path = path[0 : p+1]
		} else {
			// ...altrimenti operator viene aggiunto in fondo
			if op, ok := operator.(*ast.BinaryOperator); ok {
				op.Expr1 = operand
			}
			path = append(path, operator)
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

func parseInt32Node(tok token) *ast.Int32 {
	var n int
	for _, c := range tok.txt {
		n = n*10 + int(c-'0')
	}
	return ast.NewInt32(tok.pos, int32(n))
}

func parseInt64Node(tok token) *ast.Int64 {
	var n int64
	for _, c := range tok.txt {
		n = n*10 + int64(c-'0')
	}
	return ast.NewInt64(tok.pos, n)
}

func parseDecimalNode(tok token) *ast.Decimal {
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

func parsePath(lex *lexer) (string, error) {
	var tok, ok = <-lex.tokens
	if !ok {
		return "", lex.err
	}
	if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
		return "", fmt.Errorf("unexpected %s, expecting string at %d", tok, tok.pos)
	}
	var path = unquoteString(tok.txt)
	if !isValidFilePath(path) {
		return "", fmt.Errorf("invalid include path %q at %d", path, tok.pos)
	}
	return path, nil
}

func isValidDirName(name string) bool {
	// deve essere lungo almeno un carattere e meno di 256
	if name == "" || len(name) >= 256 {
		return false
	}
	// non deve essere '.' e non deve contenere '..'
	if name == "." || strings.Contains(name, "..") {
		return false
	}
	// il primo e ultimo carattere non devono essere spazi
	if name[0] == ' ' || name[len(name)-1] == ' ' {
		return false
	}
	// non deve contenere caratteri speciali
	for _, c := range name {
		if ('\x00' <= c && c <= '\x1f') || c == '\x22' || c == '\x2a' || c == '\x2f' ||
			c == '\x3a' || c == '\x3c' || c == '\x3e' || c == '\x3f' || c == '\x5c' ||
			c == '\x7c' || c == '\x7f' {
			return false
		}
	}
	// non deve essere un nome riservato per Window
	if name == "con" || name == "prn" || name == "aux" || name == "nul" ||
		(len(name) > 3 && name[0:4] == "com" && '0' <= name[3] && name[3] <= '9') ||
		(len(name) > 3 && name[0:4] == "lpt" && '0' <= name[3] && name[3] <= '9') {
		if len(name) == 4 || name[4] == '.' {
			return false
		}
	}
	return true
}

func isValidFileName(name string) bool {
	// deve essere lungo almeno 3 caratteri e meno di 256
	if len(name) <= 2 || len(name) >= 256 {
		return false
	}
	// deve essere presente uno e un solo punto e
	// non deve essere ne il primo e ne l'ultimo carattere
	var dot = strings.IndexByte(name, '.')
	if dot <= 0 || dot == len(name)-1 {
		return false
	}
	name = strings.ToLower(name)
	var ext = name[dot+1:]
	if strings.IndexByte(ext, '.') >= 0 {
		return false
	}
	// il primo e ultimo carattere non devono essere spazi
	if name[0] == ' ' || name[len(name)-1] == ' ' {
		return false
	}
	// non deve contenere caratteri speciali
	for _, c := range name {
		if ('\x00' <= c && c <= '\x1f') || c == '\x22' || c == '\x2a' || c == '\x2f' ||
			c == '\x3a' || c == '\x3c' || c == '\x3e' || c == '\x3f' || c == '\x5c' ||
			c == '\x7c' || c == '\x7f' {
			return false
		}
	}
	// non deve essere un nome riservato per Window
	if name == "con" || name == "prn" || name == "aux" || name == "nul" ||
		(len(name) > 3 && name[0:4] == "com" && '0' <= name[3] && name[3] <= '9') ||
		(len(name) > 3 && name[0:4] == "lpt" && '0' <= name[3] && name[3] <= '9') {
		if len(name) == 4 || name[4] == '.' {
			return false
		}
	}
	return true
}

// isValidFilePath indica se path è valido per essere
// usato come path in include ed extend.
// Sono path validi: '/a', '/a/a', 'a', 'a/a', 'a.a'.
// Sono path non validi: '', '/', 'a/', '../a'.
func isValidFilePath(path string) bool {
	// deve avere almeno un caratteri e non terminare con '/'
	if len(path) < 1 || path[len(path)-1] == '/' {
		return false
	}
	// splitta il path nei vari nomi
	var names = strings.Split(path, "/")
	// i primi nomi devono essere delle directory
	for i, name := range names[:len(names)-1] {
		// se il primo nome è vuoto...
		if i == 0 && name == "" {
			// ...allora path inizia con '/'
			continue
		}
		if !isValidDirName(name) {
			return false
		}
	}
	// l'ultimo nome deve essere un file
	if !isValidFileName(names[len(names)-1]) {
		return false
	}
	return true
}
