// compile

// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

func Spin() {
l1:
	for true {
		goto l1
	l2:
		if true {
			goto l2
		}
	}
}

func main() { }