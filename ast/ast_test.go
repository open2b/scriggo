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
	{"1", NewInt(0, 1)},
	{"3.59", NewDecimal(0, decimal.String("3.59"))},
	{`"abc"`, NewString(0, "abc")},
	{"\"a\\tb\"", NewString(0, "a\tb")},
	{"x", NewIdentifier(0, "x")},
	{"-1", NewUnaryOperator(0, OperatorSubtraction, NewInt(1, 1))},
	{"1+2", NewBinaryOperator(1, OperatorAddition, NewInt(0, 1), NewInt(2, 2))},
	{"1+2", NewBinaryOperator(1, OperatorAddition, NewInt(0, 1), NewInt(2, 2))},
	{"f()", NewCall(1, NewIdentifier(0, "f"), []Expression{})},
	{"f(a)", NewCall(1, NewIdentifier(0, "f"), []Expression{NewIdentifier(2, "a")})},
	{"f(a,b)", NewCall(1, NewIdentifier(0, "f"), []Expression{NewIdentifier(2, "a"), NewIdentifier(4, "b")})},
	{"a[2]", NewIndex(1, NewIdentifier(0, "a"), NewInt(2, 2))},
	{"a[:]", NewSlice(1, NewIdentifier(0, "a"), nil, nil)},
	{"a[2:]", NewSlice(1, NewIdentifier(0, "a"), NewInt(2, 2), nil)},
	{"a[:5]", NewSlice(1, NewIdentifier(0, "a"), nil, NewInt(3, 5))},
	{"a[2:5]", NewSlice(1, NewIdentifier(0, "a"), NewInt(2, 2), NewInt(4, 5))},
	{"a.b", NewSelector(1, NewIdentifier(0, "a"), "b")},
	{"(a)", NewParentesis(0, NewIdentifier(1, "a"))},
	{"-(1+2)", NewUnaryOperator(0, OperatorSubtraction, NewBinaryOperator(3, OperatorAddition, NewInt(2, 1), NewInt(4, 2)))},
	{"-(+1)", NewUnaryOperator(0, OperatorSubtraction, NewUnaryOperator(2, OperatorAddition, NewInt(3, 1)))},
	{"1*2+-3", NewBinaryOperator(5, OperatorAddition,
		NewBinaryOperator(2, OperatorMultiplication, NewInt(0, 1), NewInt(2, 2)),
		NewUnaryOperator(4, OperatorSubtraction, NewInt(4, 3)))},
	{"f()-2", NewBinaryOperator(3, OperatorSubtraction, NewCall(1, NewIdentifier(0, "f"), []Expression{}), NewInt(4, 2))},
	{"-a.b", NewUnaryOperator(0, OperatorSubtraction, NewSelector(2, NewIdentifier(1, "a"), "b"))},
}

func TestExpressionString(t *testing.T) {
	for _, e := range expressionStringTests {
		if e.expr.String() != e.str {
			t.Errorf("unexpected %q, expecting %q\n", e.expr.String(), e.str)
		}
	}
}
