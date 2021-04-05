// Copyright (c) 2021 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

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

//go:noinline
func g(s string) int {
	if s == "" {
		return 0
	}
	return 1
}

//go:noinline
func h(x int, s string, y []int) (int, int) {
	return x + len(s) + len(y), 2
}

//go:noinline
func p() {
	a = 1
}

//go:noinline
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
