// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package astutil

import (
	"testing"

	"scrigo/internal/compiler"
	"scrigo/internal/compiler/ast"
)

type TestVisitor struct {
	Positions []int
}

func (tv *TestVisitor) Visit(node ast.Node) Visitor {
	if node == nil {
		return nil
	}
	pos := node.Pos()
	if pos == nil {
		pos = &ast.Position{}
	}
	tv.Positions = append(tv.Positions, pos.Start)
	_, isIdentifier := node.(*ast.Identifier)
	_, isString := node.(*ast.String)
	if isIdentifier || isString {
		return nil
	}
	return tv

}

func TestWalk(t *testing.T) {
	stringCases := []struct {
		input       string
		expectedPos []int
	}{
		{"{{1}}", []int{0, 0, 2}},
		{"{{5+6}}", []int{0, 0, 2, 2, 4}},
		{`{% x := 10 %}`, []int{0, 0, 3, 8}},
		{`{% y = 10 %}`, []int{0, 0, 3, 7}},
		{`{% y = (4 + 5) %}`, []int{0, 0, 3, 7, 8, 12}},
		{`{{ call(3, 5) }}`, []int{0, 0, 3, 8, 11}},
		{`{% if 5 > 4 %} some text {% end %}`, []int{0, 0, 6, 6, 10, 0, 14}},
		{`{% if 5 > 4 %} some text {% else %} some text {% end %}`, []int{0, 0, 6, 6, 10, 0, 14, 0, 35}},
		{`{% for p in ps %} some text {% end %}`, []int{0, 0, 7, 7, 7, 12, 17}},
		{`{% macro Body %} some text {% end %}`, []int{0, 0, 16}},
		{`{{ (4+5)*6 }}`, []int{0, 0, 3, 3, 4, 6, 9}},
		{`{% x = vect[3] %}`, []int{0, 0, 3, 7, 7, 12}},
		{`{% y = !x %}`, []int{0, 0, 3, 7, 8}},
		{`{% y = !(true || false) %}`, []int{0, 0, 3, 7, 8, 9, 17}},
		{`{% y = split("a b c d", " ") %}`, []int{0, 0, 3, 7, 13, 24}},
		{`{% x := -5 %}`, []int{0, 0, 3, 8}},
		{`{% x := mystruct.field %}`, []int{0, 0, 3, 8, 8}},
		{`{% x := (getStruct()).field %}`, []int{0, 0, 3, 8, 8}},
		{`{% x := -5.189 %}`, []int{0, 0, 3, 8}},
		{`{% x := vect[3:54] %}`, []int{0, 0, 3, 8, 8, 13, 15}},
	}

	for _, c := range stringCases {
		tree, err := compiler.ParseTemplateSource([]byte(c.input), ast.ContextHTML)
		if err != nil {
			panic(err)
		}

		var visitor TestVisitor
		Walk(&visitor, tree)

		if le, lg := len(c.expectedPos), len(visitor.Positions); le != lg {
			t.Errorf("Expected a slice with %v positions (%v) when elaborating %q, but got %v (%v)", le, c.expectedPos, c.input, lg, visitor.Positions)
			continue
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
	var pos *ast.Position = &ast.Position{Line: 1, Column: 1, Start: 0, End: 0}
	var parTree *ast.Parenthesis = ast.NewParenthesis(pos, ast.NewIdentifier(pos, "a"))
	Walk(&visitor2, parTree)
	if len(visitor2.Positions) != 2 || visitor2.Positions[0] != 0 || visitor2.Positions[1] != 0 {
		t.Errorf("expected [0, 0], got %v", visitor2.Positions)
	}

}
