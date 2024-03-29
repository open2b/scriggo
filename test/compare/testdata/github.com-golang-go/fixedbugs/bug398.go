// skip : interface declaration https://github.com/open2b/scriggo/issues/218

// compile

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Used to crash compiler in interface type equality check.
// (This test used to have problems - see #15596.)

package p

// exported interfaces

type I1 interface {
      F() interface{I1}
}

type I2 interface {
      F() interface{I2}
}

var V1 I1
var V2 I2

func F() bool {
       return V1 == V2
}

// non-exported interfaces

type i1 interface {
      F() interface{i1}
}

type i2 interface {
      F() interface{i2}
}

var v1 i1
var v2 i2

func f() bool {
       return v1 == v2
}
