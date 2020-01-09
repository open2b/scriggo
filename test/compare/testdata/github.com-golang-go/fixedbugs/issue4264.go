// skip : should be a type checking error, got runtime panic https://github.com/open2b/scriggo/issues/530

// errorcheck

// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// issue 4264: reject int division by const 0

package main

func main() {
	var x int
	var y float64
	var z complex128

	_ = x

	println(x/0) // ERROR "division by zero"
	println(y/0)
	println(z/0)
}