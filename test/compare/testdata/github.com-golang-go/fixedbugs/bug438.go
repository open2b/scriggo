// compile

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Gccgo used to incorrectly give an error when compiling this.

package main

func F() (i int) {
	for first := true; first; first = false {
		i++
	}
	return
}

func main() {}