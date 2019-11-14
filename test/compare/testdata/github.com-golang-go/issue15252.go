// skip : re-enable before merging 'types' into 'master'

// run

// Copyright 2016 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This test makes sure that we use all 64 bits of an
// index, even on 32 bit machines.  It also tests that nacl
// can compile 64 bit indexes loaded from ODOTPTR properly.

package main

type T struct {
	I int64 // TODO: should be unexported
}

func f(t *T) byte {
	b := [2]byte{3, 4}
	return b[t.I]
}

func main() {
	t := &T{0x100000001}
	defer func() {
		r := recover()
		if r == nil {
			panic("panic wasn't recoverable")
		}
	}()
	f(t)
	panic("index didn't panic")
}
