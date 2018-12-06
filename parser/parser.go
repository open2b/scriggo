// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
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
	// ErrInvalidPath can be returned from the Parse method and a Reader when
	// the path argument is not valid.
	ErrInvalidPath = errors.New("template/parser: invalid path")

	// ErrNotExist can be returned from the Parse method and a Reader when the
	// path does not exist.
	ErrNotExist = errors.New("template/parser: path does not exist")

	// ErrReadTooLarge can be returned from a DirLimitedReader when a limit is
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

// ParseSource parses src in the context ctx and returns a tree. Nodes Extend,
// Import and ShowPath will not be expanded (the field Tree will be nil). To
// get an expanded tree call the method Parse of a Parser instead.
func ParseSource(src []byte, ctx ast.Context) (*ast.Tree, error) {

	switch ctx {
	case ast.ContextText, ast.ContextHTML, ast.ContextCSS, ast.ContextScript:
	default:
		return nil, errors.New("template/parser: invalid context. Valid contexts are Text, HTML, CSS and Script")
	}

	// Create the lexer.
	var lex = newLexer(src, ctx)

	// Tree result of the parsing.
	var tree = ast.NewTree("", nil, ctx)

	// Ancestors from the root up to the parent.
	var ancestors = []ast.Node{tree}

	// Indicates if it has been extended.
	var isExtended = false

	// Indicates if he is in a macro.
	var isInMacro = false

	// Current line.
	var line = 0

	// First Text node of the current line.
	var firstText *ast.Text

	// Indicates if there is a token in the current line for which it would
	// be possible to cut the initial and final spaces.
	var cutSpacesToken bool

	// Number of tokens, not Text, in the current line.
	var tokensInLine = 0

	// Index of the last byte.
	var end = len(src) - 1

	// Reads all the tokens.
	for tok := range lex.tokens {

		var text *ast.Text
		if tok.typ == tokenText {
			text = ast.NewText(tok.pos, tok.txt)
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

		// The parent is always the last ancestor.
		parent := ancestors[len(ancestors)-1]

		switch tok.typ {

		// EOF
		case tokenEOF:
			if len(ancestors) > 1 {
				return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected EOF, expecting {%% end %%}")}
			}

		// Text
		case tokenText:
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

			var err error
			var expr ast.Expression

			var ok bool
			tok, ok = <-lex.tokens
			if !ok {
				return nil, lex.err
			}

			switch tok.typ {

			// var or identifier
			case tokenVar, tokenIdentifier:
				var isVar = tok.typ == tokenVar
				if isVar {
					// identifier
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					if tok.typ != tokenIdentifier {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
					}
				}
				var ident = ast.NewIdentifier(tok.pos, string(tok.txt))
				// assignment
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenAssignment {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting assignment", tok)}
				}
				// expression
				expr, tok, err = parseExpr(lex)
				if err != nil {
					return nil, err
				}
				if expr == nil {
					return nil, &Error{"", *tok.pos, fmt.Errorf("expecting expression")}
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				if isVar {
					node = ast.NewVar(pos, ident, expr)
				} else {
					node = ast.NewAssignment(pos, ident, expr)
				}
				addChild(parent, node)
				cutSpacesToken = true

			// for
			case tokenFor:
				var index, ident *ast.Identifier
				// index
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenIdentifier {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
				}
				index = ast.NewIdentifier(tok.pos, string(tok.txt))
				// "," or "in"
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				var comma token
				switch tok.typ {
				case tokenComma:
					// Syntax: for index, ident in expr
					comma = tok
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					if tok.typ != tokenIdentifier {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
					}
					ident = ast.NewIdentifier(tok.pos, string(tok.txt))
					// "in"
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					if tok.typ != tokenIn {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting \"in\"", tok)}
					}
				case tokenIn:
					// Syntax: for ident in expr
					// Syntax: for index in expr..expr
				default:
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting comma or \"in\"", tok)}
				}
				expr, tok, err = parseExpr(lex)
				if err != nil {
					return nil, err
				}
				if expr == nil {
					return nil, &Error{"", *tok.pos, fmt.Errorf("expecting expression")}
				}
				var expr2 ast.Expression
				if tok.typ == tokenRange {
					// Syntax: for index in expr..expr
					if ident != nil {
						return nil, &Error{"", *comma.pos, fmt.Errorf("unexpected %s, expecting \"in\"", comma)}
					}
					expr2, tok, err = parseExpr(lex)
					if err != nil {
						return nil, err
					}
					if expr == nil {
						return nil, &Error{"", *tok.pos, fmt.Errorf("expecting expression")}
					}
				} else if ident == nil {
					// Syntax: for ident in expr
					ident = index
					index = nil
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewFor(pos, index, ident, expr, expr2, nil)
				addChild(parent, node)
				ancestors = append(ancestors, node)
				cutSpacesToken = true

			// break
			case tokenBreak:
				var loop bool
				for i := len(ancestors) - 1; i > 0; i-- {
					if _, loop = ancestors[i].(*ast.For); loop {
						break
					}
				}
				if !loop {
					return nil, &Error{"", *tok.pos, fmt.Errorf("break is not in a loop")}
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
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
				for i := len(ancestors) - 1; i > 0; i-- {
					if _, loop = ancestors[i].(*ast.For); loop {
						break
					}
				}
				if !loop {
					return nil, &Error{"", *tok.pos, fmt.Errorf("continue is not in a loop")}
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewContinue(pos)
				addChild(parent, node)
				cutSpacesToken = true

			// if
			case tokenIf:
				expr, tok, err = parseExpr(lex)
				if err != nil {
					return nil, err
				}
				if expr == nil {
					return nil, &Error{"", *tok.pos, fmt.Errorf("expecting expression")}
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewIf(pos, expr, nil, nil)
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
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				cutSpacesToken = true

			// show
			case tokenShow:
				if isExtended && !isInMacro {
					return nil, &Error{"", *tok.pos, fmt.Errorf("show statement outside macro")}
				}
				if tok.ctx == ast.ContextAttribute || tok.ctx == ast.ContextUnquotedAttribute {
					return nil, &Error{"", *tok.pos, fmt.Errorf("show statement inside an attribute value")}
				}
				// macro or path
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ == tokenIdentifier {
					// show <macro>
					macro := ast.NewIdentifier(tok.pos, string(tok.txt))
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					// import
					var impor *ast.Identifier
					if tok.typ == tokenPeriod {
						tok, ok = <-lex.tokens
						if !ok {
							return nil, lex.err
						}
						if tok.typ != tokenIdentifier {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
						}
						impor = macro
						macro = ast.NewIdentifier(tok.pos, string(tok.txt))
						if fc, _ := utf8.DecodeRuneInString(macro.Name); !unicode.Is(unicode.Lu, fc) {
							return nil, &Error{"", *tok.pos, fmt.Errorf("cannot refer to unexported macro %s", macro.Name)}
						}
						tok, ok = <-lex.tokens
						if !ok {
							return nil, lex.err
						}
					}
					var arguments []ast.Expression
					if tok.typ == tokenLeftParenthesis {
						// arguments
						arguments = []ast.Expression{}
						for {
							expr, tok, err = parseExpr(lex)
							if err != nil {
								return nil, err
							}
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
						tok, ok = <-lex.tokens
						if !ok {
							return nil, lex.err
						}
						if tok.typ != tokenEndStatement {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
						}
					}
					if tok.typ != tokenEndStatement {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ( or %%}", tok)}
					}
					pos.End = tok.pos.End
					node = ast.NewShowMacro(pos, impor, macro, arguments, tok.ctx)
				} else if tok.typ == tokenInterpretedString || tok.typ == tokenRawString {
					// show <path>
					var path = unquoteString(tok.txt)
					if !validPath(path) {
						return nil, fmt.Errorf("invalid path %q at %s", path, tok.pos)
					}
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					if tok.typ != tokenEndStatement {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ( or %%}", tok)}
					}
					pos.End = tok.pos.End
					node = ast.NewShowPath(pos, path, tok.ctx)
				} else {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier o string", tok)}
				}
				addChild(parent, node)
				cutSpacesToken = true

			// extend
			case tokenExtend:
				if isExtended {
					return nil, &Error{"", *tok.pos, fmt.Errorf("extend already exists")}
				}
				if len(tree.Nodes) > 0 {
					if _, ok = tree.Nodes[0].(*ast.Text); !ok || len(tree.Nodes) > 1 {
						return nil, &Error{"", *tok.pos, fmt.Errorf("extend can only be the first statement")}
					}
				}
				if tok.ctx != ctx {
					switch tok.ctx {
					case ast.ContextAttribute, ast.ContextUnquotedAttribute:
						return nil, &Error{"", *tok.pos, fmt.Errorf("extend inside an attribute value")}
					case ast.ContextScript:
						return nil, &Error{"", *tok.pos, fmt.Errorf("extend inside a script tag")}
					case ast.ContextCSS:
						return nil, &Error{"", *tok.pos, fmt.Errorf("extend inside a style tag")}
					}
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting string", tok)}
				}
				var path = unquoteString(tok.txt)
				if !validPath(path) {
					return nil, &Error{"", *tok.pos, fmt.Errorf("invalid extend path %q", path)}
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewExtend(pos, path, tok.ctx)
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
					case *ast.For:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end for", tok)}
					case *ast.If:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end if", tok)}
					case *ast.Macro:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end macro", tok)}
					}
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				var ident *ast.Identifier
				if tok.typ == tokenIdentifier {
					ident = ast.NewIdentifier(tok.pos, string(tok.txt))
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
				}
				if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
					return nil, fmt.Errorf("unexpected %s, expecting string at %s", tok, tok.pos)
				}
				var path = unquoteString(tok.txt)
				if !validPath(path) {
					return nil, fmt.Errorf("invalid import path %q at %s", path, tok.pos)
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
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
					case *ast.For:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end for", tok)}
					case *ast.If:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end if", tok)}
					case *ast.Macro:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end macro", tok)}
					}
				}
				// ident
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenIdentifier {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
				}
				ident := ast.NewIdentifier(tok.pos, string(tok.txt))
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				var parameters []*ast.Identifier
				var ellipsesPos *ast.Position
				if tok.typ == tokenLeftParenthesis {
					// parameters
					parameters = []*ast.Identifier{}
					for {
						tok, ok = <-lex.tokens
						if !ok {
							return nil, lex.err
						}
						if tok.typ != tokenIdentifier {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
						}
						if ellipsesPos != nil {
							return nil, &Error{"", *ellipsesPos, fmt.Errorf("cannot use ... with non-final parameter")}
						}
						parameters = append(parameters, ast.NewIdentifier(tok.pos, string(tok.txt)))
						tok, ok = <-lex.tokens
						if !ok {
							return nil, lex.err
						}
						if tok.typ == tokenEllipses {
							ellipsesPos = tok.pos
							tok, ok = <-lex.tokens
							if !ok {
								return nil, lex.err
							}
						}
						if tok.typ == tokenRightParenthesis {
							break
						}
						if tok.typ != tokenComma {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting , or )", tok)}
						}
					}
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
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
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					tokparent := tok
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					if tok.typ != tokenEndStatement {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
					}
					switch parent.(type) {
					case *ast.For:
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

			default:
				return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting for, if, show, extend, macro or end", tok)}

			}

		// {{ }}
		case tokenStartValue:
			if isExtended && !isInMacro {
				return nil, &Error{"", *tok.pos, fmt.Errorf("value statement outside macro")}
			}
			tokensInLine++
			expr, tok2, err := parseExpr(lex)
			if err != nil {
				return nil, err
			}
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

// Parser implements a parser that reads the tree from a Reader and expands
// the nodes Extend, Import and ShowPath. The trees are cached so only one
// call per combination of path and context is made to the reader even if
// several goroutines parse the same paths at the same time.
//
// Returned trees can only be transformed if the parser is no longer used,
// because it would be the cached trees to be transformed and a data race can
// occur. In case, use astutil.Clone to create a clone of the tree and then
// transform the clone.
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

// Parse reads the source in path, with the reader, in the ctx context,
// expands the nodes Extend, Import and ShowPath and returns the expanded tree.
//
// Parse can be called concurrently by more goroutine.
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

// parsePath parses the file at path in context ctx.
// path must be absolute and cleared.
func (pp *parsing) parsePath(path string, ctx ast.Context) (*ast.Tree, error) {

	// checks if there is a cycle
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

// expand expands the nodes parsing the sub trees in context ctx.
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

			err := pp.expand(n.Nodes, ctx)
			if err != nil {
				return err
			}

		case *ast.Macro:
			err := pp.expand(n.Body, ctx)
			if err != nil {
				return err
			}

		case *ast.Extend:

			if len(pp.paths) > 1 {
				return &Error{"", *(n.Pos()), fmt.Errorf("extended, imported and showed files can not have extend")}
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
					err = &Error{"", *(n.Pos()), fmt.Errorf("extend path %q does not exist", absPath)}
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

		case *ast.ShowPath:

			absPath, err := pp.abs(n.Path)
			if err != nil {
				return err
			}
			n.Tree, err = pp.parsePath(absPath, n.Context)
			if err != nil {
				if err == ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if err == ErrNotExist {
					err = &Error{"", *(n.Pos()), fmt.Errorf("show path %q does not exist", absPath)}
				} else if err2, ok := err.(CycleError); ok {
					err = CycleError("shows " + string(err2))
				}
				return err
			}

		}

	}

	return nil
}

func addChild(parent ast.Node, node ast.Node) {
	switch n := parent.(type) {
	case *ast.Tree:
		n.Nodes = append(n.Nodes, node)
	case *ast.URL:
		n.Value = append(n.Value, node)
	case *ast.Macro:
		n.Body = append(n.Body, node)
	case *ast.For:
		n.Nodes = append(n.Nodes, node)
	case *ast.If:
		if n.Else == nil {
			n.Then = append(n.Then, node)
		} else {
			n.Else = append(n.Else, node)
		}
	default:
		panic("template/parser: unexpected parent node")
	}
}

// cutSpaces cuts the leading and trailing spaces from a line where first and
// last are respectively the initial and final Text node.
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
		first.Cut.Right = firstCut
	}
}
