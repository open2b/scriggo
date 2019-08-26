// errorcheck

// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Verify that constant definition loops are caught during
// typechecking and that the errors print correctly.

package main

const ( A = 1 + B ; B = C - 1 ; C = A + B + 1 ) // ERROR "constant definition loop"

func main() { }