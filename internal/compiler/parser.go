// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/ast/astutil"
)

// SyntaxError records a parsing error with the path and the position where the
// error occurred.
type SyntaxError struct {
	path string
	pos  ast.Position
	msg  string
}

// Error returns a string representing the syntax error.
func (e *SyntaxError) Error() string {
	return fmt.Sprintf("%s:%s: syntax error: %s", e.path, e.pos, e.msg)
}

// Message returns the message of the syntax error, without position and path.
func (e *SyntaxError) Message() string {
	return e.msg
}

// Path returns the path of the syntax error.
func (e *SyntaxError) Path() string {
	return e.path
}

// Position returns the position of the syntax error.
func (e *SyntaxError) Position() ast.Position {
	return e.pos
}

// syntaxError returns a SyntaxError error with position pos and message
// formatted according the given format.
func syntaxError(pos *ast.Position, format string, a ...interface{}) *SyntaxError {
	return &SyntaxError{"", *pos, fmt.Sprintf(format, a...)}
}

// CycleError implements an error indicating the presence of a cycle.
type CycleError struct {
	path string
	pos  ast.Position
	msg  string
}

func (e *CycleError) Error() string {
	return e.msg
}

// Path returns the path of the first referenced file in the cycle.
func (e *CycleError) Path() string {
	return e.path
}

// Position returns the position of the statements extends and import or the
// expression render referring the second file in the cycle.
func (e *CycleError) Position() ast.Position {
	return e.pos
}

// Message returns the message of cycle error, without position and path.
func (e *CycleError) Message() string {
	return e.msg
}

// containsOnlySpaces reports whether b contains only white space characters
// as intended by Go parser.
func containsOnlySpaces(bytes []byte) bool {
	for _, b := range bytes {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}
	return true
}

// firstNonSpacePosition returns the position of the first non space character
// in token. Returns nil if token does not contain non space characters.
func firstNonSpacePosition(tok token) *ast.Position {
	pos := *(tok.pos)
	for i, b := range tok.txt {
		switch b {
		case ' ', '\t', '\r':
			pos.Column++
		case '\n':
			pos.Line++
			pos.Column = 1
		default:
			_, n := utf8.DecodeRune(tok.txt[i:])
			pos.Start += i
			pos.End = pos.Start + n
			return &pos
		}
	}
	return nil
}

// parsing is a parsing state.
type parsing struct {

	// Lexer.
	lex *lexer

	// Tree content format.
	format ast.Format

	// Report whether it is a script.
	isScript bool

	// Report whether it is imported.
	imported bool

	// Reports whether it has an extend statement.
	hasExtend bool

	// Reports whether there is a token in current line for which it is possible
	// to cut the leading and trailing spaces.
	cutSpacesToken bool

	// Ancestors from the root up to the parent.
	ancestors []ast.Node

	// Unexpanded Extends, Import and Render nodes.
	unexpanded []ast.Node
}

// addToAncestors adds node to the ancestors.
func (p *parsing) addToAncestors(node ast.Node) {
	p.ancestors = append(p.ancestors, node)
}

// removeLastAncestor removes the last ancestor from the ancestors.
func (p *parsing) removeLastAncestor() {
	p.ancestors = p.ancestors[0 : len(p.ancestors)-1]
}

// parent returns the last ancestor.
func (p *parsing) parent() ast.Node {
	return p.ancestors[len(p.ancestors)-1]
}

// inFunction reports whether it is in a function body.
func (p *parsing) inFunction() bool {
	for i := len(p.ancestors) - 1; i > 0; i-- {
		if n, ok := p.ancestors[i].(*ast.Func); ok {
			return !n.Type.Macro
		}
	}
	return false
}

// next returns the next token from the lexer. Panics if the lexer channel is
// closed.
func (p *parsing) next() token {
	tok, ok := <-p.lex.Tokens()
	if !ok {
		if p.lex.err == nil {
			panic("next called after EOF")
		}
		panic(p.lex.err)
	}
	return tok
}

// parseSource parses a program or a script and returns its tree.
// script reports whether it is a script.
func parseSource(src []byte, script bool) (tree *ast.Tree, err error) {

	tree = ast.NewTree("", nil, ast.FormatText)

	var p = &parsing{
		isScript:  script,
		ancestors: []ast.Node{tree},
	}
	if script {
		p.lex = scanScript(src)
	} else {
		p.lex = scanProgram(src)
	}

	defer func() {
		p.lex.Stop()
		if r := recover(); r != nil {
			if e, ok := r.(*SyntaxError); ok {
				tree = nil
				err = e
			} else {
				panic(r)
			}
		}
	}()

	tok := p.next()
	if tok.typ == tokenShebangLine {
		if !script {
			return nil, syntaxError(tok.pos, "invalid character U+0023 '#'")
		}
		tok = p.next()
	}

	for tok.typ != tokenEOF {
		tok = p.parse(tok, tokenEOF)
	}

	if len(p.ancestors) == 1 {
		if !script && len(tree.Nodes) == 0 {
			panic(syntaxError(tok.pos, "expected 'package', found 'EOF'"))
		}
	} else {
		switch p.ancestors[1].(type) {
		case *ast.Package:
		case *ast.Label:
			return nil, syntaxError(tok.pos, "missing statement after label")
		default:
			if len(p.ancestors) > 2 {
				return nil, syntaxError(tok.pos, "unexpected EOF, expecting }")
			}
		}
	}

	return tree, nil
}

// ParseTemplateSource parses a template with content src in the given format
// and returns its tree and the unexpanded Extends, Import, Render and
// Assignment nodes.
//
// If noParseShow is true, short show statements are not parsed.
//
// format can be Text, HTML, CSS, JavaScript, JSON and Markdown. imported
// indicates whether it is imported.
func ParseTemplateSource(src []byte, format ast.Format, imported, noParseShow, dollarIdentifier bool) (tree *ast.Tree, unexpanded []ast.Node, err error) {

	if format < ast.FormatText || format > ast.FormatMarkdown {
		return nil, nil, errors.New("scriggo: invalid format")
	}

	tree = ast.NewTree("", nil, format)

	var p = &parsing{
		lex:        scanTemplate(src, format, noParseShow, dollarIdentifier),
		format:     format,
		imported:   imported,
		ancestors:  []ast.Node{tree},
		unexpanded: []ast.Node{},
	}

	defer func() {
		p.lex.Stop()
		if r := recover(); r != nil {
			if e, ok := r.(*SyntaxError); ok {
				tree = nil
				err = e
			} else {
				panic(r)
			}
		}
	}()

	// line is the current line number.
	var line = 0

	// firstText is the first Text node of the current line.
	var firstText *ast.Text

	// numTokenInLine is the number of non-text tokens in the current line.
	var numTokenInLine = 0

	// lastIndex is the index of the last byte of the source.
	var lastIndex = len(src) - 1

	tok := p.next()

	for tok.typ != tokenEOF {

		var text *ast.Text
		if tok.typ == tokenText {
			if (imported || p.hasExtend) && len(p.ancestors) == 1 && !containsOnlySpaces(tok.txt) {
				pos := firstNonSpacePosition(tok)
				if imported {
					return nil, nil, syntaxError(pos, "unexpected text in imported file")
				}
				return nil, nil, syntaxError(pos, "unexpected text in file with extends")
			}
			text = ast.NewText(tok.pos, tok.txt, ast.Cut{})
		}

		if line < tok.lin || tok.pos.End == lastIndex {
			if p.cutSpacesToken && numTokenInLine == 1 {
				cutSpaces(firstText, text)
			}
			line = tok.lin
			firstText = text
			p.cutSpacesToken = false
			numTokenInLine = 0
		}

		var wantCase bool
		switch n := p.parent().(type) {
		case *ast.Switch:
			wantCase = len(n.Cases) == 0
		case *ast.TypeSwitch:
			wantCase = len(n.Cases) == 0
		case *ast.Select:
			wantCase = len(n.Cases) == 0
		}
		if wantCase {
			switch tok.typ {
			case tokenStartStatement:
			case tokenText:
				if !containsOnlySpaces(text.Text) {
					return nil, nil, syntaxError(tok.pos, "unexpected text, expecting {%%")
				}
				switch n := p.parent().(type) {
				case *ast.Switch:
					n.LeadingText = text
				case *ast.TypeSwitch:
					n.LeadingText = text
				case *ast.Select:
					n.LeadingText = text
				}
				tok = p.next()
				continue
			default:
				return nil, nil, syntaxError(tok.pos, "unexpected %s, expecting {%%", tok.typ)
			}
		}

		switch tok.typ {

		// Text
		case tokenText:
			p.addNode(text)
			tok = p.next()

		// {%
		case tokenStartStatement:
			numTokenInLine++
			tok = p.next()
			tok = p.parse(tok, tokenEndStatement)

		// {%%
		case tokenStartStatements:
			numTokenInLine++
			pos := tok.pos
			var statements = ast.NewStatements(pos, nil)
			p.addNode(statements)
			tok = p.next()
			for tok.typ != tokenEndStatements {
				tok = p.parse(tok, tokenEndStatements)
				if tok.typ == tokenEOF {
					return nil, nil, syntaxError(tok.pos, "unexpected EOF, expecting %%%%}")
				}
			}
			if _, ok := p.ancestors[len(p.ancestors)-1].(*ast.Statements); !ok {
				return nil, nil, syntaxError(tok.pos, "unexpected %%%%}, expecting }")
			}
			pos.End = tok.pos.End
			p.removeLastAncestor()
			tok = p.next()

		// {{
		case tokenLeftBraces:
			pos := tok.pos
			if len(p.ancestors) == 1 && (p.imported || p.hasExtend) {
				panic(syntaxError(pos, "unexpected %s, expecting declaration statement", tok))
			}
			numTokenInLine++
			var expr ast.Expression
			expr, tok = p.parseExpr(p.next(), false, false, false)
			if expr == nil {
				return nil, nil, syntaxError(tok.pos, "unexpected %s, expecting expression", tok)
			}
			if tok.typ != tokenRightBraces {
				return nil, nil, syntaxError(tok.pos, "unexpected %s, expecting }}", tok)
			}
			pos.End = tok.pos.End
			var node = ast.NewShow(pos, []ast.Expression{expr}, tok.ctx)
			p.addNode(node)
			if _, ok := expr.(*ast.Render); ok {
				p.cutSpacesToken = true
			}
			tok = p.next()

		// StartURL
		case tokenStartURL:
			node := ast.NewURL(tok.pos, tok.tag, tok.att, nil)
			p.addNode(node)
			tok = p.next()

		// EndURL
		case tokenEndURL:
			pos := p.parent().Pos()
			pos.End = tok.pos.End - 1
			p.removeLastAncestor()
			tok = p.next()

		// comment
		case tokenComment:
			numTokenInLine++
			node := ast.NewComment(tok.pos, string(tok.txt[2:len(tok.txt)-2]))
			p.addNode(node)
			p.cutSpacesToken = true
			tok = p.next()

		default:
			return nil, nil, syntaxError(tok.pos, "unexpected %s", tok)

		}

	}

	// If the ancestors are {Tree, Func, Block} check if Func is endless.
	if len(p.ancestors) == 3 {
		if fn, ok := p.ancestors[1].(*ast.Func); ok && fn.Endless {
			fn.End = tok.pos.End
			p.removeLastAncestor()
			p.removeLastAncestor()
		}
	}

	if len(p.ancestors) > 1 {
		var stmt, marker string
		switch n := p.parent().(type) {
		case *ast.Block:
			switch p.ancestors[len(p.ancestors)-2].(type) {
			case *ast.Func:
				stmt = "macro"
			case *ast.If:
				stmt = "if"
			}
		case *ast.For, *ast.ForIn, *ast.ForRange:
			stmt = "for"
		case *ast.Raw:
			stmt = "raw"
			marker = n.Marker
		case *ast.Select:
			stmt = "select"
		case *ast.Switch, *ast.TypeSwitch:
			stmt = "switch"
		}
		if marker != "" {
			return nil, nil, syntaxError(tok.pos, "unexpected EOF, expecting {%% end raw %s %%}", marker)
		}
		return nil, nil, syntaxError(tok.pos, "unexpected EOF, expecting {%% end %%} or {%% end %s %%}", stmt)
	}

	return tree, p.unexpanded, nil
}

// parse parses code.
//
// For a package, a script or a function body, tok is the first token of a
// declaration or statement and end is EOF. The returned token is the first
// token of the next statement or declaration, or it is EOF.
//
// For templates, it parses the code between {% and %} or between {%% and %%}.
// tok is the first token after {% or {%%, end is %} or %%} and the returned
// token is the first token after %} or %%}, or it is EOF.
//
// If a syntax error occurs, parse panics with a *SyntaxError value. If there
// is a cycle it panics with a *CycleError value.
func (p *parsing) parse(tok token, end tokenTyp) token {

LABEL:

	wantCase := false
	switch s := p.parent().(type) {
	case *ast.Tree:
		if p.imported || p.hasExtend {
			switch tok.typ {
			case tokenExtends, tokenImport, tokenMacro, tokenVar, tokenConst, tokenType:
			default:
				return p.parseEndlessMacro(tok, end)
			}
		} else if p.isScript || end != tokenEOF {
			if tok.typ == tokenReturn {
				panic(syntaxError(tok.pos, "return statement outside function body"))
			}
		} else if tok.typ != tokenPackage {
			panic(syntaxError(tok.pos, "expected 'package', found '%s'", tok))
		}
	case *ast.Statements:
		if (p.imported || p.hasExtend) && len(p.ancestors) == 2 {
			switch tok.typ {
			case tokenExtends, tokenImport, tokenMacro, tokenVar, tokenConst, tokenType:
			default:
				panic(syntaxError(tok.pos, "unexpected %s, expecting declaration statement", tok))
			}
		}
	case *ast.Package:
		switch tok.typ {
		case tokenImport, tokenFunc, tokenVar, tokenConst, tokenType:
		default:
			panic(syntaxError(tok.pos, "non-declaration statement outside function body"))
		}
	case *ast.Label:
		if end == tokenEndStatement {
			switch tok.typ {
			case tokenFor, tokenSwitch, tokenSelect:
			default:
				panic(syntaxError(tok.pos, "unexpected %s, expecting for, switch or select", tok))
			}
		}
	case *ast.Switch:
		wantCase = len(s.Cases) == 0
	case *ast.TypeSwitch:
		wantCase = len(s.Cases) == 0
	case *ast.Select:
		wantCase = len(s.Cases) == 0
	}
	if wantCase && tok.typ != tokenCase && tok.typ != tokenDefault {
		want := tokenRightBrace
		if end == tokenEndStatement {
			want = tokenEnd
		}
		if tok.typ != want {
			panic(syntaxError(tok.pos, "unexpected %s, expecting case or default or %s", tok, want))
		}
	}

	switch tok.typ {

	// ;
	case tokenSemicolon:
		if end == tokenEndStatement {
			panic(syntaxError(tok.pos, "unexpected semicolon, expecting statement"))
		}
		return p.next()

	// package
	case tokenPackage:
		pos := tok.pos
		if tree, ok := p.parent().(*ast.Tree); !ok || end != tokenEOF || p.isScript || len(tree.Nodes) > 0 {
			panic(syntaxError(tok.pos, "unexpected package, expecting statement"))
		}
		tok = p.next()
		if tok.typ != tokenIdentifier {
			panic(syntaxError(tok.pos, "expected 'IDENT', found %q", tok.txt))
		}
		name := string(tok.txt)
		if name == "_" {
			panic(syntaxError(tok.pos, "invalid package name _"))
		}
		pos.End = tok.pos.End
		tok = p.next()
		node := ast.NewPackage(pos, name, nil)
		p.addNode(node)
		tok = p.parseEnd(tok, tokenSemicolon, end)
		return tok

	// for
	case tokenFor:
		pos := tok.pos
		var node, init ast.Node
		init, tok = p.parseSimpleStatement(p.next(), true, true)
		switch tok.typ {
		case tokenSemicolon:
			// Parse:    for init ; cond ; post {
			//        {% for init ; cond ; post %}
			var cond ast.Node
			cond, tok = p.parseSimpleStatement(p.next(), false, true)
			if tok.typ != tokenSemicolon {
				if cond == nil {
					panic(syntaxError(tok.pos, "unexpected %s, expecting for loop condition", tok))
				}
				panic(syntaxError(tok.pos, "unexpected %s, expecting semicolon or newline", tok))
			}
			var condition ast.Expression
			if cond != nil {
				condition, _ = cond.(ast.Expression)
				if condition == nil {
					panic(cannotUseAsValueError(pos, init))
				}
			}
			var post ast.Node
			post, tok = p.parseSimpleStatement(p.next(), false, true)
			if post != nil {
				if a, ok := post.(*ast.Assignment); ok && a.Type == ast.AssignmentDeclaration {
					panic(syntaxError(pos, "cannot declare in post statement of for loop"))
				}
			}
			pos.End = tok.pos.End
			node = ast.NewFor(pos, init, condition, post, nil)
		case tokenLeftBrace, tokenEndStatement:
			// Parse:    for cond {
			//        {% for cond %}
			var condition ast.Expression
			if init != nil {
				condition, _ = init.(ast.Expression)
				if condition == nil {
					panic(cannotUseAsValueError(pos, init))
				}
			}
			pos.End = tok.pos.End
			node = ast.NewFor(pos, nil, condition, nil, nil)
		case tokenRange:
			// Parse:    for i, v := range expr {
			//        {% for i, v := range expr %}
			var assignment *ast.Assignment
			if init == nil {
				tpos := tok.pos
				assignment = ast.NewAssignment(tpos, nil, ast.AssignmentSimple, nil)
			} else {
				assignment, _ = init.(*ast.Assignment)
				if assignment == nil || assignment.Rhs != nil {
					panic(syntaxError(tok.pos, "unexpected range, expecting {"))
				}
			}
			var expr ast.Expression
			expr, tok = p.parseExpr(p.next(), false, false, true)
			if expr == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			assignment.Rhs = []ast.Expression{expr}
			assignment.End = expr.Pos().End
			pos.End = tok.pos.End
			node = ast.NewForRange(pos, assignment, nil)
		case tokenIn:
			// Parse: {% for id in expr %}
			if init == nil {
				panic(syntaxError(tok.pos, "unexpected in, expecting expression"))
			}
			ident, ok := init.(*ast.Identifier)
			if !ok {
				panic(syntaxError(tok.pos, "unexpected in, expecting {"))
			}
			var expr ast.Expression
			expr, tok = p.parseExpr(p.next(), false, false, true)
			if expr == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			node = ast.NewForIn(pos, ident, expr, nil)
		default:
			panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
		}
		p.addNode(node)
		p.cutSpacesToken = true
		tok = p.parseEnd(tok, tokenLeftBrace, end)
		return tok

	// break
	case tokenBreak:
		pos := tok.pos
		var label *ast.Identifier
		tok = p.next()
		if tok.typ == tokenIdentifier {
			label = ast.NewIdentifier(tok.pos, string(tok.txt))
			pos.End = tok.pos.End
			tok = p.next()
		}
		node := ast.NewBreak(pos, label)
		p.addNode(node)
		p.cutSpacesToken = true
		tok = p.parseEnd(tok, tokenSemicolon, end)
		return tok

	// continue
	case tokenContinue:
		pos := tok.pos
		var label *ast.Identifier
		tok = p.next()
		if tok.typ == tokenIdentifier {
			label = ast.NewIdentifier(tok.pos, string(tok.txt))
			pos.End = tok.pos.End
			tok = p.next()
		}
		node := ast.NewContinue(pos, label)
		p.addNode(node)
		p.cutSpacesToken = true
		tok = p.parseEnd(tok, tokenSemicolon, end)
		return tok

	// switch
	case tokenSwitch:
		node := p.parseSwitch(tok, end)
		p.addNode(node)
		p.cutSpacesToken = true
		return p.next()

	// case
	case tokenCase:
		pos := tok.pos
		var node ast.Node
		switch p.parent().(type) {
		case *ast.Switch, *ast.TypeSwitch:
			var expressions []ast.Expression
			expressions, tok = p.parseExprList(p.next(), false, false, false)
			if expressions == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			pos.End = expressions[len(expressions)-1].Pos().End
			node = ast.NewCase(pos, expressions, nil)
		case *ast.Select:
			var expressions []ast.Expression
			expressions, tok = p.parseExprList(p.next(), false, false, false)
			if expressions == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			var comm ast.Node
			if len(expressions) > 1 || isAssignmentToken(tok) {
				var assignment *ast.Assignment
				assignment, tok = p.parseAssignment(expressions, tok, false, false, false)
				comm = assignment
			} else {
				if tok.typ == tokenArrow {
					sendPos := tok.pos
					var value ast.Expression
					value, tok = p.parseExpr(p.next(), false, false, false)
					if value == nil {
						panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
					}
					channel := expressions[0]
					comm = ast.NewSend(sendPos, channel, value)
				} else {
					comm = expressions[0]
				}
			}
			pos.End = tok.pos.End
			node = ast.NewSelectCase(pos, comm, nil)
		default:
			// Panic with a syntax error.
			if p.inFunction() {
				p.parseEnd(tok, tokenRightBrace, end)
			}
			panic(syntaxError(tok.pos, "case is not in a switch or select"))
		}
		p.addNode(node)
		tok = p.parseEnd(tok, tokenColon, end)
		return tok

	// default
	case tokenDefault:
		pos := tok.pos
		var node ast.Node
		switch p.parent().(type) {
		case *ast.Switch, *ast.TypeSwitch:
			node = ast.NewCase(pos, nil, nil)
		case *ast.Select:
			node = ast.NewSelectCase(pos, nil, nil)
		default:
			if p.inFunction() {
				p.parseEnd(tok, tokenRightBrace, end)
			}
			panic(syntaxError(tok.pos, "default is not in a switch or select"))
		}
		p.addNode(node)
		p.cutSpacesToken = true
		tok = p.next()
		tok = p.parseEnd(tok, tokenColon, end)
		return tok

	// fallthrough
	case tokenFallthrough:
		node := ast.NewFallthrough(tok.pos)
		p.addNode(node)
		p.cutSpacesToken = true
		tok = p.next()
		tok = p.parseEnd(tok, tokenSemicolon, end)
		return tok

	// select
	case tokenSelect:
		node := ast.NewSelect(tok.pos, nil, nil)
		p.addNode(node)
		p.cutSpacesToken = true
		tok = p.next()
		tok = p.parseEnd(tok, tokenLeftBrace, end)
		return tok

	// {
	case tokenLeftBrace:
		if end == tokenEndStatement {
			panic(syntaxError(tok.pos, "unexpected %s, expecting statement", tok))
		}
		node := ast.NewBlock(tok.pos, nil)
		p.addNode(node)
		p.cutSpacesToken = true
		return p.next()

	// }
	case tokenRightBrace:
		var unexpected bool
		switch end {
		case tokenEOF:
			unexpected = p.isScript && len(p.ancestors) == 1
		case tokenEndStatement:
			unexpected = true
		case tokenEndStatements:
			_, unexpected = p.parent().(*ast.Statements)
		default:
			panic(end)
		}
		if unexpected {
			panic(syntaxError(tok.pos, "unexpected }, expecting statement"))
		}
		if _, ok := p.parent().(*ast.Label); ok {
			p.removeLastAncestor()
		}
		bracesEnd := tok.pos.End
		p.parent().Pos().End = bracesEnd
		p.removeLastAncestor()
		tok = p.next()
		switch tok.typ {
		case tokenElse:
		case tokenSemicolon:
			tok = p.next()
			fallthrough
		case tokenRightBrace:
			for {
				if n, ok := p.parent().(*ast.If); ok {
					n.Pos().End = bracesEnd
					p.removeLastAncestor()
				} else {
					return tok
				}
			}
		case tokenRightBraces, tokenEndStatement:
			return tok
		case tokenEOF:
			// TODO(marco): check if it is correct.
			return p.next()
		default:
			panic(syntaxError(tok.pos, "unexpected %s at end of statement", tok))
		}
		fallthrough

	// else
	case tokenElse:
		if end == tokenEndStatement {
			// Close the "then" block.
			if _, ok := p.parent().(*ast.Block); !ok {
				panic(syntaxError(tok.pos, "unexpected else"))
			}
			p.removeLastAncestor()
		}
		if _, ok := p.parent().(*ast.If); !ok {
			panic(syntaxError(tok.pos, "unexpected else at end of statement"))
		}
		p.cutSpacesToken = true
		tok = p.next()
		if end == tokenEndStatement && tok.typ == tokenEndStatement || end != tokenEndStatement && tok.typ == tokenLeftBrace {
			// "else"
			var blockPos *ast.Position
			if end != tokenEndStatement {
				blockPos = tok.pos
			}
			elseBlock := ast.NewBlock(blockPos, nil)
			p.addNode(elseBlock)
			return p.next()
		}
		if tok.typ != tokenIf {
			panic(syntaxError(tok.pos, "else must be followed by if or statement block"))
		}
		fallthrough

	// if
	case tokenIf:
		pos := tok.pos
		ifPos := tok.pos
		var init ast.Node
		var expr ast.Expression
		init, tok = p.parseSimpleStatement(p.next(), false, true)
		if tok.typ == tokenSemicolon {
			expr, tok = p.parseExpr(p.next(), false, false, true)
		} else if init != nil {
			expr, _ = init.(ast.Expression)
			if expr == nil {
				panic(cannotUseAsValueError(pos, init))
			}
			init = nil
		}
		if expr == nil {
			if end == tokenEndStatement && tok.typ == tokenEndStatement || end != tokenEndStatement && tok.typ == tokenLeftBrace {
				panic(syntaxError(tok.pos, "missing condition in if statement"))
			}
			panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
		}
		var blockPos *ast.Position
		if end != tokenEndStatement {
			blockPos = tok.pos
		}
		then := ast.NewBlock(blockPos, nil)
		if _, ok := p.parent().(*ast.If); !ok {
			ifPos = pos
			ifPos.End = tok.pos.End
		}
		node := ast.NewIf(ifPos, init, expr, then, nil)
		p.addNode(node)
		p.cutSpacesToken = true
		tok = p.parseEnd(tok, tokenLeftBrace, end)
		return tok

	// return
	case tokenReturn:
		pos := tok.pos
		tok = p.next()
		var values []ast.Expression
		if end == tokenEOF {
			values, tok = p.parseExprList(tok, false, false, false)
			if values != nil {
				pos.End = values[len(values)-1].Pos().End
			}
		}
		node := ast.NewReturn(pos, values)
		p.addNode(node)
		tok = p.parseEnd(tok, tokenSemicolon, end)
		return tok

	// show
	case tokenShow:
		if p.inFunction() {
			panic(syntaxError(tok.pos, "unexpected %s, expecting }", tok))
		}
		pos := tok.pos
		tok := p.next()
		ctx := tok.ctx
		var exprs []ast.Expression
		exprs, tok = p.parseExprList(tok, false, false, false)
		if exprs == nil {
			panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
		}
		pos.End = exprs[len(exprs)-1].Pos().End
		var node ast.Node
		node = ast.NewShow(pos, exprs, ctx)
		if end == tokenEndStatement && tok.typ == tokenSemicolon {
			node, tok = p.parseUsing(node, tok)
		}
		if len(exprs) == 1 {
			if _, ok := exprs[0].(*ast.Render); ok {
				p.cutSpacesToken = true
			}
		}
		p.addNode(node)
		tok = p.parseEnd(tok, tokenSemicolon, end)
		return tok

	// extends
	case tokenExtends:
		pos := tok.pos
		if tok.ctx != ast.Context(p.format) {
			panic(syntaxError(tok.pos, "extends not in %s content", ast.Context(p.format)))
		}
		if p.hasExtend {
			panic(syntaxError(tok.pos, "extends already exists"))
		}
		tree := p.ancestors[0].(*ast.Tree)
		for _, node := range tree.Nodes {
			switch n := node.(type) {
			case *ast.Comment:
			case *ast.Text:
				if !containsOnlySpaces(n.Text) {
					panic(syntaxError(tok.pos, "extends is not at the beginning of the file"))
				}
			case *ast.Statements:
				if n != p.parent() || len(n.Nodes) > 0 {
					panic(syntaxError(tok.pos, "extends is not at the beginning of the file"))
				}
			default:
				panic(syntaxError(tok.pos, "extends is not at the beginning of the file"))
			}
		}
		tok = p.next()
		if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
			panic(syntaxError(tok.pos, "unexpected %s, expecting string", tok))
		}
		var path = unquoteString(tok.txt)
		if !ValidTemplatePath(path) {
			panic(syntaxError(tok.pos, "invalid extends path %q", path))
		}
		pos.End = tok.pos.End
		node := ast.NewExtends(pos, path, p.format)
		p.unexpanded = append(p.unexpanded, node)
		p.addNode(node)
		p.hasExtend = true
		tok = p.next()
		tok = p.parseEnd(tok, tokenSemicolon, end)
		return tok

	// var or const
	case tokenVar, tokenConst:
		pos := tok.pos
		decType := tok.typ
		tok = p.next()
		if tok.typ == tokenLeftParenthesis {
			// var ( ... )
			// const ( ... )
			var prevNode ast.Node
			var prevConstValues []ast.Expression
			var prevConstType ast.Expression
			tok = p.next()
			iotaValue := 0
			for {
				if tok.typ == tokenRightParenthesis {
					tok = p.next()
					if prevNode != nil {
						prevNode.Pos().End = tok.pos.End
					}
					tok = p.parseEnd(tok, tokenSemicolon, end)
					break
				}
				prevNode, tok = p.parseVarOrConst(tok, pos, decType, iotaValue)
				switch tok.typ {
				case tokenSemicolon:
					tok = p.next()
				case tokenRightParenthesis:
				default:
					panic(syntaxError(tok.pos, "unexpected %s, expecting semicolon or newline or )", tok))
				}
				if c, ok := prevNode.(*ast.Const); ok {
					if c.Type == nil {
						c.Type = astutil.CloneExpression(prevConstType)
					}
					if len(c.Rhs) == 0 {
						c.Rhs = make([]ast.Expression, len(prevConstValues))
						for i := range prevConstValues {
							c.Rhs[i] = astutil.CloneExpression(prevConstValues[i])
						}
					}
					prevConstValues = c.Rhs
					prevConstType = c.Type
				}
				p.addNode(prevNode)
				iotaValue++
			}
		} else {
			var node ast.Node
			node, tok = p.parseVarOrConst(tok, pos, decType, 0)
			if decType == tokenVar && end == tokenEndStatement && tok.typ == tokenSemicolon {
				node, tok = p.parseUsing(node, tok)
			}
			p.addNode(node)
			tok = p.parseEnd(tok, tokenSemicolon, end)
		}
		return tok

	// import
	case tokenImport:
		switch parent := p.parent().(type) {
		case *ast.Tree:
			if !lastImportOrExtends(parent.Nodes) {
				panic(syntaxError(tok.pos, "unexpected import, expecting statement"))
			}
		case *ast.Package:
			for _, declaration := range parent.Declarations {
				if _, ok := declaration.(*ast.Import); !ok {
					panic(syntaxError(tok.pos, "non-declaration statement outside function body"))
				}
			}
		case *ast.Statements:
			tree := p.ancestors[0].(*ast.Tree)
			if len(p.ancestors) != 2 || !lastImportOrExtends(tree.Nodes) {
				panic(syntaxError(tok.pos, "unexpected import, expecting %%%%}"))
			}
		default:
			if end != tokenEndStatement {
				panic(syntaxError(tok.pos, "unexpected import, expecting }"))
			}
			panic(syntaxError(tok.pos, "unexpected import, expecting statement"))
		}
		if tok.ctx != ast.Context(p.format) {
			panic(syntaxError(tok.pos, "import not in %s content", ast.Context(p.format)))
		}
		tok = p.next()
		if tok.typ == tokenLeftParenthesis && end != tokenEndStatement {
			tok = p.next()
			for tok.typ != tokenRightParenthesis {
				var node ast.Node
				node, tok = p.parseImport(tok, end)
				p.unexpanded = append(p.unexpanded, node)
				p.addNode(node)
				if tok.typ == tokenSemicolon {
					tok = p.next()
				} else if tok.typ != tokenRightParenthesis {
					panic(syntaxError(tok.pos, "expected ';', found %s", tok))
				}
			}
			tok = p.next()
			if tok.typ != tokenSemicolon {
				panic(syntaxError(tok.pos, "unexpected %s, expecting semicolon or newline", tok))
			}
			tok = p.next()
		} else {
			var node ast.Node
			node, tok = p.parseImport(tok, end)
			p.unexpanded = append(p.unexpanded, node)
			p.addNode(node)
			tok = p.parseEnd(tok, tokenSemicolon, end)
		}
		p.cutSpacesToken = true
		return tok

	// macro
	case tokenMacro:
		if end == tokenEndStatements {
			panic(syntaxError(tok.pos, "unexpected macro in statement scope"))
		}
		if tok.ctx > ast.ContextMarkdown {
			panic(syntaxError(tok.pos, "macro declaration not allowed in %s", tok.ctx))
		}
		if tok.ctx != ast.Context(p.format) {
			panic(syntaxError(tok.pos, "macro not in %s content", ast.Context(p.format)))
		}
		var node ast.Node
		node, tok = p.parseFunc(tok, parseFuncDecl)
		if tok.typ != tokenEndStatement {
			if node.(*ast.Func).Type.Result == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting string, html, css, js, json, markdown or %%}", tok))
			}
			panic(syntaxError(tok.pos, "unexpected %s, expecting %%}", tok))
		}
		p.addNode(node)
		p.cutSpacesToken = true
		return p.next()

	// end
	case tokenEnd:
		switch p.parent().(type) {
		case *ast.Tree, *ast.URL:
			panic(syntaxError(tok.pos, "unexpected %s", tok))
		case *ast.Block:
			if len(p.ancestors) == 3 {
				if fn, ok := p.ancestors[1].(*ast.Func); ok && fn.Endless {
					panic(syntaxError(tok.pos, "unexpected %s", tok))
				}
			}
			p.removeLastAncestor()
		}
		pos := tok.pos
		tok = p.next()
		if tok.typ != tokenEndStatement {
			switch n := p.parent().(type) {
			case *ast.For, *ast.ForRange, *ast.ForIn:
				if tok.typ != tokenFor {
					panic(syntaxError(pos, "unexpected %s, expecting for or %%}", tok))
				}
			case *ast.If:
				if tok.typ != tokenIf {
					panic(syntaxError(pos, "unexpected %s, expecting if or %%}", tok))
				}
			case *ast.Func:
				if tok.typ != tokenMacro {
					panic(syntaxError(pos, "unexpected %s, expecting macro or %%}", tok))
				}
			case *ast.Switch, *ast.TypeSwitch:
				if tok.typ != tokenSwitch {
					panic(syntaxError(pos, "unexpected %s, expecting switch or %%}", tok))
				}
			case *ast.Select:
				if tok.typ != tokenSelect {
					panic(syntaxError(pos, "unexpected %s, expecting select or %%}", tok))
				}
			case *ast.Raw:
				if n.Marker != "" {
					if tok.typ != tokenRaw {
						panic(syntaxError(pos, "unexpected %s, expecting raw", tok))
					}
					tok = p.next()
				}
			case *ast.Using:
				if tok.typ != tokenUsing {
					panic(syntaxError(pos, "unexpected %s, expecting using or %%}", tok))
				}
			default:
				panic(syntaxError(pos, "unexpected %s, expecting %%}", tok))
			}
			pos = tok.pos
			tok = p.next()
			if tok.typ != tokenEndStatement {
				panic(syntaxError(tok.pos, "unexpected %s, expecting %%}", tok))
			}
		}
		p.parent().Pos().End = pos.End
		p.removeLastAncestor()
		for {
			if n, ok := p.parent().(*ast.If); ok {
				n.Pos().End = pos.End
				p.removeLastAncestor()
				continue
			}
			break
		}
		p.cutSpacesToken = true
		return p.next()

	// type
	case tokenType:
		pos := tok.pos
		tok = p.next()
		if tok.typ == tokenLeftParenthesis {
			// "type" "(" ... ")" .
			tok = p.next()
			for tok.typ != tokenRightParenthesis {
				var node *ast.TypeDeclaration
				node, tok = p.parseTypeDecl(tok)
				node.Position = pos
				p.addNode(node)
				if tok.typ == tokenSemicolon {
					tok = p.next()
				} else if tok.typ != tokenRightParenthesis {
					panic(syntaxError(tok.pos, "unexpected %s, expecting semicolon or newline or )", tok))
				}
			}
			pos.End = tok.pos.End
			tok = p.next()
		} else {
			// "type" identifier [ "=" ] type .
			var node *ast.TypeDeclaration
			node, tok = p.parseTypeDecl(tok)
			pos.End = tok.pos.End
			node.Position = pos
			p.addNode(node)
		}
		tok = p.parseEnd(tok, tokenSemicolon, end)
		return tok

	// defer or go
	case tokenDefer, tokenGo:
		pos := tok.pos
		keyword := tok.typ
		tok = p.next()
		var expr ast.Expression
		expr, tok = p.parseExpr(tok, false, false, false)
		if expr == nil {
			panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
		}
		pos.End = expr.Pos().End
		var node ast.Node
		if keyword == tokenDefer {
			node = ast.NewDefer(pos, expr)
		} else {
			node = ast.NewGo(pos, expr)
		}
		p.addNode(node)
		tok = p.parseEnd(tok, tokenSemicolon, end)
		return tok

	// goto
	case tokenGoto:
		pos := tok.pos
		if end != tokenEOF {
			panic(syntaxError(tok.pos, "goto statement outside function body"))
		}
		tok = p.next()
		if tok.typ != tokenIdentifier {
			panic(syntaxError(tok.pos, "unexpected %s, expecting name", tok))
		}
		pos.End = tok.pos.End
		node := ast.NewGoto(pos, ast.NewIdentifier(tok.pos, string(tok.txt)))
		p.addNode(node)
		tok = p.next()
		tok = p.parseEnd(tok, tokenSemicolon, end)
		return tok

	// raw
	case tokenRaw:
		if end == tokenEndStatements {
			panic(syntaxError(tok.pos, "cannot use raw between {%%%% and %%%%}"))
		}
		if tok.ctx > ast.ContextMarkdown {
			panic(syntaxError(tok.pos, "cannot use raw in %s", tok.ctx))
		}
		pos := tok.pos
		tok = p.next()
		var marker, tag string
		var tagged bool
		if tok.typ == tokenIdentifier {
			if len(tok.txt) == 1 && tok.txt[0] == '_' {
				panic(syntaxError(tok.pos, "cannot use _ as marker"))
			}
			marker = string(tok.txt)
			tok = p.next()
		}
		if tok.typ == tokenRawString || tok.typ == tokenInterpretedString {
			tagged = true
			tag = unquoteString(tok.txt)
			tok = p.next()
		}
		if !tagged && tok.typ != tokenEndStatement {
			if marker == "" {
				panic(syntaxError(tok.pos, "unexpected %s, expecting identifier, string or %%}", tok))
			}
			panic(syntaxError(tok.pos, "unexpected %s, expecting string or %%}", tok))
		}
		node := ast.NewRaw(pos, marker, tag, nil)
		p.addNode(node)
		p.cutSpacesToken = true
		tok = p.parseEnd(tok, tokenSemicolon, end)
		return tok

	// func
	case tokenFunc:
		if end != tokenEndStatement {
			// Note that parseFunc does not consume the next token in this case
			// because kind is not parseType.
			switch p.parent().(type) {
			case *ast.Tree, *ast.Package:
				node, tok := p.parseFunc(tok, parseFuncDecl)
				if tok.typ != tokenSemicolon {
					panic(syntaxError(tok.pos, "unexpected %s after top level declaration", tok))
				}
				p.addNode(node)
				return p.next()
			}
		}
		fallthrough

	// assignment, send, label or expression
	default:
		pos := tok.pos
		var expressions []ast.Expression
		expressions, tok = p.parseExprList(tok, false, false, false)
		if expressions == nil {
			// There is no statement or it is not a statement.
			if end != tokenEndStatement {
				switch p.parent().(type) {
				case *ast.Tree:
					panic(syntaxError(tok.pos, "unexpected %s, expected statement", tok))
				case *ast.Label:
					panic(syntaxError(tok.pos, "missing statement after label"))
				}
				panic(syntaxError(tok.pos, "unexpected %s, expecting }", tok))
			}
			if tok.typ == tokenEndStatement {
				panic(syntaxError(tok.pos, "missing statement"))
			}
			panic(syntaxError(tok.pos, "unexpected %s, expecting statement", tok))
		}
		if len(expressions) > 1 || isAssignmentToken(tok) {
			// Parse assignment.
			var node ast.Node
			node, tok = p.parseAssignment(expressions, tok, false, false, false)
			node.(*ast.Assignment).Position = pos.WithEnd(node.Pos().End)
			if end == tokenEndStatement && tok.typ == tokenSemicolon {
				if t := node.(*ast.Assignment).Type; t != ast.AssignmentIncrement && t != ast.AssignmentDecrement {
					node, tok = p.parseUsing(node, tok)
				}
			}
			p.addNode(node)
			p.cutSpacesToken = true
			tok = p.parseEnd(tok, tokenSemicolon, end)
		} else if tok.typ == tokenArrow {
			// Parse send.
			channel := expressions[0]
			var value ast.Expression
			value, tok = p.parseExpr(p.next(), false, false, false)
			if value == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			var node ast.Node = ast.NewSend(pos, channel, value)
			node.(*ast.Send).Position = pos.WithEnd(value.Pos().End)
			if end == tokenEndStatement && tok.typ == tokenSemicolon {
				node, tok = p.parseUsing(node, tok)
			}
			p.addNode(node)
			p.cutSpacesToken = true
			tok = p.parseEnd(tok, tokenSemicolon, end)
		} else {
			// Parse a label or an expression.
			expr := expressions[0]
			if ident, ok := expr.(*ast.Identifier); ok {
				switch tok.typ {
				case tokenColon:
					// Parse a label.
					node := ast.NewLabel(pos, ident, nil)
					node.Position = pos.WithEnd(tok.pos.End)
					p.addNode(node)
					p.cutSpacesToken = true
					tok = p.next()
					if end == tokenEndStatement {
						if tok.typ == tokenEndStatement || tok.typ == tokenEOF {
							panic(syntaxError(tok.pos, "missing statement after label"))
						}
						goto LABEL
					}
					if tok.typ == tokenSemicolon && tok.txt != nil {
						p.removeLastAncestor()
						tok = p.next()
					} else if tok.typ == tokenEndStatements {
						p.removeLastAncestor()
					}
					return tok
				case tokenIdentifier:
					if end == tokenEndStatement {
						tok2 := p.next() // It is safe to consume the next token because there is anyway a syntax error.
						if tok2.typ == tokenInterpretedString || tok2.typ == tokenRawString {
							panic(syntaxError(ident.Pos(), "unexpected %s, expecting import", ident.Name))
						}
					}
				case tokenInterpretedString, tokenRawString:
					if end == tokenEndStatement {
						panic(syntaxError(ident.Pos(), "unexpected %s, expecting extends or import", ident.Name))
					}
				}
			}
			var node ast.Node = expr
			if end == tokenEndStatement && tok.typ == tokenSemicolon {
				node, tok = p.parseUsing(node, tok)
			}
			p.addNode(node)
			p.cutSpacesToken = true
			tok = p.parseEnd(tok, tokenSemicolon, end)
		}
		return tok
	}

}

// parseUsing tries to parse a using statement and in case returns a Using
// node, otherwise it returns stmt which is the previous parsed statement.
// tok is the semicolon that follows stmt.
func (p *parsing) parseUsing(stmt ast.Node, tok token) (ast.Node, token) {
	start := tok
	if tok = p.next(); tok.typ != tokenUsing {
		if start.txt == nil {
			if tok.typ == tokenEndStatement {
				return stmt, tok
			}
			panic(syntaxError(tok.pos, "unexpected %s, expecting %%}", tok))
		} else if tok.typ == tokenEndStatement {
			panic(syntaxError(start.pos, "unexpected semicolon, expecting %%}"))
		}
		panic(syntaxError(tok.pos, "unexpected %s, expecting using", tok))
	}
	if tok.ctx > ast.ContextMarkdown {
		panic(syntaxError(tok.pos, "using not allowed in %s", tok.ctx))
	}
	block := ast.NewBlock(nil, []ast.Node{})
	using := ast.NewUsing(tok.pos, stmt, nil, block, ast.Format(tok.ctx))
	switch tok = p.next(); tok.typ {
	case tokenMacro:
		var macro ast.Node
		macro, tok = p.parseFunc(tok, parseFuncLit)
		using.Type = macro.(*ast.Func).Type
		if len(using.Type.(*ast.FuncType).Result) > 0 {
			using.Format = macro.(*ast.Func).Format
		}
	case tokenIdentifier:
		ident := p.parseIdentifierNode(tok)
		for i, name := range formatTypeName {
			if name == ident.Name {
				using.Type = ident
				using.Format = ast.Format(i)
			}
		}
		if using.Type == nil {
			panic(syntaxError(tok.pos, "unexpected %s, expecting string, html, css, js, json, markdown, macro or %%}", ident.Name))
		}
		tok = p.next()
	case tokenSemicolon, tokenEndStatement:
	default:
		panic(syntaxError(tok.pos, "unexpected %s, expecting identifier, macro or %%}", tok))
	}
	return using, tok
}

func (p *parsing) parseEnd(tok token, want, end tokenTyp) token {
	if end == tokenEndStatement {
		if tok.typ == tokenSemicolon && tok.txt == nil {
			tok = p.next()
		}
		if tok.typ != tokenEndStatement {
			panic(syntaxError(tok.pos, "unexpected %s, expecting %%}", tok))
		}
		return p.next()
	}
	if want == tokenSemicolon {
		switch tok.typ {
		case tokenSemicolon:
			return p.next()
		case tokenRightBrace, tokenEndStatements:
			return tok
		}
		panic(syntaxError(tok.pos, "unexpected %s at end of statement", tok))
	}
	if tok.typ == want {
		return p.next()
	}
	panic(syntaxError(tok.pos, "unexpected %s, expecting %s", tok, want))
}

// parseSimpleStatement parses a simple statement. tok is the next token read.
// If canBeRange is true, as a special case, parseSimpleStatement parses an
// assignment where the assignment token is followed by a range token.
func (p *parsing) parseSimpleStatement(tok token, canBeRange, nextIsBlockOpen bool) (ast.Node, token) {
	pos := tok.pos
	expressions, tok := p.parseExprList(tok, false, false, nextIsBlockOpen)
	if expressions == nil {
		// Empty statement.
		return nil, tok
	}
	if len(expressions) > 1 || isAssignmentToken(tok) {
		// Assignment statement.
		var assignment *ast.Assignment
		assignment, tok = p.parseAssignment(expressions, tok, canBeRange, false, nextIsBlockOpen)
		assignment.Position = pos.WithEnd(assignment.Pos().End)
		return assignment, tok
	}
	if tok.typ == tokenArrow {
		// Send statement.
		channel := expressions[0]
		var value ast.Expression
		value, tok = p.parseExpr(p.next(), false, false, nextIsBlockOpen)
		if value == nil {
			panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
		}
		send := ast.NewSend(pos, channel, value)
		send.Position = pos.WithEnd(value.Pos().End)
		return send, tok
	}
	// Expression statement.
	return expressions[0], tok
}

// parseIdentifiersList returns a list of identifiers separated by commas and
// the next token not used.
func (p *parsing) parseIdentifiersList(tok token) ([]*ast.Identifier, token) {
	idents := []*ast.Identifier{}
	for {
		idents = append(idents, p.parseIdentifierNode(tok))
		tok = p.next()
		if tok.typ == tokenComma {
			tok = p.next()
			continue
		}
		break
	}
	return idents, tok
}

// parseTypeDecl parses a type declaration, that is a type definition or an
// alias declaration. The token returned is the next valid token.
func (p *parsing) parseTypeDecl(tok token) (*ast.TypeDeclaration, token) {
	pos := tok.pos
	if tok.typ != tokenIdentifier {
		panic(syntaxError(tok.pos, "unexpected %s, expecting name", tok))
	}
	ident := ast.NewIdentifier(tok.pos, string(tok.txt))
	tok = p.next()
	alias := tok.typ == tokenSimpleAssignment
	if alias {
		tok = p.next()
	}
	var typ ast.Expression
	typ, tok = p.parseExpr(tok, false, true, false)
	if typ == nil {
		panic(syntaxError(tok.pos, "unexpected %s in type declaration", tok))
	}
	node := ast.NewTypeDeclaration(pos, ident, typ, alias)
	return node, tok
}

// group is nil if parseVarOrConst is called when not in a declaration group.
func (p *parsing) parseVarOrConst(tok token, pos *ast.Position, decType tokenTyp, iotaValue int) (ast.Node, token) {
	if tok.typ != tokenIdentifier {
		panic(syntaxError(tok.pos, "unexpected %s, expecting name", tok))
	}
	var exprs []ast.Expression
	var idents []*ast.Identifier
	var typ ast.Expression
	idents, tok = p.parseIdentifiersList(tok)
	if len(idents) == 0 {
		panic(syntaxError(tok.pos, "unexpected %s, expecting name", tok))
	}
	switch tok.typ {
	case tokenSimpleAssignment:
		// var/const  a     = ...
		// var/const  a, b  = ...
		exprs, tok = p.parseExprList(p.next(), false, false, false)
		if exprs == nil {
			panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
		}
	case tokenIdentifier, tokenFunc, tokenMap, tokenLeftParenthesis, tokenLeftBracket, tokenInterface,
		tokenMultiplication, tokenChan, tokenArrow, tokenStruct, tokenMacro:
		// var  a     int
		// var  a, b  int
		// var/const  a     int  =  ...
		// var/const  a, b  int  =  ...
		typ, tok = p.parseExpr(tok, false, true, false)
		if tok.typ != tokenSimpleAssignment && decType == tokenConst {
			panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
		}
		if tok.typ == tokenSimpleAssignment {
			// var/const  a     int  =  ...
			// var/const  a, b  int  =  ...
			exprs, tok = p.parseExprList(p.next(), false, false, false)
			if exprs == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
		}
	case tokenSemicolon, tokenRightParenthesis:
		// const c
		// const c, d
		if decType == tokenVar {
			panic(syntaxError(tok.pos, "unexpected %s, expecting type", tok))
		}
	default:
		panic(syntaxError(tok.pos, "unexpected %s, expecting type", tok))
	}
	// Search the maximum end position.
	var exprPosEnd, typEndPos int
	if exprs != nil {
		exprPosEnd = exprs[len(exprs)-1].Pos().End
	}
	identPosEnd := idents[len(idents)-1].Pos().End
	if typ != nil {
		typEndPos = typ.Pos().End
	}
	endPos := exprPosEnd
	if identPosEnd > endPos {
		endPos = identPosEnd
	}
	if typEndPos > endPos {
		endPos = typEndPos
	}
	pos.End = endPos
	if decType == tokenVar {
		node := ast.NewVar(pos, idents, typ, exprs)
		return node, tok
	}
	return ast.NewConst(pos, idents, typ, exprs, iotaValue), tok
}

// parseEndlessMacro parses an endless macro declaration.
func (p *parsing) parseEndlessMacro(tok token, end tokenTyp) token {
	if tok.typ != tokenIdentifier || !p.hasExtend || end != tokenEndStatement {
		panic(syntaxError(tok.pos, "unexpected %s, expecting declaration statement", tok))
	}
	if fc, _ := utf8.DecodeRune(tok.txt); !unicode.Is(unicode.Lu, fc) {
		panic(syntaxError(tok.pos, "unexpected %s, expecting declaration statement", tok))
	}
	ident := p.parseIdentifierNode(tok)
	tok = p.next()
	if tok.typ != end {
		panic(syntaxError(tok.pos, "unexpected %s, expecting declaration statement", ident))
	}
	// Make the Func node.
	pos := ident.Pos()
	typ := ast.NewFuncType(pos.WithEnd(pos.End), true, nil, nil, false)
	node := ast.NewFunc(pos, ident, typ, nil, true, ast.Format(tok.ctx))
	body := ast.NewBlock(pos, nil)
	node.Body = body
	p.addNode(node)
	p.cutSpacesToken = true
	return p.next()
}

// lastImportOrExtends reports whether the last node in nodes is Import or
// Extends, excluding Text and Comment nodes. If a node is Statements, it
// looks into that node.
func lastImportOrExtends(nodes []ast.Node) bool {
	if len(nodes) == 0 {
		return true
	}
	for i := len(nodes) - 1; i >= 0; i-- {
		switch n := nodes[i].(type) {
		case *ast.Extends, *ast.Import:
			return true
		case *ast.Statements:
			if last := len(n.Nodes) - 1; last >= 0 {
				switch n.Nodes[last].(type) {
				case *ast.Extends, *ast.Import:
					return true
				}
				return false
			}
		case *ast.Text, *ast.Comment:
		default:
			return false
		}
	}
	return true
}

// parseImport parses an import declaration. tok if the first token of
// ImportSpec and end is the argument passed to the parse method.
// It returns the parsed node and the next not parsed token.
func (p *parsing) parseImport(tok token, end tokenTyp) (*ast.Import, token) {
	pos := tok.pos
	var ident *ast.Identifier
	switch tok.typ {
	case tokenIdentifier:
		ident = ast.NewIdentifier(tok.pos, string(tok.txt))
		tok = p.next()
	case tokenPeriod:
		ident = ast.NewIdentifier(tok.pos, ".")
		tok = p.next()
	}
	if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
		panic(syntaxError(tok.pos, "unexpected %s, expecting string", tok))
	}
	var path = unquoteString(tok.txt)
	if end == tokenEOF {
		validatePackagePath(path, tok.pos)
	} else {
		if !ValidTemplatePath(path) {
			panic(syntaxError(tok.pos, "invalid import path: %q", path))
		}
	}
	pos = pos.WithEnd(tok.pos.End)
	tok = p.next()
	var forIdents []*ast.Identifier
	if end != tokenEOF && ident == nil && tok.typ == tokenFor {
		for {
			tok = p.next()
			if tok.typ != tokenIdentifier {
				panic(syntaxError(tok.pos, "unexpected %s, expecting name", tok))
			}
			name := string(tok.txt)
			if r, _ := utf8.DecodeRuneInString(name); !unicode.Is(unicode.Lu, r) {
				panic(syntaxError(tok.pos, "cannot refer to unexported name %s", name))
			}
			forIdents = append(forIdents, ast.NewIdentifier(tok.pos, name))
			pos = pos.WithEnd(tok.pos.End)
			tok = p.next()
			if tok.typ != tokenComma {
				break
			}
		}
	}
	return ast.NewImport(pos, ident, path, forIdents), tok
}

// parseAssignment parses an assignment and returns an assignment or, if there
// is no expression, returns nil. tok is the assignment, declaration,
// increment or decrement token. canBeRange reports whether a rangeToken token
// is accepted after tok. canBeSwitchGuard and nextIsBlockOpen are passed to
// the parseExprList method. Panics on error.
func (p *parsing) parseAssignment(variables []ast.Expression, tok token, canBeRange, canBeSwitchGuard, nextIsBlockOpen bool) (*ast.Assignment, token) {
	typ, ok := assignmentType(tok)
	if !ok {
		panic(syntaxError(tok.pos, "unexpected %s, expecting := or = or comma", tok))
	}
	vp := variables[0].Pos()
	pos := vp.WithEnd(tok.pos.End)
	var values []ast.Expression
	switch typ {
	case ast.AssignmentSimple, ast.AssignmentDeclaration:
		tok = p.next()
		if tok.typ != tokenRange || !canBeRange {
			values, tok = p.parseExprList(tok, canBeSwitchGuard, false, nextIsBlockOpen)
			if values == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			pos.End = values[len(values)-1].Pos().End
		}
	default:
		if len(variables) > 1 {
			panic(syntaxError(tok.pos, "unexpected %s, expecting := or = or comma", tok))
		}
		tok = p.next()
		if typ != ast.AssignmentIncrement && typ != ast.AssignmentDecrement {
			values = make([]ast.Expression, 1)
			var expr ast.Expression
			expr, tok = p.parseExpr(tok, false, false, nextIsBlockOpen)
			if expr == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			values[0] = expr
			pos.End = values[0].Pos().End
		}
	}
	node := ast.NewAssignment(pos, variables, typ, values)
	return node, tok
}

// cannotUseAsValueError returns a syntax error returned when node is used as
// value at position pos.
func cannotUseAsValueError(pos *ast.Position, node ast.Node) *SyntaxError {
	if a, ok := node.(*ast.Assignment); ok && a.Type == ast.AssignmentSimple {
		return syntaxError(pos, "cannot use assignment (%s) = (%s) as value",
			printExpressions(a.Lhs), printExpressions(a.Rhs))
	}
	return syntaxError(pos, "cannot use %s as value", node)
}

// printExpressions prints expressions as a comma separated list.
func printExpressions(expressions []ast.Expression) string {
	var b strings.Builder
	for i, e := range expressions {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(e.String())
	}
	return b.String()
}

// addNode adds a node to the tree. It adds the node as a child to the current
// parent node and sets the node as new ancestor if necessary.
func (p *parsing) addNode(node ast.Node) {
	// Add the node as child of the current parent.
	switch n := p.parent().(type) {
	case *ast.Tree:
		n.Nodes = append(n.Nodes, node)
	case *ast.Package:
		n.Declarations = append(n.Declarations, node)
	case *ast.URL:
		n.Value = append(n.Value, node)
	case *ast.For:
		n.Body = append(n.Body, node)
	case *ast.ForIn:
		n.Body = append(n.Body, node)
	case *ast.ForRange:
		n.Body = append(n.Body, node)
	case *ast.If:
		if n.Else != nil {
			panic("node already added to if node")
		}
		n.Else = node
	case *ast.Block:
		n.Nodes = append(n.Nodes, node)
	case *ast.Statements:
		n.Nodes = append(n.Nodes, node)
	case *ast.Switch:
		c, ok := node.(*ast.Case)
		if ok {
			n.Cases = append(n.Cases, c)
		} else {
			lastCase := n.Cases[len(n.Cases)-1]
			lastCase.Body = append(lastCase.Body, node)
		}
	case *ast.TypeSwitch:
		c, ok := node.(*ast.Case)
		if ok {
			n.Cases = append(n.Cases, c)
			return
		}
		lastCase := n.Cases[len(n.Cases)-1]
		lastCase.Body = append(lastCase.Body, node)
	case *ast.Select:
		cc, ok := node.(*ast.SelectCase)
		if ok {
			n.Cases = append(n.Cases, cc)
		} else {
			lastCase := n.Cases[len(n.Cases)-1]
			lastCase.Body = append(lastCase.Body, node)
		}
	case *ast.Label:
		n.Statement = node
		n.Pos().End = node.Pos().End
		p.removeLastAncestor()
	case *ast.Raw:
		n.Text = node.(*ast.Text)
	default:
		panic("scriggo/parser: unexpected parent node")
	}
	// Set the node as new ancestor if necessary.
	switch n := node.(type) {
	case *ast.If:
		p.addToAncestors(n)
		p.addToAncestors(n.Then)
	case *ast.Func:
		if n.Type.Macro {
			p.addToAncestors(n)
			p.addToAncestors(n.Body)
		}
	case *ast.Using:
		p.addToAncestors(n)
		p.addToAncestors(n.Body)
	case
		*ast.Block,
		*ast.For,
		*ast.ForRange,
		*ast.ForIn,
		*ast.Switch,
		*ast.TypeSwitch,
		*ast.Select,
		*ast.Package,
		*ast.Label,
		*ast.Statements,
		*ast.URL,
		*ast.Raw:
		p.addToAncestors(n)
	}
}

// cutSpaces cuts the leading and trailing spaces from a line. first and last
// are respectively the initial and the final Text node of the line.
func cutSpaces(first, last *ast.Text) {
	var firstCut int
	if first != nil {
		// So that spaces can be cut, first.Text must only contain '', '\t' and '\r',
		// or after the last '\n' must only contain '', '\t' and '\r'.
		txt := first.Text
		for i := len(txt) - 1; i >= 0; i-- {
			c := txt[i]
			if c == '\n' {
				firstCut = i + 1
				break
			}
			if c != ' ' && c != '\t' && c != '\r' {
				return
			}
		}
	}
	if last != nil {
		// So that the spaces can be cut, last.Text must contain only '', '\t' and '\r',
		// or before the first '\n' must only contain '', '\t' and '\r'.
		txt := last.Text
		var lastCut = len(txt)
		for i := 0; i < len(txt); i++ {
			c := txt[i]
			if c == '\n' {
				lastCut = i + 1
				break
			}
			if c != ' ' && c != '\t' && c != '\r' {
				return
			}
		}
		last.Cut.Left = lastCut
	}
	if first != nil {
		first.Cut.Right = len(first.Text) - firstCut
	}
}
