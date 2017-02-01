//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

package template

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"
)

type SyntaxError struct {
	path string
	str  string
	pos  int
	len  int
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("template: %s at %q position %d", e.str, e.path, e.pos)
}

// lexer mantiene lo stato dello scanner.
type lexer struct {
	text   []byte     // testo su cui viene eseguita la scansine
	src    []byte     // slice del testo utilizzata durante la scansione
	ctx    context    // contesto corrente utilizzato durante la scansione
	tokens chan token // tokens, viene chiuso alla fine della scansione
	err    error      // errore, al termine della scansione indica se c'è stato un errore
}

// newLexer crea un nuovo lexer.
func newLexer(text []byte) *lexer {
	tokens := make(chan token, 20)
	lex := &lexer{text: text, src: text, ctx: contextHTML, err: nil, tokens: tokens}
	go lex.scan()
	return lex
}

func (l *lexer) errorf(format string, args ...interface{}) *SyntaxError {
	return &SyntaxError{str: fmt.Sprintf(format, args...), pos: len(l.text) - len(l.src)}
}

// emit emette un token di tipo typ e lunghezza length.
func (l *lexer) emit(typ tokenType, length int) {
	var txt []byte
	if typ != tokenEOF {
		txt = l.src[0:length]
	}
	ctx := l.ctx
	if typ == tokenText {
		ctx = contextHTML
	}
	l.tokens <- token{
		typ: typ,
		pos: len(l.text) - len(l.src),
		txt: txt,
		ctx: ctx,
	}
	l.src = l.src[length:]
}

// scan esegue la scansione del testo mettendo i token sul canale tokens.
// In caso di errore mette l'errore in err, chiude il canale ed esce.
func (l *lexer) scan() {

	var p = 0

LOOP:
	for p+1 < len(l.src) {
		switch l.src[p] {
		case '{':
			switch l.src[p+1] {
			case '{':
				if p > 0 {
					l.emit(tokenText, p)
				}
				err := l.lexShow()
				if err != nil {
					l.src = nil
					l.err = err
					break LOOP
				}
				p = 0
			case '%':
				if p > 0 {
					l.emit(tokenText, p)
				}
				err := l.lexStatement()
				if err != nil {
					l.src = nil
					l.err = err
					break LOOP
				}
				p = 0
			}
		case '<':
			p += l.readHTML(l.src[p:]) + 1
		default:
			p++
		}
	}

	if len(l.src) > 0 {
		l.emit(tokenText, len(l.src))
	}

	l.emit(tokenEOF, 0)

	l.src = nil
	l.ctx = 0

	close(l.tokens)

	return
}

// readHTML legge un tag HTML sapendo che src inizia con '<'.
// Se necessario cambia il contesto del lexer.
func (l *lexer) readHTML(src []byte) int {
	if len(src) < 2 {
		return 0
	}
	var p = 0
	switch l.ctx {
	case contextHTML:
		if src[1] == 's' {
			if prefix(src, []byte("<script")) {
				if len(src) > 7 && (src[8] == '>' || isSpace(src[8]) || src[8] == '{') {
					l.ctx = contextScript
				}
			} else if prefix(src, []byte("<style")) {
				if len(src) > 6 && (src[7] == '>' || isSpace(src[7]) || src[8] == '{') {
					l.ctx = contextStyle
				}
			}
		} else if src[1] == '!' {
			if prefix(src, []byte("<!--")) {
				// salta il commmento
				var i = bytes.Index(src[4:], []byte("-->"))
				if i < 0 {
					p = len(src) - 1
				} else {
					p = i + 6
				}
			} else if prefix(src, []byte("<![CDATA[")) {
				// salta la sezione CDATA
				var i = bytes.Index(src[9:], []byte("]]>"))
				if i < 0 {
					p = len(src) - 1
				} else {
					p = i + 11
				}
			}
		}
	case contextStyle:
		if src[1] == '/' && prefix(src, []byte("</style>")) {
			l.ctx = contextHTML
		}
	case contextScript:
		if src[1] == '/' && prefix(src, []byte("</script>")) {
			l.ctx = contextHTML
		}
	}
	return p
}

// lexShow legge un tag show sapendo che src inzia con '{{'.
func (l *lexer) lexShow() error {
	l.emit(tokenStartShow, 2)
	err := l.lexCode()
	if err != nil {
		return err
	}
	if len(l.src) < 2 {
		return l.errorf("unexpected EOF, expecting }}")
	}
	if l.src[0] != '}' || l.src[1] != '}' {
		return l.errorf("unexpected %s, expecting }}", l.src[:2])
	}
	l.emit(tokenEndShow, 2)
	return nil
}

// lexStatement legge un tag statement sapendo che src inizia con '{%'.
func (l *lexer) lexStatement() error {
	l.emit(tokenStartStatement, 2)
	err := l.lexCode()
	if err != nil {
		return err
	}
	if len(l.src) < 2 {
		return l.errorf("unexpected EOF, expecting %%}")
	} else if l.src[0] != '%' && l.src[1] != '}' {
		return l.errorf("unexpected %s, expecting %%}", l.src[:2])
	}
	l.emit(tokenEndStatement, 2)
	return nil
}

// lexCode legge del codice.
func (l *lexer) lexCode() error {
	if len(l.src) == 0 {
		return nil
	}
	// endLineAsSemicolon indica se "\n" deve essere trattato come ";"
	var endLineAsSemicolon = false
LOOP:
	for len(l.src) > 0 {
		switch c := l.src[0]; c {
		case '"':
			err := l.lexInterpretedString()
			if err != nil {
				return err
			}
			endLineAsSemicolon = true
		case '`':
			err := l.lexRawString()
			if err != nil {
				return err
			}
			endLineAsSemicolon = true
		case '.':
			if len(l.src) == 1 {
				return l.errorf("unexpected EOF")
			}
			if '0' <= l.src[1] && l.src[1] <= '9' {
				l.lexNumber()
				endLineAsSemicolon = true
			} else {
				l.emit(tokenPeriod, 1)
				endLineAsSemicolon = false
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			l.lexNumber()
			endLineAsSemicolon = true
		case '=':
			if len(l.src) == 1 || l.src[1] != '=' {
				l.emit(tokenAssignment, 1)
			} else {
				l.emit(tokenEqual, 2)
			}
			endLineAsSemicolon = false
		case '+':
			l.emit(tokenAddition, 1)
			endLineAsSemicolon = false
		case '-':
			l.emit(tokenSubtraction, 1)
			endLineAsSemicolon = false
		case '*':
			l.emit(tokenMultiplication, 1)
			endLineAsSemicolon = false
		case '/':
			l.emit(tokenDivision, 1)
			endLineAsSemicolon = false
		case '%':
			if len(l.src) > 1 && l.src[1] == '}' {
				break LOOP
			}
			l.emit(tokenModulo, 1)
			endLineAsSemicolon = false
		case '&':
			if len(l.src) == 1 {
				return l.errorf("unexpected EOF")
			}
			if l.src[1] != '&' {
				c, _ := utf8.DecodeRune(l.src[1:])
				return l.errorf("unexpected %c, expecting &", c)
			}
			l.emit(tokenAnd, 2)
			endLineAsSemicolon = false
		case '|':
			if len(l.src) == 1 {
				return l.errorf("unexpected EOF")
			}
			if l.src[1] != '|' {
				c, _ := utf8.DecodeRune(l.src[1:])
				return l.errorf("unexpected %c, expecting |", c)
			}
			l.emit(tokenOr, 2)
			endLineAsSemicolon = false
		case '!':
			if len(l.src) > 1 && l.src[1] == '=' {
				l.emit(tokenNotEqual, 2)
			} else {
				l.emit(tokenNot, 1)
			}
			endLineAsSemicolon = false
		case '<':
			if len(l.src) > 1 && l.src[1] == '=' {
				l.emit(tokenLessOrEqual, 2)
			} else {
				l.emit(tokenLess, 1)
			}
			endLineAsSemicolon = false
		case '>':
			if len(l.src) > 1 && l.src[1] == '=' {
				l.emit(tokenGreaterOrEqual, 2)
			} else {
				l.emit(tokenGreater, 1)
			}
			endLineAsSemicolon = false
		case '(':
			l.emit(tokenLeftParenthesis, 1)
			endLineAsSemicolon = false
		case ')':
			l.emit(tokenRightParenthesis, 1)
			endLineAsSemicolon = true
		case '[':
			l.emit(tokenLeftBrackets, 1)
			endLineAsSemicolon = false
		case ']':
			l.emit(tokenRightBrackets, 1)
			endLineAsSemicolon = true
		case ':':
			l.emit(tokenColon, 1)
			endLineAsSemicolon = false
		case '}':
			if len(l.src) > 1 && l.src[1] == '}' {
				break LOOP
			}
			return l.errorf("unexpected }")
		case ',':
			l.emit(tokenComma, 1)
			endLineAsSemicolon = false
		case ' ', '\t', '\r':
			l.src = l.src[1:]
		case '\n':
			if endLineAsSemicolon {
				l.emit(tokenSemicolon, 1)
				endLineAsSemicolon = false
			} else {
				l.src = l.src[1:]
			}
		case ';':
			l.emit(tokenSemicolon, 1)
			endLineAsSemicolon = false
		default:
			if c == '_' || c < utf8.RuneSelf && unicode.IsLetter(rune(c)) {
				endLineAsSemicolon = l.lexIdentifierOrKeyword(1)
			} else {
				r, s := utf8.DecodeRune(l.src)
				if unicode.IsLetter(r) {
					endLineAsSemicolon = l.lexIdentifierOrKeyword(s)
				} else {
					return l.errorf("unexpected %c", r)
				}
			}
		}
	}
	return nil
}

func prefix(s, prefix []byte) bool {
	return bytes.HasPrefix(s, prefix)
}

// isSpace indica se s è uno spazio.
func isSpace(s byte) bool {
	return s == ' ' || s == '\t' || s == '\n' || s == '\r'
}

// lexIdentifierOrKeyword legge un identificatore o una keyword
// sapendo che src inizia con una lettera di s byte.
func (l *lexer) lexIdentifierOrKeyword(s int) bool {
	// si ferma solo quando un carattere
	// non può essere parte dell'identificatore o keyword
	var p = s
	for p < len(l.src) {
		r, s := utf8.DecodeRune(l.src[p:])
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}
		p += s
	}
	switch string(l.src[0:p]) {
	case "var":
		l.emit(tokenVar, p)
	case "for":
		l.emit(tokenFor, p)
	case "in":
		l.emit(tokenIn, p)
	case "if":
		l.emit(tokenIf, p)
	case "extend":
		l.emit(tokenExtend, p)
	case "include":
		l.emit(tokenInclude, p)
	case "show":
		l.emit(tokenShow, p)
	case "region":
		l.emit(tokenRegion, p)
	case "snippet":
		l.emit(tokenSnippet, p)
	case "end":
		l.emit(tokenEnd, p)
	default:
		l.emit(tokenIdentifier, p)
		return true
	}
	return false
}

// lexNumber legge un number sapendo che src inizia con '0'..'9' o '.'.
func (l *lexer) lexNumber() {
	// si ferma solo se un carattere non può essere parte del numero
	// oppure se il numero non è rappresentabile
	var hasDot = l.src[0] == '.'
	var p = 1
	for p < len(l.src) {
		if l.src[p] == '.' {
			if hasDot {
				break
			}
			hasDot = true
		} else if l.src[p] < '0' || '9' < l.src[p] {
			break
		}
		p++
	}
	if p > 29 {
		l.errorf("constant %s overflows decimal", l.src[0:p])
		return
	}
	l.emit(tokenNumber, p)
	return
}

// lexString legge una stringa "..." o `...` sapendo che src non inizia
// con spazi.
func (l *lexer) lexString() error {
	if len(l.src) == 0 {
		return l.errorf("unexpected EOF, expecting string")
	}
	switch l.src[0] {
	case '"':
		return l.lexInterpretedString()
	case '`':
		return l.lexRawString()
	default:
		c, _ := utf8.DecodeRune(l.src)
		return l.errorf("unexpected %c, expecting string", c)
	}
}

// lexInterpretedString legge una stringa "..." sapendo che src inizia con '"'.
func (l *lexer) lexInterpretedString() error {
	// si ferma quando trova il carattere '"' e ritorna errore
	// quando trova un carattere Unicode che non è valido in una stringa
	var p = 1
LOOP:
	for {
		if p == len(l.src) {
			return l.errorf("not closed string literal")
		}
		switch l.src[p] {
		case '"':
			break LOOP
		case '\\':
			if len(l.src) < p+3 {
				return l.errorf("not closed string literal")
			}
			switch c := l.src[p+1]; c {
			case 'u', 'U':
				var n = 4
				if c == 'U' {
					n = 8
				}
				if p+2+n < len(l.src) {
					return l.errorf("not closed string literal")
				}
				for i := 0; i < n; i++ {
					c = l.src[p+2+i]
					if (c < 0 || 9 < c) && (c < 'A' || 'F' < c) && (c < 'a' || 'f' < c) {
						l.src = l.src[p:]
						return l.errorf("invalid hex digit in string literal")
					}
				}
				p += 2 + n
			case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '"':
				p += 2
			default:
				l.src = l.src[p:]
				return l.errorf("invalid escape in string literal")
			}
		case '\n':
			l.src = l.src[p:]
			return l.errorf("invalid new line in string literal")
		default:
			r, s := utf8.DecodeRune(l.src[p:])
			if r == utf8.RuneError {
				l.src = l.src[p:]
				return l.errorf("invalid byte in string literal")
			}
			p += s
		}
	}
	l.emit(tokenInterpretedString, p+1)
	return nil
}

// lexRawString legge una stringa `...` sapendo che src inizia con '`'.
func (l *lexer) lexRawString() error {
	// si ferma quando trova il carattere '`' e ritorna errore
	// quando trova un carattere unicode non valido in una stringa
	var p = 1
	for {
		if p == len(l.src) {
			return l.errorf("not closed string literal")
		}
		if l.src[p] == '`' {
			break
		} else {
			r, s := utf8.DecodeRune(l.src[p:])
			if r == utf8.RuneError {
				l.src = l.src[p:]
				return l.errorf("invalid byte in string literal")
			}
			p += s
		}
	}
	l.emit(tokenRawString, p+1)
	return nil
}
