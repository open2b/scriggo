// skip : problem in variadic call https://github.com/open2b/scriggo/issues/265

// run

// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// https://golang.org/issue/662

package main

import "fmt"

func f() (int, int) { return 1, 2 }

func main() {
	s := fmt.Sprint(f())
	if s != "1 2" {	// with bug, was "{1 2}"
		panic("BUG")
	}
}
