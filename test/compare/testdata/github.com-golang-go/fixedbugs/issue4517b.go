// skip : cannot declare a constant with name 'init' https://github.com/open2b/scriggo/issues/532

// errorcheck

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

const init = 1 // ERROR "cannot declare init - must be func"

func main() {}