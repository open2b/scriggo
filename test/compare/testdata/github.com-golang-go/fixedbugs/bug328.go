// skip : import "unsafe" https://github.com/open2b/scriggo/issues/288

// run

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "unsafe"

func main() {
	var p unsafe.Pointer
	println(p)
}
