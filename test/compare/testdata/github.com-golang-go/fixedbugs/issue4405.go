// errorcheck

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

const (
	_ = iota
	_ // ERROR "invalid character|invalid character"
	_  // ERROR "invalid character|invalid character"
	_  // ERROR "invalid character|invalid character"
	_  // ERROR "invalid character|invalid character"
)

func main() {}