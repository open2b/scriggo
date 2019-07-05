// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"fmt"
	"scriggo"
	"scriggo/ast"
	"strings"
	"testing"
)

func Test_treeTransformer(t *testing.T) {
	t.Run("program", func(t *testing.T) {
		stdout := strings.Builder{}
		packages := scriggo.MapStringLoader{"main": `
		package main
	
		func main() {
			println("hello, world!")
		}
		`}
		loadOpts := &scriggo.LoadOptions{
			TreeTransformer: func(tree *ast.Tree) error {
				pkg := tree.Nodes[0].(*ast.Package)
				main := pkg.Declarations[0].(*ast.Func)
				println := main.Body.Nodes[0].(*ast.Call)
				println.Args[0].(*ast.BasicLiteral).Value = `"hello, scriggo!"`
				return nil
			},
		}
		program, err := scriggo.LoadProgram(packages, loadOpts)
		if err != nil {
			t.Fatal(err)
		}
		runOptions := scriggo.RunOptions{
			PrintFunc: func(a interface{}) {
				stdout.WriteString(fmt.Sprintf("%v", a))
			},
		}
		err = program.Run(runOptions)
		if err != nil {
			t.Fatal(err)
		}
		expectedOutput := "hello, scriggo!"
		if stdout.String() != expectedOutput {
			t.Fatalf("expecting output %q, got %q", expectedOutput, stdout.String())
		}
	})
	t.Run("script", func(t *testing.T) {
		stdout := strings.Builder{}
		reader := strings.NewReader(`println("hello, world!")`)
		loadOpts := &scriggo.LoadOptions{
			TreeTransformer: func(tree *ast.Tree) error {
				println := tree.Nodes[0].(*ast.Call)
				println.Args[0].(*ast.BasicLiteral).Value = `"hello, scriggo!"`
				return nil
			},
		}
		script, err := scriggo.LoadScript(reader, scriggo.CombinedLoaders{}, loadOpts)
		if err != nil {
			t.Fatal(err)
		}
		runOptions := scriggo.RunOptions{
			PrintFunc: func(a interface{}) {
				stdout.WriteString(fmt.Sprintf("%v", a))
			},
		}
		err = script.Run(nil, runOptions)
		if err != nil {
			t.Fatal(err)
		}
		expectedOutput := "hello, scriggo!"
		if stdout.String() != expectedOutput {
			t.Fatalf("expecting output %q, got %q", expectedOutput, stdout.String())
		}
	})
}
