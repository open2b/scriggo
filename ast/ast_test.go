// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast

import (
	"math/big"
	"testing"
)

var n1 = NewInt(nil, big.NewInt(1))
var n2 = NewInt(nil, big.NewInt(2))
var n3 = NewInt(nil, big.NewInt(3))
var n5 = NewInt(nil, big.NewInt(5))

var expressionStringTests = []struct {
	str  string
	expr Expression
}{
	{"1", n1},
	{"3.59", NewFloat(nil, big.NewFloat(3.59))},
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
	{"(a)", NewParenthesis(nil, NewIdentifier(nil, "a"))},
	{"-(1 + 2)", NewUnaryOperator(nil, OperatorSubtraction, NewBinaryOperator(nil, OperatorAddition, n1, n2))},
	{"-(+1)", NewUnaryOperator(nil, OperatorSubtraction, NewUnaryOperator(nil, OperatorAddition, n1))},
	{"1 * 2 + -3", NewBinaryOperator(nil, OperatorAddition,
		NewBinaryOperator(nil, OperatorMultiplication, n1, n2),
		NewUnaryOperator(nil, OperatorSubtraction, n3))},
	{"f() - 2", NewBinaryOperator(nil, OperatorSubtraction, NewCall(nil, NewIdentifier(nil, "f"), []Expression{}), n2)},
	{"-a.b", NewUnaryOperator(nil, OperatorSubtraction, NewSelector(nil, NewIdentifier(nil, "a"), "b"))},
	{"[]int([]int{1, 2, 3})", NewCall(nil, NewSliceType(nil, NewIdentifier(nil, "int")),
		[]Expression{NewCompositeLiteral(nil, NewSliceType(nil, NewIdentifier(nil, "int")),
			[]KeyValue{{nil, NewInt(nil, big.NewInt(1))}, {nil, NewInt(nil, big.NewInt(2))}, {nil, NewInt(nil, big.NewInt(3))}})})},
	{"[]int{1, 2}", NewCompositeLiteral(nil, NewSliceType(nil, NewIdentifier(nil, "int")),
		[]KeyValue{
			{nil, NewInt(nil, big.NewInt(1))},
			{nil, NewInt(nil, big.NewInt(2))},
		})},
	{"[2][]number{[]number{1, 2}, []number{3, 4, 5}}", NewCompositeLiteral(nil,
		NewArrayType(nil, NewInt(nil, big.NewInt(2)), NewSliceType(nil, NewIdentifier(nil, "number"))),
		[]KeyValue{
			{nil, NewCompositeLiteral(nil, NewSliceType(nil, NewIdentifier(nil, "number")), []KeyValue{{nil, NewInt(nil, big.NewInt(1))}, {nil, NewInt(nil, big.NewInt(2))}})},
			{nil, NewCompositeLiteral(nil, NewSliceType(nil, NewIdentifier(nil, "number")), []KeyValue{{nil, NewInt(nil, big.NewInt(3))}, {nil, NewInt(nil, big.NewInt(4))}, KeyValue{nil, NewInt(nil, big.NewInt(5))}})},
		})},
	{"[...]int{1, 4, 9}",
		NewCompositeLiteral(nil, NewArrayType(nil, nil, NewIdentifier(nil, "int")), []KeyValue{{nil, NewInt(nil, big.NewInt(1))}, {nil, NewInt(nil, big.NewInt(4))}, KeyValue{nil, NewInt(nil, big.NewInt(9))}})},
	{"[]string{0: \"zero\", 1: \"one\"}",
		NewCompositeLiteral(nil,
			NewSliceType(nil, NewIdentifier(nil, "string")),
			[]KeyValue{
				{NewInt(nil, big.NewInt(0)), NewString(nil, "zero")},
				{NewInt(nil, big.NewInt(1)), NewString(nil, "one")},
			})},
	{"[]int{1: 5, 6, 7: 9}",
		NewCompositeLiteral(
			nil,
			NewSliceType(nil, NewIdentifier(nil, "int")),
			[]KeyValue{
				{NewInt(nil, big.NewInt(1)), NewInt(nil, big.NewInt(5))},
				{nil, NewInt(nil, big.NewInt(6))},
				{NewInt(nil, big.NewInt(7)), NewInt(nil, big.NewInt(9))},
			})},
	{"[]T", NewSliceType(nil, NewIdentifier(nil, "T"))},
	{"map[int]bool", NewMapType(nil, NewIdentifier(nil, "int"), NewIdentifier(nil, "bool"))},
	{"map[int][]int", NewMapType(nil, NewIdentifier(nil, "int"), NewSliceType(nil, NewIdentifier(nil, "int")))},
	{"map[int][]int{}", NewCompositeLiteral(nil, NewMapType(nil, NewIdentifier(nil, "int"), NewSliceType(nil, NewIdentifier(nil, "int"))), nil)},
	{"map[int]bool{3: true, 6: false}", NewCompositeLiteral(nil, NewMapType(nil, NewIdentifier(nil, "int"), NewIdentifier(nil, "bool")), []KeyValue{
		{NewInt(nil, big.NewInt(3)), NewIdentifier(nil, "true")},
		{NewInt(nil, big.NewInt(6)), NewIdentifier(nil, "false")},
	})},
	{"map[int]bool{3: true, 6: false}[32]", NewIndex(nil, NewCompositeLiteral(nil,
		NewMapType(nil, NewIdentifier(nil, "int"), NewIdentifier(nil, "bool")), []KeyValue{
			{NewInt(nil, big.NewInt(3)), NewIdentifier(nil, "true")},
			{NewInt(nil, big.NewInt(6)), NewIdentifier(nil, "false")},
		}), NewInt(nil, big.NewInt(32)))},
	{"&x", NewUnaryOperator(nil, OperatorAmpersand, NewIdentifier(nil, "x"))},
	{`pkg.Struct{Key1: value1, Key2: "value2", Key3: 33}`,
		NewCompositeLiteral(nil,
			NewSelector(nil, NewIdentifier(nil, "pkg"), "Struct"),
			[]KeyValue{
				{NewIdentifier(nil, "Key1"), NewIdentifier(nil, "value1")},
				{NewIdentifier(nil, "Key2"), NewString(nil, "value2")},
				{NewIdentifier(nil, "Key3"), NewInt(nil, big.NewInt(33))},
			})},
	{`pkg.Struct{value1, "value2", 33}`, NewCompositeLiteral(nil,
		NewSelector(nil, NewIdentifier(nil, "pkg"), "Struct"),
		[]KeyValue{
			KeyValue{nil, NewIdentifier(nil, "value1")},
			{nil, NewString(nil, "value2")},
			{nil, NewInt(nil, big.NewInt(33))},
		})},
	{"[]*int", NewSliceType(nil, NewUnaryOperator(
		nil, OperatorMultiplication, NewIdentifier(nil, "int")),
	)},
	{"func()", NewFuncType(nil, nil, nil, false)},
	{"func(int)", NewFuncType(nil, []*Field{NewField(nil, NewIdentifier(nil, "int"))}, nil, false)},
	{"func(a int)", NewFuncType(nil, []*Field{NewField(NewIdentifier(nil, "a"), NewIdentifier(nil, "int"))}, nil, false)},
	{"func() int", NewFuncType(nil, nil, []*Field{NewField(nil, NewIdentifier(nil, "int"))}, false)},
	{"func() (n int)", NewFuncType(nil, nil, []*Field{NewField(NewIdentifier(nil, "n"), NewIdentifier(nil, "int"))}, false)},
	{"func(a int, b bool) (n int, err error)", NewFuncType(nil,
		[]*Field{
			NewField(NewIdentifier(nil, "a"), NewIdentifier(nil, "int")),
			NewField(NewIdentifier(nil, "b"), NewIdentifier(nil, "bool"))},
		[]*Field{
			NewField(NewIdentifier(nil, "n"), NewIdentifier(nil, "int")),
			NewField(NewIdentifier(nil, "err"), NewIdentifier(nil, "error"))}, false)},
	{"func literal", NewFunc(nil, nil, NewFuncType(nil,
		[]*Field{NewField(NewIdentifier(nil, "a"), NewIdentifier(nil, "int"))}, nil, false),
		NewBlock(nil, nil))},
	{"func declaration", NewFunc(nil, NewIdentifier(nil, "f"), NewFuncType(nil,
		[]*Field{NewField(NewIdentifier(nil, "a"), NewIdentifier(nil, "int"))}, nil, false),
		NewBlock(nil, nil))},
}

func TestExpressionString(t *testing.T) {
	for _, e := range expressionStringTests {
		if e.expr.String() != e.str {
			t.Errorf("unexpected %q, expecting %q\n", e.expr.String(), e.str)
		}
	}
}
