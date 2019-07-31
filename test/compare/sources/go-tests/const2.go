// run

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "fmt"

const (
	A int = 1 << (1 << iota)
	B
	C
	D
	E
)

func main() {
	s := fmt.Sprintf("%v %v %v %v %v", A, B, C, D, E)
	fmt.Println(s)
	if s != "2 4 16 256 65536" {
		println("type info didn't propagate in const: got", s)
		panic("fail")
	}
	x := uint(5)
	y := float64(uint64(1) << x) // used to fail to compile
	fmt.Println(x, y)
	if y != 32 {
		println("wrong y", y)
		panic("fail")
	}
}
