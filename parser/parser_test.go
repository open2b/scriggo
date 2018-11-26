// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"
	"testing"

	"open2b/template/ast"
)

func p(line, column, start, end int) *ast.Position {
	return &ast.Position{line, column, start, end}
}

var treeTests = []struct {
	src  string
	node ast.Node
}{
	{"", ast.NewTree("", nil, ast.ContextHTML)},
	{"a", ast.NewTree("", []ast.Node{ast.NewText(p(1, 1, 0, 0), "a")}, ast.ContextHTML)},
	{"{{a}}", ast.NewTree("", []ast.Node{ast.NewValue(p(1, 1, 0, 4), ast.NewIdentifier(p(1, 3, 2, 2), "a"), ast.ContextHTML)}, ast.ContextHTML)},
	{"a{{b}}", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 0), "a"), ast.NewValue(p(1, 2, 1, 5), ast.NewIdentifier(p(1, 4, 3, 3), "b"), ast.ContextHTML)}, ast.ContextHTML)},
	{"{{a}}b", ast.NewTree("", []ast.Node{
		ast.NewValue(p(1, 1, 0, 4), ast.NewIdentifier(p(1, 3, 2, 2), "a"), ast.ContextHTML), ast.NewText(p(1, 6, 5, 5), "b")}, ast.ContextHTML)},
	{"a{{b}}c", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 0), "a"), ast.NewValue(p(1, 2, 1, 5), ast.NewIdentifier(p(1, 4, 3, 3), "b"), ast.ContextHTML),
		ast.NewText(p(1, 7, 6, 6), "c")}, ast.ContextHTML)},
	{"<a href=\"/{{ a }}/b\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 8), "<a href=\""), ast.NewURL(p(1, 10, 9, 18), "a", "href", []ast.Node{
			ast.NewText(p(1, 10, 9, 9), "/"),
			ast.NewValue(p(1, 11, 10, 16), ast.NewIdentifier(p(1, 14, 13, 13), "a"), ast.ContextAttribute),
			ast.NewText(p(1, 18, 17, 18), "/b"),
		}),
		ast.NewText(p(1, 20, 19, 20), "\">"),
	}, ast.ContextHTML)},
	{"<a href=\"\n\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 8), "<a href=\""), ast.NewURL(p(1, 10, 9, 9), "a", "href", []ast.Node{
			ast.NewText(p(1, 10, 9, 9), "\n"),
		}),
		ast.NewText(p(2, 1, 10, 11), "\">"),
	}, ast.ContextHTML)},
	{"<div {{ a }}>", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 4), "<div "), ast.NewValue(p(1, 6, 5, 11),
			ast.NewIdentifier(p(1, 9, 8, 8), "a"), ast.ContextTag), ast.NewText(p(1, 13, 12, 12), ">"),
	}, ast.ContextHTML)},
	{"<div{{ a }}>", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 3), "<div"), ast.NewValue(p(1, 5, 4, 10),
			ast.NewIdentifier(p(1, 8, 7, 7), "a"), ast.ContextTag), ast.NewText(p(1, 12, 11, 11), ">"),
	}, ast.ContextHTML)},
	{"<div 本=\"{{ class }}\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 9), "<div 本=\""), ast.NewValue(p(1, 9, 10, 20),
			ast.NewIdentifier(p(1, 12, 13, 17), "class"), ast.ContextAttribute), ast.NewText(p(1, 20, 21, 22), "\">"),
	}, ast.ContextHTML)},
	{"<div \"{{ class }}\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 5), "<div \""), ast.NewValue(p(1, 7, 6, 16),
			ast.NewIdentifier(p(1, 10, 9, 13), "class"), ast.ContextAttribute), ast.NewText(p(1, 18, 17, 18), "\">"),
	}, ast.ContextHTML)},
	{"{% var a = 1 %}", ast.NewTree("", []ast.Node{
		ast.NewVar(p(1, 1, 0, 14), ast.NewIdentifier(p(1, 8, 7, 7), "a"), ast.NewInt(p(1, 13, 11, 11), 1))}, ast.ContextHTML)},
	{"{% a = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 10), ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewInt(p(1, 8, 7, 7), 2))}, ast.ContextHTML)},
	{"{% show a %}", ast.NewTree("", []ast.Node{
		ast.NewShowMacro(p(1, 1, 0, 11), nil, ast.NewIdentifier(p(1, 8, 7, 7), "a"), nil, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% show a(b,c) %}", ast.NewTree("", []ast.Node{
		ast.NewShowMacro(p(1, 1, 0, 16), nil, ast.NewIdentifier(p(1, 8, 7, 7), "a"), []ast.Expression{
			ast.NewIdentifier(p(1, 11, 10, 10), "b"), ast.NewIdentifier(p(1, 13, 12, 12), "c")}, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% for v in e %}b{% end for %}", ast.NewTree("", []ast.Node{ast.NewFor(p(1, 1, 0, 15),
		nil, ast.NewIdentifier(p(1, 8, 7, 7), "v"), ast.NewIdentifier(p(1, 13, 12, 12), "e"), nil, []ast.Node{ast.NewText(p(1, 17, 16, 16), "b")})}, ast.ContextHTML)},
	{"{% for i, v in e %}b{% end %}", ast.NewTree("", []ast.Node{ast.NewFor(p(1, 1, 0, 18),
		ast.NewIdentifier(p(1, 8, 7, 7), "i"), ast.NewIdentifier(p(1, 11, 10, 10), "v"), ast.NewIdentifier(p(1, 16, 15, 15), "e"), nil,
		[]ast.Node{ast.NewText(p(1, 20, 19, 19), "b")})}, ast.ContextHTML)},
	{"{% for v in e %}{% break %}{% end %}", ast.NewTree("", []ast.Node{ast.NewFor(p(1, 1, 0, 15),
		nil, ast.NewIdentifier(p(1, 8, 7, 7), "v"), ast.NewIdentifier(p(1, 13, 12, 12), "e"), nil,
		[]ast.Node{ast.NewBreak(p(1, 17, 16, 26))})}, ast.ContextHTML)},
	{"{% for v in e %}{% continue %}{% end %}", ast.NewTree("", []ast.Node{ast.NewFor(p(1, 1, 0, 15),
		nil, ast.NewIdentifier(p(1, 8, 7, 7), "v"), ast.NewIdentifier(p(1, 13, 12, 12), "e"), nil,
		[]ast.Node{ast.NewContinue(p(1, 17, 16, 29))})}, ast.ContextHTML)},
	{"{% if a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 9), ast.NewIdentifier(p(1, 7, 6, 6), "a"), []ast.Node{ast.NewText(p(1, 11, 10, 10), "b")}, nil)}, ast.ContextHTML)},
	{"{% if a %}b{% else %}c{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 9), ast.NewIdentifier(p(1, 7, 6, 6), "a"),
			[]ast.Node{ast.NewText(p(1, 11, 10, 10), "b")},
			[]ast.Node{ast.NewText(p(1, 22, 21, 21), "c")})}, ast.ContextHTML)},
	{"{% if a %}\nb{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 9), ast.NewIdentifier(p(1, 7, 6, 6), "a"), []ast.Node{ast.NewText(p(1, 11, 10, 11), "b")}, nil)}, ast.ContextHTML)},
	{"{% if a %}\nb\n{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 9), ast.NewIdentifier(p(1, 7, 6, 6), "a"), []ast.Node{ast.NewText(p(1, 11, 10, 12), "b\n")}, nil)}, ast.ContextHTML)},
	{"  {% if a %} \nb\n  {% end %} \t", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 1), ""),
		ast.NewIf(p(1, 3, 2, 11), ast.NewIdentifier(p(1, 9, 8, 8), "a"), []ast.Node{ast.NewText(p(1, 13, 12, 17), "b\n")}, nil),
		ast.NewText(p(3, 12, 27, 28), "")}, ast.ContextHTML)},
	{"{% extend \"/a.b\" %}", ast.NewTree("", []ast.Node{ast.NewExtend(p(1, 1, 0, 18), "/a.b", ast.ContextHTML)}, ast.ContextHTML)},
	{"{% show \"/a.b\" %}", ast.NewTree("", []ast.Node{ast.NewShowPath(p(1, 1, 0, 16), "/a.b", ast.ContextHTML)}, ast.ContextHTML)},
	{"{% extend \"a.e\" %}{% macro b %}c{% end macro %}", ast.NewTree("", []ast.Node{
		ast.NewExtend(p(1, 1, 0, 17), "a.e", ast.ContextHTML), ast.NewMacro(p(1, 19, 18, 30), ast.NewIdentifier(p(1, 28, 27, 27), "b"),
			nil, []ast.Node{ast.NewText(p(1, 32, 31, 31), "c")}, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% extend \"a.e\" %}{% macro b(c,d) %}txt{% end macro %}", ast.NewTree("", []ast.Node{
		ast.NewExtend(p(1, 1, 0, 17), "a.e", ast.ContextHTML), ast.NewMacro(p(1, 19, 18, 35), ast.NewIdentifier(p(1, 28, 27, 27), "b"),
			[]*ast.Identifier{ast.NewIdentifier(p(1, 30, 29, 29), "c"), ast.NewIdentifier(p(1, 32, 31, 31), "d")},
			[]ast.Node{ast.NewText(p(1, 37, 36, 38), "txt")}, ast.ContextHTML)}, ast.ContextHTML)},
	{"{# comment\ncomment #}", ast.NewTree("", []ast.Node{ast.NewComment(p(1, 1, 0, 20), " comment\ncomment ")}, ast.ContextHTML)},
}

func pageTests() map[string]struct {
	src  string
	tree *ast.Tree
} {
	var showPath = ast.NewShowPath(p(3, 7, 29, 55), "/include2.html", ast.ContextHTML)
	showPath.Tree = ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 4), "<div>"),
		ast.NewValue(p(1, 6, 5, 17), ast.NewIdentifier(p(1, 9, 8, 14), "content"), ast.ContextHTML),
		ast.NewText(p(1, 19, 18, 23), "</div>"),
	}, ast.ContextHTML)
	return map[string]struct {
		src  string
		tree *ast.Tree
	}{
		"/simple.html": {
			"<!DOCTYPE html>\n<html>\n<head><title>{{ title }}</title></head>\n<body>{{ content }}</body>\n</html>",
			ast.NewTree("", []ast.Node{
				ast.NewText(p(1, 1, 0, 35), "<!DOCTYPE html>\n<html>\n<head><title>"),
				ast.NewValue(p(3, 14, 36, 46), ast.NewIdentifier(p(3, 17, 39, 43), "title"), ast.ContextHTML),
				ast.NewText(p(3, 25, 47, 68), "</title></head>\n<body>"),
				ast.NewValue(p(4, 7, 69, 81), ast.NewIdentifier(p(4, 10, 72, 78), "content"), ast.ContextHTML),
				ast.NewText(p(4, 20, 82, 96), "</body>\n</html>"),
			}, ast.ContextHTML),
		},
		"/simple2.html": {
			"<!DOCTYPE html>\n<html>\n<body>{% show \"/include2.html\" %}</body>\n</html>",
			ast.NewTree("", []ast.Node{
				ast.NewText(p(1, 1, 0, 28), "<!DOCTYPE html>\n<html>\n<body>"),
				showPath,
				ast.NewText(p(3, 34, 56, 70), "</body>\n</html>"),
			}, ast.ContextHTML),
		},
		"/include2.inc": {
			"<div>{{ content }}</div>",
			nil,
		},
	}
}

func TestTrees(t *testing.T) {
	for _, tree := range treeTests {
		node, err := Parse([]byte(tree.src), ast.ContextHTML)
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

type testsReader map[string]struct {
	src  string
	tree *ast.Tree
}

func (tests testsReader) Read(path string, ctx ast.Context) (*ast.Tree, error) {
	return Parse([]byte(tests[path].src), ctx)
}

func TestPages(t *testing.T) {
	tests := pageTests()
	// simple.html
	parser := NewParser(testsReader(tests))
	p := tests["/simple.html"]
	tree, err := parser.Parse("/simple.html", ast.ContextHTML)
	if err != nil {
		t.Errorf("source: %q, %s\n", p.src, err)
	}
	err = equals(tree, p.tree, 0)
	if err != nil {
		t.Errorf("source: %q, %s\n", p.src, err)
	}
	// simple2.html
	p = tests["/simple2.html"]
	tree, err = parser.Parse("/simple2.html", ast.ContextHTML)
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
		}
		return fmt.Errorf("unexpected node %#v, expecting nil", n1)
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
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
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
	case *ast.Text:
		nn2, ok := n2.(*ast.Text)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		txt1 := nn1.Text[nn1.Cut.Left:nn1.Cut.Right]
		txt2 := nn2.Text[nn2.Cut.Left:nn2.Cut.Right]
		if txt1 != txt2 {
			return fmt.Errorf("unexpected %q, expecting %q", txt1, txt2)
		}
	case *ast.URL:
		nn2, ok := n2.(*ast.URL)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Tag != nn2.Tag {
			return fmt.Errorf("unexpected tag %q, expecting %q", nn1.Tag, nn2.Tag)
		}
		if nn1.Attribute != nn2.Attribute {
			return fmt.Errorf("unexpected attribute %q, expecting %q", nn1.Attribute, nn2.Attribute)
		}
		if len(nn1.Value) != len(nn2.Value) {
			return fmt.Errorf("unexpected value nodes len %d, expecting %d", len(nn1.Value), len(nn2.Value))
		}
		for i, node := range nn1.Value {
			err := equals(node, nn2.Value[i], p)
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
			return fmt.Errorf("unexpected %q, expecting %q", nn1.Value, nn2.Value)
		}
	case *ast.Number:
		nn2, ok := n2.(*ast.Number)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Value.Cmp(nn2.Value) != 0 {
			return fmt.Errorf("unexpected %s, expecting %s", nn1.Value.String(), nn2.Value.String())
		}
	case *ast.String:
		nn2, ok := n2.(*ast.String)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Text != nn2.Text {
			return fmt.Errorf("unexpected %q, expecting %q", nn1.Text, nn2.Text)
		}
	case *ast.Parentesis:
		nn2, ok := n2.(*ast.Parentesis)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Expr, nn2.Expr, p)
		if err != nil {
			return err
		}
	case *ast.UnaryOperator:
		nn2, ok := n2.(*ast.UnaryOperator)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
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
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
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
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
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
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
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
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
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
	case *ast.Value:
		nn2, ok := n2.(*ast.Value)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Expr, nn2.Expr, p)
		if err != nil {
			return err
		}
		if nn1.Context != nn2.Context {
			return fmt.Errorf("unexpected context %s, expecting %s", nn1.Context, nn2.Context)
		}
	case *ast.If:
		nn2, ok := n2.(*ast.If)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Expr, nn2.Expr, p)
		if err != nil {
			return err
		}
		if len(nn1.Then) != len(nn2.Then) {
			return fmt.Errorf("unexpected then nodes len %d, expecting %d", len(nn1.Then), len(nn2.Then))
		}
		for i, node := range nn1.Then {
			err := equals(node, nn2.Then[i], p)
			if err != nil {
				return err
			}
		}
		if nn1.Else == nil && nn2.Else != nil {
			return fmt.Errorf("unexpected else nil, expecting not nil")
		}
		if nn1.Else != nil && nn2.Else == nil {
			return fmt.Errorf("unexpected else not nil, expecting nil")
		}
		if nn1.Else != nil {
			if len(nn1.Else) != len(nn2.Else) {
				return fmt.Errorf("unexpected else nodes len %d, expecting %d", len(nn1.Else), len(nn2.Else))
			}
			for i, node := range nn1.Else {
				err := equals(node, nn2.Else[i], p)
				if err != nil {
					return err
				}
			}
		}
	case *ast.For:
		nn2, ok := n2.(*ast.For)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Expr1, nn2.Expr1, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Expr2, nn2.Expr2, p)
		if err != nil {
			return err
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
	case *ast.Macro:
		nn2, ok := n2.(*ast.Macro)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Ident, nn2.Ident, p)
		if err != nil {
			return err
		}
		if len(nn1.Parameters) != len(nn2.Parameters) {
			return fmt.Errorf("unexpected arguments len %d, expecting %d", len(nn1.Parameters), len(nn2.Parameters))
		}
		for i, parameter := range nn1.Parameters {
			err := equals(parameter, nn2.Parameters[i], p)
			if err != nil {
				return err
			}
		}
		if len(nn1.Body) != len(nn2.Body) {
			return fmt.Errorf("unexpected body len %d, expecting %d", len(nn1.Body), len(nn2.Body))
		}
		for i, node := range nn1.Body {
			err := equals(node, nn2.Body[i], p)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
