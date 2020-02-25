// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

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
}

// newCompilation returns a new compilation.
func newCompilation() *compilation {
	return &compilation{
		pkgPathToIndex: map[string]int{},
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
