// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util_test

import (
	"bytes"
	"fmt"

	"open2b/template/ast"
	"open2b/template/parser"
	"open2b/template/util"
)

func ExampleDump() {
	stringCases := []string{
		"{{ (4 + 5) * value() }}",
		"{% var x = 10 %}",
		"{% for i in 1..12 %} some text, blah blah blah {% end %}",
		"{% for i in 1..12 %} some very very very very very very very very long text, blah blah blah {% end %}",
		`{% show "ciao.txt" %}`,
		"{{5+6}}",
		`{% var x = 10 %}`,
		`{% y = 10 %}`,
		`{% y = (4 + 5) %}`,
	}

	for _, c := range stringCases {
		tree, err := parser.Parse([]byte(c), ast.ContextHTML)
		if err != nil {
			panic(err)
		}

		var buf bytes.Buffer
		err = util.Dump(&buf, tree)
		if err != nil {
			panic(err)
		}
		got := buf.String()

		fmt.Print(got)
	}

	// Output: Tree: "":1:1
	// │    Value (1:1) {{ (4 + 5) * value() }}
	// │    │    BinaryOperator (1:12) (4 + 5) * value()
	// │    │    │    BinaryOperator (1:7) 4 + 5
	// │    │    │    │    Int (1:5) 4
	// │    │    │    │    Int (1:9) 5
	// │    │    │    Call (1:19) value()

	// Tree: "":1:1
	// │    Var (1:1) {% var x = 10 %}
	// │    │    Int (1:12) 10

	// Tree: "":1:1
	// │    For (1:1) 1:1
	// │    │    Int (1:13) 1
	// │    │    Int (1:16) 12
	// │    │    Text (1:21) " some text, blah blah blah "

	// Tree: "":1:1
	// │    For (1:1) 1:1
	// │    │    Int (1:13) 1
	// │    │    Int (1:16) 12
	// │    │    Text (1:21) " some very very very very very..."

	// Tree: "":1:1
	// │    ShowPath (1:1) 1:1

	// Tree: "":1:1
	// │    Value (1:1) {{ 5 + 6 }}
	// │    │    BinaryOperator (1:4) 5 + 6
	// │    │    │    Int (1:3) 5
	// │    │    │    Int (1:5) 6

	// Tree: "":1:1
	// │    Var (1:1) {% var x = 10 %}
	// │    │    Int (1:12) 10

	// Tree: "":1:1
	// │    Assignment (1:1) {% y = 10 %}
	// │    │    Int (1:8) 10

	// Tree: "":1:1
	// │    Assignment (1:1) {% y = 4 + 5 %}
	// │    │    BinaryOperator (1:11) 4 + 5
	// │    │    │    Int (1:9) 4
	// │    │    │    Int (1:13) 5
}
