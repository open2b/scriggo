// errorcheck

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

type t struct {
	x int; x int // ERROR "duplicate field x"
}


func main() { }