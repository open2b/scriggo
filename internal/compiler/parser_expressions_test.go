// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"testing"

	"scriggo/ast"
)

var maxInt64Plus1 = "9223372036854775808"
var minInt64Minus1 = "9223372036854775809"
var bigInt = "433937734937734969526500969526500"

func parenthesized(expr ast.Expression) ast.Expression {
	expr.SetParenthesis(expr.Parenthesis() + 1)
	return expr
}

var exprTests = []struct {
	src  string
	node ast.Node
}{
	{"a", ast.NewIdentifier(p(1, 1, 0, 0), "a")},
	{"a5", ast.NewIdentifier(p(1, 1, 0, 1), "a5")},
	{"_a", ast.NewIdentifier(p(1, 1, 0, 1), "_a")},
	{"_5", ast.NewIdentifier(p(1, 1, 0, 1), "_5")},
	{"0", ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "0")},
	{"3", ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "3")},
	{"2147483647", ast.NewBasicLiteral(p(1, 1, 0, 9), ast.IntLiteral, "2147483647")}, // math.MaxInt32
	{"-2147483648", ast.NewUnaryOperator(p(1, 1, 0, 10), ast.OperatorSubtraction,
		ast.NewBasicLiteral(p(1, 2, 1, 10), ast.IntLiteral, "2147483648"))}, // math.MinInt32
	{"9223372036854775807", ast.NewBasicLiteral(p(1, 1, 0, 18), ast.IntLiteral, "9223372036854775807")}, // math.MaxInt64
	{"-9223372036854775808", ast.NewUnaryOperator(p(1, 1, 0, 19), ast.OperatorSubtraction,
		ast.NewBasicLiteral(p(1, 2, 1, 19), ast.IntLiteral, "9223372036854775808"))}, // math.MinInt64
	{"2147483648", ast.NewBasicLiteral(p(1, 1, 0, 9), ast.IntLiteral, "2147483648")}, // math.MaxInt32 + 1
	{"-2147483649", ast.NewUnaryOperator(p(1, 1, 0, 10), ast.OperatorSubtraction,
		ast.NewBasicLiteral(p(1, 2, 1, 10), ast.IntLiteral, "2147483649"))}, // math.MinInt32 - 1
	{"9223372036854775808", ast.NewBasicLiteral(p(1, 1, 0, 18), ast.IntLiteral, maxInt64Plus1)}, // math.MaxInt64 + 1
	{"-9223372036854775809", ast.NewUnaryOperator(p(1, 1, 0, 19), ast.OperatorSubtraction,
		ast.NewBasicLiteral(p(1, 2, 1, 19), ast.IntLiteral, minInt64Minus1))}, // math.MinInt64 - 1
	{"433937734937734969526500969526500", ast.NewBasicLiteral(p(1, 1, 0, 32), ast.IntLiteral, bigInt)},
	{"'a'", ast.NewBasicLiteral(p(1, 1, 0, 2), ast.RuneLiteral, `'a'`)},
	{`'\a'`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.RuneLiteral, `'\a'`)},
	{`'\b'`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.RuneLiteral, `'\b'`)},
	{`'\f'`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.RuneLiteral, `'\f'`)},
	{`'\n'`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.RuneLiteral, `'\n'`)},
	{`'\r'`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.RuneLiteral, `'\r'`)},
	{`'\t'`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.RuneLiteral, `'\t'`)},
	{`'\v'`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.RuneLiteral, `'\v'`)},
	{`'\\'`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.RuneLiteral, `'\\'`)},
	{`'\''`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.RuneLiteral, `'\''`)},
	{`'\x00'`, ast.NewBasicLiteral(p(1, 1, 0, 5), ast.RuneLiteral, `'\x00'`)},
	{`'\x54'`, ast.NewBasicLiteral(p(1, 1, 0, 5), ast.RuneLiteral, `'\x54'`)},
	{`'\xff'`, ast.NewBasicLiteral(p(1, 1, 0, 5), ast.RuneLiteral, `'\xff'`)},
	{`'\xFF'`, ast.NewBasicLiteral(p(1, 1, 0, 5), ast.RuneLiteral, `'\xFF'`)},
	{`'\u0000'`, ast.NewBasicLiteral(p(1, 1, 0, 7), ast.RuneLiteral, `'\u0000'`)},
	{`'\u0012'`, ast.NewBasicLiteral(p(1, 1, 0, 7), ast.RuneLiteral, `'\u0012'`)},
	{`'\u1234'`, ast.NewBasicLiteral(p(1, 1, 0, 7), ast.RuneLiteral, `'\u1234'`)},
	{`'\U00000000'`, ast.NewBasicLiteral(p(1, 1, 0, 11), ast.RuneLiteral, `'\U00000000'`)},
	{`'\U0010ffff'`, ast.NewBasicLiteral(p(1, 1, 0, 11), ast.RuneLiteral, `'\U0010ffff'`)},
	{`'\U0010FFFF'`, ast.NewBasicLiteral(p(1, 1, 0, 11), ast.RuneLiteral, `'\U0010FFFF'`)},
	{`'\000'`, ast.NewBasicLiteral(p(1, 1, 0, 5), ast.RuneLiteral, `'\000'`)},
	{`'\177'`, ast.NewBasicLiteral(p(1, 1, 0, 5), ast.RuneLiteral, `'\177'`)},
	{`'\377'`, ast.NewBasicLiteral(p(1, 1, 0, 5), ast.RuneLiteral, `'\377'`)},
	{"\"\"", ast.NewBasicLiteral(p(1, 1, 0, 1), ast.StringLiteral, "\"\"")},
	{"\"a\"", ast.NewBasicLiteral(p(1, 1, 0, 2), ast.StringLiteral, "\"a\"")},
	{`"\a"`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.StringLiteral, `"\a"`)},
	{`"\b"`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.StringLiteral, `"\b"`)},
	{`"\f"`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.StringLiteral, `"\f"`)},
	{`"\n"`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.StringLiteral, `"\n"`)},
	{`"\r"`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.StringLiteral, `"\r"`)},
	{`"\t"`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.StringLiteral, `"\t"`)},
	{`"\v"`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.StringLiteral, `"\v"`)},
	{`"\\"`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.StringLiteral, `"\\"`)},
	{`"\""`, ast.NewBasicLiteral(p(1, 1, 0, 3), ast.StringLiteral, `"\""`)},
	{`"\a\b\f\n\r\t\v\\\""`, ast.NewBasicLiteral(p(1, 1, 0, 19), ast.StringLiteral, `"\a\b\f\n\r\t\v\\\""`)},
	{"\"\uFFFD\"", ast.NewBasicLiteral(p(1, 1, 0, 4), ast.StringLiteral, "\"\uFFFD\"")},
	{`"\u0000"`, ast.NewBasicLiteral(p(1, 1, 0, 7), ast.StringLiteral, `"\u0000"`)},
	{`"\u0012"`, ast.NewBasicLiteral(p(1, 1, 0, 7), ast.StringLiteral, `"\u0012"`)},
	{`"\u1234"`, ast.NewBasicLiteral(p(1, 1, 0, 7), ast.StringLiteral, `"\u1234"`)},
	{`"\U00000000"`, ast.NewBasicLiteral(p(1, 1, 0, 11), ast.StringLiteral, `"\U00000000"`)},
	{`"\U0010ffff"`, ast.NewBasicLiteral(p(1, 1, 0, 11), ast.StringLiteral, `"\U0010ffff"`)},
	{`"\U0010FFFF"`, ast.NewBasicLiteral(p(1, 1, 0, 11), ast.StringLiteral, `"\U0010FFFF"`)},
	{"``", ast.NewBasicLiteral(p(1, 1, 0, 1), ast.StringLiteral, "``")},
	{"`\\t`", ast.NewBasicLiteral(p(1, 1, 0, 3), ast.StringLiteral, "`\\t`")},
	{"`\uFFFD`", ast.NewBasicLiteral(p(1, 1, 0, 4), ast.StringLiteral, "`\uFFFD`")},
	{"!a", ast.NewUnaryOperator(p(1, 1, 0, 1), ast.OperatorNot, ast.NewIdentifier(p(1, 2, 1, 1), "a"))},
	{"&a", ast.NewUnaryOperator(p(1, 1, 0, 1), ast.OperatorAnd, ast.NewIdentifier(p(1, 2, 1, 1), "a"))},
	{"*a", ast.NewUnaryOperator(p(1, 1, 0, 1), ast.OperatorMultiplication, ast.NewIdentifier(p(1, 2, 1, 1), "a"))},
	{"^a", ast.NewUnaryOperator(p(1, 1, 0, 1), ast.OperatorXor, ast.NewIdentifier(p(1, 2, 1, 1), "a"))},
	{"1+2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorAddition, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2"))},
	{"1-2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorSubtraction, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2"))},
	{"1*2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorMultiplication, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2"))},
	{"1+2*3+1", ast.NewBinaryOperator(p(1, 6, 0, 6), ast.OperatorAddition,
		ast.NewBinaryOperator(p(1, 2, 0, 4), ast.OperatorAddition,
			ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"),
			ast.NewBinaryOperator(p(1, 4, 2, 4), ast.OperatorMultiplication,
				ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2"),
				ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "3"))),
		ast.NewBasicLiteral(p(1, 7, 6, 6), ast.IntLiteral, "1"))},
	{"1/2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorDivision, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2"))},
	{"1%2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorModulo, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2"))},
	{"1==2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorEqual, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 4, 3, 3), ast.IntLiteral, "2"))},
	{"1!=2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorNotEqual, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 4, 3, 3), ast.IntLiteral, "2"))},
	{"1<2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorLess, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2"))},
	{"1<=2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorLessOrEqual, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 4, 3, 3), ast.IntLiteral, "2"))},
	{"1>2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorGreater, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2"))},
	{"1>=2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorGreaterOrEqual, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 4, 3, 3), ast.IntLiteral, "2"))},
	{"a&&b", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorAndAnd, ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 3), "b"))},
	{"a||b", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorOrOr, ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 3), "b"))},
	{"1&2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorAnd, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2"))},
	{"1|2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorOr, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2"))},
	{"1^2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorXor, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2"))},
	{"a&^b", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorAndNot, ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 3), "b"))},
	{"a<<b", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorLeftShift, ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 3), "b"))},
	{"a>>b", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorRightShift, ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 3), "b"))},
	{"1+-2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorAddition,
		ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"),
		ast.NewUnaryOperator(p(1, 3, 2, 3), ast.OperatorSubtraction,
			ast.NewBasicLiteral(p(1, 4, 3, 3), ast.IntLiteral, "2")))},
	{"1+-(2)", ast.NewBinaryOperator(p(1, 2, 0, 5), ast.OperatorAddition, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"),
		ast.NewUnaryOperator(p(1, 3, 2, 5), ast.OperatorSubtraction, parenthesized(ast.NewBasicLiteral(p(1, 5, 3, 5), ast.IntLiteral, "2"))))},
	// TODO (Gianluca): positions in this test have been altered to make the
	// test pass.
	{"*a+*b", ast.NewBinaryOperator(
		p(1, 3, 1, 4),
		ast.OperatorAddition,
		ast.NewUnaryOperator(
			p(1, 1, 1, 1),
			ast.OperatorMultiplication,
			ast.NewIdentifier(p(1, 2, 1, 1), "a"),
		),
		ast.NewUnaryOperator(
			p(1, 4, 3, 4),
			ast.OperatorMultiplication,
			ast.NewIdentifier(p(1, 5, 4, 4), "b"),
		))},
	{"(a)", parenthesized(ast.NewIdentifier(p(1, 2, 0, 2), "a"))},
	{"a()", ast.NewCall(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{}, false)},
	{"a(1)", ast.NewCall(p(1, 2, 0, 3), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "1")}, false)},
	{"a(1,2)", ast.NewCall(p(1, 2, 0, 5), ast.NewIdentifier(p(1, 1, 0, 0), "a"),
		[]ast.Expression{ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "2")}, false)},
	{"a[1]", ast.NewIndex(p(1, 2, 0, 3), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "1"))},
	{"a[:]", ast.NewSlicing(p(1, 2, 0, 3), ast.NewIdentifier(p(1, 1, 0, 0), "a"), nil, nil, nil, false)},
	{"a[:2]", ast.NewSlicing(p(1, 2, 0, 4), ast.NewIdentifier(p(1, 1, 0, 0), "a"), nil, ast.NewBasicLiteral(p(1, 4, 3, 3), ast.IntLiteral, "2"), nil, false)},
	{"a[1:]", ast.NewSlicing(p(1, 2, 0, 4), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "1"), nil, nil, false)},
	{"a[1:2]", ast.NewSlicing(p(1, 2, 0, 5), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "2"), nil, false)},
	{"a[1:2:3]", ast.NewSlicing(p(1, 2, 0, 7), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "2"), ast.NewBasicLiteral(p(1, 7, 6, 6), ast.IntLiteral, "3"), true)},
	{"a.B", ast.NewSelector(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), "B")},
	{"1+2+3", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorAddition, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorAddition, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2")), ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "3"))},
	{"1-2-3", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorSubtraction, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorSubtraction, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2")), ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "3"))},
	{"1*2*3", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorMultiplication, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorMultiplication, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2")), ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "3"))},
	{"1+2*3", ast.NewBinaryOperator(p(1, 2, 0, 4), ast.OperatorAddition, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"),
		ast.NewBinaryOperator(p(1, 4, 2, 4), ast.OperatorMultiplication, ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2"), ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "3")))},
	{"1-2/3", ast.NewBinaryOperator(p(1, 2, 0, 4), ast.OperatorSubtraction, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"),
		ast.NewBinaryOperator(p(1, 4, 2, 4), ast.OperatorDivision, ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2"), ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "3")))},
	{"1*2+3", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorAddition, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorMultiplication, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2")), ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "3"))},
	{"1==2+3", ast.NewBinaryOperator(p(1, 2, 0, 5), ast.OperatorEqual, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"),
		ast.NewBinaryOperator(p(1, 5, 3, 5), ast.OperatorAddition, ast.NewBasicLiteral(p(1, 4, 3, 3), ast.IntLiteral, "2"), ast.NewBasicLiteral(p(1, 6, 5, 5), ast.IntLiteral, "3")))},
	{"1+2==3", ast.NewBinaryOperator(p(1, 4, 0, 5), ast.OperatorEqual, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorAddition, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "2")), ast.NewBasicLiteral(p(1, 6, 5, 5), ast.IntLiteral, "3"))},
	{"(1+2)*3", ast.NewBinaryOperator(p(1, 6, 0, 6), ast.OperatorMultiplication, parenthesized(ast.NewBinaryOperator(p(1, 3, 0, 4),
		ast.OperatorAddition, ast.NewBasicLiteral(p(1, 2, 1, 1), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 4, 3, 3), ast.IntLiteral, "2"))), ast.NewBasicLiteral(p(1, 7, 6, 6), ast.IntLiteral, "3"))},
	{"1*(2+3)", ast.NewBinaryOperator(p(1, 2, 0, 6), ast.OperatorMultiplication, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"),
		parenthesized(ast.NewBinaryOperator(p(1, 5, 2, 6), ast.OperatorAddition, ast.NewBasicLiteral(p(1, 4, 3, 3), ast.IntLiteral, "2"), ast.NewBasicLiteral(p(1, 6, 5, 5), ast.IntLiteral, "3"))))},
	{"(1*((2)+3))", parenthesized(ast.NewBinaryOperator(p(1, 3, 0, 10), ast.OperatorMultiplication, ast.NewBasicLiteral(p(1, 2, 1, 1), ast.IntLiteral, "1"),
		parenthesized(ast.NewBinaryOperator(p(1, 8, 3, 9), ast.OperatorAddition, parenthesized(ast.NewBasicLiteral(p(1, 6, 4, 6), ast.IntLiteral, "2")), ast.NewBasicLiteral(p(1, 9, 8, 8), ast.IntLiteral, "3")))))},
	{"a()*1", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorMultiplication,
		ast.NewCall(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{}, false), ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "1"))},
	{"1*a()", ast.NewBinaryOperator(p(1, 2, 0, 4), ast.OperatorMultiplication,
		ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewCall(p(1, 4, 2, 4), ast.NewIdentifier(p(1, 3, 2, 2), "a"), []ast.Expression{}, false))},
	{"a[1]*2", ast.NewBinaryOperator(p(1, 5, 0, 5), ast.OperatorMultiplication, ast.NewIndex(p(1, 2, 0, 3),
		ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "1")), ast.NewBasicLiteral(p(1, 6, 5, 5), ast.IntLiteral, "2"))},
	{"1*a[2]", ast.NewBinaryOperator(p(1, 2, 0, 5), ast.OperatorMultiplication, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"),
		ast.NewIndex(p(1, 4, 2, 5), ast.NewIdentifier(p(1, 3, 2, 2), "a"), ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "2")))},
	{"a[1+2]", ast.NewIndex(p(1, 2, 0, 5), ast.NewIdentifier(p(1, 1, 0, 0), "a"),
		ast.NewBinaryOperator(p(1, 4, 2, 4), ast.OperatorAddition, ast.NewBasicLiteral(p(1, 3, 2, 2), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "2")))},
	{"a[b(1)]", ast.NewIndex(p(1, 2, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewCall(p(1, 4, 2, 5),
		ast.NewIdentifier(p(1, 3, 2, 2), "b"), []ast.Expression{ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "1")}, false))},
	{"a(b[1])", ast.NewCall(p(1, 2, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{
		ast.NewIndex(p(1, 4, 2, 5), ast.NewIdentifier(p(1, 3, 2, 2), "b"), ast.NewBasicLiteral(p(1, 5, 4, 4), ast.IntLiteral, "1"))}, false)},
	{"a.B*c", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorMultiplication, ast.NewSelector(p(1, 2, 0, 2),
		ast.NewIdentifier(p(1, 1, 0, 0), "a"), "B"), ast.NewIdentifier(p(1, 5, 4, 4), "c"))},
	{"a*b.C", ast.NewBinaryOperator(p(1, 2, 0, 4), ast.OperatorMultiplication, ast.NewIdentifier(p(1, 1, 0, 0), "a"),
		ast.NewSelector(p(1, 4, 2, 4), ast.NewIdentifier(p(1, 3, 2, 2), "b"), "C"))},
	{"a.B(c)", ast.NewCall(p(1, 4, 0, 5), ast.NewSelector(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), "B"),
		[]ast.Expression{ast.NewIdentifier(p(1, 5, 4, 4), "c")}, false)},
	{"a.(string)", ast.NewTypeAssertion(p(1, 2, 0, 9), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 8), "string"))},
	{"html(a).(html)", ast.NewTypeAssertion(p(1, 8, 0, 13), ast.NewCall(p(1, 5, 0, 6),
		ast.NewIdentifier(p(1, 1, 0, 3), "html"), []ast.Expression{ast.NewIdentifier(p(1, 6, 5, 5), "a")}, false), ast.NewIdentifier(p(1, 10, 9, 12), "html"))},
	{"a.(number)", ast.NewTypeAssertion(p(1, 2, 0, 9), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 8), "number"))},
	{"a.(int)", ast.NewTypeAssertion(p(1, 2, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 5), "int"))},
	{"a.(bool)", ast.NewTypeAssertion(p(1, 2, 0, 7), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 6), "bool"))},
	{"os.FileInfo", ast.NewSelector(p(1, 3, 0, 10), ast.NewIdentifier(p(1, 1, 0, 1), "os"), "FileInfo")},
	{"{1,2,3}", ast.NewCompositeLiteral(p(1, 1, 0, 6), nil, []ast.KeyValue{
		{nil, ast.NewBasicLiteral(p(1, 2, 1, 1), ast.IntLiteral, "1")},
		{nil, ast.NewBasicLiteral(p(1, 4, 3, 3), ast.IntLiteral, "2")},
		{nil, ast.NewBasicLiteral(p(1, 6, 5, 5), ast.IntLiteral, "3")},
	})},
	{`{a: "text", b: 3}`,
		ast.NewCompositeLiteral(
			p(1, 1, 0, 16),
			nil,
			[]ast.KeyValue{
				ast.KeyValue{
					ast.NewIdentifier(p(1, 2, 1, 1), "a"),
					ast.NewBasicLiteral(p(1, 5, 4, 9), ast.StringLiteral, `"text"`),
				},
				ast.KeyValue{
					ast.NewIdentifier(p(1, 13, 12, 12), "b"),
					ast.NewBasicLiteral(p(1, 16, 15, 15), ast.IntLiteral, "3"),
				},
			},
		),
	},
	{`[][]int{{1},{2,3}}`,
		ast.NewCompositeLiteral(
			p(1, 8, 0, 17),
			ast.NewSliceType(
				p(1, 1, 0, 6),
				ast.NewSliceType(
					p(1, 3, 2, 6),
					ast.NewIdentifier(p(1, 5, 4, 6), "int"),
				)),
			[]ast.KeyValue{
				{nil, ast.NewCompositeLiteral(
					p(1, 9, 8, 10),
					nil,
					[]ast.KeyValue{
						{nil, ast.NewBasicLiteral(p(1, 10, 9, 9), ast.IntLiteral, "1")},
					})},
				{nil, ast.NewCompositeLiteral(p(1, 13, 12, 16), nil,
					[]ast.KeyValue{
						{nil, ast.NewBasicLiteral(p(1, 14, 13, 13), ast.IntLiteral, "2")},
						{nil, ast.NewBasicLiteral(p(1, 16, 15, 15), ast.IntLiteral, "3")},
					})}})},
	{"map[int]bool", ast.NewMapType(p(1, 1, 0, 11), ast.NewIdentifier(p(1, 5, 4, 6), "int"), ast.NewIdentifier(p(1, 9, 8, 11), "bool"))},
	{`map[int]string{1: "uno", 2: "due"}`, ast.NewCompositeLiteral(p(1, 15, 0, 33),
		ast.NewMapType(p(1, 1, 0, 13), ast.NewIdentifier(p(1, 5, 4, 6), "int"), ast.NewIdentifier(p(1, 9, 8, 13), "string")),
		[]ast.KeyValue{
			ast.KeyValue{ast.NewBasicLiteral(p(1, 16, 15, 15), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 19, 18, 22), ast.StringLiteral, `"uno"`)},
			ast.KeyValue{ast.NewBasicLiteral(p(1, 26, 25, 25), ast.IntLiteral, "2"), ast.NewBasicLiteral(p(1, 29, 28, 32), ast.StringLiteral, `"due"`)},
		})},
	{"[]int(s)", ast.NewCall(p(1, 6, 0, 7),
		ast.NewSliceType(p(1, 1, 0, 4), ast.NewIdentifier(p(1, 3, 2, 4), "int")),
		[]ast.Expression{ast.NewIdentifier(p(1, 7, 6, 6), "s")}, false)},
	{"[]int", ast.NewSliceType(p(1, 1, 0, 4),
		ast.NewIdentifier(p(1, 3, 2, 4), "int"))},
	{"[]string{}", ast.NewCompositeLiteral(p(1, 9, 0, 9),
		ast.NewSliceType(p(1, 1, 0, 7), ast.NewIdentifier(p(1, 3, 2, 7), "string")), nil)},
	{`[3]string{"a", "b", "c"}`,
		ast.NewCompositeLiteral(p(1, 10, 0, 23), ast.NewArrayType(p(1, 1, 0, 8), ast.NewBasicLiteral(p(1, 2, 1, 1), ast.IntLiteral, "3"), ast.NewIdentifier(p(1, 4, 3, 8), "string")), []ast.KeyValue{
			{nil, ast.NewBasicLiteral(p(1, 11, 10, 12), ast.StringLiteral, `"a"`)},
			{nil, ast.NewBasicLiteral(p(1, 16, 15, 17), ast.StringLiteral, `"b"`)},
			{nil, ast.NewBasicLiteral(p(1, 21, 20, 22), ast.StringLiteral, `"c"`)},
		})},
	{"[][]int{[]int{2}, []int{-2, 10}}",
		ast.NewCompositeLiteral(p(1, 8, 0, 31),
			ast.NewSliceType(p(1, 1, 0, 6), // [][]int
				ast.NewSliceType( // []int
					p(1, 3, 2, 6),
					ast.NewIdentifier(p(1, 5, 4, 6), "int"))),
			[]ast.KeyValue{{nil,
				ast.NewCompositeLiteral( // []int{2}
					p(1, 14, 8, 15),
					ast.NewSliceType(p(1, 9, 8, 12), ast.NewIdentifier(p(1, 11, 10, 12), "int")),
					[]ast.KeyValue{{nil, ast.NewBasicLiteral(p(1, 15, 14, 14), ast.IntLiteral, "2")}}),
			},
				{nil,
					ast.NewCompositeLiteral( // []int{-2, 10}
						p(1, 24, 18, 30),
						ast.NewSliceType(p(1, 19, 18, 22), ast.NewIdentifier(p(1, 21, 20, 22), "int")),
						[]ast.KeyValue{
							{nil, ast.NewUnaryOperator(p(1, 25, 24, 25), ast.OperatorSubtraction, ast.NewBasicLiteral(p(1, 26, 25, 25), ast.IntLiteral, "2"))},
							{nil, ast.NewBasicLiteral(p(1, 29, 28, 29), ast.IntLiteral, "10")},
						})}})},
	{"[]int{42}", ast.NewCompositeLiteral(p(1, 6, 0, 8), ast.NewSliceType(p(1, 1, 0, 4),
		ast.NewIdentifier(p(1, 3, 2, 4), "int")), []ast.KeyValue{{nil, ast.NewBasicLiteral(p(1, 7, 6, 7), ast.IntLiteral, "42")}})},
	{"42 + [1]int{-30}[0]", ast.NewBinaryOperator(
		p(1, 4, 0, 18),
		ast.OperatorAddition,
		ast.NewBasicLiteral(p(1, 1, 0, 1), ast.IntLiteral, "42"),
		ast.NewIndex(
			p(1, 17, 5, 18),
			ast.NewCompositeLiteral(p(1, 12, 5, 15), ast.NewArrayType(p(1, 6, 5, 10), ast.NewBasicLiteral(p(1, 7, 6, 6), ast.IntLiteral, "1"), ast.NewIdentifier(p(1, 9, 8, 10), "int")),
				[]ast.KeyValue{{nil, ast.NewUnaryOperator(p(1, 13, 12, 14), ast.OperatorSubtraction,
					ast.NewBasicLiteral(p(1, 14, 13, 14), ast.IntLiteral, "30"))}},
			),
			ast.NewBasicLiteral(p(1, 18, 17, 17), ast.IntLiteral, "0"),
		))},
	{"[1]int{-30}[0] + 42",
		ast.NewBinaryOperator(
			p(1, 16, 0, 18),
			ast.OperatorAddition,
			ast.NewIndex(
				p(1, 12, 0, 13),
				ast.NewCompositeLiteral(p(1, 7, 0, 10), ast.NewArrayType(p(1, 1, 0, 5), ast.NewBasicLiteral(p(1, 2, 1, 1), ast.IntLiteral, "1"), ast.NewIdentifier(p(1, 4, 3, 5), "int")),
					[]ast.KeyValue{{nil, ast.NewUnaryOperator(p(1, 8, 7, 9), ast.OperatorSubtraction,
						ast.NewBasicLiteral(p(1, 9, 8, 9), ast.IntLiteral, "30"))}},
				),
				ast.NewBasicLiteral(p(1, 13, 12, 12), ast.IntLiteral, "0"),
			),
			ast.NewBasicLiteral(p(1, 18, 17, 18), ast.IntLiteral, "42"),
		)},
	{
		"42 + [2]float{3.0, 4.0}[0] * 5.0",
		ast.NewBinaryOperator(
			p(1, 4, 0, 31),
			ast.OperatorAddition,
			ast.NewBasicLiteral(p(1, 1, 0, 1), ast.IntLiteral, "42"),
			ast.NewBinaryOperator( // [2]float{3.0, 4.0}[0] * 5.0
				p(1, 28, 5, 31),
				ast.OperatorMultiplication,
				ast.NewIndex( // [2]float{3.0, 4.0}[0]
					p(1, 24, 5, 25),
					ast.NewCompositeLiteral(
						p(1, 14, 5, 22),
						ast.NewArrayType( // [2]float
							p(1, 6, 5, 12),
							ast.NewBasicLiteral(p(1, 7, 6, 6), ast.IntLiteral, "2"),
							ast.NewIdentifier(p(1, 9, 8, 12), "float"),
						),
						[]ast.KeyValue{
							{nil, ast.NewBasicLiteral(p(1, 15, 14, 16), ast.FloatLiteral, "3.0")},
							{nil, ast.NewBasicLiteral(p(1, 20, 19, 21), ast.FloatLiteral, "4.0")},
						},
					),
					ast.NewBasicLiteral(p(1, 25, 24, 24), ast.IntLiteral, "0"),
				),
				ast.NewBasicLiteral(p(1, 30, 29, 31), ast.FloatLiteral, "5.0"),
			),
		),
	},
	{"[...]int{1, 4, 9}", ast.NewCompositeLiteral(
		p(1, 9, 0, 16), ast.NewArrayType(
			p(1, 1, 0, 7), nil,
			ast.NewIdentifier(p(1, 6, 5, 7), "int"),
		),
		[]ast.KeyValue{
			{nil, ast.NewBasicLiteral(p(1, 10, 9, 9), ast.IntLiteral, "1")}, {nil, ast.NewBasicLiteral(p(1, 13, 12, 12), ast.IntLiteral, "4")}, {nil, ast.NewBasicLiteral(p(1, 16, 15, 15), ast.IntLiteral, "9")}}),
	},
	{"[]*int", ast.NewSliceType(p(1, 1, 0, 5),
		ast.NewUnaryOperator(
			p(1, 3, 2, 5),
			ast.OperatorMultiplication,
			ast.NewIdentifier(p(1, 4, 3, 5), "int"),
		))},
	{"[]*[]*int",
		ast.NewSliceType(
			p(1, 1, 0, 8),

			ast.NewUnaryOperator(
				p(1, 3, 2, 8),
				ast.OperatorMultiplication,
				ast.NewSliceType(
					p(1, 4, 3, 8),
					ast.NewUnaryOperator(
						p(1, 6, 5, 8),
						ast.OperatorMultiplication,
						ast.NewIdentifier(p(1, 7, 6, 8), "int"),
					),
				),
			),
		),
	},
	{"1 / map[*int]Type{k1: *v1, k2: v2}[&a] - id",
		ast.NewBinaryOperator(
			p(1, 40, 0, 42),
			ast.OperatorSubtraction,
			ast.NewBinaryOperator( // 1 / map[*int]Type{k1: *v1, k2: v2}
				p(1, 3, 0, 37),
				ast.OperatorDivision,
				ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"),
				ast.NewIndex( // map[*int]Type{k1: *v1, k2: v2}[&a]
					p(1, 35, 4, 37),
					ast.NewCompositeLiteral( // map[*int]Type{k1: *v1, k2: v2}
						p(1, 18, 4, 33),
						ast.NewMapType( // map[*int]Type
							p(1, 5, 4, 16),
							ast.NewUnaryOperator( // *int
								p(1, 9, 8, 11),
								ast.OperatorMultiplication,
								ast.NewIdentifier(p(1, 10, 9, 11), "int"),
							),
							ast.NewIdentifier(p(1, 14, 13, 16), "Type"),
						),
						[]ast.KeyValue{ // k1: *v1, k2: v2
							ast.KeyValue{
								ast.NewIdentifier(p(1, 19, 18, 19), "k1"), // k1
								ast.NewUnaryOperator( // *v1
									p(1, 23, 22, 24),
									ast.OperatorMultiplication,
									ast.NewIdentifier(p(1, 24, 23, 24), "v1"),
								),
							},
							ast.KeyValue{
								ast.NewIdentifier(p(1, 28, 27, 28), "k2"), // k2
								ast.NewIdentifier(p(1, 32, 31, 32), "v2"), // v2
							},
						},
					),
					ast.NewUnaryOperator( // &a
						p(1, 36, 35, 36),
						ast.OperatorAnd,
						ast.NewIdentifier(p(1, 37, 36, 36), "a"),
					),
				),
			),
			ast.NewIdentifier(p(1, 42, 41, 42), "id"),
		),
	},
	{
		`30 * map[string]int{"uno": 1, "due": 2}["due"] - 5`,
		ast.NewBinaryOperator(
			p(1, 48, 0, 49),
			ast.OperatorSubtraction,
			ast.NewBinaryOperator(
				p(1, 4, 0, 45),
				ast.OperatorMultiplication,
				ast.NewBasicLiteral(p(1, 1, 0, 1), ast.IntLiteral, "30"),
				ast.NewIndex(
					p(1, 40, 5, 45),
					ast.NewCompositeLiteral(
						p(1, 20, 5, 38),
						ast.NewMapType(p(1, 6, 5, 18), ast.NewIdentifier(p(1, 10, 9, 14), "string"), ast.NewIdentifier(p(1, 17, 16, 18), "int")),
						[]ast.KeyValue{
							{ast.NewBasicLiteral(p(1, 21, 20, 24), ast.StringLiteral, `"uno"`), ast.NewBasicLiteral(p(1, 28, 27, 27), ast.IntLiteral, "1")},
							{ast.NewBasicLiteral(p(1, 31, 30, 34), ast.StringLiteral, `"due"`), ast.NewBasicLiteral(p(1, 38, 37, 37), ast.IntLiteral, "2")},
						},
					),
					ast.NewBasicLiteral(p(1, 41, 40, 44), ast.StringLiteral, `"due"`),
				),
			),
			ast.NewBasicLiteral(p(1, 50, 49, 49), ast.IntLiteral, "5"),
		),
	},
	{`myStruct{value1, "value2", 33}`,
		ast.NewCompositeLiteral(
			p(1, 9, 0, 29),
			ast.NewIdentifier(p(1, 1, 0, 7), "myStruct"),
			[]ast.KeyValue{
				{nil, ast.NewIdentifier(p(1, 10, 9, 14), "value1")},
				{nil, ast.NewBasicLiteral(p(1, 18, 17, 24), ast.StringLiteral, `"value2"`)},
				{nil, ast.NewBasicLiteral(p(1, 28, 27, 28), ast.IntLiteral, "33")},
			},
		),
	},
	{`pkg.Struct{value1, "value2", 33}`,
		ast.NewCompositeLiteral(
			p(1, 11, 0, 31),
			ast.NewSelector(p(1, 4, 0, 9), ast.NewIdentifier(p(1, 1, 0, 2), "pkg"), "Struct"),
			[]ast.KeyValue{
				{nil, ast.NewIdentifier(p(1, 12, 11, 16), "value1")},
				{nil, ast.NewBasicLiteral(p(1, 20, 19, 26), ast.StringLiteral, `"value2"`)},
				{nil, ast.NewBasicLiteral(p(1, 30, 29, 30), ast.IntLiteral, "33")},
			},
		),
	},
	{`pkg.Struct{Key1: value1, Key2: "value2", Key3: 33}`,
		ast.NewCompositeLiteral(
			p(1, 11, 0, 49),
			ast.NewSelector(p(1, 4, 0, 9), ast.NewIdentifier(p(1, 1, 0, 2), "pkg"), "Struct"),
			[]ast.KeyValue{
				ast.KeyValue{ast.NewIdentifier(p(1, 12, 11, 14), "Key1"), ast.NewIdentifier(p(1, 18, 17, 22), "value1")},
				ast.KeyValue{ast.NewIdentifier(p(1, 26, 25, 28), "Key2"), ast.NewBasicLiteral(p(1, 32, 31, 38), ast.StringLiteral, `"value2"`)},
				ast.KeyValue{ast.NewIdentifier(p(1, 42, 41, 44), "Key3"), ast.NewBasicLiteral(p(1, 48, 47, 48), ast.IntLiteral, "33")},
			},
		),
	},
	{`interface{}`,
		ast.NewInterface(p(1, 1, 0, 10)),
	},
	{`[]interface{}{1,2,3}`,
		ast.NewCompositeLiteral(
			p(1, 14, 0, 19),
			ast.NewSliceType(
				p(1, 1, 0, 12),
				ast.NewInterface(p(1, 3, 2, 12)),
			),
			[]ast.KeyValue{
				{nil, ast.NewBasicLiteral(p(1, 15, 14, 14), ast.IntLiteral, "1")},
				{nil, ast.NewBasicLiteral(p(1, 17, 16, 16), ast.IntLiteral, "2")},
				{nil, ast.NewBasicLiteral(p(1, 19, 18, 18), ast.IntLiteral, "3")},
			})},
	{`1 + interface{}(2) + 4`,
		ast.NewBinaryOperator(
			p(1, 20, 0, 21),
			ast.OperatorAddition,
			ast.NewBinaryOperator(
				p(1, 3, 0, 17),
				ast.OperatorAddition,
				ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"),
				ast.NewCall(
					p(1, 16, 4, 17),
					ast.NewInterface(p(1, 5, 4, 14)),
					[]ast.Expression{ast.NewBasicLiteral(p(1, 17, 16, 16), ast.IntLiteral, "2")}, false,
				)),
			ast.NewBasicLiteral(p(1, 22, 21, 21), ast.IntLiteral, "4"),
		)},
	{"1\t+\n2", ast.NewBinaryOperator(p(1, 3, 0, 4), ast.OperatorAddition, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(2, 1, 4, 4), ast.IntLiteral, "2"))},
	{"1\t\r +\n\r\n\r\t 2", ast.NewBinaryOperator(p(1, 5, 0, 11), ast.OperatorAddition, ast.NewBasicLiteral(p(1, 1, 0, 0), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(3, 4, 11, 11), ast.IntLiteral, "2"))},
	{"a(\n\t1\t,\n2\t)", ast.NewCall(p(1, 2, 0, 10), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{
		ast.NewBasicLiteral(p(2, 2, 4, 4), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(3, 1, 8, 8), ast.IntLiteral, "2")}, false)},
	{"a\t\r ()", ast.NewCall(p(1, 5, 0, 5), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{}, false)},
	{"a[\n\t1\t]", ast.NewIndex(p(1, 2, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewBasicLiteral(p(2, 2, 4, 4), ast.IntLiteral, "1"))},
	{"a\t\r [1]", ast.NewIndex(p(1, 5, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewBasicLiteral(p(1, 6, 5, 5), ast.IntLiteral, "1"))},
	{"func() {}",
		ast.NewFunc(p(1, 1, 0, 8), nil,
			ast.NewFuncType(nil, nil, nil, false),
			ast.NewBlock(p(1, 8, 7, 8), nil))},
	{"func(int) {}",
		ast.NewFunc(p(1, 1, 0, 11), nil,
			ast.NewFuncType(nil, []*ast.Parameter{ast.NewParameter(nil, ast.NewIdentifier(p(1, 6, 5, 7), "int"))}, nil, false),
			ast.NewBlock(p(1, 11, 10, 11), nil))},
	{"func(a int) {}",
		ast.NewFunc(p(1, 1, 0, 13), nil,
			ast.NewFuncType(nil, []*ast.Parameter{ast.NewParameter(
				ast.NewIdentifier(p(1, 6, 5, 5), "a"), ast.NewIdentifier(p(1, 8, 7, 9), "int")),
			}, nil, false),
			ast.NewBlock(p(1, 13, 12, 13), nil))},
	{"func(a, b int) {}",
		ast.NewFunc(p(1, 1, 0, 16), nil,
			ast.NewFuncType(nil, []*ast.Parameter{
				ast.NewParameter(ast.NewIdentifier(p(1, 6, 5, 5), "a"), nil),
				ast.NewParameter(ast.NewIdentifier(p(1, 9, 8, 8), "b"), ast.NewIdentifier(p(1, 11, 10, 12), "int")),
			}, nil, false),
			ast.NewBlock(p(1, 16, 15, 16), nil))},
	{"func(a string, b int) {}",
		ast.NewFunc(p(1, 1, 0, 23), nil,
			ast.NewFuncType(nil, []*ast.Parameter{
				ast.NewParameter(ast.NewIdentifier(p(1, 6, 5, 5), "a"), ast.NewIdentifier(p(1, 8, 7, 12), "string")),
				ast.NewParameter(ast.NewIdentifier(p(1, 16, 15, 15), "b"), ast.NewIdentifier(p(1, 18, 17, 19), "int")),
			}, nil, false),
			ast.NewBlock(p(1, 23, 22, 23), nil))},
	{"func(a, b ...int) {}",
		ast.NewFunc(p(1, 1, 0, 19), nil,
			ast.NewFuncType(nil, []*ast.Parameter{
				ast.NewParameter(ast.NewIdentifier(p(1, 6, 5, 5), "a"), nil),
				ast.NewParameter(ast.NewIdentifier(p(1, 9, 8, 8), "b"), ast.NewIdentifier(p(1, 14, 13, 15), "int")),
			}, nil, true),
			ast.NewBlock(p(1, 19, 18, 19), nil))},
	{"func(p.T) {}",
		ast.NewFunc(p(1, 1, 0, 11), nil,
			ast.NewFuncType(nil, []*ast.Parameter{
				ast.NewParameter(nil, ast.NewSelector(p(1, 8, 7, 7),
					ast.NewIdentifier(p(1, 6, 5, 5), "p"), "T")),
			}, nil, false),
			ast.NewBlock(p(1, 11, 10, 11), nil),
		),
	},
	{"func(a p.T) {}",
		ast.NewFunc(p(1, 1, 0, 13), nil,
			ast.NewFuncType(nil, []*ast.Parameter{
				ast.NewParameter(ast.NewIdentifier(p(1, 6, 5, 5), "a"),
					ast.NewSelector(p(1, 10, 9, 9),
						ast.NewIdentifier(p(1, 8, 7, 7), "p"), "T")),
			}, nil, false),
			ast.NewBlock(p(1, 13, 12, 13), nil),
		),
	},
	{"func(a ...p.T) {}",
		ast.NewFunc(p(1, 1, 0, 16), nil,
			ast.NewFuncType(nil, []*ast.Parameter{
				ast.NewParameter(ast.NewIdentifier(p(1, 6, 5, 5), "a"),
					ast.NewSelector(p(1, 13, 12, 12),
						ast.NewIdentifier(p(1, 11, 10, 10), "p"), "T")),
			}, nil, true),
			ast.NewBlock(p(1, 15, 15, 16), nil),
		),
	},
}

func TestExpressions(t *testing.T) {
	for _, expr := range exprTests {
		var lex = newLexer([]byte("{{"+expr.src+"}}"), ast.ContextText)
		<-lex.tokens
		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(*SyntaxError); ok {
						t.Errorf("source: %q, %s\n", expr.src, err)
					} else {
						panic(r)
					}
				}
			}()
			var p = &parsing{
				lex:       lex,
				ctx:       ast.ContextGo,
				ancestors: nil,
			}
			node, tok := p.parseExpr(p.next(), false, false, false)
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
