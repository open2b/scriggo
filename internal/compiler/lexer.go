// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/open2b/scriggo/ast"
)

const BOM rune = 0xfeff
const bomErrorMsg = "invalid BOM in the middle of the file"

// scanProgram scans a program file and returns a lexer.
func scanProgram(text []byte) *lexer {
	tokens := make(chan token, 20)
	lex := &lexer{
		text:   text,
		src:    text,
		line:   1,
		column: 1,
		ctx:    ast.ContextText,
		tokens: tokens,
	}
	go lex.scan()
	return lex
}

// scanScript scans a script file and returns a lexer.
func scanScript(text []byte) *lexer {
	tokens := make(chan token, 20)
	lex := &lexer{
		text:           text,
		src:            text,
		line:           1,
		column:         1,
		ctx:            ast.ContextText,
		tokens:         tokens,
		extendedSyntax: true,
	}
	go lex.scan()
	return lex
}

// scanTemplate scans a template file and returns a lexer.
func scanTemplate(text []byte, format ast.Format, noParseShow, dollarIdentifier bool) *lexer {
	tokens := make(chan token, 20)
	lex := &lexer{
		text:             text,
		src:              text,
		line:             1,
		column:           1,
		ctx:              ast.Context(format),
		tokens:           tokens,
		templateSyntax:   true,
		extendedSyntax:   true,
		dollarIdentifier: dollarIdentifier,
		noParseShow:      noParseShow,
	}
	lex.tag.ctx = ast.ContextHTML
	if lex.ctx == ast.ContextMarkdown {
		lex.tag.ctx = ast.ContextMarkdown
	}
	go lex.scan()
	return lex
}

// Tokens returns a channel to read the scanned tokens.
func (l *lexer) Tokens() <-chan token {
	return l.tokens
}

// error returns the last occurred error or nil if no error occurred.
func (l *lexer) error() error {
	return l.err
}

// Stop stops the lexing and closes the tokens channel.
func (l *lexer) Stop() {
	for range l.tokens {
	}
}

var cdataStart = []byte("<![CDATA[")
var cdataEnd = []byte("]]>")

var emptyMarker = []byte{}

// lexer maintains the scanner status.
type lexer struct {
	text     []byte        // text on which the scans are performed
	src      []byte        // slice of the text used during the scan
	line     int           // current line starting from 1
	column   int           // current column starting from 1
	ctx      ast.Context   // current context used during the scan
	contexts []ast.Context // contexts of blocks nested in a macro or using statement.
	tag      struct {      // current tag
		name  string      // name
		attr  string      // current attribute name
		index int         // index of first byte of the current attribute value in src
		ctx   ast.Context // context of the tag's content
	}
	rawMarker        []byte     // raw marker, not nil when a raw statement has been lexed
	tokens           chan token // tokens, is closed at the end of the scan
	lastTokenType    tokenTyp   // type of the last non-empty emitted token
	totals           int        // total number of emitted tokens, excluding automatically inserted semicolons
	err              error      // error, reports whether there was an error
	templateSyntax   bool       // support template syntax with tokens 'end', 'extends', 'in', 'macro', 'raw', 'render' and 'show'
	extendedSyntax   bool       // support extended syntax with tokens 'and', 'or', 'not' and 'contains' (also support 'dollar' but only if 'dollarIdentifier' is true)
	dollarIdentifier bool       // support the dollar identifier, only if 'extendedSyntax' is true
	noParseShow      bool       // do not parse the short show statement.
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
	l.totals++
	start := len(l.text) - len(l.src)
	end := start + length - 1
	if length == 0 {
		if typ == tokenSemicolon {
			start--
			l.totals--
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
	if l.templateSyntax {
		switch typ {
		case tokenRaw:
			if l.lastTokenType == tokenStartStatement {
				l.rawMarker = emptyMarker
			}
		case tokenIdentifier:
			if l.lastTokenType == tokenRaw && l.rawMarker != nil {
				l.rawMarker = txt
			}
		case tokenEnd:
			l.rawMarker = nil
		}
	}
	if length > 0 {
		l.lastTokenType = typ
		l.src = l.src[length:]
	}
}

var jsMimeType = []byte("text/javascript")
var jsonLDMimeType = []byte("application/ld+json")
var cssMimeType = []byte("text/css")
var moduleType = []byte("module")

// scan scans the text by placing the tokens on the tokens channel. If an
// error occurs, it puts the error in err, closes the channel and returns.
func (l *lexer) scan() {

	if l.templateSyntax {

		p := 0 // token length in bytes

		lin := l.line   // token line
		col := l.column // token column

		var quote = byte(0)
		var emittedURL bool

		fileContext := l.ctx

		// Indicates if the current line contains only spaces. Used only for Markdown context.
		spacesOnlyLine := true

		isHTML := l.ctx == ast.ContextHTML || l.ctx == ast.ContextMarkdown

		if l.ctx == ast.ContextMarkdown {
			p, l.ctx = l.scanCodeBlock(0)
		}

	LOOP:
		for p < len(l.src) {

			c := l.src[p]

			if l.ctx == ast.ContextMarkdown {
				spacesOnlyLine = spacesOnlyLine && isSpace(c)
				if c == '\\' {
					p++
					l.column++
					if p < len(l.src) && l.src[p] != '\n' && l.src[p] != 'h' {
						_, s := utf8.DecodeRune(l.src[p:])
						p += s
						l.column++
					}
					continue
				}
			}

			if c == '{' && p+1 < len(l.src) {
				switch l.src[p+1] {
				case '{':
					if l.noParseShow {
						break
					}
					if p > 0 {
						l.emitAtLineColumn(lin, col, tokenText, p)
						p = 0
					}
					err := l.lexShow()
					if err != nil {
						l.err = err
						break LOOP
					}
					lin = l.line
					col = l.column
					continue
				case '%':
					if p > 0 {
						l.emitAtLineColumn(lin, col, tokenText, p)
						p = 0
					}
					var err error
					if len(l.src) > 2 && l.src[2] == '%' {
						err = l.lexStatements()
					} else {
						err = l.lexStatement()
					}
					if err != nil {
						l.err = err
						break LOOP
					}
					lin = l.line
					col = l.column
					if l.rawMarker != nil {
						p = l.skipRawContent()
					}
					continue
				case '#':
					if p > 0 {
						l.emitAtLineColumn(lin, col, tokenText, p)
						p = 0
					}
					err := l.lexComment()
					if err != nil {
						l.err = err
						break LOOP
					}
					lin = l.line
					col = l.column
					continue
				}
			} else if c == '#' && p+1 < len(l.src) && l.src[p+1] == '}' {
				l.err = l.errorf("unexpected #}")
				break LOOP
			}

			switch l.ctx {

			case ast.ContextMarkdown:
				if emittedURL {
					if isMarkdownEndURL(l.src[p:]) {
						if p > 0 {
							l.emitAtLineColumn(lin, col, tokenText, p)
						}
						l.emit(tokenEndURL, 0)
						emittedURL = false
						p = 0
						lin = l.line
						col = l.column
					}
				} else if (p == 0 || !isAlpha(l.src[p-1])) && isMarkdownStartURL(l.src[p:]) {
					if p > 0 {
						l.emitAtLineColumn(lin, col, tokenText, p)
					}
					l.emit(tokenStartURL, 0)
					emittedURL = true
					if l.src[4] == 's' {
						p = 8 // https://
					} else {
						p = 7 // http://
					}
					lin = l.line
					col = l.column + p
					continue
				}
				fallthrough

			case ast.ContextHTML:
				if c == '<' {
					// <![CDATA[...]]>
					if l.ctx == ast.ContextHTML && p+8 < len(l.src) && l.src[p+1] == '!' {
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
							l.tag.ctx = ast.ContextJS
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
					l.tag.ctx = fileContext
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
									l.ctx = ast.ContextUnquotedAttr
								} else {
									l.ctx = ast.ContextQuotedAttr
								}
								l.emit(tokenStartURL, 0)
								emittedURL = true
								p = 0
								lin = l.line
								col = l.column
							} else {
								l.tag.index = p
								if quote == 0 {
									l.ctx = ast.ContextUnquotedAttr
								} else {
									l.ctx = ast.ContextQuotedAttr
								}
							}
						}
						continue
					}
				}

			case ast.ContextQuotedAttr, ast.ContextUnquotedAttr:
				if l.ctx == ast.ContextQuotedAttr && c == quote ||
					l.ctx == ast.ContextUnquotedAttr && (c == '>' || isASCIISpace(c)) {
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
					} else if l.tag.attr == "type" {
						switch l.tag.name {
						case "script":
							typ := l.src[l.tag.index:p]
							if bytes.Equal(typ, moduleType) {
								break
							}
							typ = bytes.TrimSpace(typ)
							if len(typ) > 0 {
								if bytes.EqualFold(typ, jsonLDMimeType) {
									l.tag.ctx = ast.ContextJSON
								} else if !bytes.EqualFold(typ, jsMimeType) {
									l.tag.ctx = fileContext
								}
							}
						case "style":
							if typ := bytes.TrimSpace(l.src[l.tag.index:p]); len(typ) > 0 {
								if !bytes.EqualFold(typ, cssMimeType) {
									l.tag.ctx = fileContext
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
					l.ctx = fileContext
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
						l.ctx = fileContext
						quote = 0
						p += 7
						l.column += 7
					}
				}

			case ast.ContextJS:
				if isHTML && c == '<' && isEndScript(l.src[p:]) {
					// </script>
					l.ctx = fileContext
					p += 8
					l.column += 8
				} else if c == '"' || c == '\'' {
					l.ctx = ast.ContextJSString
					quote = c
				}

			case ast.ContextJSString:
				switch c {
				case '\\':
					if p+1 < len(l.src) && l.src[p+1] == quote {
						p++
						l.column++
					}
				case quote:
					l.ctx = ast.ContextJS
					quote = 0
				case '<':
					if isHTML && isEndScript(l.src[p:]) {
						l.ctx = fileContext
						quote = 0
						p += 8
						l.column += 8
					}
				}

			case ast.ContextJSON:
				if isHTML && c == '<' && isEndScript(l.src[p:]) {
					// </script>
					l.ctx = fileContext
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
						l.ctx = fileContext
						quote = 0
						p += 8
						l.column += 8
					}
				}

			}

			p++
			if c == '\n' {
				l.newline()
				if p < len(l.src) && l.src[p] == '\r' {
					p++
				}
				switch l.ctx {
				case ast.ContextTabCodeBlock, ast.ContextSpacesCodeBlock:
					p, l.ctx = l.scanCodeBlock(p)
				case ast.ContextMarkdown:
					if spacesOnlyLine {
						p, l.ctx = l.scanCodeBlock(p)
					} else {
						spacesOnlyLine = true
					}
				}
				continue
			}
			if isStartChar(c) {
				l.column++
			}

		}

		if l.err == nil {
			if len(l.src) > 0 {
				l.emitAtLineColumn(lin, col, tokenText, p)
			}
			if l.ctx == ast.ContextMarkdown && emittedURL {
				l.emit(tokenEndURL, 0)
			}
		}

	} else {

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

		err := l.lexCode(tokenEOF)
		if err != nil {
			l.err = err
		}

	}

	if l.err == nil {
		l.emit(tokenEOF, 0)
	}

	l.text = nil
	l.src = nil

	close(l.tokens)
}

// scanCodeBlock scans a tab or four spaces that start a Markdown code block.
// It returns the next position and the next context. If there is no tab or
// spaces, it returns p and the Markdown context.
func (l *lexer) scanCodeBlock(p int) (int, ast.Context) {
	if p < len(l.src) {
		switch l.src[p] {
		case '\t':
			return p + 1, ast.ContextTabCodeBlock
		case ' ':
			if p+3 < len(l.src) && l.src[p+1] == ' ' && l.src[p+2] == ' ' && l.src[p+3] == ' ' {
				return p + 4, ast.ContextSpacesCodeBlock
			}
		}
	}
	return p, ast.ContextMarkdown
}

// containsURL reports whether the attribute attr of tag contains an URL or a
// comma-separated list of URL.
//
// As special cases, if attr
//   - is "xmlns" or has namespace "xmlns", containsURL returns true
//   - has another namespace, it is treated as if had no namespace
//   - has "data-" prefix, it is treated as if had no "data-" prefix
//     but if it contains "src", "url" or "uri", containsURL returns true
//
// See https://www.w3.org/TR/2017/REC-html52-20171214/fullindex.html#attributes-table.
func containsURL(tag string, attr string) bool {
	if p := strings.IndexByte(attr, ':'); p != -1 {
		if attr[:p] == "xmlns" {
			return true
		}
		attr = attr[p+1:]
	} else if strings.HasPrefix(attr, "data-") {
		attr = attr[5:]
		if strings.Contains(attr, "src") ||
			strings.Contains(attr, "url") ||
			strings.Contains(attr, "uri") {
			return true
		}
	}
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
	case "xmlns":
		return true
	}
	return false
}

// scanTag scans a tag name from src starting from position p and returns the
// tag and the next position.
//
// For example, if l.src[p:] is `script src="...`, it returns "script" and
// p+6.
func (l *lexer) scanTag(p int) (string, int) {
	if p == len(l.src) || !isAlpha(l.src[p]) {
		return "", p
	}
	l.column++
	s := p
	p++
	for p < len(l.src) {
		c := l.src[p]
		if c == '>' || c == '/' || isASCIISpace(c) || c == '{' {
			break
		}
		l.column++
		if c < utf8.RuneSelf {
			p++
			continue
		}
		_, size := utf8.DecodeRune(l.src[p:])
		p += size
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

var http = []byte("http://")
var https = []byte("https://")

// isMarkdownStartURL reports whether s is the start of an URL in markdown
// context.
func isMarkdownStartURL(s []byte) bool {
	return bytes.HasPrefix(s, https) || bytes.HasPrefix(s, http)
}

// isMarkdownEndURL reports whether s cannot be part of an URL in markdown
// context.
func isMarkdownEndURL(s []byte) bool {
	if len(s) == 0 {
		return true
	}
	c := s[0]
	if c == '.' || c == '?' {
		if len(s) == 1 {
			return true
		}
		c = s[1]
	}
	return isASCIISpace(c) || c == '<' || c == '!' || c == ',' || c == ';' || c == '.' || c == ':' || c == ')' ||
		c == '\'' || c == '"' || c == '~'
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
	l.emit(tokenLeftBraces, 2)
	l.column += 2
	err := l.lexCode(tokenRightBraces)
	if err != nil {
		return err
	}
	l.emit(tokenRightBraces, 2)
	l.column += 2
	return nil
}

// lexStatement emits the tokens of a statement knowing that src starts with
// {%.
func (l *lexer) lexStatement() error {
	l.emit(tokenStartStatement, 2)
	l.column += 2
	err := l.lexCode(tokenEndStatement)
	if err != nil {
		return err
	}
	l.emit(tokenEndStatement, 2)
	l.column += 2
	return nil
}

// lexStatements emits the tokens for statements knowing that src starts with
// {%%.
func (l *lexer) lexStatements() error {
	l.emit(tokenStartStatements, 3)
	l.column += 3
	err := l.lexCode(tokenEndStatements)
	if err != nil {
		return err
	}
	l.emit(tokenEndStatements, 3)
	l.column += 3
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

// lexCode emits code tokens returning as soon as encounters a token based on
// the given end parameter.
//
//   if end is tokenEOF, it returns when encounters tokenEOF
//
//   if end is tokenEndStatement or tokenEndStatements, it returns when
//   encounters tokenEOF, tokenEndStatement or tokenEndStatements
//
//   if end is tokenRightBraces, it returns when encounters tokenEOF,
//   tokenEndStatement, tokenEndStatements or tokenRightBraces
//
func (l *lexer) lexCode(end tokenTyp) error {
	if len(l.src) == 0 {
		if end != tokenEOF {
			return l.errorf("unexpected EOF, expecting %s", end)
		}
		return nil
	}
	// first is the index of the first token in code.
	var first = l.totals + 1
	// macroOrUsing indicates if it has lexed the macro keyword or the using keyword.
	var macroOrUsing bool
	// ident stores the index and the text of the last lexed identifier after a macro or a using keyword.
	var ident struct {
		index int
		txt   string
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
				p := bytes.IndexAny(l.src, "\n"+string(BOM))
				if p == -1 {
					break LOOP
				}
				if l.src[p] != '\n' {
					return l.errorf(bomErrorMsg)
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
				l.src = l.src[2:]
				p := bytes.Index(l.src, []byte("*/"))
				if p == -1 {
					return l.errorf("comment not terminated")
				}
				nl := bytes.IndexAny(l.src[:p], "\n"+string(BOM))
				if nl >= 0 && l.src[nl] != '\n' {
					return l.errorf(bomErrorMsg)
				}
				l.src = l.src[p+2:]
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
					switch end {
					case tokenEndStatement:
						// If a macro declaration with an explicit result type
						// or a using statement with a type has been lexed, set
						// the context.
						if ident.index == l.totals {
							for i, name := range formatTypeName {
								if name == ident.txt {
									l.ctx = ast.Context(i)
									break
								}
							}
						}
						return nil
					case tokenRightBraces, tokenEndStatements:
						return l.errorf("unexpected %%}, expecting %s", end)
					}
				case '%':
					switch end {
					case tokenEndStatements:
						if endLineAsSemicolon {
							l.emit(tokenSemicolon, 0)
						}
						return nil
					case tokenRightBraces, tokenEndStatement:
						return l.errorf("unexpected %%%%}, expecting %s", end)
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
			if l.extendedSyntax && l.dollarIdentifier {
				l.emit(tokenDollar, 1)
				l.column++
				endLineAsSemicolon = false
			} else {
				return l.errorf("invalid character U+0024 '$'")
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
			l.emit(tokenLeftBracket, 1)
			l.column++
			endLineAsSemicolon = false
		case ']':
			l.emit(tokenRightBracket, 1)
			l.column++
			endLineAsSemicolon = true
		case '{':
			l.emit(tokenLeftBrace, 1)
			l.column++
			endLineAsSemicolon = false
			if end == tokenRightBraces {
				unclosedLeftBraces++
			}
		case '}':
			if end == tokenRightBraces {
				if len(l.src) > 1 && l.src[1] == '}' {
					if unclosedLeftBraces == 0 || len(l.src) == 2 || l.src[2] != '}' {
						return nil
					}
				}
				if unclosedLeftBraces > 0 {
					unclosedLeftBraces--
				}
			}
			l.emit(tokenRightBrace, 1)
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
			// Lex keyword or identifier.
			var typ tokenTyp
			var txt string
			if c == '_' || c < utf8.RuneSelf && unicode.IsLetter(rune(c)) {
				typ, txt = l.lexIdentifierOrKeyword(1)
			} else {
				r, s := utf8.DecodeRune(l.src)
				if !unicode.IsLetter(r) {
					if r == BOM {
						if first := len(l.src) == len(l.text); first {
							l.src = l.src[utf8.RuneLen(BOM):]
							continue LOOP
						}
						return l.errorf(bomErrorMsg)
					}
					if unicode.IsDigit(r) {
						return l.errorf("identifier cannot begin with digit %U '%c'", r, r)
					}
					if unicode.IsPrint(r) {
						return l.errorf("invalid character %U '%c'", r, r)
					}
					return l.errorf("invalid character %#U", r)
				}
				typ, txt = l.lexIdentifierOrKeyword(s)
			}
			// If it has lexed the first token after '{%', checks whether to store or restore the context.
			if end == tokenEndStatement {
				if l.totals == first {
					switch typ {
					case tokenMacro:
						macroOrUsing = true
						l.contexts = append(l.contexts, l.ctx)
					case tokenEnd:
						if last := len(l.contexts) - 1; last >= 0 {
							l.ctx = l.contexts[last]
							l.contexts = l.contexts[:last]
						}
					case tokenIf, tokenFor, tokenSwitch, tokenSelect:
						if len(l.contexts) > 0 {
							l.contexts = append(l.contexts, l.ctx)
						}
					}
				} else if typ == tokenUsing {
					macroOrUsing = true
					l.contexts = append(l.contexts, l.ctx)
				} else if macroOrUsing && typ == tokenIdentifier && l.totals != first+1 {
					ident.index = l.totals
					ident.txt = txt
				}
			}
			endLineAsSemicolon = false
			switch typ {
			case tokenBreak, tokenContinue, tokenFallthrough, tokenReturn, tokenIdentifier:
				endLineAsSemicolon = true
			}
		}
	}
	if end != tokenEOF {
		return l.errorf("unexpected EOF, expecting %s", end)
	}
	if endLineAsSemicolon {
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

// lexIdentifierOrKeyword reads an identifier or keyword, knowing that src
// starts with a character with a length of s bytes, and returns the type and
// the text of the emitted token.
func (l *lexer) lexIdentifierOrKeyword(s int) (tokenTyp, string) {
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
	typ := tokenIdentifier
	id := string(l.src[0:p])
	switch id {
	case "break":
		typ = tokenBreak
	case "case":
		typ = tokenCase
	case "chan":
		typ = tokenChan
	case "const":
		typ = tokenConst
	case "continue":
		typ = tokenContinue
	case "default":
		typ = tokenDefault
	case "defer":
		typ = tokenDefer
	case "else":
		typ = tokenElse
	case "fallthrough":
		typ = tokenFallthrough
	case "for":
		typ = tokenFor
	case "func":
		typ = tokenFunc
	case "go":
		typ = tokenGo
	case "goto":
		typ = tokenGoto
	case "if":
		typ = tokenIf
	case "import":
		typ = tokenImport
	case "interface":
		typ = tokenInterface
	case "map":
		typ = tokenMap
	case "package":
		typ = tokenPackage
	case "range":
		typ = tokenRange
	case "return":
		typ = tokenReturn
	case "struct":
		typ = tokenStruct
	case "select":
		typ = tokenSelect
	case "switch":
		typ = tokenSwitch
	case "type":
		typ = tokenType
	case "var":
		typ = tokenVar
	}
	if l.templateSyntax && typ == tokenIdentifier {
		switch id {
		case "end":
			typ = tokenEnd
		case "extends":
			typ = tokenExtends
		case "in":
			typ = tokenIn
		case "macro":
			typ = tokenMacro
		case "raw":
			typ = tokenRaw
		case "render":
			typ = tokenRender
		case "show":
			typ = tokenShow
		case "using":
			typ = tokenUsing
		}
	}
	if l.extendedSyntax && typ == tokenIdentifier {
		switch id {
		case "and":
			typ = tokenExtendedAnd
		case "contains":
			typ = tokenContains
		case "or":
			typ = tokenExtendedOr
		case "not":
			typ = tokenExtendedNot
		}
	}
	l.emit(typ, p)
	l.column += cols
	return typ, id
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
			} else if r == BOM {
				return l.errorf(bomErrorMsg)
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
	l.column++
	p := 1
STRING:
	for {
		if p == len(l.src) {
			return l.errorf("string not terminated")
		}
		switch l.src[p] {
		case '`':
			l.column++
			break STRING
		case '\n':
			p++
			l.newline()
		default:
			r, s := utf8.DecodeRune(l.src[p:])
			if r == utf8.RuneError && s == 1 {
				l.src = l.src[p:]
				return l.errorf("invalid UTF-8 encoding")
			} else if r == BOM {
				return l.errorf(bomErrorMsg)
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
		return l.errorf("rune literal not terminated")
	}
	switch l.src[1] {
	case '\\':
		if len(l.src) == 2 {
			return l.errorf("rune literal not terminated")
		}
		switch c := l.src[2]; c {
		case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '\'':
			if p = 3; len(l.src) < p {
				return l.errorf("rune literal not terminated")
			}
		case 'x':
			if p = 5; len(l.src) < p {
				return l.errorf("rune literal not terminated")
			}
			for i := 3; i < 5; i++ {
				if !isHexDigit(l.src[i]) {
					return l.errorf("invalid character %q in hexadecimal escape", string(l.src[i]))
				}
			}
		case 'u', 'U':
			var n = 4
			if c == 'U' {
				n = 8
			}
			if p = n + 3; len(l.src) < p {
				return l.errorf("rune literal not terminated")
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
					return l.errorf("invalid character %q in hexadecimal escape", string(c))
				}
			}
			if 0xD800 <= r && r < 0xE000 || r > '\U0010FFFF' {
				return l.errorf("escape is invalid Unicode code point %#U", r)
			}
		case '0', '1', '2', '3', '4', '5', '6', '7':
			if p = 5; len(l.src) < p {
				return l.errorf("rune literal not terminated")
			}
			r := rune(c - '0')
			for i := 3; i < 5; i++ {
				r = r * 8
				c = l.src[i]
				if c < '0' || c > '7' {
					return l.errorf("invalid character %q in octal escape", string(c))
				}
				r += rune(c - '0')
			}
			if r > 255 {
				return l.errorf("octal escape value %d > 255", r)
			}
		default:
			return l.errorf("unknown escape")
		}
	case '\n':
		return l.errorf("newline in rune literal")
	case '\'':
		return l.errorf("empty character literal or unescaped ' in character literal")
	default:
		r, s := utf8.DecodeRune(l.src[1:])
		if r == utf8.RuneError && s == 1 {
			return l.errorf("invalid UTF-8 encoding")
		} else if r == BOM {
			return l.errorf(bomErrorMsg)
		}
		p = s + 1
	}
	if len(l.src) <= p || l.src[p] != '\'' {
		return l.errorf("rune literal not terminated")
	}
	l.emit(tokenRune, p+1)
	l.column += p + 1
	return nil
}

// skipRawContent skips the content of a raw statement, updating l.line and
// l.column, and returns the index of the end statement or EOF if there is no
// end statement.
//
// It expects that l.rawMarker is not nil and l.src starts with the raw
// content.
func (l *lexer) skipRawContent() int {
	p := endRawIndex(l.src, l.rawMarker)
	if p == 0 {
		return 0
	}
	if p == -1 {
		p = len(l.src)
	}
	for i := 0; i < p; i++ {
		if c := l.src[i]; c == '\n' {
			l.line++
			l.column = 1
		} else if isStartChar(c) {
			l.column++
		}
	}
	return p
}

// endRawIndex returns the index of the first instance of the end of a raw
// statement in src with the given marker, or -1 if it is not present.
// If the raw statement has no marker, marker's length is zero.
//
// It allows the syntax {% end marker %} and allows non-printable characters
// as spaces (see the skipRawSpaces function) for which the parser will still
// returns an error.
func endRawIndex(src []byte, marker []byte) int {
	for i := 0; i < len(src); i++ {
		j := bytes.IndexByte(src[i:], '{')
		if j == -1 {
			break
		}
		i += j
		p := i
		// Read '{%'.
		if len(src) < i+2 || src[i+1] != '%' {
			continue
		}
		i += 2
		i = skipRawSpaces(src, i)
		// Read 'end'.
		if len(src) < i+3 || src[i] != 'e' || src[i+1] != 'n' || src[i+2] != 'd' {
			i = p
			continue
		}
		i += 3
		i = skipRawSpaces(src, i)
		// Read 'raw'.
		if isSpace(src[i-1]) && len(src) >= i+3 && src[i] == 'r' && src[i+1] == 'a' && src[i+2] == 'w' {
			i += 3
			i = skipRawSpaces(src, i)
		}
		// Read the marker.
		if l := len(marker); l > 0 {
			if len(src) < i+l || !bytes.Equal(src[i:i+l], marker) {
				i = p
				continue
			}
			i += l
			i = skipRawSpaces(src, i)
		}
		// Read '%}'.
		if len(src) < i+2 || src[i] != '%' || src[i+1] != '}' {
			i = p
			continue
		}
		return p
	}
	return -1
}

// skipRawSpaces skips the raw spaces from src starting from p and returns the
// index of the first non-raw-space byte or the index of EOF if there are only
// raw spaces.
//
// A raw space is the space character U+0020, an invalid code point or a rune
// not defined as a Graphic by Unicode.
func skipRawSpaces(src []byte, p int) int {
	for p < len(src) {
		r, s := utf8.DecodeRune(src[p:])
		isRawSpace := r == ' ' || (r == utf8.RuneError && s == 1) || !unicode.IsGraphic(r)
		if !isRawSpace {
			break
		}
		p += s
	}
	return p
}
