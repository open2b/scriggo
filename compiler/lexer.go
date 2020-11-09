// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"unicode"
	"unicode/utf8"

	"github.com/open2b/scriggo/compiler/ast"
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
	tag    struct {    // current tag
		name  string      // name
		attr  string      // current attribute name
		index int         // index of first byte of the current attribute value in src
		ctx   ast.Context // context of the tag's content
	}
	tokens   chan token // tokens, is closed at the end of the scan
	err      error      // error, reports whether there was an error
	andOrNot bool       // support tokens 'and', 'or' and 'not'.
}

// newLexer creates a new lexer.
func newLexer(text []byte, ctx ast.Context, andOrNot bool) *lexer {
	if ctx == ast.ContextGo && andOrNot {
		panic("unexpected option andOrNot with context Go")
	}
	tokens := make(chan token, 20)
	lex := &lexer{
		text:     text,
		src:      text,
		line:     1,
		column:   1,
		ctx:      ctx,
		tokens:   tokens,
		andOrNot: andOrNot,
	}
	lex.tag.ctx = ast.ContextHTML
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

func (l *lexer) errorf(format string, a ...interface{}) *SyntaxError {
	pos := ast.Position{
		Line:   l.line,
		Column: l.column,
		Start:  len(l.text) - len(l.src),
		End:    len(l.text) - len(l.src),
	}
	return syntaxError(&pos, format, a...)
}

// emit emits a token of type typ and length length at the current line and
// column.
func (l *lexer) emit(typ tokenTyp, length int) {
	l.emitAtLineColumn(l.line, l.column, typ, length)
}

// emitAtLineColumn emits a token of type typ and length length at a specific
// line and column.
func (l *lexer) emitAtLineColumn(line, column int, typ tokenTyp, length int) {
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
		tag: l.tag.name,
		att: l.tag.attr,
	}
	if length > 0 {
		l.src = l.src[length:]
	}
}

var javaScriptMimeType = []byte("text/javascript")
var jsonLDMimeType = []byte("application/ld+json")
var cssMimeType = []byte("text/css")

// scan scans the text by placing the tokens on the tokens channel. If an
// error occurs, it puts the error in err, closes the channel and returns.
func (l *lexer) scan() {

	if l.ctx == ast.ContextGo {

		if len(l.src) > 1 {
			// Parse shebang line.
			if l.src[0] == '#' && l.src[1] == '!' {
				t := bytes.IndexByte(l.src, '\n')
				if t == -1 {
					t = len(l.src) - 1
				}
				l.emit(tokenShebangLine, t+1)
				l.line++
			}
		}

		err := l.lexCode(false)
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
					err := l.lexBlock()
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
			} else if c == '#' && p+1 < len(l.src) && l.src[p+1] == '}' {
				l.err = l.errorf("unexpected #}")
				break LOOP
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
					l.tag.name, p = l.scanTag(p)
					if l.tag.name != "" {
						l.ctx = ast.ContextTag
						switch l.tag.name {
						case "script":
							l.tag.ctx = ast.ContextJavaScript
						case "style":
							l.tag.ctx = ast.ContextCSS
						}
					}
					continue
				}

			case ast.ContextTag:
				if c == '>' || c == '/' && p < len(l.src) && l.src[p] == '>' {
					// End tag.
					l.ctx = l.tag.ctx
					l.tag.name = ""
					l.tag.ctx = ast.ContextHTML
					if c == '/' {
						p++
						l.column++
					}
				} else if !isASCIISpace(c) {
					// Check if it is an attribute.
					var next int
					if l.tag.attr, next = l.scanAttribute(p); next > p {
						p = next
						if l.tag.attr != "" && p < len(l.src) {
							// Start attribute value.
							if c := l.src[p]; c == '"' || c == '\'' {
								quote = c
								p++
								l.column++
							}
							if containsURL(l.tag.name, l.tag.attr) {
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
							} else {
								l.tag.index = p
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
					} else if (l.tag.name == "script" || l.tag.name == "style") && l.tag.attr == "type" {
						if typ := bytes.TrimSpace(l.src[l.tag.index:p]); len(typ) > 0 {
							if l.tag.name == "script" {
								if bytes.EqualFold(typ, jsonLDMimeType) {
									l.tag.ctx = ast.ContextJSON
								} else if !bytes.EqualFold(typ, javaScriptMimeType) {
									l.tag.ctx = ast.ContextHTML
								}
							} else {
								if !bytes.EqualFold(typ, cssMimeType) {
									l.tag.ctx = ast.ContextHTML
								}
							}
						}
					}
					l.ctx = ast.ContextTag
					l.tag.attr = ""
					l.tag.index = 0
					if c == '>' {
						continue
					}
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

			case ast.ContextJavaScript:
				if isHTML && c == '<' && isEndScript(l.src[p:]) {
					// </script>
					l.ctx = ast.ContextHTML
					p += 8
					l.column += 8
				} else if c == '"' || c == '\'' {
					l.ctx = ast.ContextJavaScriptString
					quote = c
				}

			case ast.ContextJavaScriptString:
				switch c {
				case '\\':
					if p+1 < len(l.src) && l.src[p+1] == quote {
						p++
						l.column++
					}
				case quote:
					l.ctx = ast.ContextJavaScript
					quote = 0
				case '<':
					if isHTML && isEndScript(l.src[p:]) {
						l.ctx = ast.ContextHTML
						quote = 0
						p += 8
						l.column += 8
					}
				}

			case ast.ContextJSON:
				if isHTML && c == '<' && isEndScript(l.src[p:]) {
					// </script>
					l.ctx = ast.ContextHTML
					p += 8
					l.column += 8
				} else if c == '"' {
					l.ctx = ast.ContextJSONString
					quote = '"'
				}

			case ast.ContextJSONString:
				switch c {
				case '\\':
					if p+1 < len(l.src) && l.src[p+1] == '"' {
						p++
						l.column++
					}
				case '"':
					l.ctx = ast.ContextJSON
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

		if l.err == nil && len(l.src) > 0 {
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

// containsURL reports whether the attribute attr of tag contains an URL or a
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
		return tag == "button" || tag == "input"
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
		} else if isDecDigit(c) {
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
		if c == '{' && p+1 < len(l.src) {
			switch l.src[p+1] {
			case '{', '%', '#':
				return "", p
			}
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

// isEndStyle reports whether s is the start or end of "style" tag.
func isEndStyle(s []byte) bool {
	return len(s) >= 8 && s[0] == '<' && s[1] == '/' && (s[7] == '>' || isSpace(s[7])) &&
		(s[2] == 's' || s[2] == 'S') && (s[3] == 't' || s[3] == 'T') && (s[4] == 'y' || s[4] == 'Y') &&
		(s[5] == 'l' || s[5] == 'L') && (s[6] == 'e' || s[6] == 'E')
}

// isEndScript reports whether s is the start or end of "script" tag.
func isEndScript(s []byte) bool {
	return len(s) >= 9 && s[0] == '<' && s[1] == '/' && (s[8] == '>' || isSpace(s[8])) &&
		(s[2] == 's' || s[2] == 'S') && (s[3] == 'c' || s[3] == 'C') && (s[4] == 'r' || s[4] == 'R') &&
		(s[5] == 'i' || s[5] == 'I') && (s[6] == 'p' || s[6] == 'P') && (s[7] == 't' || s[7] == 'T')
}

// lexShow emits tokens knowing that src starts with '{{'.
func (l *lexer) lexShow() error {
	l.emit(tokenStartValue, 2)
	l.column += 2
	err := l.lexCode(true)
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

// lexBlock emits the tokens of a block knowing that src starts with {%.
func (l *lexer) lexBlock() error {
	l.emit(tokenStartBlock, 2)
	l.column += 2
	err := l.lexCode(false)
	if err != nil {
		return err
	}
	if len(l.src) < 2 {
		return l.errorf("unexpected EOF, expecting %%}")
	} else if l.src[0] != '%' && l.src[1] != '}' {
		return l.errorf("unexpected %s, expecting %%}", l.src[:2])
	}
	l.emit(tokenEndBlock, 2)
	l.column += 2
	return nil
}

// lexComment emits a comment token knowing that src starts with '{#'.
func (l *lexer) lexComment() error {
	nested := 0
	p := 2
	for nested >= 0 {
		i := bytes.IndexByte(l.src[p:], '#')
		if i == -1 {
			return l.errorf("comment not terminated")
		}
		if i > 0 && l.src[p+i-1] == '{' {
			nested++
		} else if i < len(l.src)-p && l.src[p+i+1] == '}' {
			nested--
			p++
		}
		p += i + 1
	}
	line := l.line
	column := l.column
	l.column += 4
	for i := 2; i < p-2; i++ {
		if c := l.src[i]; c == '\n' {
			l.newline()
		} else if isStartChar(c) {
			l.column++
		}
	}
	l.emitAtLineColumn(line, column, tokenComment, p)
	return nil
}

// lexCode emits code tokens.
func (l *lexer) lexCode(isShow bool) error {
	if len(l.src) == 0 {
		return nil
	}
	// endLineAsSemicolon reports whether "\n" should be treated as ";".
	var endLineAsSemicolon = false
	// unclosedLeftBraces is the number of left braces lexed without a
	// corresponding right brace. It is updated only if isShow is true.
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
				err := l.lexNumber()
				if err != nil {
					return err
				}
				endLineAsSemicolon = true
			} else if l.src[1] == '.' && len(l.src) > 2 && l.src[2] == '.' {
				l.emit(tokenEllipsis, 3)
				l.column += 3
				endLineAsSemicolon = false
			} else {
				l.emit(tokenPeriod, 1)
				l.column++
				endLineAsSemicolon = false
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			err := l.lexNumber()
			if err != nil {
				return err
			}
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
			if len(l.src) > 1 && l.src[1] == '/' {
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
			if len(l.src) > 1 && l.src[1] == '*' {
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
					if l.ctx != ast.ContextGo {
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
			if len(l.src) > 1 {
				switch l.src[1] {
				case '&':
					l.emit(tokenAnd, 2)
					l.column += 2
					endLineAsSemicolon = false
					continue LOOP
				case '^':
					if len(l.src) > 2 && l.src[2] == '=' {
						l.emit(tokenAndNotAssignment, 3)
						l.column += 3
					} else {
						l.emit(tokenAndNot, 2)
						l.column += 2
					}
					endLineAsSemicolon = false
					continue LOOP
				case '=':
					l.emit(tokenAndAssignment, 2)
					l.column += 2
					endLineAsSemicolon = false
					continue LOOP
				}
			}
			l.emit(tokenAmpersand, 1)
			l.column++
			endLineAsSemicolon = false
		case '|':
			if len(l.src) > 1 && l.src[1] == '|' {
				l.emit(tokenOr, 2)
				l.column += 2
			} else if len(l.src) > 1 && l.src[1] == '=' {
				l.emit(tokenOrAssignment, 2)
				l.column += 2
			} else {
				l.emit(tokenVerticalBar, 1)
				l.column++
			}
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
			} else if len(l.src) > 1 && l.src[1] == '-' {
				l.emit(tokenArrow, 2)
				l.column += 2
			} else if len(l.src) > 1 && l.src[1] == '<' {
				if len(l.src) > 2 && l.src[2] == '=' {
					l.emit(tokenLeftShiftAssignment, 3)
					l.column += 3
				} else {
					l.emit(tokenLeftShift, 2)
					l.column += 2
				}
			} else {
				l.emit(tokenLess, 1)
				l.column++
			}
			endLineAsSemicolon = false
		case '>':
			if len(l.src) > 1 && l.src[1] == '=' {
				l.emit(tokenGreaterOrEqual, 2)
				l.column += 2
			} else if len(l.src) > 1 && l.src[1] == '>' {
				if len(l.src) > 2 && l.src[2] == '=' {
					l.emit(tokenRightShiftAssignment, 3)
					l.column += 3
				} else {
					l.emit(tokenRightShift, 2)
					l.column += 2
				}
			} else {
				l.emit(tokenGreater, 1)
				l.column++
			}
			endLineAsSemicolon = false
		case '$':
			if l.andOrNot {
				l.emit(tokenDollar, 1)
				l.column++
				endLineAsSemicolon = false
			}
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
			if isShow {
				unclosedLeftBraces++
			}
		case '}':
			if isShow {
				if len(l.src) > 1 && l.src[1] == '}' {
					if unclosedLeftBraces == 0 || len(l.src) == 2 || l.src[2] != '}' {
						break LOOP
					}
				}
				if unclosedLeftBraces > 0 {
					unclosedLeftBraces--
				}
			}
			l.emit(tokenRightBraces, 1)
			l.column++
			endLineAsSemicolon = true
		case '^':
			if len(l.src) > 1 && l.src[1] == '=' {
				l.emit(tokenXorAssignment, 2)
				l.column += 2
			} else {
				l.emit(tokenXor, 1)
				l.column++
			}
			endLineAsSemicolon = false
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
		case '\x00':
			return l.errorf("unexpected NUL in input")
		default:
			if c == '_' || c < utf8.RuneSelf && unicode.IsLetter(rune(c)) {
				endLineAsSemicolon = l.lexIdentifierOrKeyword(1)
			} else {
				r, s := utf8.DecodeRune(l.src)
				if !unicode.IsLetter(r) {
					if unicode.IsPrint(r) {
						return l.errorf("illegal character %U '%c'", r, r)
					}
					return l.errorf("illegal character %U", r)
				}
				endLineAsSemicolon = l.lexIdentifierOrKeyword(s)
			}
		}
	}
	if endLineAsSemicolon && l.ctx == ast.ContextGo {
		l.emit(tokenSemicolon, 0)
	}
	return nil
}

// isSpace reports whether s is a space.
func isSpace(s byte) bool {
	return s == ' ' || s == '\t' || s == '\n' || s == '\r'
}

// isStartChar reports whether b is the first byte of an UTF-8 encoded
// character.
func isStartChar(b byte) bool {
	return b < 128 || 191 < b
}

// isASCIISpace reports whether s if a space for the HTML spec.
func isASCIISpace(s byte) bool {
	return s == ' ' || s == '\t' || s == '\n' || s == '\r' || s == '\f'
}

// isAlpha reports whether s is ASCII alpha.
func isAlpha(s byte) bool {
	return 'a' <= s && s <= 'z' || 'A' <= s && s <= 'Z'
}

func isBinDigit(c byte) bool {
	return c == '0' || c == '1'
}

func isOctDigit(c byte) bool {
	return '0' <= c && c <= '7'
}

func isDecDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

func isHexDigit(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
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
		l.emit(tokenType, p)
	case "var":
		l.emit(tokenVar, p)
	default:
		if l.ctx != ast.ContextGo {
			switch id {
			case "end":
				l.emit(tokenEnd, p)
			case "extends":
				l.emit(tokenExtends, p)
			case "in":
				l.emit(tokenIn, p)
			case "macro":
				l.emit(tokenMacro, p)
			case "show":
				l.emit(tokenShow, p)
			default:
				emitted := false
				if l.andOrNot {
					switch id {
					case "and":
						l.emit(tokenRelaxedAnd, p)
						emitted = true
					case "contains":
						l.emit(tokenContains, p)
						emitted = true
					case "or":
						l.emit(tokenRelaxedOr, p)
						emitted = true
					case "not":
						l.emit(tokenRelaxedNot, p)
						emitted = true
					}
				}
				if !emitted {
					l.emit(tokenIdentifier, p)
					endLineAsSemicolon = true
				}
			}
		} else {
			l.emit(tokenIdentifier, p)
			endLineAsSemicolon = true
		}
	}
	l.column += cols
	return endLineAsSemicolon
}

var numberBaseName = map[int]string{
	2:  "binary",
	8:  "octal",
	16: "hexadecimal",
}

// lexNumber reads a number (integer, float or imaginary) knowing that src
// starts with '0'..'9' or '.'.
func (l *lexer) lexNumber() error {
	// Stops only if a character can not be part of the number.
	var dot bool
	var exponent bool
	p := 0
	base := 10
	if c := l.src[0]; c == '0' && len(l.src) > 1 {
		switch l.src[1] {
		case '.':
		case 'x', 'X':
			base = 16
			p = 2
		case 'o', 'O':
			base = 8
			p = 2
		case '_', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			base = 8
			p = 1
		case 'b', 'B':
			base = 2
			p = 2
		}
		if p < len(l.src) && l.src[p] == '_' {
			p++
		}
	} else if c == '.' {
		dot = true
		p = 1
	}
DIGITS:
	for p < len(l.src) {
		c := l.src[p]
		switch base {
		case 10:
			if !isDecDigit(c) {
				break DIGITS
			}
		case 16:
			if exponent && !isDecDigit(c) || !exponent && !isHexDigit(c) {
				if dot && !exponent {
					return l.errorf("hexadecimal mantissa requires a 'p' exponent")
				}
				break DIGITS
			}
		case 8:
			if !isOctDigit(c) {
				if (c == '8' || c == '9') && l.src[0] == '0' && l.src[1] != 'o' && l.src[1] != 'O' {
					// It could be an imaginary literal.
					base = 10
				} else if isHexDigit(c) {
					return l.errorf("invalid digit '%d' in octal literal", c)
				} else {
					break DIGITS
				}
			}
		case 2:
			if !isBinDigit(c) {
				if isHexDigit(c) {
					return l.errorf("invalid digit '%d' in binary literal", c)
				}
				break DIGITS
			}
		}
		p++
		if p < len(l.src) {
			switch l.src[p] {
			case '_':
				p++
				continue DIGITS
			case '.':
				if dot || exponent {
					break DIGITS
				}
				if base < 10 {
					return l.errorf("invalid radix point in " + numberBaseName[base] + " literal")
				}
				dot = true
				p++
				if p == len(l.src) {
					break DIGITS
				}
			}
			switch l.src[p] {
			case 'e', 'E':
				if exponent || base != 10 {
					break
				}
				exponent = true
				p++
				if p < len(l.src) && (l.src[p] == '+' || l.src[p] == '-') {
					p++
				}
			case 'p', 'P':
				if exponent || base != 16 {
					break
				}
				exponent = true
				p++
				if p < len(l.src) && (l.src[p] == '+' || l.src[p] == '-') {
					p++
				}
			}
		}
	}
	switch l.src[p-1] {
	case 'x', 'X', 'o', 'O', 'b', 'B':
		if p == 2 {
			return l.errorf(numberBaseName[base] + " literal has no digits")
		}
	case '_':
		return l.errorf("'_' must separate successive digits")
	case 'e', 'E':
		if base != 16 {
			return l.errorf("exponent has no digits")
		}
	case 'p', 'P', '+', '-':
		return l.errorf("exponent has no digits")
	}
	imaginary := p < len(l.src) && l.src[p] == 'i'
	if imaginary {
		p++
	} else if p > 0 && base == 10 && l.src[0] == '0' && !dot && !exponent {
		for _, c := range l.src[1:p] {
			if c == '8' || c == '9' {
				return l.errorf("invalid digit '%d' in octal literal", c)
			}
		}
	}
	if imaginary {
		l.emit(tokenImaginary, p)
	} else if dot || exponent {
		l.emit(tokenFloat, p)
	} else {
		l.emit(tokenInt, p)
	}
	l.column += p
	return nil
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
			return l.errorf("string not terminated")
		}
		switch l.src[p] {
		case '"':
			break LOOP
		case '\\':
			if p+1 == len(l.src) {
				return l.errorf("string not terminated")
			}
			switch c := l.src[p+1]; c {
			case 'u', 'U':
				var n = 4
				if c == 'U' {
					n = 8
				}
				if p+1+n >= len(l.src) {
					return l.errorf("string not terminated")
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
						return l.errorf("invalid character %q in hexadecimal escape", c)
					}
				}
				if 0xD800 <= r && r < 0xE000 || r > '\U0010FFFF' {
					l.src = l.src[p:]
					return l.errorf("escape is invalid Unicode code point U+%X", r)
				}
				p += 2 + n
				cols += 2 + n
			case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '"':
				p += 2
				cols += 2
			case 'x':
				for i := 0; i < 2; i++ {
					if p+2+i == len(l.src) {
						l.src = l.src[p:]
						return l.errorf("string not terminated")
					}
					if c := l.src[p+2+i]; !isHexDigit(c) {
						l.src = l.src[p:]
						return l.errorf("invalid character %q in hexadecimal escape", c)
					}
				}
				p += 4
				cols += 4
			case '0', '1', '2', '3', '4', '5', '6', '7':
				r := rune(c - '0')
				for i := 0; i < 2; i++ {
					if p+2+i == len(l.src) {
						l.src = l.src[p:]
						return l.errorf("string not terminated")
					}
					r = r * 8
					c = l.src[p+2+i]
					if c < '0' || c > '7' {
						l.src = l.src[p:]
						return l.errorf("invalid character %q in octal escape", c)
					}
					r += rune(c - '0')
				}
				if r > 255 {
					l.src = l.src[p:]
					return l.errorf("octal escape value %d > 255", r)
				}
				p += 4
				cols += 4
			default:
				l.src = l.src[p:]
				return l.errorf("unknown escape")
			}
		case '\n':
			l.src = l.src[p:]
			return l.errorf("newline in string")
		default:
			r, s := utf8.DecodeRune(l.src[p:])
			if r == utf8.RuneError && s == 1 {
				l.src = l.src[p:]
				return l.errorf("invalid UTF-8 encoding")
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
			return l.errorf("string not terminated")
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
				return l.errorf("invalid UTF-8 encoding")
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
