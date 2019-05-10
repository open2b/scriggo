// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"

	"scrigo/compiler/ast"
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
	tok = next(p.lex)
	if tok.typ == tokenIdentifier {
		if kind&parseFuncDecl == 0 {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting (", tok)})
		}
		ident = ast.NewIdentifier(tok.pos, string(tok.txt))
		tok = next(p.lex)
	} else if kind^parseFuncDecl == 0 {
		// Node to parse must be a function declaration.
		panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting name", tok.txt)})
	}
	// Parses the function parameters.
	names := map[string]struct{}{}
	parameters, isVariadic, endPos := p.parseFuncFields(tok, names, false)
	pos.End = endPos.End
	var result []*ast.Field
	tok = next(p.lex)
	var expr ast.Expression
	// Parses the result if present.
	switch tok.typ {
	case tokenLeftParenthesis:
		result, _, endPos = p.parseFuncFields(tok, names, true)
		pos.End = endPos.End
		tok = next(p.lex)
	case tokenLeftBrackets, tokenFunc, tokenIdentifier, tokenInterface, tokenMap, tokenMultiplication, tokenStruct:
		expr, tok = p.parseExpr(tok, false, false, true, true)
		pos.End = tok.pos.End
		result = []*ast.Field{ast.NewField(nil, expr)}
	}
	// Makes the function type.
	typ := ast.NewFuncType(nil, parameters, result, isVariadic)
	if kind^parseFuncType == 0 || kind&parseFuncType != 0 && tok.typ != tokenLeftBraces {
		pos.End = tok.pos.End
		typ.Position = pos
		return typ, tok
	}
	if tok.typ != tokenLeftBraces {
		panic(&SyntaxError{"", *ident.Position, fmt.Errorf("missing function body")})
	}
	body := ast.NewBlock(tok.pos, nil)
	node := ast.NewFunc(pos, ident, typ, body)
	p.ancestors = append(p.ancestors, node, body)
	depth := len(p.ancestors)
	// Parses the function body.
	for {
		tok = next(p.lex)
		if tok.typ == tokenRightBraces {
			parent := p.ancestors[len(p.ancestors)-1]
			if _, ok := parent.(*ast.Label); ok {
				p.ancestors = p.ancestors[:len(p.ancestors)-1]
			}
			if len(p.ancestors) == depth {
				break
			}
		}
		if tok.typ == tokenEOF {
			parent := p.ancestors[len(p.ancestors)-1]
			if _, ok := parent.(*ast.Label); ok {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("missing statement after label")})
			}
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected EOF, expecting }")})
		}
		p.parseStatement(tok)
	}
	p.ancestors = p.ancestors[:len(p.ancestors)-2]
	body.Position.End = tok.pos.End
	node.Position.End = tok.pos.End
	return node, token{}
}

func (p *parsing) parseFuncFields(tok token, names map[string]struct{}, isResult bool) ([]*ast.Field, bool, *ast.Position) {

	if tok.typ != tokenLeftParenthesis {
		panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting (", tok)})
	}

	var ok bool
	var pos = tok.pos
	var fields []*ast.Field
	var expr ast.Expression
	var isVariadic bool

	for {
		tok = next(p.lex)
		field := ast.NewField(nil, nil)
		if tok.typ == tokenIdentifier {
			field.Type = parseIdentifierNode(tok)
			tok = next(p.lex)
		} else {
			field.Type, tok = p.parseExpr(tok, true, false, true, false)
		}
		if tok.typ == tokenRightParenthesis {
			if field.Type != nil {
				fields = append(fields, field)
			}
			break
		}
		if tok.typ == tokenEllipses {
			if isResult {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("cannot use ... in receiver or result parameter list")})
			}
			if field.Type != nil {
				if field.Ident, ok = field.Type.(*ast.Identifier); !ok {
					panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting )", field.Type)})
				}
				field.Type = nil
			}
			isVariadic = true
			tok = next(p.lex)
		} else if field.Type == nil {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting )", tok)})
		}
		expr, tok = p.parseExpr(tok, false, false, true, false)
		if expr == nil {
			if isVariadic {
				panic(&SyntaxError{"", *tok.pos, fmt.Errorf("final argument in variadic function missing type")})
			}
		} else {
			if !isVariadic {
				var ok bool
				if field.Ident, ok = field.Type.(*ast.Identifier); !ok {
					panic(&SyntaxError{"", *pos, fmt.Errorf("unexpected %s, expecting )", field.Type)})
				}
			}
			field.Type = expr
		}
		fields = append(fields, field)
		if tok.typ == tokenRightParenthesis {
			break
		}
		if tok.typ != tokenComma {
			panic(&SyntaxError{"", *tok.pos, fmt.Errorf("unexpected %s, expecting comma or ", tok)})
		}
		if isVariadic {
			panic(&SyntaxError{"", *field.Type.Pos(), fmt.Errorf("can only use ... with final parameter in list")})
		}
	}

	if len(fields) > 0 {
		if last := fields[len(fields)-1]; last.Ident == nil {
			for _, field := range fields {
				if field.Ident != nil {
					panic(&SyntaxError{"", *pos, fmt.Errorf("mixed named and unnamed function parameters")})
				}
			}
		} else {
			for _, field := range fields {
				if field.Ident == nil {
					if field.Ident, ok = field.Type.(*ast.Identifier); !ok {
						panic(&SyntaxError{"", *(field.Type.Pos()), fmt.Errorf("unexpected %s, expecting )", field.Type)})
					}
					field.Type = nil
				}
				name := field.Ident.Name
				if _, ok := names[name]; ok {
					panic(&SyntaxError{"", *field.Ident.Position, fmt.Errorf("duplicate argument %s", name)})
				}
				if name != "_" {
					names[name] = struct{}{}
				}
			}
		}
	}

	return fields, isVariadic, tok.pos
}
