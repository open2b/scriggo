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
	// ErrInvalidPackagePath is returned from the Parse method and a Reader
	// when the path argument is not valid.
	ErrInvalidPackagePath = errors.New("scriggo: invalid path")

	ErrNotCanonicalImportPath = errors.New("scriggo: non-canonical import path")

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
	Path string
	Pos  ast.Position
	Err  error
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("%s:%s: syntax error: %s", e.Path, e.Pos, e.Err)
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

	// Position of the last fallthrough token, used for error messages.
	lastFallthroughTokenPos ast.Position
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

// ParseSource parses a program or script. isScript reports whether it is a
// script and shebang reports whether a script can have the shebang as first
// line.
//
// Returns the AST tree and, only if it is a program, the dependencies for the
// type checker.
// TODO(Gianluca): path validation must be moved to parser.
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

	// Reads the tokens.
	tok := p.next()
TOKENS:
	for {
		switch tok.typ {
		case tokenShebangLine:
			if !shebang {
				return nil, &SyntaxError{"", *tok.pos, fmt.Errorf("illegal character U+0023 '#'")}
			}
		default:
			tok = p.parseStatement(tok)
			continue
		case tokenEOF:
			if len(p.ancestors) > 1 {
				switch p.ancestors[1].(type) {
				case *ast.Package:
				case *ast.Label:
					return nil, &SyntaxError{"", *tok.pos, fmt.Errorf("missing statement after label")}
				default:
					if len(p.ancestors) > 2 {
						return nil, &SyntaxError{"", *tok.pos, fmt.Errorf("unexpected EOF, expecting }")}
					}
				}
			}
			break TOKENS
		}
		tok = p.next()
	}

	return tree, nil
}

// ParseTemplateSource parses src in the context ctx and returns the parsed
// tree. Nodes Extends, Import and Include are not be expanded.
func ParseTemplateSource(src []byte, ctx ast.Context) (tree *ast.Tree, deps PackageDeclsDeps, err error) {

	switch ctx {
	case ast.ContextText, ast.ContextHTML, ast.ContextCSS, ast.ContextJavaScript:
	default:
		return nil, nil, errors.New("scriggo: invalid context. Valid contexts are Text, HTML, CSS and JavaScript")
	}

	// Tree result of the expansion.
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

	// Current line.
	var line = 0

	// First Text node of the current line.
	var firstText *ast.Text

	// Number of non-text tokens in current line.
	var tokensInLine = 0

	// Index of the last byte.
	var end = len(src) - 1

	// Reads the tokens.
	tok := p.next()
TOKENS:
	for {

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

		switch tok.typ {

		// EOF
		case tokenEOF:
			if len(p.ancestors) > 1 {
				return nil, nil, &SyntaxError{"", *tok.pos, fmt.Errorf("unexpected EOF, expecting {%% end %%}")}
			}
			break TOKENS

		// Text
		case tokenText:
			parent := p.ancestors[len(p.ancestors)-1]
			switch n := parent.(type) {
			case *ast.Switch:
				if len(n.Cases) == 0 {
					// TODO (Gianluca): this "if" should be moved before the switch that precedes it.
					if containsOnlySpaces(text.Text) {
						n.LeadingText = text
					}
					return nil, nil, &SyntaxError{"", *tok.pos, fmt.Errorf("unexpected text, expecting case of default or {%% end %%}")}
				}
				lastCase := n.Cases[len(n.Cases)-1]
				if lastCase.Fallthrough {
					if containsOnlySpaces(text.Text) {
						continue
					}
					return nil, nil, &SyntaxError{"", p.lastFallthroughTokenPos, fmt.Errorf("fallthrough statement out of place")}
				}
			case *ast.TypeSwitch:
				if len(n.Cases) == 0 {
					// TODO (Gianluca): this "if" should be moved before the switch that precedes it.
					if containsOnlySpaces(text.Text) {
						n.LeadingText = text
					}
					return nil, nil, &SyntaxError{"", *tok.pos, fmt.Errorf("unexpected text, expecting case of default or {%% end %%}")}
				}
			case *ast.Select:
				if len(n.Cases) == 0 {
					// TODO (Gianluca): this "if" should be moved before the switch that precedes it.
					if containsOnlySpaces(text.Text) {
						n.LeadingText = text
					}
					return nil, nil, &SyntaxError{"", *tok.pos, fmt.Errorf("unexpected text, expecting case of default or {%% end %%}")}
				}
			}
			p.addChild(text)

		// StartURL
		case tokenStartURL:
			node := ast.NewURL(tok.pos, tok.tag, tok.att, nil)
			p.addChild(node)
			p.ancestors = append(p.ancestors, node)

		// EndURL
		case tokenEndURL:
			pos := p.ancestors[len(p.ancestors)-1].Pos()
			pos.End = tok.pos.End - 1
			p.ancestors = p.ancestors[:len(p.ancestors)-1]

		// {%
		case tokenStartStatement:
			tokensInLine++
			tok = p.parseStatement(tok)
			continue

		// {{ }}
		case tokenStartValue:
			tokensInLine++
			expr, tok2 := p.parseExpr(token{}, false, false, false)
			if expr == nil {
				return nil, nil, &SyntaxError{"", *tok2.pos, fmt.Errorf("expecting expression")}
			}
			if tok2.typ != tokenEndValue {
				return nil, nil, &SyntaxError{"", *tok2.pos, fmt.Errorf("unexpected %s, expecting }}", tok2)}
			}
			tok.pos.End = tok2.pos.End
			var node = ast.NewShow(tok.pos, expr, tok.ctx)
			p.addChild(node)

		// comment
		case tokenComment:
			tokensInLine++
			var node = ast.NewComment(tok.pos, string(tok.txt[2:len(tok.txt)-2]))
			p.addChild(node)
			p.cutSpacesToken = true

		default:
			return nil, nil, &SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s", tok)}

		}

		tok = p.next()

	}

	return tree, deps, nil
}

// parseStatement parses a statement. Panics on error.
func (p *parsing) parseStatement(tok token) token {

	var node ast.Node

	var pos = tok.pos

	var expr ast.Expression

	var ok bool

	if p.isTemplate {
		tok = p.next()
	}

LABEL:

	// Parent is always the last ancestor.
	parent := p.ancestors[len(p.ancestors)-1]

	switch s := parent.(type) {
	case *ast.Tree:
		if !p.isTemplate && !p.isScript && tok.typ != tokenPackage {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("expected 'package', found '%s'", tok)})
		}
	case *ast.Package:
		switch tok.typ {
		case tokenImport, tokenFunc, tokenVar, tokenConst, tokenType:
		default:
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("non-declaration statement outside function body (%q)", tok)})
		}
	case *ast.Switch:
		if len(s.Cases) == 0 && tok.typ != tokenCase && tok.typ != tokenDefault && tok.typ != tokenEnd && tok.typ != tokenRightBraces {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting case of default or {%% end %%}", tok.String())})
		}
	case *ast.TypeSwitch:
		if len(s.Cases) == 0 && tok.typ != tokenCase && tok.typ != tokenDefault && tok.typ != tokenEnd && tok.typ != tokenRightBraces {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting case of default or {%% end %%}", tok.String())})
		}
	case *ast.Select:
		if len(s.Cases) == 0 && tok.typ != tokenCase && tok.typ != tokenDefault && tok.typ != tokenEnd && tok.typ != tokenRightBraces {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting case of default or {%% end %%}", tok.String())})
		}
	}

	switch tok.typ {

	// ;
	case tokenSemicolon:
		if p.ctx != ast.ContextGo {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected semicolon, expecting statement")})
		}
		tok = p.next()

	// package
	case tokenPackage:
		if tree, ok := parent.(*ast.Tree); !ok || p.ctx != ast.ContextGo || len(tree.Nodes) > 0 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected package, expecting statement")})
		}
		tok = p.next()
		if tok.typ != tokenIdentifier {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("expected 'IDENT', found %q", string(tok.txt))})
		}
		name := string(tok.txt)
		if name == "_" {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("invalid package name _")})
		}
		pos.End = tok.pos.End
		tok = p.next()
		if tok.typ != tokenSemicolon {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting semicolon or newline", string(tok.txt))})
		}
		tok = p.next()
		node = ast.NewPackage(pos, name, nil)
		p.addChild(node)
		p.ancestors = append(p.ancestors, node)

	// for
	case tokenFor:
		var node ast.Node
		var init *ast.Assignment
		var assignmentType ast.AssignmentType
		var variables []ast.Expression
		variables, tok = p.parseExprList(token{}, false, false, true)
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
			blank := ast.NewIdentifier(&ast.Position{Line: ipos.Line, Column: ipos.Column, Start: ipos.Start, End: ipos.Start}, "_")
			// Parses the slice expression.
			// TODO (Gianluca): nextIsBlockOpen should be true?
			expr, tok = p.parseExpr(token{}, false, false, false)
			if expr == nil {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
			}
			assignment := ast.NewAssignment(&ast.Position{Line: ipos.Line, Column: ipos.Column, Start: ipos.Start, End: expr.Pos().End},
				[]ast.Expression{blank, ident}, ast.AssignmentDeclaration, []ast.Expression{expr})
			pos.End = tok.pos.End
			node = ast.NewForRange(pos, assignment, nil)
		case tokenLeftBraces, tokenEndStatement:
			if (p.ctx == ast.ContextGo) != (tok.typ == tokenLeftBraces) {
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
			expr, tok = p.parseExpr(token{}, false, false, true)
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
				expr, tok = p.parseExpr(token{}, false, false, true)
				if expr == nil {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
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
				condition, tok = p.parseExpr(token{}, false, false, true)
				if tok.typ != tokenSemicolon {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expected semicolon", tok)})
				}
				// Parses the post iteration statement.
				var post *ast.Assignment
				variables, tok = p.parseExprList(token{}, false, false, true)
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
		if node == nil {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or %%}", tok)})
		}
		p.parseEndStatement(tok, tokenLeftBraces)
		tok = p.next()
		p.addChild(node)
		p.ancestors = append(p.ancestors, node)
		p.cutSpacesToken = true

	// break
	case tokenBreak:
		var label *ast.Identifier
		tok = p.next()
		if tok.typ == tokenIdentifier {
			label = ast.NewIdentifier(tok.pos, string(tok.txt))
			tok = p.next()
		}
		pos.End = tok.pos.End
		tok = p.parseEnd(tok)
		node = ast.NewBreak(pos, label)
		p.addChild(node)
		p.cutSpacesToken = true

	// continue
	case tokenContinue:
		var label *ast.Identifier
		tok = p.next()
		if tok.typ == tokenIdentifier {
			label = ast.NewIdentifier(tok.pos, string(tok.txt))
			tok = p.next()
		}
		pos.End = tok.pos.End
		tok = p.parseEnd(tok)
		node = ast.NewContinue(pos, label)
		p.addChild(node)
		p.cutSpacesToken = true

	// switch
	case tokenSwitch:
		node = p.parseSwitch(pos)
		p.addChild(node)
		p.ancestors = append(p.ancestors, node)
		p.cutSpacesToken = true
		tok = p.next()

	// case
	case tokenCase:
		var expressions []ast.Expression
		switch parent.(type) {
		case *ast.Switch, *ast.TypeSwitch:
			expressions, tok = p.parseExprList(token{}, false, false, false)
			pos.End = tok.pos.End
			p.parseEndStatement(tok, tokenColon)
			tok = p.next()
			node := ast.NewCase(pos, expressions, nil, false)
			p.addChild(node)
		case *ast.Select:
			expressions, tok = p.parseExprList(token{}, false, false, false)
			if len(expressions) == 0 {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
			}
			var comm ast.Node
			if len(expressions) > 1 || isAssignmentToken(tok) {
				comm, tok = p.parseAssignment(expressions, tok, false, false)
			} else {
				if tok.typ == tokenArrow {
					channel := expressions[0]
					sendPos := tok.pos
					var value ast.Expression
					value, tok = p.parseExpr(token{}, false, false, false)
					if value == nil {
						panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
					}
					comm = ast.NewSend(sendPos, channel, value)
				} else {
					comm = expressions[0]
				}
			}
			if tok.typ != tokenColon {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
			}
			pos.End = tok.pos.End
			node := ast.NewSelectCase(pos, comm, nil)
			p.addChild(node)
			tok = p.next()
		default:
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected case, expecting %%}")})
		}

	// default
	case tokenDefault:
		switch parent.(type) {
		case *ast.Switch, *ast.TypeSwitch:
			tok = p.next()
			pos.End = tok.pos.End
			p.parseEndStatement(tok, tokenColon)
			tok = p.next()
			node := ast.NewCase(pos, nil, nil, false)
			p.addChild(node)
			p.cutSpacesToken = true
		case *ast.Select:
			tok = p.next()
			pos.End = tok.pos.End
			p.parseEndStatement(tok, tokenColon)
			tok = p.next()
			node := ast.NewSelectCase(pos, nil, nil)
			p.addChild(node)
			p.cutSpacesToken = true
		default:
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected default, expecting %%}")})
		}

	// fallthrough
	case tokenFallthrough:
		// TODO (Gianluca): fallthrough must be implemented as an ast node.
		p.lastFallthroughTokenPos = *tok.pos
		tok2 := p.next()
		tok = p.parseEnd(tok2)
		switch s := parent.(type) {
		case *ast.Switch:
			lastCase := s.Cases[len(s.Cases)-1]
			// TODO (Gianluca): move this check to type-checker:
			if lastCase.Fallthrough {
				panic(&SyntaxError{"", *tok2.pos, fmt.Errorf("fallthrough statement out of place")})
			}
			lastCase.Fallthrough = true
		case *ast.TypeSwitch:
			// TODO (Gianluca): move this check to type-checker:
			panic(&SyntaxError{"", *tok2.pos, fmt.Errorf("cannot fallthrough in type switch")})
		default:
			// TODO (Gianluca): move this check to type-checker:
			panic(&SyntaxError{"", *tok2.pos, fmt.Errorf("fallthrough statement out of place")})
		}
		pos.End = tok2.pos.End
		p.cutSpacesToken = true

	// select
	case tokenSelect:
		tok = p.next()
		p.parseEndStatement(tok, tokenLeftBraces)
		node = ast.NewSelect(pos, nil, nil)
		p.addChild(node)
		p.ancestors = append(p.ancestors, node)
		p.cutSpacesToken = true
		tok = p.next()

	// {
	case tokenLeftBraces:
		if p.ctx != ast.ContextGo {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting statement", tok)})
		}
		node = ast.NewBlock(tok.pos, nil)
		p.addChild(node)
		p.ancestors = append(p.ancestors, node)
		p.cutSpacesToken = true
		tok = p.next()

	// }
	case tokenRightBraces:
		if p.ctx != ast.ContextGo {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting statement", tok)})
		}
		if len(p.ancestors) == 1 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("not opened brace")})
		}
		if _, ok := parent.(*ast.Label); ok {
			p.ancestors = p.ancestors[:len(p.ancestors)-1]
			parent = p.ancestors[len(p.ancestors)-1]
		}
		bracesEnd := tok.pos.End
		parent.Pos().End = bracesEnd
		p.ancestors = p.ancestors[:len(p.ancestors)-1]
		parent = p.ancestors[len(p.ancestors)-1]
		tok = p.next()
		switch tok.typ {
		case tokenElse:
		case tokenSemicolon:
			tok = p.next()
			fallthrough
		case tokenRightBraces:
			for {
				if _, ok := parent.(*ast.If); ok {
					parent.Pos().End = bracesEnd
					p.ancestors = p.ancestors[:len(p.ancestors)-1]
					parent = p.ancestors[len(p.ancestors)-1]
				} else {
					return tok
				}
			}
		case tokenEOF:
			// TODO(marco): check if it is correct.
			return p.next()
		default:
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s at end of statement", tok)})
		}
		fallthrough

	// else
	case tokenElse:
		if p.ctx != ast.ContextGo {
			// Closes the "then" block.
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
		tok = p.next()
		if p.ctx == ast.ContextGo && tok.typ == tokenLeftBraces || p.ctx != ast.ContextGo && tok.typ == tokenEndStatement {
			// "else"
			var blockPos *ast.Position
			if p.ctx == ast.ContextGo {
				blockPos = tok.pos
			}
			elseBlock := ast.NewBlock(blockPos, nil)
			p.addChild(elseBlock)
			p.ancestors = append(p.ancestors, elseBlock)
			return p.next()
		}
		if tok.typ != tokenIf { // "else if"
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting if or %%}", tok)})
		}
		fallthrough

	// if
	case tokenIf:
		ifPos := tok.pos
		var expressions []ast.Expression
		expressions, tok = p.parseExprList(token{}, false, false, true)
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
			expr, tok = p.parseExpr(token{}, false, false, true)
			if expr == nil {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("missing condition in if statement")})
			}
		} else {
			expr = expressions[0]
		}
		p.parseEndStatement(tok, tokenLeftBraces)
		pos.End = tok.pos.End
		var blockPos *ast.Position
		if p.ctx == ast.ContextGo {
			blockPos = tok.pos
		}
		tok = p.next()
		then := ast.NewBlock(blockPos, nil)
		if _, ok := parent.(*ast.If); !ok {
			ifPos = pos
		}
		node = ast.NewIf(ifPos, assignment, expr, then, nil)
		p.addChild(node)
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
		var values []ast.Expression
		values, tok = p.parseExprList(token{}, false, false, false)
		tok = p.parseEnd(tok)
		if len(values) > 0 {
			pos.End = values[len(values)-1].Pos().End
		}
		node = ast.NewReturn(pos, values)
		p.addChild(node)

	// include
	case tokenInclude:
		if tok.ctx == ast.ContextAttribute || tok.ctx == ast.ContextUnquotedAttribute {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("include statement inside an attribute value")})
		}
		// path
		tok = p.next()
		if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting string", tok)})
		}
		var path = unquoteString(tok.txt)
		if !ValidPath(path) {
			panic(fmt.Errorf("invalid path %q at %s", path, tok.pos))
		}
		tok = p.next()
		if tok.typ != tokenEndStatement {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ( or %%}", tok)})
		}
		pos.End = tok.pos.End
		node = ast.NewInclude(pos, path, tok.ctx)
		p.addChild(node)
		p.cutSpacesToken = true
		tok = p.next()

	// show
	case tokenShow:
		if tok.ctx == ast.ContextAttribute || tok.ctx == ast.ContextUnquotedAttribute {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("show statement inside an attribute value")})
		}
		tok = p.next()
		if tok.typ != tokenIdentifier {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)})
		}
		// TODO(Gianluca): move to typechecker.
		if len(tok.txt) == 1 && tok.txt[0] == '_' {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("cannot use _ as value")})
		}
		macro := ast.NewIdentifier(tok.pos, string(tok.txt))
		tok = p.next()
		// import
		var impor *ast.Identifier
		if tok.typ == tokenPeriod {
			tok = p.next()
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
			tok = p.next()
		}
		var args []ast.Expression
		var isVariadic bool
		if tok.typ == tokenLeftParenthesis {
			args, tok = p.parseExprList(token{}, false, false, false)
			if tok.typ == tokenEllipsis {
				if len(args) == 0 {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected ..., expecting expression")})
				}
				isVariadic = true
				tok = p.next()
			}
			if tok.typ != tokenRightParenthesis {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or )", tok)})
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
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s after or in show macro, expecting ignore, todo or error", tok)})
				}
				tok = p.next()
			}
		}
		pos.End = tok.pos.End
		if impor == nil {
			node = ast.NewShowMacro(pos, macro, args, isVariadic, or, tok.ctx)
		} else {
			node = ast.NewShowMacro(pos, ast.NewSelector(macro.Pos(), impor, macro.Name), args, isVariadic, or, tok.ctx)
		}
		p.addChild(node)
		p.parseEndStatement(tok, tokenEndStatement)
		tok = p.next()
		p.cutSpacesToken = true

	// extends
	case tokenExtends:
		if tok.ctx != p.ctx {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("extends not in %s content", p.ctx)})
		}
		if p.hasExtend {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("extends already exists")})
		}
		tree := p.ancestors[0].(*ast.Tree)
		for _, node := range tree.Nodes {
			switch node.(type) {
			case *ast.Text, *ast.Comment:
			default:
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("extends can only be the first statement")})
			}
		}
		tok = p.next()
		if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting string", tok)})
		}
		var path = unquoteString(tok.txt)
		if !ValidPath(path) {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("invalid extends path %q", path)})
		}
		tok = p.next()
		if tok.typ != tokenEndStatement {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
		}
		pos.End = tok.pos.End
		node = ast.NewExtends(pos, path, tree.Context)
		p.addChild(node)
		p.hasExtend = true
		tok = p.next()

	// var or const
	case tokenVar, tokenConst:
		var decType = tok.typ
		if tok.ctx != p.ctx {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("%s declaration not in %s content", decType, p.ctx)})
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
					tok = p.parseEnd(tok)
					break
				}
				prevNode, tok = p.parseVarOrConst(tok, pos, decType)
				switch tok.typ {
				case tokenSemicolon:
					tok = p.next()
				case tokenRightParenthesis:
				default:
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting semicolon or newline or )", tok)})
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
			node, tok = p.parseVarOrConst(tok, pos, decType)
			tok = p.parseEnd(tok)
			p.addChild(node)
		}

	// import
	case tokenImport:
		var outOfOrder bool
		switch p := parent.(type) {
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
			if tok.ctx == ast.ContextGo {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected import, expecting }")})
			} else {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected import, expecting statement")})
			}
		}
		if tok.ctx != p.ctx {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("import not in %s content", p.ctx)})
		}
		tok = p.next()
		if p.ctx == ast.ContextGo && tok.typ == tokenLeftParenthesis {
			tok = p.next()
			for tok.typ != tokenRightParenthesis {
				p.addChild(p.parseImportSpec(tok))
				tok = p.next()
				if tok.typ == tokenSemicolon {
					tok = p.next()
				} else if tok.typ != tokenRightParenthesis {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
				}
			}
			tok = p.next()
			if tok.typ != tokenSemicolon && tok.typ != tokenEOF {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting semicolon or newline", tok)})
			}
			tok = p.next()
		} else {
			p.addChild(p.parseImportSpec(tok))
			tok = p.next()
			p.parseEndStatement(tok, tokenSemicolon)
			tok = p.next()
		}
		p.cutSpacesToken = true

	// macro
	case tokenMacro:
		if len(p.ancestors) > 1 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected macro in statement scope")})
		}
		if tok.ctx != p.ctx {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("macro not in %s content", p.ctx)})
		}
		// Parses the macro name.
		tok = p.next()
		if tok.typ != tokenIdentifier {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting name", tok.txt)})
		}
		var ident = ast.NewIdentifier(tok.pos, string(tok.txt))
		tok = p.next()
		var parameters []*ast.Field
		var isVariadic bool
		if tok.typ == tokenLeftParenthesis {
			// Parses the macro parameters.
			names := map[string]struct{}{}
			var endPos *ast.Position
			parameters, isVariadic, endPos = p.parseFuncFields(tok, names, false)
			pos.End = endPos.End
			tok = p.next()
		}
		if tok.typ != tokenEndStatement {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
		}
		// Makes the macro node.
		typ := ast.NewFuncType(nil, parameters, nil, isVariadic)
		pos.End = tok.pos.End
		typ.Position = pos
		node := ast.NewMacro(pos, ident, typ, nil, tok.ctx)
		p.addChild(node)
		p.ancestors = append(p.ancestors, node)
		p.cutSpacesToken = true
		tok = p.next()

	// end
	case tokenEnd:
		if _, ok = parent.(*ast.URL); ok || len(p.ancestors) == 1 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s", tok)})
		}
		if _, ok = parent.(*ast.Block); ok {
			p.ancestors = p.ancestors[:len(p.ancestors)-1]
			parent = p.ancestors[len(p.ancestors)-1]
		}
		tok = p.next()
		if tok.typ != tokenEndStatement {
			parentTok := tok
			tok = p.next()
			if tok.typ != tokenEndStatement {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
			}
			switch parent.(type) {
			case ast.For:
				if parentTok.typ != tokenFor {
					panic(&SyntaxError{"", *parentTok.pos, fmt.Errorf("unexpected %s, expecting for or %%}", tok)})
				}
			case *ast.If:
				if parentTok.typ != tokenIf {
					panic(&SyntaxError{"", *parentTok.pos, fmt.Errorf("unexpected %s, expecting if or %%}", tok)})
				}
			case *ast.Macro:
				if parentTok.typ != tokenMacro {
					panic(&SyntaxError{"", *parentTok.pos, fmt.Errorf("unexpected %s, expecting macro or %%}", tok)})
				}
			case ast.Switch, ast.TypeSwitch:
				if parentTok.typ != tokenSwitch {
					panic(&SyntaxError{"", *parentTok.pos, fmt.Errorf("unexpected %s, expecting switch or %%}", tok)})
				}
			case ast.Select:
				if parentTok.typ != tokenSelect {
					panic(&SyntaxError{"", *parentTok.pos, fmt.Errorf("unexpected %s, expecting select or %%}", tok)})
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
		p.cutSpacesToken = true
		tok = p.next()

	// type
	case tokenType:
		var td *ast.TypeDeclaration
		pos = tok.pos
		tok = p.next()
		if tok.typ == tokenLeftParenthesis {
			// "type" "(" ... ")" .
			tok = p.next()
			for tok.typ != tokenRightParenthesis {
				td, tok = p.parseTypeDecl(tok)
				td.Position = pos
				p.addChild(td)
				if tok.typ == tokenSemicolon {
					tok = p.next()
				} else if tok.typ != tokenRightParenthesis {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting semicolon or newline or )", tok)})
				}
			}
			pos.End = tok.pos.End
			tok = p.next()
		} else {
			// "type" identifier [ "=" ] type .
			td, tok = p.parseTypeDecl(tok)
			pos.End = tok.pos.End
			td.Position = pos
			p.addChild(td)
		}
		tok = p.parseEnd(tok)

	// defer or go
	case tokenDefer, tokenGo:
		keyword := tok.typ
		tok = p.next()
		expr, tok = p.parseExpr(tok, false, false, false)
		// Errors on defer and go statements must be type checker errors and not syntax errors.
		var call *ast.Call
		switch expr := expr.(type) {
		default:
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("expression in %s must be function call", keyword)})
		case *ast.Parenthesis:
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("expression in %s must not be parenthesized", keyword)})
		case *ast.Call:
			call = expr
		}
		pos.End = tok.pos.End
		var node ast.Node
		if keyword == tokenDefer {
			node = ast.NewDefer(pos, call)
		} else {
			node = ast.NewGo(pos, call)
		}
		p.addChild(node)
		tok = p.parseEnd(tok)

	// goto
	case tokenGoto:
		if tok.ctx != ast.ContextGo {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected goto outside function body")})
		}
		tok = p.next()
		if tok.typ != tokenIdentifier {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting name", tok)})
		}
		pos.End = tok.pos.End
		node := ast.NewGoto(pos, ast.NewIdentifier(tok.pos, string(tok.txt)))
		p.addChild(node)
		tok = p.next()
		tok = p.parseEnd(tok)

	// func
	case tokenFunc:
		if p.ctx == ast.ContextGo {
			// Note that parseFunc does not consume the next token because
			// kind is not parseType.
			switch parent.(type) {
			case *ast.Tree, *ast.Package:
				node, _ = p.parseFunc(tok, parseFuncDecl)
				// Consumes the semicolon.
				tok = p.next()
				if tok.typ != tokenSemicolon && tok.typ != tokenEOF {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s after top level declaration", tok)})
				}
				p.addChild(node)
				return p.next()
			}
		}
		fallthrough

	// assignment, send, label or expression
	default:
		var expressions []ast.Expression
		expressions, tok = p.parseExprList(tok, false, false, false)
		if len(expressions) == 0 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting for, if, show, extends, include, macro or end", tok)})
		}
		if len(expressions) > 1 || isAssignmentToken(tok) {
			// Parses assignment.
			var assignment *ast.Assignment
			assignment, tok = p.parseAssignment(expressions, tok, false, false)
			if assignment == nil {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("expecting expression")})
			}
			assignment.Position = &ast.Position{Line: pos.Line, Column: pos.Column, Start: pos.Start, End: pos.End}
			assignment.Position.End = tok.pos.End
			tok = p.parseEnd(tok)
			p.addChild(assignment)
			p.cutSpacesToken = true
		} else if tok.typ == tokenArrow {
			// Parses send.
			channel := expressions[0]
			var value ast.Expression
			value, tok = p.parseExpr(token{}, false, false, false)
			if value == nil {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
			}
			tok = p.parseEnd(tok)
			node := ast.NewSend(pos, channel, value)
			node.Position = &ast.Position{Line: pos.Line, Column: pos.Column, Start: pos.Start, End: value.Pos().End}
			p.addChild(node)
			p.cutSpacesToken = true
		} else {
			// Parses expression.
			expr := expressions[0]
			if ident, ok := expr.(*ast.Identifier); ok && tok.typ == tokenColon {
				if p.isTemplate {
					if _, ok := parent.(*ast.Label); ok {
						panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected label, expecting statement")})
					}
				}
				node := ast.NewLabel(pos, ident, nil)
				node.Position = &ast.Position{Line: pos.Line, Column: pos.Column, Start: pos.Start, End: tok.pos.End}
				p.addChild(node)
				p.ancestors = append(p.ancestors, node)
				p.cutSpacesToken = true
				if p.isTemplate {
					tok = p.next()
					if tok.typ == tokenEndStatement || tok.typ == tokenEOF {
						panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting statement", tok)})
					}
					goto LABEL
				}
				tok = p.next()
			} else {
				tok = p.parseEnd(tok)
				p.addChild(expr)
				p.cutSpacesToken = true
			}
		}
	}

	return tok
}

func (p *parsing) parseEnd(tok token) token {
	if p.isTemplate {
		if tok.typ != tokenEndStatement {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("expecting expression")})
		}
		return p.next()
	}
	if tok.typ == tokenSemicolon {
		return p.next()
	}
	if tok.typ != tokenRightBraces {
		panic(&SyntaxError{"", *tok.pos, fmt.Errorf("expecting expression")})
	}
	return tok
}

func (p *parsing) parseEndStatement(tok token, want tokenTyp) {
	if p.isTemplate {
		if tok.typ == tokenSemicolon {
			tok = p.next()
		}
		want = tokenEndStatement
	}
	if tok.typ != want {
		panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %s", tok, want)})
	}
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
		panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting name", tok)})
	}
	ident := ast.NewIdentifier(tok.pos, string(tok.txt))
	tok = p.next()
	isAliasDecl := false
	if tok.typ == tokenSimpleAssignment {
		isAliasDecl = true
		tok = p.next()
	}
	var typ ast.Expression
	typ, tok = p.parseExpr(tok, false, true, false)
	td := ast.NewTypeDeclaration(nil, ident, typ, isAliasDecl)
	return td, tok
}

func (p *parsing) parseVarOrConst(tok token, nodePos *ast.Position, decType tokenTyp) (ast.Node, token) {
	if tok.typ != tokenIdentifier {
		panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting name", tok)})
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
		exprs, tok = p.parseExprList(token{}, false, false, false)
		if len(exprs) == 0 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
		}
	case tokenIdentifier, tokenFunc, tokenMap, tokenLeftBrackets, tokenInterface, tokenMultiplication, tokenChan, tokenArrow, tokenStruct:
		// var  a     int
		// var  a, b  int
		// var/const  a     int  =  ...
		// var/const  a, b  int  =  ...
		typ, tok = p.parseExpr(tok, false, true, false)
		if tok.typ != tokenSimpleAssignment && decType == tokenConst {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
		}
		if tok.typ == tokenSimpleAssignment {
			// var/const  a     int  =  ...
			// var/const  a, b  int  =  ...
			exprs, tok = p.parseExprList(token{}, false, false, false)
			if len(exprs) == 0 {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)})
			}
		}
	case tokenSemicolon, tokenRightParenthesis:
		// const c
		// const c, d
		if decType == tokenVar {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting type", tok)})
		}
	default:
		panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting type", tok)})
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
	if decType == tokenVar {
		return ast.NewVar(nodePos, idents, typ, exprs), tok
	}
	return ast.NewConst(nodePos, idents, typ, exprs), tok
}

func (p *parsing) parseImportSpec(tok token) *ast.Import {
	pos := tok.pos
	var ident *ast.Identifier
	if tok.typ == tokenIdentifier {
		ident = ast.NewIdentifier(tok.pos, string(tok.txt))
		tok = p.next()
	} else if tok.typ == tokenPeriod {
		ident = ast.NewIdentifier(tok.pos, ".")
		tok = p.next()
	}
	if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
		panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting string", tok)})
	}
	var path = unquoteString(tok.txt)
	if !ValidPath(path) {
		panic(&SyntaxError{"", *tok.pos, fmt.Errorf("invalid import path: %q", path)})
	}
	if p.ctx == ast.ContextGo {
		if err := validPackagePath(path); err != nil {
			if err == ErrNotCanonicalImportPath {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("non-canonical import path %q (should be %q)", path, cleanPath(path))})
			}
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("invalid import path: %q", path)})
		}
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
	vp := variables[0].Pos()
	pos := &ast.Position{Line: vp.Line, Column: vp.Column, Start: vp.Start, End: tok.pos.End}
	var values []ast.Expression
	switch typ {
	case ast.AssignmentSimple, ast.AssignmentDeclaration:
		values, tok = p.parseExprList(token{}, canBeSwitchGuard, false, nextIsBlockOpen)
		if len(values) == 0 {
			return nil, tok
		}
		pos.End = values[len(values)-1].Pos().End
	default:
		if len(variables) > 1 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting := or = or comma", tok)})
		}
		if typ == ast.AssignmentIncrement || typ == ast.AssignmentDecrement {
			tok = p.next()
		} else {
			values = make([]ast.Expression, 1)
			values[0], tok = p.parseExpr(token{}, false, false, false)
		}
	}
	return ast.NewAssignment(pos, variables, typ, values), tok
}

// addChild adds a child to the current parent node.
func (p *parsing) addChild(child ast.Node) {
	switch n := p.ancestors[len(p.ancestors)-1].(type) {
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
		p.ancestors = p.ancestors[:len(p.ancestors)-1]
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
