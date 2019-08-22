// run

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Test the 'for range' construct.

package main

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func testblankvars() {
	n := 0
	for range alphabet {
		n++
	}
	if n != 26 {
		println("for range: wrong count", n, "want 26")
		panic("fail")
	}
	n = 0
	for _ = range alphabet {
		n++
	}
	if n != 26 {
		println("for _ = range: wrong count", n, "want 26")
		panic("fail")
	}
	n = 0
	for _, _ = range alphabet {
		n++
	}
	if n != 26 {
		println("for _, _ = range: wrong count", n, "want 26")
		panic("fail")
	}
	s := 0
	for i, _ := range alphabet {
		s += i
	}
	if s != 325 {
		println("for i, _ := range: wrong sum", s, "want 325")
		panic("fail")
	}
	r := rune(0)
	for _, v := range alphabet {
		r += v
	}
	if r != 2847 {
		println("for _, v := range: wrong sum", r, "want 2847")
		panic("fail")
	}
}

func makeslice() []int {
	return []int{1, 2, 3, 4, 5}
}

func testslice() {
	s := 0
	for _, v := range makeslice() {
		s += v
	}
	if s != 15 {
		println("wrong sum ranging over makeslice", s)
		panic("fail")
	}
	// https://github.com/open2b/scriggo/issues/182
	// x := []int{10, 20}
	// y := []int{99}
	// i := 1
	// for i, x[i] = range y {
	// 	break
	// }
	// if i != 0 || x[0] != 10 || x[1] != 99 {
	// 	println("wrong parallel assignment", i, x[0], x[1])
	// 	panic("fail")
	// }
}

func testslice1() {
	s := 0
	for i := range makeslice() {
		s += i
	}
	if s != 10 {
		println("wrong sum ranging over makeslice", s)
		panic("fail")
	}
}

func testslice2() {
	n := 0
	for range makeslice() {
		n++
	}
	if n != 5 {
		println("wrong count ranging over makeslice", n)
		panic("fail")
	}
}

func makearray() [5]int {
	return [5]int{1, 2, 3, 4, 5}
}

func testarray() {
	s := 0
	for _, v := range makearray() {
		s += v
	}
	if s != 15 {
		println("wrong sum ranging over makearray", s)
		panic("fail")
	}
}

func testarray1() {
	s := 0
	for i := range makearray() {
		s += i
	}
	if s != 10 {
		println("wrong sum ranging over makearray", s)
		panic("fail")
	}
}

func testarray2() {
	n := 0
	for range makearray() {
		n++
	}
	if n != 5 {
		println("wrong count ranging over makearray", n)
		panic("fail")
	}
}

func main() {
	testblankvars()
	testslice()
	testslice1()
	testslice2()
	testarray()
	testarray1()
	testarray2()
}
