// skip : invalid recursive type (https://github.com/open2b/scriggo/issues/440)

// compile

// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Variable in enclosing function with same name as field in struct
// composite literal confused gccgo.

package main

type s1 struct {
	f *s1
}

func F() {
	var f *s1
	_ = func() {
		_ = s1{f: nil}
	}
	_ = f
}
