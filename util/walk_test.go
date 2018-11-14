//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package util

import (
	"testing"

	"open2b/template/ast"
	"open2b/template/parser"
)

type TestVisitor struct {
	Positions []int
}

func (tv *TestVisitor) Visit(node ast.Node) Visitor {
	if node == nil {
		return nil
	}
	tv.Positions = append(tv.Positions, node.Pos().Start)
	_, isIdentifier := node.(*ast.Identifier)
	_, isString := node.(*ast.String)
	if isIdentifier || isString {
		return nil
	}
	return tv

}

func TestWalk(t *testing.T) {
	stringCases := []struct {
		input       string // da parsare
		expectedPos []int  // posizioni attese
	}{
		{"{{1}}", []int{0, 0, 2}},
		{"{{5+6}}", []int{0, 0, 2, 2, 4}},
		{`{% var x = 10 %}`, []int{0, 0, 11}},
		{`{% y = 10 %}`, []int{0, 0, 7}},
		{`{% y = (4 + 5) %}`, []int{0, 0, 7, 8, 12}},
		{`{{ call(3, 5) }}`, []int{0, 0, 3, 8, 11}},
		{`{% if 5 > 4 %} some text {% end %}`, []int{0, 0, 6, 6, 10, 14}},
		{`{% if 5 > 4 %} some text {% else %} some text {% end %}`, []int{0, 0, 6, 6, 10, 14, 35}},
		{`{% for p in ps %} some text {% end %}`, []int{0, 0, 12, 17}},
		{`{% for p in 1..10 %} some text {% end %}`, []int{0, 0, 12, 15, 20}},
		{`{% region Body %} some text {% end %}`, []int{0, 0, 17}},
		{`{{ (4+5)*6 }}`, []int{0, 0, 3, 3, 4, 6, 9}},
		{`{% x = vect[3] %}`, []int{0, 0, 7, 7, 12}},
		{`{% y = !x %}`, []int{0, 0, 7, 8}},
		{`{% y = !(true || false) %}`, []int{0, 0, 7, 8, 9, 17}},
		{`{% y = split("a b c d", " ") %}`, []int{0, 0, 7, 13, 24}},
		{`{% var x = -5 %}`, []int{0, 0, 11}},
		{`{% var x = mystruct.field %}`, []int{0, 0, 11, 11}},
		{`{% var x = (getStruct()).field %}`, []int{0, 0, 11, 11}},
		{`{% var x = -5.189 %}`, []int{0, 0, 11}},
		{`{% var x = vect[3:54] %}`, []int{0, 0, 11, 11, 16, 18}},
	}

	for _, c := range stringCases {
		tree, err := parser.Parse([]byte(c.input), ast.ContextHTML)
		if err != nil {
			panic(err)
		}

		var visitor TestVisitor
		Walk(&visitor, tree)

		if le, lg := len(c.expectedPos), len(visitor.Positions); le != lg {
			t.Errorf("Expected a slice with %v positions (%v) when elaborating %q, but got %v (%v)", le, c.expectedPos, c.input, lg, visitor.Positions)
			break
		}

		for i := 0; i < len(c.expectedPos); i++ {
			if c.expectedPos[i] != visitor.Positions[i] {
				t.Errorf("Expected value %v at index %v, got %v (expected %v, got %v)", c.expectedPos[i], i, visitor.Positions[i], c.expectedPos, visitor.Positions)
			}
		}
	}

	// Individual test for brackets, as these are removed from the parser
	// (they can not therefore be included in the previous test list).
	var visitor2 TestVisitor
	var pos *ast.Position = &ast.Position{1, 1, 0, 0}
	var parTree *ast.Parentesis = ast.NewParentesis(pos, ast.NewIdentifier(pos, "a"))
	Walk(&visitor2, parTree)
	if len(visitor2.Positions) != 2 || visitor2.Positions[0] != 0 || visitor2.Positions[1] != 0 {
		t.Errorf("expected [0, 0], got %v", visitor2.Positions)
	}

}
