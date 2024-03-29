// errorcheck

// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Compiler rejected initialization of structs to composite literals
// in a non-static setting (e.g. in a function)
// when the struct contained a field named _.

package main

type T struct {
	_ string
}

func ok() {
	var x = T{"check"}
	_ = x
	_ = T{"et"}
}

var (
	y = T{"stare"}
	w = T{_: "look"} // ERROR "invalid field name _ in struct initializer"
	_ = T{"page"}
	_ = T{_: "out"} // ERROR "invalid field name _ in struct initializer"
)

func bad() {
	var z = T{_: "verse"} ; _ = z // ERROR "invalid field name _ in struct initializer"
	_ = T{_: "itinerary"} // ERROR "invalid field name _ in struct initializer"
}

func main() { }