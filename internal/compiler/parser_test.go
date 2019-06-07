// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"scriggo/internal/compiler/ast"
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

type predefinedPackages map[string]*Package

func (pp predefinedPackages) Load(path string) (interface{}, error) {
	p := pp[path]
	return p, nil
}

func p(line, column, start, end int) *ast.Position {
	return &ast.Position{line, column, start, end}
}

var goContextTreeTests = []struct {
	src  string
	node ast.Node
}{
	{"", ast.NewTree("", nil, ast.ContextGo)},
	{";", ast.NewTree("", nil, ast.ContextGo)},
	{"a", ast.NewTree("", []ast.Node{ast.NewIdentifier(p(1, 1, 0, 0), "a")}, ast.ContextGo)},
	{"a := 1", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 5), []ast.Expression{ast.NewIdentifier(p(1, 1, 0, 0), "a")},
			ast.AssignmentDeclaration, []ast.Expression{ast.NewInt(p(1, 6, 5, 5), big.NewInt(1))})}, ast.ContextGo)},
	{"a.b = 2", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 6), []ast.Expression{ast.NewSelector(p(1, 2, 0, 2), ast.NewIdentifier(p(1, 1, 0, 0), "a"), "b")},
			ast.AssignmentSimple, []ast.Expression{ast.NewInt(p(1, 7, 6, 6), big.NewInt(2))})}, ast.ContextGo)},
	{"if a {\n\tb\n}\n", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 10), nil, ast.NewIdentifier(p(1, 4, 3, 3), "a"),
			ast.NewBlock(p(1, 6, 5, 10), []ast.Node{ast.NewIdentifier(p(2, 2, 8, 8), "b")}), nil)}, ast.ContextGo)},
	{"if a {\t\tb\t}\t", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 10), nil, ast.NewIdentifier(p(1, 4, 3, 3), "a"),
			ast.NewBlock(p(1, 6, 5, 10), []ast.Node{ast.NewIdentifier(p(1, 9, 8, 8), "b")}), nil)}, ast.ContextGo)},
	{"if a {\n\tb\n} else {\n\tc\n}\n", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 22), nil, ast.NewIdentifier(p(1, 4, 3, 3), "a"),
			ast.NewBlock(p(1, 6, 5, 10), []ast.Node{ast.NewIdentifier(p(2, 2, 8, 8), "b")}),
			ast.NewBlock(p(3, 8, 17, 22), []ast.Node{ast.NewIdentifier(p(4, 2, 20, 20), "c")}))}, ast.ContextGo)},
	{"if a {\n\tb\n} else if c {\n\td\n} else {\n\te\n}\n", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 39), nil, ast.NewIdentifier(p(1, 4, 3, 3), "a"),
			ast.NewBlock(p(1, 6, 5, 10), []ast.Node{ast.NewIdentifier(p(2, 2, 8, 8), "b")}),
			ast.NewIf(p(3, 8, 17, 39), nil, ast.NewIdentifier(p(3, 11, 20, 20), "c"),
				ast.NewBlock(p(3, 13, 22, 27), []ast.Node{ast.NewIdentifier(p(4, 2, 25, 25), "d")}),
				ast.NewBlock(p(5, 8, 34, 39), []ast.Node{ast.NewIdentifier(p(6, 2, 37, 37), "e")})))}, ast.ContextGo)},
	{"for _, v := range e {\n\tb\n}\n", ast.NewTree("", []ast.Node{
		ast.NewForRange(p(1, 1, 0, 25), ast.NewAssignment(p(1, 5, 4, 18), []ast.Expression{
			ast.NewIdentifier(p(1, 5, 4, 4), "_"), ast.NewIdentifier(p(1, 8, 7, 7), "v")},
			ast.AssignmentDeclaration, []ast.Expression{ast.NewIdentifier(p(1, 19, 18, 18), "e")}),
			[]ast.Node{ast.NewIdentifier(p(2, 2, 23, 23), "b")})}, ast.ContextGo)},
	{"for _ = range []int(nil) { }", ast.NewTree("", []ast.Node{
		ast.NewForRange(p(1, 1, 0, 27),
			ast.NewAssignment(p(1, 5, 4, 23),
				[]ast.Expression{ast.NewIdentifier(p(1, 5, 4, 4), "_")},
				ast.AssignmentSimple,
				[]ast.Expression{
					ast.NewCall(p(1, 20, 14, 23),
						ast.NewSliceType(p(1, 15, 14, 18), ast.NewIdentifier(p(1, 17, 16, 18), "int")),
						[]ast.Expression{ast.NewIdentifier(p(1, 21, 20, 22), "nil")},
						false)}), nil)}, ast.ContextGo)},
	{"switch {\n\tdefault:\n}\n", ast.NewTree("", []ast.Node{
		ast.NewSwitch(p(1, 1, 0, 19), nil, nil, nil, []*ast.Case{
			ast.NewCase(p(2, 2, 10, 17), nil, nil, false)})}, ast.ContextGo)},
	{"if x == 5 {}",
		ast.NewTree("", []ast.Node{
			ast.NewIf(&ast.Position{Line: 1, Column: 1, Start: 0, End: 11}, nil,
				ast.NewBinaryOperator(p(1, 6, 3, 8),
					ast.OperatorEqual,
					ast.NewIdentifier(p(1, 4, 3, 3), "x"),
					ast.NewInt(p(1, 9, 8, 8), big.NewInt(5)),
				), nil, nil),
		}, ast.ContextGo)},
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
								{nil, ast.NewInt(p(1, 25, 24, 24), big.NewInt(1))},
								{nil, ast.NewInt(p(1, 27, 26, 26), big.NewInt(2))},
								{nil, ast.NewInt(p(1, 29, 28, 28), big.NewInt(3))},
							})}), nil)},
			ast.ContextGo)},
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
		}, ast.ContextGo),
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
		}, ast.ContextGo),
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
					ast.NewInt(p(1, 9, 8, 8), big.NewInt(4)),
				},
			),
		}, ast.ContextGo),
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
					ast.NewInt(p(1, 13, 12, 12), big.NewInt(4)),
				},
			),
		}, ast.ContextGo),
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
					ast.NewInt(p(1, 16, 15, 15), big.NewInt(4)),
					ast.NewInt(p(1, 19, 18, 18), big.NewInt(6)),
				},
			),
		}, ast.ContextGo),
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
					ast.NewInt(p(2, 10, 15, 15), big.NewInt(3)),
				},
			),
			ast.NewVar(
				p(1, 1, 0, 27),
				[]*ast.Identifier{
					ast.NewIdentifier(p(3, 2, 18, 18), "b"),
				},
				nil,
				[]ast.Expression{
					ast.NewFloat(p(3, 6, 22, 25), new(big.Float).SetFloat64(1.00)),
				},
			),
		}, ast.ContextGo),
	},
	{"var f func ()",
		ast.NewTree("", []ast.Node{
			ast.NewVar(
				p(1, 1, 0, 12),
				[]*ast.Identifier{
					ast.NewIdentifier(p(1, 5, 4, 4), "f"),
				},
				ast.NewFuncType(
					p(1, 7, 6, 12),
					nil,
					nil,
					false,
				),
				nil,
			),
		}, ast.ContextGo),
	},
	{"var A []T",
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
		}, ast.ContextGo),
	},
	{"const a = 4",
		ast.NewTree("", []ast.Node{
			ast.NewConst(
				p(1, 1, 0, 10),
				[]*ast.Identifier{
					ast.NewIdentifier(p(1, 7, 6, 6), "a"),
				},
				nil,
				[]ast.Expression{
					ast.NewInt(p(1, 11, 10, 10), big.NewInt(4)),
				},
			),
		}, ast.ContextGo),
	},
	{"const (\nA = 42\nB\n)\n", ast.NewTree("", []ast.Node{
		ast.NewConst(
			p(1, 1, 0, 17),
			[]*ast.Identifier{
				ast.NewIdentifier(p(2, 1, 8, 8), "A"),
			},
			nil,
			[]ast.Expression{
				ast.NewInt(p(2, 5, 12, 13), big.NewInt(42)),
			},
		),
		ast.NewConst(
			p(1, 1, 0, 17),
			[]*ast.Identifier{
				ast.NewIdentifier(p(3, 1, 15, 15), "B"),
			},
			nil,
			// TODO(Gianluca): this is the first time that parser changes
			// the tree. The constant B has no value, but according to
			// implicit repetition it gets the value of A.
			[]ast.Expression{
				ast.NewInt(p(2, 5, 12, 13), big.NewInt(42)),
			},
		),
	}, ast.ContextGo)},
	{"{}", ast.NewTree("", []ast.Node{
		ast.NewBlock(p(1, 1, 0, 1), nil),
	}, ast.ContextGo)},
	{"type Int int",
		ast.NewTree("", []ast.Node{
			ast.NewTypeDeclaration(
				p(1, 1, 0, 11),
				ast.NewIdentifier(p(1, 6, 5, 7), "Int"),
				ast.NewIdentifier(p(1, 10, 9, 11), "int"),
				false,
			),
		}, ast.ContextGo),
	},
	{"type Int []string",
		ast.NewTree("", []ast.Node{
			ast.NewTypeDeclaration(
				p(1, 1, 0, 16),
				ast.NewIdentifier(p(1, 6, 5, 7), "Int"),
				ast.NewSliceType(p(1, 10, 9, 16), ast.NewIdentifier(p(1, 12, 11, 16), "string")),
				false,
			),
		}, ast.ContextGo),
	},
	{"type Int = int",
		ast.NewTree("", []ast.Node{
			ast.NewTypeDeclaration(
				p(1, 1, 0, 13),
				ast.NewIdentifier(p(1, 6, 5, 7), "Int"),
				ast.NewIdentifier(p(1, 12, 11, 13), "int"),
				true,
			),
		}, ast.ContextGo),
	},
	{"type MyMap = map[string]interface{}",
		ast.NewTree("", []ast.Node{
			ast.NewTypeDeclaration(
				p(1, 1, 0, 34),
				ast.NewIdentifier(p(1, 6, 5, 9), "MyMap"),
				ast.NewMapType(
					p(1, 14, 13, 34),
					ast.NewIdentifier(p(1, 18, 17, 22), "string"),
					ast.NewIdentifier(p(1, 25, 24, 34), "interface{}"),
				),
				true,
			),
		}, ast.ContextGo),
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
		}, ast.ContextGo),
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
		}, ast.ContextGo),
	},
	{"struct { }",
		ast.NewTree("", []ast.Node{
			ast.NewStructType(
				p(1, 1, 0, 9),
				nil,
			),
		}, ast.ContextGo)},
	{"struct { A int }",
		ast.NewTree("", []ast.Node{
			ast.NewStructType(
				p(1, 1, 0, 15),
				[]*ast.FieldDecl{
					ast.NewFieldDecl(
						[]*ast.Identifier{
							ast.NewIdentifier(p(1, 10, 9, 9), "A"),
						},
						ast.NewIdentifier(p(1, 12, 11, 13), "int"),
						nil,
					),
				},
			),
		}, ast.ContextGo)},
	{"struct { A []int }",
		ast.NewTree("", []ast.Node{
			ast.NewStructType(
				p(1, 1, 0, 17),
				[]*ast.FieldDecl{
					ast.NewFieldDecl(
						[]*ast.Identifier{
							ast.NewIdentifier(p(1, 10, 9, 9), "A"),
						},
						ast.NewSliceType(
							p(1, 12, 11, 15),
							ast.NewIdentifier(p(1, 14, 13, 15), "int"),
						),
						nil,
					),
				},
			),
		}, ast.ContextGo)},
	{"struct { A struct { C int } }",
		ast.NewTree("", []ast.Node{
			ast.NewStructType(
				p(1, 1, 0, 28),
				[]*ast.FieldDecl{
					ast.NewFieldDecl(
						[]*ast.Identifier{
							ast.NewIdentifier(p(1, 10, 9, 9), "A"),
						},
						ast.NewStructType(
							p(1, 12, 11, 26),
							[]*ast.FieldDecl{
								ast.NewFieldDecl(
									[]*ast.Identifier{
										ast.NewIdentifier(p(1, 21, 20, 20), "C"),
									},
									ast.NewIdentifier(p(1, 23, 22, 24), "int"),
									nil,
								),
							},
						),
						nil,
					),
				},
			),
		}, ast.ContextGo)},
	{"struct { A, B int }",
		ast.NewTree("", []ast.Node{
			ast.NewStructType(
				p(1, 1, 0, 18),
				[]*ast.FieldDecl{
					ast.NewFieldDecl(
						[]*ast.Identifier{
							ast.NewIdentifier(p(1, 10, 9, 9), "A"),
							ast.NewIdentifier(p(1, 13, 12, 12), "B"),
						},
						ast.NewIdentifier(p(1, 15, 14, 16), "int"),
						nil,
					),
				},
			),
		}, ast.ContextGo)},
	{"struct { A, B int ; C, D string }",
		ast.NewTree("", []ast.Node{
			ast.NewStructType(
				p(1, 1, 0, 32),
				[]*ast.FieldDecl{
					ast.NewFieldDecl(
						[]*ast.Identifier{
							ast.NewIdentifier(p(1, 10, 9, 9), "A"),
							ast.NewIdentifier(p(1, 13, 12, 12), "B"),
						},
						ast.NewIdentifier(p(1, 15, 14, 16), "int"),
						nil,
					),
					ast.NewFieldDecl(
						[]*ast.Identifier{
							ast.NewIdentifier(p(1, 21, 20, 20), "C"),
							ast.NewIdentifier(p(1, 24, 23, 23), "D"),
						},
						ast.NewIdentifier(p(1, 26, 25, 30), "string"),
						nil,
					),
				},
			),
		}, ast.ContextGo)},
	{"struct { A int ; C ; *D }",
		ast.NewTree("", []ast.Node{
			ast.NewStructType(
				p(1, 1, 0, 24),
				[]*ast.FieldDecl{
					ast.NewFieldDecl(
						[]*ast.Identifier{
							ast.NewIdentifier(p(1, 10, 9, 9), "A"),
						},
						ast.NewIdentifier(p(1, 12, 11, 13), "int"),
						nil,
					),
					ast.NewFieldDecl(
						nil,
						ast.NewIdentifier(p(1, 18, 17, 17), "C"),
						nil,
					),
					ast.NewFieldDecl(
						nil,
						ast.NewUnaryOperator(
							p(1, 22, 21, 22),
							ast.OperatorMultiplication,
							ast.NewIdentifier(p(1, 23, 22, 22), "D"),
						),
						nil,
					),
				},
			),
		}, ast.ContextGo)},
	{"struct { A int }{ A: 10 }",
		ast.NewTree("", []ast.Node{
			ast.NewCompositeLiteral(
				p(1, 17, 0, 24),
				ast.NewStructType(
					p(1, 1, 0, 15),
					[]*ast.FieldDecl{
						ast.NewFieldDecl(
							[]*ast.Identifier{
								ast.NewIdentifier(p(1, 10, 9, 9), "A"),
							},
							ast.NewIdentifier(p(1, 12, 11, 13), "int"),
							nil,
						),
					},
				),
				[]ast.KeyValue{
					ast.KeyValue{
						Key:   ast.NewIdentifier(p(1, 19, 18, 18), "A"),
						Value: ast.NewInt(p(1, 22, 21, 22), big.NewInt(10)),
					},
				},
			),
		}, ast.ContextGo)},
	{"defer f()",
		ast.NewTree("", []ast.Node{
			ast.NewDefer(
				p(1, 1, 0, 8),
				ast.NewCall(
					p(1, 8, 6, 8),
					ast.NewIdentifier(p(1, 7, 6, 6), "f"), nil, false)),
		}, ast.ContextGo)},
	{"go f()",
		ast.NewTree("", []ast.Node{
			ast.NewGo(
				p(1, 1, 0, 5),
				ast.NewCall(
					p(1, 5, 3, 5),
					ast.NewIdentifier(p(1, 4, 3, 3), "f"), nil, false)),
		}, ast.ContextGo)},
	{"ch <- 5",
		ast.NewTree("", []ast.Node{
			ast.NewSend(
				p(1, 1, 0, 6),
				ast.NewIdentifier(p(1, 1, 0, 1), "ch"),
				ast.NewInt(p(1, 7, 6, 6), big.NewInt(5))),
		}, ast.ContextGo)},
	{"a := <-ch",
		ast.NewTree("", []ast.Node{
			ast.NewAssignment(p(1, 1, 0, 8),
				[]ast.Expression{ast.NewIdentifier(p(1, 1, 0, 0), "a")},
				ast.AssignmentDeclaration, []ast.Expression{
					ast.NewUnaryOperator(
						p(1, 6, 5, 8), ast.OperatorReceive,
						ast.NewIdentifier(p(1, 8, 7, 8), "ch"))}),
		}, ast.ContextGo)},
	{"goto LOOP",
		ast.NewTree("", []ast.Node{
			ast.NewGoto(p(1, 1, 0, 8),
				ast.NewIdentifier(p(1, 6, 5, 8), "LOOP")),
		}, ast.ContextGo)},
	{"LOOP: x",
		ast.NewTree("", []ast.Node{
			ast.NewLabel(p(1, 1, 0, 6),
				ast.NewIdentifier(p(1, 1, 0, 3), "LOOP"),
				ast.NewIdentifier(p(1, 7, 6, 6), "x")),
		}, ast.ContextGo)},
	{"LOOP: {}",
		ast.NewTree("", []ast.Node{
			ast.NewLabel(p(1, 1, 0, 6),
				ast.NewIdentifier(p(1, 1, 0, 3), "LOOP"),
				ast.NewBlock(p(1, 7, 6, 7), nil)),
		}, ast.ContextGo)},
	{"{LOOP:}",
		ast.NewTree("", []ast.Node{
			ast.NewBlock(p(1, 1, 0, 6), []ast.Node{
				ast.NewLabel(p(1, 2, 1, 5),
					ast.NewIdentifier(p(1, 2, 1, 4), "LOOP"), nil),
			}),
		}, ast.ContextGo)},
	{"break LOOP",
		ast.NewTree("", []ast.Node{
			ast.NewBreak(p(1, 1, 0, 9),
				ast.NewIdentifier(p(1, 7, 6, 9), "LOOP")),
		}, ast.ContextGo)},
	{"continue LOOP",
		ast.NewTree("", []ast.Node{
			ast.NewContinue(p(1, 1, 0, 12),
				ast.NewIdentifier(p(1, 10, 9, 12), "LOOP")),
		}, ast.ContextGo)},
	{"func f() int {}", ast.NewTree("", []ast.Node{
		ast.NewFunc(p(1, 1, 0, 14), ast.NewIdentifier(p(1, 6, 5, 5), "f"),
			ast.NewFuncType(nil, nil, []*ast.Field{ast.NewField(nil, ast.NewIdentifier(p(1, 10, 9, 11), "int"))}, false),
			ast.NewBlock(p(1, 14, 13, 14), nil))}, ast.ContextGo)},
	{"func f() int { return 5 }", ast.NewTree("", []ast.Node{
		ast.NewFunc(p(1, 1, 0, 24), ast.NewIdentifier(p(1, 6, 5, 5), "f"),
			ast.NewFuncType(nil, nil, []*ast.Field{ast.NewField(nil, ast.NewIdentifier(p(1, 10, 9, 11), "int"))}, false),
			ast.NewBlock(p(1, 14, 13, 24), []ast.Node{
				ast.NewReturn(p(1, 16, 15, 22), []ast.Expression{ast.NewInt(p(1, 23, 22, 22), big.NewInt(5))}),
			}))}, ast.ContextGo)},
	{"func f() (int, error) {}", ast.NewTree("", []ast.Node{
		ast.NewFunc(p(1, 1, 0, 23), ast.NewIdentifier(p(1, 6, 5, 5), "f"),
			ast.NewFuncType(nil, nil, []*ast.Field{
				ast.NewField(nil, ast.NewIdentifier(p(1, 11, 10, 12), "int")),
				ast.NewField(nil, ast.NewIdentifier(p(1, 16, 15, 19), "error")),
			}, false),
			ast.NewBlock(p(1, 22, 22, 23), nil))}, ast.ContextGo)},
	{"func f() (n int, err error) {}", ast.NewTree("", []ast.Node{
		ast.NewFunc(p(1, 1, 0, 29), ast.NewIdentifier(p(1, 6, 5, 5), "f"),
			ast.NewFuncType(nil, nil, []*ast.Field{
				ast.NewField(ast.NewIdentifier(p(1, 11, 10, 10), "n"), ast.NewIdentifier(p(1, 13, 12, 14), "int")),
				ast.NewField(ast.NewIdentifier(p(1, 18, 17, 19), "err"), ast.NewIdentifier(p(1, 22, 21, 25), "error")),
			}, false),
			ast.NewBlock(p(1, 29, 28, 29), nil))}, ast.ContextGo)},
	{"func f(a, b int, c bool, d ...int) (n int, err error) { a := 5; return a, nil }", ast.NewTree("", []ast.Node{
		ast.NewFunc(p(1, 1, 0, 78), ast.NewIdentifier(p(1, 6, 5, 5), "f"),
			ast.NewFuncType(nil, []*ast.Field{
				ast.NewField(ast.NewIdentifier(p(1, 8, 7, 7), "a"), nil),
				ast.NewField(ast.NewIdentifier(p(1, 11, 10, 10), "b"), ast.NewIdentifier(p(1, 13, 12, 14), "int")),
				ast.NewField(ast.NewIdentifier(p(1, 18, 17, 17), "c"), ast.NewIdentifier(p(1, 20, 19, 22), "bool")),
				ast.NewField(ast.NewIdentifier(p(1, 26, 25, 25), "d"), ast.NewIdentifier(p(1, 31, 30, 32), "int")),
			}, []*ast.Field{
				ast.NewField(ast.NewIdentifier(p(1, 37, 36, 36), "n"), ast.NewIdentifier(p(1, 39, 38, 40), "int")),
				ast.NewField(ast.NewIdentifier(p(1, 44, 43, 45), "err"), ast.NewIdentifier(p(1, 48, 47, 51), "error")),
			}, true),
			ast.NewBlock(p(1, 55, 54, 78), []ast.Node{
				ast.NewAssignment(p(1, 57, 56, 62),
					[]ast.Expression{ast.NewIdentifier(p(1, 57, 56, 56), "a")},
					ast.AssignmentDeclaration,
					[]ast.Expression{ast.NewInt(p(1, 62, 61, 61), big.NewInt(5))},
				),
				ast.NewReturn(p(1, 65, 64, 76), []ast.Expression{
					ast.NewIdentifier(p(1, 72, 71, 71), "a"),
					ast.NewIdentifier(p(1, 75, 74, 76), "nil"),
				}),
			}))}, ast.ContextGo)},
	{"select {}", ast.NewTree("", []ast.Node{
		ast.NewSelect(p(1, 1, 0, 8), nil, nil)}, ast.ContextGo)},
	{"select {\n\tdefault:\n}\n", ast.NewTree("", []ast.Node{
		ast.NewSelect(p(1, 1, 0, 19), nil, []*ast.SelectCase{
			ast.NewSelectCase(p(2, 2, 10, 17), nil, nil)})}, ast.ContextGo)},

	// TODO (Gianluca):
	// {"f = func() { println(a) }", ast.NewTree("", []ast.Node{
	// 	ast.NewAssignment(
	// 		p(1, 1, 0, 24),
	// 		[]ast.Expression{
	// 			ast.NewIdentifier(p(1, 1, 0, 0), "f"),
	// 		},
	// 		ast.AssignmentSimple,
	// 		[]ast.Expression{
	// 			ast.NewFunc(
	// 				p(1, 5, 4, 24),
	// 				nil,
	// 				ast.NewFuncType(
	// 					p(1, 5, 4, 9),
	// 					nil,
	// 					nil,
	// 					false,
	// 				),
	// 				ast.NewBlock(
	// 					nil,
	// 					[]ast.Node{
	// 						ast.NewCall(
	// 							p(1, 14, 13, 22),
	// 							ast.NewIdentifier(p(1, 14, 13, 19), "println"),
	// 							[]ast.Expression{
	// 								ast.NewIdentifier(p(1, 22, 21, 21), "a"),
	// 							},
	// 						),
	// 					},
	// 				),
	// 			),
	// 		},
	// 	),
	// }, ast.ContextGo)},
}

var treeTests = []struct {
	src  string
	node ast.Node
}{
	{"", ast.NewTree("", nil, ast.ContextHTML)},
	{"a", ast.NewTree("", []ast.Node{ast.NewText(p(1, 1, 0, 0), []byte("a"), ast.Cut{})}, ast.ContextHTML)},
	{"{{a}}", ast.NewTree("", []ast.Node{ast.NewShow(p(1, 1, 0, 4), ast.NewIdentifier(p(1, 3, 2, 2), "a"), ast.ContextHTML)}, ast.ContextHTML)},
	{"a{{b}}", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 0), []byte("a"), ast.Cut{}), ast.NewShow(p(1, 2, 1, 5), ast.NewIdentifier(p(1, 4, 3, 3), "b"), ast.ContextHTML)}, ast.ContextHTML)},
	{"{{a}}b", ast.NewTree("", []ast.Node{
		ast.NewShow(p(1, 1, 0, 4), ast.NewIdentifier(p(1, 3, 2, 2), "a"), ast.ContextHTML), ast.NewText(p(1, 6, 5, 5), []byte("b"), ast.Cut{})}, ast.ContextHTML)},
	{"a{{b}}c", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 0), []byte("a"), ast.Cut{}), ast.NewShow(p(1, 2, 1, 5), ast.NewIdentifier(p(1, 4, 3, 3), "b"), ast.ContextHTML),
		ast.NewText(p(1, 7, 6, 6), []byte("c"), ast.Cut{})}, ast.ContextHTML)},
	{"<a href=\"/{{ a }}/b\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 8), []byte("<a href=\""), ast.Cut{}), ast.NewURL(p(1, 10, 9, 18), "a", "href", []ast.Node{
			ast.NewText(p(1, 10, 9, 9), []byte("/"), ast.Cut{}),
			ast.NewShow(p(1, 11, 10, 16), ast.NewIdentifier(p(1, 14, 13, 13), "a"), ast.ContextAttribute),
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
		ast.NewText(p(1, 1, 0, 4), []byte("<div "), ast.Cut{}), ast.NewShow(p(1, 6, 5, 11),
			ast.NewIdentifier(p(1, 9, 8, 8), "a"), ast.ContextTag), ast.NewText(p(1, 13, 12, 12), []byte(">"), ast.Cut{}),
	}, ast.ContextHTML)},
	{"<div{{ a }}>", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 3), []byte("<div"), ast.Cut{}), ast.NewShow(p(1, 5, 4, 10),
			ast.NewIdentifier(p(1, 8, 7, 7), "a"), ast.ContextTag), ast.NewText(p(1, 12, 11, 11), []byte(">"), ast.Cut{}),
	}, ast.ContextHTML)},
	{"<div 本=\"{{ class }}\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 9), []byte("<div 本=\""), ast.Cut{}), ast.NewShow(p(1, 9, 10, 20),
			ast.NewIdentifier(p(1, 12, 13, 17), "class"), ast.ContextAttribute), ast.NewText(p(1, 20, 21, 22), []byte("\">"), ast.Cut{}),
	}, ast.ContextHTML)},
	{"<div a=/{{ class }}\"{{ class }}>", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 7), []byte("<div a=/"), ast.Cut{}), ast.NewShow(p(1, 9, 8, 18),
			ast.NewIdentifier(p(1, 12, 11, 15), "class"), ast.ContextUnquotedAttribute),
		ast.NewText(p(1, 20, 19, 19), []byte("\""), ast.Cut{}),
		ast.NewShow(p(1, 21, 20, 30),
			ast.NewIdentifier(p(1, 24, 23, 27), "class"), ast.ContextUnquotedAttribute),
		ast.NewText(p(1, 32, 31, 31), []byte(">"), ast.Cut{}),
	}, ast.ContextHTML)},
	{"{% if x == 5 %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewIf(&ast.Position{Line: 1, Column: 1, Start: 0, End: 23}, nil,
				ast.NewBinaryOperator(p(1, 9, 6, 11),
					ast.OperatorEqual,
					ast.NewIdentifier(p(1, 7, 6, 6), "x"),
					ast.NewInt(p(1, 12, 11, 11), big.NewInt(5)),
				), nil, nil),
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
					ast.AssignmentDeclaration, []ast.Expression{ast.NewInt(&ast.Position{Line: 1, Column: 13, Start: 12, End: 12}, big.NewInt(0))}),
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
					ast.AssignmentDeclaration, []ast.Expression{ast.NewInt(&ast.Position{Line: 1, Column: 13, Start: 12, End: 12}, big.NewInt(0))}),
				ast.NewBinaryOperator(
					&ast.Position{Line: 1, Column: 18, Start: 15, End: 20},
					ast.OperatorLess,
					ast.NewIdentifier(&ast.Position{Line: 1, Column: 16, Start: 15, End: 15}, "i"),
					ast.NewInt(&ast.Position{Line: 1, Column: 20, Start: 19, End: 20}, big.NewInt(10))),
				ast.NewAssignment(&ast.Position{Line: 1, Column: 24, Start: 23, End: 31},
					[]ast.Expression{ast.NewIdentifier(&ast.Position{Line: 1, Column: 24, Start: 23, End: 23}, "i")},
					ast.AssignmentSimple,
					[]ast.Expression{ast.NewBinaryOperator(
						&ast.Position{Line: 1, Column: 30, Start: 27, End: 31},
						ast.OperatorAddition,
						ast.NewIdentifier(&ast.Position{Line: 1, Column: 28, Start: 27, End: 27}, "i"),
						ast.NewInt(&ast.Position{Line: 1, Column: 32, Start: 31, End: 31}, big.NewInt(1)))}),
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
					ast.NewShow(
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
	{"{% for _, i := range []int{1,2,3} %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewForRange(p(1, 1, 0, 44),
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
								{nil, ast.NewInt(p(1, 28, 27, 27), big.NewInt(1))},
								{nil, ast.NewInt(p(1, 30, 29, 29), big.NewInt(2))},
								{nil, ast.NewInt(p(1, 32, 31, 31), big.NewInt(3))},
							})}), nil)}, ast.ContextHTML)},
	{"{% switch x %}{% case 1 %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewSwitch(
				p(1, 1, 0, 34),
				nil,
				ast.NewIdentifier(p(1, 11, 10, 10), "x"),
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 15, 14, 25),
						[]ast.Expression{
							ast.NewInt(p(1, 23, 22, 22), big.NewInt(1)),
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
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 15, 14, 25),
						[]ast.Expression{
							ast.NewInt(p(1, 23, 22, 22), big.NewInt(1)),
						},
						[]ast.Node{
							ast.NewText(p(1, 27, 26, 34), []byte("something"), ast.Cut{}),
						},
						true,
					),
					ast.NewCase(
						p(1, 53, 52, 63),
						[]ast.Expression{
							ast.NewInt(p(1, 61, 60, 60), big.NewInt(2)),
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
					[]ast.Expression{ast.NewInt(p(1, 16, 15, 15), big.NewInt(2))},
				),
				ast.NewBinaryOperator( // x * 4
					p(1, 21, 18, 22),
					ast.OperatorMultiplication,
					ast.NewIdentifier(p(1, 19, 18, 18), "x"),
					ast.NewInt(p(1, 23, 22, 22), big.NewInt(4)),
				),
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 27, 26, 37),
						[]ast.Expression{
							ast.NewInt(p(1, 35, 34, 34), big.NewInt(1)),
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
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 13, 12, 27),
						[]ast.Expression{
							ast.NewBinaryOperator( // 1 < 6
								p(1, 23, 20, 24),
								ast.OperatorLess,
								ast.NewInt(p(1, 21, 20, 20), big.NewInt(1)),
								ast.NewInt(p(1, 25, 24, 24), big.NewInt(6)),
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
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 13, 12, 24),
						nil,
						[]ast.Node{
							ast.NewBreak(p(1, 26, 25, 35), nil),
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
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 13, 12, 44),
						[]ast.Expression{
							ast.NewBinaryOperator( // 1 < 6
								p(1, 23, 20, 24),
								ast.OperatorLess,
								ast.NewInt(p(1, 21, 20, 20), big.NewInt(1)),
								ast.NewInt(p(1, 25, 24, 24), big.NewInt(6)),
							),
							ast.NewBinaryOperator(p(1, 30, 27, 41), // x == sum(2, 3)
								ast.OperatorEqual,
								ast.NewIdentifier(p(1, 28, 27, 27), "x"),
								ast.NewCall(
									p(1, 36, 32, 41),
									ast.NewIdentifier(p(1, 33, 32, 34), "sum"),
									[]ast.Expression{
										ast.NewInt(p(1, 37, 36, 36), big.NewInt(2)),
										ast.NewInt(p(1, 40, 39, 40), big.NewInt(-3)),
									}, false,
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
				nil,
				[]*ast.Case{
					ast.NewCase(
						p(1, 15, 14, 25),
						[]ast.Expression{
							ast.NewInt(p(1, 23, 22, 22), big.NewInt(1)),
						},
						[]ast.Node{
							ast.NewText(p(1, 27, 26, 31), []byte("is one"), ast.Cut{}),
						},
						false,
					),
					ast.NewCase(
						p(1, 33, 32, 43),
						[]ast.Expression{
							ast.NewInt(p(1, 41, 40, 40), big.NewInt(2)),
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
				nil,
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
				nil,
			),
		}, ast.ContextHTML),
	},
	{"<div \"{{ class }}\">", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 5), []byte("<div \""), ast.Cut{}), ast.NewShow(p(1, 7, 6, 16),
			ast.NewIdentifier(p(1, 10, 9, 13), "class"), ast.ContextTag), ast.NewText(p(1, 18, 17, 18), []byte("\">"), ast.Cut{}),
	}, ast.ContextHTML)},
	{"{% a := 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 11), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentDeclaration, []ast.Expression{ast.NewInt(p(1, 9, 8, 8), big.NewInt(1))})}, ast.ContextHTML)},
	{"{% a = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 10), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")}, ast.AssignmentSimple,
			[]ast.Expression{ast.NewInt(p(1, 8, 7, 7), big.NewInt(2))})}, ast.ContextHTML)},
	{"{% _ = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 10), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "_")}, ast.AssignmentSimple,
			[]ast.Expression{ast.NewInt(p(1, 8, 7, 7), big.NewInt(2))})}, ast.ContextHTML)},
	{"{% a.b = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 12), []ast.Expression{ast.NewSelector(p(1, 5, 3, 5), ast.NewIdentifier(p(1, 4, 3, 3), "a"), "b")},
			ast.AssignmentSimple, []ast.Expression{ast.NewInt(p(1, 10, 9, 9), big.NewInt(2))})}, ast.ContextHTML)},
	{"{% a[\"b\"] = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 15), []ast.Expression{ast.NewIndex(p(1, 5, 3, 8), ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewString(p(1, 6, 5, 7), "b"))},
			ast.AssignmentSimple, []ast.Expression{ast.NewInt(p(1, 13, 12, 12), big.NewInt(2))})}, ast.ContextHTML)},
	{"{% a[6] = 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 13), []ast.Expression{ast.NewIndex(p(1, 5, 3, 6), ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewInt(p(1, 6, 5, 5), big.NewInt(6)))},
			ast.AssignmentSimple, []ast.Expression{ast.NewInt(p(1, 11, 10, 10), big.NewInt(2))})}, ast.ContextHTML)},
	{"{% a, b := 1, 2 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 17),
			[]ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewIdentifier(p(1, 7, 6, 6), "b")},
			ast.AssignmentDeclaration,
			[]ast.Expression{ast.NewInt(p(1, 12, 11, 11), big.NewInt(1)), ast.NewInt(p(1, 15, 14, 14), big.NewInt(2))})}, ast.ContextHTML)},
	{"{% a, b, c = 1, 2, 3 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 22),
			[]ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewIdentifier(p(1, 7, 6, 6), "b"), ast.NewIdentifier(p(1, 10, 9, 9), "c")},
			ast.AssignmentSimple,
			[]ast.Expression{ast.NewInt(p(1, 14, 13, 13), big.NewInt(1)), ast.NewInt(p(1, 17, 16, 16), big.NewInt(2)), ast.NewInt(p(1, 20, 19, 19), big.NewInt(3))})}, ast.ContextHTML)},
	{"{% a, ok := b.c %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 17), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a"), ast.NewIdentifier(p(1, 7, 6, 7), "ok")},
			ast.AssignmentDeclaration, []ast.Expression{ast.NewSelector(p(1, 14, 12, 14),
				ast.NewIdentifier(p(1, 13, 12, 12), "b"), "c")})}, ast.ContextHTML)},
	{"{% a += 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 11), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentAddition, []ast.Expression{ast.NewInt(p(1, 9, 8, 8), big.NewInt(1))})}, ast.ContextHTML)},
	{"{% a -= 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 11), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentSubtraction, []ast.Expression{ast.NewInt(p(1, 9, 8, 8), big.NewInt(1))})}, ast.ContextHTML)},
	{"{% a *= 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 11), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentMultiplication, []ast.Expression{ast.NewInt(p(1, 9, 8, 8), big.NewInt(1))})}, ast.ContextHTML)},
	{"{% a /= 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 11), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentDivision, []ast.Expression{ast.NewInt(p(1, 9, 8, 8), big.NewInt(1))})}, ast.ContextHTML)},
	{"{% a %= 1 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 11), []ast.Expression{ast.NewIdentifier(p(1, 4, 3, 3), "a")},
			ast.AssignmentModulo, []ast.Expression{ast.NewInt(p(1, 9, 8, 8), big.NewInt(1))})}, ast.ContextHTML)},
	{"{% show a %}", ast.NewTree("", []ast.Node{
		ast.NewShowMacro(p(1, 1, 0, 11), nil, ast.NewIdentifier(p(1, 9, 8, 8), "a"), nil, false, ast.ShowMacroOrError, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% show a(b,c) %}", ast.NewTree("", []ast.Node{
		ast.NewShowMacro(p(1, 1, 0, 16), nil, ast.NewIdentifier(p(1, 9, 8, 8), "a"), []ast.Expression{
			ast.NewIdentifier(p(1, 11, 10, 10), "b"), ast.NewIdentifier(p(1, 13, 12, 12), "c")}, false, ast.ShowMacroOrError, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% show a(b,c...) %}", ast.NewTree("", []ast.Node{
		ast.NewShowMacro(p(1, 1, 0, 19), nil, ast.NewIdentifier(p(1, 9, 8, 8), "a"), []ast.Expression{
			ast.NewIdentifier(p(1, 11, 10, 10), "b"), ast.NewIdentifier(p(1, 13, 12, 12), "c")}, true, ast.ShowMacroOrError, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% show M or todo %}", ast.NewTree("", []ast.Node{
		ast.NewShowMacro(p(1, 1, 0, 19), nil, ast.NewIdentifier(p(1, 9, 8, 8), "M"), nil, false, ast.ShowMacroOrTodo, ast.ContextHTML),
	}, ast.ContextHTML)},
	{"{% show M    or  ignore    %}", ast.NewTree("", []ast.Node{
		ast.NewShowMacro(p(1, 1, 0, 28), nil, ast.NewIdentifier(p(1, 9, 8, 8), "M"), nil, false, ast.ShowMacroOrIgnore, ast.ContextHTML),
	}, ast.ContextHTML)},
	{"{% show M  or  error %}", ast.NewTree("", []ast.Node{
		ast.NewShowMacro(p(1, 1, 0, 22), nil, ast.NewIdentifier(p(1, 9, 8, 8), "M"), nil, false, ast.ShowMacroOrError, ast.ContextHTML),
	}, ast.ContextHTML)},
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
			[]ast.Node{ast.NewBreak(p(1, 17, 16, 26), nil)})}, ast.ContextHTML)},
	{"{% for v in e %}{% continue %}{% end %}", ast.NewTree("", []ast.Node{
		ast.NewForRange(p(1, 1, 0, 38), ast.NewAssignment(p(1, 8, 7, 12), []ast.Expression{
			ast.NewIdentifier(p(1, 8, 7, 7), "_"),
			ast.NewIdentifier(p(1, 8, 7, 7), "v")},
			ast.AssignmentDeclaration, []ast.Expression{ast.NewIdentifier(p(1, 13, 12, 12), "e")}),
			[]ast.Node{ast.NewContinue(p(1, 17, 16, 29), nil)})}, ast.ContextHTML)},
	{"{% if a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 22), nil, ast.NewIdentifier(p(1, 7, 6, 6), "a"), ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 11, 10, 10), []byte("b"), ast.Cut{})}), nil)}, ast.ContextHTML)},
	{"{% if a %}b{% else %}c{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(
			p(1, 1, 0, 30),
			nil,
			ast.NewIdentifier(p(1, 7, 6, 6), "a"),
			ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 11, 10, 10), []byte("b"), ast.Cut{})}),
			ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 22, 21, 21), []byte("c"), ast.Cut{})}),
		)}, ast.ContextHTML),
	},
	{"{% if a %}\nb{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 20), nil, ast.NewIdentifier(p(1, 7, 6, 6), "a"), ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 11, 10, 11), []byte("\nb"), ast.Cut{1, 0})}), nil)}, ast.ContextHTML)},
	{"{% if a %}\nb\n{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 21), nil, ast.NewIdentifier(p(1, 7, 6, 6), "a"), ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 11, 10, 12), []byte("\nb\n"), ast.Cut{1, 0})}), nil)}, ast.ContextHTML)},
	{"  {% if a %} \nb\n  {% end %} \t", ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 1), []byte("  "), ast.Cut{0, 2}),
		ast.NewIf(p(1, 3, 2, 26), nil, ast.NewIdentifier(p(1, 9, 8, 8), "a"), ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 13, 12, 17), []byte(" \nb\n  "), ast.Cut{2, 2})}), nil),
		ast.NewText(p(3, 12, 27, 28), []byte(" \t"), ast.Cut{2, 0})}, ast.ContextHTML)},
	{"{% if a = b; a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 29),
			ast.NewAssignment(p(1, 7, 6, 10), []ast.Expression{ast.NewIdentifier(p(1, 7, 6, 6), "a")}, ast.AssignmentSimple,
				[]ast.Expression{ast.NewIdentifier(p(1, 11, 10, 10), "b")}),
			ast.NewIdentifier(p(1, 14, 13, 13), "a"), ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 18, 17, 17), []byte("b"), ast.Cut{})}), nil)}, ast.ContextHTML)},
	{"{% if a := b; a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 30),
			ast.NewAssignment(p(1, 7, 6, 11), []ast.Expression{ast.NewIdentifier(p(1, 7, 6, 6), "a")}, ast.AssignmentDeclaration,
				[]ast.Expression{ast.NewIdentifier(p(1, 12, 11, 11), "b")}),
			ast.NewIdentifier(p(1, 15, 14, 14), "a"), ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 19, 18, 18), []byte("b"), ast.Cut{})}), nil)}, ast.ContextHTML)},
	{"{% if a, ok := b.c; a %}b{% end if %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 36),
			ast.NewAssignment(p(1, 7, 6, 17),
				[]ast.Expression{ast.NewIdentifier(p(1, 7, 6, 6), "a"), ast.NewIdentifier(p(1, 10, 9, 10), "ok")},
				ast.AssignmentDeclaration, []ast.Expression{ast.NewSelector(p(1, 17, 15, 17), ast.NewIdentifier(p(1, 16, 15, 15), "b"), "c")}),
			ast.NewIdentifier(p(1, 21, 20, 20), "a"), ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 25, 24, 24), []byte("b"), ast.Cut{})}), nil)}, ast.ContextHTML)},
	{"{% if x < 10 %}x is < 10{% else if x == 10 %}x is 10{% else %}x is > 10 {% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewIf(p(1, 1, 0, 80), nil,
				ast.NewBinaryOperator(
					p(1, 9, 6, 11),
					ast.OperatorLess,
					ast.NewIdentifier(p(1, 7, 6, 6), "x"),
					ast.NewInt(p(1, 11, 10, 11), big.NewInt(10)),
				),
				ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 16, 15, 23), []byte("x is < 10"), ast.Cut{})}),
				ast.NewIf(p(1, 33, 32, 80), nil, // else if
					ast.NewBinaryOperator(
						p(1, 38, 35, 41),
						ast.OperatorEqual,
						ast.NewIdentifier(p(1, 36, 35, 35), "x"),
						ast.NewInt(p(1, 41, 40, 41), big.NewInt(10)),
					),
					ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 46, 45, 51), []byte("x is 10"), ast.Cut{})}),
					ast.NewBlock(nil, []ast.Node{ast.NewText(p(1, 63, 62, 71), []byte("x is > 10 "), ast.Cut{})}),
				),
			)}, ast.ContextHTML)},
	{"{% if _, ok := []int{1,2,3}.([]int); ok %}{% end %}", ast.NewTree("", []ast.Node{
		ast.NewIf(p(1, 1, 0, 50),
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
								{nil, ast.NewInt(p(1, 22, 21, 21), big.NewInt(1))},
								{nil, ast.NewInt(p(1, 24, 23, 23), big.NewInt(2))},
								{nil, ast.NewInt(p(1, 26, 25, 25), big.NewInt(3))},
							}),
						ast.NewSliceType(p(1, 30, 29, 33),
							ast.NewIdentifier(p(1, 32, 31, 33), "int"),
						))}),
			ast.NewIdentifier(p(1, 38, 37, 38), "ok"),
			nil,
			nil,
		),
	}, ast.ContextHTML)},
	{"{% if x == 5 %}{% end %}",
		ast.NewTree("", []ast.Node{
			ast.NewIf(&ast.Position{Line: 1, Column: 1, Start: 0, End: 23}, nil,
				ast.NewBinaryOperator(p(1, 9, 6, 11),
					ast.OperatorEqual,
					ast.NewIdentifier(p(1, 7, 6, 6), "x"),
					ast.NewInt(p(1, 12, 11, 11), big.NewInt(5)),
				), nil, nil),
		}, ast.ContextHTML)},
	{"{% extends \"/a.b\" %}", ast.NewTree("", []ast.Node{ast.NewExtends(p(1, 1, 0, 19), "/a.b", ast.ContextHTML)}, ast.ContextHTML)},
	{"{% include \"/a.b\" %}", ast.NewTree("", []ast.Node{ast.NewInclude(p(1, 1, 0, 19), "/a.b", ast.ContextHTML)}, ast.ContextHTML)},
	{"{% extends \"a.e\" %}{% macro b %}c{% end macro %}", ast.NewTree("", []ast.Node{
		ast.NewExtends(p(1, 1, 0, 18), "a.e", ast.ContextHTML),
		ast.NewMacro(p(1, 20, 19, 47), ast.NewIdentifier(p(1, 29, 28, 28), "b"),
			ast.NewFuncType(&ast.Position{Line: 1, Column: 20, Start: 19, End: 47}, nil, nil, false),
			[]ast.Node{ast.NewText(p(1, 33, 32, 32), []byte("c"), ast.Cut{})}, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% extends \"a.e\" %}{% macro b(c, d int) %}txt{% end macro %}", ast.NewTree("", []ast.Node{
		ast.NewExtends(p(1, 1, 0, 18), "a.e", ast.ContextHTML),
		ast.NewMacro(p(1, 20, 19, 59), ast.NewIdentifier(p(1, 29, 28, 28), "b"),
			ast.NewFuncType(&ast.Position{Line: 1, Column: 20, Start: 19, End: 59}, []*ast.Field{
				ast.NewField(ast.NewIdentifier(p(1, 31, 30, 30), "c"), nil),
				ast.NewField(ast.NewIdentifier(p(1, 34, 33, 33), "d"),
					ast.NewIdentifier(p(1, 36, 35, 37), "int")),
			}, nil, false),
			[]ast.Node{ast.NewText(p(1, 43, 42, 44), []byte("txt"), ast.Cut{})}, ast.ContextHTML)}, ast.ContextHTML)},
	{"{# comment\ncomment #}", ast.NewTree("", []ast.Node{ast.NewComment(p(1, 1, 0, 20), " comment\ncomment ")}, ast.ContextHTML)},
	{"{% macro a(i int) %}c{% end macro %}", ast.NewTree("", []ast.Node{
		ast.NewMacro(p(1, 1, 0, 35), ast.NewIdentifier(p(1, 10, 9, 9), "a"),
			ast.NewFuncType(&ast.Position{Line: 1, Column: 1, Start: 0, End: 35}, []*ast.Field{
				ast.NewField(ast.NewIdentifier(p(1, 12, 11, 11), "i"),
					ast.NewIdentifier(p(1, 14, 13, 15), "int")),
			}, nil, false),
			[]ast.Node{ast.NewText(p(1, 21, 20, 20), []byte("c"), ast.Cut{})}, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% macro a(b bool, c ...string) %}d{% end macro %}", ast.NewTree("", []ast.Node{
		ast.NewMacro(p(1, 1, 0, 49), ast.NewIdentifier(p(1, 10, 9, 9), "a"),
			ast.NewFuncType(&ast.Position{Line: 1, Column: 1, Start: 0, End: 49}, []*ast.Field{
				ast.NewField(ast.NewIdentifier(p(1, 12, 11, 11), "b"),
					ast.NewIdentifier(p(1, 14, 13, 16), "bool")),
				ast.NewField(ast.NewIdentifier(p(1, 20, 19, 19), "c"),
					ast.NewIdentifier(p(1, 25, 24, 29), "string")),
			}, nil, true),
			[]ast.Node{ast.NewText(p(1, 35, 34, 34), []byte("d"), ast.Cut{})}, ast.ContextHTML)}, ast.ContextHTML)},
	{"{% *a = 3 %}", ast.NewTree("", []ast.Node{
		ast.NewAssignment(p(1, 1, 0, 11),
			[]ast.Expression{
				ast.NewUnaryOperator(p(1, 4, 3, 4),
					ast.OperatorMultiplication,
					ast.NewIdentifier(p(1, 5, 4, 4), "a"),
				)},
			ast.AssignmentSimple,
			[]ast.Expression{
				ast.NewInt(p(1, 9, 8, 8), big.NewInt(3)),
			})}, ast.ContextHTML)},
}

func pageTests() map[string]struct {
	src  string
	tree *ast.Tree
} {
	var include = ast.NewInclude(p(3, 7, 29, 58), "/include2.html", ast.ContextHTML)
	include.Tree = ast.NewTree("", []ast.Node{
		ast.NewText(p(1, 1, 0, 4), []byte("<div>"), ast.Cut{}),
		ast.NewShow(p(1, 6, 5, 17), ast.NewIdentifier(p(1, 9, 8, 14), "content"), ast.ContextHTML),
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
				ast.NewShow(p(3, 14, 36, 46), ast.NewIdentifier(p(3, 17, 39, 43), "title"), ast.ContextHTML),
				ast.NewText(p(3, 25, 47, 68), []byte("</title></head>\n<body>"), ast.Cut{}),
				ast.NewShow(p(4, 7, 69, 81), ast.NewIdentifier(p(4, 10, 72, 78), "content"), ast.ContextHTML),
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
		"/include2.html": {
			"<div>{{ content }}</div>",
			nil,
		},
	}
}

func TestGoContextTrees(t *testing.T) {
	for _, tree := range goContextTreeTests {
		node, _, err := ParseSource([]byte(tree.src), true, false)
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

func TestTrees(t *testing.T) {
	for _, tree := range treeTests {
		node, _, err := ParseTemplateSource([]byte(tree.src), ast.ContextHTML)
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

func (tests testsReader) Read(path string, ctx ast.Context) ([]byte, error) {
	return []byte(tests[path].src), nil
}

func TestPages(t *testing.T) {

	t.Skip("(not runnable)")

	// tests := pageTests()
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
		if nn1.Context != nn2.Context {
			return fmt.Errorf("unexpected context %s, expecting %s", nn1.Context, nn2.Context)
		}

	case *ast.Extends:
		nn2, ok := n2.(*ast.Extends)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Path != nn2.Path {
			return fmt.Errorf("unexpected path %q, expecting %q", nn1.Path, nn2.Path)
		}
		if nn1.Context != nn2.Context {
			return fmt.Errorf("unexpected context %s, expecting %s", nn1.Context, nn2.Context)
		}
		err := equals(nn1.Tree, nn2.Tree, p)
		if err != nil {
			return err
		}

	case *ast.Include:
		nn2, ok := n2.(*ast.Include)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Path != nn2.Path {
			return fmt.Errorf("unexpected path %q, expecting %q", nn1.Path, nn2.Path)
		}
		if nn1.Context != nn2.Context {
			return fmt.Errorf("unexpected context %s, expecting %s", nn1.Context, nn2.Context)
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

	case *ast.Rune:
		nn2, ok := n2.(*ast.Rune)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Value != nn2.Value {
			return fmt.Errorf("unexpected %q, expecting %q", nn1.Value, nn2.Value)
		}

	case *ast.Int:
		nn2, ok := n2.(*ast.Int)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Value.Cmp(&nn2.Value) != 0 {
			return fmt.Errorf("unexpected %q, expecting %q", nn1.Value.String(), nn2.Value.String())
		}

	case *ast.Float:
		nn2, ok := n2.(*ast.Float)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		if nn1.Value.Cmp(&nn2.Value) != 0 {
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

	case *ast.Parenthesis:
		nn2, ok := n2.(*ast.Parenthesis)
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
		if len(nn1.FieldDecl) != len(nn2.FieldDecl) {
			return fmt.Errorf("struct type: unexpected fields len %#v, expecting %#v", len(nn1.FieldDecl), len(nn2.FieldDecl))
		}
		for i := range nn1.FieldDecl {
			fd1 := nn1.FieldDecl[i]
			fd2 := nn2.FieldDecl[i]
			if len(fd1.IdentifierList) != len(fd2.IdentifierList) {
				return fmt.Errorf("struct type: field %d: expecting %d identifiers, got %d", i, len(fd2.IdentifierList), len(fd1.IdentifierList))
			}
			err := equals(fd1.Type, fd2.Type, p)
			if err != nil {
				return fmt.Errorf("struct type: field %d: %s", i, err)
			}
			// TODO (Gianluca): add tags comparison.
		}

	case *ast.TypeDeclaration:
		nn2, ok := n2.(*ast.TypeDeclaration)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Identifier, nn2.Identifier, p)
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

	case *ast.Show:
		nn2, ok := n2.(*ast.Show)
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
		err = equals(nn1.Assignment, nn2.Assignment, p)
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

	case *ast.Defer:
		nn2, ok := n2.(*ast.Defer)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Call, nn2.Call, p)
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
		err := equals(nn1.Name, nn2.Name, p)
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

	case *ast.Macro:
		nn2, ok := n2.(*ast.Macro)
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
		if len(nn1.Body) != len(nn2.Body) {
			return fmt.Errorf("unexpected body len %d, expecting %d", len(nn1.Body), len(nn2.Body))
		}
		for i, node := range nn1.Body {
			err := equals(node, nn2.Body[i], p)
			if err != nil {
				return err
			}
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

	case *ast.ShowMacro:
		nn2, ok := n2.(*ast.ShowMacro)
		if !ok {
			return fmt.Errorf("unexpected %#v, expecting %#v", n1, n2)
		}
		err := equals(nn1.Import, nn2.Import, p)
		if err != nil {
			return err
		}
		err = equals(nn1.Macro, nn2.Macro, p)
		if err != nil {
			return err
		}
		if nn1.Or != nn2.Or {
			return fmt.Errorf("unexpected %s, expecting %s", nn1.Or, nn2.Or)
		}
		if len(nn1.Args) != len(nn2.Args) {
			return fmt.Errorf("unexpected arguments len %d, expecting %d", len(nn1.Args), len(nn2.Args))
		}
		for i, node := range nn1.Args {
			err := equals(node, nn2.Args[i], p)
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

	default:
		panic(fmt.Sprintf("unexpected node of type %T\n", n1))
	}

	return nil
}
