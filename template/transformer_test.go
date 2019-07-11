// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"strings"
	"testing"

	"scriggo/ast"
)

func Test_treeTransformer(t *testing.T) {
	stdout := &strings.Builder{}
	reader := MapReader{"/index.html": []byte(`{% w := "hi, " %}{{ w }}world!`)}
	loadOpts := &LoadOptions{
		TreeTransformer: func(tree *ast.Tree) error {
			assignment := tree.Nodes[0].(*ast.Assignment)
			assignment.Rhs[0].(*ast.BasicLiteral).Value = `"hello, "`
			text := tree.Nodes[2].(*ast.Text)
			text.Text = []byte("scriggo!")
			return nil
		},
	}
	template, err := Load("/index.html", reader, nil, ContextText, loadOpts)
	if err != nil {
		t.Fatal(err)
	}
	err = template.Render(stdout, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	expectedOutput := "hello, scriggo!"
	if stdout.String() != expectedOutput {
		t.Fatalf("expecting output %q, got %q", expectedOutput, stdout.String())
	}
}
