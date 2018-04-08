//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package parser

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"

	"open2b/template/ast"
)

var nl = []byte("\n")
var cdataStart = []byte("<![CDATA[")
var cdataEnd = []byte("]]>")

type SyntaxError struct {
	path string
	str  string
	pos  int
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("template: %s at %q position %d", e.str, e.path, e.pos)
}

// lexer mantiene lo stato dello scanner.
type lexer struct {
	text   []byte      // testo su cui viene eseguita la scansine
	src    []byte      // slice del testo utilizzata durante la scansione
	line   int         // linea corrente a partire da 1
	column int         // colonna corente a partire da 1
	ctx    ast.Context // contesto corrente utilizzato durante la scansione
	tokens chan token  // tokens, viene chiuso alla fine della scansione
	err    error       // errore, al termine della scansione indica se c'è stato un errore
}

// newLexer crea un nuovo lexer.
func newLexer(text []byte, ctx ast.Context) *lexer {
	tokens := make(chan token, 20)
	lex := &lexer{
		text:   text,
		src:    text,
		line:   1,
		column: 1,
		ctx:    ctx,
		tokens: tokens}
	go lex.scan()
	return lex
}

func (l *lexer) newline() {
	l.line++
	l.column = 1
}

func (l *lexer) errorf(format string, args ...interface{}) *SyntaxError {
	return &SyntaxError{str: fmt.Sprintf(format, args...), pos: len(l.text) - len(l.src)}
}

// emit emette alla linea corrente e colonna corrente un token
// di tipo typ e lunghezza length.
func (l *lexer) emit(typ tokenType, length int) {
	l.emitAtLineColumn(l.line, l.column, typ, length)
}

// emitAtLineColumn emette alla linea line e colonna column un token
// di tipo typ e lunghezza length.
func (l *lexer) emitAtLineColumn(line, column int, typ tokenType, length int) {
	var txt []byte
	if typ != tokenEOF {
		txt = l.src[0:length]
	}
	ctx := l.ctx
	if typ == tokenText {
		ctx = ast.ContextText
	}
	start := len(l.text) - len(l.src)
	l.tokens <- token{
		typ: typ,
		pos: &ast.Position{
			Line:   line,
			Column: column,
			Start:  start,
			End:    start + length - 1,
		},
		txt: txt,
		lin: l.line,
		ctx: ctx,
	}
	l.src = l.src[length:]
}

// scan esegue la scansione del testo mettendo i token sul canale tokens.
// In caso di errore mette l'errore in err, chiude il canale ed esce.
func (l *lexer) scan() {

	p := 0 // lunghezza del token in bytes

	lin := l.line   // linea del token
	col := l.column // colonna del token

	initialContext := l.ctx // contesto iniziale

LOOP:
	for p < len(l.src) {
		c := l.src[p]
		if c == '{' && p+1 < len(l.src) {
			switch l.src[p+1] {
			case '{':
				if p > 0 {
					l.emitAtLineColumn(lin, col, tokenText, p)
				}
				err := l.lexShow()
				if err != nil {
					l.src = nil
					l.err = err
					break LOOP
				}
				p = 0
				lin = l.line
				col = l.column
				continue
			case '%':
				if p > 0 {
					l.emitAtLineColumn(lin, col, tokenText, p)
				}
				err := l.lexStatement()
				if err != nil {
					l.src = nil
					l.err = err
					break LOOP
				}
				p = 0
				lin = l.line
				col = l.column
				continue
			case '#':
				if p > 0 {
					l.emitAtLineColumn(lin, col, tokenText, p)
				}
				err := l.lexComment()
				if err != nil {
					l.src = nil
					l.err = err
					break LOOP
				}
				p = 0
				lin = l.line
				col = l.column
				continue
			}
		}
		if c == '\n' {
			p++
			l.newline()
			continue
		}
		if c == '<' && initialContext == ast.ContextHTML {
			switch l.ctx {
			case ast.ContextHTML:
				// <style>
				if p+6 < len(l.src) && isCSS(l.src[p+1:p+6]) && (l.src[p+6] == '>' || isSpace(l.src[p+6])) {
					l.ctx = ast.ContextCSS
					p += 6
					l.column += 6
					continue
				}
				// <script>
				if p+7 < len(l.src) && isJavaScript(l.src[p+1:p+7]) && (l.src[p+7] == '>' || isSpace(l.src[p+7])) {
					l.ctx = ast.ContextJavaScript
					p += 7
					l.column += 7
					continue
				}
			case ast.ContextCSS:
				// </style>
				if p+7 < len(l.src) && l.src[p+1] == '/' && isCSS(l.src[p+2:p+7]) && (l.src[p+7] == '>' || isSpace(l.src[p+7])) {
					l.ctx = ast.ContextHTML
					p += 7
					l.column += 7
					continue
				}
			case ast.ContextJavaScript:
				// </script>
				if p+8 < len(l.src) && l.src[p+1] == '/' && isJavaScript(l.src[p+2:p+8]) && (l.src[p+8] == '>' || isSpace(l.src[p+8])) {
					l.ctx = ast.ContextHTML
					p += 8
					l.column += 8
					continue
				}
			}
			// <![CDATA[...]]>
			if p+11 < len(l.src) && l.src[p+1] == '!' {
				if bytes.HasPrefix(l.src[p:], cdataStart) {
					// salta la sezione CDATA
					p += 9
					l.column += 9
					var t = bytes.Index(l.src[p:], cdataEnd)
					if t < 0 {
						t = len(l.src)
					}
					for p < t {
						c := l.src[p]
						if c == '\n' {
							l.newline()
						} else if c < 128 {
							p++
							l.column++
						} else {
							_, s := utf8.DecodeRune(l.src[p:])
							p += s
							l.column++
						}
					}
					continue
				}
			}
		}
		if c < 128 {
			p += 1
		} else {
			_, s := utf8.DecodeRune(l.src[p:])
			p += s
		}
		l.column++
	}

	if len(l.src) > 0 {
		l.emitAtLineColumn(lin, col, tokenText, p)
	}

	l.emit(tokenEOF, 0)

	l.src = nil
	l.ctx = 0

	close(l.tokens)
}

func isJavaScript(s []byte) bool {
	if len(s) < 6 {
		return false
	}
	if (s[0] == 's' || s[0] == 'S') && (s[1] == 'c' || s[1] == 'C') && (s[2] == 'r' || s[2] == 'R') &&
		(s[3] == 'i' || s[3] == 'I') && (s[4] == 'p' || s[4] == 'P') && (s[5] == 't' || s[5] == 'T') {
		return true
	}
	return false
}

func isCSS(s []byte) bool {
	if len(s) < 5 {
		return false
	}
	if (s[0] == 's' || s[0] == 'S') && (s[1] == 't' || s[1] == 'T') && (s[2] == 'y' || s[2] == 'Y') &&
		(s[3] == 'l' || s[3] == 'L') && (s[4] == 'e' || s[4] == 'E') {
		return true
	}
	return false
}

// lexShow emette i token di show sapendo che src inizia con '{{'.
func (l *lexer) lexShow() error {
	l.emit(tokenStartValue, 2)
	l.column += 2
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
	l.emit(tokenEndValue, 2)
	l.column += 2
	return nil
}

// lexStatement emette i token di uno statement sapendo che src inizia con '{%'.
func (l *lexer) lexStatement() error {
	l.emit(tokenStartStatement, 2)
	l.column += 2
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
	l.column += 2
	return nil
}

// lexComment emette il token di un commento sapendo che src inizia con '{#'.
func (l *lexer) lexComment() error {
	p := bytes.Index(l.src[2:], []byte("#}"))
	if p == -1 {
		return l.errorf("unexpected EOF, expecting #}")
	}
	line := l.line
	column := l.column
	l.column += 2
	for i := 2; i < p+2; i++ {
		if l.src[i] == '\n' {
			l.newline()
		} else {
			l.column++
		}
	}
	l.emitAtLineColumn(line, column, tokenComment, p+4)
	return nil
}

// lexCode emette i token di codice.
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
			} else if l.src[1] == '.' {
				l.emit(tokenRange, 2)
				l.column += 2
				endLineAsSemicolon = false
			} else {
				l.emit(tokenPeriod, 1)
				l.column++
				endLineAsSemicolon = false
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			l.lexNumber()
			endLineAsSemicolon = true
		case '=':
			if len(l.src) == 1 || l.src[1] != '=' {
				l.emit(tokenAssignment, 1)
				l.column++
			} else {
				l.emit(tokenEqual, 2)
				l.column += 2
			}
			endLineAsSemicolon = false
		case '+':
			l.emit(tokenAddition, 1)
			l.column++
			endLineAsSemicolon = false
		case '-':
			l.emit(tokenSubtraction, 1)
			l.column++
			endLineAsSemicolon = false
		case '*':
			l.emit(tokenMultiplication, 1)
			l.column++
			endLineAsSemicolon = false
		case '/':
			l.emit(tokenDivision, 1)
			l.column++
			endLineAsSemicolon = false
		case '%':
			if len(l.src) > 1 && l.src[1] == '}' {
				break LOOP
			}
			l.emit(tokenModulo, 1)
			l.column++
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
			l.column += 2
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
			l.column += 2
			endLineAsSemicolon = false
		case '!':
			if len(l.src) > 1 && l.src[1] == '=' {
				l.emit(tokenNotEqual, 2)
				l.column += 2
			} else {
				l.emit(tokenNot, 1)
				l.column++
			}
			endLineAsSemicolon = false
		case '<':
			if len(l.src) > 1 && l.src[1] == '=' {
				l.emit(tokenLessOrEqual, 2)
				l.column += 2
			} else {
				l.emit(tokenLess, 1)
				l.column++
			}
			endLineAsSemicolon = false
		case '>':
			if len(l.src) > 1 && l.src[1] == '=' {
				l.emit(tokenGreaterOrEqual, 2)
				l.column += 2
			} else {
				l.emit(tokenGreater, 1)
				l.column++
			}
			endLineAsSemicolon = false
		case '(':
			l.emit(tokenLeftParenthesis, 1)
			l.column++
			endLineAsSemicolon = false
		case ')':
			l.emit(tokenRightParenthesis, 1)
			l.column++
			endLineAsSemicolon = true
		case '[':
			l.emit(tokenLeftBrackets, 1)
			l.column++
			endLineAsSemicolon = false
		case ']':
			l.emit(tokenRightBrackets, 1)
			l.column++
			endLineAsSemicolon = true
		case ':':
			l.emit(tokenColon, 1)
			l.column++
			endLineAsSemicolon = false
		case '}':
			if len(l.src) > 1 && l.src[1] == '}' {
				break LOOP
			}
			return l.errorf("unexpected }")
		case ',':
			l.emit(tokenComma, 1)
			l.column++
			endLineAsSemicolon = false
		case ' ', '\t', '\r':
			l.src = l.src[1:]
			l.column++
		case '\n':
			if endLineAsSemicolon {
				l.emit(tokenSemicolon, 1)
				endLineAsSemicolon = false
			} else {
				l.src = l.src[1:]
			}
			l.newline()
		case ';':
			l.emit(tokenSemicolon, 1)
			l.column++
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
// sapendo che src inizia con un carattere di s byte.
func (l *lexer) lexIdentifierOrKeyword(s int) bool {
	// si ferma solo quando un carattere
	// non può essere parte dell'identificatore o keyword
	cols := 1
	p := s
	for p < len(l.src) {
		r, s := utf8.DecodeRune(l.src[p:])
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}
		p += s
		cols++
	}
	switch string(l.src[0:p]) {
	case "break":
		l.emit(tokenBreak, p)
	case "continue":
		l.emit(tokenContinue, p)
	case "else":
		l.emit(tokenElse, p)
	case "end":
		l.emit(tokenEnd, p)
	case "extend":
		l.emit(tokenExtend, p)
	case "for":
		l.emit(tokenFor, p)
	case "if":
		l.emit(tokenIf, p)
	case "import":
		l.emit(tokenImport, p)
	case "in":
		l.emit(tokenIn, p)
	case "region":
		l.emit(tokenRegion, p)
	case "show":
		l.emit(tokenShow, p)
	case "var":
		l.emit(tokenVar, p)
	default:
		l.emit(tokenIdentifier, p)
		l.column += cols
		return true
	}
	l.column += cols
	return false
}

// lexNumber legge un numero (int o decimal) sapendo che src inizia con '0'..'9' o '.'.
func (l *lexer) lexNumber() {
	// si ferma solo se un carattere non può essere parte del numero
	hasDot := l.src[0] == '.'
	p := 1
	for p < len(l.src) {
		if l.src[p] == '.' {
			if hasDot {
				if l.src[p-1] == '.' {
					// il punto è parte di un token range
					p--
					hasDot = false
				}
				break
			}
			hasDot = true
		} else if l.src[p] < '0' || '9' < l.src[p] {
			break
		}
		p++
	}
	l.emit(tokenNumber, p)
	l.column += p
}

// lexInterpretedString legge una stringa "..." sapendo che src inizia con '"'.
func (l *lexer) lexInterpretedString() error {
	// si ferma quando trova il carattere '"' e ritorna errore
	// quando trova un carattere Unicode che non è valido in una stringa
	cols := 1
	p := 1
LOOP:
	for {
		if p == len(l.src) {
			return l.errorf("not closed string literal")
		}
		switch l.src[p] {
		case '"':
			break LOOP
		case '\\':
			if p+1 == len(l.src) {
				return l.errorf("not closed string literal")
			}
			switch c := l.src[p+1]; c {
			case 'u', 'U':
				var n = 4
				if c == 'U' {
					n = 8
				}
				if p+1+n >= len(l.src) {
					return l.errorf("not closed string literal")
				}
				var r uint32
				for i := 0; i < n; i++ {
					r = r * 16
					c = l.src[p+2+i]
					switch {
					case '0' <= c && c <= '9':
						r += uint32(c - '0')
					case 'a' <= c && c <= 'f':
						r += uint32(c - 'a' + 10)
					case 'A' <= c && c <= 'F':
						r += uint32(c - 'A' + 10)
					default:
						l.src = l.src[p:]
						return l.errorf("invalid hex digit in string literal")
					}
				}
				if 0xD800 <= r && r < 0xE000 || r > '\U0010FFFF' {
					l.src = l.src[p:]
					return l.errorf("escape sequence is invalid Unicode code point")
				}
				p += 2 + n
				cols += 2 + n
			case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '"':
				p += 2
				cols += 2
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
			cols++
		}
	}
	l.emit(tokenInterpretedString, p+1)
	l.column += cols + 1
	return nil
}

// lexRawString legge una stringa `...` sapendo che src inizia con '`'.
func (l *lexer) lexRawString() error {
	// si ferma quando trova il carattere '`' e ritorna errore
	// quando trova un carattere unicode non valido in una stringa
	lin := l.line
	col := l.column
	p := 1
STRING:
	for {
		if p == len(l.src) {
			return l.errorf("not closed string literal")
		}
		switch l.src[p] {
		case '`':
			break STRING
		case '\n':
			p++
			l.newline()
		default:
			r, s := utf8.DecodeRune(l.src[p:])
			if r == utf8.RuneError {
				l.src = l.src[p:]
				return l.errorf("invalid byte in string literal")
			}
			p += s
			l.column++
		}
	}
	l.emitAtLineColumn(lin, col, tokenRawString, p+1)
	return nil
}
