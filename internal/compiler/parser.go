// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"

	"scriggo/ast"
	"scriggo/ast/astutil"
)

type Reader interface {
	Read(path string) ([]byte, error)
}

var (
	// ErrInvalidPath is returned from the Parse method and a Reader when the
	// path argument is not valid.
	ErrInvalidPath = errors.New("scriggo: invalid path")

	// ErrNotExist is returned from the Parse method and a Reader when the
	// path does not exist.
	ErrNotExist = errors.New("scriggo: path does not exist")

	// ErrReadTooLarge is returned from a DirLimitedReader when a limit is
	// exceeded.
	ErrReadTooLarge = errors.New("scriggo: read too large")
)

var (
	orIdent     = []byte("or")
	ignoreIdent = []byte("ignore")
	todoIdent   = []byte("todo")
	errorIdent  = []byte("error")
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

// cycleError implements an error indicating the presence of a cycle.
type cycleError string

func (e cycleError) Error() string {
	return fmt.Sprintf("cycle not allowed\n%s", string(e))
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

// parsing is a parsing state.
type parsing struct {

	// Lexer.
	lex *lexer

	// Reports whether it is a template.
	isTemplate bool

	// Report whether it is a script.
	isScript bool

	// Reports whether it has an extend statement.
	hasExtend bool

	// Reports whether there is a token in current line for which it is possible
	// to cut the leading and trailing spaces.
	cutSpacesToken bool

	// Context.
	ctx ast.Context

	// Ancestors from the root up to the parent.
	ancestors []ast.Node
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

// next returns the next token from the lexer. Panics if the lexer channel is
// closed.
func (p *parsing) next() token {
	tok, ok := <-p.lex.tokens
	if !ok {
		if p.lex.err == nil {
			panic("next called after EOF")
		}
		panic(p.lex.err)
	}
	return tok
}

// ParseSource parses a program or script and returns its tree. isScript
// reports whether it is a script and shebang reports whether a script can
// have the shebang as first line.
func ParseSource(src []byte, isScript, shebang bool) (tree *ast.Tree, err error) {

	if shebang && !isScript {
		return nil, errors.New("scriggo/parser: shebang can be true only for scripts")
	}

	tree = ast.NewTree("", nil, ast.ContextGo)

	var p = &parsing{
		lex:       newLexer(src, ast.ContextGo),
		isScript:  isScript,
		ctx:       ast.ContextGo,
		ancestors: []ast.Node{tree},
	}

	defer func() {
		p.lex.drain()
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
		if !shebang {
			return nil, syntaxError(tok.pos, "illegal character U+0023 '#'")
		}
		tok = p.next()
	}

	for tok.typ != tokenEOF {
		tok = p.parse(tok)
	}

	if len(p.ancestors) > 1 {
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

// ParseTemplateSource parses a template with source src in the context ctx
// and returns its tree. ctx can be ContextText, ContextHTML, ContextCSS or
// ContextJavaScript.
//
// ParseTemplateSource does not expand the nodes Extends, Include and Import.
func ParseTemplateSource(src []byte, ctx ast.Context) (tree *ast.Tree, err error) {

	switch ctx {
	case ast.ContextText, ast.ContextHTML, ast.ContextCSS, ast.ContextJavaScript:
	default:
		return nil, errors.New("scriggo: invalid context. Valid contexts are Text, HTML, CSS and JavaScript")
	}

	tree = ast.NewTree("", nil, ctx)

	var p = &parsing{
		lex:        newLexer(src, ctx),
		ctx:        ctx,
		ancestors:  []ast.Node{tree},
		isTemplate: true,
	}

	defer func() {
		p.lex.drain()
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

		switch tok.typ {

		// Text
		case tokenText:
			switch n := p.parent().(type) {
			case *ast.Switch:
				if len(n.Cases) == 0 {
					if containsOnlySpaces(text.Text) {
						n.LeadingText = text
					}
					return nil, syntaxError(tok.pos, "unexpected text, expecting case of default or {%% end %%}")
				}
			case *ast.TypeSwitch:
				if len(n.Cases) == 0 {
					if containsOnlySpaces(text.Text) {
						n.LeadingText = text
					}
					return nil, syntaxError(tok.pos, "unexpected text, expecting case of default or {%% end %%}")
				}
			case *ast.Select:
				if len(n.Cases) == 0 {
					if containsOnlySpaces(text.Text) {
						n.LeadingText = text
					}
					return nil, syntaxError(tok.pos, "unexpected text, expecting case of default or {%% end %%}")
				}
			}
			p.addChild(text)
			tok = p.next()

		// {%
		case tokenStartBlock:
			numTokenInLine++
			tok = p.next()
			tok = p.parse(tok)

		// {{
		case tokenStartValue:
			numTokenInLine++
			pos := tok.pos
			var expr ast.Expression
			expr, tok = p.parseExpr(p.next(), false, false, false)
			if expr == nil {
				return nil, syntaxError(tok.pos, "unexpected %s, expecting expression", tok)
			}
			if tok.typ != tokenEndValue {
				return nil, syntaxError(tok.pos, "unexpected %s, expecting }}", tok)
			}
			pos.End = tok.pos.End
			var node = ast.NewShow(pos, expr, tok.ctx)
			p.addChild(node)
			tok = p.next()

		// StartURL
		case tokenStartURL:
			node := ast.NewURL(tok.pos, tok.tag, tok.att, nil, tok.ctx)
			p.addChild(node)
			p.addToAncestors(node)
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
			p.addChild(node)
			p.cutSpacesToken = true
			tok = p.next()

		default:
			return nil, syntaxError(tok.pos, "unexpected %s", tok)

		}

	}

	if len(p.ancestors) > 1 {
		return nil, syntaxError(tok.pos, "unexpected EOF, expecting {%% end %%}")
	}

	return tree, nil
}

// parse parses a statement or part of it given its first token. Returns the
// first not parsed token.
//
// For a template the parsed code is the source code between {% and %},
// and the returned token is the first token after %}.
func (p *parsing) parse(tok token) token {

LABEL:

	if p.isTemplate {
		switch s := p.parent().(type) {
		case *ast.Switch:
			if len(s.Cases) == 0 && tok.typ != tokenCase && tok.typ != tokenDefault && tok.typ != tokenEnd {
				panic(syntaxError(tok.pos, "unexpected %s, expecting case or default or end", tok))
			}
		case *ast.TypeSwitch:
			if len(s.Cases) == 0 && tok.typ != tokenCase && tok.typ != tokenDefault && tok.typ != tokenEnd {
				panic(syntaxError(tok.pos, "unexpected %s, expecting case or default or end", tok))
			}
		case *ast.Select:
			if len(s.Cases) == 0 && tok.typ != tokenCase && tok.typ != tokenDefault && tok.typ != tokenEnd {
				panic(syntaxError(tok.pos, "unexpected %s, expecting case or default or end", tok))
			}
		case *ast.Label:
			switch tok.typ {
			case tokenFor, tokenSwitch, tokenSelect:
			default:
				panic(syntaxError(tok.pos, "unexpected %s, expecting for, switch or select", tok))
			}
		}
	} else {
		switch s := p.parent().(type) {
		case *ast.Tree:
			if !p.isScript && tok.typ != tokenPackage {
				panic(syntaxError(tok.pos, "expected 'package', found '%s'", tok))
			}
		case *ast.Package:
			switch tok.typ {
			case tokenImport, tokenFunc, tokenVar, tokenConst, tokenType:
			default:
				panic(syntaxError(tok.pos, "non-declaration statement outside function body"))
			}
		case *ast.Switch:
			if len(s.Cases) == 0 && tok.typ != tokenCase && tok.typ != tokenDefault && tok.typ != tokenRightBraces {
				panic(syntaxError(tok.pos, "unexpected %s, expecting case or default or }", tok))
			}
		case *ast.TypeSwitch:
			if len(s.Cases) == 0 && tok.typ != tokenCase && tok.typ != tokenDefault && tok.typ != tokenRightBraces {
				panic(syntaxError(tok.pos, "unexpected %s, expecting case or default or }", tok))
			}
		case *ast.Select:
			if len(s.Cases) == 0 && tok.typ != tokenCase && tok.typ != tokenDefault && tok.typ != tokenRightBraces {
				panic(syntaxError(tok.pos, "unexpected %s, expecting case or default or }", tok))
			}
		}
	}

	switch tok.typ {

	// ;
	case tokenSemicolon:
		if p.ctx != ast.ContextGo {
			panic(syntaxError(tok.pos, "unexpected semicolon, expecting statement"))
		}
		return p.next()

	// package
	case tokenPackage:
		pos := tok.pos
		if tree, ok := p.parent().(*ast.Tree); !ok || p.isTemplate || p.isScript || len(tree.Nodes) > 0 {
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
		p.addChild(node)
		p.addToAncestors(node)
		tok = p.parseEnd(tok, tokenSemicolon)
		return tok

	// for
	case tokenFor:
		pos := tok.pos
		var node ast.Node
		var init *ast.Assignment
		var assignmentType ast.AssignmentType
		var variables []ast.Expression
		variables, tok = p.parseExprList(p.next(), false, false, true)
		switch tok.typ {
		case tokenIn:
			// Parses statement "for ident in expr".
			if len(variables) == 0 {
				panic(syntaxError(variables[1].Pos(), "unexpected in, expected expression"))
			}
			if len(variables) > 1 {
				panic(syntaxError(variables[1].Pos(), "expected only one expression"))
			}
			ident, ok := variables[0].(*ast.Identifier)
			if !ok {
				panic(syntaxError(tok.pos, "unexpected in, expected assignment"))
			}
			if ident.Name == "_" {
				panic(syntaxError(ident.Pos(), "cannot use _ as value"))
			}
			ipos := ident.Pos()
			blank := ast.NewIdentifier(&ast.Position{Line: ipos.Line, Column: ipos.Column, Start: ipos.Start, End: ipos.Start}, "_")
			// Parses the slice expression.
			// TODO (Gianluca): nextIsBlockOpen should be true?
			var expr ast.Expression
			expr, tok = p.parseExpr(p.next(), false, false, false)
			if expr == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			assignment := ast.NewAssignment(&ast.Position{Line: ipos.Line, Column: ipos.Column, Start: ipos.Start, End: expr.Pos().End},
				[]ast.Expression{blank, ident}, ast.AssignmentDeclaration, []ast.Expression{expr})
			pos.End = tok.pos.End
			node = ast.NewForRange(pos, assignment, nil)
		case tokenLeftBraces, tokenEndBlock:
			if (p.ctx == ast.ContextGo) != (tok.typ == tokenLeftBraces) {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression or %%}", tok))
			}
			// Parses statement "for".
			if len(variables) > 1 {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			var condition ast.Expression
			if len(variables) == 1 {
				condition = variables[0]
			}
			pos.End = tok.pos.End
			node = ast.NewFor(pos, nil, condition, nil, nil)
		case tokenRange:
			// Parses "for range expr".
			if len(variables) > 0 {
				panic(syntaxError(tok.pos, "unexpected range, expecting := or = or comma"))
			}
			tpos := tok.pos
			// TODO (Gianluca): nextIsBlockOpen should be true?
			var expr ast.Expression
			expr, tok = p.parseExpr(p.next(), false, false, true)
			if expr == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			tpos.End = expr.Pos().End
			assignment := ast.NewAssignment(tpos, nil, ast.AssignmentSimple, []ast.Expression{expr})
			pos.End = tok.pos.End
			node = ast.NewForRange(pos, assignment, nil)
		case tokenSimpleAssignment, tokenDeclaration, tokenIncrement, tokenDecrement:
			if len(variables) == 0 {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			if tok.typ == tokenDeclaration {
				assignmentType = ast.AssignmentDeclaration
			}
			init, tok = p.parseAssignment(variables, tok, false, false)
			if init == nil && tok.typ != tokenRange {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			fallthrough
		case tokenSemicolon:
			if tok.typ == tokenRange {
				// Parses statements
				//     "for index[, ident] = range expr" and
				//     "for index[, ident] := range expr".
				var expr ast.Expression
				expr, tok = p.parseExpr(p.next(), false, false, true)
				if expr == nil {
					panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
				}
				vpos := variables[0].Pos()
				assignment := ast.NewAssignment(&ast.Position{Line: vpos.Line, Column: vpos.Column, Start: vpos.Start, End: expr.Pos().End},
					variables, assignmentType, []ast.Expression{expr})
				pos.End = tok.pos.End
				node = ast.NewForRange(pos, assignment, nil)
			} else {
				// Parses statement "for [init]; [condition]; [post]".
				// Parses the condition expression.
				var condition ast.Expression
				condition, tok = p.parseExpr(p.next(), false, false, true)
				if tok.typ != tokenSemicolon {
					panic(syntaxError(tok.pos, "unexpected %s, expected semicolon", tok))
				}
				// Parses the post iteration statement.
				var post *ast.Assignment
				variables, tok = p.parseExprList(p.next(), false, false, true)
				if len(variables) > 0 {
					pos := tok.pos
					post, tok = p.parseAssignment(variables, tok, false, true)
					if post == nil {
						panic(syntaxError(tok.pos, "expecting expression"))
					}
					if post.Type == ast.AssignmentDeclaration {
						panic(syntaxError(pos, "cannot declare in post statement of for loop"))
					}
				}
				pos.End = tok.pos.End
				node = ast.NewFor(pos, init, condition, post, nil)
			}
		}
		if node == nil {
			panic(syntaxError(tok.pos, "unexpected %s, expecting expression or %%}", tok))
		}
		p.addChild(node)
		p.addToAncestors(node)
		p.cutSpacesToken = true
		tok = p.parseEnd(tok, tokenLeftBraces)
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
		p.addChild(node)
		p.cutSpacesToken = true
		tok = p.parseEnd(tok, tokenSemicolon)
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
		p.addChild(node)
		p.cutSpacesToken = true
		tok = p.parseEnd(tok, tokenSemicolon)
		return tok

	// switch
	case tokenSwitch:
		node := p.parseSwitch(tok)
		p.addChild(node)
		p.addToAncestors(node)
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
			if len(expressions) == 0 {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			pos.End = expressions[len(expressions)-1].Pos().End
			node = ast.NewCase(pos, expressions, nil)
		case *ast.Select:
			var expressions []ast.Expression
			expressions, tok = p.parseExprList(p.next(), false, false, false)
			if len(expressions) == 0 {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			var comm ast.Node
			if len(expressions) > 1 || isAssignmentToken(tok) {
				comm, tok = p.parseAssignment(expressions, tok, false, false)
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
			p.parseEnd(tok, tokenRightBraces)
		}
		p.addChild(node)
		tok = p.parseEnd(tok, tokenColon)
		return tok

	// default
	case tokenDefault:
		pos := tok.pos
		var node ast.Node
		switch p.parent().(type) {
		case *ast.Switch, *ast.TypeSwitch:
			tok = p.next()
			pos.End = tok.pos.End
			node = ast.NewCase(pos, nil, nil)
		case *ast.Select:
			tok = p.next()
			pos.End = tok.pos.End
			node = ast.NewSelectCase(pos, nil, nil)
		default:
			// Panic with a syntax error.
			p.parseEnd(tok, tokenRightBraces)
		}
		p.addChild(node)
		p.cutSpacesToken = true
		tok = p.parseEnd(tok, tokenColon)
		return tok

	// fallthrough
	case tokenFallthrough:
		node := ast.NewFallthrough(tok.pos)
		p.addChild(node)
		p.cutSpacesToken = true
		tok = p.next()
		tok = p.parseEnd(tok, tokenSemicolon)
		return tok

	// select
	case tokenSelect:
		node := ast.NewSelect(tok.pos, nil, nil)
		p.addChild(node)
		p.addToAncestors(node)
		p.cutSpacesToken = true
		tok = p.next()
		tok = p.parseEnd(tok, tokenLeftBraces)
		return tok

	// {
	case tokenLeftBraces:
		if p.ctx != ast.ContextGo {
			panic(syntaxError(tok.pos, "unexpected %s, expecting statement", tok))
		}
		node := ast.NewBlock(tok.pos, nil)
		p.addChild(node)
		p.addToAncestors(node)
		p.cutSpacesToken = true
		return p.next()

	// }
	case tokenRightBraces:
		if p.ctx != ast.ContextGo {
			panic(syntaxError(tok.pos, "unexpected %s, expecting statement", tok))
		}
		if p.isScript && len(p.ancestors) == 1 {
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
		case tokenRightBraces:
			for {
				if n, ok := p.parent().(*ast.If); ok {
					n.Pos().End = bracesEnd
					p.removeLastAncestor()
				} else {
					return tok
				}
			}
		case tokenEOF:
			// TODO(marco): check if it is correct.
			return p.next()
		default:
			panic(syntaxError(tok.pos, "unexpected %s at end of statement", tok))
		}
		fallthrough

	// else
	case tokenElse:
		if p.ctx != ast.ContextGo {
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
		if p.ctx == ast.ContextGo && tok.typ == tokenLeftBraces || p.ctx != ast.ContextGo && tok.typ == tokenEndBlock {
			// "else"
			var blockPos *ast.Position
			if p.ctx == ast.ContextGo {
				blockPos = tok.pos
			}
			elseBlock := ast.NewBlock(blockPos, nil)
			p.addChild(elseBlock)
			p.addToAncestors(elseBlock)
			return p.next()
		}
		if tok.typ != tokenIf {
			// Panic with a syntax error.
			p.parseEnd(tok, tokenIf)
		}
		fallthrough

	// if
	case tokenIf:
		pos := tok.pos
		ifPos := tok.pos
		var expressions []ast.Expression
		expressions, tok = p.parseExprList(p.next(), false, false, true)
		if len(expressions) == 0 {
			panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
		}
		var assignment *ast.Assignment
		var expr ast.Expression
		if len(expressions) > 1 || tok.typ == tokenSimpleAssignment || tok.typ == tokenDeclaration {
			assignment, tok = p.parseAssignment(expressions, tok, false, false)
			if assignment == nil {
				panic(syntaxError(tok.pos, "expecting expression"))
			}
			if tok.typ != tokenSemicolon {
				panic(syntaxError(tok.pos, "%s used as value", assignment))
			}
			expr, tok = p.parseExpr(p.next(), false, false, true)
			if expr == nil {
				panic(syntaxError(tok.pos, "missing condition in if statement"))
			}
		} else {
			expr = expressions[0]
		}
		var blockPos *ast.Position
		if p.ctx == ast.ContextGo {
			blockPos = tok.pos
		}
		then := ast.NewBlock(blockPos, nil)
		if _, ok := p.parent().(*ast.If); !ok {
			ifPos = pos
			ifPos.End = tok.pos.End
		}
		node := ast.NewIf(ifPos, assignment, expr, then, nil)
		p.addChild(node)
		p.addToAncestors(node)
		p.addToAncestors(then)
		p.cutSpacesToken = true
		tok = p.parseEnd(tok, tokenLeftBraces)
		return tok

	// return
	case tokenReturn:
		pos := tok.pos
		if p.isScript || p.isTemplate {
			var inFunction bool
			for i := len(p.ancestors) - 1; i > 0; i-- {
				if _, ok := p.ancestors[i].(*ast.Func); ok {
					inFunction = true
					break
				}
			}
			if !inFunction {
				panic(syntaxError(tok.pos, "return statement outside function body"))
			}
		}
		tok = p.next()
		var values []ast.Expression
		if tok.typ != tokenSemicolon {
			values, tok = p.parseExprList(tok, false, false, false)
			if len(values) > 0 {
				pos.End = values[len(values)-1].Pos().End
			}
		}
		node := ast.NewReturn(pos, values)
		p.addChild(node)
		tok = p.parseEnd(tok, tokenSemicolon)
		return tok

	// include
	case tokenInclude:
		pos := tok.pos
		if tok.ctx == ast.ContextAttribute || tok.ctx == ast.ContextUnquotedAttribute {
			panic(syntaxError(tok.pos, "include statement inside an attribute value"))
		}
		// path
		tok = p.next()
		if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
			panic(syntaxError(tok.pos, "unexpected %s, expecting string", tok))
		}
		var path = unquoteString(tok.txt)
		if !ValidPath(path) {
			panic(fmt.Errorf("invalid path %q at %s", path, tok.pos))
		}
		pos.End = tok.pos.End
		node := ast.NewInclude(pos, path, tok.ctx)
		p.addChild(node)
		p.cutSpacesToken = true
		tok = p.next()
		tok = p.parseEnd(tok, tokenEndBlock)
		return tok

	// show
	case tokenShow:
		pos := tok.pos
		if tok.ctx == ast.ContextAttribute || tok.ctx == ast.ContextUnquotedAttribute {
			panic(syntaxError(tok.pos, "show statement inside an attribute value"))
		}
		tok = p.next()
		if tok.typ != tokenIdentifier {
			panic(syntaxError(tok.pos, "unexpected %s, expecting identifier", tok))
		}
		macro := ast.NewIdentifier(tok.pos, string(tok.txt))
		tok = p.next()
		// import
		var impor *ast.Identifier
		if tok.typ == tokenPeriod {
			tok = p.next()
			if tok.typ != tokenIdentifier {
				panic(syntaxError(tok.pos, "unexpected %s, expecting identifier", tok))
			}
			if len(tok.txt) == 1 && tok.txt[0] == '_' {
				panic(syntaxError(tok.pos, "cannot use _ as value"))
			}
			impor = macro
			macro = ast.NewIdentifier(tok.pos, string(tok.txt))
			if fc, _ := utf8.DecodeRuneInString(macro.Name); !unicode.Is(unicode.Lu, fc) {
				panic(syntaxError(tok.pos, "cannot refer to unexported macro %s", macro.Name))
			}
			tok = p.next()
		}
		var args []ast.Expression
		var isVariadic bool
		if tok.typ == tokenLeftParenthesis {
			args, tok = p.parseExprList(p.next(), false, false, false)
			if tok.typ == tokenEllipsis {
				if len(args) == 0 {
					panic(syntaxError(tok.pos, "unexpected ..., expecting expression"))
				}
				isVariadic = true
				tok = p.next()
			}
			if tok.typ != tokenRightParenthesis {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression or )", tok))
			}
			tok = p.next()
		}
		or := ast.ShowMacroOrError
		if tok.typ == tokenIdentifier {
			if bytes.Equal(tok.txt, orIdent) {
				tok = p.next()
				switch {
				case tok.typ == tokenIdentifier && bytes.Equal(tok.txt, ignoreIdent):
					or = ast.ShowMacroOrIgnore
				case tok.typ == tokenIdentifier && bytes.Equal(tok.txt, todoIdent):
					or = ast.ShowMacroOrTodo
				case tok.typ == tokenIdentifier && bytes.Equal(tok.txt, errorIdent):
					or = ast.ShowMacroOrError
				default:
					panic(syntaxError(tok.pos, "unexpected %s after or in show macro, expecting ignore, todo or error", tok))
				}
				tok = p.next()
			}
		}
		pos.End = tok.pos.End
		var node ast.Node
		if impor == nil {
			node = ast.NewShowMacro(pos, macro, args, isVariadic, or, tok.ctx)
		} else {
			node = ast.NewShowMacro(pos, ast.NewSelector(macro.Pos(), impor, macro.Name), args, isVariadic, or, tok.ctx)
		}
		p.addChild(node)
		p.cutSpacesToken = true
		tok = p.parseEnd(tok, tokenEndBlock)
		return tok

	// extends
	case tokenExtends:
		pos := tok.pos
		if tok.ctx != p.ctx {
			panic(syntaxError(tok.pos, "extends not in %s content", p.ctx))
		}
		if p.hasExtend {
			panic(syntaxError(tok.pos, "extends already exists"))
		}
		tree := p.ancestors[0].(*ast.Tree)
		for _, node := range tree.Nodes {
			switch node.(type) {
			case *ast.Text, *ast.Comment:
			default:
				panic(syntaxError(tok.pos, "extends can only be the first statement"))
			}
		}
		tok = p.next()
		if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
			panic(syntaxError(tok.pos, "unexpected %s, expecting string", tok))
		}
		var path = unquoteString(tok.txt)
		if !ValidPath(path) {
			panic(syntaxError(tok.pos, "invalid extends path %q", path))
		}
		pos.End = tok.pos.End
		node := ast.NewExtends(pos, path, tree.Context)
		p.addChild(node)
		p.hasExtend = true
		tok = p.next()
		tok = p.parseEnd(tok, tokenEndBlock)
		return tok

	// var or const
	case tokenVar, tokenConst:
		pos := tok.pos
		decType := tok.typ
		if tok.ctx != p.ctx {
			panic(syntaxError(tok.pos, "%s declaration not in %s content", decType, p.ctx))
		}
		tok = p.next()
		if tok.typ == tokenLeftParenthesis {
			// var ( ... )
			// const ( ... )
			var prevNode ast.Node
			var prevConstValues []ast.Expression
			var prevConstType ast.Expression
			tok = p.next()
			for {
				if tok.typ == tokenRightParenthesis {
					tok = p.next()
					if prevNode != nil {
						prevNode.Pos().End = tok.pos.End
					}
					tok = p.parseEnd(tok, tokenSemicolon)
					break
				}
				prevNode, tok = p.parseVarOrConst(tok, pos, decType)
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
				p.addChild(prevNode)
			}
		} else {
			var node ast.Node
			node, tok = p.parseVarOrConst(tok, pos, decType)
			p.addChild(node)
			tok = p.parseEnd(tok, tokenSemicolon)
		}
		return tok

	// import
	case tokenImport:
		var outOfOrder bool
		switch p := p.parent().(type) {
		case *ast.Tree:
			for i := len(p.Nodes) - 1; i >= 0; i-- {
				switch p.Nodes[i].(type) {
				case *ast.Extends, *ast.Import:
					break
				case *ast.Text, *ast.Comment:
				default:
					outOfOrder = true
					break
				}
			}
		case *ast.Package:
			if len(p.Declarations) > 0 {
				if _, ok := p.Declarations[0].(*ast.Import); !ok {
					outOfOrder = true
				}
			}
		default:
			outOfOrder = true
		}
		if outOfOrder {
			if p.isTemplate {
				panic(syntaxError(tok.pos, "unexpected import, expecting statement"))
			}
			panic(syntaxError(tok.pos, "unexpected import, expecting }"))
		}
		if tok.ctx != p.ctx {
			panic(syntaxError(tok.pos, "import not in %s content", p.ctx))
		}
		tok = p.next()
		if tok.typ == tokenLeftParenthesis && p.ctx == ast.ContextGo {
			tok = p.next()
			for tok.typ != tokenRightParenthesis {
				node := p.parseImport(tok)
				p.addChild(node)
				tok = p.next()
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
			node := p.parseImport(tok)
			p.addChild(node)
			tok = p.next()
			tok = p.parseEnd(tok, tokenSemicolon)
		}
		p.cutSpacesToken = true
		return tok

	// macro
	case tokenMacro:
		pos := tok.pos
		if len(p.ancestors) > 1 {
			panic(syntaxError(tok.pos, "unexpected macro in statement scope"))
		}
		if tok.ctx != p.ctx {
			panic(syntaxError(tok.pos, "macro not in %s content", p.ctx))
		}
		// Parses the macro name.
		tok = p.next()
		if tok.typ != tokenIdentifier {
			panic(syntaxError(tok.pos, "unexpected %s, expecting name", tok.txt))
		}
		ident := ast.NewIdentifier(tok.pos, string(tok.txt))
		var parameters []*ast.Parameter
		var isVariadic bool
		tok = p.next()
		if tok.typ == tokenLeftParenthesis {
			// Parses the macro parameters.
			names := map[string]struct{}{}
			var endPos *ast.Position
			parameters, isVariadic, endPos = p.parseFuncParameters(tok, names, false)
			pos.End = endPos.End
			tok = p.next()
		}
		if tok.typ != tokenEndBlock {
			panic(syntaxError(tok.pos, "unexpected %s, expecting %%}", tok))
		}
		// Makes the macro node.
		typ := ast.NewFuncType(nil, parameters, nil, isVariadic)
		pos.End = tok.pos.End
		typ.Position = pos
		node := ast.NewMacro(pos, ident, typ, nil, tok.ctx)
		p.addChild(node)
		p.addToAncestors(node)
		p.cutSpacesToken = true
		return p.next()

	// end
	case tokenEnd:
		if _, ok := p.parent().(*ast.URL); ok || len(p.ancestors) == 1 {
			panic(syntaxError(tok.pos, "unexpected %s", tok))
		}
		if _, ok := p.parent().(*ast.Block); ok {
			p.removeLastAncestor()
		}
		p.parent().Pos().End = tok.pos.End
		tok = p.next()
		if tok.typ != tokenEndBlock {
			parentTok := tok
			tok = p.next()
			if tok.typ != tokenEndBlock {
				panic(syntaxError(tok.pos, "unexpected %s, expecting %%}", tok))
			}
			switch p.parent().(type) {
			case ast.For:
				if parentTok.typ != tokenFor {
					panic(syntaxError(parentTok.pos, "unexpected %s, expecting for or %%}", tok))
				}
			case *ast.If:
				if parentTok.typ != tokenIf {
					panic(syntaxError(parentTok.pos, "unexpected %s, expecting if or %%}", tok))
				}
			case *ast.Macro:
				if parentTok.typ != tokenMacro {
					panic(syntaxError(parentTok.pos, "unexpected %s, expecting macro or %%}", tok))
				}
			case ast.Switch, ast.TypeSwitch:
				if parentTok.typ != tokenSwitch {
					panic(syntaxError(parentTok.pos, "unexpected %s, expecting switch or %%}", tok))
				}
			case ast.Select:
				if parentTok.typ != tokenSelect {
					panic(syntaxError(parentTok.pos, "unexpected %s, expecting select or %%}", tok))
				}
			}
		}
		p.removeLastAncestor()
		for {
			if n, ok := p.parent().(*ast.If); ok {
				n.Pos().End = tok.pos.End
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
				p.addChild(node)
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
			p.addChild(node)
		}
		tok = p.parseEnd(tok, tokenSemicolon)
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
		p.addChild(node)
		tok = p.parseEnd(tok, tokenSemicolon)
		return tok

	// goto
	case tokenGoto:
		pos := tok.pos
		if tok.ctx != ast.ContextGo {
			panic(syntaxError(tok.pos, "unexpected goto outside function body"))
		}
		tok = p.next()
		if tok.typ != tokenIdentifier {
			panic(syntaxError(tok.pos, "unexpected %s, expecting name", tok))
		}
		pos.End = tok.pos.End
		node := ast.NewGoto(pos, ast.NewIdentifier(tok.pos, string(tok.txt)))
		p.addChild(node)
		tok = p.next()
		tok = p.parseEnd(tok, tokenSemicolon)
		return tok

	// func
	case tokenFunc:
		if p.ctx == ast.ContextGo {
			// Note that parseFunc does not consume the next token in this case
			// because kind is not parseType.
			switch p.parent().(type) {
			case *ast.Tree, *ast.Package:
				node, _ := p.parseFunc(tok, parseFuncDecl)
				p.addChild(node)
				tok = p.next()
				if tok.typ != tokenSemicolon {
					panic(syntaxError(tok.pos, "unexpected %s after top level declaration", tok))
				}
				return p.next()
			}
		}
		fallthrough

	// assignment, send, label or expression
	default:
		pos := tok.pos
		var expressions []ast.Expression
		expressions, tok = p.parseExprList(tok, false, false, false)
		if len(expressions) == 0 {
			// There is no statement or it is not a statement.
			if p.isTemplate {
				if tok.typ == tokenEndBlock {
					panic(syntaxError(tok.pos, "missing statement"))
				}
				panic(syntaxError(tok.pos, "unexpected %s, expected statement", tok))
			}
			switch p.parent().(type) {
			case *ast.Tree:
				panic(syntaxError(tok.pos, "unexpected %s, expected statement", tok))
			case *ast.Label:
				panic(syntaxError(tok.pos, "missing statement after label"))
			}
			panic(syntaxError(tok.pos, "unexpected %s, expected }", tok))
		}
		if len(expressions) > 1 || isAssignmentToken(tok) {
			// Parse assignment.
			var assignment *ast.Assignment
			assignment, tok = p.parseAssignment(expressions, tok, false, false)
			if assignment == nil {
				panic(syntaxError(tok.pos, "expecting expression"))
			}
			assignment.Position = &ast.Position{Line: pos.Line, Column: pos.Column, Start: pos.Start, End: assignment.Pos().End}
			p.addChild(assignment)
			p.cutSpacesToken = true
			tok = p.parseEnd(tok, tokenSemicolon)
		} else if tok.typ == tokenArrow {
			// Parse send.
			channel := expressions[0]
			var value ast.Expression
			value, tok = p.parseExpr(p.next(), false, false, false)
			if value == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			node := ast.NewSend(pos, channel, value)
			node.Position = &ast.Position{Line: pos.Line, Column: pos.Column, Start: pos.Start, End: value.Pos().End}
			p.addChild(node)
			p.cutSpacesToken = true
			tok = p.parseEnd(tok, tokenSemicolon)
		} else {
			// Parse a label or an expression.
			expr := expressions[0]
			if ident, ok := expr.(*ast.Identifier); ok && tok.typ == tokenColon {
				// Parse a label.
				node := ast.NewLabel(pos, ident, nil)
				node.Position = &ast.Position{Line: pos.Line, Column: pos.Column, Start: pos.Start, End: tok.pos.End}
				p.addChild(node)
				p.addToAncestors(node)
				p.cutSpacesToken = true
				if p.isTemplate {
					tok = p.next()
					if tok.typ == tokenEndBlock || tok.typ == tokenEOF {
						panic(syntaxError(tok.pos, "missing statement after label"))
					}
					goto LABEL
				}
				tok = p.next()
			} else {
				p.addChild(expr)
				p.cutSpacesToken = true
				tok = p.parseEnd(tok, tokenSemicolon)
			}
		}
		return tok
	}

}

func (p *parsing) parseEnd(tok token, want tokenTyp) token {
	if p.isTemplate {
		if tok.typ != tokenEndBlock {
			if want == tokenSemicolon {
				panic(syntaxError(tok.pos, "unexpected %s at end of statement", tok))
			}
			panic(syntaxError(tok.pos, "unexpected %s, expecting %%}", tok))
		}
		return p.next()
	}
	if want == tokenSemicolon {
		if tok.typ == tokenSemicolon {
			return p.next()
		}
		if tok.typ != tokenRightBraces {
			panic(syntaxError(tok.pos, "unexpected %s at end of statement", tok))
		}
		return tok
	}
	if tok.typ != want {
		panic(syntaxError(tok.pos, "unexpected %s, expecting %s", tok, want))
	}
	return p.next()
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
	node := ast.NewTypeDeclaration(nil, ident, typ, alias)
	return node, tok
}

func (p *parsing) parseVarOrConst(tok token, pos *ast.Position, decType tokenTyp) (ast.Node, token) {
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
		if len(exprs) == 0 {
			panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
		}
	case tokenIdentifier, tokenFunc, tokenMap, tokenLeftParenthesis, tokenLeftBrackets, tokenInterface, tokenMultiplication, tokenChan, tokenArrow, tokenStruct:
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
			if len(exprs) == 0 {
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
		return ast.NewVar(pos, idents, typ, exprs), tok
	}
	return ast.NewConst(pos, idents, typ, exprs), tok
}

func (p *parsing) parseImport(tok token) *ast.Import {
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
	if p.ctx == ast.ContextGo {
		validatePackagePath(path, tok.pos)
	} else {
		if !ValidPath(path) {
			panic(syntaxError(tok.pos, "invalid import path: %q", path))
		}
		// Further restrictions on the validity of a path can be imposed by a
		// reader.
	}
	pos.End = tok.pos.End
	return ast.NewImport(pos, ident, path, tok.ctx)
}

// parseAssignment parses an assignment and returns an assignment or, if there
// is no expression, returns nil. tok can be the assignment, declaration,
// increment or decrement token. Panics on error.
func (p *parsing) parseAssignment(variables []ast.Expression, tok token, canBeSwitchGuard bool, nextIsBlockOpen bool) (*ast.Assignment, token) {
	typ, ok := assignmentType(tok)
	if !ok {
		panic(syntaxError(tok.pos, "unexpected %s, expecting := or = or comma", tok))
	}
	vp := variables[0].Pos()
	pos := &ast.Position{Line: vp.Line, Column: vp.Column, Start: vp.Start, End: tok.pos.End}
	var values []ast.Expression
	switch typ {
	case ast.AssignmentSimple, ast.AssignmentDeclaration:
		values, tok = p.parseExprList(p.next(), canBeSwitchGuard, false, nextIsBlockOpen)
		if len(values) == 0 {
			return nil, tok
		}
		pos.End = values[len(values)-1].Pos().End
	default:
		if len(variables) > 1 {
			panic(syntaxError(tok.pos, "unexpected %s, expecting := or = or comma", tok))
		}
		if typ == ast.AssignmentIncrement || typ == ast.AssignmentDecrement {
			tok = p.next()
		} else {
			values = make([]ast.Expression, 1)
			values[0], tok = p.parseExpr(p.next(), false, false, false)
			pos.End = values[0].Pos().End
		}
	}
	return ast.NewAssignment(pos, variables, typ, values), tok
}

// addChild adds a child to the current parent node.
func (p *parsing) addChild(child ast.Node) {
	switch n := p.parent().(type) {
	case *ast.Tree:
		n.Nodes = append(n.Nodes, child)
	case *ast.Package:
		n.Declarations = append(n.Declarations, child)
	case *ast.URL:
		n.Value = append(n.Value, child)
	case *ast.Macro:
		n.Body = append(n.Body, child)
	case *ast.For:
		n.Body = append(n.Body, child)
	case *ast.ForRange:
		n.Body = append(n.Body, child)
	case *ast.If:
		if n.Else != nil {
			panic("child already added to if node")
		}
		n.Else = child
	case *ast.Block:
		n.Nodes = append(n.Nodes, child)
	case *ast.Switch:
		c, ok := child.(*ast.Case)
		if ok {
			n.Cases = append(n.Cases, c)
		} else {
			lastCase := n.Cases[len(n.Cases)-1]
			lastCase.Body = append(lastCase.Body, child)
		}
	case *ast.TypeSwitch:
		c, ok := child.(*ast.Case)
		if ok {
			n.Cases = append(n.Cases, c)
			return
		}
		lastCase := n.Cases[len(n.Cases)-1]
		lastCase.Body = append(lastCase.Body, child)
	case *ast.Select:
		cc, ok := child.(*ast.SelectCase)
		if ok {
			n.Cases = append(n.Cases, cc)
		} else {
			lastCase := n.Cases[len(n.Cases)-1]
			lastCase.Body = append(lastCase.Body, child)
		}
	case *ast.Label:
		n.Statement = child
		n.Pos().End = child.Pos().End
		p.removeLastAncestor()
	default:
		panic("scriggo/parser: unexpected parent node")
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
