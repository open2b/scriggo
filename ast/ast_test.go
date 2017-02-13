//
// Copyright (c) 2017 Open2b Software Snc. All Rights Reserved.
//

package ast

import (
	"testing"

	"open2b/decimal"
)

var expressionStringTests = []struct {
	str  string
	expr Expression
}{
	{"1", NewInt(nil, 1)},
	{"3.59", NewDecimal(nil, decimal.String("3.59"))},
	{`"abc"`, NewString(nil, "abc")},
	{"\"a\\tb\"", NewString(nil, "a\tb")},
	{"x", NewIdentifier(nil, "x")},
	{"-1", NewUnaryOperator(nil, OperatorSubtraction, NewInt(nil, 1))},
	{"1+2", NewBinaryOperator(nil, OperatorAddition, NewInt(nil, 1), NewInt(nil, 2))},
	{"1+2", NewBinaryOperator(nil, OperatorAddition, NewInt(nil, 1), NewInt(nil, 2))},
	{"f()", NewCall(nil, NewIdentifier(nil, "f"), []Expression{})},
	{"f(a)", NewCall(nil, NewIdentifier(nil, "f"), []Expression{NewIdentifier(nil, "a")})},
	{"f(a,b)", NewCall(nil, NewIdentifier(nil, "f"), []Expression{NewIdentifier(nil, "a"), NewIdentifier(nil, "b")})},
	{"a[2]", NewIndex(nil, NewIdentifier(nil, "a"), NewInt(nil, 2))},
	{"a[:]", NewSlice(nil, NewIdentifier(nil, "a"), nil, nil)},
	{"a[2:]", NewSlice(nil, NewIdentifier(nil, "a"), NewInt(nil, 2), nil)},
	{"a[:5]", NewSlice(nil, NewIdentifier(nil, "a"), nil, NewInt(nil, 5))},
	{"a[2:5]", NewSlice(nil, NewIdentifier(nil, "a"), NewInt(nil, 2), NewInt(nil, 5))},
	{"a.b", NewSelector(nil, NewIdentifier(nil, "a"), "b")},
	{"(a)", NewParentesis(nil, NewIdentifier(nil, "a"))},
	{"-(1+2)", NewUnaryOperator(nil, OperatorSubtraction, NewBinaryOperator(nil, OperatorAddition, NewInt(nil, 1), NewInt(nil, 2)))},
	{"-(+1)", NewUnaryOperator(nil, OperatorSubtraction, NewUnaryOperator(nil, OperatorAddition, NewInt(nil, 1)))},
	{"1*2+-3", NewBinaryOperator(nil, OperatorAddition,
		NewBinaryOperator(nil, OperatorMultiplication, NewInt(nil, 1), NewInt(nil, 2)),
		NewUnaryOperator(nil, OperatorSubtraction, NewInt(nil, 3)))},
	{"f()-2", NewBinaryOperator(nil, OperatorSubtraction, NewCall(nil, NewIdentifier(nil, "f"), []Expression{}), NewInt(nil, 2))},
	{"-a.b", NewUnaryOperator(nil, OperatorSubtraction, NewSelector(nil, NewIdentifier(nil, "a"), "b"))},
}

func TestExpressionString(t *testing.T) {
	for _, e := range expressionStringTests {
		if e.expr.String() != e.str {
			t.Errorf("unexpected %q, expecting %q\n", e.expr.String(), e.str)
		}
	}
}
