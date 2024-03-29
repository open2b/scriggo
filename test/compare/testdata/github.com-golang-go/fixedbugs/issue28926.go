// skip : interface definition https://github.com/open2b/scriggo/issues/218

// errorcheck

// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

type Stringer interface {
	String() string
}

func main() {
	var e interface{}
	switch e := e.(type) {
	case G: // ERROR "undefined: G"
		e.M() // ok: this error should be ignored because the case failed its typecheck
	case E: // ERROR "undefined: E"
		e.D() // ok: this error should be ignored because the case failed its typecheck
	case Stringer:
		// ok: this error should not be ignored to prove that passing legs aren't left out
		_ = e.(T) // ERROR "undefined: T"
	}
}
