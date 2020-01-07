// errorcheck

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// issue 1664

package main

func main() {
	var i uint = 33
	var a = (1<<i) + 4.5 ; println(a) // ERROR "shift of type float64|invalid.*shift"
	
	
	var b = (1<<i) + 4.0 ; println(b) // ERROR "shift of type float64|invalid.*shift"
	

	var c int64 = (1<<i) + 4.0 ; println(c) // ok - it's all int64
	
}
