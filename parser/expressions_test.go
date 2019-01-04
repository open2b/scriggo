// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"testing"

	"open2b/template/ast"

	"github.com/shopspring/decimal"
)

var maxInt64Plus1, _ = decimal.NewFromString("9223372036854775808")
var minInt64Minus1, _ = decimal.NewFromString("-9223372036854775809")
var bigInt, _ = decimal.NewFromString("433937734937734969526500969526500")

var exprTests = []struct {
	src  string
	node ast.Node
}{
	{"a", ast.NewIdentifier(p(1, 1, 0, 0), "a")},
	{"a5", ast.NewIdentifier(p(1, 1, 0, 1), "a5")},
	{"_a", ast.NewIdentifier(p(1, 1, 0, 1), "_a")},
	{"_5", ast.NewIdentifier(p(1, 1, 0, 1), "_5")},
	{"0", ast.NewInt(p(1, 1, 0, 0), 0)},
	{"3", ast.NewInt(p(1, 1, 0, 0), 3)},
	{"2147483647", ast.NewInt(p(1, 1, 0, 9), 2147483647)},                      // math.MaxInt32
	{"-2147483648", ast.NewInt(p(1, 1, 0, 10), -2147483648)},                   // math.MinInt32
	{"9223372036854775807", ast.NewInt(p(1, 1, 0, 18), 9223372036854775807)},   // math.MaxInt64
	{"-9223372036854775808", ast.NewInt(p(1, 1, 0, 19), -9223372036854775808)}, // math.MinInt64
	{"2147483648", ast.NewInt(p(1, 1, 0, 9), 2147483648)},                      // math.MaxInt32 + 1
	{"-2147483649", ast.NewInt(p(1, 1, 0, 10), -2147483649)},                   // math.MinInt32 - 1
	{"9223372036854775808", ast.NewNumber(p(1, 1, 0, 18), maxInt64Plus1)},      // math.MaxInt64 + 1
	{"-9223372036854775809", ast.NewNumber(p(1, 1, 0, 19), minInt64Minus1)},    // math.MinInt64 - 1
	{"433937734937734969526500969526500", ast.NewNumber(p(1, 1, 0, 32), bigInt)},
	{"\"\"", ast.NewString(p(1, 1, 0, 1), "")},
	{"\"a\"", ast.NewString(p(1, 1, 0, 2), "a")},
	{`"\t"`, ast.NewString(p(1, 1, 0, 3), "\t")},
	{`"\a\b\f\n\r\t\v\\\""`, ast.NewString(p(1, 1, 0, 19), "\a\b\f\n\r\t\v\\\"")},
	{"\"\uFFFD\"", ast.NewString(p(1, 1, 0, 4), "\uFFFD")},
	{`"\u0000"`, ast.NewString(p(1, 1, 0, 7), "\u0000")},
	{`"\u0012"`, ast.NewString(p(1, 1, 0, 7), "\u0012")},
	{`"\u1234"`, ast.NewString(p(1, 1, 0, 7), "\u1234")},
	{`"\U00000000"`, ast.NewString(p(1, 1, 0, 11), "\U00000000")},
	{`"\U0010ffff"`, ast.NewString(p(1, 1, 0, 11), "\U0010FFFF")},
	{`"\U0010FFFF"`, ast.NewString(p(1, 1, 0, 11), "\U0010FFFF")},
	{"``", ast.NewString(p(1, 1, 0, 1), "")},
	{"`\\t`", ast.NewString(p(1, 1, 0, 3), "\\t")},
	{"`\uFFFD`", ast.NewString(p(1, 1, 0, 4), "\uFFFD")},
	{"!a", ast.NewUnaryOperator(p(1, 1, 0, 1), ast.OperatorNot, ast.NewIdentifier(p(1, 2, 1, 1), "a"))},
	{"1+2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 3, 2, 2), 2))},
	{"1-2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorSubtraction, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 3, 2, 2), 2))},
	{"1*2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorMultiplication, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 3, 2, 2), 2))},
	{"1/2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorDivision, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 3, 2, 2), 2))},
	{"1%2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorModulo, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 3, 2, 2), 2))},
	{"1==2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorEqual, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 4, 3, 3), 2))},
	{"1!=2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorNotEqual, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 4, 3, 3), 2))},
	{"1<2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorLess, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 3, 2, 2), 2))},
	{"1<=2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorLessOrEqual, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 4, 3, 3), 2))},
	{"1>2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorGreater, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 3, 2, 2), 2))},
	{"1>=2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorGreaterOrEqual, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 4, 3, 3), 2))},
	{"a&&b", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorAnd, ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 3), "b"))},
	{"a||b", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorOr, ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 3), "b"))},
	{"1+-2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 3, 2, 3), -2))},
	{"1+-(2)", ast.NewBinaryOperator(p(1, 2, 0, 5), ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), 1),
		ast.NewUnaryOperator(p(1, 3, 2, 5), ast.OperatorSubtraction, ast.NewInt(p(1, 5, 3, 5), 2)))},
	{"(a)", ast.NewIdentifier(p(1, 2, 0, 2), "a")},
	{"a()", ast.NewCall(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{})},
	{"a(1)", ast.NewCall(p(1, 2, 0, 3), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{ast.NewInt(p(1, 3, 2, 2), 1)})},
	{"a(1,2)", ast.NewCall(p(1, 2, 0, 5), ast.NewIdentifier(p(1, 1, 0, 0), "a"),
		[]ast.Expression{ast.NewInt(p(1, 3, 2, 2), 1), ast.NewInt(p(1, 5, 4, 4), 2)})},
	{"a[1]", ast.NewIndex(p(1, 2, 0, 3), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(1, 3, 2, 2), 1))},
	{"a[:]", ast.NewSlicing(p(1, 2, 0, 3), ast.NewIdentifier(p(1, 1, 0, 0), "a"), nil, nil)},
	{"a[:2]", ast.NewSlicing(p(1, 2, 0, 4), ast.NewIdentifier(p(1, 1, 0, 0), "a"), nil, ast.NewInt(p(1, 4, 3, 3), 2))},
	{"a[1:]", ast.NewSlicing(p(1, 2, 0, 4), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(1, 3, 2, 2), 1), nil)},
	{"a[1:2]", ast.NewSlicing(p(1, 2, 0, 5), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(1, 3, 2, 2), 1), ast.NewInt(p(1, 5, 4, 4), 2))},
	{"a.B", ast.NewSelector(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), "B")},
	{"1+2+3", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorAddition, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 3, 2, 2), 2)), ast.NewInt(p(1, 5, 4, 4), 3))},
	{"1-2-3", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorSubtraction, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorSubtraction, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 3, 2, 2), 2)), ast.NewInt(p(1, 5, 4, 4), 3))},
	{"1*2*3", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorMultiplication, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorMultiplication, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 3, 2, 2), 2)), ast.NewInt(p(1, 5, 4, 4), 3))},
	{"1+2*3", ast.NewBinaryOperator(p(1, 2, 0, 4), ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), 1),
		ast.NewBinaryOperator(p(1, 4, 2, 4), ast.OperatorMultiplication, ast.NewInt(p(1, 3, 2, 2), 2), ast.NewInt(p(1, 5, 4, 4), 3)))},
	{"1-2/3", ast.NewBinaryOperator(p(1, 2, 0, 4), ast.OperatorSubtraction, ast.NewInt(p(1, 1, 0, 0), 1),
		ast.NewBinaryOperator(p(1, 4, 2, 4), ast.OperatorDivision, ast.NewInt(p(1, 3, 2, 2), 2), ast.NewInt(p(1, 5, 4, 4), 3)))},
	{"1*2+3", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorAddition, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorMultiplication, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 3, 2, 2), 2)), ast.NewInt(p(1, 5, 4, 4), 3))},
	{"1==2+3", ast.NewBinaryOperator(p(1, 2, 0, 5), ast.OperatorEqual, ast.NewInt(p(1, 1, 0, 0), 1),
		ast.NewBinaryOperator(p(1, 5, 3, 5), ast.OperatorAddition, ast.NewInt(p(1, 4, 3, 3), 2), ast.NewInt(p(1, 6, 5, 5), 3)))},
	{"1+2==3", ast.NewBinaryOperator(p(1, 4, 0, 5), ast.OperatorEqual, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(1, 3, 2, 2), 2)), ast.NewInt(p(1, 6, 5, 5), 3))},
	{"(1+2)*3", ast.NewBinaryOperator(p(1, 6, 0, 6), ast.OperatorMultiplication, ast.NewBinaryOperator(p(1, 3, 0, 4),
		ast.OperatorAddition, ast.NewInt(p(1, 2, 1, 1), 1), ast.NewInt(p(1, 4, 3, 3), 2)), ast.NewInt(p(1, 7, 6, 6), 3))},
	{"1*(2+3)", ast.NewBinaryOperator(p(1, 2, 0, 6), ast.OperatorMultiplication, ast.NewInt(p(1, 1, 0, 0), 1),
		ast.NewBinaryOperator(p(1, 5, 2, 6), ast.OperatorAddition, ast.NewInt(p(1, 4, 3, 3), 2), ast.NewInt(p(1, 6, 5, 5), 3)))},
	{"(1*((2)+3))", ast.NewBinaryOperator(p(1, 3, 0, 10), ast.OperatorMultiplication, ast.NewInt(p(1, 2, 1, 1), 1),
		ast.NewBinaryOperator(p(1, 8, 3, 9), ast.OperatorAddition, ast.NewInt(p(1, 6, 4, 6), 2), ast.NewInt(p(1, 9, 8, 8), 3)))},
	{"a()*1", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorMultiplication,
		ast.NewCall(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{}), ast.NewInt(p(1, 5, 4, 4), 1))},
	{"1*a()", ast.NewBinaryOperator(p(1, 2, 0, 4), ast.OperatorMultiplication,
		ast.NewInt(p(1, 1, 0, 0), 1), ast.NewCall(p(1, 4, 2, 4), ast.NewIdentifier(p(1, 3, 2, 2), "a"), []ast.Expression{}))},
	{"a[1]*2", ast.NewBinaryOperator(p(1, 5, 0, 5), ast.OperatorMultiplication, ast.NewIndex(p(1, 2, 0, 3),
		ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(1, 3, 2, 2), 1)), ast.NewInt(p(1, 6, 5, 5), 2))},
	{"1*a[2]", ast.NewBinaryOperator(p(1, 2, 0, 5), ast.OperatorMultiplication, ast.NewInt(p(1, 1, 0, 0), 1),
		ast.NewIndex(p(1, 4, 2, 5), ast.NewIdentifier(p(1, 3, 2, 2), "a"), ast.NewInt(p(1, 5, 4, 4), 2)))},
	{"a[1+2]", ast.NewIndex(p(1, 2, 0, 5), ast.NewIdentifier(p(1, 1, 0, 0), "a"),
		ast.NewBinaryOperator(p(1, 4, 2, 4), ast.OperatorAddition, ast.NewInt(p(1, 3, 2, 2), 1), ast.NewInt(p(1, 5, 4, 4), 2)))},
	{"a[b(1)]", ast.NewIndex(p(1, 2, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewCall(p(1, 4, 2, 5),
		ast.NewIdentifier(p(1, 3, 2, 2), "b"), []ast.Expression{ast.NewInt(p(1, 5, 4, 4), 1)}))},
	{"a(b[1])", ast.NewCall(p(1, 2, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{
		ast.NewIndex(p(1, 4, 2, 5), ast.NewIdentifier(p(1, 3, 2, 2), "b"), ast.NewInt(p(1, 5, 4, 4), 1))})},
	{"a.B*c", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorMultiplication, ast.NewSelector(p(1, 2, 0, 2),
		ast.NewIdentifier(p(1, 1, 0, 0), "a"), "B"), ast.NewIdentifier(p(1, 5, 4, 4), "c"))},
	{"a*b.C", ast.NewBinaryOperator(p(1, 2, 0, 4), ast.OperatorMultiplication, ast.NewIdentifier(p(1, 1, 0, 0), "a"),
		ast.NewSelector(p(1, 4, 2, 4), ast.NewIdentifier(p(1, 3, 2, 2), "b"), "C"))},
	{"a.B(c)", ast.NewCall(p(1, 4, 0, 5), ast.NewSelector(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), "B"),
		[]ast.Expression{ast.NewIdentifier(p(1, 5, 4, 4), "c")})},
	{"a.(string)", ast.NewTypeAssertion(p(1, 2, 0, 9), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 8), "string"))},
	{"html(a).(html)", ast.NewTypeAssertion(p(1, 8, 0, 13), ast.NewCall(p(1, 5, 0, 6),
		ast.NewIdentifier(p(1, 1, 0, 3), "html"), []ast.Expression{ast.NewIdentifier(p(1, 6, 5, 5), "a")}), ast.NewIdentifier(p(1, 10, 9, 12), "html"))},
	{"a.(number)", ast.NewTypeAssertion(p(1, 2, 0, 9), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 8), "number"))},
	{"a.(int)", ast.NewTypeAssertion(p(1, 2, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 5), "int"))},
	{"a.(bool)", ast.NewTypeAssertion(p(1, 2, 0, 7), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 6), "bool"))},
	{"a.(map)", ast.NewTypeAssertion(p(1, 2, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 5), "map"))},
	{"a.(slice)", ast.NewTypeAssertion(p(1, 2, 0, 8), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 7), "slice"))},
	{"map{}", ast.NewMap(p(1, 1, 0, 4), []ast.KeyValue{})},
	{"map{`a`:5}", ast.NewMap(p(1, 1, 0, 9), []ast.KeyValue{
		{ast.NewString(p(1, 5, 4, 6), "a"), ast.NewInt(p(1, 9, 8, 8), 5)}})},
	{"map{`a`:5,}", ast.NewMap(p(1, 1, 0, 10), []ast.KeyValue{
		{ast.NewString(p(1, 5, 4, 6), "a"), ast.NewInt(p(1, 9, 8, 8), 5)}})},
	{"map{`a`:5,`b`:7}", ast.NewMap(p(1, 1, 0, 15), []ast.KeyValue{
		{ast.NewString(p(1, 5, 4, 6), "a"), ast.NewInt(p(1, 9, 8, 8), 5)},
		{ast.NewString(p(1, 11, 10, 12), "b"), ast.NewInt(p(1, 15, 14, 14), 7)}})},
	{"map(nil)", ast.NewCall(p(1, 1, 0, 7), ast.NewIdentifier(p(1, 1, 0, 2), "map"), []ast.Expression{ast.NewIdentifier(p(1, 5, 4, 6), "nil")})},
	{"slice{}", ast.NewSlice(p(1, 1, 0, 6), []ast.Expression{})},
	{"slice{5}", ast.NewSlice(p(1, 1, 0, 7), []ast.Expression{ast.NewInt(p(1, 7, 6, 6), 5)})},
	{"slice{5,6,7}", ast.NewSlice(p(1, 1, 0, 11), []ast.Expression{ast.NewInt(p(1, 7, 6, 6), 5),
		ast.NewInt(p(1, 9, 8, 8), 6), ast.NewInt(p(1, 11, 10, 10), 7)})},
	{"slice{slice{}}", ast.NewSlice(p(1, 1, 0, 13), []ast.Expression{ast.NewSlice(p(1, 7, 6, 12), []ast.Expression{})})},
	{"slice(nil)", ast.NewCall(p(1, 1, 0, 9), ast.NewIdentifier(p(1, 1, 0, 4), "slice"), []ast.Expression{ast.NewIdentifier(p(1, 7, 6, 8), "nil")})},
	{"1\t+\n2", ast.NewBinaryOperator(p(1, 3, 0, 4), ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(2, 1, 4, 4), 2))},
	{"1\t\r +\n\r\n\r\t 2", ast.NewBinaryOperator(p(1, 5, 0, 11), ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(3, 4, 11, 11), 2))},
	{"a(\n\t1\t,\n2\t)", ast.NewCall(p(1, 2, 0, 10), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{
		ast.NewInt(p(2, 2, 4, 4), 1), ast.NewInt(p(3, 1, 8, 8), 2)})},
	{"a\t\r ()", ast.NewCall(p(1, 5, 0, 5), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{})},
	{"a[\n\t1\t]", ast.NewIndex(p(1, 2, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(2, 2, 4, 4), 1))},
	{"a\t\r [1]", ast.NewIndex(p(1, 5, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(1, 6, 5, 5), 1))},
}

func TestExpressions(t *testing.T) {
	for _, expr := range exprTests {
		var lex = newLexer([]byte("{{"+expr.src+"}}"), ast.ContextText)
		<-lex.tokens
		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(*Error); ok {
						t.Errorf("source: %q, %s\n", expr.src, err)
					} else {
						panic(r)
					}
				}
			}()
			node, tok := parseExpr(token{}, lex, false)
			if node == nil {
				t.Errorf("source: %q, unexpected %s, expecting expression\n", expr.src, tok)
			} else {
				err := equals(node, expr.node, 2)
				if err != nil {
					t.Errorf("source: %q, %s\n", expr.src, err)
				}
			}
		}()
	}
}
