//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

package parser

import (
	"fmt"
	"testing"

	"open2b/template/ast"
)

func p(line, column, start, end int) *ast.Position {
	return &ast.Position{line, column, start, end}
}

var exprTests = []struct {
	src  string
	node ast.Node
}{
	{"_", ast.NewIdentifier(p(1, 1, 0, 0), "_")},
	{"a", ast.NewIdentifier(p(1, 1, 0, 0), "a")},
	{"a5", ast.NewIdentifier(p(1, 1, 0, 1), "a5")},
	{"_a", ast.NewIdentifier(p(1, 1, 0, 1), "_a")},
	{"_5", ast.NewIdentifier(p(1, 1, 0, 1), "_5")},
	{"0", ast.NewInt(p(1, 1, 0, 0), 0)},
	{"3", ast.NewInt(p(1, 1, 0, 0), 3)},
	{"2147483647", ast.NewInt(p(1, 1, 0, 9), 2147483647)},
	{"-2147483648", ast.NewInt(p(1, 1, 0, 10), -2147483648)},
	{"\"\"", ast.NewString(p(1, 1, 0, 1), "")},
	{"\"a\"", ast.NewString(p(1, 1, 0, 2), "a")},
	{`"\t"`, ast.NewString(p(1, 1, 0, 3), "\t")},
	{`"\a\b\f\n\r\t\v\\\""`, ast.NewString(p(1, 1, 0, 19), "\a\b\f\n\r\t\v\\\"")},
	{"``", ast.NewString(p(1, 1, 0, 1), "")},
	{"`\\t`", ast.NewString(p(1, 1, 0, 3), "\\t")},
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
	{"a[:]", ast.NewSlice(p(1, 2, 0, 3), ast.NewIdentifier(p(1, 1, 0, 0), "a"), nil, nil)},
	{"a[:2]", ast.NewSlice(p(1, 2, 0, 4), ast.NewIdentifier(p(1, 1, 0, 0), "a"), nil, ast.NewInt(p(1, 4, 3, 3), 2))},
	{"a[1:]", ast.NewSlice(p(1, 2, 0, 4), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(1, 3, 2, 2), 1), nil)},
	{"a[1:2]", ast.NewSlice(p(1, 2, 0, 5), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(1, 3, 2, 2), 1), ast.NewInt(p(1, 5, 4, 4), 2))},
	{"a.b", ast.NewSelector(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), "b")},
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
	{"a.b*c", ast.NewBinaryOperator(p(1, 4, 0, 4), ast.OperatorMultiplication, ast.NewSelector(p(1, 2, 0, 2),
		ast.NewIdentifier(p(1, 1, 0, 0), "a"), "b"), ast.NewIdentifier(p(1, 5, 4, 4), "c"))},
	{"a*b.c", ast.NewBinaryOperator(p(1, 2, 0, 4), ast.OperatorMultiplication, ast.NewIdentifier(p(1, 1, 0, 0), "a"),
		ast.NewSelector(p(1, 4, 2, 4), ast.NewIdentifier(p(1, 3, 2, 2), "b"), "c"))},
	{"a.b(c)", ast.NewCall(p(1, 4, 0, 5), ast.NewSelector(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), "b"),
		[]ast.Expression{ast.NewIdentifier(p(1, 5, 4, 4), "c")})},
	{"1\t+\n2", ast.NewBinaryOperator(p(1, 3, 0, 4), ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(2, 1, 4, 4), 2))},
	{"1\t\r +\n\r\n\r\t 2", ast.NewBinaryOperator(p(1, 5, 0, 11), ast.OperatorAddition, ast.NewInt(p(1, 1, 0, 0), 1), ast.NewInt(p(3, 4, 11, 11), 2))},
	{"a(\n\t1\t,\n2\t)", ast.NewCall(p(1, 2, 0, 10), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{
		ast.NewInt(p(2, 2, 4, 4), 1), ast.NewInt(p(3, 1, 8, 8), 2)})},
	{"a\t\r ()", ast.NewCall(p(1, 5, 0, 5), ast.NewIdentifier(p(1, 1, 0, 0), "a"), []ast.Expression{})},
	{"a[\n\t1\t]", ast.NewIndex(p(1, 2, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(2, 2, 4, 4), 1))},
	{"a\t\r [1]", ast.NewIndex(p(1, 5, 0, 6), ast.NewIdentifier(p(1, 1, 0, 0), "a"), ast.NewInt(p(1, 6, 5, 5), 1))},
}

var treeTests = []struct {
	src  string
	node ast.Node
}{
	{"", ast.NewTree("", nil)},
	{"a", ast.NewTree("", []ast.Node{ast.NewText(p(1, 1, 0, 0), "a")})},
	{"{{a}}", ast.NewTree("", []ast.Node{ast.NewShow(p(1, 1, 0, 4), ast.NewIdentifier(p(1, 3, 2, 2), "a"), ast.ContextHTML)})},
	{"a{{b}}", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 0), "a"), ast.NewShow(p(1, 2, 1, 5), ast.NewIdentifier(p(1, 4, 3, 3), "b"), ast.ContextHTML)})},
	{"{{a}}b", ast.NewTree("", []ast.Node{
		ast.NewShow(p(1, 1, 0, 4), ast.NewIdentifier(p(1, 3, 2, 2), "a"), ast.ContextHTML), ast.NewText(p(1, 6, 5, 5), "b")})},
	{"a{{b}}c", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 0), "a"), ast.NewShow(p(1, 2, 1, 5), ast.NewIdentifier(p(1, 4, 3, 3), "b"), ast.ContextHTML),
		ast.NewText(p(1, 7, 6, 6), "c")})},
	{"{% var a = 1 %}", ast.NewTree("", []ast.Node{
		ast.NewVar(p(1, 1, 0, 14), ast.NewIdentifier(p(1, 8, 7, 7), "a"), ast.NewInt(p(1, 13, 11, 11), 1))})},
	{"{% a = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 10), ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewInt(p(1, 8, 7, 7), 2))})},
	{"{% show a %}{% end %}", ast.NewTree("", []ast.Node{
		ast.NewShow(p(1, 1, 0, 11), ast.NewIdentifier(p(1, 9, 8, 8), "a"), ast.ContextHTML)})},
	{"{% show a %}b{% end %}", ast.NewTree("", []ast.Node{
		ast.NewShow(p(1, 1, 0, 11), ast.NewIdentifier(p(1, 9, 8, 8), "a"), ast.ContextHTML)})},
	{"{% for v in e %}b{% end %}", ast.NewTree("", []ast.Node{ast.NewFor(p(1, 1, 0, 15),
		nil, ast.NewIdentifier(p(1, 8, 7, 7), "v"), ast.NewIdentifier(p(1, 13, 12, 12), "e"), []ast.Node{ast.NewText(p(1, 17, 16, 16), "b")})})},
	{"{% for i, v in e %}b{% end %}", ast.NewTree("", []ast.Node{ast.NewFor(p(1, 1, 0, 18),
		ast.NewIdentifier(p(1, 8, 7, 7), "i"), ast.NewIdentifier(p(1, 11, 10, 10), "v"), ast.NewIdentifier(p(1, 16, 15, 15), "e"),
		[]ast.Node{ast.NewText(p(1, 20, 19, 19), "b")})})},
	{"{% if a %}b{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 9), ast.NewIdentifier(p(1, 7, 6, 6), "a"), []ast.Node{ast.NewText(p(1, 11, 10, 10), "b")})})},
	{"{% extend \"/a.b\" %}", ast.NewTree("", []ast.Node{ast.NewExtend(p(1, 1, 0, 18), "/a.b", nil)})},
	{"{% include \"/a.b\" %}", ast.NewTree("", []ast.Node{ast.NewInclude(p(1, 1, 0, 19), "/a.b", nil)})},
	{"{% region \"a\" %}b{% end %}", ast.NewTree("", []ast.Node{
		ast.NewRegion(p(1, 1, 0, 15), "a", []ast.Node{ast.NewText(p(1, 17, 16, 16), "b")})})},
}

var pageTests = map[string]struct {
	src  string
	tree *ast.Tree
}{
	"/simple.html": {
		"<!DOCTYPE html>\n<html>\n<head><title>{{ title }}</title></head>\n<body>{{ content }}</body>\n</html>",
		ast.NewTree("", []ast.Node{
			ast.NewText(p(1, 1, 0, 35), "<!DOCTYPE html>\n<html>\n<head><title>"),
			ast.NewShow(p(3, 14, 36, 46), ast.NewIdentifier(p(3, 17, 39, 43), "title"), ast.ContextHTML),
			ast.NewText(p(3, 25, 47, 68), "</title></head>\n<body>"),
			ast.NewShow(p(4, 7, 69, 81), ast.NewIdentifier(p(4, 10, 72, 78), "content"), ast.ContextHTML),
			ast.NewText(p(4, 20, 82, 96), "</body>\n</html>"),
		}),
	},
	"/simple2.html": {
		"<!DOCTYPE html>\n<html>\n<body>{% include \"/include2.html\" %}</body>\n</html>",
		ast.NewTree("", []ast.Node{
			ast.NewText(p(1, 1, 0, 28), "<!DOCTYPE html>\n<html>\n<body>"),
			ast.NewInclude(p(3, 7, 29, 58), "/include2.html", ast.NewTree("", []ast.Node{
				ast.NewText(p(1, 1, 0, 4), "<div>"),
				ast.NewShow(p(1, 6, 5, 17), ast.NewIdentifier(p(1, 9, 8, 14), "content"), ast.ContextHTML),
				ast.NewText(p(1, 19, 18, 23), "</div>"),
			})),
			ast.NewText(p(3, 37, 59, 73), "</body>\n</html>"),
		}),
	},
	"/include2.inc": {
		"<div>{{ content }}</div>",
		nil,
	},
}

func TestExpressions(t *testing.T) {
	for _, expr := range exprTests {
		var lex = newLexer([]byte("{{" + expr.src + "}}"))
		_ = <-lex.tokens
		node, tok, err := parseExpr(lex)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		if node == nil {
			t.Errorf("source: %q, unexpected %s, expecting expression\n", expr.src, tok)
			continue
		}
		err = equals(node, expr.node, 2)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
		}
	}
}

func TestTrees(t *testing.T) {
	for _, tree := range treeTests {
		node, err := Parse([]byte(tree.src))
		if err != nil {
			t.Errorf("source: %q, %s\n", tree.src, err)
			continue
		}
		err = equals(node, tree.node, 0)
		if err != nil {
			t.Errorf("source: %q, %s\n", tree.src, err)
		}
	}
}

func readFunc(path string) (*ast.Tree, error) {
	return Parse([]byte(pageTests[path].src))
}

func TestPages(t *testing.T) {
	// simple.html
	parser := NewParser(readFunc)
	p := pageTests["/simple.html"]
	tree, err := parser.Parse("/simple.html")
	if err != nil {
		t.Errorf("source: %q, %s\n", p.src, err)
	}
	err = equals(tree, p.tree, 0)
	if err != nil {
		t.Errorf("source: %q, %s\n", p.src, err)
	}
	// simple2.html
	p = pageTests["/simple2.html"]
	tree, err = parser.Parse("/simple2.html")
	if err != nil {
		t.Errorf("source: %q, %s\n", p.src, err)
	}
	err = equals(tree, p.tree, 0)
	if err != nil {
		t.Errorf("source: %q, %s\n", p.src, err)
	}
}

func equals(n1, n2 ast.Node, p int) error {
	if n1 == nil && n2 == nil {
		return nil
	}
	if (n1 == nil) != (n2 == nil) {
		if n1 == nil {
			return fmt.Errorf("unexpected node nil, expecting %#v", n2)
		} else {
			return fmt.Errorf("unexpected node %#v, expecting nil", n1)
		}
	}
	var pos1 = n1.Pos()
	var pos2 = n2.Pos()
	if pos1.Line != pos2.Line {
		return fmt.Errorf("unexpected line %d, expecting %d", pos1.Line, pos2.Line)
	}
	if pos1.Line == 1 {
		if pos1.Column-p != pos2.Column {
			return fmt.Errorf("unexpected column %d, expecting %d", pos1.Column-p, pos2.Column)
		}
	} else {
		if pos1.Column != pos2.Column {
			return fmt.Errorf("unexpected column %d, expecting %d", pos1.Column, pos2.Column)
		}
	}
	if pos1.Start-p != pos2.Start {
		return fmt.Errorf("unexpected start %d, expecting %d", pos1.Start-p, pos2.Start)
	}
	if pos1.End-p != pos2.End {
		return fmt.Errorf("unexpected end %d, expecting %d", pos1.End-p, pos2.End)
	}
	switch nn1 := n1.(type) {
	case *ast.Tree:
		nn2, ok := n2.(*ast.Tree)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", nn1, nn2)
		}
		if len(nn1.Nodes) != len(nn2.Nodes) {
			return fmt.Errorf("unexpected nodes len %d, expecting %d", len(nn1.Nodes), len(nn2.Nodes))
		}
		for i, node := range nn1.Nodes {
			err := equals(node, nn2.Nodes[i], p)
			if err != nil {
				return err
			}
		}
	case *ast.Identifier:
		nn2, ok := n2.(*ast.Identifier)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Name != nn2.Name {
			return fmt.Errorf("unexpected %q, expecting %q", nn1.Name, nn2.Name)
		}
	case *ast.Int:
		nn2, ok := n2.(*ast.Int)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Value != nn2.Value {
			return fmt.Errorf("unexpected %d, expecting %d", nn1.Value, nn2.Value)
		}
	case *ast.String:
		nn2, ok := n2.(*ast.String)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", nn1, nn2)
		}
		if nn1.Text != nn2.Text {
			return fmt.Errorf("unexpected %q, expecting %q", nn1.Text, nn2.Text)
		}
	case *ast.Parentesis:
		nn2, ok := n2.(*ast.Parentesis)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", nn1, nn2)
		}
		err := equals(nn1.Expr, nn2.Expr, p)
		if err != nil {
			return err
		}
	case *ast.UnaryOperator:
		nn2, ok := n2.(*ast.UnaryOperator)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", nn1, nn2)
		}
		if nn1.Op != nn2.Op {
			return fmt.Errorf("unexpected operator %d, expecting %d", nn1.Op, nn2.Op)
		}
		err := equals(nn1.Expr, nn2.Expr, p)
		if err != nil {
			return err
		}
	case *ast.BinaryOperator:
		nn2, ok := n2.(*ast.BinaryOperator)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", nn1, nn2)
		}
		if nn1.Op != nn2.Op {
			return fmt.Errorf("unexpected operator %d, expecting %d", nn1.Op, nn2.Op)
		}
		err := equals(nn1.Expr1, nn2.Expr1, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Expr2, nn2.Expr2, p)
		if err != nil {
			return err
		}
	case *ast.Call:
		nn2, ok := n2.(*ast.Call)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", nn1, nn2)
		}
		err := equals(nn1.Func, nn2.Func, p)
		if err != nil {
			return err
		}
		if len(nn1.Args) != len(nn2.Args) {
			return fmt.Errorf("unexpected arguments len %d, expecting %d", len(nn1.Args), len(nn2.Args))
		}
		for i, arg := range nn1.Args {
			err = equals(arg, nn2.Args[i], p)
			if err != nil {
				return err
			}
		}
	case *ast.Index:
		nn2, ok := n2.(*ast.Index)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", nn1, nn2)
		}
		err := equals(nn1.Expr, nn2.Expr, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Index, nn2.Index, p)
		if err != nil {
			return err
		}
	case *ast.Slice:
		nn2, ok := n2.(*ast.Slice)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", nn1, nn2)
		}
		err := equals(nn1.Expr, nn2.Expr, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Low, nn2.Low, p)
		if err != nil {
			return err
		}
		err = equals(nn1.High, nn2.High, p)
		if err != nil {
			return err
		}
	case *ast.Show:
		nn2, ok := n2.(*ast.Show)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", nn1, nn2)
		}
		err := equals(nn1.Expr, nn2.Expr, p)
		if err != nil {
			return err
		}
		if nn1.Context != nn2.Context {
			return fmt.Errorf("unexpected context %d, expecting %d", nn1.Context, nn2.Context)
		}
	case *ast.Region:
		nn2, ok := n2.(*ast.Region)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", nn1, nn2)
		}
		if nn1.Name != nn2.Name {
			return fmt.Errorf("unexpected %q, expecting %q", nn1.Name, nn2.Name)
		}
		if len(nn1.Nodes) != len(nn2.Nodes) {
			return fmt.Errorf("unexpected nodes len %d, expecting %d", len(nn1.Nodes), len(nn2.Nodes))
		}
		for i, node := range nn1.Nodes {
			err := equals(node, nn2.Nodes[i], p)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
