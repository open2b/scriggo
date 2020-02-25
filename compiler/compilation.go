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
	pkgPathToIndex map[string]int
}

// newCompilation returns a new compilation.
func newCompilation() *compilation {
	return &compilation{
		pkgPathToIndex: map[string]int{},
	}
}
