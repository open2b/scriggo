// skip : this test must be rewritten https://github.com/open2b/scriggo/issues/417

// errorcheck

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "fmt"	// GCCGO_ERROR "previous"

var fmt int	// ERROR "redecl|redefinition"
