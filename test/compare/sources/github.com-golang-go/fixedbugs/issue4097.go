// skip : constant depending on variable https://github.com/open2b/scriggo/issues/206

// do not run : infinite loop

// errorcheck

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package foo

var s [][10]int
const m = len(s[len(s)-1]) // ERROR "is not a constant|is not constant" 

