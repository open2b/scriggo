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

// lexer maintains the scanner status.
type lexer struct {
	text   []byte      // text on which the scans are performed
	src    []byte      // slice of the text used during the scan
	line   int         // current line starting from 1
	column int         // current column starting from 1
	ctx    ast.Context // current context used during the scan
	tokens chan token  // tokens, is closed at the end of the scan
	err    error       // error, at the end of the scan indicates if there was an error
}

// newLexer creates a new lexer.
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

// emit emits a token of type typ and length length to the current line and
// current column.
func (l *lexer) emit(typ tokenType, length int) {
	l.emitAtLineColumn(l.line, l.column, typ, length)
}

// emitAtLineColumn emits a token of type typ and length length to the line
// line and column column.
func (l *lexer) emitAtLineColumn(line, column int, typ tokenType, length int) {
	var txt []byte
	if length > 0 {
		txt = l.src[0:length]
	}
	ctx := l.ctx
	if typ == tokenText {
		ctx = ast.ContextText
	}
	start := len(l.text) - len(l.src)
	end := start + length -1
	if length == 0 {
		end = start
	}
	l.tokens <- token{
		typ: typ,
		pos: &ast.Position{
			Line:   line,
			Column: column,
			Start:  start,
			End:    end,
		},
		txt: txt,
		lin: l.line,
		ctx: ctx,
	}
	if length > 0 {
		l.src = l.src[length:]
	}
}

// scan scans the text by placing the tokens on the tokens channel.
// In the event of an error, it puts the error in err, closes the channel
// and exits.
func (l *lexer) scan() {

	p := 0 // token length in bytes

	lin := l.line   // token line
	col := l.column // token column

	var tag string
	var quote = byte(0)
	var emittedURL bool

	initialContext := l.ctx // initial context

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
		if tag != "" && quote == 0 && isAlpha(c) {
			// check if it is an attribute
			var attr string
			attr, p = l.scanAttribute(p)
			if attr != "" {
				quote = l.src[p-1]
				if containsURL(tag, attr) {
					l.emitAtLineColumn(lin, col, tokenText, p)
					l.ctx = ast.ContextAttribute
					l.emit(tokenStartURL, 0)
					emittedURL = true
					p = 0
					lin = l.line
					col = l.column
				} else {
					l.ctx = ast.ContextAttribute
				}
			}
			continue
		}
		if quote != 0 && c == quote {
			// end attribute
			quote = 0
			if emittedURL {
				if p > 0 {
					l.emitAtLineColumn(lin, col, tokenText, p)
				}
				l.emit(tokenEndURL, 0)
				emittedURL = false
				p = 0
				lin = l.line
				col = l.column
			}
			p++
			l.column++
			l.ctx = ast.ContextHTML
			continue
		}

		p++
		if c == '\n' {
			l.newline()
			continue
		}
		if isStartChar(c) {
			l.column++
		}

		if initialContext == ast.ContextHTML {

			switch l.ctx {

			case ast.ContextHTML:
				if tag == "" {
					if c == '<' {
						// <![CDATA[...]]>
						if p+10 < len(l.src) && l.src[p] == '!' {
							if bytes.HasPrefix(l.src[p-1:], cdataStart) {
								// skips the CDATA section
								p += 8
								l.column += 8
								var t int
								if i := bytes.Index(l.src[p:], cdataEnd); i < 0 {
									t = len(l.src)
								} else {
									t = p + i + 2
								}
								for ; p < t; p++ {
									if c := l.src[p]; c == '\n' {
										l.newline()
									} else if isStartChar(c) {
										l.column++
									}
								}
								continue
							}
						}
						// start tag
						tag, p = l.scanTag(p)
					}
				} else if c == '>' || c == '/' && p < len(l.src) && l.src[p] == '>' {
					// end tag
					switch tag {
					case "script":
						l.ctx = ast.ContextJavaScript
					case "style":
						l.ctx = ast.ContextCSS
					}
					tag = ""
					quote = 0
				}

			case ast.ContextCSS:
				// </style>
				if c == '<' && p+6 < len(l.src) && l.src[p] == '/' && isStyle(l.src[p+1:p+6]) && (l.src[p+6] == '>' || isSpace(l.src[p+6])) {
					l.ctx = ast.ContextHTML
					p += 6
					l.column += 6
				}

			case ast.ContextJavaScript:
				// </script>
				if c == '<' && p+7 < len(l.src) && l.src[p] == '/' && isScript(l.src[p+1:p+7]) && (l.src[p+7] == '>' || isSpace(l.src[p+7])) {
					l.ctx = ast.ContextHTML
					p += 7
					l.column += 7
				}

			}
		}
	}

	if len(l.src) > 0 {
		l.emitAtLineColumn(lin, col, tokenText, p)
	}

	l.emit(tokenEOF, 0)

	l.src = nil
	l.ctx = 0

	close(l.tokens)
}

func containsURL(tag string, attr string) bool {
	switch tag {
	case "a":
		return attr == "href"
	case "img":
		return attr == "src"
	case "script", "style":
		return attr == "src"
	case "form":
		return attr == "action"
	}
	return false
}

// scanTag scans a tag name from src starting from position p
// and returns the tag and the next position.
//
// For example, if l.src[p:] is `script src="...`, it returns
// "script" and p+6.
func (l *lexer) scanTag(p int) (string, int) {
	s := p
	for ; p < len(l.src); p++ {
		c := l.src[p]
		if isAlpha(c) {
			l.column++
		} else if isDigit(c) {
			l.column++
			if p == s {
				return "", p + 1
			}
		} else if c == '>' || c == '/' || isASCIISpace(c) {
			break
		} else if c == '\n' {
			l.newline()
		} else {
			return "", p
		}
	}
	return string(bytes.ToLower(l.src[s:p])), p
}

// scanAttribute scans an attribute from src starting from position p
// and returns the attribute name and the next position.
//
// For example, if l.src[p:] is `src="...`, it returns "src" and p+5.
func (l *lexer) scanAttribute(p int) (string, int) {
	// reads attribute name
	s := p
	for ; p < len(l.src); p++ {
		if c := l.src[p]; !isAlpha(c) {
			if !isSpace(c) && c != '=' {
				return "", p
			}
			break
		}
		l.column++
	}
	if p == len(l.src) {
		return "", p
	}
	name := string(bytes.ToLower(l.src[s:p]))
	// reads =
	for ; p < len(l.src); p++ {
		c := l.src[p]
		if !isSpace(c) {
			if c != '=' {
				return "", p
			}
			p++
			l.column++
			break
		}
		if c == '\n' {
			l.newline()
		} else {
			l.column++
		}
	}
	if p == len(l.src) {
		return "", p
	}
	// reads quote
	for ; p < len(l.src); p++ {
		c := l.src[p]
		if !isSpace(c) {
			if c != '"' && c != '\'' {
				return "", p
			}
			break
		}
		if c == '\n' {
			l.newline()
		} else {
			l.column++
		}
	}
	if p == len(l.src) {
		return "", p
	}
	l.column++
	return name, p + 1
}

func isScript(s []byte) bool {
	if len(s) < 6 {
		return false
	}
	if (s[0] == 's' || s[0] == 'S') && (s[1] == 'c' || s[1] == 'C') && (s[2] == 'r' || s[2] == 'R') &&
		(s[3] == 'i' || s[3] == 'I') && (s[4] == 'p' || s[4] == 'P') && (s[5] == 't' || s[5] == 'T') {
		return true
	}
	return false
}

func isStyle(s []byte) bool {
	if len(s) < 5 {
		return false
	}
	if (s[0] == 's' || s[0] == 'S') && (s[1] == 't' || s[1] == 'T') && (s[2] == 'y' || s[2] == 'Y') &&
		(s[3] == 'l' || s[3] == 'L') && (s[4] == 'e' || s[4] == 'E') {
		return true
	}
	return false
}

// lexShow emits tokens knowing that src starts with '{{'.
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

// lexStatement emits tokens of a statement knowing that src starts with '{%'.
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

// lexComment emits a comment token knowing that src starts with '{#'.
func (l *lexer) lexComment() error {
	p := bytes.Index(l.src[2:], []byte("#}"))
	if p == -1 {
		return l.errorf("unexpected EOF, expecting #}")
	}
	line := l.line
	column := l.column
	l.column += 4
	for i := 2; i < p+2; i++ {
		if c := l.src[i]; c == '\n' {
			l.newline()
		} else if isStartChar(c) {
			l.column++
		}
	}
	l.emitAtLineColumn(line, column, tokenComment, p+4)
	return nil
}

// lexCode emits code tokens.
func (l *lexer) lexCode() error {
	if len(l.src) == 0 {
		return nil
	}
	// endLineAsSemicolon indicates if "\n" should be treated as ";"
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

// isSpace indicates if s is a space.
func isSpace(s byte) bool {
	return s == ' ' || s == '\t' || s == '\n' || s == '\r'
}

// isStartChar indicates if b is the first byte of an UTF-8 encoded character.
func isStartChar(b byte) bool {
	return b < 128 || 191 < b
}

func isASCIISpace(s byte) bool {
	return s == ' ' || s == '\t' || s == '\n' || s == '\r' || s == '\f'
}

// isAlpha indicates if s is ASCII alpha.
func isAlpha(s byte) bool {
	return 'a' <= s && s <= 'z' || 'A' <= s && s <= 'Z'
}

// isDigit indicates if s is ASCII digit.
func isDigit(s byte) bool {
	return '0' <= s && s <= '9'
}

// lexIdentifierOrKeyword reads an identifier or keyword knowing that
// src starts with a character with a length of s bytes.
func (l *lexer) lexIdentifierOrKeyword(s int) bool {
	// stops only when a character can not be part
	// of the identifier or keyword
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

// lexNumber reads a number (int or decimal) knowing that src starts with
// '0'..'9' or '.'.
func (l *lexer) lexNumber() {
	// it stops only if a character can not be part of the number
	hasDot := l.src[0] == '.'
	p := 1
	for p < len(l.src) {
		if l.src[p] == '.' {
			if hasDot {
				if l.src[p-1] == '.' {
					// the point is part of a token range
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

// lexInterpretedString reads a string "..." knowing that src starts with '"'.
func (l *lexer) lexInterpretedString() error {
	// stops when it finds the '"' character and returns an error when
	// it finds a Unicode character that is not valid in a string
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

// lexRawString reads a string `...` knowing that src starts with '`'.
func (l *lexer) lexRawString() error {
	// stops when it finds the '`' character and returns an error
	// when it finds an invalid Unicode character in a string
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
