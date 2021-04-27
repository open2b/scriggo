// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"github.com/open2b/scriggo/compiler/ast"
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
	} else if kind^parseFuncDecl == 0 {
		// This check could be avoided (the code panics anyway) but improves the
		// readability of the error message.
		if tok.typ == tokenLeftParenthesis {
			panic(syntaxError(tok.pos, "method declarations are not supported in this release of Scriggo"))
		}
		// Node to parse must be a function declaration.
		panic(syntaxError(tok.pos, "unexpected %s, expecting name", tok.txt))
	}
	// Parses the function parameters.
	parameters, isVariadic, endPos := p.parseFuncParameters(tok, false)
	pos.End = endPos.End
	var result []*ast.Parameter
	tok = p.next()
	var expr ast.Expression
	// Parses the result.
	if isMacro {
		name := string(tok.txt)
		switch name {
		case "string", "html", "css", "js", "json", "markdown":
		default:
			panic(syntaxError(tok.pos, "unexpected %s, expecting string, html, css, js, json or markdown", tok))
		}
		pos.End = tok.pos.End
		result = []*ast.Parameter{{nil, ast.NewIdentifier(tok.pos, name)}}
		tok = p.next()
	} else {
		switch tok.typ {
		case tokenLeftParenthesis:
			result, _, endPos = p.parseFuncParameters(tok, true)
			pos.End = endPos.End
			tok = p.next()
		case tokenLeftBracket, tokenFunc, tokenIdentifier, tokenInterface, tokenMap, tokenMultiplication, tokenStruct, tokenChan:
			expr, tok = p.parseExpr(tok, false, true, true)
			pos.End = expr.Pos().End
			result = []*ast.Parameter{ast.NewParameter(nil, expr)}
		}
	}
	// Makes the function type.
	typ := ast.NewFuncType(nil, isMacro, parameters, result, isVariadic)
	if kind == parseFuncType || kind&parseFuncType != 0 && tok.typ != tokenLeftBrace {
		typ.Position = &ast.Position{pos.Line, pos.Column, pos.Start, pos.End}
		return typ, tok
	}
	// Parses the function body.
	node := ast.NewFunc(pos, ident, typ, nil, false, ast.Format(tok.ctx))
	if tok.typ != tokenLeftBrace {
		return node, tok
	}
	body := ast.NewBlock(tok.pos, nil)
	node.Body = body
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

func (p *parsing) parseFuncParameters(tok token, isResult bool) ([]*ast.Parameter, bool, *ast.Position) {

	if tok.typ != tokenLeftParenthesis {
		panic(syntaxError(tok.pos, "unexpected %s, expecting (", tok))
	}
	tok = p.next()
	pos := tok.pos
	if tok.typ == tokenRightParenthesis {
		return nil, false, pos
	}

	var ide ast.Expression
	var ellipses *ast.Parameter
	var parameters []*ast.Parameter

	for {
		param := ast.NewParameter(nil, nil)
		param.Type, tok = p.parseExpr(tok, false, true, false)
		if tok.typ == tokenEllipsis {
			if ellipses == nil {
				ellipses = param
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
			if ellipses != param {
				param.Ident, ok = param.Type.(*ast.Identifier)
			}
			if !ok {
				panic(syntaxError(tok.pos, "mixed named and unnamed function parameters"))
			}
			param.Type = nil
		}
	}

	if ellipses != nil {
		if isResult {
			panic(syntaxError(ellipses.Type.Pos(), "cannot use ... in receiver or result parameter list"))
		}
		if ellipses.Type == nil {
			panic(syntaxError(tok.pos, "final argument in variadic function missing type"))
		}
		if ellipses != last {
			s := "cannot use ... with non-final parameter"
			if ellipses.Ident == nil {
				panic(syntaxError(ellipses.Type.Pos(), "%s", s))
			}
			panic(syntaxError(ellipses.Ident.Pos(), "%s %s", s, ellipses.Ident))
		}
	}

	return parameters, ellipses != nil, tok.pos
}
