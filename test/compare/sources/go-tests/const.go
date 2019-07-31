

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

const (
	c0      = 0
	cm1     = -1
	chuge   = 1 << 100
	chuge_1 = chuge - 1
	c1      = chuge >> 100
	c3div2  = 3 / 2
	c1e3    = 1e3

	rsh1 = 1e100 >> 1000
	rsh2 = 1e302 >> 1000

	ctrue  = true
	cfalse = !ctrue
)

const (
	f0              = 0.0
	fm1             = -1.
	fhuge   float64 = 1 << 100
	fhuge_1 float64 = chuge - 1
	f1      float64 = chuge >> 100
	f3div2          = 3. / 2.
	f1e3    float64 = 1e3
)

func assert(t bool, s string) {
	if !t {
		panic(s)
	}
}

func ints() {
	assert(c0 == 0, "c0")
	assert(c1 == 1, "c1")
	assert(chuge > chuge_1, "chuge")
	assert(chuge_1+1 == chuge, "chuge 1")
	assert(chuge+cm1+1 == chuge, "cm1")
	assert(c3div2 == 1, "3/2")
	assert(c1e3 == 1000, "c1e3 int")
	assert(c1e3 == 1e3, "c1e3 float")
	assert(rsh1 == 0, "rsh1")
	assert(rsh2 == 9, "rsh2")

	// verify that all (in range) are assignable as ints
	var i int
	i = c0
	assert(i == c0, "i == c0")
	i = cm1
	assert(i == cm1, "i == cm1")
	i = c1
	assert(i == c1, "i == c1")
	i = c3div2
	assert(i == c3div2, "i == c3div2")
	i = c1e3
	assert(i == c1e3, "i == c1e3")

	// verify that all are assignable as floats
	// TODO(Gianluca): https://github.com/open2b/scriggo/issues/201
	var f float64
	// f = c0
	// assert(f == c0, "f == c0")
	// f = cm1
	// assert(f == cm1, "f == cm1")
	f = chuge
	assert(f == chuge, "f == chuge")
	f = chuge_1
	assert(f == chuge_1, "f == chuge_1")
	f = c1
	assert(f == c1, "f == c1")
	f = c3div2
	assert(f == c3div2, "f == c3div2")
	// f = c1e3
	// assert(f == c1e3, "f == c1e3")
}

func floats() {
	assert(f0 == c0, "f0")
	assert(f1 == c1, "f1")
	assert(fhuge_1+1 == fhuge, "fhuge 1")
	assert(fhuge+fm1+1 == fhuge, "fm1")
	assert(f3div2 == 1.5, "3./2.")
	assert(f1e3 == 1000, "f1e3 int")
	assert(f1e3 == 1.e3, "f1e3 float")

	// verify that all (in range) are assignable as ints
	var i int
	i = f0
	assert(i == f0, "i == f0")
	i = fm1
	assert(i == fm1, "i == fm1")
}

func main() {
	ints()
	floats()
	assert(ctrue == true, "ctrue == true")
	assert(cfalse == false, "cfalse == false")
}
