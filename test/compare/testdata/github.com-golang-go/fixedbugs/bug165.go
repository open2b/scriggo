// skip : interface definition https://github.com/open2b/scriggo/issues/218

// errorcheck

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

type I interface {
	m(map[I]bool) // ok
}

type S struct {
	m map[S]bool // ERROR "map key type"
}
