// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/open2b/scriggo/compiler/ast"
)

// combinedLoaders combines more loaders in one loader.
type combinedLoaders []PackageLoader

func (loaders combinedLoaders) Load(path string) (interface{}, error) {
	for _, loader := range loaders {
		p, err := loader.Load(path)
		if p != nil || err != nil {
			return p, err
		}
	}
	return nil, nil
}

// loaders returns a loader combining more loaders.
func loaders(loaders ...PackageLoader) PackageLoader {
	return combinedLoaders(loaders)
}

// mapStringLoader implements PackageLoader for not predefined packages as a
// map with string values. Paths and sources are respectively the keys and the
// values of the map.
type mapStringLoader map[string]string

func (r mapStringLoader) Load(path string) (interface{}, error) {
	if src, ok := r[path]; ok {
		return strings.NewReader(src), nil
	}
	return nil, nil
}

type predefinedPackages map[string]predefinedPackage

func (pp predefinedPackages) Load(path string) (interface{}, error) {
	p := pp[path]
	return p, nil
}

func p(line, column, start, end int) *ast.Position {
	return &ast.Position{Line: line, Column: column, Start: start, End: end}
}

var goContextTreeTests = []struct {
	src  string
	node ast.Node
}{
	{"", ast.NewTree("", nil, ast.FormatText)},
	{";", ast.NewTree("", nil, ast.FormatText)},
	{"a", ast.NewTree("", []ast.Node{ast.NewIdentifier(p(1, 1, 0, 0), "a")}, ast.FormatText)},
	{"a := 1", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 5), []ast.Expression{ast.NewIdentifier(p(1, 1, 0, 0), "a")},
			ast.AssignmentDeclaration, []ast.Expression{ast.NewBasicLiteral(p(1, 6, 5, 5), ast.IntLiteral, "1")})}, ast.FormatText)},
	{"a -= 1", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 5), []ast.Expression{ast.NewIdentifier(p(1, 1, 0, 0), "a")},
			ast.AssignmentSubtraction, []ast.Expression{ast.NewBasicLiteral(p(1, 6, 5, 5), ast.IntLiteral, "1")})}, ast.FormatText)},
	{"a %= 1", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 5), []ast.Expression{ast.NewIdentifier(p(1, 1, 0, 0), "a")},
			ast.AssignmentModulo, []ast.Expression{ast.NewBasicLiteral(p(1, 6, 5, 5), ast.IntLiteral, "1")})}, ast.FormatText)},
	{"a.b = 2", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 6), []ast.Expression{ast.NewSelector(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), "b")},
			ast.AssignmentSimple, []ast.Expression{ast.NewBasicLiteral(p(1, 7, 6, 6), ast.IntLiteral, "2")})}, ast.FormatText)},
	{"if a {\n\tb\n}\n", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 10), nil, ast.NewIdentifier(p(1, 4, 3, 3), "a"),
			ast.NewBlock(p(1, 6, 5, 10), []ast.Node{ast.NewIdentifier(p(2, 2, 8, 8), "b")}), nil)}, ast.FormatText)},
	{"if a {\t\tb\t}\t", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 10), nil, ast.NewIdentifier(p(1, 4, 3, 3), "a"),
			ast.NewBlock(p(1, 6, 5, 10), []ast.Node{ast.NewIdentifier(p(1, 9, 8, 8), "b")}), nil)}, ast.FormatText)},
	{"if a {\n\tb\n} else {\n\tc\n}\n", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 22), nil, ast.NewIdentifier(p(1, 4, 3, 3), "a"),
			ast.NewBlock(p(1, 6, 5, 10), []ast.Node{ast.NewIdentifier(p(2, 2, 8, 8), "b")}),
			ast.NewBlock(p(3, 8, 17, 22), []ast.Node{ast.NewIdentifier(p(4, 2, 20, 20), "c")}))}, ast.FormatText)},
	{"if a {\n\tb\n} else if c {\n\td\n} else {\n\te\n}\n", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 39), nil, ast.NewIdentifier(p(1, 4, 3, 3), "a"),
			ast.NewBlock(p(1, 6, 5, 10), []ast.Node{ast.NewIdentifier(p(2, 2, 8, 8), "b")}),
			ast.NewIf(p(3, 8, 17, 39), nil, ast.NewIdentifier(p(3, 11, 20, 20), "c"),
				ast.NewBlock(p(3, 13, 22, 27), []ast.Node{ast.NewIdentifier(p(4, 2, 25, 25), "d")}),
				ast.NewBlock(p(5, 8, 34, 39), []ast.Node{ast.NewIdentifier(p(6, 2, 37, 37), "e")})))}, ast.FormatText)},
	{"for _, v := range e {\n\tb\n}\n", ast.NewTree("", []ast.Node{
		ast.NewForRange(p(1, 1, 0, 25), ast.NewAssignment(p(1, 5, 4, 18), []ast.Expression{
			ast.NewIdentifier(p(1, 5, 4, 4), "_"), ast.NewIdentifier(p(1, 8, 7, 7), "v")},
			ast.AssignmentDeclaration, []ast.Expression{ast.NewIdentifier(p(1, 19, 18, 18), "e")}),
			[]ast.Node{ast.NewIdentifier(p(2, 2, 23, 23), "b")})}, ast.FormatText)},
	{"for _ = range []int(nil) { }", ast.NewTree("", []ast.Node{
		ast.NewForRange(p(1, 1, 0, 27),
			ast.NewAssignment(p(1, 5, 4, 23),
				[]ast.Expression{ast.NewIdentifier(p(1, 5, 4, 4), "_")},
				ast.AssignmentSimple,
				[]ast.Expression{
					ast.NewCall(p(1, 20, 14, 23),
						ast.NewSliceType(p(1, 15, 14, 18), ast.NewIdentifier(p(1, 17, 16, 18), "int")),
						[]ast.Expression{ast.NewIdentifier(p(1, 21, 20, 22), "nil")},
						false)}), nil)}, ast.FormatText)},
	{"switch {\n\tdefault:\n}\n", ast.NewTree("", []ast.Node{
		ast.NewSwitch(p(1, 1, 0, 19), nil, nil, nil, []*ast.Case{
			ast.NewCase(p(2, 2, 10, 16), nil, nil)})}, ast.FormatText)},
	{"if x == 5 {}",
		ast.NewTree("", []ast.Node{
			ast.NewIf(p(1, 1, 0, 11), nil,
				ast.NewBinaryOperator(p(1, 6, 3, 8),
					ast.OperatorEqual,
					ast.NewIdentifier(p(1, 4, 3, 3), "x"),
					ast.NewBasicLiteral(p(1, 9, 8, 8), ast.IntLiteral, "5"),
				), nil, nil),
		}, ast.FormatText)},
	{"for _, i := range []int{1,2,3} { }",
		ast.NewTree("", []ast.Node{
			ast.NewForRange(
				p(1, 1, 0, 33),
				ast.NewAssignment(
					p(1, 5, 4, 29),
					[]ast.Expression{
						ast.NewIdentifier(p(1, 5, 4, 4), "_"),
						ast.NewIdentifier(p(1, 8, 7, 7), "i"),
					},
					ast.AssignmentDeclaration,
					[]ast.Expression{
						ast.NewCompositeLiteral(p(1, 24, 18, 29),
							ast.NewSliceType(
								p(1, 19, 18, 22),
								ast.NewIdentifier(p(1, 21, 20, 22), "int"),
							),
							[]ast.KeyValue{
								{nil, ast.NewBasicLiteral(p(1, 25, 24, 24), ast.IntLiteral, "1")},
								{nil, ast.NewBasicLiteral(p(1, 27, 26, 26), ast.IntLiteral, "2")},
								{nil, ast.NewBasicLiteral(p(1, 29, 28, 28), ast.IntLiteral, "3")},
							})}), nil)},
			ast.FormatText)},
	{"var a int",
		ast.NewTree("", []ast.Node{
			ast.NewVar(
				p(1, 1, 0, 8),
				[]*ast.Identifier{
					ast.NewIdentifier(p(1, 5, 4, 4), "a"),
				},
				ast.NewIdentifier(p(1, 7, 6, 8), "int"),
				nil,
			),
		}, ast.FormatText),
	},
	{"var a, b int",
		ast.NewTree("", []ast.Node{
			ast.NewVar(
				p(1, 1, 0, 11),
				[]*ast.Identifier{
					ast.NewIdentifier(p(1, 5, 4, 4), "a"),
					ast.NewIdentifier(p(1, 8, 7, 7), "b"),
				},
				ast.NewIdentifier(p(1, 10, 9, 11), "int"),
				nil,
			),
		}, ast.FormatText),
	},
	{"var a = 4",
		ast.NewTree("", []ast.Node{
			ast.NewVar(
				p(1, 1, 0, 8),
				[]*ast.Identifier{
					ast.NewIdentifier(p(1, 5, 4, 4), "a"),
				},
				nil,
				[]ast.Expression{
					ast.NewBasicLiteral(p(1, 9, 8, 8), ast.IntLiteral, "4"),
				},
			),
		}, ast.FormatText),
	},
	{"var a int = 4",
		ast.NewTree("", []ast.Node{
			ast.NewVar(
				p(1, 1, 0, 12),
				[]*ast.Identifier{
					ast.NewIdentifier(p(1, 5, 4, 4), "a"),
				},
				ast.NewIdentifier(p(1, 7, 6, 8), "int"),
				[]ast.Expression{
					ast.NewBasicLiteral(p(1, 13, 12, 12), ast.IntLiteral, "4"),
				},
			),
		}, ast.FormatText),
	},
	{"var a, b int = 4, 6",
		ast.NewTree("", []ast.Node{
			ast.NewVar(
				p(1, 1, 0, 18),
				[]*ast.Identifier{
					ast.NewIdentifier(p(1, 5, 4, 4), "a"),
					ast.NewIdentifier(p(1, 8, 7, 7), "b"),
				},
				ast.NewIdentifier(p(1, 10, 9, 11), "int"),
				[]ast.Expression{
					ast.NewBasicLiteral(p(1, 16, 15, 15), ast.IntLiteral, "4"),
					ast.NewBasicLiteral(p(1, 19, 18, 18), ast.IntLiteral, "6"),
				},
			),
		}, ast.FormatText),
	},
	{"var (\n\ta int = 3\n\tb = 1.00\n)",
		ast.NewTree("", []ast.Node{
			ast.NewVar(
				p(1, 1, 0, 27),
				[]*ast.Identifier{
					ast.NewIdentifier(p(2, 2, 7, 7), "a"),
				},
				ast.NewIdentifier(p(2, 4, 9, 11), "int"),
				[]ast.Expression{
					ast.NewBasicLiteral(p(2, 10, 15, 15), ast.IntLiteral, "3"),
				},
			),
			ast.NewVar(
				p(1, 1, 0, 27),
				[]*ast.Identifier{
					ast.NewIdentifier(p(3, 2, 18, 18), "b"),
				},
				nil,
				[]ast.Expression{
					ast.NewBasicLiteral(p(3, 6, 22, 25), ast.FloatLiteral, "1.00"),
				},
			),
		}, ast.FormatText),
	},
	{"var f func ()",
		ast.NewTree("", []ast.Node{
			ast.NewVar(
				p(1, 1, 0, 12),
				[]*ast.Identifier{
					ast.NewIdentifier(p(1, 5, 4, 4), "f"),
				},
				ast.NewFuncType(p(1, 7, 6, 12), false, nil, nil, false),
				nil,
			),
		}, ast.FormatText),
	},
	{"var f func (p.T)",
		ast.NewTree("", []ast.Node{
			ast.NewVar(
				p(1, 1, 0, 15),
				[]*ast.Identifier{
					ast.NewIdentifier(p(1, 5, 4, 4), "f"),
				},
				ast.NewFuncType(p(1, 7, 6, 15), false, []*ast.Parameter{
					ast.NewParameter(nil, ast.NewSelector(p(1, 15, 14, 14),
						ast.NewIdentifier(p(1, 13, 12, 12), "p"), "T")),
				}, nil, false),
				nil,
			),
		}, ast.FormatText),
	}, {"var A []T",
		ast.NewTree("", []ast.Node{
			ast.NewVar(
				p(1, 1, 0, 8),
				[]*ast.Identifier{
					ast.NewIdentifier(p(1, 5, 4, 4), "A"),
				},
				ast.NewSliceType(
					p(1, 7, 6, 8),
					ast.NewIdentifier(p(1, 9, 8, 8), "T"),
				),
				nil,
			),
		}, ast.FormatText),
	},
	{"const a = 4",
		ast.NewTree("", []ast.Node{
			ast.NewConst(
				p(1, 1, 0, 10),
				[]*ast.Identifier{
					ast.NewIdentifier(p(1, 7, 6, 6), "a"),
				},
				nil, // no type
				[]ast.Expression{
					ast.NewBasicLiteral(p(1, 11, 10, 10), ast.IntLiteral, "4"),
				},
				0, // iota
			),
		}, ast.FormatText),
	},
	{"const ()", ast.NewTree("", []ast.Node{}, ast.FormatText)},
	{"const (\nA = 42\nB\n)\n", ast.NewTree("", []ast.Node{
		ast.NewConst(
			p(1, 1, 0, 17),
			[]*ast.Identifier{
				ast.NewIdentifier(p(2, 1, 8, 8), "A"),
			},
			nil, // no type
			[]ast.Expression{
				ast.NewBasicLiteral(p(2, 5, 12, 13), ast.IntLiteral, "42"),
			},
			0, // iota
		),
		ast.NewConst(
			p(1, 1, 0, 17),
			[]*ast.Identifier{
				ast.NewIdentifier(p(3, 1, 15, 15), "B"),
			},
			nil, // no type
			[]ast.Expression{
				ast.NewBasicLiteral(p(2, 5, 12, 13), ast.IntLiteral, "42"),
			},
			1, // iota
		),
	}, ast.FormatText)},
	{"{}", ast.NewTree("", []ast.Node{
		ast.NewBlock(p(1, 1, 0, 1), nil),
	}, ast.FormatText)},
	{"type Int int",
		ast.NewTree("", []ast.Node{
			ast.NewTypeDeclaration(
				p(1, 1, 0, 11),
				ast.NewIdentifier(p(1, 6, 5, 7), "Int"),
				ast.NewIdentifier(p(1, 10, 9, 11), "int"),
				false,
			),
		}, ast.FormatText),
	},
	{"type Int []string",
		ast.NewTree("", []ast.Node{
			ast.NewTypeDeclaration(
				p(1, 1, 0, 16),
				ast.NewIdentifier(p(1, 6, 5, 7), "Int"),
				ast.NewSliceType(p(1, 10, 9, 16), ast.NewIdentifier(p(1, 12, 11, 16), "string")),
				false,
			),
		}, ast.FormatText),
	},
	{"type Int = int",
		ast.NewTree("", []ast.Node{
			ast.NewTypeDeclaration(
				p(1, 1, 0, 13),
				ast.NewIdentifier(p(1, 6, 5, 7), "Int"),
				ast.NewIdentifier(p(1, 12, 11, 13), "int"),
				true,
			),
		}, ast.FormatText),
	},
	{"type MyMap = map[string]interface{}",
		ast.NewTree("", []ast.Node{
			ast.NewTypeDeclaration(
				p(1, 1, 0, 34),
				ast.NewIdentifier(p(1, 6, 5, 9), "MyMap"),
				ast.NewMapType(
					p(1, 14, 13, 34),
					ast.NewIdentifier(p(1, 18, 17, 22), "string"),
					ast.NewInterface(p(1, 25, 24, 34)),
				),
				true,
			),
		}, ast.FormatText),
	},
	{"type ( Int int ; String string )",
		ast.NewTree("", []ast.Node{
			ast.NewTypeDeclaration(
				p(1, 1, 0, 31),
				ast.NewIdentifier(p(1, 8, 7, 9), "Int"),
				ast.NewIdentifier(p(1, 12, 11, 13), "int"),
				false,
			),
			ast.NewTypeDeclaration(
				p(1, 1, 0, 31),
				ast.NewIdentifier(p(1, 18, 17, 22), "String"),
				ast.NewIdentifier(p(1, 25, 24, 29), "string"),
				false,
			),
		}, ast.FormatText),
	},
	{"type ( Int = int ; String string )",
		ast.NewTree("", []ast.Node{
			ast.NewTypeDeclaration(
				p(1, 1, 0, 33),
				ast.NewIdentifier(p(1, 8, 7, 9), "Int"),
				ast.NewIdentifier(p(1, 14, 13, 15), "int"),
				true,
			),
			ast.NewTypeDeclaration(
				p(1, 1, 0, 33),
				ast.NewIdentifier(p(1, 20, 19, 24), "String"),
				ast.NewIdentifier(p(1, 27, 26, 31), "string"),
				false,
			),
		}, ast.FormatText),
	},
	{"struct { }",
		ast.NewTree("", []ast.Node{
			ast.NewStructType(
				p(1, 1, 0, 9),
				nil,
			),
		}, ast.FormatText)},
	{"struct { A int }",
		ast.NewTree("", []ast.Node{
			ast.NewStructType(
				p(1, 1, 0, 15),
				[]*ast.Field{
					ast.NewField(
						[]*ast.Identifier{
							ast.NewIdentifier(p(1, 10, 9, 9), "A"),
						},
						ast.NewIdentifier(p(1, 12, 11, 13), "int"),
						"",
					),
				},
			),
		}, ast.FormatText)},
	{"struct { A []int }",
		ast.NewTree("", []ast.Node{
			ast.NewStructType(
				p(1, 1, 0, 17),
				[]*ast.Field{
					ast.NewField(
						[]*ast.Identifier{
							ast.NewIdentifier(p(1, 10, 9, 9), "A"),
						},
						ast.NewSliceType(
							p(1, 12, 11, 15),
							ast.NewIdentifier(p(1, 14, 13, 15), "int"),
						),
						"",
					),
				},
			),
		}, ast.FormatText)},
	{"struct { A struct { C int } }",
		ast.NewTree("", []ast.Node{
			ast.NewStructType(
				p(1, 1, 0, 28),
				[]*ast.Field{
					ast.NewField(
						[]*ast.Identifier{
							ast.NewIdentifier(p(1, 10, 9, 9), "A"),
						},
						ast.NewStructType(
							p(1, 12, 11, 26),
							[]*ast.Field{
								ast.NewField(
									[]*ast.Identifier{
										ast.NewIdentifier(p(1, 21, 20, 20), "C"),
									},
									ast.NewIdentifier(p(1, 23, 22, 24), "int"),
									"",
								),
							},
						),
						"",
					),
				},
			),
		}, ast.FormatText)},
	{"struct { A, B int \"ab\" }",
		ast.NewTree("", []ast.Node{
			ast.NewStructType(
				p(1, 1, 0, 23),
				[]*ast.Field{
					ast.NewField(
						[]*ast.Identifier{
							ast.NewIdentifier(p(1, 10, 9, 9), "A"),
							ast.NewIdentifier(p(1, 13, 12, 12), "B"),
						},
						ast.NewIdentifier(p(1, 15, 14, 16), "int"),
						"ab",
					),
				},
			),
		}, ast.FormatText)},
	{"struct { A, B int ; C, D string }",
		ast.NewTree("", []ast.Node{
			ast.NewStructType(
				p(1, 1, 0, 32),
				[]*ast.Field{
					ast.NewField(
						[]*ast.Identifier{
							ast.NewIdentifier(p(1, 10, 9, 9), "A"),
							ast.NewIdentifier(p(1, 13, 12, 12), "B"),
						},
						ast.NewIdentifier(p(1, 15, 14, 16), "int"),
						"",
					),
					ast.NewField(
						[]*ast.Identifier{
							ast.NewIdentifier(p(1, 21, 20, 20), "C"),
							ast.NewIdentifier(p(1, 24, 23, 23), "D"),
						},
						ast.NewIdentifier(p(1, 26, 25, 30), "string"),
						"",
					),
				},
			),
		}, ast.FormatText)},
	{"struct { A int ; C ; *D }",
		ast.NewTree("", []ast.Node{
			ast.NewStructType(
				p(1, 1, 0, 24),
				[]*ast.Field{
					ast.NewField(
						[]*ast.Identifier{
							ast.NewIdentifier(p(1, 10, 9, 9), "A"),
						},
						ast.NewIdentifier(p(1, 12, 11, 13), "int"),
						"",
					),
					ast.NewField(
						nil,
						ast.NewIdentifier(p(1, 18, 17, 17), "C"),
						"",
					),
					ast.NewField(
						nil,
						ast.NewUnaryOperator(
							p(1, 22, 21, 22),
							ast.OperatorPointer,
							ast.NewIdentifier(p(1, 23, 22, 22), "D"),
						),
						"",
					),
				},
			),
		}, ast.FormatText)},
	{"struct { A int `a` }{ A: 10 }",
		ast.NewTree("", []ast.Node{
			ast.NewCompositeLiteral(
				p(1, 21, 0, 28),
				ast.NewStructType(
					p(1, 1, 0, 19),
					[]*ast.Field{
						ast.NewField(
							[]*ast.Identifier{
								ast.NewIdentifier(p(1, 10, 9, 9), "A"),
							},
							ast.NewIdentifier(p(1, 12, 11, 13), "int"),
							"a",
						),
					},
				),
				[]ast.KeyValue{
					ast.KeyValue{
						Key:   ast.NewIdentifier(p(1, 23, 22, 22), "A"),
						Value: ast.NewBasicLiteral(p(1, 26, 25, 26), ast.IntLiteral, "10"),
					},
				},
			),
		}, ast.FormatText)},
	{"defer f()",
		ast.NewTree("", []ast.Node{
			ast.NewDefer(
				p(1, 1, 0, 8),
				ast.NewCall(
					p(1, 8, 6, 8),
					ast.NewIdentifier(p(1, 7, 6, 6), "f"), nil, false)),
		}, ast.FormatText)},
	{"go f()",
		ast.NewTree("", []ast.Node{
			ast.NewGo(
				p(1, 1, 0, 5),
				ast.NewCall(
					p(1, 5, 3, 5),
					ast.NewIdentifier(p(1, 4, 3, 3), "f"), nil, false)),
		}, ast.FormatText)},
	{"ch <- 5",
		ast.NewTree("", []ast.Node{
			ast.NewSend(
				p(1, 1, 0, 6),
				ast.NewIdentifier(p(1, 1, 0, 1), "ch"),
				ast.NewBasicLiteral(p(1, 7, 6, 6), ast.IntLiteral, "5")),
		}, ast.FormatText)},
	{"a := <-ch",
		ast.NewTree("", []ast.Node{
			ast.NewAssignment(p(1, 1, 0, 8),
				[]ast.Expression{ast.NewIdentifier(p(1, 1, 0, 0), "a")},
				ast.AssignmentDeclaration, []ast.Expression{
					ast.NewUnaryOperator(
						p(1, 6, 5, 8), ast.OperatorReceive,
						ast.NewIdentifier(p(1, 8, 7, 8), "ch"))}),
		}, ast.FormatText)},
	{"goto LOOP",
		ast.NewTree("", []ast.Node{
			ast.NewGoto(p(1, 1, 0, 8),
				ast.NewIdentifier(p(1, 6, 5, 8), "LOOP")),
		}, ast.FormatText)},
	{"LOOP: x",
		ast.NewTree("", []ast.Node{
			ast.NewLabel(p(1, 1, 0, 6),
				ast.NewIdentifier(p(1, 1, 0, 3), "LOOP"),
				ast.NewIdentifier(p(1, 7, 6, 6), "x")),
		}, ast.FormatText)},
	{"LOOP: {}",
		ast.NewTree("", []ast.Node{
			ast.NewLabel(p(1, 1, 0, 6),
				ast.NewIdentifier(p(1, 1, 0, 3), "LOOP"),
				ast.NewBlock(p(1, 7, 6, 7), nil)),
		}, ast.FormatText)},
	{"{LOOP:}",
		ast.NewTree("", []ast.Node{
			ast.NewBlock(p(1, 1, 0, 6), []ast.Node{
				ast.NewLabel(p(1, 2, 1, 5),
					ast.NewIdentifier(p(1, 2, 1, 4), "LOOP"), nil),
			}),
		}, ast.FormatText)},
	{"break LOOP",
		ast.NewTree("", []ast.Node{
			ast.NewBreak(p(1, 1, 0, 9),
				ast.NewIdentifier(p(1, 7, 6, 9), "LOOP")),
		}, ast.FormatText)},
	{"continue LOOP",
		ast.NewTree("", []ast.Node{
			ast.NewContinue(p(1, 1, 0, 12),
				ast.NewIdentifier(p(1, 10, 9, 12), "LOOP")),
		}, ast.FormatText)},
	{"func f() int {}", ast.NewTree("", []ast.Node{
		ast.NewFunc(p(1, 1, 0, 14), ast.NewIdentifier(p(1, 6, 5, 5), "f"), ast.NewFuncType(p(1, 1, 0, 14), false, nil, []*ast.Parameter{ast.NewParameter(nil, ast.NewIdentifier(p(1, 10, 9, 11), "int"))}, false), ast.NewBlock(p(1, 14, 13, 14), nil), false, ast.FormatText)}, ast.FormatText)},
	{"func f() int { return 5 }", ast.NewTree("", []ast.Node{
		ast.NewFunc(p(1, 1, 0, 24), ast.NewIdentifier(p(1, 6, 5, 5), "f"), ast.NewFuncType(p(1, 1, 0, 24), false, nil, []*ast.Parameter{ast.NewParameter(nil, ast.NewIdentifier(p(1, 10, 9, 11), "int"))}, false), ast.NewBlock(p(1, 14, 13, 24), []ast.Node{
			ast.NewReturn(p(1, 16, 15, 22), []ast.Expression{ast.NewBasicLiteral(p(1, 23, 22, 22), ast.IntLiteral, "5")}),
		}), false, ast.FormatText)}, ast.FormatText)},
	{"func f() (int, error) {}", ast.NewTree("", []ast.Node{
		ast.NewFunc(p(1, 1, 0, 23), ast.NewIdentifier(p(1, 6, 5, 5), "f"), ast.NewFuncType(p(1, 1, 0, 23), false, nil, []*ast.Parameter{
			ast.NewParameter(nil, ast.NewIdentifier(p(1, 11, 10, 12), "int")),
			ast.NewParameter(nil, ast.NewIdentifier(p(1, 16, 15, 19), "error")),
		}, false), ast.NewBlock(p(1, 22, 22, 23), nil), false, ast.FormatText)}, ast.FormatText)},
	{"func f() (n int, err error) {}", ast.NewTree("", []ast.Node{
		ast.NewFunc(p(1, 1, 0, 29), ast.NewIdentifier(p(1, 6, 5, 5), "f"), ast.NewFuncType(p(1, 1, 0, 29), false, nil, []*ast.Parameter{
			ast.NewParameter(ast.NewIdentifier(p(1, 11, 10, 10), "n"), ast.NewIdentifier(p(1, 13, 12, 14), "int")),
			ast.NewParameter(ast.NewIdentifier(p(1, 18, 17, 19), "err"), ast.NewIdentifier(p(1, 22, 21, 25), "error")),
		}, false), ast.NewBlock(p(1, 29, 28, 29), nil), false, ast.FormatText)}, ast.FormatText)},
	{"func f(a, b int, c bool, d ...int) (n int, err error) { a := 5; return a, nil }", ast.NewTree("", []ast.Node{
		ast.NewFunc(p(1, 1, 0, 78), ast.NewIdentifier(p(1, 6, 5, 5), "f"), ast.NewFuncType(p(1, 1, 0, 78), false, []*ast.Parameter{
			ast.NewParameter(ast.NewIdentifier(p(1, 8, 7, 7), "a"), nil),
			ast.NewParameter(ast.NewIdentifier(p(1, 11, 10, 10), "b"), ast.NewIdentifier(p(1, 13, 12, 14), "int")),
			ast.NewParameter(ast.NewIdentifier(p(1, 18, 17, 17), "c"), ast.NewIdentifier(p(1, 20, 19, 22), "bool")),
			ast.NewParameter(ast.NewIdentifier(p(1, 26, 25, 25), "d"), ast.NewIdentifier(p(1, 31, 30, 32), "int")),
		}, []*ast.Parameter{
			ast.NewParameter(ast.NewIdentifier(p(1, 37, 36, 36), "n"), ast.NewIdentifier(p(1, 39, 38, 40), "int")),
			ast.NewParameter(ast.NewIdentifier(p(1, 44, 43, 45), "err"), ast.NewIdentifier(p(1, 48, 47, 51), "error")),
		}, true), ast.NewBlock(p(1, 55, 54, 78), []ast.Node{
			ast.NewAssignment(p(1, 57, 56, 61),
				[]ast.Expression{ast.NewIdentifier(p(1, 57, 56, 56), "a")},
				ast.AssignmentDeclaration,
				[]ast.Expression{ast.NewBasicLiteral(p(1, 62, 61, 61), ast.IntLiteral, "5")},
			),
			ast.NewReturn(p(1, 65, 64, 76), []ast.Expression{
				ast.NewIdentifier(p(1, 72, 71, 71), "a"),
				ast.NewIdentifier(p(1, 75, 74, 76), "nil"),
			}),
		}), false, ast.FormatText)}, ast.FormatText)},
	{"select {}", ast.NewTree("", []ast.Node{
		ast.NewSelect(p(1, 1, 0, 8), nil, nil)}, ast.FormatText)},
	{"select {\n\tdefault:\n}\n", ast.NewTree("", []ast.Node{
		ast.NewSelect(p(1, 1, 0, 19), nil, []*ast.SelectCase{
			ast.NewSelectCase(p(2, 2, 10, 16), nil, nil)})}, ast.FormatText)},
	{"chan int", ast.NewTree("", []ast.Node{
		ast.NewChanType(p(1, 1, 0, 7), ast.NoDirection, ast.NewIdentifier(p(1, 6, 5, 7), "int")),
	}, ast.FormatText)},
	{"chan <- int", ast.NewTree("", []ast.Node{
		ast.NewChanType(p(1, 1, 0, 10), ast.SendDirection, ast.NewIdentifier(p(1, 9, 8, 10), "int")),
	}, ast.FormatText)},
	{"<- chan int", ast.NewTree("", []ast.Node{
		ast.NewChanType(p(1, 1, 0, 10), ast.ReceiveDirection, ast.NewIdentifier(p(1, 9, 8, 10), "int")),
	}, ast.FormatText)},
	{"var c chan int", ast.NewTree("", []ast.Node{ast.NewVar(p(1, 1, 0, 13),
		[]*ast.Identifier{ast.NewIdentifier(p(1, 5, 4, 4), "c")},
		ast.NewChanType(p(1, 7, 6, 13), ast.NoDirection, ast.NewIdentifier(p(1, 12, 11, 13), "int")),
		nil,
	)}, ast.FormatText)},
	{"var c <- chan int", ast.NewTree("", []ast.Node{ast.NewVar(p(1, 1, 0, 16),
		[]*ast.Identifier{ast.NewIdentifier(p(1, 5, 4, 4), "c")},
		ast.NewChanType(p(1, 7, 6, 16), ast.ReceiveDirection, ast.NewIdentifier(p(1, 15, 14, 16), "int")),
		nil,
	)}, ast.FormatText)},
	{"var c chan <- int", ast.NewTree("", []ast.Node{ast.NewVar(p(1, 1, 0, 16),
		[]*ast.Identifier{ast.NewIdentifier(p(1, 5, 4, 4), "c")},
		ast.NewChanType(p(1, 7, 6, 16), ast.SendDirection, ast.NewIdentifier(p(1, 15, 14, 16), "int")),
		nil,
	)}, ast.FormatText)},
	{"f = func() { println(a) }", ast.NewTree("", []ast.Node{
		ast.NewAssignment(
			p(1, 1, 0, 24),
			[]ast.Expression{
				ast.NewIdentifier(p(1, 1, 0, 0), "f"),
			},
			ast.AssignmentSimple,
			[]ast.Expression{
				ast.NewFunc(p(1, 5, 4, 24), nil, ast.NewFuncType(p(1, 5, 4, 24), false, nil, nil, false), ast.NewBlock(
					nil,
					[]ast.Node{
						ast.NewCall(
							p(1, 21, 13, 22),
							ast.NewIdentifier(p(1, 14, 13, 19), "println"),
							[]ast.Expression{
								ast.NewIdentifier(p(1, 22, 21, 21), "a"),
							},
							false,
						),
					},
				), false, ast.FormatText),
			},
		),
	}, ast.FormatText)},
	{"import \"p\"", ast.NewTree("", []ast.Node{
		ast.NewImport(p(1, 8, 7, 9), nil, "p")}, ast.FormatText)},
	{"import _ \"foo\"", ast.NewTree("", []ast.Node{
		ast.NewImport(p(1, 8, 7, 13),
			ast.NewIdentifier(p(1, 8, 7, 7), "_"), "foo")}, ast.FormatText)},
	{"import boo \"foo\"", ast.NewTree("", []ast.Node{
		ast.NewImport(p(1, 8, 7, 15),
			ast.NewIdentifier(p(1, 8, 7, 9), "boo"), "foo")}, ast.FormatText)},
}

var treeTests = []struct {
	src  string
	node ast.Node
}{
	{"", ast.NewTree("", nil, ast.FormatHTML)},
	{"a", ast.NewTree("", []ast.Node{ast.NewText(p(1, 1, 0, 0), []byte("a"), ast.Cut{})}, ast.FormatHTML)},
	{"{{a}}", ast.NewTree("", []ast.Node{ast.NewShow(p(1, 1, 0, 4), []ast.Expression{ast.NewIdentifier(p(1, 3, 2, 2), "a")}, ast.ContextHTML)}, ast.FormatHTML)},
	{"a{{b}}", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 0), []byte("a"), ast.Cut{}), ast.NewShow(p(1, 2, 1, 5), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "b")}, ast.ContextHTML)}, ast.FormatHTML)},
	{"{{a}}b", ast.NewTree("", []ast.Node{
		ast.NewShow(p(1, 1, 0, 4), []ast.Expression{ast.NewIdentifier(p(1, 3, 2, 2), "a")}, ast.ContextHTML), ast.NewText(p(1, 6, 5, 5), []byte("b"), ast.Cut{})}, ast.FormatHTML)},
	{"a{{b}}c", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 0), []byte("a"), ast.Cut{}), ast.NewShow(p(1, 2, 1, 5), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "b")}, ast.ContextHTML),
		ast.NewText(p(1, 7, 6, 6), []byte("c"), ast.Cut{})}, ast.FormatHTML)},
	{"<a href=\"/{{ a }}/b\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 8), []byte("<a href=\""), ast.Cut{}), ast.NewURL(p(1, 10, 9, 18), "a", "href", []ast.Node{
			ast.NewText(p(1, 10, 9, 9), []byte("/"), ast.Cut{}),
			ast.NewShow(p(1, 11, 10, 16), []ast.Expression{ast.NewIdentifier(p(1, 14, 13, 13), "a")}, ast.ContextQuotedAttr),
			ast.NewText(p(1, 18, 17, 18), []byte("/b"), ast.Cut{}),
		}),
		ast.NewText(p(1, 20, 19, 20), []byte("\">"), ast.Cut{}),
	}, ast.FormatHTML)},
	{"<a href=\"\n\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 8), []byte("<a href=\""), ast.Cut{}), ast.NewURL(p(1, 10, 9, 9), "a", "href", []ast.Node{
			ast.NewText(p(1, 10, 9, 9), []byte("\n"), ast.Cut{}),
		}),
		ast.NewText(p(2, 1, 10, 11), []byte("\">"), ast.Cut{}),
	}, ast.FormatHTML)},
	{"<div {{ a }}>", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 4), []byte("<div "), ast.Cut{}), ast.NewShow(p(1, 6, 5, 11),
			[]ast.Expression{ast.NewIdentifier(p(1, 9, 8, 8), "a")}, ast.ContextTag), ast.NewText(p(1, 13, 12, 12), []byte(">"), ast.Cut{}),
	}, ast.FormatHTML)},
	{"<div{{ a }}>", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 3), []byte("<div"), ast.Cut{}), ast.NewShow(p(1, 5, 4, 10),
			[]ast.Expression{ast.NewIdentifier(p(1, 8, 7, 7), "a")}, ast.ContextTag), ast.NewText(p(1, 12, 11, 11), []byte(">"), ast.Cut{}),
	}, ast.FormatHTML)},
	{"<div 本=\"{{ class }}\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 9), []byte("<div 本=\""), ast.Cut{}), ast.NewShow(p(1, 9, 10, 20),
			[]ast.Expression{ast.NewIdentifier(p(1, 12, 13, 17), "class")}, ast.ContextQuotedAttr), ast.NewText(p(1, 20, 21, 22), []byte("\">"), ast.Cut{}),
	}, ast.FormatHTML)},
	{"<div a=/{{ class }}\"{{ class }}>", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 7), []byte("<div a=/"), ast.Cut{}), ast.NewShow(p(1, 9, 8, 18),
			[]ast.Expression{ast.NewIdentifier(p(1, 12, 11, 15), "class")}, ast.ContextUnquotedAttr),
		ast.NewText(p(1, 20, 19, 19), []byte("\""), ast.Cut{}),
		ast.NewShow(p(1, 21, 20, 30),
			[]ast.Expression{ast.NewIdentifier(p(1, 24, 23, 27), "class")}, ast.ContextUnquotedAttr),
		ast.NewText(p(1, 32, 31, 31), []byte(">"), ast.Cut{}),
	}, ast.FormatHTML)},
	{"{% if x == 5 %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewIf(p(1, 4, 3, 20), nil,
				ast.NewBinaryOperator(p(1, 9, 6, 11),
					ast.OperatorEqual,
					ast.NewIdentifier(p(1, 7, 6, 6), "x"),
					ast.NewBasicLiteral(p(1, 12, 11, 11), ast.IntLiteral, "5"),
				), nil, nil),
		}, ast.FormatHTML)},
	{"{% if $x %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewIf(p(1, 4, 3, 16), nil,
				ast.NewDollarIdentifier(p(1, 7, 6, 7),
					ast.NewIdentifier(p(1, 8, 7, 7), "x"),
				), nil, nil),
		}, ast.FormatHTML)},
	{"{% for %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewFor(p(1, 4, 3, 14), nil, nil, nil, nil),
		}, ast.FormatHTML)},
	{"{% for a %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewFor(p(1, 4, 3, 16),
				nil, ast.NewIdentifier(p(1, 8, 7, 7), "a"), nil,
				nil),
		}, ast.FormatHTML)},
	{"{% for ; ; %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewFor(p(1, 4, 3, 18),
				nil, nil, nil, nil),
		}, ast.FormatHTML)},
	{"{% for i := 0; ; %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewFor(p(1, 4, 3, 24),
				ast.NewAssignment(p(1, 8, 7, 12),
					[]ast.Expression{ast.NewIdentifier(p(1, 8, 7, 7), "i")},
					ast.AssignmentDeclaration, []ast.Expression{ast.NewBasicLiteral(p(1, 13, 12, 12), ast.IntLiteral, "0")}),
				nil, nil, nil),
		}, ast.FormatHTML)},
	{"{% for ; true ; %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewFor(p(1, 4, 3, 23),
				nil, ast.NewIdentifier(p(1, 10, 9, 12), "true"), nil, nil),
		}, ast.FormatHTML)},
	{"{% for i := 0; i < 10; i = i + 1 %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewFor(p(1, 4, 3, 40),
				ast.NewAssignment(p(1, 8, 7, 12),
					[]ast.Expression{ast.NewIdentifier(p(1, 8, 7, 7), "i")},
					ast.AssignmentDeclaration, []ast.Expression{ast.NewBasicLiteral(p(1, 13, 12, 12), ast.IntLiteral, "0")}),
				ast.NewBinaryOperator(
					p(1, 18, 15, 20),
					ast.OperatorLess,
					ast.NewIdentifier(p(1, 16, 15, 15), "i"),
					ast.NewBasicLiteral(p(1, 20, 19, 20), ast.IntLiteral, "10")),
				ast.NewAssignment(p(1, 24, 23, 31),
					[]ast.Expression{ast.NewIdentifier(p(1, 24, 23, 23), "i")},
					ast.AssignmentSimple,
					[]ast.Expression{ast.NewBinaryOperator(
						p(1, 30, 27, 31),
						ast.OperatorAddition,
						ast.NewIdentifier(p(1, 28, 27, 27), "i"),
						ast.NewBasicLiteral(p(1, 32, 31, 31), ast.IntLiteral, "1"))}),
				nil),
		}, ast.FormatHTML)},
	{"{% for article in articles %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewForIn(
				p(1, 4, 3, 34),
				ast.NewIdentifier(p(1, 8, 7, 13), "article"),
				ast.NewIdentifier(p(1, 19, 18, 25), "articles"),
				nil),
		}, ast.FormatHTML)},
	{"{% for range articles %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewForRange(
				p(1, 4, 3, 29),
				ast.NewAssignment(p(1, 8, 7, 20), nil,
					ast.AssignmentSimple, []ast.Expression{ast.NewIdentifier(p(1, 14, 13, 20), "articles")}),
				nil),
		}, ast.FormatHTML)},
	{"{% for i := range articles %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewForRange(
				p(1, 4, 3, 34),
				ast.NewAssignment(p(1, 8, 7, 25),
					[]ast.Expression{ast.NewIdentifier(p(1, 8, 7, 7), "i")},
					ast.AssignmentDeclaration, []ast.Expression{ast.NewIdentifier(p(1, 19, 18, 25), "articles")}),
				nil),
		}, ast.FormatHTML)},
	{"{% for i, article := range articles %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewForRange(
				p(1, 4, 3, 43),
				ast.NewAssignment(p(1, 8, 7, 34), []ast.Expression{
					ast.NewIdentifier(p(1, 8, 7, 7), "i"),
					ast.NewIdentifier(p(1, 11, 10, 16), "article")},
					ast.AssignmentDeclaration, []ast.Expression{ast.NewIdentifier(p(1, 28, 27, 34), "articles")}),
				nil),
		}, ast.FormatHTML)},
	{"{% for article in articles %}\n<div>{{ article.title }}</div>\n{% end %}",
		ast.NewTree("articles.txt", []ast.Node{
			ast.NewForIn(
				p(1, 4, 3, 66),
				ast.NewIdentifier(p(1, 8, 7, 13), "article"),
				ast.NewIdentifier(p(1, 19, 18, 25), "articles"),
				[]ast.Node{
					ast.NewText(p(1, 30, 29, 34), []byte("\n<div>"), ast.Cut{1, 0}),
					ast.NewShow(
						p(2, 6, 35, 53), []ast.Expression{
							ast.NewSelector(
								p(2, 16, 38, 50),
								ast.NewIdentifier(
									p(2, 9, 38, 44),
									"article",
								),
								"title")},
						ast.ContextHTML),
					ast.NewText(p(2, 25, 54, 60), []byte("</div>\n"), ast.Cut{}),
				},
			),
		}, ast.FormatHTML)},
	{"{% for _, i := range []int{1,2,3} %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewForRange(p(1, 4, 3, 41),
				ast.NewAssignment(p(1, 8, 7, 32),
					[]ast.Expression{
						ast.NewIdentifier(p(1, 8, 7, 7), "_"),
						ast.NewIdentifier(p(1, 11, 10, 10), "i"),
					},
					ast.AssignmentDeclaration,
					[]ast.Expression{
						ast.NewCompositeLiteral(p(1, 27, 21, 32),
							ast.NewSliceType(
								p(1, 22, 21, 25),
								ast.NewIdentifier(p(1, 24, 23, 25), "int"),
							),
							[]ast.KeyValue{
								{nil, ast.NewBasicLiteral(p(1, 28, 27, 27), ast.IntLiteral, "1")},
								{nil, ast.NewBasicLiteral(p(1, 30, 29, 29), ast.IntLiteral, "2")},
								{nil, ast.NewBasicLiteral(p(1, 32, 31, 31), ast.IntLiteral, "3")},
							})}), nil)}, ast.FormatHTML)},
	{"{% switch x %}{% case 1 %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 4, 3, 31),
				nil,
				ast.NewIdentifier(p(1, 11, 10, 10), "x"),
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 18, 17, 22),
						[]ast.Expression{
							ast.NewBasicLiteral(p(1, 23, 22, 22), ast.IntLiteral, "1"),
						},
						nil,
					),
				},
			),
		}, ast.FormatHTML),
	},
	{"{% switch x %}{% case 1 %}something{% fallthrough %}{% case 2 %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 4, 3, 69),
				nil,
				ast.NewIdentifier(p(1, 11, 10, 10), "x"),
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 18, 17, 22),
						[]ast.Expression{
							ast.NewBasicLiteral(p(1, 23, 22, 22), ast.IntLiteral, "1"),
						},
						[]ast.Node{
							ast.NewText(p(1, 27, 26, 34), []byte("something"), ast.Cut{}),
							ast.NewFallthrough(p(1, 39, 38, 48)),
						},
					),
					ast.NewCase(
						p(1, 56, 55, 60),
						[]ast.Expression{
							ast.NewBasicLiteral(p(1, 61, 60, 60), ast.IntLiteral, "2"),
						},
						nil,
					),
				},
			),
		}, ast.FormatHTML),
	},
	{"{% switch x := 2; x * 4 %}{% case 1 %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 4, 3, 43),
				ast.NewAssignment( // x := 2
					p(1, 11, 10, 15),
					[]ast.Expression{ast.NewIdentifier(p(1, 11, 10, 10), "x")},
					ast.AssignmentDeclaration,
					[]ast.Expression{ast.NewBasicLiteral(p(1, 16, 15, 15), ast.IntLiteral, "2")},
				),
				ast.NewBinaryOperator( // x * 4
					p(1, 21, 18, 22),
					ast.OperatorMultiplication,
					ast.NewIdentifier(p(1, 19, 18, 18), "x"),
					ast.NewBasicLiteral(p(1, 23, 22, 22), ast.IntLiteral, "4"),
				),
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 30, 29, 34),
						[]ast.Expression{
							ast.NewBasicLiteral(p(1, 35, 34, 34), ast.IntLiteral, "1"),
						},
						nil,
					),
				},
			),
		}, ast.FormatHTML),
	},
	{"{% switch %}{% case 1 < 6 %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 4, 3, 33),
				nil,
				nil,
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 16, 15, 24),
						[]ast.Expression{
							ast.NewBinaryOperator( // 1 < 6
								p(1, 23, 20, 24),
								ast.OperatorLess,
								ast.NewBasicLiteral(p(1, 21, 20, 20), ast.IntLiteral, "1"),
								ast.NewBasicLiteral(p(1, 25, 24, 24), ast.IntLiteral, "6"),
							),
						},
						nil,
					),
				},
			),
		}, ast.FormatHTML),
	},
	{"{% switch %}{% default %}{% break %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 4, 3, 41),
				nil,
				nil,
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 16, 15, 21),
						nil,
						[]ast.Node{
							ast.NewBreak(p(1, 29, 28, 32), nil),
						},
					),
				},
			),
		}, ast.FormatHTML),
	},
	{"{% switch %}{% case 1 < 6, x == sum(2, -3) %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 4, 3, 50),
				nil,
				nil,
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 16, 15, 41),
						[]ast.Expression{
							ast.NewBinaryOperator( // 1 < 6
								p(1, 23, 20, 24),
								ast.OperatorLess,
								ast.NewBasicLiteral(p(1, 21, 20, 20), ast.IntLiteral, "1"),
								ast.NewBasicLiteral(p(1, 25, 24, 24), ast.IntLiteral, "6"),
							),
							ast.NewBinaryOperator(p(1, 30, 27, 41), // x == sum(2, 3)
								ast.OperatorEqual,
								ast.NewIdentifier(p(1, 28, 27, 27), "x"),
								ast.NewCall(
									p(1, 36, 32, 41),
									ast.NewIdentifier(p(1, 33, 32, 34), "sum"),
									[]ast.Expression{
										ast.NewBasicLiteral(p(1, 37, 36, 36), ast.IntLiteral, "2"),
										ast.NewUnaryOperator(p(1, 40, 39, 40), ast.OperatorSubtraction,
											ast.NewBasicLiteral(p(1, 41, 40, 40), ast.IntLiteral, "3")),
									}, false,
								),
							),
						},
						nil,
					),
				},
			),
		}, ast.FormatHTML),
	},
	{"{% switch x %}{% case 1 %}is one{% case 2 %}is two{% default %}is a number{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 4, 3, 79),
				nil,
				ast.NewIdentifier(p(1, 11, 10, 10), "x"),
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 18, 17, 22),
						[]ast.Expression{
							ast.NewBasicLiteral(p(1, 23, 22, 22), ast.IntLiteral, "1"),
						},
						[]ast.Node{
							ast.NewText(p(1, 27, 26, 31), []byte("is one"), ast.Cut{}),
						},
					),
					ast.NewCase(
						p(1, 36, 35, 40),
						[]ast.Expression{
							ast.NewBasicLiteral(p(1, 41, 40, 40), ast.IntLiteral, "2"),
						},
						[]ast.Node{
							ast.NewText(p(1, 45, 44, 49), []byte("is two"), ast.Cut{}),
						},
					),
					ast.NewCase(
						p(1, 54, 53, 59),
						nil,
						[]ast.Node{
							ast.NewText(p(1, 64, 63, 73), []byte("is a number"), ast.Cut{}),
						},
					),
				},
			),
		}, ast.FormatHTML),
	},
	{"{% switch x.(type) %}{% case int, float %}is a number{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewTypeSwitch(p(1, 4, 3, 58),
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
				nil,
				[]*ast.Case{
					ast.NewCase(p(1, 25, 24, 38),
						[]ast.Expression{
							ast.NewIdentifier(p(1, 30, 29, 31), "int"),
							ast.NewIdentifier(p(1, 35, 34, 38), "float"),
						},
						[]ast.Node{
							ast.NewText(p(1, 43, 42, 52), []byte("is a number"), ast.Cut{}),
						},
					),
				},
			),
		}, ast.FormatHTML),
	},
	{"{% switch v := x.(type) %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewTypeSwitch(p(1, 4, 3, 31),
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
				nil,
			),
		}, ast.FormatHTML),
	},
	{"<div \"{{ class }}\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 5), []byte("<div \""), ast.Cut{}), ast.NewShow(p(1, 7, 6, 16),
			[]ast.Expression{ast.NewIdentifier(p(1, 10, 9, 13), "class")}, ast.ContextTag), ast.NewText(p(1, 18, 17, 18), []byte("\">"), ast.Cut{}),
	}, ast.FormatHTML)},
	{"{% a := 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 8), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentDeclaration, []ast.Expression{ast.NewBasicLiteral(p(1, 9, 8, 8), ast.IntLiteral, "1")})}, ast.FormatHTML)},
	{"{% a = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 7), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")}, ast.AssignmentSimple,
			[]ast.Expression{ast.NewBasicLiteral(p(1, 8, 7, 7), ast.IntLiteral, "2")})}, ast.FormatHTML)},
	{"{% _ = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 7), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "_")}, ast.AssignmentSimple,
			[]ast.Expression{ast.NewBasicLiteral(p(1, 8, 7, 7), ast.IntLiteral, "2")})}, ast.FormatHTML)},
	{"{% a.b = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 9), []ast.Expression{ast.NewSelector(p(1, 5, 3, 5), ast.NewIdentifier(p(1, 4, 3, 3), "a"), "b")},
			ast.AssignmentSimple, []ast.Expression{ast.NewBasicLiteral(p(1, 10, 9, 9), ast.IntLiteral, "2")})}, ast.FormatHTML)},
	{"{% a[\"b\"] = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 12), []ast.Expression{ast.NewIndex(p(1, 5, 3, 8), ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewBasicLiteral(p(1, 6, 5, 7), ast.StringLiteral, `"b"`))},
			ast.AssignmentSimple, []ast.Expression{ast.NewBasicLiteral(p(1, 13, 12, 12), ast.IntLiteral, "2")})}, ast.FormatHTML)},
	{"{% a[6] = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 10), []ast.Expression{ast.NewIndex(p(1, 5, 3, 6), ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewBasicLiteral(p(1, 6, 5, 5), ast.IntLiteral, "6"))},
			ast.AssignmentSimple, []ast.Expression{ast.NewBasicLiteral(p(1, 11, 10, 10), ast.IntLiteral, "2")})}, ast.FormatHTML)},
	{"{% a, b := 1, 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 14),
			[]ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewIdentifier(p(1, 7, 6, 6), "b")},
			ast.AssignmentDeclaration,
			[]ast.Expression{ast.NewBasicLiteral(p(1, 12, 11, 11), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 15, 14, 14), ast.IntLiteral, "2")})}, ast.FormatHTML)},
	{"{% a, b, c = 1, 2, 3 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 19),
			[]ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewIdentifier(p(1, 7, 6, 6), "b"), ast.NewIdentifier(p(1, 10, 9, 9), "c")},
			ast.AssignmentSimple,
			[]ast.Expression{ast.NewBasicLiteral(p(1, 14, 13, 13), ast.IntLiteral, "1"), ast.NewBasicLiteral(p(1, 17, 16, 16), ast.IntLiteral, "2"), ast.NewBasicLiteral(p(1, 20, 19, 19), ast.IntLiteral, "3")})}, ast.FormatHTML)},
	{"{% a, ok := b.c %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 14), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewIdentifier(p(1, 7, 6, 7), "ok")},
			ast.AssignmentDeclaration, []ast.Expression{ast.NewSelector(p(1, 14, 12, 14),
				ast.NewIdentifier(p(1, 13, 12, 12), "b"), "c")})}, ast.FormatHTML)},
	{"{% a += 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 8), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentAddition, []ast.Expression{ast.NewBasicLiteral(p(1, 9, 8, 8), ast.IntLiteral, "1")})}, ast.FormatHTML)},
	{"{% a -= 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 8), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentSubtraction, []ast.Expression{ast.NewBasicLiteral(p(1, 9, 8, 8), ast.IntLiteral, "1")})}, ast.FormatHTML)},
	{"{% a *= 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 8), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentMultiplication, []ast.Expression{ast.NewBasicLiteral(p(1, 9, 8, 8), ast.IntLiteral, "1")})}, ast.FormatHTML)},
	{"{% a /= 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 8), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentDivision, []ast.Expression{ast.NewBasicLiteral(p(1, 9, 8, 8), ast.IntLiteral, "1")})}, ast.FormatHTML)},
	{"{% a %= 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 8), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentModulo, []ast.Expression{ast.NewBasicLiteral(p(1, 9, 8, 8), ast.IntLiteral, "1")})}, ast.FormatHTML)},
	{"{% show a %}", ast.NewTree("", []ast.Node{ast.NewShow(p(1, 4, 3, 8), []ast.Expression{ast.NewIdentifier(p(1, 9, 8, 8), "a")}, ast.ContextHTML)}, ast.FormatHTML)},
	{"{% show a, b, c %}", ast.NewTree("", []ast.Node{ast.NewShow(p(1, 4, 3, 14), []ast.Expression{
		ast.NewIdentifier(p(1, 9, 8, 8), "a"),
		ast.NewIdentifier(p(1, 12, 11, 11), "b"),
		ast.NewIdentifier(p(1, 15, 14, 14), "c")}, ast.ContextHTML)}, ast.FormatHTML)},
	{"{% show a(b,c) %}", ast.NewTree("", []ast.Node{
		ast.NewShow(p(1, 4, 3, 13), []ast.Expression{
			ast.NewCall(p(1, 10, 8, 13),
				ast.NewIdentifier(p(1, 9, 8, 8), "a"),
				[]ast.Expression{ast.NewIdentifier(p(1, 11, 10, 10), "b"), ast.NewIdentifier(p(1, 13, 12, 12), "c")},
				false)},
			ast.ContextHTML),
	}, ast.FormatHTML)},
	{"<script>{% show a %}</script>", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 7), []byte("<script>"), ast.Cut{}),
		ast.NewShow(p(1, 12, 11, 16), []ast.Expression{ast.NewIdentifier(p(1, 17, 16, 16), "a")}, ast.ContextJS),
		ast.NewText(p(1, 21, 20, 28), []byte("</script>"), ast.Cut{}),
	}, ast.FormatHTML)},
	{"{% for v in e %}b{% end for %}", ast.NewTree("", []ast.Node{
		ast.NewForIn(p(1, 4, 3, 26),
			ast.NewIdentifier(p(1, 8, 7, 7), "v"),
			ast.NewIdentifier(p(1, 13, 12, 12), "e"),
			[]ast.Node{ast.NewText(p(1, 17, 16, 16), []byte("b"), ast.Cut{})})}, ast.FormatHTML)},
	{"{% for v in e %}{% break %}{% end %}", ast.NewTree("", []ast.Node{
		ast.NewForIn(p(1, 4, 3, 32),
			ast.NewIdentifier(p(1, 8, 7, 7), "v"),
			ast.NewIdentifier(p(1, 13, 12, 12), "e"),
			[]ast.Node{ast.NewBreak(p(1, 20, 19, 23), nil)})}, ast.FormatHTML)},
	{"{% for v in e %}{% continue %}{% end %}", ast.NewTree("", []ast.Node{
		ast.NewForIn(p(1, 4, 3, 35),
			ast.NewIdentifier(p(1, 8, 7, 7), "v"),
			ast.NewIdentifier(p(1, 13, 12, 12), "e"),
			[]ast.Node{ast.NewContinue(p(1, 20, 19, 26), nil)})}, ast.FormatHTML)},
	{"{% if a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 4, 3, 19), nil, ast.NewIdentifier(p(1, 7, 6, 6), "a"), ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 11, 10, 10), []byte("b"), ast.Cut{})}), nil)}, ast.FormatHTML)},
	{"{% if a %}b{% else %}c{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(
			p(1, 4, 3, 27),
			nil,
			ast.NewIdentifier(p(1, 7, 6, 6), "a"),
			ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 11, 10, 10), []byte("b"), ast.Cut{})}),
			ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 22, 21, 21), []byte("c"), ast.Cut{})}),
		)}, ast.FormatHTML),
	},
	{"{% if a %}\nb{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 4, 3, 17), nil, ast.NewIdentifier(p(1, 7, 6, 6), "a"), ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 11, 10, 11), []byte("\nb"), ast.Cut{1, 0})}), nil)}, ast.FormatHTML)},
	{"{% if a %}\nb\n{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 4, 3, 18), nil, ast.NewIdentifier(p(1, 7, 6, 6), "a"), ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 11, 10, 12), []byte("\nb\n"), ast.Cut{1, 0})}), nil)}, ast.FormatHTML)},
	{"  {% if a %} \nb\n  {% end %} \t", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 1), []byte("  "), ast.Cut{0, 2}),
		ast.NewIf(p(1, 6, 5, 23), nil, ast.NewIdentifier(p(1, 9, 8, 8), "a"), ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 13, 12, 17), []byte(" \nb\n  "), ast.Cut{2, 2})}), nil),
		ast.NewText(p(3, 12, 27, 28), []byte(" \t"), ast.Cut{2, 0})}, ast.FormatHTML)},
	{"{% if a = b; a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 4, 3, 26),
			ast.NewAssignment(p(1, 7, 6, 10), []ast.Expression{ast.NewIdentifier(p(1, 7, 6, 6), "a")}, ast.AssignmentSimple,
				[]ast.Expression{ast.NewIdentifier(p(1, 11, 10, 10), "b")}),
			ast.NewIdentifier(p(1, 14, 13, 13), "a"), ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 18, 17, 17), []byte("b"), ast.Cut{})}), nil)}, ast.FormatHTML)},
	{"{% if a := b; a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 4, 3, 27),
			ast.NewAssignment(p(1, 7, 6, 11), []ast.Expression{ast.NewIdentifier(p(1, 7, 6, 6), "a")}, ast.AssignmentDeclaration,
				[]ast.Expression{ast.NewIdentifier(p(1, 12, 11, 11), "b")}),
			ast.NewIdentifier(p(1, 15, 14, 14), "a"), ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 19, 18, 18), []byte("b"), ast.Cut{})}), nil)}, ast.FormatHTML)},
	{"{% if a, ok := b.c; a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 4, 3, 33),
			ast.NewAssignment(p(1, 7, 6, 17),
				[]ast.Expression{ast.NewIdentifier(p(1, 7, 6, 6), "a"), ast.NewIdentifier(p(1, 10, 9, 10), "ok")},
				ast.AssignmentDeclaration, []ast.Expression{ast.NewSelector(p(1, 17, 15, 17), ast.NewIdentifier(p(1, 16, 15, 15), "b"), "c")}),
			ast.NewIdentifier(p(1, 21, 20, 20), "a"), ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 25, 24, 24), []byte("b"), ast.Cut{})}), nil)}, ast.FormatHTML)},
	{"{% if x < 10 %}x is < 10{% else if x == 10 %}x is 10{% else %}x is > 10 {% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewIf(p(1, 4, 3, 77), nil,
				ast.NewBinaryOperator(
					p(1, 9, 6, 11),
					ast.OperatorLess,
					ast.NewIdentifier(p(1, 7, 6, 6), "x"),
					ast.NewBasicLiteral(p(1, 11, 10, 11), ast.IntLiteral, "10"),
				),
				ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 16, 15, 23), []byte("x is < 10"), ast.Cut{})}),
				ast.NewIf(p(1, 33, 32, 77), nil, // else if
					ast.NewBinaryOperator(
						p(1, 38, 35, 41),
						ast.OperatorEqual,
						ast.NewIdentifier(p(1, 36, 35, 35), "x"),
						ast.NewBasicLiteral(p(1, 41, 40, 41), ast.IntLiteral, "10"),
					),
					ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 46, 45, 51), []byte("x is 10"), ast.Cut{})}),
					ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 63, 62, 71), []byte("x is > 10 "), ast.Cut{})}),
				),
			)}, ast.FormatHTML)},
	{"{% if _, ok := []int{1,2,3}.([]int); ok %}{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 4, 3, 47),
			ast.NewAssignment(
				p(1, 7, 6, 34),
				[]ast.Expression{
					ast.NewIdentifier(p(1, 7, 6, 6), "_"),
					ast.NewIdentifier(p(1, 10, 9, 10), "ok"),
				},
				ast.AssignmentDeclaration,
				[]ast.Expression{
					ast.NewTypeAssertion(p(1, 28, 15, 34),
						ast.NewCompositeLiteral(p(1, 21, 15, 26),
							ast.NewSliceType(
								p(1, 16, 15, 19),
								ast.NewIdentifier(p(1, 18, 17, 19), "int"),
							),
							[]ast.KeyValue{
								{nil, ast.NewBasicLiteral(p(1, 22, 21, 21), ast.IntLiteral, "1")},
								{nil, ast.NewBasicLiteral(p(1, 24, 23, 23), ast.IntLiteral, "2")},
								{nil, ast.NewBasicLiteral(p(1, 26, 25, 25), ast.IntLiteral, "3")},
							}),
						ast.NewSliceType(p(1, 30, 29, 33),
							ast.NewIdentifier(p(1, 32, 31, 33), "int"),
						))}),
			ast.NewIdentifier(p(1, 38, 37, 38), "ok"),
			nil,
			nil,
		),
	}, ast.FormatHTML)},
	{"{% extends \"/a.b\" %}", ast.NewTree("", []ast.Node{ast.NewExtends(p(1, 4, 3, 16), "/a.b", ast.FormatHTML)}, ast.FormatHTML)},
	{"{{ render \"/a.b\" }}", ast.NewTree("", []ast.Node{ast.NewShow(p(1, 1, 0, 18), []ast.Expression{
		ast.NewRender(p(1, 4, 3, 15), "/a.b")}, ast.ContextHTML)}, ast.FormatHTML)},
	{"{% extends \"a.e\" %}{% macro b %}c{% end macro %}", ast.NewTree("", []ast.Node{
		ast.NewExtends(p(1, 4, 3, 15), "a.e", ast.FormatHTML),
		ast.NewFunc(p(1, 23, 22, 44), ast.NewIdentifier(p(1, 29, 28, 28), "b"), ast.NewFuncType(p(1, 23, 22, 44), false, nil, nil, false), ast.NewBlock(p(1, 23, 22, 44), []ast.Node{ast.NewText(p(1, 33, 32, 32), []byte("c"), ast.Cut{})}), false, ast.FormatHTML)}, ast.FormatHTML)},
	{"{% extends \"a.e\" %}{% macro b(c, d int) %}txt{% end macro %}", ast.NewTree("", []ast.Node{
		ast.NewExtends(p(1, 4, 3, 15), "a.e", ast.FormatHTML),
		ast.NewFunc(p(1, 23, 22, 56), ast.NewIdentifier(p(1, 29, 28, 28), "b"), ast.NewFuncType(p(1, 23, 22, 56), false, []*ast.Parameter{
			ast.NewParameter(ast.NewIdentifier(p(1, 31, 30, 30), "c"), nil),
			ast.NewParameter(ast.NewIdentifier(p(1, 34, 33, 33), "d"),
				ast.NewIdentifier(p(1, 36, 35, 37), "int")),
		}, nil, false), ast.NewBlock(p(1, 23, 22, 56), []ast.Node{ast.NewText(p(1, 43, 42, 44), []byte("txt"), ast.Cut{})}), false, ast.FormatHTML)}, ast.FormatHTML)},
	{"{# comment\ncomment #}", ast.NewTree("", []ast.Node{ast.NewComment(p(1, 1, 0, 20), " comment\ncomment ")}, ast.FormatHTML)},
	{"{% macro a(i int) %}c{% end macro %}", ast.NewTree("", []ast.Node{
		ast.NewFunc(p(1, 4, 3, 32), ast.NewIdentifier(p(1, 10, 9, 9), "a"), ast.NewFuncType(p(1, 4, 3, 32), false, []*ast.Parameter{
			ast.NewParameter(ast.NewIdentifier(p(1, 12, 11, 11), "i"),
				ast.NewIdentifier(p(1, 14, 13, 15), "int")),
		}, nil, false), ast.NewBlock(p(1, 4, 3, 32), []ast.Node{ast.NewText(p(1, 21, 20, 20), []byte("c"), ast.Cut{})}), false, ast.FormatHTML)}, ast.FormatHTML)},
	{"{% macro a(b bool, c ...string) %}d{% end macro %}", ast.NewTree("", []ast.Node{
		ast.NewFunc(p(1, 4, 3, 46), ast.NewIdentifier(p(1, 10, 9, 9), "a"), ast.NewFuncType(p(1, 4, 3, 46), false, []*ast.Parameter{
			ast.NewParameter(ast.NewIdentifier(p(1, 12, 11, 11), "b"),
				ast.NewIdentifier(p(1, 14, 13, 16), "bool")),
			ast.NewParameter(ast.NewIdentifier(p(1, 20, 19, 19), "c"),
				ast.NewIdentifier(p(1, 25, 24, 29), "string")),
		}, nil, true), ast.NewBlock(p(1, 4, 3, 46), []ast.Node{ast.NewText(p(1, 35, 34, 34), []byte("d"), ast.Cut{})}), false, ast.FormatHTML)}, ast.FormatHTML)},
	{"{% *a = 3 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 4, 3, 8),
			[]ast.Expression{
				ast.NewUnaryOperator(p(1, 4, 3, 4),
					ast.OperatorPointer,
					ast.NewIdentifier(p(1, 5, 4, 4), "a"),
				)},
			ast.AssignmentSimple,
			[]ast.Expression{
				ast.NewBasicLiteral(p(1, 9, 8, 8), ast.IntLiteral, "3"),
			})}, ast.FormatHTML)},
	{"{%% if x == 5 { } %%}",
		ast.NewTree("", []ast.Node{
			ast.NewStatements(p(1, 1, 0, 20), []ast.Node{
				ast.NewIf(p(1, 5, 4, 16), nil,
					ast.NewBinaryOperator(p(1, 10, 7, 12),
						ast.OperatorEqual,
						ast.NewIdentifier(p(1, 8, 7, 7), "x"),
						ast.NewBasicLiteral(p(1, 13, 12, 12), ast.IntLiteral, "5"),
					), nil, nil),
			})}, ast.FormatHTML)},
	{"{% raw code %}\t\n{{ v }}\n{% end raw code %}{{ v }}", ast.NewTree("", []ast.Node{
		ast.NewRaw(p(1, 4, 3, 38), "code", "",
			ast.NewText(p(1, 15, 14, 23), []byte("\t\n{{ v }}\n"), ast.Cut{2, 0})),
		ast.NewShow(p(3, 19, 42, 48), []ast.Expression{
			ast.NewIdentifier(p(3, 22, 45, 45), "v")}, ast.ContextHTML),
	}, ast.FormatHTML)},
	{"{% raw code `lang:\"javascript\"` %}\t\n{{ v }}\n{% end raw code %}{{ v }}", ast.NewTree("", []ast.Node{
		ast.NewRaw(p(1, 4, 3, 58), "code", "lang:\"javascript\"",
			ast.NewText(p(1, 35, 34, 43), []byte("\t\n{{ v }}\n"), ast.Cut{2, 0})),
		ast.NewShow(p(3, 19, 62, 68), []ast.Expression{
			ast.NewIdentifier(p(3, 22, 65, 65), "v")}, ast.ContextHTML),
	}, ast.FormatHTML)},
}

// TODO: this function is never called, because it is referenced in commented
// code. Uncomment the call or delete this function.
func fileTests() map[string]struct {
	src  string
	tree *ast.Tree
} {
	var render = ast.NewRender(p(3, 7, 29, 61), "/partial2.html")
	render.Tree = ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 4), []byte("<div>"), ast.Cut{}),
		ast.NewShow(p(1, 6, 5, 17), []ast.Expression{ast.NewIdentifier(p(1, 9, 8, 14), "content")}, ast.ContextHTML),
		ast.NewText(p(1, 19, 18, 23), []byte("</div>"), ast.Cut{}),
	}, ast.FormatHTML)
	return map[string]struct {
		src  string
		tree *ast.Tree
	}{
		"/simple.html": {
			"<!DOCTYPE html>\n<html>\n<head><title>{{ title }}</title></head>\n<body>{{ content }}</body>\n</html>",
			ast.NewTree("", []ast.Node{
				ast.NewText(p(1, 1, 0, 35), []byte("<!DOCTYPE html>\n<html>\n<head><title>"), ast.Cut{}),
				ast.NewShow(p(3, 14, 36, 46), []ast.Expression{ast.NewIdentifier(p(3, 17, 39, 43), "title")}, ast.ContextHTML),
				ast.NewText(p(3, 25, 47, 68), []byte("</title></head>\n<body>"), ast.Cut{}),
				ast.NewShow(p(4, 7, 69, 81), []ast.Expression{ast.NewIdentifier(p(4, 10, 72, 78), "content")}, ast.ContextHTML),
				ast.NewText(p(4, 20, 82, 96), []byte("</body>\n</html>"), ast.Cut{}),
			}, ast.FormatHTML),
		},
		"/simple2.html": {
			"<!DOCTYPE html>\n<html>\n<body>{{ render \"/partial2.html\" }}</body>\n</html>",
			ast.NewTree("", []ast.Node{
				ast.NewText(p(1, 1, 0, 28), []byte("<!DOCTYPE html>\n<html>\n<body>"), ast.Cut{}),
				render,
				ast.NewText(p(3, 37, 59, 73), []byte("</body>\n</html>"), ast.Cut{}),
			}, ast.FormatHTML),
		},
		"/partial2.html": {
			"<div>{{ content }}</div>",
			nil,
		},
	}
}

func TestGoContextTrees(t *testing.T) {
	for _, tree := range goContextTreeTests {
		node, err := parseSource([]byte(tree.src), true, false)
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

var shebangTests = []struct {
	src     string
	shebang bool
	err     string
}{
	{"#! /usr/bin/scriggo", true, ""},
	{"#! /usr/bin/scriggo\n=", true, ":2:1: syntax error: unexpected =, expected statement"},
	{"a = 5\n#! /usr/bin/scriggo\n", true, ":2:1: syntax error: illegal character U+0023 '#'"},
	{"#! /usr/bin/scriggo", false, ":1:1: syntax error: illegal character U+0023 '#'"},
}

func TestShebang(t *testing.T) {
	for _, test := range shebangTests {
		_, err := parseSource([]byte(test.src), true, test.shebang)
		if err == nil {
			if test.err != "" {
				t.Errorf("source: %q, expected error %q, got nothing\n", test.src, test.err)
			}
		} else {
			if err.Error() != test.err {
				t.Errorf("source: %q, %s\n", test.src, err)
			}
			continue
		}
	}
}

func TestTrees(t *testing.T) {
	for _, tree := range treeTests {
		node, _, err := ParseTemplateSource([]byte(tree.src), ast.FormatHTML, false, false)
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

func TestFiles(t *testing.T) {

	t.Skip("(not runnable)")

	// tests := fileTests()
	// // simple.html
	// parser := NewParser(testsReader(tests), nil, false)
	// p := tests["/simple.html"]
	// tree, err := parser.Parse("/simple.html", ast.ContextHTML)
	// if err != nil {
	// 	t.Errorf("source: %q, %s\n", p.src, err)
	// }
	// err = equals(tree, p.tree, 0)
	// if err != nil {
	// 	t.Errorf("source: %q, %s\n", p.src, err)
	// }
	// // simple2.html
	// p = tests["/simple2.html"]
	// tree, err = parser.Parse("/simple2.html", ast.ContextHTML)
	// if err != nil {
	// 	t.Errorf("source: %q, %s\n", p.src, err)
	// }
	// err = equals(tree, p.tree, 0)
	// if err != nil {
	// 	t.Errorf("source: %q, %s\n", p.src, err)
	// }
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
	if pos1 == nil && pos2 != nil {
		return fmt.Errorf("unexpected nil pos, expecting %#v", pos2)
	}
	if pos1 != nil && pos2 == nil {
		return fmt.Errorf("expected nil pos, got %#v", pos1)
	}
	if pos1 != nil && pos2 != nil {
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
	}

	if e1, ok := n1.(ast.Expression); ok {
		if e2, ok := n2.(ast.Expression); ok {
			if e1.Parenthesis() != e2.Parenthesis() {
				return fmt.Errorf("unexpected %d parenthesis, expecting %d", e1.Parenthesis(), e2.Parenthesis())
			}
		}
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
		if nn1.Format != nn2.Format {
			return fmt.Errorf("unexpected context %s, expecting %s", nn1.Format, nn2.Format)
		}

	case *ast.Import:
		nn2, ok := n2.(*ast.Import)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Ident, nn2.Ident, p)
		if err != nil {
			return err
		}
		if nn1.Path != nn2.Path {
			return fmt.Errorf("unexpected name %q expecting %q", nn1.Path, nn2.Path)
		}

	case *ast.Extends:
		nn2, ok := n2.(*ast.Extends)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Path != nn2.Path {
			return fmt.Errorf("unexpected path %q, expecting %q", nn1.Path, nn2.Path)
		}
		if nn1.Format != nn2.Format {
			return fmt.Errorf("unexpected format %s, expecting %s", nn1.Format, nn2.Format)
		}
		err := equals(nn1.Tree, nn2.Tree, p)
		if err != nil {
			return err
		}

	case *ast.Render:
		nn2, ok := n2.(*ast.Render)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Path != nn2.Path {
			return fmt.Errorf("unexpected path %q, expecting %q", nn1.Path, nn2.Path)
		}
		err := equals(nn1.Tree, nn2.Tree, p)
		if err != nil {
			return err
		}

	case *ast.Package:
		nn2, ok := n2.(*ast.Package)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if len(nn1.Declarations) != len(nn2.Declarations) {
			return fmt.Errorf("unexpected declarations len %d, expecting %d", len(nn1.Declarations), len(nn2.Declarations))
		}
		for i, node := range nn1.Declarations {
			err := equals(node, nn2.Declarations[i], p)
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

	case *ast.Comment:
		nn2, ok := n2.(*ast.Comment)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Text != nn2.Text {
			return fmt.Errorf("unexpected text %q, expecting %q", nn1.Text, nn2.Text)
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

	case *ast.DollarIdentifier:
		nn2, ok := n2.(*ast.DollarIdentifier)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Ident, nn2.Ident, p)
		if err != nil {
			return err
		}

	case *ast.BasicLiteral:
		nn2, ok := n2.(*ast.BasicLiteral)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Type != nn2.Type {
			return fmt.Errorf("unexpected literal type %q, expecting %q", nn1.Type, nn2.Type)
		}
		if nn1.Value != nn2.Value {
			return fmt.Errorf("unexpected %q, expecting %q", nn1.Value, nn2.Value)
		}

	case *ast.UnaryOperator:
		nn2, ok := n2.(*ast.UnaryOperator)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Op != nn2.Op {
			return fmt.Errorf("unexpected operator %q, expecting %q", nn1.Op.String(), nn2.Op.String())
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
			return fmt.Errorf("unexpected operator %q, expecting %q", nn1.Op.String(), nn2.Op.String())
		}
		err := equals(nn1.Expr1, nn2.Expr1, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Expr2, nn2.Expr2, p)
		if err != nil {
			return err
		}

	case *ast.Selector:
		nn2, ok := n2.(*ast.Selector)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Expr, nn2.Expr, p)
		if err != nil {
			return err
		}
		if nn1.Ident != nn2.Ident {
			return fmt.Errorf("unexpected ident %q, expecting %q", nn1.Ident, nn2.Ident)
		}

	case *ast.CompositeLiteral:
		nn2, ok := n2.(*ast.CompositeLiteral)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Type, nn2.Type, p)
		if err != nil {
			return err
		}
		if len(nn1.KeyValues) != len(nn2.KeyValues) {
			return fmt.Errorf("unexpected len %d, expecting %d for KeyValue", len(nn1.KeyValues), len(nn2.KeyValues))
		}
		for i := range nn1.KeyValues {
			err := equals(nn1.KeyValues[i].Key, nn2.KeyValues[i].Key, p)
			if err != nil {
				return err
			}
			err = equals(nn1.KeyValues[i].Value, nn2.KeyValues[i].Value, p)
			if err != nil {
				return err
			}
		}

	case *ast.Interface:
		_, ok := n2.(*ast.Interface)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}

	case *ast.ArrayType:
		nn2, ok := n2.(*ast.ArrayType)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Len, nn2.Len, p)
		if err != nil {
			return err
		}
		err = equals(nn1.ElementType, nn2.ElementType, p)
		if err != nil {
			return err
		}

	case *ast.SliceType:
		nn2, ok := n2.(*ast.SliceType)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.ElementType, nn2.ElementType, p)
		if err != nil {
			return err
		}

	case *ast.MapType:
		nn2, ok := n2.(*ast.MapType)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.KeyType, nn2.KeyType, p)
		if err != nil {
			return err
		}
		err = equals(nn1.ValueType, nn2.ValueType, p)
		if err != nil {
			return err
		}

	case *ast.ChanType:
		nn2, ok := n2.(*ast.ChanType)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Direction != nn2.Direction {
			return fmt.Errorf("unexpected direction %s, expecting %s", nn1.Direction, nn2.Direction)
		}
		err := equals(nn1.ElementType, nn2.ElementType, p)
		if err != nil {
			return err
		}

	case *ast.StructType:
		nn2, ok := n2.(*ast.StructType)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if len(nn1.Fields) != len(nn2.Fields) {
			return fmt.Errorf("struct type: unexpected fields len %#v, expecting %#v", len(nn1.Fields), len(nn2.Fields))
		}
		for i := range nn1.Fields {
			fd1 := nn1.Fields[i]
			fd2 := nn2.Fields[i]
			if len(fd1.Idents) != len(fd2.Idents) {
				return fmt.Errorf("struct type: field %d: expecting %d identifiers, got %d", i, len(fd2.Idents), len(fd1.Idents))
			}
			err := equals(fd1.Type, fd2.Type, p)
			if err != nil {
				return fmt.Errorf("struct type: field %d: %s", i, err)
			}
			// https://github.com/open2b/scriggo/issues/61.
		}

	case *ast.TypeDeclaration:
		nn2, ok := n2.(*ast.TypeDeclaration)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Ident, nn2.Ident, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Type, nn2.Type, p)
		if err != nil {
			return err
		}
		if nn1.IsAliasDeclaration && !nn2.IsAliasDeclaration {
			return fmt.Errorf("expecting type definition, got alias declaration")
		}
		if !nn1.IsAliasDeclaration && nn2.IsAliasDeclaration {
			return fmt.Errorf("expecting alias declaration, got type definition")
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
		if nn1.IsVariadic && !nn2.IsVariadic {
			return fmt.Errorf("unexpected not variadic, expecting variadic")
		}
		if !nn1.IsVariadic && nn2.IsVariadic {
			return fmt.Errorf("unexpected variadic, expecting not variadic")
		}

	case *ast.Assignment:
		nn2, ok := n2.(*ast.Assignment)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if len(nn1.Lhs) != len(nn2.Lhs) {
			return fmt.Errorf("unexpected variables len %d, expecting %d", len(nn1.Lhs), len(nn2.Lhs))
		}
		for i, v := range nn1.Lhs {
			err := equals(v, nn2.Lhs[i], p)
			if err != nil {
				return err
			}
		}
		if nn1.Type != nn2.Type {
			return fmt.Errorf("unexpected assignment type %d, expecting %d", nn1.Type, nn2.Type)
		}
		if len(nn1.Rhs) != len(nn2.Rhs) {
			return fmt.Errorf("unexpected values len %d, expecting %d", len(nn1.Rhs), len(nn2.Rhs))
		}
		for i, v := range nn1.Rhs {
			err := equals(v, nn2.Rhs[i], p)
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
		err = equals(nn1.Max, nn2.Max, p)
		if err != nil {
			return err
		}
		if nn1.IsFull != nn2.IsFull {
			if nn1.IsFull {
				return fmt.Errorf("unexpected full expression, expecting not full")
			} else {
				return fmt.Errorf("unexpected not full expression, expecting full")
			}
		}

	case *ast.Show:
		nn2, ok := n2.(*ast.Show)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		for i, expr := range nn1.Expressions {
			err := equals(expr, nn2.Expressions[i], p)
			if err != nil {
				return err
			}
		}
		if nn1.Context != nn2.Context {
			return fmt.Errorf("unexpected context %s, expecting %s", nn1.Context, nn2.Context)
		}

	case *ast.Block:
		nn2, ok := n2.(*ast.Block)
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

	case *ast.Statements:
		nn2, ok := n2.(*ast.Statements)
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

	case *ast.If:
		nn2, ok := n2.(*ast.If)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Condition, nn2.Condition, p)
		if err != nil {
			return err
		}
		if len(nn1.Then.Nodes) != len(nn2.Then.Nodes) {
			return fmt.Errorf("unexpected then nodes len %d, expecting %d", len(nn1.Then.Nodes), len(nn2.Then.Nodes))
		}
		for i, node := range nn1.Then.Nodes {
			err := equals(node, nn2.Then.Nodes[i], p)
			if err != nil {
				return err
			}
		}
		err = equals(nn1.Else, nn2.Else, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Init, nn2.Init, p)
		if err != nil {
			return err
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

	case *ast.ForIn:
		nn2, ok := n2.(*ast.ForIn)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Ident, nn2.Ident, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Expr, nn2.Expr, p)
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

	case *ast.Var:
		nn2, ok := n2.(*ast.Var)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if len(nn1.Lhs) != len(nn2.Lhs) {
			return fmt.Errorf("unexpected nodes len %d, expecting %d", len(nn1.Lhs), len(nn2.Lhs))
		}
		for i, node := range nn1.Lhs {
			err := equals(node, nn2.Lhs[i], p)
			if err != nil {
				return err
			}
		}
		err := equals(nn1.Type, nn2.Type, p)
		if err != nil {
			return err
		}
		if len(nn1.Rhs) != len(nn2.Rhs) {
			return fmt.Errorf("unexpected nodes len %d, expecting %d", len(nn1.Rhs), len(nn2.Rhs))
		}
		for i, node := range nn1.Rhs {
			err := equals(node, nn2.Rhs[i], p)
			if err != nil {
				return err
			}
		}

	case *ast.Const:
		nn2, ok := n2.(*ast.Const)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if len(nn1.Lhs) != len(nn2.Lhs) {
			return fmt.Errorf("unexpected nodes len %d, expecting %d", len(nn1.Lhs), len(nn2.Lhs))
		}
		for i, node := range nn1.Lhs {
			err := equals(node, nn2.Lhs[i], p)
			if err != nil {
				return err
			}
		}
		err := equals(nn1.Type, nn2.Type, p)
		if err != nil {
			return err
		}
		if len(nn1.Rhs) != len(nn2.Rhs) {
			return fmt.Errorf("unexpected nodes len %d, expecting %d", len(nn1.Rhs), len(nn2.Rhs))
		}
		for i, node := range nn1.Rhs {
			err := equals(node, nn2.Rhs[i], p)
			if err != nil {
				return err
			}
		}
		if nn1.Index != nn2.Index {
			return fmt.Errorf("unexpected index value %d, expecting %d", nn1.Index, nn2.Index)
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
		err = equals(nn1.LeadingText, nn2.LeadingText, p)
		if err != nil {
			return err
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
		err = equals(nn1.LeadingText, nn2.LeadingText, p)
		if err != nil {
			return err
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
		if len(nn1.Expressions) != len(nn2.Expressions) {
			return fmt.Errorf("unexpected expressions nodes len %d, expected %d", len(nn1.Expressions), len(nn2.Expressions))
		}
		for i, expr := range nn1.Expressions {
			err := equals(expr, nn2.Expressions[i], p)
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

	case *ast.Fallthrough:
		if _, ok := n2.(*ast.Fallthrough); !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}

	case *ast.Select:
		nn2, ok := n2.(*ast.Select)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.LeadingText, nn2.LeadingText, p)
		if err != nil {
			return err
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

	case *ast.SelectCase:
		nn2, ok := n2.(*ast.SelectCase)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Comm, nn2.Comm, p)
		if err != nil {
			return err
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

	case *ast.FuncType:
		nn2, ok := n2.(*ast.FuncType)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if len(nn1.Parameters) != len(nn2.Parameters) {
			return fmt.Errorf("unexpected parameters len %d, expecting %d", len(nn1.Parameters), len(nn2.Parameters))
		}
		for i, f1 := range nn1.Parameters {
			f2 := nn2.Parameters[i]
			err := equals(f1.Ident, f2.Ident, p)
			if err != nil {
				return err
			}
			err = equals(f1.Type, f2.Type, p)
			if err != nil {
				return err
			}
		}
		if len(nn1.Result) != len(nn2.Result) {
			return fmt.Errorf("unexpected result len %d, expecting %d", len(nn1.Result), len(nn2.Result))
		}
		for i, r1 := range nn1.Result {
			r2 := nn2.Result[i]
			err := equals(r1.Ident, r2.Ident, p)
			if err != nil {
				return err
			}
			err = equals(r1.Type, r2.Type, p)
			if err != nil {
				return err
			}
		}
		if nn1.IsVariadic != nn2.IsVariadic {
			if nn1.IsVariadic {
				return fmt.Errorf("unexpected variadic func, expecting not variadic")
			} else {
				return fmt.Errorf("unexpected not variadic func, expecting nvariadic")
			}
		}

	case *ast.Func:
		nn2, ok := n2.(*ast.Func)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Ident, nn2.Ident, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Type, nn2.Type, p)
		if err != nil {
			return err
		}
		if len(nn1.Body.Nodes) != len(nn2.Body.Nodes) {
			return fmt.Errorf("unexpected body nodes len %d, expecting %d", len(nn1.Body.Nodes), len(nn2.Body.Nodes))
		}
		for i, node := range nn1.Body.Nodes {
			err := equals(node, nn2.Body.Nodes[i], p)
			if err != nil {
				return err
			}
		}
		if nn1.Endless != nn2.Endless {
			if nn1.Endless {
				return fmt.Errorf("unexpected endless, expecting not endless")
			}
			return fmt.Errorf("unexpected not endless, expecting endless")
		}
		if nn1.Format != nn2.Format {
			return fmt.Errorf("unexpected format %s, expecting %s", nn1.Format, nn2.Format)
		}

	case *ast.Defer:
		nn2, ok := n2.(*ast.Defer)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Call, nn2.Call, p)
		if err != nil {
			return err
		}

	case *ast.Default:
		nn2, ok := n2.(*ast.Default)
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

	case *ast.Go:
		nn2, ok := n2.(*ast.Go)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Call, nn2.Call, p)
		if err != nil {
			return err
		}

	case *ast.Goto:
		nn2, ok := n2.(*ast.Goto)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Label, nn2.Label, p)
		if err != nil {
			return err
		}

	case *ast.Label:
		nn2, ok := n2.(*ast.Label)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Ident, nn2.Ident, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Statement, nn2.Statement, p)
		if err != nil {
			return err
		}

	case *ast.Send:
		nn2, ok := n2.(*ast.Send)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Channel, nn2.Channel, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Value, nn2.Value, p)
		if err != nil {
			return err
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

	case *ast.Return:
		nn2, ok := n2.(*ast.Return)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if len(nn1.Values) != len(nn2.Values) {
			return fmt.Errorf("unexpected values len %d, expecting %d", len(nn1.Values), len(nn2.Values))
		}
		for i, node := range nn1.Values {
			err := equals(node, nn2.Values[i], p)
			if err != nil {
				return err
			}
		}

	case *ast.Break:
		nn2, ok := n2.(*ast.Break)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Label, nn2.Label, p)
		if err != nil {
			return err
		}

	case *ast.Continue:
		nn2, ok := n2.(*ast.Continue)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Label, nn2.Label, p)
		if err != nil {
			return err
		}

	case *ast.Raw:
		nn2, ok := n2.(*ast.Raw)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Marker != nn2.Marker {
			return fmt.Errorf("unexpected marker %q, expecting %q", nn1.Marker, nn2.Marker)
		}
		if nn1.Tag != nn2.Tag {
			return fmt.Errorf("unexpected tag %q, expecting %q", nn1.Tag, nn2.Tag)
		}
		err := equals(nn1.Text, nn2.Text, p)
		if err != nil {
			return err
		}

	default:
		panic(fmt.Sprintf("unexpected node of type %T\n", n1))
	}

	return nil
}

func TestRooted(t *testing.T) {
	tests := []struct {
		parent string
		name   string
		rooted string
		error  error
	}{
		{"a/b/c", "/d/e", "d/e", nil},
		{"a", "b/c", "b/c", nil},
		{"a/b/c", "d/e", "a/b/d/e", nil},
		{"a/b/c", "../d/e", "a/d/e", nil},
		{"a/b/c", "../../d/e", "d/e", nil},
		{"a/b", "../c/d", "c/d", nil},
		{"a/b", "../../c/d", "", os.ErrInvalid},
	}
	for _, test := range tests {
		root, err := rooted(test.parent, test.name)
		if root != test.rooted {
			t.Errorf("unexpected %q, expecting %q", root, test.rooted)
		}
		if err != test.error {
			t.Errorf("unexpected error %v, expecting error %v", err, test.error)
		}
	}
}
