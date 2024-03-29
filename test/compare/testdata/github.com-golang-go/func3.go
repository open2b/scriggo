// skip : https://github.com/open2b/scriggo/issues/541

// errorcheck

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Verify that illegal function signatures are detected.
// Does not compile.

package main

type t1 int
type t2 int
type t3 int

func f1(*t2, x t3)	// ERROR "named"
func f2(t1, *t2, x t3)	// ERROR "named"
func f3() (x int, *string)	// ERROR "named"

func f4() (t1 t1)	// legal - scope of parameter named t1 starts in body of f4.
