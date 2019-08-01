// skip : this test cannot be run with the given implementation of errorcheck https://github.com/open2b/scriggo/issues/247

// errorcheck

// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Verify that constant definition loops are caught during
// typechecking and that the errors print correctly.

package main

const A = 1 + B // ERROR "constant definition loop\n.*A uses B\n.*B uses C\n.*C uses A"
const B = C - 1 // ERROR "constant definition loop\n.*B uses C\n.*C uses B"
const C = A + B + 1
