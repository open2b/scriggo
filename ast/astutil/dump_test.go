// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package astutil_test

import (
	"bytes"
	"fmt"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/ast/astutil"
	"github.com/open2b/scriggo/internal/compiler"
)

func ExampleDump() {
	cases := []string{
		"{{ (4 + 5) * value() }}",
		"{% var x = 10 %}",
		"{% if true %} some text, blah blah {% end %}",
		"{% for i in x %} some text, blah blah blah {% end %}",
		"{% for i in x %} some very very very very very very very very long text, blah blah blah {% end %}",
		`{{ render "ciao.txt" }}`,
		"{{5+6}}",
		`{% var x = 10 %}`,
		`{% y = 10 %}`,
		`{% y = (4 + 5) %}`,
	}

	for _, c := range cases {
		tree, _, err := compiler.ParseTemplateSource([]byte(c), ast.FormatHTML, false, false, false)
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

	// Output: Tree: "":1:1
	// │    Show (1:1) show (4 + 5) * value()
	// │    │    BinaryOperator (1:12) (4 + 5) * value()
	// │    │    │    BinaryOperator (1:7) 4 + 5
	// │    │    │    │    BasicLiteral (1:5) 4
	// │    │    │    │    BasicLiteral (1:9) 5
	// │    │    │    Call (1:19) value()
	//
	// Tree: "":1:1
	// │    Var (1:4) var x = 10
	// │    │    Identifier (1:8) x
	// │    │    BasicLiteral (1:12) 10
	//
	// Tree: "":1:1
	// │    If (1:4) true
	// │    │    Identifier (1:7) true
	// │    │    Block (-) block statement
	// │    │    │    Text (1:14) " some text, blah blah "
	//
	// Tree: "":1:1
	// │    ForIn (1:4) 1:4
	// │    │    Identifier (1:8) i
	// │    │    Identifier (1:13) x
	// │    │    Text (1:17) " some text, blah blah blah "
	//
	// Tree: "":1:1
	// │    ForIn (1:4) 1:4
	// │    │    Identifier (1:8) i
	// │    │    Identifier (1:13) x
	// │    │    Text (1:17) " some very very very very very..."
	//
	// Tree: "":1:1
	// │    Show (1:1) show render "ciao.txt"
	// │    │    Render (1:4) render "ciao.txt"
	//
	// Tree: "":1:1
	// │    Show (1:1) show 5 + 6
	// │    │    BinaryOperator (1:4) 5 + 6
	// │    │    │    BasicLiteral (1:3) 5
	// │    │    │    BasicLiteral (1:5) 6
	//
	// Tree: "":1:1
	// │    Var (1:4) var x = 10
	// │    │    Identifier (1:8) x
	// │    │    BasicLiteral (1:12) 10
	//
	// Tree: "":1:1
	// │    Assignment (1:4) y = 10
	// │    │    Identifier (1:4) y
	// │    │    BasicLiteral (1:8) 10
	//
	// Tree: "":1:1
	// │    Assignment (1:4) y = 4 + 5
	// │    │    Identifier (1:4) y
	// │    │    BinaryOperator (1:11) 4 + 5
	// │    │    │    BasicLiteral (1:9) 4
	// │    │    │    BasicLiteral (1:13) 5

}
