// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"scriggo/ast"
)

type funcKindToParse int

const (
	parseFuncType funcKindToParse = 1 << iota // func type
	parseFuncLit                              // func literal
	parseFuncDecl                             // func declaration
)

// parseFunc parses a function type, literal o declaration. tok must be the
// "func" token.
//
// If kind does not include func type, the next token is not consumed.
func (p *parsing) parseFunc(tok token, kind funcKindToParse) (ast.Node, token) {
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
		// Node to parse must be a function declaration.
		panic(syntaxError(tok.pos, "unexpected %s, expecting name", tok.txt))
	}
	// Parses the function parameters.
	names := map[string]struct{}{}
	parameters, isVariadic, endPos := p.parseFuncParameters(tok, names, false)
	pos.End = endPos.End
	var result []*ast.Parameter
	tok = p.next()
	var expr ast.Expression
	// Parses the result if present.
	switch tok.typ {
	case tokenLeftParenthesis:
		result, _, endPos = p.parseFuncParameters(tok, names, true)
		pos.End = endPos.End
		tok = p.next()
	case tokenLeftBrackets, tokenFunc, tokenIdentifier, tokenInterface, tokenMap, tokenMultiplication, tokenStruct, tokenChan:
		expr, tok = p.parseExpr(tok, false, true, true)
		pos.End = expr.Pos().End
		result = []*ast.Parameter{ast.NewParameter(nil, expr)}
	}
	// Makes the function type.
	typ := ast.NewFuncType(nil, parameters, result, isVariadic)
	if kind^parseFuncType == 0 || kind&parseFuncType != 0 && tok.typ != tokenLeftBraces {
		typ.Position = &ast.Position{pos.Line, pos.Column, pos.Start, pos.End}
		return typ, tok
	}
	// Parses the function body.
	if tok.typ != tokenLeftBraces {
		p := pos
		if ident != nil {
			p = ident.Position
		}
		panic(syntaxError(p, "missing function body"))
	}
	body := ast.NewBlock(tok.pos, nil)
	node := ast.NewFunc(pos, ident, typ, body)
	p.addToAncestors(node)
	p.addToAncestors(body)
	depth := len(p.ancestors)
	isTemplate := p.isTemplate
	p.isTemplate = false
	tok = p.next()
	for {
		if tok.typ == tokenRightBraces {
			if _, ok := p.parent().(*ast.Label); ok {
				p.removeLastAncestor()
			}
			if len(p.ancestors) == depth {
				break
			}
		}
		if tok.typ == tokenEOF {
			if _, ok := p.parent().(*ast.Label); ok {
				panic(syntaxError(tok.pos, "missing statement after label"))
			}
			panic(syntaxError(tok.pos, "unexpected EOF, expecting }"))
		}
		tok = p.parseBlock(tok)
	}
	body.Position.End = tok.pos.End
	node.Position.End = tok.pos.End
	p.removeLastAncestor()
	p.removeLastAncestor()
	p.isTemplate = isTemplate
	return node, token{}
}

func (p *parsing) parseFuncParameters(tok token, names map[string]struct{}, isResult bool) ([]*ast.Parameter, bool, *ast.Position) {

	if tok.typ != tokenLeftParenthesis {
		panic(syntaxError(tok.pos, "unexpected %s, expecting (", tok))
	}

	var ok bool
	var pos = tok.pos
	var parameters []*ast.Parameter
	var expr ast.Expression
	var isVariadic bool

	for {
		tok = p.next()
		param := ast.NewParameter(nil, nil)
		param.Type, tok = p.parseExpr(tok, false, true, false)
		if tok.typ == tokenRightParenthesis {
			if param.Type != nil {
				parameters = append(parameters, param)
			}
			break
		}
		if tok.typ == tokenEllipsis {
			if isResult {
				panic(syntaxError(tok.pos, "cannot use ... in receiver or result parameter list"))
			}
			if param.Type != nil {
				if param.Ident, ok = param.Type.(*ast.Identifier); !ok {
					panic(syntaxError(tok.pos, "unexpected %s, expecting )", param.Type))
				}
				param.Type = nil
			}
			isVariadic = true
			tok = p.next()
		} else if param.Type == nil {
			panic(syntaxError(tok.pos, "unexpected %s, expecting )", tok))
		}
		expr, tok = p.parseExpr(tok, false, true, false)
		if expr == nil {
			if isVariadic {
				panic(syntaxError(tok.pos, "final argument in variadic function missing type"))
			}
		} else {
			if !isVariadic {
				var ok bool
				if param.Ident, ok = param.Type.(*ast.Identifier); !ok {
					panic(syntaxError(pos, "unexpected %s, expecting )", param.Type))
				}
			}
			param.Type = expr
		}
		parameters = append(parameters, param)
		if tok.typ == tokenRightParenthesis {
			break
		}
		if tok.typ != tokenComma {
			panic(syntaxError(tok.pos, "unexpected %s, expecting comma or ", tok))
		}
		if isVariadic {
			panic(syntaxError(param.Type.Pos(), "can only use ... with final parameter in list"))
		}
	}

	if len(parameters) > 0 {
		if last := parameters[len(parameters)-1]; last.Ident == nil {
			for _, field := range parameters {
				if field.Ident != nil {
					panic(syntaxError(pos, "mixed named and unnamed function parameters"))
				}
			}
		} else {
			for _, field := range parameters {
				if field.Ident == nil {
					if field.Ident, ok = field.Type.(*ast.Identifier); !ok {
						panic(syntaxError(field.Type.Pos(), "unexpected %s, expecting )", field.Type))
					}
					field.Type = nil
				}
				name := field.Ident.Name
				if _, ok := names[name]; ok {
					panic(syntaxError(field.Ident.Position, "duplicate argument %s", name))
				}
				if name != "_" {
					names[name] = struct{}{}
				}
			}
		}
	}

	return parameters, isVariadic, tok.pos
}
