// skip : interface definition (https://github.com/open2b/scriggo/issues/218)

// errorcheck -lang=go1.17

// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Issue 10975: Returning an invalid interface would cause
// `internal compiler error: getinarg: not a func`.

package main

type I interface {
	int // ERROR "interface contains embedded non-interface|not an interface"
}

func New() I {
	return struct{}{}
}
