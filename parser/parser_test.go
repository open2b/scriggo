// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"bytes"
	"fmt"
	"reflect"
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
	{"{% for %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewFor(&ast.Position{Line: 1, Column: 1, Start: 0, End: 17}, nil, nil, nil, nil),
		}, ast.ContextHTML)},
	{"{% for a %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewFor(&ast.Position{Line: 1, Column: 1, Start: 0, End: 19},
				nil, ast.NewIdentifier(&ast.Position{Line: 1, Column: 8, Start: 7, End: 7}, "a"), nil,
				nil),
		}, ast.ContextHTML)},
	{"{% for ; ; %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewFor(&ast.Position{Line: 1, Column: 1, Start: 0, End: 21},
				nil, nil, nil, nil),
		}, ast.ContextHTML)},
	{"{% for i := 0; ; %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewFor(&ast.Position{Line: 1, Column: 1, Start: 0, End: 27},
				ast.NewAssignment(&ast.Position{Line: 1, Column: 8, Start: 7, End: 12},
					[]ast.Expression{ast.NewIdentifier(&ast.Position{Line: 1, Column: 8, Start: 7, End: 7}, "i")},
					ast.AssignmentDeclaration, []ast.Expression{ast.NewInt(&ast.Position{Line: 1, Column: 13, Start: 12, End: 12}, 0)}),
				nil, nil, nil),
		}, ast.ContextHTML)},
	{"{% for ; true ; %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewFor(&ast.Position{Line: 1, Column: 1, Start: 0, End: 26},
				nil, ast.NewIdentifier(&ast.Position{Line: 1, Column: 10, Start: 9, End: 12}, "true"), nil, nil),
		}, ast.ContextHTML)},
	{"{% for i := 0; i < 10; i = i + 1 %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewFor(&ast.Position{Line: 1, Column: 1, Start: 0, End: 43},
				ast.NewAssignment(&ast.Position{Line: 1, Column: 8, Start: 7, End: 12},
					[]ast.Expression{ast.NewIdentifier(&ast.Position{Line: 1, Column: 8, Start: 7, End: 7}, "i")},
					ast.AssignmentDeclaration, []ast.Expression{ast.NewInt(&ast.Position{Line: 1, Column: 13, Start: 12, End: 12}, 0)}),
				ast.NewBinaryOperator(
					&ast.Position{Line: 1, Column: 18, Start: 15, End: 20},
					ast.OperatorLess,
					ast.NewIdentifier(&ast.Position{Line: 1, Column: 16, Start: 15, End: 15}, "i"),
					ast.NewInt(&ast.Position{Line: 1, Column: 20, Start: 19, End: 20}, 10)),
				ast.NewAssignment(&ast.Position{Line: 1, Column: 24, Start: 23, End: 31},
					[]ast.Expression{ast.NewIdentifier(&ast.Position{Line: 1, Column: 24, Start: 23, End: 23}, "i")},
					ast.AssignmentSimple,
					[]ast.Expression{ast.NewBinaryOperator(
						&ast.Position{Line: 1, Column: 30, Start: 27, End: 31},
						ast.OperatorAddition,
						ast.NewIdentifier(&ast.Position{Line: 1, Column: 28, Start: 27, End: 27}, "i"),
						ast.NewInt(&ast.Position{Line: 1, Column: 32, Start: 31, End: 31}, 1))}),
				nil),
		}, ast.ContextHTML)},
	{"{% for article in articles %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewForRange(
				&ast.Position{Line: 1, Column: 1, Start: 0, End: 37},
				ast.NewAssignment(&ast.Position{Line: 1, Column: 8, Start: 7, End: 25}, []ast.Expression{
					ast.NewIdentifier(&ast.Position{Line: 1, Column: 8, Start: 7, End: 7}, "_"),
					ast.NewIdentifier(&ast.Position{Line: 1, Column: 8, Start: 7, End: 13}, "article")},
					ast.AssignmentDeclaration, []ast.Expression{ast.NewIdentifier(&ast.Position{Line: 1, Column: 19, Start: 18, End: 25}, "articles")}),
				nil),
		}, ast.ContextHTML)},
	{"{% for range articles %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewForRange(
				&ast.Position{Line: 1, Column: 1, Start: 0, End: 32},
				ast.NewAssignment(&ast.Position{Line: 1, Column: 8, Start: 7, End: 20}, nil,
					ast.AssignmentSimple, []ast.Expression{ast.NewIdentifier(&ast.Position{Line: 1, Column: 14, Start: 13, End: 20}, "articles")}),
				nil),
		}, ast.ContextHTML)},
	{"{% for i := range articles %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewForRange(
				&ast.Position{Line: 1, Column: 1, Start: 0, End: 37},
				ast.NewAssignment(&ast.Position{Line: 1, Column: 8, Start: 7, End: 25},
					[]ast.Expression{ast.NewIdentifier(&ast.Position{Line: 1, Column: 8, Start: 7, End: 7}, "i")},
					ast.AssignmentDeclaration, []ast.Expression{ast.NewIdentifier(&ast.Position{Line: 1, Column: 19, Start: 18, End: 25}, "articles")}),
				nil),
		}, ast.ContextHTML)},
	{"{% for i, article := range articles %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewForRange(
				&ast.Position{Line: 1, Column: 1, Start: 0, End: 46},
				ast.NewAssignment(&ast.Position{Line: 1, Column: 8, Start: 7, End: 34}, []ast.Expression{
					ast.NewIdentifier(&ast.Position{Line: 1, Column: 8, Start: 7, End: 7}, "i"),
					ast.NewIdentifier(&ast.Position{Line: 1, Column: 11, Start: 10, End: 16}, "article")},
					ast.AssignmentDeclaration, []ast.Expression{ast.NewIdentifier(&ast.Position{Line: 1, Column: 28, Start: 27, End: 34}, "articles")}),
				nil),
		}, ast.ContextHTML)},
	{"{% for article in articles %}\n<div>{{ article.title }}</div>\n{% end %}",
		ast.NewTree("articles.txt", []ast.Node{
			ast.NewForRange(
				&ast.Position{Line: 1, Column: 1, Start: 0, End: 69},
				ast.NewAssignment(&ast.Position{Line: 1, Column: 8, Start: 7, End: 25}, []ast.Expression{
					ast.NewIdentifier(&ast.Position{Line: 1, Column: 8, Start: 7, End: 7}, "_"),
					ast.NewIdentifier(&ast.Position{Line: 1, Column: 8, Start: 7, End: 13}, "article")},
					ast.AssignmentDeclaration, []ast.Expression{ast.NewIdentifier(&ast.Position{Line: 1, Column: 19, Start: 18, End: 25}, "articles")}),
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
	{"{% switch x %}{% case 1 %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 1, 0, 34),
				nil,
				ast.NewIdentifier(p(1, 11, 10, 10), "x"),
				[]*ast.Case{
					ast.NewCase(
						p(1, 15, 14, 25),
						[]ast.Expression{
							ast.NewInt(p(1, 23, 22, 22), 1),
						},
						nil,
						false,
					),
				},
			),
		}, ast.ContextHTML),
	},
	{"{% switch x %}{% case 1 %}something{% fallthrough %}{% case 2 %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 1, 0, 72),
				nil,
				ast.NewIdentifier(p(1, 11, 10, 10), "x"),
				[]*ast.Case{
					ast.NewCase(
						p(1, 15, 14, 25),
						[]ast.Expression{
							ast.NewInt(p(1, 23, 22, 22), 1),
						},
						[]ast.Node{
							ast.NewText(p(1, 27, 26, 34), []byte("something"), ast.Cut{}),
						},
						true,
					),
					ast.NewCase(
						p(1, 53, 52, 63),
						[]ast.Expression{
							ast.NewInt(p(1, 61, 60, 60), 2),
						},
						nil,
						false,
					),
				},
			),
		}, ast.ContextHTML),
	},
	{"{% switch x := 2; x * 4 %}{% case 1 %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 1, 0, 46),
				ast.NewAssignment( // x := 2
					p(1, 11, 10, 15),
					[]ast.Expression{ast.NewIdentifier(p(1, 11, 10, 10), "x")},
					ast.AssignmentDeclaration,
					[]ast.Expression{ast.NewInt(p(1, 16, 15, 15), 2)},
				),
				ast.NewBinaryOperator( // x * 4
					p(1, 21, 18, 22),
					ast.OperatorMultiplication,
					ast.NewIdentifier(p(1, 19, 18, 18), "x"),
					ast.NewInt(p(1, 23, 22, 22), 4),
				),
				[]*ast.Case{
					ast.NewCase(
						p(1, 27, 26, 37),
						[]ast.Expression{
							ast.NewInt(p(1, 35, 34, 34), 1),
						},
						nil,
						false,
					),
				},
			),
		}, ast.ContextHTML),
	},
	{"{% switch %}{% case 1 < 6 %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 1, 0, 36),
				nil,
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 13, 12, 27),
						[]ast.Expression{
							ast.NewBinaryOperator( // 1 < 6
								p(1, 23, 20, 24),
								ast.OperatorLess,
								ast.NewInt(p(1, 21, 20, 20), 1),
								ast.NewInt(p(1, 25, 24, 24), 6),
							),
						},
						nil,
						false,
					),
				},
			),
		}, ast.ContextHTML),
	},
	{"{% switch %}{% default %}{% break %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 1, 0, 44),
				nil,
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 13, 12, 24),
						nil,
						[]ast.Node{
							ast.NewBreak(p(1, 26, 25, 35)),
						},
						false,
					),
				},
			),
		}, ast.ContextHTML),
	},
	{"{% switch %}{% case 1 < 6, x == sum(2, -3) %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 1, 0, 53),
				nil,
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 13, 12, 44),
						[]ast.Expression{
							ast.NewBinaryOperator( // 1 < 6
								p(1, 23, 20, 24),
								ast.OperatorLess,
								ast.NewInt(p(1, 21, 20, 20), 1),
								ast.NewInt(p(1, 25, 24, 24), 6),
							),
							ast.NewBinaryOperator(p(1, 30, 27, 41), // x == sum(2, 3)
								ast.OperatorEqual,
								ast.NewIdentifier(p(1, 28, 27, 27), "x"),
								ast.NewCall(
									p(1, 36, 32, 41),
									ast.NewIdentifier(p(1, 33, 32, 34), "sum"),
									[]ast.Expression{
										ast.NewInt(p(1, 37, 36, 36), 2),
										ast.NewInt(p(1, 40, 39, 40), -3),
									},
								),
							),
						},
						nil,
						false,
					),
				},
			),
		}, ast.ContextHTML),
	},
	{"{% switch x %}{% case 1 %}is one{% case 2 %}is two{% default %}is a number{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 1, 0, 82),
				nil,
				ast.NewIdentifier(p(1, 11, 10, 10), "x"),
				[]*ast.Case{
					ast.NewCase(
						p(1, 15, 14, 25),
						[]ast.Expression{
							ast.NewInt(p(1, 23, 22, 22), 1),
						},
						[]ast.Node{
							ast.NewText(p(1, 27, 26, 31), []byte("is one"), ast.Cut{}),
						},
						false,
					),
					ast.NewCase(
						p(1, 33, 32, 43),
						[]ast.Expression{
							ast.NewInt(p(1, 41, 40, 40), 2),
						},
						[]ast.Node{
							ast.NewText(p(1, 45, 44, 49), []byte("is two"), ast.Cut{}),
						},
						false,
					),
					ast.NewCase(
						p(1, 51, 50, 62),
						nil,
						[]ast.Node{
							ast.NewText(p(1, 64, 63, 73), []byte("is a number"), ast.Cut{}),
						},
						false,
					),
				},
			),
		}, ast.ContextHTML),
	},
	{"{% switch x.(type) %}{% case int, float %}is a number{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewTypeSwitch(p(1, 1, 0, 61),
				nil,
				ast.NewAssignment(
					p(1, 12, 10, 17),
					[]ast.Expression{
						ast.NewIdentifier(p(1, 12, 10, 17), "_"),
					},
					ast.AssignmentSimple,
					[]ast.Expression{
						ast.NewTypeAssertion(
							p(1, 12, 10, 17),
							ast.NewIdentifier(p(1, 11, 10, 10), "x"),
							nil,
						),
					},
				),
				[]*ast.Case{
					ast.NewCase(p(1, 22, 21, 41),
						[]ast.Expression{
							ast.NewIdentifier(p(1, 30, 29, 31), "int"),
							ast.NewIdentifier(p(1, 35, 34, 38), "float"),
						},
						[]ast.Node{
							ast.NewText(p(1, 43, 42, 52), []byte("is a number"), ast.Cut{}),
						},
						false,
					),
				},
			),
		}, ast.ContextHTML),
	},
	{"{% switch v := x.(type) %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewTypeSwitch(p(1, 1, 0, 34),
				nil,
				ast.NewAssignment(
					p(1, 11, 10, 22),
					[]ast.Expression{
						ast.NewIdentifier(p(1, 11, 10, 10), "v"),
					},
					ast.AssignmentDeclaration,
					[]ast.Expression{
						ast.NewTypeAssertion(
							p(1, 17, 15, 22),
							ast.NewIdentifier(p(1, 16, 15, 15), "x"),
							nil,
						),
					},
				),
				nil,
			),
		}, ast.ContextHTML),
	},
	{"<div \"{{ class }}\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 5), []byte("<div \""), ast.Cut{}), ast.NewValue(p(1, 7, 6, 16),
			ast.NewIdentifier(p(1, 10, 9, 13), "class"), ast.ContextTag), ast.NewText(p(1, 18, 17, 18), []byte("\">"), ast.Cut{}),
	}, ast.ContextHTML)},
	{"{% a := 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 11), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentDeclaration, []ast.Expression{ast.NewInt(p(1, 9, 8, 8), 1)})}, ast.ContextHTML)},
	{"{% a = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 10), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")}, ast.AssignmentSimple,
			[]ast.Expression{ast.NewInt(p(1, 8, 7, 7), 2)})}, ast.ContextHTML)},
	{"{% _ = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 10), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "_")}, ast.AssignmentSimple,
			[]ast.Expression{ast.NewInt(p(1, 8, 7, 7), 2)})}, ast.ContextHTML)},
	{"{% a.b = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 12), []ast.Expression{ast.NewSelector(p(1, 5, 3, 5), ast.NewIdentifier(p(1, 4, 3, 3), "a"), "b")},
			ast.AssignmentSimple, []ast.Expression{ast.NewInt(p(1, 10, 9, 9), 2)})}, ast.ContextHTML)},
	{"{% a[\"b\"] = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 15), []ast.Expression{ast.NewIndex(p(1, 5, 3, 8), ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewString(p(1, 6, 5, 7), "b"))},
			ast.AssignmentSimple, []ast.Expression{ast.NewInt(p(1, 13, 12, 12), 2)})}, ast.ContextHTML)},
	{"{% a[6] = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 13), []ast.Expression{ast.NewIndex(p(1, 5, 3, 6), ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewInt(p(1, 6, 5, 5), 6))},
			ast.AssignmentSimple, []ast.Expression{ast.NewInt(p(1, 11, 10, 10), 2)})}, ast.ContextHTML)},
	{"{% a, b := 1, 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 17),
			[]ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewIdentifier(p(1, 7, 6, 6), "b")},
			ast.AssignmentDeclaration,
			[]ast.Expression{ast.NewInt(p(1, 12, 11, 11), 1), ast.NewInt(p(1, 15, 14, 14), 2)})}, ast.ContextHTML)},
	{"{% a, b, c = 1, 2, 3 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 22),
			[]ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewIdentifier(p(1, 7, 6, 6), "b"), ast.NewIdentifier(p(1, 10, 9, 9), "c")},
			ast.AssignmentSimple,
			[]ast.Expression{ast.NewInt(p(1, 14, 13, 13), 1), ast.NewInt(p(1, 17, 16, 16), 2), ast.NewInt(p(1, 20, 19, 19), 3)})}, ast.ContextHTML)},
	{"{% a, ok := b.c %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 17), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewIdentifier(p(1, 7, 6, 7), "ok")},
			ast.AssignmentDeclaration, []ast.Expression{ast.NewSelector(p(1, 14, 12, 14),
				ast.NewIdentifier(p(1, 16, 15, 15), "b"), "c")})}, ast.ContextHTML)},
	{"{% a += 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 11), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentAddition, []ast.Expression{ast.NewInt(p(1, 9, 8, 8), 1)})}, ast.ContextHTML)},
	{"{% a -= 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 11), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentSubtraction, []ast.Expression{ast.NewInt(p(1, 9, 8, 8), 1)})}, ast.ContextHTML)},
	{"{% a *= 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 11), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentMultiplication, []ast.Expression{ast.NewInt(p(1, 9, 8, 8), 1)})}, ast.ContextHTML)},
	{"{% a /= 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 11), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentDivision, []ast.Expression{ast.NewInt(p(1, 9, 8, 8), 1)})}, ast.ContextHTML)},
	{"{% a %= 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 11), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentModulo, []ast.Expression{ast.NewInt(p(1, 9, 8, 8), 1)})}, ast.ContextHTML)},
	{"{% show a %}", ast.NewTree("", []ast.Node{
		ast.NewShowMacro(p(1, 1, 0, 11), nil, ast.NewIdentifier(p(1, 8, 7, 7), "a"), nil, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% show a(b,c) %}", ast.NewTree("", []ast.Node{
		ast.NewShowMacro(p(1, 1, 0, 16), nil, ast.NewIdentifier(p(1, 8, 7, 7), "a"), []ast.Expression{
			ast.NewIdentifier(p(1, 11, 10, 10), "b"), ast.NewIdentifier(p(1, 13, 12, 12), "c")}, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% for v in e %}b{% end for %}", ast.NewTree("", []ast.Node{
		ast.NewForRange(p(1, 1, 0, 29), ast.NewAssignment(p(1, 8, 7, 12), []ast.Expression{
			ast.NewIdentifier(p(1, 8, 7, 7), "_"), ast.NewIdentifier(p(1, 8, 7, 7), "v")},
			ast.AssignmentDeclaration, []ast.Expression{ast.NewIdentifier(p(1, 13, 12, 12), "e")}),
			[]ast.Node{ast.NewText(p(1, 17, 16, 16), []byte("b"), ast.Cut{})})}, ast.ContextHTML)},
	{"{% for v in e %}{% break %}{% end %}", ast.NewTree("", []ast.Node{
		ast.NewForRange(p(1, 1, 0, 35), ast.NewAssignment(p(1, 8, 7, 12), []ast.Expression{
			ast.NewIdentifier(p(1, 8, 7, 7), "_"),
			ast.NewIdentifier(p(1, 8, 7, 7), "v")},
			ast.AssignmentDeclaration, []ast.Expression{ast.NewIdentifier(p(1, 13, 12, 12), "e")}),
			[]ast.Node{ast.NewBreak(p(1, 17, 16, 26))})}, ast.ContextHTML)},
	{"{% for v in e %}{% continue %}{% end %}", ast.NewTree("", []ast.Node{
		ast.NewForRange(p(1, 1, 0, 38), ast.NewAssignment(p(1, 8, 7, 12), []ast.Expression{
			ast.NewIdentifier(p(1, 8, 7, 7), "_"),
			ast.NewIdentifier(p(1, 8, 7, 7), "v")},
			ast.AssignmentDeclaration, []ast.Expression{ast.NewIdentifier(p(1, 13, 12, 12), "e")}),
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
			ast.NewAssignment(p(1, 7, 6, 10), []ast.Expression{ast.NewIdentifier(p(1, 7, 6, 6), "a")}, ast.AssignmentSimple,
				[]ast.Expression{ast.NewIdentifier(p(1, 11, 10, 10), "b")}),
			ast.NewIdentifier(p(1, 14, 13, 13), "a"), []ast.Node{ast.NewText(p(1, 18, 17, 17), []byte("b"), ast.Cut{})}, nil)}, ast.ContextHTML)},
	{"{% if a := b; a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 30),
			ast.NewAssignment(p(1, 7, 6, 11), []ast.Expression{ast.NewIdentifier(p(1, 7, 6, 6), "a")}, ast.AssignmentDeclaration,
				[]ast.Expression{ast.NewIdentifier(p(1, 12, 11, 11), "b")}),
			ast.NewIdentifier(p(1, 15, 14, 14), "a"), []ast.Node{ast.NewText(p(1, 19, 18, 18), []byte("b"), ast.Cut{})}, nil)}, ast.ContextHTML)},
	{"{% if a, ok := b.c; a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 36),
			ast.NewAssignment(p(1, 7, 6, 17),
				[]ast.Expression{ast.NewIdentifier(p(1, 7, 6, 6), "a"), ast.NewIdentifier(p(1, 10, 9, 10), "ok")},
				ast.AssignmentDeclaration, []ast.Expression{ast.NewSelector(p(1, 17, 15, 17), ast.NewIdentifier(p(1, 16, 15, 15), "b"), "c")}),
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
	if n1 == nil || n2 == nil {
		if n1 == nil {
			return fmt.Errorf("unexpected node nil, expecting %#v", n2)
		}
		return fmt.Errorf("unexpected node %#v, expecting nil", n1)
	}
	rv1 := reflect.ValueOf(n1)
	rv2 := reflect.ValueOf(n2)
	if rv1.IsNil() && rv2.IsNil() {
		if rv1.Type() != rv1.Type() {
			return fmt.Errorf("unexpected node %#v, expecting %#v", n1, n2)
		}
		return nil
	}
	if rv1.IsNil() || rv2.IsNil() {
		if rv1.IsNil() {
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
		if len(nn1.Variables) != len(nn2.Variables) {
			return fmt.Errorf("unexpected variables len %d, expecting %d", len(nn1.Variables), len(nn2.Variables))
		}
		for i, v := range nn1.Variables {
			err := equals(v, nn2.Variables[i], p)
			if err != nil {
				return err
			}
		}
		if nn1.Type != nn2.Type {
			return fmt.Errorf("unexpected assignment type %d, expecting %d", nn1.Type, nn2.Type)
		}
		if len(nn1.Values) != len(nn2.Values) {
			return fmt.Errorf("unexpected values len %d, expecting %d", len(nn1.Values), len(nn2.Values))
		}
		for i, v := range nn1.Values {
			err := equals(v, nn2.Values[i], p)
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
		err := equals(nn1.Init, nn2.Init, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Condition, nn2.Condition, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Post, nn2.Post, p)
		if err != nil {
			return err
		}
		if len(nn1.Body) != len(nn2.Body) {
			return fmt.Errorf("unexpected nodes len %d, expecting %d", len(nn1.Body), len(nn2.Body))
		}
		for i, node := range nn1.Body {
			err := equals(node, nn2.Body[i], p)
			if err != nil {
				return err
			}
		}
	case *ast.ForRange:
		nn2, ok := n2.(*ast.ForRange)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Assignment, nn2.Assignment, p)
		if err != nil {
			return err
		}
		if len(nn1.Body) != len(nn2.Body) {
			return fmt.Errorf("unexpected nodes len %d, expecting %d", len(nn1.Body), len(nn2.Body))
		}
		for i, node := range nn1.Body {
			err := equals(node, nn2.Body[i], p)
			if err != nil {
				return err
			}
		}

	case *ast.Switch:
		nn2, ok := n2.(*ast.Switch)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Init != nil && nn2.Init == nil {
			return fmt.Errorf("unexpected assignment %#v, expecting nil assignment", nn1.Init)
		}
		if nn1.Init == nil && nn2.Init != nil {
			return fmt.Errorf("unexpected nil assignment, expecting assignment %#v", nn2.Init)
		}
		if nn1.Expr != nil && nn2.Expr == nil {
			return fmt.Errorf("unexpected expression %#v, expecting nil", nn1.Expr)
		}
		if nn1.Expr == nil && nn2.Expr != nil {
			return fmt.Errorf("unexpected nil expression, expecting expression %#v", nn2.Expr)
		}
		err := equals(nn1.Expr, nn2.Expr, p)
		if err != nil {
			return fmt.Errorf("expression: %s", err)
		}
		err = equals(nn1.Init, nn2.Init, p)
		if err != nil {
			return fmt.Errorf("assignment: %s", err)
		}
		if nn1.Cases == nil && nn2.Cases != nil {
			return fmt.Errorf("unexpected nil body, expecting %#v", nn2.Cases)
		}
		if nn1.Cases != nil && nn2.Cases == nil {
			return fmt.Errorf("unexpected body %#v, expecting nil", nn1.Cases)
		}
		if len(nn1.Cases) != len(nn2.Cases) {
			return fmt.Errorf("unexpected body len %d, expecting %d", len(nn1.Cases), len(nn2.Cases))
		}
		for i, c := range nn1.Cases {
			err := equals(c, nn2.Cases[i], p)
			if err != nil {
				return fmt.Errorf("case #%d: %s", i+1, err)
			}
		}

	case *ast.TypeSwitch:
		nn2, ok := n2.(*ast.TypeSwitch)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Init != nil && nn2.Init == nil {
			return fmt.Errorf("unexpected assignment %#v, expecting nil assignment", nn1.Init)
		}
		if nn1.Init == nil && nn2.Init != nil {
			return fmt.Errorf("unexpected nil assignment, expecting assignment %#v", nn2.Init)
		}
		if nn1.Assignment != nil && nn2.Assignment == nil {
			return fmt.Errorf("unexpected expression %#v, expecting nil", nn1.Assignment)
		}
		if nn1.Assignment == nil && nn2.Assignment != nil {
			return fmt.Errorf("unexpected nil expression, expecting expression %#v", nn2.Assignment)
		}
		err := equals(nn1.Assignment, nn2.Assignment, p)
		if err != nil {
			return fmt.Errorf("expression: %s", err)
		}
		err = equals(nn1.Init, nn2.Init, p)
		if err != nil {
			return fmt.Errorf("assignment: %s", err)
		}
		if nn1.Cases == nil && nn2.Cases != nil {
			return fmt.Errorf("unexpected nil body, expecting %#v", nn2.Cases)
		}
		if nn1.Cases != nil && nn2.Cases == nil {
			return fmt.Errorf("unexpected body %#v, expecting nil", nn1.Cases)
		}
		if len(nn1.Cases) != len(nn2.Cases) {
			return fmt.Errorf("unexpected body len %d, expecting %d", len(nn1.Cases), len(nn2.Cases))
		}
		for i, c := range nn1.Cases {
			err := equals(c, nn2.Cases[i], p)
			if err != nil {
				return fmt.Errorf("case #%d: %s", i+1, err)
			}
		}

	case *ast.Case:
		nn2, ok := n2.(*ast.Case)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if len(nn1.ExprList) != len(nn2.ExprList) {
			return fmt.Errorf("unexpected expressions nodes len %d, expected %d", len(nn1.ExprList), len(nn2.ExprList))
		}
		for i, expr := range nn1.ExprList {
			err := equals(expr, nn2.ExprList[i], p)
			if err != nil {
				return fmt.Errorf("expressions: %s", err)
			}
		}
		if len(nn1.Body) != len(nn2.Body) {
			return fmt.Errorf("unexpected Body nodes len %d, expected %d", len(nn1.Body), len(nn2.Body))
		}
		for i, expr := range nn1.Body {
			err := equals(expr, nn2.Body[i], p)
			if err != nil {
				return fmt.Errorf("Body: %s", err)
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
	case *ast.TypeAssertion:
		nn2, ok := n2.(*ast.TypeAssertion)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Expr, nn2.Expr, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Type, nn2.Type, p)
		if err != nil {
			return err
		}
	}
	return nil
}
