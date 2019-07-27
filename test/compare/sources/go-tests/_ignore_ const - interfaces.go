//+build ignore

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

func assert(t bool, s string) {
	if !t {
		panic(s)
	}
}

func main() {
	var (
		nilN interface{}
		// nilI *int
		five = 5

		_ = nil == interface{}(nil)
		_ = interface{}(nil) == nil
	)
	ii := func(i1 interface{}, i2 interface{}) bool { return i1 == i2 }
	ni := func(n interface{}, i int) bool { return n == i }
	// in := func(i int, n interface{}) bool { return i == n }
	pi := func(p *int, i interface{}) bool { return p == i }
	ip := func(i interface{}, p *int) bool { return i == p }

	assert((interface{}(nil) == interface{}(nil)) == ii(nilN, nilN),
		"for interface{}==interface{} compiler == runtime")

	// TODO(Gianluca): https://github.com/open2b/scriggo/issues/203
	// assert(((*int)(nil) == interface{}(nil)) == pi(nilI, nilN),
	// 	"for *int==interface{} compiler == runtime")
	// assert((interface{}(nil) == (*int)(nil)) == ip(nilN, nilI),
	// 	"for interface{}==*int compiler == runtime")

	assert((&five == interface{}(nil)) == pi(&five, nilN),
		"for interface{}==*int compiler == runtime")
	assert((interface{}(nil) == &five) == ip(nilN, &five),
		"for interface{}==*int compiler == runtime")

	// TODO(Gianluca): https://github.com/open2b/scriggo/issues/204
	assert((5 == interface{}(5)) == ni(five, five),
		"for int==interface{} compiler == runtime")
	// assert((interface{}(5) == 5) == in(five, five),
	// 	"for interface{}==int compiler == runtime")
}
