// skip : require interface definition https://github.com/open2b/scriggo/issues/218

// errorcheck

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Verify that various erroneous type switches are caught by the compiler.
// Does not compile.

package main

import "io"

func whatis(x interface{}) string {
	switch x.(type) {
	case int:
		return "int"
	case int: // ERROR "duplicate"
		return "int8"
	case io.Reader:
		return "Reader1"
	case io.Reader: // ERROR "duplicate"
		return "Reader2"
	case interface {
		r()
		w()
	}:
		return "rw"
	case interface {	// ERROR "duplicate"
		w()
		r()
	}:
		return "wr"

	}
	return ""
}

func notused(x interface{}) {
	// The first t is in a different scope than the 2nd t; it cannot
	// be accessed (=> declared and not used error); but it is legal
	// to declare it.
	switch t := 0; t := x.(type) { // ERROR "declared and not used"
	case int:
		_ = t // this is using the t of "t := x.(type)"
	}
}
