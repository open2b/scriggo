// errorcheck

// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

const (
	c0   = 1 << 100
	c1   = c0 * c0
	c2   = c1 * c1
	c3   = c2 * c2 // ERROR "overflow"
)

func main() { }
