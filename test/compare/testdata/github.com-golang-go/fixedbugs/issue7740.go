// skip : Scriggo prints "BUG: got 512; want +Inf". See https://github.com/golang/go/commit/6a4739ccc5198449d58d2e90a040c4fb908b3cb0

// run

// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This test computes the precision of the compiler's internal multiprecision floats.

package main

import (
	"fmt"
	"math"
	"runtime"
)

const ulp = (1.0 + (2.0 / 3.0)) - (5.0 / 3.0)

func main() {
	// adjust precision depending on compiler
	var prec float64
	switch runtime.Compiler {
	case "gc":
		prec = math.Inf(1) // exact precision using rational arithmetic
	case "gccgo":
		prec = 256
	default:
		// unknown compiler
		return
	}
	p := 1 - math.Log(math.Abs(ulp))/math.Log(2)
	if math.Abs(p-prec) > 1e-10 {
		fmt.Printf("BUG: got %g; want %g\n", p, prec)
	}
}
