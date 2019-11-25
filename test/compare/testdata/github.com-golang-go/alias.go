// skip : different error

// errorcheck

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Test that error messages say what the source file says
// (uint8 vs byte, int32 vs. rune).
// Does not compile.

package main

import (
	"fmt"
	"unicode/utf8"
)

func f(byte)  {}
func g(uint8) {}

func main() {
	var x float64
	_ = x
	f(x) // ERROR "byte"
	g(x) // ERROR "uint8"

	// Test across imports.

	var ff fmt.Formatter
	_ = ff
	var fs fmt.State
	_ = fs
	ff.Format(fs, x) // ERROR "rune"
	
	_ = utf8.RuneStart
	utf8.RuneStart(x) // ERROR "byte"
}
