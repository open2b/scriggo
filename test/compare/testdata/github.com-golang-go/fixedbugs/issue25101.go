// skip : interface definition https://github.com/open2b/scriggo/issues/218

// compile

// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Indexed export format must not crash when writing
// the anonymous parameter for m.

package p

var x interface {
	m(int)
}

var M = x.m
