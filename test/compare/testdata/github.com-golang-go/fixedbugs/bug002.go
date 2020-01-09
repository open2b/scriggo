// skip : unexpected syntax error https://github.com/open2b/scriggo/issues/535

// run

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

func main() {
	if ; false {}  // compiles; should be an error (should be simplevardecl before ;)
}
