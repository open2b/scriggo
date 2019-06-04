// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package astutil_test

import (
	"bytes"
	"fmt"
	"scriggo/internal/compiler"
	"scriggo/internal/compiler/ast"
	"scriggo/internal/compiler/ast/astutil"
)

func ExampleDump() {
	cases := map[ast.Context][]string{
		ast.ContextHTML: {
			"{{ (4 + 5) * value() }}",
			"{% var x = 10 %}",
			"{% for i in x %} some text, blah blah blah {% end %}",
			"{% for i in x %} some very very very very very very very very long text, blah blah blah {% end %}",
			`{% include "ciao.txt" %}`,
			"{{5+6}}",
			`{% var x = 10 %}`,
			`{% y = 10 %}`,
			`{% y = (4 + 5) %}`,
		},
		ast.ContextGo: {
			"5 + 6",
			"map[string]([]interface{})",
		},
	}

	for _, ctx := range []ast.Context{ast.ContextHTML, ast.ContextGo} {
		stringCases := cases[ctx]
		for _, c := range stringCases {
			var tree *ast.Tree
			var err error
			if ctx == ast.ContextGo {
				tree, _, err = compiler.ParseSource([]byte(c), true, false)
			} else {
				tree, err = compiler.ParseTemplateSource([]byte(c), ctx)
			}
			if err != nil {
				panic(err)
			}
			var buf bytes.Buffer
			err = astutil.Dump(&buf, tree)
			if err != nil {
				panic(err)
			}
			got := buf.String()

			fmt.Print(got)
		}
	}

	// Output: Tree: "":1:1
	// │    Show (1:1) {{ (4 + 5) * value() }}
	// │    │    BinaryOperator (1:12) (4 + 5) * value()
	// │    │    │    BinaryOperator (1:7) 4 + 5
	// │    │    │    │    Int (1:5) 4
	// │    │    │    │    Int (1:9) 5
	// │    │    │    Call (1:19) value()
	//
	// Tree: "":1:1
	// │    Var (1:4) var x = 10
	// │    │    Identifier (1:8) x
	// │    │    Int (1:12) 10
	//
	// Tree: "":1:1
	// │    ForRange (1:1) 1:1
	// │    │    Assignment (1:8) _, i := x
	// │    │    │    Identifier (1:8) _
	// │    │    │    Identifier (1:8) i
	// │    │    │    Identifier (1:13) x
	// │    │    Text (1:17) " some text, blah blah blah "
	//
	// Tree: "":1:1
	// │    ForRange (1:1) 1:1
	// │    │    Assignment (1:8) _, i := x
	// │    │    │    Identifier (1:8) _
	// │    │    │    Identifier (1:8) i
	// │    │    │    Identifier (1:13) x
	// │    │    Text (1:17) " some very very very very very..."
	//
	// Tree: "":1:1
	// │    Include (1:1) 1:1
	//
	// Tree: "":1:1
	// │    Show (1:1) {{ 5 + 6 }}
	// │    │    BinaryOperator (1:4) 5 + 6
	// │    │    │    Int (1:3) 5
	// │    │    │    Int (1:5) 6
	//
	// Tree: "":1:1
	// │    Var (1:4) var x = 10
	// │    │    Identifier (1:8) x
	// │    │    Int (1:12) 10
	//
	// Tree: "":1:1
	// │    Assignment (1:1) y = 10
	// │    │    Identifier (1:4) y
	// │    │    Int (1:8) 10
	//
	// Tree: "":1:1
	// │    Assignment (1:1) y = 4 + 5
	// │    │    Identifier (1:4) y
	// │    │    BinaryOperator (1:11) 4 + 5
	// │    │    │    Int (1:9) 4
	// │    │    │    Int (1:13) 5
	//
	// Tree: "":1:1
	// │    BinaryOperator (1:3) 5 + 6
	// │    │    Int (1:1) 5
	// │    │    Int (1:5) 6
	//
	// Tree: "":1:1
	// │    MapType (1:1) map[string][]interface{}
	// │    │    Identifier (1:5) string
	// │    │    SliceType (1:13) []interface{}
	// │    │    │    Identifier (1:15) interface{}

}
