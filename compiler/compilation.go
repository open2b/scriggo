// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import "github.com/open2b/scriggo/compiler/ast"

// A compilation holds the state of a single compilation.
//
// This is necessary to store information across compilation of different
// packages/template pages, where a new type checker is created for every
// package/template page.
//
// Currently the compilation is used only by the typechecker.
//
type compilation struct {
	// pkgPathToIndex maps the path of a package to an unique int identifier.
	pkgPathToIndex map[string]int

	// pkgInfos maps the packages path to their respective package infos.
	pkgInfos map[string]*packageInfo

	// typeInfos associates a TypeInfo to the nodes of the AST that is
	// currently being type checked.
	//
	// It is correct to store type infos in the compilation because we
	// guarantee that an AST node is type checked always in the same way, no
	// matter what "path" is taken to reach it.
	typeInfos map[ast.Node]*typeInfo

	// alreadySortedPkgs tracks the packages that have already been sorted.
	// Sorting a package twice is wrong because it may have been transformed by
	// the type checker.
	alreadySortedPkgs map[*ast.Package]bool

	// indirectVars contains the list of all declarations of variables which
	// must be emitted as "indirect".
	indirectVars map[*ast.Identifier]bool

	// partialMacros contains the dummy macro declarations that have the
	// partial files in their bodies.
	partialMacros map[*ast.Tree]*ast.Func

	// partialImports contains the dummy 'import' statements that import the
	// files declaring the dummy macros.
	partialImports map[*ast.Tree]*ast.Import
}

// newCompilation returns a new compilation.
func newCompilation() *compilation {
	return &compilation{
		pkgInfos:          map[string]*packageInfo{},
		pkgPathToIndex:    map[string]int{},
		typeInfos:         map[ast.Node]*typeInfo{},
		alreadySortedPkgs: map[*ast.Package]bool{},
		indirectVars:      map[*ast.Identifier]bool{},
		partialMacros:     map[*ast.Tree]*ast.Func{},
		partialImports:    map[*ast.Tree]*ast.Import{},
	}
}

// UniqueIndex returns an index related to the current package; such index is
// unique for every package path.
//
// TODO(Gianluca): we should keep an index of the last (or the next) package
// index, instead of recalculate it every time.
func (compilation *compilation) UniqueIndex(path string) int {
	i, ok := compilation.pkgPathToIndex[path]
	if ok {
		return i
	}
	max := -1
	for _, i := range compilation.pkgPathToIndex {
		if i > max {
			max = i
		}
	}
	compilation.pkgPathToIndex[path] = max + 1
	return max + 1
}
