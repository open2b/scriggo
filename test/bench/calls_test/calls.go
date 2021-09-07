// Copyright 2021 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

var a int

func main() {
	for i := 0; i < 100; i++ {
		a = k(i)
	}
	_ = a
}

func f(x, y int) int {
	return x + y
}

func g(s string) int {
	if s == "" {
		return 0
	}
	return 1
}

func h(x int, s string, y []int) (int, int) {
	return x + len(s) + len(y), 2
}

func p() {
	a = 1
}

func k(x int) int {
	v := func() int {
		return g("")
	}
	w := func(x int) int {
		return x + 3
	}
	p()
	y, z := h(x, "", nil)
	return w(g("")) + v() + f(3, 5) + y - z
}
