package parser

import (
	"fmt"

	"open2b/template/ast"
)

// isTypeGuard indicates if node is a switch type guard, as x.(type) and v :=
// x.(type).
func isTypeGuard(node ast.Node) bool {
	switch v := node.(type) {
	case *ast.Assignment:
		if len(v.Values) != 1 {
			return false
		}
		if ta, ok := v.Values[0].(*ast.TypeAssertion); ok {
			return ta.Type == nil
		}
	case *ast.TypeAssertion:
		return v.Type == nil
	}
	return false
}

// parseSwitch parses a switch statement and returns an Switch or TypeSwitch
// node. Panics on error.
//
// TODO (Gianluca): instead of returning an error, panic.
//
// TODO (Gianluca): instead of using "pos" (*ast.Position), use full token "tok"
// (token)
//
// TODO (Gianluca): change parameters order (final result should be tok token,
// lex *lexer)
//
func (p *parsing) parseSwitch(pos *ast.Position) ast.Node {

	// TODO (Gianluca): parsing this line: switch x := 2; x, y := a.(type), b
	// should fail, cause it's not checking the number of expressions. There
	// must be only one.

	var assignment *ast.Assignment

	// "{%" "switch" [ beforeSemicolon "," ] afterSemicolon "%}"
	var beforeSemicolon, afterSemicolon ast.Node

	expressions, tok := parseExprList(token{}, p.lex, false, true, false, true)

	end := tokenLeftBraces
	if p.ctx != ast.ContextNone {
		end = tokenEndStatement
	}

	switch {

	case tok.typ == end:
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
			panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %%}, expecting := or = or comma")})
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
			panic(&Error{"", *tok.pos, fmt.Errorf("unexpected semicolon, expecting := or = or comma")})
		}
		if isTypeGuard(beforeSemicolon) {
			// TODO (Gianluca): use type assertion node position instead of last read token position
			panic(&Error{"", *tok.pos, fmt.Errorf("use of .(type) outside type switch")})
		}
		expressions, tok = parseExprList(token{}, p.lex, false, true, false, true)
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
			panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %%}, expecting := or = or comma")})
		}

	case isAssignmentToken(tok):
		// switch x := 3; x {
		// switch x := 3; x + y {
		// switch x = y.(type) {
		// switch x := 2; x = y.(type) {
		assignment, tok = parseAssignment(expressions, tok, p.lex, true)
		switch tok.typ {
		case tokenSemicolon:

			if isTypeGuard(assignment) {
				// TODO (Gianluca): use type assertion node position instead of last read token position
				panic(&Error{"", *tok.pos, fmt.Errorf("use of .(type) outside type switch")})
			}

			beforeSemicolon = assignment
			// switch x := 2; {
			// switch x := 3; x {
			// switch x := 3; x + y {
			// switch x := 2; x = y.(type) {
			expressions, tok = parseExprList(token{}, p.lex, false, true, false, true)
			if isAssignmentToken(tok) {
				// This is the only valid case where there is an assignment
				// before and after the semicolon token:
				//     switch x := 2; x = y.(type) {
				assignment, tok = parseAssignment(expressions, tok, p.lex, true)
				ta, ok := assignment.Values[0].(*ast.TypeAssertion)
				// TODO (Gianluca): should error contain the position of the
				// expression which caused the error instead of the token (as Go
				// does)?
				if !ok {
					panic(&Error{"", *tok.pos, fmt.Errorf("assignment %s used as value", assignment)})
				}
				if ta.Type != nil {
					panic(&Error{"", *tok.pos, fmt.Errorf("%s used as value", assignment)})
				}
				if len(assignment.Variables) != 1 {
					panic(&Error{"", *tok.pos, fmt.Errorf("%s used as value", assignment)})
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
					panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %%}, expecting := or = or comma")})
				}
			}

		case end:
			// switch x = y.(type) {
			// switch x := y.(type) {
			if len(assignment.Values) != 1 {
				panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %%}, expecting expression")})
			}
			if len(assignment.Variables) != 1 {
				panic(&Error{"", *tok.pos, fmt.Errorf("%s used as value", assignment)})
			}
			ta, ok := assignment.Values[0].(*ast.TypeAssertion)
			if !ok {
				panic(&Error{"", *tok.pos, fmt.Errorf("assignment %s used as value", assignment)})
			}
			if ta.Type != nil {
				panic(&Error{"", *tok.pos, fmt.Errorf("%s used as value", assignment)})
			}
			afterSemicolon = assignment
		}

	}

	if tok.typ != end {
		panic(&Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)})
	}

	pos.End = tok.pos.End

	var node ast.Node
	if isTypeGuard(afterSemicolon) {
		if a, ok := afterSemicolon.(*ast.TypeAssertion); ok {
			afterSemicolon = ast.NewAssignment(
				a.Pos(), []ast.Expression{ast.NewIdentifier(a.Pos(), "_")}, ast.AssignmentSimple, []ast.Expression{a},
			)
		}
		node = ast.NewTypeSwitch(pos, beforeSemicolon, afterSemicolon.(*ast.Assignment), nil)
	} else {
		if afterSemicolon != nil {
			node = ast.NewSwitch(pos, beforeSemicolon, afterSemicolon.(ast.Expression), nil)
		} else {
			node = ast.NewSwitch(pos, beforeSemicolon, nil, nil)
		}
	}
	return node
}
