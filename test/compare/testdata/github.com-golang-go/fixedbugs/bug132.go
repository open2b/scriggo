// skip : type definition (see https://github.com/open2b/scriggo/issues/194)

// errorcheck

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

type T struct {
	x, x int  // ERROR "duplicate"
}
