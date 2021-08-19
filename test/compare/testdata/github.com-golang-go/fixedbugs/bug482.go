// skip : false positive 'type checking loop' (https://github.com/open2b/scriggo/issues/823)

// compile

// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Using the same name for a field in a composite literal and for a
// global variable that depends on the variable being initialized
// caused gccgo to erroneously report "variable initializer refers to
// itself".

package main

type S struct {
	F int
}

var V = S{F: 1}

var F = V.F

func main() {}
