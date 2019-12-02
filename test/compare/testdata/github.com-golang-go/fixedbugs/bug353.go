// errorcheck

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// issue 2089 - internal compiler error

package main

import (
	"io"
	"os"
)

func echo(fd io.ReadWriterCloser) { } // ERROR "undefined.*io.ReadWriterCloser"

func main() {
	_ = io.Reader(nil)
	fd, _ := os.Open("a.txt")
	_ = fd
	// echo(fd)
}
