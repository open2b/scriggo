// ignore

//+build ignore

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

type structType = struct {
	A [10]int
}

var b structType

var m map[string][20]int

var s [][30]int

const (
	n1 = len(b.A)

// n2 = len(m[""])
// n3 = len(s[10])
)

func main() {

}
