// errorcheck

// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

func foo() (int, int) { return 2.3 } // ERROR "not enough arguments to return"

func foo2() { return int(2), 2 } // ERROR "too many arguments to return"

func foo3(v int) (a, b, c, d int) {
	if v >= 0 {
		return 1 // ERROR "not enough arguments to return"
	}
	return 2, 3 // ERROR "not enough arguments to return"
	return
}

func foo4(name string) (string, int) {
	switch name {
	case "cow":
		return "moo" // ERROR "not enough arguments to return"
	case "dog":
		return "dog", 10, true // ERROR "too many arguments to return"
	case "fish":
		return "" // ERROR "not enough arguments to return"
	default:
		return "lizard", 10
	}
	return "", 0
}

type S int
type T string
type U float64

func foo5() (S, T, U) {
	if false {
		return "" // ERROR "not enough arguments to return"
	} else {
		ptr := new(T)
		_ = ptr
		return ptr // ERROR "not enough arguments to return"
	}
	return new(S), 12.34, 1 + 0i, 'r', true // ERROR "too many arguments to return"
	return 0, "", 0
}

func foo6() (T, string) { return "T", true, true } // ERROR "too many arguments to return"

func main() {}
