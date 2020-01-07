// skip : the test pass but it slows down tests execution https://github.com/open2b/scriggo/issues/503

// compile

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// issue 1908
// unreasonable width used to be internal fatal error

package main

func main() {
	buf := [1<<30]byte{}
	_ = buf[:]
}
