// skip : invalid recursive type alias https://github.com/open2b/scriggo/issues/440

// errorcheck

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

type T *struct {
	T;	// ERROR "embed.*pointer"
}

func main() { }