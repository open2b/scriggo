// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"strings"
	"testing"

	"scrigo/internal/compiler/ast"
)

func extractVariableInitializationOrder(tree *ast.Tree) []string {
	order := []string{}
	pkg, _ := tree.Nodes[0].(*ast.Package)
	for _, n := range pkg.Declarations {
		switch n := n.(type) {
		case *ast.Var:
			for _, ident := range n.Lhs {
				order = append(order, ident.Name)
			}
		}
	}
	return order
}

func TestVariablesInitializationOrder(t *testing.T) {
	cases := []struct {
		src   string
		order []string
	}{
		// Just one variable.
		{`    var A = 1
			`, []string{"A"}},

		// Variables are independent from each others.
		{`    var A = 1
			var B = 2
			`, []string{"A", "B"}},

		// B depends on A.
		{`    var A = B
			var B = 10
			`, []string{"B", "A"}},

		// // Three variables in reverse order.
		// {`    var A = B
		// 	var B = C
		// 	var C = 1
		// 	`, []string{"C", "B", "A"}},

		// B and C depends from A, but variables are already ordered.
		{`    var A = 1
			var B = A
			var C = B
			`, []string{"A", "B", "C"}},
	}
CasesLoop:
	for _, c := range cases {
		pkgInfos := make(map[string]*PackageInfo)
		c.src = "package main\n" + c.src + "func main() { }"
		tree, _, err := ParseSource([]byte(c.src), true, ast.ContextNone)
		errorSrc := strings.ReplaceAll(c.src, "\n", " ")
		errorSrc = strings.ReplaceAll(errorSrc, "\t", "")
		if err != nil {
			t.Errorf("source: %q, parsing error: %s", errorSrc, err)
			continue
		}
		err = CheckPackage(tree, nil, pkgInfos)
		if err != nil {
			t.Errorf("source: %q, type-checking error: %s", errorSrc, err)
			continue
		}
		got := extractVariableInitializationOrder(tree)
		expected := c.order
		if len(got) != len(expected) {
			t.Errorf("source: %q, expecting %s, got %s", errorSrc, expected, got)
			continue
		}
		for i := range got {
			if expected[i] != got[i] {
				t.Errorf("source: %q, expecting %s, got %s", errorSrc, expected, got)
				continue CasesLoop
			}
		}
	}
}
