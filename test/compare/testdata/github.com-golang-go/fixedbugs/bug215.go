// skip : method definition (see https://github.com/open2b/scriggo/issues/194)

// errorcheck

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Used to crash the compiler.
// https://golang.org/issue/158

package main

type A struct {	a A }	// ERROR "recursive"
func foo()		{ new(A).bar() }
func (a A) bar()	{}
