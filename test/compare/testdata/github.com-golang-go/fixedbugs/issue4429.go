// errorcheck

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

type a struct {
  a int
}

func main() {
  av := a{};
  _ = av
  _ = *a(av); // ERROR "invalid indirect|expected pointer"
}
