// errorcheck

// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

func f() {
	g(f..3) // ERROR `syntax error: unexpected float, expecting name or (`
}

func main() { }
