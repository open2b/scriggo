// skip : different errors

// errorcheck

// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Verify that copy arguments requirements are enforced by the
// compiler.

package main

func main() {

	si := make([]int, 8)
	_ = si
	sf := make([]float64, 8)
	_ = sf

	_ = copy()        // ERROR "not enough arguments"
	_ = copy(1, 2, 3) // ERROR "too many arguments"

	_ = copy(si, "hi") // ERROR "have different element types.*int.*string"
	_ = copy(si, sf)   // ERROR "have different element types.*int.*float64"

	_ = copy(1, 2)  // ERROR "must be slices; have int, int"
	_ = copy(1, si) // ERROR "first argument to copy should be"
	_ = copy(si, 2) // ERROR "second argument to copy should be"

}
