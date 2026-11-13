// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/open2b/scriggo/ast"
)

// BOM contains a Byte Order Mark.
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

// scanTemplate scans a template file and returns a lexer.
func scanTemplate(text []byte, format ast.Format, noParseShow bool) *lexer {
	tokens := make(chan token, 20)
	lex := &lexer{
		text:           text,
		src:            text,
		line:           1,
		column:         1,
		ctx:            ast.Context(format),
		tokens:         tokens,
		templateSyntax: true,
		noParseShow:    noParseShow,
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
var jsHTMLCommentStart = []byte("<!--")
var jsHTMLCommentEnd = []byte("-->")

var emptyMarker = []byte{}

// lexer maintains the scanner status.
type lexer struct {
	text   []byte      // text on which the scans are performed
	src    []byte      // slice of the text used during the scan
	line   int         // current line starting from 1
	column int         // current column starting from 1
	ctx    ast.Context // current context used during the scan
	base   ast.Context // context outside tags and script and style elements, as described in lexState
	blocks []lexBlock  // blocks of the statements being scanned
	code   *codeBlocks // braces of the code of the {%% %%} statement being scanned
	state  scanState   // scan state, in addition to the context
	tag    struct {    // current tag
		name    string      // name
		attr    string      // current attribute name
		index   int         // index of first byte of the current attribute value in src
		dynamic bool        // the current attribute value contains a show, a statement or a comment
		ctx     ast.Context // context of the tag's content
	}
	tokenLine      int        // line in which the parser places the last show, statement or comment
	cut            newlineCut // whether the parser cuts the newline after the statement or comment being scanned
	cutEnd         int        // index in src of the end of the spaces and newline that the parser cuts
	rawMarker      []byte     // raw marker, not nil when a raw statement has been lexed
	inRaw          bool       // whether the content of a raw statement is being scanned
	rawEnd         int        // index in src of the end of the raw content, if inRaw is true
	rawMarkdown    bool       // whether the raw content starts in Markdown, if inRaw is true
	tokens         chan token // tokens, is closed at the end of the scan
	lastTokenType  tokenTyp   // type of the last non-empty emitted token
	totals         int        // total number of emitted tokens, excluding automatically inserted semicolons
	err            error      // error, reports whether there was an error
	templateSyntax bool       // support template syntax.
	noParseShow    bool       // do not parse the short show statement.
}

// newlineCut reports whether the parser cuts the newline after a show,
// statement or comment alone in its line, as described in endLine.
type newlineCut uint8

const (
	cutUnknown newlineCut = iota // it is not known
	cutAlways                    // it cuts it, as after an else, default or end statement or a comment
	cutNever                     // it does not cut it, as after a case statement
)

// newline is called when the lexer encounters a new line.
func (l *lexer) newline() {
	l.line++
	l.column = 1
}

// errorf returns a syntax error at the current lexer position with a message
// formatted according to the format specifier.
func (l *lexer) errorf(format string, a ...any) *SyntaxError {
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
	} else if ctx == ast.ContextJSRegExp && l.state.jsClass {
		ctx = ast.ContextJSRegExpClass
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
	if l.code != nil {
		l.code.scan(typ)
	}
	if length > 0 {
		l.lastTokenType = typ
		l.src = l.src[length:]
		l.cutEnd = 0
	}
}

var cssMimeType = []byte("text/css")

type jsCommentState uint8

const (
	jsCommentNone jsCommentState = iota
	jsCommentLine
	jsCommentBlock
)

// jsParen is the kind of a JavaScript parenthesis.
type jsParen uint8

const (
	jsParenOther   jsParen = iota // a '/' after the ')' is a division
	jsParenControl                // condition of an if, while, for or with statement
	jsParenUnknown                // it is not known whether it is a condition
)

// jsName is the kind of a JavaScript identifier.
type jsName uint8

const (
	jsNameAny      jsName = iota // it can be a keyword
	jsNameProperty               // property name, as after '.', or private name
	jsNameUnknown                // it is not known whether it is a property name
)

// scanState is the state of the scan that, in addition to the context, is
// needed to determine the context of the text that follows.
type scanState struct {
	quote      byte           // quote of the current attribute value or string
	jsComment  jsCommentState // current JavaScript comment
	jsDivision bool           // a JavaScript '/' is a division, not a regular expression
	jsClass    bool           // in a class of a JavaScript regular expression
	jsBraces   []int          // depth of the braces in JavaScript template literal substitutions
	jsLineCode bool           // the current JavaScript line contains code before the current position
	jsClassic  bool           // the JavaScript code is in a classic script, in which "<!--" starts a comment
	typeAttr   bool           // the current tag has a type attribute

	// attrNameChar reports whether the text of the current tag could end
	// with a character of an attribute name, so that a following attribute
	// name could be joined to it.
	attrNameChar bool

	// jsClassStart reports whether, in a class of a JavaScript regular
	// expression, only shows, statements and comments follow the '['.
	// jsClassDynamic reports whether at least one of them follows it. If
	// they render nothing, a '^' after them would negate the class.
	jsClassStart   bool
	jsClassDynamic bool

	// jsParens contains the kinds of the unclosed JavaScript parentheses,
	// and jsNextParen the kind of a '(' that follows. After the ')' that
	// closes the condition of an if, while, for or with statement, a '/'
	// starts a regular expression. jsParensFloor is the number of the
	// parentheses that are unclosed at the beginning of the body of the
	// innermost for statement: from the second iteration on, a ')' that
	// closes one of them can close a different parenthesis.
	jsParens      []jsParen
	jsNextParen   jsParen
	jsParensFloor int

	// jsParensDeep reports whether, below jsParens, there can be other
	// unclosed parentheses of unknown kind, as after a for statement whose
	// iterations open or close parentheses.
	jsParensDeep bool

	// jsIdentEnd reports whether the code before the current position can
	// end with a character of an identifier, or with the decimal point of a
	// number, that an identifier or number after a show, statement or
	// comment would continue.
	jsIdentEnd bool

	// cutSpaces reports whether the text since the beginning of the body of
	// the innermost for statement, or since an else, default or end statement
	// alone in its line, only contains spaces, tabs, carriage returns and
	// newlines. The parser can cut them, at the end of the lines of the
	// statement and of the statements and comments that follow it, so from
	// the second iteration on the body can continue the text at the end of
	// the previous one, and the text after an else, default or end statement
	// can continue the text before the block or at the end of a branch. To
	// keep it simple, they are all considered cut, except the spaces, tabs and
	// carriage returns before the first other character, which the parser
	// does not cut.
	cutSpaces bool

	// jsNextName is the kind of a JavaScript identifier that follows, and
	// jsIdentName and jsIdentParen are the values of jsNextName and
	// jsNextParen at the start of the identifier being scanned. A property
	// name is not a keyword, and the '(' in "for await (" is the one of a
	// for statement. jsIdentJoined reports whether the identifier being
	// scanned can continue one before a show, statement or comment.
	jsNextName    jsName
	jsIdentName   jsName
	jsIdentParen  jsParen
	jsIdentJoined bool

	// jsDivisionUnknown, jsLineCodeUnknown and typeAttrUnknown report whether
	// jsDivision, jsLineCode and typeAttr are unknown, because they differ in
	// the branches of a statement. jsDivision is also unknown after a '}' that
	// can end a block or an expression.
	jsDivisionUnknown bool
	jsLineCodeUnknown bool
	typeAttrUnknown   bool
}

// clone returns a copy of s.
func (s scanState) clone() scanState {
	s.jsBraces = slices.Clone(s.jsBraces)
	s.jsParens = slices.Clone(s.jsParens)
	return s
}

// lexState is the state of the scan at a position.
type lexState struct {
	ctx    ast.Context
	tagCtx ast.Context // context of the content of the current tag
	state  scanState

	// base is the context outside tags and script and style elements, to
	// which the scan returns at the end of a tag or of such an element. It
	// is the context of the file or, in the body of a macro or using
	// statement with html or markdown type, the context of that type, as
	// in an html body of a Markdown file.
	//
	// A body with another type does not change it: a js, css or json body
	// is scanned as the content of a script or style element of the
	// content that encloses it, where it can be shown, so the end tag of
	// the element is recognized also in the body.
	base ast.Context
}

// joinLexStates returns the state at the end of a statement whose branches
// end in the states a and b. It returns false if the states cannot be
// joined. If they differ only in jsDivision or jsLineCode, the joined
// state has them unknown.
func joinLexStates(a, b lexState) (lexState, bool) {
	sa, sb := a.state, b.state
	if a.ctx != b.ctx || a.tagCtx != b.tagCtx || a.base != b.base || sa.quote != sb.quote ||
		sa.jsComment != sb.jsComment || sa.jsClass != sb.jsClass || !slices.Equal(sa.jsBraces, sb.jsBraces) {
		return lexState{}, false
	}
	if sa.jsDivision != sb.jsDivision || sb.jsDivisionUnknown {
		sa.jsDivision = false
		sa.jsDivisionUnknown = true
	}
	if sa.jsLineCode != sb.jsLineCode || sb.jsLineCodeUnknown {
		sa.jsLineCode = false
		sa.jsLineCodeUnknown = true
	}
	sa.jsClassic = sa.jsClassic && sb.jsClassic
	if sa.typeAttr != sb.typeAttr || sb.typeAttrUnknown {
		sa.typeAttr = false
		sa.typeAttrUnknown = true
	}
	sa.attrNameChar = sa.attrNameChar || sb.attrNameChar
	sa.jsClassStart = sa.jsClassStart || sb.jsClassStart
	sa.jsClassDynamic = sa.jsClassDynamic || sb.jsClassDynamic
	parens := joinJSParens(sa.jsParens, sb.jsParens)
	sa.jsParensFloor = max(alignJSParensFloor(sa, len(parens)), alignJSParensFloor(sb, len(parens)))
	sa.jsParens = parens
	if sa.jsNextParen != sb.jsNextParen {
		sa.jsNextParen = jsParenUnknown
	}
	sa.jsParensDeep = sa.jsParensDeep || sb.jsParensDeep
	sa.jsIdentEnd = sa.jsIdentEnd || sb.jsIdentEnd
	sa.cutSpaces = sa.cutSpaces || sb.cutSpaces
	sa.jsIdentJoined = sa.jsIdentJoined || sb.jsIdentJoined
	if sa.jsIdentParen != sb.jsIdentParen {
		sa.jsIdentParen = jsParenUnknown
	}
	if sa.jsNextName != sb.jsNextName {
		sa.jsNextName = jsNameUnknown
	}
	if sa.jsIdentName != sb.jsIdentName {
		sa.jsIdentName = jsNameUnknown
	}
	return lexState{ctx: a.ctx, tagCtx: a.tagCtx, state: sa.clone(), base: a.base}, true
}

// joinJSParens returns the unclosed JavaScript parentheses at the end of a
// statement whose branches end with the unclosed parentheses a and b. They
// are compared starting from the innermost, the one that a ')' closes, and
// a missing parenthesis is compared as jsParenOther, because a ')' with no
// unclosed parenthesis is followed by a division. The parentheses that
// differ have the kind jsParenUnknown.
func joinJSParens(a, b []jsParen) []jsParen {
	if slices.Equal(a, b) {
		return a
	}
	n := max(len(a), len(b))
	parens := make([]jsParen, n)
	for i := 1; i <= n; i++ {
		pa, pb := jsParenOther, jsParenOther
		if i <= len(a) {
			pa = a[len(a)-i]
		}
		if i <= len(b) {
			pb = b[len(b)-i]
		}
		if pa != pb {
			pa = jsParenUnknown
		}
		parens[n-i] = pa
	}
	return parens
}

// alignJSParensFloor returns the floor of the parentheses of s, aligned to n
// joined parentheses. The parentheses are joined starting from the
// innermost, so the floor is moved up by the number of the parentheses that
// s does not have. These are below the floor too, if it is not zero: a ')'
// that would close one of them has no parenthesis to close, and makes the
// code invalid.
func alignJSParensFloor(s scanState, n int) int {
	if s.jsParensFloor == 0 {
		return 0
	}
	return s.jsParensFloor + n - len(s.jsParens)
}

// blockKind is the kind of a block statement.
type blockKind uint8

const (
	blockIf blockKind = iota
	blockFor
	blockSwitch
	blockSelect
	blockMacro
	blockUsing
	blockRaw
)

// lexBlock represents a block statement being scanned.
type lexBlock struct {
	kind     blockKind
	start    lexState  // state at the beginning of the block
	end      *lexState // joined state at the end of the branches scanned so far
	implicit bool      // whether the block can be executed without executing any branch
	cases    bool      // whether a case has been scanned, for switch and select blocks
	body     lexState  // state at the beginning of the body, for macro and using blocks
	inPlace  bool      // whether the content of a using body is shown in place
	next     *lexState // state at the beginning of the next case, after a fallthrough
	label    string    // label of the statement, if any
	inElse   bool      // whether the else branch of a for statement is being scanned
	parens   bool      // whether an iteration of a for statement changes the JavaScript parentheses
}

// codeOwner is the statement or expression that owns a brace in the code of
// a {%% %%} statement.
type codeOwner uint8

const (
	ownerOther  codeOwner = iota // block of another statement, composite literal or type
	ownerFor                     // for statement
	ownerSwitch                  // switch or select statement
	ownerFunc                    // function literal
)

// codeBlocks tracks the braces in the code of a {%% %%} statement, so that
// a break, continue or return statement in a for statement, switch or select
// statement, or function literal of the code is not taken as a statement
// that ends a branch of a template statement or a macro or using body.
//
// The owner of a brace is approximated. A brace can be attributed to a for,
// switch or select statement or a function literal it is not part of only if
// it is a brace of a composite literal, as in "[]func(){}", which cannot
// contain statements outside function literals. When in doubt, a statement
// is attributed to the template, so that at worst the lexer reports a false
// error.
type codeBlocks struct {
	braces  []codeOwner   // owners of the unclosed braces
	parens  int           // number of unclosed parentheses and brackets
	pending []codePending // keywords waiting for their braces
	last    tokenTyp      // type of the last token
}

// codePending is a for, switch, select or func keyword waiting for its
// braces.
type codePending struct {
	owner  codeOwner
	braces int  // number of unclosed braces at the keyword
	parens int  // number of unclosed parentheses and brackets at the keyword
	closed bool // whether a brace owned by a for, switch or select is closed
}

// scan updates b with the token of type typ.
func (b *codeBlocks) scan(typ tokenTyp) {
	last := b.last
	b.last = typ
	if n := len(b.pending); n > 0 {
		p := b.pending[n-1]
		if p.braces == len(b.braces) && p.parens == b.parens {
			if p.owner == ownerFunc {
				// The signature of a function literal ends at its body.
				switch typ {
				case tokenIdentifier, tokenPeriod, tokenMultiplication, tokenLeftParenthesis,
					tokenLeftBracket, tokenLeftBrace, tokenChan, tokenArrow, tokenMap, tokenFunc,
					tokenStruct, tokenInterface:
				default:
					b.pending = b.pending[:n-1]
				}
			} else if p.closed && (typ == tokenSemicolon || typ == tokenElse) {
				// A for, switch or select statement ends at a semicolon
				// after its body. Its header can contain semicolons, as in
				// "for i := 0; i < n; i++ {", so the semicolon must follow a
				// closed brace. In "switch s := []int{1}; s[0] {", the brace
				// of the composite literal ends the statement early, so its
				// body is attributed to the template. The else branch of a
				// for statement is not part of the loop.
				b.pending = b.pending[:n-1]
			}
		}
	}
	switch typ {
	case tokenFor:
		b.pending = append(b.pending, codePending{owner: ownerFor, braces: len(b.braces), parens: b.parens})
	case tokenSwitch, tokenSelect:
		b.pending = append(b.pending, codePending{owner: ownerSwitch, braces: len(b.braces), parens: b.parens})
	case tokenFunc:
		b.pending = append(b.pending, codePending{owner: ownerFunc, braces: len(b.braces), parens: b.parens})
	case tokenLeftParenthesis, tokenLeftBracket:
		b.parens++
	case tokenRightParenthesis, tokenRightBracket:
		b.parens--
		b.dropPending()
	case tokenLeftBrace:
		// A for, switch or select statement owns all the braces at the level
		// of its keyword, including those of the composite literals in its
		// header, which cannot contain statements outside function literals.
		// A brace after struct or interface is part of a type.
		owner := ownerOther
		if n := len(b.pending); n > 0 && last != tokenStruct && last != tokenInterface {
			if p := b.pending[n-1]; p.braces == len(b.braces) && p.parens == b.parens {
				owner = p.owner
				if owner == ownerFunc {
					// In "func() func() {", the brace is the body of the
					// first function, and the signature of both ends.
					for n > 0 && b.pending[n-1] == p {
						n--
					}
					b.pending = b.pending[:n]
				}
			}
		}
		b.braces = append(b.braces, owner)
	case tokenRightBrace:
		owner := ownerOther
		if n := len(b.braces); n > 0 {
			owner = b.braces[n-1]
			b.braces = b.braces[:n-1]
		}
		b.dropPending()
		if n := len(b.pending); n > 0 {
			p := &b.pending[n-1]
			if p.owner == owner && p.owner != ownerFunc && p.braces == len(b.braces) && p.parens == b.parens {
				p.closed = true
			}
		}
	}
}

// dropPending drops the pending keywords inside parentheses, brackets or
// braces that are closed.
func (b *codeBlocks) dropPending() {
	n := len(b.pending)
	for n > 0 && (b.pending[n-1].braces > len(b.braces) || b.pending[n-1].parens > b.parens) {
		n--
	}
	b.pending = b.pending[:n]
}

// captures reports whether a statement of type typ, a break, continue or
// return statement, refers to a statement of the code. labeled reports
// whether a break or continue statement has a label.
func (b *codeBlocks) captures(typ tokenTyp, labeled bool) bool {
	for _, owner := range b.braces {
		switch owner {
		case ownerFunc:
			return true
		case ownerFor:
			// A label can refer to a template statement, so a statement with
			// a label is attributed to the template, even if the label is in
			// the code.
			if !labeled && (typ == tokenBreak || typ == tokenContinue) {
				return true
			}
		case ownerSwitch:
			if !labeled && typ == tokenBreak {
				return true
			}
		}
	}
	return false
}

// snapshot returns the current state of the scan.
func (l *lexer) snapshot() lexState {
	return lexState{ctx: l.ctx, tagCtx: l.tag.ctx, state: l.state.clone(), base: l.base}
}

// restore restores the state of the scan to s.
func (l *lexer) restore(s lexState) {
	l.ctx = s.ctx
	l.tag.ctx = s.tagCtx
	l.state = s.state.clone()
	l.base = s.base
}

// beginBlock begins a block of kind kind.
func (l *lexer) beginBlock(kind blockKind) {
	b := lexBlock{kind: kind, start: l.snapshot()}
	switch kind {
	case blockIf, blockFor, blockSwitch:
		// An if statement without else, a for statement with no
		// iterations, and a switch statement without default do not
		// execute any branch.
		b.implicit = true
	case blockMacro, blockUsing:
		// The body of a macro or using statement is scanned starting from
		// an empty state, except for a using body whose content is shown
		// in place, whose state is set by lexCode. lexCode also updates
		// the state at the beginning of the body at the end of the
		// statement, after the type of the macro or using statement.
		l.state = scanState{}
		b.body = l.snapshot()
	}
	if kind == blockFor {
		// From the second iteration on, the body starts in the state at the
		// end of the previous one, which is not known yet. A ')' that closes
		// a parenthesis already unclosed here is of unknown kind, and an
		// identifier at the beginning can continue one at the end. The kinds
		// of a following '(' and identifier are assumed to be the same as
		// here, and checked at the end of the iterations, or are unknown if
		// they are not the usual ones, as after "if" or '.'.
		l.state.jsDivisionUnknown = true
		l.state.jsLineCodeUnknown = true
		l.state.jsIdentEnd = true
		l.state.cutSpaces = true
		l.state.jsParensFloor = len(l.state.jsParens)
		if l.state.jsNextParen != jsParenOther {
			l.state.jsNextParen = jsParenUnknown
		}
		if l.state.jsNextName != jsNameAny {
			l.state.jsNextName = jsNameUnknown
		}
	}
	l.blocks = append(l.blocks, b)
}

// endBranch ends the current branch of the block b. col is the column of the
// keyword of the statement that ends the branch, used to report errors.
func (l *lexer) endBranch(b *lexBlock, col int) error {
	cur := l.snapshot()
	if b.end == nil {
		b.end = &cur
		return nil
	}
	end, ok := joinLexStates(*b.end, cur)
	if !ok {
		return l.errorAtColumnf(col, "%s branches end in different contexts", b.kind)
	}
	b.end = &end
	return nil
}

// endIteration ends a branch of the body of the for statement b, as
// endBranch does. If next is true, the next iteration can follow, so it
// checks that the state of the parentheses and of the property names is the
// one in which the body has been scanned. col is the column of the keyword
// of the statement that ends the branch, used to report errors.
func (l *lexer) endIteration(b *lexBlock, next bool, col int) error {
	start, end := b.start.state, l.state
	if next && (start.jsNextParen == jsParenOther && end.jsNextParen != jsParenOther ||
		start.jsNextName == jsNameAny && end.jsNextName != jsNameAny) {
		return l.errorAtColumnf(col, "%s branches end in different contexts", b.kind)
	}
	if next && (!slices.Equal(start.jsParens, end.jsParens) || start.jsParensDeep != end.jsParensDeep) {
		b.parens = true
	}
	// After the statement, the floor of the parentheses is the one of the
	// enclosing for statement, if any.
	floor := l.state.jsParensFloor
	l.state.jsParensFloor = min(floor, start.jsParensFloor)
	err := l.endBranch(b, col)
	l.state.jsParensFloor = floor
	return err
}

// lastBlock returns the innermost block of one of the given kinds, or nil
// if there is no such block.
func (l *lexer) lastBlock(kinds ...blockKind) *lexBlock {
	for i := len(l.blocks) - 1; i >= 0; i-- {
		kind := l.blocks[i].kind
		if slices.Contains(kinds, kind) {
			return &l.blocks[i]
		}
		if kind == blockMacro || kind == blockUsing {
			// A statement cannot refer to a block outside the body.
			break
		}
	}
	return nil
}

// nextIdentifier returns the identifier at the beginning of l.src, after
// spaces, tabs, carriage returns and general comments, or an empty string if
// there is no identifier.
func (l *lexer) nextIdentifier() string {
	s := bytes.TrimLeft(l.src, " \t\r")
	for bytes.HasPrefix(s, []byte("/*")) {
		i := bytes.Index(s[2:], []byte("*/"))
		if i < 0 {
			return ""
		}
		s = bytes.TrimLeft(s[i+4:], " \t\r")
	}
	n := 0
	for n < len(s) {
		r, size := utf8.DecodeRune(s[n:])
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}
		n += size
	}
	return string(s[:n])
}

// endBody checks that the scan is in a state in which the content of the
// macro or using body b can end. col is the column of the keyword of the
// statement that ends the body, used to report errors.
//
// The content is shown where the macro is called or where itea is shown,
// so it must end in the state in which it starts. Markdown code blocks are
// considered Markdown: a macro or using body cannot start in a code block,
// so a body that ends in a code block can only change how the following
// text is laid out, not how it is escaped.
//
// If the content of a using body is shown in place, the states in which it
// ends are joined, as for the branches of a statement.
func (l *lexer) endBody(b *lexBlock, col int) error {
	if _, ok := joinLexStates(markdownLexState(b.body), markdownLexState(l.snapshot())); !ok {
		return l.errorAtColumnf(col, "%s body ends in a different context", b.kind)
	}
	if b.inPlace {
		return l.endBranch(b, col)
	}
	return nil
}

// scanBlockStatement updates the state of the scan when the statement that
// starts with the keyword typ is scanned. elseIf reports whether typ is
// the 'if' keyword of an 'else if' statement. col is the column of the
// keyword.
func (l *lexer) scanBlockStatement(typ tokenTyp, elseIf bool, col int) error {
	if elseIf {
		if b := l.lastBlock(blockIf); b != nil {
			b.implicit = true
		}
		return nil
	}
	switch typ {
	case tokenShow:
		l.endShow()
	case tokenIf:
		l.beginBlock(blockIf)
	case tokenFor:
		l.beginBlock(blockFor)
	case tokenSwitch:
		l.beginBlock(blockSwitch)
	case tokenSelect:
		l.beginBlock(blockSelect)
	case tokenMacro:
		l.beginBlock(blockMacro)
	case tokenUsing:
		l.beginBlock(blockUsing)
	case tokenRaw:
		l.beginBlock(blockRaw)
	case tokenElse:
		n := len(l.blocks)
		if n == 0 {
			return nil
		}
		b := &l.blocks[n-1]
		if b.kind != blockIf && b.kind != blockFor {
			return nil
		}
		var err error
		if b.kind == blockFor {
			err = l.endIteration(b, true, col)
		} else {
			err = l.endBranch(b, col)
		}
		if err != nil {
			return err
		}
		l.restore(b.start)
		l.cut = cutAlways
		if b.kind == blockIf {
			b.implicit = false
		} else {
			b.inElse = true
		}
	case tokenCase, tokenDefault:
		n := len(l.blocks)
		if n == 0 {
			return nil
		}
		b := &l.blocks[n-1]
		if b.kind != blockSwitch && b.kind != blockSelect {
			return nil
		}
		start := b.start
		if b.next != nil {
			// The previous case ends with a fallthrough statement.
			start = *b.next
			b.next = nil
		} else if b.cases {
			if err := l.endBranch(b, col); err != nil {
				return err
			}
		}
		b.cases = true
		l.restore(start)
		l.cut = cutNever
		if typ == tokenDefault {
			b.implicit = false
			l.cut = cutAlways
		}
	case tokenBreak, tokenContinue:
		// A break or continue statement ends a branch of the innermost
		// statement it can refer to. With a label, it also ends a branch of
		// every statement up to the one with the label: Go exits the
		// statement with the label, while Scriggo currently exits only the
		// innermost one.
		kinds := []blockKind{blockFor}
		if typ == tokenBreak {
			kinds = append(kinds, blockSwitch, blockSelect)
		}
		label := l.nextIdentifier()
		for i := len(l.blocks) - 1; i >= 0; i-- {
			b := &l.blocks[i]
			if b.kind == blockMacro || b.kind == blockUsing {
				// A statement cannot refer to a block outside the body.
				break
			}
			if !slices.Contains(kinds, b.kind) {
				continue
			}
			if b.inElse {
				// The else branch of a for statement is not part of the loop,
				// so the statement refers to an enclosing statement.
				continue
			}
			var err error
			if b.kind == blockFor {
				err = l.endIteration(b, typ == tokenContinue, col)
			} else {
				err = l.endBranch(b, col)
			}
			if err != nil {
				return err
			}
			if label == "" || b.label == label {
				break
			}
		}
	case tokenReturn:
		// A return statement ends the content of a macro or using body.
		if b := l.lastBlock(blockMacro, blockUsing); b != nil {
			return l.endBody(b, col)
		}
	case tokenFallthrough:
		// The next case is scanned starting from the join of the state at
		// the beginning of the switch statement and the current state.
		if b := l.lastBlock(blockSwitch); b != nil {
			next, ok := joinLexStates(b.start, l.snapshot())
			if !ok {
				return l.errorAtColumnf(col, "%s branches end in different contexts", b.kind)
			}
			b.next = &next
		}
	case tokenEnd:
		n := len(l.blocks)
		if n == 0 {
			return nil
		}
		b := l.blocks[n-1]
		l.blocks = l.blocks[:n-1]
		switch b.kind {
		case blockMacro, blockUsing:
			if err := l.endBody(&b, col); err != nil {
				return err
			}
			// If the content is shown in place, in the context of the
			// statement, the scan continues from the joined state at the end
			// of the body. Otherwise, the content is shown as a value.
			if !b.inPlace || b.body.ctx != b.start.ctx {
				l.restore(b.start)
			} else {
				l.restore(*b.end)
			}
			l.cut = cutAlways
		case blockRaw:
		default:
			var err error
			if b.kind == blockFor && !b.inElse {
				err = l.endIteration(&b, true, col)
			} else {
				err = l.endBranch(&b, col)
			}
			if err != nil {
				return err
			}
			if b.implicit {
				end, ok := joinLexStates(*b.end, b.start)
				if !ok {
					return l.errorAtColumnf(col, "%s branches end in different contexts", b.kind)
				}
				b.end = &end
			}
			if b.kind == blockFor && b.end.ctx == ast.ContextTag && b.end.state.attrNameChar {
				// From the second iteration on, the body can join an attribute
				// name to the one at the end of the previous iteration, and so
				// form a type attribute.
				b.end.state.typeAttr = false
				b.end.state.typeAttrUnknown = true
			}
			l.restore(*b.end)
			if b.kind == blockFor && b.parens {
				// The parentheses depend on the number of iterations.
				for i := range l.state.jsParens {
					l.state.jsParens[i] = jsParenUnknown
				}
				l.state.jsParensDeep = true
			}
			l.cut = cutAlways
		}
	}
	return nil
}

// startsAloneInLine reports whether the show, statement or comment that
// starts at l.src[p] can be alone in its line, as the parser determines it:
// it is the first one placed in the line, and only spaces, tabs and carriage
// returns precede it in the line. The parser places a show or statement in
// the line in which it starts, and a comment in the line in which it ends,
// so a multiline comment is always the first one in its line.
func (l *lexer) startsAloneInLine(p int) bool {
	s := l.src[:p]
	i := bytes.LastIndexByte(s, '\n')
	for _, c := range s[i+1:] {
		if c != ' ' && c != '\t' && c != '\r' {
			return false
		}
	}
	return i >= 0 || l.tokenLine < l.line
}

// cutLength returns the length of the spaces, tabs and carriage returns
// followed by a newline at the beginning of s, which the parser cuts after a
// show, statement or comment alone in its line. It returns 0 if s does not
// start with them.
func cutLength(s []byte) int {
	for i, c := range s {
		switch c {
		case ' ', '\t', '\r':
		case '\n':
			return i + 1
		default:
			return 0
		}
	}
	return 0
}

// endLine is called after a show, statement or comment, and aloneBefore
// reports whether it can be alone in its line, as reported by
// startsAloneInLine.
//
// If it is alone in its line, the parser can cut the spaces and the newline
// that follow it, and l.cut reports whether it cuts them. The parser cuts
// them after an else, default or end statement, after which the scan
// resumes from another state, so the text after the line can continue a
// text that does not end with a newline: in "a{% if c %}\n{% end %}\nelse",
// if c is false, "a" is followed by "else", and in
// "// a{% if c %}\n{% else %}\n`", the comment continues with "`". It cuts
// them also after a comment, which can follow such a statement in the same
// line, as in "{% end %}{# a\nb #}\n", where the comment is in the next line.
// The parser does not cut them after a case statement.
//
// If the parser cuts them, they are not scanned in JavaScript code, and the
// spaces and newlines that follow are considered cut, because the parser can
// also cut the newlines after the shows, statements and comments alone in
// their lines that follow.
//
// If it is not known whether the parser cuts them, the scan continues from
// the text before the line, which ends with a newline, so a cut newline
// does not change the context, unless the newline before the line was cut
// too and the scan is in a line comment or a regular expression. In this
// case, it returns an error.
func (l *lexer) endLine(aloneBefore bool) error {
	cut := l.cut
	l.cut = cutUnknown
	n := 0
	if aloneBefore {
		n = cutLength(l.src)
	}
	if n == 0 {
		// It is not alone in its line.
		return nil
	}
	switch cut {
	case cutAlways:
		l.cutEnd = n
		l.state.cutSpaces = true
	case cutUnknown:
		if l.ctx == ast.ContextJS && l.state.jsComment == jsCommentLine {
			return l.errorf("JavaScript comment could continue on the next line")
		}
		if l.ctx == ast.ContextJSRegExp {
			return l.errorf("JavaScript regular expression could continue on the next line")
		}
	}
	return nil
}

// endShow updates the state after a show.
func (l *lexer) endShow() {
	switch l.ctx {
	case ast.ContextJS:
		if l.state.jsComment == jsCommentNone {
			l.endJSOperand()
		}
	case ast.ContextTag:
		// The shown value could contain attribute names, such as "type", or
		// continue the name of the previous attribute.
		if !l.state.typeAttr || l.state.attrNameChar {
			l.state.typeAttr = false
			l.state.typeAttrUnknown = true
		}
		l.state.attrNameChar = true
	}
}

// endJSOperand updates the state at the end of a JavaScript operand, such
// as a string, a template literal, a regular expression literal or a shown
// value: a following '/' is a division, and the line contains code.
func (l *lexer) endJSOperand() {
	l.state.jsDivision = true
	l.state.jsDivisionUnknown = false
	l.state.jsLineCode = true
	l.state.jsLineCodeUnknown = false
	l.state.jsNextParen = jsParenOther
	l.state.jsNextName = jsNameAny
	l.state.jsIdentEnd = false
	l.state.cutSpaces = false
}

// unclosedJSError returns an error if a JavaScript string, template literal,
// regular expression or block comment is not closed at the current
// position, the end of a script. A line comment does not need to be
// closed. Otherwise it returns nil.
func (l *lexer) unclosedJSError() error {
	var what string
	switch l.ctx {
	case ast.ContextJSString:
		what = "string"
		if l.state.quote == '`' {
			what = "template literal"
		}
	case ast.ContextJSRegExp:
		what = "regular expression"
	case ast.ContextJS:
		if l.state.jsComment == jsCommentBlock {
			what = "comment"
		} else if len(l.state.jsBraces) > 0 {
			what = "template literal"
		}
	}
	if what == "" {
		return nil
	}
	return l.errorf("unclosed JavaScript %s at the end of the script", what)
}

// markdownLexState returns s with the context of a Markdown code block
// replaced with the Markdown context.
func markdownLexState(s lexState) lexState {
	if s.ctx == ast.ContextTabCodeBlock || s.ctx == ast.ContextSpacesCodeBlock {
		s.ctx = ast.ContextMarkdown
	}
	return s
}

// errorAtColumnf is like errorf but reports the error at the column col of
// the current line.
func (l *lexer) errorAtColumnf(col int, format string, a ...any) *SyntaxError {
	err := l.errorf(format, a...)
	err.pos.Column = col
	return err
}

// String returns the keyword of the block kind.
func (k blockKind) String() string {
	switch k {
	case blockIf:
		return "if"
	case blockFor:
		return "for"
	case blockSwitch:
		return "switch"
	case blockSelect:
		return "select"
	case blockMacro:
		return "macro"
	case blockUsing:
		return "using"
	case blockRaw:
		return "raw"
	}
	panic("invalid block kind")
}

// scan scans the text by placing the tokens on the tokens channel. If an
// error occurs, it puts the error in err, closes the channel and returns.
func (l *lexer) scan() {

	if l.templateSyntax && len(l.src) > 1 {
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

	if l.templateSyntax {

		p := 0 // token length in bytes

		lin := l.line   // token line
		col := l.column // token column

		var emittedURL bool

		fileContext := l.ctx
		l.base = l.ctx

		// Indicates if the current line contains only spaces. Used only for Markdown context.
		spacesOnlyLine := true

		// ltShow reports whether the text before the last shows, in the
		// content of a script or style element, ends with '<'.
		var ltShow bool

		if l.ctx == ast.ContextMarkdown {
			p, l.ctx = l.scanCodeBlock(0)
		}

	LOOP:
		for p < len(l.src) {

			c := l.src[p]

			// isHTML reports whether the end tags of script and style
			// elements are recognized.
			isHTML := l.base == ast.ContextHTML || l.base == ast.ContextMarkdown

			if ltShow && (c != '{' || !l.isShowStatementOrCommentStart(p) || l.src[p+1] != '{') {
				// c follows the shows after a '<'. If they render nothing, a
				// '/' or a statement or comment could start an end tag.
				ltShow = false
				if name := l.rawTextElement(); name != "" {
					if c == '/' && l.couldBeEndTag(p+1, name) {
						l.err = l.errorf("end tag could close a %s element", name)
						break LOOP
					}
					if l.isShowStatementOrCommentStart(p) {
						l.err = l.errorf("'<' followed by a statement or comment could start a %s end tag", name)
						break LOOP
					}
				}
			}

			if l.ctx == ast.ContextMarkdown {
				spacesOnlyLine = spacesOnlyLine && isSpace(c)
				if c == '\\' {
					if l.inRaw && p+1 == l.rawEnd {
						// The backslash would escape the first character
						// rendered after the raw content.
						l.err = l.errorf("unfinished escape sequence in %s", l.ctx)
						break LOOP
					}
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

			// In the content of a raw statement, shows, statements and
			// comments are text.
			if c == '{' && p+1 < len(l.src) && !(l.inRaw && p < l.rawEnd) {
				if (l.ctx == ast.ContextQuotedAttr || l.ctx == ast.ContextUnquotedAttr) && l.isShowStatementOrCommentStart(p) {
					// The attribute value is not known at compile time, and
					// l.tag.index will no longer be valid.
					l.tag.dynamic = true
				}
				if l.state.jsClassStart && l.isShowStatementOrCommentStart(p) {
					l.state.jsClassDynamic = true
				}
				if l.ctx == ast.ContextJS && l.state.jsComment == jsCommentNone && p > 0 && l.src[p-1] == '.' &&
					l.isShowStatementOrCommentStart(p) {
					// If the '.' can be a decimal point, the digits after the
					// show, statement or comment can continue the number, as
					// in "1.{# c #}5.x", where x is a property name.
					spread := p >= 3 && l.src[p-2] == '.' && l.src[p-3] == '.'
					l.state.jsIdentEnd = l.state.jsNextName != jsNameProperty && !spread
				}
				var aloneBefore bool
				if l.isShowStatementOrCommentStart(p) {
					aloneBefore = l.startsAloneInLine(p)
					l.tokenLine = l.line
				}
				switch l.src[p+1] {
				case '{':
					if l.noParseShow {
						break
					}
					// A '<' before the show could start an end tag with the
					// text after it, if the show renders nothing. A '/' at
					// the start of a shown value is escaped.
					ltShow = l.rawTextElement() != "" && (p > 0 && l.src[p-1] == '<' || p == 0 && ltShow)
					if p > 0 {
						l.emitAtLineColumn(lin, col, tokenText, p)
						p = 0
					}
					err := l.lexShow()
					if err != nil {
						l.err = err
						break LOOP
					}
					l.endShow()
					if err := l.endLine(aloneBefore); err != nil {
						l.err = err
						break LOOP
					}
					lin = l.line
					col = l.column
					continue
				case '%':
					if l.inRaw {
						// End of the raw content. It must end in a context in
						// which it can start, or in a Markdown code block,
						// because the URLs in it are not emitted. If it
						// starts in Markdown, it must end in Markdown, because
						// the lexer does not recognize code spans and fenced
						// code blocks, so a tag shown in them, as in
						// `<script>`, would change the context.
						l.inRaw = false
						ok := false
						switch l.ctx {
						case ast.ContextMarkdown, ast.ContextTabCodeBlock, ast.ContextSpacesCodeBlock:
							ok = true
						case ast.ContextText, ast.ContextHTML, ast.ContextCSS, ast.ContextJS, ast.ContextJSON:
							ok = !l.rawMarkdown
						}
						if !ok {
							l.err = l.errorf("raw content cannot end in %s", l.ctx)
							break LOOP
						}
					}
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
					if err == nil {
						err = l.endLine(aloneBefore)
					}
					if err != nil {
						l.err = err
						break LOOP
					}
					lin = l.line
					col = l.column
					if l.rawMarker != nil {
						if l.ctx <= ast.ContextMarkdown {
							// The raw content is rendered as is, so it is
							// scanned as text.
							l.inRaw = true
							l.rawMarkdown = l.ctx == ast.ContextMarkdown
							l.rawEnd = endRawIndex(l.src, l.rawMarker)
							if l.rawEnd == -1 {
								l.rawEnd = len(l.src)
							}
						} else {
							// The parser reports an error.
							p = l.skipRawContent()
						}
					}
					continue
				case '#':
					if p > 0 {
						l.emitAtLineColumn(lin, col, tokenText, p)
						p = 0
					}
					line := l.line
					err := l.lexComment()
					if err == nil {
						if l.line > line {
							// The parser places a comment in the line in
							// which it ends, so a multiline comment is the
							// first one in its line.
							aloneBefore = true
							l.tokenLine = l.line
						}
						l.cut = cutAlways
						err = l.endLine(aloneBefore)
					}
					if err != nil {
						l.err = err
						break LOOP
					}
					lin = l.line
					col = l.column
					continue
				}
			} else if c == '#' && p+1 < len(l.src) && l.src[p+1] == '}' && !l.inRaw {
				l.err = l.errorf("unexpected #}")
				break LOOP
			} else if c == '\\' && l.isStringContext() && l.isShowStatementOrCommentStart(p+1) {
				// The backslash would escape the first character rendered by
				// the statement or comment that follows or, if it renders
				// nothing, the character after it, which the lexer cannot
				// determine here.
				l.err = l.errorf("unfinished escape sequence in %s", l.ctx)
				break LOOP
			} else if c == '<' && p+1 < len(l.src) {
				// Browsers end a script or style element at its end tag, also
				// in a string or a comment, so the end tag must be known at
				// compile time. This is also checked in JavaScript, JSON and
				// CSS files, which can be rendered in these elements.
				if name := l.rawTextElement(); name != "" {
					if l.src[p+1] == '/' && l.isDynamicEndTag(p+2, name) {
						l.err = l.errorf("end tag could close a %s element (write <\\/ for a literal </)", name)
						break LOOP
					}
					if l.isShowStatementOrCommentStart(p+1) && l.src[p+2] != '{' {
						// A statement or comment after '<' could be followed
						// by a '/' that starts an end tag.
						l.err = l.errorf("'<' followed by a statement or comment could start a %s end tag", name)
						break LOOP
					}
				}
			}

			// cut reports whether c can be cut by the parser at the beginning
			// of the body of a for statement, as described for cutSpaces.
			cut := false
			if l.state.cutSpaces {
				cut = c == ' ' || c == '\t' || c == '\r' || c == '\n'
				if !cut && p > 0 && (l.src[p-1] == ' ' || l.src[p-1] == '\t' || l.src[p-1] == '\r') {
					// The spaces before c are not cut, because c follows them
					// before a newline.
					l.state.jsIdentEnd = false
				}
				l.state.cutSpaces = cut
			}

			switch l.ctx {

			case ast.ContextMarkdown:
				// The content of a raw statement is a single text token, so
				// no URL is emitted in it.
				if emittedURL && !l.inRaw {
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
				} else if !l.inRaw && (p == 0 || !isAlpha(l.src[p-1])) && isMarkdownStartURL(l.src[p:]) {
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
							if l.inRaw && t > l.rawEnd {
								// The section continues after the raw
								// content, where the lexer scans the text as
								// HTML, and so escapes the shows as HTML.
								t = l.rawEnd
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
					// A tag name that starts or continues with a show,
					// statement or comment is not known at compile time.
					if l.isShowStatementOrCommentStart(p) {
						l.err = l.errorf("tag name could be script or style")
						break LOOP
					}
					l.tag.name, p = l.scanTag(p)
					if l.tag.name != "" && l.isShowStatementOrCommentStart(p) &&
						(strings.HasPrefix("script", l.tag.name) || strings.HasPrefix("style", l.tag.name)) {
						if l.tag.name == "script" || l.tag.name == "style" {
							// The show, statement or comment could continue the
							// name, as in <script{{ "x" }}>.
							l.err = l.errorf("tag name could continue after %s (add a space after it)", l.tag.name)
						} else {
							l.err = l.errorf("tag name could be script or style")
						}
						break LOOP
					}
					if l.tag.name != "" {
						l.ctx = ast.ContextTag
						l.state.jsClassic = l.tag.name == "script"
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
				// An attribute name that starts here could be joined to the
				// text before it, if it ends with a show or with an attribute
				// name followed by a statement or comment.
				joined := l.state.attrNameChar
				l.state.attrNameChar = false
				if c == '>' || c == '/' && p < len(l.src) && l.src[p] == '>' {
					// End tag.
					l.ctx = l.tag.ctx
					// The content of the element, such as a script, is
					// scanned starting from an empty state.
					l.state = scanState{jsClassic: l.state.jsClassic}
					l.tag.name = ""
					l.tag.ctx = l.base
					if c == '/' {
						p++
						l.column++
					}
				} else if !isASCIISpace(c) {
					if l.tag.name == "script" || l.tag.name == "style" {
						// The content of these elements is scanned according
						// to their type attribute, so attribute names must be
						// known at compile time.
						if c == '=' {
							l.err = l.errorf("'=' could start an attribute name or value")
							break LOOP
						}
						if joined && c != '/' {
							l.err = l.errorf("attribute name could be joined to the previous one")
							break LOOP
						}
					}
					// Check if it is an attribute.
					var next int
					var value bool
					if l.tag.attr, value, next = l.scanAttribute(p); next > p {
						p = next
						// An attribute name directly followed by a show,
						// statement or comment could continue after it.
						l.state.attrNameChar = !value && l.isShowStatementOrCommentStart(p) && !isASCIISpace(l.src[p-1])
						if l.tag.attr == "type" {
							if l.state.typeAttr {
								// Browsers ignore a repeated attribute.
								l.tag.attr = ""
							} else if l.state.typeAttrUnknown && value {
								// The attribute is repeated only in some
								// branches, so it is handled as if its value
								// were not known at compile time.
								l.tag.dynamic = true
							}
							l.state.typeAttr = true
							l.state.typeAttrUnknown = false
						}
						if value {
							// Start attribute value.
							if c := l.src[p]; c == '"' || c == '\'' {
								l.state.quote = c
								p++
								l.column++
							}
							// No URL is emitted in the content of a raw
							// statement, which is a single text token.
							if containsURL(l.tag.name, l.tag.attr) && !l.inRaw {
								l.emitAtLineColumn(lin, col, tokenText, p)
								if l.state.quote == 0 {
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
								if l.state.quote == 0 {
									l.ctx = ast.ContextUnquotedAttr
								} else {
									l.ctx = ast.ContextQuotedAttr
								}
							}
						} else {
							l.tag.attr = ""
						}
						continue
					}
				}

			case ast.ContextQuotedAttr, ast.ContextUnquotedAttr:
				if l.ctx == ast.ContextQuotedAttr && c == l.state.quote ||
					l.ctx == ast.ContextUnquotedAttr && (c == '>' || isASCIISpace(c)) {
					// End attribute.
					l.state.quote = 0
					if emittedURL && !l.inRaw {
						if p > 0 {
							l.emitAtLineColumn(lin, col, tokenText, p)
						}
						l.emit(tokenEndURL, 0)
						emittedURL = false
						p = 0
						lin = l.line
						col = l.column
					} else if l.tag.attr == "type" && l.tag.dynamic {
						// If the type is not known at compile time, the
						// content is scanned as if the attribute were missing,
						// but a script could be a module.
						l.state.jsClassic = false
					} else if l.tag.attr == "type" {
						switch l.tag.name {
						case "script":
							typ := bytes.TrimSpace(l.src[l.tag.index:p])
							if bytes.EqualFold(typ, []byte("module")) {
								l.state.jsClassic = false
							} else if len(typ) > 0 && !isJSMimeType(typ) {
								if isJSONMimeType(typ) {
									l.tag.ctx = ast.ContextJSON
								} else {
									l.tag.ctx = l.base
								}
							}
						case "style":
							if typ := bytes.TrimSpace(l.src[l.tag.index:p]); len(typ) > 0 {
								if !bytes.EqualFold(typ, cssMimeType) {
									l.tag.ctx = l.base
								}
							}
						}
					}
					l.ctx = ast.ContextTag
					l.tag.attr = ""
					l.tag.index = 0
					l.tag.dynamic = false
					if c == '>' {
						continue
					}
				}

			case ast.ContextCSS:
				if isHTML && c == '<' && isEndStyle(l.src[p:]) {
					// </style>
					l.ctx = l.base
					p += 7
					l.column += 7
				} else if c == '"' || c == '\'' {
					l.ctx = ast.ContextCSSString
					l.state.quote = c
				}

			case ast.ContextCSSString:
				switch c {
				case '\\':
					if p+1 < len(l.src) && (l.src[p+1] == l.state.quote || l.src[p+1] == '\\') {
						p++
						l.column++
					}
				case l.state.quote:
					l.ctx = ast.ContextCSS
					l.state.quote = 0
				case '<':
					if isHTML && isEndStyle(l.src[p:]) {
						l.ctx = l.base
						l.state.quote = 0
						p += 7
						l.column += 7
					}
				}

			case ast.ContextJS:
				if p < l.cutEnd {
					// Cut by the parser, as described in endLine.
					break
				}
				if c >= utf8.RuneSelf && !isStartChar(c) {
					// Continuation byte of a character already scanned.
					break
				}
				lineTerminator := c == '\n' || c == '\r' || c == 0xE2 && isJSLineTerminator(l.src[p:])
				code := l.state.jsComment == jsCommentNone && !isJSSpace(l.src[p:])
				// braceEnd reports whether c is a '}' that does not end a
				// template literal substitution, and parenUnknown whether c
				// is a ')' of a parenthesis of unknown kind.
				braceEnd := false
				parenUnknown := false
				if isHTML && c == '<' && isEndScript(l.src[p:]) {
					// </script>
					if err := l.unclosedJSError(); err != nil {
						l.err = err
						break LOOP
					}
					l.ctx = l.base
					l.state = scanState{}
					p += 8
					l.column += 8
				} else if l.state.jsComment == jsCommentLine {
					if lineTerminator {
						l.state.jsComment = jsCommentNone
					}
				} else if l.state.jsComment == jsCommentBlock {
					if c == '*' && p+1 < len(l.src) && l.src[p+1] == '/' {
						l.state.jsComment = jsCommentNone
						p++
						l.column++
					}
				} else if c == '/' && l.state.jsDivisionUnknown && p+1 < len(l.src) && l.src[p+1] != '/' && l.src[p+1] != '*' ||
					c == '-' && l.state.jsLineCodeUnknown && bytes.HasPrefix(l.src[p:], jsHTMLCommentEnd) ||
					c == '<' && !l.state.jsClassic && l.base != ast.ContextMarkdown && bytes.HasPrefix(l.src[p:], jsHTMLCommentStart) {
					// It is not known whether '/' starts a regular expression
					// or "-->" starts a comment, for example because the
					// branches of a statement end in different states or
					// because '/' follows a '}' that can end a block or an
					// expression, or whether "<!--" starts a comment, as it
					// does only in a classic script and not in a module.
					// "<!--" is not checked in Markdown, where code spans and
					// code blocks are not recognized.
					if c == '/' {
						l.err = l.errorf("'/' could start a division or a regular expression")
					} else if c == '<' {
						l.err = l.errorf("'<!--' could start a comment or be an operator")
					} else {
						l.err = l.errorf("'-->' could start a comment or be an operator")
					}
					break LOOP
				} else if c == '<' && bytes.HasPrefix(l.src[p:], jsHTMLCommentStart) ||
					c == '-' && !l.state.jsLineCode && bytes.HasPrefix(l.src[p:], jsHTMLCommentEnd) ||
					c == '#' && p+1 < len(l.src) && l.src[p+1] == '!' {
					// HTML-like comments and hashbang comments are treated as
					// line comments. "-->" is a comment only at the beginning of
					// a line, as in "n-->0" it is a decrement and a comparison.
					l.state.jsComment = jsCommentLine
				} else if c == '/' && p+1 < len(l.src) && l.src[p+1] == '/' {
					l.state.jsComment = jsCommentLine
					p++
					l.column++
				} else if c == '/' && p+1 < len(l.src) && l.src[p+1] == '*' {
					l.state.jsComment = jsCommentBlock
					p++
					l.column++
				} else if c == '/' {
					if l.state.jsDivision {
						l.state.jsDivision = false
					} else {
						l.ctx = ast.ContextJSRegExp
						l.state.jsClass = false
					}
				} else if c == '"' || c == '\'' {
					l.ctx = ast.ContextJSString
					l.state.quote = c
				} else if c == '`' {
					// Template literal.
					l.ctx = ast.ContextJSString
					l.state.quote = c
				} else if c == '{' {
					if n := len(l.state.jsBraces); n > 0 {
						l.state.jsBraces[n-1]++
					}
					l.state.jsDivision = false
				} else if c == '}' {
					n := len(l.state.jsBraces)
					if n > 0 && l.state.jsBraces[n-1] == 0 {
						// End of a template literal substitution.
						l.state.jsBraces = l.state.jsBraces[:n-1]
						l.ctx = ast.ContextJSString
						l.state.quote = '`'
					} else {
						if n > 0 {
							l.state.jsBraces[n-1]--
						}
						// The '}' can end a block, after which a '/' starts a
						// regular expression, or an expression, such as an
						// object literal or a function expression, after
						// which a '/' is a division.
						braceEnd = true
					}
					l.state.jsDivision = false
				} else if c == '(' {
					l.state.jsParens = append(l.state.jsParens, l.state.jsNextParen)
					l.state.jsDivision = false
				} else if c == ')' {
					// After the condition of an if, while, for or with
					// statement, a '/' starts a regular expression.
					paren := jsParenOther
					if l.state.jsParensDeep {
						paren = jsParenUnknown
					}
					if n := len(l.state.jsParens); n > 0 {
						paren = l.state.jsParens[n-1]
						l.state.jsParens = l.state.jsParens[:n-1]
						if n <= l.state.jsParensFloor {
							// From the second iteration of a for statement on,
							// it can close a different parenthesis.
							paren = jsParenUnknown
							l.state.jsParensFloor = n - 1
						}
					}
					l.state.jsDivision = paren == jsParenOther
					parenUnknown = paren == jsParenUnknown
				} else if (c == '+' || c == '-') && p > 0 && l.src[p-1] == c {
					// "++" and "--" are followed by a division, "+++" and
					// "---" by a regular expression, and so on.
					l.state.jsDivision = !l.state.jsDivision
				} else if code {
					l.state.jsDivision = isJSDivisionAfter(l.src[:p+1])
				}
				if (!code || l.state.jsComment != jsCommentNone) && !cut {
					l.state.jsIdentEnd = false
				}
				// "-->" starts a comment only at the beginning of a line.
				if lineTerminator && !cut {
					l.state.jsLineCode = false
					l.state.jsLineCodeUnknown = false
				} else if lineTerminator {
					// The parser can cut the line terminator, so the line can
					// continue, and it is not known whether it contains code.
					if l.state.jsLineCode {
						l.state.jsLineCode = false
						l.state.jsLineCodeUnknown = true
					}
				} else if code && l.state.jsComment == jsCommentNone {
					l.state.jsLineCode = true
					l.state.jsLineCodeUnknown = false
					l.state.jsDivisionUnknown = braceEnd || parenUnknown
					if isJSIdentifierPart(c) {
						l.scanJSIdentifier(p)
					} else {
						l.state.jsNextParen = jsParenOther
						l.state.jsNextName = jsNameAny
						switch c {
						case '.':
							l.state.jsNextName = l.jsNameAfterDot(p)
							if l.state.jsIdentEnd && (p == 0 || !endsWithJSIdentifierPart(l.src[:p])) {
								// The '.' can be the decimal point of a number
								// before a show, statement or comment.
								l.state.jsDivisionUnknown = true
							}
						case '#':
							// Private name.
							l.state.jsNextName = jsNameProperty
						}
						l.state.jsIdentEnd = false
					}
				}

			case ast.ContextJSString:
				switch c {
				case '\\':
					if p+1 < len(l.src) && (l.src[p+1] == l.state.quote || l.src[p+1] == '\\' ||
						l.state.quote == '`' && l.src[p+1] == '$') {
						p++
						l.column++
					}
				case l.state.quote:
					l.ctx = ast.ContextJS
					l.state.quote = 0
					l.endJSOperand()
				case '$':
					if l.state.quote == '`' && p+1 < len(l.src) && l.src[p+1] == '{' {
						if !l.isShowStatementOrCommentStart(p + 1) {
							// Start of a template literal substitution.
							l.state.jsBraces = append(l.state.jsBraces, 0)
							l.ctx = ast.ContextJS
							l.state.quote = 0
							l.state.jsDivision = false
							p++
							l.column++
						} else {
							// The show, statement or comment could render
							// nothing, so the text that follows could start
							// a substitution. A literal '$' can be written
							// as \$.
							l.err = l.errorf("unexpected show, statement or comment after $ in JavaScript template literal (write \\$ for a literal $)")
							break LOOP
						}
					}
				case '<':
					if isHTML && isEndScript(l.src[p:]) {
						l.err = l.unclosedJSError()
						break LOOP
					}
				}

			case ast.ContextJSRegExp:
				if p < l.cutEnd {
					// Cut by the parser, as described in endLine.
					break
				}
				if l.state.jsClassStart {
					if c == '^' && l.state.jsClassDynamic {
						// The shows, statements or comments before it could
						// render nothing.
						l.err = l.errorf("'^' could negate the class (write \\^ for a literal ^)")
						break LOOP
					}
					l.state.jsClassStart = false
					l.state.jsClassDynamic = false
				}
				switch c {
				case '\\':
					if p+1 < len(l.src) {
						switch l.src[p+1] {
						case '/', '\\', '[', ']':
							p++
							l.column++
						}
					}
				case '[':
					l.state.jsClass = true
					l.state.jsClassStart = true
				case ']':
					l.state.jsClass = false
				case '/':
					if !l.state.jsClass {
						l.ctx = ast.ContextJS
						l.endJSOperand()
					}
				case '\n', '\r', 0xE2:
					// A regular expression cannot span lines.
					if isJSLineTerminator(l.src[p:]) {
						l.ctx = ast.ContextJS
						l.state.jsClass = false
						l.state.jsDivision = false
					}
				case '<':
					if isHTML && isEndScript(l.src[p:]) {
						l.err = l.unclosedJSError()
						break LOOP
					}
				}

			case ast.ContextJSON:
				if isHTML && c == '<' && isEndScript(l.src[p:]) {
					// </script>
					l.ctx = l.base
					p += 8
					l.column += 8
				} else if c == '"' {
					l.ctx = ast.ContextJSONString
					l.state.quote = '"'
				}

			case ast.ContextJSONString:
				switch c {
				case '\\':
					if p+1 < len(l.src) && (l.src[p+1] == '"' || l.src[p+1] == '\\') {
						p++
						l.column++
					}
				case '"':
					l.ctx = ast.ContextJSON
					l.state.quote = 0
				case '<':
					if isHTML && isEndScript(l.src[p:]) {
						l.ctx = l.base
						l.state.quote = 0
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

		if l.err == nil && (fileContext == ast.ContextHTML || fileContext == ast.ContextJS) {
			// An HTML or JavaScript file cannot end in a JavaScript string,
			// template literal, regular expression or block comment. It is
			// not checked for Markdown, where the lexer does not recognize
			// code spans and code blocks.
			l.err = l.unclosedJSError()
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
// returns the attribute name, whether it has a value, and the next position
// to scan. As in HTML, the name ends at a space, '/', '>' or '=', unless '='
// is its first character. It also ends at a show, statement or comment.
//
// For example, if l.src[p:] is
//   - `src="a"` it returns "src", true and p+4
//   - `src=a` it returns "src", true and p+4
//   - `src>` it returns "src", false and p+3
//   - `src img` it returns "src", false and p+4.
//   - `a"b=c` it returns `a"b`, true and p+4.
//   - `/` it returns "", false and p.
func (l *lexer) scanAttribute(p int) (string, bool, int) {
	// Reads the attribute name.
	s := p
	for ; p < len(l.src); p++ {
		c := l.src[p]
		if c == '=' && p > s || c == '>' || c == '/' || isASCIISpace(c) || l.isShowStatementOrCommentStart(p) {
			break
		}
		if c >= utf8.RuneSelf {
			_, size := utf8.DecodeRune(l.src[p:])
			p += size - 1
		}
		l.column++
	}
	if p == s || p == len(l.src) {
		return "", false, p
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
			return name, false, p
		}
	}
	// Reads the quote.
	for ; p < len(l.src); p++ {
		if c := l.src[p]; c == '>' {
			return name, false, p
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
		return "", false, p
	}
	return name, true, p
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

// jsMimeTypes contains the types, in lower case and without parameters, of
// the script elements whose content is executed as JavaScript. They are the
// same types as in Go's html/template, except the JSON types.
var jsMimeTypes = map[string]bool{
	"application/ecmascript":   true,
	"application/javascript":   true,
	"application/x-ecmascript": true,
	"application/x-javascript": true,
	"module":                   true,
	"text/ecmascript":          true,
	"text/javascript":          true,
	"text/javascript1.0":       true,
	"text/javascript1.1":       true,
	"text/javascript1.2":       true,
	"text/javascript1.3":       true,
	"text/javascript1.4":       true,
	"text/javascript1.5":       true,
	"text/jscript":             true,
	"text/livescript":          true,
	"text/x-ecmascript":        true,
	"text/x-javascript":        true,
}

// isJSMimeType reports whether typ, the value of the type attribute of a
// script element, is a JavaScript type. Parameters, case and spaces are
// ignored.
func isJSMimeType(typ []byte) bool {
	typ, _, _ = bytes.Cut(typ, []byte(";"))
	return jsMimeTypes[strings.ToLower(string(bytes.TrimSpace(typ)))]
}

// isJSONMimeType reports whether typ, the value of the type attribute of a
// script element, is a type whose content is JSON: application/json, a type
// with the +json suffix, such as application/ld+json, or one of the types
// importmap and speculationrules. Parameters, case and spaces are ignored.
func isJSONMimeType(typ []byte) bool {
	typ, _, _ = bytes.Cut(typ, []byte(";"))
	switch t := strings.ToLower(string(bytes.TrimSpace(typ))); t {
	case "application/json", "importmap", "speculationrules":
		return true
	default:
		return strings.HasSuffix(t, "+json")
	}
}

// isEndScript reports whether s is the start or end of "script" tag.
func isEndScript(s []byte) bool {
	return len(s) >= 9 && s[0] == '<' && s[1] == '/' && (s[8] == '>' || isSpace(s[8])) &&
		(s[2] == 's' || s[2] == 'S') && (s[3] == 'c' || s[3] == 'C') && (s[4] == 'r' || s[4] == 'R') &&
		(s[5] == 'i' || s[5] == 'I') && (s[6] == 'p' || s[6] == 'P') && (s[7] == 't' || s[7] == 'T')
}

// rawTextElement returns the name of the element, script or style, whose
// content can be scanned in the current context, or an empty string if it
// is not the content of a script or style element. In JavaScript, JSON and
// CSS files, and in macros and using blocks with these types, it is the
// element in which the content can be rendered.
func (l *lexer) rawTextElement() string {
	switch l.ctx {
	case ast.ContextJS, ast.ContextJSString, ast.ContextJSRegExp, ast.ContextJSON, ast.ContextJSONString:
		return "script"
	case ast.ContextCSS, ast.ContextCSSString:
		return "style"
	}
	return ""
}

// isDynamicEndTag reports whether l.src[p:], which follows "</", starts
// with a prefix of name, in any case, followed by a show, statement or
// comment, so that it could be an end tag of the element name when the
// template is executed.
func (l *lexer) isDynamicEndTag(p int, name string) bool {
	for i := 0; ; i++ {
		if l.isShowStatementOrCommentStart(p) {
			return true
		}
		if i == len(name) || p == len(l.src) {
			return false
		}
		c := l.src[p]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != name[i] {
			return false
		}
		p++
	}
}

// couldBeEndTag reports whether l.src[p:], which follows "</", could be an
// end tag of the element name: it starts with name, in any case, followed by
// a character that ends a tag name or by the end of the source, or it is a
// dynamic end tag as reported by isDynamicEndTag.
func (l *lexer) couldBeEndTag(p int, name string) bool {
	if l.isDynamicEndTag(p, name) {
		return true
	}
	if len(l.src)-p < len(name) {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := l.src[p+i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != name[i] {
			return false
		}
	}
	p += len(name)
	return p == len(l.src) || l.src[p] == '>' || l.src[p] == '/' || isASCIISpace(l.src[p])
}

// isStringContext reports whether the current context is a CSS, JavaScript
// or JSON string, including a JavaScript template literal, or a JavaScript
// regular expression.
func (l *lexer) isStringContext() bool {
	switch l.ctx {
	case ast.ContextCSSString, ast.ContextJSString, ast.ContextJSONString, ast.ContextJSRegExp:
		return true
	}
	return false
}

// regExpPrecederKeywords contains the JavaScript keywords that can be
// followed by a regular expression literal. Keep isRegExpPrecederKeyword in
// sync.
var regExpPrecederKeywords = []string{
	"break", "case", "continue", "debugger", "default", "delete", "do", "else", "extends",
	"finally", "in", "instanceof", "new", "return", "throw", "try", "typeof", "void",
}

// isRegExpPrecederKeyword reports whether word is one of the keywords in
// regExpPrecederKeywords. It uses a switch, which is faster than a map.
func isRegExpPrecederKeyword(word []byte) bool {
	switch string(word) {
	case "break", "case", "continue", "debugger", "default", "delete", "do", "else", "extends",
		"finally", "in", "instanceof", "new", "return", "throw", "try", "typeof", "void":
		return true
	}
	return false
}

// isJSIdentifierPart reports whether c can be part of a JavaScript
// identifier or number. Bytes of non-ASCII characters are considered part
// of an identifier.
func isJSIdentifierPart(c byte) bool {
	return c == '_' || c == '$' || isDecDigit(c) || isAlpha(c) || c >= 0x80
}

// isJSSpace reports whether s starts with a JavaScript white space or line
// terminator.
func isJSSpace(s []byte) bool {
	if c := s[0]; c < utf8.RuneSelf {
		// Tab, line feed, vertical tab, form feed, carriage return or space.
		return '\t' <= c && c <= '\r' || c == ' '
	}
	r, _ := utf8.DecodeRune(s)
	return unicode.IsSpace(r) && r != '\u0085' || r == '\uFEFF'
}

// isJSLineTerminator reports whether s starts with a JavaScript line
// terminator.
func isJSLineTerminator(s []byte) bool {
	switch s[0] {
	case '\n', '\r':
		return true
	case 0xE2:
		r, _ := utf8.DecodeRune(s)
		return r == '\u2028' || r == '\u2029'
	}
	return false
}

// isJSDivisionAfter reports whether a '/' that follows the JavaScript code s
// is a division operator, and not the start of a regular expression. s ends
// with a non-space character that is not a '(', a ')', or a '+' or '-'
// following the same character. An identifier is considered an operand: the
// keywords are handled by scanJSIdentifier.
func isJSDivisionAfter(s []byte) bool {
	c := s[len(s)-1]
	if c == ']' {
		return true
	}
	if c == '.' {
		// A '.' preceded by a digit ends a number, as in "1.".
		return len(s) > 1 && isDecDigit(s[len(s)-2])
	}
	return isJSIdentifierPart(c)
}

// scanJSIdentifier updates the state after the character at l.src[p], that
// is part of a JavaScript identifier, as if the identifier ended there.
func (l *lexer) scanJSIdentifier(p int) {
	s := &l.state
	if p == 0 || !endsWithJSIdentifierPart(l.src[:p]) {
		// Start of the identifier.
		s.jsIdentName = s.jsNextName
		s.jsIdentParen = s.jsNextParen
		s.jsIdentJoined = s.jsIdentEnd
	}
	s.jsIdentEnd = true
	s.jsNextName = jsNameAny
	s.jsNextParen = jsParenOther
	if s.jsIdentName == jsNameProperty && !s.jsIdentJoined {
		return
	}
	if p+1 < len(l.src) && l.src[p+1] < utf8.RuneSelf && isJSIdentifierPart(l.src[p+1]) {
		// The identifier continues with the next character, which updates
		// the state again.
		return
	}
	word := jsWordAtEnd(l.src[:p+1])
	var preceder, ambiguous bool
	next := jsParenOther
	switch string(word) {
	case "if", "while", "for", "with":
		next = jsParenControl
	case "await":
		// A '(' after "for await" starts the head of the for statement.
		if s.jsIdentParen != jsParenOther {
			next = s.jsIdentParen
		}
		ambiguous = true
	case "yield", "of":
		ambiguous = true
	default:
		preceder = isRegExpPrecederKeyword(word)
	}
	if s.jsIdentName == jsNameProperty {
		// The identifier can only be a property name, or the end of an
		// identifier that continues one before a show, statement or comment.
		next, preceder, ambiguous = jsParenOther, false, false
	}
	keyword := preceder || ambiguous || next != jsParenOther
	if s.jsIdentJoined && (keyword || isJSKeywordEnd(word)) || s.jsIdentName == jsNameUnknown && keyword {
		// It is not known whether the identifier is a keyword: it can be a
		// property name, or continue an identifier before a show, statement
		// or comment, possibly ending a keyword.
		if next != jsParenOther || s.jsIdentJoined {
			next = jsParenUnknown
		}
		preceder = false
		ambiguous = true
	}
	if preceder {
		s.jsDivision = false
	}
	if ambiguous {
		// await, yield and of can be keywords, after which a '/' starts a
		// regular expression, or identifiers, after which it is a division.
		s.jsDivisionUnknown = true
	}
	s.jsNextParen = next
}

// jsNameAfterDot returns the kind of a JavaScript identifier after the '.' at
// l.src[p]. It is a property name, unless the '.' ends a spread operator or
// is the decimal point of a decimal integer literal, as in "1.".
func (l *lexer) jsNameAfterDot(p int) jsName {
	s := l.src[:p]
	if p >= 2 && s[p-1] == '.' && s[p-2] == '.' {
		return jsNameAny
	}
	i := p
	for i > 0 && (isDecDigit(s[i-1]) || s[i-1] == '_') {
		i--
	}
	if i == p {
		if l.state.jsIdentEnd && (p == 0 || !endsWithJSIdentifierPart(s)) {
			// The '.' can follow a number or identifier before a show,
			// statement or comment.
			return jsNameUnknown
		}
		return jsNameProperty
	}
	if !isDecDigit(s[i]) || i > 0 && (isJSIdentifierPart(s[i-1]) || s[i-1] == '.') {
		return jsNameProperty
	}
	if l.state.jsIdentJoined {
		// The number can continue an identifier before a show, statement
		// or comment.
		return jsNameUnknown
	}
	if l.state.jsIdentName != jsNameAny {
		// The digits follow a '.' before a show, statement or comment, or
		// before cut spaces, as in ".{# c #}5.x", so they are the decimal
		// part of a number, and this '.' is not a decimal point. If the
		// '.' does not start a number, the code is not valid.
		return l.state.jsIdentName
	}
	if d := s[i:]; len(d) > 1 && d[0] == '0' && !bytes.ContainsAny(d, "89") {
		// A legacy octal literal, as "07", has no decimal point.
		return jsNameProperty
	}
	return jsNameAny
}

// jsKeywords contains the JavaScript keywords after which the kind of a '('
// or of a '/' depends on the keyword.
var jsKeywords = append([]string{"if", "while", "for", "with", "await", "yield", "of"}, regExpPrecederKeywords...)

// isJSKeywordEnd reports whether word is the end of a keyword in jsKeywords
// that is longer than word.
func isJSKeywordEnd(word []byte) bool {
	if len(word) == 0 {
		return false
	}
	for _, k := range jsKeywords {
		if len(k) > len(word) && strings.HasSuffix(k, string(word)) {
			return true
		}
	}
	return false
}

// endsWithJSIdentifierPart reports whether the JavaScript code s ends with a
// character that can be part of an identifier.
func endsWithJSIdentifierPart(s []byte) bool {
	c := s[len(s)-1]
	if c < utf8.RuneSelf {
		return isJSIdentifierPart(c)
	}
	_, size := utf8.DecodeLastRune(s)
	return !isJSSpace(s[len(s)-size:])
}

// jsWordAtEnd returns the identifier at the end of the JavaScript code s, if
// it is not longer than the longest keyword, otherwise it returns nil.
func jsWordAtEnd(s []byte) []byte {
	const maxLen = len("instanceof")
	start := len(s)
	for start > 0 && isJSIdentifierPart(s[start-1]) {
		if s[start-1] >= utf8.RuneSelf {
			// A non-ASCII space or line terminator ends the identifier,
			// any other non-ASCII character is part of it.
			_, size := utf8.DecodeLastRune(s[:start])
			if isJSSpace(s[start-size : start]) {
				break
			}
			return nil
		}
		if len(s)-start == maxLen {
			return nil
		}
		start--
	}
	return s[start:]
}

// isShowStatementOrCommentStart reports whether l.src[p:] starts with a
// short show statement, a statement or a comment. In the content of a raw
// statement, they are text.
func (l *lexer) isShowStatementOrCommentStart(p int) bool {
	if p+1 >= len(l.src) || l.src[p] != '{' || l.inRaw && p < l.rawEnd {
		return false
	}
	switch l.src[p+1] {
	case '{':
		return !l.noParseShow
	case '%', '#':
		return true
	}
	return false
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
	l.code = &codeBlocks{}
	err := l.lexCode(tokenEndStatements)
	l.code = nil
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
//	if end is tokenEOF, it returns when encounters tokenEOF
//
//	if end is tokenEndStatement or tokenEndStatements, it returns when
//	encounters tokenEOF, tokenEndStatement or tokenEndStatements
//
//	if end is tokenRightBraces, it returns when encounters tokenEOF,
//	tokenEndStatement, tokenEndStatements or tokenRightBraces
func (l *lexer) lexCode(end tokenTyp) error {
	if len(l.src) == 0 {
		if end != tokenEOF {
			return l.errorf("unexpected EOF, expecting %s", end)
		}
		return nil
	}
	// first is the index of the first token in code.
	var first = l.totals + 1
	// afterElse reports whether the first token is the else keyword.
	var afterElse bool
	// label is the label of the statement, if any, and firstIdent the first
	// token if it is an identifier, which is a label if followed by a colon.
	var label, firstIdent string
	// afterEnd reports whether the first token is the end keyword.
	var afterEnd bool
	// afterShow reports whether the first token is the show keyword, and
	// showItea whether it is followed by the itea identifier.
	var afterShow, showItea bool
	// beforeShow is the scan state before the show keyword.
	var beforeShow scanState
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
			if len(l.src) > 1 && '0' <= l.src[1] && l.src[1] <= '9' {
				err := l.lexNumber()
				if err != nil {
					return err
				}
				endLineAsSemicolon = true
			} else if len(l.src) > 1 && l.src[1] == '.' && len(l.src) > 2 && l.src[2] == '.' {
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
									if l.ctx == ast.ContextHTML || l.ctx == ast.ContextMarkdown {
										// The body is scanned in the context of
										// the type, also after its tags.
										l.base = l.ctx
										l.tag.ctx = l.ctx
									}
									break
								}
							}
						}
						// Store the state at the beginning of the body of a
						// macro or using statement.
						if n := len(l.blocks); macroOrUsing && n > 0 {
							if b := &l.blocks[n-1]; b.kind == blockMacro || b.kind == blockUsing {
								b.body = l.snapshot()
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
					if unclosedLeftBraces == 0 {
						return nil
					}
					if unclosedLeftBraces == 1 && !(len(l.src) > 2 && l.src[2] == '}') {
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
				if end == tokenEndStatement && firstIdent != "" && l.totals == first+1 {
					// In a labeled statement, the first token is the one that
					// follows the label and the colon.
					label = firstIdent
					first += 2
				}
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
			// A show statement among other statements may not be executed,
			// so it is not known whether a following '/' is a division.
			if end == tokenEndStatements && typ == tokenShow && l.ctx == ast.ContextJS && l.state.jsComment == jsCommentNone {
				l.state.jsDivisionUnknown = true
				l.state.jsLineCodeUnknown = true
			}
			// A break, continue or return statement among other statements
			// ends a branch or a body as in a single statement, unless it
			// refers to a statement of the code.
			if end == tokenEndStatements && (typ == tokenBreak || typ == tokenContinue || typ == tokenReturn) &&
				!l.code.captures(typ, typ != tokenReturn && l.nextIdentifier() != "") {
				if err := l.scanBlockStatement(typ, false, l.column-len(txt)); err != nil {
					return err
				}
			}
			// If it has lexed the first token after '{%', or the using
			// keyword, updates the state of the scan for block statements.
			if end == tokenEndStatement {
				if l.totals == first && typ == tokenIdentifier {
					firstIdent = txt
				} else if l.totals == first {
					if typ == tokenMacro {
						macroOrUsing = true
					}
					afterElse = typ == tokenElse
					afterEnd = typ == tokenEnd
					afterShow = typ == tokenShow
					if afterShow {
						beforeShow = l.state.clone()
					}
					n := len(l.blocks)
					if err := l.scanBlockStatement(typ, false, l.column-len(txt)); err != nil {
						return err
					}
					if label != "" && len(l.blocks) > n {
						l.blocks[len(l.blocks)-1].label = label
					}
				} else if afterElse && l.totals == first+1 && typ == tokenIf {
					if err := l.scanBlockStatement(typ, true, l.column-len(txt)); err != nil {
						return err
					}
				} else if afterShow && l.totals == first+1 && typ == tokenIdentifier && txt == "itea" {
					showItea = true
				} else if typ == tokenUsing && !afterEnd {
					// A using keyword after end, as in {% end using %}, does
					// not start a using statement.
					macroOrUsing = true
					if err := l.scanBlockStatement(typ, false, l.column-len(txt)); err != nil {
						return err
					}
					// The semicolon between itea and using is not counted
					// if it is inserted at a newline.
					if showItea && (l.totals == first+2 || l.totals == first+3) {
						// In {% show itea; using %}, the content is shown in
						// place, so the body is scanned starting from the
						// state before the show keyword.
						b := &l.blocks[len(l.blocks)-1]
						b.inPlace = true
						l.state = beforeShow.clone()
					}
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

// isBinDigit reports whether c is a binary digit.
func isBinDigit(c byte) bool {
	return c == '0' || c == '1'
}

// isOctDigit reports whether c is an octal digit.
func isOctDigit(c byte) bool {
	return '0' <= c && c <= '7'
}

// isDecDigit reports whether c is a decimal digit.
func isDecDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

// isHexDigit reports whether c is a hexadecimal digit.
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
		case "and":
			typ = tokenExtendedAnd
		case "end":
			typ = tokenEnd
		case "contains":
			typ = tokenContains
		case "extends":
			typ = tokenExtends
		case "in":
			typ = tokenIn
		case "macro":
			typ = tokenMacro
		case "not":
			typ = tokenExtendedNot
		case "or":
			typ = tokenExtendedOr
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
	var exponent rune
	var is0o bool
	p := 0
	base := 10
	if c := l.src[0]; c == '0' && len(l.src) > 1 {
		switch l.src[1] {
		case 'x', 'X':
			base = 16
			p = 2
		case 'o', 'O':
			is0o = true
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
			if p+1 < len(l.src) && !isHexDigit(l.src[p+1]) {
				return l.errorf("'_' must separate successive digits")
			}
			p++
		}
	}
	if p < len(l.src) && l.src[p] == '.' {
		if base == 8 && !is0o {
			base = 10
		}
		if base < 10 {
			return l.errorf("invalid radix point in %s literal", numberBaseName[base])
		}
		dot = true
		p++
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
			if exponent == 0 && !isHexDigit(c) || exponent != 0 && !isDecDigit(c) {
				if dot && exponent == 0 {
					return l.errorf("hexadecimal mantissa requires a 'p' exponent")
				}
				break DIGITS
			}
		case 8:
			if !isOctDigit(c) {
				if (c == '8' || c == '9') && !is0o {
					// It could be an imaginary literal.
					base = 10
				} else if isDecDigit(c) {
					return l.errorf("invalid digit '%c' in octal literal", c)
				} else {
					break DIGITS
				}
			}
		case 2:
			if !isBinDigit(c) {
				if isDecDigit(c) {
					return l.errorf("invalid digit '%c' in binary literal", c)
				}
				break DIGITS
			}
		}
		p++
		if p < len(l.src) {
			switch l.src[p] {
			case '_':
				p++
				if p < len(l.src) && !isHexDigit(l.src[p]) {
					break DIGITS
				}
				continue DIGITS
			case '.':
				if dot || exponent != 0 {
					break DIGITS
				}
				if base == 8 && !is0o {
					base = 10
				}
				if base < 10 {
					return l.errorf("invalid radix point in %s literal", numberBaseName[base])
				}
				dot = true
				p++
				if p == len(l.src) {
					break DIGITS
				}
			}
			switch l.src[p] {
			case 'e', 'E':
				if base == 16 {
					if dot {
						return l.errorf("hexadecimal mantissa requires a 'p' exponent")
					}
					break
				}
				if base == 8 && !is0o {
					base = 10
				}
				if base != 10 {
					return l.errorf("'%c' exponent requires decimal mantissa", l.src[p])
				}
				if exponent != 0 {
					break
				}
				exponent = 'e'
				p++
				if p < len(l.src) && (l.src[p] == '+' || l.src[p] == '-') {
					p++
				}
			case 'p', 'P':
				if base != 16 {
					return l.errorf("'%c' exponent requires hexadecimal mantissa", l.src[p])
				}
				if exponent != 0 {
					break
				}
				exponent = 'p'
				p++
				if p < len(l.src) && (l.src[p] == '+' || l.src[p] == '-') {
					p++
				}
			}
		}
	}
	if p < len(l.src) && l.src[p] == '_' {
		return l.errorf("'_' must separate successive digits")
	}
	switch l.src[p-1] {
	case 'x', 'X', 'o', 'O', 'b', 'B':
		if p == 2 {
			return l.errorf("%s literal has no digits", numberBaseName[base])
		}
	case '.':
		if p == 3 && base == 16 {
			return l.errorf("hexadecimal literal has no digits")
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
	if base == 16 && dot && exponent != 'p' {
		return l.errorf("hexadecimal mantissa requires a 'p' exponent")
	}
	imaginary := p < len(l.src) && l.src[p] == 'i'
	if imaginary {
		p++
	} else if p > 0 && base == 10 && l.src[0] == '0' && !dot && exponent == 0 {
		for _, c := range l.src[1:p] {
			if c == '8' || c == '9' {
				return l.errorf("invalid digit '%c' in octal literal", c)
			}
		}
	}
	if imaginary {
		l.emit(tokenImaginary, p)
	} else if dot || exponent != 0 {
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
				for i := range 2 {
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
				for i := range 2 {
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
