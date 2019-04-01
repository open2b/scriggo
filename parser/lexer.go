// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"

	"scrigo/ast"
)

var cdataStart = []byte("<![CDATA[")
var cdataEnd = []byte("]]>")

// lexer maintains the scanner status.
type lexer struct {
	text   []byte      // text on which the scans are performed
	src    []byte      // slice of the text used during the scan
	line   int         // current line starting from 1
	column int         // current column starting from 1
	ctx    ast.Context // current context used during the scan
	tag    string      // current tag
	attr   string      // current attribute
	tokens chan token  // tokens, is closed at the end of the scan
	err    error       // error, indicates if there was an error
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
		tokens: tokens,
	}
	go lex.scan()
	return lex
}

// drain drains the tokens. Called by the parser to terminate the lexer
// goroutine.
func (l *lexer) drain() {
	for range l.tokens {
	}
}

func (l *lexer) newline() {
	l.line++
	l.column = 1
}

func (l *lexer) errorf(format string, args ...interface{}) *SyntaxError {
	return &SyntaxError{
		Path: "",
		Pos: ast.Position{
			Line:   l.line,
			Column: l.column,
			Start:  len(l.text) - len(l.src),
			End:    len(l.text) - len(l.src),
		},
		Err: fmt.Errorf(format, args...),
	}
}

// emit emits a token of type typ and length length at the current line and
// column.
func (l *lexer) emit(typ tokenType, length int) {
	l.emitAtLineColumn(l.line, l.column, typ, length)
}

// emitAtLineColumn emits a token of type typ and length length at a specific
// line and column.
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
	end := start + length - 1
	if length == 0 {
		if typ == tokenSemicolon {
			start--
		}
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
		tag: l.tag,
		att: l.attr,
	}
	if length > 0 {
		l.src = l.src[length:]
	}
}

// scan scans the text by placing the tokens on the tokens channel. If an
// error occurs, it puts the error in err, closes the channel and returns.
func (l *lexer) scan() {

	if l.ctx == ast.ContextNone {

		err := l.lexCode()
		if err != nil {
			l.err = err
		}

	} else {

		p := 0 // token length in bytes

		lin := l.line   // token line
		col := l.column // token column

		var quote = byte(0)
		var emittedURL bool

		isHTML := l.ctx == ast.ContextHTML

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
						l.err = err
						break LOOP
					}
					p = 0
					lin = l.line
					col = l.column
					continue
				}
			}

			switch l.ctx {

			case ast.ContextHTML:
				if c == '<' {
					// <![CDATA[...]]>
					if p+8 < len(l.src) && l.src[p+1] == '!' {
						if bytes.HasPrefix(l.src[p:], cdataStart) {
							// Skips the CDATA section.
							p += 6
							l.column += 6
							var t int
							if i := bytes.Index(l.src[p:], cdataEnd); i < 0 {
								t = len(l.src)
							} else {
								t = p + i + 2
							}
							for ; p < t; p++ {
								if c = l.src[p]; c == '\n' {
									l.newline()
								} else if isStartChar(c) {
									l.column++
								}
							}
							continue
						}
					}
					// Start tag.
					p++
					l.column++
					l.tag, p = l.scanTag(p)
					if l.tag != "" {
						l.ctx = ast.ContextTag
					}
					continue
				}

			case ast.ContextTag:
				if c == '>' || c == '/' && p < len(l.src) && l.src[p] == '>' {
					// End tag.
					switch l.tag {
					case "script":
						l.ctx = ast.ContextScript
					case "style":
						l.ctx = ast.ContextCSS
					default:
						l.ctx = ast.ContextHTML
					}
					l.tag = ""
					if c == '/' {
						p++
						l.column++
					}
				} else if !isASCIISpace(c) {
					// Checks if it is an attribute.
					var next int
					if l.attr, next = l.scanAttribute(p); next > p {
						p = next
						if l.attr != "" && p < len(l.src) {
							// Start attribute value.
							if c := l.src[p]; c == '"' || c == '\'' {
								quote = c
								p++
								l.column++
							}
							if containsURL(l.tag, l.attr) {
								l.emitAtLineColumn(lin, col, tokenText, p)
								if quote == 0 {
									l.ctx = ast.ContextUnquotedAttribute
								} else {
									l.ctx = ast.ContextAttribute
								}
								l.emit(tokenStartURL, 0)
								emittedURL = true
								p = 0
								lin = l.line
								col = l.column
								continue
							} else {
								if quote == 0 {
									l.ctx = ast.ContextUnquotedAttribute
								} else {
									l.ctx = ast.ContextAttribute
								}
							}
						}
						continue
					}
				}

			case ast.ContextAttribute, ast.ContextUnquotedAttribute:
				if l.ctx == ast.ContextAttribute && c == quote ||
					l.ctx == ast.ContextUnquotedAttribute && (c == '>' || isASCIISpace(c)) {
					// End attribute.
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
					l.ctx = ast.ContextTag
					l.attr = ""
				}

			case ast.ContextCSS:
				if isHTML && c == '<' && isEndStyle(l.src[p:]) {
					// </style>
					l.ctx = ast.ContextHTML
					p += 7
					l.column += 7
				} else if c == '"' || c == '\'' {
					l.ctx = ast.ContextCSSString
					quote = c
				}

			case ast.ContextCSSString:
				switch c {
				case '\\':
					if p+1 < len(l.src) && l.src[p+1] == quote {
						p++
						l.column++
					}
				case quote:
					l.ctx = ast.ContextCSS
					quote = 0
				case '<':
					if isHTML && isEndStyle(l.src[p:]) {
						l.ctx = ast.ContextHTML
						quote = 0
						p += 7
						l.column += 7
					}
				}

			case ast.ContextScript:
				if isHTML && c == '<' && isEndScript(l.src[p:]) {
					// </script>
					l.ctx = ast.ContextHTML
					p += 8
					l.column += 8
				} else if c == '"' || c == '\'' {
					l.ctx = ast.ContextScriptString
					quote = c
				}

			case ast.ContextScriptString:
				switch c {
				case '\\':
					if p+1 < len(l.src) && l.src[p+1] == quote {
						p++
						l.column++
					}
				case quote:
					l.ctx = ast.ContextScript
					quote = 0
				case '<':
					if isHTML && isEndScript(l.src[p:]) {
						l.ctx = ast.ContextHTML
						quote = 0
						p += 8
						l.column += 8
					}
				}

			}

			p++
			if c == '\n' {
				l.newline()
				continue
			}
			if isStartChar(c) {
				l.column++
			}

		}

		if len(l.src) > 0 {
			l.emitAtLineColumn(lin, col, tokenText, p)
		}

	}

	if l.err == nil {
		l.emit(tokenEOF, 0)
	}

	l.text = nil
	l.src = nil

	close(l.tokens)
}

// containsURL indicates if the attribute attr of tag contains an URL or a
// comma-separated list of URL.
//
// See https://www.w3.org/TR/2017/REC-html52-20171214/fullindex.html#attributes-table.
func containsURL(tag string, attr string) bool {
	switch attr {
	case "action":
		return tag == "form"
	case "cite":
		switch tag {
		case "blockquote", "del", "ins", "q":
			return true
		}
	case "data":
		return tag == "object"
	case "formaction":
		return attr == "button" || attr == "input"
	case "href":
		switch tag {
		case "a", "area", "link", "base":
			return true
		}
	case "longdesc":
		return tag == "img"
	case "manifest":
		return tag == "html"
	case "poster":
		return tag == "video"
	case "src":
		switch tag {
		case "audio", "embed", "iframe", "img", "input", "script", "source", "track", "video":
			return true
		}
	case "srcset":
		return tag == "img" || tag == "source"
	}
	return false
}

// scanTag scans a tag name from src starting from position p and returns the
// tag and the next position.
//
// For example, if l.src[p:] is `script src="...`, it returns "script" and
// p+6.
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
		} else if c == '>' || c == '/' || c == '{' || isASCIISpace(c) {
			break
		} else if c == '\n' {
			l.newline()
		} else {
			return "", p
		}
	}
	return string(bytes.ToLower(l.src[s:p])), p
}

// scanAttribute scans an attribute from src starting from position p and
// returns the attribute name and the next position to scan.
//
// For example, if l.src[p:] is
//    - `src="a"` it returns "src" and p+4
//    - `src=a` it returns "src" and p+4
//    - `src>` it returns "" and p+3
//    - `src img` it returns "" and p+4.
//    - `,` it returns "" and p.
func (l *lexer) scanAttribute(p int) (string, int) {
	// Reads the attribute name.
	s := p
	for ; p < len(l.src); p++ {
		c := l.src[p]
		if c == '=' || isASCIISpace(c) {
			break
		}
		const DEL = 0x7F
		if c <= 0x1F || c == '"' || c == '\'' || c == '>' || c == '/' || c == DEL {
			return "", p
		}
		if c >= 0x80 {
			r, size := utf8.DecodeRune(l.src[p:])
			if r == utf8.RuneError && size == 1 {
				return "", p
			}
			if 0x7F <= r && r <= 0x9F || unicode.Is(unicode.Noncharacter_Code_Point, r) {
				return "", p
			}
			p += size - 1
		}
		l.column++
	}
	if p == s || p == len(l.src) {
		return "", p
	}
	name := string(bytes.ToLower(l.src[s:p]))
	// Reads '='.
	for ; p < len(l.src); p++ {
		if c := l.src[p]; c == '=' {
			p++
			l.column++
			break
		} else if isASCIISpace(c) {
			if c == '\n' {
				l.newline()
			} else {
				l.column++
			}
		} else {
			return "", p
		}
	}
	if p == len(l.src) {
		return "", p
	}
	// Reads the quote.
	for ; p < len(l.src); p++ {
		if c := l.src[p]; c == '>' {
			return "", p
		} else if isASCIISpace(c) {
			if c == '\n' {
				l.newline()
			} else {
				l.column++
			}
		} else {
			break
		}
	}
	if p == len(l.src) {
		return "", p
	}
	return name, p
}

// isEndStyle indicates if s is the start or end of "style" tag.
func isEndStyle(s []byte) bool {
	return len(s) >= 8 && s[0] == '<' && s[1] == '/' && (s[7] == '>' || isSpace(s[7])) &&
		(s[2] == 's' || s[2] == 'S') && (s[3] == 't' || s[3] == 'T') && (s[4] == 'y' || s[4] == 'Y') &&
		(s[5] == 'l' || s[5] == 'L') && (s[6] == 'e' || s[6] == 'E')
}

// isEndScript indicates if s is the start or end of "script" tag.
func isEndScript(s []byte) bool {
	return len(s) >= 9 && s[0] == '<' && s[1] == '/' && (s[8] == '>' || isSpace(s[8])) &&
		(s[2] == 's' || s[2] == 'S') && (s[3] == 'c' || s[3] == 'C') && (s[4] == 'r' || s[4] == 'R') &&
		(s[5] == 'i' || s[5] == 'I') && (s[6] == 'p' || s[6] == 'P') && (s[7] == 't' || s[7] == 'T')
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
	// endLineAsSemicolon indicates if "\n" should be treated as ";".
	var endLineAsSemicolon = false
	// unclosedLeftBraces is the number of left braces lexed without a
	// corresponding right brace.
	var unclosedLeftBraces = 0
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
		case '\'':
			err := l.lexRuneLiteral()
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
			} else if l.src[1] == '.' && len(l.src) > 2 && l.src[2] == '.' {
				l.emit(tokenEllipses, 3)
				l.column += 3
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
				l.emit(tokenSimpleAssignment, 1)
				l.column++
			} else {
				l.emit(tokenEqual, 2)
				l.column += 2
			}
			endLineAsSemicolon = false
		case '+':
			if len(l.src) > 1 {
				switch l.src[1] {
				case '+':
					l.emit(tokenIncrement, 2)
					l.column += 2
					endLineAsSemicolon = true
					continue LOOP
				case '=':
					l.emit(tokenAdditionAssignment, 2)
					l.column += 2
					endLineAsSemicolon = false
					continue LOOP
				}
			}
			l.emit(tokenAddition, 1)
			l.column++
			endLineAsSemicolon = false
		case '-':
			if len(l.src) > 1 {
				switch l.src[1] {
				case '-':
					l.emit(tokenDecrement, 2)
					l.column += 2
					endLineAsSemicolon = true
					continue LOOP
				case '=':
					l.emit(tokenSubtractionAssignment, 2)
					l.column += 2
					endLineAsSemicolon = false
					continue LOOP
				}
			}
			l.emit(tokenSubtraction, 1)
			l.column++
			endLineAsSemicolon = false
		case '*':
			if len(l.src) > 1 && l.src[1] == '=' {
				l.emit(tokenMultiplicationAssignment, 2)
				l.column += 2
				endLineAsSemicolon = false
			} else {
				l.emit(tokenMultiplication, 1)
				l.column++
				endLineAsSemicolon = false
			}
		case '/':
			if len(l.src) > 1 && l.src[1] == '/' && l.ctx == ast.ContextNone {
				p := bytes.Index(l.src, []byte("\n"))
				if p == -1 {
					break LOOP
				}
				l.src = l.src[p:]
				if endLineAsSemicolon {
					l.emit(tokenSemicolon, 0)
					endLineAsSemicolon = false
				}
				l.newline()
				l.src = l.src[1:]
				continue LOOP
			}
			if len(l.src) > 1 && l.src[1] == '*' && l.ctx == ast.ContextNone {
				p := bytes.Index(l.src, []byte("*/"))
				if p == -1 {
					return l.errorf("comment not terminated")
				}
				p += 2
				nl := bytes.Index(l.src[2:p], []byte("\n"))
				l.src = l.src[p:]
				if nl >= 0 {
					if endLineAsSemicolon {
						l.emit(tokenSemicolon, 0)
						endLineAsSemicolon = false
					}
					l.newline()
				}
				continue LOOP
			}
			if len(l.src) > 1 && l.src[1] == '=' {
				l.emit(tokenDivisionAssignment, 2)
				l.column += 2
				endLineAsSemicolon = false
			} else {
				l.emit(tokenDivision, 1)
				l.column++
				endLineAsSemicolon = false
			}
		case '%':
			if len(l.src) > 1 {
				switch l.src[1] {
				case '}':
					if l.ctx != ast.ContextNone {
						break LOOP
					}
				case '=':
					l.emit(tokenModuloAssignment, 2)
					l.column += 2
					endLineAsSemicolon = false
					continue LOOP
				}
			}
			l.emit(tokenModulo, 1)
			l.column++
			endLineAsSemicolon = false
		case '&':
			if len(l.src) > 2 && l.src[1] == '&' {
				l.emit(tokenAnd, 2)
				l.column += 2
			} else {
				l.emit(tokenAmpersand, 1)
				l.column++
			}
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
		case '{':
			l.emit(tokenLeftBraces, 1)
			l.column++
			endLineAsSemicolon = false
			unclosedLeftBraces++
		case '}':
			if unclosedLeftBraces > 0 {
				if endLineAsSemicolon && l.ctx == ast.ContextNone {
					l.emit(tokenSemicolon, 0)
				}
				l.emit(tokenRightBraces, 1)
				l.column++
				endLineAsSemicolon = true
				unclosedLeftBraces--
			} else if l.ctx != ast.ContextNone && len(l.src) > 1 && l.src[1] == '}' {
				break LOOP
			} else {
				return l.errorf("unexpected }")
			}
		case ':':
			if len(l.src) > 1 && l.src[1] == '=' {
				l.emit(tokenDeclaration, 2)
				l.column += 2
			} else {
				l.emit(tokenColon, 1)
				l.column++
			}
			endLineAsSemicolon = false
		case ',':
			l.emit(tokenComma, 1)
			l.column++
			endLineAsSemicolon = false
		case ' ', '\t', '\r':
			l.src = l.src[1:]
			l.column++
		case '\n':
			if endLineAsSemicolon {
				l.emit(tokenSemicolon, 0)
				endLineAsSemicolon = false
			}
			l.newline()
			l.src = l.src[1:]
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
	if endLineAsSemicolon && l.ctx == ast.ContextNone {
		l.emit(tokenSemicolon, 0)
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

// isASCIISpace indicates if s if a space for the HTML spec.
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

// lexIdentifierOrKeyword reads an identifier or keyword knowing that src
// starts with a character with a length of s bytes.
func (l *lexer) lexIdentifierOrKeyword(s int) bool {
	// Stops only when a character can not be part
	// of the identifier or keyword.
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
	endLineAsSemicolon := false
	switch id := string(l.src[0:p]); id {
	case "break":
		l.emit(tokenBreak, p)
		endLineAsSemicolon = true
	case "case":
		l.emit(tokenCase, p)
	case "chan":
		l.emit(tokenChan, p)
	case "const":
		l.emit(tokenConst, p)
	case "continue":
		l.emit(tokenContinue, p)
		endLineAsSemicolon = true
	case "default":
		l.emit(tokenDefault, p)
	case "defer":
		l.emit(tokenDefer, p)
	case "else":
		l.emit(tokenElse, p)
	case "fallthrough":
		l.emit(tokenFallthrough, p)
		endLineAsSemicolon = true
	case "for":
		l.emit(tokenFor, p)
	case "func":
		l.emit(tokenFunc, p)
	case "go":
		l.emit(tokenGo, p)
	case "goto":
		l.emit(tokenGoto, p)
	case "if":
		l.emit(tokenIf, p)
	case "import":
		l.emit(tokenImport, p)
	case "interface":
		l.emit(tokenInterface, p)
	case "map":
		l.emit(tokenMap, p)
	case "package":
		l.emit(tokenPackage, p)
	case "range":
		l.emit(tokenRange, p)
	case "return":
		l.emit(tokenReturn, p)
		endLineAsSemicolon = true
	case "struct":
		l.emit(tokenStruct, p)
	case "select":
		l.emit(tokenSelect, p)
	case "switch":
		l.emit(tokenSwitch, p)
	case "type":
		l.emit(tokenSwitchType, p)
	case "var":
		l.emit(tokenVar, p)
	default:
		if l.ctx != ast.ContextNone {
			switch id {
			case "end":
				l.emit(tokenEnd, p)
			case "extends":
				l.emit(tokenExtends, p)
			case "in":
				l.emit(tokenIn, p)
			case "include":
				l.emit(tokenInclude, p)
			case "macro":
				l.emit(tokenMacro, p)
			case "show":
				l.emit(tokenShow, p)
			default:
				l.emit(tokenIdentifier, p)
				endLineAsSemicolon = true
			}
		} else {
			l.emit(tokenIdentifier, p)
			endLineAsSemicolon = true
		}
	}
	l.column += cols
	return endLineAsSemicolon
}

// lexNumber reads a number (integer or float) knowing that src starts with
// '0'..'9' or '.'.
func (l *lexer) lexNumber() {
	// Stops only if a character can not be part of the number.
	hasDot := l.src[0] == '.'
	p := 1
	for p < len(l.src) {
		if l.src[p] == '.' {
			if hasDot {
				if l.src[p-1] == '.' {
					// The point is part of a token range.
					p--
				}
				break
			}
			hasDot = true
		} else if l.src[p] < '0' || '9' < l.src[p] {
			break
		}
		p++
	}
	if hasDot {
		l.emit(tokenFloat, p)
	} else {
		l.emit(tokenInt, p)
	}
	l.column += p
}

// lexInterpretedString reads a string "..." knowing that src starts with '"'.
func (l *lexer) lexInterpretedString() error {
	// Stops when it finds the '"' character and returns an error when
	// it finds a Unicode character that is not valid in a string.
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
				var r rune
				for i := 0; i < n; i++ {
					r = r * 16
					c = l.src[p+2+i]
					switch {
					case '0' <= c && c <= '9':
						r += rune(c - '0')
					case 'a' <= c && c <= 'f':
						r += rune(c - 'a' + 10)
					case 'A' <= c && c <= 'F':
						r += rune(c - 'A' + 10)
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
			if r == utf8.RuneError && s == 1 {
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
	// Stops when it finds the '`' character and returns an error
	// when it finds an invalid Unicode character in a string.
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
			if r == utf8.RuneError && s == 1 {
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

// lexRuneLiteral reads a rune literal '...' knowing that src starts with "'".
func (l *lexer) lexRuneLiteral() error {
	// Stops when it finds the "'" character and returns an error when
	// it finds a Unicode character that is not valid in a rune literal.
	var p int
	if len(l.src) == 1 {
		return l.errorf("invalid character literal (missing closing ')")
	}
	switch l.src[1] {
	case '\\':
		if len(l.src) == 2 {
			return l.errorf("invalid character literal (missing closing ')")
		}
		switch c := l.src[2]; c {
		case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '\'':
			if p = 3; len(l.src) < p {
				return l.errorf("invalid character literal (missing closing ')")
			}
		case 'x':
			if p = 5; len(l.src) < p {
				return l.errorf("invalid character literal (missing closing ')")
			}
			for i := 3; i < 5; i++ {
				if !isHexDigit(l.src[i]) {
					return l.errorf("non-hex character in escape sequence: " + string(l.src[i]))
				}
			}
		case 'u', 'U':
			var n = 4
			if c == 'U' {
				n = 8
			}
			if p = n + 3; len(l.src) < p {
				return l.errorf("invalid character literal (missing closing ')")
			}
			var r uint32
			for i := 3; i < n+3; i++ {
				r = r * 16
				c = l.src[i]
				switch {
				case '0' <= c && c <= '9':
					r += uint32(c - '0')
				case 'a' <= c && c <= 'f':
					r += uint32(c - 'a' + 10)
				case 'A' <= c && c <= 'F':
					r += uint32(c - 'A' + 10)
				default:
					return l.errorf("non-hex character in escape sequence: %s", string(c))
				}
			}
			if 0xD800 <= r && r < 0xE000 || r > '\U0010FFFF' {
				return l.errorf("escape sequence is invalid Unicode code point")
			}
		case '0', '1', '2', '3', '4', '5', '6', '7':
			if p = 5; len(l.src) < p {
				return l.errorf("invalid character literal (missing closing ')")
			}
			r := rune(c - '0')
			for i := 3; i < 5; i++ {
				r = r * 8
				c = l.src[i]
				if c < '0' || c > '7' {
					return l.errorf("non-octal character in escape sequence: %s", string(c))
				}
				r += rune(c - '0')
			}
			if r > 255 {
				return l.errorf("octal escape value > 255: %d", r)
			}
		default:
			return l.errorf("invalid escape in string literal")
		}
	case '\n':
		return l.errorf("newline in character literal")
	case '\'':
		return l.errorf("empty character literal or unescaped ' in character literal")
	default:
		r, s := utf8.DecodeRune(l.src[1:])
		if r == utf8.RuneError && s == 1 {
			return l.errorf("invalid UTF-8 encoding")
		}
		p = s + 1
	}
	if len(l.src) <= p || l.src[p] != '\'' {
		return l.errorf("invalid character literal (missing closing ')")
	}
	l.emit(tokenRune, p+1)
	l.column += p + 1
	return nil
}

func isHexDigit(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}
