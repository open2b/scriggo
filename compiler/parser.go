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

	"scriggo/compiler/ast"
	"scriggo/compiler/ast/astutil"
)

type Reader interface {
	Read(path string) ([]byte, error)
}

var (
	// ErrInvalidPath is returned from the ParseTemplate function when the path
	// argument is not valid.
	ErrInvalidPath = errors.New("scriggo: invalid path")

	// ErrNotExist is returned from the ParseTemplate function when when the path
	// argument does not exist.
	ErrNotExist = errors.New("scriggo: path does not exist")
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

	// Tree source language.
	language ast.Language

	// Reports whether the tree source language is Go or
	// the parsing is in a function body.
	inGo bool

	// Report whether it is a package-less program.
	isPackageLessProgram bool

	// Reports whether it has an extend statement.
	hasExtend bool

	// Reports whether there is a token in current line for which it is possible
	// to cut the leading and trailing spaces.
	cutSpacesToken bool

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

// ParseSource parses a program or a package-less program and returns its tree.
// isPackageLessProgram reports whether it is a package-less program and shebang
// reports whether a package-less program can have the shebang as first line.
func ParseSource(src []byte, isPackageLessProgram, shebang bool) (tree *ast.Tree, err error) {

	if shebang && !isPackageLessProgram {
		return nil, errors.New("scriggo/parser: shebang can be true only for package-less programs")
	}

	tree = ast.NewTree("", nil, ast.LanguageGo)

	var p = &parsing{
		lex:                  newLexer(src, ast.ContextGo),
		language:             ast.LanguageGo,
		inGo:                 true,
		isPackageLessProgram: isPackageLessProgram,
		ancestors:            []ast.Node{tree},
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

	if len(p.ancestors) == 1 {
		if !isPackageLessProgram && len(tree.Nodes) == 0 {
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

// ParseTemplateSource parses a template with source src written in the
// language lang and returns its tree. language can be Text, HTML, CSS or
// JavaScript.
//
// ParseTemplateSource does not expand the nodes Extends, Include and Import.
func ParseTemplateSource(src []byte, lang ast.Language) (tree *ast.Tree, err error) {

	if lang < ast.LanguageText || lang > ast.LanguageJavaScript {
		return nil, errors.New("scriggo: invalid language")
	}

	tree = ast.NewTree("", nil, lang)

	var p = &parsing{
		lex:       newLexer(src, ast.Context(lang)),
		language:  lang,
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

	if p.inGo {
		switch s := p.parent().(type) {
		case *ast.Tree:
			if !p.isPackageLessProgram && tok.typ != tokenPackage {
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
	} else {
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
	}

	switch tok.typ {

	// ;
	case tokenSemicolon:
		if !p.inGo {
			panic(syntaxError(tok.pos, "unexpected semicolon, expecting statement"))
		}
		return p.next()

	// package
	case tokenPackage:
		pos := tok.pos
		if tree, ok := p.parent().(*ast.Tree); !ok || !p.inGo || p.isPackageLessProgram || len(tree.Nodes) > 0 {
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
					if a, ok := cond.(*ast.Assignment); ok && a.Type == ast.AssignmentSimple {
						panic(syntaxError(tok.pos, "assignment %s used as value", init))
					}
					panic(syntaxError(tok.pos, "%s used as value", init))
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
		case tokenLeftBraces, tokenEndBlock:
			// Parse:    for cond {
			//        {% for cond %}
			var condition ast.Expression
			if init != nil {
				condition, _ = init.(ast.Expression)
				if condition == nil {
					if a, ok := init.(*ast.Assignment); ok && a.Type == ast.AssignmentSimple {
						panic(syntaxError(tok.pos, "assignment %s used as value", init))
					}
					panic(syntaxError(tok.pos, "%s used as value", init))
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
			ident, ok := init.(ast.Expression)
			if !ok {
				panic(syntaxError(tok.pos, "unexpected in, expecting {"))
			}
			var expr ast.Expression
			expr, tok = p.parseExpr(p.next(), false, false, true)
			if expr == nil {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
			}
			ipos := ident.Pos()
			blank := ast.NewIdentifier(&ast.Position{Line: ipos.Line, Column: ipos.Column,
				Start: ipos.Start, End: ipos.Start}, "_")
			aPos := &ast.Position{Line: ipos.Line, Column: ipos.Column, Start: ipos.Start, End: expr.Pos().End}
			assignment := ast.NewAssignment(aPos, []ast.Expression{blank, ident}, ast.AssignmentDeclaration, []ast.Expression{expr})
			assignment.End = expr.Pos().End
			node = ast.NewForRange(pos, assignment, nil)
		default:
			panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
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
			node = ast.NewCase(pos, nil, nil)
		case *ast.Select:
			node = ast.NewSelectCase(pos, nil, nil)
		default:
			// Panic with a syntax error.
			p.parseEnd(tok, tokenRightBraces)
		}
		p.addChild(node)
		p.cutSpacesToken = true
		tok = p.next()
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
		if !p.inGo {
			panic(syntaxError(tok.pos, "unexpected %s, expecting statement", tok))
		}
		node := ast.NewBlock(tok.pos, nil)
		p.addChild(node)
		p.addToAncestors(node)
		p.cutSpacesToken = true
		return p.next()

	// }
	case tokenRightBraces:
		if !p.inGo {
			panic(syntaxError(tok.pos, "unexpected %s, expecting statement", tok))
		}
		if p.isPackageLessProgram && len(p.ancestors) == 1 {
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
		case tokenEndValue, tokenEndBlock:
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
		if !p.inGo {
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
		if p.inGo && tok.typ == tokenLeftBraces || !p.inGo && tok.typ == tokenEndBlock {
			// "else"
			var blockPos *ast.Position
			if p.inGo {
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
		var init ast.Node
		var expr ast.Expression
		init, tok = p.parseSimpleStatement(p.next(), false, true)
		if tok.typ == tokenSemicolon {
			expr, tok = p.parseExpr(p.next(), false, false, true)
		} else if init != nil {
			expr, _ = init.(ast.Expression)
			if expr == nil {
				if a, ok := init.(*ast.Assignment); ok && a.Type == ast.AssignmentSimple {
					panic(syntaxError(tok.pos, "assignment %s used as value", init))
				}
				panic(syntaxError(tok.pos, "%s used as value", init))
			}
			init = nil
		}
		if expr == nil {
			if p.inGo && tok.typ == tokenLeftBraces || !p.inGo && tok.typ == tokenEndBlock {
				panic(syntaxError(tok.pos, "missing condition in if statement"))
			}
			panic(syntaxError(tok.pos, "unexpected %s, expecting expression", tok))
		}
		var blockPos *ast.Position
		if p.inGo {
			blockPos = tok.pos
		}
		then := ast.NewBlock(blockPos, nil)
		if _, ok := p.parent().(*ast.If); !ok {
			ifPos = pos
			ifPos.End = tok.pos.End
		}
		node := ast.NewIf(ifPos, init, expr, then, nil)
		p.addChild(node)
		p.addToAncestors(node)
		p.addToAncestors(then)
		p.cutSpacesToken = true
		tok = p.parseEnd(tok, tokenLeftBraces)
		return tok

	// return
	case tokenReturn:
		pos := tok.pos
		if !p.inGo || p.isPackageLessProgram {
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
			if values != nil {
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
		switch tok.ctx {
		case ast.ContextText, ast.ContextHTML, ast.ContextCSS, ast.ContextJavaScript:
		default:
			panic(syntaxError(tok.pos, "include statement inside %s", tok.ctx))
		}
		// path
		tok = p.next()
		if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
			panic(syntaxError(tok.pos, "unexpected %s, expecting string", tok))
		}
		var path = unquoteString(tok.txt)
		if !ValidTemplatePath(path) {
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
		pos.End = tok.pos.End
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
			pos.End = tok.pos.End
			tok = p.next()
		}
		var args []ast.Expression
		var isVariadic bool
		if tok.typ == tokenLeftParenthesis {
			args, tok = p.parseExprListInParenthesis(p.next())
			if tok.typ == tokenEllipsis {
				if args == nil {
					panic(syntaxError(tok.pos, "unexpected ..., expecting expression"))
				}
				isVariadic = true
				tok = p.next()
			}
			if tok.typ != tokenRightParenthesis {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression or )", tok))
			}
			pos.End = tok.pos.End
			tok = p.next()
		}
		or := ast.ShowMacroOrError
		if tok.typ == tokenIdentifier && bytes.Equal(tok.txt, orIdent) {
			tok = p.next()
			if tok.typ != tokenIdentifier {
				panic(syntaxError(tok.pos, "unexpected %s after or in show macro, expecting ignore, todo or error", tok))
			}
			switch {
			case bytes.Equal(tok.txt, ignoreIdent):
				or = ast.ShowMacroOrIgnore
			case bytes.Equal(tok.txt, todoIdent):
				or = ast.ShowMacroOrTodo
			case bytes.Equal(tok.txt, errorIdent):
				or = ast.ShowMacroOrError
			default:
				panic(syntaxError(tok.pos, "unexpected %s after or in show macro, expecting ignore, todo or error", tok))
			}
			pos.End = tok.pos.End
			tok = p.next()
		}
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
		if tok.ctx != ast.Context(p.language) {
			panic(syntaxError(tok.pos, "extends not in %s content", ast.Context(p.language)))
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
		if !ValidTemplatePath(path) {
			panic(syntaxError(tok.pos, "invalid extends path %q", path))
		}
		pos.End = tok.pos.End
		node := ast.NewExtends(pos, path, tok.ctx)
		p.addChild(node)
		p.hasExtend = true
		tok = p.next()
		tok = p.parseEnd(tok, tokenEndBlock)
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
					tok = p.parseEnd(tok, tokenSemicolon)
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
				p.addChild(prevNode)
				iotaValue++
			}
		} else {
			var node ast.Node
			node, tok = p.parseVarOrConst(tok, pos, decType, 0)
			p.addChild(node)
			tok = p.parseEnd(tok, tokenSemicolon)
		}
		return tok

	// import
	case tokenImport:
		switch parent := p.parent().(type) {
		case *ast.Tree:
			for i := len(parent.Nodes) - 1; i >= 0; i-- {
				switch parent.Nodes[i].(type) {
				case *ast.Extends, *ast.Import:
					break
				case *ast.Text, *ast.Comment:
				default:
					panic(syntaxError(tok.pos, "unexpected import, expecting statement"))
				}
			}
		case *ast.Package:
			for _, declaration := range parent.Declarations {
				if _, ok := declaration.(*ast.Import); !ok {
					panic(syntaxError(tok.pos, "non-declaration statement outside function body"))
				}
			}
		default:
			if p.inGo {
				panic(syntaxError(tok.pos, "unexpected import, expecting }"))
			}
			panic(syntaxError(tok.pos, "unexpected import, expecting statement"))
		}
		if tok.ctx != ast.Context(p.language) {
			panic(syntaxError(tok.pos, "import not in %s content", ast.Context(p.language)))
		}
		tok = p.next()
		if tok.typ == tokenLeftParenthesis && p.inGo {
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
		if tok.ctx != ast.Context(p.language) {
			panic(syntaxError(tok.pos, "macro not in %s content", ast.Context(p.language)))
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
			var endPos *ast.Position
			parameters, isVariadic, endPos = p.parseFuncParameters(tok, false)
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
		pos := tok.pos
		tok = p.next()
		if tok.typ != tokenEndBlock {
			parentTok := tok
			pos = parentTok.pos
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
		if p.inGo {
			// Note that parseFunc does not consume the next token in this case
			// because kind is not parseType.
			switch p.parent().(type) {
			case *ast.Tree, *ast.Package:
				node, tok := p.parseFunc(tok, parseFuncDecl)
				if tok.typ != tokenSemicolon {
					panic(syntaxError(tok.pos, "unexpected %s after top level declaration", tok))
				}
				p.addChild(node)
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
			if p.inGo {
				switch p.parent().(type) {
				case *ast.Tree:
					panic(syntaxError(tok.pos, "unexpected %s, expected statement", tok))
				case *ast.Label:
					panic(syntaxError(tok.pos, "missing statement after label"))
				}
				panic(syntaxError(tok.pos, "unexpected %s, expecting }", tok))
			}
			if tok.typ == tokenEndBlock {
				panic(syntaxError(tok.pos, "missing statement"))
			}
			panic(syntaxError(tok.pos, "unexpected %s, expecting statement", tok))
		}
		if len(expressions) > 1 || isAssignmentToken(tok) {
			// Parse assignment.
			var assignment *ast.Assignment
			assignment, tok = p.parseAssignment(expressions, tok, false, false, false)
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
				tok = p.next()
				if !p.inGo {
					if tok.typ == tokenEndBlock || tok.typ == tokenEOF {
						panic(syntaxError(tok.pos, "missing statement after label"))
					}
					goto LABEL
				}
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
	if p.inGo {
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
	if tok.typ != tokenEndBlock {
		if want == tokenSemicolon {
			panic(syntaxError(tok.pos, "unexpected %s at end of statement", tok))
		}
		panic(syntaxError(tok.pos, "unexpected %s, expecting %%}", tok))
	}
	return p.next()
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
		assignment.Position = &ast.Position{Line: pos.Line, Column: pos.Column, Start: pos.Start, End: assignment.Pos().End}
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
		send.Position = &ast.Position{Line: pos.Line, Column: pos.Column, Start: pos.Start, End: value.Pos().End}
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
		return ast.NewVar(pos, idents, typ, exprs), tok
	}
	return ast.NewConst(pos, idents, typ, exprs, iotaValue), tok
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
	if p.inGo {
		validatePackagePath(path, tok.pos)
	} else {
		if !ValidTemplatePath(path) {
			panic(syntaxError(tok.pos, "invalid import path: %q", path))
		}
		// Further restrictions on the validity of a path can be imposed by a
		// reader.
	}
	pos.End = tok.pos.End
	return ast.NewImport(pos, ident, path, tok.ctx)
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
	pos := &ast.Position{Line: vp.Line, Column: vp.Column, Start: vp.Start, End: tok.pos.End}
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
