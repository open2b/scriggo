// skip : this test needs to be rewritten https://github.com/open2b/scriggo/issues/417

// errorcheck

// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Make sure we don't crash when reporting this error.

package p

func f() {
	if err := http.ListenAndServe(
} // ERROR "unexpected }, expecting expression"
