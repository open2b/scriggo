// skip : expected error on iota, got nothing https://github.com/open2b/scriggo/issues/426

// errorcheck

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

const X = iota

func f(x int) { }

func main() {
	f(X);
	f(iota);	// ERROR "iota"
	f(X);
}
