// skip : interface definition is not supported in Scriggo https://github.com/open2b/scriggo/issues/218

// errorcheck

// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Issue 20245: panic while formatting an error message

package p

var e = interface{ I1 } // ERROR "undefined: I1"
