// compile

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// used to crash the compiler

package main

type T = struct {
	X [4]byte
}

var p *T
var v = *p

func main() { }

