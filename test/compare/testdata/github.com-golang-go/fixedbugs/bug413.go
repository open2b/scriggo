// errorcheck

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

func f(i int) int { return i }

// TODO: gc error is different

var i = func() int {a := f(i); return a}()  // ERROR "typechecking loop"

func main() { }