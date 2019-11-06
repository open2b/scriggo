// errorcheck

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

type S struct {
	A, B int // TODO: these two fields should be unexported
}

func main() {
	s1 := S{A: 7};	// ok - field is named
	s3 := S{7, 11};	// ok - all fields have values
	s2 := S{7}; _ = s2;	// ERROR "too few"
	_, _ = s1, s3,
}
