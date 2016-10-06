//
// Copyright (c) 2016 Open2b Software Snc. All Rights Reserved.
//

package template

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"open2b/template/ast"
)

var exprTests = []struct {
	src  string
	node ast.Node
}{
	{"_", ast.NewIdentifier(0, "_")},
	{"a", ast.NewIdentifier(0, "a")},
	{"a5", ast.NewIdentifier(0, "a5")},
	{"_a", ast.NewIdentifier(0, "_a")},
	{"_5", ast.NewIdentifier(0, "_5")},
	{"0", ast.NewInt32(0, 0)},
	{"3", ast.NewInt32(0, 3)},
	{"2147483647", ast.NewInt32(0, 2147483647)},
	{"2147483648", ast.NewInt64(0, 2147483648)},
	{"9223372036854775807", ast.NewInt64(0, 9223372036854775807)},
	{"\"\"", ast.NewString(0, "")},
	{"\"a\"", ast.NewString(0, "a")},
	{`"\t"`, ast.NewString(0, "\t")},
	{`"\a\b\f\n\r\t\v\\\""`, ast.NewString(0, "\a\b\f\n\r\t\v\\\"")},
	{"``", ast.NewString(0, "")},
	{"`\\t`", ast.NewString(0, "\\t")},
	{"!a", ast.NewUnaryOperator(0, ast.OperatorNot, ast.NewIdentifier(1, "a"))},
	{"1+2", ast.NewBinaryOperator(1, ast.OperatorAddition, ast.NewInt32(0, 1), ast.NewInt32(2, 2))},
	{"1-2", ast.NewBinaryOperator(1, ast.OperatorSubtraction, ast.NewInt32(0, 1), ast.NewInt32(2, 2))},
	{"1*2", ast.NewBinaryOperator(1, ast.OperatorMultiplication, ast.NewInt32(0, 1), ast.NewInt32(2, 2))},
	{"1/2", ast.NewBinaryOperator(1, ast.OperatorDivision, ast.NewInt32(0, 1), ast.NewInt32(2, 2))},
	{"1%2", ast.NewBinaryOperator(1, ast.OperatorModulo, ast.NewInt32(0, 1), ast.NewInt32(2, 2))},
	{"1==2", ast.NewBinaryOperator(1, ast.OperatorEqual, ast.NewInt32(0, 1), ast.NewInt32(3, 2))},
	{"1!=2", ast.NewBinaryOperator(1, ast.OperatorNotEqual, ast.NewInt32(0, 1), ast.NewInt32(3, 2))},
	{"1<2", ast.NewBinaryOperator(1, ast.OperatorLess, ast.NewInt32(0, 1), ast.NewInt32(2, 2))},
	{"1<=2", ast.NewBinaryOperator(1, ast.OperatorLessOrEqual, ast.NewInt32(0, 1), ast.NewInt32(3, 2))},
	{"1>2", ast.NewBinaryOperator(1, ast.OperatorGreater, ast.NewInt32(0, 1), ast.NewInt32(2, 2))},
	{"1>=2", ast.NewBinaryOperator(1, ast.OperatorGreaterOrEqual, ast.NewInt32(0, 1), ast.NewInt32(3, 2))},
	{"a&&b", ast.NewBinaryOperator(1, ast.OperatorAnd, ast.NewIdentifier(0, "a"), ast.NewIdentifier(3, "b"))},
	{"a||b", ast.NewBinaryOperator(1, ast.OperatorOr, ast.NewIdentifier(0, "a"), ast.NewIdentifier(3, "b"))},
	{"(a)", ast.NewParentesis(0, ast.NewIdentifier(1, "a"))},
	{"a()", ast.NewCall(1, ast.NewIdentifier(0, "a"), []ast.Expression{})},
	{"a(1)", ast.NewCall(1, ast.NewIdentifier(0, "a"), []ast.Expression{ast.NewInt32(2, 1)})},
	{"a(1,2)", ast.NewCall(1, ast.NewIdentifier(0, "a"), []ast.Expression{ast.NewInt32(2, 1), ast.NewInt32(4, 2)})},
	{"a[1]", ast.NewIndex(1, ast.NewIdentifier(0, "a"), ast.NewInt32(2, 1))},
	{"a.b", ast.NewSelector(1, ast.NewIdentifier(0, "a"), "b")},
	{"1+2+3", ast.NewBinaryOperator(3, ast.OperatorAddition,
		ast.NewBinaryOperator(1, ast.OperatorAddition, ast.NewInt32(0, 1), ast.NewInt32(2, 2)), ast.NewInt32(4, 3))},
	{"1-2-3", ast.NewBinaryOperator(3, ast.OperatorSubtraction,
		ast.NewBinaryOperator(1, ast.OperatorSubtraction, ast.NewInt32(0, 1), ast.NewInt32(2, 2)), ast.NewInt32(4, 3))},
	{"1*2*3", ast.NewBinaryOperator(3, ast.OperatorMultiplication,
		ast.NewBinaryOperator(1, ast.OperatorMultiplication, ast.NewInt32(0, 1), ast.NewInt32(2, 2)), ast.NewInt32(4, 3))},
	{"1+2*3", ast.NewBinaryOperator(1, ast.OperatorAddition, ast.NewInt32(0, 1),
		ast.NewBinaryOperator(3, ast.OperatorMultiplication, ast.NewInt32(2, 2), ast.NewInt32(4, 3)))},
	{"1-2/3", ast.NewBinaryOperator(1, ast.OperatorSubtraction, ast.NewInt32(0, 1),
		ast.NewBinaryOperator(3, ast.OperatorDivision, ast.NewInt32(2, 2), ast.NewInt32(4, 3)))},
	{"1*2+3", ast.NewBinaryOperator(3, ast.OperatorAddition,
		ast.NewBinaryOperator(1, ast.OperatorMultiplication, ast.NewInt32(0, 1), ast.NewInt32(2, 2)), ast.NewInt32(4, 3))},
	{"1==2+3", ast.NewBinaryOperator(1, ast.OperatorEqual, ast.NewInt32(0, 1),
		ast.NewBinaryOperator(4, ast.OperatorAddition, ast.NewInt32(3, 2), ast.NewInt32(5, 3)))},
	{"1+2==3", ast.NewBinaryOperator(3, ast.OperatorEqual,
		ast.NewBinaryOperator(1, ast.OperatorAddition, ast.NewInt32(0, 1), ast.NewInt32(2, 2)), ast.NewInt32(5, 3))},
	{"(1+2)*3", ast.NewBinaryOperator(5, ast.OperatorMultiplication, ast.NewParentesis(0,
		ast.NewBinaryOperator(2, ast.OperatorAddition, ast.NewInt32(1, 1), ast.NewInt32(3, 2))), ast.NewInt32(6, 3))},
	{"1*(2+3)", ast.NewBinaryOperator(1, ast.OperatorMultiplication, ast.NewInt32(0, 1), ast.NewParentesis(2,
		ast.NewBinaryOperator(4, ast.OperatorAddition, ast.NewInt32(3, 2), ast.NewInt32(5, 3))))},
	{"a()*1", ast.NewBinaryOperator(3, ast.OperatorMultiplication,
		ast.NewCall(1, ast.NewIdentifier(0, "a"), []ast.Expression{}), ast.NewInt32(4, 1))},
	{"1*a()", ast.NewBinaryOperator(1, ast.OperatorMultiplication,
		ast.NewInt32(0, 1), ast.NewCall(3, ast.NewIdentifier(2, "a"), []ast.Expression{}))},
	{"a[1]*2", ast.NewBinaryOperator(4, ast.OperatorMultiplication,
		ast.NewIndex(1, ast.NewIdentifier(0, "a"), ast.NewInt32(2, 1)), ast.NewInt32(5, 2))},
	{"1*a[2]", ast.NewBinaryOperator(1, ast.OperatorMultiplication,
		ast.NewInt32(0, 1), ast.NewIndex(3, ast.NewIdentifier(2, "a"), ast.NewInt32(4, 2)))},
	{"a[1+2]", ast.NewIndex(1, ast.NewIdentifier(0, "a"),
		ast.NewBinaryOperator(3, ast.OperatorAddition, ast.NewInt32(2, 1), ast.NewInt32(4, 2)))},
	{"a[b(1)]", ast.NewIndex(1, ast.NewIdentifier(0, "a"), ast.NewCall(3,
		ast.NewIdentifier(2, "b"), []ast.Expression{ast.NewInt32(4, 1)}))},
	{"a(b[1])", ast.NewCall(1, ast.NewIdentifier(0, "a"), []ast.Expression{
		ast.NewIndex(3, ast.NewIdentifier(2, "b"), ast.NewInt32(4, 1))})},
	{"a.b*c", ast.NewBinaryOperator(3, ast.OperatorMultiplication, ast.NewSelector(1, ast.NewIdentifier(0, "a"), "b"),
		ast.NewIdentifier(4, "c"))},
	{"a*b.c", ast.NewBinaryOperator(1, ast.OperatorMultiplication, ast.NewIdentifier(0, "a"),
		ast.NewSelector(3, ast.NewIdentifier(2, "b"), "c"))},
	{"a.b(c)", ast.NewCall(3, ast.NewSelector(1, ast.NewIdentifier(0, "a"), "b"), []ast.Expression{ast.NewIdentifier(4, "c")})},
	{"1\t+\n2", ast.NewBinaryOperator(2, ast.OperatorAddition, ast.NewInt32(0, 1), ast.NewInt32(4, 2))},
	{"1\t\r +\n\r\n\r\t 2", ast.NewBinaryOperator(4, ast.OperatorAddition, ast.NewInt32(0, 1), ast.NewInt32(11, 2))},
	{"a(\n\t1\t,\n2\t)", ast.NewCall(1, ast.NewIdentifier(0, "a"), []ast.Expression{ast.NewInt32(4, 1), ast.NewInt32(8, 2)})},
	{"a\t\r ()", ast.NewCall(4, ast.NewIdentifier(0, "a"), []ast.Expression{})},
	{"a[\n\t1\t]", ast.NewIndex(1, ast.NewIdentifier(0, "a"), ast.NewInt32(4, 1))},
	{"a\t\r [1]", ast.NewIndex(4, ast.NewIdentifier(0, "a"), ast.NewInt32(5, 1))},
}

var treeTests = []struct {
	src  string
	node ast.Node
}{
	{"", ast.NewTree(nil)},
	{"a", ast.NewTree([]ast.Node{ast.NewText(0, "a")})},
	{"{{a}}", ast.NewTree([]ast.Node{ast.NewShow(0, ast.NewIdentifier(2, "a"), nil, ast.ContextHTML)})},
	{"a{{b}}", ast.NewTree([]ast.Node{
		ast.NewText(0, "a"), ast.NewShow(1, ast.NewIdentifier(3, "b"), nil, ast.ContextHTML)})},
	{"{{a}}b", ast.NewTree([]ast.Node{
		ast.NewShow(0, ast.NewIdentifier(2, "a"), nil, ast.ContextHTML), ast.NewText(5, "b")})},
	{"a{{b}}c", ast.NewTree([]ast.Node{
		ast.NewText(0, "a"), ast.NewShow(1, ast.NewIdentifier(3, "b"), nil, ast.ContextHTML), ast.NewText(6, "c")})},
	{"{% var a = 1 %}", ast.NewTree([]ast.Node{
		ast.NewVar(0, ast.NewIdentifier(8, "a"), ast.NewInt32(12, 1))})},
	{"{% a = 2 %}", ast.NewTree([]ast.Node{
		ast.NewAssignment(0, ast.NewIdentifier(4, "a"), ast.NewInt32(8, 2))})},
	{"{% show a %}{% end %}", ast.NewTree([]ast.Node{
		ast.NewShow(0, ast.NewIdentifier(8, "a"), nil, ast.ContextHTML)})},
	{"{% show a %}b{% end %}", ast.NewTree([]ast.Node{
		ast.NewShow(0, ast.NewIdentifier(8, "a"), ast.NewText(12, "b"), ast.ContextHTML)})},
	{"{% for a %}b{% end %}", ast.NewTree([]ast.Node{
		ast.NewFor(0, ast.NewIdentifier(7, "a"), []ast.Node{ast.NewText(11, "b")})})},
	{"{% if a %}b{% end %}", ast.NewTree([]ast.Node{
		ast.NewIf(0, ast.NewIdentifier(6, "a"), []ast.Node{ast.NewText(10, "b")})})},
	{"{% extend \"/a.b\" %}", ast.NewTree([]ast.Node{ast.NewExtend(0, "/a.b")})},
	{"{% include \"/a.b\" %}", ast.NewTree([]ast.Node{ast.NewInclude(0, "/a.b", nil)})},
	{"{% region \"a\" %}b{% end %}", ast.NewTree([]ast.Node{
		ast.NewRegion(0, "a", []ast.Node{ast.NewText(10, "b")})})},
}

var pageTests = map[string]struct {
	src  string
	tree *ast.Tree
}{
	"/simple.html": {
		"<!DOCTYPE html>\n<html>\n<head><title>{{ title }}</title></head>\n<body>{{ content }}</body>\n</html>",
		ast.NewTree([]ast.Node{
			ast.NewText(0, "<!DOCTYPE html>\n<html>\n<head><title>"),
			ast.NewShow(36, ast.NewIdentifier(39, "title"), nil, ast.ContextHTML),
			ast.NewText(47, "</title></head>\n<body>"),
			ast.NewShow(69, ast.NewIdentifier(72, "content"), nil, ast.ContextHTML),
			ast.NewText(82, "</body>\n</html>"),
		}),
	},
	"/simple2.html": {
		"<!DOCTYPE html>\n<html>\n<body>{% include \"/include2.html\" %}</body>\n</html>",
		ast.NewTree([]ast.Node{
			ast.NewText(0, "<!DOCTYPE html>\n<html>\n<body>"),
			ast.NewInclude(29, "/include2.html", []ast.Node{
				ast.NewText(0, "<div>"),
				ast.NewShow(5, ast.NewIdentifier(8, "content"), nil, ast.ContextHTML),
				ast.NewText(18, "</div>"),
			}),
			ast.NewText(59, "</body>\n</html>"),
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
		node, _, err := parseExpr(lex)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
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
		node, err := Parse(strings.NewReader(tree.src))
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

func sourceFunc(path string) (io.Reader, error) {
	return strings.NewReader(pageTests[path].src), nil
}

func TestPages(t *testing.T) {
	var parser = NewParser(sourceFunc)
	// simple.html
	var p = pageTests["/simple.html"]
	var tree, err = parser.Parse("/simple.html")
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
	if n1.Pos()-p != n2.Pos() {
		return fmt.Errorf("unexpected position %d, expecting %d", n1.Pos()-2, n2.Pos())
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
	case *ast.Int32:
		nn2, ok := n2.(*ast.Int32)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Value != nn2.Value {
			return fmt.Errorf("unexpected %d, expecting %d", nn1.Value, nn2.Value)
		}
	case *ast.Int64:
		nn2, ok := n2.(*ast.Int64)
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
	case *ast.Show:
		nn2, ok := n2.(*ast.Show)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", nn1, nn2)
		}
		err := equals(nn1.Expr, nn2.Expr, p)
		if err != nil {
			return err
		}
		if nn1.Text == nil && nn2.Text != nil {
			return fmt.Errorf("unexpected nil, expecting %#v", nn2)
		}
		if nn1.Text != nil && nn2.Text == nil {
			return fmt.Errorf("unexpected %#v, expecting nil", nn1)
		}
		if nn1.Text != nil && nn2.Text != nil {
			err = equals(nn1.Text, nn2.Text, p)
			if err != nil {
				return err
			}
		}
		if nn1.Context != nn2.Context {
			return fmt.Errorf("unexpected context %d, expecting %d", nn1.Context, nn2.Context)
		}
	}
	return nil
}
