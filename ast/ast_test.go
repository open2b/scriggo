// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast

import (
	"fmt"
	"testing"
)

func init() {
	expandedPrint = true
}

var n1 = NewBasicLiteral(nil, IntLiteral, "1")
var n2 = NewBasicLiteral(nil, IntLiteral, "2")
var n3 = NewBasicLiteral(nil, IntLiteral, "3")
var n5 = NewBasicLiteral(nil, IntLiteral, "5")
var n7 = NewBasicLiteral(nil, IntLiteral, "7")

var expressionStringTests = []struct {
	str  string
	expr Expression
}{
	{"1", n1},
	{"3.59", NewBasicLiteral(nil, FloatLiteral, "3.59")},
	{`"abc"`, NewBasicLiteral(nil, StringLiteral, `"abc"`)},
	{"\"a\\tb\"", NewBasicLiteral(nil, StringLiteral, `"a\tb"`)},
	{"x", NewIdentifier(nil, "x")},
	{"-1", NewUnaryOperator(nil, OperatorSubtraction, n1)},
	{"1 + 2", NewBinaryOperator(nil, OperatorAddition, n1, n2)},
	{"1 + 2", NewBinaryOperator(nil, OperatorAddition, n1, n2)},
	{"f()", NewCall(nil, NewIdentifier(nil, "f"), []Expression{}, false)},
	{"f(a)", NewCall(nil, NewIdentifier(nil, "f"), []Expression{NewIdentifier(nil, "a")}, false)},
	{"f(a, b)", NewCall(nil, NewIdentifier(nil, "f"), []Expression{NewIdentifier(nil, "a"), NewIdentifier(nil, "b")}, false)},
	{"a[2]", NewIndex(nil, NewIdentifier(nil, "a"), n2)},
	{"a[:]", NewSlicing(nil, NewIdentifier(nil, "a"), nil, nil, nil, false)},
	{"a[2:]", NewSlicing(nil, NewIdentifier(nil, "a"), n2, nil, nil, false)},
	{"a[:5]", NewSlicing(nil, NewIdentifier(nil, "a"), nil, n5, nil, false)},
	{"a[2:5]", NewSlicing(nil, NewIdentifier(nil, "a"), n2, n5, nil, false)},
	{"a[2:5:7]", NewSlicing(nil, NewIdentifier(nil, "a"), n2, n5, n7, true)},
	{"a[:5:7]", NewSlicing(nil, NewIdentifier(nil, "a"), nil, n5, n7, true)},
	{"a.b", NewSelector(nil, NewIdentifier(nil, "a"), "b")},
	{"(a)", NewParenthesis(nil, NewIdentifier(nil, "a"))},
	{"-(1 + 2)", NewUnaryOperator(nil, OperatorSubtraction, NewBinaryOperator(nil, OperatorAddition, n1, n2))},
	{"-(+1)", NewUnaryOperator(nil, OperatorSubtraction, NewUnaryOperator(nil, OperatorAddition, n1))},
	{"1 * 2 + -3", NewBinaryOperator(nil, OperatorAddition,
		NewBinaryOperator(nil, OperatorMultiplication, n1, n2),
		NewUnaryOperator(nil, OperatorSubtraction, n3))},
	{"f() - 2", NewBinaryOperator(nil, OperatorSubtraction, NewCall(nil, NewIdentifier(nil, "f"), []Expression{}, false), n2)},
	{"-a.b", NewUnaryOperator(nil, OperatorSubtraction, NewSelector(nil, NewIdentifier(nil, "a"), "b"))},
	{"[]int([]int{1, 2, 3})", NewCall(nil, NewSliceType(nil, NewIdentifier(nil, "int")),
		[]Expression{NewCompositeLiteral(nil, NewSliceType(nil, NewIdentifier(nil, "int")),
			[]KeyValue{{nil, NewBasicLiteral(nil, IntLiteral, "1")}, {nil, NewBasicLiteral(nil, IntLiteral, "2")}, {nil, NewBasicLiteral(nil, IntLiteral, "3")}})}, false)},
	{"[]int{1, 2}", NewCompositeLiteral(nil, NewSliceType(nil, NewIdentifier(nil, "int")),
		[]KeyValue{
			{nil, NewBasicLiteral(nil, IntLiteral, "1")},
			{nil, NewBasicLiteral(nil, IntLiteral, "2")},
		})},
	{"[2][]number{[]number{1, 2}, []number{3, 4, 5}}", NewCompositeLiteral(nil,
		NewArrayType(nil, NewBasicLiteral(nil, IntLiteral, "2"), NewSliceType(nil, NewIdentifier(nil, "number"))),
		[]KeyValue{
			{nil, NewCompositeLiteral(nil, NewSliceType(nil, NewIdentifier(nil, "number")), []KeyValue{{nil, NewBasicLiteral(nil, IntLiteral, "1")}, {nil, NewBasicLiteral(nil, IntLiteral, "2")}})},
			{nil, NewCompositeLiteral(nil, NewSliceType(nil, NewIdentifier(nil, "number")), []KeyValue{{nil, NewBasicLiteral(nil, IntLiteral, "3")}, {nil, NewBasicLiteral(nil, IntLiteral, "4")}, KeyValue{nil, NewBasicLiteral(nil, IntLiteral, "5")}})},
		})},
	{"[...]int{1, 4, 9}",
		NewCompositeLiteral(nil, NewArrayType(nil, nil, NewIdentifier(nil, "int")), []KeyValue{{nil, NewBasicLiteral(nil, IntLiteral, "1")}, {nil, NewBasicLiteral(nil, IntLiteral, "4")}, KeyValue{nil, NewBasicLiteral(nil, IntLiteral, "9")}})},
	{"[]string{0: \"zero\", 1: \"one\"}",
		NewCompositeLiteral(nil,
			NewSliceType(nil, NewIdentifier(nil, "string")),
			[]KeyValue{
				{NewBasicLiteral(nil, IntLiteral, "0"), NewBasicLiteral(nil, StringLiteral, `"zero"`)},
				{NewBasicLiteral(nil, IntLiteral, "1"), NewBasicLiteral(nil, StringLiteral, `"one"`)},
			})},
	{"[]int{1: 5, 6, 7: 9}",
		NewCompositeLiteral(
			nil,
			NewSliceType(nil, NewIdentifier(nil, "int")),
			[]KeyValue{
				{NewBasicLiteral(nil, IntLiteral, "1"), NewBasicLiteral(nil, IntLiteral, "5")},
				{nil, NewBasicLiteral(nil, IntLiteral, "6")},
				{NewBasicLiteral(nil, IntLiteral, "7"), NewBasicLiteral(nil, IntLiteral, "9")},
			})},
	{"[]T", NewSliceType(nil, NewIdentifier(nil, "T"))},
	{"map[int]bool", NewMapType(nil, NewIdentifier(nil, "int"), NewIdentifier(nil, "bool"))},
	{"map[int][]int", NewMapType(nil, NewIdentifier(nil, "int"), NewSliceType(nil, NewIdentifier(nil, "int")))},
	{"map[int][]int{}", NewCompositeLiteral(nil, NewMapType(nil, NewIdentifier(nil, "int"), NewSliceType(nil, NewIdentifier(nil, "int"))), nil)},
	{"map[int]bool{3: true, 6: false}", NewCompositeLiteral(nil, NewMapType(nil, NewIdentifier(nil, "int"), NewIdentifier(nil, "bool")), []KeyValue{
		{NewBasicLiteral(nil, IntLiteral, "3"), NewIdentifier(nil, "true")},
		{NewBasicLiteral(nil, IntLiteral, "6"), NewIdentifier(nil, "false")},
	})},
	{"map[int]bool{3: true, 6: false}[32]", NewIndex(nil, NewCompositeLiteral(nil,
		NewMapType(nil, NewIdentifier(nil, "int"), NewIdentifier(nil, "bool")), []KeyValue{
			{NewBasicLiteral(nil, IntLiteral, "3"), NewIdentifier(nil, "true")},
			{NewBasicLiteral(nil, IntLiteral, "6"), NewIdentifier(nil, "false")},
		}), NewBasicLiteral(nil, IntLiteral, "32"))},
	{"&x", NewUnaryOperator(nil, OperatorAnd, NewIdentifier(nil, "x"))},
	{`pkg.Struct{Key1: value1, Key2: "value2", Key3: 33}`,
		NewCompositeLiteral(nil,
			NewSelector(nil, NewIdentifier(nil, "pkg"), "Struct"),
			[]KeyValue{
				{NewIdentifier(nil, "Key1"), NewIdentifier(nil, "value1")},
				{NewIdentifier(nil, "Key2"), NewBasicLiteral(nil, StringLiteral, `"value2"`)},
				{NewIdentifier(nil, "Key3"), NewBasicLiteral(nil, IntLiteral, "33")},
			})},
	{`pkg.Struct{value1, "value2", 33}`, NewCompositeLiteral(nil,
		NewSelector(nil, NewIdentifier(nil, "pkg"), "Struct"),
		[]KeyValue{
			KeyValue{nil, NewIdentifier(nil, "value1")},
			{nil, NewBasicLiteral(nil, StringLiteral, `"value2"`)},
			{nil, NewBasicLiteral(nil, IntLiteral, "33")},
		})},
	{"[]*int", NewSliceType(nil, NewUnaryOperator(
		nil, OperatorMultiplication, NewIdentifier(nil, "int")),
	)},
	{"func()", NewFuncType(nil, nil, nil, false)},
	{"func(int)", NewFuncType(nil, []*Parameter{NewParameter(nil, NewIdentifier(nil, "int"))}, nil, false)},
	{"func(a int)", NewFuncType(nil, []*Parameter{NewParameter(NewIdentifier(nil, "a"), NewIdentifier(nil, "int"))}, nil, false)},
	{"func() int", NewFuncType(nil, nil, []*Parameter{NewParameter(nil, NewIdentifier(nil, "int"))}, false)},
	{"func() (n int)", NewFuncType(nil, nil, []*Parameter{NewParameter(NewIdentifier(nil, "n"), NewIdentifier(nil, "int"))}, false)},
	{"func(a int, b bool) (n int, err error)", NewFuncType(nil,
		[]*Parameter{
			NewParameter(NewIdentifier(nil, "a"), NewIdentifier(nil, "int")),
			NewParameter(NewIdentifier(nil, "b"), NewIdentifier(nil, "bool"))},
		[]*Parameter{
			NewParameter(NewIdentifier(nil, "n"), NewIdentifier(nil, "int")),
			NewParameter(NewIdentifier(nil, "err"), NewIdentifier(nil, "error"))}, false)},
	{"func literal", NewFunc(nil, nil, NewFuncType(nil,
		[]*Parameter{NewParameter(NewIdentifier(nil, "a"), NewIdentifier(nil, "int"))}, nil, false),
		NewBlock(nil, nil))},
	{"func declaration", NewFunc(nil, NewIdentifier(nil, "f"), NewFuncType(nil,
		[]*Parameter{NewParameter(NewIdentifier(nil, "a"), NewIdentifier(nil, "int"))}, nil, false),
		NewBlock(nil, nil))},
}

func TestExpressionString(t *testing.T) {
	for _, e := range expressionStringTests {
		if e.expr.String() != e.str {
			t.Errorf("unexpected %q, expecting %q\n", e.expr.String(), e.str)
		}
	}
}

func ptrToStr(s string) *string {
	return &s
}

func TestStatementString(t *testing.T) {
	cases := []struct {
		node Node
		str  string
	}{
		{
			NewTypeDeclaration(nil, NewIdentifier(nil, "Int"), NewIdentifier(nil, "int"), false),
			"type Int int",
		},
		{
			NewTypeDeclaration(nil, NewIdentifier(nil, "StringSlice"), NewSliceType(nil, NewIdentifier(nil, "string")), false),
			"type StringSlice []string",
		},
		{
			NewTypeDeclaration(nil, NewIdentifier(nil, "Int"), NewIdentifier(nil, "int"), true),
			"type Int = int",
		},
		{
			NewTypeDeclaration(nil, NewIdentifier(nil, "StringSlice"), NewSliceType(nil, NewIdentifier(nil, "string")), true),
			"type StringSlice = []string",
		},
		{
			NewStructType(nil, []*Field{
				NewField(
					[]*Identifier{
						NewIdentifier(nil, "a"),
						NewIdentifier(nil, "b"),
					},
					NewIdentifier(nil, "Int"),
					nil,
				),
				NewField(
					[]*Identifier{
						NewIdentifier(nil, "c"),
					},
					NewIdentifier(nil, "String"),
					nil,
				),
				NewField(
					nil,
					NewIdentifier(nil, "Implicit"),
					nil,
				),
			}),
			"struct { a, b Int; c String; Implicit }",
		},
		{
			NewStructType(nil, []*Field{
				NewField(
					[]*Identifier{
						NewIdentifier(nil, "a"),
						NewIdentifier(nil, "b"),
					},
					NewIdentifier(nil, "Int"),
					ptrToStr("tag1"),
				),
				NewField(
					[]*Identifier{
						NewIdentifier(nil, "c"),
					},
					NewIdentifier(nil, "String"),
					ptrToStr("tag2"),
				),
				NewField(
					nil,
					NewIdentifier(nil, "Implicit"),
					ptrToStr("tag3"),
				),
			}),
			"struct { a, b Int `tag1`; c String `tag2`; Implicit `tag3` }",
		},
	}
	for _, c := range cases {
		got := fmt.Sprintf("%v", c.node)
		expected := c.str
		if got != expected {
			t.Errorf("expecting: %q, got: %q", expected, got)
		}
	}
}
