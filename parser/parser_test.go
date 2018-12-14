// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"bytes"
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
	{"a", ast.NewTree("", []ast.Node{ast.NewText(p(1, 1, 0, 0), []byte("a"), ast.Cut{})}, ast.ContextHTML)},
	{"{{a}}", ast.NewTree("", []ast.Node{ast.NewValue(p(1, 1, 0, 4), ast.NewIdentifier(p(1, 3, 2, 2), "a"), ast.ContextHTML)}, ast.ContextHTML)},
	{"a{{b}}", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 0), []byte("a"), ast.Cut{}), ast.NewValue(p(1, 2, 1, 5), ast.NewIdentifier(p(1, 4, 3, 3), "b"), ast.ContextHTML)}, ast.ContextHTML)},
	{"{{a}}b", ast.NewTree("", []ast.Node{
		ast.NewValue(p(1, 1, 0, 4), ast.NewIdentifier(p(1, 3, 2, 2), "a"), ast.ContextHTML), ast.NewText(p(1, 6, 5, 5), []byte("b"), ast.Cut{})}, ast.ContextHTML)},
	{"a{{b}}c", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 0), []byte("a"), ast.Cut{}), ast.NewValue(p(1, 2, 1, 5), ast.NewIdentifier(p(1, 4, 3, 3), "b"), ast.ContextHTML),
		ast.NewText(p(1, 7, 6, 6), []byte("c"), ast.Cut{})}, ast.ContextHTML)},
	{"<a href=\"/{{ a }}/b\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 8), []byte("<a href=\""), ast.Cut{}), ast.NewURL(p(1, 10, 9, 18), "a", "href", []ast.Node{
			ast.NewText(p(1, 10, 9, 9), []byte("/"), ast.Cut{}),
			ast.NewValue(p(1, 11, 10, 16), ast.NewIdentifier(p(1, 14, 13, 13), "a"), ast.ContextAttribute),
			ast.NewText(p(1, 18, 17, 18), []byte("/b"), ast.Cut{}),
		}),
		ast.NewText(p(1, 20, 19, 20), []byte("\">"), ast.Cut{}),
	}, ast.ContextHTML)},
	{"<a href=\"\n\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 8), []byte("<a href=\""), ast.Cut{}), ast.NewURL(p(1, 10, 9, 9), "a", "href", []ast.Node{
			ast.NewText(p(1, 10, 9, 9), []byte("\n"), ast.Cut{}),
		}),
		ast.NewText(p(2, 1, 10, 11), []byte("\">"), ast.Cut{}),
	}, ast.ContextHTML)},
	{"<div {{ a }}>", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 4), []byte("<div "), ast.Cut{}), ast.NewValue(p(1, 6, 5, 11),
			ast.NewIdentifier(p(1, 9, 8, 8), "a"), ast.ContextTag), ast.NewText(p(1, 13, 12, 12), []byte(">"), ast.Cut{}),
	}, ast.ContextHTML)},
	{"<div{{ a }}>", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 3), []byte("<div"), ast.Cut{}), ast.NewValue(p(1, 5, 4, 10),
			ast.NewIdentifier(p(1, 8, 7, 7), "a"), ast.ContextTag), ast.NewText(p(1, 12, 11, 11), []byte(">"), ast.Cut{}),
	}, ast.ContextHTML)},
	{"<div 本=\"{{ class }}\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 9), []byte("<div 本=\""), ast.Cut{}), ast.NewValue(p(1, 9, 10, 20),
			ast.NewIdentifier(p(1, 12, 13, 17), "class"), ast.ContextAttribute), ast.NewText(p(1, 20, 21, 22), []byte("\">"), ast.Cut{}),
	}, ast.ContextHTML)},
	{"<div a=/{{ class }}\"{{ class }}>", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 7), []byte("<div a=/"), ast.Cut{}), ast.NewValue(p(1, 9, 8, 18),
			ast.NewIdentifier(p(1, 12, 11, 15), "class"), ast.ContextUnquotedAttribute),
		ast.NewText(p(1, 20, 19, 19), []byte("\""), ast.Cut{}),
		ast.NewValue(p(1, 21, 20, 30),
			ast.NewIdentifier(p(1, 24, 23, 27), "class"), ast.ContextUnquotedAttribute),
		ast.NewText(p(1, 32, 31, 31), []byte(">"), ast.Cut{}),
	}, ast.ContextHTML)},
	{"{% for article in articles %}\n<div>{{ article.title }}</div>\n{% end %}",
		ast.NewTree("articles.txt", []ast.Node{
			ast.NewFor(
				&ast.Position{Line: 1, Column: 1, Start: 0, End: 69},
				nil,
				ast.NewIdentifier(&ast.Position{Line: 1, Column: 8, Start: 7, End: 13}, "article"),
				ast.NewIdentifier(&ast.Position{Line: 1, Column: 19, Start: 18, End: 25}, "articles"),
				nil,
				[]ast.Node{
					ast.NewText(&ast.Position{Line: 1, Column: 30, Start: 29, End: 34}, []byte("\n<div>"), ast.Cut{1, 0}),
					ast.NewValue(
						&ast.Position{Line: 2, Column: 6, Start: 35, End: 53},
						ast.NewSelector(
							&ast.Position{Line: 2, Column: 16, Start: 38, End: 50},
							ast.NewIdentifier(
								&ast.Position{Line: 2, Column: 9, Start: 38, End: 44},
								"article",
							),
							"title"),
						ast.ContextHTML),
					ast.NewText(&ast.Position{Line: 2, Column: 25, Start: 54, End: 60}, []byte("</div>\n"), ast.Cut{}),
				},
			),
		}, ast.ContextHTML)},
	{"<div \"{{ class }}\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 5), []byte("<div \""), ast.Cut{}), ast.NewValue(p(1, 7, 6, 16),
			ast.NewIdentifier(p(1, 10, 9, 13), "class"), ast.ContextTag), ast.NewText(p(1, 18, 17, 18), []byte("\">"), ast.Cut{}),
	}, ast.ContextHTML)},
	{"{% a := 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 11), ast.NewIdentifier(p(1, 4, 3, 3), "a"), nil,
			ast.NewInt(p(1, 9, 8, 8), 1), true)}, ast.ContextHTML)},
	{"{% a = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 10), ast.NewIdentifier(p(1, 4, 3, 3), "a"), nil, ast.NewInt(p(1, 8, 7, 7), 2), false)}, ast.ContextHTML)},
	{"{% a, ok := b.c %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 17), ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewIdentifier(p(1, 7, 6, 7), "ok"),
			ast.NewSelector(p(1, 14, 12, 14),
				ast.NewIdentifier(p(1, 16, 15, 15), "b"), "c"), true)}, ast.ContextHTML)},
	{"{% show a %}", ast.NewTree("", []ast.Node{
		ast.NewShowMacro(p(1, 1, 0, 11), nil, ast.NewIdentifier(p(1, 8, 7, 7), "a"), nil, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% show a(b,c) %}", ast.NewTree("", []ast.Node{
		ast.NewShowMacro(p(1, 1, 0, 16), nil, ast.NewIdentifier(p(1, 8, 7, 7), "a"), []ast.Expression{
			ast.NewIdentifier(p(1, 11, 10, 10), "b"), ast.NewIdentifier(p(1, 13, 12, 12), "c")}, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% for v in e %}b{% end for %}", ast.NewTree("", []ast.Node{ast.NewFor(p(1, 1, 0, 29),
		nil, ast.NewIdentifier(p(1, 8, 7, 7), "v"), ast.NewIdentifier(p(1, 13, 12, 12), "e"), nil, []ast.Node{ast.NewText(p(1, 17, 16, 16), []byte("b"), ast.Cut{})})}, ast.ContextHTML)},
	{"{% for i, v in e %}b{% end %}", ast.NewTree("", []ast.Node{ast.NewFor(p(1, 1, 0, 28),
		ast.NewIdentifier(p(1, 8, 7, 7), "i"), ast.NewIdentifier(p(1, 11, 10, 10), "v"), ast.NewIdentifier(p(1, 16, 15, 15), "e"), nil,
		[]ast.Node{ast.NewText(p(1, 20, 19, 19), []byte("b"), ast.Cut{})})}, ast.ContextHTML)},
	{"{% for v in e %}{% break %}{% end %}", ast.NewTree("", []ast.Node{ast.NewFor(p(1, 1, 0, 35),
		nil, ast.NewIdentifier(p(1, 8, 7, 7), "v"), ast.NewIdentifier(p(1, 13, 12, 12), "e"), nil,
		[]ast.Node{ast.NewBreak(p(1, 17, 16, 26))})}, ast.ContextHTML)},
	{"{% for v in e %}{% continue %}{% end %}", ast.NewTree("", []ast.Node{ast.NewFor(p(1, 1, 0, 38),
		nil, ast.NewIdentifier(p(1, 8, 7, 7), "v"), ast.NewIdentifier(p(1, 13, 12, 12), "e"), nil,
		[]ast.Node{ast.NewContinue(p(1, 17, 16, 29))})}, ast.ContextHTML)},
	{"{% if a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 22), nil, ast.NewIdentifier(p(1, 7, 6, 6), "a"), []ast.Node{ast.NewText(p(1, 11, 10, 10), []byte("b"), ast.Cut{})}, nil)}, ast.ContextHTML)},
	{"{% if a %}b{% else %}c{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 30), nil, ast.NewIdentifier(p(1, 7, 6, 6), "a"),
			[]ast.Node{ast.NewText(p(1, 11, 10, 10), []byte("b"), ast.Cut{})},
			[]ast.Node{ast.NewText(p(1, 22, 21, 21), []byte("c"), ast.Cut{})})}, ast.ContextHTML)},
	{"{% if a %}\nb{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 20), nil, ast.NewIdentifier(p(1, 7, 6, 6), "a"), []ast.Node{ast.NewText(p(1, 11, 10, 11), []byte("\nb"), ast.Cut{1, 0})}, nil)}, ast.ContextHTML)},
	{"{% if a %}\nb\n{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 21), nil, ast.NewIdentifier(p(1, 7, 6, 6), "a"), []ast.Node{ast.NewText(p(1, 11, 10, 12), []byte("\nb\n"), ast.Cut{1, 0})}, nil)}, ast.ContextHTML)},
	{"  {% if a %} \nb\n  {% end %} \t", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 1), []byte("  "), ast.Cut{0, 2}),
		ast.NewIf(p(1, 3, 2, 26), nil, ast.NewIdentifier(p(1, 9, 8, 8), "a"), []ast.Node{ast.NewText(p(1, 13, 12, 17), []byte(" \nb\n  "), ast.Cut{2, 2})}, nil),
		ast.NewText(p(3, 12, 27, 28), []byte(" \t"), ast.Cut{2, 0})}, ast.ContextHTML)},
	{"{% if a = b; a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 29),
			ast.NewAssignment(p(1, 7, 6, 10), ast.NewIdentifier(p(1, 7, 6, 6), "a"), nil, ast.NewIdentifier(p(1, 11, 10, 10), "b"), false),
			ast.NewIdentifier(p(1, 14, 13, 13), "a"), []ast.Node{ast.NewText(p(1, 18, 17, 17), []byte("b"), ast.Cut{})}, nil)}, ast.ContextHTML)},
	{"{% if a := b; a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 30),
			ast.NewAssignment(p(1, 7, 6, 11), ast.NewIdentifier(p(1, 7, 6, 6), "a"), nil, ast.NewIdentifier(p(1, 12, 11, 11), "b"), true),
			ast.NewIdentifier(p(1, 15, 14, 14), "a"), []ast.Node{ast.NewText(p(1, 19, 18, 18), []byte("b"), ast.Cut{})}, nil)}, ast.ContextHTML)},
	{"{% if a, ok := b.c; a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 36),
			ast.NewAssignment(p(1, 7, 6, 17),
				ast.NewIdentifier(p(1, 7, 6, 6), "a"), ast.NewIdentifier(p(1, 10, 9, 10), "ok"),
				ast.NewSelector(p(1, 17, 15, 17), ast.NewIdentifier(p(1, 16, 15, 15), "b"), "c"), true),
			ast.NewIdentifier(p(1, 21, 20, 20), "a"), []ast.Node{ast.NewText(p(1, 25, 24, 24), []byte("b"), ast.Cut{})}, nil)}, ast.ContextHTML)},
	{"{% extends \"/a.b\" %}", ast.NewTree("", []ast.Node{ast.NewExtends(p(1, 1, 0, 19), "/a.b", ast.ContextHTML)}, ast.ContextHTML)},
	{"{% include \"/a.b\" %}", ast.NewTree("", []ast.Node{ast.NewInclude(p(1, 1, 0, 19), "/a.b", ast.ContextHTML)}, ast.ContextHTML)},
	{"{% extends \"a.e\" %}{% macro b %}c{% end macro %}", ast.NewTree("", []ast.Node{
		ast.NewExtends(p(1, 1, 0, 18), "a.e", ast.ContextHTML), ast.NewMacro(p(1, 20, 19, 47), ast.NewIdentifier(p(1, 29, 28, 28), "b"),
			nil, []ast.Node{ast.NewText(p(1, 33, 32, 32), []byte("c"), ast.Cut{})}, false, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% extends \"a.e\" %}{% macro b(c,d) %}txt{% end macro %}", ast.NewTree("", []ast.Node{
		ast.NewExtends(p(1, 1, 0, 18), "a.e", ast.ContextHTML), ast.NewMacro(p(1, 20, 19, 54), ast.NewIdentifier(p(1, 29, 28, 28), "b"),
			[]*ast.Identifier{ast.NewIdentifier(p(1, 31, 30, 30), "c"), ast.NewIdentifier(p(1, 33, 32, 32), "d")},
			[]ast.Node{ast.NewText(p(1, 38, 37, 39), []byte("txt"), ast.Cut{})}, false, ast.ContextHTML)}, ast.ContextHTML)},
	{"{# comment\ncomment #}", ast.NewTree("", []ast.Node{ast.NewComment(p(1, 1, 0, 20), " comment\ncomment ")}, ast.ContextHTML)},
	{"{% macro a(b) %}c{% end macro %}", ast.NewTree("", []ast.Node{
		ast.NewMacro(p(1, 1, 0, 31), ast.NewIdentifier(p(1, 10, 9, 9), "a"),
			[]*ast.Identifier{ast.NewIdentifier(p(1, 12, 11, 11), "b")},
			[]ast.Node{ast.NewText(p(1, 17, 16, 16), []byte("c"), ast.Cut{})}, false, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% macro a(b, c...) %}d{% end macro %}", ast.NewTree("", []ast.Node{
		ast.NewMacro(p(1, 1, 0, 37), ast.NewIdentifier(p(1, 10, 9, 9), "a"),
			[]*ast.Identifier{ast.NewIdentifier(p(1, 12, 11, 11), "b"), ast.NewIdentifier(p(1, 15, 14, 14), "c")},
			[]ast.Node{ast.NewText(p(1, 23, 22, 22), []byte("d"), ast.Cut{})}, true, ast.ContextHTML)}, ast.ContextHTML)},
}

func pageTests() map[string]struct {
	src  string
	tree *ast.Tree
} {
	var include = ast.NewInclude(p(3, 7, 29, 58), "/include2.html", ast.ContextHTML)
	include.Tree = ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 4), []byte("<div>"), ast.Cut{}),
		ast.NewValue(p(1, 6, 5, 17), ast.NewIdentifier(p(1, 9, 8, 14), "content"), ast.ContextHTML),
		ast.NewText(p(1, 19, 18, 23), []byte("</div>"), ast.Cut{}),
	}, ast.ContextHTML)
	return map[string]struct {
		src  string
		tree *ast.Tree
	}{
		"/simple.html": {
			"<!DOCTYPE html>\n<html>\n<head><title>{{ title }}</title></head>\n<body>{{ content }}</body>\n</html>",
			ast.NewTree("", []ast.Node{
				ast.NewText(p(1, 1, 0, 35), []byte("<!DOCTYPE html>\n<html>\n<head><title>"), ast.Cut{}),
				ast.NewValue(p(3, 14, 36, 46), ast.NewIdentifier(p(3, 17, 39, 43), "title"), ast.ContextHTML),
				ast.NewText(p(3, 25, 47, 68), []byte("</title></head>\n<body>"), ast.Cut{}),
				ast.NewValue(p(4, 7, 69, 81), ast.NewIdentifier(p(4, 10, 72, 78), "content"), ast.ContextHTML),
				ast.NewText(p(4, 20, 82, 96), []byte("</body>\n</html>"), ast.Cut{}),
			}, ast.ContextHTML),
		},
		"/simple2.html": {
			"<!DOCTYPE html>\n<html>\n<body>{% include \"/include2.html\" %}</body>\n</html>",
			ast.NewTree("", []ast.Node{
				ast.NewText(p(1, 1, 0, 28), []byte("<!DOCTYPE html>\n<html>\n<body>"), ast.Cut{}),
				include,
				ast.NewText(p(3, 37, 59, 73), []byte("</body>\n</html>"), ast.Cut{}),
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
		node, err := ParseSource([]byte(tree.src), ast.ContextHTML)
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
	return ParseSource([]byte(tests[path].src), ctx)
}

func TestPages(t *testing.T) {
	tests := pageTests()
	// simple.html
	parser := New(testsReader(tests))
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
		if !bytes.Equal(nn1.Text, nn2.Text) {
			return fmt.Errorf("unexpected text %q, expecting %q", nn1.Text, nn2.Text)
		}
		if nn1.Cut.Left != nn2.Cut.Left || nn1.Cut.Right != nn2.Cut.Right {
			return fmt.Errorf("unexpected cut (%d,%d), expecting (%d,%d)",
				nn1.Cut.Left, nn1.Cut.Right, nn2.Cut.Left, nn2.Cut.Right)
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
	case *ast.Slice:
		nn2, ok := n2.(*ast.Slice)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if len(nn1.Elements) != len(nn2.Elements) {
			return fmt.Errorf("unexpected elements len %d, expecting %d", len(nn1.Elements), len(nn2.Elements))
		}
		for i, arg := range nn1.Elements {
			err := equals(arg, nn2.Elements[i], p)
			if err != nil {
				return err
			}
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
	case *ast.Assignment:
		nn2, ok := n2.(*ast.Assignment)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Ident, nn2.Ident, p)
		if err != nil {
			return err
		}
		if nn1.Ident2 != nil {
			err := equals(nn1.Ident2, nn2.Ident2, p)
			if err != nil {
				return err
			}
		}
		err = equals(nn1.Expr, nn2.Expr, p)
		if err != nil {
			return err
		}
		if nn1.Declaration != nn2.Declaration {
			return fmt.Errorf("unexpected declaretion %t, expecting %t", nn1.Declaration, nn2.Declaration)
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
	case *ast.Slicing:
		nn2, ok := n2.(*ast.Slicing)
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
		err := equals(nn1.Condition, nn2.Condition, p)
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
		if nn1.Assignment == nil && nn2.Assignment != nil {
			return fmt.Errorf("unexpected assignment nil, expecting not nil")
		}
		if nn1.Assignment != nil && nn2.Assignment == nil {
			return fmt.Errorf("unexpected assignment not nil, expecting nil")
		}
		if nn1.Assignment != nil {
			err = equals(nn1.Assignment, nn2.Assignment, p)
			if err != nil {
				return err
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
		if nn1.IsVariadic != nn2.IsVariadic {
			return fmt.Errorf("unexpected is variadic %t, expecting %t", nn1.IsVariadic, nn2.IsVariadic)
		}
	}
	return nil
}
