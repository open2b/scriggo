// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast

import (
	"testing"

	"github.com/shopspring/decimal"
)

var n1 = NewNumber(nil, decimal.New(1, 0))
var n2 = NewNumber(nil, decimal.New(2, 0))
var n3 = NewNumber(nil, decimal.New(3, 0))
var n5 = NewNumber(nil, decimal.New(5, 0))

var expressionStringTests = []struct {
	str  string
	expr Expression
}{
	{"1", n1},
	{"3.59", NewNumber(nil, decimal.NewFromFloat(3.59))},
	{`"abc"`, NewString(nil, "abc")},
	{"\"a\\tb\"", NewString(nil, "a\tb")},
	{"x", NewIdentifier(nil, "x")},
	{"-1", NewUnaryOperator(nil, OperatorSubtraction, n1)},
	{"1 + 2", NewBinaryOperator(nil, OperatorAddition, n1, n2)},
	{"1 + 2", NewBinaryOperator(nil, OperatorAddition, n1, n2)},
	{"f()", NewCall(nil, NewIdentifier(nil, "f"), []Expression{})},
	{"f(a)", NewCall(nil, NewIdentifier(nil, "f"), []Expression{NewIdentifier(nil, "a")})},
	{"f(a, b)", NewCall(nil, NewIdentifier(nil, "f"), []Expression{NewIdentifier(nil, "a"), NewIdentifier(nil, "b")})},
	{"a[2]", NewIndex(nil, NewIdentifier(nil, "a"), n2)},
	{"a[:]", NewSlicing(nil, NewIdentifier(nil, "a"), nil, nil)},
	{"a[2:]", NewSlicing(nil, NewIdentifier(nil, "a"), n2, nil)},
	{"a[:5]", NewSlicing(nil, NewIdentifier(nil, "a"), nil, n5)},
	{"a[2:5]", NewSlicing(nil, NewIdentifier(nil, "a"), n2, n5)},
	{"a.b", NewSelector(nil, NewIdentifier(nil, "a"), "b")},
	{"(a)", NewParentesis(nil, NewIdentifier(nil, "a"))},
	{"-(1 + 2)", NewUnaryOperator(nil, OperatorSubtraction, NewBinaryOperator(nil, OperatorAddition, n1, n2))},
	{"-(+1)", NewUnaryOperator(nil, OperatorSubtraction, NewUnaryOperator(nil, OperatorAddition, n1))},
	{"1 * 2 + -3", NewBinaryOperator(nil, OperatorAddition,
		NewBinaryOperator(nil, OperatorMultiplication, n1, n2),
		NewUnaryOperator(nil, OperatorSubtraction, n3))},
	{"f() - 2", NewBinaryOperator(nil, OperatorSubtraction, NewCall(nil, NewIdentifier(nil, "f"), []Expression{}), n2)},
	{"-a.b", NewUnaryOperator(nil, OperatorSubtraction, NewSelector(nil, NewIdentifier(nil, "a"), "b"))},
}

func TestExpressionString(t *testing.T) {
	for _, e := range expressionStringTests {
		if e.expr.String() != e.str {
			t.Errorf("unexpected %q, expecting %q\n", e.expr.String(), e.str)
		}
	}
}
