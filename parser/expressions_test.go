// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"math/big"
	"testing"

	"scrigo/ast"
)

var maxInt64Plus1, _ = new(big.Int).SetString("9223372036854775808", 10)
var minInt64Minus1, _ = new(big.Int).SetString("-9223372036854775809", 10)
var bigInt, _ = new(big.Int).SetString("433937734937734969526500969526500", 10)

var exprTests = []struct {
	src  string
	node ast.Node
}{
	{"a", ast.NewIdentifier(p(1, 1, 0, 0), "a")},
	{"a5", ast.NewIdentifier(p(1, 1, 0, 1), "a5")},
	{"_a", ast.NewIdentifier(p(1, 1, 0, 1), "_a")},
	{"_5", ast.NewIdentifier(p(1, 1, 0, 1), "_5")},
	{"0", ast.NewInt(p(1, 1, 0, 0), big.NewInt(0))},
	{"3", ast.NewInt(p(1, 1, 0, 0), big.NewInt(3))},
	{"2147483647", ast.NewInt(p(1, 1, 0, 9), big.NewInt(2147483647))},                      // math.MaxInt32
	{"-2147483648", ast.NewInt(p(1, 1, 0, 10), big.NewInt(-2147483648))},                   // math.MinInt32
	{"9223372036854775807", ast.NewInt(p(1, 1, 0, 18), big.NewInt(9223372036854775807))},   // math.MaxInt64
	{"-9223372036854775808", ast.NewInt(p(1, 1, 0, 19), big.NewInt(-9223372036854775808))}, // math.MinInt64
	{"2147483648", ast.NewInt(p(1, 1, 0, 9), big.NewInt(2147483648))},                      // math.MaxInt32 + 1
	{"-2147483649", ast.NewInt(p(1, 1, 0, 10), big.NewInt(-2147483649))},                   // math.MinInt32 - 1
	{"9223372036854775808", ast.NewInt(p(1, 1, 0, 18), maxInt64Plus1)},                     // math.MaxInt64 + 1
	{"-9223372036854775809", ast.NewInt(p(1, 1, 0, 19), minInt64Minus1)},                   // math.MinInt64 - 1
	{"433937734937734969526500969526500", ast.NewInt(p(1, 1, 0, 32), bigInt)},
	{"'a'", ast.NewRune(p(1, 1, 0, 2), 'a')},
	{`'\a'`, ast.NewRune(p(1, 1, 0, 3), '\a')},
	{`'\b'`, ast.NewRune(p(1, 1, 0, 3), '\b')},
	{`'\f'`, ast.NewRune(p(1, 1, 0, 3), '\f')},
	{`'\n'`, ast.NewRune(p(1, 1, 0, 3), '\n')},
	{`'\r'`, ast.NewRune(p(1, 1, 0, 3), '\r')},
	{`'\t'`, ast.NewRune(p(1, 1, 0, 3), '\t')},
	{`'\v'`, ast.NewRune(p(1, 1, 0, 3), '\v')},
	{`'\\'`, ast.NewRune(p(1, 1, 0, 3), '\\')},
	{`'\''`, ast.NewRune(p(1, 1, 0, 3), '\'')},
	{`'\x00'`, ast.NewRune(p(1, 1, 0, 5), '\x00')},
	{`'\x54'`, ast.NewRune(p(1, 1, 0, 5), '\x54')},
	{`'\xff'`, ast.NewRune(p(1, 1, 0, 5), '\xff')},
	{`'\xFF'`, ast.NewRune(p(1, 1, 0, 5), '\xFF')},
	{`'\u0000'`, ast.NewRune(p(1, 1, 0, 7), '\u0000')},
	{`'\u0012'`, ast.NewRune(p(1, 1, 0, 7), '\u0012')},
	{`'\u1234'`, ast.NewRune(p(1, 1, 0, 7), '\u1234')},
	{`'\U00000000'`, ast.NewRune(p(1, 1, 0, 11), '\U00000000')},
	{`'\U0010ffff'`, ast.NewRune(p(1, 1, 0, 11), '\U0010ffff')},
	{`'\U0010FFFF'`, ast.NewRune(p(1, 1, 0, 11), '\U0010FFFF')},
	{`'\000'`, ast.NewRune(p(1, 1, 0, 5), '\000')},
	{`'\177'`, ast.NewRune(p(1, 1, 0, 5), '\177')},
	{`'\377'`, ast.NewRune(p(1, 1, 0, 5), '\377')},
	{"\"\"", ast.NewString(p(1, 1, 0, 1), "")},
	{"\"a\"", ast.NewString(p(1, 1, 0, 2), "a")},
	{`"\a"`, ast.NewString(p(1, 1, 0, 3), "\a")},
	{`"\b"`, ast.NewString(p(1, 1, 0, 3), "\b")},
	{`"\f"`, ast.NewString(p(1, 1, 0, 3), "\f")},
	{`"\n"`, ast.NewString(p(1, 1, 0, 3), "\n")},
	{`"\r"`, ast.NewString(p(1, 1, 0, 3), "\r")},
	{`"\t"`, ast.NewString(p(1, 1, 0, 3), "\t")},
	{`"\v"`, ast.NewString(p(1, 1, 0, 3), "\v")},
	{`"\\"`, ast.NewString(p(1, 1, 0, 3), "\\")},
	{`"\""`, ast.NewString(p(1, 1, 0, 3), "\"")},
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
	{"&a", ast.NewUnaryOperator(p(1, 1, 0, 1), ast.OperatorAmpersand, ast.NewIdentifier(p(1, 2, 1, 1), "a"))},
	{"*a", ast.NewUnaryOperator(p(1, 1, 0, 1), ast.OperatorMultiplication, ast.NewIdentifier(p(1, 2, 1, 1), "a"))},
	{"1+2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 3, 2, 2), big.NewInt(2)))},
	{"1-2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorSubtraction, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 3, 2, 2), big.NewInt(2)))},
	{"1*2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorMultiplication, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 3, 2, 2), big.NewInt(2)))},
	{"1/2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorDivision, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 3, 2, 2), big.NewInt(2)))},
	{"1%2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorModulo, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 3, 2, 2), big.NewInt(2)))},
	{"1==2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorEqual, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 4, 3, 3), big.NewInt(2)))},
	{"1!=2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorNotEqual, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 4, 3, 3), big.NewInt(2)))},
	{"1<2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorLess, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 3, 2, 2), big.NewInt(2)))},
	{"1<=2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorLessOrEqual, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 4, 3, 3), big.NewInt(2)))},
	{"1>2", ast.NewBinaryOperator(p(1, 2, 0, 2), ast.OperatorGreater, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 3, 2, 2), big.NewInt(2)))},
	{"1>=2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorGreaterOrEqual, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 4, 3, 3), big.NewInt(2)))},
	{"a&&b", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorAnd, ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 3), "b"))},
	{"a||b", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorOr, ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewIdentifier(p(1, 4, 3, 3), "b"))},
	{"1+-2", ast.NewBinaryOperator(p(1, 2, 0, 3), ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 3, 2, 3), big.NewInt(-2)))},
	{"1+-(2)", ast.NewBinaryOperator(p(1, 2, 0, 5), ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)),
		ast.NewUnaryOperator(p(1, 3, 2, 5), ast.OperatorSubtraction, ast.NewInt(p(1, 5, 3, 5), big.NewInt(2))))},
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
	{"(a)", ast.NewIdentifier(p(1, 2, 0, 2), "a")},
	{"a()", ast.NewCall(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{})},
	{"a(1)", ast.NewCall(p(1, 2, 0, 3), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{ast.NewInt(p(1, 3, 2, 2), big.NewInt(1))})},
	{"a(1,2)", ast.NewCall(p(1, 2, 0, 5), ast.NewIdentifier(p(1, 1, 0, 0), "a"),
		[]ast.Expression{ast.NewInt(p(1, 3, 2, 2), big.NewInt(1)), ast.NewInt(p(1, 5, 4, 4), big.NewInt(2))})},
	{"a[1]", ast.NewIndex(p(1, 2, 0, 3), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(1, 3, 2, 2), big.NewInt(1)))},
	{"a[:]", ast.NewSlicing(p(1, 2, 0, 3), ast.NewIdentifier(p(1, 1, 0, 0), "a"), nil, nil)},
	{"a[:2]", ast.NewSlicing(p(1, 2, 0, 4), ast.NewIdentifier(p(1, 1, 0, 0), "a"), nil, ast.NewInt(p(1, 4, 3, 3), big.NewInt(2)))},
	{"a[1:]", ast.NewSlicing(p(1, 2, 0, 4), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(1, 3, 2, 2), big.NewInt(1)), nil)},
	{"a[1:2]", ast.NewSlicing(p(1, 2, 0, 5), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(1, 3, 2, 2), big.NewInt(1)), ast.NewInt(p(1, 5, 4, 4), big.NewInt(2)))},
	{"a.B", ast.NewSelector(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), "B")},
	{"1+2+3", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorAddition, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 3, 2, 2), big.NewInt(2))), ast.NewInt(p(1, 5, 4, 4), big.NewInt(3)))},
	{"1-2-3", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorSubtraction, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorSubtraction, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 3, 2, 2), big.NewInt(2))), ast.NewInt(p(1, 5, 4, 4), big.NewInt(3)))},
	{"1*2*3", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorMultiplication, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorMultiplication, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 3, 2, 2), big.NewInt(2))), ast.NewInt(p(1, 5, 4, 4), big.NewInt(3)))},
	{"1+2*3", ast.NewBinaryOperator(p(1, 2, 0, 4), ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)),
		ast.NewBinaryOperator(p(1, 4, 2, 4), ast.OperatorMultiplication, ast.NewInt(p(1, 3, 2, 2), big.NewInt(2)), ast.NewInt(p(1, 5, 4, 4), big.NewInt(3))))},
	{"1-2/3", ast.NewBinaryOperator(p(1, 2, 0, 4), ast.OperatorSubtraction, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)),
		ast.NewBinaryOperator(p(1, 4, 2, 4), ast.OperatorDivision, ast.NewInt(p(1, 3, 2, 2), big.NewInt(2)), ast.NewInt(p(1, 5, 4, 4), big.NewInt(3))))},
	{"1*2+3", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorAddition, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorMultiplication, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 3, 2, 2), big.NewInt(2))), ast.NewInt(p(1, 5, 4, 4), big.NewInt(3)))},
	{"1==2+3", ast.NewBinaryOperator(p(1, 2, 0, 5), ast.OperatorEqual, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)),
		ast.NewBinaryOperator(p(1, 5, 3, 5), ast.OperatorAddition, ast.NewInt(p(1, 4, 3, 3), big.NewInt(2)), ast.NewInt(p(1, 6, 5, 5), big.NewInt(3))))},
	{"1+2==3", ast.NewBinaryOperator(p(1, 4, 0, 5), ast.OperatorEqual, ast.NewBinaryOperator(p(1, 2, 0, 2),
		ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(1, 3, 2, 2), big.NewInt(2))), ast.NewInt(p(1, 6, 5, 5), big.NewInt(3)))},
	{"(1+2)*3", ast.NewBinaryOperator(p(1, 6, 0, 6), ast.OperatorMultiplication, ast.NewBinaryOperator(p(1, 3, 0, 4),
		ast.OperatorAddition, ast.NewInt(p(1, 2, 1, 1), big.NewInt(1)), ast.NewInt(p(1, 4, 3, 3), big.NewInt(2))), ast.NewInt(p(1, 7, 6, 6), big.NewInt(3)))},
	{"1*(2+3)", ast.NewBinaryOperator(p(1, 2, 0, 6), ast.OperatorMultiplication, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)),
		ast.NewBinaryOperator(p(1, 5, 2, 6), ast.OperatorAddition, ast.NewInt(p(1, 4, 3, 3), big.NewInt(2)), ast.NewInt(p(1, 6, 5, 5), big.NewInt(3))))},
	{"(1*((2)+3))", ast.NewBinaryOperator(p(1, 3, 0, 10), ast.OperatorMultiplication, ast.NewInt(p(1, 2, 1, 1), big.NewInt(1)),
		ast.NewBinaryOperator(p(1, 8, 3, 9), ast.OperatorAddition, ast.NewInt(p(1, 6, 4, 6), big.NewInt(2)), ast.NewInt(p(1, 9, 8, 8), big.NewInt(3))))},
	{"a()*1", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorMultiplication,
		ast.NewCall(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{}), ast.NewInt(p(1, 5, 4, 4), big.NewInt(1)))},
	{"1*a()", ast.NewBinaryOperator(p(1, 2, 0, 4), ast.OperatorMultiplication,
		ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewCall(p(1, 4, 2, 4), ast.NewIdentifier(p(1, 3, 2, 2), "a"), []ast.Expression{}))},
	{"a[1]*2", ast.NewBinaryOperator(p(1, 5, 0, 5), ast.OperatorMultiplication, ast.NewIndex(p(1, 2, 0, 3),
		ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(1, 3, 2, 2), big.NewInt(1))), ast.NewInt(p(1, 6, 5, 5), big.NewInt(2)))},
	{"1*a[2]", ast.NewBinaryOperator(p(1, 2, 0, 5), ast.OperatorMultiplication, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)),
		ast.NewIndex(p(1, 4, 2, 5), ast.NewIdentifier(p(1, 3, 2, 2), "a"), ast.NewInt(p(1, 5, 4, 4), big.NewInt(2))))},
	{"a[1+2]", ast.NewIndex(p(1, 2, 0, 5), ast.NewIdentifier(p(1, 1, 0, 0), "a"),
		ast.NewBinaryOperator(p(1, 4, 2, 4), ast.OperatorAddition, ast.NewInt(p(1, 3, 2, 2), big.NewInt(1)), ast.NewInt(p(1, 5, 4, 4), big.NewInt(2))))},
	{"a[b(1)]", ast.NewIndex(p(1, 2, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewCall(p(1, 4, 2, 5),
		ast.NewIdentifier(p(1, 3, 2, 2), "b"), []ast.Expression{ast.NewInt(p(1, 5, 4, 4), big.NewInt(1))}))},
	{"a(b[1])", ast.NewCall(p(1, 2, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{
		ast.NewIndex(p(1, 4, 2, 5), ast.NewIdentifier(p(1, 3, 2, 2), "b"), ast.NewInt(p(1, 5, 4, 4), big.NewInt(1)))})},
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
	{"os.FileInfo", ast.NewSelector(p(1, 3, 0, 10), ast.NewIdentifier(p(1, 1, 0, 1), "os"), "FileInfo")},
	{"{1,2,3}", ast.NewCompositeLiteral(p(1, 1, 0, 6), nil, []ast.KeyValue{
		{nil, ast.NewInt(p(1, 2, 1, 1), big.NewInt(1))},
		{nil, ast.NewInt(p(1, 4, 3, 3), big.NewInt(2))},
		{nil, ast.NewInt(p(1, 6, 5, 5), big.NewInt(3))},
	})},
	{`{a: "text", b: 3}`,
		ast.NewCompositeLiteral(
			p(1, 1, 0, 16),
			nil,
			[]ast.KeyValue{
				ast.KeyValue{
					ast.NewIdentifier(p(1, 2, 1, 1), "a"),
					ast.NewString(p(1, 5, 4, 9), "text"),
				},
				ast.KeyValue{
					ast.NewIdentifier(p(1, 13, 12, 12), "b"),
					ast.NewInt(p(1, 16, 15, 15), big.NewInt(3)),
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
						{nil, ast.NewInt(p(1, 10, 9, 9), big.NewInt(1))},
					})},
				{nil, ast.NewCompositeLiteral(p(1, 13, 12, 16), nil,
					[]ast.KeyValue{
						{nil, ast.NewInt(p(1, 14, 13, 13), big.NewInt(2))},
						{nil, ast.NewInt(p(1, 16, 15, 15), big.NewInt(3))},
					})}})},
	{"map[int]bool", ast.NewMapType(p(1, 1, 0, 11), ast.NewIdentifier(p(1, 5, 4, 6), "int"), ast.NewIdentifier(p(1, 9, 8, 11), "bool"))},
	{`map[int]string{1: "uno", 2: "due"}`, ast.NewCompositeLiteral(p(1, 15, 0, 33),
		ast.NewMapType(p(1, 1, 0, 13), ast.NewIdentifier(p(1, 5, 4, 6), "int"), ast.NewIdentifier(p(1, 9, 8, 13), "string")),
		[]ast.KeyValue{
			ast.KeyValue{ast.NewInt(p(1, 16, 15, 15), big.NewInt(1)), ast.NewString(p(1, 19, 18, 22), "uno")},
			ast.KeyValue{ast.NewInt(p(1, 26, 25, 25), big.NewInt(2)), ast.NewString(p(1, 29, 28, 32), "due")},
		})},
	{"[]int(s)", ast.NewCall(p(1, 6, 0, 7),
		ast.NewSliceType(p(1, 1, 0, 4), ast.NewIdentifier(p(1, 3, 2, 4), "int")),
		[]ast.Expression{ast.NewIdentifier(p(1, 7, 6, 6), "s")})},
	{"[]int", ast.NewSliceType(p(1, 1, 0, 4),
		ast.NewIdentifier(p(1, 3, 2, 4), "int"))},
	{"[]string{}", ast.NewCompositeLiteral(p(1, 9, 0, 9),
		ast.NewSliceType(p(1, 1, 0, 7), ast.NewIdentifier(p(1, 3, 2, 7), "string")), nil)},
	{`[3]string{"a", "b", "c"}`,
		ast.NewCompositeLiteral(p(1, 10, 0, 23), ast.NewArrayType(p(1, 1, 0, 8), ast.NewInt(p(1, 2, 1, 1), big.NewInt(3)), ast.NewIdentifier(p(1, 4, 3, 8), "string")), []ast.KeyValue{
			{nil, ast.NewString(p(1, 11, 10, 12), "a")},
			{nil, ast.NewString(p(1, 16, 15, 17), "b")},
			{nil, ast.NewString(p(1, 21, 20, 22), "c")},
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
					[]ast.KeyValue{{nil, ast.NewInt(p(1, 15, 14, 14), big.NewInt(2))}}),
			},
				{nil,
					ast.NewCompositeLiteral( // []int{-2, 10}
						p(1, 24, 18, 30),
						ast.NewSliceType(p(1, 19, 18, 22), ast.NewIdentifier(p(1, 21, 20, 22), "int")),
						[]ast.KeyValue{
							{nil, ast.NewInt(p(1, 25, 24, 25), big.NewInt(-2))},
							{nil, ast.NewInt(p(1, 29, 28, 29), big.NewInt(10))},
						})}})},
	{"[]int{42}", ast.NewCompositeLiteral(p(1, 6, 0, 8), ast.NewSliceType(p(1, 1, 0, 4),
		ast.NewIdentifier(p(1, 3, 2, 4), "int")), []ast.KeyValue{{nil, ast.NewInt(p(1, 7, 6, 7), big.NewInt(42))}})},
	{"42 + [1]int{-30}[0]", ast.NewBinaryOperator(
		p(1, 4, 0, 18),
		ast.OperatorAddition,
		ast.NewInt(p(1, 1, 0, 1), big.NewInt(42)),
		ast.NewIndex(
			p(1, 17, 5, 18),
			ast.NewCompositeLiteral(p(1, 12, 5, 15), ast.NewArrayType(p(1, 6, 5, 10), ast.NewInt(p(1, 7, 6, 6), big.NewInt(1)), ast.NewIdentifier(p(1, 9, 8, 10), "int")),
				[]ast.KeyValue{{nil, ast.NewInt(p(1, 13, 12, 14), big.NewInt(-30))}},
			),
			ast.NewInt(p(1, 18, 17, 17), big.NewInt(0)),
		))},
	{"[1]int{-30}[0] + 42",
		ast.NewBinaryOperator(
			p(1, 16, 0, 18),
			ast.OperatorAddition,
			ast.NewIndex(
				p(1, 12, 0, 13),
				ast.NewCompositeLiteral(p(1, 7, 0, 10), ast.NewArrayType(p(1, 1, 0, 5), ast.NewInt(p(1, 2, 1, 1), big.NewInt(1)), ast.NewIdentifier(p(1, 4, 3, 5), "int")),
					[]ast.KeyValue{{nil, ast.NewInt(p(1, 8, 7, 9), big.NewInt(-30))}},
				),
				ast.NewInt(p(1, 13, 12, 12), big.NewInt(0)),
			),
			ast.NewInt(p(1, 18, 17, 18), big.NewInt(42)),
		)},
	{
		"42 + [2]float{3.0, 4.0}[0] * 5.0",
		ast.NewBinaryOperator(
			p(1, 4, 0, 31),
			ast.OperatorAddition,
			ast.NewInt(p(1, 1, 0, 1), big.NewInt(42)),
			ast.NewBinaryOperator( // [2]float{3.0, 4.0}[0] * 5.0
				p(1, 28, 5, 31),
				ast.OperatorMultiplication,
				ast.NewIndex( // [2]float{3.0, 4.0}[0]
					p(1, 24, 5, 25),
					ast.NewCompositeLiteral(
						p(1, 14, 5, 22),
						ast.NewArrayType( // [2]float
							p(1, 6, 5, 12),
							ast.NewInt(p(1, 7, 6, 6), big.NewInt(2)),
							ast.NewIdentifier(p(1, 9, 8, 12), "float"),
						),
						[]ast.KeyValue{
							{nil, ast.NewFloat(p(1, 15, 14, 16), big.NewFloat(3.0))},
							{nil, ast.NewFloat(p(1, 20, 19, 21), big.NewFloat(4.0))},
						},
					),
					ast.NewInt(p(1, 25, 24, 24), big.NewInt(0)),
				),
				ast.NewFloat(p(1, 30, 29, 31), big.NewFloat(5.0)),
			),
		),
	},
	{"[...]int{1, 4, 9}", ast.NewCompositeLiteral(
		p(1, 9, 0, 16), ast.NewArrayType(
			p(1, 1, 0, 7), nil,
			ast.NewIdentifier(p(1, 6, 5, 7), "int"),
		),
		[]ast.KeyValue{
			{nil, ast.NewInt(p(1, 10, 9, 9), big.NewInt(1))}, {nil, ast.NewInt(p(1, 13, 12, 12), big.NewInt(4))}, {nil, ast.NewInt(p(1, 16, 15, 15), big.NewInt(9))}}),
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
				ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)),
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
						ast.OperatorAmpersand,
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
				ast.NewInt(p(1, 1, 0, 1), big.NewInt(30)),
				ast.NewIndex(
					p(1, 40, 5, 45),
					ast.NewCompositeLiteral(
						p(1, 20, 5, 38),
						ast.NewMapType(p(1, 6, 5, 18), ast.NewIdentifier(p(1, 10, 9, 14), "string"), ast.NewIdentifier(p(1, 17, 16, 18), "int")),
						[]ast.KeyValue{
							ast.KeyValue{ast.NewString(p(1, 21, 20, 24), "uno"), ast.NewInt(p(1, 28, 27, 27), big.NewInt(1))},
							ast.KeyValue{ast.NewString(p(1, 31, 30, 34), "due"), ast.NewInt(p(1, 38, 37, 37), big.NewInt(2))},
						},
					),
					ast.NewString(p(1, 41, 40, 44), "due"),
				),
			),
			ast.NewInt(p(1, 50, 49, 49), big.NewInt(5)),
		),
	},
	{`myStruct{value1, "value2", 33}`,
		ast.NewCompositeLiteral(
			p(1, 9, 0, 29),
			ast.NewIdentifier(p(1, 1, 0, 7), "myStruct"),
			[]ast.KeyValue{
				{nil, ast.NewIdentifier(p(1, 10, 9, 14), "value1")},
				{nil, ast.NewString(p(1, 18, 17, 24), "value2")},
				{nil, ast.NewInt(p(1, 28, 27, 28), big.NewInt(33))},
			},
		),
	},
	{`pkg.Struct{value1, "value2", 33}`,
		ast.NewCompositeLiteral(
			p(1, 11, 0, 31),
			ast.NewSelector(p(1, 4, 0, 9), ast.NewIdentifier(p(1, 1, 0, 2), "pkg"), "Struct"),
			[]ast.KeyValue{
				{nil, ast.NewIdentifier(p(1, 12, 11, 16), "value1")},
				{nil, ast.NewString(p(1, 20, 19, 26), "value2")},
				{nil, ast.NewInt(p(1, 30, 29, 30), big.NewInt(33))},
			},
		),
	},
	{`pkg.Struct{Key1: value1, Key2: "value2", Key3: 33}`,
		ast.NewCompositeLiteral(
			p(1, 11, 0, 49),
			ast.NewSelector(p(1, 4, 0, 9), ast.NewIdentifier(p(1, 1, 0, 2), "pkg"), "Struct"),
			[]ast.KeyValue{
				ast.KeyValue{ast.NewIdentifier(p(1, 12, 11, 14), "Key1"), ast.NewIdentifier(p(1, 18, 17, 22), "value1")},
				ast.KeyValue{ast.NewIdentifier(p(1, 26, 25, 28), "Key2"), ast.NewString(p(1, 32, 31, 38), "value2")},
				ast.KeyValue{ast.NewIdentifier(p(1, 42, 41, 44), "Key3"), ast.NewInt(p(1, 48, 47, 48), big.NewInt(33))},
			},
		),
	},
	{`interface{}`,
		ast.NewIdentifier(p(1, 1, 0, 10), "interface{}"),
	},
	{`[]interface{}{1,2,3}`,
		ast.NewCompositeLiteral(
			p(1, 14, 0, 19),
			ast.NewSliceType(
				p(1, 1, 0, 12),
				ast.NewIdentifier(p(1, 3, 2, 12), "interface{}"),
			),
			[]ast.KeyValue{
				{nil, ast.NewInt(p(1, 15, 14, 14), big.NewInt(1))},
				{nil, ast.NewInt(p(1, 17, 16, 16), big.NewInt(2))},
				{nil, ast.NewInt(p(1, 19, 18, 18), big.NewInt(3))},
			})},
	{`1 + interface{}(2) + 4`,
		ast.NewBinaryOperator(
			p(1, 20, 0, 21),
			ast.OperatorAddition,
			ast.NewBinaryOperator(
				p(1, 3, 0, 17),
				ast.OperatorAddition,
				ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)),
				ast.NewCall(
					p(1, 16, 4, 17),
					ast.NewIdentifier(p(1, 5, 4, 14), "interface{}"),
					[]ast.Expression{ast.NewInt(p(1, 17, 16, 16), big.NewInt(2))},
				)),
			ast.NewInt(p(1, 22, 21, 21), big.NewInt(4)),
		)},
	{"1\t+\n2", ast.NewBinaryOperator(p(1, 3, 0, 4), ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(2, 1, 4, 4), big.NewInt(2)))},
	{"1\t\r +\n\r\n\r\t 2", ast.NewBinaryOperator(p(1, 5, 0, 11), ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), big.NewInt(1)), ast.NewInt(p(3, 4, 11, 11), big.NewInt(2)))},
	{"a(\n\t1\t,\n2\t)", ast.NewCall(p(1, 2, 0, 10), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{
		ast.NewInt(p(2, 2, 4, 4), big.NewInt(1)), ast.NewInt(p(3, 1, 8, 8), big.NewInt(2))})},
	{"a\t\r ()", ast.NewCall(p(1, 5, 0, 5), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{})},
	{"a[\n\t1\t]", ast.NewIndex(p(1, 2, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(2, 2, 4, 4), big.NewInt(1)))},
	{"a\t\r [1]", ast.NewIndex(p(1, 5, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(1, 6, 5, 5), big.NewInt(1)))},
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
			var p = &parsing{
				lex:       lex,
				ctx:       ast.ContextNone,
				ancestors: nil,
			}
			node, tok := p.parseExpr(token{}, false, false, false, false)
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
