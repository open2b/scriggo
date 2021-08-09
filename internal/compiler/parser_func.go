// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"github.com/open2b/scriggo/ast"
)

type funcKindToParse int

const (
	parseFuncType funcKindToParse = 1 << iota // func type
	parseFuncLit                              // func literal
	parseFuncDecl                             // func declaration
)

// parseFunc parses a function type, literal or declaration or a macro type.
// tok must be either the token "func" or "macro". If tok is "macro", kind is
// parseFuncType.
func (p *parsing) parseFunc(tok token, kind funcKindToParse) (ast.Node, token) {
	isMacro := tok.typ == tokenMacro
	pos := tok.pos
	// Parses the function name if present.
	var ident *ast.Identifier
	tok = p.next()
	if tok.typ == tokenIdentifier {
		if kind&parseFuncDecl == 0 {
			panic(syntaxError(tok.pos, "unexpected %s, expecting (", tok))
		}
		ident = ast.NewIdentifier(tok.pos, string(tok.txt))
		tok = p.next()
	} else if kind == parseFuncDecl {
		// This check could be avoided (the code panics anyway) but improves the
		// readability of the error message.
		if !isMacro && tok.typ == tokenLeftParenthesis {
			panic(syntaxError(tok.pos, "method declarations are not supported in this release of Scriggo"))
		}
		// Node to parse must be a function declaration.
		panic(syntaxError(tok.pos, "unexpected %s, expecting name", tok.txt))
	}
	// Parses the input parameters.
	var parameters []*ast.Parameter
	var isVariadic bool
	var last *ast.Position
	parameters, isVariadic, last, tok = p.parseFuncParameters(tok, isMacro, false)
	if parameters != nil {
		pos.End = last.End
	} else if !isMacro || kind == parseFuncType {
		panic(syntaxError(tok.pos, "unexpected %s, expecting (", tok))
	}
	// Parses the result parameters.
	var result []*ast.Parameter
	result, _, last, tok = p.parseFuncParameters(tok, isMacro, true)
	if result != nil {
		pos.End = last.End
	} else if isMacro && kind == parseFuncType {
		panic(syntaxError(tok.pos, "unexpected %s, expecting string, html, css, js, json or markdown", tok))
	}

	// Make the nodes.
	typ := ast.NewFuncType(pos, isMacro, parameters, result, isVariadic)
	if kind == parseFuncType || kind&parseFuncType != 0 && tok.typ != tokenLeftBrace {
		typ.Position = &ast.Position{pos.Line, pos.Column, pos.Start, pos.End}
		return typ, tok
	}
	node := ast.NewFunc(pos, ident, typ, nil, false, ast.Format(tok.ctx))
	if !isMacro && tok.typ != tokenLeftBrace {
		return node, tok
	}
	body := ast.NewBlock(tok.pos, nil)
	node.Body = body
	if isMacro {
		return node, tok
	}
	// Parses the function body.
	p.addToAncestors(node)
	p.addToAncestors(body)
	depth := len(p.ancestors)
	tok = p.next()
	for {
		if tok.typ == tokenRightBrace {
			if _, ok := p.parent().(*ast.Label); ok {
				p.removeLastAncestor()
			}
			if len(p.ancestors) == depth {
				break
			}
		} else if tok.typ == tokenEOF {
			if _, ok := p.parent().(*ast.Label); ok {
				panic(syntaxError(tok.pos, "missing statement after label"))
			}
			panic(syntaxError(tok.pos, "unexpected EOF, expecting }"))
		}
		tok = p.parse(tok, tokenEOF)
	}
	body.Position.End = tok.pos.End
	node.Position.End = tok.pos.End
	p.removeLastAncestor()
	p.removeLastAncestor()
	return node, p.next()
}

// parseFuncParameters parses the parameters of a function or macro. tok is
// the first token of the parameters. isMacro indicates if it is a macro and
// isResult indicates if the parameters to parse are the result parameters.
//
// Returns the parameters, a boolean value indicating if the function is
// variadic, the position of the last token belonging to the parameters and
// the next token.
//
// If there are no parameters to parse, it returns nil, false, nil and tok.
func (p *parsing) parseFuncParameters(tok token, isMacro, isResult bool) ([]*ast.Parameter, bool, *ast.Position, token) {

	if isResult {
		if isMacro {
			switch name := string(tok.txt); name {
			case "string", "html", "css", "js", "json", "markdown":
				return []*ast.Parameter{{nil, ast.NewIdentifier(tok.pos, name)}}, false, tok.pos, p.next()
			}
			return nil, false, nil, tok
		}
		switch tok.typ {
		case tokenLeftBracket, tokenFunc, tokenIdentifier, tokenInterface, tokenMap, tokenMultiplication, tokenStruct, tokenChan:
			var expr ast.Expression
			expr, tok = p.parseExpr(tok, false, true, true)
			return []*ast.Parameter{ast.NewParameter(nil, expr)}, false, expr.Pos(), tok
		}
	}

	if tok.typ != tokenLeftParenthesis {
		return nil, false, nil, tok
	}

	tok = p.next()
	pos := tok.pos
	if tok.typ == tokenRightParenthesis {
		return []*ast.Parameter{}, false, pos, p.next()
	}

	var ide ast.Expression
	var ellipses struct {
		param *ast.Parameter
		index int
	}
	var parameters []*ast.Parameter

	for {
		param := ast.NewParameter(nil, nil)
		param.Type, tok = p.parseExpr(tok, false, true, false)
		if tok.typ == tokenEllipsis {
			if ellipses.param == nil {
				ellipses.param = param
				ellipses.index = len(parameters)
			}
			tok = p.next()
		}
		ide, tok = p.parseExpr(tok, false, true, false)
		if ide != nil {
			if param.Type != nil {
				var ok bool
				param.Ident, ok = param.Type.(*ast.Identifier)
				if !ok {
					panic(syntaxError(tok.pos, "unexpected %s, expecting )", param.Type))
				}
			}
			param.Type = ide
		}
		if param.Ident == nil && param.Type == nil {
			if tok.typ != tokenRightParenthesis {
				panic(syntaxError(tok.pos, "unexpected %s, expecting )", tok))
			}
			break
		}
		parameters = append(parameters, param)
		if tok.typ != tokenComma {
			if tok.typ != tokenRightParenthesis {
				panic(syntaxError(tok.pos, "unexpected %s, expecting comma or )", tok))
			}
			break
		}
		tok = p.next()
	}

	var last *ast.Parameter
	if parameters != nil {
		last = parameters[len(parameters)-1]
	}

	for _, param := range parameters {
		if last.Ident == nil {
			if param.Ident != nil {
				panic(syntaxError(pos, "mixed named and unnamed function parameters"))
			}
		} else if param.Ident == nil {
			var ok bool
			if ellipses.param != param {
				param.Ident, ok = param.Type.(*ast.Identifier)
			}
			if !ok {
				panic(syntaxError(tok.pos, "mixed named and unnamed function parameters"))
			}
			param.Type = nil
		}
	}

	if ellipses.param != nil {
		if isResult {
			panic(syntaxError(ellipses.param.Type.Pos(), "cannot use ... in receiver or result parameter list"))
		}
		if ellipses.param.Type == nil {
			panic(syntaxError(tok.pos, "final argument in variadic function missing type"))
		}
		final := ellipses.param
		for i := ellipses.index - 1; i >= 0; i-- {
			if param := parameters[i]; param.Type == nil {
				final = param
			} else {
				break
			}
		}
		if final != last {
			s := "cannot use ... with non-final parameter"
			if final.Ident == nil {
				panic(syntaxError(final.Type.Pos(), "%s", s))
			}
			panic(syntaxError(final.Ident.Pos(), "%s %s", s, final.Ident))
		}
	}

	return parameters, ellipses.param != nil, tok.pos, p.next()
}
