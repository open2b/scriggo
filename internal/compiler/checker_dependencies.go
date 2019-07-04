// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"scriggo/internal/compiler/ast"
)

// Makes a dependency analysis on packages during parsing.
// See https://golang.org/ref/spec#Package_initialization for further
// informations.

type PackageDeclsDeps map[*ast.Identifier][]*ast.Identifier

// dependencies analyzes dependencis between global declarations.
type dependencies struct {
}

func depsAnalysis(t *ast.Tree, opts Options) PackageDeclsDeps {
	for _, n := range t.Nodes {
		switch n := n.(type) {
		case *ast.Var:
			_ = n
		default:

		}
	}
	return nil
}
