// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"github.com/open2b/scriggo/ast"
)

// isTypeGuard reports whether node is a switch type guard, as x.(type) and
// v := x.(type).
func isTypeGuard(node ast.Node) bool {
	switch v := node.(type) {
	case *ast.Assignment:
		if len(v.Rhs) != 1 {
			return false
		}
		if ta, ok := v.Rhs[0].(*ast.TypeAssertion); ok {
			return ta.Type == nil
		}
	case *ast.TypeAssertion:
		return v.Type == nil
	}
	return false
}

// parseSwitch parses a switch statement and returns an Switch or TypeSwitch
// node. Panics on error.
func (p *parsing) parseSwitch(tok token, end tokenTyp) ast.Node {

	pos := tok.pos

	var assignment *ast.Assignment

	// "{%" "switch" [ beforeSemicolon ";" ] afterSemicolon "%}"
	var beforeSemicolon, afterSemicolon ast.Node

	expressions, tok := p.parseExprList(p.next(), true, false, true)

	want := tokenLeftBrace
	if end == tokenEndStatement {
		want = tokenEndStatement
	}

	switch {

	case tok.typ == want:
		switch len(expressions) {
		case 0:
			// switch {
		case 1:
			// switch x {
			// switch x + 2 {
			// switch f(2) {
			// switch x.(type) {
			afterSemicolon = expressions[0]
		default:
			// switch x + 2, y + 1 {
			panic(syntaxError(tok.pos, "unexpected %s, expecting := or = or comma", want))
		}

	case tok.typ == tokenSemicolon:
		switch len(expressions) { // # of expressions before ;
		case 0:
			// switch ; x + 2 {
			// switch ; {
			// switch ; x := a.(type) {
			// switch ; a.(type) {
		case 1:
			// switch f(3); x {
			beforeSemicolon = expressions[0]
		default:
			// switch f(), g(); x + 2 {
			// switch f(), g(); {
			panic(syntaxError(tok.pos, "unexpected semicolon, expecting := or = or comma"))
		}
		if isTypeGuard(beforeSemicolon) {
			// TODO (Gianluca): use type assertion node position instead of last read token position
			// TODO (Gianluca): move to type-checker:
			panic(syntaxError(tok.pos, "use of .(type) outside type switch"))
		}
		expressions, tok = p.parseExprList(p.next(), true, false, true)
		switch len(expressions) { // # of expressions after ;
		case 0:
			// switch ; {
			// switch f3(); {
		case 1:
			// switch f(3); x {
			// switch ; x + 2 {
			// switch x + 3; x.(type) {
			// switch ; x.(type) {
			afterSemicolon = expressions[0]
		default:
			// switch x; a, b {
			// switch ; a, b {
			// switch ; a, b, c {
			panic(syntaxError(tok.pos, "unexpected %s, expecting := or = or comma", want))
		}

	case isAssignmentToken(tok):
		// switch x := 3; x {
		// switch x := 3; x + y {
		// switch x = y.(type) {
		// switch x := 2; x = y.(type) {
		assignment, tok = p.parseAssignment(expressions, tok, false, true, true)
		switch tok.typ {
		case tokenSemicolon:

			if isTypeGuard(assignment) {
				// TODO (Gianluca): use type assertion node position instead of last read token position
				panic(syntaxError(tok.pos, "use of .(type) outside type switch"))
			}

			beforeSemicolon = assignment
			// switch x := 2; {
			// switch x := 3; x {
			// switch x := 3; x + y {
			// switch x := 2; x = y.(type) {
			expressions, tok = p.parseExprList(p.next(), true, false, true)
			if isAssignmentToken(tok) {
				// This is the only valid case where there is an assignment
				// before and after the semicolon token:
				//     switch x := 2; x = y.(type) {
				assignment, tok = p.parseAssignment(expressions, tok, false, true, true)
				ta, ok := assignment.Rhs[0].(*ast.TypeAssertion)
				// TODO (Gianluca): should error contain the position of the
				// expression which caused the error instead of the token (as Go
				// does)?
				if !ok || ta.Type != nil || len(assignment.Lhs) != 1 {
					panic(cannotUseAsValueError(tok.pos, assignment))
				}
				afterSemicolon = assignment
			} else {
				switch len(expressions) {
				case 0:
					// switch x := 2; {
				case 1:
					// switch x := 3; x {
					// switch x := 3; x + y {
					// switch x := 2; y.(type) {
					afterSemicolon = expressions[0]
				default:
					// switch x := 2; x + y, y + z {
					panic(syntaxError(tok.pos, "unexpected %s, expecting := or = or comma", want))
				}
			}

		case want:
			// switch x = y.(type) {
			// switch x := y.(type) {
			if len(assignment.Rhs) != 1 {
				panic(syntaxError(tok.pos, "unexpected %s, expecting expression", want))
			}
			ta, ok := assignment.Rhs[0].(*ast.TypeAssertion)
			if !ok || ta.Type != nil || len(assignment.Lhs) != 1 {
				panic(cannotUseAsValueError(tok.pos, assignment))
			}
			afterSemicolon = assignment
		}

	}

	if tok.typ != want {
		panic(syntaxError(tok.pos, "unexpected %s, expecting %s", tok, want))
	}

	pos.End = tok.pos.End

	var node ast.Node
	if isTypeGuard(afterSemicolon) {
		if a, ok := afterSemicolon.(*ast.TypeAssertion); ok {
			afterSemicolon = ast.NewAssignment(
				a.Pos(), []ast.Expression{ast.NewIdentifier(a.Pos(), "_")}, ast.AssignmentSimple, []ast.Expression{a},
			)
		}
		node = ast.NewTypeSwitch(pos, beforeSemicolon, afterSemicolon.(*ast.Assignment), nil, nil)
	} else {
		if afterSemicolon != nil {
			node = ast.NewSwitch(pos, beforeSemicolon, afterSemicolon.(ast.Expression), nil, nil)
		} else {
			node = ast.NewSwitch(pos, beforeSemicolon, nil, nil, nil)
		}
	}
	return node
}
