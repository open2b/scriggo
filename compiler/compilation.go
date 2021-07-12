// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"sort"
	"strconv"

	"github.com/open2b/scriggo/compiler/ast"
)

// A compilation holds the state of a single compilation.
//
// This is necessary to store information across compilation of different
// packages/template files, where a new type checker is created for every
// package/template file.
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

	// renderImportMacro stores the dummy 'import' nodes and the dummy macro
	// declarations that are used to implement the 'render' expression. This
	// maps avoid making useless copies of AST nodes that may lead to
	// inconsistent type checks and invalid behaviors.
	renderImportMacro map[*ast.Tree]renderIR

	// thisToUsingData maps 'this' identifiers ($this0, $this1... ) to their
	// corresponding usingData.
	thisToUsingData map[string]usingData

	// currentThisIndex is the index used to generate the name of the current
	// 'this' identifier.
	currentThisIndex int
}

type renderIR struct {
	Import *ast.Import
	Macro  *ast.Func
}

// newCompilation returns a new compilation.
func newCompilation() *compilation {
	return &compilation{
		pkgInfos:          map[string]*packageInfo{},
		pkgPathToIndex:    map[string]int{},
		typeInfos:         map[ast.Node]*typeInfo{},
		alreadySortedPkgs: map[*ast.Package]bool{},
		indirectVars:      map[*ast.Identifier]bool{},
		renderImportMacro: map[*ast.Tree]renderIR{},
		currentThisIndex:  -1,
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

// thisIncreaseIndex increases the index used in the name of the 'this'
// identifier, in order to make it unique.
func (compilation *compilation) thisIncreaseIndex() {
	compilation.currentThisIndex++
}

// thisCurrentName returns the current name of the 'this' identifier.
func (compilation *compilation) thisCurrentName() string {
	return "$this" + strconv.Itoa(compilation.currentThisIndex)
}

// close closes a compilation, using tc to report any type checking error
// encountered during the closing.
func (compilation *compilation) close(tc *typechecker) error {
	thisNames := make([]string, 0, len(compilation.thisToUsingData))
	for name := range compilation.thisToUsingData {
		thisNames = append(thisNames, name)
	}
	sort.Strings(thisNames)
	for _, thisName := range thisNames {
		ud := compilation.thisToUsingData[thisName]
		if !ud.used {
			return tc.errorf(ud.pos, "predeclared identifier this not used")
		}
		if !ud.toBeEmitted {
			varDecl := ud.thisDeclaration
			if len(varDecl.Lhs) != 1 || len(varDecl.Rhs) != 1 {
				panic("BUG: unexpected")
			}
			lh := ast.NewIdentifier(nil, "_")
			rh := ast.NewBasicLiteral(nil, ast.IntLiteral, "0")
			varDecl.Lhs = []*ast.Identifier{lh}
			varDecl.Rhs = []ast.Expression{rh}
			tc.checkNodes([]ast.Node{varDecl})
		}
	}
	return nil
}
