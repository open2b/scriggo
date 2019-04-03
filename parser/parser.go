// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package parser implements methods to parse a template source and expand a
// parsed tree.
package parser

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"scrigo/ast"
)

var (
	// ErrInvalidPath is returned from the Parse method and a Reader when the
	// path argument is not valid.
	ErrInvalidPath = errors.New("scrigo/parser: invalid path")

	// ErrNotExist is returned from the Parse method and a Reader when the
	// path does not exist.
	ErrNotExist = errors.New("scrigo/parser: path does not exist")

	// ErrReadTooLarge is returned from a DirLimitedReader when a limit is
	// exceeded.
	ErrReadTooLarge = errors.New("scrigo/parser: read too large")
)

// SyntaxError records a parsing error with the path and the position where the
// error occurred.
type SyntaxError struct {
	Path string
	Pos  ast.Position
	Err  error
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("%s:%s: syntax error: %s", e.Path, e.Pos, e.Err)
}

// CycleError implements an error indicating the presence of a cycle.
type CycleError string

func (e CycleError) Error() string {
	return fmt.Sprintf("cycle not allowed\n%s", string(e))
}

// next returns the next token from the lexer. Panics if the lexer channel is
// closed.
func next(lex *lexer) token {
	tok, ok := <-lex.tokens
	if !ok {
		if lex.err == nil {
			panic("next called after EOF")
		}
		panic(lex.err)
	}
	return tok
}

// containsOnlySpaces indicates if b contains only white space characters as
// intended by Go parser.
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

	// Indicates if it has been extended.
	isExtended bool

	// Indicates if it is in a macro.
	isInMacro bool

	// Indicates if there is a token in current line for which it is possible
	// to cut the leading and trailing spaces.
	cutSpacesToken bool

	// Context.
	ctx ast.Context

	// Ancestors from the root up to the parent.
	ancestors []ast.Node

	// Position of the last fallthrough token, used for error messages.
	lastFallthroughTokenPos ast.Position
}

// ParseSource parses src in the context ctx and returns a tree. Nodes
// Extends, Import and Include will not be expanded (the field Tree will be
// nil). To get an expanded tree call the method Parse of a Parser instead.
func ParseSource(src []byte, ctx ast.Context) (tree *ast.Tree, err error) {

	switch ctx {
	case ast.ContextNone, ast.ContextText, ast.ContextHTML, ast.ContextCSS, ast.ContextScript:
	default:
		return nil, errors.New("scrigo/parser: invalid context. Valid contexts are None, Text, HTML, CSS and Script")
	}

	// Tree result of the expansion.
	tree = ast.NewTree("", nil, ctx)

	var p = &parsing{
		lex:       newLexer(src, ctx),
		ctx:       ctx,
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

	if ctx == ast.ContextNone {

		// Reads the tokens.
		for tok := range p.lex.tokens {
			if tok.typ == tokenEOF {
				if len(p.ancestors) > 1 {
					if _, ok := p.ancestors[1].(*ast.Package); !ok || len(p.ancestors) > 2 {
						return nil, &SyntaxError{"", *tok.pos, fmt.Errorf("unexpected EOF, expecting }")}
					}
				}
			} else {
				p.parseStatement(tok)
			}
		}

	} else {

		// Current line.
		var line = 0

		// First Text node of the current line.
		var firstText *ast.Text

		// Number of non-text tokens in current line.
		var tokensInLine = 0

		// Index of the last byte.
		var end = len(src) - 1

		// Reads the tokens.
		for tok := range p.lex.tokens {

			var text *ast.Text
			if tok.typ == tokenText {
				text = ast.NewText(tok.pos, tok.txt, ast.Cut{})
			}

			if line < tok.lin || tok.pos.End == end {
				if p.cutSpacesToken && tokensInLine == 1 {
					cutSpaces(firstText, text)
				}
				line = tok.lin
				firstText = text
				p.cutSpacesToken = false
				tokensInLine = 0
			}

			// Parent is always the last ancestor.
			parent := p.ancestors[len(p.ancestors)-1]

			switch tok.typ {

			// EOF
			case tokenEOF:
				if len(p.ancestors) > 1 {
					return nil, &SyntaxError{"", *tok.pos, fmt.Errorf("unexpected EOF, expecting {%% end %%}")}
				}

			// Text
			case tokenText:
				if s, ok := parent.(*ast.Switch); ok { // TODO (Gianluca): what about Type Switches?
					if len(s.Cases) == 0 {
						// TODO (Gianluca): this "if" should be moved before the
						// switch that precedes it.
						if containsOnlySpaces(text.Text) {
							s.LeadingText = text
						}
						return nil, &SyntaxError{"", *tok.pos, fmt.Errorf("unexpected text, expecting case of default or {%% end %%}")}
					}
					lastCase := s.Cases[len(s.Cases)-1]
					if lastCase.Fallthrough {
						if containsOnlySpaces(text.Text) {
							continue
						}
						return nil, &SyntaxError{"", p.lastFallthroughTokenPos, fmt.Errorf("fallthrough statement out of place")}
					}
				}
				addChild(parent, text)

			// StartURL
			case tokenStartURL:
				node := ast.NewURL(tok.pos, tok.tag, tok.att, nil)
				addChild(parent, node)
				p.ancestors = append(p.ancestors, node)

			// EndURL
			case tokenEndURL:
				pos := p.ancestors[len(p.ancestors)-1].Pos()
				pos.End = tok.pos.End - 1
				p.ancestors = p.ancestors[:len(p.ancestors)-1]

			// {%
			case tokenStartStatement:

				tokensInLine++

				p.parseStatement(tok)

			// {{ }}
			case tokenStartValue:
				if p.isExtended && !p.isInMacro {
					return nil, &SyntaxError{"", *tok.pos, fmt.Errorf("value statement outside macro")}
				}
				tokensInLine++
				expr, tok2 := p.parseExpr(token{}, false, false, false, false)
				if expr == nil {
					return nil, &SyntaxError{"", *tok2.pos, fmt.Errorf("expecting expression")}
				}
				if tok2.typ != tokenEndValue {
					return nil, &SyntaxError{"", *tok2.pos, fmt.Errorf("unexpected %s, expecting }}", tok2)}
				}
				tok.pos.End = tok2.pos.End
				var node = ast.NewShow(tok.pos, expr, tok.ctx)
				addChild(parent, node)

			// comment
			case tokenComment:
				tokensInLine++
				var node = ast.NewComment(tok.pos, string(tok.txt[2:len(tok.txt)-2]))
				addChild(parent, node)
				p.cutSpacesToken = true

			default:
				return nil, &SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s", tok)}

			}

		}

	}

	if p.lex.err != nil {
		return nil, p.lex.err
	}

	return tree, nil
}

// parseStatement parses a statement. Panics on error.
func (p *parsing) parseStatement(tok token) {

	var node ast.Node

	var pos = tok.pos

	var expr ast.Expression

	var ok bool

	if p.ctx != ast.ContextNone {
		tok = next(p.lex)
	}

	// Parent is always the last ancestor.
	parent := p.ancestors[len(p.ancestors)-1]

	l := -1
	switch s := parent.(type) {
	case *ast.Package:
		switch tok.typ {
		case tokenImport, tokenFunc, tokenVar, tokenConst, tokenType:
		default:
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("non-declaration statement outside function body (%q)", tok)})
		}
	case *ast.Switch:
		l = len(s.Cases)
	case *ast.TypeSwitch:
		l = len(s.Cases)
	}
	if l == 0 {
		if tok.typ != tokenCase && tok.typ != tokenDefault && tok.typ != tokenEnd && tok.typ != tokenRightBraces {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting case of default or {%% end %%}", tok.String())})
		}
	}

	switch tok.typ {

	// ;
	case tokenSemicolon:
		if p.ctx != ast.ContextNone {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected semicolon, expecting %%}")})
		}

	// package
	case tokenPackage:
		if tree, ok := parent.(*ast.Tree); !ok || p.ctx != ast.ContextNone || len(tree.Nodes) > 0 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected package, expecting statement")})
		}
		tok = next(p.lex)
		if tok.typ != tokenIdentifier {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("expected 'IDENT', found %q", string(tok.txt))})
		}
		name := string(tok.txt)
		if name == "_" {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("invalid package name _")})
		}
		pos.End = tok.pos.End
		tok = next(p.lex)
		if tok.typ != tokenSemicolon {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting semicolon or newline", string(tok.txt))})
		}
		node = ast.NewPackage(pos, name, nil)
		addChild(parent, node)
		p.ancestors = append(p.ancestors, node)

	// for
	case tokenFor:
		var node ast.Node
		var init *ast.Assignment
		var assignmentType ast.AssignmentType
		variables, tok := p.parseExprList(token{}, true, false, false, true)
		switch tok.typ {
		case tokenIn:
			// Parses statement "for ident in expr".
			if len(variables) == 0 {
				panic(&SyntaxError{"", *(variables[1].Pos()), fmt.Errorf("unexpected in, expected expression")})
			}
			if len(variables) > 1 {
				panic(&SyntaxError{"", *(variables[1].Pos()), fmt.Errorf("expected only one expression")})
			}
			ident, ok := variables[0].(*ast.Identifier)
			if !ok {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected in, expected assignment")})
			}
			if ident.Name == "_" {
				panic(&SyntaxError{"", *(ident.Pos()), fmt.Errorf("cannot use _ as value")})
			}
			ipos := ident.Pos()
			blank := ast.NewIdentifier(&ast.Position{ipos.Line, ipos.Column, ipos.Start, ipos.Start}, "_")
			// Parses the slice expression.
			// TODO (Gianluca): nextIsBlockOpen should be true?
			expr, tok = p.parseExpr(token{}, false, false, false, false)
			if expr == nil {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
			}
			assignment := ast.NewAssignment(&ast.Position{ipos.Line, ipos.Column, ipos.Start, expr.Pos().End},
				[]ast.Expression{blank, ident}, ast.AssignmentDeclaration, []ast.Expression{expr})
			pos.End = tok.pos.End
			node = ast.NewForRange(pos, assignment, nil)
		case tokenLeftBraces, tokenEndStatement:
			if (p.ctx == ast.ContextNone) != (tok.typ == tokenLeftBraces) {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or %%}", tok)})
			}
			// Parses statement "for".
			if len(variables) > 1 {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
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
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected range, expecting := or = or comma")})
			}
			tpos := tok.pos
			// TODO (Gianluca): nextIsBlockOpen should be true?
			expr, tok = p.parseExpr(token{}, false, false, false, true)
			if expr == nil {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
			}
			tpos.End = expr.Pos().End
			assignment := ast.NewAssignment(tpos, nil, ast.AssignmentSimple, []ast.Expression{expr})
			pos.End = tok.pos.End
			node = ast.NewForRange(pos, assignment, nil)
		case tokenSimpleAssignment, tokenDeclaration, tokenIncrement, tokenDecrement:
			if len(variables) == 0 {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
			}
			if tok.typ == tokenDeclaration {
				assignmentType = ast.AssignmentDeclaration
			}
			init, tok = p.parseAssignment(variables, tok, false, false)
			if init == nil && tok.typ != tokenRange {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
			}
			fallthrough
		case tokenSemicolon:
			if tok.typ == tokenRange {
				// Parses statements
				//     "for index[, ident] = range expr" and
				//     "for index[, ident] := range expr".
				expr, tok = p.parseExpr(token{}, false, false, false, true)
				if expr == nil {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
				}
				vpos := variables[0].Pos()
				assignment := ast.NewAssignment(&ast.Position{vpos.Line, vpos.Column, vpos.Start, expr.Pos().End},
					variables, assignmentType, []ast.Expression{expr})
				pos.End = tok.pos.End
				node = ast.NewForRange(pos, assignment, nil)
			} else {
				// Parses statement "for [init]; [condition]; [post]".
				// Parses the condition expression.
				var condition ast.Expression
				condition, tok = p.parseExpr(token{}, false, false, false, true)
				if tok.typ != tokenSemicolon {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expected semicolon", tok)})
				}
				// Parses the post iteration statement.
				var post *ast.Assignment
				variables, tok = p.parseExprList(token{}, true, false, false, true)
				if len(variables) > 0 {
					pos := tok.pos
					post, tok = p.parseAssignment(variables, tok, false, true)
					if post == nil {
						panic(&SyntaxError{"", *tok.pos, fmt.Errorf("expecting expression")})
					}
					if post.Type == ast.AssignmentDeclaration {
						panic(&SyntaxError{"", *pos, fmt.Errorf("cannot declare in post statement of for loop")})
					}
				}
				pos.End = tok.pos.End
				node = ast.NewFor(pos, init, condition, post, nil)
			}
		}
		if node == nil || (p.ctx == ast.ContextNone && tok.typ != tokenLeftBraces) || (p.ctx != ast.ContextNone && tok.typ != tokenEndStatement) {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or %%}", tok)})
		}
		addChild(parent, node)
		p.ancestors = append(p.ancestors, node)
		p.cutSpacesToken = true

	// break
	case tokenBreak:
		tok = next(p.lex)
		if (p.ctx == ast.ContextNone && tok.typ != tokenSemicolon) || (p.ctx != ast.ContextNone && tok.typ != tokenEndStatement) {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
		}
		pos.End = tok.pos.End
		node = ast.NewBreak(pos)
		addChild(parent, node)
		p.cutSpacesToken = true

	// continue
	case tokenContinue:
		// TODO (Gianluca): move to type-checker.
		var loop bool
		for i := len(p.ancestors) - 1; !loop && i > 0; i-- {
			switch p.ancestors[i].(type) {
			case *ast.For, *ast.ForRange:
				loop = true
				break
			}
		}
		if !loop {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("continue is not in a loop")})
		}
		tok = next(p.lex)
		if (p.ctx == ast.ContextNone && tok.typ != tokenSemicolon) || (p.ctx != ast.ContextNone && tok.typ != tokenEndStatement) {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
		}
		pos.End = tok.pos.End
		node = ast.NewContinue(pos)
		addChild(parent, node)
		p.cutSpacesToken = true

	// switch
	case tokenSwitch:
		node = p.parseSwitch(pos)
		addChild(parent, node)
		p.ancestors = append(p.ancestors, node)
		p.cutSpacesToken = true

	// case:
	case tokenCase:

		// TODO (Gianluca): check if all expressions contained in this case have
		// not been previously declarated in another "case". In such condition
		// return an error as:
		//
		// prog.go:12:2: duplicate case int in type switch previous case at
		// prog.go:11:2

		switch parent.(type) {
		case *ast.Switch, *ast.TypeSwitch:
		default:
			// TODO (Gianluca): should be "unexpected case, expecting ...".
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected case")})
		}
		var node *ast.Case

		// TODO (Gianluca): allMustBeTypes should be set to "true" when parsing
		// TypeSwitch cases.

		expressions, tok := p.parseExprList(token{}, false, false, false, false)
		if (p.ctx == ast.ContextNone && tok.typ != tokenColon) || (p.ctx != ast.ContextNone && tok.typ != tokenEndStatement) {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
		}
		pos.End = tok.pos.End
		node = ast.NewCase(pos, expressions, nil, false)
		addChild(parent, node)

	// default:
	case tokenDefault:
		// TODO (Gianluca): move to type-checker.
		switch s := parent.(type) {
		case *ast.Switch:
			for _, c := range s.Cases {
				if c.Expressions == nil {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("multiple defaults in switch (first at %s)", c.Pos())})
				}
			}
		case *ast.TypeSwitch:
			for _, c := range s.Cases {
				if c.Expressions == nil {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("multiple defaults in switch (first at %s)", c.Pos())})
				}
			}
		default:
			// TODO (Gianluca): should be "unexpected case, expecting ...".
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected case")})
		}
		tok = next(p.lex)
		if (p.ctx == ast.ContextNone && tok.typ != tokenColon) || (p.ctx != ast.ContextNone && tok.typ != tokenEndStatement) {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
		}
		pos.End = tok.pos.End
		node := ast.NewCase(pos, nil, nil, false)
		addChild(parent, node)
		p.cutSpacesToken = true

		// fallthrough
	case tokenFallthrough:
		// TODO (Gianluca): fallthrough must be implemented as an ast node.
		p.lastFallthroughTokenPos = *tok.pos
		tok = next(p.lex)
		if (p.ctx == ast.ContextNone && tok.typ != tokenSemicolon) || (p.ctx != ast.ContextNone && tok.typ != tokenEndStatement) {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
		}
		switch s := parent.(type) {
		case *ast.Switch:
			lastCase := s.Cases[len(s.Cases)-1]
			// TODO (Gianluca): move this check to type-checker:
			if lastCase.Fallthrough {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("fallthrough statement out of place")})
			}
			lastCase.Fallthrough = true
		case *ast.TypeSwitch:
			// TODO (Gianluca): move this check to type-checker:
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("cannot fallthrough in type switch")})
		default:
			// TODO (Gianluca): move this check to type-checker:
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("fallthrough statement out of place")})
		}
		pos.End = tok.pos.End
		p.cutSpacesToken = true

	// {
	case tokenLeftBraces:
		if p.ctx != ast.ContextNone {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting for, if, show, extends, include, macro or end", tok)})
		}
		node = ast.NewBlock(tok.pos, nil)
		addChild(parent, node)
		p.ancestors = append(p.ancestors, node)
		p.cutSpacesToken = true

	// "}"
	case tokenRightBraces:
		if p.ctx != ast.ContextNone {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting for, if, show, extends, include, macro or end", tok)})
		}
		if len(p.ancestors) == 1 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("not opened brace")})
		}
		bracesEnd := tok.pos.End
		parent.Pos().End = bracesEnd
		p.ancestors = p.ancestors[:len(p.ancestors)-1]
		parent = p.ancestors[len(p.ancestors)-1]
		tok = next(p.lex)
		switch tok.typ {
		case tokenElse:
		case tokenSemicolon:
			for {
				if _, ok := parent.(*ast.If); ok {
					parent.Pos().End = bracesEnd
					p.ancestors = p.ancestors[:len(p.ancestors)-1]
					parent = p.ancestors[len(p.ancestors)-1]
				} else {
					return
				}
			}
		case tokenEOF:
			return
		default:
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s at end of statement", tok)})
		}
		fallthrough

	// else
	case tokenElse:
		if p.ctx == ast.ContextNone {
			if len(p.ancestors) == 1 {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected else")})
			}
		} else {
			// Closes the parent block.
			if _, ok = parent.(*ast.Block); !ok {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected else")})
			}
			p.ancestors = p.ancestors[:len(p.ancestors)-1]
			parent = p.ancestors[len(p.ancestors)-1]
		}
		if _, ok = parent.(*ast.If); !ok {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected else at end of statement")})
		}
		p.cutSpacesToken = true
		tok = next(p.lex)
		if p.ctx == ast.ContextNone && tok.typ == tokenLeftBraces || p.ctx != ast.ContextNone && tok.typ == tokenEndStatement {
			// "else"
			var blockPos *ast.Position
			if p.ctx == ast.ContextNone {
				blockPos = tok.pos
			}
			elseBlock := ast.NewBlock(blockPos, nil)
			addChild(parent, elseBlock)
			p.ancestors = append(p.ancestors, elseBlock)
			return
		}
		if tok.typ != tokenIf { // "else if"
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting if or %%}", tok)})
		}
		fallthrough

	// if
	case tokenIf:
		ifPos := tok.pos
		expressions, tok := p.parseExprList(token{}, true, false, false, true)
		if len(expressions) == 0 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
		}
		var assignment *ast.Assignment
		if len(expressions) > 1 || tok.typ == tokenSimpleAssignment || tok.typ == tokenDeclaration {
			assignment, tok = p.parseAssignment(expressions, tok, false, false)
			if assignment == nil {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("expecting expression")})
			}
			if tok.typ != tokenSemicolon {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("%s used as value", assignment)})
			}
			expr, tok = p.parseExpr(token{}, false, false, false, true)
			if expr == nil {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("missing condition in if statement")})
			}
		} else {
			expr = expressions[0]
		}
		if (p.ctx == ast.ContextNone && tok.typ != tokenLeftBraces) || (p.ctx != ast.ContextNone && tok.typ != tokenEndStatement) {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
		}
		pos.End = tok.pos.End
		var blockPos *ast.Position
		if p.ctx == ast.ContextNone {
			blockPos = tok.pos
		}
		then := ast.NewBlock(blockPos, nil)
		if _, ok := parent.(*ast.If); !ok {
			ifPos = pos
		}
		node = ast.NewIf(ifPos, assignment, expr, then, nil)
		addChild(parent, node)
		p.ancestors = append(p.ancestors, node, then)
		p.cutSpacesToken = true

	// return
	case tokenReturn:
		var inFunction bool
		for i := len(p.ancestors) - 1; i > 0; i-- {
			if _, ok := p.ancestors[i].(*ast.Func); ok {
				inFunction = true
				break
			}
		}
		if !inFunction {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("non-declaration statement outside function body")})
		}
		values, tok := p.parseExprList(token{}, false, false, false, false)
		if tok.typ != tokenSemicolon {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s at end of statement", tok)})
		}
		if len(values) > 0 {
			pos.End = values[len(values)-1].Pos().End
		}
		node = ast.NewReturn(pos, values)
		addChild(parent, node)

	// include
	case tokenInclude:
		if p.ctx == ast.ContextNone {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("include statement not in template")})
		}
		if p.isExtended && !p.isInMacro {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("include statement outside macro")})
		}
		if tok.ctx == ast.ContextAttribute || tok.ctx == ast.ContextUnquotedAttribute {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("include statement inside an attribute value")})
		}
		// path
		tok = next(p.lex)
		if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting string", tok)})
		}
		var path = unquoteString(tok.txt)
		if !validPath(path) {
			panic(fmt.Errorf("invalid path %q at %s", path, tok.pos))
		}
		tok = next(p.lex)
		if tok.typ != tokenEndStatement {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ( or %%}", tok)})
		}
		pos.End = tok.pos.End
		node = ast.NewInclude(pos, path, tok.ctx)
		addChild(parent, node)
		p.cutSpacesToken = true

	// show
	case tokenShow:
		if p.ctx == ast.ContextNone {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("show statement not in template")})
		}
		if p.isExtended && !p.isInMacro {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("show statement outside macro")})
		}
		if tok.ctx == ast.ContextAttribute || tok.ctx == ast.ContextUnquotedAttribute {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("show statement inside an attribute value")})
		}
		tok = next(p.lex)
		if tok.typ != tokenIdentifier {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)})
		}
		if len(tok.txt) == 1 && tok.txt[0] == '_' {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("cannot use _ as value")})
		}
		macro := ast.NewIdentifier(tok.pos, string(tok.txt))
		tok = next(p.lex)
		// import
		var impor *ast.Identifier
		if tok.typ == tokenPeriod {
			tok = next(p.lex)
			if tok.typ != tokenIdentifier {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)})
			}
			if len(tok.txt) == 1 && tok.txt[0] == '_' {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("cannot use _ as value")})
			}
			impor = macro
			macro = ast.NewIdentifier(tok.pos, string(tok.txt))
			if fc, _ := utf8.DecodeRuneInString(macro.Name); !unicode.Is(unicode.Lu, fc) {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("cannot refer to unexported macro %s", macro.Name)})
			}
			tok = next(p.lex)
		}
		var arguments []ast.Expression
		if tok.typ == tokenLeftParenthesis {
			// arguments
			arguments = []ast.Expression{}
			for {
				expr, tok = p.parseExpr(token{}, false, false, false, false)
				if expr == nil {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
				}
				arguments = append(arguments, expr)
				if tok.typ == tokenRightParenthesis {
					break
				}
				if tok.typ != tokenComma {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting , or )", tok)})
				}
			}
			tok = next(p.lex)
			if tok.typ != tokenEndStatement {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
			}
		}
		if tok.typ != tokenEndStatement {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ( or %%}", tok)})
		}
		pos.End = tok.pos.End
		node = ast.NewShowMacro(pos, impor, macro, arguments, tok.ctx)
		addChild(parent, node)
		p.cutSpacesToken = true

	// extends
	case tokenExtends:
		if p.ctx == ast.ContextNone {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("extends statement not in template")})
		}

		if p.isExtended {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("extends already exists")})
		}
		tree := p.ancestors[0].(*ast.Tree)
		if len(tree.Nodes) > 0 {
			if _, ok = tree.Nodes[0].(*ast.Text); !ok || len(tree.Nodes) > 1 {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("extends can only be the first statement")})
			}
		}
		if tok.ctx != p.ctx {
			switch tok.ctx {
			case ast.ContextAttribute, ast.ContextUnquotedAttribute:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("extends inside an attribute value")})
			case ast.ContextScript:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("extends inside a script tag")})
			case ast.ContextCSS:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("extends inside a style tag")})
			}
		}
		tok = next(p.lex)
		if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting string", tok)})
		}
		var path = unquoteString(tok.txt)
		if !validPath(path) {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("invalid extends path %q", path)})
		}
		tok = next(p.lex)
		if tok.typ != tokenEndStatement {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
		}
		pos.End = tok.pos.End
		node = ast.NewExtends(pos, path, tok.ctx)
		addChild(parent, node)
		p.isExtended = true

	// var, const
	case tokenVar, tokenConst:
		var kind string
		if tok.typ == tokenVar {
			kind = "var"
		} else {
			kind = "const"
		}
		if tok.ctx != p.ctx {
			switch tok.ctx {
			case ast.ContextAttribute, ast.ContextUnquotedAttribute:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("var inside an attribute value")})
			case ast.ContextScript:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("var inside a script tag")})
			case ast.ContextCSS:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("var inside a style tag")})
			}
		}
		nodePos := &ast.Position{Line: tok.pos.Line, Column: tok.pos.Column, Start: tok.pos.Start}
		tok = next(p.lex)
		if tok.typ == tokenLeftParenthesis {
			// var ( ... )
			// const ( ... )
			var lastNode ast.Node
			for {
				tok = next(p.lex)
				if tok.typ == tokenRightParenthesis {
					// TODO (Gianluca): what happens if there are no )?
					lastNode.Pos().End = tok.pos.End
					tok = next(p.lex)
					if tok.typ != tokenSemicolon {
						panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s at the end of statement", tok.txt)})
					}
					break
				}
				lastNode = p.parseVarOrConst(tok, nodePos, kind)
				addChild(parent, lastNode)
			}
		} else {
			addChild(parent, p.parseVarOrConst(tok, nodePos, kind))
		}

	// import
	case tokenImport:
		if tok.ctx != p.ctx {
			switch tok.ctx {
			case ast.ContextAttribute, ast.ContextUnquotedAttribute:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("import inside an attribute value")})
			case ast.ContextScript:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("import inside a script tag")})
			case ast.ContextCSS:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("import inside a style tag")})
			}
		}
		for i := len(p.ancestors) - 1; i > 0; i-- {
			switch p.ancestors[i].(type) {
			case ast.For, ast.ForRange:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end for", tok)})
			case *ast.If:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end if", tok)})
			case *ast.Func:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting }", tok)})
			case *ast.Macro:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end macro", tok)})
			case *ast.Case:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting statement", tok)})
			}
		}
		if p.ctx == ast.ContextNone {
			if _, ok := parent.(*ast.Package); !ok {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("import not inside a package")})
			}
		}
		tok = next(p.lex)
		if p.ctx == ast.ContextNone && tok.typ == tokenLeftParenthesis {
			tok = next(p.lex)
			for tok.typ != tokenRightParenthesis {
				addChild(parent, p.parseImportSpec(tok))
				tok = next(p.lex)
				if tok.typ == tokenSemicolon {
					tok = next(p.lex)
				} else if tok.typ != tokenRightParenthesis {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
				}
			}
			tok = next(p.lex)
			if tok.typ != tokenSemicolon && tok.typ != tokenEOF {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting semicolon or newline", tok)})
			}
		} else {
			addChild(parent, p.parseImportSpec(tok))
			tok = next(p.lex)
			if tok.typ != tokenSemicolon {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
			}
		}
		p.cutSpacesToken = true

	// macro
	case tokenMacro:
		if p.ctx == ast.ContextNone {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("macro statement not in template")})
		}
		if tok.ctx == ast.ContextAttribute || tok.ctx == ast.ContextUnquotedAttribute {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("macro inside an attribute value")})
		}
		for i := len(p.ancestors) - 1; i > 0; i-- {
			switch p.ancestors[i].(type) {
			case ast.For:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end for", tok)})
			case *ast.If:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end if", tok)})
			case *ast.Macro:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end macro", tok)})
			}
		}
		// ident
		tok = next(p.lex)
		if tok.typ != tokenIdentifier {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)})
		}
		if len(tok.txt) == 1 && tok.txt[0] == '_' {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("cannot use _ as value")})
		}
		ident := ast.NewIdentifier(tok.pos, string(tok.txt))
		tok = next(p.lex)
		var parameters []*ast.Identifier
		var ellipsesPos *ast.Position
		if tok.typ == tokenLeftParenthesis {
			// parameters
			parameters = []*ast.Identifier{}
			for {
				tok = next(p.lex)
				if tok.typ != tokenIdentifier {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)})
				}
				if ellipsesPos != nil {
					panic(&SyntaxError{"", *ellipsesPos, fmt.Errorf("cannot use ... with non-final parameter")})
				}
				parameters = append(parameters, ast.NewIdentifier(tok.pos, string(tok.txt)))
				tok = next(p.lex)
				if tok.typ == tokenEllipses {
					ellipsesPos = tok.pos
					tok = next(p.lex)
				}
				if tok.typ == tokenRightParenthesis {
					break
				}
				if tok.typ != tokenComma {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting , or )", tok)})
				}
			}
			tok = next(p.lex)
			if tok.typ != tokenEndStatement {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
			}
		} else if tok.typ != tokenEndStatement {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ( or %%}", tok)})
		}
		pos.End = tok.pos.End
		node = ast.NewMacro(pos, ident, parameters, nil, ellipsesPos != nil, tok.ctx)
		addChild(parent, node)
		p.ancestors = append(p.ancestors, node)
		p.cutSpacesToken = true
		p.isInMacro = true

	// end
	case tokenEnd:
		if p.ctx == ast.ContextNone {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("end statement not in template")})
		}
		if _, ok = parent.(*ast.URL); ok || len(p.ancestors) == 1 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s", tok)})
		}
		if _, ok = parent.(*ast.Block); ok {
			p.ancestors = p.ancestors[:len(p.ancestors)-1]
			parent = p.ancestors[len(p.ancestors)-1]
		}
		tok = next(p.lex)
		if tok.typ != tokenEndStatement {
			tokparent := tok
			tok = next(p.lex)
			if tok.typ != tokenEndStatement {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
			}
			switch parent.(type) {
			case ast.For:
				if tokparent.typ != tokenFor {
					panic(&SyntaxError{"", *tokparent.pos, fmt.Errorf("unexpected %s, expecting for or %%}", tok)})
				}
			case *ast.If:
				if tokparent.typ != tokenIf {
					panic(&SyntaxError{"", *tokparent.pos, fmt.Errorf("unexpected %s, expecting if or %%}", tok)})
				}
			case *ast.Macro:
				if tokparent.typ != tokenMacro {
					panic(&SyntaxError{"", *tokparent.pos, fmt.Errorf("unexpected %s, expecting macro or %%}", tok)})
				}
			}
		}
		parent.Pos().End = tok.pos.End
		p.ancestors = p.ancestors[:len(p.ancestors)-1]
		for {
			parent = p.ancestors[len(p.ancestors)-1]
			if _, ok := parent.(*ast.If); ok {
				parent.Pos().End = tok.pos.End
				p.ancestors = p.ancestors[:len(p.ancestors)-1]
				continue
			}
			break
		}
		if _, ok := parent.(*ast.Macro); ok {
			p.isInMacro = false
		}
		p.cutSpacesToken = true

	// func
	case tokenFunc:
		if p.ctx == ast.ContextNone {
			// Note that parseFunc does not consume the next token because
			// kind is not parseType.
			if _, ok := parent.(*ast.Package); ok {
				node, _ = p.parseFunc(tok, parseFuncDecl)
				// Consumes the semicolon.
				tok = next(p.lex)
				if tok.typ != tokenSemicolon && tok.typ != tokenEOF {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s after top level declaration", tok)})
				}
			} else if len(p.ancestors) == 1 {
				node, _ = p.parseFunc(tok, parseFuncLit|parseFuncDecl)
			} else {
				node, _ = p.parseFunc(tok, parseFuncLit)
			}
			addChild(parent, node)
			return
		}
		fallthrough

	// type declaration
	case tokenType:
		if p.ctx == ast.ContextNone {
			var td *ast.TypeDeclaration
			pos = tok.pos
			tok = next(p.lex)
			if tok.typ == tokenLeftParenthesis {
				// "type" "(" ... ")" .
				for {
					tok = next(p.lex)
					td, tok = p.parseTypeDecl(tok)
					td.Position = pos
					addChild(parent, td)
					if tok.typ == tokenRightParenthesis {
						break
					}
				}
				pos.End = tok.pos.End
			} else {
				// "type" identifier [ "=" ] type .
				td, tok = p.parseTypeDecl(tok)
				pos.End = tok.pos.End
				td.Position = pos
				addChild(parent, td)

			}
			return
		}

	// expression or assignment
	default:
		expressions, tok := p.parseExprList(tok, true, false, false, false)
		if len(expressions) == 0 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting for, if, show, extends, include, macro or end", tok)})
		}
		if len(expressions) > 1 || isAssignmentToken(tok) {
			// Parses assignment.
			assignment, tok := p.parseAssignment(expressions, tok, false, false)
			if assignment == nil {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("expecting expression")})
			}
			if (p.ctx == ast.ContextNone && tok.typ != tokenSemicolon) || (p.ctx != ast.ContextNone && tok.typ != tokenEndStatement) {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
			}
			assignment.Position = &ast.Position{pos.Line, pos.Column, pos.Start, pos.End}
			assignment.Position.End = tok.pos.End
			addChild(parent, assignment)
			p.cutSpacesToken = true
		} else {
			// Parses expression.
			expr := expressions[0]
			if (p.ctx == ast.ContextNone && tok.typ != tokenSemicolon) || (p.ctx != ast.ContextNone && tok.typ != tokenEndStatement) {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
			}
			addChild(parent, expr)
			p.cutSpacesToken = true
		}
	}

	return
}

// parseIdentifiersList returns a list of identifiers separated by commas and
// the next token not used.
func (p *parsing) parseIdentifiersList(tok token) ([]*ast.Identifier, token) {
	idents := []*ast.Identifier{}
	for {
		idents = append(idents, parseIdentifierNode(tok))
		tok = next(p.lex)
		if tok.typ == tokenComma {
			tok = next(p.lex)
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
		panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting name", tok)})
	}
	ident := ast.NewIdentifier(tok.pos, string(tok.txt))
	tok = next(p.lex)
	isAliasDecl := false
	if tok.typ == tokenSimpleAssignment {
		isAliasDecl = true
		tok = next(p.lex)
	}
	// TODO (Gianluca): to review: redeclaring tok?
	typ, tok := p.parseExpr(tok, false, false, true, false)
	td := ast.NewTypeDeclaration(nil, ident, typ, isAliasDecl)
	return td, tok
}

func (p *parsing) parseVarOrConst(tok token, nodePos *ast.Position, kind string) ast.Node {
	if tok.typ != tokenIdentifier {
		panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting name", tok)})
	}
	if kind != "var" && kind != "const" {
		panic("bug: kind must be var or const")
	}
	var exprs []ast.Expression
	var idents []*ast.Identifier
	var typ ast.Expression
	idents, tok = p.parseIdentifiersList(tok)
	if len(idents) == 0 {
		panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting name", tok)})
	}
	switch tok.typ {
	case tokenSimpleAssignment:
		// var/const  a     = ...
		// var/const  a, b  = ...
		exprs, tok = p.parseExprList(token{}, false, false, false, false)
		if len(exprs) == 0 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
		}
	case tokenIdentifier, tokenFunc, tokenMap, tokenLeftBrackets, tokenInterface, tokenMultiplication:
		// var  a     int
		// var  a, b  int
		// var/const  a     int  =  ...
		// var/const  a, b  int  =  ...
		typ, tok = p.parseExpr(tok, false, false, true, false)
		if tok.typ != tokenSimpleAssignment && kind == "const" {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
		}
		if tok.typ == tokenSimpleAssignment {
			// var/const  a     int  =  ...
			// var/const  a, b  int  =  ...
			exprs, tok = p.parseExprList(token{}, false, false, false, false)
			if len(exprs) == 0 {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
			}
		}
	default:
		// var/const  a
		// var/const  a, b
	}
	// Searches the maximum end position.
	var exprPosEnd, identPosEnd, typEndPos int
	if exprs != nil {
		exprPosEnd = exprs[len(exprs)-1].Pos().End
	}
	if idents != nil {
		identPosEnd = idents[len(idents)-1].Pos().End
	}
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
	nodePos.End = endPos
	if kind == "var" {
		return ast.NewVar(nodePos, idents, typ, exprs)
	}
	return ast.NewConst(nodePos, idents, typ, exprs)
}

func (p *parsing) parseImportSpec(tok token) *ast.Import {
	pos := tok.pos
	var ident *ast.Identifier
	if tok.typ == tokenIdentifier {
		ident = ast.NewIdentifier(tok.pos, string(tok.txt))
		tok = next(p.lex)
	} else if tok.typ == tokenPeriod {
		ident = ast.NewIdentifier(tok.pos, ".")
		tok = next(p.lex)
	}
	if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
		panic(fmt.Errorf("unexpected %s, expecting string at %s", tok, tok.pos))
	}
	var path = unquoteString(tok.txt)
	if p.ctx == ast.ContextNone && !validPackageImportPath(path) || p.ctx != ast.ContextNone && !validPath(path) {
		panic(fmt.Errorf("invalid import path %q at %s", path, tok.pos))
	}
	pos.End = tok.pos.End
	return ast.NewImport(pos, ident, path, tok.ctx)
}

// parseAssignment parses an assignment and returns an assignment or, if there
// is no expression, returns nil. tok can be the assignment, declaration,
// increment or decrement token. Panics on error.
func (p *parsing) parseAssignment(variables []ast.Expression, tok token, canBeSwitchGuard bool, nextIsBlockOpen bool) (*ast.Assignment, token) {
	var typ, ok = assignmentType(tok)
	if !ok {
		panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting := or = or comma", tok)})
	}
	for _, v := range variables {
		switch v := v.(type) {
		case *ast.Identifier:
			continue
		case *ast.Selector, *ast.Index:
			if typ != ast.AssignmentDeclaration {
				continue
			}
		case *ast.UnaryOperator:
			if v.Operator() == ast.OperatorMultiplication { // pointer.
				continue
			}
		}
	}
	vp := variables[0].Pos()
	pos := &ast.Position{Line: vp.Line, Column: vp.Column, Start: vp.Start, End: tok.pos.End}
	var values []ast.Expression
	switch typ {
	case ast.AssignmentSimple, ast.AssignmentDeclaration:
		values, tok = p.parseExprList(token{}, false, canBeSwitchGuard, false, nextIsBlockOpen)
		if len(values) == 0 {
			return nil, tok
		}
		pos.End = values[len(values)-1].Pos().End
	default:
		if len(variables) > 1 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting := or = or comma", tok)})
		}
		if typ == ast.AssignmentIncrement || typ == ast.AssignmentDecrement {
			tok = next(p.lex)
		} else {
			values = make([]ast.Expression, 1)
			values[0], tok = p.parseExpr(token{}, false, false, false, false)
		}
	}
	return ast.NewAssignment(pos, variables, typ, values), tok
}

// Parser implements a parser that reads the tree from a Reader and expands
// the nodes Extends, Import and Include. The trees are cached so only one
// call per combination of path and context is made to the reader even if
// several goroutines parse the same paths at the same time.
//
// Returned trees can only be transformed if the parser is no longer used,
// because it would be the cached trees to be transformed and a data race can
// occur. In case, use the function Clone in the astutil package to create a
// clone of the tree and then transform the clone.
type Parser struct {
	reader   Reader
	packages map[string]*GoPackage
	trees    *cache
	// TODO (Gianluca): does packageInfos need synchronized access?
	packageInfos map[string]*PackageInfo // key is path.
	typeCheck    bool
}

// New returns a new Parser that reads the trees from the reader r. typeCheck
// indicates if a type-checking must be done after parsing.
func New(r Reader, packages map[string]*GoPackage, typeCheck bool) *Parser {
	p := &Parser{
		reader:    r,
		packages:  packages,
		trees:     &cache{},
		typeCheck: typeCheck,
	}
	if typeCheck {
		p.packageInfos = make(map[string]*PackageInfo)
	}
	return p
}

// Parse reads the source at path, with the reader, in the ctx context,
// expands the nodes Extends, Import and Include and returns the expanded tree.
//
// Parse is safe for concurrent use.
func (p *Parser) Parse(path string, ctx ast.Context) (*ast.Tree, error) {

	// Path must be absolute.
	if path == "" {
		return nil, ErrInvalidPath
	}
	if path[0] == '/' {
		path = path[1:]
	}
	// Cleans the path by removing "..".
	path, err := toAbsolutePath("/", path)
	if err != nil {
		return nil, err
	}

	pp := &expansion{p.reader, p.trees, p.packages, []string{}}

	tree, err := pp.parsePath(path, ctx)
	if err != nil {
		if err2, ok := err.(*SyntaxError); ok && err2.Path == "" {
			err2.Path = path
		} else if err2, ok := err.(CycleError); ok {
			err = CycleError(path + "\n\t" + string(err2))
		}
		return nil, err
	}

	if p.typeCheck {
		if len(tree.Nodes) == 0 {
			return nil, &SyntaxError{"", ast.Position{1, 1, 0, 0}, fmt.Errorf("expected 'package' or script, found 'EOF'")}
		}
		if _, ok := tree.Nodes[0].(*ast.Package); ok {
			err := checkPackage(tree, p.packages, p.packageInfos)
			if err != nil {
				return nil, err
			}
		} else {
			pkgInfo, err := checkScript(tree, p.packages["main"])
			if err != nil {
				return nil, err
			}
			p.packageInfos["main"] = pkgInfo
		}

	}

	return tree, nil
}

// TypeCheckInfos returns the type-checking infos collected during
// type-checking.
func (p *Parser) TypeCheckInfos() map[string]*PackageInfo {
	return p.packageInfos
}

// expansion is an expansion state.
type expansion struct {
	reader   Reader
	trees    *cache
	packages map[string]*GoPackage
	paths    []string
}

// abs returns path as absolute.
func (pp *expansion) abs(path string) (string, error) {
	var err error
	if path[0] == '/' {
		path, err = toAbsolutePath("/", path[1:])
	} else {
		parent := pp.paths[len(pp.paths)-1]
		dir := parent[:strings.LastIndex(parent, "/")+1]
		path, err = toAbsolutePath(dir, path)
	}
	return path, err
}

// parsePath parses the source at path in context ctx. path must be absolute
// and cleared.
func (pp *expansion) parsePath(path string, ctx ast.Context) (*ast.Tree, error) {

	// Checks if there is a cycle.
	for _, p := range pp.paths {
		if p == path {
			return nil, CycleError(path)
		}
	}

	// Checks if it has already been parsed.
	if tree, ok := pp.trees.get(path, ctx); ok {
		return tree, nil
	}
	defer pp.trees.done(path, ctx)

	src, err := pp.reader.Read(path, ctx)
	if err != nil {
		return nil, err
	}

	tree, err := ParseSource(src, ctx)
	if err != nil {
		return nil, err
	}
	tree.Path = path

	// Expands the nodes.
	pp.paths = append(pp.paths, path)
	err = pp.expand(tree.Nodes, ctx)
	if err != nil {
		if e, ok := err.(*SyntaxError); ok && e.Path == "" {
			e.Path = path
		}
		return nil, err
	}
	pp.paths = pp.paths[:len(pp.paths)-1]

	// Adds the tree to the cache.
	pp.trees.add(path, ctx, tree)

	return tree, nil
}

// expand expands the nodes parsing the sub-trees in context ctx.
func (pp *expansion) expand(nodes []ast.Node, ctx ast.Context) error {

	for _, node := range nodes {

		switch n := node.(type) {

		case *ast.Package:
			err := pp.expand(n.Declarations, ctx)
			if err != nil {
				return err
			}

		case *ast.If:

			for {
				err := pp.expand(n.Then.Nodes, ctx)
				if err != nil {
					return err
				}
				switch e := n.Else.(type) {
				case *ast.If:
					n = e
					continue
				case *ast.Block:
					err := pp.expand(e.Nodes, ctx)
					if err != nil {
						return err
					}
				}
				break
			}

		case *ast.For:

			err := pp.expand(n.Body, ctx)
			if err != nil {
				return err
			}

		case *ast.ForRange:

			err := pp.expand(n.Body, ctx)
			if err != nil {
				return err
			}

		case *ast.Switch:
			var err error
			for _, c := range n.Cases {
				err = pp.expand(c.Body, ctx)
				if err != nil {
					return err
				}
			}

		case *ast.TypeSwitch:
			var err error
			for _, c := range n.Cases {
				err = pp.expand(c.Body, ctx)
				if err != nil {
					return err
				}
			}

		case *ast.Macro:
			err := pp.expand(n.Body, ctx)
			if err != nil {
				return err
			}

		case *ast.Extends:

			if len(pp.paths) > 1 {
				return &SyntaxError{"", *(n.Pos()), fmt.Errorf("extended, imported and included paths can not have extends")}
			}
			absPath, err := pp.abs(n.Path)
			if err != nil {
				return err
			}
			n.Tree, err = pp.parsePath(absPath, n.Context)
			if err != nil {
				if err == ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if err == ErrNotExist {
					err = &SyntaxError{"", *(n.Pos()), fmt.Errorf("extends path %q does not exist", absPath)}
				} else if err2, ok := err.(CycleError); ok {
					err = CycleError("imports " + string(err2))
				}
				return err
			}

		case *ast.Import:

			absPath, err := pp.abs(n.Path)
			if err != nil {
				return err
			}
			if ctx == ast.ContextNone {
				found := false
				for path := range pp.packages {
					if path == n.Path {
						found = true
						break
					}
				}
				if found {
					continue
				}
				n.Tree, err = pp.parsePath(absPath+".go", n.Context)
				if err != nil {
					if err == ErrInvalidPath {
						err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
					} else if err == ErrNotExist {
						err = &SyntaxError{"", *(n.Pos()), fmt.Errorf("cannot find package \"%s\"", n.Path)}
					} else if err2, ok := err.(CycleError); ok {
						err = CycleError("imports " + string(err2))
					}
					return err
				}
			} else {
				n.Tree, err = pp.parsePath(absPath, n.Context)
				if err != nil {
					if err == ErrInvalidPath {
						err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
					} else if err == ErrNotExist {
						err = &SyntaxError{"", *(n.Pos()), fmt.Errorf("import path %q does not exist", absPath)}
					} else if err2, ok := err.(CycleError); ok {
						err = CycleError("imports " + string(err2))
					}
					return err
				}
			}

		case *ast.Include:

			absPath, err := pp.abs(n.Path)
			if err != nil {
				return err
			}
			n.Tree, err = pp.parsePath(absPath, n.Context)
			if err != nil {
				if err == ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if err == ErrNotExist {
					err = &SyntaxError{"", *(n.Pos()), fmt.Errorf("included path %q does not exist", absPath)}
				} else if err2, ok := err.(CycleError); ok {
					err = CycleError("include " + string(err2))
				}
				return err
			}

		}

	}

	return nil
}

// addChild adds node as child of parent.
func addChild(parent ast.Node, node ast.Node) {
	switch n := parent.(type) {
	case *ast.Tree:
		n.Nodes = append(n.Nodes, node)
	case *ast.Package:
		n.Declarations = append(n.Declarations, node)
	case *ast.URL:
		n.Value = append(n.Value, node)
	case *ast.Macro:
		n.Body = append(n.Body, node)
	case *ast.For:
		n.Body = append(n.Body, node)
	case *ast.ForRange:
		n.Body = append(n.Body, node)
	case *ast.If:
		if n.Else != nil {
			panic("child already added to if node")
		}
		n.Else = node
	case *ast.Block:
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
	default:
		panic("scrigo/parser: unexpected parent node")
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
