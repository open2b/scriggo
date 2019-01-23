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

	"open2b/template/ast"
)

var (
	// ErrInvalidPath is returned from the Parse method and a Reader when the
	// path argument is not valid.
	ErrInvalidPath = errors.New("template/parser: invalid path")

	// ErrNotExist is returned from the Parse method and a Reader when the
	// path does not exist.
	ErrNotExist = errors.New("template/parser: path does not exist")

	// ErrReadTooLarge is returned from a DirLimitedReader when a limit is
	// exceeded.
	ErrReadTooLarge = errors.New("template/parser: read too large")
)

// Error records a parsing error with the path and the position where the
// error occurred.
type Error struct {
	Path string
	Pos  ast.Position
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s:%s: %s", e.Path, e.Pos, e.Err)
}

// CycleError implements an error indicating the presence of a cycle.
type CycleError string

func (e CycleError) Error() string {
	return fmt.Sprintf("cycle not allowed\n%s", string(e))
}

// next returns the next token from the lexer lex. It panics on error.
func next(lex *lexer) token {
	tok, ok := <-lex.tokens
	if !ok {
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

// ParseSource parses src in the context ctx and returns a tree. Nodes
// Extends, Import and Include will not be expanded (the field Tree will be
// nil). To get an expanded tree call the method Parse of a Parser instead.
func ParseSource(src []byte, ctx ast.Context) (tree *ast.Tree, err error) {

	switch ctx {
	case ast.ContextText, ast.ContextHTML, ast.ContextCSS, ast.ContextScript:
	default:
		return nil, errors.New("template/parser: invalid context. Valid contexts are Text, HTML, CSS and Script")
	}

	// Create the lexer.
	var lex = newLexer(src, ctx)

	// Tree result of the parsing.
	tree = ast.NewTree("", nil, ctx)

	// Ancestors from the root up to the parent.
	var ancestors = []ast.Node{tree}

	// Indicates if it has been extended.
	var isExtended = false

	// Indicates if it is in a macro.
	var isInMacro = false

	// Current line.
	var line = 0

	// First Text node of the current line.
	var firstText *ast.Text

	// Indicates if there is a token in current line for which it is possible
	// to cut the leading and trailing spaces.
	var cutSpacesToken bool

	// Number of non-text tokens in current line.
	var tokensInLine = 0

	// Index of the last byte.
	var end = len(src) - 1

	// Position of the last fallthrough token, used for error messages.
	var lastFallthroughTokenPos ast.Position

	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(*Error); ok {
				tree = nil
				err = e
			} else {
				panic(r)
			}
		}
	}()

	// Reads the tokens.
	for tok := range lex.tokens {

		var text *ast.Text
		if tok.typ == tokenText {
			text = ast.NewText(tok.pos, tok.txt, ast.Cut{})
		}

		if line < tok.lin || tok.pos.End == end {
			if cutSpacesToken && tokensInLine == 1 {
				cutSpaces(firstText, text)
			}
			line = tok.lin
			firstText = text
			cutSpacesToken = false
			tokensInLine = 0
		}

		// Parent is always the last ancestor.
		parent := ancestors[len(ancestors)-1]

		switch tok.typ {

		// EOF
		case tokenEOF:
			if len(ancestors) > 1 {
				return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected EOF, expecting {%% end %%}")}
			}

		// Text
		case tokenText:
			if s, ok := parent.(*ast.Switch); ok {
				if len(s.Cases) == 0 {
					// TODO (Gianluca): this "if" should be moved before the
					// switch that precedes it.
					if containsOnlySpaces(text.Text) {
						// TODO (Gianluca): instead of discarding whitespaces,
						// add them to the switch node field called
						// "LeadingText" in order to preserve the original
						// source.

						// If all characters are whitespaces, there's no need to
						// show an error: just discard the text.
						continue
					}
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected text, expecting case of default or {%% end %%}")}
				}
				lastCase := s.Cases[len(s.Cases)-1]
				if lastCase.Fallthrough {
					if containsOnlySpaces(text.Text) {
						continue
					}
					return nil, &Error{"", lastFallthroughTokenPos, fmt.Errorf("fallthrough statement out of place")}
				}
			}
			addChild(parent, text)

		// StartURL
		case tokenStartURL:
			node := ast.NewURL(tok.pos, tok.tag, tok.att, nil)
			addChild(parent, node)
			ancestors = append(ancestors, node)

		// EndURL
		case tokenEndURL:
			pos := ancestors[len(ancestors)-1].Pos()
			pos.End = tok.pos.End - 1
			ancestors = ancestors[:len(ancestors)-1]

		// {%
		case tokenStartStatement:

			tokensInLine++

			var node ast.Node

			var pos = tok.pos

			var expr ast.Expression

			var ok bool
			tok = next(lex)

			l := -1
			switch s := parent.(type) {
			case *ast.Switch:
				l = len(s.Cases)
			case *ast.TypeSwitch:
				l = len(s.Cases)
			}
			if l == 0 {
				if tok.typ != tokenCase && tok.typ != tokenDefault && tok.typ != tokenEnd {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting case of default or {%% end %%}", tok.String())}
				}
			}

			switch tok.typ {

			// for
			case tokenFor:
				var node ast.Node
				var init *ast.Assignment
				var assignmentType ast.AssignmentType
				variables, tok := parseExprList(token{}, lex, true, false)
				switch tok.typ {
				case tokenIn:
					// {% for ident in expr %}
					if len(variables) == 0 {
						return nil, &Error{"", *(variables[1].Pos()), fmt.Errorf("unexpected in, expected expression")}
					}
					if len(variables) > 1 {
						return nil, &Error{"", *(variables[1].Pos()), fmt.Errorf("expected only one expression")}
					}
					ident, ok := variables[0].(*ast.Identifier)
					if !ok {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected in, expected assignment")}
					}
					if ident.Name == "_" {
						return nil, &Error{"", *(ident.Pos()), fmt.Errorf("cannot use _ as value")}
					}
					p := ident.Pos()
					blank := ast.NewIdentifier(&ast.Position{p.Line, p.Column, p.Start, p.Start}, "_")
					// Parses the slice expression
					expr, tok = parseExpr(token{}, lex, false, false)
					if expr == nil {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)}
					}
					assignment := ast.NewAssignment(&ast.Position{p.Line, p.Column, p.Start, expr.Pos().End},
						[]ast.Expression{blank, ident}, ast.AssignmentDeclaration, []ast.Expression{expr})
					pos.End = tok.pos.End
					node = ast.NewForRange(pos, assignment, nil)
				case tokenEndStatement:
					// {% for %}
					if len(variables) > 1 {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)}
					}
					var condition ast.Expression
					if len(variables) == 1 {
						condition = variables[0]
					}
					pos.End = tok.pos.End
					node = ast.NewFor(pos, nil, condition, nil, nil)
				case tokenRange:
					// {% for range expr %}
					if len(variables) > 0 {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected range, expecting := or = or comma")}
					}
					p := tok.pos
					expr, tok = parseExpr(token{}, lex, false, false)
					if expr == nil {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)}
					}
					p.End = expr.Pos().End
					assignment := ast.NewAssignment(p, nil, ast.AssignmentSimple, []ast.Expression{expr})
					pos.End = tok.pos.End
					node = ast.NewForRange(pos, assignment, nil)
				case tokenSimpleAssignment, tokenDeclaration, tokenIncrement, tokenDecrement:
					if len(variables) == 0 {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)}
					}
					if tok.typ == tokenDeclaration {
						assignmentType = ast.AssignmentDeclaration
					}
					init, tok = parseAssignment(variables, tok, lex, false)
					if init == nil && tok.typ != tokenRange {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)}
					}
					fallthrough
				case tokenSemicolon:
					if tok.typ == tokenRange {
						// {% for index[, ident] = range expr %}
						// {% for index[, ident] := range expr %}
						if len(variables) > 2 {
							return nil, &Error{"", *tok.pos, fmt.Errorf("too many variables in range")}
						}
						expr, tok = parseExpr(token{}, lex, false, false)
						if expr == nil {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)}
						}
						p := variables[0].Pos()
						assignment := ast.NewAssignment(&ast.Position{p.Line, p.Column, p.Start, expr.Pos().End},
							variables, assignmentType, []ast.Expression{expr})
						pos.End = tok.pos.End
						node = ast.NewForRange(pos, assignment, nil)
					} else {
						// {% for [init]; [condition]; [post] %}
						// Parses the condition expression.
						var condition ast.Expression
						condition, tok = parseExpr(token{}, lex, false, false)
						if tok.typ != tokenSemicolon {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expected semicolon", tok)}
						}
						// Parses the post iteration statement.
						var post *ast.Assignment
						variables, tok = parseExprList(token{}, lex, true, false)
						if len(variables) > 0 {
							pos := tok.pos
							post, tok = parseAssignment(variables, tok, lex, false)
							if post == nil {
								return nil, &Error{"", *tok.pos, fmt.Errorf("expecting expression")}
							}
							if post.Type == ast.AssignmentDeclaration {
								return nil, &Error{"", *pos, fmt.Errorf("cannot declare in post statement of for loop")}
							}
						}
						pos.End = tok.pos.End
						node = ast.NewFor(pos, init, condition, post, nil)
					}
				}
				if node == nil || tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression or %%}", tok)}
				}
				addChild(parent, node)
				ancestors = append(ancestors, node)
				cutSpacesToken = true

			// break
			case tokenBreak:
				var breakable bool
				for i := len(ancestors) - 1; !breakable && i > 0; i-- {
					switch ancestors[i].(type) {
					case *ast.For, *ast.ForRange, *ast.Switch:
						breakable = true
						break
					}
				}
				if !breakable {
					return nil, &Error{"", *tok.pos, fmt.Errorf("break is not in a loop or switch")}
				}
				tok = next(lex)
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewBreak(pos)
				addChild(parent, node)
				cutSpacesToken = true

			// continue
			case tokenContinue:
				var loop bool
				for i := len(ancestors) - 1; !loop && i > 0; i-- {
					switch ancestors[i].(type) {
					case *ast.For, *ast.ForRange:
						loop = true
						break
					}
				}
				if !loop {
					return nil, &Error{"", *tok.pos, fmt.Errorf("continue is not in a loop")}
				}
				tok = next(lex)
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewContinue(pos)
				addChild(parent, node)
				cutSpacesToken = true

			// switch
			case tokenSwitch:
				node, err = parseSwitch(lex, pos)
				if err != nil {
					return nil, err
				}
				addChild(parent, node)
				ancestors = append(ancestors, node)
				cutSpacesToken = true

			// case:
			case tokenCase:
				// TODO (Gianluca): check all if expressions contained in this
				// case have not been previously declarated in another "case".
				// In such condition return an error as:
				//
				// prog.go:12:2: duplicate case int in type switch previous case
				// at prog.go:11:2
				switch parent.(type) {
				case *ast.Switch, *ast.TypeSwitch:
				default:
					// TODO (Gianluca): should be "unexpected case, expecting ...".
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected case")}
				}
				var node *ast.Case
				expressions, tok := parseExprList(token{}, lex, false, false)
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				if _, ok := parent.(*ast.TypeSwitch); ok {
					for _, expr := range expressions {
						switch n := expr.(type) {
						case *ast.Identifier:
							switch n.Name {
							case "true", "false":
								return nil, &Error{"", *tok.pos, fmt.Errorf("%s (type bool) is not a type", n.Name)}
							}
						case *ast.Int:
							return nil, &Error{"", *tok.pos, fmt.Errorf("%s (type int) is not a type", n)}
						case *ast.Number:
							return nil, &Error{"", *tok.pos, fmt.Errorf("%s (type number) is not a type", n)}
						case *ast.String:
							return nil, &Error{"", *tok.pos, fmt.Errorf("%s (type string) is not a type", n)}
						default:
							return nil, &Error{"", *tok.pos, fmt.Errorf("%s is not a type", expr)}
						}
					}
				}
				node = ast.NewCase(pos, expressions, nil, false)
				addChild(parent, node)

			// default:
			case tokenDefault:
				switch s := parent.(type) {
				case *ast.Switch:
					for _, c := range s.Cases {
						if c.Expressions == nil {
							return nil, &Error{"", *tok.pos, fmt.Errorf("multiple defaults in switch (first at %s)", c.Pos())}
						}
					}
				case *ast.TypeSwitch:
					for _, c := range s.Cases {
						if c.Expressions == nil {
							return nil, &Error{"", *tok.pos, fmt.Errorf("multiple defaults in switch (first at %s)", c.Pos())}
						}
					}
				default:
					// TODO (Gianluca): should be "unexpected case, expecting ...".
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected case")}
				}
				tok = next(lex)
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node := ast.NewCase(pos, nil, nil, false)
				addChild(parent, node)
				cutSpacesToken = true

				// fallthrough
			case tokenFallthrough:
				// TODO (Gianluca): fallthrough must be implemented as an ast
				// node.
				lastFallthroughTokenPos = *tok.pos
				tok = next(lex)
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				switch s := parent.(type) {
				case *ast.Switch:
					lastCase := s.Cases[len(s.Cases)-1]
					if lastCase.Fallthrough {
						return nil, &Error{"", *tok.pos, fmt.Errorf("fallthrough statement out of place")}
					}
					lastCase.Fallthrough = true
				case *ast.TypeSwitch:
					return nil, &Error{"", *tok.pos, fmt.Errorf("cannot fallthrough in type switch")}
				default:
					return nil, &Error{"", *tok.pos, fmt.Errorf("fallthrough statement out of place")}
				}
				pos.End = tok.pos.End
				cutSpacesToken = true

			// if
			case tokenIf:
				expressions, tok := parseExprList(token{}, lex, true, false)
				if len(expressions) == 0 {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)}
				}
				var assignment *ast.Assignment
				if len(expressions) > 1 || tok.typ == tokenSimpleAssignment || tok.typ == tokenDeclaration {
					assignment, tok = parseAssignment(expressions, tok, lex, false)
					if assignment == nil {
						return nil, &Error{"", *tok.pos, fmt.Errorf("expecting expression")}
					}
					if tok.typ != tokenSemicolon {
						return nil, &Error{"", *tok.pos, fmt.Errorf("%s used as value", assignment)}
					}
					expr, tok = parseExpr(token{}, lex, false, false)
					if expr == nil {
						return nil, &Error{"", *tok.pos, fmt.Errorf("missing condition in if statement")}
					}
				} else {
					expr = expressions[0]
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewIf(pos, assignment, expr, nil, nil)
				addChild(parent, node)
				ancestors = append(ancestors, node)
				cutSpacesToken = true

			// else
			case tokenElse:
				var p *ast.If
				if p, ok = parent.(*ast.If); ok && p.Else == nil {
					p.Else = []ast.Node{}
				} else {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end, for, if or show", tok)}
				}
				tok = next(lex)
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				cutSpacesToken = true

			// include
			case tokenInclude:
				if isExtended && !isInMacro {
					return nil, &Error{"", *tok.pos, fmt.Errorf("include statement outside macro")}
				}
				if tok.ctx == ast.ContextAttribute || tok.ctx == ast.ContextUnquotedAttribute {
					return nil, &Error{"", *tok.pos, fmt.Errorf("include statement inside an attribute value")}
				}
				// path
				tok = next(lex)
				if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting string", tok)}
				}
				var path = unquoteString(tok.txt)
				if !validPath(path) {
					return nil, fmt.Errorf("invalid path %q at %s", path, tok.pos)
				}
				tok = next(lex)
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ( or %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewInclude(pos, path, tok.ctx)
				addChild(parent, node)
				cutSpacesToken = true

			// show
			case tokenShow:
				if isExtended && !isInMacro {
					return nil, &Error{"", *tok.pos, fmt.Errorf("show statement outside macro")}
				}
				if tok.ctx == ast.ContextAttribute || tok.ctx == ast.ContextUnquotedAttribute {
					return nil, &Error{"", *tok.pos, fmt.Errorf("show statement inside an attribute value")}
				}
				tok = next(lex)
				if tok.typ != tokenIdentifier {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
				}
				if len(tok.txt) == 1 && tok.txt[0] == '_' {
					return nil, &Error{"", *tok.pos, fmt.Errorf("cannot use _ as value")}
				}
				macro := ast.NewIdentifier(tok.pos, string(tok.txt))
				tok = next(lex)
				// import
				var impor *ast.Identifier
				if tok.typ == tokenPeriod {
					tok = next(lex)
					if tok.typ != tokenIdentifier {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
					}
					if len(tok.txt) == 1 && tok.txt[0] == '_' {
						return nil, &Error{"", *tok.pos, fmt.Errorf("cannot use _ as value")}
					}
					impor = macro
					macro = ast.NewIdentifier(tok.pos, string(tok.txt))
					if fc, _ := utf8.DecodeRuneInString(macro.Name); !unicode.Is(unicode.Lu, fc) {
						return nil, &Error{"", *tok.pos, fmt.Errorf("cannot refer to unexported macro %s", macro.Name)}
					}
					tok = next(lex)
				}
				var arguments []ast.Expression
				if tok.typ == tokenLeftParenthesis {
					// arguments
					arguments = []ast.Expression{}
					for {
						expr, tok = parseExpr(token{}, lex, false, false)
						if expr == nil {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)}
						}
						arguments = append(arguments, expr)
						if tok.typ == tokenRightParenthesis {
							break
						}
						if tok.typ != tokenComma {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting , or )", tok)}
						}
					}
					tok = next(lex)
					if tok.typ != tokenEndStatement {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
					}
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ( or %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewShowMacro(pos, impor, macro, arguments, tok.ctx)
				addChild(parent, node)
				cutSpacesToken = true

			// extends
			case tokenExtends:
				if isExtended {
					return nil, &Error{"", *tok.pos, fmt.Errorf("extends already exists")}
				}
				if len(tree.Nodes) > 0 {
					if _, ok = tree.Nodes[0].(*ast.Text); !ok || len(tree.Nodes) > 1 {
						return nil, &Error{"", *tok.pos, fmt.Errorf("extends can only be the first statement")}
					}
				}
				if tok.ctx != ctx {
					switch tok.ctx {
					case ast.ContextAttribute, ast.ContextUnquotedAttribute:
						return nil, &Error{"", *tok.pos, fmt.Errorf("extends inside an attribute value")}
					case ast.ContextScript:
						return nil, &Error{"", *tok.pos, fmt.Errorf("extends inside a script tag")}
					case ast.ContextCSS:
						return nil, &Error{"", *tok.pos, fmt.Errorf("extends inside a style tag")}
					}
				}
				tok = next(lex)
				if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting string", tok)}
				}
				var path = unquoteString(tok.txt)
				if !validPath(path) {
					return nil, &Error{"", *tok.pos, fmt.Errorf("invalid extends path %q", path)}
				}
				tok = next(lex)
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewExtends(pos, path, tok.ctx)
				addChild(parent, node)
				isExtended = true

			// import
			case tokenImport:
				if tok.ctx != ctx {
					switch tok.ctx {
					case ast.ContextAttribute, ast.ContextUnquotedAttribute:
						return nil, &Error{"", *tok.pos, fmt.Errorf("import inside an attribute value")}
					case ast.ContextScript:
						return nil, &Error{"", *tok.pos, fmt.Errorf("import inside a script tag")}
					case ast.ContextCSS:
						return nil, &Error{"", *tok.pos, fmt.Errorf("import inside a style tag")}
					}
				}
				for i := len(ancestors) - 1; i > 0; i-- {
					switch ancestors[i].(type) {
					case ast.For, ast.ForRange:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end for", tok)}
					case *ast.If:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end if", tok)}
					case *ast.Macro:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end macro", tok)}
					}
				}
				tok = next(lex)
				var ident *ast.Identifier
				if tok.typ == tokenIdentifier {
					ident = ast.NewIdentifier(tok.pos, string(tok.txt))
					tok = next(lex)
				}
				if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
					return nil, fmt.Errorf("unexpected %s, expecting string at %s", tok, tok.pos)
				}
				var path = unquoteString(tok.txt)
				if !validPath(path) {
					return nil, fmt.Errorf("invalid import path %q at %s", path, tok.pos)
				}
				tok = next(lex)
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewImport(pos, ident, path, tok.ctx)
				addChild(parent, node)
				cutSpacesToken = true

			// macro
			case tokenMacro:
				if tok.ctx == ast.ContextAttribute || tok.ctx == ast.ContextUnquotedAttribute {
					return nil, &Error{"", *tok.pos, fmt.Errorf("macro inside an attribute value")}
				}
				for i := len(ancestors) - 1; i > 0; i-- {
					switch ancestors[i].(type) {
					case ast.For:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end for", tok)}
					case *ast.If:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end if", tok)}
					case *ast.Macro:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end macro", tok)}
					}
				}
				// ident
				tok = next(lex)
				if tok.typ != tokenIdentifier {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
				}
				if len(tok.txt) == 1 && tok.txt[0] == '_' {
					return nil, &Error{"", *tok.pos, fmt.Errorf("cannot use _ as value")}
				}
				ident := ast.NewIdentifier(tok.pos, string(tok.txt))
				tok = next(lex)
				var parameters []*ast.Identifier
				var ellipsesPos *ast.Position
				if tok.typ == tokenLeftParenthesis {
					// parameters
					parameters = []*ast.Identifier{}
					for {
						tok = next(lex)
						if tok.typ != tokenIdentifier {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
						}
						if ellipsesPos != nil {
							return nil, &Error{"", *ellipsesPos, fmt.Errorf("cannot use ... with non-final parameter")}
						}
						parameters = append(parameters, ast.NewIdentifier(tok.pos, string(tok.txt)))
						tok = next(lex)
						if tok.typ == tokenEllipses {
							ellipsesPos = tok.pos
							tok = next(lex)
						}
						if tok.typ == tokenRightParenthesis {
							break
						}
						if tok.typ != tokenComma {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting , or )", tok)}
						}
					}
					tok = next(lex)
					if tok.typ != tokenEndStatement {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
					}
				} else if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ( or %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewMacro(pos, ident, parameters, nil, ellipsesPos != nil, tok.ctx)
				addChild(parent, node)
				ancestors = append(ancestors, node)
				cutSpacesToken = true
				isInMacro = true

			// end
			case tokenEnd:
				if _, ok = parent.(*ast.URL); ok || len(ancestors) == 1 {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s", tok)}
				}
				tok = next(lex)
				if tok.typ != tokenEndStatement {
					tokparent := tok
					tok = next(lex)
					if tok.typ != tokenEndStatement {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
					}
					switch parent.(type) {
					case ast.For:
						if tokparent.typ != tokenFor {
							return nil, &Error{"", *tokparent.pos, fmt.Errorf("unexpected %s, expecting for or %%}", tok)}
						}
					case *ast.If:
						if tokparent.typ != tokenIf {
							return nil, &Error{"", *tokparent.pos, fmt.Errorf("unexpected %s, expecting if or %%}", tok)}
						}
					case *ast.Macro:
						if tokparent.typ != tokenMacro {
							return nil, &Error{"", *tokparent.pos, fmt.Errorf("unexpected %s, expecting macro or %%}", tok)}
						}
					}
				}
				parent.Pos().End = tok.pos.End
				if _, ok := parent.(*ast.Macro); ok {
					isInMacro = false
				}
				ancestors = ancestors[:len(ancestors)-1]
				cutSpacesToken = true

			// expression or assignment
			default:
				expressions, tok := parseExprList(tok, lex, true, false)
				if len(expressions) == 0 {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting for, if, show, extends, include, macro or end", tok)}
				}
				if len(expressions) > 1 || isAssignmentToken(tok) {
					// Parses assignment.
					assignment, tok := parseAssignment(expressions, tok, lex, false)
					if assignment == nil {
						return nil, &Error{"", *tok.pos, fmt.Errorf("expecting expression")}
					}
					if tok.typ != tokenEndStatement {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
					}
					pos.End = tok.pos.End
					assignment.Position = pos
					addChild(parent, assignment)
					cutSpacesToken = true
				} else {
					// Parses expression.
					expr := expressions[0]
					if ident, ok := expr.(*ast.Identifier); ok && ident.Name == "_" {
						return nil, &Error{"", *expr.Pos(), fmt.Errorf("cannot use _ as value")}
					}
					if tok.typ != tokenEndStatement {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
					}
					addChild(parent, expr)
					cutSpacesToken = true
				}
			}

		// {{ }}
		case tokenStartValue:
			if isExtended && !isInMacro {
				return nil, &Error{"", *tok.pos, fmt.Errorf("value statement outside macro")}
			}
			tokensInLine++
			expr, tok2 := parseExpr(token{}, lex, false, false)
			if expr == nil {
				return nil, &Error{"", *tok2.pos, fmt.Errorf("expecting expression")}
			}
			if tok2.typ != tokenEndValue {
				return nil, &Error{"", *tok2.pos, fmt.Errorf("unexpected %s, expecting }}", tok2)}
			}
			tok.pos.End = tok2.pos.End
			var node = ast.NewValue(tok.pos, expr, tok.ctx)
			addChild(parent, node)

		// comment
		case tokenComment:
			tokensInLine++
			var node = ast.NewComment(tok.pos, string(tok.txt[2:len(tok.txt)-2]))
			addChild(parent, node)
			cutSpacesToken = true

		default:
			return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s", tok)}

		}

	}

	if lex.err != nil {
		return nil, lex.err
	}

	return tree, nil
}

// parseAssignment parses an assignment and returns an assignment or, if there
// is no expression, returns nil. tok can be the assignment, declaration,
// increment or decrement token. It panics on error.
func parseAssignment(variables []ast.Expression, tok token, lex *lexer, canBeSwitchGuard bool) (*ast.Assignment, token) {
	var typ, ok = assignmentType(tok)
	if !ok {
		panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting := or = or comma", tok)})
	}
	for _, v := range variables {
		switch v.(type) {
		case *ast.Identifier:
			continue
		case *ast.Selector, *ast.Index:
			if typ != ast.AssignmentDeclaration {
				continue
			}
		}
		panic(&Error{"", *(v.Pos()), fmt.Errorf("%s used as value", v)})
	}
	assignToken := tok
	p := variables[0].Pos()
	pos := &ast.Position{Line: p.Line, Column: p.Column, Start: p.Start, End: tok.pos.End}
	var values []ast.Expression
	switch typ {
	case ast.AssignmentSimple, ast.AssignmentDeclaration:
		values, tok = parseExprList(token{}, lex, false, canBeSwitchGuard)
		if len(values) == 0 {
			return nil, tok
		}
		if len(values) == 1 {
			var mismatch bool
			switch values[0].(type) {
			case *ast.Call:
			case *ast.Index, *ast.Selector, *ast.TypeAssertion:
				mismatch = len(variables) > 2
			default:
				mismatch = len(variables) > 1
			}
			if mismatch {
				panic(&Error{"", *assignToken.pos, fmt.Errorf("assignment mismatch: %d variables but 1 values", len(variables))})
			}
		} else if len(variables) != len(values) {
			panic(&Error{"", *assignToken.pos, fmt.Errorf("assignment mismatch: %d variables but %d values", len(variables), len(values))})
		}
		pos.End = values[len(values)-1].Pos().End
	default:
		if len(variables) > 1 {
			panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting := or = or comma", tok)})
		}
		if ident, ok := variables[0].(*ast.Identifier); ok && ident.Name == "_" {
			panic(&Error{"", *variables[0].Pos(), fmt.Errorf("cannot use _ as value")})
		}
		if typ == ast.AssignmentIncrement || typ == ast.AssignmentDecrement {
			tok = next(lex)
		} else {
			values = make([]ast.Expression, 1)
			values[0], tok = parseExpr(token{}, lex, false, false)
			if ident, ok := values[0].(*ast.Identifier); ok && ident.Name == "_" {
				panic(&Error{"", *values[0].Pos(), fmt.Errorf("cannot use _ as value")})
			}
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
	reader Reader
	trees  *cache
}

// New returns a new Parser that reads the trees from the reader r.
func New(r Reader) *Parser {
	return &Parser{
		reader: r,
		trees:  &cache{},
	}
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

	pp := &parsing{p.reader, p.trees, []string{}}

	tree, err := pp.parsePath(path, ctx)
	if err != nil {
		if err2, ok := err.(*Error); ok && err2.Path == "" {
			err2.Path = path
		} else if err2, ok := err.(CycleError); ok {
			err = CycleError(path + "\n\t" + string(err2))
		}
		return nil, err
	}

	return tree, nil
}

// parsing is a parsing state.
type parsing struct {
	reader Reader
	trees  *cache
	paths  []string
}

// abs returns path as absolute.
func (pp *parsing) abs(path string) (string, error) {
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
func (pp *parsing) parsePath(path string, ctx ast.Context) (*ast.Tree, error) {

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

	tree, err := pp.reader.Read(path, ctx)
	if err != nil {
		return nil, err
	}
	tree.Path = path

	// Expands the nodes.
	pp.paths = append(pp.paths, path)
	err = pp.expand(tree.Nodes, ctx)
	if err != nil {
		if e, ok := err.(*Error); ok && e.Path == "" {
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
func (pp *parsing) expand(nodes []ast.Node, ctx ast.Context) error {

	for _, node := range nodes {

		switch n := node.(type) {

		case *ast.If:

			err := pp.expand(n.Then, ctx)
			if err != nil {
				return err
			}
			err = pp.expand(n.Else, ctx)
			if err != nil {
				return err
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
				return &Error{"", *(n.Pos()), fmt.Errorf("extended, imported and included paths can not have extends")}
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
					err = &Error{"", *(n.Pos()), fmt.Errorf("extends path %q does not exist", absPath)}
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
			n.Tree, err = pp.parsePath(absPath, n.Context)
			if err != nil {
				if err == ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if err == ErrNotExist {
					err = &Error{"", *(n.Pos()), fmt.Errorf("import path %q does not exist", absPath)}
				} else if err2, ok := err.(CycleError); ok {
					err = CycleError("imports " + string(err2))
				}
				return err
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
					err = &Error{"", *(n.Pos()), fmt.Errorf("included path %q does not exist", absPath)}
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
	case *ast.URL:
		n.Value = append(n.Value, node)
	case *ast.Macro:
		n.Body = append(n.Body, node)
	case *ast.For:
		n.Body = append(n.Body, node)
	case *ast.ForRange:
		n.Body = append(n.Body, node)
	case *ast.If:
		if n.Else == nil {
			n.Then = append(n.Then, node)
		} else {
			n.Else = append(n.Else, node)
		}
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
		} else {
			lastCase := n.Cases[len(n.Cases)-1]
			lastCase.Body = append(lastCase.Body, node)
		}
	default:
		panic("template/parser: unexpected parent node")
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
