// errorcheck

// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Issue 20185: type switching on untyped values (e.g. nil or consts)
// caused an internal compiler error.

package main

func F() {
	switch t := nil.(type) { default: _ = t } // ERROR "cannot type switch on non-interface value nil"
}

const x = 1

func G() {
	// TODO: Scriggo error: "cannot type switch on non-interface value x (type int)"
	// switch t := x.(type) { default: } ERROR "cannot type switch on non-interface value x \(type untyped number\)"
}

func main() { }