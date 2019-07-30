//+build ignore

// run

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Test floating-point comparison involving NaN.

package main

import (
	"fmt"
	"math"
)

var nan float64 = math.NaN()
var f float64 = 1

var tests = []struct {
	Name string
	Expr bool
	Want bool
}{
	{"nan == nan", nan == nan, false},
	{"nan != nan", nan != nan, true},
	{"nan < nan", nan < nan, false},
	{"nan > nan", nan > nan, false},
	{"nan <= nan", nan <= nan, false},
	{"nan >= nan", nan >= nan, false},
	{"f == nan", f == nan, false},
	{"f != nan", f != nan, true},
	{"f < nan", f < nan, false},
	{"f > nan", f > nan, false},
	{"f <= nan", f <= nan, false},
	{"f >= nan", f >= nan, false},
	{"nan == f", nan == f, false},
	{"nan != f", nan != f, true},
	{"nan < f", nan < f, false},
	{"nan > f", nan > f, false},
	{"nan <= f", nan <= f, false},
	{"nan >= f", nan >= f, false},
	{"!(nan == nan)", !(nan == nan), true},
	{"!(nan != nan)", !(nan != nan), false},
	{"!(nan < nan)", !(nan < nan), true},
	{"!(nan > nan)", !(nan > nan), true},
	{"!(nan <= nan)", !(nan <= nan), true},
	{"!(nan >= nan)", !(nan >= nan), true},
	{"!(f == nan)", !(f == nan), true},
	{"!(f != nan)", !(f != nan), false},
	{"!(f < nan)", !(f < nan), true},
	{"!(f > nan)", !(f > nan), true},
	{"!(f <= nan)", !(f <= nan), true},
	{"!(f >= nan)", !(f >= nan), true},
	{"!(nan == f)", !(nan == f), true},
	{"!(nan != f)", !(nan != f), false},
	{"!(nan < f)", !(nan < f), true},
	{"!(nan > f)", !(nan > f), true},
	{"!(nan <= f)", !(nan <= f), true},
	{"!(nan >= f)", !(nan >= f), true},
	{"!!(nan == nan)", !!(nan == nan), false},
	{"!!(nan != nan)", !!(nan != nan), true},
	{"!!(nan < nan)", !!(nan < nan), false},
	{"!!(nan > nan)", !!(nan > nan), false},
	{"!!(nan <= nan)", !!(nan <= nan), false},
	{"!!(nan >= nan)", !!(nan >= nan), false},
	{"!!(f == nan)", !!(f == nan), false},
	{"!!(f != nan)", !!(f != nan), true},
	{"!!(f < nan)", !!(f < nan), false},
	{"!!(f > nan)", !!(f > nan), false},
	{"!!(f <= nan)", !!(f <= nan), false},
	{"!!(f >= nan)", !!(f >= nan), false},
	{"!!(nan == f)", !!(nan == f), false},
	{"!!(nan != f)", !!(nan != f), true},
	{"!!(nan < f)", !!(nan < f), false},
	{"!!(nan > f)", !!(nan > f), false},
	{"!!(nan <= f)", !!(nan <= f), false},
	{"!!(nan >= f)", !!(nan >= f), false},
}

func main() {
	bad := false
	for _, t := range tests {
		if t.Expr != t.Want {
			if !bad {
				bad = true
				fmt.Println("BUG: floatcmp")
			}
			fmt.Println(t.Name, "=", t.Expr, "want", t.Want)
		}
	}
	if bad {
		panic("floatcmp failed")
	}
}
