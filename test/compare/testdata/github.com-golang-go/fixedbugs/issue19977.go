// errorcheck

// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Issue 19977: multiple error messages when type switching on an undefined

package main

func Foo() {
	switch x := a.(type) { default: _ = x }// ERROR "undefined: a"
}

func main() { }
